---
Title: go-go-goja Extraction Recommendation from the tinyidp-xapp Device API
Ticket: TINYIDP-XAPP-DEVICE-001
Status: active
Topics:
    - architecture
    - xgoja
    - oauth2
    - security
    - durable-objects
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp-xapp/development_app.go
      Note: Actual generated-host service composition evidence.
    - Path: cmd/tinyidp-xapp/device_api.go
      Note: Application-owned API/actor mapping that must not be extracted.
    - Path: cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go
      Note: Candidate Go-only opaque bearer primitive.
    - Path: cmd/tinyidp-xapp/internal/xgojaruntime/xgoja_runtime.gen.go
      Note: Generated runtime artifact showing the ConfigureServices integration point consumed by the host.
    - Path: cmd/tinyidp-xapp/production_app.go
      Note: Production-shaped host composition evidence.
ExternalSources: []
Summary: Evidence-based decision on reusable go-go-goja primitives versus application-owned identity policy in the xapp device API.
LastUpdated: 2026-07-16T00:00:00Z
WhatFor: Avoid premature framework extraction while identifying the smallest verified primitive a second host can validate.
WhenToUse: When a second xgoja application needs an opaque OAuth bearer resource API or when planning a go-go-goja feature ticket.
---



# go-go-goja Extraction Recommendation from the tinyidp-xapp Device API

## Decision

Do **not** extract the entire device-authentication stack into `go-go-goja`.
The embedded provider, OAuth client registration, device CLI, durable-object
BBS contract, identity mapping, state files, and audit event names are product
policy. Making them framework defaults would make one application's choices
look portable when they are not.

The implementation does prove one promising Go-only primitive: a host-owned
opaque bearer resource-server authenticator. It validates an RFC 7662 response,
returns a constrained principal, and never exposes the resource-client secret
or raw bearer credential to JavaScript. The recommended next step is a focused
`go-go-goja` ticket to validate that core with a second host before freezing a
public API.

## Evidence from the xapp

`development_app.go` composes the application through existing go-go-goja and
go-go-objects host services:

1. It constructs an auth-aware `gojahttp.Host`.
2. `xgojaruntime.Options.ConfigureServices` installs it via
   `httpprovider.ExternalHostService`.
3. It installs `hostauth.ServiceFactoryKey`, the Durable Objects gateway, and
   the actor-bound dispatcher service.
4. Generated JavaScript routes use browser session identity through
   `gojahttp.ActorFromContext`.
5. The Go mux mounts the embedded provider, login UI assets, generated browser
   routes, and native bearer API as separate responsibilities.

```text
existing framework primitives                         xapp-owned policy
-----------------------------                         -----------------
ConfigureServices                                    tiny-idp bootstrap
ExternalHostService                                  client IDs/audiences
hostauth.ServiceFactoryKey                           device CLI and cache
Durable Object gateway + bound dispatcher            BBS routes/JSON/audit
gojahttp ActorFromContext                             OIDC subject mapping
```

This is the desired trust division. Browser identity is established by existing
host auth. Machine identity is established by native Go code. JavaScript
declares application behavior but does not hold the bearer credential or
introspection credential.

Phase 5 supplies behavioral evidence for that boundary:

- Playwright exercises browser login, a CSRF-protected post, and logout.
- Alice and Bob device tokens create posts with distinct verified subjects.
- A valid token without `bbs.post.create` gets `403` and cannot mutate the
  durable object.
- Multiple/malformed bearer headers get `401` before dispatch.
- Wrong audience is rejected at device authorization.
- A password-security change revokes an unobserved device token at the real
  resource API.
- Initialized TLS mode proves discovery, approval, token polling,
  introspection, and bearer post composition.

The evidence proves a native resource-server seam. It does not prove that the
framework should own route names, scopes, persistent state, an OAuth provider,
or durable-object identity semantics.

## What already belongs in go-go-goja

No framework code change is required for the browser-session/durable-object
composition. These existing abstractions are sufficient and should be used as
the documented pattern:

| Primitive | Proven role | Recommendation |
| --- | --- | --- |
| `xgojaruntime.Options.ConfigureServices` | The host installs trusted services before runtime creation. | Keep as central composition seam. |
| `httpprovider.ExternalHostService` | Go owns mux/listener lifecycle needed by an embedded IdP and native API. | Keep. |
| `hostauth.ServiceFactoryKey` | Browser OIDC/session configuration stays in Go rather than JS. | Keep. |
| Durable Objects gateway | Generated routes can reach durable objects. | Keep. |
| Bound dispatcher | Browser actor comes from trusted context, not caller data. | Keep. |
| `gojahttp.ActorFromContext` | Host retrieves verified browser actor. | Keep; it is intentionally not bearer authentication. |

The correct extraction conclusion is therefore narrow: add no second wrapper
around host services merely because this application combines them.

## Candidate primitive: `pkg/xgoja/oidcresource`

The local `cmd/tinyidp-xapp/internal/resourceauth` component has a reusable
core. It consumes host-owned configuration and yields only a minimal principal:

