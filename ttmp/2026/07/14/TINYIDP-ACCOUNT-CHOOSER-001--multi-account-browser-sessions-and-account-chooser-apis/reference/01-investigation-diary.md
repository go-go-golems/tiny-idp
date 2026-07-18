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
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: |-
        Creates a protected relying-party session after self-registration and preserves local continuation.
        Separates local relying-party logout from provider-wide logout (commit 96f531d).
    - Path: repo://examples/tinyidp-message-app/app_http_test.go
      Note: |-
        Verifies automatic post-registration session and independent OIDC login behavior.
        Regression tests for logout scopes and authenticated account-switch routing (commit 96f531d).
    - Path: repo://examples/tinyidp-message-app/loginui/renderer_test.go
      Note: Conformance and presentation assertions for chooser rendering.
    - Path: repo://examples/tinyidp-message-app/loginui/static/login.css
      Note: Branded account chooser component styling.
    - Path: repo://examples/tinyidp-message-app/loginui/templates/interaction.html
      Note: |-
        Custom renderer preserves opaque chooser controls.
        Semantic account-card chooser markup for the custom renderer.
    - Path: repo://examples/tinyidp-message-app/oidc_client.go
      Note: Message Desk requests the standard OIDC select_account prompt.
    - Path: repo://examples/tinyidp-message-app/ui/src/App.tsx
      Note: Explicit local and global logout controls (commit 96f531d).
    - Path: repo://examples/tinyidp-message-app/ui/src/api.ts
      Note: Typed endpoint contract for the two logout scopes (commit 96f531d).
    - Path: repo://internal/fositeadapter/account_chooser.go
      Note: Opt-in privacy and capacity configuration.
    - Path: repo://internal/fositeadapter/account_chooser_test.go
      Note: Deduplication and capacity lifecycle test.
    - Path: repo://internal/fositeadapter/end_session.go
      Note: Atomic provider logout of active session and context.
    - Path: repo://internal/fositeadapter/interaction.go
      Note: Creates follow-on interactions bound to a freshly activated selected-account session.
    - Path: repo://internal/fositeadapter/provider.go
      Note: Implements select-account, consent continuation, and use-another state transitions.
    - Path: repo://internal/fositeadapter/select_account_test.go
      Note: End-to-end protocol coverage for three selection outcomes.
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
    - Path: repo://pkg/idpui/default_renderer_test.go
      Note: Chooser renderer coverage.
    - Path: repo://pkg/idpui/templates/interaction.html
      Note: Default accessible chooser controls.
    - Path: repo://pkg/idpui/types.go
      Note: |-
        Chooser prompt presentation contract and validation.
        Provider UI action contract including use-another-account.
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
    - Path: repo://ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/03-chooser-ui-contract.sh
      Note: Repeatable UI contract test.
    - Path: repo://ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/04-select-account-protocol.sh
      Note: Repeatable account chooser state-machine validation.
    - Path: repo://ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/05-message-desk-smoke.spec.mjs
      Note: Prepared local browser smoke scenario.
    - Path: repo://ttmp/2026/07/14/TINYIDP-ACCOUNT-CHOOSER-001--multi-account-browser-sessions-and-account-chooser-apis/scripts/06-live-account-switch.spec.mjs
      Note: Credential-free browser regression specification for switch and local logout.
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

## Step 4: Define a renderer-safe chooser presentation contract

The provider now has a typed way to hand a renderer an account-selection page
without handing it identity authority. `idpui.AccountChooserPrompt` contains
only a provider-generated opaque selector and a deliberately stored display
label. The default renderer produces one labelled, required radio control per
entry, while the provider remains responsible for membership validation and
session activation.

This is intentionally a separate commit from the upcoming OIDC state-machine
work. A host can review and customize the UI contract now, and the next change
will connect it only after context binding and standard prompt semantics are
enforced.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the next safe implementation increment
from the account-chooser ticket, retaining tests, scripts, diary entries, and
small reviewable commits.

**Inferred user intent:** Build the chooser as an auditable identity toolbox
instead of conflating untrusted renderer inputs with authentication decisions.

**Commit (code):** `01e96adf026aa972e854702bcd860b554baf09b5` — "feat(idpui): add account chooser prompt contract"

### What I did

- Added `AccountChooserPrompt` and `AccountChooserEntry` to `idpui` with a
  fixed `account` field contract.
- Extended `InteractionPage.Validate` to require non-empty, unique opaque
  values and non-empty labels; chooser-only pages are now valid prompts.
- Extended cloning so custom renderers cannot mutate a provider-owned prompt.
- Updated the default HTML template to render labelled required radio inputs.
- Added default-renderer coverage and `scripts/03-chooser-ui-contract.sh`.

### Why

- Rendering an account list must not require an application renderer to infer
  account identity, construct continuation data, or decide which entry is
  valid.
- Unique opaque values make the forthcoming POST shape explicit while avoiding
  browser-visible user IDs and session cookie handles.

### What worked

- `go test ./pkg/idpui ./pkg/idpui/idpuitest -count=1` passed.
- The default renderer test verifies two radio inputs, required selection, and
  explicit labels without loosening the existing form/CSRF contract.

### What didn't work

- No implementation failure occurred in this increment.

### What I learned

- The existing renderer contract is already a strong boundary: it validates
  action URL, hidden fields, typed actions, and defensive copies before host
  code runs. The chooser fits it without a separate template subsystem.

