package enums

type Permission string

const (
	// management.users.store and management.users.destroy are deliberately
	// absent: this app has no endpoint for either yet — accounts are seeded or
	// created out of band, and nothing deletes one.
	PermissionManagementUsersShow   Permission = "management.users.show"
	PermissionManagementUsersUpdate Permission = "management.users.update"

	PermissionManagementPermissionsShow Permission = "management.permissions.show"

	PermissionManagementRolesShow    Permission = "management.roles.show"
	PermissionManagementRolesStore   Permission = "management.roles.store"
	PermissionManagementRolesUpdate  Permission = "management.roles.update"
	PermissionManagementRolesDestroy Permission = "management.roles.destroy"

	PermissionManagementConfigurationsShow   Permission = "management.configurations.show"
	PermissionManagementConfigurationsUpdate Permission = "management.configurations.update"

	// PermissionManagementActivityLogsShow gates the HTTP request log
	// (app/activitylog); PermissionManagementAuditsShow gates app/audit's
	// field-change trail (who changed what, from what to what).
	PermissionManagementActivityLogsShow Permission = "management.activity-logs.show"
	PermissionManagementAuditsShow       Permission = "management.audits.show"

	PermissionManagementJobLogsShow Permission = "management.job-logs.show"

	PermissionManagementQueueShow   Permission = "management.queue.show"
	PermissionManagementQueueManage Permission = "management.queue.manage"

	PermissionManagementSchedulesShow   Permission = "management.schedules.show"
	PermissionManagementSchedulesManage Permission = "management.schedules.manage"

	// Every write in this app already drops the lookup cache, so these are the
	// escape hatch for the one case that cannot: another app changing a shared table.
	PermissionManagementCacheShow    Permission = "management.cache.show"
	PermissionManagementCacheDestroy Permission = "management.cache.destroy"

	PermissionManagementM2MClientsShow    Permission = "management.m2m-clients.show"
	PermissionManagementM2MClientsStore   Permission = "management.m2m-clients.store"
	PermissionManagementM2MClientsUpdate  Permission = "management.m2m-clients.update"
	PermissionManagementM2MClientsDestroy Permission = "management.m2m-clients.destroy"

	// Lookups back the select-option dropdowns (id + name only). They are
	// deliberately separate from the *.show permission of the resource they
	// read: "can see the list of user names in a filter" is a far smaller
	// capability than "can manage users".
	//
	// One per resource rather than a single lookups.show, so a role can be
	// given exactly the dropdowns its pages need.
	PermissionLookupsUsersShow       Permission = "lookups.users.show"
	PermissionLookupsRolesShow       Permission = "lookups.roles.show"
	PermissionLookupsPermissionsShow Permission = "lookups.permissions.show"
	PermissionLookupsSpacesShow      Permission = "lookups.spaces.show"
	PermissionLookupsCategoriesShow  Permission = "lookups.categories.show"
	PermissionLookupsTagsShow        Permission = "lookups.tags.show"

	PermissionMastersSpacesShow    Permission = "masters.spaces.show"
	PermissionMastersSpacesStore   Permission = "masters.spaces.store"
	PermissionMastersSpacesUpdate  Permission = "masters.spaces.update"
	PermissionMastersSpacesDestroy Permission = "masters.spaces.destroy"

	// Categories are a tree inside a space, not a resource of their own — but
	// they are maintained by whoever curates the space's structure, which is
	// not always whoever may rename the space itself.
	PermissionMastersCategoriesShow    Permission = "masters.categories.show"
	PermissionMastersCategoriesStore   Permission = "masters.categories.store"
	PermissionMastersCategoriesUpdate  Permission = "masters.categories.update"
	PermissionMastersCategoriesDestroy Permission = "masters.categories.destroy"

	PermissionDocumentsShow  Permission = "documents.show"
	PermissionDocumentsStore Permission = "documents.store"

	// The PRD's Contributor/Editor split. documents.update is "edit what I
	// wrote"; documents.update.all is "edit anyone's". Holding .all implies
	// the narrower one, so a role is given whichever single grant matches what
	// it is — see user.User.CanEditDocument.
	PermissionDocumentsUpdate    Permission = "documents.update"
	PermissionDocumentsUpdateAll Permission = "documents.update.all"

	PermissionDocumentsDestroy Permission = "documents.destroy"

	// Reviewer and up: approve, reject, publish, unpublish. Deliberately one
	// permission and not four — the PRD grants them to the same roles, and a
	// reviewer who may approve but not publish cannot finish the job they were
	// given.
	PermissionDocumentsReview Permission = "documents.review"

	// Archive and restore. Separate from review because it takes a published
	// document out of circulation, which is the one action here a reviewer is
	// not automatically trusted with.
	PermissionDocumentsArchive Permission = "documents.archive"

	// Tags are created freely by anyone who may edit a document (FR-6), so
	// there is no tags.store gate on tagging. These govern the tag *master* —
	// renaming the vocabulary, and deleting a tag out of every document that
	// uses it, which is a different and much larger act.
	PermissionMastersTagsShow  Permission = "masters.tags.show"
	PermissionMastersTagsStore Permission = "masters.tags.store"
	// Separate from store: renaming relabels every document already carrying
	// the tag, which is closer to deleting it than to adding one.
	PermissionMastersTagsUpdate  Permission = "masters.tags.update"
	PermissionMastersTagsDestroy Permission = "masters.tags.destroy"

	// Restricting a document to named people, and lifting that restriction.
	// Deliberately not folded into documents.update: everyone who may edit a
	// document would otherwise be able to lock everyone else out of it.
	PermissionDocumentsAccessManage Permission = "documents.access.manage"

	// Modul 4. Three separate capabilities, because they are three different
	// kinds of trust:
	//
	//   protection.manage  decides how hard a document defends itself
	//   share              hands a document to someone with no account at all
	//   access-log.show    reads who opened what, from where and when
	//
	// The last one is the one to be most careful with: the trail is a record of
	// people's reading, and a module that exists to make leaks traceable would
	// be self-defeating if everyone could read it.
	PermissionDocumentsProtectionManage Permission = "documents.protection.manage"
	PermissionDocumentsShare            Permission = "documents.share"
	PermissionDocumentsAccessLogShow    Permission = "documents.access-log.show"
)

