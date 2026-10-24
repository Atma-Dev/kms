package middleware

import (
	stderrors "errors"
	"api-kms/app/m2mclient"
	"api-kms/config"
	"api-kms/errors"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goaudit "github.com/iqbalatma/go-audit"
	"gorm.io/gorm"
)

var (
	errM2MMissingCredentials = stderrors.New("m2m: missing X-Client-Id/X-Client-Secret headers")
	errM2MSecretMismatch     = stderrors.New("m2m: secret does not match")
)

// loadM2MClient folds "unknown client" and "revoked client" into the same
// not-found path: from the caller's side there is no difference, and telling
// them apart in the response would tell a stolen credential's holder that the
// id was right and only the secret (or its revocation) is the problem.
//
// Matched on id (the row's UUID), not the client name: a name is
// human-chosen and often guessable, where the id is opaque
// and already generated for every client regardless — no reason to keep two
// identifiers when one already does the job intended for X-Client-Id.
func loadM2MClient(c *gin.Context, clientID string) (*m2mclient.M2MClient, *utils.HTTPError) {
	// Checked before the query: Postgres rejects a non-UUID compared with the
	// uuid id column (22P02) rather than finding nothing, and that would turn
	// a bad credential into a 500.
	if _, err := uuid.Parse(clientID); err != nil {
		return nil, errors.InvalidCredential().WithCause(err)
	}

	var client m2mclient.M2MClient
	err := config.DB.WithContext(c.Request.Context()).
		Where("id = ? AND revoked_at IS NULL", clientID).
		First(&client).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.InvalidCredential().WithCause(err)
		}
		return nil, errors.InternalServerError("failed to load m2m client").WithCause(err)
	}
	return &client, nil
}

// M2MAuthMiddleware authenticates a machine-to-machine caller via the
// X-Client-Id / X-Client-Secret header pair rather than HTTP Basic Auth.
// X-Client-Id is the client's id (UUID), not its display name — see
// loadM2MClient. Sets "m2m_client" in the gin context, mirroring how
// AuthMiddleware sets "user".
func M2MAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetHeader("X-Client-Id")
		secret := c.GetHeader("X-Client-Secret")
		if clientID == "" || secret == "" {
			abortWithError(c, errors.InvalidCredential().WithCause(errM2MMissingCredentials))
			return
		}

		client, httpErr := loadM2MClient(c, clientID)
		if httpErr != nil {
			abortWithError(c, httpErr)
			return
		}

		if !utils.CheckHash(client.Secret, secret) {
			abortWithError(c, errors.InvalidCredential().WithCause(errM2MSecretMismatch))
			return
		}

		c.Set("m2m_client", client)

		// Mirrors AuditContextMiddleware for the user-JWT path: without
		// this, an M2M-triggered mutation (e.g. DELETE
		// /v1/m2m/invoices/:id) still gets an audits row, just with an
		// empty actor/request — this is what makes it traceable to which
		// client did it.
		ctx := goaudit.WithRequestInfo(c.Request.Context(), goaudit.RequestInfoFromHTTP(c.Request))
		ctx = goaudit.WithActor(ctx, goaudit.Actor{
			Table: "m2m_clients",
			ID:    client.ID.String(),
			Name:  client.Client,
		})
		c.Request = c.Request.WithContext(ctx)
	}
}
