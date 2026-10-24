package m2mclient

import (
	"api-kms/enums"
	"api-kms/utils"
	"api-kms/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

// Index godoc
// @Summary      List M2M clients
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]m2mclient.M2MClientResource}}
// @Failure      401  {object}  utils.HTTPError
// @Router       /management/m2m-clients [get]
func (h *Handler) Index(c *gin.Context) error {
	clients, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "M2M clients retrieved", NewM2MClientResourceCollection(clients))
	return nil
}

// Show godoc
// @Summary      Get an M2M client
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "M2M client ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=m2mclient.M2MClientResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /management/m2m-clients/{id} [get]
func (h *Handler) Show(c *gin.Context) error {
	client, err := h.service.GetById(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "M2M client retrieved", NewM2MClientResource(client))
	return nil
}

// Store godoc
// @Summary      Create an M2M client
// @Description  The response includes the plaintext secret once; it cannot be retrieved again.
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      m2mclient.StoreM2MClientRequest  true  "M2M client"
// @Success      201  {object}  utils.HTTPResponse{payload=object{data=m2mclient.M2MClientCreatedResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/m2m-clients [post]
func (h *Handler) Store(c *gin.Context) error {
	var request StoreM2MClientRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	client, plaintextSecret, err := h.service.AddNew(c.Request.Context(), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.CREATED, "M2M client created", NewM2MClientCreatedResource(client, plaintextSecret))
	return nil
}

// Update godoc
// @Summary      Update an M2M client
// @Tags         management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                            true  "M2M client ID"
// @Param        request  body      m2mclient.UpdateM2MClientRequest  true  "M2M client"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=m2mclient.M2MClientResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/m2m-clients/{id} [patch]
func (h *Handler) Update(c *gin.Context) error {
	var request UpdateM2MClientRequest
	if err := validator.BindAndValidate(c, &request); err != nil {
		return err
	}

	client, err := h.service.UpdateById(c.Request.Context(), c.Param("id"), &request)
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "M2M client updated", NewM2MClientResource(client))
	return nil
}

// RegenerateSecret godoc
// @Summary      Regenerate an M2M client's secret
// @Description  Issues a new secret and invalidates the old one immediately. The response includes the plaintext secret once; it cannot be retrieved again. Refused on a revoked client.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "M2M client ID"
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=m2mclient.M2MClientCreatedResource}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      422  {object}  utils.HTTPError
// @Router       /management/m2m-clients/{id}/regenerate-secret [post]
func (h *Handler) RegenerateSecret(c *gin.Context) error {
	client, plaintextSecret, err := h.service.RegenerateSecret(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	utils.ResponseJSON(c, enums.SUCCESS, "M2M client secret regenerated", NewM2MClientCreatedResource(client, plaintextSecret))
	return nil
}

// Destroy godoc
// @Summary      Revoke an M2M client
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "M2M client ID"
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Router       /management/m2m-clients/{id} [delete]
func (h *Handler) Destroy(c *gin.Context) error {
	if err := h.service.RevokeById(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	utils.ResponseJSON(c, enums.SUCCESS, "M2M client revoked", nil)
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}
