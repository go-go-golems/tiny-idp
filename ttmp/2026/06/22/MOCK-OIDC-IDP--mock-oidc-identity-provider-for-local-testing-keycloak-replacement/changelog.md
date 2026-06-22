# Changelog

## 2026-06-22

- Initial workspace created


## 2026-06-22

Step 1: Created ticket MOCK-OIDC-IDP, design doc, phased task breakdown, and initial diary. No code yet.

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/ttmp/2026/06/22/MOCK-OIDC-IDP--mock-oidc-identity-provider-for-local-testing-keycloak-replacement/design-doc/01-mock-oidc-idp-design-and-implementation-guide.md — Intern-ready design and implementation guide


## 2026-06-22

Step 2: Phase 0 baseline OIDC happy path — main.go + tests, go build/vet/test green (commit d473d513).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main.go — server


## 2026-06-22

Step 3: Phase 1 multiple synthetic users + refactor into internal/server with go:embed login page (commit f9ece67).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/authorize.go — GET+POST login flow + parseAuthorizeRequest (commit f9ece67)

