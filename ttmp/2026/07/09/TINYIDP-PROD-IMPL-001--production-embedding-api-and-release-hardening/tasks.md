# Tasks

This file is the durable execution ledger. Check a task only when its stated
verification is present in the repository or ticket. Phase gates are separate
tasks so partial implementation never looks like phase completion.

## Ticket foundation and design

- [x] Create the implementation ticket, design document, diary, and phase ledger.
- [x] Write the intern-oriented architecture, API, and implementation guide.
- [x] Relate the implementation guide and diary to the production files that shape the design.
- [x] Validate the ticket with `docmgr doctor` and resolve actionable findings.
- [x] Dry-run and upload the implementation guide bundle to reMarkable.

## Phase 0 — dependency and toolchain security baseline

- [x] Record the current selected Go, Fosite, go-jose, SQLite, and CGO dependency graph.
- [x] Pin the supported release Go patch level in repository and CI configuration.
- [x] Upgrade `github.com/go-jose/go-jose/v3` to v3.0.5 or later through the smallest compatible dependency change.
- [x] Run `go mod tidy` and review every module-graph change.
- [x] Run build, unit, vet, race, lint, Staticcheck, and ticket audit analyzers on the exact graph.
- [x] Run `govulncheck` and require zero reachable known vulnerabilities or a documented expiry-bound exception.
- [x] Run existing parser fuzz seeds and strict local conformance smoke tests.
- [x] Add or update CI so the exact supported Go patch level and `govulncheck` are release gates.
- [x] Produce or wire an SBOM and provenance record for release artifacts.
- [x] Phase 0 gate: commit evidence for a reproducible, vulnerability-clean release graph.

## Phase 1 — consumable public embedding API

- [x] Inventory every type currently crossing `pkg/embeddedidp` into `internal/` packages.
- [x] Define public identity, policy, audit, limiter, authenticator, and store contracts without Fosite types.
- [x] Create `pkg/idp` for stable public policy and runtime contracts.
- [x] Create `pkg/idpstore` for public records, read contracts, transactions, and invariant operations.
- [x] Create `pkg/sqlitestore` as the supported durable implementation.
- [x] Replace `embeddedidp.Options` with public or standard-library types only.
- [x] Change construction to `New(ctx, Options)` and propagate startup cancellation.
- [x] Add `Readiness(ctx)` with structured component results.
- [x] Add idempotent `Close(ctx)` and document host/provider lifecycle ownership.
- [x] Fail production construction on missing audit, limiter, client-address, secret, schema, key, or persistent-store requirements.
- [x] Delete the unusable pre-release surface directly; add no compatibility shim.
- [x] Update repository examples and README to use only public packages.
- [x] Convert the external-consumer failure probe into a separate-module positive integration test.
- [x] Complete Authorization Code + PKCE through the external-module fixture backed by public SQLite.
- [x] Phase 1 gate: an external application compiles, starts, checks readiness, completes strict OIDC, and shuts down cleanly.

## Phase 2 — transactional persistence, backup, and restore

- [x] Document every security transition spanning multiple SQL statements or tables.
- [x] Define `View` and `Update` transaction boundaries without exposing raw driver types.
- [x] Implement atomic user-plus-credential creation.
- [x] Implement atomic password-plus-security-state replacement.
- [x] Implement atomic failed-login increment and lockout derivation.
- [x] Implement atomic success reset and session creation where required.
- [x] Implement atomic refresh rotation and reuse-family revocation.
- [x] Implement atomic signing-key activation, retirement, and last-key protection.
- [x] Add schema-level active-key and uniqueness protections where SQLite permits them.
- [x] Add explicit schema versions, migration ordering, and migration checksums.
- [x] Make migrations transactional and add upgrade failure-injection tests.
- [x] Replace raw file-copy backup with SQLite online backup semantics.
- [x] Write backups to owner-only temporary files and atomically publish after `fsync`.
- [x] Verify backups read-only with `integrity_check`, schema version, and source manifest comparisons.
- [x] Add a restore command/path that refuses incompatible or failed verification artifacts.
- [x] Enforce owner-only database, WAL, SHM, backup directory, and backup file permissions.
- [x] Configure and document busy timeout, journal mode, synchronous policy, connection limits, and supported filesystems.
- [x] Add concurrent-writer, WAL backup, restore, corruption, disk-full, busy-lock, and interruption tests.
- [x] Phase 2 gate: invariant, failure-injection, backup, and restore suites pass without silent partial state.

