package utils

import (
	"context"
	"api-kms/config"
	"time"
)

// wsOnlyTokenKeyPrefix marks a token's jti as scoped to WebSocket routes
// only. AuthMiddleware refuses any token found under this prefix — that
// refusal, not anything in the WS middleware itself, is what actually keeps
// a ws-token from working as a general-purpose bearer token. See
// POST /auth/ws-token (app/auth/handler.go).
const wsOnlyTokenKeyPrefix = "ws_token:"

// RegisterWSOnlyToken marks jti as WS-only for ttl (matching the token's own
// expiry, so the registry entry never outlives the token it describes).
func RegisterWSOnlyToken(ctx context.Context, jti string, ttl time.Duration) error {
	return config.RDB.Set(ctx, wsOnlyTokenKeyPrefix+jti, "1", ttl).Err()
}

// IsWSOnlyToken reports whether jti was issued as WS-only. A Redis error is
// treated as "not WS-only" — failing open here would only ever let a WS-only
// token slip through as a regular one, never the reverse, and Redis being
// unreachable already breaks the JWT blacklist this app depends on for
// logout, so it is not a new failure mode.
func IsWSOnlyToken(ctx context.Context, jti string) bool {
	n, err := config.RDB.Exists(ctx, wsOnlyTokenKeyPrefix+jti).Result()
	return err == nil && n > 0
}