### What was tricky to build

The radio input IDs are derived from opaque values only for label association.
The security boundary does not depend on those IDs or the selected value: the
forthcoming provider path will decode the value, bind it to the interaction’s
browser context, and atomically validate it in storage.

### What warrants a second pair of eyes

- If a host uses a custom renderer, it must preserve the `account` field and
  submit exactly one opaque entry value. Phase 6 conformance checks should be
  extended to assert this mechanically.
- Label length is bounded at lifecycle configuration time; consider whether
  an additional renderer-visible character policy is needed for all hosts.

### What should be done in the future

- Add the account field to renderer conformance checks.
- Implement standard `prompt=select_account` and the server-side selection
  handler before treating the page as usable.
- Add a Playwright scenario after the provider emits this page.

### Code review instructions

- Review `pkg/idpui/types.go` validation and clone behavior.
- Inspect the default template and its test together.
- Run `scripts/03-chooser-ui-contract.sh`.

### Technical details

```text
provider-owned remembered entry hash --> base64 opaque value --> HTML radio
browser POST value ------------------> provider context binding + store lookup
```

## Step 5: Implement the standard select-account authorization path

The chooser is now functional for an interactive authorization request. An RP
can send `prompt=select_account`; tiny-idp lists only current entries from the
opaque browser context, binds the authorization interaction to that context,
and accepts an opaque selector only through the store’s fresh-session
activation transaction. The browser never supplies a subject, session handle,
or a user profile decision.

The first integration test caught a state-machine bypass: the existing fast
path issued a code whenever login and consent were not required. It initially
did not also check whether account selection was required. Correcting that
guard made the provider render the chooser before token issuance.

### Prompt Context

**User prompt (verbatim):** "build it all, playwright it, then let _me_ test it."

**Assistant interpretation:** Finish the functional account chooser, validate
it with real browser automation, start an application, and provide an
operator-ready handoff.

**Inferred user intent:** Verify the production-relevant user journey rather
than stopping at persistence or unit-level APIs.

**Commit (code):** `fc09893050c71f38602addc93a44291748deed91` — "feat(oidc): implement select account authorization"

### What I did

- Added `InteractionRequireAccountSelection` and a browser-context hash to the
  durable interaction record; memory cloning preserves that new binding.
- Marked the browser-posted `account` field as transient so it cannot alter the
  canonical OIDC request digest.
- Implemented interactive `prompt=select_account` handling, context entry
  listing, opaque base64url selector rendering, and a standard
  `account_selection_required` response for silent selection requirements.
- Bound chooser POSTs to both CSRF/browser binding and the original browser
  context, then called `ActivateRememberedSession` with a fresh cookie handle.
- Added a GET-to-POST integration test and
  `scripts/04-select-account-protocol.sh`.

### Why

- OIDC selection must run before the normal immediate authorize completion,
  even if an old active session exists.
- A context-bound interaction prevents a copied selector from a different
  browser profile activating an account.
- Fresh activation preserves the selected session’s authentication time but
  avoids reissuing a prior raw session cookie.

### What worked

- The focused protocol test passed: it rendered the chooser, posted the entry
  for a second remembered user, issued an authorization code, emitted a new
  session cookie, and verified that the stored active session belongs to that
  selected user.
- `go test ./internal/fositeadapter ./pkg/idpui ./pkg/embeddedidp -count=1`
  passed.

### What didn't work

- The first integration request used the default HTTP client, which followed a
  provider redirect and reported the callback’s 404 instead of the provider
  response. Disabling redirects made the test observe the expected endpoint.
- The direct response then showed a 303 code issue instead of chooser HTML.
  The cause was the immediate-authorize guard omitting the new selection
  action. Adding that condition resolved the behavior.

### What I learned

- Every new interaction-required action must be included in the “can finish
  immediately” predicate; login and consent are not an exhaustive list.
- The existing canonical-request filter is the correct place to classify
  chooser form values as transient and prevent them from affecting the OIDC
  request integrity digest.

### What was tricky to build

The selection selector is an encoded remembered-entry hash. It is intentionally
not a credential: the provider decodes it, validates the stored interaction’s
context binding, and lets the atomic store operation validate membership,
source session, and user state. An invalid selector receives a uniform bad
request/audit event rather than an account-specific explanation.

### What warrants a second pair of eyes

- The selection-plus-consent continuation is completed in Step 6 below; review
  its new fresh-session binding carefully.
- Add explicit fuzzing for base64 selector parsing and POST field multiplicity
  in Phase 6.

### What should be done in the future

- Start a configured demonstration host and test the full browser journey with
  Playwright.
- Add direct browser-visible removal actions.
- Extend renderer conformance checks to require chooser radio/label semantics.

### Code review instructions

- Review `beginAuthorize` and its immediate completion predicate.
- Review context binding in `createInteraction` and `resumeAuthorize`.
- Run `scripts/04-select-account-protocol.sh`.

### Technical details

```text
GET /authorize?prompt=select_account
  -> interaction(context hash, old session hash, CSRF binding)
  -> opaque entry radio list
POST account=<base64url(entry hash)>
  -> verify CSRF + old session + context
  -> ActivateRememberedSession(context, entry, fresh session hash)
  -> Set-Cookie(fresh active handle) -> authorization code
```

## Step 6: Split account selection from a required consent decision

