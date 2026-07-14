---
Title: Investigation Diary
Ticket: TINYIDP-MSGAPP-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - architecture
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: |-
        Issues anonymous registration pre-sessions and CSRF material
        Strict bounded registration decoding, one-time CSRF consumption, account-service delegation, and password-byte clearing
        Origin, Fetch Metadata, trusted address, and canonical-login registration boundary
        Stable non-secret registration audit event codes
        Public GET message feed with bounded pagination and subject-redacted response
        Authenticated message creation with exact Origin, app-session CSRF, and server-derived author fields
    - Path: repo://examples/tinyidp-message-app/app_http_test.go
      Note: |-
        Exercises the browser-visible login form and complete callback flow
        Validates cookie and durable one-time registration state
        Tests registration CSRF, strict JSON, normal authentication, and absence of auto-login
        HTTP registration followed by genuine embedded provider login
        Mutation authorization, spoofing, and stored-author regression coverage
    - Path: repo://examples/tinyidp-message-app/appstore.go
      Note: Checksummed app schema and SQLite envelope (commit c41ba0b)
    - Path: repo://examples/tinyidp-message-app/appstore_test.go
      Note: Migration, checksum, permissions, and pragma evidence
    - Path: repo://examples/tinyidp-message-app/contracts.go
      Note: Phase 0 executable route, cookie, and invariant contract
    - Path: repo://examples/tinyidp-message-app/contracts_test.go
      Note: Public consumer import-boundary enforcement
    - Path: repo://examples/tinyidp-message-app/login_attempts.go
      Note: Atomic durable OAuth state consumption (commit 2603c18)
    - Path: repo://examples/tinyidp-message-app/login_attempts_test.go
      Note: Replay, expiry, and concurrent single-winner tests
    - Path: repo://examples/tinyidp-message-app/messages.go
      Note: Opaque cursor codec for stable message-feed continuation
    - Path: repo://examples/tinyidp-message-app/oidc_client.go
      Note: |-
        OIDC discovery, durable PKCE, callback verification core (commit 36c1727)
        Keeps the token exchange and ID-token verification on the explicit in-process transport
    - Path: repo://examples/tinyidp-message-app/oidc_client_integration_test.go
      Note: Exact in-process issuer discovery evidence
    - Path: repo://examples/tinyidp-message-app/state.go
      Note: Phase 3 state-root paths, manifest, secrets, and atomic publication (commit 9f4a4e2)
    - Path: repo://examples/tinyidp-message-app/state_test.go
      Note: Permission, conflict, idempotency, and damage tests
    - Path: repo://pkg/idpaccounts/accounts.go
      Note: Public canonical login normalization for embedding-safe rate-limit keys
    - Path: repo://pkg/idpaccounts/password.go
      Note: Authentication now shares the exported canonical-login function
    - Path: repo://ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/design-doc/01-embedded-tiny-idp-sqlite-message-application-analysis-design-and-implementation-guide.md
      Note: Primary design artifact whose evidence and construction are recorded in this diary.
    - Path: repo://ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/sources/01-go-oidc-package.md
      Note: Stored OIDC package reference used to design discovery and ID token verification.
    - Path: repo://ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/sources/02-go-oauth2-package.md
      Note: Stored OAuth2 and PKCE package reference used to design the relying-party flow.
    - Path: repo://ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/sources/05-owasp-authentication-cheat-sheet.md
      Note: Stored authentication and registration guidance used in the threat analysis.
    - Path: repo://ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/sources/06-owasp-csrf-prevention-cheat-sheet.md
      Note: Stored CSRF guidance used for registration and application mutation design.
ExternalSources:
    - https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc
    - https://pkg.go.dev/golang.org/x/oauth2
    - https://www.sqlite.org/wal.html
    - https://www.sqlite.org/foreignkeys.html
    - https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
    - https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
Summary: Chronological evidence, decisions, commands, and delivery notes for the embedded tiny-idp SQLite message application design.
LastUpdated: 2026-07-13T20:27:13-04:00
WhatFor: Use this diary to review how the application design was derived and to continue the future implementation without repeating investigation.
WhenToUse: Read before implementing or revising TINYIDP-MSGAPP-001.
---


















# Investigation Diary

## Goal

This diary records the creation of a design package for a small Go web application that embeds tiny-idp, persists identity and application state in SQLite, supports self-service account creation, and lets authenticated users publish plain-text messages. It preserves the repository evidence, external references, design decisions, commands, failures, and handoff instructions needed for future implementation.

## Step 1: Create the ticket and establish the investigation scope

The first step created a dedicated docmgr workspace before drawing architecture conclusions. This ensures the design, sources, tasks, diary, and later implementation evidence remain associated with one stable ticket.

The working interpretation is that “showcases how to integrate tiny-idp” means the example must use supported public packages and standard OIDC behavior. An example that imports `internal/admin` only because it resides inside this repository would compile locally but would not teach an external application how to integrate.

### Prompt Context

**User prompt (verbatim):** "Craete a new docmgr ticket for small go application that showcases how to integrate tiny-idp into a web application, with sqlite, that allows people to write simple messages and createa new account when they want.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new ticket, investigate current tiny-idp embedding facilities, design a durable single-process message application with registration, write an intern-ready implementation guide and diary, validate the ticket, and upload a bundled PDF to reMarkable.

**Inferred user intent:** Produce a durable engineering specification that can be handed to a new contributor and later implemented phase by phase without rediscovering the identity, storage, security, UI, and operational boundaries.

### What I did

- Ran `docmgr status --summary-only` from the tiny-idp repository.
- Inspected the existing ticket list to avoid an ID collision.
- Created `TINYIDP-MSGAPP-001` with architecture, Go, identity, OIDC, and security topics.
- Added the primary design document.
- Added this investigation diary.
- Added ten ticket tasks covering discovery, sources, scope, public APIs, SQLite, OIDC, UI/API design, implementation phases, validation, and upload.

Commands:

```bash
docmgr status --summary-only

docmgr ticket create-ticket \
  --ticket TINYIDP-MSGAPP-001 \
  --title 'Embedded SQLite Message Application with Self-Service Accounts' \
  --topics go,identity,oidc,architecture,security

docmgr doc add \
  --ticket TINYIDP-MSGAPP-001 \
  --doc-type design-doc \
  --title 'Embedded Tiny-IDP SQLite Message Application Analysis Design and Implementation Guide'

docmgr doc add \
  --ticket TINYIDP-MSGAPP-001 \
  --doc-type reference \
  --title 'Investigation Diary'
```

### Why

- A dedicated ticket provides stable paths for the design and later code evidence.
- Creating the diary at the beginning prevents retrospective loss of commands and corrected assumptions.
- Detailed tasks make the work resumable across long implementation intervals.
- The ticket topics reuse the repository vocabulary so docmgr validation remains meaningful.

### What worked

- `docmgr` created the expected index, tasks, changelog, design-doc, reference, sources, scripts, playbooks, and supporting directories.
- Ticket ID `TINYIDP-MSGAPP-001` did not collide with existing work.
- The generated ticket was immediately visible through `docmgr ticket list --ticket TINYIDP-MSGAPP-001`.

### What didn't work

- An initial attempt to issue several repository-inspection commands concurrently through a JavaScript orchestration cell failed before executing with:

```text
Script error:
SyntaxError: Invalid or unexpected token
```

- I switched to individual `exec_command` calls. No repository state was changed by the failed orchestration cell.

### What I learned

- The repository already has extensive production-readiness and xapp tickets, but no focused conventional Go message application example with registration.
- The ticket vocabulary does not currently include dedicated `sqlite`, `web`, or `registration` topics. Existing topics adequately classify this design, so no vocabulary expansion was required at creation time.

### What was tricky to build

The main scoping difficulty was distinguishing a small application from a superficial example. A two-handler demo could post messages, but it would omit OIDC client state, separate application sessions, CSRF, bootstrap, database migrations, and restart behavior. The design therefore keeps the user-facing product small while retaining the real identity and persistence boundaries.

### What warrants a second pair of eyes

- Confirm that the example should be copyable by an external module rather than merely runnable inside tiny-idp.
- Confirm the proposed location `examples/tinyidp-message-app` and binary naming before implementation.
- Review whether React/Redux/RTK Query is desirable for this showcase or merely required by repository conventions.

### What should be done in the future

- Implement the ticket in its documented phases after architecture review.
- Keep the task identifiers stable when updating progress.

### Code review instructions

- Start with the ticket `index.md` and `tasks.md`.
- Verify ticket discovery with:

```bash
docmgr ticket list --ticket TINYIDP-MSGAPP-001
docmgr doc list --ticket TINYIDP-MSGAPP-001
docmgr task list --ticket TINYIDP-MSGAPP-001
```

### Technical details

Ticket path:

```text
ttmp/2026/07/13/
  TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/
```

## Step 2: Map the current embedding, persistence, and account APIs

The repository investigation followed the request path from public provider construction through SQLite, account creation, xapp initialization, same-process OIDC, route composition, and the stylable interaction renderer. Claims in the design were tied to concrete files and line ranges.

The critical finding was not an absence of functionality. The functionality exists, but several pieces are behind internal package boundaries or are coupled to the xapp's go-go-goja host-auth stack. The proposed example must make those capabilities public in a narrow form rather than copying their implementation.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Inspect the existing code deeply enough that the design describes actual integration boundaries and required API work.

**Inferred user intent:** Make the future example technically credible and useful to a new intern, not a speculative architecture disconnected from the repository.

### What I did

- Searched for `embeddedidp.Options`, provider constructors, route mounts, SQLite opening, account creation, password services, signing keys, and in-process OIDC transports.
- Read line-numbered sections of:
  - `pkg/embeddedidp/options.go`;
  - `pkg/embeddedidp/provider.go`;
  - `pkg/idpstore/interfaces.go`;
  - `pkg/sqlitestore/store.go`;
  - `internal/admin/users.go`;
  - `internal/authn/password.go`;
  - `cmd/tinyidp-xapp/state.go`;
  - `cmd/tinyidp-xapp/development_app.go`;
  - `cmd/tinyidp-xapp/production_app.go`;
  - `docs/storage.md`;
  - `docs/users-and-passwords.md`.
- Examined the earlier stylable interaction-renderer work so the new example can use the supported `pkg/idpui` boundary.

Representative commands:

