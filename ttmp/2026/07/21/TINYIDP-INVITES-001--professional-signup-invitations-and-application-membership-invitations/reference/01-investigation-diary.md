---
Title: Investigation diary
Ticket: TINYIDP-INVITES-001
Status: complete
Topics:
    - oidc
    - identity
    - auth
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological evidence and decisions behind the professional TinyIDP signup-invitation and go-go-goja membership-invitation design."
LastUpdated: 2026-07-21T15:15:38-04:00
WhatFor: "Review the evidence, commands, scope choices, and risks that shaped TINYIDP-INVITES-001."
WhenToUse: "Read before implementing or revising the invitation design, especially when resuming work in a later session."
---

# Investigation diary

## Goal

This diary records the code-backed investigation and design work for a pragmatic professional invitation core. It explains why TinyIDP account-creation invitations and go-go-goja application-membership invitations are separate, which primitives already exist, which seams remain incomplete, and how to review the resulting guide.

## Step 1: Establish scope and inspect both invitation layers

I created a dedicated docmgr ticket and treated the request as a design and code-review task rather than assuming that a new invitation subsystem had to be built. The investigation followed the complete path from TinyIDP's JavaScript declaration model through its native transaction boundary, then through go-go-goja's OIDC normalization, capability storage, and organization-invite example.

The decisive finding was that TinyIDP already has a professional one-time invitation core and an atomic signup commit. Its production command deliberately refuses to activate those capabilities. Conversely, go-go-goja already has durable application capabilities, but its example acceptance route stops after consuming the token and never creates membership. The design can therefore close two bounded seams instead of adding generic storage, raw database access, or a broad feature platform.

### Prompt Context

**User prompt (verbatim):** "Ok, craete a new ticket and a Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


We are again trying to balance pragmatism and not add all kinds of features, but also provide a robust professional core that can ship real apps."

**Assistant interpretation:** Create a new ticket containing an intern-oriented, code-backed design for both identity signup invitations and application organization invitations, explicitly minimizing new primitives while preserving production-grade security and durability, then publish it to reMarkable.

**Inferred user intent:** Resume the path toward shipping real TinyIDP-backed applications without letting flexible Goja workflows turn into an unbounded configuration, database, or feature framework.

### What I did

- Created ticket `TINYIDP-INVITES-001` with design and diary documents.
- Added phased tasks covering current-contract analysis, TinyIDP activation, invitation policy, application membership, OIDC orchestration, local acceptance, and documentation delivery.
- Read the ticket research, textbook authoring, docmgr, diary, and reMarkable delivery instructions.
- Inspected TinyIDP durable invitation types and interfaces in `pkg/idpstore`.
- Inspected HMAC code handling and transaction-aware redemption in `pkg/idpinvite/durable.go`.
- Inspected the atomic scripted signup commit in `internal/fositeadapter/scripted_signup.go`.
- Inspected the compiled program, provider metadata, effect vocabulary, and JS commit binding.
- Inspected production program validation in `internal/cmds/serve_production.go`.
- Inspected go-go-goja capability services, SQL redemption, membership stores, host normalization tests, and JavaScript example routes.
- Inspected Message Desk's distinct login and registration initiation routes and the shared two-app Compose environment.
- Wrote a long-form implementation guide with system orientation, invariants, APIs, pseudocode, diagrams, phases, file references, a test matrix, alternatives, and definition of done.

### Why

- The feature crosses an identity-provider boundary and an application-authorization boundary; merging them would make the shared IDP responsible for application roles.
- Existing primitives should be reused before introducing new configuration or storage APIs.
- A new intern needs a conceptual model of OIDC, browser continuations, Goja invocation, transaction ownership, and application membership before safely editing the code.

### What worked

- Targeted `rg` and numbered source reads quickly located the durable invitation lifecycle and its native commit boundary.
- Existing comments clearly state the security intent: raw codes do not enter the store interface, and `RedeemInTransaction` exists specifically for all-or-nothing signup.
- The go-go-goja test `TestDefaultOIDCUserNormalizerUpsertsUserWithoutGrantingMemberships` provides an executable architecture invariant.
- The current example JavaScript made the missing membership write visible: it returns accepted claims immediately after capability consumption.

