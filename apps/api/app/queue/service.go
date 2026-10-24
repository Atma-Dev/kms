package queue

import (
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Service struct {
	repository *Repository
	registry   *Registry
	hub        *Hub
	notifier   *Notifier
}

func NewService(repository *Repository, registry *Registry, hub *Hub, notifier *Notifier) *Service {
	return &Service{repository: repository, registry: registry, hub: hub, notifier: notifier}
}

func (s *Service) GetAllPaginated(c *gin.Context) ([]Job, *utils.PaginationMeta, error) {
	return s.repository.GetAllPaginated(c)
}

func (s *Service) GetStats(c *gin.Context) (*Stats, error) {
	stats, err := s.repository.GetStats(c.Request.Context())
	if err != nil {
		return nil, err
	}
	stats.RegisteredJobs = s.registry.Names()

	queues, err := s.repository.DistinctQueues(c.Request.Context())
	if err != nil {
		return nil, err
	}
	stats.Queues = queues
	return stats, nil
}

func (s *Service) Retry(c *gin.Context, id string) (*Job, error) {
	job, err := s.repository.Retry(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	s.hub.Broadcast(NewResource(job, s.registry))
	s.notifier.Notify(c.Request.Context(), job.Queue)
	return job, nil
}

// RetryFailed doesn't broadcast/wake per row: a bulk retry can touch many
// jobs across many queues, and the worker's own poll ticker (PollInterval)
// picks them all up regardless — not worth another query per queue for a
// manual, infrequent action.
func (s *Service) RetryFailed(c *gin.Context, request *RetryFailedRequest) (int64, error) {
	return s.repository.RetryFailed(c.Request.Context(), request.Name)
}

func (s *Service) Cancel(c *gin.Context, id string) (*Job, error) {
	job, err := s.repository.Cancel(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	s.hub.Broadcast(NewResource(job, s.registry))
	return job, nil
}

func (s *Service) Purge(c *gin.Context, request *PurgeRequest) (int64, error) {
	return s.repository.Purge(c.Request.Context(), request.Statuses)
}
