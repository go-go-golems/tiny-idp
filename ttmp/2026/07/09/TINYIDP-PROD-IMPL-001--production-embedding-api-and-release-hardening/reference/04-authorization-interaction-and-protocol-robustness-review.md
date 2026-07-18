---
Title: Authorization interaction and protocol robustness review
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - https://openid.net/specs/openid-connect-core-1_0.html
    - https://datatracker.ietf.org/doc/html/rfc6749
    - https://www.rfc-editor.org/rfc/rfc9700.html
    - https://datatracker.ietf.org/doc/html/rfc6750
Summary: Review of authorization interaction state, prompt/max-age enforcement, consent, token limiting, UserInfo, request objects, and Fosite persistence failure paths, with a prioritized regression and fault-testing plan.
LastUpdated: 2026-07-10T11:16:00.175060122-04:00
WhatFor: Preventing browser interaction state loss, forced-reauthentication bypass, invalid-request credential collection, limiter sharding, and partial OAuth protocol state before production approval.
WhenToUse: Use before changing the strict authorization handler, interaction forms, sessions, consent, token/UserInfo endpoints, Fosite storage, or the release conformance gate.
---

# Authorization interaction and protocol robustness review

## Goal

Identify paths analogous to the unhandled forced-reauthentication POST review
finding, rank them by release risk, and define tests that prove protocol and
security invariants rather than only checking response status codes.

## Context

The strict handler validates a GET authorization request, renders an HTML form,
then reconstructs a second authorization request from browser-submitted hidden
fields on POST. The known review defect is that a valid browser session plus an
empty login can complete the POST even when the original GET required active
reauthentication through `prompt=login` or expired `max_age`.

OpenID Connect Core requires active reauthentication for `prompt=login` and
when elapsed authentication age exceeds `max_age`. It also forbids combining
`prompt=none` with other prompt values, prohibits UI interaction for
`prompt=none`, requires an authorization decision before releasing claims, and
requires an OAuth error response when authentication or authorization is
denied. RFC 6749 requires validation before resource-owner authentication and
forbids redirecting to an invalid redirect URI.

The review inspected:

- `internal/fositeadapter/provider.go`;
- `internal/fositeadapter/session.go`, `csrf.go`, and `consent.go`;
- `internal/fositeadapter/sqlstore.go`;
- strict adapter tests and the external-consumer flow;
- Fosite v0.49.0 prompt/max-age validation and authorize handler ordering;
- the four captured primary sources under this ticket's `sources/` directory.

No production code was changed during this review. Findings labeled
**confirmed** follow directly from current control flow. Findings labeled
**test to confirm** have a plausible path that needs an executable regression
before severity is finalized.

## Quick Reference

### Release conclusion

The interaction continuation is the highest-leverage problem. Fixing the one
empty-login branch while continuing to trust reconstructed hidden fields would
leave the same class of defect available through other parameters and future
features. The preferred remediation is an opaque, expiring, one-time,
server-side interaction record. The browser should submit only the interaction
identifier, CSRF proof, credentials when required, and an explicit
approve/deny decision.

| Priority | Finding | Status | Required gate |
| --- | --- | --- | --- |
| P1 | Forced reauthentication can be bypassed on POST with a valid session and empty login. | Confirmed | Negative regression must fail before fix and pass after server-side required-action enforcement. |
| P1 | `max_age` parsing and error recovery fail open; any GET error with a nonempty `max_age` can enter the login UI branch. | Confirmed control flow; invalid-client/redirect variants need executable confirmation | Invalid/negative/overflow values never reuse a session or collect credentials; unrelated validation errors never become login forms. |
| P1 | GET-to-POST continuation loses and permits mutation of authorization parameters. | Confirmed | Immutable server-side request snapshot; mutation/replay/concurrent-tab tests. |
| P1 | Token endpoint limiting is keyed by untrusted form `client_id` plus address and can be sharded. | Confirmed | Address/global bucket plus normalized claimed/authenticated client dimensions; arbitrary IDs cannot increase attempts. |
| P1/P2 | Fosite authorize-code, PKCE, and OIDC storage writes are separate handler calls. | Confirmed architecture; consequence needs fault injection | Failure at every storage call leaves no redeemable or indefinitely inconsistent partial protocol state. |
| P2 | Consent denial returns a local 403 after creating a login session rather than an OAuth `access_denied` response. | Confirmed | Explicit approve/deny semantics; denial redirects only to a previously validated URI with original state and creates no consent/code. |
| P2 | UserInfo accepts Fosite's query-token path and lacks an explicit method/cache/challenge contract. | Confirmed by dependency and handler inspection | GET/header and supported POST/body pass; query token and unsupported methods fail; sensitive responses are non-cacheable. |
| P2 | Unsupported request-object error routing reads unverified JWT claims. | Confirmed | Conflict/malformed/signature-shaped corpus never creates an unvalidated redirect or trusts inner state/client data. |
| P2 | Session/store errors are collapsed into “no session.” | Confirmed | Store failure produces server/unavailable behavior, not `login_required` or a credential prompt indistinguishable from absence. |