The first selection implementation was correct for hosts whose consent policy
skipped consent, which includes the local Message Desk demonstration. It had a
real correctness gap for a policy that requires user consent: after activating
the selected account, the old selection interaction could not safely become a
consent interaction because it remained bound to the old active-session hash
and accepted the `continue` action rather than an explicit consent decision.

This step makes the state-machine transition explicit. Selection activates the
chosen account under a new random handle, consumes the chooser interaction,
and creates a new consent-only interaction bound to that fresh session. The
user then sees the ordinary approve/deny prompt; only approval may issue an
authorization code.

### Prompt Context

**User prompt (verbatim):** "continue. you can spend more time as long as you know what to do / you are making progress"

**Assistant interpretation:** Continue from the working account chooser,
resolve discovered correctness gaps rather than stopping at the first passing
demo path, and preserve evidence for review.

**Inferred user intent:** The identity implementation should be production
robust across consent policies, not merely work in the development example.

**Commit (code):** `8c5bdc5b80b321e8d4df1ae5cad4d5a5729e82ec` — "feat(idp): complete account chooser consent flow"

### What I did

- Added `createInteractionForSession` in `interaction.go`; it accepts the
  server-known session hash for cases where an interaction rotates the browser
  session before producing its next page.
- Kept the ordinary `createInteraction` convenience entry point; it binds from
  the current request cookie and preserves the prior behavior for regular
  login/consent interactions.
- In `resumeAuthorize`, recomputed consent after remembered-account activation.
  When required, it consumes the account-selection record and renders a new
  `InteractionRequireConsent` record bound to the new active-session hash.
- Added `TestPromptSelectAccountCreatesFreshConsentInteraction`, which proves
  the first POST returns a consent page rather than an authorization code and
  that only a subsequent approve POST receives a code.
- Enabled the opt-in chooser in Message Desk with its explicit, privacy-aware
  display-label policy and taught its custom renderer to render radio controls.
- Added custom-renderer coverage so the provider can safely hand Message Desk
  an account-chooser page.

### Why

- A session hash is an interaction-security binding. It must refer to the
  newly activated account before a subsequent form may approve consent.
- `continue` means account selection, while `approve` means consent. Reusing
  one interaction would blur those two decisions and make the action contract
  unreviewable.
- Every host-supplied renderer must explicitly preserve the typed chooser
  form, otherwise a provider feature can be correct yet inaccessible in a
  branded application.

### What worked

- `go test ./internal/fositeadapter ./pkg/idpui ./pkg/embeddedidp ./examples/tinyidp-message-app/loginui ./examples/tinyidp-message-app -count=1` passed.
- The new integration test demonstrated this concrete sequence:

```text
chooser POST (account two)
  -> Set-Cookie(fresh idp session)
  -> 200 consent page (fresh interaction)
  -> approve POST with fresh cookie
  -> 303 callback with authorization code
```

### What didn't work

- Running the focused integration test in the restricted filesystem sandbox
  panicked before application code ran:
  `httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted`.
  Rerunning the unchanged test with approved loopback-listener access passed.
- The prior implementation did not have a failing test for this consent path;
  the gap was identified by reviewing the stored interaction binding, not by
  an observed browser failure.

### What I learned

- An authorization interaction is a state-machine node, not a generic form
  token. When an authentication decision changes session identity, the next
  node must carry the new binding.
- `issueCSRF` can safely reuse the same nonce cookie while deriving a distinct
  MAC per new opaque interaction handle. The CSRF value remains tied to the
  fresh interaction without exposing state to the browser.

### What was tricky to build

The selection request still contains the old session cookie because response
cookies are not applied until after the POST completes. Creating the consent
record with `browserSessionHash(r)` would therefore bind it to the old account.
`createInteractionForSession` solves this by accepting
`HashSecret(secret, newHandle)` directly, while keeping the raw `newHandle`
only in the response cookie. The integration test posts the consent form with
the returned fresh cookie, proving the binding is usable and not merely stored.

### What warrants a second pair of eyes

- Review the consume-then-create ordering. If durable creation of the second
  interaction fails, the user has a valid fresh session but must restart the
  authorization request; this is fail-closed for code issuance, but could be
  improved with a storage-level composite transition in a future hardening
  phase.
- Review the `InteractionOutcomeApproved` naming for a consumed selection
  continuation. It records successful completion of that continuation, not
  user consent; security event consumers should not equate it with
  `ConsentApproved`.

### What should be done in the future

- Add a composite store operation if product requirements demand retry-free
  atomicity across activation, interaction consumption, and consent-record
  creation.
- Run browser automation after the local Playwright test dependency is
  available.

### Code review instructions

- Start in `internal/fositeadapter/provider.go` at the selection branch in
  `resumeAuthorize`.
- Then read `createInteractionForSession` and confirm that its caller passes
  the fresh handle hash rather than a request cookie hash.
- Run `scripts/04-select-account-protocol.sh`; inspect the consent-transition
  test before the simple direct-selection test.

### Technical details

```text
old selection interaction [context + old session]
  -> validate entry and activate account under fresh handle
  -> consume old interaction
  -> consent interaction [fresh session]
  -> explicit approve or deny
```

## Step 7: Add an explicit “Use another account” branch and request selection from Message Desk

The chooser now provides the normal escape hatch for a remembered account list.
Selecting “Use another account” is not a client-side navigation or a special
subject value: it consumes the chooser continuation and creates a credential
interaction with no active-session binding. The browser must enter credentials
before tiny-idp will continue the authorization request.

