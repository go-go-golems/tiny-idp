---
Title: OIDC intern textbook
Ticket: TINYIDP-PROD-001
Status: active
Topics:
    - auth
    - go
    - identity
    - oidc
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/server/authorize.go
      Note: Concrete code reference for Authorization Code Flow explanation
    - Path: repo://internal/server/jwt.go
      Note: Concrete code reference for ID Token, JWKS, and PKCE explanation
    - Path: repo://internal/server/session.go
      Note: Concrete code reference for current IdP session and cookie behavior
    - Path: repo://internal/server/token.go
      Note: Concrete code reference for token exchange and refresh rotation explanation
    - Path: repo://internal/user/user.go
      Note: Concrete code reference for stable synthetic subject derivation
ExternalSources:
    - ../sources/01-openid-net-specs-openid-connect-core-1-0-html.md
    - ../sources/02-datatracker-ietf-org-doc-html-rfc9700.md
    - ../sources/05-openid-net-specs-openid-connect-discovery-1-0-html.md
    - ../sources/07-cheatsheetseries-owasp-org-cheatsheets-oauth2-cheat-sheet-html.md
    - ../sources/09-cheatsheetseries-owasp-org-cheatsheets-session-management-cheat-sheet-html.md
Summary: Textbook-style introduction to the internals of an embeddable OpenID Connect identity provider.
LastUpdated: 2026-07-07T14:48:25.177893634-04:00
WhatFor: Use this to onboard a new intern before they implement or review the production embedded IdP.
WhenToUse: Read before touching OIDC flows, tokens, sessions, clients, keys, consent, storage, or Fosite integration.
---


# OIDC intern textbook

## Goal

This textbook explains the internals of an OpenID Connect identity provider as they apply to `tinyidp` and the planned embedded production provider. By the end, a new engineer should understand what an IdP is responsible for, which parts are protocol mechanics, which parts are product policy, and why the production implementation must be stricter than the local mock.

The text avoids shortcuts. Each chapter starts with the concept, then grounds it in concrete request flows, state records, pseudocode, and file references from this repository.

## 1. The job of an OpenID Provider

An OpenID Provider authenticates a user and issues verifiable claims about that authentication event to a relying party. The relying party is the application that wants to know who the user is. In OAuth terms, the relying party is also a client. In OIDC terms, the identity provider is the OpenID Provider, often abbreviated OP.

The provider does not merely return a username. It returns signed data with specific protocol fields. The relying party can validate those fields without trusting the browser. This distinction is the reason ID Tokens exist: the user agent transports protocol messages, but cryptographic verification happens between the relying party and the provider's public keys.

The key vocabulary is compact:

| Term | Meaning in this project |
|---|---|
| End-user | The human authenticating at the IdP. |
| OpenID Provider / IdP / OP | The server that authenticates the user and issues ID Tokens. |
| Relying Party / RP / Client | The application that starts login and consumes tokens. |
| Authorization endpoint | Browser-facing endpoint that validates the login request and returns an authorization code. |
| Token endpoint | Back-channel endpoint where the RP exchanges a code for tokens. |
| ID Token | Signed JWT containing identity and authentication claims. |
| Access token | Token used to call an API such as UserInfo. |
| Refresh token | Long-lived secret used to obtain new tokens without another browser login. |
| JWKS | Public key set used by RPs to verify ID Token signatures. |

The provider's responsibilities are deliberately split:

- Protocol responsibilities include request validation, redirect URI enforcement, code issuance, token exchange, ID Token construction, JWKS publication, and discovery metadata.
- Product responsibilities include login UI, password or upstream authentication, consent policy, account locking, rate limiting, audit events, storage choices, key rotation policy, and operational configuration.

A good implementation keeps this split visible in code. For this project, Fosite should handle much of the protocol machinery, while `embeddedidp` owns product policy and public API shape.

## 2. The Authorization Code Flow

The production provider initially supports one browser login flow: Authorization Code Flow with PKCE. This flow keeps tokens out of the front-channel redirect. The browser receives only a short-lived authorization code. The relying party exchanges that code at the token endpoint over a server-to-server or application-to-provider request.