```go
type Config struct {
    IssuerURL    string
    ClientID     string
    ClientSecret []byte
    Audience     string
    HTTPClient   *http.Client
    PositiveCacheTTL time.Duration
    NegativeCacheTTL time.Duration
}

type Principal struct {
    Subject   string
    ClientID  string
    Scopes    []string
    ExpiresAt time.Time
}

type Result struct {
    Outcome   Outcome // authenticated, unauthorized, forbidden, unavailable
    Principal Principal
}
```

The reusable guarantees are:

- accept exactly one valid Bearer authorization header;
- validate discovery and retain an issuer-bound introspection endpoint;
- authenticate the resource server with host-only RFC 7662 Basic credentials;
- require active status, exact issuer, Bearer type, expected audience, subject,
  unexpired token, and route scopes;
- cache only HMAC-derived token keys, with positive entries bounded by both
  token expiry and a short maximum, and short definitive-inactive caching;
- return coarse outcomes so consumers do not create an invalid-token oracle;
- fail closed when introspection is unavailable.

The proposed first package is intentionally Go-only:

```text
pkg/xgoja/oidcresource/
    authenticator.go   Config, Authenticator, Principal, Outcome
    discovery.go       constrained discovery validation
    introspection.go   RFC 7662 client and response validation
    cache.go           HMAC-keyed bounded decision cache
    http.go            optional outcome-to-response helper
    *_test.go          fake issuer and security matrix
```

It must not import tiny-idp, Fosite, xgoja JavaScript runtime types, or Durable
Objects. It should depend only on `net/http` and allow either an in-process or
TLS transport. A host remains responsible for native endpoint ownership and
its own actor mapping:

```go
result := auth.Authenticate(ctx, r.Header.Values("Authorization"), []string{"bbs.post.create"})
if result.Outcome != oidcresource.OutcomeAuthenticated {
    oidcresource.WriteAuthorizationFailure(w, result.Outcome)
    return
}
dispatchApplicationObject(ctx, result.Principal.Subject, input)
```

Do not create a JavaScript bearer-context bridge in this extraction. That would
put opaque credentials in the runtime and blur browser session identity with
machine credential identity. A future JS middleware need would require a
separate threat model.

## Deliberately non-extracted pieces

| Component | Why it remains local |
| --- | --- |
| Embedded tiny-idp bootstrap/client registration | Redirect URIs, scopes, client IDs, and allowed audiences are product/issuer policy. |
| Device CLI | User experience, cache location, cancellation, scope set, and current `audience` parameter convention are not framework semantics. |
| Native BBS handler | Paths, JSON limits, scopes, data model, and audit event names are application API contract. |
| Subject-to-actor mapping | Tenancy, privacy, and durable-object policy are application-owned. |
| State manifest/secrets | Backup, rotation, ownership, and migration are deployment policy. |
| Login UI | Branding and provider interaction UX belong to tiny-idp/idpui and the product. |

## Extraction gates and tasks

### Gate A — second consumer

- Identify a second generated Go host with a native opaque-bearer API.
- Compare discovery, client authentication, audience representation, scopes,
  cache requirements, and audit requirements.
- Keep the implementation local if the public API would gain options useful to
  only one consumer.

### Gate B — pure Go extraction

- Move only the local `resourceauth` core after preserving its security test
  matrix.
- Use neutral names and standard-library types.
- Verify raw bearer values never occur in errors, logs, cache keys, or test
  snapshots.
- Add vectors from both consumers, including unavailable provider and expiry.

### Gate C — host examples

- Demonstrate an in-process issuer transport and a TLS HTTP transport.
- Keep browser auth on existing `hostauth` services and bearer auth in native
  Go handlers.
- Prove no cookie fallback, no caller-selected actor, and fail-closed `503`.

### Gate D — release

- Review cancellation, timeout, cache, revocation-latency, and API naming.
- Define versioning and security-advisory ownership.
- Publish migration notes only after the second consumer confirms the exported
  API has no hidden xapp assumptions.

## Risks reviewers must retain

- A positive cache creates a bounded revocation visibility delay after a token
  has already been accepted. The xapp lifecycle test uses an unobserved token
  so it proves provider invalidation rather than cache expiration.
- Discovery validation must continue to reject a cross-origin provider endpoint
  by default.
- Generic code should return structured outcomes, not impose application audit
  names or log protocol data.
- tiny-idp's `audience` device parameter is client-flow interoperability
  policy; it must not leak into the resource-server package.

## Conclusion

The current go-go-goja primitives are sufficient for an elegant embedded IdP,
browser OIDC, and durable-object application. The next reusable candidate is a
small host-owned opaque bearer verification component, not a framework-owned
identity stack. Keep the code local until a second host validates the public
contract; then extract only that narrow Go security boundary.

## Review references

- `cmd/tinyidp-xapp/development_app.go`
- `cmd/tinyidp-xapp/production_app.go`
- `cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go`
- `cmd/tinyidp-xapp/device_api.go`
- `reference/01-implementation-diary.md`
