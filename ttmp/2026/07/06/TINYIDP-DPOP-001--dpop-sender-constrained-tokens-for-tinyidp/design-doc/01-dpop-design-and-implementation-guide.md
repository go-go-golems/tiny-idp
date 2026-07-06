---
Title: DPoP Design and Implementation Guide
Ticket: TINYIDP-DPOP-001
Status: active
Topics:
    - oidc
    - auth
    - identity
    - testing
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/jwt.go
      Note: |-
        Discovery metadata advertises DPoP signing algorithms.
        Discovery metadata for DPoP proof algorithms
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/server.go
      Note: |-
        In-memory token state and replay cache state live on Server.
        In-memory token and replay-cache state
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/token.go
      Note: |-
        Token endpoint issues opaque access and refresh tokens; DPoP proof validation and token binding attach here.
        Token endpoint grant dispatch and issuance helpers where DPoP binding is added
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/userinfo.go
      Note: |-
        Protected resource endpoint; DPoP-bound access tokens must require proof and ath validation here.
        Protected resource endpoint that enforces DPoP-bound access tokens
ExternalSources:
    - 'RFC 9449: OAuth 2.0 Demonstrating Proof of Possession (DPoP): https://www.rfc-editor.org/rfc/rfc9449.html'
Summary: Design and implementation guide for adding DPoP sender-constrained token support to tinyidp.
LastUpdated: 2026-07-06T00:00:00-04:00
WhatFor: Use this guide before implementing or reviewing DPoP support in tinyidp.
WhenToUse: Read when adding proof JWT validation, cnf/jkt token binding, DPoP token responses, replay protection, or DPoP resource checks.
---


# DPoP Sender-Constrained Tokens for tinyidp

## Executive summary

DPoP, Demonstrating Proof-of-Possession, binds an OAuth access token to a public key controlled by the client. The authorization server validates a signed proof JWT at the token endpoint, computes the JWK thumbprint of the public key in that proof, and records that thumbprint with the token it issues. A protected resource later accepts the token only when the caller presents a fresh DPoP proof signed by the same key and, for access-token use, containing an `ath` hash of the access token.

tinyidp currently issues opaque bearer access tokens. A bearer token is sufficient by itself: whoever has the token can call `/userinfo`. DPoP changes the test semantics for clients that opt in. A DPoP-bound token is not enough by itself; the caller must also prove possession of the matching private key on every protected-resource request. This makes tinyidp useful for testing RPs and API clients that implement sender-constrained OAuth without turning tinyidp into a production identity provider.

This implementation should be deliberately small and deterministic:

- DPoP is opt-in per token request, based on the presence of the `DPoP` HTTP header.
- Opaque access-token state stores `DPoPJKT`, the RFC 7638 JWK thumbprint.
- Opaque refresh-token state also stores `DPoPJKT` so refreshes must use the same proof key.
- `/userinfo` enforces DPoP proof validation only for DPoP-bound access tokens.
- Proof replay protection is in memory, like every other tinyidp runtime state map.
- Nonce support is explicitly out of scope for the first implementation; it can be added later with `DPoP-Nonce` and `use_dpop_nonce` errors.

## The problem DPoP solves

Bearer tokens are simple. The token endpoint creates a random string, stores it in memory, and `/userinfo` accepts `Authorization: Bearer <token>` if the string exists and has not expired. The resource server does not know who is holding the token. That is appropriate for many local tests, but it cannot exercise clients that implement proof-of-possession requirements.

DPoP adds one more check to token use. The client generates an asymmetric key pair and signs a proof JWT for a specific HTTP method and URI. The server verifies that signature, validates the method and URI claims, prevents replay through `jti`, and computes the public key thumbprint. When the server issues a DPoP-bound access token, it stores the thumbprint beside the token. Later, `/userinfo` validates another proof and compares the proof key thumbprint with the token's stored thumbprint.

The important distinction is that DPoP does not replace OAuth grant validation. Authorization code, refresh token, and device-code grants still decide whether a token may be issued. DPoP only constrains the token so it can be used by the holder of a particular private key.

## Current tinyidp architecture

The relevant server state is in `internal/server/server.go`:

```go
type Server struct {
    mu            sync.Mutex
    codes         map[string]authCode
    tokens        map[string]accessToken
    sessions      map[string]*session
    refreshTokens map[string]refreshToken
    deviceGrants  map[string]deviceGrant
}

type accessToken struct {
    User     user.User
    Expires  time.Time
    Scenario *scenario.Scenario
}

type refreshToken struct {
    User     user.User
    Scenario *scenario.Scenario
    ClientID string
    Scope    string
    Expires  time.Time
}
```

The token endpoint in `internal/server/token.go` authenticates clients, dispatches on grant type, and issues opaque tokens through helper functions. The recent device authorization implementation already extracted reusable issuance helpers:

