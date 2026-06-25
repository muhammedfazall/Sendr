CREATE TABLE unsubscriptions (
    email      TEXT PRIMARY KEY,
    reason     TEXT NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
