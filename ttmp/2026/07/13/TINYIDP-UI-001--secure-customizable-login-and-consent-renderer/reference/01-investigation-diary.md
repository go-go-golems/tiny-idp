---
Title: Investigation Diary
Ticket: TINYIDP-UI-001
Status: active
Topics:
    - oidc
    - identity
    - security
    - go
    - architecture
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/interaction_hardening_test.go
      Note: Existing regression evidence for forced login consent replay expiry and opaque form state
    - Path: repo://internal/fositeadapter/provider.go
      Note: Primary source traced during current-state investigation
    - Path: repo://internal/server/static/login.html
      Note: Synthetic server template inspected to rule out the wrong customization path
ExternalSources:
    - https://pkg.go.dev/html/template
    - https://www.w3.org/TR/CSP/
    - https://openid.net/specs/openid-connect-core-1_0-18.html
    - https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
    - https://www.rfc-editor.org/rfc/rfc9700.html
    - https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html
    - https://www.w3.org/WAI/WCAG22/Understanding/accessible-authentication-minimum.html
Summary: Chronological evidence and decision log for designing a host-supplied, security-preserving tiny-idp login and consent renderer.
LastUpdated: 2026-07-13T17:38:39.058258495-04:00
WhatFor: Records what was inspected, researched, decided, authored, validated, and delivered for TINYIDP-UI-001.
WhenToUse: Read before resuming implementation or reviewing why the renderer API, trust boundary, CSP, asset policy, and assurance phases were chosen.
---


# Investigation Diary

## Goal

This diary records the creation and design investigation for a customizable
tiny-idp interaction renderer. It is intended to let a new engineer reproduce
the evidence, understand failed or misleading paths, and continue with Phase 1
without reconstructing the reasoning from conversation history.

## Step 1: Create the ticket and map the current rendering path

The first step established a clean docmgr workspace and identified the exact
runtime page that must change. The central question was whether the existing
embedded `static/login.html` controlled the xapp login page. Source tracing
showed that it does not: xapp constructs `pkg/embeddedidp`, which constructs the
strict Fosite adapter, whose `renderInteraction` function writes a hard-coded
HTML string.

This distinction prevented the design from targeting the synthetic development
server while leaving the production-oriented path unchanged.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new ticket for the preferred method, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new ticket for the recommended
host-supplied renderer architecture, investigate the real code path, write an
intern-ready design and implementation guide, and deliver the ticket as a
reMarkable bundle.

**Inferred user intent:** Make login UI customization an intentional product API
rather than a source fork, without weakening the security properties already
built into strict authorization interactions.

### What I did

- Ran `docmgr status --summary-only` and inspected the existing ticket list and
  vocabulary.
- Checked `git status --short` before writing and found two unrelated untracked
  OIDC source directories under `TINYIDP-PROD-001`; I left them untouched.
- Created `TINYIDP-UI-001` with topics `oidc,identity,security,go,architecture,auth`.
- Added the primary design document and this diary through `docmgr doc add`.
- Located the strict renderer with:

  ```bash
  rg -n "func \(p \*Provider\) renderInteraction|renderInteraction\(" internal/fositeadapter -g '*.go'
  ```

- Read line-numbered sections of `internal/fositeadapter/provider.go`,
  `interaction.go`, `csrf.go`, `pkg/idpstore/types.go`, the embedded provider,
  xapp composition, the synthetic server template, and interaction hardening
  tests.
- Confirmed the xapp outer mux mounts `/idp/` separately and gives the Goja HTTP
  host the remaining paths, including existing `/static/assets/...` resources.

### Why

- A public API cannot be designed safely without knowing which layer owns
  protocol state, response headers, cookies, and application routing.
- The mock and strict implementations use materially different continuation
  models, so conflating their templates would produce the wrong feature.

### What worked

