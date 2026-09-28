-- app/activitylog's permission is renamed from management.audits.show to
-- management.activity-logs.show — that name and route now belong to
-- app/audit's field-change trail instead.
--
-- Renamed in place (UPDATE, not delete+insert) so this preserves the row's
-- id and every existing role_has_permissions grant. Doing this as a plain
-- seeder change instead would have been wrong: database/seeders/permission.go
-- deletes any permission name no longer in the enum (role_has_permissions
-- cascades with it), so an enum-only rename would have silently revoked
-- management.audits.show from every role that held it, and the new
-- management.audits.show row app/audit needs would start with zero grants.
UPDATE permissions
SET name = 'management.activity-logs.show',
    description = 'can show activity logs (HTTP request history)',
    feature_group = 'management - activity logs'
WHERE name = 'management.audits.show';
