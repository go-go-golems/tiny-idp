---
Title: Professional invitation core and application membership invitation design and implementation guide
Ticket: TINYIDP-INVITES-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/serve_production.go
      Note: Contains the current production validator that rejects native invitation capabilities
    - Path: repo://internal/fositeadapter/scripted_signup.go
      Note: Owns the atomic native signup commit including optional invitation consumption
    - Path: repo://internal/gojamodules/tinyidp/module.go
      Note: Defines the closed JavaScript signup commit and invitation effect request
    - Path: repo://pkg/idpinvite/durable.go
      Note: Defines HMAC-protected durable invitation issuance, revocation, and transaction-aware redemption
    - Path: ws://go-go-goja/examples/xgoja/21-generated-host-auth/verbs/sites.js
      Note: Shows current org-invite issuance and incomplete acceptance without membership creation
    - Path: ws://go-go-goja/pkg/gojahttp/auth/appauth/appauth.go
      Note: Defines application users, memberships, and the authentication-versus-authorization boundary
    - Path: ws://go-go-goja/pkg/gojahttp/auth/capability/capability.go
      Note: Defines the durable application capability lifecycle and token-secrecy contract
ExternalSources: []
Summary: A pragmatic design for durable TinyIDP account-creation invitations and atomic go-go-goja application-membership invitations, including trust boundaries, JS APIs, browser flows, transactions, tests, and phased implementation.
LastUpdated: 2026-07-21T18:20:00-04:00
WhatFor: Implement invitation-gated signup and organization membership without turning JavaScript into a database layer or coupling TinyIDP to application authorization.
WhenToUse: Use before changing TinyIDP signup scripting, go-go-goja capability acceptance, OIDC registration routing, or the shared two-application deployment.
---








# Professional invitation core and application membership invitation design and implementation guide

## Executive summary

The product needs two invitation mechanisms because two different authorities are making two different decisions.

- **TinyIDP account-creation invitations** decide whether a person may create an identity in the identity provider.
- **Application membership invitations** decide whether an already authenticated identity may become a member of an application organization, and with which role.

These mechanisms may appear as one journey to a user, but they must not become one database record or one authorization decision. TinyIDP owns authentication, credentials, browser sessions, and OIDC subjects. A go-go-goja application owns tenants, memberships, roles, and application resources. Keeping those boundaries intact makes a shared identity provider safe for applications with different admission rules.

The pragmatic conclusion of the code review is that the professional core is mostly present already. TinyIDP has a hashed, expiring, revocable, one-time durable invitation store and a signup commit transaction that can consume the invitation together with identity creation. Its JavaScript program model already declares invitation providers and a `consumeInvitation` effect. The immediate TinyIDP gap is production wiring: `serve-production` currently rejects all native capabilities and rejects the invitation effect. The go-go-goja side also has a durable capability service, but its example acceptance route consumes an organization invite without authenticating a user or creating a membership. That is the application-side gap.

This design therefore does **not** propose a generic database API, raw SQL, a general-purpose key/value store, an arbitrary network client, an invitation campaign engine, or a new administrator web application. It proposes a narrow completion of existing contracts:

1. enable TinyIDP's existing durable invitation provider and atomic consumption path in production;
2. add small operator-facing issue and revoke commands;
3. let JavaScript choose policy and presentation while native Go owns secrets and durable state transitions;
4. add one atomic go-go-goja operation that validates an authenticated invitee, creates membership, and consumes the application capability;
5. preserve a pending application invite across registration and OIDC login; and
6. prove the complete path in the shared local Compose deployment before changing k3s/GitOps.

The result is deliberately small enough to ship and strong enough to support real applications.

## 1. System orientation

### 1.1 The three actors

An intern should begin by separating three kinds of software that participate in the browser flow.

**TinyIDP** is an OpenID Connect provider. It authenticates a person and issues signed protocol artifacts. It stores a local identity, password credential, browser session, authorization interaction, and OIDC grants. A successful TinyIDP signup creates an identity; it does not make that identity a member of any application organization.

**Message Desk** is a relying party and application. It redirects a browser to TinyIDP, receives an authorization callback, validates tokens, and establishes its own application session. Its current example permits provider-owned registration through `GET /auth/register` and the `tinyidp_signup=1` authorization parameter.

**The go-go-goja auth example** is another relying party with a JavaScript route layer and native authentication services. It normalizes an OIDC identity into an application user, authorizes access to tenants and resources, and exposes application capabilities to JavaScript. Its application user and membership records are not TinyIDP records.