- `rg` immediately located the single strict render function and its call site.
- Line-numbered inspection made the evidence directly usable in the design doc.
- Existing hardening tests exposed the important invariant that OAuth protocol
  continuation fields never return to the browser form.
- The xapp mux confirmed that a host-owned stylesheet can use the existing
  `/static/` boundary.

### What didn't work

- One parallel evidence command produced 1,130 output lines and was truncated by
  the tool. I reran narrower `nl -ba ... | sed -n ...` ranges for the authorize
  resume path, interaction test helpers, embedded construction, and xapp mounts.
  No evidence was inferred from the truncated portion.

### What I learned

- The strict page is generated at `internal/fositeadapter/provider.go:941-959`.
- The synthetic `internal/server/static/login.html` is unrelated to xapp's
  embedded strict provider.
- The provider already has a strong server-owned continuation: the browser sees
  only an opaque interaction handle and interaction-bound CSRF value.
- `pkg/embeddedidp.Options` is the correct public injection point, but a renderer
  contract cannot live there if the internal adapter must import it; that would
  create a cycle.
- The global CSP currently blocks every stylesheet because `default-src 'none'`
  has no `style-src` exception.

### What was tricky to build

The renderer looks like a frontend feature, but the current function sits inside
an OAuth state machine. The tricky part was separating presentation authority
from authorization authority. The page must carry a handle, CSRF token,
credentials, and an approve/deny action, yet none of those rendered controls can
become authoritative. The existing resume path already enforces that separation,
so the new API must expose less than an HTTP handler, not more.

### What warrants a second pair of eyes

- Confirm that the proposed `pkg/idpui` package is the right stable public
  boundary and does not duplicate a suitable existing package.
- Review the current combined login/consent behavior, especially when consent
  policy depends on the user who has not authenticated yet.
- Confirm that every success redirect after a credential POST has safe method
  rewriting semantics.

### What should be done in the future

- Obtain explicit design approval before adding the public package.
- Keep interaction and CSRF invariants covered as structural renderer tests are
  introduced.

### Code review instructions

- Start at `internal/fositeadapter/provider.go:407` and trace GET then POST
  authorization handling.
- Read `internal/fositeadapter/interaction.go` before reviewing any page model.
- Run:

  ```bash
  go test ./internal/fositeadapter -run 'TestInteractionFormContainsNoProtocolContinuation|TestForcedPromptLoginCannotReuseExistingSession' -count=1
  ```

### Technical details

The core construction path observed was:

```text
cmd/tinyidp-xapp -> pkg/embeddedidp -> internal/fositeadapter -> net/http
```

The relevant current response policy is:

```text
default-src 'none'; frame-ancestors 'none';
form-action 'self' https:; base-uri 'none'
```

## Step 2: Preserve research and derive the security policy

This step tied the proposed API to primary language, web-security, OAuth/OIDC,
and accessibility material. The research was not used as a general bibliography;
each source answered a specific design question about escaping, CSS loading,
CSRF token placement, forced login, authorization UI, redirects, or form errors.

The Defuddle captures were stored locally so later implementation and review can
retrace the decision even if a web page moves.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Research enough authoritative material to justify
the preferred method and preserve that evidence with the ticket.

**Inferred user intent:** Make the design professionally reviewable and teach an
intern both the implementation and the security theory behind it.

### What I did

- Searched authoritative Go, W3C, OpenID Foundation, RFC Editor, OWASP, and WCAG
  sources.
- Saved seven full Markdown captures with commands of this form:

  ```bash
  defuddle parse https://pkg.go.dev/html/template --md \
    -o ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/sources/go-html-template-security-model.md
  ```

- Preserved Go `html/template`, CSP Level 3, OpenID Connect Core, OWASP CSRF,
  RFC 9700, WCAG error identification, and WCAG accessible authentication.
- Checked capture sizes with:

  ```bash
  wc -lc ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/sources/*.md
  ```

- Mapped each source to a concrete design consequence in the main document.

### Why

