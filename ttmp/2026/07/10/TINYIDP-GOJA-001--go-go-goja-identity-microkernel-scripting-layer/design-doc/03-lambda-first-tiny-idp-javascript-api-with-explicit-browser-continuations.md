---
Title: Lambda-first Tiny-IDP JavaScript API with explicit browser continuations
Ticket: TINYIDP-GOJA-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/interaction.go
      Note: Existing hashed one-use browser continuation creation, binding, and OAuth request reconstruction
    - Path: repo://internal/fositeadapter/provider.go
      Note: Current authorization and provider-owned registration request flow that the workflow executor must integrate without bypassing Fosite validation
    - Path: repo://internal/gojamodules/tinyidp
      Note: Isolated require("tinyidp").v1 builder and per-runtime callback collector (commit 1b8cb17).
    - Path: repo://internal/gojaverify/compiler.go
      Note: Existing isolated compile-only Goja implementation and forbidden ambient-module precedent
    - Path: repo://pkg/idpprogram
      Note: Phase 0 runtime-independent program, lambda, schema, outcome, validation, and fingerprint implementation (commit 0e0a4b0).
    - Path: repo://pkg/idpscript
      Note: Compiler, immutable artifact, isolated owned-runtime loader, and fingerprint tests (commit 1b8cb17).
    - Path: repo://pkg/idpstore/types.go
      Note: Existing interaction and security-state vocabulary that motivates the new versioned workflow continuation record
    - Path: repo://pkg/idpui/types.go
      Note: Existing provider-owned presentation models and renderer security boundary
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Deprecated graph-first predecessor whose security findings remain historical context
    - Path: ws://go-go-goja/pkg/engine/factory.go
      Note: Owned runtime factory, explicit module selection, event loop, and runtime lifecycle APIs
    - Path: ws://go-go-goja/pkg/runtimeowner/runner.go
      Note: Serialized Call and Post primitives used to invoke lambdas on the runtime owner
ExternalSources:
    - https://openid.net/specs/openid-connect-core-1_0-final.html
    - https://www.rfc-editor.org/rfc/rfc9700.txt
Summary: Superseding intern-oriented design for a lambda-first Tiny-IDP API in which trusted JavaScript implements workflows and virtual resources while Go owns protocol validation, explicit browser continuations, secrets, challenges, atomic effects, and artifact issuance.
LastUpdated: 2026-07-19T12:00:00-04:00
WhatFor: Implementing the primary Tiny-IDP scripting API, runtime invocation seams, signup workflows, virtual identities, virtual invitations, browser continuations, capability boundaries, and atomic native effects.
WhenToUse: Use this document instead of design-doc/01 when designing or implementing the Tiny-IDP JavaScript configuration and runtime API. Read design-doc/02 alongside it for the assurance vocabulary and refactoring sequence.
---




# Lambda-first Tiny-IDP JavaScript API with explicit browser continuations

> **Supersession notice:** This document supersedes
> `design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md`
> as the normative JavaScript API and runtime design. The older document is
> retained as historical research and is deprecated. Its security boundary,
> isolated compiler, capability discipline, runtime ownership, typed outcomes,
> and assurance recommendations remain useful; its emphasis on JavaScript as a
> richer configuration language and its authorization/claims-first API plan are
> replaced here by a lambda-first workflow model with explicit browser
> continuations.

## 1. Executive summary

Tiny-IDP should be programmable, not merely configurable. A deployment author
should be able to write JavaScript lambdas that decide how signup works, resolve
virtual users, validate virtual invitations, call application capabilities,
compute identity attributes, route between authentication methods, and assemble
effect plans. Adding a new application-specific workflow should normally change
JavaScript, not add another Go enum, command-line flag, database table, or YAML
mode.

Go remains the identity kernel. It validates OAuth and OpenID Connect requests,
owns HTTP and browser security, controls secret values, implements credential
and cryptographic operations, persists replay-sensitive challenges, applies
atomic effects, establishes sessions, issues protocol artifacts, and records
audits. JavaScript receives bounded typed values and narrow capabilities. It
returns one of a small set of structured outcomes. It never receives Fosite
objects, SQL transactions, signing keys, password bytes, cookies, authorization
codes, or unconstrained network access.

Browser interaction creates a special execution boundary. A JavaScript `await`
can safely wait for a bounded capability call during one HTTP request. It cannot
be the sole representation of a signup form or email-confirmation wait, because
the browser response ends and the next POST may arrive after a restart or on a
different process. Browser boundaries therefore use **explicit continuations**:
a handler returns a presentation or challenge with a named `resume` handler; Go
stores a versioned continuation record; the later HTTP request is validated by
Go; and a fresh lambda invocation receives the validated event and evidence.

The core programming model is:

```javascript
const A = require("tinyidp").v1;

module.exports = A.program("community-idp", program => {
  program.workflow("signup", {
    start: ctx => ctx.present.signupForm({
      fields: [
        A.field.displayName({ required: true }),
        A.field.email({ required: true }),
        A.field.inviteCode({ required: false }),
      ],
      resume: "submitted",
    }),

    submitted: async ctx => {
      const member = await ctx.cap.community.lookup({
        email: ctx.input.email.address,
      });

      if (!member && !ctx.input.inviteCode.present) {
        return ctx.present.signupForm({
          fields: ctx.presentation.fields,
          errors: { registration: "Account creation was not accepted." },
          values: ctx.input.publicValues(),
          resume: "submitted",
        });
      }

      return ctx.challenge.emailCode({
        email: ctx.input.email,
        resume: "emailVerified",
        carry: {
          displayName: ctx.input.displayName,
          email: ctx.input.email,
        },
      });
    },

    emailVerified: ctx => ctx.commit.signup({
      identity: {
        kind: "virtual",
        subjectSeed: ctx.evidence.email.address,
        email: ctx.evidence.email.address,
        emailVerified: true,
        displayName: ctx.carry.displayName,
      },
      establishSession: true,
    }),
  });
});
```

This is executable application logic. The graph surrounding it records handler
names, schemas, allowed outcomes, capabilities, effects, budgets, and
continuation edges. Static analysis treats each lambda as an opaque but
constrained transition. Model checking explores every outcome declared by that
transition. Runtime enforcement verifies the actual input, capability calls,
output, and effects.

## 2. What this document changes

The original scripting design made the immutable graph the primary abstraction
and initially limited request-time JavaScript to authorization and claims
callbacks. That was a sound conservative starting point, but it underused the
language and postponed the most valuable application behavior: signup,
application-specific identity resolution, virtual resources, and multi-request
workflows.

This document makes four changes.

1. **Lambdas are primary behavior.** The graph describes their contracts and
   composition; it does not replace ordinary branching and computation with a
   large catalog of configuration nodes.
2. **Signup is the first vertical slice.** The implementation begins with the
   provider-owned registration flow that exists today, then generalizes it into
   named workflow handlers.
3. **Virtual resources are first-class.** A user or invite provider may be
   durable, external, static, derived, or computed by a lambda. SQLite is one
   implementation, not the meaning of identity.
4. **Browser waits are explicit continuations.** The public API does not pretend
   that an ordinary Promise is durable. A future source transform may offer
   browser-spanning `await` as syntax sugar, but its compiled contract must be
   the continuation model defined here.

The assurance-oriented core grammar in `design-doc/02` remains complementary.
Its resource, fact, obligation, step, effect, outcome, observation, and property
identifiers should become the stable names used by this runtime.

## 3. System orientation for a new implementer

### 3.1 Tiny-IDP is an OAuth authorization server and OpenID Provider

The browser-facing application does not collect the user's Tiny-IDP password or
create the Tiny-IDP account. It starts an OAuth Authorization Code flow with
PKCE. Tiny-IDP validates the client, exact redirect URI, requested scopes,
prompt rules, PKCE request, browser session, and consent requirements. When the
flow completes, Tiny-IDP redirects the browser back with a short-lived
authorization code. The application exchanges that code at the token endpoint.