```bash
rg -n 'type Options|type UIConfig|func New|CreateUser|sqlite|SQLite' \
  pkg internal cmd docs -g '*.go' -g '*.md'

rg -n 'CreateUser\(|CreateUserRequest|NewProvider|RegisterRoutes|Handler\(' \
  pkg cmd/tinyidp-xapp internal/admin internal/fositeadapter -g '*.go'

nl -ba pkg/embeddedidp/options.go | sed -n '1,240p'
nl -ba pkg/embeddedidp/provider.go | sed -n '1,180p'
nl -ba cmd/tinyidp-xapp/development_app.go | sed -n '60,185p'
nl -ba cmd/tinyidp-xapp/development_app.go | sed -n '230,305p'
nl -ba cmd/tinyidp-xapp/state.go | sed -n '1,270p'
nl -ba internal/admin/users.go | sed -n '1,120p'
nl -ba pkg/idpstore/interfaces.go | sed -n '90,180p'
nl -ba pkg/sqlitestore/store.go | sed -n '1,150p'
```

### Why

- The design needs to separate observed capability from proposed changes.
- The example should reuse existing atomic identity operations and production controls.
- Public API gaps should be stated precisely before adding a new package.
- Existing xapp work is the strongest local evidence for same-process provider and relying-party composition.

### What worked

- `pkg/embeddedidp` already provides a clean handler and production validation boundary.
- `pkg/sqlitestore` already provides owner-only durable SQLite with migrations and named atomic security operations.
- `internal/admin.Service.CreateUser` already creates the user and credential atomically using the same password service used for authentication.
- Xapp already demonstrates separate IdP and application sessions and uses an in-process OIDC transport.
- The xapp production layout already separates the identity database and application authentication database.
- The new `pkg/idpui` renderer allows the example to style the provider interaction without owning protocol logic.

### What didn't work

- No public API currently generates the initial signing key for an external embedding host.
- No public account service currently exposes correct self-registration to an external host.
- The in-process transport used by xapp comes from go-go-goja rather than tiny-idp.
- A direct `go-oidc` dependency is not currently declared by tiny-idp, although `golang.org/x/oauth2` is already present indirectly.

These are design gaps rather than command failures.

### What I learned

- A useful example cannot be implemented only under `examples/` without some public library work.
- Directly reading the IdP session cookie would be shorter but would destroy the relying-party boundary the example is supposed to teach.
- Registration and authentication must share normalization, acceptance policy, hashing parameters, Argon2 work limits, store, and audit behavior.
- The application needs its own server-side session even though the issuer and app share one process.
- Message ownership must use verified OIDC `sub`, not login or display name.

### What was tricky to build

The most subtle issue is bootstrap. `idpstore.Store` publicly exposes raw client and signing-key operations, but an external host cannot safely manufacture the private signing-key representation because generation lives in `internal/keys`. Exporting the raw helper would expose representation rather than intent. The design therefore proposes a narrow idempotent bootstrap reconciler that returns only non-secret status.

The second subtle issue is OIDC discovery before or during server startup. Calling the application's public URL from the same process adds DNS and TLS dependencies. Bypassing discovery by reading provider stores would no longer demonstrate OIDC. An exact-origin in-process `RoundTripper` preserves protocol semantics and avoids the network dependency.

### What warrants a second pair of eyes

- Review the public surface and package name of `pkg/idpaccounts`.
- Review whether bootstrap belongs in `pkg/embeddedidp` or `pkg/idpbootstrap`.
- Threat-model exact URL matching in the proposed in-process transport.
- Review the audit semantics when user creation commits but durable audit delivery fails.

### What should be done in the future

- Before implementation, write API compile tests from a synthetic external package.
- Consider migrating xapp to the public bootstrap and transport helpers to prevent two implementations.

### Code review instructions

- Begin at `pkg/embeddedidp/options.go:50` and `pkg/embeddedidp/provider.go:40`.
- Continue with `internal/admin/users.go:54` and `internal/authn/password.go:70`.
- Compare the proposed public surface to `cmd/tinyidp-xapp/state.go:65-239`.
- Validate the same-process architecture in `cmd/tinyidp-xapp/development_app.go:76-178` and route mounts at lines 269-276.

### Technical details

Current boundary summary:

```text
public and reusable:
  embeddedidp.Provider
  embeddedidp.Options
  idpstore.Store
  sqlitestore.Open
  idpui.InteractionRenderer

correct but internal/coupled:
  admin account creation
  password-service construction
  signing-key generation
  xapp bootstrap reconciliation
  go-go-goja in-process issuer transport
```

## Step 3: Collect and preserve authoritative references

The source collection focused on APIs and security properties that materially affect the design: standard Go OIDC verification, OAuth2 authorization code and PKCE helpers, SQLite WAL and foreign-key behavior, account-registration responses, password handling, and CSRF for cookie-authenticated state changes.

The sources were extracted into the ticket rather than left as links only. This allows a later intern to reproduce the reasoning even if web navigation or page layout changes.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Gather enough primary and authoritative material to justify the protocol, storage, registration, and CSRF decisions in the guide.

**Inferred user intent:** Preserve the intellectual and standards context behind the proposed implementation.

### What I did

- Searched current official package documentation and security guidance.
- Used Defuddle to extract six sources as Markdown.
- Verified line and character counts to ensure extraction succeeded.

Commands:

```bash
defuddle parse 'https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc' \
  --md -o sources/01-go-oidc-package.md

defuddle parse 'https://pkg.go.dev/golang.org/x/oauth2' \
  --md -o sources/02-go-oauth2-package.md

defuddle parse 'https://www.sqlite.org/wal.html' \
  --md -o sources/03-sqlite-write-ahead-log.md

defuddle parse 'https://www.sqlite.org/foreignkeys.html' \
  --md -o sources/04-sqlite-foreign-key-support.md

defuddle parse 'https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html' \
  --md -o sources/05-owasp-authentication-cheat-sheet.md

defuddle parse 'https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html' \
  --md -o sources/06-owasp-csrf-prevention-cheat-sheet.md
```

### Why

- `go-oidc` documents the intended verifier and discovery API.
- `x/oauth2` documents state options, authorization URL construction, code exchange, and PKCE helpers.
- SQLite documentation defines WAL and connection-level foreign-key behavior.
- OWASP guidance informs registration enumeration, TLS, password bounds, rate controls, and CSRF pre-sessions.

### What worked

- All six Defuddle commands succeeded.
- The files contain 3,679 lines and approximately 247 KiB of extracted Markdown in total.
- None of the files exhibited the known one-line extraction problem.
- The package docs include current PKCE helpers such as `S256ChallengeOption`.

### What didn't work

- The first general web search returned the go-oidc GitHub pull-request page rather than the package reference for one query.
- I used the direct official package URL for extraction instead of relying on that search result.

### What I learned

- The current `x/oauth2` package provides explicit S256 PKCE helpers, so the example does not need to implement challenge encoding itself.
- OWASP explicitly treats login and unauthenticated pre-session CSRF as relevant, which supports a registration-attempt cookie and token.
- OWASP recommends generic registration responses for enumeration resistance, but a no-email-verification username system cannot completely conceal availability.
- SQLite foreign-key enforcement is connection-specific, which supports enabling it explicitly on every app database connection.

### What was tricky to build

The design needed to use external guidance without overstating it. Generic registration responses reduce direct discrepancy, but this example activates accounts immediately and allows chosen usernames. A duplicate username remains behaviorally observable. The guide records this residual risk instead of claiming that a generic JSON message eliminates enumeration.

### What warrants a second pair of eyes

- Confirm whether the example should include email verification before treating it as production-oriented registration.
- Review current go-oidc release selection at implementation time.
- Confirm the app database connection-pool size under WAL through load tests rather than assuming a default.

### What should be done in the future

- Pin exact direct versions of `go-oidc` and `x/oauth2` during implementation.
- Add sources for any new account-recovery or email-verification phase before designing those flows.

### Code review instructions

- Read the source files in numerical order under the ticket's `sources/` directory.
- Compare PKCE and verifier usage in the design to `sources/01-go-oidc-package.md` and `sources/02-go-oauth2-package.md`.
- Compare registration and CSRF decisions to sources 05 and 06.

### Technical details

Source inventory:

```text
01-go-oidc-package.md                     813 lines
02-go-oauth2-package.md                   698 lines
03-sqlite-write-ahead-log.md              193 lines
04-sqlite-foreign-key-support.md          507 lines
05-owasp-authentication-cheat-sheet.md    412 lines
06-owasp-csrf-prevention-cheat-sheet.md  1056 lines
```

## Step 4: Write the intern-ready architecture and implementation guide

The design document was written as a complete implementation map rather than a feature summary. It begins with the current repository boundary, derives the missing public APIs, defines the two-database model, then walks through registration, OIDC, sessions, message authorization, frontend composition, operations, tests, and phased delivery.

The document is 1,666 lines. It uses prose, tables, Mermaid diagrams, Go and TypeScript API sketches, SQL schema, pseudocode, decision records, route tables, security invariants, test matrices, file references, and exit criteria.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Produce the primary technical deliverable at a depth sufficient for a new intern to begin implementation safely.

**Inferred user intent:** Reduce onboarding time while preserving the design rationale and preventing shortcuts that compromise the identity boundary.

### What I did

- Defined explicit included and excluded scope.
- Added user stories and observable acceptance criteria.
- Mapped current implementation with file and line references.
- Added a gap matrix.
- Designed public `pkg/idpaccounts`, bootstrap, and in-process transport APIs.
- Defined separate identity and application databases.
- Wrote the application SQL schema.
- Specified state, nonce, PKCE, callback, session, and logout algorithms.
- Specified anonymous registration CSRF and abuse controls.
- Specified the message API and subject-bound authorization.
- Designed the React, Redux Toolkit, RTK Query, Vite, Bootstrap, and `go:embed` frontend.
- Defined route composition and security headers.
- Added operational state layout, commands, lifecycle, doctor, backup, and restore.
- Added eight implementation phases with detailed tasks and exit criteria.
- Added a definition of done and intern reading order.

### Why

- A new intern needs the rationale before the file list.
- The public API gaps affect the implementation sequence; they cannot be deferred behind UI work.
- The example must distinguish IdP sessions from application sessions.
- The security properties are easier to review when stated as invariants and tests.
- Phase exit criteria provide durable progress checkpoints.

### What worked

- The final guide exceeds the requested detail level while remaining divided into stable numbered sections.
- `git diff --check` reported no whitespace errors for the ticket.
- Searches found no remaining template comments, TODOs, FIXMEs, or forbidden analogy language in the design document.
- The guide ties every major current-state assertion to concrete repository files.