## Phase 3 — mandatory authentication and abuse controls

- [ ] Define password acceptance policy separately from Argon2id encoding parameters.
- [ ] Enforce password policy on user creation, reset, and password change.
- [ ] Implement a real must-change-password flow or remove the unsupported state.
- [ ] Require a production rate limiter; reject nil or permissive defaults.
- [ ] Define trusted-proxy configuration and a public client-address resolver contract.
- [ ] Rate-limit by account, client, and trusted client address without leaking account existence.
- [ ] Bound concurrent Argon2id work with a context-aware semaphore.
- [ ] Export saturation, wait, rejection, and duration metrics for password work.
- [ ] Make storage failures in lockout/reset paths fail closed and observable.
- [ ] Define password-change revocation behavior for sessions, authorization codes, and refresh families.
- [ ] Add simultaneous failed-login tests that prove no lost updates.
- [ ] Add abuse/load tests at production Argon2id parameters and memory limits.
- [ ] Convert the security-invariants probe from reproducing gaps to asserting protections.
- [ ] Phase 3 gate: authentication controls are mandatory, atomic, bounded, observable, and load-tested.

## Phase 4 — keys, audit, readiness, and maintenance

- [ ] Validate signing-key algorithm, size, parseability, not-before, expiry, and active uniqueness at startup.
- [ ] Prevent retiring the final usable signing key.
- [ ] Define rotation overlap and published-JWKS retention for still-valid tokens.
- [ ] Require a production audit sink and define delivery, buffering, dropping, backpressure, and health semantics.
- [ ] Propagate or surface audit delivery failures according to the accepted policy.
- [ ] Add structured readiness checks for store, schema, signing keys, secret sources, audit, limiter, and maintenance state.
- [ ] Separate liveness from readiness and document orchestration behavior.
- [ ] Implement retention/maintenance for expired sessions, codes, tokens, requests, and audit buffers.
- [ ] Resolve effective `SameSite`, per-client TTL, issuer/path, RNG-error, and route-registration contracts.
- [ ] Add key-expiry, rotation, audit-failure, maintenance, and readiness transition tests.
- [ ] Phase 4 gate: unsafe configuration cannot report ready and critical lifecycle operations have observable health.

## Phase 5 — release engineering and deployment proof

- [ ] Add always-on CI for build, unit, vet, lint, custom analyzers, `govulncheck`, fuzz seeds, external consumer, and backup/restore.
- [ ] Add release CI for race, longer fuzzing, concurrency/fault injection, hosted conformance, and restore drills.
- [ ] Build a production host example with TLS expectations, HTTP timeouts, request limits, proxy trust, and graceful shutdown.
- [ ] Document the single-active-node SQLite deployment envelope and unsupported topologies.
- [ ] Run sustained login/token/read load with production Argon2id settings and capture runtime/DB/audit metrics.
- [ ] Run OpenID Foundation conformance against the exact release candidate artifact.
- [ ] Perform backup restore, migration, signing-key rotation, token-secret rotation, downgrade, and rollback drills.
- [ ] Write operator runbooks for corruption, key compromise, dependency emergencies, and administrative lockout.
- [ ] Produce signed artifacts, checksums, SBOM, provenance, toolchain manifest, and dependency licenses.
- [ ] Assemble a release evidence packet with artifact hashes and links to every gate result.
- [ ] Obtain independent security/code review and explicit release-owner sign-off.
- [ ] Phase 5 gate: production-like deployment and incident drills pass on the exact signed release candidate.

## Final production decision

- [ ] Re-run the full production review against the release candidate.
- [ ] Confirm no P0 or unaccepted P1 findings remain.
- [ ] Record residual risks, owners, expiry dates, and rollback criteria.
- [ ] Mark the release candidate approved only after every phase gate is checked.
