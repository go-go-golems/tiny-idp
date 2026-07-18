---
Title: Authorization interaction hardening implementation ledger
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Primary authorization interaction and adjacent endpoint implementation target
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Public one-time interaction persistence contract target
    - Path: repo://pkg/sqlitestore/store.go
      Note: Durable interaction and protocol atomicity implementation target
ExternalSources: []
Summary: Task-by-task implementation ledger for replacing browser-owned authorization continuation, enforcing required actions, hardening adjacent endpoints, and proving protocol persistence behavior.
LastUpdated: 2026-07-10T19:40:00-04:00
WhatFor: Tracking precise implementation, validation, diary, and commit progress for authorization interaction hardening.
WhenToUse: Read before changing Provider.authorize, browser sessions, consent, interaction persistence, token limiting, UserInfo, or Fosite lifecycle storage.
---




# Authorization interaction hardening implementation ledger

## Goal

Complete items 1 through 5 from the production-hardening sequence: write the
regressions, add a server-owned one-time interaction, remove browser-owned
continuation, revalidate mutable state, and harden adjacent protocol paths. Each
task below has a stable docmgr ID and an independently reviewable exit condition.

## Non-negotiable invariants

- A required fresh authentication cannot disappear between GET and POST.
- Browser input does not carry the authoritative OAuth/OIDC request.
- Invalid, negative, or overflowing `max_age` never weakens authentication.
- `prompt=none` never renders interaction UI.
- Consent is an explicit approve or deny action bound to a validated request.
- One interaction has at most one terminal consumption under concurrency.
- Resume revalidates mutable client, redirect, user, session, and key state.
- UserInfo bearer transport and token rate-limit identity are explicit.
- Authorization-code, PKCE, and OIDC state is complete or absent after failure.
- No compatibility fallback retains the hidden-field continuation.

## Phase 0 — semantics and baseline

### Task `omhr`: baseline evidence

- Record branch, dirty files, targeted/full test commands, and confirmed current
  reproductions in the implementation diary.
- Preserve unrelated untracked hosted-conformance artifacts.
- Exit: exact baseline is reproducible and the diary names all known failures.

### Task `lrt0`: accepted behavior contract

- Define fresh-login ordering and `auth_time` provenance.
- Define strict `max_age` parse/range semantics.
- Define `prompt=none`, consent denial, UserInfo method/transport, and terminal
  interaction outcomes.
- Exit: tests can assert product behavior without guessing from current code.

## Phase 1 — executable regressions

### Task `40vm`: browser harness

- Preserve CSRF/session/interaction cookies.
- Extract the opaque interaction handle and submit explicit actions.
- Inspect redirects, OAuth errors, cookies, and durable state.
- Exit: later tests express state transitions rather than duplicate HTTP setup.

### Task `yo4c`: forced-login regressions

- Existing session plus `prompt=login` and blank submit.
- Existing session plus crafted submit that omits credentials.
- Exit: neither path issues a code or reuses the old `auth_time`.

### Task `b7h5`: `max_age` regressions

- Expired `max_age` plus blank submit.
- Invalid, negative, and overflowing values.
- Invalid request plus `max_age` must not collect credentials.
- Exit: required login is enforced and malformed input is rejected.

### Task `go72`: non-interactive regressions

- `prompt=none` with missing login.
- `prompt=none` with required consent.
- Invalid `prompt` combinations.
- Exit: no interaction HTML is rendered.

### Task `gm7y`: consent regressions

- Explicit approve, explicit deny, and omitted decision.
- Displayed client/scope binding.
- Exit: denial returns `access_denied`; omission cannot approve.

### Task `cnzp`: mutation, replay, and concurrency

- Mutate client, redirect, scope, nonce, PKCE, and state on resume.
- Replay terminal handles.
- Submit one handle concurrently and exercise two browser tabs.
- Exit: server-owned request wins and at most one consume succeeds.

## Phase 2 — interaction persistence

### Task `bn89`: domain types

- Add `InteractionRecord`, required-action bits, terminal/consume state, canonical
  request representation, request digest, timestamps, and browser binding.
- Exit: types contain no Fosite, HTTP, SQL, or secret raw handles.

### Task `zaw5`: public store contracts

- Add create, get, and atomic consume to `StoreOperations`, `ReadStore`, and
  transaction-scoped interfaces.
- Exit: exactly-once semantics are an explicit public persistence operation.

### Task `sidg`: memory implementation

- Copy inputs/outputs, enforce duplicate/expiry/consume behavior, and support
  transaction snapshots.
- Exit: store contract and concurrent-consume tests pass.

### Task `k8e6`: SQLite migration

- Add a checksum-tracked interaction table and expiry/consumption indexes.
- Exit: new and upgraded databases report the new supported schema version.

### Task `ce20`: SQLite implementation

- Implement create/get/conditional atomic consume.
- Test duplicate, expiry, replay, rollback, and concurrent consumption.
- Exit: exactly one concurrent consume returns success.