### What didn't work

- N/A. The document patch applied successfully on the first attempt.

### What I learned

- The smallest correct integration sequence starts with public account/bootstrap APIs, not with the message UI.
- An embedded IdP still benefits from a complete relying-party OIDC flow because this preserves the same trust boundary as a separate deployment.
- The example can omit refresh tokens entirely; a local app session is sufficient for the requested use case.
- Separate databases allow messages to survive identity profile changes while ownership remains bound to immutable subject.

### What was tricky to build

The document had to reconcile three goals: remain a small showcase, comply with the repository's React/Redux conventions, and demonstrate production-shaped identity behavior. The solution keeps the product feature set small but does not collapse protocol or session boundaries. The design explicitly defers email verification, password reset, MFA, moderation, federation, and multi-node operation.

### What warrants a second pair of eyes

- The `pkg/idpaccounts` relocation affects existing internal admin and xapp code.
- The audit result when account creation commits but audit delivery fails needs a precise public contract.
- The public-client versus confidential-client decision deserves explicit security review.
- The two-database backup consistency statement should be reviewed by the operator responsible for deployment.
- The React requirement increases example size; confirm the teaching value remains acceptable.

### What should be done in the future

- Resolve the eight open questions in Section 33 before implementation.
- Mark decision records accepted or superseded during Phase 0.
- Keep this diary updated task by task with implementation commits and exact failures.

### Code review instructions

- Read Sections 5 and 6 first to verify the current-state evidence.
- Review Sections 9 through 11 as the proposed public API change.
- Review Sections 13 through 17 as the persistence and security core.
- Review Section 25 against the invariants in Section 24.
- Review Section 29 to confirm implementation ordering.
- Run:

```bash
wc -l ttmp/2026/07/13/TINYIDP-MSGAPP-001--*/design-doc/*.md
git diff --check -- ttmp/2026/07/13/TINYIDP-MSGAPP-001--*
```

### Technical details

The central runtime boundary is:

```text
tiny-idp identity/protocol SQLite
    -> embedded OIDC issuer
    -> authorization code + verified ID token
    -> application session SQLite
    -> subject-authorized message SQLite rows
```

The application never treats direct access to the identity store as authentication.

## Step 5: Relate, reconcile, and validate the ticket

The ticket bookkeeping was brought into alignment with the design before publication. The design relates the seven repository files that most strongly shaped it, while the diary relates the primary design and four decisive stored sources. The index explains the ticket's current status and directs the reader to each artifact.

Docmgr validation passed with no warnings. This establishes that frontmatter, vocabulary, relationships, tasks, and expected ticket structure are internally consistent at the publication gate.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Make the design package searchable, traceable, and structurally valid before upload.

**Inferred user intent:** Ensure the ticket can be trusted as a long-lived implementation handoff rather than a detached Markdown file.

### What I did

- Related seven key repository files to the primary design.
- Related the design and four stored research sources to this diary.
- Replaced the generated index placeholders with a technical overview and direct links.
- Checked the first eight research and design tasks.
- Added three meaningful changelog entries with file notes.
- Ran ticket-scoped doctor validation.

Validation command:

```bash
docmgr doctor --ticket TINYIDP-MSGAPP-001 --stale-after 30
```

Result:

```text
## Doctor Report (1 findings)

### TINYIDP-MSGAPP-001

- ✅ All checks passed
```

### Why

- File relations tie design claims back to implementation evidence.
- Task and changelog state tells a future contributor which work is complete.
- Doctor catches malformed metadata and missing ticket conventions before the PDF is generated.

### What worked

- Every `docmgr doc relate` command succeeded.
- The task checker accepted all eight completed research and design task IDs in one operation.
- Doctor passed on the first run.
- `git diff --check` remained clean.

### What didn't work

- N/A. No vocabulary, frontmatter, stale-document, or relationship warning was reported.

### What I learned

- Seven focused code relationships were sufficient to explain the current architecture without turning the design frontmatter into a repository file inventory.
- The ticket's existing vocabulary covers the design even without dedicated `sqlite` and `registration` topic slugs.

### What was tricky to build

The relation set had to remain focused. Many tests and docs informed the design, but listing all of them in frontmatter would reduce signal. The guide carries the extended file reading order; frontmatter contains only the primary architectural evidence.

### What warrants a second pair of eyes

- Review whether future implementation docs should add `sqlite`, `web`, and `registration` vocabulary topics.
- Review the open architectural questions before checking implementation-phase tasks.

### What should be done in the future

- Rerun doctor after every substantial implementation-diary update.
- Relate new implementation files to the most focused subdocument, not indiscriminately to the ticket index.

### Code review instructions

- Inspect `RelatedFiles` in the design and diary frontmatter.
- Inspect `tasks.md` and `changelog.md` for reconciliation.
- Re-run the exact doctor command above.

### Technical details

Task state after validation:

```text
8 research/design tasks complete
1 validation task ready to check
1 reMarkable delivery task open
```

## Step 6: Render and upload the reMarkable design bundle

The publication workflow first performed a dry run that enumerated every input, the generated PDF name, and the target remote directory. The real bundle command then completed with exit status zero.

The uploaded bundle contains the ticket index, 1,666-line primary design, investigation diary, tasks, and changelog. It uses a level-two table of contents and is stored below the ticket-specific reMarkable directory.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Publish the validated design package to the user's reMarkable device as one navigable PDF.

**Inferred user intent:** Make the full architecture and handoff available for focused reading away from the development workspace.

### What I did

- Ran the required non-destructive bundle dry run.
- Confirmed all five ticket documents and the target path in dry-run output.
- Ran the real non-interactive bundle upload.
- Recorded the command result here.

Dry-run and upload shape:

```bash
remarquee upload bundle \
  index.md \
  design-doc/01-embedded-tiny-idp-sqlite-message-application-analysis-design-and-implementation-guide.md \
  reference/01-investigation-diary.md \
  tasks.md \
  changelog.md \
  --name 'TINYIDP MSGAPP 001 Design Package' \
  --remote-dir '/ai/2026/07/13/TINYIDP-MSGAPP-001' \
  --toc-depth 2 \
  --non-interactive
```

### Why

- A bundle preserves context around the design instead of publishing the primary document without its tasks and diary.
- The dry run verifies input paths and destination without rendering or uploading.
- A ticket-specific directory keeps future implementation and follow-up bundles organized.

### What worked

- The dry run listed the five intended inputs and no unrelated file.
- The real upload process completed with exit status zero.
- Pandoc and the local reMarkable upload pipeline reported no rendering error.

### What didn't work

- The successful real upload emitted no textual confirmation in the captured command output. The process nevertheless completed normally with exit status zero. Per the upload workflow, no routine post-upload listing was performed after a successful command.

### What I learned

- The long design, Mermaid blocks, tables, and nested code examples passed the PDF rendering path without requiring document changes.
- Keeping the bundle limited to first-party ticket documents avoids adding roughly 247 KiB of extracted web sources to the reading PDF.

### What was tricky to build

The ticket contains extensive stored sources, but including them would make the bundle much less navigable and would duplicate reference documentation already summarized and cited by the design. The sources remain in the ticket for traceability; the reMarkable bundle contains the authored deliverables.

### What warrants a second pair of eyes

- Inspect the generated table of contents and Mermaid rendering on the device when convenient.
- If a future upload uses `--force`, remember that replacement can remove annotations from the prior PDF.

### What should be done in the future

- Upload a new version only after substantive implementation updates.
- Prefer a separately named implementation report rather than overwriting an annotated design bundle.

### Code review instructions

- Compare the five dry-run input paths to the ticket document list.
- Confirm the intended remote path is `/ai/2026/07/13/TINYIDP-MSGAPP-001`.
- Do not repeat the upload merely to obtain a textual confirmation when exit status is already zero.

### Technical details

```text
Bundle name: TINYIDP MSGAPP 001 Design Package
Remote directory: /ai/2026/07/13/TINYIDP-MSGAPP-001
Table of contents depth: 2
Upload exit status: 0
```

## Step 7: Reconcile the implementation plan and freeze the contract

Implementation resumed after the public embedding-foundations ticket had landed.
The first step compared the 2026-07-13 design against current source and Git
history. This established that the originally proposed account, bootstrap, and
in-process transport work already exists and is tested. Reimplementing those
phases would create duplication and violate the repository's no-adapter rule.

The application contract is now executable in
`examples/tinyidp-message-app/contracts.go`. It fixes the example path, client
ID, issuer mount, application routes, cookie names, and named security-invariant
test inventory. A Go AST import-boundary test prevents the example from using
tiny-idp internal packages merely because it shares the module.

**Commit (code and contract docs):** `ec45bb2` — "feat(msgapp): freeze implementation contract"

### What I did

- Read the complete diary template and existing ticket diary before editing.
- Read the phased plan and all open questions in Sections 29 through 34.
- Inspected current `pkg/idpaccounts`, `embeddedidp.Bootstrap`, and
  `NewInProcessIssuerTransport` implementations and their Git commits.
- Added 36 phase-tagged implementation tasks to `tasks.md`.
- Created `reference/02-implementation-contract-and-task-map.md`.
- Added `contracts.go` and `contracts_test.go` in the new example directory.
- Accepted the recommended design defaults, including public PKCE, absolute
  eight-hour app sessions, append-only messages, and a no-fallback transport.

Commands:

```bash
rg --files examples cmd pkg | rg 'message|idpaccounts|embeddedidp/(bootstrap|inprocess)'
git log --oneline -12
gofmt -w examples/tinyidp-message-app/contracts.go \
  examples/tinyidp-message-app/contracts_test.go
go test ./examples/tinyidp-message-app ./pkg/idpaccounts ./pkg/embeddedidp
```

Result:

```text
ok github.com/manuel/tinyidp/examples/tinyidp-message-app
ok github.com/manuel/tinyidp/pkg/idpaccounts
ok github.com/manuel/tinyidp/pkg/embeddedidp
```

### Why

- Implementation tasks must be independently checkable over multiple sessions.
- Public foundation work is a prerequisite, not application-owned code.
- Freezing names and routes before schema work prevents backend, frontend, and
  documentation from drifting independently.
- The AST guard makes copyability a continuously tested property.

### What worked

- The focused public-package suites passed without modification.
- A package containing only contract source and tests compiles normally under
  `go test`; the application entry point can be added later.
