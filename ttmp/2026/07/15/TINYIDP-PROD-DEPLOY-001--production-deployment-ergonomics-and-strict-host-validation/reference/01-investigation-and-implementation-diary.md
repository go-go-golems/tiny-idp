---
Title: Investigation and Implementation Diary
Ticket: TINYIDP-PROD-DEPLOY-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - operations
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/device_authorization_test.go
      Note: SQLite browser continuation through token, UserInfo, and replay regression
    - Path: repo://internal/fositeadapter/device_token_handler.go
      Note: Moves ID-token signing before the one-connection SQLite token transaction
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: Regression test prevents device-token signing self-deadlock
    - Path: repo://ttmp/2026/07/15/TINYIDP-PROD-DEPLOY-001--production-deployment-ergonomics-and-strict-host-validation/scripts/01-provision-local-strict-host.sh
      Note: Exact local fixture provisioning sequence
    - Path: repo://ttmp/2026/07/15/TINYIDP-PROD-DEPLOY-001--production-deployment-ergonomics-and-strict-host-validation/scripts/02-run-local-strict-host.sh
      Note: Foreground strict-host launch command
ExternalSources: []
Summary: Chronological evidence for strict-host provisioning, naming, smoke testing, and unresolved release blockers.
LastUpdated: 2026-07-15T18:10:00Z
WhatFor: Preserve exact operational commands, outcomes, and review guidance for strict-host deployment work.
WhenToUse: Read before resuming this ticket or evaluating strict device authorization for release.
---




# Investigation and Implementation Diary

## Goal

Record the work needed to make `serve-production` operationally discoverable
and to distinguish it from the local development server.

## Context

`serve-production` already enforced the important production prerequisites in
code. This diary captures the first real assembly of all of them, rather than
claiming readiness from unit tests alone.

## Quick Reference

## Step 1: Establish an explicit strict-host deployment path

The work began by inspecting the executable host, administration commands,
embedded-provider validation, and SQLite store. The conclusion was that the
host was buildable immediately: it did not need a new cache or a replacement
server. It needed unambiguous naming, a reproducible provisioning sequence,
and real-process evidence.

I created this ticket, changed the local-only command name to `serve-dev`, and
added a local strict-host fixture plus a least-privilege systemd template. The
browser part of the device flow passed against a direct-TLS host. The token
portion did not: after approval, later requests became unresponsive. That is
recorded as a release blocker rather than being hidden by the successful page.

### Prompt Context

**User prompt (verbatim):** "ok, do it."

**Assistant interpretation:** Implement the proposed production-deployment
ergonomics work, including a clear server-command surface and a real strict
host validation path.

**Inferred user intent:** Make tinyidp operable with a production-quality
workflow and collect honest evidence rather than relying on an in-memory demo.

**Commit (code):** b886362 — "feat(deploy): clarify strict production hosting"

### What I did

- Inspected `internal/cmds/serve_production.go`, `internal/cmds/admin*.go`,
  `pkg/embeddedidp/options.go`, and `pkg/sqlitestore/store.go`.
- Created ticket `TINYIDP-PROD-DEPLOY-001` with this diary, a design guide,
  tasks, and numbered scripts.
- Renamed the local command from `serve` to `serve-dev` in
  `internal/cmds/serve.go` and `cmd/tinyidp/main.go`, deliberately without a
  compatibility alias.
- Updated maintained README, CLI-guide, conformance, security, admin, and
  production-host references to use `serve-dev`.
- Added `scripts/01-provision-local-strict-host.sh`, which creates a fresh
  owner-only local fixture and provisions schema, RSA key, user, device client,
  and doctor check; added foreground launcher and TLS-verifying probe scripts.
- Added `examples/production-host/systemd/tinyidp.service` with a dedicated
  user, strict filesystem access, state/log directories, restricted address
  families, and direct secret/certificate file paths.
- Ran the real provisioning commands in `/tmp/tinyidp-prod-smoke-20260715`.
  `admin doctor` reported schema version 9, one valid client, one active key,
  and one verification key.
- Started `serve-production` in tmux at `https://localhost:9443`, using the
  isolated SQLite database, audit file, secret, and self-signed localhost
  certificate.
- Created a real device authorization request. Playwright reached the expected
  self-signed-certificate interstitial, then explicitly proceeded for the
  disposable local host. Its accessibility snapshot showed the requested
  client `device-smoke`, scopes `openid` and `profile`, username/password
  fields, and Approve/Deny buttons. Submitting `smoke` and the fixture password
  reached `Device verification complete`.

