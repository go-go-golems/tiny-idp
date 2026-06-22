---
Title: Mock OIDC IdP Design and Implementation Guide
Ticket: MOCK-OIDC-IDP
Status: active
Topics:
    - oidc
    - go
    - testing
    - identity
    - auth
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp/main.go
      Note: Planned server entrypoint (Phase 0)
    - Path: cmd/tinyidp/main.go:Planned server entrypoint
    - Path: internal/client/client.go:Planned client registry
    - Path: internal/jwt/jwt.go:Planned RS256 JWT signing + JWKS
    - Path: internal/scenario/scenario.go:Planned scenario registry
    - Path: internal/server/server.go:Planned HTTP server and handlers
ExternalSources:
    - https://openid.net/specs/openid-connect-core-1_0.html
    - https://openid.net/specs/openid-connect-discovery-1_0.html
    - https://datatracker.ietf.org/doc/html/rfc6749
    - https://datatracker.ietf.org/doc/html/rfc7636
    - https://datatracker.ietf.org/doc/html/rfc7519
    - https://datatracker.ietf.org/doc/html/rfc7517
Summary: Design and intern-ready implementation guide for a minimal mock OpenID Connect Identity Provider (IdP) in Go, intended to replace Keycloak for local development and integration testing.
LastUpdated: 2026-06-22T14:51:07.588631044-04:00
WhatFor: 'Onboarding an unfamiliar engineer to the mock OIDC IdP: what it is, why it exists, how OIDC works, the architecture, the API surface, and the phased implementation plan.'
WhenToUse: Read this before implementing or extending the IdP. It is the single source of truth for scope, architecture, and phasing.
---


# Mock OIDC IdP Design and Implementation Guide

> Audience: a new intern engineer who has not worked on this project before.
> Goal: after reading this, you should be able to implement Phase 0–4 of the mock IdP without guessing.
> Scope: this is a **local testing tool**, not a production identity system. Security hardening is intentionally out of scope (see "Non-Goals").

---

## 1. Executive Summary

This project builds a **minimal mock OpenID Connect (OIDC) Identity Provider** in Go.
Its sole purpose is to replace Keycloak-in-Docker for **local development and integration testing** of applications that act as OIDC clients (Relying Parties, or RPs).

The mock IdP supports the OIDC "happy path": **discovery**, **JWKS**, **authorize**, **token**, and **userinfo**.
On top of the happy path it adds a **scenario model** so a developer can log in as `alice`, `bob`, or special usernames like `id-expired` / `userinfo-401` to reproduce real OIDC client bugs without a database, without Docker, and without account management.

The first usable release covers five phases:

1. **Phase 0** — Baseline OIDC happy path (discovery, JWKS, authorize, token, userinfo).
2. **Phase 1** — Multiple synthetic users (log in as any username; stable `sub`).
3. **Phase 2** — Scenario registry (replace ad-hoc string switches with a data model).
4. **Phase 3** — Login page with selectable scenarios (self-documenting UI).
5. **Phase 4** — High-value failure scenarios (expired token, wrong aud/iss, bad nonce, broken userinfo).

Later phases (multiple clients, session cookies/`prompt`, claims/authorization shapes, debug UI, refresh tokens, JWKS rotation, logout, Go test helper) are documented but explicitly **not** part of the first release.

---

## 2. Problem Statement and Scope

### 2.1 The problem

Many web apps implement OIDC login. To test that login locally, developers usually run **Keycloak in Docker Compose**. This is heavy:

- Keycloak boots slowly and uses real database-backed realms, admin UIs, and migrations.
- Configuring a realm, client, redirect URI, and a test user takes many clicks.
- Reproducing failure modes (expired token, wrong audience, broken userinfo) is hard or impossible without custom scripting.
- CI becomes coupled to Docker availability and image pulls.

For **local and test** scenarios, this is overkill. Most apps only need: "give me a working OIDC provider that issues RS256-signed ID tokens, accepts `authorization_code`, and lets me pick who I log in as."

### 2.2 Goals

- A single Go binary, no external dependencies beyond the standard library.
- Implements the OIDC endpoints a typical RP needs: discovery, JWKS, authorize, token, userinfo.
- Issues **RS256-signed** ID tokens and publishes keys via **JWKS**.
- Supports **`authorization_code`** with optional **PKCE (S256/plain)**.
- Supports **multiple synthetic users** keyed off any typed login, with a **stable `sub`**.
- Supports a **scenario registry** that can simulate authorization errors, token errors, malformed ID tokens, and userinfo failures.
- A **self-documenting login page** that lists usable test logins as buttons.
- Configurable via environment variables (issuer, client, redirect URIs, user claims).
- An **importable Go test helper** (later phase) so integration tests can spin up an IdP without Docker.

### 2.3 Non-Goals (intentionally out of scope)

This is **not production-grade**. It deliberately omits:

- Real login, consent, or password validation.
- Persistent keys (signing key is generated in-memory at startup).
- Refresh tokens (until Phase 9), revocation, pairwise subjects.
- Logout / end-session (until Phase 11).
- TLS enforcement, hardened redirect handling, dynamic client registration.
- Multi-tenancy, account databases, email verification flows.

> ⚠️ **Security boundary:** Never expose this IdP to the internet. Bind to `127.0.0.1` by default and do not add TLS termination as a substitute for "now it's safe to deploy."

### 2.4 Success conditions

| Phase | Success condition |
|-------|-------------------|
| 0 | A normal OIDC client can log in as `alice` and receive an ID token + access token. |
| 1 | Logging in as `alice` and `bob` creates distinct stable OIDC subjects. |
| 2 | Adding a new failure case requires adding one scenario, not editing every handler. |
| 3 | A developer can open the login page and immediately see available test identities and failure modes. |
| 4 | The app under test handles auth denial, token exchange failure, invalid ID tokens, and broken userinfo responses. |