```text
                         OIDC protocol
  +----------------+  <---------------->  +----------------------+
  |    TinyIDP     |                      | Message Desk         |
  | identity       |                      | app session/messages |
  | credential     |                      +----------------------+
  | browser session|
  | signup invite  |       OIDC protocol  +----------------------+
  | OIDC grants    |  <---------------->  | go-go-goja app       |
  +----------------+                      | app user/membership  |
                                          | org invite/resource  |
                                          +----------------------+
```

### 1.2 Authentication is not membership

OIDC answers, "Which identity did TinyIDP authenticate?" Application authorization answers, "What may this application user do here?" A normal OIDC callback may create or update an application user mapping, but it must not silently grant organization membership. This invariant is already tested by `TestDefaultOIDCUserNormalizerUpsertsUserWithoutGrantingMemberships` in `go-go-goja/pkg/xgoja/hostauth/builder_test.go`.

This distinction matters for both open and closed signup:

- Message Desk may permit anyone to create a TinyIDP identity and then use its public or individually scoped features.
- A second application may allow the same person to authenticate but require an organization invitation before granting any tenant access.
- Preventing TinyIDP signup does not replace application authorization. If another client permits open signup, identities will exist anyway.

### 1.3 Where JavaScript fits

TinyIDP's Goja layer is an application-policy runtime, not an HTTP server inside the identity provider and not a mutable database session. A compiled program declares workflows, named lambdas, schemas, permitted outcomes, permitted effects, and required native capabilities. Native Go invokes one lambda with bounded data, receives a data-only result, destroys or returns the VM worker, and later resumes through an explicit durable continuation after browser input.

JavaScript is appropriate for decisions such as:

- whether this client requires an invitation;
- which signup fields to display;
- whether an invitation resolver's claims satisfy application policy;
- which error field and safe error message to show;
- whether email verification is required; and
- which declared effects should be requested after validation.

Native Go remains responsible for:

- generating and hashing bearer secrets;
- storing invitation records;
- checking expiry, revocation, audience, and replay state;
- password hashing;
- creating identities and sessions;
- consuming continuations and interactions;
- committing all related mutations in one transaction; and
- emitting security audit events.

## 2. Required product behavior

The initial professional feature set must support these scenarios.

### 2.1 TinyIDP account creation

- An operator can issue an opaque, one-time signup invitation for a particular OIDC client and expiry time.
- The raw invitation code is returned once. Only a keyed hash is stored.
- A signup program can require an invitation for one client and allow open signup for another.
- JavaScript can inspect invitation-derived, non-secret evidence before presenting or committing a signup.
- The final invitation consumption occurs in the same transaction as identity, credential, browser session, workflow continuation, and authorization interaction updates.
- Expired, revoked, wrong-audience, already-used, and concurrent second-use attempts fail safely.
- An operator can revoke an unused invitation.

### 2.2 Application membership

- An authorized application administrator can issue an organization invitation with an email, role, organization, expiry, and single-use property.
- A browser can land on an application invite URL before it is authenticated.
- The application preserves the pending invite across signup or login without placing the raw token in logs or a general-purpose session claim.
- Acceptance requires an authenticated application user.
- Email-bound invitations require a verified OIDC email that matches case-insensitively after the application's normalization rules. A future subject-bound invitation can use issuer plus subject instead.
- Creating membership and consuming the application invite are atomic in the application database.
- A retry is safe if TinyIDP signup succeeded but application membership acceptance failed.

### 2.3 Explicit non-goals for the first delivery

The following may be useful later, but they are not prerequisites for shipping the two-app example:

- a generic TinyIDP capability framework replacing `DurableInvitation`;
- multi-use marketing campaigns or quota counters;
- a full invitation administration web UI;
- arbitrary JavaScript SQL, key/value, filesystem, SMTP, or HTTP access;
- a cross-database distributed transaction between TinyIDP and an application;
- organization and role storage inside TinyIDP;
- automatic membership grants during OIDC user normalization;
- a universal application invitation protocol shared by unrelated products;
- email delivery for invite links; an operator can copy the returned link initially; and
- compatibility adapters for superseded APIs. This design changes current incomplete example behavior directly.

Email confirmation is a separate primitive already represented by TinyIDP's challenge and continuation machinery. It can be activated after invitation signup works, but invitation delivery and email ownership verification must not be treated as the same operation.

## 3. Current implementation inventory

### 3.1 TinyIDP already has the durable record

`pkg/idpstore/types.go:83` defines `DurableInvitation`. It stores only `CodeHash`, an ID, an audience, a policy version, an expiry, revocation and redemption timestamps, and safe redemption evidence. It deliberately does not store the browser-visible code.

`pkg/idpstore/interfaces.go:126` defines four narrow lifecycle operations:

