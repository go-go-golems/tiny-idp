CREATE TABLE email_challenges (
    id TEXT PRIMARY KEY,
    expires_at_ns INTEGER NOT NULL,
    data BLOB NOT NULL
);
CREATE INDEX idx_email_challenges_expiry ON email_challenges(expires_at_ns);