---

## 3. Background: OIDC Concepts (read this if you are new)

If you already know OIDC well, skip to §4. Otherwise, read this carefully — every later section assumes these terms.

### 3.1 The cast of characters

- **End User (Resource Owner):** the human who wants to log in. In our mock, this is whoever types a login on the login page.
- **Relying Party (RP):** the application that wants to authenticate the user. It is the OIDC *client*. Your app under test is the RP.
- **Identity Provider (IdP) / OpenID Provider (OP):** the server that authenticates the user and issues tokens. **Our mock is the IdP.**
- **Authorization Server:** in OAuth2 terms, the thing that issues access tokens. For OIDC, the IdP *is* the authorization server.

### 3.2 The tokens

- **Authorization Code:** a short-lived, one-time string returned to the RP via redirect after login. The RP exchanges it for tokens. Lives ~5 minutes in our mock.
- **ID Token:** a **JWT** that proves who the user is. Contains claims like `iss`, `sub`, `aud`, `exp`, `iat`, `auth_time`, `nonce`, `email`, `name`. Signed by the IdP with RS256.
- **Access Token:** an opaque bearer token the RP sends to resource servers (and, in our mock, to `/userinfo`). In our mock it is a random string mapped in-memory to a user.
- **Refresh Token:** (Phase 9) lets the RP get a new access token without re-login. Not in the first release.

### 3.3 The endpoints (all under the `issuer` URL)

| Endpoint | Purpose |
|----------|---------|
| `/.well-known/openid-configuration` | **Discovery.** Returns provider metadata: where the other endpoints are, what's supported. |
| `/jwks` (`jwks_uri`) | **JWKS.** Public signing keys so clients can verify ID token signatures. |
| `/authorize` (`authorization_endpoint`) | Starts login. The RP redirects the user here; after login, the IdP redirects back with a `code`. |
| `/token` (`token_endpoint`) | The RP POSTs the `code` here (server-to-server) and receives ID + access tokens. |
| `/userinfo` (`userinfo_endpoint`) | The RP sends the access token here to get user claims. |

### 3.4 The authorization code flow (happy path)

This is the flow our mock implements. Walk through it slowly:

```
 RP (your app)                 IdP (mock)                   User browser
 ─────────────                 ─────────                    ─────────────
 1. RP redirects browser to /authorize
      ?client_id=dev-client
      &redirect_uri=http://localhost:3000/callback
      &response_type=code
      &scope=openid profile email
      &state=xyz
      &nonce=abc
      &code_challenge=...
      &code_challenge_method=S256
                            ─────────────────────────────►
                              2. IdP shows login page
                                                            3. User types "alice"
                              4. IdP stores auth code in memory,
                                 redirects to redirect_uri?code=CODE&state=xyz
                            ◄─────────────────────────────
 5. RP receives code, POSTs to /token
      grant_type=authorization_code
      code=CODE
      redirect_uri=...
      client_id=...
      [client_secret=...]
      [code_verifier=...]
                            ─────────────────────────────►
                              6. IdP validates code + PKCE + client,
                                 issues:
                                   - id_token (RS256 JWT)
                                   - access_token (opaque)
                                   - expires_in
                            ◄─────────────────────────────
 7. RP verifies id_token signature via /jwks,
    validates iss/aud/exp/nonce.
 8. RP calls /userinfo with access token
                            ─────────────────────────────►
                              9. IdP returns user claims
                            ◄─────────────────────────────
```

### 3.5 PKCE (Proof Key for Code Exchange)

PKCE protects the authorization code from being intercepted (important for SPAs and mobile). The RP:

1. Generates a random `code_verifier`.
2. Derives `code_challenge = base64url(sha256(verifier))` for `S256` (or `challenge = verifier` for `plain`).
3. Sends the `code_challenge` (and method) in `/authorize`.
4. Sends the original `code_verifier` in `/token`.

The IdP recomputes the challenge from the verifier and checks it matches. Our mock verifies both `S256` and `plain`.

### 3.6 JWT structure (what we sign)

A JWT is `base64url(header) . base64url(payload) . base64url(signature)`.

- **Header:** `{"typ":"JWT","alg":"RS256","kid":"dev-key-1"}`
- **Payload (claims):** `iss`, `sub`, `aud`, `exp`, `iat`, `auth_time`, `nonce`, `email`, `email_verified`, `name`, etc.
- **Signature:** `RS256(header.payload, private_key)` — i.e., PKCS#1v15 over SHA-256 of `header.payload`.

Clients fetch the public key from `/jwks` (keyed by `kid`) and verify the signature.

### 3.7 JWKS (JSON Web Key Set)

`/jwks` returns the **public** half of our signing key as a JWK:

```json
{
  "keys": [
    {"kty":"RSA","use":"sig","kid":"dev-key-1","alg":"RS256","n":"...","e":"AQAB"}
  ]
}
```

`n` and `e` are the RSA modulus and exponent, base64url-encoded. **Never expose the private key.**

### 3.8 Discovery metadata

`/.well-known/openid-configuration` advertises the provider. A compliant RP only needs the `issuer` and endpoint URLs to configure itself. Our mock advertises: supported `response_types`, `grant_types`, `subject_types`, signing `alg`s, `scopes`, `claims`, `code_challenge_methods`, and `token_endpoint_auth_methods`.

### 3.9 Standard references (bookmark these)