## Detailed findings

### 1. Forced reauthentication is not a durable required action

**Problem:** GET decides whether login is required, but POST does not retain
that decision. A valid session makes an empty login acceptable regardless of
why the form was rendered.

**Where to look:** `Provider.authorize` at
`internal/fositeadapter/provider.go:340-438`, especially GET lines 356-379 and
POST lines 406-438.

```go
login := strings.ToLower(strings.TrimSpace(r.PostForm.Get("login")))
if login != "" {
    // authenticate and replace u/authTime
} else if !hasSession {
    return missingLogin
}
p.finishAuthorize(..., u, authTime, ...)
```

**Why it matters:** `prompt=login` is a client request for active
reauthentication, not merely for displaying a form. A stale `max_age` session
is also insufficient authentication. Reusing either session produces a code
and an ID Token with the old `auth_time`.

**Required test:** establish a session, request `prompt=login`, extract the
form and CSRF token, submit it with no login/password and the old session
cookie, disable redirect following, and assert:

- no redirect contains a code;
- response is a local interaction error or a new login form;
- no authorize-code, PKCE, or OIDC row was created;
- `auth_time` cannot remain the old value in any issued token.

Repeat for expired `max_age`, `max_age=0`, and a stale session with previously
stored consent.

**Remediation sketch:** required actions belong to a server-side interaction:

```text
interaction.required = {login, consent}
POST:
    load interaction by opaque ID
    if login is required and credentials are absent: reject
    if login is required: authenticate and record fresh auth_time
    if consent is required: require explicit approve or deny
    consume interaction exactly once when issuing response
```

### 2. `max_age` parsing and GET error recovery fail open

**Problem:** `sessionSatisfiesMaxAge` returns true for nonnumeric and negative
values. Very large parsed integers can overflow `time.Duration` multiplication.
Separately, GET treats any Fosite error as a reason to render a login form when
the partially built requester contains nonempty `max_age`.

**Where to look:** `internal/fositeadapter/session.go:53-61` and
`internal/fositeadapter/provider.go:346-354`.

```go
maxAge, err := strconv.ParseInt(maxAgeValue, 10, 64)
if err != nil || maxAge < 0 {
    return true
}
```

```go
ar, err := p.oauth2.NewAuthorizeRequest(...)
if err != nil {
    if ar != nil && ar.GetRequestForm().Get("max_age") != "" {
        p.renderInteraction(w, ar, true, true)
        return
    }
    // write the actual OAuth error
}
```

Fosite v0.49.0 also maps `ParseInt` failure to zero in its prompt validator, so
the application must validate the parameter explicitly rather than assume the
library rejects it.

**Why it matters:** malformed security parameters must not weaken the requested
authentication policy. More importantly, an invalid client, redirect, scope,
prompt combination, or other request error must not cause the IdP to collect a
password before the client and redirect are trusted.

**Required table:** `max_age` values `""`, `0`, `1`, `-1`, `+1`, whitespace,
decimal, `abc`, `9223372036854775807`, and one digit beyond `int64`. Cross each
with no session, fresh session, stale session, `prompt=none`, `prompt=login`,
invalid client, invalid redirect, invalid scope, and missing PKCE.

Properties:

```text
invalid max_age => invalid_request, never silent success
invalid client/redirect => local error, never redirect and never credential form
valid max_age with age > value => active login required
valid max_age with age <= value => session may be reused
```

Use an injected clock. Do not use `time.Sleep` for boundary tests.

### 3. The browser is carrying an incomplete, mutable continuation

**Problem:** `hidden(ar)` copies only response type, client, redirect, scope,
state, nonce, and PKCE fields. It omits at least `prompt`, `max_age`,
`id_token_hint`, `login_hint`, `ui_locales`, `claims`, `acr_values`, response
mode, requested audience/resource extensions, and unknown extension
parameters. POST then calls `NewAuthorizeRequest` on the reconstructed browser
form as though it were the original request.

