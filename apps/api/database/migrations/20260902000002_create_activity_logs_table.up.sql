-- New table owned by this app.
--
-- One row per HTTP request, written by middleware.LoggerMiddleware after the
-- request completes. user_id is nullable since unauthenticated requests
-- (login, health checks) still get logged.
CREATE TABLE activity_logs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    user_id CHAR(36) NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(255) NOT NULL,
    status_code SMALLINT NOT NULL,
    ip VARCHAR(45) NOT NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    INDEX activity_logs_user_id_created_at_index (user_id, created_at)
);
