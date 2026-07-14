---
Title: Investigation Diary
Ticket: TINYIDP-ACCOUNT-CHOOSER-001
Status: active
Topics:
    - identity
    - oidc
    - security
    - architecture
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/account_chooser.go
      Note: Opt-in privacy and capacity configuration.
    - Path: repo://internal/fositeadapter/account_chooser_test.go
      Note: Deduplication and capacity lifecycle test.
    - Path: repo://internal/fositeadapter/end_session.go
      Note: Atomic provider logout of active session and context.
    - Path: repo://internal/fositeadapter/session.go
      Note: Atomic context/session/membership lifecycle.
    - Path: repo://internal/store/memory/store.go
      Note: Reference in-memory atomic activation implementation.
    - Path: repo://pkg/embeddedidp/options.go
      Note: Public embedding configuration validation.
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Public configuration forwarding.
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Defines atomic activation and lifecycle store contract.
    - Path: repo://pkg/idpstore/testsuite.go
      Note: Cross-store account chooser invariant tests.
    - Path: repo://pkg/idpstore/types.go
      Note: Defines opaque browser contexts and remembered account membership.
    - Path: repo://pkg/sqlitestore/maintenance.go
      Note: Retention and orphan cleanup.
    - Path: repo://pkg/sqlitestore/migrations/007_browser_contexts.sql
      Note: Durable browser context schema.
    - Path: repo://pkg/sqlitestore/store.go
      Note: Durable SQLite activation implementation.
    - Path: repo://ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/01-store-contract.sh
      Note: Repeatable focused store test command.
    - Path: repo://ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/02-browser-context-lifecycle.sh
      Note: Repeatable lifecycle validation.
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-14T17:53:02.926941303-04:00
WhatFor: ""
WhenToUse: ""
---



# Investigation Diary

## Goal

Record the evidence and design decisions for adding safe, standard OIDC
multi-account selection to tiny-idp.

## Step 1: Map the single-session model and standards boundary

The investigation began from an observed Message Desk behavior: creating a
second account after local logout still authorized the original IdP account.
Repository inspection showed that this is not a random application bug.
Message Desk revoked only its RP session; tiny-idp correctly retained the
provider browser session and reused it on the next authorization request.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket for that, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


We want tiny-idp to be well designed and robust, abd provide a full toolbox of apis and tools and packages to build higih quality identity experiences"

**Assistant interpretation:** Create a dedicated research/design ticket for a reusable multi-account account-chooser capability, rather than implementing an ad hoc Message Desk switch.

**Inferred user intent:** tiny-idp should offer hosts standard, auditable, extensible identity UX primitives while retaining provider control of all authentication decisions.

### What I did

- Created `TINYIDP-ACCOUNT-CHOOSER-001` with an analysis document and diary.
- Inspected `session.go`, `provider.go`, `interaction.go`, `end_session.go`,
  `idpstore`, `idpui`, and SQLite migration boundaries.
- Retrieved and stored official OpenID Core, RP-Initiated Logout, Session
  Management, and Prompt Create specifications with Defuddle.
- Confirmed Core defines `prompt=select_account` and
  `account_selection_required`; confirmed Fosite already allows the prompt.

### What worked

```text
docmgr ticket create --ticket TINYIDP-ACCOUNT-CHOOSER-001 ...
defuddle parse https://openid.net/specs/openid-connect-core-1_0-18.html --md ...
```

The source bundle contains 320,655 bytes of retraceable specification text.

### What didn't work

The first Defuddle attempt failed in the restricted sandbox with `Error: fetch
failed`. A normal networked retry wrote all four official source files. An
initial documentation patch used an incorrect absolute workspace path and
removed the generated template before its add operation failed; the design
document was then recreated at the confirmed ticket path.

### What I learned

Account choice is already standardized. The engineering problem is not a URL
parameter; it is representing multiple valid sessions and activating a selected
one without reissuing an old raw cookie handle or trusting browser claims.

### What was tricky to build

The current store retains only hashes of session handles. That is correct for
security, but it means a chooser cannot simply retrieve and resend a selected
session cookie. The design therefore requires an atomic fresh-handle activation
operation that verifies server-side membership/session/user state.

