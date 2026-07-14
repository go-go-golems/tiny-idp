---
Title: Investigation and Implementation Diary
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - architecture
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/embedded/app.go
      Note: PKCE browser flow and application sessions
    - Path: examples/embedded/app_test.go
      Note: Complete relying-party callback harness
    - Path: examples/embedded/main.go
      Note: One-origin IdP and relying-party composition
    - Path: examples/embedded/oidc.go
      Note: Back-channel exchange and ID-token verification
ExternalSources:
    - sources/rfc-8628-oauth-device-authorization-grant.md
    - sources/rfc-9700-oauth-security-bcp.md
Summary: Chronological evidence log for the production device design and self-contained embedded relying-party implementation.
LastUpdated: 2026-07-14T18:25:00Z
WhatFor: Preserves commands, findings, mistakes, decisions, and validation so another engineer can continue without reconstructing the investigation.
WhenToUse: Read at handoff, before implementing a task, or when reviewing why the design chose durable atomic state and Fosite integration.
---


# Investigation and Implementation Diary

## 2026-07-14 13:45 EDT — Request decomposition

### Goal

The request contained two work streams:

1. create a durable ticket and intern-facing design for production RFC 8628
   device authorization in the strict provider;
2. correct `examples/embedded`, which registered a callback on a different
   application origin without implementing that relying party.

I treated the first as research and design, not authorization to implement the
large security feature. I treated the second as an explicit implementation
request. This separation prevents an incomplete device grant from being
advertised while still producing a complete runnable embedding example.

### Workspace state

Repository: `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp`

Branch: `task/prod-tiny-idp`

The worktree already contained two unrelated untracked OIDF source directories
under `TINYIDP-PROD-001`. They were not read, edited, staged, or removed.

## 2026-07-14 13:55 EDT — Ticket creation

Created ticket `TINYIDP-DEVICE-PROD-001`, titled **Production Device
Authorization for the Strict Embedded Provider**.

Created:

- `design-doc/01-production-device-authorization-analysis-design-and-implementation-guide.md`;
- `reference/01-investigation-and-implementation-diary.md`;
- standard ticket index, task, changelog, and README files.

The initial generated documents contained templates. I kept them only long
enough to inspect the expected frontmatter, then replaced the bodies with the
design and this diary.

## 2026-07-14 14:00 EDT — Primary-source collection

I used Defuddle because the user explicitly requested retraceable sources and
the workflow calls for cleaned Markdown under `sources/`.

Preserved:

- RFC 8628, OAuth 2.0 Device Authorization Grant;
- RFC 9700, Best Current Practice for OAuth 2.0 Security;
- RFC 8414, OAuth 2.0 Authorization Server Metadata;
- OpenID Connect Discovery 1.0.

The first attempt to parse the RFC Editor presentation of RFC 8628 failed with
`No content could be extracted`. I retried using the IETF Datatracker HTML
publication, which Defuddle successfully converted. This failure matters
because it explains why the saved source URL differs from the first URL tried.

Key requirements extracted from RFC 8628:

- the authorization server exposes a device authorization endpoint;
- the response returns device code, user code, verification URI, expiry, and
  optionally complete verification URI and polling interval;
- polling before approval returns `authorization_pending`;
- overly frequent polling returns `slow_down` and adds five seconds to the
  polling interval;
- denial and expiry are terminal;
- user-code brute force requires finite lifetime and rate limiting;
- device codes require high entropy;
- complete verification URIs do not remove the need to show and confirm the
  code because remote phishing remains relevant.

## 2026-07-14 14:05 EDT — Existing device code inspection

Inspected `internal/server/device.go`, `internal/server/token.go`, discovery,
tests, and client fixtures.

Findings:

- The mock server already models pending, approved, and denied grants.
- It returns appropriate pending, slowdown, expiry, denial, and invalid-grant
  errors for development tests.
- It uses an eight-character human code with an ambiguity-reduced alphabet.
- State is a process-local `deviceGrants` map.
- Raw codes remain in memory and user-code lookup scans the map.
- Decisions do not participate in a durable token transaction.
- Authentication uses scenario users rather than production accounts.
- Empty verification action behavior is too permissive for the strict flow.

