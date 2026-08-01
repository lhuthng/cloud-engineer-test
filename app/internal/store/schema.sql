CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    current_version INTEGER NOT NULL DEFAULT 1,
    extension       TEXT NOT NULL DEFAULT 'mp4',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS jobs (
    id             TEXT PRIMARY KEY,
    session_id     TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    operation      TEXT NOT NULL,
    params         JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_version  INTEGER NOT NULL,
    output_version INTEGER NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    error          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS jobs_session_id_idx ON jobs (session_id, created_at);
CREATE INDEX IF NOT EXISTS jobs_status_idx ON jobs (status, created_at);
