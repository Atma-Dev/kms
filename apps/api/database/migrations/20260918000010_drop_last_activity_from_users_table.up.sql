-- users.last_activity was stamped on every authenticated request by
-- LastActivityMiddleware, and read by exactly one thing: the idle-user job that
-- went with is_active (20260918000009). With that gone it was a write on the
-- hot path maintaining a column nobody queried.
--
-- If "last seen" is ever wanted, activity_logs already records every request
-- with its user and timestamp — a MAX(created_at) there answers the same
-- question without a second write per request.
ALTER TABLE users DROP COLUMN last_activity;
