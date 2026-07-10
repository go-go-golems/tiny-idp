ALTER TABLE fosite_authorize_codes ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE fosite_pkces ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE fosite_oidc_sessions ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE fosite_access_tokens ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE fosite_refresh_tokens ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX idx_fosite_authorize_created_at ON fosite_authorize_codes(created_at);
CREATE INDEX idx_fosite_pkce_created_at ON fosite_pkces(created_at);
CREATE INDEX idx_fosite_oidc_created_at ON fosite_oidc_sessions(created_at);
CREATE INDEX idx_fosite_access_created_at ON fosite_access_tokens(created_at);
CREATE INDEX idx_fosite_refresh_created_at ON fosite_refresh_tokens(created_at);