### What didn't work

- The command `rg -n "ConsumeInvitation|DurableInvitations|RedeemDurableInvitation" pkg/idp* pkg/fosite* | head -100` failed under zsh with the exact error `zsh:1: no matches found: pkg/fosite*`. I reran the search against explicit existing roots: `rg -n "DurableInvitations|ConsumeInvitation|RedeemDurableInvitation|CreateUserWithCredential" pkg internal`.
- The command `rg --files deployments examples ...` reported `rg: deployments: No such file or directory (os error 2)` because this repository has no `deployments/` directory. The relevant local deployment lives under `examples/tinyidp-shared-two-apps/`.
- An attempted `docker-compose*.yml` argument also produced `zsh:1: no matches found: docker-compose*.yml`. No repository files were changed by either failed search.

### What I learned

- The requested "professional core" is not hypothetical. `DurableInvitation`, `DurableService`, conditional SQL redemption, `consumeInvitation`, and the atomic signup committer already implement its central invariants.
- The main TinyIDP blocker is a production allowlist and construction problem, not a need for JavaScript database primitives.
- Application invite acceptance is less complete: durable token handling exists, but authenticated identity binding and atomic membership creation do not.
- Virtual invitations are compatible with the provider model, but one-time semantics require a native durable claim ledger. That ledger is a legitimate later primitive, not part of the stored-invite MVP.
- Email invitation delivery and email ownership verification are different concerns. The application must not trust an email-bound invite unless OIDC supplies a verified matching email.

### What was tricky to build

- The largest conceptual risk was making a single friendly browser journey look like a single authority. The design instead uses a retryable saga: TinyIDP atomically creates the identity and consumes its signup invitation; after OIDC, the application atomically creates membership and consumes its own invitation.
- The current package structure contains more invitation support than the production constructor exposes. The guide distinguishes "implemented core" from "production-ready feature" so an intern does not mistakenly remove validation or reimplement the store.
- The proposed JS examples must remain conceptual until the exact production provider-binding shape is activated. The guide labels the intended contract and directs implementation to follow existing compiled provider conventions.

### What warrants a second pair of eyes

- Confirm that the production capability allowlist binds only the durable invitation lookup required by the declared program and cannot expose an ambient service.
- Confirm that invitation inspection does not reserve or consume a record and that commit always revalidates it.
- Review the application SQL transaction design for both SQLite and PostgreSQL, especially conditional capability use and membership conflict behavior.
- Confirm the chosen verified-email normalization contract and whether current TinyIDP deployments can truthfully issue `email_verified=true` before enabling email-bound organization invites.
- Verify that pending invite state never leaves a raw token in logs, analytics, browser history after landing, or OIDC state payloads.

### What should be done in the future

- Implement the phases in `tasks.md` in order, proving the browser path locally before k3s/GitOps changes.
- Activate email challenges only when a real mail-delivery binding and operational runbook are ready.
- Add a native claim-once ledger only when a concrete virtual-invitation use case requires one-time replay protection.

### Code review instructions

- Start with `pkg/idpinvite/durable.go`, then read `internal/fositeadapter/scripted_signup.go:332` to understand the existing correct transaction.
- Read `internal/cmds/serve_production.go:347` to see why the feature is currently unavailable in production.
- In go-go-goja, compare `pkg/gojahttp/auth/capability/invite.go` with `examples/xgoja/21-generated-host-auth/verbs/sites.js:87`; the missing membership commit is the key gap.
- Use the test matrix in the design guide as the acceptance contract.
- During implementation, run focused package tests after coherent batches and `go test ./...` at phase boundaries rather than after every small edit.

### Technical details

The TinyIDP commit order currently is:

```text
consume continuation
commit prepared identity and credential
redeem invitation in the same TxStore
create browser session
consume authorization interaction
```

The proposed application commit order is:

```text
load and validate application capability
load authenticated application user
validate verified identity binding and stored role
insert membership
conditionally mark capability used
commit
```