const (
	FeatureGroupManagementUsers          = "management - users"
	FeatureGroupManagementPermissions    = "management - permissions"
	FeatureGroupManagementRoles          = "management - roles"
	FeatureGroupManagementConfigurations = "management - configurations"
	FeatureGroupManagementActivityLogs   = "management - activity logs"
	FeatureGroupManagementAudits         = "management - audits"
	FeatureGroupManagementJobLogs        = "management - job logs"
	FeatureGroupManagementQueue          = "management - queue"
	FeatureGroupManagementSchedules      = "management - schedules"
	FeatureGroupManagementM2MClients     = "management - m2m clients"
	FeatureGroupManagementCache          = "management - cache"
	FeatureGroupLookups                  = "lookups"
	FeatureGroupMastersSpaces            = "masters - spaces"
	FeatureGroupMastersCategories        = "masters - categories"
	FeatureGroupDocuments                = "documents"
	FeatureGroupMastersTags              = "masters - tags"
)

type PermissionDetail struct {
	Name         Permission
	Description  string
	FeatureGroup string
}

// PermissionSeeder is grouped by feature so the role editor lists permissions
// in this order without needing an extra sort.
var PermissionSeeder = []PermissionDetail{
	{PermissionManagementUsersShow, "can show data user", FeatureGroupManagementUsers},
	{PermissionManagementUsersUpdate, "can update data user", FeatureGroupManagementUsers},

	{PermissionManagementPermissionsShow, "can show all data permission", FeatureGroupManagementPermissions},

	{PermissionManagementRolesShow, "can show data role", FeatureGroupManagementRoles},
	{PermissionManagementRolesStore, "can add new data role", FeatureGroupManagementRoles},
	{PermissionManagementRolesUpdate, "can update data role", FeatureGroupManagementRoles},
	{PermissionManagementRolesDestroy, "can delete data role", FeatureGroupManagementRoles},

	{PermissionManagementConfigurationsShow, "can show data configuration", FeatureGroupManagementConfigurations},
	{PermissionManagementConfigurationsUpdate, "can update data configuration", FeatureGroupManagementConfigurations},

	{PermissionManagementActivityLogsShow, "can show activity logs (HTTP request history)", FeatureGroupManagementActivityLogs},
	{PermissionManagementAuditsShow, "can show audit trails (field-level change history)", FeatureGroupManagementAudits},

	{PermissionManagementJobLogsShow, "can show scheduled/queue job run history", FeatureGroupManagementJobLogs},

	// Manage is a separate, larger capability than Show: a retry re-runs
	// whatever the job does (sends the email again, charges the card again),
	// so it isn't granted alongside "can view".
	{PermissionManagementQueueShow, "can view the job queue", FeatureGroupManagementQueue},
	{PermissionManagementQueueManage, "can retry, cancel and purge queued jobs", FeatureGroupManagementQueue},

	{PermissionManagementSchedulesShow, "can view scheduled jobs", FeatureGroupManagementSchedules},
	{PermissionManagementSchedulesManage, "can enable, reschedule and trigger scheduled jobs", FeatureGroupManagementSchedules},

	{PermissionManagementCacheShow, "can show what is cached", FeatureGroupManagementCache},
	{PermissionManagementCacheDestroy, "can clear the lookup cache", FeatureGroupManagementCache},

	{PermissionManagementM2MClientsShow, "can show data m2m client", FeatureGroupManagementM2MClients},
	{PermissionManagementM2MClientsStore, "can add new data m2m client", FeatureGroupManagementM2MClients},
	{PermissionManagementM2MClientsUpdate, "can update data m2m client", FeatureGroupManagementM2MClients},
	{PermissionManagementM2MClientsDestroy, "can delete data m2m client", FeatureGroupManagementM2MClients},

	{PermissionLookupsUsersShow, "can list user names for a select option", FeatureGroupLookups},
	{PermissionLookupsRolesShow, "can list role names for a select option", FeatureGroupLookups},
	{PermissionLookupsPermissionsShow, "can list permission names for a select option", FeatureGroupLookups},
	{PermissionLookupsSpacesShow, "can list space names for a select option", FeatureGroupLookups},
	{PermissionLookupsCategoriesShow, "can list category names for a select option", FeatureGroupLookups},
	{PermissionLookupsTagsShow, "can list tag names for a select option", FeatureGroupLookups},

	{PermissionMastersSpacesShow, "can show spaces", FeatureGroupMastersSpaces},
	{PermissionMastersSpacesStore, "can add a new space", FeatureGroupMastersSpaces},
	{PermissionMastersSpacesUpdate, "can update a space", FeatureGroupMastersSpaces},
	{PermissionMastersSpacesDestroy, "can archive a space", FeatureGroupMastersSpaces},

	{PermissionMastersCategoriesShow, "can show a space's categories", FeatureGroupMastersCategories},
	{PermissionMastersCategoriesStore, "can add a new category", FeatureGroupMastersCategories},
	{PermissionMastersCategoriesUpdate, "can update and reorder categories", FeatureGroupMastersCategories},
	{PermissionMastersCategoriesDestroy, "can delete a category", FeatureGroupMastersCategories},

	{PermissionDocumentsShow, "can read documents", FeatureGroupDocuments},
	{PermissionDocumentsStore, "can create a new draft", FeatureGroupDocuments},
	{PermissionDocumentsUpdate, "can edit documents they authored", FeatureGroupDocuments},
	{PermissionDocumentsUpdateAll, "can edit any document, not only their own", FeatureGroupDocuments},
	{PermissionDocumentsDestroy, "can delete documents", FeatureGroupDocuments},
	{PermissionDocumentsReview, "can approve, reject, publish and unpublish documents", FeatureGroupDocuments},
	{PermissionDocumentsArchive, "can archive and restore documents", FeatureGroupDocuments},
	{PermissionDocumentsAccessManage, "can restrict a document to named users or roles", FeatureGroupDocuments},
	{PermissionDocumentsProtectionManage, "can set the protection level of a space or a document", FeatureGroupDocuments},
	{PermissionDocumentsShare, "can create an expiring view link to a published document", FeatureGroupDocuments},
	{PermissionDocumentsAccessLogShow, "can read the document access log", FeatureGroupDocuments},

	{PermissionMastersTagsShow, "can show the tag master", FeatureGroupMastersTags},
	{PermissionMastersTagsStore, "can add a new tag", FeatureGroupMastersTags},
	{PermissionMastersTagsUpdate, "can rename a tag, relabelling every document using it", FeatureGroupMastersTags},
	{PermissionMastersTagsDestroy, "can delete a tag from every document using it", FeatureGroupMastersTags},
}