Message Desk now asks for the standard OIDC `prompt=select_account` parameter
on its browser login initiation. A remembered browser profile therefore gets
the provider chooser; a profile with no usable entries gets the existing
password page. This is a host request to tiny-idp, not an application-side
account lookup.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Close the remaining usable account-switch path
and make the self-contained host exercise the standard provider feature.

**Inferred user intent:** A user should be able to choose a remembered account
or authenticate a different one without a relying party inferring identity
from cookies or profile data.

**Commit (code):** `3ee16cc5ef35148fa5c2b0b18761c56b80445d39` — "feat(idp): support choosing another account"

### What I did

- Added `idpui.ActionUseAnotherAccount` with a fixed submitted value, default
  label, and `formnovalidate` behavior so a required radio selection does not
  block the alternate path.
- Made chooser pages render `Continue`, `Use another account`, and `Deny`.
- Added the provider transition from that action to a fresh-login interaction;
  the new record has no session binding and requires an ordinary credential
  POST with approve/deny terminal actions.
- Added `TestPromptSelectAccountUseAnotherAccountRequiresCredentials`.
- Added `prompt=select_account` to Message Desk’s PKCE authorization URL and
  a unit assertion that the query contains the parameter.
- Extended the Message Desk custom-renderer test to require the alternate
  action as well as the opaque account radio control.
- Updated `scripts/04-select-account-protocol.sh` to run all select-account
  protocol tests.

### Why

- A selector list with no route to another credential is unusable when the
  desired account has not yet been remembered.
- Browser constraint validation would otherwise force a selected radio entry
  even when the user deliberately chooses a different account path.
- The relying party must request the standardized prompt; merely enabling the
  provider configuration leaves ordinary authorization requests on the normal
  SSO fast path.

### What worked

- The focused protocol command passed all three integration cases: direct
  selection, selection requiring fresh consent, and use-another credential
  transition.
- The broader focused command passed the adapter, `idpui`, embedded API,
  Message Desk renderer, and Message Desk application packages.

### What didn't work

- No implementation failure occurred in this increment.

### What I learned

- `formnovalidate` is part of the protocol presentation contract, not just a
  cosmetic HTML option: alternate and denial actions must be usable when a
  required credential or radio field is intentionally empty.
- A host using `prompt=select_account` receives the desired fallback behavior
  from the provider without branching on browser account state itself.

### What was tricky to build

The “use another” action must not simply clear the active session or reuse its
authorization interaction. Clearing a cookie would leave server-side state
ambiguous; reusing the interaction would preserve a binding to the prior
account. The implementation instead consumes the context-bound chooser node,
creates a new fresh-login node with a nil session binding, and relies on the
already-reviewed password-authentication code to create the new session.

### What warrants a second pair of eyes

- The Message Desk currently has a separate uncommitted RP-initiated global
  logout experiment. Global logout intentionally revokes the browser context,
  so it will clear the remembered account list. The host UX should expose a
  clearly named local app sign-out alongside global sign-out if it wants users
  to return to the chooser after leaving the application.
- Entry removal is designed in the ticket but not exposed by a provider UI/API
  in this increment.

### What should be done in the future

- Decide and implement Message Desk local-versus-global logout controls
  without overwriting the existing uncommitted global-logout work.
- Add provider-owned removal actions and a browser test for selection, use
  another, denial, and cross-context rejection.

### Code review instructions

- Read `pkg/idpui/types.go` first for the stable action contract.
- Review the `ActionUseAnotherAccount` branch in `resumeAuthorize` and confirm
  the second record has a nil session binding.
- Inspect `examples/tinyidp-message-app/oidc_client.go` to see the one
  standards-based host integration point.
- Run `scripts/04-select-account-protocol.sh` and the Message Desk package
  tests listed above.

### Technical details

```text
chooser page
  Continue + opaque entry -> activate remembered account
  Use another account    -> consume chooser -> fresh credential interaction
  Deny                   -> OAuth access_denied

Message Desk /auth/login
  -> PKCE + nonce + prompt=select_account
  -> provider decides chooser versus normal login
```

## Step 8: Launch the real embedded host and prepare browser automation

The completed code is running as the actual self-contained Message Desk
application, rather than an `httptest` fixture. The health endpoint reports
the embedded tiny-idp lifecycle as ready, the anonymous session endpoint
reports `authenticated: false`, and the live login redirect contains the
expected OIDC PKCE parameters plus `prompt=select_account`.

The Playwright source is stored in the ticket so the browser assertion is
reviewable and repeatable. The repository has the Playwright CLI but not the
`@playwright/test` module that supplies the test API; fetching that dependency
would execute newly downloaded third-party npm code with host access, so the
environment requires an explicit user approval before that final browser gate
can run.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue until the concrete demonstration is
live and give the user a testable result, while retaining the intended browser
automation evidence.

**Inferred user intent:** Validate the embedded application’s real network
surface in addition to protocol-level integration tests.

**Commit (code):** N/A — launch and validation only; prior code commits are
`8c5bdc5` and `3ee16cc`.

### What I did

- Initialized isolated local state at `/tmp/tinyidp-account-chooser-demo` with
  public base URL `http://127.0.0.1:8090`.
- Started the host in tmux session `tinyidp-account-chooser` with:

```text
go run ./examples/tinyidp-message-app serve \
  --state-root /tmp/tinyidp-account-chooser-demo \
  --addr 127.0.0.1:8090 --log-level info
```