- The new tasks map directly to the eight design phases.

### What didn't work

- The first broad ticket read exceeded the tool output limit and was truncated.
  I repeated the read using line counts, section headings, and targeted chunks,
  including the complete phased plan and continuation point.

### What I learned

- Commits `edd1479`, `7481ee1`, and `3e17e79` satisfy the original Phase 1 and
  Phase 2 architectural gaps.
- The previously completed `examples/embedded` relying party is useful protocol
  evidence, but MSGAPP still needs durable login attempts and app sessions.
- The ticket's original ten tasks describe research delivery only, so phase
  implementation could not be tracked precisely until now.

### What was tricky to build

The main risk was confusing chronological ticket ownership with code ownership.
The foundational APIs were motivated by MSGAPP but implemented in a later,
shared ticket. The correct action is to verify and cite those commits, not copy
their behavior into the example or mark them as new MSGAPP code.

### What warrants a second pair of eyes

- Confirm `examples/tinyidp-message-app` is the desired long-term example name.
- Confirm append-only messages remain the preferred first release.
- Review the public PKCE choice before a future confidential-client variant is
  introduced.

### What should be done in the future

- Each invariant name in `securityInvariantTests` must acquire an actual test as
  its owning phase lands.
- Keep the no-internal-import test even after the example is moved or copied.

### Code review instructions

- Start at `reference/02-implementation-contract-and-task-map.md`.
- Review constants and inventories in `examples/tinyidp-message-app/contracts.go`.
- Review AST enforcement in `contracts_test.go`.
- Run the focused test command above.

### Technical details

The next unchecked task is Phase 3 state-root and manifest implementation,
followed by the application SQLite migration ledger and repositories.

## Continuation point

Phase 0 and the inherited public foundations are reconciled. Continue with
Phase 3 in `examples/tinyidp-message-app`: state paths, owner-only secrets,
application SQLite migrations, login attempts, sessions, registration attempts,
messages, cleanup, restart, and concurrency tests.

## Step 8: Implement the owner-only application state root

This step implemented the first Phase 3 task as an independent unit. The
message application now has deterministic paths for identity, application,
secret, audit, and manifest state. Initialization is idempotent for the same
public origin and fails on origin or client-contract drift.

Secrets are generated with `crypto/rand`, written through owner-only temporary
files, synced, and atomically renamed. Validation checks both file mode and
exact secret length before later provider or session construction can use the
state.

**Commit (code):** `9f4a4e2` — "feat(msgapp): add secure state root"

### What I did

- Added `statePaths` and `resolveStatePaths`.
- Added a versioned, non-secret JSON manifest.
- Added canonical HTTP(S) origin validation; plain HTTP is loopback-only.
- Added owner-only directory creation and two independent 32-byte secrets.
- Added atomic JSON and byte-file writers.
- Added state validation and damage detection.
- Added tests for permissions, idempotency, conflicts, unsafe origins, and
  truncated secrets.

Command:

```bash
gofmt -w examples/tinyidp-message-app/state.go \
  examples/tinyidp-message-app/state_test.go
go test ./examples/tinyidp-message-app
```

Result:

```text
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.013s
```

### Why

- Identity and application databases require a stable, explicit ownership
  boundary before either is opened.
- Token and application-session secrets serve different cryptographic purposes
  and must not be reused.
- An origin change modifies issuer and redirect semantics, so startup must fail
  rather than silently rewrite initialized state.

### What worked

- First focused test run passed.
- Repeating initialization preserved `CreatedAt` and both secrets.
- File and directory permission assertions pass on the target filesystem.

### What didn't work

- N/A. The implementation formatted and passed on the first test run.

### What I learned

- The xapp state implementation remains a useful operational reference, but the
  message app needs a smaller manifest and does not need object-storage paths.
- Loopback HTTP can be admitted without a broad insecure-mode fallback.

### What was tricky to build

Atomic creation must not overwrite an existing temporary file left by another
initializer. `O_EXCL` makes concurrent or stale temporary state explicit. The
published destination is never opened with truncation.

### What warrants a second pair of eyes

- Review whether directory `Chmod(0700)` should reject rather than repair broad
  permissions on an existing root.
- Review whether manifest publication needs an explicit parent-directory sync
  for the required crash-consistency level.

### What should be done in the future

- Doctor must reuse `validateStateRoot` and report stable reason codes without
  printing secret contents.
- CLI initialization must avoid logging raw public input beyond the normalized
  non-secret origin.

### Code review instructions

- Start at `initializeStateRoot` in `state.go`.
- Review `writeBytesAtomic` for cleanup and publication ordering.
- Run `go test ./examples/tinyidp-message-app -run 'Test(State|Initialize|Normalize|Validate)'`.

## Continuation point

State-root task is complete. Implement the application SQLite schema ledger and
checksum enforcement next, then add repositories in separate tasks.

## Step 9: Add the checksummed application schema

The second Phase 3 task added the application-owned SQLite boundary. The store
opens only the message application's database, applies a checksummed migration
ledger, configures WAL and production-shaped connection pragmas, and refuses to
start when an applied migration's filename or contents no longer match history.

**Commit (code):** `c41ba0b` — "feat(msgapp): add checksummed app schema"

### What I did

- Added `appStore` over `database/sql` and `github.com/mattn/go-sqlite3`.
- Configured busy timeout, WAL, FULL synchronous mode, foreign keys, immediate
  transactions, and a bounded four-connection pool.
- Created the migration ledger and initial login-attempt, session,
  registration-attempt, and message tables.
- Added SHA-256 checksums over version, filename, and SQL text.
- Protected the database directory as 0700 and database file as 0600.
- Added tests for tables, schema version, checksums, modes, and pragmas.

Command and result:

```text
go test ./examples/tinyidp-message-app
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.021s
```

### Why

- The app schema must not be added to tiny-idp's identity migration stream.
- Checksums turn historical migration edits into startup failures rather than
  undetected schema divergence.
- WAL and bounded waiting support concurrent browser reads and short writes
  without permitting unbounded request stalls.

### What worked

- A clean database reached schema version one.
- Tampering the recorded checksum caused the next open to fail.
- SQLite reported `wal`, synchronous level `2`, foreign keys enabled, and a
  5000-millisecond busy timeout.

### What didn't work

- N/A. Focused tests passed on the first run.

### What I learned

- The sqlite3 DSN can encode every required connection pragma without executing
  mutable setup SQL after pooling begins.
- The application directory should be chmod'd even when its parent already
  exists, because `MkdirAll` does not tighten an existing mode.

### What was tricky to build

The migration ledger itself must exist before a migration transaction can
record version one. The ledger creation is intentionally a small bootstrap DDL;
all application domain tables remain inside the checksummed migration.

### What warrants a second pair of eyes

- Review whether four open SQLite connections is the correct initial bound.
- Review the desired durability/performance tradeoff of `synchronous=FULL`.
- Confirm the migration checksum should continue to include the filename.

### What should be done in the future

- Doctor must compare supported and applied schema versions and checksums.
- Every new schema change must append a migration; applied SQL is immutable.

### Code review instructions

- Start at `openAppStore`, then `applyMigration`.
- Inspect `appMigrations[0]` against Section 13 of the design.
- Run `go test ./examples/tinyidp-message-app -run 'TestApplication|TestMigration'`.

## Continuation point

Add the login-attempt repository and its atomic, one-time consume tests next.

## Step 10: Persist and atomically consume OIDC login attempts

The third Phase 3 task made authorization responses one-time across restarts and
concurrent callbacks. Only SHA-256 of OAuth state is persisted. Nonce, PKCE
verifier, local return path, and lifetime remain server-side and are returned
only by the winning consume operation.

**Commit (code):** `2603c18` — "feat(msgapp): persist one-time login attempts"

### What I did

- Added bounded login-attempt creation.
- Added one-statement `UPDATE ... RETURNING` consumption conditioned on an
  unconsumed row and a strictly future expiry.
- Mapped wrong, expired, and replayed state to one public sentinel error.
- Added successful, wrong-state, expiration-boundary, replay, and sixteen-way
  concurrent consume tests.

Command and result:

```text
go test ./examples/tinyidp-message-app -run TestLoginAttempt -count=1
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.023s
```

### Why

- State stored only in a browser cookie cannot provide durable replay defense.
- Hashing state reduces useful credential material in a database disclosure.
- One SQL statement gives the consume transition a clear linearization point.

### What worked

- Exactly one of sixteen concurrent consumers won.
- Wrong, expired-at-the-boundary, and replayed values were indistinguishable to
  the caller.

### What didn't work

- N/A. The focused test passed on its first run.

### What I learned

- SQLite `UPDATE ... RETURNING` expresses the one-time transition without a
  caller-managed read/modify/write transaction.

### What was tricky to build

Expiry uses `expires_at > now`, so a request at the exact expiry instant is
invalid. This boundary is intentional and covered explicitly.

### What warrants a second pair of eyes

- Confirm a 1024-byte raw state input cap is appropriately conservative.
- Review whether nonce and PKCE verifier should receive explicit database
  length constraints in a later migration.

### What should be done in the future

- The callback handler must consume before token exchange and must never restore
  a consumed attempt after downstream failure.
- Cleanup must retain recently consumed attempts long enough for audit/debugging
  policy, then delete them.

### Code review instructions

- Review `consumeLoginAttempt` and the concurrent winner assertion.
- Run the focused command above with `-race` during the Phase 3 exit gate.

## Continuation point

Implement hashed application sessions with independent CSRF secrets, lookup,
revocation, expiry, and restart tests.

## Step 11: Add durable, independently revocable application sessions

This task implemented the relying party's own session boundary. The raw browser
token is accepted only at the repository API and reduced to SHA-256 before
persistence. The row contains verified subject and display name snapshots plus
an independent 32-byte CSRF secret, absolute expiry, revocation time, and
bounded last-seen metadata.

**Commit (code):** `cf25884` — "feat(msgapp): add durable app sessions"

### What I did

- Added session creation, lookup, optimistic last-seen touch, and revocation.
- Made revoked, expired, unknown, malformed, and replayed session tokens share a
  stable unavailable error.
- Added tests proving only the token hash is stored, exact expiry is rejected,
  revocation is immediate, and a session survives store restart.

Command and result:

```text
go test ./examples/tinyidp-message-app -run 'TestSession|TestApplicationSession' -count=1
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.019s
```

### Why

