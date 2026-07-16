# Tasks

## Phase 0 — Contract and baseline

- [x] Map current xapp route composition, actor-to-durable-object binding, initialized state, and tiny-idp device/introspection endpoints.
- [x] Publish the intern-ready architecture, threat model, API contract, and decision records.
- [x] Record the executable acceptance criteria, risks, and test matrix in the ticket.
- [x] Run and record the focused current xapp test baseline before behavior changes.

## Phase 1 — Resource authentication core

- [x] Define Go-only immutable resource-auth configuration, principal, failure categories, and context key.
- [x] Implement strict single-Bearer parsing and deterministic `401`/`403`/`503` response helpers without credential disclosure.
- [x] Implement discovery validation, timeout-bounded authenticated RFC 7662 client, and constrained response decoder.
- [x] Validate active, exact issuer, Bearer type, exact audience membership, subject, expiry, and route scopes.
- [x] Add HMAC-keyed positive decision cache bounded by token expiry and maximum TTL; add bounded negative cache policy only for definitive inactive results.
- [x] Add table-driven tests for parser, discovery, active/inactive, issuer/audience/type/expiry/scope, provider authentication, unavailable provider, redaction, and cache bounds.

## Phase 2 — xapp state and IDP registrations

- [x] Define device client ID, resource client ID, and exact API audience derivation in one package-local configuration location.
- [x] Extend development bootstrap with distinct browser, public device, and confidential introspection resource clients.
- [x] Extend initialized state manifest/paths with API identity settings and a generated owner-only resource-client secret not serialized into the manifest.
- [x] Bump and validate state schema deliberately; document and implement the selected initialized-state migration/reinitialization behavior.
- [x] Construct resource authentication in development through the in-process provider transport and in initialized mode through the production transport policy.
- [x] Test bootstrap client roles/audiences and state file permissions without leaking generated secrets.

## Phase 3 — Bearer API and durable-object bridge

- [x] Add host-owned read endpoint for the BBS requiring `bbs.read` bearer scope.
- [x] Add host-owned post endpoint requiring `bbs.post.create` bearer scope and bounded JSON input validation.
- [x] Derive BBS actor ID and author only from verified principal data; forbid caller-selected identity fields.
- [x] Preserve browser routes and their CSRF requirement; ensure no cookie-session fallback on bearer endpoints.
- [x] Emit redacted API authentication and BBS audit events.
- [ ] Add integration tests proving denied bearer requests do not dispatch to the durable object and accepted posts carry the token subject.

## Phase 4 — Device CLI and owner-only token cache

- [ ] Add Glazed `device-login` command with discovery, RFC 8628 start/poll semantics, browser instructions, and terminal error handling.
- [ ] Add owner-only token-cache load/write/expiry validation with explicit flags and no environment configuration.
- [ ] Add Glazed BBS read command using cached bearer token.
- [ ] Add Glazed BBS post command with title/body/category validation and stable output.
- [ ] Unit test polling interval, `slow_down`, denial/expiry, malformed discovery, cache mode, and HTTP request formation.

## Phase 5 — End-to-end and regression verification

- [ ] Add a ticket-owned tmux/smoke harness that starts the app, completes device approval through a real browser, and invokes CLI read/post.
- [ ] Add Playwright coverage for browser login, BBS mutation, logout, and the unchanged browser CSRF boundary.
- [ ] Assert a device-authenticated second user posts a message authored by that subject after switching from the browser user.
- [ ] Exercise wrong-audience, insufficient-scope, revoked/expired, and malformed-token denial cases in an application-level suite.
- [ ] Exercise initialized TLS production mode with device discovery, approval, token polling, introspection, and API post.

## Phase 6 — Handoff and reusable-primitives decision

- [ ] Write an operator runbook with development and initialized-mode commands, secret locations, audit expectations, and incident responses.
- [ ] Publish an evidence-backed extraction recommendation for `go-go-goja` based on the actual host interfaces used here.
- [ ] Relate source, test, and script files to this ticket; update diary/changelog after each committed phase.
- [ ] Run `docmgr doctor`, upload the final bundle to reMarkable, and close the ticket only after all acceptance criteria pass.

## TODO