The sequence is:

```text
1. RP constructs an authorization request.
2. Browser navigates to IdP /authorize.
3. IdP validates the request.
4. IdP authenticates the user or reuses an existing IdP session.
5. IdP applies consent policy.
6. IdP creates a one-time authorization code.
7. Browser is redirected to the RP callback with code and state.
8. RP posts code, redirect_uri, client identity, and PKCE verifier to /token.
9. IdP consumes the code once and returns tokens.
10. RP validates the ID Token and starts its own application session.
```

The important property is that the code is not an identity assertion. A code is an intermediate credential. It is valid only for one client, one redirect URI, one short time window, and one PKCE verifier. If any of those bindings fail, the token endpoint must reject the exchange.

Pseudocode for the token endpoint looks like this:

```go
func ExchangeAuthorizationCode(req TokenRequest) (TokenResponse, error) {
    client := authenticateClient(req)
    code := store.ConsumeAuthorizationCode(hash(req.Code), now)

    if code.ClientID != client.ID {
        return errorInvalidGrant()
    }
    if code.RedirectURI != req.RedirectURI {
        return errorInvalidGrant()
    }
    if code.ExpiresAt.Before(now) {
        return errorInvalidGrant()
    }
    if !VerifyPKCES256(code.PKCEChallenge, req.CodeVerifier) {
        return errorInvalidGrant()
    }

    idToken := signer.SignIDToken(IDTokenClaims{
        Issuer: issuer,
        Subject: code.User.Sub,
        Audience: client.ID,
        IssuedAt: now,
        ExpiresAt: now.Add(idTokenTTL),
        Nonce: code.Nonce,
        AuthTime: code.AuthTime,
    })

    accessToken := tokens.CreateOpaqueAccessToken(code.GrantID, client.ID, code.User.ID)
    refreshToken := maybeCreateRefreshToken(code)
    return TokenResponse{idToken, accessToken, refreshToken}, nil
}
```

The current mock implements a handwritten version of this path. `internal/server/authorize.go` validates authorize parameters and issues the code. `internal/server/token.go` deletes the code atomically, checks client, redirect URI, and PKCE, then issues tokens. The production implementation keeps the same conceptual steps but moves protocol enforcement into Fosite and persistent stores.

Key points:

- The authorization code is bound to the original request.
- The token endpoint is where ID Token issuance occurs.
- The code must be consumed exactly once.
- The redirect URI must match exactly at both authorization and token exchange.
- PKCE binds the front-channel authorization request to the back-channel token exchange.

## 3. Clients and redirect URIs

A client is a configured relying party. The provider must know each client's identifier, redirect URIs, authentication method, allowed scopes, and PKCE requirements before accepting an authorization request.

Redirect URI validation is one of the highest-value security checks in the system. The provider must not use prefix matching, wildcard matching, or host-only matching. It should compare the full registered redirect URI string against the requested redirect URI. The current mock already does exact matching in `internal/client/client.go`, where `AllowsRedirectURI` returns true only when the requested URI equals an allowlisted string.

A production client record should contain enough data to validate both browser and token endpoint requests:

```go
type Client struct {
    ID string
    SecretHash []byte
    Public bool
    RedirectURIs []string
    PostLogoutRedirectURIs []string
    AllowedScopes []string
    RequirePKCE bool
    Disabled bool
}
```

Client validation rules:

- `ID` is non-empty and unique.
- Public clients have no secret and require PKCE.
- Confidential clients store a secret hash, not a plaintext secret.
- Redirect URIs are exact strings.
- Redirect URIs must not contain wildcards.
- Redirect URIs must not contain fragments.
- Production redirect URIs should use HTTPS except explicit loopback development clients.
- Disabled clients cannot start or complete flows.

The current built-in clients are useful for development: `dev-client`, `public-spa`, and `web-app`. Production should not create permissive clients implicitly. A production operator or embedding application must configure each client explicitly.

## 4. Scopes and claims

