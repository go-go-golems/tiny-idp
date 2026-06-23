---
Title: "Getting started"
Slug: getting-started
Short: "Install tinyidp, run it, and complete your first OIDC login in five minutes."
Topics:
- oidc
- testing
- identity
Commands:
- serve
- print-config
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

tinyidp is a minimal mock OpenID Connect Identity Provider written in Go.
It exists to replace Keycloak-in-Docker for local development and
integration testing of applications that act as OIDC Relying Parties. This
page takes you from a clean checkout to a verified ID token in five
minutes, and points you at the deeper documentation once you are oriented.

## What it is

tinyidp implements the OIDC happy path — discovery, JWKS, authorize,
token, and userinfo — and issues RS256-signed ID tokens. It supports the
`authorization_code` grant with optional PKCE (S256 and plain), derives
synthetic users from any typed login so you can test different principals
without an account database, and ships a scenario registry that
reproduces the failure modes real relying parties must handle: expired
tokens, wrong audiences, broken userinfo, missing JWKS keys, and more.

Beyond the happy path it also models the behaviors a real IdP exposes:
multiple clients (public, confidential, and permissive), IdP sessions
with `prompt` and `max_age`, refresh tokens with rotation, multi-key
JWKS, and RP-initiated logout.

## What it is not

tinyidp is not production grade. It performs no real authentication,
stores no persistent keys, and enforces no TLS. Bind it to loopback (the
default) and never expose it publicly. It is a test tool: its value is
determinism and failure coverage, not hardening.

## Prerequisites

- Go 1.25 or later.
- An application that acts as an OIDC Relying Party, configured to talk
  to an issuer URL you control.

## Step 1 — build and run

From the repository root:

    go build -o tinyidp ./cmd/tinyidp
    ./tinyidp serve

The server starts on `http://localhost:5556`. Leave it running in one
terminal; the examples below assume it is reachable at that URL.

## Step 2 — confirm discovery

A relying party discovers its provider by fetching the OpenID
configuration. Confirm the endpoint responds:

    curl -s http://localhost:5556/.well-known/openid-configuration | jq .issuer

The `issuer` is `http://localhost:5556`, and the document advertises every
endpoint tinyidp implements, including `end_session_endpoint`.

## Step 3 — point your relying party at tinyidp

Configure your RP with:

    issuer:        http://localhost:5556
    client_id:     dev-client
    client_secret: (leave empty — dev-client is public)
    redirect_uri:  http://localhost:3000/callback
    scope:         openid profile email

The `dev-client` is a permissive builtin: it accepts the default
redirect URIs, does not require PKCE, and allows every scope. It is the
right client for a first run.

## Step 4 — log in as alice

When your RP redirects you to `/authorize`, tinyidp shows a login page.
Type `alice` and submit. tinyidp issues an authorization code, your RP
exchanges it for tokens, and you are logged in. The ID token is signed
with an RS256 key whose public half is published at `/jwks`.

The login page also lists every scenario as quick-pick buttons, grouped
by category. Each scenario reproduces a specific behavior — a normal
user, a claim variant, or a failure. You select a scenario by logging in
as its name.

If your integration tests need fixed subjects or custom claims for names
such as `alice` and `bob`, start tinyidp with `--users-file ./users.yaml`.
The users file overrides or adds normal login scenarios without changing
the relying party configuration.

## Step 5 — inspect what was issued

tinyidp exposes a loopback-only debug UI. In another terminal:

    curl -s http://localhost:5556/debug | jq .
    curl -s http://localhost:5556/debug/tokens | jq .

You see the issued access token (as an 8-character prefix), its subject,
and its expiry — enough to correlate a flow against the IdP's internal
state without adding log statements.

## Where to go next

- `tinyidp help tutorial` — a guided walkthrough that exercises the
  happy path and then a failure scenario, to learn the testing model.
- `tinyidp help scenarios` — the full catalog of scenarios and the model
  behind them.
- `tinyidp help reference` — configuration, clients, endpoints, and
  behaviors, organized for lookup.

## See also

- `tinyidp serve --help`
- `tinyidp print-config` — print the resolved provider configuration.