The two transactions cannot be one ACID transaction because they belong to separate services and databases. Failure after identity creation leaves the application invite retryable, which is an intended recovery state.

## Step 2: Validate, commit, and deliver the design

I validated the completed ticket as a structured docmgr artifact, reviewed its staged scope, and committed only the new invitation ticket. The unrelated local Compose ticket remained outside the commit.

I then bundled the design guide and this diary into one PDF and uploaded it to the ticket-specific reMarkable directory. The upload completed successfully, leaving the implementation phases intentionally open in `tasks.md`.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Complete the ticket-based documentation deliverable and publish it in a form suitable for extended reading and review.

**Inferred user intent:** Have a durable, reviewable plan available both in the repository and on reMarkable before implementation begins.

**Commit (documentation):** `ae1637d` — "docs: design professional invitation core"

### What I did

- Ran `docmgr validate frontmatter` for the design guide.
- Ran `docmgr doctor --ticket TINYIDP-INVITES-001 --stale-after 30`.
- Replaced three unknown ticket topics with the established vocabulary values `identity`, `auth`, and `xgoja`.
- Ran `git diff --check`, staged only `TINYIDP-INVITES-001`, and committed the ticket.
- Uploaded a bundle named `TinyIDP Professional Invitation Core Design.pdf` to `/ai/2026/07/21/TINYIDP-INVITES-001`.

### Why

- Frontmatter and relationship validation make the document discoverable and resumable through docmgr.
- A focused commit prevents unrelated work from becoming coupled to the design ticket.
- The reMarkable bundle provides the requested long-form reading artifact while retaining the source Markdown in Git.

### What worked

- Final docmgr output was `✅ All checks passed`.
- The commit hook correctly skipped Go test and lint jobs because the staged change contained documentation only.
- reMarkable delivery returned `OK: uploaded TinyIDP Professional Invitation Core Design.pdf -> /ai/2026/07/21/TINYIDP-INVITES-001`.

### What didn't work

- The first doctor run warned that `authorization`, `goja`, and `tiny-idp` were not registered topic vocabulary values. I used the existing, semantically appropriate values `auth`, `xgoja`, and `identity` rather than expanding global vocabulary for one ticket.
- The first staged `git diff --cached --check` found `changelog.md:15: new blank line at EOF.` I removed the extra final blank line, restaged, and reran the check successfully before committing.

### What I learned

- The repository vocabulary already has the correct broader categories for this work.
- Documentation-only commits avoid unnecessary full Go suite execution under the configured pre-commit hook.

### What was tricky to build

- The delivery step had to include useful diary context without staging the neighboring, unrelated Compose ticket. Explicit path staging kept the boundary exact.

### What warrants a second pair of eyes

- Review the substantive architecture decisions in the design rather than treating completed documentation tasks as evidence that any implementation phase is complete.

### What should be done in the future

- Begin Phase 1 from the first unchecked task and keep implementation commits separate from this design baseline.

### Code review instructions

- Review commit `ae1637d` and start with the design's executive summary, Sections 3–7, and the test matrix.
- Run `docmgr doctor --ticket TINYIDP-INVITES-001 --stale-after 30` to revalidate ticket structure.

### Technical details

The published reMarkable destination is:

```text
/ai/2026/07/21/TINYIDP-INVITES-001/TinyIDP Professional Invitation Core Design.pdf
```

## Step 3: Activate durable signup invitations in TinyIDP

I implemented the first two phases as one coherent production seam. The production server can now bind a declared `invitation.lookup@v1` capability to the durable invitation service, while the final `consumeInvitation` effect remains inside the existing signup transaction. The shared two-application program leaves Message Desk open and requires a signup code only for `goja-auth-host-demo`.

### Prompt Context

**User prompt (verbatim):** "Implement it task byt ask, commit at appropriate intervals, keep a detailed diary as you work."

**Assistant interpretation:** Follow the ticket phases in order, proving durable semantics rather than merely enabling the UI field.

**Inferred user intent:** Reach a professional, locally testable invitation flow without broad JavaScript database access or a new generic feature framework.

