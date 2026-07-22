---
Title: Implementation diary
Ticket: TINYIDP-EMAIL-SIGNUP-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-shared-two-apps/compose.yaml
      Note: Private outbox topology implemented in commit ef597a1
    - Path: repo://examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py
      Note: Restart retry membership and redaction evidence implemented in commit ef597a1
    - Path: repo://internal/cmds/serve_production.go
      Note: Fail-closed production activation implemented in commit b79f77d
    - Path: repo://pkg/idpemailchallenge/smtpmailer/smtpmailer.go
      Note: Native SMTP adapter implemented in commit 9a5605f
    - Path: repo://pkg/idpsignup/verified_invite_signup.js
      Note: Combined two-client verified signup policy implemented in commit b79f77d
ExternalSources: []
Summary: Chronological decisions and evidence for verified-email signup with a private first-deploy delivery outbox.
LastUpdated: 2026-07-21T16:09:43.745179461-04:00
WhatFor: Record why fake delivery is allowed only as a private transport substitution and how the design reuses TinyIDP's durable native challenge workflow.
WhenToUse: Read before implementing or resuming TINYIDP-EMAIL-SIGNUP-001.
---


# Implementation diary

## Goal

Activate durable, truthful email-verified signup for both applications while using a private operator outbox for the first deployment and deferring real SMTP credentials and cluster delivery work.

## Step 1: Separate fake delivery from fake verification

I created a separate ticket rather than reopening the completed invitation-core ticket. I inspected TinyIDP's challenge service, compiled email-verified signup program, SQLite persistence tests, browser continuation handler, account commit, and production validation.

### Prompt Context

**User prompt (verbatim):** "create a separate ticket + design doc for the email signup, for now, fake it for the first deploy."

**Assistant interpretation:** Design email-verified signup as an independent workstream, retain the real durable verification core, and use a temporary private delivery sink for the first deployment instead of integrating the personal SMTP server now.

**Inferred user intent:** Unblock the first small deployment without creating a dishonest `email_verified` claim or a disposable alternate signup implementation.

### What I did

- Created `TINYIDP-EMAIL-SIGNUP-001` with a design document, implementation diary, detailed tasks, and changelog.
- Located the existing typed `Mailer` interface and durable challenge service.
- Confirmed the SQLite store already owns challenge attempts, resends, expiry, evidence, and cleanup.
- Confirmed explicit continuations preserve pending and verified challenge references across browser waits and process restarts.
- Confirmed the native signup commit sets `EmailVerified` only when verified evidence is present and matches the account login.
- Confirmed `serve-production` intentionally rejects challenge outcomes because it constructs no mailer or challenge service.
- Selected a private operator SMTP catcher as the first-deploy transport substitution.
- Rejected displaying the code in the public browser or automatically marking first-deploy accounts verified.

### Why

- The invitation core and email-delivery activation have different scope, secrets, operational dependencies, and acceptance criteria.
- Reusing the real challenge path means later SMTP activation changes deployment binding rather than application semantics.
- A publicly visible code cannot prove email possession and must not produce a verified claim.

### What worked

- The relevant packages and restart tests make the boundary unusually clear: durable verification is complete, while production delivery construction is missing.
- The existing `Mailer` request is already narrow enough for a fixed-template SMTP adapter.

### What didn't work

- No code or experiment failed during this design step.

### What I learned

- The production blocker is smaller than a general email subsystem: one delivery adapter, one key/constructor path, one validator change, and one combined signup program.
- `CreateAndSend` persists before SMTP delivery, so delivery failure recovery must be tested through the existing bounded resend path.

### What was tricky to build

- The term “fake” could describe either transport or verification. Only fake transport is acceptable if TinyIDP continues to emit `email_verified=true`.
- The operator outbox contains live bearer codes and therefore needs private access and short retention even though it is temporary.

### What warrants a second pair of eyes

- Review whether operator-assisted code relay provides an acceptable first-deploy identity assurance statement for the actual invited user population.
- Review the proposed SMTP TLS modes and ensure plaintext is accepted only on an isolated private workload network.

### What should be done in the future

- Implement the phases in `tasks.md`.
- Replace the private mail catcher with the personal SMTP server through configuration after the separate deployment work is authorized.

