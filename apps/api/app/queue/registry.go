package queue

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// JobHandler is one named unit of work a queue job can run. Register it once
// in routes.NewContainer's queue.NewRegistry(...) call and every
// Dispatch(ctx, name, payload) call for that name routes here.
type JobHandler interface {
	Name() string
	Handle(ctx context.Context, payload []byte) (message string, err error)
}

// ConfigurableAttempts lets a handler override DefaultMaxAttempts — for a
// job that calls something flakier than usual and wants more than 3 tries.
type ConfigurableAttempts interface {
	MaxAttempts() int
}

// SensitivePayload marks a handler whose payload must never sit in
// queue_jobs (or be returned over the API) once nothing will retry it. A
// durable table with a monitoring screen attached is not where a working
// password-reset link belongs.
type SensitivePayload interface {
	SensitivePayload() bool
}

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

func NewRegistry(handlers ...JobHandler) *Registry {
	r := &Registry{handlers: make(map[string]JobHandler, len(handlers))}
	for _, h := range handlers {
		r.Register(h)
	}
	return r
}

// Register panics on a duplicate name — two handlers silently fighting over
// the same job name should crash at startup, not surface later as "the
// wrong one ran".
func (r *Registry) Register(h JobHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[h.Name()]; exists {
		panic(fmt.Sprintf("queue: handler %q already registered", h.Name()))
	}
	r.handlers[h.Name()] = h
}

func (r *Registry) Get(name string) (JobHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// Names lists every registered handler, sorted — feeds the filter dropdown
// and the "what's registered" panel on the queue monitoring page.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) IsSensitive(name string) bool {
	h, ok := r.Get(name)
	if !ok {
		return false
	}
	sp, ok := h.(SensitivePayload)
	return ok && sp.SensitivePayload()
}

func (r *Registry) maxAttempts(name string) int {
	h, ok := r.Get(name)
	if !ok {
		return DefaultMaxAttempts
	}
	if ca, ok := h.(ConfigurableAttempts); ok {
		return ca.MaxAttempts()
	}
	return DefaultMaxAttempts
}