- **OIDC Core:** `https://openid.net/specs/openid-connect-core-1_0.html` — the authorize/token/userinfo behavior.
- **OIDC Discovery:** `https://openid.net/specs/openid-connect-discovery-1_0.html` — the metadata format.
- **RFC 6749 (OAuth 2.0):** `https://datatracker.ietf.org/doc/html/rfc6749` — the authorization code grant.
- **RFC 7636 (PKCE):** `https://datatracker.ietf.org/doc/html/rfc7636`.
- **RFC 7519 (JWT):** `https://datatracker.ietf.org/doc/html/rfc7519`.
- **RFC 7517 (JWK):** `https://datatracker.ietf.org/doc/html/rfc7517`.

---

## 4. Current-State Analysis

### 4.1 Repository state

The repository at `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp` is currently **scaffolding only**:

- One git commit: `Initial commit`.
- No Go module, no `main.go`, no source code yet.
- A docmgr ticket workspace under `ttmp/` (this documentation).

> Evidence: `git log --oneline` shows a single `:art: Initial commit`; `ls -la` shows only `.git`, `ttmp/`, and `.ttmp.yaml`.

### 4.2 Baseline design artifact

The design is grounded in a **single-file Go reference implementation** produced during research. It compiles with the standard library only and demonstrates:

- In-memory RSA key generation at startup (`crypto/rsa`, 2048-bit).
- A `server` struct holding issuer config, client config, redirect URI allowlist, the signing key, and in-memory `codes` / `tokens` maps guarded by a `sync.Mutex`.
- Handlers for discovery, JWKS, authorize, token, userinfo, and `/healthz`.
- A CORS middleware (permissive, for local browser testing).

The single-file version is the **starting point for Phase 0**. The phased plan (§9) refactors it into packages and layers the scenario model on top.

### 4.3 Key characteristics of the baseline (preserved in the mock)

- **No external dependencies.** Only the Go standard library. This keeps `go run .` instant and CI trivial.
- **In-memory everything.** Codes, tokens, and the signing key live in process memory. Restart loses all state. This is intentional for a test tool.
- **Fixed/single client by default.** Configurable via env (`OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URIS`). Multiple clients arrive in Phase 5.
- **RS256 only.** No other signing algorithms in the first release (avoids `alg=none` and HS/RS confusion attack surface).

---

## 5. Gap Analysis

| Need | Baseline status | Gap | Filled by |
|------|-----------------|-----|-----------|
| Happy-path OIDC | Implemented in reference | Not committed / not packaged | Phase 0 |
| Multiple users | Single fixed user in baseline | No login page, no stable per-login `sub` | Phase 1 |
| Failure simulation | None | No way to reproduce token/userinfo errors | Phase 2 + Phase 4 |
| Self-documenting UI | None | Developer must memorize magic usernames | Phase 3 |
| Scenario model | None (handlers would need switches) | Brittle string-switch code | Phase 2 |
| Multiple clients | Single client | Can't test public vs confidential clients | Phase 5 (later) |
| Sessions / `prompt` / `max_age` | None | Can't test silent login / reauth | Phase 6 (later) |
| Claim variants (groups/roles/tenant) | Minimal claims | Can't test authorization logic | Phase 7 (later) |
| Debug UI | None | Hard to inspect issued tokens | Phase 8 (later) |
| Refresh tokens | None | Can't test renewal | Phase 9 (later) |
| JWKS rotation | Single key | Can't test key rollover | Phase 10 (later) |
| Logout | None | Can't test RP-initiated logout | Phase 11 (later) |
| Go test helper | None | Can't embed in `go test` | Phase 12 (later) |

---

## 6. Proposed Architecture and APIs

### 6.1 High-level architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         cmd/tinyidp/main.go                          │
│   - parse env (OIDC_ISSUER, OIDC_ADDR, client config, users)         │
│   - generate RSA signing key (in-memory)                              │
│   - construct *Server, register handlers, ListenAndServe             │
└───────────────────────────────┬──────────────────────────────────────┘
                                │ owns
                                ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        internal/server                               │
│  Server struct:                                                      │
│    issuer, clients, signing key + kid, sessions?,                    │
│    codes map, tokens map, refresh map (Phase 9), mutex              │
│  HTTP handlers:                                                      │
│    /.well-known/openid-configuration  → discovery()                  │
│    /jwks                              → jwks()                       │
│    /authorize                         → authorize() (GET form, POST)│
│    /token                             → token()                      │
│    /userinfo                          → userinfo()                   │
│    /healthz                           → ok                           │
│  Middleware: withCORS (permissive, localhost only)                   │
└──────┬──────────────┬───────────────────┬───────────────────────────┘
       │ uses         │ uses              │ uses
       ▼              ▼                   ▼
┌────────────┐  ┌──────────────┐  ┌──────────────────┐
│ scenario   │  │ client       │  │ jwt              │
│  registry  │  │  registry    │  │  SignJWT(claims) │
│  Lookup()  │  │  Validate()  │  │  JWKS()          │
│  built-ins │  │  PKCE check  │  │  VerifyPKCE()    │
└────────────┘  └──────────────┘  └──────────────────┘
```

### 6.2 Proposed package layout

```
mock-oidc-idp/
├── cmd/
│   └── tinyidp/
│       └── main.go              # env parsing + server bootstrap
├── internal/
│   ├── server/
│   │   └── server.go            # Server struct + HTTP handlers + middleware
│   ├── scenario/
│   │   └── scenario.go          # Scenario type + registry + built-ins
│   ├── client/
│   │   └── client.go            # Client type + registry + validation
│   ├── jwt/
│   │   └── jwt.go               # RS256 signing, JWKS, PKCE verify
│   └── user/
│       └── user.go              # User type + userFromLogin()
├── go.mod
└── README.md
```

> Rationale: `cmd/` for the binary, `internal/` so nothing is importable by accident (until Phase 12 adds a public test-helper package). Each concern is a package so handlers stay thin.

### 6.3 Core data types

#### User

```go
// internal/user/user.go
package user