### Code review instructions

- Begin with `pkg/idpemailchallenge/service.go` and `internal/fositeadapter/scripted_signup.go`.
- Compare `pkg/idpsignup/email_verified_signup.js` with the current two-application `open-signup.js`.
- Confirm `internal/cmds/serve_production.go` remains fail-closed until the native service is fully constructed.

### Technical details

```text
temporary: TinyIDP -> SMTP -> private operator outbox -> manual relay -> user
later:     TinyIDP -> SMTP -> personal mail server -> recipient mailbox

unchanged in both modes:
code generation -> keyed hash -> durable bindings -> attempts/resends ->
native verification -> evidence reference -> atomic verified account commit
```

## Step 2: Implement the narrow SMTP challenge mailer

I added a concrete `idpemailchallenge.Mailer` implementation without changing the existing challenge-service interface. The adapter accepts only a native template catalog and the typed `MailRequest`; it cannot be used by JavaScript to choose arbitrary headers, bodies, senders, or recipient lists.

The implementation supports verified STARTTLS and implicit TLS for the eventual personal mail server. It also supports an explicitly named unauthenticated `private-plaintext` mode for the isolated first-deploy mail catcher. Connection and complete SMTP exchange deadlines are mandatory.

### Prompt Context

**User prompt (verbatim):** "implement."

**Assistant interpretation:** Implement every phase of `TINYIDP-EMAIL-SIGNUP-001`, not only the documentation or a mock demonstration.

**Inferred user intent:** Make the first deployment's verified signup behavior executable and testable while retaining the later path to real SMTP.

### What I did

- Added `pkg/idpemailchallenge/smtpmailer`.
- Added the fixed `signup-code` subject and plaintext body renderer.
- Added STARTTLS, implicit TLS, and private-plaintext modes.
- Required positive connect/send timeouts and exact host/port configuration.
- Required SMTP username/password together and rejected authentication over plaintext.
- Parsed exactly one sender and recipient mailbox and rejected line-break injection.
- Added SMTP/network retry classification through the existing `MailFailure` interface.
- Added an in-process SMTP test server and focused delivery, validation, redaction, and transient-failure tests.

### Why

- The existing challenge core deliberately delegates only delivery. A focused adapter preserves that boundary.
- The first-deploy catcher and later personal server should use the same protocol and mailer code.
- Fixed templates prevent signup JavaScript from becoming an arbitrary mail-sending API.

### What worked

- `go test ./pkg/idpemailchallenge/smtpmailer -count=1` passes.
- The in-process SMTP server captured the complete fixed message and verified recipient, code, and expiry rendering.
- Invalid recipients and templates return permanent classified failures without containing the raw code.
- A simulated SMTP `451` response returns a transient classified failure.

### What didn't work

- The first focused run failed because the test expected `From: TinyIDP <accounts@example.test>`, while `net/mail.Address.String` correctly rendered `From: "TinyIDP" <accounts@example.test>`. The exact failure was `message does not contain "From: TinyIDP <accounts@example.test>"`. I updated the test to assert the standards-compliant quoted form; no production code change was needed.

### What I learned

- Go's standard SMTP client can operate over a context-dialed connection and explicit deadline, including implicit TLS through `smtp.NewClient` on a wrapped connection.
- The repository's existing `MailFailure` interface already provides the right closed retry classification seam.

### What was tricky to build

- Context cancellation after dialing must interrupt SMTP reads and writes. Setting the connection deadline to the earlier of the context deadline and configured send timeout gives every protocol operation a bound.
- Plain authentication must never be accepted in the private plaintext mode even if the mail catcher happens to advertise AUTH.

### What warrants a second pair of eyes

- Review SMTP response classification and whether permanent classification is appropriate for every non-network, non-4xx error.
- Review message rendering for the personal server's expected envelope and header requirements.

### What should be done in the future

- Wire the adapter into `serve-production` using file-backed secrets.
- Add end-to-end TLS tests if the production server exhibits interoperability differences.

### Code review instructions

- Start at `smtpmailer.New`, then read `SendEmailChallenge` and `send`.
- Run `go test ./pkg/idpemailchallenge/smtpmailer -count=1`.

### Technical details

