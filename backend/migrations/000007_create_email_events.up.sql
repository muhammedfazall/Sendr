CREATE TABLE email_events (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    sg_event_id   TEXT,
    sg_message_id TEXT,
    job_id        TEXT REFERENCES jobs(id),
    timestamp     TIMESTAMPTZ NOT NULL,
    metadata      JSONB DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_events_event_type ON email_events (event_type);
CREATE INDEX idx_email_events_timestamp ON email_events (timestamp DESC);
CREATE INDEX idx_email_events_job_id ON email_events (job_id);
