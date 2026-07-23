CREATE TABLE integration_transactions (
    state_hash BLOB PRIMARY KEY,
    plugin_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    callback_path TEXT NOT NULL,
    nonce_hash BLOB NOT NULL,
    pkce_verifier_box BLOB NOT NULL,
    plugin_state_box BLOB NOT NULL,
    browser_binding_hash BLOB NOT NULL,
    created_at_ns INTEGER NOT NULL,
    expires_at_ns INTEGER NOT NULL,
    consumed_at_ns INTEGER
);

CREATE INDEX integration_transactions_expiry_idx
    ON integration_transactions(expires_at_ns);
