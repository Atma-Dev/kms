package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	AppEnv     string
	AppDebug   bool
	DbUser     string
	DbPassword string
	DbHost     string
	DbPort     string
	DbName     string

	OauthGoogleClientID     string
	OauthGoogleClientSecret string
	OauthGoogleRedirectURL  string

	FrontendURL string

	JWTRedisHost     string
	JWTRedisPort     string
	JWTRedisPassword string
	JWTRedisDB       string

	// A plain env var since this app has no runtime-editable configuration
	// store yet — same shape as JWT_ACCESS_TOKEN_TTL below.
	ScheduleJobLogRetentionDays int
	// ScheduleLeaseTTLMinutes bounds how long a scheduled job may hold its
	// run lease without a heartbeat before another instance may claim it —
	// see app/scheduler's Runner.heartbeat.
	ScheduleLeaseTTLMinutes int

	// Queue* configure app/queue's Worker pool. Plain env vars for the same
	// reason as the Schedule* ones above: no runtime-editable configuration
	// store yet.
	QueueWorkers             int
	QueueNames               []string
	QueuePollIntervalSeconds int
	QueueMaxRunTimeSeconds   int
	QueueBackoffBaseSeconds  int
	QueueBackoffMaxSeconds   int
	QueueRetentionDays       int

	// SearchHistoryKeepPerUser is how many past queries survive
	// app/search's nightly prune. Comfortably above HistoryLimit (10), which
	// is all the UI reads: the extra rows cost nothing and mean a user who
	// clears a few entries still has a history behind them.
	SearchHistoryKeepPerUser int

	// Storage* configure app/attachment's on-disk store. StoragePath is the
	// root every file_path is relative to, so moving the store is one env var
	// and no database change.
	//
	// The two size caps are separate because the limits are for different
	// things: an attachment is a file someone downloads, an inline image is
	// decoded by every reader's browser on every view.
	StoragePath          string
	UploadMaxSizeMB      int
	UploadImageMaxSizeMB int
	// UploadOrphanRetentionHours bounds how long an inline image that never
	// reached a document is kept. Hours, not days: it only has to outlive a
	// writing session.
	UploadOrphanRetentionHours int

	// ViewSessionMaxTTLHours caps how long a share link (Modul 4 FR-5) may be
	// set to last. Whoever creates one picks the expiry; this is the ceiling,
	// because a link with no ceiling is a permanent grant somebody meant as a
	// temporary one. 720 hours is 30 days.
	ViewSessionMaxTTLHours int
}

var AppConfig *Config

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	AppConfig = &Config{
		AppPort:    os.Getenv("APP_PORT"),
		AppEnv:     os.Getenv("APP_ENV"),
		AppDebug:   appDebug(),
		DbUser:     os.Getenv("DB_USER"),
		DbPassword: os.Getenv("DB_PASSWORD"),
		DbHost:     os.Getenv("DB_HOST"),
		DbPort:     os.Getenv("DB_PORT"),
		DbName:     os.Getenv("DB_NAME"),

		JWTRedisHost:     os.Getenv("JWT_REDIS_HOST"),
		JWTRedisPort:     os.Getenv("JWT_REDIS_PORT"),
		JWTRedisPassword: os.Getenv("JWT_REDIS_PASSWORD"),
		JWTRedisDB:       os.Getenv("JWT_REDIS_DB"),

		OauthGoogleClientID:     os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
		OauthGoogleClientSecret: os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
		OauthGoogleRedirectURL:  os.Getenv("OAUTH_GOOGLE_REDIRECT_URL"),

		FrontendURL: os.Getenv("FRONTEND_URL"),

		ScheduleJobLogRetentionDays: envIntOrDefault("SCHEDULE_JOB_LOG_RETENTION_DAYS", 3),
		ScheduleLeaseTTLMinutes:     envIntOrDefault("SCHEDULE_LEASE_TTL_MINUTES", 5),

		QueueWorkers:             envIntOrDefault("QUEUE_WORKERS", 5),
		QueueNames:               envListOrDefault("QUEUE_NAMES", []string{"default"}),
		QueuePollIntervalSeconds: envIntOrDefault("QUEUE_POLL_INTERVAL_SECONDS", 3),
		QueueMaxRunTimeSeconds:   envIntOrDefault("QUEUE_MAX_RUN_TIME_SECONDS", 300),
		QueueBackoffBaseSeconds:  envIntOrDefault("QUEUE_BACKOFF_BASE_SECONDS", 10),
		QueueBackoffMaxSeconds:   envIntOrDefault("QUEUE_BACKOFF_MAX_SECONDS", 600),
		QueueRetentionDays:       envIntOrDefault("QUEUE_RETENTION_DAYS", 7),

		SearchHistoryKeepPerUser: envIntOrDefault("SEARCH_HISTORY_KEEP_PER_USER", 50),

		StoragePath:                envOrDefault("STORAGE_PATH", "storage"),
		UploadMaxSizeMB:            envIntOrDefault("UPLOAD_MAX_SIZE_MB", 10),
		UploadImageMaxSizeMB:       envIntOrDefault("UPLOAD_IMAGE_MAX_SIZE_MB", 5),
		UploadOrphanRetentionHours: envIntOrDefault("UPLOAD_ORPHAN_RETENTION_HOURS", 24),

		ViewSessionMaxTTLHours: envIntOrDefault("VIEW_SESSION_MAX_TTL_HOURS", 720),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}

// envListOrDefault reads a comma-separated env var (e.g. "default,emails");
// blank or unset falls back rather than yielding a single empty-string entry.
func envListOrDefault(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

// appDebug reads APP_DEBUG and fails closed. Anything missing, empty, or
// unparseable means off, so a broken .env in production cannot switch error
// detail on by accident.
func appDebug() bool {
	enabled, err := strconv.ParseBool(os.Getenv("APP_DEBUG"))
	return err == nil && enabled
}

// DebugEnabled reports whether error detail may be sent to the client. Only
// ErrorHandler should consult it: keeping the decision in one place is what
// stops it from being forgotten somewhere else.
func DebugEnabled() bool {
	return AppConfig != nil && AppConfig.AppDebug
}
