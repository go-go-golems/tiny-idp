# tiny-idp Production Conformance Runbook

This runbook captures the repeatable checks for the strict `fosite` engine. The mock engine is intentionally excluded because it contains debug/failure routes for relying-party tests.

## Scope

Current strict-engine profile:

- OAuth 2.0 Authorization Code Grant.
- PKCE with `S256`.
- OIDC Core ID Token issuance.
- Discovery and JWKS publication.
- Refresh token rotation/reuse rejection.
- No debug routes.
- Durable SQLite protocol state.

Out of scope until explicitly implemented:

- Implicit/hybrid response types.
- Dynamic client registration.
- Device Authorization Grant in the production engine.
- DPoP in the production engine.

## Local regression command

```bash
scripts/run-conformance.sh
```

The script runs the protocol/security regression tests that are suitable for CI without external services.

## Hosted OpenID Foundation suite automation

Use `scripts/oidf_hosted_runner.py` after logging into `https://www.certification.openid.net` in a browser and copying the active `JSESSIONID` cookie.

```bash
export OIDF_JSESSIONID='<cookie value>'
scripts/oidf_hosted_runner.py \
  --plan '<plan id>' \
  --remaining \
  --artifacts ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/oidf-hosted-python
```

The Python runner creates test instances through the suite API, polls `/api/info`, saves `/api/info` and `/api/log` JSON artifacts, and follows exported authorization URLs with an HTTP browser session. It submits the tiny-idp login/consent form as `alice` and reproduces the suite's implicit callback POST so code-flow responses continue without manual JavaScript. It stops on failures, timeouts, or manual review states unless `--keep-going` is supplied.

For hosted Basic OP refresh-token tests, configure distinct static clients in the suite and start the strict CLI with an extra client that shares the suite callback redirect URI:

```bash
CB='https://www.certification.openid.net/test/a/<alias>/callback'
tinyidp serve-dev --engine fosite \
  --issuer '<public issuer>' \
  --client-id web-app \
  --client-secret dev-secret \
  --redirect-uris "$CB" \
  --redirect-uris "$CB?dummy1=lorem&dummy2=ipsum" \
  --extra-clients "web-app-2|dev-secret-2|$CB|$CB?dummy1=lorem&dummy2=ipsum"
```

## Manual OpenID Foundation suite preparation

1. Start `tinyidp` strict engine on a publicly reachable HTTPS URL.
2. Configure a public test client with an exact redirect URI supplied by the conformance suite.
3. Use Authorization Code + PKCE tests only.
4. Verify discovery metadata before starting:
   - `issuer`
   - `authorization_endpoint`
   - `token_endpoint`
   - `userinfo_endpoint`
   - `jwks_uri`
   - `response_types_supported: ["code"]`
   - `code_challenge_methods_supported: ["S256"]`
5. Run the suite manually or with `scripts/oidf_hosted_runner.py`.
6. Save the suite export or captured `/api/info` and `/api/log` artifacts under the ticket/source bundle.
7. Do not mark a profile conformant until all failures have either been fixed or mapped to an intentionally unsupported feature.

## Required evidence for release

- `go test ./...` passes.
- `scripts/run-conformance.sh` passes.
- Discovery advertises only supported capabilities.
- ID Tokens verify against JWKS, including `kid` lookup.
- Retired signing keys remain in JWKS long enough for previously issued ID Tokens.
- Refresh-token reuse is rejected after rotation.
- Audit logs contain stable reason codes and no raw bearer tokens, codes, passwords, or client secrets.