### What warrants a second pair of eyes

- The privacy policy for remembered account labels needs product review.
- The semantics of per-entry removal versus global session revocation need an
  explicit decision before storage work begins.

### What should be done in the future

Implement the phased task plan after review of the proposed public store and
embedding APIs.

### Code review instructions

- Read the design’s current-state section against `session.go` and
  `provider.go`.
- Verify each protocol claim against the stored sources.

### Technical details

```text
single active IdP cookie -> existing correct SSO behavior
multiple remembered sessions -> provider-owned context/entry records
prompt=select_account -> context-bound chooser interaction -> fresh activation
```

## Step 2: Implement durable, atomic account-chooser storage

This step turned the design’s security boundary into the `idpstore` contract.
The browser context and remembered account records are all server-side and use
keyed hashes; selecting an account creates a new active session record rather
than recovering an old cookie handle. Both supported stores now provide the
same behavior, and their shared contract suite exercises it.

The SQLite implementation required one implementation-specific correction.
tiny-idp deliberately opens SQLite with one connection. A first list
implementation tried to read referenced sessions while its entry-row cursor
was open, which would wait on that same connection. The final implementation
decodes the candidate rows, closes the cursor, and then validates each session
and user. This preserves the production durability envelope rather than
silently adding another connection.

### Prompt Context

**User prompt (verbatim):** "Ok, go ahead and implement, and test with playwright then the time comes. Keep detailed diary entries as you work, and commit at appropriate intervals. Store all scripts of purpose in the scripts/ folder of the ticket so that we can retrace yoursteps."

**Assistant interpretation:** Implement the account-chooser ticket in small,
tested increments; record each increment and preserve repeatable validation
commands under the ticket.

**Inferred user intent:** Establish a production-quality reusable identity
primitive with a reviewable evidence trail, not a Message Desk-only account
switch workaround.

**Commit (code):** `7b4fa5ec347ce088b32468381eca4a0e4471cdb5` — "feat(store): add account chooser persistence"

### What I did

- Added `BrowserContext` and `RememberedBrowserSession` to
  `pkg/idpstore/types.go`, documenting that the browser receives no account
  list or authentication evidence.
- Added `BrowserContextStore` to `pkg/idpstore/interfaces.go`; it includes
  context/entry creation, list, removal, revocation, and atomic remembered
  session activation.
- Implemented parity in memory and SQLite, including fresh active-session
  creation that retains the original authentication time and expiration.
- Added SQLite migration `007_browser_contexts.sql`, backup-manifest coverage,
  and retention cleanup for contexts, removed entries, and orphaned entries.
- Added store-suite tests for context isolation, fresh handles, revoked source
  sessions, removal, context revocation, and maintenance; added
  `scripts/01-store-contract.sh` to rerun the focused contract.
- Ran the ticket script and `go test ./... -count=1` successfully.

### Why

- An account chooser cannot safely resend a prior session cookie because
  tiny-idp intentionally stores only its keyed hash.
- Session validity, user disablement, entry removal, and context membership
  must be verified together, inside the store transaction that creates the
  new session handle.
- Backup and maintenance support are necessary because browser-context state
  is now durable production security state rather than an incidental cache.

### What worked

- `ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/01-store-contract.sh`
  passed for `internal/store/memory`, `pkg/sqlitestore`, and `pkg/idpstore`.
- `go test ./... -count=1` passed, including existing Message Desk and xapp
  packages.
- The shared store suite caught cross-context activation, source-session
  revocation, and cleanup behavior in both concrete stores.

### What didn't work

- The first focused test invocation in the filesystem sandbox failed before
  compilation with `open /home/manuel/.cache/go-build/...: read-only file
  system`; rerunning unchanged with normal host access used the existing Go
  cache and passed.
- The first SQLite list implementation attempted nested reads while a result
  cursor remained open. Under the deliberately single-connection SQLite
  configuration this is unsafe; it was corrected before the final test run.
- An intermediate maintenance compile had
  `pkg/sqlitestore/maintenance.go:110:14: no new variables on left side of :=`.
  Renaming the local count to `orphanedCount` fixed it, and the focused suite
  then passed.

### What I learned