The current path begins in `internal/fositeadapter/provider.go`. At lines
650-760, `beginAuthorize` asks Fosite to parse the request before it creates any
browser interaction. At lines 684-697, the current implementation recognizes
provider-owned signup and changes the required interaction action. At lines
775-970, `resumeAuthorize` validates form parsing, rate limiting, CSRF, expiry,
browser binding, the reconstructed OAuth request, current client generation,
allowed actions, and registration input before creating an account and browser
session.

Those checks are not JavaScript hooks. The workflow executor is inserted after
the relevant native validation and before irreversible native effects.

### 3.2 The existing interaction is already a continuation

`pkg/idpstore/types.go:258-280` defines `InteractionRecord`. It persists a keyed
hash of an opaque browser handle, the canonical validated authorization request,
request digest, client and redirect binding, required actions, browser/session
binding hashes, generation hash, expiry, and terminal consumption state.

`internal/fositeadapter/interaction.go:80-123` creates that record.
`interaction.go:157-176` reconstructs the OAuth request from the stored canonical
form and asks Fosite to validate it again, then compares the request digest.

This is strong existing infrastructure. The scripting design should generalize
it rather than introduce an unrelated `/signup` session mechanism.

### 3.3 Provider-owned UI is already separated from HTTP authority

`pkg/idpui/types.go:145-220` defines presentation-only models. An
`InteractionPage` contains a form model, optional login, consent, account
chooser, or registration prompt, and a public error. It deliberately contains
no password, cookie, original OAuth request, authorization code, or store
object.

`pkg/idpui/renderer.go:8-21` gives renderers an `io.Writer`, not an
`http.ResponseWriter`. A renderer cannot set cookies, status codes, redirects,
or security headers. The lambda-first UI API preserves this boundary: scripts
produce validated presentation specifications; Go produces the authoritative
form action, hidden handles, CSRF values, headers, and HTML model.

### 3.4 Goja runs on an owned event loop

`go-go-goja/pkg/engine/factory.go:182-289` constructs a Goja VM, starts its event
loop, assigns a `runtimeowner.RuntimeOwner`, installs explicitly selected
modules, and tracks runtime lifetime. `pkg/runtimeowner/runner.go:90-187`
provides `Call` and `Post`; both serialize VM access onto the owner scheduler.

Goja is not safe for concurrent arbitrary access. Every callback invocation,
Promise settlement, and runtime mutation must run through its owner.

The go-go-goja async guide documents the supported native Promise pattern. A
native function creates a Promise on the VM thread, performs bounded work off
the VM thread, then uses `PostWithCustomContext` to resolve or reject on the
owner. That pattern is appropriate for database or application capability calls
that finish within one request. It is not itself a durable browser
continuation.

### 3.5 Tiny-IDP already has a compile-only Goja precedent

`internal/gojaverify/compiler.go:30-84` creates an isolated Goja runtime,
installs only `tinyidp/verify`, rejects ambient modules, applies a source-size
limit and execution deadline, exports plain data, and validates it in Go. The
new program compiler should reuse this posture, but it must additionally retain
the exact source and deterministic callback registry needed to create request
runtimes.

## 4. Goals, non-goals, and trust model

### 4.1 Goals

- Application authors can implement signup, login routing, virtual users,
  virtual invitations, authorization, consent policy, and claims with named
  JavaScript lambdas.
- Scripts compose a small set of robust native capabilities rather than opening
  files, sockets, SQL databases, or cryptographic key material.
- Browser workflows remain correct across HTTP requests, process restarts, and
  script reloads through explicit versioned continuations.
- Scripts choose whether an identity or invitation is stored, external,
  stateless, derived, or virtual.
- Native code validates every input and output at the JavaScript boundary.
- Native code applies all replay-sensitive or issuance-sensitive effects.
- The system exposes enough metadata for static analysis, model checking,
  runtime tracing, policy tests, and human explanation.
- The initial implementation can replace the current hardcoded registration
  branch without rewriting Fosite or weakening the current interaction checks.

### 4.2 Non-goals

- JavaScript does not parse OAuth requests or validate redirect URIs, PKCE,
  JWTs, signatures, nonces, or replay markers.
- JavaScript does not receive raw password bytes, password hashes, signing
  material, cookies, authorization codes, refresh tokens, or SQLite handles.
- Scripts do not hold open SQL transactions.
- Browser continuations do not serialize or retain Goja heaps, closures, or
  Promise resolvers.
- The first release does not support arbitrary Node.js packages or ambient
  `fs`, `fetch`, `exec`, `database`, `os`, `process`, or environment access.
- Untrusted tenant-authored code does not run in the IdP process. This design is
  for trusted deployment code reviewed and deployed with the IdP.
- The scripting layer does not replace the separate compile-only verification
  plane in `tinyidp/verify`.

### 4.3 Trust levels

| Surface | Author | Authority | Lifetime |
|---|---|---|---|
| Program compiler | Trusted operator | Build definitions and callback registry only | Startup/reload |
| Workflow lambda | Trusted operator | Declared inputs, outcomes, capabilities, and effects | One invocation |
| Virtual provider lambda | Trusted operator | Provider-specific bounded contract | One invocation |
| Native capability | Go host | Exact typed service operation | Request or runtime |
| Browser | Untrusted user agent | Submit bounded form values and opaque handles | HTTP request |
| Verification script | Test author | Describe native-run scenarios only | Offline command |

The runtime sandbox limits accidental and exploitable authority. It is not a
claim that hostile JavaScript is safely contained inside the serving process.

## 5. Core execution model

### 5.1 Programs, workflows, handlers, and providers

A **program** is the complete compiled Tiny-IDP JavaScript source for one graph
generation. It registers clients, workflows, virtual providers, lambdas,
capability requirements, and tests.

A **workflow** is a named collection of handlers connected by continuation
labels. A handler is a JavaScript lambda invoked for one event. Handlers can
call bounded capabilities during that invocation and return one structured
outcome.

A **provider** is a named collection of lambdas implementing a typed resource
contract, such as user lookup, invitation validation, identity attribute
projection, or application membership lookup.

```javascript
const A = require("tinyidp").v1;

module.exports = A.program("community", program => {
  program.capabilities({
    "community.lookup": { version: 1 },
    "mail.transactional": { version: 1 },
  });

  program.provider("users", "community-users", {
    find: A.lambda("community-users.find", {
      input: "identityLookupV1",
      output: "identityCandidateV1",
      capabilities: ["community.lookup"],
      effects: ["read"],
      run: async ctx => {
        const member = await ctx.cap.community.lookup({
          login: ctx.input.login,
        });
        if (!member) return A.result.notFound();
        return A.result.found({
          kind: "virtual",
          subjectSeed: member.id,
          displayName: member.name,
          roles: member.roles,
        });
      },
    }),
  });
});
```

The compiler records the function under `community-users.find`; it does not
serialize the function into the graph. Every runtime generation loads the exact
same source, registers the same callback names, and must produce the same
registry fingerprint before activation.

### 5.2 Lambda contract

Every lambda has explicit metadata:

```go
type LambdaSpec struct {
    ID                   string
    Kind                 LambdaKind
    InputSchema          string
    OutputSchema         string
    AllowedOutcomes      []OutcomeKind
    RequiredCapabilities []CapabilityRequirement
    AllowedEffects       []EffectKind
    Timeout              time.Duration
    MaxCapabilityCalls   int
    MaxOutputBytes       int
    SourceLocation       SourceLocation
}
```

The metadata is a runtime contract, not documentation. Activation fails if the
host cannot supply a capability or an effect is illegal at the handler's slot.
Invocation fails closed if the output does not match the declared schema or
outcome set.

