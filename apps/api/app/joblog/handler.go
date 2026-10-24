package joblog

import (
	"api-kms/enums"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

// Definition describes one registered job for the "what's scheduled" list —
// static, since app/scheduler's jobs are all wired at startup rather than
// registered dynamically at runtime.
type Definition struct {
	Type     Type   `json:"type"`
	Name     string `json:"name"`
	Interval string `json:"interval"`
}

type Handler struct {
	service     *Service
	hub         *Hub
	definitions []Definition
}

func NewHandler(service *Service, hub *Hub, definitions []Definition) *Handler {
	return &Handler{service: service, hub: hub, definitions: definitions}
}

// Definitions godoc
// @Summary      List registered background jobs
// @Description  Static list of every scheduled/queue job this app runs, with its interval. Requires the management.job-logs.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=[]joblog.Definition}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/job-logs/definitions [get]
func (h *Handler) Definitions(c *gin.Context) error {
	utils.ResponseJSON(c, enums.SUCCESS, "Job definitions retrieved", h.definitions)
	return nil
}

// Index godoc
// @Summary      List job runs
// @Description  Paginated run history for every scheduled/queue job. Requires the management.job-logs.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number"      default(1)
// @Param        per_page  query     int     false  "Items per page"   default(10)
// @Param        type      query     string  false  "Filter by type (schedule|queue)"
// @Param        name      query     string  false  "Filter by job name"
// @Param        status    query     string  false  "Filter by status (success|failed)"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]joblog.Resource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/job-logs [get]
func (h *Handler) Index(c *gin.Context) error {
	logs, meta, err := h.service.GetAllPaginated(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Job logs retrieved", utils.Payload{
		Data: NewResourceCollection(logs),
		Meta: meta,
	})
	return nil
}

// Stats godoc
// @Summary      Job log dashboard stats
// @Description  Summary, trend and per-job rollup for scheduled/queue runs. Requires the management.job-logs.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        range  query     string  false  "24h|7d|30d"  default(24h)
// @Param        type   query     string  false  "Filter by type (schedule|queue)"
// @Success      200  {object}  utils.HTTPResponse{payload=joblog.Stats}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/job-logs/stats [get]
func (h *Handler) Stats(c *gin.Context) error {
	stats, err := h.service.GetStats(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Job log stats retrieved", stats)
	return nil
}

// Serve godoc
// @Summary      Stream job runs over WebSocket
// @Description  Upgrades to a WebSocket connection and streams every new job log row as it's recorded. Requires the management.job-logs.show permission. The JWT travels as ?token=... since a WebSocket handshake can't carry a custom Authorization header.
// @Tags         management
// @Param        token  query  string  true  "Access token"
// @Success      101  {string}  string  "switching protocols"
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/job-logs/ws [get]
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