- The existing `Store.Update` transaction boundary is sufficient for atomic
  remembered-session activation. No bespoke transaction API or cache is
  required.
- SQLite cursor lifetime is part of the storage design when
  `MaxOpenConnections == 1`; code must not issue nested queries before closing
  rows.
- Deleting expired source sessions can leave remembered membership rows. The
  maintenance path must delete those orphaned records so a backup does not
  accumulate unselectable account metadata.

### What was tricky to build

The critical invariant is not simply “entry belongs to context.” Activation
must validate the context is live, entry membership is live, the source session
is unrevoked and unexpired, the source subject agrees with the entry, and the
user still exists and is enabled. Only then may it insert a session under the
new handle hash and update last-used timestamps. Returning `ErrNotFound` for
invalid membership/session/user avoids turning the chooser endpoint into a
subject-state oracle.

### What warrants a second pair of eyes

- The API deliberately clones a valid source session’s original `AuthTime`.
  This is correct for account selection, but Phase 4 must ensure
  `prompt=login` and failed `max_age` never invoke it.
- The first release stores host-provided display labels. Phase 3/5 must enforce
  a configuration policy that is privacy-safe by default on shared browsers.
- Consider whether a global logout should revoke only the context and active
  session, or all remembered source sessions, before wiring the UI action.

### What should be done in the future

- Create/rotate the opaque browser-context cookie after password login and
  attach newly authenticated sessions to it.
- Bind chooser interactions to the browser context and implement
  `prompt=select_account` / `account_selection_required` behavior.
- Add Playwright browser scenarios once a renderer exposes the chooser.

### Code review instructions

- Start with `pkg/idpstore/interfaces.go` and
  `pkg/idpstore/types.go` to review the public contract and its security
  comments.
- Follow `ActivateRememberedSession` in
  `internal/store/memory/store.go` and `pkg/sqlitestore/store.go`; confirm the
  validation sequence precedes new-session insertion.
- Review migration, backup manifest, and maintenance together to verify that
  durable records are migratable, recoverable, and collectable.
- Validate with `scripts/01-store-contract.sh`, then `go test ./... -count=1`.

### Technical details

```text
context cookie handle --keyed hash--> BrowserContext
                                    |
                                    +--> RememberedBrowserSession
                                           | context hash + source session hash
                                           v
                                      validate context, entry, session, user
                                           |
                                           +--> new random active session hash
```

```go
// Store-side activation invariant, condensed.
if !contextLive || !entryBelongsToContext || !sourceSessionLive || user.Disabled {
    return ErrNotFound
}
active := clone(sourceSession)
active.IDHash = newRandomHandleHash
active.CreatedAt, active.LastSeenAt = now, now
CreateSession(active) // inside the same transaction
```

## Step 3: Add opt-in browser-context lifecycle and global logout

The persistence layer is now exercised by the provider. Hosts can opt in to
remembering password-authenticated accounts, must supply a label policy, and
receive a separate HttpOnly browser-context cookie. A repeat login refreshes
that account’s membership; a bounded context removes the oldest membership,
not its underlying session. This makes the pending chooser safe to build on
without changing default single-session behavior.

RP-Initiated Logout now treats the browser context as part of global provider
logout. In one store transaction it revokes the active session and context,
then clears session, CSRF, and context cookies. The Message Desk’s paused
application-specific logout changes remain deliberately outside this commit:
an RP deciding whether to end its own session or perform provider-wide logout
is a host UX decision.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the implementation after the durable
store contract, keeping each behavior increment auditable and testable.

**Inferred user intent:** Make multi-account support a provider capability
with explicit privacy controls and correct logout semantics before exposing a
chooser to applications.

**Commit (code):** `7b19d58014e60a92fa3a7d49a9de62a374e9dd7c` — "feat(idp): add remembered browser contexts"

### What I did

- Added the opt-in `AccountChooserConfig` to the Fosite adapter and public
  `embeddedidp.Options`, with context-cookie naming, TTL, bounded account
  count, optional remembering, and a host label callback.
- Required an explicit non-empty, bounded label policy when password logins
  are remembered; kept the whole feature disabled by default.
- Changed password-session creation to create/reuse a context and attach a
  remembered membership atomically with the new active session.
