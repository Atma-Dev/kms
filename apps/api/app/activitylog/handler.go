package activitylog

import (
	"api-kms/enums"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	hub     *Hub
}

// Index godoc
// @Summary      List activity logs
// @Description  Requires the management.audits.show permission.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int  false  "Page number"      default(1)
// @Param        per_page  query     int  false  "Items per page"   default(10)
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]activitylog.Resource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/audits [get]
func (h *Handler) Index(c *gin.Context) error {
	logs, meta, err := h.service.GetAllPaginated(c)
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Activity logs retrieved", utils.Payload{
		Data: NewResourceCollection(logs),
		Meta: meta,
	})
	return nil
}

// Serve godoc
// @Summary      Stream activity logs over WebSocket
// @Description  Upgrades to a WebSocket connection and streams every new activity log row as it's recorded. Requires the management.audits.show permission. The JWT travels as ?token=... since a WebSocket handshake can't carry a custom Authorization header.
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

func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{service: service, hub: hub}
}
