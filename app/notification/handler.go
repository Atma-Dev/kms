package notification

import (
	"api-kms/app/user"
	"api-kms/enums"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	hub     *Hub
}

// Index godoc
// @Summary      List the authenticated user's notifications
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int  false  "Page number"      default(1)
// @Param        per_page  query     int  false  "Items per page"   default(10)
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]notification.Resource,meta=utils.PaginationMeta}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /notifications [get]
func (h *Handler) Index(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	notifications, meta, err := h.service.GetAllPaginated(c, authUser.ID.String())
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Notifications retrieved", utils.Payload{
		Data: NewResourceCollection(notifications),
		Meta: meta,
	})
	return nil
}

// MarkRead godoc
// @Summary      Mark one notification read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Notification ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /notifications/{id}/read [patch]
func (h *Handler) MarkRead(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	if err := h.service.MarkRead(c.Request.Context(), c.Param("id"), authUser.ID.String()); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Notification marked read", nil)
	return nil
}

// MarkAllRead godoc
// @Summary      Mark all of the authenticated user's notifications read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Router       /notifications/read-all [patch]
func (h *Handler) MarkAllRead(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	if err := h.service.MarkAllRead(c.Request.Context(), authUser.ID.String()); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "Notifications marked read", nil)
	return nil
}

// Serve godoc
// @Summary      Stream notifications over WebSocket
// @Description  Upgrades to a WebSocket connection and blocks until the client disconnects. The JWT travels as ?token=... since a WebSocket handshake can't carry a custom Authorization header.
// @Tags         notifications
// @Param        token  query  string  true  "Access token"
// @Success      101  {string}  string  "switching protocols"
// @Failure      401  {object}  utils.HTTPError
// @Router       /notifications/ws [get]
func (h *Handler) Serve(c *gin.Context) error {
	authUser := c.MustGet("user").(*user.User)

	conn, err := h.hub.Upgrade(c.Writer, c.Request, authUser.ID.String())
	if err != nil {
		// gorilla/websocket's Upgrade already wrote an HTTP error response on
		// failure — returning our own here would double-write the header.
		return nil
	}
	defer func() {
		h.hub.Unregister(authUser.ID.String(), conn)
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
