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

## Continuation point

The research and design turn is complete. Implementation should resume at Phase 0 in the primary guide, resolve the open API decisions, and then begin Phase 1 with `pkg/idpaccounts`. Every implementation task should add a new diary step with commit hashes, exact test commands, failures, and review instructions.
