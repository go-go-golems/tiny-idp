---
Title: Strict Provider Review Findings and Fixes
Ticket: TINYIDP-PR3-REVIEW-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/serve.go
      Note: Seeded strict dev passwords are converted into real credentials
    - Path: repo://internal/fositeadapter/provider.go
      Note: Production token secret enforcement and disabled-client filtering
    - Path: repo://internal/fositeadapter/session.go
      Note: max_age=0 fresh-auth semantics
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Disabled clients are rejected on direct strict lookup and persisted requester restore
    - Path: repo://internal/oidcmeta/discovery.go
      Note: Strict discovery omits unimplemented end_session endpoint
    - Path: repo://pkg/embeddedidp/options.go
      Note: Embedded production validation requires a token secret
ExternalSources:
    - https://github.com/go-go-golems/tiny-idp/pull/3
Summary: Textbook-style report explaining the PR 3 strict-provider review findings, the security model behind each issue, and the implemented fixes.
LastUpdated: 2026-07-09T12:00:00-04:00
WhatFor: Use this report to understand why each PR 3 review finding mattered and how the implementation fixes preserve strict IdP invariants.
WhenToUse: Read when reviewing PR 3, changing strict provider storage/session/discovery behavior, or extending production configuration/admin flows.
---


# Strict Provider Review Findings and Fixes

## 1. Purpose

This report explains the review fixes made for PR 3. Each finding was small in code size, but each one touched an invariant that a strict OpenID Provider must preserve: disabled clients must stop working, production secrets must not silently fall back to development values, configured fixture passwords must mean what they say, discovery metadata must describe real endpoints, and `max_age=0` must request fresh authentication rather than silent reuse.

The useful way to read these fixes is not as isolated patches. They are boundary checks. A strict provider receives input from clients, browser cookies, persisted protocol state, configuration files, and discovery metadata. Each input is useful only if the provider revalidates it at the point where it becomes security-relevant.

## 2. The Review Findings as Invariants

The PR review identified five behaviors:

| Finding | Broken invariant | Fixed in |
| --- | --- | --- |
| Disabled SQLite clients could still be returned to Fosite | Administrative disable must make a client unusable for new and persisted protocol paths | `internal/fositeadapter/sqlstore.go`, `provider.go` |
| Production provider accepted a missing token secret | Production must never install hard-coded development HMAC/session/CSRF secrets | `internal/fositeadapter/provider.go`, `pkg/embeddedidp/options.go` |
| Strict dev seeded users ignored configured fixture passwords | If a seeded fixture declares a password, strict dev login should verify it | `internal/cmds/serve.go` |
| Strict discovery advertised `/end-session` without mounting it | Discovery metadata must not advertise unimplemented strict endpoints | `internal/oidcmeta/discovery.go` |
| `max_age=0` silently reused sessions | `max_age=0` means fresh authentication is required | `internal/fositeadapter/session.go` |

The common pattern is that every trusted shortcut needed a second look. A value that was safe during development can become unsafe once admin commands and production storage exist.

## 3. Disabled Clients: Revalidate Administrative State at Protocol Boundaries

A disabled client is not deleted. It remains in storage so operators can inspect it, re-enable it, and preserve historical audit references. That means every runtime path that loads a client must ask an additional question: is this client still active?

The first direct lookup is straightforward. The SQL-backed Fosite store now maps missing or disabled clients to `fosite.ErrNotFound`:

```go
func (s *sqlFositeStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
    c, err := s.project.GetClient(ctx, id)
    if err != nil || c.Disabled {
        return nil, fosite.ErrNotFound
    }
    return s.toFositeClient(ctx, c)
}
```

The less obvious path is persisted protocol state. Authorization codes, PKCE records, OIDC sessions, access tokens, and refresh tokens store a serialized requester. That requester contains a Fosite client snapshot. If the provider restores the snapshot without checking the current domain client, then an operator can disable the client after authorization and the client can still redeem the code or refresh token.

