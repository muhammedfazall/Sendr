CREATE INDEX idx_jobs_user_created_at ON jobs(user_id, created_at DESC, id DESC);
CREATE INDEX idx_jobs_user_status_created_at ON jobs(user_id, status, created_at DESC, id DESC);
