# Sources

These files were downloaded with `defuddle parse <url> --md | fold -w 100 -s` for TINYIDP-PROD-001.

| File | Source | Why it matters |
|---|---|---|
| `01-openid-net-specs-openid-connect-core-1-0-html.md` | OpenID Connect Core 1.0 | Defines Authorization Code Flow, ID Token rules, authentication request validation, UserInfo, TLS and CSRF/clickjacking requirements. |
| `02-datatracker-ietf-org-doc-html-rfc9700.md` | RFC 9700 OAuth 2.0 Security Best Current Practice | Modern OAuth security profile: PKCE, redirect URI handling, deprecating unsafe grants and response types. |
| `03-pkg-go-dev-github-com-ory-fosite.md` | Ory Fosite package docs | API reference for authorize/token request parsing and response/error writers. |
| `04-pkg-go-dev-github-com-ory-fosite-compose.md` | Ory Fosite compose package docs | API reference for explicit handler composition and factory selection. |
| `05-openid-net-specs-openid-connect-discovery-1-0-html.md` | OpenID Connect Discovery 1.0 | Discovery URL construction and issuer metadata requirements. |
| `06-openid-net-certification.md` | OpenID Foundation Certification | Conformance test suite reference and certification context. |
| `07-cheatsheetseries-owasp-org-cheatsheets-oauth2-cheat-sheet-html.md` | OWASP OAuth2 Cheat Sheet | Security validation checklist for OAuth/OIDC deployments. |
| `08-cheatsheetseries-owasp-org-cheatsheets-authentication-cheat-sheet-html.md` | OWASP Authentication Cheat Sheet | Password/login, authentication failure, account lifecycle guidance. |
| `09-cheatsheetseries-owasp-org-cheatsheets-session-management-cheat-sheet-html.md` | OWASP Session Management Cheat Sheet | Cookie/session identifier generation, transport and invalidation requirements. |
| `10-cheatsheetseries-owasp-org-cheatsheets-cross-site-request-forgery-prevention-che.md` | OWASP CSRF Prevention Cheat Sheet | CSRF token and browser-request validation guidance for login/consent forms. |
| `11-cheatsheetseries-owasp-org-cheatsheets-transport-layer-security-cheat-sheet-html.md` | OWASP TLS Cheat Sheet | HTTPS/TLS deployment requirements for production mode. |
| `12-cheatsheetseries-owasp-org-cheatsheets-logging-cheat-sheet-html.md` | OWASP Logging Cheat Sheet | Audit event selection and sensitive-data logging exclusions. |