**Where to look:** `internal/fositeadapter/provider.go:574-588` and POST line
400.

**Why it matters:** omission changes semantics; mutation lets browser state
change client, redirect, scope, nonce, state, and PKCE between validation and
authorization. Fosite revalidates many fields, which limits impact, but it does
not restore the original user-visible decision. The CSRF token proves that a
form came from this IdP; it does not bind one CSRF token to one immutable
authorization request.

It also creates a concurrent-tab failure: issuing a second CSRF cookie replaces
the cookie required by the first form. A CSRF token can be replayed with a
different valid hidden request until it is cleared.

**Required mutation tests:** for every supported authorization parameter,
start GET with value A and submit POST with value B, omitted, duplicated, or
reordered. Assert either the original A is used or the interaction is rejected;
never silently accept B. Run two forms in parallel and swap interaction IDs,
CSRF tokens, cookies, sessions, and submit order. Replay a successful and a
denied interaction.

**Preferred model:** persist a sanitized requester snapshot and required
actions:

```go
type Interaction struct {
    IDHash          []byte
    Request         PersistedAuthorizeRequest
    ClientID        string
    RedirectURI     string
    RequiredLogin   bool
    RequiredConsent bool
    SessionIDHash   []byte
    CSRFHash        []byte
    CreatedAt       time.Time
    ExpiresAt       time.Time
    ConsumedAt      *time.Time
}
```

Only an opaque random ID is placed in the form. On POST, load and consume the
record transactionally, revalidate mutable client/user status, and use the
stored request. Do not add a compatibility path that continues accepting the
old hidden request.

### 4. Consent is not represented as an explicit authorization decision

**Problem:** the form has one checkbox. An unchecked submission calls
`finishAuthorize`, which returns local HTTP 403 when consent is required.
Password login and browser-session creation happen before that denial. The RP
does not receive `error=access_denied` with its original state.

**Where to look:** `renderInteraction` and `finishAuthorize` at
`internal/fositeadapter/provider.go:708-765`.

**Why it matters:** consent approval must be clear about client and scopes;
denial is a protocol outcome, while invalid client/redirect is a local error.
The current form does not display the client or requested scopes. A raw 403
leaves the RP waiting and obscures the user's decision.

**Required tests:** explicit approve and deny buttons; exact client/scopes
rendered; denial returns `access_denied` only to the prevalidated redirect with
the original state; denial records no consent and creates no code; approve
cannot add scopes not displayed; stored consent cannot satisfy a strict
superset; disabled client/user between GET and POST fails closed.

Decide and document whether authentication alone may create an IdP browser
session after consent denial. Test the chosen behavior; do not let it be an
ordering accident.

### 5. Token endpoint limiting can be sharded by claimed client ID

**Problem:** before client authentication, the limiter key is
`token:<form client_id>:<address>`. A caller can vary form `client_id` values,
or use HTTP Basic without a form client ID, to choose buckets.

**Where to look:** `internal/fositeadapter/provider.go:456-479`.

**Why it matters:** token exchange and refresh perform storage and
cryptographic work and are targets for credential guessing and replay. A key
derived only from an untrusted, arbitrarily variable claim plus address is not
an address limit.

**Required tests:** send failures from one address using random client IDs,
missing client ID, Basic usernames, form authentication, and conflicting
Basic/form identities. The address/global limit must trigger after the same
number of requests regardless of claimed ID. A valid client's own bucket
should also trigger without logging its secret.

**Control sketch:** consume all applicable buckets, as login already does:

```text
token:global
token:address:<trusted address>
token:claimed-client:<hash normalized claimed ID>
after authentication, record authenticated-client metrics/audit
```

### 6. Fosite authorization state is written by separate handlers

**Problem:** Fosite `NewAuthorizeResponse` executes handlers in order. The
authorization-code handler inserts `fosite_authorize_codes`; PKCE then inserts
`fosite_pkces`; OIDC then inserts `fosite_oidc_sessions`. The current store
methods each execute independently.

**Where to look:** `internal/fositeadapter/sqlstore.go:178-242` and Fosite
v0.49.0 `authorize_response_writer.go`, OAuth explicit, PKCE, and OIDC
handlers.

