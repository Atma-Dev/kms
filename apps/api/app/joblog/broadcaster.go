package joblog

import "context"

// Broadcaster is the single entry point app/scheduler uses to record a job
// run: persist then push keeps the live feed and the stored list always
// consistent — nothing goes out over the socket that isn't also in the list
// a page refresh would show.
type Broadcaster struct {
	repo *Repository
	hub  *Hub
}

func NewBroadcaster(repo *Repository, hub *Hub) *Broadcaster {
	return &Broadcaster{repo: repo, hub: hub}
}

func (b *Broadcaster) Record(ctx context.Context, log *JobLog) error {
	if err := b.repo.Create(ctx, log); err != nil {
		return err
	}
	b.hub.Broadcast(NewResource(log))
	return nil
}
