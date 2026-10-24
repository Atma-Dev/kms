package m2mclient

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"api-kms/app"
	"time"
)

// Scope names the routes a client may call. "*" grants every scope — a single
// wildcard string instead of a whole roles table for a handful of M2M routes.
const (
	ScopeAll = "*"
)

// Scopes is a JSONB array column: the list is short and never queried by
// element, so a scan/value pair is simpler than a join table.
type Scopes []string

func (s Scopes) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *Scopes) Scan(value any) error {
	if value == nil {
		*s = Scopes{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("m2m_clients.scope: unexpected column type")
	}
	return json.Unmarshal(b, s)
}

// Has reports whether the client may call a route gated on scope.
func (s Scopes) Has(scope string) bool {
	for _, held := range s {
		if held == ScopeAll || held == scope {
			return true
		}
	}
	return false
}

// M2MClient authenticates a machine-to-machine caller by client + secret.
// RevokedAt (not a delete) keeps the row so a revoked client's past requests
// stay attributable.
type M2MClient struct {
	app.BaseModel `gorm:"embedded"`
	Client        string     `json:"client" gorm:"column:client"`
	Secret        string     `json:"-" gorm:"column:secret"`
	Scope         Scopes     `json:"scope" gorm:"column:scope"`
	RevokedAt     *time.Time `json:"revoked_at" gorm:"column:revoked_at"`
}

func (M2MClient) TableName() string {
	return "m2m_clients"
}
