-- Local deployment bootstrap. The application user ID is OIDCUserID for
-- issuer https://idp.localhost:8443 and subject local-admin.
BEGIN;
INSERT INTO auth_app_users (id, oidc_issuer, oidc_subject, email, display_name, email_verified)
VALUES ('user:oidc:BxKqq4eDe_cf6-nOKK9OhbnF55jWpLExKN-4vSqptXU', 'https://idp.localhost:8443', 'local-admin', 'admin@example.test', 'Local Administrator', true)
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, email_verified = EXCLUDED.email_verified, disabled_at = NULL;

INSERT INTO auth_app_external_identities (issuer, subject, user_id)
VALUES ('https://idp.localhost:8443', 'local-admin', 'user:oidc:BxKqq4eDe_cf6-nOKK9OhbnF55jWpLExKN-4vSqptXU')
ON CONFLICT (issuer, subject) DO UPDATE SET user_id = EXCLUDED.user_id;

INSERT INTO auth_app_tenants (id, slug, name) VALUES ('o1', 'local-demo', 'Local Demo Organization')
ON CONFLICT (id) DO UPDATE SET slug = EXCLUDED.slug, name = EXCLUDED.name, disabled_at = NULL;

INSERT INTO auth_app_resources (type, id, name, tenant_id, owner_id, claims_json)
VALUES
  ('org', 'o1', 'Local Demo Organization', 'o1', '', '{}'),
  ('project', 'p1', 'Local Demo Project', 'o1', '', '{}')
ON CONFLICT (type, id) DO UPDATE SET name = EXCLUDED.name, tenant_id = EXCLUDED.tenant_id;

INSERT INTO auth_app_memberships (user_id, tenant_id, role)
VALUES ('user:oidc:BxKqq4eDe_cf6-nOKK9OhbnF55jWpLExKN-4vSqptXU', 'o1', 'admin')
ON CONFLICT (user_id, tenant_id, role) DO UPDATE SET revoked_at = NULL;
COMMIT;
