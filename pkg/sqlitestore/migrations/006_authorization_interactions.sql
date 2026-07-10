CREATE TABLE authorization_interactions (
    hash TEXT PRIMARY KEY,
    expires_at TIMESTAMP NOT NULL,
    consumed_at TIMESTAMP,
    data BLOB NOT NULL
);

CREATE INDEX idx_authorization_interactions_expires_at
    ON authorization_interactions(expires_at);

CREATE INDEX idx_authorization_interactions_consumed_at
    ON authorization_interactions(consumed_at);
