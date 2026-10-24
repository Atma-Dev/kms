// Mirrors enums/permission.go (PermissionSeeder) in ../api-kms.
export enum Permission {
  MANAGEMENT_USERS_SHOW = 'management.users.show',
  MANAGEMENT_USERS_UPDATE = 'management.users.update',

  MANAGEMENT_PERMISSIONS_SHOW = 'management.permissions.show',

  MANAGEMENT_ROLES_SHOW = 'management.roles.show',
  MANAGEMENT_ROLES_STORE = 'management.roles.store',
  MANAGEMENT_ROLES_UPDATE = 'management.roles.update',
  MANAGEMENT_ROLES_DESTROY = 'management.roles.destroy',

  MANAGEMENT_CONFIGURATIONS_SHOW = 'management.configurations.show',
  MANAGEMENT_CONFIGURATIONS_UPDATE = 'management.configurations.update',

  // HTTP request log.
  MANAGEMENT_ACTIVITY_LOGS_SHOW = 'management.activity-logs.show',

  // Field-level change history (who changed what, from what to what) — not
  // to be confused with MANAGEMENT_ACTIVITY_LOGS_SHOW above (HTTP request log).
  MANAGEMENT_AUDITS_SHOW = 'management.audits.show',

  MANAGEMENT_JOB_LOGS_SHOW = 'management.job-logs.show',

  // MANAGEMENT_QUEUE_MANAGE (retry/cancel/purge) is separate from SHOW: a
  // retry re-runs whatever the job does, which isn't the same capability as
  // looking at the list.
  MANAGEMENT_QUEUE_SHOW = 'management.queue.show',
  MANAGEMENT_QUEUE_MANAGE = 'management.queue.manage',

  MANAGEMENT_SCHEDULES_SHOW = 'management.schedules.show',
  MANAGEMENT_SCHEDULES_MANAGE = 'management.schedules.manage',

  MANAGEMENT_CACHE_DESTROY = 'management.cache.destroy',

  MANAGEMENT_M2M_CLIENTS_SHOW = 'management.m2m-clients.show',
  MANAGEMENT_M2M_CLIENTS_STORE = 'management.m2m-clients.store',
  MANAGEMENT_M2M_CLIENTS_UPDATE = 'management.m2m-clients.update',
  MANAGEMENT_M2M_CLIENTS_DESTROY = 'management.m2m-clients.destroy',

  // Back the select-option dropdowns. Separate from the *.show permission of
  // the resource they read: seeing names in a filter is a smaller capability
  // than managing the records behind them. A page that hides a dropdown for
  // want of one of these must also stop sending the matching filter param —
  // the API refuses it with a 403 rather than ignoring it.
  LOOKUPS_USERS_SHOW = 'lookups.users.show',
  LOOKUPS_ROLES_SHOW = 'lookups.roles.show',
  LOOKUPS_PERMISSIONS_SHOW = 'lookups.permissions.show',
  LOOKUPS_SPACES_SHOW = 'lookups.spaces.show',
  LOOKUPS_CATEGORIES_SHOW = 'lookups.categories.show',
  LOOKUPS_TAGS_SHOW = 'lookups.tags.show',

  MASTERS_SPACES_SHOW = 'masters.spaces.show',
  MASTERS_SPACES_STORE = 'masters.spaces.store',
  MASTERS_SPACES_UPDATE = 'masters.spaces.update',
  MASTERS_SPACES_DESTROY = 'masters.spaces.destroy',

  MASTERS_CATEGORIES_SHOW = 'masters.categories.show',
  MASTERS_CATEGORIES_STORE = 'masters.categories.store',
  MASTERS_CATEGORIES_UPDATE = 'masters.categories.update',
  MASTERS_CATEGORIES_DESTROY = 'masters.categories.destroy',

  DOCUMENTS_SHOW = 'documents.show',
  DOCUMENTS_STORE = 'documents.store',

  // The Contributor/Editor split. DOCUMENTS_UPDATE is "edit what I wrote",
  // DOCUMENTS_UPDATE_ALL is "edit anyone's". Whether *this* document is
  // editable is answered by the API on the detail response (`can_edit`) — it
  // knows who authored the row, which this enum cannot.
  DOCUMENTS_UPDATE = 'documents.update',
  DOCUMENTS_UPDATE_ALL = 'documents.update.all',

  DOCUMENTS_DESTROY = 'documents.destroy',
  DOCUMENTS_REVIEW = 'documents.review',
  DOCUMENTS_ARCHIVE = 'documents.archive',

  // Restricting a document to named people. Deliberately not implied by
  // DOCUMENTS_UPDATE: everyone who may edit a document would otherwise be able
  // to lock everyone else out of it.
  DOCUMENTS_ACCESS_MANAGE = 'documents.access.manage',

  // Modul 4. Three different kinds of trust, so three permissions:
  // how hard a document defends itself, handing one to somebody with no
  // account, and reading the record of who opened what.
  DOCUMENTS_PROTECTION_MANAGE = 'documents.protection.manage',
  DOCUMENTS_SHARE = 'documents.share',
  DOCUMENTS_ACCESS_LOG_SHOW = 'documents.access-log.show',

  // The tag master. Tagging a document needs none of these — the vocabulary is
  // open, and retagging answers to the document's own edit rule.
  MASTERS_TAGS_SHOW = 'masters.tags.show',
  MASTERS_TAGS_STORE = 'masters.tags.store',
  MASTERS_TAGS_UPDATE = 'masters.tags.update',
  MASTERS_TAGS_DESTROY = 'masters.tags.destroy',
}
