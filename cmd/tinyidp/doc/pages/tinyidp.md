---
Title: "tinyidp"
Slug: tinyidp
Short: "A mock OpenID Connect Identity Provider for local development and integration testing."
Topics:
- oidc
- testing
- identity
Commands:
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

tinyidp is a minimal mock OpenID Connect Identity Provider written in Go.
It exists to replace Keycloak-in-Docker for local development and
integration testing of applications that act as OIDC Relying Parties.

## What it is

tinyidp implements the OIDC happy path: discovery, JWKS, authorize, token,
and userinfo. It issues RS256-signed ID tokens, supports the
authorization_code grant with optional PKCE (S256 and plain), and derives
synthetic users from any typed login so you can test "different
authenticated principals" without an account database.

It also ships a scenario registry that reproduces real OIDC client bugs:
authorization errors, token exchange failures, malformed ID tokens, and
broken userinfo responses.

## What it is not

tinyidp is not production grade. It has no real login, consent, persistent
keys, refresh tokens, revocation, logout, or TLS enforcement. Bind to
loopback (the default) and never expose it publicly.

## Getting started

Run the server:

    tinyidp serve

Then point your OIDC client at `http://localhost:5556` with
`client_id=dev-client` and `scopes=openid profile email`.

## Configuration

Configuration is layered using the Glazed command framework, with
precedence (low to high): section defaults, profiles, config files,
environment variables (`TINYIDP_*`), CLI flags.

The OIDC section (`tinyidp help oidc-config`) groups the provider
settings. Run `tinyidp serve --print-parsed-fields` to inspect the
resolved configuration.

## See also

- `tinyidp help oidc-config`
- `tinyidp help profiles`
- `tinyidp serve --help`
