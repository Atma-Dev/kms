package routes

import (
	"net/http"

	"api-kms/app/document"
	"api-kms/app/lookup"
	"api-kms/enums"
	middleware2 "api-kms/middleware"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func ErrorHandleWrapper(h func(*gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c); err != nil {
			c.Error(err)
			c.Abort()
		}
	}
}

func RegisterRoute(router *gin.Engine, c *Container) {
	router.GET("/up", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, utils.NewHttpError("Route does not exist", enums.ERR_ROUTE_NOT_FOUND, nil))
	})

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api")
	protected := api.Group("", middleware2.AuthMiddleware(), middleware2.AuditContextMiddleware())

	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", ErrorHandleWrapper(c.AuthHandler.Login))
		google := authGroup.Group("/google")
		google.GET("/redirect", ErrorHandleWrapper(c.AuthHandler.GoogleRedirect))
		google.GET("/callback", ErrorHandleWrapper(c.AuthHandler.GoogleCallback))
		authGroup.POST("/logout", middleware2.AuthMiddleware(), ErrorHandleWrapper(c.AuthHandler.Logout))
		authGroup.POST("/ws-token", middleware2.AuthMiddleware(), ErrorHandleWrapper(c.AuthHandler.WSToken))

		authGroup.GET("/dev-login", ErrorHandleWrapper(c.AuthHandler.DevLogin))
		refresh := authGroup.Group("", middleware2.RefreshMiddleware())
		refresh.POST("/refresh", ErrorHandleWrapper(c.AuthHandler.Refresh))
	}

	managementGroup := protected.Group("/management")
	{
		managementGroup.GET("/permissions", middleware2.RequirePermission(enums.PermissionManagementPermissionsShow), ErrorHandleWrapper(c.RBACHandler.PermissionIndex))
		managementGroup.GET("/roles", middleware2.RequirePermission(enums.PermissionManagementRolesShow), ErrorHandleWrapper(c.RBACHandler.RoleIndex))
		managementGroup.POST("/roles", middleware2.RequirePermission(enums.PermissionManagementRolesStore), ErrorHandleWrapper(c.RBACHandler.RoleStore))
		managementGroup.GET("/roles/:id", middleware2.RequirePermission(enums.PermissionManagementRolesShow), ErrorHandleWrapper(c.RBACHandler.RoleShow))
		managementGroup.PATCH("/roles/:id", middleware2.RequirePermission(enums.PermissionManagementRolesUpdate), ErrorHandleWrapper(c.RBACHandler.RoleUpdate))
		managementGroup.DELETE("/roles/:id", middleware2.RequirePermission(enums.PermissionManagementRolesDestroy), ErrorHandleWrapper(c.RBACHandler.RoleDestroy))
		managementGroup.GET("/users", middleware2.RequirePermission(enums.PermissionManagementUsersShow), ErrorHandleWrapper(c.UserHandler.Index))
		managementGroup.PATCH("/users/:id", middleware2.RequirePermission(enums.PermissionManagementUsersUpdate), ErrorHandleWrapper(c.UserHandler.Update))
		// Modul 4 FR-4, read by person rather than by document. Under
		// /management like every other account screen, not at the PRD's
		// /users/{id}/access-log — the account pages all live here, and one
		// route somewhere else is the one nobody thinks to secure the same way.
		// Handled by app/document, which owns the other half of the same trail.
		managementGroup.GET("/users/:id/access-log", middleware2.RequirePermission(enums.PermissionDocumentsAccessLogShow), ErrorHandleWrapper(c.DocumentHandler.UserAccessLog))

		// No handler on the container: see routes/cache.go.
		managementGroup.GET("/cache", middleware2.RequirePermission(enums.PermissionManagementCacheShow), ErrorHandleWrapper(listCache))
		managementGroup.DELETE("/cache", middleware2.RequirePermission(enums.PermissionManagementCacheDestroy), ErrorHandleWrapper(clearCache))

		managementGroup.GET("/m2m-clients", middleware2.RequirePermission(enums.PermissionManagementM2MClientsShow), ErrorHandleWrapper(c.M2MClientHandler.Index))
		managementGroup.POST("/m2m-clients", middleware2.RequirePermission(enums.PermissionManagementM2MClientsStore), ErrorHandleWrapper(c.M2MClientHandler.Store))
		managementGroup.GET("/m2m-clients/:id", middleware2.RequirePermission(enums.PermissionManagementM2MClientsShow), ErrorHandleWrapper(c.M2MClientHandler.Show))
		managementGroup.PATCH("/m2m-clients/:id", middleware2.RequirePermission(enums.PermissionManagementM2MClientsUpdate), ErrorHandleWrapper(c.M2MClientHandler.Update))
		// Reuses the update permission: regenerating a secret changes the
		// client's credential material, the same category of action as renaming it.
		managementGroup.POST("/m2m-clients/:id/regenerate-secret", middleware2.RequirePermission(enums.PermissionManagementM2MClientsUpdate), ErrorHandleWrapper(c.M2MClientHandler.RegenerateSecret))
		managementGroup.DELETE("/m2m-clients/:id", middleware2.RequirePermission(enums.PermissionManagementM2MClientsDestroy), ErrorHandleWrapper(c.M2MClientHandler.Destroy))

		// The HTTP-request log (app/activitylog). The user_id filter is fed by
		// the users lookup, so it answers to that lookup's permission —
		// present-but-unpermitted is refused rather than dropped, or the
		// caller would get everyone's activity back believing it was one user's.
		managementGroup.GET("/activity-logs",
			middleware2.RequirePermission(enums.PermissionManagementActivityLogsShow),
			middleware2.RequirePermissionForQueryParam("user_id", enums.PermissionLookupsUsersShow),
			ErrorHandleWrapper(c.ActivityLogHandler.Index),
		)

		// app/audit's field-level change trail (who changed what, from what to what).
		managementGroup.GET("/audits",
			middleware2.RequirePermission(enums.PermissionManagementAuditsShow),
			ErrorHandleWrapper(c.AuditHandler.Index),
		)

		managementGroup.GET("/job-logs", middleware2.RequirePermission(enums.PermissionManagementJobLogsShow), ErrorHandleWrapper(c.JobLogHandler.Index))
		managementGroup.GET("/job-logs/definitions", middleware2.RequirePermission(enums.PermissionManagementJobLogsShow), ErrorHandleWrapper(c.JobLogHandler.Definitions))
		managementGroup.GET("/job-logs/stats", middleware2.RequirePermission(enums.PermissionManagementJobLogsShow), ErrorHandleWrapper(c.JobLogHandler.Stats))

		managementGroup.GET("/schedules", middleware2.RequirePermission(enums.PermissionManagementSchedulesShow), ErrorHandleWrapper(c.ScheduleHandler.Index))
		managementGroup.PATCH("/schedules/:id", middleware2.RequirePermission(enums.PermissionManagementSchedulesManage), ErrorHandleWrapper(c.ScheduleHandler.Update))
		managementGroup.POST("/schedules/:id/run", middleware2.RequirePermission(enums.PermissionManagementSchedulesManage), ErrorHandleWrapper(c.ScheduleHandler.RunNow))

		managementGroup.GET("/queue/jobs", middleware2.RequirePermission(enums.PermissionManagementQueueShow), ErrorHandleWrapper(c.QueueHandler.Index))
		managementGroup.GET("/queue/stats", middleware2.RequirePermission(enums.PermissionManagementQueueShow), ErrorHandleWrapper(c.QueueHandler.Stats))
		// Manage-type actions (retry/cancel/purge) are gated by a separate
		// *.manage permission from *.show: a retry re-runs whatever the job
		// does, which isn't the same capability as looking at the list.
		managementGroup.POST("/queue/jobs/:id/retry", middleware2.RequirePermission(enums.PermissionManagementQueueManage), ErrorHandleWrapper(c.QueueHandler.Retry))
		managementGroup.POST("/queue/jobs/:id/cancel", middleware2.RequirePermission(enums.PermissionManagementQueueManage), ErrorHandleWrapper(c.QueueHandler.Cancel))
		managementGroup.POST("/queue/jobs/retry-failed", middleware2.RequirePermission(enums.PermissionManagementQueueManage), ErrorHandleWrapper(c.QueueHandler.RetryFailed))
		managementGroup.DELETE("/queue/jobs", middleware2.RequirePermission(enums.PermissionManagementQueueManage), ErrorHandleWrapper(c.QueueHandler.Purge))
	}
	// Its own auth path, same reason as /notifications/ws: a WebSocket
	// handshake can't carry a custom Authorization header, so the token
	// travels as ?token= (see middleware.WSTokenAuthMiddleware). Preloads
	// Roles.Permissions itself since RequirePermission reads it from context
	// with no query of its own.
	api.GET("/management/activity-logs/ws",
		middleware2.WSTokenAuthMiddleware("Roles.Permissions"),
		middleware2.RequirePermission(enums.PermissionManagementActivityLogsShow),
		ErrorHandleWrapper(c.ActivityLogHandler.Serve),
	)
	api.GET("/management/job-logs/ws",
		middleware2.WSTokenAuthMiddleware("Roles.Permissions"),
		middleware2.RequirePermission(enums.PermissionManagementJobLogsShow),
		ErrorHandleWrapper(c.JobLogHandler.Serve),
	)
	api.GET("/management/queue/ws",
		middleware2.WSTokenAuthMiddleware("Roles.Permissions"),
		middleware2.RequirePermission(enums.PermissionManagementQueueShow),
		ErrorHandleWrapper(c.QueueHandler.Serve),
	)
	api.GET("/management/audits/ws",
		middleware2.WSTokenAuthMiddleware("Roles.Permissions"),
		middleware2.RequirePermission(enums.PermissionManagementAuditsShow),
		ErrorHandleWrapper(c.AuditHandler.Serve),
	)

	// Select-option dropdowns, one route per resource so each keeps its own
	// lookups.* permission visible here alongside every other gate. These are
	// deliberately not the *.show permission of the underlying resource: a
	// dropdown of user names is a smaller capability than managing users.
	lookupGroup := protected.Group("/lookups")
	{
		lookupGroup.GET("/users", middleware2.RequirePermission(enums.PermissionLookupsUsersShow), ErrorHandleWrapper(c.LookupHandler.Get(lookup.ResourceUsers)))
		lookupGroup.GET("/roles", middleware2.RequirePermission(enums.PermissionLookupsRolesShow), ErrorHandleWrapper(c.LookupHandler.Get(lookup.ResourceRoles)))
		lookupGroup.GET("/permissions", middleware2.RequirePermission(enums.PermissionLookupsPermissionsShow), ErrorHandleWrapper(c.LookupHandler.Get(lookup.ResourcePermissions)))
		lookupGroup.GET("/spaces", middleware2.RequirePermission(enums.PermissionLookupsSpacesShow), ErrorHandleWrapper(c.LookupHandler.Get(lookup.ResourceSpaces)))
		lookupGroup.GET("/categories", middleware2.RequirePermission(enums.PermissionLookupsCategoriesShow), ErrorHandleWrapper(c.LookupHandler.Get(lookup.ResourceCategories)))
		lookupGroup.GET("/tags", middleware2.RequirePermission(enums.PermissionLookupsTagsShow), ErrorHandleWrapper(c.LookupHandler.Get(lookup.ResourceTags)))
	}

	// Spaces are the top-level container. DELETE archives rather than removes:
	// a space owns documents, and dropping the row would take them with it.
	spaceGroup := protected.Group("/spaces")
	{
		spaceGroup.GET("", middleware2.RequirePermission(enums.PermissionMastersSpacesShow), ErrorHandleWrapper(c.SpaceHandler.Index))
		spaceGroup.POST("", middleware2.RequirePermission(enums.PermissionMastersSpacesStore), ErrorHandleWrapper(c.SpaceHandler.Store))
		spaceGroup.GET("/:spaceId", middleware2.RequirePermission(enums.PermissionMastersSpacesShow), ErrorHandleWrapper(c.SpaceHandler.Show))
		spaceGroup.PUT("/:spaceId", middleware2.RequirePermission(enums.PermissionMastersSpacesUpdate), ErrorHandleWrapper(c.SpaceHandler.Update))
		spaceGroup.DELETE("/:spaceId", middleware2.RequirePermission(enums.PermissionMastersSpacesDestroy), ErrorHandleWrapper(c.SpaceHandler.Destroy))

		// The category tree of one space. Reorder sits at its own path segment
		// and takes the whole ordered sibling list — see app/category.
		spaceGroup.GET("/:spaceId/categories", middleware2.RequirePermission(enums.PermissionMastersCategoriesShow), ErrorHandleWrapper(c.CategoryHandler.Index))
		spaceGroup.POST("/:spaceId/categories", middleware2.RequirePermission(enums.PermissionMastersCategoriesStore), ErrorHandleWrapper(c.CategoryHandler.Store))
		spaceGroup.PATCH("/:spaceId/categories/order", middleware2.RequirePermission(enums.PermissionMastersCategoriesUpdate), ErrorHandleWrapper(c.CategoryHandler.Reorder))

		spaceGroup.GET("/:spaceId/documents", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Index))
		// Registered before :id-shaped routes would matter, and gated harder
		// than reading: the bin is only useful to whoever may delete.
		spaceGroup.GET("/:spaceId/documents/deleted", middleware2.RequirePermission(enums.PermissionDocumentsDestroy), ErrorHandleWrapper(c.DocumentHandler.Deleted))
		spaceGroup.POST("/:spaceId/documents", middleware2.RequirePermission(enums.PermissionDocumentsStore), ErrorHandleWrapper(c.DocumentHandler.Store))

		// Modul 4 FR-6: the default every document in the space inherits.
		// There is no matching GET — the space detail already carries
		// protection_level.
		spaceGroup.PUT("/:spaceId/protection-setting", middleware2.RequirePermission(enums.PermissionDocumentsProtectionManage), ErrorHandleWrapper(c.SpaceHandler.ProtectionSetting))
	}

	// Categories are edited by id, not through their space: the client already
	// holds the id from the tree, and routing through the space would let the
	// two disagree about which space a category is in.
	categoryGroup := protected.Group("/categories")
	{
		categoryGroup.PUT("/:id", middleware2.RequirePermission(enums.PermissionMastersCategoriesUpdate), ErrorHandleWrapper(c.CategoryHandler.Update))
		categoryGroup.DELETE("/:id", middleware2.RequirePermission(enums.PermissionMastersCategoriesDestroy), ErrorHandleWrapper(c.CategoryHandler.Destroy))
	}

	// A document by id, plus its lifecycle and its versions.
	//
	// Every lifecycle action gets its own route so the permission behind it is
	// visible here rather than hidden in a switch. submit-review deliberately
	// has no RequirePermission: it belongs to whoever may edit the document,
	// which is decided per row in app/document (author vs any) and cannot be
	// expressed as a route-level grant.
	documentGroup := protected.Group("/documents")
	{
		documentGroup.GET("/:id", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Show))
		documentGroup.PUT("/:id", middleware2.RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll), ErrorHandleWrapper(c.DocumentHandler.Update))
		documentGroup.DELETE("/:id", middleware2.RequirePermission(enums.PermissionDocumentsDestroy), ErrorHandleWrapper(c.DocumentHandler.Destroy))

		documentGroup.POST("/:id/submit-review", ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionSubmitReview)))
		documentGroup.POST("/:id/approve", middleware2.RequirePermission(enums.PermissionDocumentsReview), ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionApprove)))
		documentGroup.POST("/:id/reject", middleware2.RequirePermission(enums.PermissionDocumentsReview), ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionReject)))
		documentGroup.POST("/:id/publish", middleware2.RequirePermission(enums.PermissionDocumentsReview), ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionPublish)))
		documentGroup.POST("/:id/unpublish", middleware2.RequirePermission(enums.PermissionDocumentsReview), ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionUnpublish)))
		documentGroup.POST("/:id/archive", middleware2.RequirePermission(enums.PermissionDocumentsArchive), ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionArchive)))
		documentGroup.POST("/:id/restore", middleware2.RequirePermission(enums.PermissionDocumentsArchive), ErrorHandleWrapper(c.DocumentHandler.Transition(document.ActionRestore)))

		// compare is registered before :versionNumber so the static segment
		// wins the match — the same shape as /queue/jobs/retry-failed.
		documentGroup.GET("/:id/versions/compare", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Compare))
		documentGroup.GET("/:id/versions", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Versions))
		documentGroup.GET("/:id/versions/:versionNumber", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Version))

		// Rolling back rewrites the live content, so it answers to the edit
		// permission, not to documents.show.
		documentGroup.POST("/:id/versions/:versionNumber/rollback", middleware2.RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll), ErrorHandleWrapper(c.DocumentHandler.Rollback))

		// Retagging is an edit of the document, so no tag permission gates it
		// — app/document checks the row's own edit rule.
		documentGroup.POST("/:id/tags", middleware2.RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll), ErrorHandleWrapper(c.DocumentHandler.SyncTags))

		// Pinning curates what the space leads with, which is the same
		// judgement as editing what it says.
		documentGroup.PATCH("/:id/pin", middleware2.RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll), ErrorHandleWrapper(c.DocumentHandler.Pin))

		// Undoing a delete answers to the permission that did it.
		documentGroup.POST("/:id/undelete", middleware2.RequirePermission(enums.PermissionDocumentsDestroy), ErrorHandleWrapper(c.DocumentHandler.Undelete))

		// Related sits with the document rather than under /search: what it
		// needs — the tags, the space, the access rule — all belongs to
		// app/document, and it answers to reading a document like the rest of
		// this group.
		documentGroup.GET("/:id/related", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Related))

		documentGroup.GET("/:id/attachments", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.AttachmentHandler.Index))
		documentGroup.POST("/:id/attachments", middleware2.RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll), ErrorHandleWrapper(c.AttachmentHandler.Store))

		// Who may reach a restricted document is gated on managing access, not
		// on reading it: the list names people, and a document's allowlist
		// should not leak to everyone who can open it.
		// Modul 4. Rendering a document is reading it, so the export and the
		// page images answer to documents.show like the detail does — the
		// watermark is what makes handing someone a copy safe, not a further
		// permission.
		documentGroup.GET("/:id/export-pdf", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.ExportPDF))
		documentGroup.GET("/:id/pages", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.Pages))

		// Changing how hard a document defends itself is not an edit of it:
		// whoever may rewrite the text is not automatically whoever decides
		// how far it travels.
		documentGroup.PUT("/:id/protection-setting", middleware2.RequirePermission(enums.PermissionDocumentsProtectionManage), ErrorHandleWrapper(c.DocumentHandler.ProtectionSetting))

		// The trail is a record of what people read, so reading it is gated
		// well above reading the document. Writing to it is not: the POST only
		// accepts `print`, which is the one access the server cannot observe,
		// and it answers to documents.show because reporting it is part of
		// having the document open.
		documentGroup.GET("/:id/access-log", middleware2.RequirePermission(enums.PermissionDocumentsAccessLogShow), ErrorHandleWrapper(c.DocumentHandler.AccessLog))
		documentGroup.POST("/:id/access-log", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.DocumentHandler.ReportAccess))

		// Share links reach people with no account at all, which is a bigger
		// act than reading — and listing them is gated the same as creating
		// them, since the list is what tells you a link you did not make
		// exists.
		documentGroup.GET("/:id/view-sessions", middleware2.RequirePermission(enums.PermissionDocumentsShare), ErrorHandleWrapper(c.DocumentHandler.ViewSessions))
		documentGroup.POST("/:id/view-sessions", middleware2.RequirePermission(enums.PermissionDocumentsShare), ErrorHandleWrapper(c.DocumentHandler.CreateViewSession))
		documentGroup.DELETE("/:id/view-sessions/:sessionId", middleware2.RequirePermission(enums.PermissionDocumentsShare), ErrorHandleWrapper(c.DocumentHandler.RevokeViewSession))

		documentGroup.GET("/:id/access", middleware2.RequirePermission(enums.PermissionDocumentsAccessManage), ErrorHandleWrapper(c.DocumentHandler.Access))
		documentGroup.POST("/:id/access", middleware2.RequirePermission(enums.PermissionDocumentsAccessManage), ErrorHandleWrapper(c.DocumentHandler.StoreAccess))
		documentGroup.DELETE("/:id/access/:accessId", middleware2.RequirePermission(enums.PermissionDocumentsAccessManage), ErrorHandleWrapper(c.DocumentHandler.DestroyAccess))
	}

	// Modul 3. Searching is reading documents, so it answers to documents.show
	// rather than a search permission of its own — a second permission naming
	// the same capability is one more row to seed and one more to forget.
	//
	// The history routes carry no permission at all. They only ever touch the
	// caller's own rows, exactly like PATCH /profiles/password.
	searchGroup := protected.Group("/search")
	{
		searchGroup.GET("", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.SearchHandler.Search))
		searchGroup.GET("/suggest", middleware2.RequirePermission(enums.PermissionDocumentsShow), ErrorHandleWrapper(c.SearchHandler.Suggest))

		searchGroup.GET("/history", ErrorHandleWrapper(c.SearchHandler.History))
		searchGroup.DELETE("/history/:id", ErrorHandleWrapper(c.SearchHandler.DestroyHistory))
	}

	// The tag master. Tagging a document does not come through here — see the
	// documents group above.
	tagGroup := protected.Group("/tags")
	{
		tagGroup.GET("", middleware2.RequirePermission(enums.PermissionMastersTagsShow), ErrorHandleWrapper(c.TagHandler.Index))
		tagGroup.POST("", middleware2.RequirePermission(enums.PermissionMastersTagsStore), ErrorHandleWrapper(c.TagHandler.Store))
		tagGroup.PUT("/:id", middleware2.RequirePermission(enums.PermissionMastersTagsUpdate), ErrorHandleWrapper(c.TagHandler.Update))
		tagGroup.DELETE("/:id", middleware2.RequirePermission(enums.PermissionMastersTagsDestroy), ErrorHandleWrapper(c.TagHandler.Destroy))
	}

	// Modul 4 FR-5. The one route in the application reachable with no account
	// at all, and deliberately the only one: the token opens one document and
	// is not a session on the API. Everything the response contains — which
	// document, whose name is on the watermark, how long it lasts — comes from
	// the row the token resolves to, never from the request.
	api.GET("/view-session/:token", ErrorHandleWrapper(c.DocumentHandler.OpenViewSession))

	// Files. Download is a handler and not a static mount on purpose: it asks
	// whether this caller may see the document first, which a static folder
	// cannot.
	// Download sits outside `protected` because it accepts `?token=` as well as
	// the Bearer header — an <img src> for an inline image and an <a download>
	// are issued by the browser, which attaches no Authorization header. The
	// access check inside the handler is unchanged either way.
	api.GET("/attachments/:id/download",
		middleware2.TokenOrHeaderAuthMiddleware("Roles.Permissions"),
		middleware2.RequirePermission(enums.PermissionDocumentsShow),
		ErrorHandleWrapper(c.AttachmentHandler.Download),
	)
	protected.DELETE("/attachments/:id", middleware2.RequireAnyPermission(enums.PermissionDocumentsUpdate, enums.PermissionDocumentsUpdateAll), ErrorHandleWrapper(c.AttachmentHandler.Destroy))

	// The editor's image upload (FR-8), filed against a space rather than a
	// document: the editor needs the URL while the document is still being
	// written, often before it has been saved at all.
	protected.POST("/uploads/inline-image", middleware2.RequirePermission(enums.PermissionDocumentsStore), ErrorHandleWrapper(c.AttachmentHandler.StoreInlineImage))

	// Machine-to-machine callers sit under one versioned prefix. Empty until
	// the first M2M route exists; RequireM2MScope is what keeps one client's
	// credential from reaching another's routes once they do.
	_ = api.Group("/v1/m2m", middleware2.M2MAuthMiddleware())

	profileGroup := protected.Group("/profiles")
	{
		profileGroup.GET("/me", ErrorHandleWrapper(c.ProfileHandler.Me))
		// Ungated like the rest of /profiles: it only ever acts on the caller's
		// own row, and it re-checks the current password itself.
		profileGroup.PATCH("/password", ErrorHandleWrapper(c.ProfileHandler.UpdatePassword))
	}

	notificationGroup := protected.Group("/notifications")
	{
		notificationGroup.GET("", ErrorHandleWrapper(c.NotificationHandler.Index))
		notificationGroup.PATCH("/:id/read", ErrorHandleWrapper(c.NotificationHandler.MarkRead))
		notificationGroup.PATCH("/read-all", ErrorHandleWrapper(c.NotificationHandler.MarkAllRead))
	}
	// Its own auth path: a WebSocket handshake can't carry a custom
	// Authorization header, so the token travels as ?token= instead (see
	// middleware.WSTokenAuthMiddleware).
	api.GET("/notifications/ws", middleware2.WSTokenAuthMiddleware(), ErrorHandleWrapper(c.NotificationHandler.Serve))
}