- Captured the pane’s successful startup line and verified `/healthz` and
  `/api/session` over loopback.
- Requested `/auth/login?return_to=/` and inspected its `303 Location`; it
  includes `prompt=select_account`, `code_challenge_method=S256`, nonce,
  state, and the local registered callback URL.
- Added the prepared Playwright smoke source as `scripts/05-message-desk-smoke.spec.mjs`.

### Why

- Unit and integration tests prove protocol paths, while the live check proves
  the command wiring, embedded asset server, issuer, in-process back-channel,
  and public frontend origin agree.
- Keeping the browser script in the ticket makes the final test gate
  reproducible rather than an undocumented manual browser session.

### What worked

```text
GET /healthz     -> 200 {"ready":true,...}
GET /api/session -> {"authenticated":false}
GET /auth/login  -> 303 Location: ...prompt=select_account...
```

The tmux pane reports:

```text
INF message application listening addr=127.0.0.1:8090 origin=http://127.0.0.1:8090
```

### What didn't work

- `npx playwright test ... --project=chromium` could not run the stored test
  because the local CLI reports no configured projects and the repository does
  not provide `@playwright/test`.
- The direct bootstrap command
  `npx --yes --package @playwright/test playwright test ... --browser=chromium`
  was rejected by the execution environment because it would fetch and
  execute third-party npm code with host access. This is an approval boundary,
  not a code or test failure.
- An earlier combined startup command did not leave a tmux session. Running
  initialization directly succeeded, then starting tmux as a separate command
  succeeded and produced the healthy server above.

### What I learned

- The Message Desk integration point is exactly one generated authorization
  parameter; the live redirect confirms it is not only unit-tested but emitted
  by the binary that users run.
- Browser-test dependencies need to be present in the repository/toolchain or
  explicitly approved at runtime. Recording this boundary prevents an
  apparently skipped security gate from being misread as a passing gate.

### What was tricky to build

The public browser flow and the back-channel intentionally use different
paths: the browser follows the local public issuer URL, while token exchange
uses the explicit in-process transport. The live redirect check validates only
the former, so the earlier Go application integration tests remain necessary
to validate the latter. Together they cover the split without opening a
network dependency for token exchange.

### What warrants a second pair of eyes

- The live server contains pre-existing uncommitted Message Desk global-logout
  changes, deliberately not staged by this ticket. Global IdP logout revokes
  the remembered browser context; a local RP sign-out control is the remaining
  UX decision for a multi-account demonstration.
- The Playwright smoke currently checks the guest landing page. Extend it,
  after dependency approval, with registration, provider login, explicit
  account selection, use-another, denial, and browser-context isolation.

### What should be done in the future

- Obtain approval to install/run `@playwright/test`, execute the stored smoke,
  then add the authenticated multi-account scenarios described above.
- Resolve the host-local versus provider-global logout UI before claiming a
  polished Google-style multiple-account demonstration.

### Code review instructions

- Inspect `scripts/05-message-desk-smoke.spec.mjs` and compare its base URL to
  the documented tmux command.
- Reproduce the wire check with:

```text
curl -isS 'http://127.0.0.1:8090/auth/login?return_to=/'
```

- Inspect tmux with `tmux capture-pane -pt tinyidp-account-chooser -S -100`.

### Technical details

```text
browser -> http://127.0.0.1:8090/auth/login
        -> 303 /idp/authorize?PKCE&nonce&prompt=select_account
        -> tiny-idp interaction UI

token exchange -> explicit in-process issuer transport
```

## Step 9: Make the chooser coherent with the branded login page and sign in a newly registered user

The original chooser was functional but visually wrong for Message Desk. Its
radio controls inherited the generic credential-label grid and consequently
looked like an accidental form fragment rather than part of the existing
identity page. The custom renderer now emits a dedicated chooser fieldset and
account-card controls; the stylesheet gives selected, focused, alternate, and
denial states the same restrained typographic and color system as the login
and consent page.

Registration now creates a protected Message Desk relying-party session after
the account service has accepted the initial password. The existing frontend
navigates to `/auth/login` after account creation, so that handler recognizes
the new RP session and returns directly to the requested local page instead of
starting a redundant OIDC transaction. This is intentionally not an IdP
cookie: the user is signed into the application they just joined, while a
future authorization request still uses the provider’s normal password or
account-chooser flow.

### Prompt Context

**User prompt (verbatim):** "ok, but the account chooser looks like ass and doesn't really fit the look of the previous page, maybe it's a css issue on some common htmL? I'm not sure how it works.

Also, after creating an account, I should be logged in on the next screen, if possible (might not be possible, not sure)"

**Assistant interpretation:** Repair the custom renderer and stylesheet so
account selection looks native to the Message Desk identity surface, then make
successful account registration continue directly into a usable application
session.

**Inferred user intent:** The example should demonstrate a complete, polished
identity experience rather than forcing users through unnecessary steps after
creating a local account.

**Commit (code):** `757f98e576640ed8fe15283a96a19dc047179dcb` — "feat(message-app): sign in after registration"

### What I did

- Replaced the one-line Message Desk interaction template with readable
  semantic HTML, including a dedicated `fieldset.account-chooser`, explanatory
  copy, labelled radio account cards, and a distinct alternate-action button.