type User struct {
    Sub   string
    Email string
    Name  string
}

// userFromLogin derives a stable synthetic user from any typed login.
// "alice" -> sub=user-<sha256(alice)[:16]>, email=alice@example.test, name=alice
// "bob@example.test" -> sub=user-<sha256(...)>, email=bob@example.test, name=bob
func FromLogin(login string) User
```

- `sub` is **deterministic**: the same login always produces the same `sub`. Logging in as `alice` tomorrow yields the same subject as today.
- `sub = "user-" + base64url(sha256("tinyidp:user:" + normalizedLogin)[:16])`.
- If the login has no `@`, email is `<login>@example.test`; otherwise email is the login itself.
- `name` is the login with the `@domain` stripped.

#### Client (Phase 5; simplified single-client first)

```go
// internal/client/client.go
package client

type Client struct {
    ID            string
    Secret        string   // "" = public client
    RedirectURIs  []string
    RequirePKCE   bool
    AllowedScopes []string
}
```

#### AuthCode and AccessToken

```go
// internal/server/server.go
type authCode struct {
    ClientID            string
    RedirectURI         string
    Scope               string
    Nonce               string
    CodeChallenge       string
    CodeChallengeMethod string
    Expires             time.Time
    User                user.User
    FailureMode         string   // replaced by *Scenario in Phase 2
}

type accessToken struct {
    User        user.User
    Expires     time.Time
    FailureMode string   // replaced by *Scenario in Phase 2
}
```

### 6.4 Scenario model (Phase 2)

The scenario model replaces string switches scattered across handlers with a single registry. A scenario describes everything that should happen (or fail) for a given login.

```go
// internal/scenario/scenario.go
package scenario

type Scenario struct {
    Name        string
    Description string
    User        user.User

    // Failures, at most one set per scenario:
    AuthError     string  // OAuth error code returned at /authorize, e.g. "access_denied"
    TokenError    string  //OAuth error at /token, e.g. "invalid_grant"
    UserInfoError string  // "401" | "500" | "sub_mismatch"

    // Optional claim mutation (bad aud, expired, wrong nonce, etc.):
    MutateClaims func(claims map[string]any, now time.Time)
}

// Registry maps a normalized login to a Scenario.
type Registry struct{ m map[string]Scenario }

func New() *Registry           // preloaded with built-ins
func (r *Registry) Lookup(login string) (Scenario, bool)
func (r *Registry) All() []Scenario
```

**Built-in scenarios** (Phase 2 + Phase 4):

| Login | Kind | Behavior |
|-------|------|----------|
| `alice` | normal | user alice |
| `bob` | normal | user bob |
| `admin` | normal + claims | user with `groups:[admin]` (Phase 7) |
| `viewer` | normal + claims | user with `groups:[viewer]` |
| `fail-access-denied` | auth error | `/authorize` redirects with `error=access_denied` |
| `fail-login-required` | auth error | `error=login_required` |
| `fail-consent-required` | auth error | `error=consent_required` |
| `fail-server-error` | auth error | `error=server_error` |
| `token-invalid-grant` | token error | `/token` returns `invalid_grant` |
| `token-server-error` | token error | `/token` returns 500 `server_error` |
| `token-slow` | token error | `/token` sleeps 10s then returns normally |
| `id-expired` | claim mutation | `exp = now - 1h` |
| `id-wrong-aud` | claim mutation | `aud = "some-other-client"` |
| `id-wrong-iss` | claim mutation | `iss = issuer + "/wrong"` |
| `id-missing-email` | claim mutation | delete `email` + `email_verified` |
| `id-email-unverified` | claim mutation | `email_verified = false` |
| `id-bad-nonce` | claim mutation | `nonce = "wrong-nonce"` (if RP sent one) |
| `id-future-iat` | claim mutation | `iat`, `auth_time` = now + 10m |
| `userinfo-401` | userinfo error | `/userinfo` returns 401 |
| `userinfo-500` | userinfo error | `/userinfo` returns 500 |
| `userinfo-sub-mismatch` | userinfo error | `/userinfo` returns a different `sub` |

### 6.5 HTTP API reference

All responses are JSON unless noted. The issuer is `http://localhost:5556` by default.

#### `GET /.well-known/openid-configuration`

Discovery metadata. 200 OK.

```json
{
  "issuer": "http://localhost:5556",
  "authorization_endpoint": "http://localhost:5556/authorize",
  "token_endpoint": "http://localhost:5556/token",
  "userinfo_endpoint": "http://localhost:5556/userinfo",
  "jwks_uri": "http://localhost:5556/jwks",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "scopes_supported": ["openid","profile","email"],
  "claims_supported": ["sub","iss","aud","exp","iat","auth_time","nonce","email","email_verified","name"],
  "code_challenge_methods_supported": ["S256","plain"],
  "token_endpoint_auth_methods_supported": ["none","client_secret_basic","client_secret_post"]
}
```

#### `GET /jwks`

Public signing keys. 200 OK.

```json
{
  "keys": [
    {"kty":"RSA","use":"sig","kid":"dev-key-1","alg":"RS256","n":"<base64url>","e":"AQAB"}
  ]
}
```

#### `GET /authorize` (start login, show form)

Query params: `response_type=code`, `client_id`, `redirect_uri`, `scope` (must include `openid`), `state`, `nonce`, `code_challenge`, `code_challenge_method`.

- Validates `client_id`, `redirect_uri` allowlist, `response_type`, `scope`.
- Renders an HTML login page that echoes the authorize params as hidden fields + lists scenario buttons (Phase 3).

#### `POST /authorize` (submit login)

Form fields: the hidden authorize params + `login` + `password` (ignored).