- Go's escaping model determines the template/data trust boundary.
- CSP determines whether custom CSS will load and which new resource channels
  become available.
- OIDC `prompt` and `max_age` semantics constrain the view model.
- OAuth security guidance makes anti-framing and credential-POST redirect
  behavior release concerns.
- WCAG requirements turn error rerendering and field labels into correctness
  work, not optional styling.

### What worked

- All seven Defuddle extractions completed successfully.
- The captures totaled 10,372 lines and approximately 739 KiB, so they contain
  substantial local context rather than link stubs.
- The Go documentation directly supported a trusted-template/untrusted-data
  contract.
- RFC 9700 exposed a redirect-status question that was not obvious from the
  initial CSS customization request.

### What didn't work

- N/A. Every selected capture completed successfully and had plausible content
  size and line count.

### What I learned

- `html/template` assumes template authors are trusted and execution data is
  untrusted; that matches a host-compiled renderer exactly.
- Trusted-content wrapper types such as `template.HTML` can bypass contextual
  escaping and should be excluded from the public page model.
- Same-origin external CSS requires `style-src 'self'`; the current CSP will
  block it.
- CSRF values belong in POST form fields and must not appear in URLs or logs.
- `prompt=login` and `max_age` require the UI model to preserve why fresh login
  was requested.
- RFC 9700 recommends 303 after credential POSTs to prevent POST-body forwarding
  risks associated with 307.

### What was tricky to build

CSS policy is not isolated from host architecture. `style-src 'self'` trusts all
CSS on the identity origin, not one file. The design therefore pairs CSP with
embedded immutable assets and a reviewed `/static/` route. A raw configurable
CSP string would make the public API easier to use but would destroy the central
security invariant.

### What warrants a second pair of eyes

- Review whether v1 should permit `img-src 'self' data:` and `font-src 'self'` or
  retain `'none'` until an approved theme needs them.
- Review same-origin CSS assumptions for hosts that allow user uploads.
- Decide whether the redirect-status improvement belongs in this ticket or a
  focused protocol-hardening ticket.

### What should be done in the future

- Keep the source packet linked from any implementation report.
- Update captures only when a material standard revision changes a decision.

### Code review instructions

- Begin with the source-to-decision mapping in Section 4 of the design guide.
- Compare the proposed CSP with `Provider.securityHeaders` and the xapp static
  asset routes.

### Technical details

Captured sources are in `sources/` and are linked individually from Section 20
of the design guide.

## Step 3: Author the design and durable implementation plan

This step converted source evidence into a public API proposal and a work plan.
The main document is intentionally textbook-like: it defines terms, traces the
current state machine, shows authority and dependency diagrams, sketches the Go
API, records decisions and alternatives, and then decomposes implementation and
assurance work into phases.

No runtime code was changed. The ticket remains a proposed design awaiting
review before Phase 1 begins.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Produce a comprehensive design that a new intern
can use to understand and implement the feature task by task.

**Inferred user intent:** Preserve enough context and precise tasks that work can
continue over multiple sessions without architectural drift.

### What I did

- Wrote a design guide of approximately 1,560 lines with 21 major sections.
- Proposed `pkg/idpui` to avoid an import cycle.
- Proposed a writer-based renderer rather than an HTTP handler.
- Defined typed page, form, login, consent, action, scope, and public-error
  models.
- Designed bounded rendering before response commit.
- Designed host-owned `/static/` CSS with provider-owned CSP.
- Added seven explicit decision records.
- Added Phase 0 through Phase 6 with task-level implementation guidance and exit
  criteria.
- Added 47 docmgr tasks and checked the four completed evidence/design tasks.

### Why

- A stable public API requires decisions about dependency direction, trust,
  failure behavior, and testability before implementation.
- Detailed tasks make the ticket usable for long-running work and intern
  handoff.

### What worked

- The writer-based boundary removed HTTP header, cookie, status, redirect, and
  streaming authority while preserving normal Go template execution.