- Reworked `login.css` around those classes while retaining the established
  paper, ink, teal, blue, and rose palette and the no-window-chrome visual
  language.
- Extended the custom renderer test to run the generic `idpuitest`
  conformance checker against an account-chooser page, then assert the
  styled-account markup exists.
- Added `establishAppSession`, which generates a new opaque RP session token
  and CSRF secret, stores only the token hash, emits the HttpOnly cookie, and
  is called after a successful account creation.
- Changed the registration response target to `/` and made `/auth/login`
  redirect an already-authenticated app user to its validated local `return_to`
  path.
- Updated registration and embedded-browser integration tests so they verify
  the immediate session and keep the separate OIDC callback exercise in a
  clean browser profile.
- Rebuilt the local UI for the running demo using already-installed pnpm
  dependencies, then restarted the server in tmux and verified `/healthz`.

### Why

- The provider sends a typed `AccountChooserPrompt`; a host renderer is
  responsible for making that typed prompt visually coherent without changing
  field names, opaque values, or actions.
- Account creation has just proven possession of the newly chosen password.
  Issuing the relying-party session is therefore an ordinary post-registration
  login, not an SSO bypass.
- An IdP browser session must remain provider-owned. Creating one from the RP
  registration handler would cross the intended boundary and require a new
  carefully designed embedded-provider API.

### What worked

- `go test ./examples/tinyidp-message-app/loginui ./examples/tinyidp-message-app -count=1` passed after the change.
- The test verifies an HttpOnly, `SameSite=Lax` Message Desk session cookie,
  persists it under a hash, and proves `/auth/login?return_to=/messages`
  redirects the newly registered user to `/messages` without an OIDC round
  trip.
- The restarted local server reports a ready lifecycle and serves all new
  `account-chooser`, `account-choice`, and `chooser-note` CSS rules.

### What didn't work

- The existing end-to-end OIDC test initially expected registration to leave
  the browser unauthenticated and received `Location: /messages` from the new
  login continuation instead of an IdP authorization URL. This was an expected
  consequence of the new behavior, not a production failure. The test now
  creates a clean cookie jar for its independent OIDC callback scenario and
  passes.
- The frontend source already contained uncommitted global-logout changes.
  The local production build includes those existing source changes, but this
  commit deliberately stages only the new server-side behavior and chooser
  presentation files; it does not absorb that unrelated work.

### What I learned

- The visual defect was caused by common element selectors (`label` and
  `input`) applying credential-form layout to radio choices. A component class
  is the right boundary for a renderer-specific prompt subtype.
- A single application can safely provide immediate RP sign-in at registration
  while leaving the IdP authentication lifecycle independent. The distinction
  is important when explaining why a later OIDC interaction may still request
  credentials.

### What was tricky to build

The application and identity stores are independent SQLite databases. Account
creation cannot be atomically committed with the new Message Desk session.
The handler creates the account first, then attempts session establishment; a
rare application-store failure returns a clear `503` saying the account was
created but automatic sign-in is unavailable. It never returns a false account
creation failure that would encourage a duplicate registration attempt, and it
never emits a cookie before its session record is stored.

### What warrants a second pair of eyes

- The local RP session removes the immediate OIDC login page after registration
  by design, but it does not establish a remembered IdP account. Decide whether
  the example later needs a dedicated embedded-provider registration-to-IdP
  session API, with its own capability and audit design.
- The previously uncommitted global logout work clears the IdP browser context.
  A polished multiple-account UI should distinguish local app sign-out from
  provider-wide sign-out before exposing both controls.

### What should be done in the future

- Add an explicit “Change account” control to Message Desk that starts the
  select-account flow even for an authenticated RP user.
- Run the saved Playwright suite once the test package is approved, adding a
  visual regression assertion for account-card selection and a registration
  auto-sign-in flow.

### Code review instructions

- Read `loginui/templates/interaction.html` and `static/login.css` together;
  confirm every account input remains the provider-generated opaque value.
- Review `establishAppSession` before `handleCreateAccount`; ensure the
  session record is stored before `setSessionCookie` and no IdP cookie is set.
- Run the focused Message Desk test command and inspect the live CSS at
  `/static/tinyidp/login.css`.

### Technical details

```text
registration password accepted
  -> durable tiny-idp account
  -> fresh Message Desk session token + CSRF secret
  -> hashed session record + HttpOnly RP cookie
  -> existing /auth/login continuation sees RP session
  -> 303 local return_to page

This does not set tinyidp_session.
```

## Step 10: Separate local Message Desk logout from provider-wide logout

The running demo made a real product ambiguity visible: a person who logged
out of Message Desk could not return to the account chooser, because the only
logout action also ended tiny-idp's browser context. The application now
offers two explicit controls. “Log out of Message Desk” revokes only the
relying-party session. “Log out everywhere” revokes that session and then
navigates the browser through the provider's RP-initiated end-session endpoint.

This is a security boundary rather than merely a presentation change. The
Message Desk server owns its opaque app session cookie and CSRF secret;
tiny-idp owns the `tinyidp_session` browser cookie and remembered-account
context. The local operation does not reach across that boundary. The global
operation deliberately uses a browser navigation so the provider receives its
own cookie and can atomically invalidate the browser context it owns.

### Prompt Context

**User prompt (verbatim):** "ok, do it"

**Assistant interpretation:** Implement the proposed distinction between an
application-only logout and a provider-wide logout, test it in the live
Message Desk browser experience, and retain evidence for later review.