- Normalizes `login`, looks up a scenario.
- If scenario has `AuthError`, redirects to `redirect_uri` with `error`/`error_description`/`state` (does **not** issue a code).
- Otherwise stores an `authCode` in memory (with scenario attached), redirects to `redirect_uri?code=...&state=...`.

#### `POST /token` (exchange code)

Form body: `grant_type=authorization_code`, `code`, `redirect_uri`, `client_id`, optional `client_secret`, optional `code_verifier`.

- Supports `client_secret_basic` (HTTP Basic) and `client_secret_post`.
- Validates client, code existence/expiry, `client_id`+`redirect_uri` match, PKCE.
- If scenario has `TokenError`, returns the matching OAuth error (or sleeps, for `token-slow`).
- Applies `MutateClaims` to the ID token claims.
- Returns `access_token`, `token_type=Bearer`, `expires_in=3600`, `scope`, `id_token`.

Response (200):
```json
{
  "access_token": "<opaque>",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile email",
  "id_token": "<RS256 JWT>"
}
```

Error response (e.g. 400):
```json
{"error":"invalid_grant","error_description":"unknown or expired code"}
```

#### `GET /userinfo`

Header: `Authorization: Bearer <access_token>`.

- Looks up the access token in memory.
- If scenario has `UserInfoError`, returns 401 / 500 / mismatched `sub`.
- Otherwise returns `sub`, `email`, `email_verified`, `name`.

#### `GET /healthz`

Returns `ok\n`. Use for liveness in tests.

### 6.6 Environment variables

| Variable | Default | Meaning |
|----------|---------|---------|
| `OIDC_ISSUER` | `http://localhost:5556` | Issuer URL; endpoints are derived from it. |
| `OIDC_ADDR` | `127.0.0.1:5556` | Listen address. Bind to loopback by default. |
| `OIDC_CLIENT_ID` | `dev-client` | Single client ID (Phase 0–4). |
| `OIDC_CLIENT_SECRET` | (empty) | If set, token endpoint enforces it. |
| `OIDC_REDIRECT_URIS` | `http://localhost:3000/callback,http://127.0.0.1:3000/callback` | CSV allowlist. |

---

## 7. Decision Records

### Decision: Single binary, standard library only