```go
func (s *Server) issueAccessToken(u user.User, sc *scenario.Scenario, now time.Time) string
func (s *Server) issueIDToken(u user.User, sc *scenario.Scenario, clientID, nonce string, authTime, now time.Time) (string, error)
func (s *Server) issueRefreshToken(u user.User, sc *scenario.Scenario, clientID, scope string, now time.Time) string
```

The userinfo endpoint in `internal/server/userinfo.go` looks up the opaque access token and returns claims. It currently accepts only bearer authorization:

```go
auth := r.Header.Get("Authorization")
if !strings.HasPrefix(auth, "Bearer ") {
    http.Error(w, "missing bearer token", http.StatusUnauthorized)
    return
}
```

DPoP must extend the token metadata and userinfo authentication path without breaking existing bearer behavior.

## DPoP protocol pieces

### Proof JWT

A DPoP proof is a signed JWT sent in the `DPoP` HTTP header. It has a JOSE header with the public JWK and a payload with request-binding claims.

Header fields:

| Field | Meaning |
|---|---|
| `typ` | Must be `dpop+jwt`. |
| `alg` | Asymmetric signing algorithm. First implementation supports `ES256` and `RS256`. |
| `jwk` | Public key used to verify the proof. It must not contain private key material. |

Payload fields:

| Field | Meaning |
|---|---|
| `jti` | Unique proof identifier used for replay detection. |
| `htm` | HTTP method, such as `POST` or `GET`. |
| `htu` | HTTP URI of the endpoint, without query and fragment. |
| `iat` | Issued-at timestamp. The proof must be fresh. |
| `ath` | Base64url SHA-256 hash of the access token. Required when using a DPoP-bound access token at `/userinfo`; not required for initial token issuance. |

### JWK thumbprint

The authorization server computes a JWK thumbprint from the public key according to RFC 7638. This guide uses `jkt` for the base64url-encoded SHA-256 thumbprint. The exact canonical JSON differs by key type.

For `EC` P-256 keys:

```json
{"crv":"P-256","kty":"EC","x":"...","y":"..."}
```

For RSA keys:

```json
{"e":"AQAB","kty":"RSA","n":"..."}
```

The thumbprint, not the raw key, is stored with tokens:

```go
type accessToken struct {
    User     user.User
    Expires  time.Time
    Scenario *scenario.Scenario
    DPoPJKT  string
}
```

### Token response semantics

If the token request includes a valid `DPoP` header, tinyidp should issue a DPoP-bound access token and return:

```json
{
  "access_token": "opaque-access-token",
  "token_type": "DPoP",
  "expires_in": 3600,
  "scope": "openid profile email"
}
```

If no `DPoP` header is present, the existing bearer response remains unchanged:

```json
{
  "access_token": "opaque-access-token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile email"
}
```

Refresh tokens issued from a DPoP-bound flow should also be bound to the same `jkt`. A refresh request for that token must include a DPoP proof signed by the same key.

## Endpoint behavior

### Discovery

`internal/server/jwt.go` should advertise supported DPoP proof algorithms:

```go
"dpop_signing_alg_values_supported": []string{"ES256", "RS256"},
```

This metadata tells clients which proof JWT algorithms are accepted. It does not force all clients to use DPoP. DPoP remains opt-in per token request.

### Token endpoint

At `/token`, DPoP validation runs after form parsing and client authentication but before issuing tokens. The validation target is the request method and URL of `/token`.

Pseudocode:

```go
func (s *Server) dpopProofForRequest(w http.ResponseWriter, r *http.Request, accessToken string) (proof dpopProof, ok bool) {
    raw := r.Header.Get("DPoP")
    if raw == "" {
        return dpopProof{}, true // no DPoP requested
    }

    proof, err := s.validateDPoPProof(r, raw, accessToken)
    if err != nil {
        tokenError(w, 400, "invalid_dpop_proof", err.Error())
        return dpopProof{}, false
    }
    return proof, true
}
```

Grant-specific behavior:

| Grant | DPoP behavior |
|---|---|
| `authorization_code` | Optional `DPoP` proof binds newly issued access and refresh tokens. |
| `urn:ietf:params:oauth:grant-type:device_code` | Optional `DPoP` proof binds newly issued access and refresh tokens. |
| `refresh_token` | If the refresh token is DPoP-bound, a matching `DPoP` proof is required. If an unbound refresh token is refreshed with a DPoP proof, the newly issued tokens may become DPoP-bound. |

A DPoP-bound refresh-token check should look like this:

```go
proof, ok := s.dpopProofForRequest(w, r, "")
if !ok { return }

if rtok.DPoPJKT != "" {
    if proof.JKT == "" {
        tokenError(w, 400, "invalid_dpop_proof", "refresh token requires DPoP proof")
        return
    }
    if proof.JKT != rtok.DPoPJKT {
        tokenError(w, 400, "invalid_dpop_proof", "DPoP proof key does not match refresh token")
        return
    }
}

newJKT := proof.JKT
if rtok.DPoPJKT != "" {
    newJKT = rtok.DPoPJKT
}
```