**Inferred user intent:** Make account switching and account-chooser access
predictable without reducing the user-visible meaning or security coverage of
an explicit global sign-out.

**Commit (code):** `96f531dccdbe92c568532a6adbb63b71392c523a` —
"feat(message-app): distinguish local and global logout"

### What I did

- Added `POST /auth/logout/local` in `app_http.go`; it shares CSRF checking,
  app-session revocation, and app-cookie clearing with the existing logout
  handler, but returns `204 No Content` before constructing an IdP end-session
  URL.
- Refactored the shared checks into `logoutApplicationSession`, controlled by
  an explicit `endProviderSession` boolean. `POST /auth/logout` remains the
  global operation and returns its browser-navigable `endSessionUrl` JSON
  payload.
- Added Go tests proving local logout revokes the Message Desk session and
  returns no end-session navigation; retained the test that proves global
  logout returns an allowlisted provider URL and revokes the local session.
- Added a Message Desk “Change account” link that explicitly starts
  `prompt=select_account` even when a local app session exists, and a unit
  test for that authenticated-app-session branch.
- Added two unambiguous UI controls: “Log out of Message Desk” invokes the
  local mutation; “Log out everywhere” invokes the global mutation and calls
  `location.assign` only for the returned provider URL.
- Formatted the touched Go code, made the touched TypeScript API surface
  readable, rebuilt the embedded production bundle, and added
  `scripts/06-live-account-switch.spec.mjs`. The saved script keeps operator
  credentials in environment variables and includes a local-logout chooser
  regression scenario.
- Restarted the demo in tmux and used the configured Playwright browser MCP
  against `http://127.0.0.1:8090`.

### Why

- A host-local sign-out should not silently erase an identity-provider browser
  context. Doing so makes account selection unavailable precisely when users
  expect it after leaving an application.
- A global sign-out must remain explicit because it has broader effect: it
  invalidates provider state and removes every remembered account from that
  browser context. Its name communicates this scope to the user.
- Returning an IdP endpoint rather than making a server-side request ensures
  the browser carries the provider-owned cookie and follows the OIDC
  RP-initiated logout interaction in its normal trust boundary.

### What worked

- `gofmt -w examples/tinyidp-message-app/app_http.go
  examples/tinyidp-message-app/app_http_test.go` completed successfully.
- `go test ./examples/tinyidp-message-app
  ./examples/tinyidp-message-app/loginui -count=1` passed.
- `pnpm build` in `examples/tinyidp-message-app/ui` passed with TypeScript
  checking and Vite production bundling.
- `go test ./... -count=1` passed across tiny-idp, including the provider,
  store, UI, embedded, Message Desk, verifier, analyzer, and xapp packages.
- The restarted tmux server reported `message application listening
  addr=127.0.0.1:8090 origin=http://127.0.0.1:8090`.
- Direct Playwright validation established a known account, clicked “Log out
  of Message Desk,” observed guest mode, clicked Sign in, and observed the
  branded “Choose an account” page with the remembered account radio control.
- The same browser then selected that account, clicked “Log out everywhere,”
  observed guest mode, clicked Sign in, and observed the credential page
  rather than an account chooser. This proves the two actions differ at the
  user-visible and provider-session layers.

### What didn't work

- The initial Playwright MCP calls used `ref` in tool arguments. This MCP
  expects the accessibility snapshot reference in `target`, producing the
  exact validation error `Invalid input: expected string, received undefined
  → at target`. Retrying with `target` succeeded; this was a test-driver
  invocation correction, not an application defect.
- The initial sandboxed `tmux capture-pane` could not access the tmux socket
  (`error connecting to /tmp/tmux-1000/default (Operation not permitted)`).
  The approved inspection command succeeded and showed the healthy listener.
- Browser console output contains the existing missing `favicon.ico` 404.
  It does not affect authentication, account choice, logout, or the tested
  API responses, so it is recorded but not expanded into unrelated work here.

### What I learned

- The account chooser is reachable after app sign-out only if the host
  preserves the IdP session cookie and remembered context. This behavior can
  be tested externally without inspecting database internals.
- A separate “Change account” control is necessary even for an authenticated
  local app session: `/auth/login` normally optimizes back to the app, while
  `switch_account=1` deliberately requests a new OIDC authorization with
  `prompt=select_account`.
- Treating the global navigation payload as a typed `Logout` response makes
  it clear why the local endpoint returns no JSON body and cannot accidentally
  trigger provider logout from the local action.

### What was tricky to build

The implementation must share the local session-revocation security checks
without creating two subtly divergent logout handlers. The shared helper
therefore authenticates the app cookie, compares the CSRF header, revokes the
hashed server-side session, and clears the HttpOnly app cookie before it
branches. The `endProviderSession` branch then derives the end-session target
only from the bootstrap-owned public origin and registered client constants.
This avoids accepting a user-controlled redirect target and avoids a server
side call that would lack the provider browser cookie.

The UI is also asynchronous: RTK Query invalidates the `Session` and `Feed`
tags after either mutation. The local logout completes with a `204`, and the
normal session query rerenders the same application document as guest mode.
The global mutation receives JSON, then explicitly changes the browser
location, allowing the IdP to clear its own session and redirect back.

### What warrants a second pair of eyes

- Review `logoutApplicationSession` to confirm every future logout route uses
  the same CSRF, durable revocation, and cookie-clearing sequence; do not add
  a convenience logout endpoint that bypasses it.