- **Context:** The tool must replace Keycloak-in-Docker for local testing; boot time and setup friction are the core pain points.
- **Options considered:** (a) standard library only; (b) a framework like `chi`/`gin`; (c) `zitadel/oidc` or `ory/fosite` server packages; (d) a small Docker image.
- **Decision:** Single Go binary, standard library only (mirrors the `go-web-frontend-embed` skill's preference for `net/http` + `http.ServeMux`).
- **Rationale:** Zero-config `go run .`; no module cache churn; trivial CI; the OIDC surface we need is small enough to implement by hand. External OIDC frameworks pull in real consent/account models we explicitly don't want.
- **Consequences:** We must implement JWT/JWKS/PKCE by hand (~150 lines). We accept this to keep the dependency surface at zero. Future Phase 12 (test helper) stays importable without heavy deps.
- **Status:** accepted

### Decision: In-memory state, generated-at-startup RSA key

- **Context:** A test tool should be deterministic enough for assertions but must not require a database.
- **Options considered:** (a) in-memory maps + generated key; (b) file-backed key + BoltDB for codes/tokens; (c) Redis.
- **Decision:** In-memory `map`s guarded by a `sync.Mutex`; RSA key generated with `crypto/rsa.GenerateKey` on startup.
- **Rationale:** Codes/tokens are short-lived and few; a mutex is sufficient. A random key per start is fine because clients fetch JWKS dynamically. Determinism comes from `sub` being derived from the login (not from key stability).
- **Consequences:** Restart invalidates all outstanding codes/tokens (acceptable). JWKS changes per restart (acceptable; clients refetch). Cannot survive across processes — by design.
- **Status:** accepted

### Decision: Scenario registry over string switches

- **Context:** Phase 1 hardcodes failures as `switch login` blocks inside each handler; this becomes brittle as failure modes grow.
- **Options considered:** (a) per-handler `switch` on login string; (b) a scenario struct with optional failure fields + a `MutateClaims` hook; (c) an external YAML/JSON scenario file.
- **Decision:** A `Scenario` struct (§6.4) with `AuthError`, `TokenError`, `UserInfoError`, and `MutateClaims`; a `Registry` mapping login → scenario.
- **Rationale:** Adding a failure case becomes one entry, not edits in three handlers. `MutateClaims` keeps ID-token mutations composable. Keeping it in-code (not YAML) avoids config drift and keeps the single-binary property.
- **Consequences:** Handlers call `registry.Lookup(login)` once and branch on scenario fields. Phase 3 reads `registry.All()` to render the login page, so the page is always in sync with supported scenarios.
- **Status:** accepted

### Decision: RS256 only (no `alg=none`, no HS256)

- **Context:** JWT `alg` confusion and `alg=none` are classic attack vectors; we want to test real client behavior without exposing those vectors by default.
- **Options considered:** (a) RS256 only; (b) also support `none`/HS256 behind a flag.
- **Decision:** RS256 only in the first release. Discovery advertises only `RS256`.
- **Rationale:** Most production IdPs use RS256; clients should be tested against that. Exotic alg attacks are deferred to Phase 10 (JWKS chaos), where they can be opt-in scenarios.
- **Consequences:** Clients that only handle RS256 work. Clients with HS256-specific bugs aren't testable until Phase 10.
- **Status:** accepted

### Decision: Deterministic `sub` derived from login

- **Context:** Tests need stable subjects to assert against, but we don't want a user database.
- **Options considered:** (a) random `sub` per login; (b) `sub = login` directly; (c) `sub = "user-" + base64url(sha256(login))`.
- **Decision:** `sub = "user-" + base64url(sha256("tinyidp:user:" + normalize(login))[:16])`.
- **Rationale:** Stable across restarts (no storage needed). Not equal to the raw login (so apps don't accidentally treat login as subject). Normalized (lowercased/trimmed) so `Alice` and `alice` match.
- **Consequences:** `sub` is opaque to humans; the debug UI (Phase 8) will map them back for readability. Two different logins cannot collide on `sub` in practice.
- **Status:** accepted

### Decision: Redirect URI allowlist enforced before any redirect

- **Context:** OAuth redirect handling is a well-known attack surface; even a mock must not redirect to arbitrary URLs.
- **Options considered:** (a) allow any `redirect_uri`; (b) enforce an allowlist from env.
- **Decision:** Enforce an allowlist (`OIDC_REDIRECT_URIS`) **before** showing the login page or issuing errors. Disallowed URIs produce a direct 400, never a redirect.
- **Rationale:** A test tool must not train apps to tolerate open redirects. Validation happens in `parseAuthorizeRequest`, before login.
- **Consequences:** Developers must list their callback URLs. This is a one-time setup cost; acceptable.
- **Status:** accepted

### Decision: Bind to 127.0.0.1 by default

- **Context:** The IdP has no real security; it must not be network-reachable.
- **Options considered:** (a) default `0.0.0.0`; (b) default `127.0.0.1`; (c) enforce loopback only.
- **Decision:** `OIDC_ADDR` defaults to `127.0.0.1:5556`.
- **Rationale:** Prevents accidental exposure. Developers who need LAN access (e.g., mobile device testing) can opt in by setting `OIDC_ADDR=0.0.0.0:5556`.
- **Consequences:** Device-on-LAN testing requires an env override; documented in the README.
- **Status:** accepted

---

## 8. Pseudocode and Key Flows

### 8.1 `main()`

```text
func main():
    issuer   = env("OIDC_ISSUER", "http://localhost:5556")  # trimmed of trailing /
    clientID = env("OIDC_CLIENT_ID", "dev-client")
    secret   = os.Getenv("OIDC_CLIENT_SECRET")
    redirs   = parseCSV(env("OIDC_REDIRECT_URIS", default...))

    key, _ = rsa.GenerateKey(rand.Reader, 2048)
    s = &Server{
        issuer, clientID, secret, redirs,
        key, kid: "dev-key-1",
        codes: {}, tokens: {},
        registry: scenario.New(),   # Phase 2
    }
    mux = http.NewServeMux()
    mux.HandleFunc("/.well-known/openid-configuration", s.discovery)
    mux.HandleFunc("/jwks", s.jwks)
    mux.HandleFunc("/authorize", s.authorize)
    mux.HandleFunc("/token", s.token)
    mux.HandleFunc("/userinfo", s.userinfo)
    mux.HandleFunc("/healthz", ok)

    addr = env("OIDC_ADDR", "127.0.0.1:5556")
    log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
```

### 8.2 `/authorize` (GET + POST)

```text
GET /authorize:
    ar = parseAuthorizeRequest(query)        # validates client_id, redirect_uri, scope, response_type
    if error: 400 (never redirect an invalid request)
    render login page with hidden fields + scenario buttons from registry.All()

POST /authorize:
    parseForm()
    ar = parseAuthorizeRequest(form)
    login = normalizeLogin(form.Get("login"))
    if login == "": 400 "login required"
    sc = registry.Lookup(login)              # Phase 2; defaults to a normal user derived from login

    # Auth-error scenarios: redirect back with OAuth error
    if sc.AuthError != "":
        redirectOAuthError(redirect_uri, state, sc.AuthError, "simulated")
        return

    issueCodeAndRedirect(ar, sc.User, sc)    # stores authCode{..., FailureMode/Scenario: sc}
```

### 8.3 `/token`

```text
POST /token:
    if method != POST: 405 invalid_request
    parseForm()
    if grant_type != "authorization_code": 400 unsupported_grant_type

    # client auth: Basic or client_secret_post
    clientID, basicSecret, hasBasic = r.BasicAuth()
    if clientID == "": clientID = form.Get("client_id")
    if clientID != s.clientID: 401 invalid_client
    if secret != "":
        provided = hasBasic ? basicSecret : form.Get("client_secret")
        if provided != secret: 401 invalid_client

    code = form.Get("code")
    ac, ok = s.codes.pop(code)               # one-time use
    if !ok or now > ac.Expires: 400 invalid_grant "unknown or expired code"
    if ac.ClientID != clientID or ac.RedirectURI != form.redirect_uri:
        400 invalid_grant "client_id or redirect_uri mismatch"
    if !verifyPKCE(ac.CodeChallenge, ac.CodeChallengeMethod, form.code_verifier):
        400 invalid_grant "PKCE verification failed"

    sc = ac.Scenario

    # Token-error scenarios
    if sc.TokenError == "invalid_grant": 400 invalid_grant
    if sc.TokenError == "server_error":  500 server_error
    if sc.TokenError == "slow":           sleep 10s

    access = randomB64(32)
    s.tokens[access] = {User: ac.User, Expires: now+1h, Scenario: sc}

    claims = {
        iss, sub, aud: clientID, exp: now+1h, iat: now, auth_time: now,
        email, email_verified: true, name
    }
    if ac.Nonce != "": claims.nonce = ac.Nonce
    if sc.MutateClaims != nil: sc.MutateClaims(claims, now)   # id-expired, id-wrong-aud, ...

    idToken = signJWT(claims)                 # RS256
    200 {access_token, token_type: Bearer, expires_in: 3600, scope, id_token}
```

### 8.4 `/userinfo`

```text
GET /userinfo:
    auth = header.Authorization
    if not "Bearer ": 401
    token = trimPrefix(auth, "Bearer ")
    at, ok = s.tokens[token]
    if !ok or now > at.Expires: 401

    sc = at.Scenario
    switch sc.UserInfoError:
        "401":           return 401 "simulated invalid bearer token"
        "500":           return 500 "simulated userinfo server error"
        "sub_mismatch":  return 200 {sub: at.User.Sub+"-different", email, name, ...}

    200 {sub, email, email_verified: true, name}
```

### 8.5 PKCE verification

```text
func verifyPKCE(challenge, method, verifier):
    if challenge == "": return true              # PKCE optional
    if verifier == "": return false
    switch method:
        "", "plain":  return verifier == challenge
        "S256":       return base64url(sha256(verifier)) == challenge
        default:      return false
```

### 8.6 JWT signing (RS256)

```text
func signJWT(claims):
    header = {"typ":"JWT","alg":"RS256","kid": s.kid}
    input  = base64url(header) + "." + base64url(claims)
    sum    = sha256(input)
    sig    = rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, sum)
    return input + "." + base64url(sig)
```

### 8.7 `userFromLogin`

```text
func userFromLogin(login):
    login = lower(trim(login))
    sub   = "user-" + base64url(sha256("tinyidp:user:" + login)[:16])
    email = contains(login, "@") ? login : login + "@example.test"
    name  = contains(login, "@") ? login[:index("@")] : login
    return {Sub: sub, Email: email, Name: name}
```

---

## 9. Implementation Plan (Phased)

Each phase is independently shippable. Do not skip ahead; later phases build on earlier ones.

### Phase 0 — Baseline OIDC happy path

**Goal:** a normal OIDC client can log in as `alice` and receive tokens.

1. `go mod init` (module name e.g. `github.com/manuel/tinyidp`).
2. Create `cmd/tinyidp/main.go` with the `server` struct, env parsing, key generation, `ListenAndServe`.
3. Implement `discovery()`, `jwks()`, `authorize()` (GET only, immediate redirect), `token()`, `userinfo()`, `/healthz`.
4. Implement `signJWT`, `verifyPKCE`, `randomB64`, `b64`, `writeJSON`, `tokenError`, `withCORS`.
5. Add `README.md` with run instructions and env vars.

**Files:**
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main.go`
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/go.mod`
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/README.md`

**Validate:** `go run .`; point a real OIDC client (or `curl` the endpoints manually) and confirm an ID token is issued and verifiable against `/jwks`.

### Phase 1 — Multiple synthetic users

**Goal:** log in as any username; distinct stable `sub`s.

1. Replace the fixed default user with a `userFromLogin` derivation.
2. Convert `/authorize` to a GET (show form) + POST (submit login) handler using `html/template`.
3. Echo authorize params as hidden fields on the login form.
4. Normalize login (lowercase/trim); require non-empty.

**Files:**
- `cmd/tinyidp/main.go` (or extract `internal/user/user.go` + `internal/server/server.go`).

**Validate:** log in as `alice` and `bob` separately; confirm `sub` differs and is stable across restarts (depends only on the login string).

### Phase 2 — Scenario registry

**Goal:** adding a failure case is one scenario, not three handler edits.

1. Create `internal/scenario/scenario.go` with the `Scenario` struct and `Registry`.
2. Preload normal scenarios (`alice`, `bob`) and the failure scenarios from §6.4.
3. Replace `FailureMode string` on `authCode`/`accessToken` with `*scenario.Scenario`.
4. In handlers, call `registry.Lookup(login)` once; branch on `AuthError` / `TokenError` / `UserInfoError` / `MutateClaims`.

**Files:**
- `internal/scenario/scenario.go`
- `internal/server/server.go` (handler changes)

**Validate:** add a throwaway scenario and confirm only one file changed; remove it.

### Phase 3 — Login page with selectable scenarios

**Goal:** the login page is self-documenting.

1. Group `registry.All()` into categories (normal / auth-failure / token-failure / id-token-failure / userinfo-failure).
2. Render each as a button that fills the `login` field (and submits).
3. Keep the manual login input for arbitrary usernames.

**Validate:** open `/authorize?...` in a browser; confirm every listed scenario is reachable in one click and matches registry contents.

### Phase 4 — High-value failure scenarios

**Goal:** reproduce real OIDC client bugs.

Implement and verify each (pseudocode in §8):

- `fail-access-denied`, `fail-login-required`, `fail-consent-required`, `fail-server-error`
- `token-invalid-grant`, `token-server-error`, `token-slow`
- `id-expired`, `id-wrong-aud`, `id-wrong-iss`, `id-missing-email`, `id-email-unverified`, `id-bad-nonce`, `id-future-iat`
- `userinfo-401`, `userinfo-500`, `userinfo-sub-mismatch`

**Validate:** for each, run the full flow against a sample RP and confirm the failure surfaces where expected (RP error page, token exchange error, id-token validation failure, userinfo error).

### Later phases (documented, not in first release)

- **Phase 5** — Multiple clients (`public-spa` PKCE-only, `web-app` confidential, `dev-client` permissive).
- **Phase 6** — Session cookie + `prompt=none`/`prompt=login` + `max_age` + `login_hint` + `auth_time`.
- **Phase 7** — Claim variants: `admin`, `viewer`, `no-email`, `unverified-email`, `no-groups`, `many-groups`, `tenant-a-admin`, `unicode-name`; `groups`/`roles`/`tenant`/`preferred_username`.
- **Phase 8** — Debug UI: `/debug`, `/debug/sessions`, `/debug/codes`, `/debug/tokens`, `/debug/reset` (loopback only).
- **Phase 9** — Refresh tokens: `offline_access`, `refresh_token` grant, rotation, reuse detection.
- **Phase 10** — JWKS/key rotation: multiple kids, `kid-not-found`, bad signature, JWKS 500/slow/empty.
- **Phase 11** — Logout: `/end-session`, `id_token_hint`, `post_logout_redirect_uri`, `state`.
- **Phase 12** — Go test helper: `func Start(t testing.TB, opts Options) *Provider` returning `Issuer()`.

---

## 10. Testing Strategy

### 10.1 Unit tests

- `verifyPKCE`: cover `S256` (valid/invalid), `plain` (valid/invalid), empty challenge, empty verifier, unknown method.
- `userFromLogin`: cover plain name, email form, mixed case, leading/trailing spaces, unicode; assert `sub` stability and distinctness.
- `signJWT` + JWKS round-trip: sign claims, parse the JWT, fetch the JWK, verify the signature with `crypto/rsa.VerifyPKCS1v15`.
- `scenario.Registry.Lookup`: confirm every built-in resolves; confirm unknown login falls back to a normal derived user.

### 10.2 HTTP handler tests

Use `httptest.NewServer` or `httptest.NewRequest` + `httptest.NewRecorder`:

- **Discovery** returns all required fields and correct issuer-derived URLs.
- **Authorize** rejects bad `client_id`, bad `redirect_uri`, missing `openid` scope, wrong `response_type` — each with a 400 and **no redirect**.
- **Authorize** stores a code and redirects with `code` + `state`.
- **Token** rejects reused codes (second exchange fails), expired codes, PKCE mismatch, wrong `client_id`/`redirect_uri`.
- **Token** applies each ID-token mutator (e.g., `id-expired` → `exp` in the past).
- **Userinfo** rejects missing/invalid bearer tokens; returns correct claims; honors `userinfo-401/500/sub-mismatch`.

### 10.3 End-to-end / conformance

- A Go test that acts as an RP: drive the full authorize→token→userinfo flow for `alice`; parse and validate the ID token (signature via JWKS, `iss`, `aud`, `exp`, `nonce`).
- A matrix test: for each scenario, assert the expected outcome (redirect error / token error / mutated claim / userinfo error).

### 10.4 Manual smoke test

```bash
go run .
# in another shell:
curl -s http://localhost:5556/.well-known/openid-configuration | jq .
curl -s http://localhost:5556/jwks | jq .
```

Then point a real OIDC client at `http://localhost:5556` with `client_id=dev-client`.

---

## 11. Risks, Alternatives, and Open Questions

### 11.1 Risks

- **Security misuse:** someone exposes the IdP to the internet. Mitigation: loopback default, README warning, no production claims.
- **Concurrency bugs:** the `sync.Mutex` around `codes`/`tokens` must be held for read-modify-write (especially code redemption, which is pop-and-delete). A naive "get then delete" is a race. Mitigation: pop under the lock (see §8.3).
- **Key rotation confusion:** because the key is per-startup, a client that caches JWKS forever will break after a restart. This is acceptable for a test tool; document it.
- **Scenario explosion:** too many scenarios clutter the login page. Mitigation: Phase 3 groups them; consider a "show advanced" toggle later.
- **Single-client assumption:** Phase 0–4 assumes one client; apps with multiple clients can't be tested until Phase 5.

### 11.2 Alternatives considered (rejected)

- **`zitadel/oidc` or `ory/fosite`:** powerful, but pull in real consent/account models and a large dependency tree. Rejected to preserve the zero-dep, instant-boot property. Good reference for spec compliance, though.
- **Keycloak in Docker:** the thing we're replacing. Rejected for the reasons in §2.1.
- **Mock at the RP level (fake the OIDC client, not the IdP):** doesn't test the actual OIDC client code path. Rejected.
- **YAML-driven scenarios:** rejected for Phase 2 (in-code registry keeps the single-binary property and avoids config drift). Could be revisited for Phase 12 if the test helper needs user-supplied scenarios.

### 11.3 Open questions

1. Should `token-slow` be 10s or configurable? (10s is fine for manual testing; may break tight CI timeouts.)
2. Should the debug UI (Phase 8) require a token or just rely on loopback binding?
3. For Phase 12, is the public API `Start(t, opts)` enough, or do tests need per-scenario helpers like `idp.LoginAs("alice")` that complete the whole flow?

---

## 12. References

### 12.1 Standards

- OIDC Core — `https://openid.net/specs/openid-connect-core-1_0.html`
- OIDC Discovery — `https://openid.net/specs/openid-connect-discovery-1_0.html`
- RFC 6749 (OAuth 2.0) — `https://datatracker.ietf.org/doc/html/rfc6749`
- RFC 7636 (PKCE) — `https://datatracker.ietf.org/doc/html/rfc7636`
- RFC 7519 (JWT) — `https://datatracker.ietf.org/doc/html/rfc7519`
- RFC 7517 (JWK) — `https://datatracker.ietf.org/doc/html/rfc7517`

### 12.2 Planned project files (absolute paths)

- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main.go` — server entrypoint, env parsing, bootstrap.
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/server.go` — `Server` struct, HTTP handlers, middleware.
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go` — `Scenario` + `Registry` + built-ins.
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/client/client.go` — `Client` + validation (Phase 5).
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/jwt/jwt.go` — RS256 signing, JWKS, PKCE verify.
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/user/user.go` — `User` + `FromLogin`.
- `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/README.md` — run/config instructions.

### 12.3 Related Go libraries (for reference only, not dependencies)

- `zitadel/oidc` — full OIDC client/server in Go (good spec reference).
- `ory/fosite` — OAuth2/OIDC framework for Go.

### 12.4 Related ticket documents

- Diary: `ttmp/2026/06/22/MOCK-OIDC-IDP--mock-oidc-identity-provider-for-local-testing-keycloak-replacement/reference/01-implementation-diary.md`
- Tasks: `.../tasks.md`
- Changelog: `.../changelog.md`
