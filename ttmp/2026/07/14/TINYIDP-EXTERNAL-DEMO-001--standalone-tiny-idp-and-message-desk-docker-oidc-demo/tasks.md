# Tasks

## TODO

- [ ] Phase 1: Define two-container topology, trust boundaries, issuer and redirect contract <!-- t:cvqz -->
- [ ] Phase 2: Add standalone tiny-idp Docker image, persistent state, seeded demo accounts, and client bootstrap <!-- t:i82l -->
- [ ] Phase 3: Refactor Message Desk into external-issuer relying-party mode without embedded provider imports <!-- t:ovm9 -->
- [ ] Phase 4: Reuse the Message Desk visual language for provider login, chooser, consent, and logout <!-- t:d4dw -->
- [ ] Phase 5: Add Docker Compose, HTTPS/reverse-proxy development guidance, health checks, and operator runbook <!-- t:l2ec -->
- [ ] Phase 6: Add two-origin integration, browser, logout, scope, CSRF, and failure-path assurance <!-- t:xaai -->
- [ ] Phase 7: Complete delivery diary, security review, reMarkable bundle, and handoff <!-- t:f4gs -->
- [x] Phase 1.1: Define external RP configuration schema and canonical origin validation <!-- t:enhj -->
- [x] Phase 1.2: Define IdP seed manifest, client redirect/logout registration, and idempotency semantics <!-- t:qg9j -->
- [x] Phase 1.3: Define development HTTP versus production HTTPS/cookie profile boundaries <!-- t:dxv2 -->
- [x] Phase 2.1: Add standalone tiny-idp container command and Dockerfile <!-- t:98rp -->
- [x] Phase 2.2: Add idempotent seeded-account and browser-client bootstrap <!-- t:8w1h -->
- [x] Phase 3.1: Extract or copy external Message Desk RP composition without embedded imports <!-- t:nl19 -->
- [x] Phase 3.2: Add external discovery, token, JWKS, and end-session endpoint handling <!-- t:6bcu -->
- [x] Phase 3.3: Remove self-registration from external mode and document account provisioning boundary <!-- t:por4 -->
- [x] Phase 4.1: Package the Message Desk interaction renderer with standalone tiny-idp <!-- t:dxaq -->
- [x] Phase 5.1: Add compose topology, named volumes, health checks, and startup ordering <!-- t:acph -->
- [x] Phase 5.2: Add operator README and reset/restart runbook <!-- t:77cp -->
- [ ] Phase 6.1: Add two-origin OIDC integration tests <!-- t:zktj -->
- [x] Phase 6.2: Add browser smoke for login, scopes, chooser, local logout, and global logout <!-- t:p7x6 -->
- [ ] Phase 6.3: Add failure, secret-leak, and persistence isolation checks <!-- t:hj1t -->
