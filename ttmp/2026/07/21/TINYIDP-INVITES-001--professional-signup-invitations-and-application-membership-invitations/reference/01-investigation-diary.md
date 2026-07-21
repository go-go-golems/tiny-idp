---
Title: Investigation diary
Ticket: TINYIDP-INVITES-001
Status: active
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
LastUpdated: 2026-07-21T18:20:00-04:00
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