**Why it matters:** an injected failure after the code row but before PKCE or
OIDC completion leaves partial state. The code might not be returned to the
browser, but stale or inconsistent protocol records affect cleanup,
diagnostics, uniqueness, retries, and confidence in the old-or-new-state rule.
Token response construction has analogous multi-handler storage behavior and
needs the same investigation.

**Required fault suite:** wrap the SQL adapter so the Nth create/delete/update
fails. For every handler boundary in authorize issuance, code exchange, access
plus refresh issuance, refresh rotation, and code-reuse handling, assert:

- no response credential is returned after a failed persistence step;
- no usable partial credential remains;
- retry behavior is deterministic;
- cleanup/maintenance can remove any deliberately retained tombstone;
- reuse detection revokes all tokens required by policy.

If Fosite cannot accept a transaction spanning handlers, implement a
request-scoped transactional store coordinator or explicit compensation whose
failure semantics are tested. Do not infer atomicity merely because each
individual method is safe.

### 7. UserInfo transport and response semantics are implicit

**Problem:** `userinfo` does not restrict methods, set an explicit cache
policy, or write `WWW-Authenticate`. Fosite's `AccessTokenFromRequest` accepts
the `access_token` query parameter as well as header/form transport.

**Where to look:** `internal/fositeadapter/provider.go:501-515` and Fosite
v0.49.0 `introspect.go`.

**Why it matters:** RFC 9700 says clients must not pass access tokens in URI
query parameters; URLs leak into logs and history. Sensitive UserInfo should
have a deliberate method, cache, content-type, and bearer-error contract.

**Required matrix:** GET with Bearer header; supported POST with
form-urlencoded body; GET query token; wrong content type; multiple token
locations; missing/invalid/expired/revoked token; PUT/DELETE/HEAD/OPTIONS.
Assert status, `WWW-Authenticate`, `Cache-Control`, `Pragma`, content type, and
that tokens never appear in audit logs.

### 8. Unsupported request-object error routing reads unverified claims

**Problem:** the provider intentionally rejects request objects, but manually
base64-decodes the JWT payload and uses unverified `client_id`, `redirect_uri`,
and `state` claims to choose error behavior.

**Where to look:** `rejectUnsupportedRequestObject`, `requestObjectClaims`, and
`stringClaim` at `internal/fositeadapter/provider.go:590-661`.

**Why it matters:** registered-redirect checks prevent a direct arbitrary open
redirect, but conflicting outer/inner claims, malformed JWTs, duplicate JSON
keys, very large payloads, and attacker-chosen echoed state need a stable rule.
Error routing should use only independently validated outer request data unless
the request object has been cryptographically verified.

**Required corpus:** malformed segment counts/base64/JSON; unsigned and signed-
shaped JWTs; oversized payload; duplicate keys; outer/inner client mismatch;
outer/inner redirect mismatch; disabled/unknown client; registered and
unregistered redirects; state only inner, only outer, and conflicting. Assert
that no invalid URI receives a redirect and no unverified inner claim becomes
trusted state.

### 9. Session/store failure is indistinguishable from absence

**Problem:** `readBrowserSession` returns `(zero, zero, false)` for a missing
cookie, bad handle, store error, expired/revoked session, user lookup error, or
disabled user.

**Where to look:** `internal/fositeadapter/session.go:27-41`.

**Why it matters:** revoked/expired/missing can correctly mean unauthenticated;
database unavailability should normally mean service unavailable. Collapsing
them can show a credential form during an outage and returns `login_required`
for `prompt=none`, hiding an operational failure.

**Required tests:** inject `ErrNotFound`, deadline, cancellation, corruption,
and generic store errors independently for session and user reads. Define and
assert which outcomes are unauthenticated, disabled, and unavailable. Audit and
readiness must reflect unavailable storage without revealing account/session
existence.

## Test program

### A. State-machine table tests

Create a table-driven strict authorization test whose dimensions are:

```text
session: absent | fresh | stale | revoked | store-error
prompt: absent | login | none | consent | none+login | unknown
max_age: absent | 0 | 1 | negative | malformed | overflow
login submission: absent | valid | invalid | different-account
consent: already-stored | required+approve | required+deny | tampered-scopes
request: valid | bad-client | bad-redirect | bad-scope | bad-PKCE
```

Expected results should be semantic actions, not only status codes:

```go
type Expected struct {
    ShowLogin       bool
    ShowConsent     bool
    OAuthError      string
    LocalStatus     int
    CodeIssued      bool
    FreshAuth       bool
    ConsentRecorded bool
}
```

