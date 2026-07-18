# Tasks

## Phase 0 — Research, product contract, and vertical seam

- [x] Create TINYIDP-XAPP-001 with design guide, diary, source packet, tasks, and changelog. <!-- t:78uu -->
- [x] Inventory tiny-idp embedding, go-go-goja Express/hostauth/xgoja, and go-go-objects manager/provider APIs. <!-- t:9nu3 -->
- [x] Run all three repositories' complete Go test suites successfully under the shared go.work. <!-- t:bg14 -->
- [x] Preserve nine primary local guides and working xgoja specifications in the ticket source packet. <!-- t:0qp4 -->
- [x] Freeze v1 product story, URL layout, single-node deployment class, and non-goals. <!-- t:i785 -->
- [x] Create product directory, xgoja runtime-package specification, route/object/frontend skeletons, and Glazed command root. <!-- t:21x5 -->
- [x] Generate runtime package and TypeScript declarations from a clean workspace. <!-- t:18w8 -->
- [x] Build a minimal custom Go host with development tiny-idp, Express, assets, and durable objects. <!-- t:w8ei -->
- [x] Complete one login-to-private-object read/write vertical smoke test. <!-- t:m7ar -->
- [ ] Obtain architecture and threat-model approval before persistence/product work. <!-- t:vzqy -->

## Phase 1 — Provider-neutral OIDC and in-process issuer transport

- [x] Rename keycloakauth and Keycloak-specific app user fields to provider-neutral OIDC names without compatibility adapters unless explicitly required. <!-- t:jam5 -->
- [x] Add injectable HTTP client/transport to OIDC discovery, JWKS verification, and token exchange. <!-- t:77da -->
- [x] Implement origin-restricted InProcessIssuerTransport over an http.Handler. <!-- t:5mkm -->
- [x] Test discovery, JWKS refresh, and code exchange make no network dial. <!-- t:paaq -->
- [x] Test issuer/audience/nonce/state/PKCE validation remains unchanged. <!-- t:g7o1 -->
- [x] Test malformed public origins and transport origin mismatches fail closed. <!-- t:2zhc -->
- [ ] Add persistent OIDC transaction-store configuration for production. <!-- t:fj8t -->
- [x] Document app-session versus IdP-session and logout semantics. <!-- t:sypf -->

## Phase 2 — tiny-idp application embedding

- [x] Add cookie name/path configuration with issuer-path default and coexistence tests. <!-- t:o3fe -->
- [x] Define idempotent initialization for schema, secrets, signing key, RP client, and first user. <!-- t:8r8o -->
- [x] Add explicit serve refusal for missing initialization or ephemeral production state. <!-- t:fegd -->
- [x] Configure public PKCE client with exact callback and scopes. <!-- t:3vlx -->
- [x] Construct production audit, limiter, address resolver, authenticator, and maintenance services. <!-- t:xb9m -->
- [x] Mount issuer under /idp and aggregate liveness/readiness. <!-- t:vecw -->
- [x] Schedule immediate and periodic maintenance under errgroup context. <!-- t:keba -->
- [ ] Add signing-key rotation and retained verification-key integration test. <!-- t:t20n -->

## Phase 3 — Subject-bound durable objects

- [x] Define provider-neutral Issuer+Subject application identity derivation. <!-- t:d5yg -->
- [x] Generate and persist an owner-only object-binding key. <!-- t:25dn -->
- [x] Implement BoundDispatcher with allowed namespace policy. <!-- t:71id -->
- [x] Add fetch/rpc-for-actor xgoja adapter and TypeScript declarations. <!-- t:eg3d -->
- [x] Keep raw /rpc and /fetch gateways disabled in product mode. <!-- t:bbyu -->
- [ ] Add two-user isolation, object-name injection, namespace confusion, and disabled-user tests. <!-- t:ihzp -->
- [ ] Add bounded request/value/nesting validation. <!-- t:6ral -->
- [ ] Add per-object storage accounting, quota, and bounded metrics. <!-- t:bqo1 -->
- [ ] Document binding-key rotation/migration and backup requirements. <!-- t:pmed -->

## Phase 4 — Custom host and generated runtime composition

- [x] Implement generated runtime-package xgoja.yaml with Express, assets, and durableobjects providers. <!-- t:6irf -->
- [x] Inject Go-owned HTTP host, auth services, object dispatcher, and asset resolver through HostServices. <!-- t:57m6 -->
- [x] Load route JS before accepting traffic and fail on partial registration. <!-- t:omdh -->
- [ ] Build outer ServeMux with duplicate-safe native mount contributions. <!-- t:gdgh -->
- [ ] Add Glazed init, serve, doctor, backup, restore, and print-config commands. <!-- t:73t4 -->
- [x] Run HTTP server with TLS/proxy assumptions, timeouts, request limits, and graceful shutdown. <!-- t:2fai -->
- [x] Close runtime, object manager, app stores, IdP provider, and audit resources in dependency order. <!-- t:90ms -->
- [ ] Add aggregate readiness for identity, app auth, object manager, JS routes, and background loops. <!-- t:1iq0 -->