- The application cannot use the IdP's private browser cookie as authorization.
- Hash-only storage reduces direct bearer-cookie exposure from an application
  database disclosure.
- Local revocation and lifetime remain relying-party policy.

### What worked

- Lookup after close/reopen succeeded.
- Revoked and exactly expired sessions failed consistently.
- The persisted token bytes matched SHA-256 and not the raw token.

### What didn't work

- N/A. Focused tests passed on the first run.

### What I learned

- Session CSRF material is not a browser bearer credential and must remain
  available to the server after restart, unlike the raw session token.

### What was tricky to build

Last-seen updates use the prior timestamp in the SQL predicate. This prevents
concurrent requests from moving metadata backward or blindly overwriting a more
recent touch without turning reads into mandatory writes.

### What warrants a second pair of eyes

- Review whether SHA-256 token hashes should instead be keyed with the app
  session secret; the raw token already has high entropy, so offline guessing is
  not practical, but domain separation may improve uniformity.
- Review the future bounded touch frequency in middleware.

### What should be done in the future

- Middleware must cap cookie length before lookup and never expose CSRF secret
  outside the authenticated session endpoint.
- Cleanup must remove expired/revoked sessions according to retention policy.

### Code review instructions

- Review `createAppSession`, `getAppSession`, and `revokeAppSession`.
- Run the focused command above and the Phase 3 race gate later.

## Continuation point

Implement durable one-time registration attempts with independent anonymous
CSRF material next.

## Step 12: Add one-time registration attempts

Registration now has a durable anonymous pre-session. The raw cookie token is
hashed before persistence; a separate 32-byte CSRF secret is returned only to
the single successful consume operation.

**Commit (code):** `db24682` — "feat(msgapp): persist registration attempts"

### What I did

- Added bounded create and atomic `UPDATE ... RETURNING` consume operations.
- Added wrong-token, replay, expiry semantics and sixteen-way concurrency tests.

### What worked

- Exactly one concurrent consumer won; all losers received the stable
  unavailable error.

### What didn't work

- N/A. `go test ./examples/tinyidp-message-app -run TestRegistrationAttempt
  -count=1` passed in 0.015 seconds.

### What was tricky to build

Anonymous CSRF material must survive restart but must not authenticate the user.
The record authorizes only one registration submission.

### What warrants a second pair of eyes

- Confirm the HTTP phase consumes the attempt before password hashing begins.

### What should be done in the future

- Bind the CSRF proof to this secret and enforce Origin independently.

### Code review instructions

- Review `registration_attempts.go` beside `login_attempts.go` for consistent
  one-time semantics.

## Step 13: Add the append-only message repository

The application can now persist plain-text messages and list them with a stable
keyset cursor ordered by `(created_at DESC, id DESC)`. Ownership fields are
required inputs to the repository and will be supplied only from session
middleware by the Phase 6 HTTP handler.

**Commit (code):** `b58c057` — "feat(msgapp): add message repository"

### What I did

- Added body line-ending normalization, whitespace rejection, 1000-rune and
  4096-byte bounds.
- Added inserts and cursor pagination with ID tie-breaking.
- Added validation, pagination, and sixteen-way concurrent insertion tests.

### What worked

- Identical timestamps paginated without gaps or duplicates.
- Concurrent inserts received sixteen distinct IDs.

### What didn't work

- N/A. Focused tests passed in 0.052 seconds.

### What was tricky to build

Timestamp-only pagination is unstable when multiple writes share a timestamp;
the row ID is therefore part of both ordering and cursor predicates.

### What warrants a second pair of eyes

- Review preservation of leading/trailing user whitespace; only all-whitespace
  bodies are rejected.

### What should be done in the future

- The HTTP request type must contain only `body`; subject spoofing must fail as
  an unknown JSON field.

### Code review instructions

- Review `listMessages` SQL and `TestMessageCursorPaginationIsStable`.

## Step 14: Add retention cleanup and close Phase 3

The final repository task added bounded cleanup for expired or sufficiently old
terminal login attempts, registration attempts, and sessions. Messages are not
part of protocol cleanup and remain append-only.

**Commit (code):** `3782f2c` — "feat(msgapp): add protocol state cleanup"

### What I did

- Added typed cleanup counts and configurable terminal retention.
- Proved fresh pending login state survives cleanup.
- Ran the complete focused suite and race detector.

### What worked

```text
go test ./examples/tinyidp-message-app -count=1
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.130s
go test -race ./examples/tinyidp-message-app -count=1
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 1.135s
```

### What didn't work

- N/A.

### What was tricky to build

Retention compares both expiry and terminal timestamps to the same cutoff so a
recently consumed record is retained even though it can no longer be used.

### What warrants a second pair of eyes

- Review the eventual default retention duration and whether cleanup should use
  one transaction for all counts.

### What should be done in the future

- Schedule cleanup with provider maintenance and expose only aggregate results.

### Code review instructions

- Review `cleanup.go`, then run the two phase-exit commands above.

## Continuation point

Phase 3 persistence is complete. Begin Phase 4 by composing the embedded
provider and OIDC discovery/verifier over the exact in-process transport, then
implement begin-login and callback against the durable repositories.

## Step 15: Add the standards-based OIDC PKCE client

The message application now has a protocol client that uses the ticket's
selected public libraries: `github.com/coreos/go-oidc/v3` for discovery and ID
token verification, and `golang.org/x/oauth2` for authorization URL, code
exchange, and S256 PKCE options. It keeps browser state in the durable
application database rather than in a transient in-memory map.

**Commit (code):** `36c1727` — "feat(msgapp): add OIDC PKCE client"

### What I did

- Added the two direct module dependencies at the shared graph's resolved
  versions: go-oidc v3.19.0 and x/oauth2 v0.36.0.
- Constructed discovery and a verifier using a caller-supplied HTTP client.
- Implemented durable state, nonce, PKCE verifier creation and S256 redirect.
- Implemented callback core: consume state, exchange code with verifier,
  verify ID token, compare nonce, and create a separate opaque app session.
- Added strict local `return_to` normalization.
- Added discovery integration coverage against a real embedded provider through
  `NewInProcessIssuerTransport`.

Commands:

```text
go test ./examples/tinyidp-message-app -run 'Test(BeginLogin|ReturnTo|OIDCClient)' -count=1
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.171s
```

### What didn't work

`go mod tidy` attempted to rewrite the entire shared module graph, upgraded
unrelated xapp dependencies, and changed 67 `go.mod` lines plus 525 `go.sum`
lines. I restored those two agent-modified files, then added only the direct
OIDC dependencies manually. The final commit changes three `go.mod` lines and
two `go.sum` lines.

### What was tricky to build

The callback consumes the login attempt before token exchange. A downstream
exchange failure therefore cannot make an authorization response replayable.
This is intentional: an OAuth authorization code is one-time, and retrying
after an uncertain exchange risks an ambiguous security state.

### What warrants a second pair of eyes

- Review whether callback exchange failures should have a dedicated operator
  audit event without revealing code or token material.
- Review the eight-hour absolute app session lifetime before production mode.

### What should be done in the future

- Bind `finishLogin` to HTTP handlers and test a full browser/IdP callback.
- Add a test asserting nonce mismatch does not establish a session.

### Code review instructions

- Start with `newOIDCClient`, `beginLogin`, then `finishLogin`.
- Review the in-process discovery test and the return-path adversarial cases.
- Run the focused command above.

## Continuation point

Add HTTP handlers for `/auth/login`, `/auth/callback`, session middleware, and
local logout. Then drive a full IdP login form in a same-process test.

## Step 16: Bind OIDC and app sessions to HTTP routes

The OIDC client is now exposed through application routes. These routes keep
the IdP handler under `/idp/`, expose a separate `/auth/*` and `/api/session`
surface, set only an opaque app-session cookie, and use the session's independent
CSRF secret for local logout.

**Commit (code):** `bfb79fa` — "feat(msgapp): add auth HTTP handlers"

### What I did

- Added `GET /auth/login`, `GET /auth/callback`, `GET /api/session`, and
  `POST /auth/logout`.
- Propagated the exact in-process back-channel HTTP client into code exchange
  and ID-token verification context.
- Added response `no-store`, no-referrer, and nosniff headers.
- Added JSON session disclosure for same-origin UI code and constant-time CSRF
  comparison for local logout.
- Added tests showing an app session is independent of the IdP and that logout
  revokes it durably.

Command and result:

```text
go test ./examples/tinyidp-message-app -run 'Test(SessionEndpoint|LoginRejects|BeginLogin|ReturnTo|OIDCClient)' -count=1
ok github.com/manuel/tinyidp/examples/tinyidp-message-app 0.233s
```

### What didn't work

- N/A. The focused handler test passed on the first run.

### What was tricky to build

`oauth2.Config.Exchange` accepts its HTTP client through context. Without
placing the exact in-process client into callback context, code exchange would
try the network even though discovery used the in-process provider correctly.

### What warrants a second pair of eyes

- Review whether logout should later offer a separate explicit RP-initiated IdP
  logout route; local logout currently has deliberately narrower semantics.
- Review the CSP selected once the React shell is added.

### What should be done in the future

- Add a successful full callback test that parses the IdP form and follows the
  exact redirects.
- Attach current-session lookup as middleware for the message API.

### Code review instructions

- Review `handleCallback`, then `handleLogout`.
- Run the focused command above.

## Continuation point

Phase 4 needs its positive browser/IdP callback test and then can be closed.
Proceed to Phase 5 registration endpoints only after the runnable composition
can initialize identity and application stores together.

## Step 17: Prove the complete browser-to-embedded-IdP login flow

The application now has a positive integration test that takes the same route
as a browser: application login redirect, the mounted IdP authorization page,
credential form submission, callback, and application-session lookup. This
closes the critical gap between testing the OIDC client methods independently
and proving their actual HTTP composition.

The test also exposed a transport-boundary detail that independent unit tests
had not exercised. Discovery used the explicit in-process transport, but the
callback exchange and JWKS verification need that client placed in their own
context as well. The callback now makes that binding before consuming state and
performing back-channel requests, so it cannot silently use the host network.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Continue the planned implementation without pausing, completing the next coherent security-critical increment and its evidence.

**Inferred user intent:** Produce a runnable, thoroughly verified self-contained relying-party example through small reviewable commits and a traceable implementation diary.

**Commit (code):** `ee793d8` — "test(msgapp): cover embedded browser login flow"

### What I did