### Task `ebk2`: lifecycle integration

- Include interactions in maintenance and backup/restore evidence.
- Exit: expired terminal records are retained/purged by documented policy.

## Phase 3 — provider migration

### Task `qt3o`: canonical request

- Copy every validated Fosite request form value into a normalized server record.
- Compute a deterministic digest with sorted keys and preserved repeated values.
- Exit: GET and reconstructed POST requests have the same digest.

### Task `60z0`: interaction creation and rendering

- Create the interaction after Fosite validation and requirement calculation.
- Render only CSRF, opaque interaction handle, and explicit action controls.
- Exit: HTML contains none of the authoritative OAuth/OIDC fields.

### Task `gaax`: server-owned resume

- Load by keyed handle hash.
- Reconstruct a synthetic POST request solely from canonical stored values.
- Exit: submitted OAuth fields are ignored/rejected and cannot change behavior.

### Task `vi2k`: required fresh authentication

- Require credentials when the stored action set requires fresh login.
- Set `auth_time` only from the new successful authentication.
- Exit: old browser sessions cannot satisfy forced reauthentication.

### Task `14do`: explicit consent

- Bind approve/deny to the interaction.
- Use Fosite OAuth error writing for safe `access_denied` redirects.
- Exit: consent omission/denial never issues an artifact.

### Task `jres`: mutable-state revalidation

- Re-fetch client and compare registration/redirect.
- Re-fetch user/session state and active signing readiness.
- Exit: disabled/revoked/changed state cannot complete an old interaction.

### Task `n0ua`: terminal consume

- Atomically consume before irreversible response creation.
- Distinguish not found, expired, replayed, and infrastructure errors.
- Exit: one interaction produces at most one terminal attempt.

### Task `l8kv`: delete hidden continuation

- Remove `hidden(ar)` and every compatibility branch.
- Exit: search and analyzer checks find no browser-owned continuation path.

## Phase 4 — adjacent path hardening

### Task `mpvu`: strict `max_age`

- Parse once into a typed policy.
- Reject invalid/negative/out-of-range values.
- Compare instants without `time.Duration` overflow.
- Exit: table and fuzz tests pass at numeric boundaries.

### Task `bhl2`: token limiter identity

- Always apply a stable client-address/global pre-authentication bucket.
- Apply a validated client bucket after Fosite authenticates the client.
- Exit: changing form `client_id` cannot obtain a fresh expensive-work budget.

### Task `4156`: UserInfo contract

- Permit documented GET/POST methods only.
- Extract bearer credentials explicitly from `Authorization`.
- Reject query/form credentials, add `no-store`, and return RFC challenges.
- Exit: method/transport/cache/challenge matrix passes.

### Task `6qtj`: browser-session errors

- Preserve internal distinctions for absent, expired, revoked, disabled, corrupt,
  and unavailable stores while keeping safe external messages.
- Exit: infrastructure failure cannot silently behave like an anonymous browser.

## Phase 5 — Fosite lifecycle atomicity and release validation

### Task `altv`: map mutation lifecycle

- Trace exact authorize handler order and code keys.
- Select transaction propagation or compensation design without exposing raw SQL.
- Exit: an accepted decision record documents the integration seam.

### Task `exej`: atomic/compensated persistence

- Make code, PKCE, and OIDC writes one transaction or provide idempotent rollback.
- Exit: the storage contract represents one lifecycle unit.

### Task `ds0r`: failpoints

- Inject before/after each write and at commit.
- Record named failure point in test traces.
- Exit: every mutation boundary is addressable and reproducible.

### Task `f69o`: all-or-none proof by test

- Inspect durable code, PKCE, OIDC, interaction, and audit state after every
  injected failure.
- Exit: each state group is complete or absent and has one terminal outcome.

### Task `ah32`: validation and evidence

- Run targeted tests, full tests, race, shuffle/repeat, analyzers, external
  consumer, and relevant conformance smoke.
- Update diary, changelog, task status, related files, and candidate evidence.
- Exit: commits and evidence name the exact validated revision.

## Commit boundaries

1. Ledger and failing regressions.
2. Interaction domain/store contracts plus memory/SQLite persistence.
3. Provider opaque-continuation migration and consent/revalidation behavior.
4. Adjacent token/UserInfo/session hardening.
5. Fosite lifecycle atomicity, failpoints, and final evidence.

## Review commands

```bash
go test ./internal/fositeadapter -run 'Interaction|Prompt|MaxAge|Consent|UserInfo|RateLimit' -count=1
go test ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore -run 'Interaction|Migration|Maintenance|Backup' -count=1
go test ./... -count=1
go test -race ./... -count=1
go test ./... -shuffle=on -count=10
docmgr doctor --ticket TINYIDP-PROD-IMPL-001 --stale-after 30
```

## Related

- `reference/04-authorization-interaction-and-protocol-robustness-review.md`
- `design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`
- `reference/01-implementation-diary.md`