### Why

- A command named `serve` made it too easy to mistake a development server for
  a deployment host. Explicit command names are a security control for human
  operators.
- Production prerequisite failures must occur before the listener starts. The
  admin lifecycle and `embeddedidp.Options.Validate` already supplied this
  property; the scripts make it repeatable.
- A direct-TLS local fixture is the closest safe smoke topology to the current
  host design. It exercises actual SQLite, file permissions, TLS, audit, and
  browser behavior without inventing a proxy contract.

### What worked

- The exact provisioning sequence worked:

  ```text
  init --generate-signing-key
  user create --password-from-stdin
  client create --public --grant-type urn:ietf:params:oauth:grant-type:device_code
  doctor
  serve-production with secret, TLS, SQLite, and audit paths
  ```

- `admin doctor` output was fully healthy: schema version 9, valid client,
  active key, and verification key.
- The strict host emitted `device.authorization.created`,
  `password.login.success`, and `device.verification.approved` audit events.
- Playwright successfully rendered and submitted the real browser verification
  page. This confirmed the flow was not merely an HTTP-unit-test artifact.
- Focused verification passed:

  ```text
  sh -n .../scripts/01-provision-local-strict-host.sh
  go test ./cmd/tinyidp ./internal/cmds ./pkg/embeddedidp -count=1
  ```

### What didn't work

- A `curl -k` device token request exited without a visible body. A repeated
  timed request showed:

  ```text
  curl: (28) Operation timed out after 8000 milliseconds with 0 bytes received
  curl_exit=28 http_status=000 bytes=0
  ```

- The same browser context subsequently reported `TypeError: Failed to fetch`
  for `fetch('/readyz')`. The audit log had the successful approval event but
  no token-redemption event.
- I sent `SIGQUIT` to the disposable Go child process to obtain a goroutine
  dump, but the tmux session exited before `capture-pane` could retain it:

  ```text
  can't find pane: tinyidp-prod-device-smoke
  ```

  This was an evidence-collection failure, not a reason to downgrade the
  observed unresponsiveness.
- `docmgr doctor` initially warned that `operations` was not in the controlled
  vocabulary. The vocabulary is added in the ticket bookkeeping step.

### What I learned

- `serve-production` is not hypothetical. All static prerequisites can be
  provisioned through public commands and the process starts with them.
- A public device client can legitimately have no redirect URI when it only
  declares the device-code grant; production validation still requires explicit
  grant types and PKCE for public clients.
- Local self-signed certificates require an explicit trust decision in the
  Playwright browser. Future automation should install a locally trusted CA,
  not click through an interstitial.
- The source and docs had drifted: prior documentation said strict device flow
  was unsupported, though the endpoint/browser implementation now exists.

### What was tricky to build

The hard part was preserving the difference between an executable local smoke
fixture and a deployment recipe. The script may generate a one-day localhost
certificate and a disposable password because its work directory is explicitly
chosen, owner-only, and refused on reuse. The systemd unit, by contrast,
requires operator-managed `/etc/tinyidp` secrets and certificates and never
generates credentials. Treating these as the same workflow would encourage
unsafe production defaults.

The other sharp edge is SQLite's intentional one-open-connection policy. The
unresponsiveness after the device approval may involve an outstanding
transaction or connection lifecycle; the next investigation must retain a
goroutine dump and make a small regression test before changing transaction
code.

### What warrants a second pair of eyes

- Review the breaking `serve` to `serve-dev` rename and ensure all maintained
  command references are updated before a release.
- Review `tinyidp.service` against the target distribution and certificate
  ownership model. It is a template, not an HA topology.
- Reproduce the post-approval host unresponsiveness with a retained goroutine
  dump. Do not claim strict device authorization production readiness until
  device-code redemption, UserInfo, replay rejection, and responsiveness pass.

### What should be done in the future

- Add a production-SQLite browser regression test for the post-approval hang.
- Run a smoke with a locally trusted CA and complete token/UserInfo/replay
  assertions.
- Add backup/restore, certificate rotation, signing-key rotation, and service
  restart drills.
- Decide a complete reverse-proxy forwarded-origin policy before publishing a
  container or Kubernetes deployment manifest.

### Code review instructions

- Start at `internal/cmds/serve_production.go` for host requirements and
  `pkg/embeddedidp/options.go` for the final validation boundary.