**Commit (TinyIDP implementation):** `c984bfd` — "feat: activate durable signup invitations"

### What I did

- Added read-only durable invitation inspection and the native `invitation.lookup@v1` capability.
- Bound lookup to the declared durable provider and bound it to the current OIDC client audience.
- Enabled only the declared `consumeInvitation` effect in production validation.
- Added `--invitation-lookup-key-file`; startup requires it only when the compiled signup program needs durable invitation lookup.
- Added `tinyidp admin invitation issue` and `revoke`; issue prints the raw code once and revoke reads the code from an owner-only file.
- Added the shared `invite_required_signup.js` program and synchronized the deployable two-app example.
- Added browser-adapter tests for per-client forms, generic denial rendering, successful signup, transactional redemption, and audit.
- Added lifecycle, concurrency, restart, audience, and rollback tests for the durable core.
- Marked all Phase 1 and Phase 2 ticket tasks complete after the full pre-commit suite passed.

### Why

- Inspection must be non-consuming so invalid form submissions do not burn a one-time code.
- The native lookup must bind the invitation audience itself; JavaScript must not choose which client an invite is valid for.
- The final redemption remains in the same transaction as identity, credential, session, continuation, and interaction changes so partial signup cannot consume a code.
- Raw invitation codes are bearer secrets, so operator tooling does not list or store them in recoverable form.

### What worked

- Focused invitation, signup executor, adapter, command, and production validator tests passed.
- The final pre-commit hook ran `go test ./...`, golangci-lint, glazed-lint, and the UI analyzer successfully.
- The rollback and concurrent-redemption tests demonstrate that there is one winner and no partial identity or invitation state.

### What didn't work

- The first command implementation used raw Cobra flags beneath the Glazed CLI tree. Glazed lint rejected that structure. I converted invitation issue and revoke into Glazed commands and retained only the existing Cobra parent used by the administrative command group.

### What I learned

- Most durable invitation machinery already existed; production activation required a narrow capability binding and constructor path rather than a second invitation subsystem.
- Conditional startup validation keeps deployments that use open signup from needing an irrelevant lookup key.

### What was tricky to build

- The lookup result had to be useful to policy code while redacting code hashes and lifecycle internals.
- The same program serves two clients, so field presentation, provider invocation, evidence, and commit behavior all had to agree on the client-specific policy.

### What warrants a second pair of eyes

- Review the production allowlist and verify that only `invitation.lookup@v1` can reach the durable service.
- Review the generic `invitation.rejected` rendering to ensure no lifecycle oracle was introduced.

### What should be done in the future

- Mount the lookup key and issue the first goja signup invitation in the local Compose acceptance environment.
- Add real email ownership verification before relying on email-bound application membership invitations outside the local demo.

### Code review instructions

- Start with `pkg/idpinvite/lookup.go`, then follow construction in `internal/cmds/serve_production.go` and the transaction in `internal/fositeadapter/scripted_signup.go`.
- Run `go test ./pkg/idpinvite ./pkg/idpsignup ./internal/fositeadapter ./internal/cmds` for the focused suite.

### Technical details

The signup path now has two deliberately separate moments:

```text
POST form -> inspect invitation and produce trusted evidence
commit    -> revalidate + consume invitation inside the signup transaction
```

## Step 4: Add atomic application membership acceptance in go-go-goja

I implemented the central Phase 3 operation in the sibling `go-go-goja` repository. A new native service spans the application user, tenant, membership, and capability tables in one SQL transaction. The generated JavaScript example now calls this service under authenticated-session and CSRF enforcement.

### Prompt Context

**User prompt (verbatim):** (see Step 3)

**Assistant interpretation:** Continue from identity creation into the distinct application-authorization invitation layer.

**Inferred user intent:** Ensure application invites grant real membership exactly once instead of merely validating or consuming a token.

**Commit (go-go-goja implementation):** `7761bdd` — "feat: atomically accept membership invitations"

### What I did