- A standalone `pkg/idpui` resolved the otherwise unavoidable import cycle.
- Existing interaction tests provided a strong base for recording-renderer and
  state-matrix tests.
- The ticket task list now separates public API, provider integration, state
  semantics, xapp styling, assurance tooling, and release work.

### What didn't work

- The generated design document initially retained its empty scaffold sections
  after the completed report. I detected them with `tail`, removed them with a
  focused patch, and rechecked the beginning and end of the document.

### What I learned

- A renderer API should not live in `pkg/embeddedidp` because the adapter it
  constructs also needs the interface.
- Recoverable invalid-credential responses are part of the renderer design; a
  branded initial form followed by plain `http.Error` is not a complete UI.
- Client presentation metadata is absent from `idpstore.Client`. Adding names or
  logos would be a persistence/admin feature and is deliberately not hidden in
  this ticket.
- Splitting combined login and consent requires a real interaction transition,
  not merely a second template.

### What was tricky to build

The view model has to be useful without making rendered data authoritative. The
design resolves this by providing exact action and field constants for form
construction while requiring the server to validate every submitted value
against stored required actions. The browser can always forge form data, so the
HTML contract is a usability and conformance contract, never an authorization
predicate.

### What warrants a second pair of eyes

- Public type names and future extensibility of exported page structs.
- Defensive copying and renderer concurrency requirements.
- The fixed output-size limit and render-failure audit taxonomy.
- Retry status codes and whether the same CSRF token should be reused after a
  validated recoverable failure.
- Combined login/consent semantics.

### What should be done in the future

- Do not begin Phase 1 until Task 0.6 design approval is recorded.
- If review changes the public contract, update decision records and tasks before
  code.

### Code review instructions

- Read the executive summary, current-state Sections 3–6, API Sections 7–13,
  decision records, then the implementation phases.
- Validate ticket structure with:

  ```bash
  docmgr doctor --ticket TINYIDP-UI-001 --stale-after 30
  ```

### Technical details

The preferred contract is:

```go
type InteractionRenderer interface {
    RenderInteraction(context.Context, io.Writer, InteractionPage) error
}
```

The renderer is trusted presentation code. The provider remains the only owner
of headers, cookies, status, CSRF validation, stored interactions, required
actions, authentication, consent, code issuance, and redirects.

## Step 4: Validate and deliver the design bundle

This step completed the ticket hygiene and published the core documentation as
one reMarkable PDF with a table of contents. Validation was run before delivery
so the uploaded document reflects a structurally sound ticket rather than a
working draft.

The preserved standards packet remains local in `sources/`; the reMarkable
bundle contains the authored index, design guide, diary, tasks, and changelog so
the main engineering narrative remains navigable.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the complete ticket and upload the
intern-facing documentation to the requested device.

**Inferred user intent:** Make the design available for offline review and
handoff while retaining reproducible local evidence.

### What I did

- Ran frontmatter validation on the design guide and diary.
- Ran:

  ```bash
  docmgr doctor --ticket TINYIDP-UI-001 --stale-after 30
  ```

- Completed a dry-run bundle upload.
- Uploaded the five core ticket documents as
  `TINYIDP UI 001 Secure Interaction Renderer.pdf`.
- Recorded the destination `/ai/2026/07/13/TINYIDP-UI-001`.

### Why

- `docmgr doctor` detects missing files, metadata errors, vocabulary drift, and
  stale documentation before handoff.
- A bundled PDF preserves document order and provides one table of contents for
  reMarkable review.
- Full source captures remain better suited to repository search than a very
  large device bundle.

### What worked

- Both substantive documents reported `Frontmatter OK`.
- `docmgr doctor` reported one ticket with all checks passed.
- The dry run listed all five intended documents and the expected destination.
- The real upload reported:

  ```text
  OK: uploaded TINYIDP UI 001 Secure Interaction Renderer.pdf -> /ai/2026/07/13/TINYIDP-UI-001
  ```

