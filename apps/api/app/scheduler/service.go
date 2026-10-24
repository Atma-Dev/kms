package scheduler

import (
	"api-kms/errors"

	"github.com/gin-gonic/gin"
)

type Service struct {
	repository *Repository
	runner     *Runner
}

func NewService(repository *Repository, runner *Runner) *Service {
	return &Service{repository: repository, runner: runner}
}

func (s *Service) GetAll(c *gin.Context) ([]ScheduledJob, error) {
	return s.repository.GetAll(c.Request.Context())
}

func (s *Service) UpdateById(c *gin.Context, id string, request *UpdateScheduleRequest) (*ScheduledJob, error) {
	if _, err := ParseCron(request.CronExpression); err != nil {
		return nil, errors.InvalidAction("invalid cron expression: " + err.Error())
	}

	schedule, err := s.repository.GetById(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}

	schedule.CronExpression = request.CronExpression
	schedule.Enabled = request.Enabled
	if err := s.repository.UpdateById(c.Request.Context(), schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

// RunNow queues a manual run and returns immediately — the run happens on
// the runner's own goroutine, not tied to this request's cancelable
// context, so a client navigating away doesn't cut off a job that outlives
// the HTTP request.
func (s *Service) RunNow(c *gin.Context, id string) (*ScheduledJob, error) {
	schedule, err := s.repository.GetById(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	s.runner.Trigger(schedule.Name)
	return schedule, nil
}
