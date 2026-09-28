-- New table owned by this app.
--
-- request_body/response_body are written by activitylog.SanitizeBody
-- (redacted + truncated before storage), so no further processing is needed
-- to render them.
ALTER TABLE activity_logs
    ADD COLUMN request_body TEXT NULL AFTER ip,
    ADD COLUMN response_body TEXT NULL AFTER request_body;