- Added `membershipinvite.Service` and a narrow `Acceptor` interface.
- Added SQLite/PostgreSQL SQL acceptance using the same shared database handle as appauth and capabilities.
- Required an authenticated application user, an enabled account, and a verified case-insensitive email match when the invite contains an email claim.
- Restricted accepted roles to the native closed set `viewer`, `member`, and `admin`.
- Required the canonical `org.invite.accept` purpose, an `org` resource, and an enabled tenant.
- Inserted/restored membership and conditionally marked the single-use capability consumed before one commit.
- Exposed `auth.membershipInvites.accept(token).actor(ctx.actor.id).run()` without exposing a database or transaction object to JavaScript.
- Changed the generated example acceptance route from public capability consumption to authenticated, CSRF-protected membership acceptance.
- Added native SQL tests, JavaScript binding tests, host-store construction checks, and regenerated package logging stubs through the normal hook.

### Why

- Capability consumption without membership creation permanently burns access without granting the promised authorization.
- Identity binding belongs in native code because request fields such as `email`, `role`, and `orgId` are attacker-controlled.
- One database transaction makes retry behavior deterministic: either both membership and consumption commit, or neither does.

### What worked

- Focused membership-invite, hostauth, and provider tests passed.
- An injected SQLite trigger forced capability consumption to fail and proved that the membership insert rolled back.
- Two concurrent acceptance attempts produced exactly one winner.
- The repository pre-commit hook completed lint, code generation, and the full `go test ./...` suite before creating commit `7761bdd`.

### What didn't work

- An early test tried to revalidate an invitation with a service whose clock defaulted to wall time, making the fixed 2026 fixture appear expired. I changed the assertion to inspect `used_at` directly, which tests the intended non-mutation property without coupling to an unrelated clock.
- I initially tried a nonexistent `make compile` target in example 21. The correct validation entry points are `go run ./cmd/xgoja doctor -f ...`, `make build`, and the example's smoke targets.

### What I learned

- The host store builder already deduplicates SQL connections by driver and DSN, which gives the atomic operation the correct transaction boundary without changing the generic appauth or capability interfaces.
- The existing example used the informal purpose `org-invite`; using the package's canonical `org.invite.accept` constant prevents incompatible invitation dialects.

### What was tricky to build

- The service must distinguish a trusted application actor ID supplied by reviewed route code from untrusted request-body identity claims. The route passes only `ctx.actor.id`; native SQL reloads the user and ignores body-provided user, email, role, and tenant values.
- Memory stores cannot honestly provide a multi-store ACID transaction, so the production operation is constructed only when appauth and capabilities share one SQL driver and DSN.

### What warrants a second pair of eyes

- Review PostgreSQL row locking (`FOR UPDATE`) and the conditional `used_at IS NULL` update under concurrent load.
- Confirm the initial deployment's desired closed role vocabulary before treating `admin` invites as generally issuable.

### What should be done in the future

- Complete deployment bootstrap for the first tenant, resource, and administrator.
- Add opaque pending-invite continuation and the explicit OIDC registration entry route before browser acceptance testing.

### Code review instructions

- In go-go-goja, read `pkg/gojahttp/auth/membershipinvite/membershipinvite.go`, its `sqlstore`, then the binding in `pkg/xgoja/providers/hostauth/hostauth.go`.
- Compare the updated route in `examples/xgoja/21-generated-host-auth/verbs/sites.js` with the old public consume-only behavior described in the design.

### Technical details

```text
BEGIN
  SELECT capability FOR UPDATE
  SELECT authenticated app user
  verify enabled + email_verified + email match
  verify purpose + tenant + native role allowlist
  UPSERT membership
  UPDATE capability SET used_at = now WHERE used_at IS NULL
COMMIT
```

## Step 5: Preserve application invitations across OIDC registration

I completed Phase 4 in go-go-goja by separating the raw application bearer token from the browser's long-lived navigation state. The public landing operation validates the capability once, persists only a hash of a fresh short-lived pending handle, and returns local login and registration URLs. OIDC stores only the safe local `return_to`, and final acceptance reloads both records inside the application transaction.

The generated host now exposes an explicit `/auth/register` entry point. It uses the same state, nonce, PKCE, callback, and server-side transaction machinery as login, adding only TinyIDP's namespaced signup intent. An identity created successfully at TinyIDP can therefore retry application acceptance without either database pretending to share a distributed transaction.