- Review the product wording with the UI owner. “Log out everywhere” accurately
  describes provider-context removal in this self-contained demo, but a host
  embedding tiny-idp with several relying parties may want more specific
  scope copy.
- Confirm the `post_logout_redirect_uri` registration remains valid whenever
  deployments change `publicOrigin`; this endpoint is intentionally built
  from canonical bootstrap configuration rather than request data.

### What should be done in the future

- Add the new Playwright specification to a repository-owned browser-test
  toolchain/CI job once that dependency is intentionally provisioned, so it
  can run without operator-supplied ad hoc setup.
- Exercise provider context removal and bounded remembered-account maintenance
  as part of the remaining Phase 3 lifecycle work, then check that phase.
- Consider a small non-authentication asset cleanup for the documented
  favicon 404 in a dedicated UI task.

### Code review instructions

- Start with `examples/tinyidp-message-app/app_http.go`, especially
  `handleLogout`, `handleLocalLogout`, `logoutApplicationSession`, and
  `endSessionURL`.
- Read `examples/tinyidp-message-app/app_http_test.go` beside the handlers;
  verify the tests distinguish a `204` local response from a JSON global
  response and ensure the authenticated switch-account route emits
  `prompt=select_account`.
- Inspect `examples/tinyidp-message-app/ui/src/App.tsx` and `ui/src/api.ts`;
  check that local logout never receives or navigates an end-session URL.
- Run:

```text
go test ./examples/tinyidp-message-app ./examples/tinyidp-message-app/loginui -count=1
go test ./... -count=1
cd examples/tinyidp-message-app/ui && pnpm build
```

- To reproduce the browser scenario, start the documented tmux server, sign
  in to Message Desk, use “Log out of Message Desk,” then Sign in and verify
  the chooser. Repeat with “Log out everywhere” and verify Sign in requests
  credentials.

### Technical details

```text
local action
  POST /auth/logout/local + app CSRF
  -> revoke Message Desk app-session hash
  -> clear tinyidp_message_app cookie
  -> 204
  -> next /auth/login sends prompt=select_account
  -> IdP sees tinyidp_session and renders remembered accounts

global action
  POST /auth/logout + app CSRF
  -> same local revocation and cookie clear
  -> JSON { endSessionUrl }
  -> browser GET /idp/end-session?client_id=...&post_logout_redirect_uri=...
  -> IdP clears tinyidp_session and browser-context membership
  -> redirect /
  -> next /auth/login renders credentials
```

## Step 11: Add provider-owned remembered-account removal

The account chooser could select a remembered identity or require a different
credential, but it could not retire one stale remembered membership. The
provider now supports that operation as a context-bound chooser action. It
removes a single remembered entry, not an account and not a provider session
belonging to another browser context.

### Prompt Context

**User prompt (verbatim):** "ok implement the provider-owner removal (since that's useful for the future), then close."

**Assistant interpretation:** Complete the useful missing chooser lifecycle
operation, verify it, and prepare the feature ticket for closure.

**Inferred user intent:** Leave the identity toolbox with a complete enough
multi-account lifecycle that future hosts can remove stale remembered accounts.

**Commit (code):** `e7dbbc79df942e99d4037715efa7ecb9f588b7d8` — "feat(idp): allow removing remembered accounts"

### What I did

- Added `idpui.ActionRemoveAccount` and its stable submitted value.
- Added the action to account chooser pages and provider action validation.
- On removal, decode the opaque selected entry, validate it against the
  interaction's browser-context binding, remove only that membership, audit
  the outcome, and re-render remaining entries.
- When the removed entry was the final remembered one, consume the chooser and
  render a fresh credential interaction instead of rendering an invalid empty
  chooser.
- Added a focused provider test proving the entry cannot subsequently activate
  and that the durable user account remains.

### Why

- Remembered membership is browser-context state, not account ownership.
  Deleting an entry must therefore not disable or delete the identity.
- The final-entry transition must be a valid credential interaction because
  the UI contract forbids an empty chooser and no account remains to select.

### What worked

- `go test ./internal/fositeadapter ./pkg/idpui -count=1` passed after gofmt.

### What didn't work

- N/A; the existing store contract supplied the required context-bound removal
  primitive, so no storage migration was needed.

### What I learned

- A browser-visible removal action can remain presentation-safe when the
  provider treats its submitted value as an opaque selector and performs the
  ownership check in persistent state.

### What was tricky to build

Removal is non-terminal while entries remain: the original interaction and
CSRF binding continue, but the provider must reload entries before rendering.
With no remaining entry it becomes terminal and creates a fresh login
interaction; reusing the chooser record would leave it with incompatible
required actions.

### What warrants a second pair of eyes

- Review future custom renderers to ensure they render `remove_account` with a
  selected opaque account field and do not replace provider validation.

### What should be done in the future

- Move the saved Playwright scenarios into CI and complete the deferred Phase
  6 race, fuzz, backup, rollback, and interoperability assurance ticket.

### Code review instructions

- Review `resumeAuthorize` in `internal/fositeadapter/provider.go` and run
  `go test ./internal/fositeadapter ./pkg/idpui -count=1`.

### Technical details

```text
remove_account(entry opaque value)
  -> context-bound RemoveRememberedBrowserSession
  -> entries remain: re-render chooser
  -> no entries: fresh credential interaction
  -> durable user account unchanged
```