## Phase 5 — Frontend product loop

- [ ] Build embedded TypeScript/HTML/CSS frontend using pnpm and React/Redux/RTK Query if the UI exceeds the initial minimal shell. <!-- t:fw3t -->
- [x] Serve assets only under /static and serve index through an explicit route. <!-- t:dvyc -->
- [x] Implement session bootstrap, login redirect, app logout, and visible IdP-session semantics. <!-- t:jxa1 -->
- [x] Implement CSRF-aware object read/write API client. <!-- t:x474 -->
- [ ] Add loading, unauthenticated, authenticated, forbidden, conflict, offline, and error states. <!-- t:hgot -->
- [ ] Add accessible keyboard/focus/status behavior. <!-- t:1jom -->
- [ ] Add browser end-to-end login, persistence, logout, expiry, and two-user isolation tests. <!-- t:j5ba -->

## Phase 6 — Persistent operations and recovery

- [x] Define state-root layout and owner-only permissions. <!-- t:z87a -->
- [x] Use persistent SQLite stores for identity, app sessions/auth/audit, and durable objects. <!-- t:eat7 -->
- [ ] Implement quiesced or snapshot-consistent full state-root backup. <!-- t:e6lw -->
- [ ] Implement restore into a new directory with schema, secret, client, key, and object reconciliation checks. <!-- t:8d70 -->
- [ ] Test restart, backup/restore, missing alarm index repair, and partial-state failure. <!-- t:5qfj -->
- [ ] Add explicit durable-object schema migrations before incompatible changes. <!-- t:c07r -->
- [ ] Write single-replica deployment and reverse-proxy runbooks. <!-- t:7zvb -->

## Phase 7 — tiny-idp xgoja provider extraction

- [ ] Extract the proven IdP lifecycle into pkg/xgoja/providers/tinyidp. <!-- t:3bi6 -->
- [ ] Add provider configuration capability, Glazed sections, host service, native mount, health, and closers. <!-- t:c8zu -->
- [ ] Avoid a JS module unless a narrow read-only API has a concrete use case. <!-- t:kgss -->
- [ ] Add provider registry, config, host-service, mount, duplicate, closer, and require-negative tests. <!-- t:keou -->
- [ ] Add generated-host example, help pages, doctor, DTS/build validation, and smoke test. <!-- t:ln87 -->
- [ ] Demonstrate a second application consumes the provider without copied lifecycle code. <!-- t:df2d -->

## Phase 8 — Assurance and production release

- [ ] Run go test, race, fuzz, static analysis, model scenarios, and xgoja generation gates. <!-- t:303l -->
- [ ] Run hosted OIDC conformance and preserve exact configuration/results. <!-- t:4vf1 -->
- [ ] Inject IdP/app/object SQLite failures, JS timeout, maintenance failure, and shutdown races. <!-- t:uvhr -->
- [ ] Load-test login, app sessions, actor startup, hot objects, eviction, and disk growth. <!-- t:4lia -->
- [ ] Verify logs, traces, metrics, errors, and frontend never expose credentials, tokens, cookies, subjects, or object names. <!-- t:n8y4 -->
- [ ] Complete canary, backup/restore drill, residual-risk review, and release approval. <!-- t:2smn -->
- [x] Moved the BBS schema, API, and security contract to TINYIDP-BBS-001. <!-- t:ojkr -->
- [x] Superseded native Go BBS handlers with trusted planned xgoja routes in TINYIDP-BBS-001. <!-- t:ejpq -->
- [x] Moved BBS Durable Object implementation tracking to TINYIDP-BBS-001. <!-- t:x51w -->
- [x] Moved the React frontend implementation to TINYIDP-BBS-001. <!-- t:m06c -->
- [x] Moved the monochrome visual system to TINYIDP-BBS-001. <!-- t:le8m -->
- [x] Moved BBS integration and security tests to TINYIDP-BBS-001. <!-- t:s09k -->
- [x] Moved the Alice/Bob browser scenario to TINYIDP-BBS-001. <!-- t:pg4w -->
- [x] Moved BBS generation, gates, documentation, and tmux delivery to TINYIDP-BBS-001. <!-- t:z9v7 -->