- Review `internal/cmds/serve.go` and `cmd/tinyidp/main.go` for the command
  rename.
- Run the scripts from repository root with an unused `/tmp` work directory,
  then start script 02 inside tmux.
- Validate the non-blocked portion with:

  ```text
  go test ./cmd/tinyidp ./internal/cmds ./pkg/embeddedidp -count=1
  docmgr doctor --ticket TINYIDP-PROD-DEPLOY-001 --stale-after 30
  ```

### Technical details

The strict host requires these flags: `--addr`, `--issuer`, `--db`,
`--audit-path`, `--token-secret-file`, `--tls-cert`, and `--tls-key`. Its token
secret file must be regular, owner-only, and at least 32 bytes. The local
provisioning evidence used issuer `https://localhost:9443`, a certificate with
SANs `DNS:localhost` and `IP:127.0.0.1`, and a public client with grant type
`urn:ietf:params:oauth:grant-type:device_code`.

## Step 2: Fix the SQLite device-token self-deadlock

The persistent tmux shell made the failure diagnosable. A fresh host accepted
the browser approval and remained ready, but the approved device-code request
to `/token` blocked. Holding that request open and sending `SIGQUIT` produced
the needed goroutine dump without losing the pane.

The dump showed the token handler held the sole SQLite connection in Fosite's
token transaction while Fosite tried to sign the ID token. Signing calls
`Provider.activePrivateKey`, which reads the active signing key through the
same project store. `sqlitestore.DefaultConfig` correctly limits production
SQLite to one open connection, so the second query waited indefinitely for the
connection owned by its own transaction.

### Prompt Context

**User prompt (verbatim):** "ok now figure out the bug"

**Assistant interpretation:** Reproduce the strict device-token failure,
capture decisive runtime evidence, identify the root cause, fix it narrowly,
and validate it through tests and a real host.

**Inferred user intent:** Replace a suspected production blocker with an
evidence-backed diagnosis and a regression-protected implementation.

### What I did

- Restarted the disposable host in a persistent tmux `zsh` shell.
- Reproduced the flow with a fresh device grant and Playwright browser approval.
- Confirmed `/readyz` returned 200 after approval while `/token` timed out.
- Held a token request open, sent `SIGQUIT` only to the compiled local host,
  and captured its goroutine dump.
- Moved ID-token response construction/signing before
  `storage.MaybeBeginTx` in `device_token_handler.go`.
- Added `TestSQLiteDeviceTokenRedemptionSignsBeforeSingleConnectionTransaction`,
  a two-second-deadline regression test using the real one-connection SQLite
  store and requiring both access and ID tokens.
- Re-ran the real browser flow after rebuilding the tmux host. The approved
  device code returned 200 with access token, ID token, `openid profile`, and
  bearer token type. Replay returned `400 invalid_grant`; `/readyz` returned
  200 afterward.

### Why

The device-code consumption and Fosite token-session writes must remain one
transaction. The fix therefore does not split or weaken that transaction. It
moves only the database-dependent signing-key read ahead of transaction
acquisition; a later persistence error still causes Fosite to write an error
response rather than emit the prepared token response.

### What worked

- The goroutine dump identified the exact cycle:

  ```text
  token transaction owns SQLite's only connection
  -> IssueExplicitIDToken
  -> activePrivateKey
  -> Store.ActiveSigningKey
  -> database/sql waits for a connection
  ```

- Focused regression plus existing device tests passed.
- Full `go test ./internal/fositeadapter -count=1` passed.
- Real direct-TLS browser approval, token redemption, replay rejection, and
  readiness verification passed.

### What didn't work

- The first PID selection for `SIGQUIT` matched the searching shell as well as
  the Go child, producing `zsh:kill:5: illegal pid`. I selected the compiled
  child by its parent/PID and captured the next dump successfully.
- A first Playwright fill used stale accessibility references; a new snapshot
  supplied current references. Neither issue affected provider behavior.

### What I learned

- The earlier failure was not a leaked approval transaction: readiness stayed
  healthy after approval.
- It was a self-deadlock inside the token response phase, caused by combining
  a correct single-connection SQLite policy with a late signing-key lookup.
- Success-only SQLite device-token testing was missing; failpoint tests proved
  rollback but did not exercise the signing path after opening a transaction.

### What was tricky to build

The ordering constraint is subtle. Device-grant consumption and Fosite token
session persistence require a shared transaction for replay safety, while ID
token signing needs the active signing key from the project store. Generating
the ID token before the transaction satisfies both: no second database read is
made while the sole connection is reserved, and no durable device state changes
until all signing preparation has succeeded.