Scopes are requested permissions or claim groups. Claims are the actual fields emitted in an ID Token or returned from UserInfo. The provider must decide which claims a client receives based on the granted scopes and policy.

The minimum OIDC scope is `openid`. Without `openid`, the request is OAuth but not OpenID Connect. The current mock rejects authorization requests whose scope does not contain `openid`. Production should preserve that rule.

Common scopes for this project:

| Scope | Effect |
|---|---|
| `openid` | Required for OIDC; allows an ID Token with `sub`. |
| `profile` | Allows profile claims such as `name`, `preferred_username`, `locale`. |
| `email` | Allows `email` and `email_verified`. |
| `offline_access` | Allows refresh token issuance when policy permits it. |

ID Token claims always include protocol claims:

```json
{
  "iss": "https://example.com/idp",
  "sub": "user-123",
  "aud": "web-app",
  "exp": 1893456000,
  "iat": 1893452400
}
```

Optional or conditional claims include:

- `auth_time`, especially when `max_age` was used or when policy requires it.
- `nonce`, when the authorization request included a nonce.
- `email` and `email_verified`, when `email` scope is granted.
- `name`, `preferred_username`, `locale`, when `profile` scope is granted.
- `groups`, `roles`, and `tenant`, when the project policy grants them.

The important invariant is that `sub` is stable. Email is not a good subject identifier because email can change. The current mock derives synthetic stable subjects from normalized login strings in `internal/user/user.go`; production should store stable subjects in the user store.

## 5. ID Tokens and JWKS

An ID Token is a signed JWT. It has a header, payload, and signature. The header names the signing algorithm and key id. The payload contains claims. The signature covers the encoded header and payload.

A relying party validates an ID Token by checking at least:

- The signature verifies against the public key selected by `kid` from JWKS.
- `iss` exactly matches the configured issuer.
- `aud` contains the client ID.
- `exp` is in the future and `iat` is acceptable.
- `nonce` matches when the request included one.
- `auth_time` satisfies `max_age` when relevant.
- The signing algorithm is expected and not `none`.

The provider publishes public verification keys at `/jwks`. The JWKS response must never contain private key material. In production, JWKS should include the active signing key and recently retired verification keys until all tokens signed by those keys have expired.

Key lifecycle:

```text
create inactive key -> publish public key -> activate for signing -> retire from signing -> keep for verification -> remove after retention
```

The current mock deliberately uses additional JWKS behavior for tests: it publishes a rotated key and a bad-signature key, and scenarios can issue tokens with an unknown `kid` or mismatched signing key. Those are excellent relying-party tests. They must not exist in the production engine.

## 6. Access tokens and UserInfo

An access token represents permission to call a protected resource. For this project, opaque access tokens are simpler than JWT access tokens. The provider can store a hash of the token and resolve it to a user, client, grant, scopes, expiry, and optional sender-constraining data.

UserInfo flow:

```text
1. RP calls GET /userinfo with Authorization: Bearer <access_token>.
2. IdP hashes the presented token and looks it up.
3. IdP rejects unknown, expired, revoked, wrong-client, or wrong-proof tokens.
4. IdP loads user and grant data.
5. IdP returns only claims allowed by the granted scopes.
```

Pseudocode:

```go
func UserInfo(w http.ResponseWriter, r *http.Request) {
    token := bearerToken(r)
    record := tokens.LookupAccessToken(hash(token))
    if record == nil || record.ExpiresAt.Before(now) || record.RevokedAt != nil {
        unauthorized(w)
        return
    }
    user := users.Get(record.UserID)
    claims := claimsForScopes(user, record.Scope)
    writeJSON(w, http.StatusOK, claims)
}
```

The UserInfo response should be unsigned JSON in the first production version. Signed/encrypted UserInfo responses are a separate feature.

## 7. Refresh tokens and rotation

A refresh token is a sensitive credential. It can extend a session without another browser login, so it must be treated more carefully than a short-lived access token.

Production refresh-token rules:

- Store only a hash of the refresh token.
- Rotate on every successful use.
- Mark the old token as replaced by the new token.
- Detect reuse of an already consumed token.
- Revoke the token family when reuse is detected.
- Audit reuse detection.

The current mock deletes the old refresh token and issues a new one. That behavior catches simple reuse, but production needs durable token-family state.

A production record should include parent and replacement fields:

```go
type RefreshToken struct {
    TokenHash []byte
    GrantID string
    ClientID string
    UserID string
    ParentTokenHash []byte
    ReplacedByHash []byte
    CreatedAt time.Time
    ExpiresAt time.Time
    RevokedAt *time.Time
    ReuseDetectedAt *time.Time
}
```

Rotation must be a transaction:

```go
func RotateRefreshToken(oldHash []byte, next RefreshToken) error {
    tx := db.Begin()
    old := tx.GetRefreshTokenForUpdate(oldHash)

    if old == nil {
        tx.Rollback()
        return ErrInvalidGrant
    }
    if old.ReplacedByHash != nil || old.RevokedAt != nil {
        tx.MarkReuseDetected(oldHash, now)
        tx.RevokeFamily(old.GrantID, now)
        tx.Commit()
        return ErrRefreshReuseDetected
    }

    next.ParentTokenHash = oldHash
    tx.InsertRefreshToken(next)
    tx.MarkRefreshTokenReplaced(oldHash, next.TokenHash, now)
    tx.Commit()
    return nil
}
```

Key points:

- Rotation is not a read followed by an unrelated write. It is one transaction.
- Reuse detection requires remembering consumed tokens.
- Audit must not log the raw token.

## 8. Sessions and cookies

The IdP session is different from the relying party session. The IdP session says the browser has authenticated at the provider. The RP session says the browser is logged in to an application. The provider sets the IdP session cookie. The RP sets its own application cookie after validating the ID Token.

Production IdP session cookie rules:

```text
HttpOnly: true
Secure: true
SameSite: Lax or Strict
Path: issuer path
Value: random opaque handle only
```

Server-side session record:

```go
type Session struct {
    IDHash []byte
    UserID string
    AuthTime time.Time
    CreatedAt time.Time
    LastSeenAt time.Time
    ExpiresAt time.Time
    ACR string
    AMR []string
    RevokedAt *time.Time
}
```

Why the cookie contains only a handle: the provider can revoke the session, rotate server-side metadata, and avoid putting user data into the browser. The server stores a hash of the handle so a database read does not expose usable session IDs.

The current mock cookie is `HttpOnly` and `SameSite=Lax`, but intentionally not `Secure` because it serves plain loopback HTTP. Production mode must require `Secure` and HTTPS issuer URLs.

## 9. Login and consent

OIDC defines the protocol shape, but it does not dictate how the provider authenticates the user. Login is product behavior. It may be password-based, upstream-OIDC-based, LDAP-backed, WebAuthn-based, or implemented by the host application embedding the provider.

The production API should represent login as an interface:

```go
type LoginHandler interface {
    RenderLogin(w http.ResponseWriter, r *http.Request, ar AuthorizationRequest, data LoginPageData) error
    Authenticate(ctx context.Context, r *http.Request) (domain.User, error)
}
```

Consent is also product policy. Some embedded applications will skip consent because the IdP and RP are part of one installed product. Other deployments should remember consent per user/client/scope.

```go
type ConsentPolicy interface {
    RequireConsent(ctx context.Context, user User, client Client, scopes []string) (bool, error)
    RecordConsent(ctx context.Context, user User, client Client, scopes []string) error
}
```

The production default should be `RememberConsent`. The development default can be `AlwaysSkipConsent`.

Login and consent forms must have CSRF protection. OAuth `state` protects the relying party's callback flow. It does not replace CSRF protection for the provider's own form POSTs.

## 10. Discovery metadata

Discovery tells clients where endpoints are and what the provider supports. It is a contract. If discovery advertises a feature, clients may use it.

Production discovery should be conservative. If the provider does not support implicit flow, discovery must not advertise it. If production disables plain PKCE, discovery must not advertise `plain`. If the provider only signs ID Tokens with RS256, discovery must not advertise other algorithms.