### 5.3 Outcomes

Handlers return exactly one of these outcome families:

```text
continue   Advance immediately to another handler in this HTTP request.
present    Render a browser form and persist a continuation.
challenge  Start or continue a native proof challenge and persist a continuation.
commit     Ask Go to validate and atomically apply an effect plan.
complete   Return an already established identity or terminal workflow value.
deny       Valid negative policy decision with a stable reason code.
skip       Provider or branch is not applicable; only explicit combinators may continue.
error      Infrastructure or internal failure; fail closed.
```

Exceptions do not mean `deny`; they mean `error`. Returning `undefined` does not
mean `skip`; it is invalid output. A wrong password or invite is `deny`, not
`skip`, because silently trying another factor after rejected evidence creates
factor-confusion behavior.

### 5.4 Invocation pipeline

```text
resolve active generation and handler
        |
load and validate native workflow event
        |
project bounded immutable JavaScript input
        |
acquire one runtime worker exclusively
        |
install invocation-scoped capability bindings and budgets
        |
invoke named lambda on the runtime owner
        |
await bounded in-request Promises
        |
copy and validate the structured result
        |
erase invocation bindings and release/discard worker
        |
interpret outcome in native workflow executor
```

Pseudocode:

```go
func (e *Executor) Invoke(
    ctx context.Context,
    generation *Generation,
    handlerID string,
    event NativeEvent,
) (Outcome, error) {
    spec, ok := generation.Registry.Handler(handlerID)
    if !ok {
        return Outcome{}, ErrUnknownHandler
    }

    input, err := e.projector.Project(spec.InputSchema, event)
    if err != nil {
        return Outcome{}, fmt.Errorf("project lambda input: %w", err)
    }

    worker, err := generation.Pool.Acquire(ctx)
    if err != nil {
        return Outcome{}, ErrRuntimeSaturated
    }

    result, invokeErr := worker.Invoke(ctx, spec, input)
    if invokeErr != nil {
        generation.Pool.Discard(worker)
        return Outcome{}, invokeErr
    }

    outcome, err := e.codec.DecodeAndValidate(spec, result)
    if err != nil {
        generation.Pool.Discard(worker)
        return Outcome{}, err
    }

    generation.Pool.Release(worker)
    return outcome, nil
}
```

## 6. In-request `await` and browser continuations

### 6.1 In-request Promise

This is supported:

```javascript
submitted: async ctx => {
  const member = await ctx.cap.community.lookup({
    email: ctx.input.email.address,
  });

  return member
    ? ctx.complete.identity(member)
    : ctx.deny("not_a_member");
}
```

The HTTP request remains open. The capability has its own context deadline and
call budget. Its native implementation settles the Promise on the runtime
owner. If the request is canceled, the capability context is canceled. If the
lambda exceeds its total budget, Goja is interrupted and the worker is
discarded.

### 6.2 Browser boundary

This public v1 API is intentionally explicit:

```javascript
start: ctx => ctx.present.signupForm({
  fields: [...],
  resume: "submitted",
});
```

It does not return a Promise that remains pending. The handler returns normally,
the worker returns to the pool, and Go persists a continuation.

The later browser POST invokes `submitted` as a fresh call:

```javascript
submitted: ctx => {
  // ctx.input was parsed and validated from the new POST by Go.
};
```

### 6.3 Why the continuation is explicit

An ordinary pending Promise lives only in one Goja heap. It is lost on process
restart, ties the interaction to one runtime and source generation, consumes
runtime capacity, complicates reload, and may retain secret-bearing values.
Explicit continuations make the durable state small, typed, inspectable, and
independent of a JavaScript heap.

The ergonomic tradeoff is visible control flow. The handler map shows every
browser boundary. A future compiler may transform restricted browser-spanning
`await` syntax into this representation, but the runtime contract remains the
same.

## 7. Browser and OAuth sequence in detail

### 7.1 Initial redirect from the application

Message Desk begins signup by generating state, nonce, and an S256 PKCE pair,
then redirecting the browser to Tiny-IDP:

```http
GET /authorize?
    client_id=tinyidp-message-app
    &redirect_uri=https%3A%2F%2Fmessage-desk.example%2Fauth%2Fcallback
    &response_type=code
    &scope=openid+profile
    &code_challenge=...
    &code_challenge_method=S256
    &state=...
    &nonce=...
    &tinyidp_signup=1
```

Fosite parses and validates the authorization request before the workflow is
selected. The native adapter loads the client, rejects a disabled client,
validates the signup intent, applies prompt rules, and decides whether the
`signup` workflow may begin.

### 7.2 Invoke `signup.start`

Go constructs a bounded start event:

```go
type WorkflowStart struct {
    Workflow string
    Client   ClientView
    Request  AuthorizationRequestView
    Browser  BrowserContextView
}
```

The view contains validated client ID, permitted public display metadata,
scopes, audience, and prompt facts. It does not contain raw headers, query
parameters, redirect mutators, cookies, or Fosite interfaces.

The lambda returns `present`:

```javascript
return ctx.present.signupForm({
  title: "Create your account",
  fields: [
    A.field.displayName({ required: true }),
    A.field.email({ required: true }),
    A.field.inviteCode({ required: false }),
  ],
  actions: [A.action.submit("Create account"), A.action.deny("Cancel")],
  resume: "submitted",
});
```

### 7.3 Persist continuation and render

Go validates the presentation and creates a continuation record:

```go
type WorkflowContinuation struct {
    IDHash               []byte
    WorkflowID           string
    ResumeHandlerID      string
    ProgramHash          []byte
    GraphSchemaVersion   string
    WorkflowVersion      int

    CanonicalRequest     map[string][]string
    RequestDigest        []byte
    ClientID             string
    RedirectURI          string
    ClientGenerationHash []byte

    BrowserBindingHash   []byte
    SessionIDHash        []byte
    BrowserContextHash   []byte

    PresentationID       string
    InputSchema          string
    Carry                []byte
    SecretRefs           []SecretReference
    Evidence             []EvidenceReference

    CreatedAt            time.Time
    ExpiresAt            time.Time
    ConsumedAt           *time.Time
    Outcome              ContinuationOutcome
}
```

The raw handle goes to the browser; only a domain-separated keyed hash is
stored. `Carry` contains bounded schema-validated public or protected data, not
raw secrets. `SecretRefs` are opaque references to native pending-secret state
when a later step genuinely requires it.

The renderer receives a provider-owned page model. Go adds the authoritative
form action, interaction field, CSRF field, CSRF token, and allowed action
identifiers. The renderer cannot change redirects or headers.

### 7.4 Browser POST

The browser later submits:

```http
POST /authorize
Content-Type: application/x-www-form-urlencoded
Origin: https://idp.example

interaction=...
&csrf_token=...
&action=submit
&display_name=Alice
&email=alice@example.org
&invite_code=...
```

Before JavaScript runs, Go validates:

1. HTTP method, content type, and bounded body size.
2. Trusted proxy/public-origin and same-origin requirements.
3. Interaction and CSRF tokens.
4. Continuation existence, expiry, and unconsumed status.
5. Browser, session, and browser-context bindings.
6. Program generation compatibility.
7. Reconstructed OAuth request digest.
8. Current client generation, redirect URI, scopes, and enabled state.
9. Allowed action and exact submitted field set.
10. Field multiplicity, length, normalization, and schema.

Only then does it invoke `signup.submitted`.

### 7.5 Email confirmation

The submitted handler may return:

```javascript
return ctx.challenge.emailCode({
  email: ctx.input.email,
  mailer: "transactional",
  expiresIn: "15m",
  maximumAttempts: 5,
  resume: "emailVerified",
  carry: {
    displayName: ctx.input.displayName,
    email: ctx.input.email,
    inviteEvidence: invite.evidence,
  },
});
```

