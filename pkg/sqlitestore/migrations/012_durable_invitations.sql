CREATE TABLE durable_invitations (
    code_hash BLOB PRIMARY KEY,
    expires_at_ns INTEGER NOT NULL,
    revoked_at_ns INTEGER,
    redeemed_at_ns INTEGER,
    data BLOB NOT NULL
);

CREATE INDEX idx_durable_invitations_expiry
    ON durable_invitations(expires_at_ns);