### UserInfo endpoint

`/userinfo` is tinyidp's protected resource endpoint. For unbound bearer tokens, current behavior remains valid:

```http
Authorization: Bearer opaque-token
```

For DPoP-bound access tokens, `/userinfo` must require:

```http
Authorization: DPoP opaque-token
DPoP: <proof-jwt>
```

The proof must satisfy all normal proof checks and also include `ath`, the base64url SHA-256 hash of the access token. The proof key thumbprint must equal the access token's stored `DPoPJKT`.

Pseudocode:

```go
if at.DPoPJKT == "" {
    require Authorization: Bearer
    return userinfoClaims(at)
}

require Authorization: DPoP
proof := validateDPoPProof(r, header, token)
if proof.JKT != at.DPoPJKT { reject }
if proof.ATH != hashAccessToken(token) { reject }
return userinfoClaims(at)
```

## Proof validation details

### Request URI construction

`htu` binds a proof to one endpoint. tinyidp should build the expected URL from the incoming request:

```go
func requestURLWithoutQuery(r *http.Request) string {
    scheme := "http"
    if r.TLS != nil { scheme = "https" }
    if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "http" || forwarded == "https" {
        scheme = forwarded
    }
    return scheme + "://" + r.Host + r.URL.EscapedPath()
}
```

This matches local `httptest` and loopback usage. It is not a general reverse-proxy security model, which is acceptable for tinyidp's local/test scope.

### JWT parsing

The proof validator should reject malformed JWTs before touching state:

1. Split on exactly three JWT segments.
2. Base64url-decode header and payload.
3. Decode JSON objects.
4. Reject `alg=none` and unsupported algorithms.
5. Reject missing or private `jwk` fields.
6. Reconstruct the public key from JWK.
7. Verify the signature over `header.payload`.
8. Validate `typ`, `htm`, `htu`, `iat`, `jti`, and optional/required `ath`.
9. Compute `jkt`.
10. Record proof replay key.

The replay key should include the thumbprint and proof ID, for example:

```go
replayKey := proof.JKT + "\x00" + proof.JTI
```

If the same proof is replayed against a different endpoint with the same key, the same `jti` still fails. This is stricter than keying by endpoint and easier to reason about in a test IdP.

### Replay cache

Add a replay cache to `Server`:

```go
type Server struct {
    // ...
    dpopReplay map[string]time.Time
}
```

The value is the proof expiry time. Because tinyidp is in memory, an opportunistic cleanup on validation is sufficient:

```go
func (s *Server) rememberDPoPJTI(jkt, jti string, expires time.Time) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    now := time.Now()
    for key, exp := range s.dpopReplay {
        if now.After(exp) { delete(s.dpopReplay, key) }
    }
    key := jkt + "\x00" + jti
    if _, exists := s.dpopReplay[key]; exists { return false }
    s.dpopReplay[key] = expires
    return true
}
```

Proof freshness can use a small skew window, such as five minutes:

```go
const dpopProofMaxAge = 5 * time.Minute
```

A proof with `iat` too far in the past or too far in the future should fail.

## Data model changes

Add DPoP metadata to opaque token state:

```go
type accessToken struct {
    User     user.User
    Expires  time.Time
    Scenario *scenario.Scenario
    DPoPJKT  string
}

type refreshToken struct {
    User     user.User
    Scenario *scenario.Scenario
    ClientID string
    Scope    string
    Expires  time.Time
    DPoPJKT  string
}
```

Update issuance helpers:

```go
func (s *Server) issueAccessToken(u user.User, sc *scenario.Scenario, now time.Time, dpopJKT string) string
func (s *Server) issueRefreshToken(u user.User, sc *scenario.Scenario, clientID, scope string, now time.Time, dpopJKT string) string
```

Token response helpers should derive `token_type` from the binding:

```go
func tokenTypeForJKT(jkt string) string {
    if jkt != "" { return "DPoP" }
    return "Bearer"
}
```

## Testing plan

### Proof helper tests

Add focused tests for `internal/server/dpop.go`:

- valid ES256 proof validates and returns a stable JWK thumbprint;
- valid RS256 proof validates if RS256 is supported;
- wrong `htm` is rejected;
- wrong `htu` is rejected;
- stale `iat` is rejected;
- future `iat` is rejected;
- missing `jti` is rejected;
- replayed `jti` is rejected;
- `ath` mismatch is rejected when an access token is supplied;
- public JWK containing private members is rejected.

### Token endpoint tests

Add server-flow tests:

1. Discovery advertises `dpop_signing_alg_values_supported`.
2. Authorization-code token exchange with a valid proof returns `token_type: DPoP`.
3. The resulting access token is stored with `DPoPJKT`.
4. The resulting refresh token is stored with `DPoPJKT` when `offline_access` was requested.
5. Invalid proof at the token endpoint returns `invalid_dpop_proof`.
6. DPoP-bound refresh token requires a DPoP proof.
7. DPoP-bound refresh token rejects a proof from a different key.
8. DPoP-bound refresh token rotates to a DPoP-bound replacement.
9. Device-code token exchange can issue a DPoP-bound token.

### UserInfo tests

Add protected-resource tests:

1. Unbound bearer access tokens still work with `Authorization: Bearer`.
2. DPoP-bound access tokens reject missing `DPoP` proof.
3. DPoP-bound access tokens reject `Authorization: Bearer`.
4. DPoP-bound access tokens accept `Authorization: DPoP` with matching proof and `ath`.
5. DPoP-bound access tokens reject wrong key.
6. DPoP-bound access tokens reject wrong `ath`.
7. DPoP-bound access tokens reject replayed proof JWTs.

## Implementation phases

### Phase 1: data model and discovery

- Add `DPoPJKT` to `accessToken` and `refreshToken`.
- Add `dpopReplay` to `Server` and initialize it in constructors/tests.
- Add `dpop_signing_alg_values_supported` to discovery.
- Update debug reset to clear the replay cache.

### Phase 2: proof parser and verifier

- Add `internal/server/dpop.go`.
- Implement compact JWT parsing.
- Implement ES256 and RS256 JWK parsing and signature verification.
- Implement RFC 7638 JWK thumbprints.
- Implement `htm`, `htu`, `iat`, `jti`, replay, and `ath` validation.

### Phase 3: token endpoint binding

- Validate optional DPoP proof in authorization-code and device-code token exchange.
- Bind issued access tokens and refresh tokens to `jkt`.
- Return `token_type: DPoP` for bound access tokens.
- Enforce proof presence and matching key for bound refresh tokens.

### Phase 4: UserInfo enforcement

- Accept `Authorization: DPoP <token>` for DPoP-bound access tokens.
- Require matching proof key and `ath`.
- Preserve existing bearer behavior for unbound tokens.

### Phase 5: docs and smoke tests

- Update README and Glazed help pages.
- Add a `tutorial-dpop.md` or DPoP section in the reference.
- Add a manual curl or Go smoke description that creates a proof, obtains a DPoP-bound token, and calls `/userinfo`.

## Decision records

### Decision: make DPoP opt-in by `DPoP` header

- **Context:** tinyidp must keep existing browser and xgoja smokes working while adding sender-constrained token tests.
- **Options considered:** require DPoP globally; configure per client; opt in per token request.
- **Decision:** DPoP is opt-in when the token request includes a `DPoP` proof header.
- **Rationale:** This matches common OAuth client behavior and avoids introducing config surface area before tests need it.
- **Consequences:** The same client can obtain bearer or DPoP tokens depending on request headers. Tests must assert token type.
- **Status:** proposed.

### Decision: store `jkt` as opaque-token metadata

- **Context:** tinyidp access tokens are opaque random strings, not JWT access tokens.
- **Options considered:** convert access tokens to JWTs with `cnf.jkt`; keep opaque tokens and store `jkt`; add both.
- **Decision:** Keep opaque tokens and store `DPoPJKT` in the token map.
- **Rationale:** This preserves tinyidp's current token model and keeps validation local to the server.
- **Consequences:** Clients cannot introspect the binding from the access token itself. That is acceptable because tinyidp does not expose token introspection.
- **Status:** proposed.

### Decision: defer nonce support

- **Context:** RFC 9449 defines optional nonce support for servers that want additional replay hardening.
- **Options considered:** implement nonce immediately; defer nonce and use `jti` replay cache plus `iat` freshness.
- **Decision:** Defer nonce support in the first tinyidp implementation.
- **Rationale:** Nonce support adds response headers and retry loops that are useful but not necessary for local sender-constrained token tests.
- **Consequences:** The first implementation should not advertise `use_dpop_nonce` behavior. A future ticket can add it deliberately.
- **Status:** proposed.

## Review checklist

A reviewer should verify these invariants:

- A request without `DPoP` keeps existing bearer behavior.
- A request with valid `DPoP` returns `token_type: DPoP`.
- A DPoP-bound token cannot call `/userinfo` with only `Authorization: Bearer`.
- A DPoP-bound token cannot call `/userinfo` with a proof from a different key.
- Replaying the same proof fails.
- Refresh tokens preserve binding across rotation.
- Device-code token exchange and authorization-code token exchange share the same binding semantics.
- Path-based issuers do not change DPoP `htu`; validation uses the actual request endpoint URL.