```go
type DurableInvitationStore interface {
    CreateDurableInvitation(ctx context.Context, invitation DurableInvitation) error
    GetDurableInvitation(ctx context.Context, codeHash []byte) (DurableInvitation, error)
    RedeemDurableInvitation(ctx context.Context, codeHash []byte, audience string, now time.Time) (DurableInvitation, error)
    RevokeDurableInvitation(ctx context.Context, codeHash []byte, now time.Time) error
}
```

`pkg/sqlitestore/invitation.go` implements the store. Its redemption update includes unredeemed, unrevoked, and unexpired predicates. `RowsAffected == 1` is the concurrency boundary: two requests cannot both redeem the same record.

`pkg/idpinvite/durable.go` adds the secret boundary. It derives the lookup hash with HMAC-SHA-256 and a domain separator. Its `RedeemInTransaction` method accepts a caller-owned transaction so the signup committer can include invitation use in a larger atomic operation.

### 3.2 TinyIDP already has the closed commit boundary

`internal/fositeadapter/scripted_signup.go:332` is the native signup commit boundary. It accepts only a validated data-only effect plan. The allowed sequence is:

```text
createLocalIdentity
attachPasswordCredential
[optional] consumeInvitation
```

Inside one `Store.Update` callback, it consumes the workflow continuation, commits the prepared account and credential, consumes the invitation if present, creates the browser session, and consumes the authorization interaction. Any failure rolls back every mutation.

```text
BEGIN TinyIDP transaction
  consume workflow continuation
  create identity + password credential
  redeem durable signup invitation       (when requested)
  create browser session
  approve authorization interaction
COMMIT
```

This transaction is the correct professional primitive. It prevents the two dangerous partial states: "invitation spent but account missing" and "account created while invitation remained reusable."

### 3.3 The TinyIDP scripting vocabulary is already suitable

`pkg/idpprogram/providers.go` declares invitation providers as either virtual or durable and records their replay and revocation properties. `pkg/idpprogram/capabilities.go` declares `EffectConsumeInvitation`. `internal/gojamodules/tinyidp/module.go:503` creates a closed `ctx.commit.signup(spec)` API; an optional `inviteCode` adds the native invitation-consumption effect. Password values are opaque invocation-scoped secret handles, not JavaScript strings.

The program structure in `pkg/idpprogram/program.go` is serializable and reviewable. JavaScript functions never leak into stored continuations. A continuation identifies a named handler and a fingerprinted program generation.

### 3.4 The actual TinyIDP production gap

`internal/cmds/serve_production.go:366` rejects every program that declares a native capability. It also rejects `OutcomeChallenge` and every effect other than identity creation and password attachment. As a result, the durable invitation and email challenge code can exist and pass package tests while remaining unavailable from the production binary.

Production construction must therefore bind a small allowlist, not remove validation. For this project the allowlist is only the durable invitation lookup capability and `consumeInvitation`. Email challenges remain a later activation unless they are required by the chosen signup policy.

### 3.5 go-go-goja already has application capabilities

`go-go-goja/pkg/gojahttp/auth/capability/capability.go` defines opaque capabilities with hashed tokens, purpose, subject or resource binding, claims, expiry, single-use state, revocation, creator, and audit events. Its SQL store performs redemption in a database transaction. `capability/invite.go` adds a concrete organization-invite shape.

The generated auth example in `examples/xgoja/21-generated-host-auth/verbs/sites.js:62` issues an `org-invite`. However, the public route at line 87 only consumes the capability and returns its claims. It does not require an actor, compare a verified email, or write membership. The comment on `AcceptOrgInvite` says its result is "needed to create a membership," accurately describing the incomplete seam.

The membership store can query memberships, and the concrete SQL store has an `AddMembership` method, but the application-facing interface has no atomic "consume invite plus add membership" contract. Calling the two existing services sequentially would be unsafe.

## 4. Trust boundaries and invariants

Every implementation and review should test these statements explicitly.

### 4.1 Identity-provider invariants

1. A raw invitation code is a bearer secret. It is returned once, accepted only from the relevant form or trusted operator input, redacted from audit, and never stored.
2. The invitation audience is the validated OIDC client ID. JavaScript cannot substitute an audience at commit time.
3. Inspection is read-only. It never reserves or consumes a code.
4. Only the native signup committer consumes an invitation.
5. Invitation consumption, account creation, credential attachment, session creation, continuation termination, and interaction approval are atomic.
6. A program may request only effects declared at compile time and allowed by the native host.
7. A virtual invitation resolver may decide eligibility in JavaScript, but any promise of one-time use requires a native durable claim record.
8. Failure responses do not reveal whether a guessed code existed, expired, was revoked, or belonged to another client.

### 4.2 Application invariants

