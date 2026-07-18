# tiny-idp Strict Engine Security Profile

This document summarizes the production strict-engine baseline.

## Enabled controls

- Fosite-backed OAuth/OIDC validation and response writing.
- Authorization Code + PKCE (`S256`) only.
- Exact redirect URI allow-listing through Fosite client metadata.
- Durable SQLite Fosite protocol state when using the SQLite store.
- Server-side IdP browser sessions with opaque cookies and hashed handles.
- RP-initiated current-browser logout with server-side session revocation, exact client-scoped post-logout redirect matching, configured cookie clearing, and audit events.
- CSRF token/cookie requirement on strict login and consent POSTs.
- `HttpOnly`, `SameSite=Lax`, issuer-path-scoped cookies; production embedding validation requires secure cookies.
- Security headers: `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, and restrictive CSP.
- `Cache-Control: no-store` on browser interaction and token paths.
- Persistent consent records in production mode.
- Rate-limiting hook with a fixed-window implementation available to embedders.
- Structured audit sink with stable reason codes for Fosite/OAuth errors.
- Persistent signing keys and rotation helper that keeps retired keys verifiable.
- Strict RFC 8628 device authorization endpoint, browser verification flow,
  durable poll/decision/consume state, and transactional Fosite token issuance.
  Release approval still requires the evidence and independent-review gate in
  `TINYIDP-DEVICE-PROD-001`.

## Explicitly unsupported in strict mode today

- Debug routes.
- Scenario-driven malformed token/JWKS behavior.
- Implicit and hybrid response types.
- Production DPoP.
- Dynamic client registration.

## Release gate

A production release candidate must pass:

```bash
go test ./...
scripts/run-conformance.sh
docmgr doctor --ticket TINYIDP-PROD-001 --stale-after 30
docmgr doctor --ticket TINYIDP-PROD-DEPLOY-001 --stale-after 30
```
