CREATE TABLE workflow_continuations (
    handle_hash BLOB PRIMARY KEY,
    revision INTEGER NOT NULL,
    status TEXT NOT NULL,
    expires_at_ns INTEGER NOT NULL,
    data BLOB NOT NULL
);

CREATE INDEX idx_workflow_continuations_expiry
    ON workflow_continuations(expires_at_ns);