- Made remembered entries deterministic newest-first in memory to match
  SQLite; refreshes remove previous membership for the same subject, and a
  full list removes only the oldest membership.
- Extended end-session processing to atomically revoke the active browser
  session and context, then clear the context cookie.
- Added lifecycle tests and `scripts/02-browser-context-lifecycle.sh`.

### Why

- Remembered labels are privacy-sensitive. A host must actively choose their
  content rather than tiny-idp accidentally displaying login names or email.
- The browser context must not be a second active session. It is a selector
  container, and its cookie is cleared/revoked when provider logout occurs.
- A “remove account” operation must remove only the browser membership; it
  should not silently revoke a valid server session belonging to a different
  browser or future feature.

### What worked

- The lifecycle script passed:
  - opt-in password login emits an HttpOnly context cookie and remembered entry;
  - missing label policy and invalid public configuration are rejected;
  - relogin deduplicates a subject and bounded replacement preserves source
    sessions;
  - end-session revokes both state objects and clears all provider cookies.
- Focused adapter and embedded API suites passed after the change.
- A full `go test ./... -count=1` invocation completed without a reported
  failure; the focused scripts provide the detailed, retraceable result lines.

### What didn't work

- A first new test called `memory.Store.PutClient` with an obsolete extra login
  argument and failed with
  `too many arguments in call to store.PutClient`; correcting it to the public
  `PutClient(context.Context, Client)` signature made the test pass.
- A combined multi-package `go test` display returned only one package line in
  this execution environment despite later focused tests passing. The ticket
  scripts therefore retain package-specific commands and verbose output for
  reviewers.

### What I learned

- The remembered-entry ID can safely be a transmitted opaque *hash selector*:
  it is never authentication evidence, and context cookie, CSRF, and server
  membership checks remain mandatory. This avoids persisting a recoverable raw
  entry handle.
- Context capacity is a membership policy. Removing its oldest entry must not
  call `RevokeSession`, because the session is independent provider security
  state.
- A public embedding API must validate its default cookie names as well as
  explicitly supplied names; otherwise a collision would be rejected later by
  the adapter with a less useful error.

### What was tricky to build

Session creation has to serve two paths. With the feature disabled it keeps the
old direct `CreateSession` behavior. With remembering enabled it runs a store
transaction that rechecks the candidate context, creates one if absent or
revoked, creates the active session, removes stale duplicate membership for the
subject, applies the capacity policy, and creates the new membership. The raw
context handle is sent only after that transaction commits. If it rolls back,
no cookie authorizes an absent server record.

### What warrants a second pair of eyes

- The host-supplied display-label callback executes in the login request. It
  should remain deterministic, low-latency, and avoid sensitive values on
  shared devices.
- “Global logout” now means active-cookie plus remembered browser context.
  Deliberate revocation of all source sessions should be a separately named
  administrative/security operation, not an accidental RP logout side effect.
- Phase 4 must bind a chooser interaction to `browserContextHash`, not merely
  the existing session hash, because selection can intentionally replace the
  active session.

### What should be done in the future

- Add explicit chooser POST actions for selecting and removing entries; the
  existing store operation is ready but no UI/API invokes it yet.
- Implement OIDC `prompt=select_account`, including silent
  `account_selection_required` handling and precedence with `login`/`max_age`.
- Add Playwright flows only after a renderer contains usable chooser controls.

### Code review instructions

- Read `internal/fositeadapter/account_chooser.go` for default-off privacy and
  configuration invariants.
- Trace `createBrowserSession` and `persistBrowserSession` in `session.go`;
  review where cookies are emitted relative to transactions.
- Review `end_session.go` with the new end-session test to confirm atomic
  revocation and every cleared cookie.
- Run `scripts/02-browser-context-lifecycle.sh` and then the full Go suite.

### Technical details

```text
password success
  -> label policy
  -> Store.Update {
       validate/reuse context OR create fresh context
       create fresh active session
       remove same-subject entry; evict oldest if over bound
       create remembered membership
     }
  -> Set-Cookie(active session [, new browser context])

end-session
  -> Store.Update { revoke active session; revoke browser context }
  -> clear active, CSRF, and browser-context cookies
```