The fix is to restore the requester, then revalidate the current client state before returning it:

```go
func (s *sqlFositeStore) restoreActiveRequester(ctx context.Context, b []byte) (fosite.Requester, error) {
    req, err := restoreRequester(b)
    if err != nil {
        return nil, err
    }
    client := req.GetClient()
    if client == nil || client.GetID() == "" {
        return nil, fosite.ErrNotFound
    }
    domainClient, err := s.project.GetClient(ctx, client.GetID())
    if err != nil || domainClient.Disabled {
        return nil, fosite.ErrNotFound
    }
    return req, nil
}
```

This revalidation rule is the important part. Persisted protocol state is a record of what happened earlier. It is not permission to ignore the current administrative state.

The regression test issues an authorization code, disables the client in SQLite, and then verifies that token exchange fails. This test covers the path that would not be caught by checking only `GetClient` during authorization.

## 4. Production Secrets: Development Defaults Must Stop at the Mode Boundary

The strict provider uses one secret key for HMAC-related provider behavior such as token strategy configuration, CSRF signing, and browser-session handle hashing. A development default is useful for tests and local runs. In production it is dangerous because every deployment that omits the secret would share the same predictable value.

The provider now rejects production mode when the key is missing or shorter than 32 bytes:

```go
if opts.Mode == domain.ProductionMode && len(opts.SecretKey) < 32 {
    return nil, fmt.Errorf("production mode requires a token secret key of at least 32 bytes")
}
if len(opts.SecretKey) == 0 {
    opts.SecretKey = []byte("tinyidp-dev-secret-key-at-least-32-bytes")
}
```

The embedded API performs the same check before construction:

```go
if mode == ProductionMode {
    if len(o.Token.SecretKey) < 32 {
        return fmt.Errorf("production mode requires token secret key of at least 32 bytes")
    }
}
```

This creates a clear mode boundary. Development can remain convenient. Production must be explicit.

The tests cover both the provider-level constructor and the embedded API validation boundary. That matters because embedders normally call `embeddedidp.New`, while internal tests and future product wiring may call the strict adapter directly.

## 5. Seeded Passwords: Fixture Data Must Be Enforced When Present

The mock engine has long supported seeded users with optional passwords. An empty password keeps the permissive development behavior. A non-empty password is a fixture contract: a test author intentionally declared a credential.

Before this fix, strict dev mode imported the user's profile into the store but ignored `sc.Password`. The password authenticator saw no credential and took the dev-mode passwordless path. The result was surprising: a fixture that looked password-protected accepted any password.

The strict dev startup path now creates a password credential whenever a seeded scenario declares a password:

```go
passwords, err := authn.NewPasswordService(st, authn.Options{})
for _, sc := range scenarios.All() {
    u := domain.User{ID: sc.User.Sub, Sub: sc.User.Sub, ...}
    _ = st.PutUser(ctx, sc.Name, u)
    if sc.Password != "" {
        credential, err := passwords.HashCredential(u.ID, sc.Name, []byte(sc.Password), time.Now().UTC())
        _ = st.PutPasswordCredential(ctx, credential)
    }
}
```

The invariant is precise:

- If no seeded credential exists in dev mode, strict login may remain passwordless for compatibility.
- If a seeded credential exists, strict login must verify it.
- Production mode already requires real credentials and does not use the passwordless compatibility branch.

The regression test builds a strict provider from a seeded `alice` fixture. A wrong password returns `401`; the configured password produces an authorization code.

## 6. Discovery Metadata: Advertise Only Implemented Strict Endpoints

Discovery metadata is a contract with clients. If the strict discovery document says `end_session_endpoint` exists, clients are entitled to call it. The strict adapter did not mount `/end-session`, so discovery advertised a URL that returned 404.

There are two valid fixes:

1. implement strict RP-initiated logout;
2. omit the metadata field until strict logout exists.

This PR chooses the second option because it is the smallest correct change. The mock engine still has its own logout behavior. The strict metadata now describes only strict endpoints that are actually mounted:

```go
return Discovery{
    Issuer:                iss.String(),
    AuthorizationEndpoint: iss.Endpoint("/authorize"),
    TokenEndpoint:         iss.Endpoint("/token"),
    UserinfoEndpoint:      iss.Endpoint("/userinfo"),
    JWKSURI:               iss.Endpoint("/jwks"),
    // no end_session_endpoint until strict adapter mounts /end-session
}
```

The test asserts that strict production discovery omits `end_session_endpoint`. When strict logout is implemented later, this test should be changed alongside the route and logout behavior tests.

## 7. `max_age=0`: Fresh Authentication Means No Silent Session Reuse

`max_age` asks the provider to limit how old the user's authentication may be. `max_age=0` is the strictest version: the relying party is asking for fresh authentication now. The old helper treated non-positive values as if no constraint existed, so a session could be silently reused even when the client requested fresh login.

The helper now treats only missing, invalid, or negative values as unconstrained. Zero is evaluated normally:

```go
maxAge, err := strconv.ParseInt(maxAgeValue, 10, 64)
if err != nil || maxAge < 0 {
    return true
}
return !authTime.Add(time.Duration(maxAge) * time.Second).Before(time.Now().UTC())
```

For an existing session, `authTime.Add(0)` is before the current time, so the helper returns false and the provider renders the login form. The regression test logs in, reuses the session, then sends `max_age=0` and expects the interaction form rather than a redirect with a code.

## 8. Similar-Issue Sweep

While addressing the review comments, I looked for related instances of the same patterns:

- **Persisted requester paths:** The disabled-client fix was applied to the shared requester restore path, not only to the direct `GetClient` method. That covers authorization code, PKCE, OIDC, access-token, and refresh-token session loading.
- **Request-object redirect validation:** `clientAllowsRedirect` now rejects disabled clients as well as missing clients.
- **Memory Fosite store startup:** disabled clients are skipped when the in-memory Fosite client store is built, so a disabled client is not active at provider startup.
- **Production validation boundary:** both `fositeadapter.NewProvider` and `embeddedidp.Options.Validate` reject missing production token secrets.
- **Metadata consistency:** strict discovery now avoids advertising an endpoint absent from `fositeadapter.registerAt`.

The remaining known limitation is dynamic client disabling for the in-memory Fosite client store after provider startup. The production/admin path is SQLite-backed and now revalidates through the project store. In-memory mode is still a development/test path, and disabled clients are skipped at startup.

## 9. Validation

The following commands passed after the fixes:

```bash
make lint
make logcopter-check
go test ./...
scripts/run-conformance.sh
```

Focused tests added or updated:

- `TestFositeSQLiteDisabledClientRejectsPersistedAuthorizationCode`
- `TestProductionProviderRejectsMissingSecretKey`
- `TestProductionValidationRejectsMissingTokenSecret`
- `TestStrictProviderHonorsSeededUserPassword`
- `TestProductionDiscoveryOmitsUnimplementedEndSessionEndpoint`
- `TestBrowserSessionSilentAuthorizeAndPromptNone` now covers `max_age=0`

## 10. Key Points

- Disabled clients must be rejected at direct lookup time and when restoring persisted protocol sessions.
- Production mode must reject missing token secrets before any provider installs development defaults.
- Seeded passwords are part of fixture semantics; strict dev mode must enforce them when present.
- Discovery metadata is not documentation only. It is machine-readable behavior clients will follow.
- `max_age=0` is not the absence of a constraint. It is the strongest fresh-authentication constraint.

## References

- PR: `https://github.com/go-go-golems/tiny-idp/pull/3`
- `internal/fositeadapter/sqlstore.go`
- `internal/fositeadapter/provider.go`
- `internal/fositeadapter/session.go`
- `internal/cmds/serve.go`
- `internal/oidcmeta/discovery.go`
- `pkg/embeddedidp/options.go`