Go validates the challenge request, generates the code, stores only its keyed
hash and native state, calls the named mailer, creates the next continuation,
and renders a code-entry page. The later POST is validated and the code is
atomically consumed before `emailVerified` runs.

The resumed lambda receives native evidence:

```javascript
ctx.evidence.email = {
  address: "alice@example.org",
  verified: true,
  method: "email_code",
  verifiedAt: "2026-07-19T16:00:00Z",
};
```

JavaScript cannot create `Evidence<verifiedEmail>` from a plain object. The JS
projection may look like an object, but the native output codec accepts
evidence references only from the executor's invocation context.

### 7.6 Terminal commit and OAuth callback

The final handler returns a typed effect request. Go validates it, opens a
short native transaction, rechecks authoritative versions, applies compatible
effects atomically, consumes the workflow continuation, creates the browser
session, records audit/security events, and resumes the original Fosite request.

Fosite issues the authorization code. Tiny-IDP redirects the browser to the
exact previously validated callback. Message Desk exchanges the code at the
token endpoint and creates its application session.

## 8. UI API

### 8.1 Scripts describe presentation intent

The UI namespace creates data-only specifications:

```javascript
A.field.displayName({
  label: "Display name",
  required: true,
  maximumLength: 100,
});

A.field.email({
  label: "Email",
  required: true,
  autocomplete: "email",
});

A.field.inviteCode({
  label: "Invite code",
  required: false,
  sensitive: true,
});
```

Field types are registered native descriptors. Scripts cannot invent an input
type with unreviewed parsing or rendering semantics. A future trusted extension
may register another descriptor through Go and expose its builder through the
module.

### 8.2 Go owns form authority

Scripts cannot specify:

- arbitrary form actions or external URLs;
- hidden interaction or CSRF values;
- raw HTML event handlers;
- CSP, cookies, redirects, or HTTP status codes;
- unregistered action identifiers;
- fields outside the handler's declared input schema.

This preserves the current renderer boundary in `pkg/idpui/renderer.go`.

### 8.3 Rerendering errors

A handler may return the same presentation with safe public values and stable
error categories:

```javascript
return ctx.present.signupForm({
  presentation: ctx.presentation.id,
  resume: "submitted",
  values: ctx.input.publicValues(),
  errors: {
    registration: A.error.public("registration_rejected"),
  },
});
```

`publicValues()` omits secrets. The script may select a registered public error
code, but the host owns its safe default message. Development profiles may
permit custom copy; production must prevent exception text, capability errors,
or account/invite existence details from reaching the browser.

## 9. Secret values

### 9.1 Opaque handles

Password, invite-code, email-code, recovery-code, and similar fields enter Go
as bounded byte buffers. Go immediately wraps them in invocation-scoped opaque
handles before projecting input to JavaScript:

```javascript
ctx.input.password;                 // SecretHandle<password>
ctx.input.password.present;         // true
String(ctx.input.password);         // "[SecretHandle password]"
JSON.stringify(ctx.input);          // secret contents omitted
ctx.input.password.value;           // undefined
```

The handle can be passed to an authorized native capability or effect:

```javascript
await ctx.cap.credentials.verifyPassword({
  identity: candidate.ref,
  candidate: ctx.input.password,
});

return ctx.commit.signup({
  credential: ctx.effect.passwordCredential(ctx.input.password),
});
```

Handles are valid only for the invocation unless Go explicitly converts them to
a pending native secret reference as part of a continuation. They cannot be
exported through ordinary result JSON.

### 9.2 Avoid carrying passwords across email waits

The preferred flow verifies email before collecting a password:

```text
collect public profile and invite
verify email
collect password
commit account
```

If product requirements collect the password earlier, Go must hash or encrypt
it into a bounded pending-credential record before suspending. Neither plaintext
nor a VM handle is stored in `Carry`.

### 9.3 Explicit secret-reveal authority

Some legacy virtual-provider algorithms may require inspecting a raw code. V1
should not ship general reveal authority. If it is later added, it must be a
distinct lambda profile and permission such as `secret.reveal:inviteCode`, with
isolated runtimes, no console inspection, strict output redaction, and a
production-policy gate. It must never apply to passwords or signing material.

## 10. Capabilities

### 10.1 Capability contract

A capability is a host-owned typed service projected into a lambda invocation:

```go
type CapabilityDescriptor struct {
    ID             string
    Version        int
    Effect         EffectClass
    InputSchema    string
    OutputSchema   string
    DefaultTimeout time.Duration
    MaximumCalls   int
    Bind           func(InvocationContext) (CapabilityBinding, error)
}
```

Scripts declare requirements at compile time:

```javascript
A.lambda("signup.memberLookup", {
  input: "signupSubmissionV1",
  output: "signupDecisionV1",
  capabilities: ["community.lookup@1"],
  effects: ["read"],
  run: async ctx => {
    return await ctx.cap.community.lookup({
      email: ctx.input.email.address,
    });
  },
});
```

Activation verifies that the host supplies a compatible descriptor. Invocation
binds the request context, deadline, call count, and redaction policy. A lambda
cannot acquire a capability dynamically by string name.

### 10.2 Capability categories

| Category | Examples | Typical effect |
|---|---|---|
| Directory | `community.lookup`, `organization.membership` | read |
| Credential | `credentials.verifyPassword`, `passkey.verify` | credential/challenge |
| Notification | `mail.sendVerification`, `sms.sendCode` | write |
| Cryptographic verifier | `invite.verifySigned`, `credential.verifyAssertion` | pure/read |
| Risk signal | `risk.registration`, `ip.reputation` | read |
| Audit annotation | `audit.annotate` | observe |
| Native challenge | `challenge.emailCode`, `challenge.passkey` | challenge |

There is no generic SQL, HTTP, filesystem, crypto-key, or fetch capability. A
host adapter may use those internally, but its script-visible schema is narrow.

## 11. Virtual users and identity providers

### 11.1 Identity is not synonymous with a user table row

A workflow may establish an identity from verified evidence without creating a
local user record. A virtual identity still needs a stable subject, bounded
claims, status policy, and authentication evidence.

```javascript
program.provider("identity", "verified-email-users", {
  establish: A.lambda("verified-email-users.establish", {
    input: "verifiedEmailEvidenceV1",
    output: "identityPlanV1",
    effects: ["pure"],
    run: ctx => A.identity.virtual({
      subject: A.subject.pairwise({
        namespace: "message-desk",
        seed: ctx.evidence.email.address,
      }),
      email: ctx.evidence.email.address,
      emailVerified: true,
      displayName: ctx.input.displayName,
      roles: ["member"],
    }),
  }),
});
```

`A.subject.pairwise` is a native operation backed by host secret material. The
script supplies a namespace and verified seed; it does not implement subject
cryptography or receive the key.

### 11.2 Provider result

```go
type IdentityCandidate struct {
    Kind              IdentityKind // durable, virtual, external
    StableReference   string
    SubjectPlan       SubjectPlan
    DisplayName       string
    Email             string
    EmailVerified     bool
    PreferredUsername string
    Groups            []string
    Roles             []string
    Tenant            string
    Locale            string
    Authentication    EvidenceSet
    Version            string
}
```

Native validation enforces subject and claim bounds, protected claim ownership,
evidence provenance, disabled-state policy, and per-client release policy.

### 11.3 Authentication remains separate from lookup

A lambda may decide which candidate to use, but password verification must call
a credential capability. A virtual user may refer to a named Secret-backed
credential or avoid passwords entirely by using verified email, upstream OIDC,
passkeys, or another native factor.

## 12. Virtual and durable invitations

### 12.1 Stateless signed invite

```javascript
const invite = await ctx.cap.invite.verifySigned({
  code: ctx.input.inviteCode,
  verifier: "community-invites",
  audience: ctx.client.id,
  maximumLifetime: "14d",
});
```

