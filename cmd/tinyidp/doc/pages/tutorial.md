---
Title: "Tutorial: testing a relying party with scenarios"
Slug: tutorial
Short: "Drive a relying party through the happy path, then through failure scenarios, to learn tinyidp's testing model."
Topics:
- oidc
- testing
- scenarios
Commands:
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial walks through the testing model that makes tinyidp useful:
run a flow, observe the happy path, then switch a single input — the
login name — and watch your relying party handle a real failure. By the
end you will know how to reproduce any of tinyidp's scenarios against
your own RP without changing the server or the client configuration.

The tutorial assumes you have completed `tinyidp help getting-started`:
tinyidp is running on `http://localhost:5556`, and your RP is configured
with `client_id=dev-client` and `redirect_uri=http://localhost:3000/callback`.

## The model, in one sentence

A scenario is a named bundle of behavior attached to a synthetic user.
You select a scenario by logging in as its name. The scenario decides
what happens at each endpoint — whether the authorize step fails, whether
the token is malformed, whether the userinfo response is broken — so a
single login change exercises a different code path in your RP.

## Step 1 — establish the happy path

Start the server:

    ./tinyidp serve

Trigger a login from your RP. At tinyidp's login page, type `alice` and
submit. Your RP receives an authorization code, exchanges it for an ID
token, and verifies the signature against `/jwks`. The login succeeds.

This is your baseline. Every subsequent step changes exactly one thing:
the name you type at the login page. The server, the client, and the RP
configuration stay identical.

## Step 2 — an expired ID token

Real ID tokens expire. A robust RP validates `exp` and rejects tokens
whose expiry is in the past. tinyidp's `id-expired` scenario produces
exactly that: an ID token whose `exp` claim is one hour in the past.

Log out of your RP, then trigger a fresh login. At the login page, type
`id-expired` (or click the button labeled `id-expired` under "ID token
failures"). tinyidp issues a code and your RP exchanges it for tokens —
but the ID token it receives has `exp` set to the past.

Observe your RP. A correct RP rejects the token during ID-token
validation, before establishing a session. If your RP logs the user in,
it has a bug: it is not checking `exp`.

What happened: tinyidp's scenario registry looked up the `id-expired`
entry, which carries a `MutateClaims` hook that rewrites `exp` after the
token is built. The signature is valid; the claim is wrong. That
distinction matters — it lets you test validation separately from
signature verification.

## Step 3 — a broken userinfo response

Some RPs fetch claims from the userinfo endpoint rather than (or in
addition to) the ID token. The `userinfo-sub-mismatch` scenario returns a
userinfo response whose `sub` differs from the ID token's `sub`. A
correct RP must detect the mismatch and refuse to trust the userinfo.

Log out, trigger a login, and type `userinfo-sub-mismatch`. Your RP gets
a valid ID token, then fetches `/userinfo`. The userinfo response is
well-formed and signed-looking, but its `sub` does not match the token.

Observe your RP. A correct RP rejects the userinfo response. The
scenario isolates this one failure — the ID token is valid, the userinfo
shape is valid, only the `sub` binding is broken.

## Step 4 — refresh-token rotation and reuse

When your RP requests the `offline_access` scope, tinyidp issues a
refresh token alongside the access token. Refresh tokens rotate: each
use deletes the presented token and issues a new one. Reusing a rotated
token must fail.

Configure your RP to request `scope=openid offline_access`, log in as
`alice`, and let your RP refresh the access token once. Then trigger a
second refresh with the *old* refresh token (the one your RP already
used). tinyidp returns `invalid_grant`.

Observe your RP. A correct RP treats `invalid_grant` on refresh as a
signal to re-authenticate, not as a transient error to retry. If your RP
loops on the stale token, it is not honoring rotation.

## Step 5 — a JWKS outage

Relying parties cache JWKS, but they must also handle the case where the
key endpoint fails. tinyidp can make `/jwks` return a 500, return an
empty key set, or hang — toggled at runtime through the debug UI, not
through a scenario, because `/jwks` is global rather than tied to a
login.

In a second terminal, put the JWKS endpoint into failure mode:

    curl -s -X POST http://localhost:5556/debug/jwks-mode \
      -H 'Content-Type: application/json' -d '{"mode":"500"}'

Now any JWKS fetch your RP makes returns 500. Observe your RP: it should
fall back to a cached key if it has one, or fail closed if it does not.
Restore normal operation when you are done:

    curl -s -X POST http://localhost:5556/debug/jwks-mode \
      -H 'Content-Type: application/json' -d '{"mode":"normal"}'

## What you now know

You have exercised four distinct failure classes — a malformed ID token
claim, a broken userinfo binding, refresh-token reuse, and a JWKS
outage — each by changing a single input or toggling a single mode. That
is the whole testing model: scenarios select per-user behavior at the
endpoints, and the debug UI selects global endpoint behavior.

The full catalog of scenarios, with the field that controls each one,
is in `tinyidp help scenarios`. The endpoints, configuration, and
client model are in `tinyidp help reference`.

## See also

- `tinyidp help getting-started` — install and first login.
- `tinyidp help tutorial-first-rp-login` — first relying-party login walkthrough before scenario testing.
- `tinyidp help tutorial-seeded-users-and-claims` — deterministic users, fixture passwords, and generic claims.
- `tinyidp help tutorial-device-authorization` — OAuth device-code approval and polling.
- `tinyidp help tutorial-dpop` — DPoP sender-constrained access-token tests.
- `tinyidp help tutorial-xgoja-personal-inbox` — xgoja personal-inbox Steps 06, 07, and 08.
- `tinyidp help user-guide` — operational guide for configuring tinyidp.
- `tinyidp help scenarios` — the full scenario catalog and model.
- `tinyidp help reference` — endpoints, clients, and configuration.
