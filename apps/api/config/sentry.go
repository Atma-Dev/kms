package config

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// LoadSentry is a no-op when SENTRY_DSN is unset, so local dev and CI need no
// account. AppEnv tags events so staging noise doesn't mix with production.
func LoadSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: AppConfig.AppEnv,
	})
	if err != nil {
		AppLogger.Errorf("sentry init failed: %v", err)
	}
}

// FlushSentry blocks until buffered events are sent, or the timeout passes.
// Call it before the process exits so the last panic isn't lost.
func FlushSentry() {
	sentry.Flush(2 * time.Second)
}
