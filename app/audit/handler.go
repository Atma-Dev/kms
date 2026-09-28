package audit

import (
	"api-kms/enums"
	"api-kms/utils"

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
// @Summary      List audit trails
// @Description  Field-level change history (who changed what, from what to what). Requires the management.audits.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        page                query     int     false  "Page number"      default(1)
// @Param        per_page            query     int     false  "Items per page"   default(10)
// @Param        entry_object_table  query     string  false  "Filter by entry object table, e.g. deposits"
// @Param        entry_object_id     query     string  false  "Filter by entry object id"
// @Param        actor_id            query     string  false  "Filter by actor id"
// @Param        action              query     string  false  "Filter by action, e.g. UPDATE_DEPOSIT_STATUS"
// @Param        date_from           query     string  false  "Filter created_at >= this date (YYYY-MM-DD)"
// @Param        date_to             query     string  false  "Filter created_at <= this date (YYYY-MM-DD)"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]audit.Resource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/audits [get]
func (h *Handler) Index(c *gin.Context) error {
	rows, meta, err := h.service.GetAllPaginated(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Audit trails retrieved", utils.Payload{
		Data: NewResourceCollection(rows),
		Meta: meta,
	})
	return nil
}

// Serve godoc
// @Summary      Stream audits over WebSocket
// @Description  Upgrades to a WebSocket connection and pushes every new audit as it's recorded — a live "something changed" feed, separate from the (manually refreshed) list above. Requires the management.audits.show permission. The JWT travels as ?token=... since a WebSocket handshake can't carry a custom Authorization header.
// @Tags         management
// @Param        token  query  string  true  "Access token"
// @Success      101  {string}  string  "switching protocols"
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/audits/ws [get]
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