- Bound `finishLogin` to `c.http` with `oauth2.HTTPClient`, matching OIDC
  discovery and preventing a network fallback during code exchange or JWKS
  verification.
- Added `TestBrowserLoginCompletesAgainstEmbeddedProvider`, which constructs an
  identity SQLite store, account service, conservative browser client,
  provider, in-process transport, and application store.
- Used an `httptest.NewUnstartedServer` listener address to configure exact
  redirect URIs before starting the shared public application/IdP handler.
- Used a cookie jar to fetch the IdP form, extract only its test-visible hidden
  CSRF and interaction inputs, submit valid credentials, follow the callback,
  prove the application session exists, and prove callback replay fails.
- Ran the focused and full message-app package tests.

### Why

OIDC correctness is a composition property. State, nonce, PKCE, token exchange,
ID-token verification, cookie routing, and redirect handling can each pass
isolated tests while their integrations use the wrong HTTP client or cookie
scope. The end-to-end test makes the intended browser/public-handler versus
back-channel/in-process-transport split executable.

### What worked

```text
go test ./examples/tinyidp-message-app -run 'TestBrowserLoginCompletesAgainstEmbeddedProvider' -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.572s

go test ./examples/tinyidp-message-app -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.949s
```

The callback redirects to its originally requested local path, establishes an
opaque application session, and returns `502` for the replayed state. The
one-time state therefore remains the first application-level replay boundary
even when the browser retains both app and IdP cookies.

### What didn't work

The first sandboxed integration-test invocation could not open the loopback
listener required by `httptest`:

```text
panic: httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted
```

I reran the same `go test` command with the approved loopback-network
permission. The code did not need a change for that expected sandbox boundary.

### What I learned

- `oauth2.Config.Exchange` takes a custom client from context, while
  `oidc.IDTokenVerifier.Verify` also needs that context to fetch JWKS. Setting
  it only during discovery is insufficient.
- An unstarted test server supplies a listener early enough to establish
  identity-client redirect registrations before its public handler starts.
- A genuine browser-flow test should use a cookie jar and the public handler;
  directly invoking `Provider.Handler` would bypass the precise routing and
  cookie conditions the example is meant to demonstrate.

### What was tricky to build

The provider needs a fixed issuer and exact redirect URI, while an ordinary
`httptest.NewServer` chooses its port only after construction. I used
`httptest.NewUnstartedServer`, derived `baseURL` from its preallocated
listener, built the identity client and provider with that exact value, then
assigned the combined app handler and started the server. This preserves strict
redirect validation without a hard-coded port or a test-only configuration
bypass.

### What warrants a second pair of eyes

- Review the desired HTTP error mapping for a consumed callback state. The
  current external `502` is intentionally generic but a production app may
  prefer a non-diagnostic user-facing error page and a distinct operator audit
  event.
- Review future test coverage for nonce mismatch and expired state. The
  callback core checks these conditions, and repository tests cover expiry,
  but an end-to-end negative case can make the externally observable behavior
  more explicit.

### What should be done in the future

- Begin Phase 5 with an anonymous registration pre-session and CSRF endpoint.
- Keep login and registration abuse control separate: the provider's login
  limiter does not automatically protect anonymous account creation.
- Add an explicit RP-initiated IdP logout decision before claiming coordinated
  logout semantics.

### Code review instructions

- Start in `examples/tinyidp-message-app/oidc_client.go` at `finishLogin` and
  verify that every OIDC back-channel action shares `c.http`.
- Read `TestBrowserLoginCompletesAgainstEmbeddedProvider` in
  `examples/tinyidp-message-app/app_http_test.go` from setup through replay;
  confirm it goes through `/auth/login`, `/idp/authorize`, `/auth/callback`,
  and `/api/session` rather than calling private implementation methods.
- Validate with:

  ```text
  go test ./examples/tinyidp-message-app -count=1
  ```

### Technical details

The tested request sequence is:

```text
GET /auth/login?return_to=/messages
  -> 303 /idp/authorize?...state...nonce...code_challenge...
GET /idp/authorize?...                              [browser + IdP cookies]
  -> 200 login/consent form with CSRF + interaction
POST /idp/authorize                                 [credentials + CSRF]
  -> 303 /auth/callback?state=...&code=...
GET /auth/callback?state=...&code=...
  -> in-process POST /idp/token and JWKS verification
  -> 303 /messages + tinymsg_app_session
GET /api/session
  -> {"authenticated":true,...}
```

The exact redirect URI is registered through
`embeddedidp.BrowserClient(clientID, []string{baseURL + callbackPath}, ...)`;
the app never relaxes that provider validation merely to simplify tests.

## Step 18: Issue anonymous registration pre-sessions

Phase 5 now exposes its first public registration route: `GET
/api/registration`. It creates a fresh short-lived durable attempt, places only
the opaque random token in an `HttpOnly`, `SameSite=Lax` cookie, and returns the
independent CSRF value to same-origin JavaScript as JSON. No user, browser
session, or application session is created by this endpoint.

Keeping issuance separate from account creation makes the registration state
machine explicit and reviewable. The following POST increment can consume this
single-use record before invoking password work, rather than accepting a
cookie-only account-creation request.

### Prompt Context

**User prompt (verbatim):** (same as Step 17)

**Assistant interpretation:** Continue the next planned registration task as a small independently testable implementation unit.

**Inferred user intent:** Ensure account creation will begin with a durable anti-CSRF precondition rather than a loose anonymous form submission.

**Commit (code):** `44f2fda` — "feat(msgapp): issue registration pre-sessions"

### What I did

- Added `registrationAttemptLifetime`, currently ten minutes.
- Registered `GET /api/registration` before any account-creation handler.
- Generated an opaque 32-byte registration token and a separate 32-byte CSRF
  secret with `crypto/rand`.
- Stored only the token hash through `createRegistrationAttempt` and set the
  opaque token as an `HttpOnly`, Lax, path-root cookie.
- Returned the CSRF secret base64url-encoded in a JSON response while the
  outer handler supplies `Cache-Control: no-store`.
- Added a test that reads the cookie and JSON response, atomically consumes
  the corresponding durable attempt, and checks its secret and expiry.

### Why

Registration changes identity state but occurs before the visitor has an app
session. A dedicated one-time pre-session gives it a CSRF boundary that is
independent of both the application session and tiny-idp's authorization-form
CSRF mechanism. It also gives later rate-limit and audit code a durable
attempt-level result to associate with the request.

### What worked

```text
go test ./examples/tinyidp-message-app -run 'TestRegistrationEndpointCreatesOneTimePreSession' -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.030s
```

The test proves the client sees no raw database identifier, the cookie is not
available to JavaScript, the CSRF response is 32 bytes after decoding, and the
server can retrieve its exact durable counterpart once.

### What didn't work

The first test compilation omitted `encoding/json` from the test imports:

```text
examples/tinyidp-message-app/app_http_test.go:95:12: undefined: json
```

I added that single import and reran the focused test successfully. This was a
test-only build omission; no runtime design changed.

### What I learned

- The registration cookie need not contain CSRF material. Keeping the two
  values independent means a cookie theft alone is not enough for a forged
  same-origin request header, while an exposed JSON response alone is not
  enough without the `HttpOnly` token cookie.
- The existing repository's SHA-256 token-at-rest convention applies directly
  to anonymous pre-sessions and keeps the database from becoming a bearer-token
  recovery source.

### What was tricky to build

The CSRF value has two representations: a fixed-size raw byte slice required
by the SQLite record and a base64url string required by the JSON/header
protocol. The handler creates raw random bytes, stores those bytes unchanged,
and encodes only at the response boundary; the test decodes and compares the
raw values. This avoids an accidental double-encoding or variable-length
database secret.

### What warrants a second pair of eyes

- Review the ten-minute lifetime against the eventual frontend's form
  behavior; a timeout should lead to a clean request for a replacement
  pre-session, not a confused retry.
- Review whether a new registration GET should invalidate prior anonymous
  attempts in the same browser. The current intentionally simple model allows
  independently issued short-lived attempts; the POST remains one-time.

### What should be done in the future

- Add `POST /api/accounts` with strict bounded JSON decoding, token consumption,
  constant-time CSRF comparison, and password-buffer clearing.
- Add origin and Fetch Metadata checks before consuming registration state, then
  add independent address and normalized-login rate limits.
- Ensure successful account creation returns a login next-step and never
  manufactures an IdP or app session.

### Code review instructions

- Review `handleRegistration` in
  `examples/tinyidp-message-app/app_http.go`, especially the separation between
  token, CSRF secret, durable store, cookie, and JSON response.
- Review `TestRegistrationEndpointCreatesOneTimePreSession` in
  `examples/tinyidp-message-app/app_http_test.go`.
- Validate with:

  ```text
  go test ./examples/tinyidp-message-app -run 'TestRegistrationEndpointCreatesOneTimePreSession' -count=1
  ```

### Technical details

```text
GET /api/registration
  raw registration token --SHA-256--> registration_attempts.token_hash
  raw CSRF secret ------------------> registration_attempts.csrf_secret
  raw registration token -----------> HttpOnly tinymsg_registration cookie
  base64url(CSRF secret) -----------> {"csrfToken":"..."}
```

The next unsafe request must present both final arrows' values. The response
does not include an account identifier, an IdP cookie, an app session token, or
a password-related field.

## Step 19: Create accounts through the public identity service

`POST /api/accounts` now consumes the anonymous registration pre-session and
uses the public `idpaccounts.Service` to create the identity record and password
credential. The handler is intentionally an relying-party boundary rather than
a second identity implementation: it never writes identity tables directly and
it passes through the same password policy, Argon2 work bound, normalization,
and audit behavior used by tiny-idp login.

The endpoint accepts only a single bounded JSON object with the four documented
fields. It rejects unknown fields and trailing JSON, requires the one-time
cookie plus matching CSRF header, uses a stable public error for invalid or
duplicate requests, clears its mutable password byte copy, and returns only the
next login route. Successful registration creates no browser or app session.

### Prompt Context

**User prompt (verbatim):** (same as Step 17)

**Assistant interpretation:** Implement the next planned registration task with strict request handling and a demonstrable supported tiny-idp integration boundary.

**Inferred user intent:** Let a visitor create a durable account safely, while preserving the architectural rule that the embedded provider owns authentication and the relying party owns only its own session after OIDC login.

**Commit (code):** `f4a57ce` — "feat(msgapp): add bounded account creation"

