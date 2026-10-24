package routes

import (
	"api-kms/app/activitylog"
	"api-kms/app/attachment"
	"api-kms/app/audit"
	"api-kms/app/auth"
	"api-kms/app/category"
	"api-kms/app/document"
	"api-kms/app/joblog"
	"api-kms/app/lookup"
	"api-kms/app/m2mclient"
	"api-kms/app/notification"
	"api-kms/app/protection"
	"api-kms/app/queue"
	"api-kms/app/rbac"
	"api-kms/app/scheduler"
	"api-kms/app/search"
	"api-kms/app/space"
	"api-kms/app/tag"
	"api-kms/app/user"
	"api-kms/config"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	AuthHandler         *auth.Handler
	RBACHandler         *rbac.Handler
	UserHandler         *user.Handler
	ProfileHandler      *user.ProfileHandler
	LookupHandler       *lookup.Handler
	M2MClientHandler    *m2mclient.Handler
	NotificationHandler *notification.Handler
	ActivityLogHandler  *activitylog.Handler
	AuditHandler        *audit.Handler
	JobLogHandler       *joblog.Handler
	ScheduleHandler     *scheduler.Handler
	QueueHandler        *queue.Handler

	SpaceHandler      *space.Handler
	CategoryHandler   *category.Handler
	DocumentHandler   *document.Handler
	TagHandler        *tag.Handler
	AttachmentHandler *attachment.Handler
	SearchHandler     *search.Handler

	// Exposed for main.go to wire app/scheduler's jobs and app/queue's
	// worker without constructing a second copy of the repository/service
	// this container already built.
	UserService     *user.Service
	QueueRepository *queue.Repository

	// SchedulerRunner/QueueWorker are fully built here (see NewContainer)
	// since the scheduler's job list needs the queue's maintenance jobs;
	// main.go just calls Start on each once the server is set up.
	SchedulerRunner *scheduler.Runner
	QueueWorker     *queue.Worker
}