### Prompt Context

**User prompt (verbatim):** (see Step 3)

**Assistant interpretation:** Finish the browser-orchestration phase without moving raw invitation tokens into OIDC state, cookies, or application URLs.

**Inferred user intent:** Make the friendly signup-and-join journey recoverable while preserving the independent TinyIDP and application trust boundaries.

**Commit (go-go-goja implementation):** `41cc3f6` — "feat: preserve invites through OIDC registration"

### What I did

- Added hashed, expiring membership-invite pending records to SQLite and PostgreSQL auth schemas.
- Added `Begin` and `AcceptPending` to the native membership-invitation service.
- Made successful pending acceptance atomically create membership, consume the capability, and consume the pending record.
- Left denied acceptance retryable: email mismatch or missing verification mutates neither the capability nor the pending record.
- Added `auth.membershipInvites.begin(token).run()` and `acceptPending(handle).actor(id).run()` JavaScript bindings.
- Added `/auth/register` to the generated OIDC host and preserved a validated local `return_to` through the server-side OIDC transaction.
- Updated the example frontend to exchange a raw token for a pending handle before navigation and accept it after the authenticated session is available.
- Added SQL, host binding, OIDC handler, and frontend smoke coverage.

### Why

- A raw application invitation is a bearer credential and should leave the address bar as soon as the landing request can exchange it.
- OIDC `state` must remain an unpredictable correlation token, not become a bag of application state or secrets.
- TinyIDP signup and application membership live in separate databases, so the correct failure model is a retryable saga rather than a fictitious cross-service transaction.

### What worked

- SQL tests prove a failed verified-email check leaves both pending and capability records reusable.
- The host callback restores the exact safe local path selected before OIDC without accepting absolute or scheme-relative redirect targets.
- The full go-go-goja pre-commit suite passed before commit `41cc3f6`.

### What didn't work

- N/A for the committed Phase 4 batch; local Compose execution had not yet started at this point.

### What I learned

- The existing OIDC transaction store was already the correct place for `return_to`; no new cookie or signed browser payload was necessary.
- Pending handles need their own one-time lifecycle even though they point at another one-time capability, because they independently cross the browser boundary.

### What was tricky to build

- Acceptance must lock and revalidate the pending record and underlying capability in one transaction. Validating only at `Begin` would permit expiry, revocation, or replay races before the authenticated actor appears.
- The registration handler must differ from login only in authorization intent; duplicating callback or token-exchange logic would create two subtly different security paths.

### What warrants a second pair of eyes

- Review pending-handle expiry and conditional-use SQL for PostgreSQL concurrency behavior.
- Verify every future OIDC entry route continues to validate `return_to` as a local absolute path and never serializes the raw application token.

### What should be done in the future

- Prove the complete saga, its denied retry state, and successful existing-user acceptance in the shared local Compose environment.

### Code review instructions

- Start with `pkg/gojahttp/auth/membershipinvite/sqlstore/sqlstore.go`, then follow `Begin`/`AcceptPending` through `pkg/xgoja/providers/hostauth/hostauth.go`.
- Review `pkg/gojahttp/auth/oidcauth/oidcauth.go` for the registration-intent delta and local-return validation.
- Run `go test ./pkg/gojahttp/auth/membershipinvite/... ./pkg/gojahttp/auth/oidcauth ./pkg/xgoja/hostauth -count=1`.

### Technical details

```text
raw application token
  -> Begin: validate capability, generate handle, store Hash(handle)
  -> /auth/register?return_to=/?pending=<opaque handle>
  -> TinyIDP authorization + optional signup
  -> application callback + session
  -> AcceptPending(handle, actor)
  -> membership + capability use + pending use, one application transaction
```

## Step 6: Prove both applications and both invitation layers in the local HTTPS stack

I completed Phase 5 without changing k3s or GitOps. The local Compose environment now builds the sibling go-go-goja checkout, initializes only local owner-readable secrets, creates deterministic verified test identities, bootstraps the first application administrator explicitly, and runs a seven-stage browser-protocol acceptance suite through Caddy's real local HTTPS origins.

