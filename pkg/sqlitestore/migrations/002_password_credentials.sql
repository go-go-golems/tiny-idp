CREATE TABLE IF NOT EXISTS password_credentials (
  user_id TEXT PRIMARY KEY,
  login TEXT NOT NULL UNIQUE,
  data BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_password_credentials_login ON password_credentials(login);

CREATE TABLE IF NOT EXISTS account_security_states (
  user_id TEXT PRIMARY KEY,
  data BLOB NOT NULL
);
