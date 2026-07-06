---
Title: "Tutorial: seeded users, fixture passwords, and claims"
Slug: tutorial-seeded-users-and-claims
Short: "Create deterministic users with fixture passwords and generic authorization claims, then verify token and userinfo behavior."
Topics:
- oidc
- testing
- identity
- claims
Commands:
- serve
- print-config
Flags:
- users-file
- client-id
- redirect-uris
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows how to replace synthetic login-derived users with deterministic seeded users. Seeded users are useful when a test needs stable subject identifiers, fixed emails, fixture passwords, and predictable authorization claims.

By the end, you will have a users file with Alice and Bob, a running tinyidp instance that loads it, and a clear understanding of how fixture passwords and generic claims reach ID tokens and userinfo.

## Step 1 — create a users file

Create `users.yaml`:

    users:
      - login: alice
        password: alice-password
        sub: user-alice-fixed
        email: alice@example.test
        name: Alice Inbox
        email_verified: true
        groups: [inbox-users, engineering]
        roles: [writer]
        tenant: personal
        preferred_username: alice
        locale: en-US

      - login: bob
        password: bob-password
        sub: user-bob-fixed
        email: bob@example.test
        name: Bob Inbox
        email-verified: true
        groups: [inbox-users]
        roles: [reader]
        tenant: personal
        preferred_username: bob
        locale: en-US

The two email-verified spellings are both accepted: `email_verified` and `email-verified`. This lets JSON-style and YAML-style fixtures remain readable.

## Step 2 — start tinyidp with the users file

Run:

    tinyidp serve \
      --issuer http://127.0.0.1:19087 \
      --addr 127.0.0.1:19087 \
      --client-id dev-client \
      --redirect-uris http://localhost:3000/callback \
      --users-file ./users.yaml

If you use a config file, remember that `oidc.users-file` is resolved relative to the process working directory. Run from the directory containing `users.yaml` or pass an absolute path.

## Step 3 — inspect resolved config

Before starting a long-running test, confirm the resolved configuration:

    tinyidp print-config \
      --issuer http://127.0.0.1:19087 \
      --addr 127.0.0.1:19087 \
      --client-id dev-client \
      --redirect-uris http://localhost:3000/callback \
      --users-file ./users.yaml

The output should include:

    issuer: http://127.0.0.1:19087
    client_id: dev-client
    users_file: ./users.yaml

`print-config` resolves the same OIDC section as `serve`, so it is the safest way to check flags, env vars, profiles, and config files before starting the server.

## Step 4 — log in as Alice

Configure your RP with:

    issuer:       http://127.0.0.1:19087
    client_id:    dev-client
    redirect_uri: http://localhost:3000/callback
    scope:        openid profile email

Trigger login. At tinyidp's login form, submit:

    login:    alice
    password: alice-password

The password matters because the seeded Alice fixture defines one. If the form submits no password, or submits a different value, tinyidp returns:

    HTTP 401 invalid login or password

A failed password attempt creates no session and no authorization code.

## Step 5 — check claims through your RP

After login, inspect what your RP received. The exact mechanism depends on the RP, but the ID token and userinfo should agree on the seeded claims.

Expected identity claims:

| Claim | Value |
|---|---|
| `sub` | `user-alice-fixed` |
| `email` | `alice@example.test` |
| `name` | `Alice Inbox` |
| `email_verified` | `true` |

Expected generic authorization claims:

| Claim | Value |
|---|---|
| `groups` | `['inbox-users', 'engineering']` |
| `roles` | `['writer']` |
| `tenant` | `personal` |
| `preferred_username` | `alice` |
| `locale` | `en-US` |

The same values should appear in `/userinfo` when your RP calls it with the access token.

## Step 6 — verify Bob is a different principal

Log out and repeat the flow as Bob:

    login:    bob
    password: bob-password

Bob should have a different subject and different role:

    sub:   user-bob-fixed
    email: bob@example.test
    roles: [reader]

Use this pattern for isolation tests. If your application stores records by authenticated subject, Alice and Bob should see different data even though both were authenticated by the same tinyidp instance.

## Step 7 — override generic fields with raw claims

The generic fields are conveniences. The raw `claims` map remains the final override.

    users:
      - login: alice
        roles: [writer]
        claims:
          roles: [owner]
          feature_flags: [compact-inbox]

The emitted `roles` claim is `[owner]`, because explicit `claims` entries override generic helper fields with the same claim name. This rule is useful when a test needs an unusual shape or app-specific value.

## Step 8 — omit claims deliberately

Use `omit_claims` when a seeded user should lack a base claim:

    users:
      - login: no-email-fixture
        omit_claims: [email, email_verified]

This creates a positive scenario for applications that must handle missing profile claims. It is different from an invalid-token scenario: the user is real, but the claims are absent.

## What you learned

Seeded users are compiled into normal scenarios. That means fixture identity, fixture passwords, generic claims, raw claims, and omitted claims all flow through the same OIDC path as built-in scenarios. The HTTP handlers do not need a special seeded-user branch; they only consult the scenario selected by login.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Login as Alice returns 401. | Wrong or missing fixture password. | Submit `alice-password` or remove `password` from the fixture. |
| RP still sees synthetic `sub`. | Users file was not loaded or path was wrong. | Check `tinyidp print-config` and use an absolute users-file path. |
| Claims appear in ID token but not in userinfo. | RP may not be calling userinfo, or it may be caching old state. | Clear RP session and check tinyidp `/debug/tokens`. |
| Raw `claims.roles` replaced top-level `roles`. | Explicit claims intentionally override generic helpers. | Remove the explicit claim or use a different claim name. |

## See also

- `tinyidp help user-guide` — seeded-user field overview.
- `tinyidp help reference` — complete seeded-user schema.
- `tinyidp help scenarios` — built-in scenario catalog.
- `tinyidp help tutorial-device-authorization` — use seeded-user fixture passwords to approve device requests.
- `tinyidp help tutorial-xgoja-personal-inbox` — seeded users in the xgoja personal-inbox smokes.