The native verifier returns validated claims and evidence. No database row is
required. Expiry and audience restriction are available; strict one-time use
and immediate individual revocation are not.

### 12.2 Computed virtual invitation

```javascript
program.provider("invites", "community-eligibility", {
  validate: A.lambda("community-eligibility.validate", {
    input: "inviteProbeV1",
    output: "inviteDecisionV1",
    capabilities: ["community.lookup@1", "invite.verifySigned@1"],
    effects: ["read"],
    run: async ctx => {
      const signed = await ctx.cap.invite.verifySigned({
        code: ctx.input.code,
        audience: ctx.client.id,
      });
      if (signed.valid) return A.invite.accept(signed.claims);

      const member = await ctx.cap.community.lookup({
        email: ctx.registration.email.address,
      });
      return member?.mayRegister
        ? A.invite.accept({ source: "community" })
        : A.invite.reject("not_eligible");
    },
  }),
});
```

The lambda is real behavior. Native capabilities handle secrets and external
I/O. The provider can combine code validation with application state.

### 12.3 Durable one-time invitation

A durable provider may return versioned redemption evidence. The final commit
requests `consumeInvite`. Go rechecks and consumes it in the same native
transaction as local account creation. The lambda does not hold a transaction
open or mutate the invitation directly.

| Property | Signed virtual invite | Durable invite |
|---|---|---|
| Database row required | No | Yes |
| Expiry | Yes | Yes |
| Audience restriction | Yes | Yes |
| Single use | No | Yes, atomically |
| Immediate revocation | Key rotation/deny service | Yes |
| Exact redemption link | Audit only | Durable redemption record |

## 13. Effects and transactions

### 13.1 Lambdas return effect plans

Security-sensitive mutation is described, validated, and applied by Go:

```javascript
return ctx.commit({
  effects: [
    ctx.effect.createLocalIdentity(identity),
    ctx.effect.attachPassword(ctx.input.password),
    ctx.effect.consumeInvite(invite.evidence),
    ctx.effect.establishBrowserSession(),
  ],
});
```

For a virtual identity:

```javascript
return ctx.commit({
  effects: [
    ctx.effect.establishVirtualIdentity(identity),
    ctx.effect.establishBrowserSession(),
  ],
});
```

### 13.2 No JavaScript inside store transactions

The executor follows this order:

```text
read/project facts
invoke lambda and capabilities
validate returned outcome and effect plan

begin native transaction
reload authoritative mutable records
verify evidence versions and preconditions
apply all compatible effects
consume continuation
commit

perform defined post-commit protocol/session response
```

This avoids holding SQLite locks during JavaScript or network calls. It also
prevents a timed-out callback from leaving an ambiguous transaction.

### 13.3 Named atomic operations

Generic `Update(func(TxStore))` remains an internal implementation tool. The
workflow layer should invoke named atomic operations such as:

```go
type SignupCommitter interface {
    CommitSignup(context.Context, SignupCommitRequest) (SignupCommitResult, error)
}

type SignupCommitRequest struct {
    ContinuationHash []byte
    ProgramHash      []byte
    IdentityPlan     IdentityPlan
    CredentialPlan   CredentialPlan
    InviteEvidence   *InviteEvidence
    SessionPlan      SessionPlan
    At               time.Time
}
```

The implementation may use `idpstore.AtomicStore.Update`, but the public
operation encodes the invariant: compatible identity creation, credential
attachment, invitation consumption, and continuation consumption succeed or
fail together.

## 14. Continuation storage and generation semantics

### 14.1 Separate workflow continuation from required-action bits

The current `InteractionRequiredAction` bitset is useful for today's fixed
login, consent, account-selection, and registration paths. It cannot represent
an arbitrary handler name, schema, carry value, native evidence, or graph
generation.

Add a versioned workflow continuation record or extend interaction storage with
a typed workflow payload. Prefer a separate `WorkflowContinuationStore` if
doing so keeps existing protocol interaction invariants easier to review.

### 14.2 Generation pinning

Every continuation records the program/source hash and workflow version. On
resume:

- If the exact generation remains available, route to it.
- If a declared native migration exists, migrate the data and record the event.
- Otherwise terminate safely and ask the user to restart the interaction.

Never resume a continuation against a new lambda merely because it has the same
string name.

### 14.3 Retention

The generation manager retains old callback registries only while compatible
unexpired continuations reference them, subject to an operator limit. Reload
must fail or explicitly invalidate old continuations if retaining another
generation would exceed the bound.

Continuation cleanup deletes expired records, pending secrets, and native
challenge state together. Audit entries retain only stable IDs and terminal
reasons.

## 15. Runtime generation and worker pool

### 15.1 Compile and activate

```text
read bounded source
compile with only require("tinyidp")
collect program definition and lambda metadata
validate graph, schemas, effects, and continuation edges
compile source into reusable Goja program
create N owned runtimes
load exact source in every runtime
verify callback registry fingerprint in every runtime
bind host capability descriptors
run embedded tests
warm readiness probes
atomically activate generation
```

### 15.2 Worker ownership

One worker is used by one invocation at a time. It may await bounded native
Promises during that request, but it is released only after the top-level
handler Promise settles and output is copied.

On timeout, panic, invalid output, late Promise settlement, capability contract
failure, or interrupt uncertainty, discard and replace the worker. Do not return
an uncertain VM to the pool.

### 15.3 Invocation bindings

`ctx` is created per invocation. Capability bindings are closures over the
request context and a call-budget object. After invocation, the host invalidates
them. Retaining `ctx` in a script global does not retain authority:

```javascript
let leaked;

program.lambda("bad", {
  run(ctx) {
    leaked = ctx;
    return A.decision.allow();
  },
});

// Later use fails: invocation binding expired.
```

### 15.4 Fail-closed behavior

| Failure | Native response | Worker |
|---|---|---|
| Lambda returns `deny` | Stable OAuth/browser denial | Reuse |
| Lambda throws | Server error; no effects | Discard |
| Lambda timeout | Server error; no effects | Interrupt and discard |
| Pool saturation | Service unavailable/server error | None acquired |
| Unknown handler | Activation/resume error | N/A |
| Invalid output | Server error; no effects | Discard |
| Missing capability | Activation failure | N/A |
| Capability timeout | Server error; no effects | Discard |
| Commit conflict | Safe retry/restart according to native policy | Reuse |

There is no production fallback from a failed custom policy to implicit allow.

## 16. Static analysis, model checking, and traces

### 16.1 Treat lambda bodies as opaque constrained transitions

General JavaScript cannot be completely understood by a graph validator. The
system instead analyzes the contract:

```text
handler signup.submitted
input      signupSubmissionV1
outcomes   present | challenge | deny | error
effects    read
caps       community.lookup@1, invite.verifySigned@1
next       signup.submitted, signup.emailVerified
budget     25 ms, 3 calls, 8 KiB output
```

Static analysis can prove that this handler cannot issue a token, mutate a
store, establish a session, or return an undeclared continuation. A separate
JavaScript linter may inspect source for API misuse, but kernel safety does not
depend on proving arbitrary source semantics.

### 16.2 Model checking

The model represents the lambda as nondeterministic over its declared outcomes.
If `signup.submitted` may return `challenge`, `deny`, or `error`, the checker
explores all three and verifies that no path reaches `issue.authorizationCode`
without required native evidence and a successful commit.

This is conservative: a real lambda may never choose one declared branch, but
the workflow must remain safe if it does.

### 16.3 Runtime trace

Every invocation emits bounded secret-free observations:

```json
{
  "kind": "lambda.completed",
  "generation": "sha256:...",
  "workflow": "signup",
  "handler": "signup.submitted",
  "inputSchema": "signupSubmissionV1",
  "outcome": "challenge",
  "nextHandler": "signup.emailVerified",
  "capabilityCalls": 2,
  "durationMs": 7
}
```

