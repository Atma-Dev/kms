ALTER TABLE job_logs
    MODIFY started_at TIMESTAMP NOT NULL,
    MODIFY finished_at TIMESTAMP NULL,
    MODIFY created_at TIMESTAMP NULL,
    MODIFY updated_at TIMESTAMP NULL;
