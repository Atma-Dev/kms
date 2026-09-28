package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"api-kms/app/activitylog"
	auditpkg "api-kms/app/audit"
	"api-kms/app/joblog"
	"api-kms/app/queue"
	_ "api-kms/docs"
	"api-kms/enums"
	middleware2 "api-kms/middleware"
	"api-kms/validator"

	"api-kms/config"
	"api-kms/routes"
	"api-kms/utils"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	goaudit "github.com/iqbalatma/go-audit"
	"github.com/iqbalatma/gofortify"
)

// @title           Atma KMS API
// @version         1.0
// @description     Atma KMS API. Owns its accounts (email + password, with Google login as an alternative), issues and validates its own JWT, and serves the RBAC and operational resources behind them.
// @BasePath        /api
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Go JWT access token, as "Bearer <token>". Obtained from /auth/login, the Google flow (/auth/google/redirect), or, in local dev, /auth/dev-login.
//
// @securityDefinitions.apikey M2MClientID
// @in header
// @name X-Client-Id
// @description Machine-to-machine client id (m2m_clients.id, the row's uuid — not its display name), paired with X-Client-Secret.
//
// @securityDefinitions.apikey M2MClientSecret
// @in header
// @name X-Client-Secret
// @description Machine-to-machine client secret, paired with X-Client-Id.
func main() {
	config.LoadEnv()
	config.LoadLogger()
	config.LoadSentry()
	defer config.FlushSentry()
	config.ConnectDB()
	config.ConnectRDB()

	gofortify.LoadJWTConfig()
	validator.RegisterUniqueColumnValidator()
	watchCacheClearSignal()

	runServer()
}

// watchCacheClearSignal drops every cached lookup when the process is sent
// SIGHUP:
//
//	docker compose kill -s HUP web
//
// It has to be a signal rather than a binary under cmd/: the cache is a map in
// this process (utils.Remember), so a second process could only ever clear its
// own empty copy. SIGHUP is the conventional "reload without restarting" signal
// and nothing else here uses it; SIGINT and SIGTERM keep their default meaning
// so deploys still stop the container cleanly.
//
// Restarting the container clears the cache too, but it also drops every open
// notification and activitylog WebSocket. This does not.
func watchCacheClearSignal() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)

	go func() {
		for range signals {
			utils.Forget("")
			config.AppLogger.Info("cache cleared on SIGHUP")
		}
	}()
}

func runServer() {
	// Shared with the container below so LoggerMiddleware's broadcasts reach
	// the WebSocket viewers ActivityLogHandler registers on the same hub.
	activityLogHub := activitylog.NewHub()
	// Same idea, for the scheduler runner started below and JobLogHandler's
	// WebSocket route.
	jobLogHub := joblog.NewHub()
	// Same idea again, for the queue worker started below and QueueHandler's
	// WebSocket route — a separate feed from jobLogHub since it pushes on
	// every state transition (enqueue, claim, finish), not just a finished
	// attempt.
	queueHub := queue.NewHub()
	// Same idea again, for AuditHandler's WebSocket route — the GormStorer
	// below broadcasts on this hub itself (unlike the others, there's no
	// separate middleware/worker producing audit rows, go-audit's own
	// Execute() is what calls Storer.Save()).
	auditHub := auditpkg.NewHub()
	goaudit.Configure(auditpkg.NewGormStorer(config.DB, config.AppLogger, auditHub), "api-kms")

	router := gin.New()
	router.
		Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
			// sentrygin (below) already captured and repanicked; this just
			// shapes the response.
			c.AbortWithStatusJSON(http.StatusInternalServerError, utils.NewHttpError(
				"Internal server error",
				enums.ERR_INTERNAL_SERVER_ERROR,
				nil,
			))
		})).
		// Binds a per-request hub to the context carrying method/URL/headers,
		// so every downstream CaptureException (ErrorHandler) reports with
		// that request attached. Repanic: true so it re-raises after
		// capturing instead of writing its own response, leaving that to the
		// CustomRecovery handler above.
		Use(sentrygin.New(sentrygin.Options{Repanic: true})).
		Use(middleware2.LoggerMiddleware(activityLogHub)).
		Use(middleware2.CorsMiddleware()).
		Use(middleware2.ErrorHandler())

	container := routes.NewContainer(config.DB, config.RDB, activityLogHub, jobLogHub, queueHub, auditHub)
	routes.RegisterRoute(router, container)

	container.SchedulerRunner.Start(context.Background())
	container.QueueWorker.Start(context.Background())

	err := router.Run(":" + config.AppConfig.AppPort)
	if err != nil {
		return
	}
}