1. OIDC normalization may upsert an application user but never grants membership.
2. An application invite is scoped to one application database and one organization.
3. The accepting actor must be authenticated.
4. An email-bound invite is accepted only for a verified, normalized matching email.
5. Role and organization come from the capability, not from the acceptance request body.
6. Membership creation and application-invite consumption are atomic.
7. Repeating a successful acceptance is either an idempotent success for the same membership or a stable conflict; it never changes the role from request data.
8. TinyIDP does not store the organization ID or application role.

## 5. Proposed TinyIDP API and workflow

### 5.1 Do not add a generic storage API

The JS API should expose an invitation **resolver**, not tables or records. The resolver contract allows policy code to work with a stored invitation today and a virtual invitation source later without giving JavaScript persistence authority.

The proposed JavaScript-facing operation is conceptually:

```javascript
const result = ctx.providers.invitation("invitation.signup").validate({
  code: input.fields.inviteCode,
  audience: input.clientId
});

// result is data only; it does not contain the raw code or lookup hash.
// { valid, invitationId, policyVersion, expiresAt, claims?, denialReason? }
```

The exact object path should follow the already compiled provider-binding conventions rather than introducing an unrelated global. The important contract is:

- the provider is declared by ID, kind, version, state, replay protection, revocation, handler, and schemas;
- `audience` is supplied or overwritten by the native invocation context;
- the binding returns a redacted value;
- validation has no side effects; and
- an accepted durable invitation produces an effect only through `ctx.commit.signup`.

A concrete invite-required handler can then be small:

```javascript
signup.lambda("signup.submitted", async (input, ctx) => {
  const invite = await ctx.providers
    .invitation("invitation.signup")
    .validate({ code: input.fields.inviteCode });

  if (!invite.valid) {
    return ctx.ui.present({
      view: "signup",
      fields: ["displayName", "email", "password",
               "passwordConfirmation", "inviteCode"],
      errors: { inviteCode: "This invitation cannot be used." },
      resume: "signup.submitted"
    });
  }

  return ctx.commit.signup({
    login: input.fields.email,
    displayName: input.fields.displayName,
    password: input.secrets.password,
    passwordConfirmation: input.secrets.passwordConfirmation,
    inviteCode: input.fields.inviteCode
  });
});
```

The validation result helps JavaScript decide presentation. The native commit revalidates and redeems the raw code inside the transaction. This intentional second check closes the time-of-check/time-of-use window.

### 5.2 Stored and virtual invitation resolvers

A provider declaration already says whether its state is `durable` or `virtual`. The host can implement both behind a narrow native interface:

```go
type InvitationResolver interface {
    Inspect(ctx context.Context, request InspectInvitationRequest) (InvitationEvidence, error)
}

type InspectInvitationRequest struct {
    ProviderID string
    Code       SecretInput
    Audience   string // fixed by native authorization interaction
    Now        time.Time
}

type InvitationEvidence struct {
    Accepted      bool
    InvitationID  string
    PolicyVersion string
    ExpiresAt     time.Time
    Claims        map[string]string
}
```

For the first delivery, only the existing durable resolver must be production-enabled. A later virtual resolver can compute eligibility from signed data, a deterministic code family, or a virtual-user function. It must declare its honest security properties:

- `state=virtual`, `replayProtection=expiry`, `revocation=key_rollover` is valid for stateless signed evidence;
- it must not claim `replayProtection=one_time` without a durable native claim ledger; and
- if a virtual resolver is used to gate account creation, the commit plan needs a native `claimOnce(providerID, evidenceID)` effect in the same account transaction.

That ledger is a justified future primitive because it supplies one narrow missing property—durable uniqueness—without turning JavaScript into configuration or storage code. It is not required to ship ordinary one-time stored invitations.

### 5.3 Operator API

The smallest useful operator surface is a CLI, implemented with Glazed commands in the TinyIDP production command tree.

```text
tinyidp invitation issue \
  --audience goja-auth \
  --policy-version invite-signup-v1 \
  --ttl 24h

tinyidp invitation revoke --code <opaque-code>
```

`issue` generates at least 256 bits of cryptographic randomness, calls `DurableService.Issue`, and prints the raw code or complete registration URL exactly once. Structured output must mark the code as secret and default logs must include only invitation ID, audience, policy version, expiry, and result.

The lookup HMAC key is operator-managed secret material. In k3s it belongs in a Secret populated through the established Vault/GitOps mechanism and mounted read-only. In local Compose it belongs in an ignored secret file or Compose secret. It must not be a CLI flag visible in process listings, checked-in YAML, JavaScript source, or the invitation database.

### 5.4 Browser request lifecycle

The explicit continuation model means no Goja VM waits for the browser.