For every successful code, exchange it and verify signed ID Token `sub`,
`nonce`, `auth_time`, `aud`, scopes, and PKCE binding. For every failure, query
protocol tables and assert no forbidden state was created.

### B. Deterministic clock tests

Inject a clock into session evaluation, provider request construction, and
authentication. Test just-before, exact, and just-after `max_age` boundaries
without sleeping. Include clock moving backward/forward and `auth_time` in the
future beyond permitted skew.

### C. Interaction mutation, replay, and concurrency

- Submit two tabs in both orders.
- Replay the same approve POST concurrently from 20 goroutines.
- Replay after success, denial, expiry, logout, password reset, client disable,
  and user disable.
- Swap interaction ID, CSRF token, and session cookie independently.
- Mutate each authorization parameter between GET and POST.
- Assert at most one code and one consent outcome per interaction.
- Run under `go test -race` and SQLite fault injection.

### D. Property and fuzz tests

Fuzz parsers and the state transition rather than only individual helpers:

```text
FuzzPromptAndMaxAge
FuzzAuthorizationContinuationMutation
FuzzRequestObjectRejection
FuzzDuplicateAuthorizeParameters
FuzzBearerTokenLocations
```

Useful properties:

```text
required actions never decrease because a parameter was omitted on POST
invalid request never reaches credential collection
no code is issued without validated client+redirect+PKCE and required actions
original state/nonce/challenge remain immutable across interaction
one interaction produces at most one terminal authorization outcome
errors never redirect to an unvalidated URI
```

Seed with the hosted OIDF prompt/max-age/request-object cases and every review
regression.

### E. Storage fault injection

Add a Fosite-store decorator with named failpoints for every mutation. Avoid
depending only on SQL syntax analyzers: handler ordering crosses several store
methods and packages. Run the full flow against memory and SQLite where
possible, then inspect row counts and redeemability.

### F. External and hosted gates

Extend the outside-module production fixture with:

- forced reauth and stale `max_age` negative submissions;
- consent denial and state preservation;
- two simultaneous browser interactions;
- UserInfo transport/error matrix;
- token limiter sharding attempts.

After local/race/fuzz/fault gates pass, rerun the fresh hosted OpenID Foundation
profile against the exact new artifact hash. The earlier Basic OP run marked
`prompt=login`, `max_age=1`, registered redirect, and request-object variants as
manual review cases, so they need explicit retained evidence rather than a
summary pass count.

## Recommended implementation order

1. Add failing regression tests for the known empty-login reauth bypass,
   invalid/negative `max_age`, and invalid redirect/client plus `max_age`.
2. Introduce the server-side pending interaction model and delete hidden-field
   reconstruction directly; do not retain a compatibility fallback.
3. Add explicit approve/deny consent and bind displayed client/scopes to the
   stored request.
4. Add deterministic clock and full state-machine/property tests.
5. Harden token limiter and UserInfo endpoint contracts.
6. Add Fosite lifecycle fault injection and make/compensate multi-handler
   protocol transitions.
7. Run full test, race, analyzer, fuzz, external-module, recovery, and exact-
   artifact hosted conformance gates.

The first three items should be treated as a release-hardening phase inserted
before the currently open Phase 5 approval gate. Candidate `2930981` must not
remain the candidate identity after these behavior changes.

## Usage Examples

Targeted commands after tests are implemented:

```bash
go test ./internal/fositeadapter -run 'TestAuthorizeInteraction|TestForcedReauth|TestMaxAge|TestConsentDenial|TestUserInfo' -count=1
go test ./internal/fositeadapter -run TestInteractionConcurrentReplay -race -count=10
go test ./internal/fositeadapter -run TestFositeStoreFailpoints -count=1
go test ./... -count=1
go test -race ./... -count=1
go test ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-consumer -count=1
```

Review protocol artifacts after each negative case:

```text
fosite_authorize_codes
fosite_pkces
fosite_oidc_sessions
fosite_access_tokens
fosite_refresh_tokens
consents
sessions
```

## Related

- `design-doc/01-production-embedding-api-and-release-implementation-guide.md`
- `reference/01-implementation-diary.md`
- `reference/03-release-candidate-evidence-packet-and-approval-ledger.md`
- `sources/openid-connect-core-1.0-errata2.md`
- `sources/rfc6749-oauth2-authorization-framework.md`
- `sources/rfc9700-oauth-security-bcp.md`
- `sources/rfc6750-bearer-token-usage.md`