### What warrants a second pair of eyes

- Review the response-preparation-before-transaction ordering in
  `PopulateTokenEndpointResponse`; confirm every later failure is surfaced as
  a Fosite error response and does not serialize prepared tokens.
- Consider an explicit static/runtime guard for database reads while a Fosite
  token transaction is active, because future token extensions can recreate
  this pattern.

### What should be done in the future

- Add the strict direct-TLS browser smoke to automated CI with a locally
  trusted certificate.
- Finish UserInfo verification against the real strict-host device token and
  retain redacted audit evidence in the smoke harness.

### Code review instructions

- Start at `device_token_handler.go:120`; compare the signing block with the
  transaction boundary.
- Read `sqlitestore.DefaultConfig`, which deliberately sets one connection.
- Run `go test ./internal/fositeadapter -count=1`.
- Run the ticket provisioning scripts and browser flow; confirm token 200,
  replay 400 `invalid_grant`, and readiness 200.

### Technical details

The captured trace crossed `device_token_handler.go`,
`openid.IDTokenHandleHelper.IssueExplicitIDToken`,
`Provider.activePrivateKey`, and `sqlitestore.Store.ActiveSigningKey`, where
`database/sql.(*DB).conn` waited for the one available connection. The new
regression test uses an HTTP client timeout so this failure cannot silently
become an indefinitely running test again.

## Step 3: Add durable browser-continuation release coverage

I added a SQLite-backed end-to-end adapter test to cover the verification
continuation that a browser exercises: a device request, code-entry page,
cookie/CSRF-bound approval, password authentication, token redemption,
UserInfo, and replay rejection. This complements the real Playwright evidence
without misrepresenting the current self-signed local certificate as trusted
browser automation.

### Prompt Context

**User prompt (verbatim):** "go ahead"

**Assistant interpretation:** Continue converting the fixed device flow into
durable release-gate coverage.

**Inferred user intent:** Ensure the production path remains testable after
the immediate deadlock fix, not merely manually verified once.

### What I did

- Added `TestSQLiteDeviceBrowserApprovalTokenUserInfoAndReplay`.
- Built it on the one-connection `sqlitestore.DefaultConfig` and a real RSA
  signing key.
- Used the actual `/device_authorization`, `/device`, `/token`, and `/userinfo`
  routes, plus the existing cookie-jar and hidden-CSRF form helpers.
- Added a two-second HTTP client deadline so a recurrence of the deadlock
  fails deterministically.

### Why

The earlier successful full flow used the in-memory store. The regression was
production-specific because it depended on durable SQLite's single connection,
so the release gate needs the same storage configuration.

### What worked

```text
go test ./internal/fositeadapter -run 'TestSQLiteDeviceBrowserApprovalTokenUserInfoAndReplay|TestSQLiteDeviceTokenRedemptionSignsBeforeSingleConnectionTransaction' -count=1
ok github.com/manuel/tinyidp/internal/fositeadapter
```

### What didn't work

`mkcert` and `certutil` are not installed on this workstation. The future
Playwright CI gate must provision a dedicated locally trusted CA or a
Playwright-managed test trust store; it must not use `ignoreHTTPSErrors` and
call that trusted transport coverage.

### What I learned

The existing browser-form test helpers are sufficiently protocol-real to make
the SQLite full flow concise and deterministic. A real browser smoke remains
valuable for renderer/accessibility and TLS policy, while this test protects
the durable state and endpoint chain cheaply in every Go test run.

### What was tricky to build

The test must use the normal browser interaction record instead of mutating a
grant directly. Otherwise it would not cover CSRF, cookie binding, and the
transaction sequence that led to the observed production failure.

### What warrants a second pair of eyes

- Review whether the userinfo response should assert the complete expected
  claim set, not only its successful authorization, in this release gate.

### What should be done in the future

- Add a locally trusted certificate setup and execute a pinned Playwright
  version in CI.
- Add restart/persistence and audit-redaction assertions to the same harness.

### Code review instructions

- Review `TestSQLiteDeviceBrowserApprovalTokenUserInfoAndReplay` beside the
  memory-store flow test and the single-connection deadlock regression.
- Run the focused test command shown above.

### Technical details

The test holds the exact client capability,
`urn:ietf:params:oauth:grant-type:device_code`, and confirms the consumed code
subsequently maps to `invalid_grant`.

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