```text
Browser                 TinyIDP native host           Goja worker        SQLite
   | GET /authorize            |                          |                |
   | tinyidp_signup=1          |                          |                |
   |-------------------------->| invoke signup.start     |                |
   |                           |------------------------->|                |
   |                           |  Present outcome         |                |
   |                           |<-------------------------|                |
   |                           | persist continuation -------------------->|
   |<----- render HTML form ---|                          | worker returns |
   |                           |                          |                |
   | POST submitted fields     |                          |                |
   |-------------------------->| load continuation       |                |
   |                           | invoke named handler    |                |
   |                           |------------------------->| inspect invite |
   |                           |                          |----native read->|
   |                           |  Commit effect plan      |                |
   |                           |<-------------------------|                |
   |                           | BEGIN atomic commit --------------------->|
   |                           | account + invite + session + interaction  |
   |                           | COMMIT <----------------------------------|
   |<----- OIDC continuation --|                          |                |
```

The browser carries an opaque interaction handle, CSRF value, continuation handle, and user-entered invite code. It never calls JavaScript directly. Native HTTP handlers validate method, content type, CSRF, handles, expiry, generation fingerprint, schemas, and effect plans before any state mutation.

## 6. Proposed go-go-goja membership acceptance

### 6.1 The required native atomic operation

The application capability SQL store and membership SQL store currently own related rows but expose separate operations. Add a focused application service backed by one database transaction:

```go
type MembershipInvitationService interface {
    Accept(ctx context.Context, request AcceptMembershipInvitationRequest) (Membership, error)
}

type AcceptMembershipInvitationRequest struct {
    Token         string
    ActorUserID   string
    ActorEmail    string
    EmailVerified bool
}
```

Its SQL implementation performs the following pseudocode:

```text
BEGIN application transaction
  load capability FOR UPDATE / through conditional mutation contract
  verify purpose == org.invite.accept
  verify unexpired, unrevoked, unused
  verify capability resource type == org
  load authenticated application user
  verify user is enabled
  verify invite email is nonempty
  verify actor email is verified and normalized-email matches
  validate role against the application's closed role vocabulary
  insert membership(user_id, org_id, role)
  mark capability used with a conditional update
COMMIT
```

If membership already exists for the same user, organization, and role, the service may return idempotent success and consume an otherwise valid invitation. A conflicting existing role should return a stable conflict and leave the invitation unused for operator review. This behavior must be selected explicitly in tests; request data never chooses it.

The SQL abstraction must support both SQLite and PostgreSQL using the repository's existing dialect helpers. Do not compose the existing `capability.Service.Consume` and `Store.AddMembership` methods sequentially because they commit independently.

### 6.2 JavaScript route contract

JavaScript may own routing and response presentation while native code owns the acceptance transaction. The route changes from public consumption to authenticated acceptance:

```javascript
app.post("/org-invites/accept")
  .auth(express.user().required())
  .csrf()
  .audit("org.invite.accepted")
  .handle(async (ctx, res) => {
    const accepted = await auth.membershipInvites.accept({
      token: ctx.body.token
      // actor identity is injected by native code, not accepted here
    });

    res.redirect(`/orgs/${accepted.orgId}`);
  });
```

The JS function receives an opaque native operation whose implementation uses `ctx.actor`. The host must ignore or reject `userId`, `email`, `role`, and `orgId` supplied by request data. The only browser-provided authority is the invite token; all other authority comes from the authenticated actor and stored capability.

### 6.3 Issuance remains application policy

The existing issuance route already has the right outline: require an authenticated user, resolve the organization resource, enforce `org.member.invite`, and then issue a short-lived, single-use resource capability. Strengthen it by:

- validating and normalizing the requested email;
- validating role against an application-owned allowlist;
- bounding TTL by native policy even if JavaScript requests a longer duration;
- returning the token once;
- never logging the token; and
- generating a complete HTTPS landing URL for operator or mailer use.

## 7. Joining the two journeys

### 7.1 One user-visible link, two authorities

The clean entry point is the application, not TinyIDP. An organization invite link points to the application:

```text
https://goja.example/invites/<opaque-application-token>
```

The application inspects the token without consuming it. If the browser has no application session, the app stores a hashed or encrypted pending-invite reference in a short-lived server-side transaction and offers two actions:

- **Sign in** for a person who already has a TinyIDP identity.
- **Create identity** when this application is allowed to initiate TinyIDP signup.

For an existing identity, no TinyIDP signup invitation is necessary. For a new identity in an invite-gated TinyIDP client, the application must also supply a TinyIDP signup invitation. Initially this can be a separately issued opaque token associated with the application invite by an operator workflow. A later trusted server-to-server issuance API may automate that association, but it is not needed to establish the security model.