### What I did

- Added the `accounts *idpaccounts.Service` dependency to the application
  composition constructor and passed it through the full embedded-login test.
- Added `POST /api/accounts` and `createAccountRequest` with login, display
  name, password, and password confirmation only.
- Required `application/json`, applied `http.MaxBytesReader` at 64 KiB,
  enabled `json.Decoder.DisallowUnknownFields`, and rejected a second JSON
  value.
- Consumed the registration token once, checked its independent CSRF secret
  with the existing constant-time comparison, and rejected absent/invalid
  pre-sessions with `403`.
- Copied the password string into a mutable byte slice, deferred clearing that
  slice, and passed it to `idpaccounts.Service.Create`.
- Returned the same `422 {"error":"account could not be created"}` JSON for
  validation, duplicate-login, and password-policy rejection; unexpected
  storage or account-service errors become `503`.
- Deleted the anonymous registration cookie on success and returned
  `201 {"next":"/auth/login"}` without an app-session cookie.
- Added focused tests for missing pre-session/CSRF, successful normal
  authentication after registration, unknown fields, and multiple JSON values.

### Why

The identity service already contains the security-sensitive implementation of
user normalization, password acceptance, bounded Argon2 work, credential
storage, and account-creation audit. Reimplementing any of that in the example
would create divergent authentication behavior and teach consumers an internal
or unsafe integration path. The example's responsibility is to validate its
public HTTP protocol and call that service once.

### What worked

```text
go test ./examples/tinyidp-message-app -run 'TestCreateAccount' -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.524s

go test ./examples/tinyidp-message-app -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 1.687s
```

The successful-path test calls `AuthenticatePassword` through the same public
service after HTTP registration. This shows the persisted account is usable by
the provider API, rather than merely asserting a local database row.

### What didn't work

N/A. The focused account-creation tests passed on their first implementation
run.

### What I learned

- `idpaccounts.Service.Create` exposes the right narrow abstraction for
  registration: the app supplies public profile fields and a password buffer;
  the service owns generated identity IDs, credential hashing, and atomic
  persistence.
- `json.Decoder.DisallowUnknownFields` alone does not prove a request contains
  exactly one value. A second `Decode` must explicitly require `io.EOF`.
- Buffer clearing is necessarily partial at a Go HTTP/JSON boundary because
  JSON string values are immutable strings; copying to a dedicated mutable
  password slice limits the clearable secret-bearing lifetime before hashing.

### What was tricky to build

The registration attempt must become terminal before calling expensive password
work. `currentRegistrationAttempt` performs the atomic SQL consume and only
then the handler checks the CSRF header and decodes the request. This means a
failed malformed or forged submission spends that one pre-session, which is
intentional replay resistance and matches the ticket algorithm. The frontend
can obtain a replacement pre-session before retrying a local validation error.

### What warrants a second pair of eyes

- Review the account-service error taxonomy as it evolves. The handler maps
  `idpstore.ErrDuplicate` and `idp.ErrPasswordRejected` to stable 422 output;
  a future new expected public rejection type should be deliberately classified
  rather than silently treated as `503`.
- Review whether the visible display-name limits should be enforced at the HTTP
  boundary in addition to `idpstore.User.Validate`; current behavior delegates
  final identity-profile validation to the public service.
- Review the remaining abuse controls before exposing the route publicly:
  Origin/Fetch Metadata validation and address/login-key limits are the next
  task and are not implied by the CSRF token alone.

### What should be done in the future

- Add strict same-origin and cross-site Fetch Metadata rejection before token
  consumption, then add independent fixed-window address and normalized-login
  rate limits.
- Add fixed audit reason codes and tests for duplicate, capacity saturation,
  and provider login after account creation through the complete browser flow.
- Decide and document registration retry UX after a consumed invalid attempt.

### Code review instructions

- Read `handleCreateAccount`, `currentRegistrationAttempt`, and
  `decodeCreateAccountRequest` in
  `examples/tinyidp-message-app/app_http.go` in that order.
- Verify the only identity mutation is
  `a.accounts.Create(... idpaccounts.CreateRequest ...)`; there are no identity
  database SQL statements in the example.
- Review `TestCreateAccountRequiresPreSessionCSRFAndUsesPublicAccountService`
  and `TestCreateAccountRejectsUnknownAndMultipleJSONValues`.
- Validate with:

  ```text
  go test ./examples/tinyidp-message-app -run 'TestCreateAccount' -count=1
  go test ./examples/tinyidp-message-app -count=1
  ```

### Technical details

```text
POST /api/accounts
  cookie tinymsg_registration
  + X-CSRF-Token
       |
       v
atomic consume registration_attempts row
       |
       +-- no row / bad CSRF --> 403
       |
       v
bounded strict JSON -> mutable password []byte -> deferred zero
       |
       v
idpaccounts.Service.Create
       |
       +-- duplicate / policy --> generic 422
       +-- unavailable --------> 503
       +-- success ------------> 201 next=/auth/login, expire registration cookie
```

The endpoint deliberately stops before any call to `/idp/authorize`. The
visitor must still authenticate through the provider-owned form, preserving the
separation between registration and login.

## Step 20: Add origin and abuse controls to the registration perimeter

The account-creation path now rejects cross-origin browser requests before it
spends the one-time registration attempt. It requires the configured exact
public Origin, rejects `Sec-Fetch-Site: cross-site`, and applies fixed-window
limits to both the resolved client address and a hash of tiny-idp's canonical
login form. These controls are additive to, not substitutes for, the existing
cookie-plus-CSRF proof.

This work also made a small public embedding API improvement. The application
must not invent its own username canonicalization for a security key, so
`idpaccounts.NormalizeLogin` now exposes the exact normalization shared by
account creation and password authentication. It returns a canonical value for
non-secret correlation such as a rate-limit key; it is not an account-existence
oracle.

### Prompt Context

**User prompt (verbatim):** (same as Step 17)

**Assistant interpretation:** Complete the registration abuse-control phase using the public tiny-idp API, with tests that make ordering and canonicalization properties visible.

**Inferred user intent:** Make anonymous account creation defensible against cross-site forgery and inexpensive high-volume abuse before considering the example usable.

**Commit (code):** `ed64768` — "feat(msgapp): harden registration boundary"

### What I did

- Retained the normalized public origin in `oidcClient` and copy it into the
  message app at composition time; a missing origin fails closed for account
  creation.
- Required `Origin` to exactly equal that configured origin for
  `POST /api/accounts`, and rejected `Sec-Fetch-Site: cross-site`.
- Performed those checks before `currentRegistrationAttempt` atomically spends
  the pre-session, allowing a normal same-origin retry after a rejected foreign
  browser request.
- Added a direct-address resolver and fixed-window limiter defaults for the
  single-process example. Requests must pass both
  `registration:address:<address>` and
  `registration:login:<base64url(sha256(canonical-login))>` keys.
- Returned a generic JSON body with `429` and `Retry-After: 60` when either
  limiter rejects a request.
- Added the exported `idpaccounts.NormalizeLogin`, and changed both account
  creation and password authentication to use it.
- Added tests for foreign Origin, cross-site Fetch Metadata, pre-session
  preservation after those rejections, address limits, and equivalent
  `"Alice"`/`" alice "` login keys.

### Why

The CSRF token protects a cookie-authenticated state transition, but it does
not bound attempts made by a legitimate same-origin client or a script with its
own registration pre-session. Origin and Fetch Metadata cheaply reject a class
of cross-site requests before durable state changes; address and canonical-login
keys limit two independent attack dimensions. The OWASP CSRF and authentication
references stored in this ticket support this layered approach.

### What worked

```text
go test ./examples/tinyidp-message-app ./pkg/idpaccounts -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 1.745s
ok   github.com/manuel/tinyidp/pkg/idpaccounts 0.245s
```

The foreign-Origin and `Sec-Fetch-Site: cross-site` cases return `403`; the
same cookie and CSRF token can subsequently produce a successful account,
demonstrating that rejection occurs before atomic consumption. The limiter test
shows that canonical login spelling shares a key even when its client address
changes, and that a changed login still shares the address key.

### What didn't work

N/A. The first focused implementation and package test run passed.

### What I learned

- A relying party cannot safely derive a login-based security key using an
  approximation if the identity service owns Unicode/case normalization. A
  narrow public normalization API prevents policy drift without exposing
  storage internals.
- `Origin` validation must use the configured external origin, not a Host
  header reconstructed from the incoming request, because a Host header is not
  the application's deployment configuration.
- The ordering is security-relevant: cheap cross-site checks precede durable
  token consumption, whereas a same-origin malformed request intentionally
  still consumes its token before expensive password work.

### What was tricky to build

There are two rate keys but one `RateLimiter.Allow` interface. The handler
must require both independently: a login-only key lets an attacker distribute
attempts over addresses, while an address-only key lets them distribute attempts
over usernames. The implementation first resolves the address through the
public resolver, canonicalizes the login through `idpaccounts.NormalizeLogin`,
hashes that non-secret canonical value, then requires both keys to admit the
operation. It uses no raw password or raw username in the limiter key.

### What warrants a second pair of eyes

- The in-process fixed-window limiter is appropriate only for this
  single-process example. The Phase 8 production composition must make the
  limiter and trusted-proxy resolver explicit configuration and require a
  shared production implementation when horizontally scaled.
- Review the deliberate Origin requirement for non-browser API clients. This
  example is a browser-oriented application; a separately authenticated native
  registration API would need a different anti-CSRF model rather than weakening
  this check.
- Consider a bounded cleanup strategy for the in-memory limiter's bucket map
  as part of long-running operational work.

### What should be done in the future

- Add fixed registration audit reason codes and an end-to-end test that
  registers through HTTP and subsequently signs in through the provider form.
- Make endpoint-specific rate limits and trusted-proxy policy explicit in the
  future Glazed `serve` command rather than relying on example defaults.
- Add message-API protections using the same exact-origin helper but the
  authenticated app-session CSRF secret.

### Code review instructions

- Start at `handleCreateAccount` in
  `examples/tinyidp-message-app/app_http.go`; verify the Origin/Fetch Metadata
  check is above token consumption and limiter use is below strict decoding.
- Review `isSameOriginRegistrationRequest` and `allowRegistration`; check that
  keys contain only resolved address and a hash, never raw login or password.
- Review `idpaccounts.NormalizeLogin` and its use in both `Create` and
  `AuthenticatePassword`.