The run found five integration defects that unit tests alone had not exposed: Docker secret ownership, missing route-builder policy transitions, a separate TinyIDP consent form, the expected HTTP status of a redisplayed rejected signup form, and a successful domain audit record that lacked its tenant association. Each correction was made at the narrow owning seam. The final acceptance run passed with both raw-invitation audit-leak assertions enabled.

### Prompt Context

**User prompt (verbatim):** "all the way to phase 5, do not yet deploy to k3s"

**Assistant interpretation:** Finish every task through local Phase 5, prove the full two-application invitation behavior through public HTTPS/browser boundaries, commit coherent batches, and stop before any cluster mutation.

**Inferred user intent:** Establish a trustworthy, repeatable local product gate before spending time on Vault, k3s, Traefik, or GitOps integration.

**Commits:**

- go-go-goja `c19969b` — "fix: harden membership invitation acceptance"
- tiny-idp `63dfc5f` — "feat: validate shared invitation flows locally"

### What I did

- Extended the TinyIDP container entrypoint to copy Compose-mounted secrets into an owner-only service directory before dropping from root to the dedicated `tinyidp` identity.
- Added `scripts/00-init-secrets.sh` to create the two local fixture password files and a random 32-byte invitation lookup key under gitignored `runtime/secrets/`, with directory mode `0700` and file mode `0600`.
- Enabled the production durable invitation lookup key in the shared TinyIDP service.
- Added idempotent TinyIDP creation of `admin@example.test` and `invitee@example.test` as email-verified local fixtures.
- Changed the local goja service to build directly from the sibling checkout so the invitation implementation under review is the implementation being exercised.
- Added an idempotent PostgreSQL bootstrap job for application user normalization, tenant `o1`, organization/project resources, and the initial administrator membership.
- Added `scripts/03-browser-acceptance.py` using independent cookie jars, the exported local CA, HTML form parsing, normal redirects, application JSON/CSRF calls, operator CLI issuance, and read-only SQL/audit assertions.
- Required a non-empty invited email in the example issuance route.
- Added the required `.allow("user.self.read")` transition to the authenticated continuation and acceptance routes.
- Corrected the example response to read the native result's closed `role` field.
- Required native application invitations to contain an email binding, an application subject binding, or both; acceptance rejects a missing binding and verifies every binding that is present.
- Added tenant identity to successful `org.invite.accepted` service audit records and a focused tenant-queryability regression test.
- Added explicit acceptance checks proving raw TinyIDP and application bearer values do not appear in retrieved audit evidence.

### Why

- A local integration gate must execute the same TLS origins, OIDC redirects, cookies, continuations, CSRF checks, and database transitions that the browser uses.
- Initial application authority must be an explicit deployment bootstrap; ordinary OIDC normalization must never make the first user an administrator.
- Email-bound membership authorization is valid only for a verified matching identity. A capability without email or subject binding would degrade into an unrestricted bearer-only organization grant.
- Audit completion is operationally useful only when the owning tenant can retrieve the record through the supported tenant query.
- Testing absence of raw codes in audit output turns a security intention into executable evidence.

### What worked

- `go test ./pkg/gojahttp/auth/membershipinvite/... -count=1` passed after the identity-binding and tenant-audit changes.
- The go-go-goja pre-commit hook passed code generation, the complete `go test ./...` suite, golangci-lint, Glazed lint, and vet before commit `c19969b`.
- `docker compose ... build goja-auth` produced image `sha256:ad7e49db2971...` from the current sibling source.
- `scripts/02-smoke.sh` passed readiness for all three public origins and verified both OIDC client redirects.
- The final `scripts/03-browser-acceptance.py` run printed `PASS: shared TinyIDP Phase 5 browser acceptance completed` with exit status zero.
- The denied new-user path was retried twice and left the underlying application capability unused.
- The verified existing-user path created exactly one active `viewer` membership and rejected both pending-handle and raw-token replay.

### What didn't work