Do not place an application organization capability directly into a TinyIDP invitation record. TinyIDP should not know application tenant or role semantics.

### 7.2 Cross-database saga

There is no distributed transaction across the identity provider and application databases. The sequence is a small retryable saga:

```text
User          goja application             TinyIDP
 | open app invite |                           |
 |---------------->| inspect, retain pending  |
 | choose signup   |                           |
 |<-- redirect ----|-------------------------->|
 |                 |  TinyIDP signup + consume |
 |                 |  account invitation       |
 |<---------------- OIDC callback --------------|
 |                 | normalize application user|
 |                 | atomically add membership |
 |                 | + consume app invitation  |
 |<-- organization home ------------------------|
```

If TinyIDP signup succeeds and the callback or membership transaction fails, the identity remains valid and the application invite remains unconsumed. The user signs in again and retries acceptance. This is the correct recoverable state. Reversing the order would consume application access before an authenticated identity exists.

Pending state should contain a random transaction handle and a server-side reference to the application capability, with an expiry and intended post-login return path. It must be bound into the OIDC `state` transaction and validated on callback. A raw invite token need not round-trip in a query string after the landing request.

### 7.3 Register route for the goja host

Message Desk already distinguishes `/auth/login` from `/auth/register`. The generated goja host currently exposes `/auth/login` only. Add a native `GET /auth/register` route that uses the same OIDC transaction protections but adds `tinyidp_signup=1`. It must preserve the pending application-invite transaction and safe local `return_to` value.

Login and registration must remain distinct user intentions. An identity provider may decline registration for a client while still permitting login by existing identities.

## 8. Error handling, auditing, and operations

### 8.1 Browser-safe errors

Invitation errors presented to an unauthenticated browser should be deliberately coarse:

```text
"This invitation cannot be used. Ask the sender for a new invitation."
```

Detailed categories belong in structured audit and metrics, not the form:

- `not_found_or_wrong_audience`
- `expired`
- `revoked`
- `already_consumed`
- `identity_mismatch`
- `role_rejected`
- `state_conflict`
- `internal_failure`

HTTP semantics for authenticated application acceptance may distinguish a used invite (`409 Conflict`) from invalid input (`400`) and missing authentication (`401`), but should still avoid revealing email-bound invitation details to the wrong actor.

### 8.2 Audit events

At minimum record:

| Layer | Event | Safe fields |
|---|---|---|
| TinyIDP | `signup_invitation.issued` | ID, audience, policy version, expiry, issuer/operator |
| TinyIDP | `signup_invitation.revoked` | ID, audience, result |
| TinyIDP | `signup_invitation.validated` | ID when known, client ID, result category |
| TinyIDP | `signup_invitation.consumed` | ID, client ID, new subject, transaction result |
| App | `org_invite.issued` | capability ID, org ID, normalized email fingerprint, role, actor ID, expiry |
| App | `org_invite.accepted` | capability ID, org ID, user ID, role, result |

Never record raw invite codes, lookup hashes, passwords, password handles, cookies, authorization codes, OIDC tokens, or CSRF tokens.

### 8.3 Rate limits and readiness

Existing TinyIDP registration rate limiting remains active. Invitation validation should have its own low-cost client/IP budget because it can become an online guessing oracle even when responses are coarse. The application landing and acceptance endpoints need similar limits.

Production readiness should fail when a program declares `invitation.durable` but the durable service or HMAC lookup key is absent. It should not silently fall back to open signup or a virtual provider. `tinyidp script check`/explain output should show the provider's state, replay protection, revocation mode, native effect, and whether the production host can bind it.

## 9. Implementation phases and file-level guide

The ticket's `tasks.md` is authoritative. The phases below explain dependencies and review order.

### Phase 0 — Lock down contracts

Read these files first:

- `tiny-idp/pkg/idpprogram/program.go` — compiled, runtime-independent program contract.
- `tiny-idp/pkg/idpprogram/providers.go` — invitation provider metadata and guarantees.
- `tiny-idp/internal/gojamodules/tinyidp/module.go` — JS context construction and commit-effect generation.
- `tiny-idp/internal/fositeadapter/scripted_signup.go` — browser continuation and sole native commit boundary.
- `go-go-goja/pkg/gojahttp/auth/capability/capability.go` — application capability lifecycle.
- `go-go-goja/pkg/gojahttp/auth/appauth/appauth.go` — application user and membership model.

Write tests for the trust-boundary invariants before changing activation. Do not rename APIs or add compatibility aliases as part of this work.

### Phase 1 — Activate durable TinyIDP invitations

Change `internal/cmds/serve_production.go` so validation recognizes an exact allowlist rather than rejecting every capability. Construct `idpinvite.DurableService` from the production store and a dedicated secret lookup key, and pass it through `embeddedidp.ScriptedSignupConfig`.