- Validate with:

  ```text
  go test ./examples/tinyidp-message-app ./pkg/idpaccounts -count=1
  ```

### Technical details

```text
POST /api/accounts
  exact Origin? --- no ---> 403; registration attempt remains pending
  Fetch Metadata cross-site? -> 403; registration attempt remains pending
  one-time cookie + CSRF? --- no ---> 403
  strict JSON? ------------ no ---> generic 422; attempt is terminal
  address key admitted? --- no ---> generic 429 + Retry-After: 60
  canonical-login key? ---- no ---> generic 429 + Retry-After: 60
  account service create ------> normal generic result mapping
```

The canonical login is SHA-256 hashed and base64url encoded only as a limiter
key component. It is neither logged by this handler nor returned to the caller.

## Step 21: Close Phase 5 with auditable registration and normal login

Phase 5 is now complete. Registration records fixed, non-secret audit outcomes
and the strongest browser test begins with no account at all: it obtains a
registration pre-session, creates Alice through `POST /api/accounts`, then
uses the provider-owned authorization form to authenticate and obtains the
application session through the normal OIDC callback. This proves registration
is durable and usable without collapsing it into auto-login.

### Prompt Context

**User prompt (verbatim):** (same as Step 17)

**Assistant interpretation:** Finish the remaining registration exit criteria, particularly operator-visible outcomes and a genuine post-registration authentication path.

**Inferred user intent:** Establish that the example is a complete relying-party flow, not a collection of independently passing storage and handler fragments.

**Commit (code):** `9363945` — "test(msgapp): prove registration then browser login"

### What I did

- Added `account.self_registration` audit events with stable result/reason
  pairs: `origin_rejected`, `csrf_rejected`, `invalid_request`, `rate_limited`,
  `account_rejected`, and `unavailable`; successful events carry only the new
  subject.
- Kept event fields empty so the handler never records raw login, password,
  CSRF token, cookie, or request body.
- Added MemorySink assertions for rejected and accepted events.
- Replaced the browser test's direct account-service seeding with real GET
  registration and POST account-creation calls through the mounted app.
- Continued the same cookie-jar browser through OIDC authorize, IdP form
  credentials, callback, app-session lookup, and callback replay rejection.

### Why

Fixed reason codes let operations count and alert on security-relevant outcomes
without parsing unstable prose or leaking request values. More importantly, an
account-service unit test is not proof that the public registration protocol,
cookie jar, exact Origin rule, and provider authentication agree. The complete
browser test binds those contracts together.

### What worked

```text
go test ./examples/tinyidp-message-app -run 'Test(CreateAccountRequires|BrowserLoginCompletes)' -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.896s

go test ./examples/tinyidp-message-app -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 1.683s
```

The browser begins unauthenticated, registers Alice, signs in with the exact
password it registered, obtains an independent application session, and cannot
reuse the authorization callback state.

### What didn't work

The first focused build omitted the standard-library `context` import required
by the audit helper:

```text
examples/tinyidp-message-app/app_http.go:193:45: undefined: context
```

After adding the import, the browser test initially submitted the old seeded
password and correctly received the provider's `401 Invalid login or password`.
I aligned the test's IdP-form value with the password used by the HTTP
registration step; the focused and complete suites then passed.

### What I learned

- An IdP can only prove the integration when its login form sees the credential
  produced by the app's public registration path, not one seeded around it.
- Security audit records should contain a fixed outcome and subject when
  appropriate, but no raw protocol or credential material.
- The example currently uses `idp.NoopSink` by default; Phase 8 composition
  must inject a durable audit sink for production rather than treating the
  default as operationally sufficient.

### What was tricky to build

The test needs browser cookies for registration and then provider cookies for
login, while its OIDC back-channel remains an in-process transport. A single
cookie-jar client handles only the public HTTP calls; discovery, token exchange,
and JWKS verification still use the configured in-process client. This mirrors
the intended deployment topology and avoids falsely using the browser client
for a private back-channel.

### What warrants a second pair of eyes

- Review the production audit-delivery policy. `recordRegistration` currently
  records best-effort through the injected sink; a later production composition
  must decide how a durable sink failure is surfaced after an identity mutation
  has already committed.
- Review audit event naming with the broader tiny-idp audit vocabulary before
  dashboards or retention rules depend on it.

### What should be done in the future

- Begin Phase 6: public cursor-paginated feed reads, then authenticated
  session/CSRF-protected message creation with subject derived only from the
  app session.
- Add message audit events and apply exact-origin checks to its unsafe route.
- Configure durable audit, a production rate limiter, and trusted proxy policy
  in the Phase 8 executable composition.

### Code review instructions

- Review `recordRegistration` and every early return in `handleCreateAccount`.
- Read `TestBrowserLoginCompletesAgainstEmbeddedProvider` as an executable
  sequence from registration through callback replay; confirm no test-side
  `accounts.Create` remains.
- Validate with `go test ./examples/tinyidp-message-app -count=1`.

### Technical details

```text
anonymous browser
  -> GET /api/registration
  -> POST /api/accounts (Origin + pre-session + CSRF)
  -> account.self_registration accepted(subject)
  -> GET /auth/login
  -> provider-owned /idp/authorize form
  -> OIDC callback
  -> independent tinymsg_app_session
```

The audit event never includes the registration cookie, CSRF token, password,
or the login string; the subject appears only after identity creation succeeds.

## Step 22: Publish the cursor-paginated message feed

Phase 6 now exposes `GET /api/messages`. The public response contains only the
message ID, display name, plain-text body, and creation time; it intentionally
does not disclose the internal author subject. Pagination continues the store's
stable `(created_at DESC, id DESC)` order with an opaque base64url cursor.

### Prompt Context

**User prompt (verbatim):** "go ahead"

**Assistant interpretation:** Continue into the next implementation phase with another independently reviewable security boundary.

**Inferred user intent:** Turn the persisted message repository into a usable public application API without exposing identity internals or introducing unstable feed paging.

**Commit (code):** `423e6f6` — "feat(msgapp): add public message feed"

### What I did

- Added strict encode/decode helpers for a cursor containing the persisted timestamp and ID.
- Added `GET /api/messages?before=&limit=` with limit default 20 and bounds 1–100.
- Returned a typed JSON response with a continuation cursor when a full page is returned.
- Added codec, HTTP pagination, ordering, and subject-redaction tests.

### Why

Offset pagination can duplicate or skip rows as new messages arrive. The
repository already uses a keyset predicate over a total order, so the HTTP API
preserves that property rather than adding a competing pagination scheme.

### What worked

```text
go test ./examples/tinyidp-message-app -run 'TestMessage' -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.025s
```

### What didn't work

- N/A.

### What I learned

- A public presentation API should not inherit every persistence field; author
  subjects are server-side authorization material and are deliberately omitted.

### What was tricky to build

The cursor must preserve the exact timestamp text used by the SQLite keyset
comparison. The codec therefore uses `formatAppTime` and `parseAppTime`, not a
second ad-hoc time representation.

### What warrants a second pair of eyes

- A full page receives a continuation cursor even if it happens to be the last
  page; following it safely returns an empty page. A future `hasMore` query can
  refine that UX without weakening cursor correctness.

### What should be done in the future

- Add authenticated message creation with exact Origin, session CSRF, and
  server-derived author identity.

### Code review instructions

- Review `decodeMessageCursor`, then `handleListMessages`, then
  `TestMessageFeedUsesCursorAndDoesNotExposeSubject`.
- Validate with `go test ./examples/tinyidp-message-app -run 'TestMessage' -count=1`.

### Technical details

```text
GET /api/messages?limit=20
  -> ordered rows (created_at DESC, id DESC)
  -> nextCursor = base64url(RFC3339Nano timestamp | ID)
GET /api/messages?before=nextCursor
  -> WHERE (created_at < ts OR created_at = ts AND id < id)
```

## Step 23: Create messages from verified application sessions

Phase 6 now has its state-changing half. `POST /api/messages` requires the
configured exact Origin, rejects cross-site Fetch Metadata, loads the opaque
app-session cookie, compares its independent CSRF secret in constant time, and
accepts only a strict JSON object containing `body`. The request cannot supply
an author subject or display name.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Continue implementation with the next security-sensitive application route.

**Inferred user intent:** Make the message application genuinely usable while preserving identity ownership and anti-forgery invariants.

**Commit (code):** `d47efe7` — "feat(msgapp): add authenticated message creation"

### What I did

- Added `POST /api/messages`, strict bounded JSON decoding, and a typed response.
- Reused the exact-origin/cross-site rejection boundary before session lookup.
- Required the opaque app session plus `X-CSRF-Token` matching that session.
- Passed only session subject and display name to the repository.
- Added tests for missing CSRF, foreign Origin, unknown spoofed author fields,
  response subject redaction, and persisted verified authorship.

### Why

The browser must never be able to choose the identity attached to a message.
That value is authorization data, not presentation input; it comes only from
the post-OIDC app session created by the callback.

### What worked

```text
go test ./examples/tinyidp-message-app -run 'Test(CreateMessage|Message)' -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 0.035s

go test ./examples/tinyidp-message-app -count=1
ok   github.com/manuel/tinyidp/examples/tinyidp-message-app 1.116s
```

### What didn't work

- N/A.

### What I learned

- Reusing the same-origin helper is appropriate only because both registration
  and message mutation are browser-cookie APIs; their CSRF secrets remain
  separate and purpose-specific.

### What was tricky to build

The test must distinguish a rejected spoofed `authorSubject` field from a
successful request that silently ignores it. `DisallowUnknownFields` makes the
wire contract fail closed, and the database assertion independently proves the
only stored author is the verified session principal.

### What warrants a second pair of eyes

- The handler maps repository validation and decoding failures to the same
  public 422 response. Review desired UI error detail before frontend work,
  without returning subject or CSRF information.

### What should be done in the future

- Begin Phase 7 frontend work; its composer must send only `{body}` and read
  the CSRF token from `/api/session`.

### Code review instructions

- Review `handleCreateMessage`, then `decodeCreateMessageRequest`, then
  `TestCreateMessageUsesVerifiedSessionAuthorAndCSRF`.
- Validate with `go test ./examples/tinyidp-message-app -count=1`.

### Technical details

```text
POST /api/messages {"body":"..."}
  Origin + Fetch Metadata -> app-session cookie -> session CSRF
  -> createMessage(subject=session.Subject, name=session.DisplayName)
  -> 201 message response without subject
```