```text
TLS modes:
  starttls          TCP -> EHLO -> STARTTLS -> optional AUTH -> message
  implicit          TLS handshake -> SMTP -> optional AUTH -> message
  private-plaintext isolated TCP -> SMTP -> message; AUTH forbidden
```

## Step 3: Activate challenge services and compose the two-client policy

I extended `serve-production` with conditional, file-backed email delivery settings and a dedicated challenge HMAC key. Production now compiles the selected signup program, detects whether it declares a challenge outcome, constructs the durable email service only when required, and still refuses to listen when any dependency is absent or partially configured.

I also added a reviewed combined JavaScript program. Message Desk starts with display name and email. goja adds its signup invitation field. Both paths inspect applicable admission policy before creating the same durable email challenge, then collect passwords only after native verified evidence exists.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Continue from the delivery adapter into the production command and executable signup policy.

**Inferred user intent:** Make the existing durable email workflow usable by the real production host without weakening invitation or verification boundaries.

**Commit (code):** `b79f77d` — "feat: activate verified email signup in production"

### What I did

- Added Glazed fields for challenge key, SMTP endpoint/TLS/server name, optional username/password file, fixed sender, and timeouts.
- Generalized owner-only file reading so the 32-byte challenge key and arbitrary-length SMTP password retain `0400`/`0600` enforcement.
- Added conditional program requirement detection and rejected both missing required configuration and irrelevant partial mail configuration.
- Constructed `idpemailchallenge.Service` from the production SQLite store, SMTP mailer, and dedicated key.
- Passed the service through `embeddedidp.ScriptedSignupConfig`.
- Changed production validation to accept `challenge` outcomes only when the native service is available.
- Added `VerifiedInviteSource` and synchronized the deployable shared application program.
- Added command, construction, schema, and workflow tests.

### Why

- Program declarations and native service availability must agree before the listener starts.
- The challenge key has a distinct rotation and compromise domain from OIDC tokens and signup invitations.
- Invitation inspection must occur before sending mail so an invalid goja invite cannot use TinyIDP as a mail source.

### What worked

- `go test ./pkg/idpsignup ./internal/cmds ./pkg/idpemailchallenge/smtpmailer -count=1` passed.
- Production accepts the combined program only with `EmailChallenges: true` and preserves the previous fail-closed rejection otherwise.
- Command schema tests confirm every mail flag exists but remains conditionally required.
- The repository pre-commit lint and test hook passed for commit `b79f77d`.

### What didn't work

- The first commit invocation started the pre-commit hook but the terminal wrapper detached before returning final output. It did not create a commit. I verified the staged index was intact and reran `git commit -m 'feat: activate verified email signup in production'`. The second wrapper also detached while the parallel test process finished, but commit `b79f77d` was created and subsequent process inspection confirmed no hook remained. The focused suites had already passed independently.

### What I learned

- Program compilation must precede conditional service construction because the reviewed program itself declares whether the challenge dependency is required.
- Glazed defaults such as sender display name and timeout must not be mistaken for an operator's partial attempt to configure email delivery.

### What was tricky to build

- Validation needs two states: challenge outcomes are structurally valid, but they are operationally valid only when the service exists. Passing an explicit `productionSignupServices` value keeps that decision reviewable.
- The invitation provider is invoked by the native adapter when the active form contains `invite_code`. That ordering validates the invite before the submitted lambda returns the email challenge.

### What warrants a second pair of eyes

- Review clearing and lifetime of the SMTP credential copy held by the long-lived mailer.
- Review whether future templates need locale selection without expanding JavaScript into arbitrary message content.

### What should be done in the future

- Bind the same fields to the personal SMTP server only after the separate deployment work is authorized.

### Code review instructions

- Read `newProductionEmailChallenges`, `validateProductionSignupProgram`, and `verified_invite_signup.js`.
- Run `go test ./pkg/idpsignup ./internal/cmds -count=1`.

### Technical details

```text
program has no challenge + no mail flags -> service nil, startup allowed
program has no challenge + mail flags    -> startup rejected
program has challenge + missing fields   -> startup rejected
program has challenge + complete fields  -> durable service constructed
```

## Step 4: Deploy the private outbox and prove the browser workflow