Subjects, emails, invite codes, passwords, arbitrary exception text, and
unbounded callback labels are not metric dimensions. Audit events may contain
stable subject IDs after identity establishment, according to existing audit
policy.

## 17. Public JavaScript API reference

### 17.1 Module

```typescript
interface TinyIDPModule {
  readonly v1: TinyIDPV1;
}

interface TinyIDPV1 {
  program(name: string, define: (program: ProgramBuilder) => void): Program;
  lambda<I, O>(id: string, spec: LambdaSpec<I, O>): Lambda<I, O>;
  readonly field: FieldBuilders;
  readonly action: ActionBuilders;
  readonly result: ResultBuilders;
  readonly identity: IdentityBuilders;
  readonly subject: SubjectBuilders;
  readonly invite: InviteBuilders;
  readonly error: ErrorBuilders;
}
```

### 17.2 Program builder

```typescript
interface ProgramBuilder {
  capabilities(requirements: Record<string, CapabilityRequirement>): void;
  client(id: string, spec: ClientSpec): void;
  workflow(name: string, handlers: WorkflowHandlers): void;
  provider(kind: ProviderKind, name: string, handlers: ProviderHandlers): void;
  lambda<I, O>(id: string, spec: LambdaSpec<I, O>): Lambda<I, O>;
  test(name: string, spec: PolicyTestSpec): void;
}
```

### 17.3 Handler context

```typescript
interface HandlerContext<I> {
  readonly invocation: InvocationView;
  readonly client: ClientView;
  readonly request: RequestView;
  readonly browser: BrowserView;
  readonly input: I;
  readonly carry: Readonly<Record<string, unknown>>;
  readonly evidence: EvidenceView;
  readonly presentation: PresentationView;
  readonly cap: CapabilityBindings;
  readonly present: PresentationOutcomes;
  readonly challenge: ChallengeOutcomes;
  readonly commit: CommitOutcomes;
  readonly complete: CompletionOutcomes;
  deny(code: string): DenyOutcome;
  skip(code?: string): SkipOutcome;
}
```

There is no `ctx.store`, `ctx.sql`, `ctx.fetch`, `ctx.fs`, `ctx.oauth`,
`ctx.tokens`, or `ctx.signingKey`.

### 17.4 Continuation requirements

Every `present` or `challenge` outcome must specify:

- a handler name registered in the same workflow;
- a compatible next-handler input schema;
- bounded serializable carry data;
- declared native evidence produced before resume;
- an expiry no longer than the host maximum;
- no secret value outside a permitted native secret reference.

The compiler validates static handler names. The runtime validates dynamic
selection against the compiled allowed-edge set.

## 18. Go package and file plan

### 18.1 New packages

```text
pkg/idpprogram/
  program.go             serializable program and registry contracts
  workflow.go            workflows, handler specs, continuation edges
  lambda.go              lambda metadata and bounded schemas
  outcomes.go            typed outcome and effect plan DTOs
  capabilities.go        capability requirements and effects
  validate.go            deterministic validation passes
  canonical.go           canonical JSON and source/program hashes

pkg/idpworkflow/
  executor.go            native event-to-handler execution loop
  continuation.go        continuation service and generation routing
  projector.go           immutable JS input projection
  codec.go               output/outcome/schema validation
  effects.go             effect validation and commit dispatch
  generation.go          activation, retention, draining, rollback

pkg/idpscript/
  compiler.go            isolated source compiler and artifact
  runtime_factory.go     explicit tinyidp-only runtime factory
  registry.go            deterministic callback registration/fingerprint
  pool.go                owned worker pool and replacement
  invoke.go              Promise-aware bounded invocation
  secrets.go             invocation secret handles
  capabilities.go        invocation capability binding

internal/gojamodules/tinyidp/
  module.go              require("tinyidp") registration
  program.go             ProgramBuilder projection
  lambda.go              lambda registration and metadata
  ui.go                  field/action/presentation builders
  outcomes.go            outcome builders
  typescript.go          generated/maintained declarations

pkg/idpcontinuation/
  types.go               continuation and native reference types
  store.go               narrow persistence contract
  service.go             create/load/advance/consume invariants
  testsuite.go            memory and SQLite conformance suite
```

Names may move under `internal/` until two consumers stabilize them. Contracts
used by embedders should remain runtime-independent.

### 18.2 Existing files to change

| File | Change |
|---|---|
| `internal/fositeadapter/provider.go` | Replace the hardcoded registration branch with workflow start/resume integration while preserving native OAuth checks. |
| `internal/fositeadapter/interaction.go` | Share canonical request and binding helpers with workflow continuations. |
| `pkg/idpstore/types.go` | Add or reference versioned continuation, evidence, and terminal outcome types. |
| `pkg/idpstore/interfaces.go` | Add narrow continuation operations and named signup commit operation. |
| `pkg/sqlitestore/migrations/*` | Add continuation, challenge, pending-secret, and optional invite tables by phase. |
| `pkg/idpui/types.go` | Generalize registration presentation into validated typed fields without exposing HTTP authority. |
| `pkg/idpui/renderer.go` | Add generic workflow presentation renderer contract if one model cannot remain compatible. |
| `pkg/embeddedidp/options.go` | Accept an activated program/executor and validate production readiness. |
| `internal/gojaverify/compiler.go` | Reuse isolation patterns; do not merge production and verification modules. |
| `internal/cmds/serve_production.go` | Add script artifact/resource binding without shifting listener or secret ownership into JS. |

## 19. Implementation phases

### Phase 0: freeze contracts with a no-browser spike

Purpose: prove lambda registration, owned invocation, Promise settlement, and
capability isolation before persistence changes.

Tasks:

1. Define pure-Go `Program`, `Workflow`, `LambdaSpec`, schemas, outcomes, and
   validation diagnostics.
2. Implement `require("tinyidp").v1` with `program`, `workflow`, and `lambda`.
3. Compile a source and verify deterministic callback fingerprints across at
   least two owned runtimes.
4. Invoke a pure lambda and a Promise-returning capability lambda.
5. Reject all ambient modules and undeclared capabilities.
6. Implement timeout interruption, late-settlement tests, and worker discard.
7. Add TypeScript declarations for the spike surface.

Gate: two runtime workers load the same callback registry, concurrent calls pass
under the race detector, forbidden modules fail, and a timed-out worker is
never reused.

### Phase 1: explicit continuation domain

Purpose: create a durable state machine independent of Fosite response code.

Tasks:

1. Define `WorkflowContinuation`, evidence references, secret references, and
   terminal outcomes.
2. Add memory and SQLite stores with keyed handle hashes, expiry, generation
   binding, one-use advancement, and atomic consume.
3. Add store conformance, replay, expiry, conflict, and concurrent-advance tests.
4. Add generation lookup and safe incompatible-generation termination.
5. Add maintenance cleanup for continuations and attached native state.

Gate: restartable tests create a continuation in one service instance and
resume it in another without retaining any Goja object.

### Phase 2: generic provider-owned presentation

Purpose: let handlers select safe forms without granting HTTP authority.

Tasks:

1. Define field and action descriptor registries.
2. Generalize `InteractionPage` or add `WorkflowPage`.
3. Implement `ctx.present.*` outcome builders.
4. Validate exact POST field sets, multiplicity, normalization, sensitive
   handles, public values, and errors.
5. Preserve current CSP, header, CSRF, origin, browser-binding, and renderer
   tests.

Gate: a script can present and rerender the current signup form, and the browser
smoke test completes the existing account-creation flow.

### Phase 3: signup workflow vertical slice

Purpose: replace hardcoded provider registration with named lambdas while
preserving behavior.

Tasks:

