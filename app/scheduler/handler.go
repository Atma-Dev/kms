package scheduler

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Index godoc
// @Summary      List scheduled jobs
// @Description  One row per registered background job, with its cron expression, enabled state and last-run outcome. Requires the management.schedules.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]scheduler.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/schedules [get]
func (h *Handler) Index(c *gin.Context) error {
	schedules, err := h.service.GetAll(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Scheduled jobs retrieved", NewResourceCollection(schedules))
	return nil
}

// Update godoc
// @Summary      Update a scheduled job's cron expression / enabled state
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                           true  "Scheduled job ID"
// @Param        request  body      scheduler.UpdateScheduleRequest  true  "Schedule"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=scheduler.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/schedules/{id} [patch]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateScheduleRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	schedule, err := h.service.UpdateById(c, c.Param("id"), &request)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Scheduled job updated", NewResource(schedule))
	return nil
}

// RunNow godoc
// @Summary      Run a scheduled job immediately
// @Description  Queues the job to run now, independent of its cron schedule. The run itself happens asynchronously.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Scheduled job ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=scheduler.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /management/schedules/{id}/run [post]
func (h *Handler) RunNow(c *gin.Context) error {
	schedule, err := h.service.RunNow(c, c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Scheduled job queued to run", NewResource(schedule))
	return nil
}