- The first TinyIDP container start failed with `cannot open /run/secrets/local_admin_password: Permission denied`. Compose bind-mounted the host `0600` file with an owner that did not match the container's unprivileged UID. Copying secrets into `/state/.secrets`, setting mode `0400`, changing ownership, and only then dropping privileges fixed the runtime boundary.
- The first goja start failed with `TypeError: Object has no member 'handle' at demo (/sites.js:115:12(202))`. The Express auth API is a staged builder; `.audit(...)` on `RouteNeedsPolicy` does not expose `.handle(...)` until `.allow(...)` establishes route policy. Both new authenticated routes needed that transition.
- The first browser driver stopped after signup because TinyIDP rendered a distinct consent form before the OIDC callback. The driver now advances only bounded IDP forms that contain no credential or signup fields.
- The first replay assertion expected HTTP `200`; TinyIDP correctly redisplayed the invalid signup form with HTTP `400` and the generic field error `This value could not be accepted.` The test now asserts the actual safe error contract.
- The first audit stage could find the successful row directly in PostgreSQL but not through `/orgs/o1/audit`. The service event had resource ID `o1` and an empty tenant ID. Populating both fields made the domain event tenant-queryable.

### What I learned

- Browser acceptance must distinguish protocol forms by their fields and action, not assume a fixed number of redirects after signup.
- Staged JavaScript builders provide a runtime guard as well as generated TypeScript guidance; bypassing a required policy transition fails before the server can listen.
- A row existing in an audit table is insufficient evidence. The supported operator query and tenancy filter are part of the audit contract.
- The cross-database saga behaves correctly: successful TinyIDP signup can coexist with denied application membership, while the separate application capability remains retryable.

### What was tricky to build

- The test driver had to preserve hidden continuation, interaction, consent, and CSRF fields across three HTTPS origins without using a browser automation dependency.
- Local Caddy trust had to cover browser-facing requests and server-to-server discovery/token/JWKS calls without exposing Caddy's CA private key.
- The verified fixture needed the exact deterministic application user ID derived by the host from OIDC issuer and subject, while still keeping initial administrator authority explicit.
- The membership transaction had to support email-only pre-account invitations and subject-only invitations for known principals without accepting a record that carried neither binding.

### What warrants a second pair of eyes

- Review the local entrypoint's copy/chown/drop sequence and confirm production images will instead receive secrets through the intended Vault/Kubernetes mechanism.
- Review the PostgreSQL `FOR UPDATE` queries and conditional capability/pending updates under the deployment's intended transaction isolation level.
- Review whether the route-level and service-level `org.invite.accepted` audit vocabulary should use distinct event names before production dashboards consume it.

### What should be done in the future

- Add a real TinyIDP email delivery/confirmation binding before expecting a newly created password identity to accept an email-bound application invitation successfully.
- Translate the proven local configuration into Vault-backed, Traefik-aware k3s manifests and GitOps changes in a separate deployment phase.
- Remove deterministic fixture users and passwords from any non-local environment.

### Code review instructions

- Run `./examples/tinyidp-shared-two-apps/scripts/02-smoke.sh` and `./examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py` while the Compose stack is running.
- In TinyIDP, review `compose.yaml`, `bootstrap.sql`, the entrypoint, and the acceptance script as one local deployment unit.
- In go-go-goja, review `membershipinvite.Service.record`, `sqlstore.acceptRecord`, both capability-loading queries, and the two JavaScript acceptance routes.
- Confirm the final browser output contains all seven `OK` stages and that the application audit includes the exact capability ID accepted in that run.

### Technical details

```text
new Message Desk visitor
  -> open TinyIDP signup
  -> identity + provider session + OIDC callback
  -> Message Desk session

new goja visitor
  -> app invite Begin -> pending handle
  -> one-time TinyIDP signup invite -> identity
  -> OIDC callback -> app user/session
  -> app acceptance denied: email_verified=false
  -> app capability and pending handle remain unused

verified goja fixture
  -> app invite Begin -> pending handle
  -> OIDC login -> app session
  -> atomic membership + capability use + pending use
  -> pending and raw-token replay rejected
```