1. Route validated `tinyidp_signup=1` requests into `signup.start`.
2. Invoke `signup.submitted` on the validated POST.
3. Add secret handles and native password-credential effects.
4. Add `SignupCommitter` for atomic identity, credential, continuation, and
   session effects.
5. Express current open signup entirely in a checked-in JS program.
6. Differential-test old and new flows until the new flow replaces the old
   implementation; do not retain a compatibility adapter after replacement.

Gate: existing PKCE registration, replay, CSRF, origin, duplicate-login,
password-policy, consent, session, audit, and callback tests pass through the
scripted workflow.

### Phase 4: virtual identity and invite providers

Purpose: demonstrate that resources need not be stored or hardcoded.

Tasks:

1. Add identity and invite provider contracts.
2. Add virtual identity subject derivation and validated claim projection.
3. Add signed stateless invitation verification.
4. Add a capability-backed computed invitation example.
5. Add a durable one-time invitation provider and atomic redemption evidence.
6. Add explanation output showing state/replay/revocation consequences.

Gate: tests cover open signup, email-domain policy, stateless signed invites,
computed virtual eligibility, durable one-time invites, virtual identities, and
local stored accounts using the same workflow API.

### Phase 5: email challenge and multi-request signup

Purpose: exercise explicit continuations with native verified evidence.

Tasks:

1. Implement pending email challenge records and mailer capability.
2. Add email-code presentation and submission schemas.
3. Produce unforgeable verified-email evidence on atomic code consumption.
4. Add attempt limits, resend policy, expiry, cleanup, and generic public errors.
5. Ensure passwords are collected after verification or converted to safe
   pending credential state before suspension.

Gate: a process-restart integration test resumes email signup, while replay,
wrong-code, expiry, browser mismatch, and graph-generation mismatch fail safely.

### Phase 6: activation, tests, explain, and operations

Purpose: make programs safe to deploy and change.

Tasks:

1. Add `tinyidp script validate`, `test`, and `explain`.
2. Run embedded lambda tests with fake capabilities.
3. Warm worker pools and verify fingerprints before atomic activation.
4. Retain bounded old generations for compatible continuations.
5. Add readiness, saturation, invocation, continuation, and generation metrics.
6. Audit activation with source/program hashes and redacted diagnostics.

Gate: a failed compile, test, capability bind, pool warmup, or readiness probe
leaves the previous generation active.

### Phase 7: authorization, claims, and additional workflows

Purpose: apply the lambda-first contracts beyond signup.

Tasks:

1. Add authorization lambdas after native authentication and request validation.
2. Add claims lambdas before OIDC session persistence with protected claims.
3. Express account selection and consent as workflow handlers where useful.
4. Add password recovery only after its native credential and challenge effects
   exist.
5. Add device-verification presentation integration without moving RFC 8628
   state transitions into JS.

Gate: strict conformance, refresh/UserInfo consistency, device flow, and current
security-model tests remain intact.

## 20. Testing strategy

### 20.1 Pure contract tests

- Canonical program serialization and hashing are deterministic.
- Callback IDs, continuation edges, schemas, capabilities, and effects are
  unique and valid.
- Every presentation/challenge edge reaches a registered compatible handler.
- Every handler path can terminate in an allowed outcome.
- Production profiles reject forbidden modules, capabilities, field types, and
  effects.

### 20.2 Runtime tests

- VM access occurs only on the owner.
- Multiple workers register the same fingerprint.
- Promise capabilities settle on the owner with the invocation context.
- Request cancellation cancels capability calls.
- Timeouts interrupt and discard workers.
- Late Promise settlements cannot affect a reused worker.
- Retained globals cannot reuse expired capability bindings.
- Saturation fails closed and remains observable.

### 20.3 Continuation store tests

- Raw handles never enter storage, audit, or metrics.
- Advance and terminal consume are atomic and one-use.
- Concurrent POSTs yield exactly one accepted transition.
- Expired, browser-mismatched, client-changed, and generation-mismatched records
  fail safely.
- Restart does not require a Goja heap.
- Cleanup removes attached pending secrets and challenge state.

### 20.4 Browser and protocol tests

- Initial signup begins only after Fosite validates client and redirect URI.
- CSRF, origin, body-size, action, and field-set checks happen before lambda
  invocation.
- Invalid invite/email/account responses do not leak existence information.
- Successful completion returns to the exact registered callback and passes
  PKCE code exchange.
- Consent, session creation, ID token, UserInfo, logout, and account chooser
  remain correct.

### 20.5 Property, fuzz, and model tests

- Fuzz field normalization, outcome decoding, carry serialization, and
  continuation codecs.
- Model continuation advance/consume for replay and concurrency.
- Treat lambdas nondeterministically over declared outcomes and prove issuance
  requires native evidence and commit.
- Replay normalized runtime traces through the assurance model.
- Race-test pool ownership, generation swap/drain, and store transitions.

## 21. Decision records

### Decision: JavaScript is an execution language, not only graph syntax

- **Context:** The original design made most workflows predefined nodes and
  limited runtime lambdas to authorization and claims.
- **Options considered:** Rich declarative configuration; unrestricted scripts;
  typed lambda-first programs.
- **Decision:** Use typed named lambdas as primary application behavior, within
  declared capability/effect/outcome contracts.
- **Rationale:** This supports virtual resources and application-specific
  workflows without adding Go modes while preserving enforceable authority.
- **Consequences:** Static analysis reasons about lambda contracts rather than
  arbitrary bodies; runtime testing, budgets, and tracing become essential.
- **Status:** accepted

### Decision: browser boundaries use explicit named continuations

- **Context:** Browser forms and email verification cross HTTP requests and may
  outlive processes and program reloads.
- **Options considered:** Park Goja Promises; replay workflow history;
  source-transform async functions; explicit handler continuations.
- **Decision:** V1 uses explicit `resume` handler names and persisted native
  continuation records.
- **Rationale:** It is restartable, bounded, inspectable, and implementable with
  the existing interaction model.
- **Consequences:** Authors write handler maps; future syntax sugar must compile
  to the same contract.
- **Status:** accepted

### Decision: bounded capability calls may use ordinary Promises

- **Context:** Application membership, external directories, and notifications
  perform I/O during one handler invocation.
- **Options considered:** Synchronous-only calls; general ambient async I/O;
  typed Promise-returning capabilities.
- **Decision:** Permit Promises only from declared capabilities with invocation
  and operation deadlines.
- **Rationale:** It uses existing go-go-goja owner/Promise patterns while
  preserving explicit authority.
- **Consequences:** Top-level handler settlement and late-settlement cleanup need
  rigorous tests.
- **Status:** accepted

### Decision: important mutations are native effect plans

- **Context:** Lambdas need to choose what happens, but JavaScript execution
  inside SQL transactions creates lock, timeout, and atomicity hazards.
- **Options considered:** Expose store mutations; execute transaction callbacks
  in JS; return typed effect plans.
- **Decision:** Lambdas return effect plans; named native operations revalidate
  and commit them atomically after invocation.
- **Rationale:** This separates programmable intent from irreversible authority.
- **Consequences:** Effect vocabulary must be sufficient and versioned; adding a
  truly new privileged primitive still requires Go.
- **Status:** accepted

### Decision: virtual resources are first-class providers

- **Context:** Users and invites may be derived, signed, external, or computed
  and should not require local rows.
- **Options considered:** Require store interfaces for all resources; hardcode
  provider modes; type providers uniformly across storage models.
- **Decision:** Define typed provider contracts implemented by lambdas or native
  adapters, with explicit state/replay semantics.
- **Rationale:** The workflow can depend on identity/invite behavior rather than
  storage representation.
- **Consequences:** Operators must understand that stateless invites cannot
  provide one-time use or immediate revocation without additional state.
- **Status:** accepted

### Decision: keep verification scripting separate

- **Context:** Verification needs fake clocks, failpoints, store inspection, and
  assertions that production lambdas must not possess.