I added a pinned Mailpit service to the shared local Compose environment. Its operator UI/API is bound only to host loopback, protected by Basic authentication, absent from Caddy and every public application network, capped at 100 messages, and configured to remove messages after one hour. TinyIDP submits unauthenticated SMTP only through the isolated IDP backend.

The browser acceptance driver now performs the complete multi-page workflow instead of treating signup as one form. It retrieves codes through the authenticated operator API, proves restart and wrong-code retry behavior, and verifies that the new goja identity can accept the email-bound application invitation that originally motivated this ticket.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Complete the fake first-deploy delivery topology and prove the full product behavior locally.

**Inferred user intent:** Have an immediately testable first deployment path whose only deferred element is delivery through the personal SMTP server.

**Commit (code):** `ef597a1` — "feat: verify signup through private outbox"

### What I did

- Pinned `axllent/mailpit:v1.30.5`, selected after checking the official current release and API documentation.
- Bound the operator service to `127.0.0.1:8025`, enabled Basic authentication, and kept it out of Caddy/public ingress.
- Added one-hour retention, a 100-message cap, and strict RFC header parsing.
- Added a second random local secret for email challenge HMAC state.
- Configured TinyIDP for private-plaintext SMTP only on `idp-backend`.
- Extended the standard-library browser driver to query `/view/latest.txt` with recipient filtering and Basic authentication.
- Split signup into identity, email-code, password, consent, and callback phases.
- Restarted the TinyIDP container after Message Desk challenge delivery and completed the same continuation afterward.
- Submitted an incorrect goja code, asserted a stable denial, then submitted the real code successfully.
- Accepted the application invitation as the newly verified user and asserted exactly one membership.
- Asserted invalid signup-invite replay generated no outbox message.
- Asserted raw signup invitation and both raw email codes were absent from audit and service logs.

### Why

- The acceptance suite must exercise the same durable boundaries and HTTP forms that a human browser uses.
- Restart and wrong-code retry are observable properties of professional challenge state, not implementation details.
- The new user's successful application membership proves that `email_verified=true` propagates through OIDC normalization and satisfies native application identity binding.

### What worked

- The image built as `sha256:81de2d4cbc06...`; TinyIDP and Mailpit both reported healthy.
- An unauthenticated Mailpit API request returned `401`; the operator credential returned `200`.
- The original seven-stage acceptance passed immediately after enabling verified signup.
- The strengthened run with an actual IDP restart and wrong-code retry also printed `PASS: shared TinyIDP Phase 5 browser acceptance completed` and exited zero.
- Invalid signup-invite replay returned the generic field denial and an outbox lookup for that address returned `404`.

### What didn't work

- No application defect occurred during Compose execution. The TinyIDP image build took approximately 138 seconds because the Go binary layer was rebuilt from the changed command source.

### What I learned

- Mailpit's official API supports authenticated filtered retrieval of the latest text message, which is sufficient for deterministic browser acceptance without parsing its internal database.
- A container restart does not invalidate the browser continuation, challenge binding, or verified path because all relevant state resides in SQLite and the browser retains only opaque handles.

### What was tricky to build

- The acceptance driver must not auto-approve a form containing `email_code`; consent automation now remains distinct from identity, challenge, and password forms.
- Invalid signup invitations must be rejected before `CreateAndSend`; the outbox absence check proves the ordering externally.

### What warrants a second pair of eyes

- Review whether loopback publishing plus Basic authentication meets the operator-access policy on every intended first-deploy host.
- Review Mailpit retention and backup behavior; its captured messages must never enter routine backups.

### What should be done in the future

- Replace Mailpit with the personal SMTP submission endpoint and remove the local operator fixture credential during the separately authorized deployment phase.

### Code review instructions

- Start with the `mailpit` and `idp` services in `compose.yaml`, then read `signup`, `outbox_code`, and `restart_idp_and_wait` in the acceptance driver.
- Run `./scripts/00-init-secrets.sh`, `docker compose up --build -d`, `./scripts/02-smoke.sh`, and `./scripts/03-browser-acceptance.py`.

### Technical details

```text
public HTTPS: message.localhost, goja.localhost, idp.localhost
operator only: http://127.0.0.1:8025 with Basic authentication
private SMTP:  idp -> mailpit:1025 on idp-backend
```
