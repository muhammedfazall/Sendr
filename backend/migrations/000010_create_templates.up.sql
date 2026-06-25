CREATE TABLE templates (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id),
    name             TEXT NOT NULL,
    subject_template TEXT NOT NULL DEFAULT '',
    html_template    TEXT NOT NULL DEFAULT '',
    text_template    TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_templates_user_id ON templates (user_id);