- **Options considered:** One universal runtime; shared modules with flags;
  separate compile-only verification plane.
- **Decision:** Preserve `tinyidp/verify` and its native runner as a separate
  authority profile.
- **Rationale:** Test authority cannot accidentally enter request processing.
- **Consequences:** Program tests may call lambda contracts with fakes, while
  multi-request security scenarios remain verification plans.
- **Status:** accepted

## 22. Alternatives considered

### Rich configuration DSL only

This keeps runtime behavior simple but recreates a mode taxonomy in JavaScript.
Every novel workflow eventually requires another builder and native node. It is
retained for common field/outcome/effect constructors, not as the primary
behavior model.

### Unrestricted JavaScript service

Giving scripts HTTP, SQL, crypto, and filesystem modules makes application code
easy to write but moves identity invariants into unreviewed ambient effects. It
also makes capability analysis and safe deployment profiles ineffective. This
is rejected for the in-process production runtime.

### Suspended Goja VM per browser interaction

This enables literal browser-spanning `await`, but makes the VM heap the
continuation store. Restart, scaling, reload, memory pressure, abandoned flows,
and secret retention become much harder. It may remain a development experiment
but is not the v1 production contract.

### Deterministic event-history replay

Re-executing an async function from the beginning can preserve pleasant syntax,
but requires deterministic APIs, recorded effects, versioning, replay-safe
exceptions, and a durable workflow engine. It is a possible later authoring
layer after explicit continuations are stable.

### Put every resource in SQLite

This simplifies atomicity but makes identity and invitation semantics equal to
one storage implementation. It conflicts with signed invitations, upstream
identity, verified-email virtual users, and application capabilities. SQLite
remains the default protocol/continuation store and an optional identity/invite
provider.

## 23. Risks and open questions

1. **Program API size.** The builders must not grow into a second web framework.
   Keep HTTP authority and generic I/O out; keep workflow inputs and outcomes
   specific to identity operations.
2. **Promise interruption.** Context cancellation alone does not stop executing
   JavaScript. The worker interruption and late-settlement protocol requires a
   dedicated spike and race tests.
3. **Generation retention.** Long email challenges can retain old generations.
   Define maximum generations, challenge TTL, reload refusal/invalidation, and
   operator visibility before hot reload ships.
4. **Pending password material.** Prefer collecting passwords after long
   challenges. If not, define encrypted or already-hashed pending credentials
   and destruction guarantees before implementation.
5. **Virtual-user disable semantics.** A virtual provider must define where
   disabled/revoked status lives and how quickly it is observed.
6. **Capability consistency.** A lambda may read external state that changes
   before commit. Capability results that authorize durable effects need opaque
   versions or short-lived evidence revalidated by the native committer.
7. **Renderer generalization.** Determine whether `InteractionPage` can evolve
   without becoming a loose map. Prefer registered typed fields and prompts.
8. **Continuation schema location.** Compare extending `InteractionRecord` with
   introducing a separate store. Choose the design with the clearest atomic and
   migration story after a focused implementation spike.
9. **Multiple clients.** Per-client workflow selection is expected, but account
   namespace and subject derivation rules must be explicit when identities are
   shared across clients.
10. **Source trust.** GitOps review and signed program artifacts may eventually
    be required. The runtime sandbox is not a replacement for deployment
    authorization.

## 24. Intern implementation checklist

Before coding:

- Read `provider.go` from `beginAuthorize` through `resumeAuthorize` and
  `finishAuthorize`.
- Read `interaction.go` and the interaction store tests.
- Read `idpui` models and renderer contracts.
- Read `internal/gojaverify/compiler.go` and its forbidden-module/timeout tests.
- Read the go-go-goja runtime factory, runtime owner, and async patterns guide.
- Run `go test ./... -count=1` from `tiny-idp` and record the baseline.

For every phase:

- Start with pure Go DTOs and validators.
- Write memory and SQLite contract tests together.
- Use `var _ Interface = (*Implementation)(nil)` for every implementation.
- Keep contexts on blocking or request-scoped operations.
- Never call a Goja VM outside its owner.
- Never expose a raw secret or host object in JS DTOs.
- Never invoke JavaScript while holding a transaction.
- Treat exception, timeout, invalid output, and missing capability as fail-closed
  infrastructure errors.
- Add trace and audit coverage with bounded identifiers.
- Run focused tests, `go test ./...`, race tests for concurrent components, and
  the existing security/conformance gates.

## 25. Review guide

Review the proposed implementation in this order:

1. **Contracts:** `idpprogram` schemas, lambda metadata, outcomes, effects, and
   validation. Verify that illegal authority cannot be expressed.
2. **Continuation store:** handle hashing, binding, expiry, generation pinning,
   atomic advance/consume, and restart tests.
3. **Runtime factory:** explicit modules, callback fingerprinting, owner-only
   access, Promise settlement, interruption, and worker replacement.
4. **Boundary codecs:** input projection, secret handles, output bounds, evidence
   provenance, and capability expiry.
5. **Effect commit:** no JavaScript inside transactions, authoritative
   revalidation, and atomic terminal state.
6. **Fosite integration:** native OAuth validation still precedes workflow
   invocation; only a validated request can be redirected with an OAuth error or
   successful code.
7. **UI:** scripts cannot select form actions, headers, cookies, redirects, or
   raw HTML authority.
8. **Operations:** generation activation, retention, readiness, metrics, cleanup,
   audit, and safe rollback.

## 26. Key points

- JavaScript is executable identity behavior, not just a richer configuration
  file.
- Lambdas are named, typed, bounded, capability-scoped transitions.
- Ordinary `await` is supported for bounded capability work inside one HTTP
  request.
- Browser waits return explicit presentation or challenge outcomes and resume at
  named handlers through native continuation records.
- Virtual users and virtual invitations are first-class; durable storage is
  selected only when its semantics are needed.
- Secrets remain opaque handles and cryptographic/credential operations remain
  native.
- Lambdas return effect plans; Go revalidates and commits irreversible changes
  atomically.
- Static analysis constrains lambda authority, model checking explores declared
  outcomes, and runtime tracing records what actually occurred.
- The current provider, interaction, UI, store, and Goja ownership code provide
  useful foundations; the hardcoded registration branch is the first migration
  target.

## 27. References

- `internal/fositeadapter/provider.go` — validated authorization, login,
  registration, consent, and issuance flow.
- `internal/fositeadapter/interaction.go` — current interaction creation,
  browser binding, canonical request digest, and reconstruction.
- `pkg/idpstore/types.go` — interaction, session, user, claims, device, and
  security-state types.
- `pkg/idpstore/interfaces.go` — persistence and named atomic operation
  contracts.
- `pkg/idpui/types.go` and `pkg/idpui/renderer.go` — provider-owned
  presentation boundary.
- `pkg/idpaccounts/accounts.go` — current atomic local account and credential
  creation service.
- `pkg/sqlitestore/store.go` and `internal/store/memory/store.go` — transaction
  implementations and test targets.
- `internal/gojaverify/compiler.go` — current isolated compile-only Goja
  implementation.
- `internal/gojamodules/verify/module.go` — current explicit native module and
  plain-data normalization pattern.
- `go-go-goja/pkg/engine/factory.go` — runtime creation, explicit module
  selection, event loop, owner, and runtime lifecycle.
- `go-go-goja/pkg/runtimeowner/runner.go` — serialized `Call` and `Post` access.
- `go-go-goja/pkg/doc/03-async-patterns.md` — context and Promise settlement
  conventions.
- `design-doc/02-assurance-oriented-core-grammar-and-codebase-refactoring-proposal.md`
  — stable assurance vocabulary and refactoring sequence.
- `reference/02-security-verification-scripting-plane-assessment.md` — separate
  production and verification authority profiles.
