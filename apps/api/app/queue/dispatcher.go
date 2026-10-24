package queue

import (
	"context"
	"encoding/json"
	"api-kms/errors"
	"time"

	"gorm.io/gorm"
)

// Options overrides a dispatched job's defaults.
type Options struct {
	Queue        string
	Delay        time.Duration
	MaxAttempts  int
	DispatchedBy string
}

type Dispatcher struct {
	repository *Repository
	registry   *Registry
	hub        *Hub
	notifier   *Notifier
}

func NewDispatcher(repository *Repository, registry *Registry, hub *Hub, notifier *Notifier) *Dispatcher {
	return &Dispatcher{repository: repository, registry: registry, hub: hub, notifier: notifier}
}

// Dispatch builds and persists a job on its own connection, then announces
// it immediately. Do not call this from inside an already-open transaction
// — see Transaction below, the only sanctioned way to dispatch alongside
// writes that must commit or roll back together. Calling Dispatch there
// either runs the job before the surrounding write commits, or lets a
// worker claim and fail it against data that was rolled back.
func (d *Dispatcher) Dispatch(ctx context.Context, name string, payload any, options ...Options) (*Job, error) {
	job, err := d.build(ctx, name, payload, options...)
	if err != nil {
		return nil, err
	}
	if err := d.repository.Create(ctx, job); err != nil {
		return nil, err
	}
	d.Announce(ctx, job)
	return job, nil
}

// Tx is a Dispatcher scoped to one already-open transaction: Dispatch here
// writes on tx, not a fresh connection, so the job row commits or rolls
// back with whatever else the caller is doing in that transaction.
type Tx struct {
	dispatcher *Dispatcher
	tx         *gorm.DB
	pending    []*Job
}

func (t *Tx) Dispatch(ctx context.Context, name string, payload any, options ...Options) (*Job, error) {
	job, err := t.dispatcher.build(ctx, name, payload, options...)
	if err != nil {
		return nil, err
	}
	if err := t.dispatcher.repository.CreateTx(ctx, t.tx, job); err != nil {
		return nil, err
	}
	t.pending = append(t.pending, job)
	return job, nil
}

// Transaction runs fn inside a DB transaction; every job dispatched through
// the *Tx it's given commits or vanishes with the caller's own writes (an
// outbox pattern without a separate outbox table), and is only announced
// over the hub/notifier once that commit has actually succeeded.
func (d *Dispatcher) Transaction(ctx context.Context, fn func(tx *gorm.DB, q *Tx) error) error {
	qtx := &Tx{dispatcher: d}
	err := d.repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		qtx.tx = tx
		return fn(tx, qtx)
	})
	if err != nil {
		return err
	}
	d.Announce(ctx, qtx.pending...)
	return nil
}

// DispatchTx dispatches into a transaction the caller already opened and
// will commit itself. The caller must call Announce after that commit
// succeeds — this only writes the row.
func (d *Dispatcher) DispatchTx(ctx context.Context, tx *gorm.DB, name string, payload any, options ...Options) (*Job, error) {
	job, err := d.build(ctx, name, payload, options...)
	if err != nil {
		return nil, err
	}
	if err := d.repository.CreateTx(ctx, tx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// Announce pushes each job over the hub and wakes a worker for its queue —
// one wake-up per distinct queue name, not per job, since a worker draining
// its queue picks up every pending row in one pass.
func (d *Dispatcher) Announce(ctx context.Context, jobs ...*Job) {
	woken := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		d.hub.Broadcast(NewResource(job, d.registry))
		if !woken[job.Queue] {
			d.notifier.Notify(ctx, job.Queue)
			woken[job.Queue] = true
		}
	}
}

func (d *Dispatcher) build(_ context.Context, name string, payload any, options ...Options) (*Job, error) {
	if _, registered := d.registry.Get(name); !registered {
		return nil, errors.InvalidAction("no queue handler is registered for " + name)
	}

	opts := Options{}
	if len(options) > 0 {
		opts = options[0]
	}

	queueName := opts.Queue
	if queueName == "" {
		queueName = DefaultQueue
	}

	maxAttempts := opts.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = d.registry.maxAttempts(name)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.InvalidAction("failed to encode queue job payload")
	}

	availableAt := time.Now()
	if opts.Delay > 0 {
		availableAt = availableAt.Add(opts.Delay)
	}

	job := &Job{
		Queue:       queueName,
		Name:        name,
		Payload:     body,
		Status:      StatusPending,
		MaxAttempts: maxAttempts,
		AvailableAt: availableAt,
	}
	if opts.DispatchedBy != "" {
		job.DispatchedBy = &opts.DispatchedBy
	}
	return job, nil
}
