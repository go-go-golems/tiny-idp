---
Title: "Tutorial: xgoja personal-inbox smokes with tinyidp"
Slug: tutorial-xgoja-personal-inbox
Short: "Use tinyidp as the OIDC provider for go-go-goja personal-inbox Steps 06, 07, and 08 with root or path-based issuers."
Topics:
- oidc
- testing
- xgoja
- go-go-goja
Commands:
- serve
- print-config
Flags:
- issuer
- addr
- users-file
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows how to use tinyidp as the OIDC provider for the `go-go-goja` personal knowledge inbox examples. It covers Step 06 browser login, Step 07 Alice/Bob inbox isolation, and Step 08 device-token capture isolation.

The device authorization endpoints in the current xgoja Step 08 example are implemented by the generated xgoja host. tinyidp supplies browser-login OIDC behavior for that app-owned flow. tinyidp also has its own native OAuth Device Authorization Grant endpoints; see `tinyidp help tutorial-device-authorization` when you want tinyidp itself to be the device authorization server.

## Repositories

The tutorial assumes these repository paths:

    tinyidp:   /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp
    go-go-goja:/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja

Adjust paths if your checkout lives elsewhere.

## Step 1 — inspect the tinyidp fixture

The personal-inbox fixture lives in the tinyidp repo:

    examples/users/personal-inbox-users.yaml

It defines Alice and Bob with stable subjects, emails, fixture passwords, groups, roles, tenant, preferred usernames, and locale. The important values are:

| User | Password | Subject | Email | Role |
|---|---|---|---|---|
| `alice` | `alice-password` | `user-alice-fixed` | `alice@example.test` | `writer` |
| `bob` | `bob-password` | `user-bob-fixed` | `bob@example.test` | `reader` |

The smoke helpers submit those passwords. If you use a different users file, update the helper arguments or the fixture values.

## Step 2 — understand the ports

Each step uses separate ports to avoid conflicts:

| Step | tinyidp addr | generated app addr |
|---|---|---|
| Step 06 | `127.0.0.1:19087` | `127.0.0.1:19794` |
| Step 07 | `127.0.0.1:19088` | `127.0.0.1:19795` |
| Step 08 | `127.0.0.1:19089` | `127.0.0.1:19796` |

The Makefiles build tinyidp, start it, start the generated app, run the browser smoke, and clean up both processes.

## Step 3 — run Step 06 with a root issuer

From the Step 06 directory:

    cd /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak

Run:

    make tinyidp-smoke \
      TINYIDP_ROOT=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp \
      TINYIDP_USERS_FILE=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml

Expected output ends with:

    ok tinyidp login smoke; session email=alice@example.test
    ok tinyidp replacement smoke

This proves the generated app can complete browser OIDC login, establish an app session, and access the authenticated inbox API.

## Step 4 — run Step 06 with a path-based issuer

Use a Make command-line variable for `TINYIDP_ISSUER`. The Makefile defines `TINYIDP_ISSUER := ...`, so shell environment assignment is not enough to override it reliably.

    make tinyidp-smoke \
      TINYIDP_ROOT=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp \
      TINYIDP_USERS_FILE=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml \
      TINYIDP_ISSUER=http://127.0.0.1:19087/realms/personal-inbox

Expected output is the same successful smoke. The path issuer checks that discovery, authorize, token, userinfo, JWKS, and logout URLs can be advertised under an issuer path.

## Step 5 — run Step 07 Alice/Bob isolation

Step 07 proves that authenticated browser API state is scoped by user.

    cd /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox

Root issuer:

    make tinyidp-smoke \
      TINYIDP_ROOT=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp \
      TINYIDP_USERS_FILE=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml

Path issuer:

    make tinyidp-smoke \
      TINYIDP_ROOT=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp \
      TINYIDP_USERS_FILE=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml \
      TINYIDP_ISSUER=http://127.0.0.1:19088/realms/personal-inbox

Expected output:

    ok tinyidp alice/bob inbox isolation
    ok tinyidp isolation smoke

The smoke uses two independent browser sessions. Alice captures an item, Bob captures another item, and each user lists only their own item.

## Step 6 — run Step 08 device-token capture isolation

Step 08 adds the generated host's app-owned device authorization flow. In this example, tinyidp is still used for browser login rather than for its native `/device_authorization` endpoint.

    cd /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/08-device-authorization

Root issuer:

    make tinyidp-smoke \
      TINYIDP_ROOT=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp \
      TINYIDP_USERS_FILE=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml

Path issuer:

    make tinyidp-smoke \
      TINYIDP_ROOT=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp \
      TINYIDP_USERS_FILE=/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml \
      TINYIDP_ISSUER=http://127.0.0.1:19089/realms/personal-inbox

Expected output:

    ok tinyidp device capture isolation
    ok tinyidp device authorization smoke

The test starts device flows through the generated app, approves them through browser sessions, captures through programmatic tokens, and verifies ownership isolation.

## Step 7 — diagnose failures

The most common failure is issuer mismatch:

    oidc: issuer URL provided to client (...) did not match the issuer URL returned by provider (...)

Check for stale processes before changing code:

    pgrep -af tinyidp
    ss -ltnp | grep -E '19087|19088|19089|19794|19795|19796'

Kill stale processes, then rerun the smoke.

If the smoke fails with `401 Unauthorized`, the helper probably did not submit the fixture password expected by the users file. The current helpers submit `alice-password` and `bob-password` for the checked-in personal-inbox users fixture.

If `users file ... no such file` appears, pass an absolute `TINYIDP_USERS_FILE`. The Makefiles run from the xgoja example directories, not from the tinyidp repository root.

## What you learned

The xgoja smokes exercise tinyidp as a complete local OIDC dependency, not only as a static discovery server. Step 06 proves browser login. Step 07 proves user-scoped application data. Step 08 proves that xgoja-owned device authorization can coexist with tinyidp-owned browser login. Running each step with both root and path issuers proves that path-based issuer support is routing-compatible without changing claims.

## See also

- `tinyidp help user-guide` — operational tinyidp usage.
- `tinyidp help tutorial-seeded-users-and-claims` — the users fixture model.
- `tinyidp help reference` — endpoint and config reference.
- `tinyidp help developer-guide` — implementation details for route mounting and seeded users.
