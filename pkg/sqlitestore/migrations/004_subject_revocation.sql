ALTER TABLE fosite_authorize_codes ADD COLUMN subject TEXT NOT NULL DEFAULT '';
ALTER TABLE fosite_pkces ADD COLUMN subject TEXT NOT NULL DEFAULT '';
ALTER TABLE fosite_oidc_sessions ADD COLUMN subject TEXT NOT NULL DEFAULT '';
ALTER TABLE fosite_access_tokens ADD COLUMN subject TEXT NOT NULL DEFAULT '';
ALTER TABLE fosite_refresh_tokens ADD COLUMN subject TEXT NOT NULL DEFAULT '';

UPDATE fosite_authorize_codes SET subject = COALESCE(json_extract(request_json, '$.session.subject'), '');
UPDATE fosite_pkces SET subject = COALESCE(json_extract(request_json, '$.session.subject'), '');
UPDATE fosite_oidc_sessions SET subject = COALESCE(json_extract(request_json, '$.session.subject'), '');
UPDATE fosite_access_tokens SET subject = COALESCE(json_extract(request_json, '$.session.subject'), '');
UPDATE fosite_refresh_tokens SET subject = COALESCE(json_extract(request_json, '$.session.subject'), '');

CREATE INDEX idx_fosite_authorize_subject ON fosite_authorize_codes(subject);
CREATE INDEX idx_fosite_pkce_subject ON fosite_pkces(subject);
CREATE INDEX idx_fosite_oidc_subject ON fosite_oidc_sessions(subject);
CREATE INDEX idx_fosite_access_subject ON fosite_access_tokens(subject);
CREATE INDEX idx_fosite_refresh_subject ON fosite_refresh_tokens(subject);
