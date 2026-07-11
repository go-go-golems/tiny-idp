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

- [x] Define password acceptance policy separately from Argon2id encoding parameters.
- [x] Enforce password policy on user creation, reset, and password change.
- [x] Implement a real must-change-password flow or remove the unsupported state.
- [x] Require a production rate limiter; reject nil or permissive defaults.
- [x] Define trusted-proxy configuration and a public client-address resolver contract.
- [x] Rate-limit by account, client, and trusted client address without leaking account existence.
- [x] Bound concurrent Argon2id work with a context-aware semaphore.
- [x] Export saturation, wait, rejection, and duration metrics for password work.
- [x] Make storage failures in lockout/reset paths fail closed and observable.
- [x] Define password-change revocation behavior for sessions, authorization codes, and refresh families.
- [x] Add simultaneous failed-login tests that prove no lost updates.
- [x] Add abuse/load tests at production Argon2id parameters and memory limits.
- [x] Convert the security-invariants probe from reproducing gaps to asserting protections.
- [x] Phase 3 gate: authentication controls are mandatory, atomic, bounded, observable, and load-tested.

## Phase 4 — keys, audit, readiness, and maintenance

- [x] Validate signing-key algorithm, size, parseability, not-before, expiry, and active uniqueness at startup.
- [x] Prevent retiring the final usable signing key.
- [x] Define rotation overlap and published-JWKS retention for still-valid tokens.
- [x] Require a production audit sink and define delivery, buffering, dropping, backpressure, and health semantics.
- [x] Propagate or surface audit delivery failures according to the accepted policy.
- [x] Add structured readiness checks for store, schema, signing keys, secret sources, audit, limiter, and maintenance state.
- [x] Separate liveness from readiness and document orchestration behavior.
- [x] Implement retention/maintenance for expired sessions, codes, tokens, requests, and audit buffers.
- [x] Resolve effective `SameSite`, per-client TTL, issuer/path, RNG-error, and route-registration contracts.
- [x] Add key-expiry, rotation, audit-failure, maintenance, and readiness transition tests.
- [x] Phase 4 gate: unsafe configuration cannot report ready and critical lifecycle operations have observable health.

## Phase 5 — release engineering and deployment proof

- [x] Add always-on CI for build, unit, vet, lint, custom analyzers, `govulncheck`, fuzz seeds, external consumer, and backup/restore.
- [x] Add release CI for race, longer fuzzing, concurrency/fault injection, hosted conformance, and restore drills.
- [x] Build a production host example with TLS expectations, HTTP timeouts, request limits, proxy trust, and graceful shutdown.
- [x] Document the single-active-node SQLite deployment envelope and unsupported topologies.
- [x] Run sustained login/token/read load with production Argon2id settings and capture runtime/DB/audit metrics.
- [ ] Run OpenID Foundation conformance against the exact release candidate artifact.
- [x] Perform backup restore, migration, signing-key rotation, token-secret rotation, downgrade, and rollback drills.
- [x] Write operator runbooks for corruption, key compromise, dependency emergencies, and administrative lockout.
- [ ] Produce signed artifacts, checksums, SBOM, provenance, toolchain manifest, and dependency licenses.
- [x] Assemble a release evidence packet with artifact hashes and links to every gate result.
- [ ] Obtain independent security/code review and explicit release-owner sign-off.
- [ ] Phase 5 gate: production-like deployment and incident drills pass on the exact signed release candidate.

## Final production decision