Conclusion: retain it as a mock and source of test cases; do not promote it into
`internal/fositeadapter`.

## 2026-07-14 14:10 EDT — Strict-provider extension analysis

Inspected:

- `internal/fositeadapter/provider.go` route and factory assembly;
- `internal/fositeadapter/sqlstore.go` token/client persistence;
- Fosite v0.49.0 `TokenEndpointHandler`;
- Fosite authorization-code, refresh, and OpenID handlers;
- `pkg/idpstore/interfaces.go` and `types.go`;
- SQLite migrations, maintenance, backup, and verification;
- `pkg/embeddedidp/bootstrap.go` and `options.go`.

Fosite has no built-in RFC 8628 handler in this version. It does provide the
correct extension interface. `CanSkipClientAuth` must remain false: public
client identification is not equivalent to omitting client identity.

The public store already encourages named transitions through transactional
interfaces. That pattern led directly to the design for `PollDeviceGrant`,
`DecideDeviceGrant`, and `ConsumeDeviceGrant` instead of a callback-based
generic update.

The central atomicity problem found during this step was:

```text
consume grant, then fail token writes  => approved grant is lost
write tokens, then fail grant consume  => one device code can replay
```

Therefore consumption and Fosite token persistence must share one SQLite
transaction. A standalone token endpoint was rejected because it would create
a second token format and lifecycle.

## 2026-07-14 14:25 EDT — Embedded example gap

The old `examples/embedded/main.go`:

- served only `provider.Handler()` on `127.0.0.1:5556`;
- registered `http://127.0.0.1:8080/auth/callback`;
- implemented nothing on port 8080;
- demonstrated discovery through an in-process transport only in a separate
  package example.

This meant a reader could start the example and view an IdP, but could not run
the registered OIDC application.

Chosen composition:

```text
one process / one public origin: http://127.0.0.1:5556
    /                    relying-party home
    /login               begin Authorization Code + PKCE
    /auth/callback       implemented callback
    /logout              local + RP-initiated logout
    /idp/*               embedded tiny-idp provider
```

Browser redirects use normal HTTP paths. Discovery, token exchange, JWKS, and
UserInfo are dispatched by `NewInProcessIssuerTransport`, which admits only the
exact issuer origin and has no network fallback.

## 2026-07-14 14:35 EDT — Relying-party implementation

Replaced `examples/embedded/main.go` and added:

- `app.go`: handlers, PKCE/state/nonce generation, transient login flows,
  application sessions, CSRF-protected logout, and HTML;
- `oidc.go`: discovery, code exchange, bounded JSON, RS256/JWKS verification,
  issuer/audience/expiry/nonce checks, and UserInfo retrieval;
- `app_test.go`: a complete callback harness with a generated RSA key and fake
  exact endpoint transport.

Security properties implemented in the example:

- 256-bit random state and nonce;
- high-entropy PKCE verifier and S256 challenge;
- opaque HttpOnly SameSite application cookies;
- one-time, five-minute login-flow consumption;
- exact state check before back-channel activity;
- bounded one-megabyte OIDC responses;
- discovery issuer equality;
- JWT compact structure, `alg=RS256`, `kid`, JWK type/use/algorithm checks;
- RSA signature, issuer, subject, audience, expiration, issued-at, and nonce
  validation;
- UserInfo subject equality with the ID token;
- independent eight-hour application session;
- POST + CSRF logout and RP-initiated IdP logout;
- CSP, no-referrer, and nosniff response headers.

This is intentionally still a development example. Its session store is
in-memory and its public URL is loopback HTTP. The production xapp demonstrates
durable application sessions and HTTPS validation.

## 2026-07-14 14:50 EDT — Build and test corrections

First test command:

```text
gofmt -w examples/embedded/main.go examples/embedded/app.go examples/embedded/oidc.go
go test ./examples/embedded
go test ./pkg/embeddedidp ./internal/fositeadapter
```

Observed issues:

1. A constant expression used to guard conversion of the JWK exponent
   overflowed `uint32`. Replaced it with the explicit `int32` upper bound.
2. Broader packages attempted to create `httptest` listeners, which the
   filesystem/network sandbox prohibited. This was an environment restriction,
   not a product failure.
3. The shared Go build cache was read-only inside the sandbox. Retried the
   example test with approved normal cache access, per the repository's
   `go.work` guidance.
4. The new test initially assigned a string to a byte slice while trimming
   leading exponent zeroes. Replaced it with an in-place slice loop.
5. The deterministic JWT test timestamp was in the future relative to the
   verifier's real clock. Aligned the fixture to current UTC time.

Final focused result:

```text
ok github.com/manuel/tinyidp/examples/embedded
```

The callback test also proves a state-mismatch consumes the transient flow and
does not initiate a back-channel request.

## 2026-07-14 15:05 EDT — Real-process smoke test

Per repository instructions, attempted to start the server in tmux. The first
sandboxed attempt failed:

```text
error connecting to /tmp/tmux-1000/default (Operation not permitted)
```

Retried with approval and started session `tinyidp-embedded-smoke`.
Captured output:

```text
self-contained app listening at http://127.0.0.1:5556
```

Loopback checks established:

- `GET /` returned 200 and the relying-party home page;
- `GET /login` returned 302 to `/idp/authorize` on the same origin;
- the authorization request included exact callback, state, nonce, and S256
  challenge;
- the IdP rendered its login/consent form;
- submitting `alice / correct horse battery staple` completed the real code
  exchange and established the application session;
- authenticated `/` displayed `Alice Example`, email, and the stable subject.

One curl invocation explicitly set `-X POST` while following redirects. Curl
therefore retained POST at the final `/` redirect and received `Method Not
Allowed`. This was a smoke-driver mistake, not an application failure. The
cookie jar already contained the successfully established session; a normal
GET verified it.

Stopped the process with the required command:

```text
lsof-who -p 5556 -k
```

## 2026-07-14 15:20 EDT — Design synthesis

Wrote the design from the primary specifications and repository evidence. The
design connects protocol requirements to concrete public types, Fosite
extension points, SQL constraints, transitions, rendering boundaries,
operational controls, and test harnesses. It includes phased work and a release
checklist but deliberately does not claim that device authorization is already
implemented.

## Review instructions

For the embedded example:

```bash
go test ./examples/embedded
go run ./examples/embedded
```

Open `http://127.0.0.1:5556`, sign in with the documented development account,
verify the application profile, then use the POST logout button. Confirm that a
new sign-in requires credentials after coordinated logout.

For the device ticket, begin with Phase 0 and Phase 1. Do not add discovery
metadata until durable creation, verification, polling, and token issuance are
complete behind the strict provider.

## 2026-07-14 15:35 EDT — Final validation, commits, and publication

The complete repository suite passed with normal loopback and shared-cache
access:

```text
go test ./...
```

Created two scoped commits with `LEFTHOOK=0`:

- `b2f31d3 feat(examples): make embedded host a complete OIDC app`
- `7d46268 docs(device): design production authorization grant`

The first sandboxed staging attempt could not create the linked-worktree
`index.lock` because the actual Git directory is outside the writable workspace.
Staging and committing were repeated with the required approval. Only the five
example files, the new ticket, and its vocabulary entry were staged. The two
pre-existing untracked OIDF directories remained untouched.

Docmgr frontmatter validation passed, and `docmgr doctor` reported all checks
passed after adding the `oauth2` topic vocabulary and notes for every related
file.

Uploaded one combined PDF containing the ticket index, design, diary, tasks,
and four preserved specifications:

```text
/ai/2026/07/14/TINYIDP-DEVICE-PROD-001/
TINYIDP Device Production Design Bundle.pdf
```

### Remaining implementation work

- Implement the device phases tracked in `tasks.md`.
- Keep device discovery advertisement disabled until the endpoint, browser
  verification, and atomic Fosite token issuance phases pass together.
- Obtain an independent security review before declaring RFC 8628 production
  ready.