Implement Glazed issue/revoke commands near the existing production/admin command definitions. Add audit records at the service or command boundary. Test CLI JSON output for secret redaction and verify failures never echo the code.

### Phase 2 — Ship one invite-required signup program

Add a concrete example next to `pkg/idpsignup/open_signup.js` or under `examples/tinyidp-script/`. It should use the declared durable invitation provider, present a field-level error, and request `consumeInvitation` only in the final commit.

Test the program in package-level deterministic tests and through the production constructor. Add SQLite concurrency and rollback integration tests. Run focused tests during development, then `go test ./...` once at the phase boundary.

### Phase 3 — Complete membership acceptance in go-go-goja

Introduce the smallest store/service interface necessary for a single application DB transaction. Implement SQLite and PostgreSQL variants. Update the generated auth example route to require authentication and CSRF and call the native acceptance service. Keep OIDC normalization membership-free.

Bootstrap an initial organization/resource/admin through deployment initialization so someone is authorized to issue the first invite. Bootstrap is an operator action, not a side effect of the first login.

### Phase 4 — Preserve pending invites through OIDC

Extend the host OIDC transaction data with a server-side pending-invite reference and safe return target. Add `/auth/register` alongside `/auth/login`. On callback, normalize the user, restore pending state, and direct the browser to authenticated acceptance. Do not automatically accept in the callback handler unless CSRF and explicit user confirmation requirements are deliberately satisfied.

### Phase 5 — Prove the product locally, then deploy

Use `tiny-idp/examples/tinyidp-shared-two-apps/compose.yaml` as the fast feedback environment. Keep Message Desk open-signup and make the goja application invite-gated so one shared TinyIDP demonstrates different client policies. Add a browser test covering issuance, new-account signup, callback, membership, replay rejection, and existing-user acceptance.

Only after local acceptance should the deployment repositories receive k3s/GitOps changes: secret mounts, program source/config map, initial bootstrap job, database migrations, and ingress routes. The invitation core must not depend on Traefik or k3s; those are deployment bindings around the same HTTPS public origins.

## 10. Test matrix

### 10.1 TinyIDP tests

| Case | Expected result |
|---|---|
| valid code, correct client | account, credential, session, interaction, continuation, and redemption all commit |
| wrong client audience | generic denial; no mutation |
| expired code | generic denial; no mutation |
| revoked code | generic denial; no mutation |
| already-used code | generic denial or stable conflict; no second account |
| two concurrent submissions | exactly one transaction commits |
| duplicate login after valid inspect | transaction rolls back and invitation remains unused |
| password policy failure | invitation remains unused |
| continuation expired | invitation remains unused |
| program asks for undeclared capability | compile/activation fails |
| required invitation service absent | readiness/activation fails closed |
| open-signup client | signup succeeds without invitation effect |

### 10.2 Application tests

| Case | Expected result |
|---|---|
| authenticated, verified matching email | membership and capability use commit together |
| unauthenticated actor | `401`; invite remains unused |
| unverified email | denial; invite remains unused |
| different verified email | denial; invite remains unused |
| invalid requested role at issuance | issuance rejected |
| changed role in acceptance body | ignored/rejected; stored invite role is authoritative |
| expired/revoked/used token | no membership |
| membership insert fails | capability remains unused |
| capability conditional update loses race | membership transaction rolls back |
| normal OIDC login without invite | user upserted, no membership |
| existing member repeats same acceptance | documented idempotent success or stable conflict |

### 10.3 End-to-end browser tests

1. An admin signs in to the goja application and issues an organization invite.
2. A fresh browser opens the application landing link.
3. The browser chooses registration and reaches TinyIDP's styled signup form.
4. A valid TinyIDP account invitation permits account creation.
5. OIDC returns to the application with verified identity data.
6. The user confirms membership acceptance.
7. The organization page becomes accessible with the invited role.
8. Reopening either invitation fails safely.
9. The same TinyIDP identity can still use Message Desk according to Message Desk policy.

## 11. Review checklist

### Security review

- [ ] Raw invitation tokens cannot appear in logs, database rows, metrics labels, URLs after landing, or error text.
- [ ] Audience and actor identity come from validated native context.
- [ ] Both redemption paths use conditional atomic state transitions.
- [ ] TinyIDP signup commits all related records in one transaction.
- [ ] Application membership and app-capability consumption share one transaction.
- [ ] Email matching requires a verified claim and documented normalization.
- [ ] Program activation fails closed when a declared service is absent.
- [ ] Rate limits and audit events cover issue, inspect, accept, revoke, and replay.