Minimal production metadata:

```json
{
  "issuer": "https://example.com/idp",
  "authorization_endpoint": "https://example.com/idp/authorize",
  "token_endpoint": "https://example.com/idp/token",
  "userinfo_endpoint": "https://example.com/idp/userinfo",
  "jwks_uri": "https://example.com/idp/jwks",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "code_challenge_methods_supported": ["S256"]
}
```

The current mock supports path-based issuers by registering routes at root and under the issuer path. That is useful and should be preserved through a shared `internal/oidcmeta` package.

## 11. Audit events

Audit events are structured security records. They are not debug logs and they are not raw request dumps. The provider should emit events for decisions that matter during incident response.

Required event names:

```text
login.success
login.failure
authorize.request.accepted
authorize.request.rejected
consent.granted
token.authorization_code.issued
token.refresh.rotated
token.refresh.reuse_detected
client.created
client.updated
client.disabled
key.created
key.activated
key.retired
session.created
session.revoked
```

Do not log:

- Raw authorization codes.
- Raw access tokens.
- Raw refresh tokens.
- Raw passwords.
- Raw client secrets.
- Full session cookie values.

Do log:

- Event time.
- Event name.
- Client ID when known.
- User ID or subject when known.
- Request ID.
- Remote address after trusted proxy processing.
- Result and reason category.
- Hash or prefix of a secret only when correlation is necessary and safe.

## 12. Mock engine versus production engine

The mock engine is valuable because it is controllable. It can issue bad ID Tokens, slow responses, wrong claims, missing claims, bad JWKS, and debug state. Those behaviors make relying parties better.

The production engine is valuable because it is strict. It should reject ambiguous configuration, expose only supported features, persist keys, store secrets safely, and make negative paths reliable.

The split is not optional. If one engine tries to do both jobs, either the mock becomes less useful or production becomes too easy to misconfigure.

```text
mock engine
  accepts local HTTP issuers
  allows loopback debug routes
  supports synthetic users and scenarios
  can intentionally break tokens and JWKS
  stores state in memory

production engine
  requires HTTPS issuer in production
  exposes no debug routes
  uses real user/login/consent stores
  never intentionally emits malformed protocol artifacts
  stores keys, codes, tokens, grants, and sessions durably
```

## 13. Reading the current repository

Start with these files in order:

1. `README.md` explains the product boundary and explicitly says the current server is not production-grade.
2. `internal/cmds/serve.go` shows how CLI settings become a server.
3. `internal/sections/oidc/settings.go` shows reusable configuration decoding.
4. `internal/server/server.go` shows in-memory state and route registration.
5. `internal/server/authorize.go` shows authorize/login/session/code issuance.
6. `internal/server/token.go` shows code exchange, refresh rotation, and token response headers.
7. `internal/server/jwt.go` shows discovery, JWKS, signing, and PKCE verification.
8. `internal/server/session.go` shows current cookie/session behavior.
9. `internal/client/client.go` shows exact redirect URI matching and built-in clients.
10. `internal/scenario/scenario.go` shows why mock behavior needs a separate engine.

After reading, run:

```bash
go test ./...
go run ./cmd/tinyidp print-config
go run ./cmd/tinyidp help reference
```

## 14. What to remember

- The IdP authenticates users and issues signed identity claims.
- The RP validates ID Tokens and owns its own application session.
- Authorization codes are short-lived, one-time, and bound to client, redirect URI, and PKCE.
- ID Tokens are signed JWTs; access tokens can be opaque.
- Refresh tokens must rotate and reuse must be detected.
- Discovery metadata must advertise only what production supports.
- JWKS contains public verification keys only.
- Production signing keys must persist across restart.
- Mock-only features are not production features guarded by a flag; they belong in the mock engine.
- The production provider should fail closed when configuration is incomplete or unsafe.

## Related

- Design and implementation guide: `../design-doc/01-production-embeddable-idp-design-and-implementation-guide.md`.
- Downloaded source index: `../sources/README.md`.
