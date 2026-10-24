package queue

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	hub     *Hub
}

func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

// Index godoc
// @Summary      List queue jobs
// @Description  Paginated queue job list. Requires the management.queue.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number"      default(1)
// @Param        per_page  query     int     false  "Items per page"   default(10)
// @Param        queue     query     string  false  "Filter by queue name"
// @Param        name      query     string  false  "Filter by job name"
// @Param        status    query     string  false  "Filter by status"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]queue.Resource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/queue/jobs [get]
func (h *Handler) Index(c *gin.Context) error {
	jobs, meta, err := h.service.GetAllPaginated(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Queue jobs retrieved", utils.Payload{
		Data: NewResourceCollection(jobs, h.service.registry),
		Meta: meta,
	})
	return nil
}

// Stats godoc
// @Summary      Queue depth and health
// @Description  Current counts by status/queue/name, delay backlog and oldest pending job age. Requires the management.queue.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=queue.Stats}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/queue/stats [get]
func (h *Handler) Stats(c *gin.Context) error {
	stats, err := h.service.GetStats(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Queue stats retrieved", stats)
	return nil
}

// Retry godoc
// @Summary      Retry a finished queue job
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Queue job ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=queue.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/queue/jobs/{id}/retry [post]
func (h *Handler) Retry(c *gin.Context) error {
	job, err := h.service.Retry(c, c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Queue job queued for retry", NewResource(job, h.service.registry))
	return nil
}

// Cancel godoc
// @Summary      Cancel a pending queue job
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Queue job ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=queue.Resource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/queue/jobs/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) error {
	job, err := h.service.Cancel(c, c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Queue job canceled", NewResource(job, h.service.registry))
	return nil
}

// RetryFailed godoc
// @Summary      Retry every failed queue job
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      queue.RetryFailedRequest  false  "Optionally narrow to one job name"
// @Success      200  {object}  utils.HTTPResponse{payload=object{retried=int}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/queue/jobs/retry-failed [post]
func (h *Handler) RetryFailed(c *gin.Context) error {
	var request RetryFailedRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}
	n, err := h.service.RetryFailed(c, &request)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Failed queue jobs retried", gin.H{"retried": n})
	return nil
}

// Purge godoc
// @Summary      Delete finished queue jobs
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      queue.PurgeRequest  true  "Statuses to purge (success|failed|canceled)"
// @Success      200  {object}  utils.HTTPResponse{payload=object{purged=int}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/queue/jobs [delete]
func (h *Handler) Purge(c *gin.Context) error {
	var request PurgeRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}
	n, err := h.service.Purge(c, &request)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Finished queue jobs purged", gin.H{"purged": n})
	return nil
}

// Serve godoc
// @Summary      Stream queue job state over WebSocket
// @Description  Upgrades to a WebSocket connection and streams every queue job state transition (enqueue, claim, finish). Requires the management.queue.show permission. The JWT travels as ?token=... since a WebSocket handshake can't carry a custom Authorization header.
// @Tags         management
// @Param        token  query  string  true  "Access token"
// @Success      101  {string}  string  "switching protocols"
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/queue/ws [get]
func (h *Handler) Serve(c *gin.Context) error {
	conn, err := h.hub.Upgrade(c.Writer, c.Request)
	if err != nil {
		// gorilla/websocket's Upgrade already wrote an HTTP error response on
		// failure — returning our own here would double-write the header.
		return nil
	}
	defer func() {
		h.hub.Unregister(conn)
		conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	return nil
}
