# Tasks

## TODO

- [x] Audit current Glazed configuration and production command construction. <!-- t:cfg1 -->
- [x] Audit current Goja capability and workflow seams. <!-- t:js01 -->
- [x] Audit readiness, metrics, logging, audit, and deployment boundaries. <!-- t:ops1 -->
- [x] Compare compiled-in, shared-library, subprocess, JavaScript-only, and separate-service plugin models. <!-- t:api1 -->
- [x] Decide the browser-identity integration seam: use a host-owned OIDC authorization-code and PKCE broker in version one. <!-- t:id01 -->
- [x] Write the full plugin API and Jitsi integration system design. <!-- t:des1 -->

## Phase 1 — Production Glazed composition

- [x] Extract reusable production settings and field definitions into `internal/sections/production`. <!-- t:p1s1 -->
- [x] Compose production and plugin sections into `serve-production`. <!-- t:p1s2 -->
- [x] Wire production through the existing profile, config, environment, argument, and flag middleware chain. <!-- t:p1s3 -->
- [x] Add precedence, source-provenance, and redacted configuration-inspection tests. <!-- t:p1s4 -->

## Phase 2 — Plugin kernel

- [x] Implement descriptor validation and the immutable compiled-in registry. <!-- t:p2s1 -->
- [x] Implement prepare/build phases, client requirements, reverse cleanup, and compile-time interface assertions. <!-- t:p2s2 -->
- [x] Mount derived scoped routes and extract common production HTTP security middleware. <!-- t:p2s3 -->
- [x] Compose plugin readiness into host readiness. <!-- t:p2s4 -->

## Phase 3 — OIDC relying-party broker

- [ ] Define the broker, identity, completion, transaction, and stable error contracts. <!-- t:p3s1 -->
- [ ] Add durable encrypted one-time integration transactions and the SQLite migration. <!-- t:p3s2 -->
- [ ] Implement state, nonce, PKCE S256, browser binding, expiry, replay protection, and atomic consumption. <!-- t:p3s3 -->
- [ ] Implement the provider-backed in-process HTTP transport, code exchange, ID-token validation, and userinfo mapping. <!-- t:p3s4 -->
- [ ] Validate plugin OIDC client requirements and test login, signup, session, cancellation, replay, expiry, and restart paths. <!-- t:p3s5 -->

## Phase 4 — Jitsi Goja policy

- [ ] Define `integration.jitsi.authorize@v1` JSON schemas, TypeScript declarations, and deterministic fixtures. <!-- t:p4s1 -->
- [ ] Implement the bounded Jitsi policy executor, warmed pool, readiness, metrics, and shutdown. <!-- t:p4s2 -->
- [ ] Add policy allow, deny, malformed output, timeout, saturation, interruption, and capability tests. <!-- t:p4s3 -->

## Phase 5 — Jitsi runtime

- [ ] Implement the Jitsi Glazed section, typed settings, strict validation, and secret resolution. <!-- t:p5s1 -->
- [ ] Implement exact HS256 Jitsi claim construction and token signing. <!-- t:p5s2 -->
- [ ] Implement start and callback handlers, safe redirects, themed errors, structured logs, and durable audit. <!-- t:p5s3 -->
- [ ] Add wrong-secret, expired-token, wrong-app, wrong-domain, wrong-room, privacy, and redaction tests. <!-- t:p5s4 -->

## Phase 6 — Observability and deployment

- [ ] Add OpenTelemetry meters and traces plus the internal health, readiness, and Prometheus listener. <!-- t:p6s1 -->
- [ ] Add Kubernetes configuration, probes, NetworkPolicy, ConfigMaps, and Vault Secrets Operator mounts. <!-- t:p6s2 -->
- [ ] Configure Prosody token mode and document coordinated HS256 rotation. <!-- t:p6s3 -->

## Phase 7 — End-to-end validation

- [ ] Validate local login, signup, account chooser, policy denial, cancellation, logout, and Jitsi redirects with Playwright. <!-- t:p7s1 -->
- [ ] Validate Prosody token enforcement and a two-browser media-connected conference. <!-- t:p7s2 -->
- [ ] Deploy through GitOps and verify Argo CD health, logs, metrics, audit, and absence of sensitive data. <!-- t:p7s3 -->
