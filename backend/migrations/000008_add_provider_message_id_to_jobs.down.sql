DROP INDEX IF EXISTS idx_jobs_provider_message_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS provider_message_id;
