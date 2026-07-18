CREATE TABLE device_grants (
    id TEXT PRIMARY KEY,
    device_code_hash TEXT NOT NULL UNIQUE,
    user_code_hash TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'consumed')),
    expires_at TIMESTAMP NOT NULL,
    next_poll_at TIMESTAMP NOT NULL,
    data BLOB NOT NULL,
    FOREIGN KEY (client_id) REFERENCES clients(id)
);

CREATE INDEX idx_device_grants_expiry ON device_grants(expires_at);
CREATE INDEX idx_device_grants_status_expiry ON device_grants(status, expires_at);
