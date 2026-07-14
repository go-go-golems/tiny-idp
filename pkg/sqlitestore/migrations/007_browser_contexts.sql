CREATE TABLE browser_contexts (
    hash TEXT PRIMARY KEY,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    data BLOB NOT NULL
);

CREATE INDEX idx_browser_contexts_expires_at ON browser_contexts(expires_at);
CREATE INDEX idx_browser_contexts_revoked_at ON browser_contexts(revoked_at);

CREATE TABLE remembered_browser_sessions (
    hash TEXT PRIMARY KEY,
    context_hash TEXT NOT NULL,
    session_hash TEXT NOT NULL,
    user_id TEXT NOT NULL,
    removed_at TIMESTAMP,
    last_used_at TIMESTAMP NOT NULL,
    data BLOB NOT NULL
);

CREATE INDEX idx_remembered_browser_sessions_context
    ON remembered_browser_sessions(context_hash);
CREATE INDEX idx_remembered_browser_sessions_session
    ON remembered_browser_sessions(session_hash);
CREATE INDEX idx_remembered_browser_sessions_user
    ON remembered_browser_sessions(user_id);
CREATE INDEX idx_remembered_browser_sessions_removed_at
    ON remembered_browser_sessions(removed_at);