- [x] Re-run the full production review against the release candidate.
- [ ] Confirm no P0 or unaccepted P1 findings remain.
- [x] Record residual risks, owners, expiry dates, and rollback criteria.
- [ ] Mark the release candidate approved only after every phase gate is checked.
- [x] Add failing regressions for forced reauthentication POST bypass, invalid/negative/overflow max_age, and invalid request plus max_age credential-form behavior. <!-- t:dcd5 -->
- [x] Replace browser-hidden authorization continuation with an opaque expiring one-time server-side interaction record; add no compatibility fallback. <!-- t:1y0n -->
- [x] Implement explicit approve/deny consent bound to the validated client and displayed scopes, with OAuth access_denied semantics. <!-- t:dfg5 -->
- [x] Add deterministic-clock authorization state-machine, mutation, replay, concurrent-tab, duplicate-parameter, property, and fuzz tests. <!-- t:zg5j -->
- [x] Harden token endpoint rate-limit identities and UserInfo method, bearer transport, cache, and challenge contracts. <!-- t:scll -->
- [x] Add Fosite authorize/token lifecycle fault injection and prove or enforce atomic/compensated protocol persistence across handler boundaries. <!-- t:3uhz -->
- [ ] Rerun external-module flows, race, analyzers, fuzzing, recovery, and hosted OIDF against the new exact artifact hash. <!-- t:9ajq -->
- [x] Authorization interaction hardening gate: no invalid request collects credentials, required actions cannot disappear, and one interaction has at most one terminal outcome. <!-- t:3ufn -->
- [ ] Assurance Phase 0: review and accept the invariant catalog, threat model, event schema, consent semantics, UserInfo transport, and release profile <!-- t:uum1 -->
- [x] Assurance Phase 1: implement a server-owned hashed one-time authorization InteractionRecord with canonical request digest, required actions, expiry, generation, and atomic terminal transitions <!-- t:iosc -->
- [x] Assurance Phase 1: replace browser hidden continuation without a compatibility fallback and add forced-login, max_age, prompt-none, consent, mutation, and replay regressions <!-- t:2cd9 -->
- [x] Assurance Phase 2: add an injected security Clock, pure Go interaction model, strict-provider scenario driver, typed actions, and observations <!-- t:qru9 -->
- [x] Assurance Phase 2: add Rapid state-machine properties, metamorphic relations, native fuzz action sequences, seed persistence, and shrunk replay tests <!-- t:tlmk -->
- [x] Assurance Phase 3: extend auditlint with strict security parsing, explicit bearer transport, and injected-clock analyzers plus analysistest fixtures <!-- t:j7dm -->
- [x] Assurance Phase 3: add interaction-continuation, limiter-identity taint, ignored-security-error, and protocol-lifecycle analyzers with documented precision limits <!-- t:mo8x -->
- [x] Assurance Phase 4: implement versioned secret-free SecurityEvent instrumentation for interaction, authentication, consent, protocol mutation, and terminal boundaries <!-- t:6ruz -->
- [x] Assurance Phase 4: build a typed parametric offline trace monitor and feed deterministic, property, fuzz, and failpoint executions through it <!-- t:671q -->
- [x] Assurance Phase 5: add test scheduling probes and Porcupine linearizability histories for interaction consumption and refresh rotation <!-- t:zbyl -->
- [x] Assurance Phase 5: enumerate Fosite authorization lifecycle failpoints and verify all-or-none code, PKCE, OIDC, audit, and terminal state <!-- t:nvcc -->
- [x] Assurance Phase 6: integrate the isolated tinyidp/verify VerificationPlan runner after native interaction semantics stabilize <!-- t:1qfc -->
- [ ] Assurance Phase 7: run exact-candidate static, race, fuzz, failpoint, local and hosted OIDF, reverse-proxy, and generic web gates <!-- t:13it -->
- [ ] Assurance Phase 8: canary with native guards, shadow monitors, audit delivery verification, rollback drills, and signed residual-risk approval <!-- t:bwsn -->
- [x] AH Phase 0.1: capture baseline tests, current branch status, and confirmed interaction defects in the diary <!-- t:omhr -->
- [x] AH Phase 0.2: define accepted semantics for fresh login, max_age, prompt none, consent denial, UserInfo transport, and terminal outcomes <!-- t:lrt0 -->
- [x] AH Phase 1.1: add a reusable strict-provider browser test harness that preserves cookies and parses opaque interactions <!-- t:40vm -->
- [x] AH Phase 1.2: add forced prompt login blank and crafted POST regressions with an existing browser session <!-- t:yo4c -->
- [x] AH Phase 1.3: add expired max_age blank POST and invalid negative overflow max_age regressions <!-- t:b7h5 -->
- [x] AH Phase 1.4: add prompt none login and consent required non-interaction regressions <!-- t:go72 -->
- [x] AH Phase 1.5: add explicit consent denial and omitted decision regressions <!-- t:gm7y -->
- [x] AH Phase 1.6: add continuation mutation replay concurrent duplicate and concurrent-tab regressions <!-- t:cnzp -->
- [x] AH Phase 2.1: define InteractionRecord required-action terminal-state and canonical-request public types <!-- t:bn89 -->
- [x] AH Phase 2.2: add InteractionStore create get and atomic consume contracts to Store ReadStore and TxStore <!-- t:zaw5 -->
- [x] AH Phase 2.3: implement memory interaction persistence with copy isolation expiry and exactly-once consume <!-- t:sidg -->
- [x] AH Phase 2.4: add SQLite interaction migration indexes retention metadata and checksum <!-- t:k8e6 -->
- [x] AH Phase 2.5: implement SQLite create get consume and concurrent consume tests <!-- t:ce20 -->
- [x] AH Phase 2.6: include expired interactions in maintenance and backup restore validation <!-- t:ebk2 -->
- [x] AH Phase 3.1: canonicalize validated Fosite authorization forms and compute a stable server-side request digest <!-- t:qt3o -->
- [x] AH Phase 3.2: create interactions on GET and render only opaque handle plus CSRF and explicit action fields <!-- t:60z0 -->
- [x] AH Phase 3.3: load the stored interaction on POST and reconstruct Fosite input only from server-owned canonical values <!-- t:gaax -->
- [x] AH Phase 3.4: enforce required fresh login after interaction creation and preserve authoritative auth_time <!-- t:vi2k -->
- [x] AH Phase 3.5: implement explicit consent approve deny and OAuth access_denied response semantics <!-- t:14do -->
- [x] AH Phase 3.6: revalidate client redirect user session signing readiness and generation before terminal consume <!-- t:jres -->
- [x] AH Phase 3.7: atomically consume the interaction before artifact issuance and reject replay expiry or browser mismatch <!-- t:n0ua -->
- [x] AH Phase 3.8: remove hidden authorization continuation with no compatibility fallback <!-- t:l8kv -->
- [x] AH Phase 4.1: replace max_age boolean helper with strict parsed policy and overflow-safe comparison <!-- t:mpvu -->
- [x] AH Phase 4.2: make token pre-authentication rate limiting use stable address and authenticated client dimensions <!-- t:bhl2 -->
- [x] AH Phase 4.3: enforce UserInfo GET POST policy explicit Authorization header bearer extraction no-store and RFC challenge responses <!-- t:4156 -->
- [x] AH Phase 4.4: classify session store absence expiry revocation disabled user and infrastructure failure without fail-open collapse <!-- t:6qtj -->
- [x] AH Phase 5.1: identify the exact Fosite authorization response mutation sequence and transaction key propagation options <!-- t:altv -->
- [x] AH Phase 5.2: implement atomic authorization code PKCE and OIDC persistence or explicit compensation without public raw SQL <!-- t:exej -->
- [x] AH Phase 5.3: add named before after and commit failure injection across authorization persistence <!-- t:ds0r -->
- [x] AH Phase 5.4: prove all-or-none durable state and one terminal outcome under every injected failure <!-- t:f69o -->
- [x] AH Phase 5.5: run targeted full race shuffle and external consumer validation and update candidate evidence <!-- t:ah32 -->