// activityLogHub is shared with middleware.LoggerMiddleware, which is the
// only thing that ever broadcasts to it — this container just needs the same
// instance so the WebSocket handler registers viewers on it. jobLogHub is
// shared the same way with the scheduler.Runner main.go starts, and
// queueHub with the queue.Worker main.go starts.
func NewContainer(db *gorm.DB, rdb *redis.Client, activityLogHub *activitylog.Hub, jobLogHub *joblog.Hub, queueHub *queue.Hub, auditHub *audit.Hub) *Container {
	// repo
	permissionRepository := rbac.NewPermissionRepository(db)
	roleRepository := rbac.NewRoleRepository(db)
	userRepository := user.NewRepository(db)
	m2mClientRepository := m2mclient.NewRepository(db)
	notificationRepository := notification.NewRepository(db)
	notificationHub := notification.NewHub()
	activityLogRepository := activitylog.NewRepository(db)
	auditRepository := audit.NewRepository(db)
	spaceRepository := space.NewRepository(db)
	categoryRepository := category.NewRepository(db)
	documentRepository := document.NewRepository(db)
	attachmentRepository := attachment.NewRepository(db)
	tagRepository := tag.NewRepository(db)
	searchRepository := search.NewRepository(db)
	protectionRepository := protection.NewRepository(db)

	// service
	rbacService := rbac.NewService(permissionRepository, roleRepository, notificationHub)
	userService := user.NewService(userRepository, notificationHub)
	notificationService := notification.NewService(notificationRepository)
	lookupService := lookup.NewService(userRepository, roleRepository, permissionRepository, spaceRepository, categoryRepository, tagRepository)
	spaceService := space.NewService(spaceRepository)
	tagService := tag.NewService(tagRepository)
	// Modul 4. app/protection stores the access trail and the share links and
	// knows nothing about documents; app/document is what decides who may do
	// any of it, so the dependency runs one way only.
	protectionService := protection.NewService(
		protectionRepository,
		config.AppLogger,
		time.Duration(config.AppConfig.ViewSessionMaxTTLHours)*time.Hour,
	)
	documentService := document.NewService(documentRepository, tagService, protectionService)
	// After documentService: the category tree's per-node counts exclude the
	// documents FR-9 hides from this caller, which only app/document can say.
	// After documentService too: search hides the documents FR-9 hides, and
	// asks app/document which those are rather than restating the rule.
	// The same repository twice: it is the MySQL Engine and the history
	// store, which are separate arguments precisely so the first can be
	// swapped for another index without touching the second.
	searchService := search.NewService(searchRepository, searchRepository, documentService)
	categoryService := category.NewService(categoryRepository, documentService)
	attachmentService := attachment.NewService(attachmentRepository, documentService)
	m2mClientService := m2mclient.NewService(m2mClientRepository)
	activityLogService := activitylog.NewService(activityLogRepository)
	auditService := audit.NewService(auditRepository)
	jobLogRepository := joblog.NewRepository(db)
	jobLogService := joblog.NewService(jobLogRepository)
	jobLogBroadcaster := joblog.NewBroadcaster(jobLogRepository, jobLogHub)

	// Queue stack. queueRegistry starts empty — see app/queue's package doc
	// for how to add a job handler; the queue exists as ready infrastructure
	// even before anything is registered.
	queueRepository := queue.NewRepository(db)
	queueRegistry := queue.NewRegistry()
	queueNotifier := queue.NewNotifier(rdb)
	queueDispatcher := queue.NewDispatcher(queueRepository, queueRegistry, queueHub, queueNotifier)
	queueService := queue.NewService(queueRepository, queueRegistry, queueHub, queueNotifier)
	queueWorker := queue.NewWorker(queueRepository, queueRegistry, jobLogBroadcaster, queueHub, queueNotifier, queue.WorkerConfig{
		Concurrency:  config.AppConfig.QueueWorkers,
		Queues:       config.AppConfig.QueueNames,
		PollInterval: time.Duration(config.AppConfig.QueuePollIntervalSeconds) * time.Second,
		MaxRunTime:   time.Duration(config.AppConfig.QueueMaxRunTimeSeconds) * time.Second,
		BackoffBase:  time.Duration(config.AppConfig.QueueBackoffBaseSeconds) * time.Second,
		BackoffMax:   time.Duration(config.AppConfig.QueueBackoffMaxSeconds) * time.Second,
	})
	_ = queueDispatcher // held for the first queue.JobHandler caller; not yet exposed since nothing dispatches to the queue today

	// Scheduler. Built here (rather than in main.go) because its job list
	// includes app/queue's own maintenance jobs, which need queueWorker —
	// everything the runner needs already exists at this point in
	// NewContainer.
	scheduleRepository := scheduler.NewRepository(db)
	schedulerRunner := scheduler.NewRunner(
		scheduleRepository,
		jobLogBroadcaster,
		time.Duration(config.AppConfig.ScheduleLeaseTTLMinutes)*time.Minute,
		scheduler.NewPruneJobLogsJob(jobLogRepository),
		queue.NewPruneJobsJob(queueRepository, time.Duration(config.AppConfig.QueueRetentionDays)*24*time.Hour),
		queue.NewReleaseStaleJobsJob(queueWorker),
		attachment.NewPruneOrphanUploadsJob(attachmentRepository),
		search.NewPruneHistoriesJob(searchRepository),
		protection.NewPruneViewSessionsJob(protectionRepository),
	)
	scheduleService := scheduler.NewService(scheduleRepository, schedulerRunner)

	jobLogDefinitions := []joblog.Definition{
		{Type: joblog.TypeSchedule, Name: scheduler.PruneJobLogsJobName, Interval: "0 3 * * *"},
		{Type: joblog.TypeQueue, Name: queue.PruneJobsJobName, Interval: "30 3 * * *"},
		{Type: joblog.TypeQueue, Name: queue.ReleaseStaleJobName, Interval: "*/5 * * * *"},
		{Type: joblog.TypeSchedule, Name: attachment.PruneOrphanUploadsJobName, Interval: "15 * * * *"},
		{Type: joblog.TypeSchedule, Name: search.PruneHistoriesJobName, Interval: "45 3 * * *"},
		{Type: joblog.TypeSchedule, Name: protection.PruneViewSessionsJobName, Interval: "50 3 * * *"},
	}
	return &Container{
		AuthHandler:         auth.NewHandler(),
		RBACHandler:         rbac.NewHandler(rbacService),
		UserHandler:         user.NewHandler(userService),
		ProfileHandler:      user.NewProfileHandler(userService),
		LookupHandler:       lookup.NewHandler(lookupService),
		M2MClientHandler:    m2mclient.NewHandler(m2mClientService),
		NotificationHandler: notification.NewHandler(notificationService, notificationHub),
		ActivityLogHandler:  activitylog.NewHandler(activityLogService, activityLogHub),
		AuditHandler:        audit.NewHandler(auditService, auditHub),
		JobLogHandler:       joblog.NewHandler(jobLogService, jobLogHub, jobLogDefinitions),
		ScheduleHandler:     scheduler.NewHandler(scheduleService),
		QueueHandler:        queue.NewHandler(queueService, queueHub),

		SpaceHandler:      space.NewHandler(spaceService),
		CategoryHandler:   category.NewHandler(categoryService),
		DocumentHandler:   document.NewHandler(documentService),
		TagHandler:        tag.NewHandler(tagService),
		AttachmentHandler: attachment.NewHandler(attachmentService),
		SearchHandler:     search.NewHandler(searchService),

		UserService:     userService,
		QueueRepository: queueRepository,
		SchedulerRunner: schedulerRunner,
		QueueWorker:     queueWorker,
	}
}
