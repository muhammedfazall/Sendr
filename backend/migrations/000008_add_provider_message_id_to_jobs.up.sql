ALTER TABLE jobs ADD COLUMN provider_message_id TEXT;
CREATE INDEX idx_jobs_provider_message_id ON jobs (provider_message_id);