### Architecture review

- [ ] TinyIDP contains no organization or role policy.
- [ ] OIDC normalization contains no membership grant.
- [ ] JavaScript receives narrow lambdas/capabilities, not raw stores.
- [ ] No Goja VM or promise survives a browser round trip.
- [ ] Explicit continuations are versioned and generation-bound.
- [ ] Open signup remains possible per client without weakening invite-gated app authorization.
- [ ] No backwards-compatibility shim was added for the incomplete public acceptance route.

### Operational review

- [ ] Lookup keys and other bearer-secret material use mounted secrets.
- [ ] Issuance produces a copyable URL exactly once.
- [ ] Revocation has a supported operator command.
- [ ] Database migrations work for SQLite and PostgreSQL where supported.
- [ ] Local Compose acceptance passes before k3s/GitOps changes.
- [ ] Runbooks explain recovery when identity creation succeeds but membership acceptance must be retried.

## 12. Alternatives considered

### Put organization invites in TinyIDP

Rejected because TinyIDP would become coupled to every application's tenant and role model. It would also make a shared identity provider an application authorization database.

### Use one invite record for both account and membership

Rejected because the two commits occur in different databases under different authorities. It creates ambiguous ownership, difficult revocation semantics, and a false expectation of distributed atomicity.

### Expose SQL or generic key/value storage to JavaScript

Rejected because it makes workflows difficult to audit, grants excessive authority, creates transaction-lifetime hazards, and moves schema and concurrency correctness into untyped policy code.

### Consume an invite during validation

Rejected because a later password, duplicate-login, continuation, or database failure would spend the invite without creating the requested result.

### Grant membership automatically on every OIDC callback

Rejected because authentication is not authorization. It would turn possession of any TinyIDP identity into access to every application organization.

### Require email confirmation before doing invitation work

Deferred. Email challenges are important for email-bound access, and TinyIDP already models them, but activating mail delivery is a separate operational project. The application must nevertheless require `email_verified=true` when accepting an email-bound membership invite. Until TinyIDP email challenges are production-bound, deployments must obtain verified email from an explicitly trusted identity flow or avoid claiming that email-bound invites are production-ready.

## 13. Definition of done

This project is done when a new intern can run the shared local stack and demonstrate all of the following without modifying a database by hand:

- issue and revoke a TinyIDP signup invitation through an operator command;
- create an identity for an invite-gated client exactly once;
- continue to allow open signup for Message Desk;
- issue an application organization invitation as an authorized administrator;
- accept it only as the intended verified identity;
- observe membership creation and single-use consumption in one application transaction;
- retry safely after an interrupted OIDC/application step;
- inspect audit records without finding bearer secrets; and
- run focused, full-suite, and browser acceptance tests successfully.

Deployment to k3s and GitOps PRs comes after this local definition is satisfied. That ordering keeps infrastructure iteration from hiding protocol, transaction, or policy defects.

## References

### TinyIDP

- `pkg/idpstore/types.go:83` — durable invitation data model.
- `pkg/idpstore/interfaces.go:126` — invitation store and transaction contracts.
- `pkg/sqlitestore/invitation.go:14` — SQLite issue, lookup, redemption, and revocation.
- `pkg/idpinvite/durable.go:14` — HMAC hashing and transaction-aware service.
- `pkg/idpprogram/providers.go:8` — provider kinds and durability declarations.
- `pkg/idpprogram/capabilities.go:15` — native effect vocabulary.
- `pkg/idpprogram/program.go:10` — compiled program artifact.
- `internal/gojamodules/tinyidp/module.go:503` — closed JS commit and challenge contexts.
- `internal/fositeadapter/scripted_signup.go:332` — atomic signup commit.
- `internal/cmds/serve_production.go:347` — current production-program restriction.
- `examples/tinyidp-message-app/app_http.go:74` — distinct login and registration routes.
- `examples/tinyidp-message-app/oidc_client.go:117` — signup authorization intent.
- `examples/tinyidp-shared-two-apps/compose.yaml` — local shared-provider integration environment.

### go-go-goja

- `pkg/gojahttp/auth/capability/capability.go:28` — durable application capability model and service.
- `pkg/gojahttp/auth/capability/invite.go:8` — organization invitation helper.
- `pkg/gojahttp/auth/capability/sqlstore/sqlstore.go:112` — transactional capability redemption.
- `pkg/gojahttp/auth/appauth/appauth.go:51` — application membership model.
- `pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go:116` — current non-atomic membership write.
- `pkg/xgoja/hostauth/builder_test.go:235` — OIDC normalization without membership grant.
- `examples/xgoja/21-generated-host-auth/verbs/sites.js:62` — current issue and incomplete acceptance routes.
