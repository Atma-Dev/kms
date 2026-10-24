-- Queue attempts routinely finish in well under a second; whole-second
-- TIMESTAMP columns would make every queue job_logs row report a 0ms or
-- 1000ms duration instead of the real figure.
ALTER TABLE job_logs
    MODIFY started_at TIMESTAMP(6) NOT NULL,
    MODIFY finished_at TIMESTAMP(6) NULL,
    MODIFY created_at TIMESTAMP(6) NULL,
    MODIFY updated_at TIMESTAMP(6) NULL;
