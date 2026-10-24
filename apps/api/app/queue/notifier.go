package queue

import (
	"context"
	"api-kms/config"
	"time"

	"github.com/redis/go-redis/v9"
)

const notifyChannel = "queue:dispatch"

// Notifier is Redis used only as a doorbell to wake an idle worker early.
// MySQL holds the jobs, so a missed or lost publish costs latency (the
// worker's own poll ticker finds the job anyway), never a dropped job.
type Notifier struct {
	rdb *redis.Client
}

func NewNotifier(rdb *redis.Client) *Notifier {
	return &Notifier{rdb: rdb}
}

func (n *Notifier) Notify(ctx context.Context, queueName string) {
	if n == nil || n.rdb == nil {
		return
	}
	if err := n.rdb.Publish(ctx, notifyChannel, queueName).Err(); err != nil {
		config.AppLogger.WithError(err).Debug("failed to publish queue doorbell")
	}
}

// Listen returns a channel of queue names to wake a worker's poll loop
// early. A nil client (Redis unreachable/not configured) returns a channel
// that never delivers — workers degrade to poll-only, which is still
// correct.
func (n *Notifier) Listen(ctx context.Context) <-chan string {
	out := make(chan string)
	if n == nil || n.rdb == nil {
		return out
	}

	go func() {
		for ctx.Err() == nil {
			sub := n.rdb.Subscribe(ctx, notifyChannel)
			for msg := range sub.Channel() {
				select {
				case out <- msg.Payload:
				case <-ctx.Done():
					_ = sub.Close()
					return
				}
			}
			_ = sub.Close()

			if ctx.Err() != nil {
				return
			}
			// Connection dropped — brief pause before resubscribing rather
			// than a hot retry loop.
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}