### What didn't work

- N/A. Validation, PDF rendering, and upload completed without errors.

### What I learned

- The core authored bundle is large enough to carry the complete design while
  remaining separate from the 739 KiB raw source packet.
- Ticket validation is clean before implementation begins.

### What was tricky to build

The delivery bundle needed to balance traceability with readability. Including
all captured standards would add more than ten thousand source lines and obscure
the authored guide. The design links every local capture, so excluding raw
sources from the device bundle does not weaken repository traceability.

### What warrants a second pair of eyes

- Confirm that the reMarkable bundle ordering is comfortable for review: index,
  design, diary, tasks, then changelog.
- Review the seven proposed decisions before checking Task 0.6.

### What should be done in the future

- Begin Phase 1 only after architecture and security approval.
- Record all implementation commands, failures, tests, and commits as new diary
  steps.

### Code review instructions

- Run `docmgr doctor --ticket TINYIDP-UI-001 --stale-after 30` after any
  documentation edit.
- Review Task 0.6 and the open questions before creating `pkg/idpui`.

### Technical details

The delivered PDF contains:

1. `index.md`
2. the primary design guide
3. this investigation diary
4. `tasks.md`
5. `changelog.md`

## Step 5: Receive implementation approval and prepare the baseline

The user explicitly directed implementation to begin task by task, with a
detailed diary and focused commits. This satisfies the ticket's architecture
approval gate for beginning Phase 1. It does not pre-approve every open release
question; those still require evidence and explicit decisions as their tasks are
reached.

Before changing runtime code, I reread the ticket, verified the worktree, and
prepared a documentation-only baseline commit. This keeps the approved design
and source packet independently reviewable from subsequent implementation
checkpoints.

### Prompt Context

**User prompt (verbatim):** "Ok, now implement this task by task, keeping a detailed diaryas you work, committing at appropriate intervals."

**Assistant interpretation:** Treat the design as approved, implement the
recorded phases sequentially, update the diary and docmgr artifacts continuously,
and create focused Git commits at meaningful boundaries.

**Inferred user intent:** Complete the feature professionally while preserving a
reviewable history that a new engineer can resume or audit.

### What I did

- Reread the diary, task list, recent Git history, and worktree status.
- Confirmed the only unrelated worktree entries are the two pre-existing
  untracked source directories under `TINYIDP-PROD-001`.
- Marked Task 0.6 complete based on the explicit implementation instruction.
- Prepared to stage only the `TINYIDP-UI-001` ticket for the baseline commit.

### Why

- A clean design checkpoint separates approved intent from implementation
  details and makes later review bisectable.
- Explicit staging prevents unrelated research directories from entering the
  commit.

### What worked

- The ticket remained clean under `docmgr doctor` before implementation.
- The current branch and recent commits were identified without modifying any
  existing source file.

### What didn't work

- N/A.

### What I learned

- The ticket directory was still untracked because the design-only request did
  not include a commit. It must be committed before Phase 1 code so later diary
  entries can reference a stable baseline.

### What was tricky to build

The approval gate distinguishes permission to start implementation from closure
of technical questions. The user's instruction closes Task 0.6, while CSP image
and font policy, retry status, redirect status, and combined consent semantics
remain decisions to resolve with tests during their recorded phases.

### What warrants a second pair of eyes

- Confirm the baseline commit contains only `TINYIDP-UI-001`.
- Confirm future commits never include the unrelated `TINYIDP-PROD-001` source
  directories.

### What should be done in the future

- Begin Phase 1 with the dependency-light public package and no adapter changes
  in the same code checkpoint.

### Code review instructions

- Review the baseline commit as documentation only.
- Run `git diff --cached --name-only` before committing and verify every path is
  under the new ticket.

### Technical details

The planned checkpoint message is:

```text
Docs: design secure interaction renderer
```
