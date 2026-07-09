---
Title: 'tiny-idp as an OIDC Identity Provider for Jitsi Meet: Analysis, Design, and Implementation Guide'
Ticket: TINYIDP-JITSI-001
Status: active
Topics:
    - oidc
    - jitsi
    - authentication
    - research
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/serve.go
      Note: serve engines + client registry + users-file (how to register the adapter client)
    - Path: repo://internal/domain/claims.go
      Note: ClaimsForScopes - the user claims mapped into the Jitsi JWT
    - Path: repo://internal/keys/keys.go
      Note: RS256 signing + JWKS shape (why JWKS can't feed Jitsi directly)
    - Path: repo://internal/oidcmeta/discovery.go
      Note: OIDC discovery metadata tiny-idp advertises (endpoints an adapter consumes)
    - Path: repo://ttmp/2026/07/09/TINYIDP-JITSI-001--use-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet/scripts/01-oidc-smoke.sh
      Note: Verified full OIDC auth-code probe against tiny-idp
    - Path: repo://ttmp/2026/07/09/TINYIDP-JITSI-001--use-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet/scripts/02-oidc-to-jitsi-jwt.py
      Note: Verified OIDC->Jitsi-JWT claim-mapping demo
    - Path: repo://ttmp/2026/07/09/TINYIDP-JITSI-001--use-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet/sources/web/01-lib-jitsi-meet-tokens.md
      Note: Canonical Jitsi mod_auth_token JWT/ASAP spec
    - Path: repo://ttmp/2026/07/09/TINYIDP-JITSI-001--use-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet/sources/web/13-jitsi-oidc-adapter-adapter-ts.txt
      Note: Recommended adapter source (OIDC endpoints it calls)
ExternalSources:
    - https://github.com/jitsi/lib-jitsi-meet/blob/master/doc/tokens.md
    - https://jitsi.github.io/handbook/docs/devops-guide/secure-domain/
    - https://github.com/jitsi-contrib/jitsi-oidc-adapter
    - https://github.com/nordeck/jitsi-keycloak-adapter
    - https://github.com/jitsi/jitsi-meet/issues/16576
Summary: Analysis of whether tiny-idp can act as the identity provider for Jitsi Meet, the design of the required OIDC-to-Jitsi-JWT adapter, and a step-by-step implementation guide for a new intern.
LastUpdated: 2026-07-09T13:00:00-04:00
WhatFor: Teach a new engineer how Jitsi authentication works, prove tiny-idp is a usable OIDC IdP for it, and give a concrete, validated integration plan.
WhenToUse: When integrating tiny-idp (or any OIDC IdP) with Jitsi Meet, or when evaluating Jitsi's authentication options.
---


# tiny-idp as an OIDC Identity Provider for Jitsi Meet

**Analysis · Design · Implementation Guide (intern edition)**

> How to read this document. Sections 1–3 are the *why* and the *background primer* — read
> them even if you only skim the rest. Sections 4–5 prove what each system actually does today,
> with file and source references you can open yourself. Sections 6–9 are the *design*: the
> component you will build/deploy, its data shapes, its API, and the decisions behind them.
> Sections 10–14 are the *how*: a phased build plan, tests, risks, and an onboarding checklist.
> Every claim that comes from source is cited as `(→ sources/web/NN-*.md)` or `(→ path:line)`.

---

## 1. Executive Summary

**The question.** Can we use **tiny-idp** — our minimal, fosite-shaped OpenID Connect (OIDC) provider —
as the identity provider (IdP) that logs users into **Jitsi Meet** (self-hosted video conferencing)?

**The answer.** **Yes — but not by pointing Jitsi directly at tiny-idp.** Jitsi Meet has **no native
OIDC login**. It authenticates users with a **Jitsi-specific JWT** that its XMPP server (Prosody)
validates through the `mod_auth_token` module. That JWT has a *different shape* from a standard OIDC ID
token (it needs `room`, a tenant-scoped `sub`, an `aud`/`iss` that match Prosody's configured app id, and
a `context.user` object), and it is delivered to Jitsi through a *different channel* (a `?jwt=` **query
string** parameter, not an OIDC redirect fragment). Because of this, **every** production OIDC-with-Jitsi
setup in the wild inserts a small **translation adapter** between the IdP and Jitsi. This is an inherent
property of Jitsi, **not a limitation of tiny-idp**.

**What tiny-idp provides that the adapter needs, and already has today** (verified live — see §5.1 and
`scripts/01-oidc-smoke.output.txt`):

- OIDC **discovery** at `/.well-known/openid-configuration`
- **Authorization-Code** flow (`/authorize` → `/token`)
- a **UserInfo** endpoint returning `sub`, `name`, `preferred_username`, `email`
- a registrable **client** with a redirect URI and scopes

That is the entire contract the recommended adapter (`jitsi-contrib/jitsi-oidc-adapter`) consumes
(→ `sources/web/13-jitsi-oidc-adapter-adapter-ts.txt`). **No new tiny-idp features are required for a
working integration.**

**Recommendation.** Deploy the standalone **`jitsi-contrib/jitsi-oidc-adapter`** (a ~200-line Deno
service) as the shim. Wire Prosody to `token` auth with a shared `app_secret`; set the adapter's
`JWT_APP_SECRET` equal to it; point `config.tokenAuthUrl` at the adapter; point the adapter's
`OIDC_ISSUER_URL` at tiny-idp. tiny-idp's RS256 signing and JWKS are **never used by Prosody** in this
design, which neatly sidesteps Jitsi's biggest OIDC gap (it cannot read JWKS — see §4.4).

**Maturity / risk posture.** tiny-idp is explicitly **"not production-grade"** (→ `tiny-idp/README.md`;
in-memory keys/sessions/refresh tokens, loopback-only intent). It is an excellent choice for **local dev
and integration testing** of the Jitsi login flow, and a reasonable reference IdP for a controlled
internal deployment, but it is **not** a hardened public IdP. The integration architecture below is
identical whether the IdP is tiny-idp, Keycloak, or Authentik — so you can prototype against tiny-idp and
swap later with zero adapter changes.

---

## 2. Problem Statement and Scope

### 2.1 Goal

Let a user click "join meeting", be redirected to **tiny-idp** to log in (as `alice`/`bob` or a seeded
user), and land back in a Jitsi room as an authenticated participant — with their display name and email
shown, and (optionally) moderator rights.

### 2.2 In scope

- Understanding Jitsi's authentication model end to end (web client → Prosody → Jicofo/JVB).
- Understanding tiny-idp's OIDC surface and token/claim shapes.
- Designing the **OIDC → Jitsi-JWT adapter** integration (deploy-existing vs. build-our-own).
- A concrete claim-mapping specification and validated experiments.
- A phased implementation and test plan an intern can execute.

### 2.3 Out of scope

- Hardening tiny-idp for public production (separate tickets: `TINYIDP-PROD-001`, `TINYIDP-USERS-001`).
- Full Jitsi deployment/ops (TURN, JVB scaling, TLS certificates) — we assume a working Jitsi.
- SAML, LDAP, or non-OIDC IdP paths.

### 2.4 Success criteria

1. An unauthenticated user hitting a room is redirected to tiny-idp, authenticates, and joins.
2. The participant's `name`/`email` come from tiny-idp claims.
3. The flow is reproducible from scripts in this ticket.
4. Moderator/room-scoping behavior is documented and demonstrated.

---

## 3. Technology Primer (read this before the design)

You need four concepts: **OIDC**, **Jitsi's component model**, **JWT**, and **Jitsi's ASAP token scheme**.

### 3.1 OpenID Connect (OIDC) in 90 seconds

OIDC is an identity layer on top of OAuth 2.0. The **relying party (RP)** — the app that wants to know
who the user is — redirects the browser to the **IdP**. After login, the IdP hands the RP an **ID token**
(a signed JWT of *who the user is*) and an **access token** (a bearer credential for *calling APIs like
UserInfo*). The **Authorization Code** flow used here is:

```
Browser        RP (adapter)                       IdP (tiny-idp)
  │  GET /login    │                                   │
  │──────────────▶ │  302 to /authorize?response_type=code&client_id&redirect_uri&scope&state
  │◀───────────────┤                                   │
  │  GET /authorize?...  ─────────────────────────────▶│  (renders login form)
  │  POST login+password ─────────────────────────────▶│  (checks credentials)
  │◀──────────────────────  302 redirect_uri?code=… ───┤
  │  GET redirect_uri?code=… │                         │
  │──────────────▶ │  POST /token (code) ─────────────▶│  issues id_token+access_token
  │                │◀──────────────  {id_token, access_token} 
  │                │  GET /userinfo (access_token) ────▶│  returns user claims
  │                │◀──────────────  {sub,name,email…} ─┤
```

Key OIDC endpoints, all advertised by **discovery** (`/.well-known/openid-configuration`):
`authorization_endpoint`, `token_endpoint`, `userinfo_endpoint`, `jwks_uri`. tiny-idp exposes all of them
(§5.1).

### 3.2 Jitsi Meet's component model

Jitsi Meet is not one process. The pieces that matter for auth:

| Component | Role | Auth relevance |
|-----------|------|----------------|
| **lib-jitsi-meet** (browser) | The web client. | Connects to Prosody over **BOSH/WebSocket**; carries the JWT as `?token=`/`?jwt=`. Never speaks OIDC. |
| **Prosody** (XMPP server) | Signaling backbone; owns authentication. | `authentication = "token"` + `mod_auth_token` validate the JWT. This is the integration surface. |
| **Jicofo** (focus) | Allocates conferences. | An "admin" XMPP user; **must NOT require a token** when token auth is on. |
| **JVB** (videobridge) | Media relay. | Authenticates to the internal `auth.` domain with `internal_hashed`; not token-driven. |

**Load-bearing fact:** authentication is enforced by **Prosody**, and the credential is a **JWT**. The
token is checked in two places (→ `sources/web/01-lib-jitsi-meet-tokens.md`): (a) when the client opens the
BOSH/WebSocket connection (`?token=` query param, anonymous SASL), and (b) when a MUC room is
created/joined, where `mod_token_verification` compares the JWT's `room` and `sub` to the actual room and
tenant/domain.

### 3.3 JWT (JSON Web Token) refresher

A JWT is `base64url(header) . base64url(payload) . base64url(signature)`. The **header** names the
algorithm (`alg`) and, for asymmetric keys, a key id (`kid`). Signature options relevant here:

- **HS256** — symmetric HMAC with a **shared secret**. Both signer and verifier hold the same secret.
- **RS256** — asymmetric RSA. Signer holds the private key; verifier fetches the **public key**.

tiny-idp signs its ID tokens with **RS256** (→ `tiny-idp/internal/keys/keys.go:83`). The Jitsi adapters
mint the Jitsi JWT with **HS256** by default (→ `sources/web/15-jitsi-oidc-adapter-config-ts.txt`).

### 3.4 Jitsi's ASAP token scheme (the sharp edge)

Jitsi supports two ways for Prosody to trust a JWT:

1. **`app_secret`** — an HS256 shared secret in `prosody.cfg.lua`. Simple; used by all adapters.
2. **`asap_key_server`** — for RS256. Prosody takes the JWT header's `kid`, computes **`sha256hex(kid)`**,
   and downloads a **PEM public key** from `{asap_key_server}/{sha256hex(kid)}.pem`
   (→ `sources/web/01-lib-jitsi-meet-tokens.md`). **There is no JWKS support** — confirmed by open issue
   jitsi/jitsi-meet #15182 (§4.4). Keys are cached until Prosody restarts (no rotation, no revocation).

This is why tiny-idp's standards-compliant **JWKS** cannot be handed to Jitsi directly, and why the
recommended design avoids RS256-to-Prosody entirely.

---

## 4. Current-State Evidence — Jitsi side

All claims here are backed by the captures in `sources/web/`.

### 4.1 Prosody authentication modes (→ 06, 07)

`authentication = "..."` per XMPP VirtualHost:

- `anonymous` / `jitsi-anonymous` — default, no login.
- `internal_plain` / `internal_hashed` — local accounts ("Secure Domain"); also how Jicofo/JVB always
  authenticate to `auth.<domain>`.
- **`token`** — JWT via `mod_auth_token` + `token_verification`. **This is our integration mode.**

### 4.2 The Jitsi JWT claim contract (→ 01)

Validation claims (checked by Prosody):

| Claim | Meaning | Notes |
|-------|---------|-------|
| `iss` | application id | Must be in `asap_accepted_issuers`; commonly equals `app_id`. Default is **no longer `*`** — must be set. |
| `aud` | audience | Must be in `asap_accepted_audiences`. |
| `sub` | tenant (lowercase) or base domain, or `*` | For `TENANT1/ROOM` URLs → `tenant1`; for a full MUC → base domain. |
| `room` | room name (not the full JID), or `*` | `conference1@muc.server.net` → `conference1`. |
| `exp`, `nbf`, `iat` | RFC 7519 time claims | Honored. |

Display/analytics claims (NOT validated; surfaced to the client only if `mod_presence_identity` is on):

```json
"context": {
  "group": "analytics-string",
  "user": { "id": "…", "name": "John Doe", "email": "jdoe@example.com", "avatar": "https://…" }
}
```

**Gotcha (→ 01):** every field in `context.user` must be a **valid string** — `null`/number throws in
lib-jitsi-meet. Our mapping (§7) coerces everything to strings.

### 4.3 No native OIDC; a bridge is required (→ 03)

Jitsi issue #16576 ("Native OIDC redirect support") is **open/unimplemented**. An OIDC IdP returns its
`id_token` in the URL **fragment** (`#id_token=…`) or via `form_post`; **Jitsi reads neither** — it wants a
**Jitsi-shaped JWT in the query string** (`?jwt=…`). Fragments never reach the server, so nginx cannot
rewrite them. Therefore a **server-side adapter** is mandatory today.

### 4.4 No JWKS support (→ research summary, issue #15182)

jitsi/jitsi-meet #15182 ("Support fetching JWT keys in JWKS format") is **open**. Jitsi only supports the
ASAP **PEM-by-kid** scheme (§3.4). This closes the door on "just give Jitsi tiny-idp's `jwks_uri`".

### 4.5 The redirect hook: `tokenAuthUrl` (→ 08, 12, 05)

`config.js` fields:

- `config.tokenAuthUrl = 'https://<host>/oidc/auth?state={state}'` — where Jitsi sends unauthenticated
  users. Placeholders: **`{room}`** and **`{state}`** (a JSON blob with room/tenant/client-type).
- `config.tokenAuthUrlAutoRedirect = true` — redirect automatically instead of showing a login button.
- `config.enableUserRolesBasedOnToken = true` — derive moderator/roles from the token's context.

The adapter at `tokenAuthUrl` eventually redirects the browser back to
`https://<host>/<tenant>/<room>?jwt=<Jitsi-JWT>`, and Jitsi feeds that JWT into Prosody.

### 4.6 Existing adapters (→ 04, 05, 08, 09, 10, 12, 13, 14, 15)

All converge on the same shape: **run the OIDC auth-code flow, read UserInfo, mint a fresh HS256
Jitsi-JWT, redirect with `?jwt=`.**

| Adapter | Runtime | OIDC endpoints used | Recommended? |
|---------|---------|---------------------|--------------|
| **jitsi-contrib/jitsi-oidc-adapter** | Deno | **discovery → authorize, token, userinfo** (no JWKS, no ID-token sig check) | **Yes — generic, provider-agnostic.** |
| nordeck/jitsi-keycloak-adapter v2 | Deno | Keycloak URLs hardcoded (no discovery) | Keycloak-only; superseded. |
| MarcelCoding/jitsi-openid | Rust | OIDC discovery (`ISSUER_URL`) | Good drop-in alternative. |

**Crucial (→ 13):** the recommended adapter **does not** validate the IdP's ID-token signature and **does
not** call the IdP's JWKS. It relies on the code exchange + UserInfo. So tiny-idp's RS256/JWKS is
irrelevant to correctness here — only discovery/authorize/token/userinfo matter.

---

## 5. Current-State Evidence — tiny-idp side

tiny-idp is a Go OIDC provider built on the Glazed CLI framework, labelled **not production-grade**
(→ `tiny-idp/README.md`). Default engine is `mock` (→ `tiny-idp/internal/cmds/serve.go:99`); a stricter
`fosite` engine also exists (`serve.go:114`).

### 5.1 Endpoints (verified live)

Running `scripts/01-oidc-smoke.sh` against a fresh instance produced (full transcript in
`scripts/01-oidc-smoke.output.txt`):

```
GET /.well-known/openid-configuration → issuer, authorization_endpoint, token_endpoint,
                                        userinfo_endpoint, jwks_uri, response_types_supported=["code"],
                                        grant_types=["authorization_code","refresh_token"],
                                        code_challenge_methods=["S256"], id_token_signing=["RS256"]
GET /jwks                             → 3 RSA keys (dev-key-1, rotated-key-2, bad-sig-key), RS256
GET /authorize?…  + POST login        → 302 redirect_uri?code=…&state=st-123
POST /token (authorization_code)      → { token_type:Bearer, id_token, access_token, expires_in:3600 }
GET /userinfo (Bearer access_token)   → user claims
```

Endpoint definitions live in `tiny-idp/internal/server/*` (e.g. `server.go:142` mounts `/authorize`) and
the discovery struct is `tiny-idp/internal/oidcmeta/discovery.go`.

### 5.2 Token / claim shapes (verified live)

The ID token tiny-idp issued for seeded user `alice` (decoded):

```json
{ "iss": "http://127.0.0.1:15573", "aud": "dev-client", "sub": "user-alice-fixed",
  "name": "Alice Inbox", "preferred_username": "alice", "email": "alice@example.test",
  "email_verified": true, "groups": ["inbox-users"], "roles": ["writer"], "tenant": "personal",
  "locale": "en-US", "nonce": "nonce-xyz", "auth_time": …, "iat": …, "exp": … }
```

UserInfo returns the same **user** claims (without protocol claims). Claim assembly is
`tiny-idp/internal/domain/claims.go` (`ClaimsForScopes`): `sub` always; `email`/`email_verified` with the
`email` scope; `name`/`preferred_username`/`groups`/`roles`/`tenant`/`locale` with the `profile` scope.

### 5.3 Users and clients

- **Seeded users** (`--users-file`, e.g. `examples/users/personal-inbox-users.yaml`): pin
  `sub`/`email`/`name`/`password`/`groups`/`roles` and arbitrary raw `claims`. When `password` is set it is
  enforced at `/authorize` (→ README; `internal/server/authorize.go:175 passwordAccepted`).
- **Built-in clients:** `dev-client` (public), `public-spa` (PKCE-required), `web-app` (confidential,
  `dev-secret`). `--client-id/--client-secret/--redirect-uris` register or merge a client
  (`internal/cmds/serve.go:160 buildClientRegistry`).

### 5.4 What tiny-idp has that we will NOT use for Jitsi

Device grant, DPoP, refresh tokens, RP-initiated logout, JWKS/RS256 verification. They are irrelevant to
the browser auth-code flow the adapter drives. (Logout could be wired separately later — §12.)

---

## 6. System Overview and Architecture

### 6.1 Components and responsibilities

| Component | Responsibility | We deploy? |
|-----------|----------------|------------|
| **tiny-idp** | Authenticate the user; issue OIDC id_token/UserInfo. | Yes (already have it). |
| **jitsi-oidc-adapter** | Run OIDC auth-code flow; map claims; mint HS256 Jitsi-JWT; redirect `?jwt=`. | Yes (deploy off-the-shelf). |
| **Prosody** | Validate the Jitsi-JWT (`app_secret`, `room`, `sub`, `aud`). | Configure. |
| **Jitsi Meet web + Jicofo + JVB** | The meeting. | Assumed running. |
| **nginx** | Reverse-proxy `/oidc/*` → adapter. | Configure. |

### 6.2 Boundary diagram

```
                       ┌─────────────────────────────────────────────────────────┐
                       │                    Jitsi host (nginx)                     │
   Browser             │   /            → Jitsi Meet web (config.js)              │
  (lib-jitsi-meet)     │   /oidc/*      → jitsi-oidc-adapter (Deno) ───────────┐  │
      │                │   xmpp-websocket → Prosody (mod_auth_token, app_secret)│  │
      │                └────────────────────────────────────────────────────┼──┼──┘
      │   tokenAuthUrl                                                        │  │ shared app_secret
      │   redirect ─────────────────────────────────────────────────────────┘  │ (HS256)
      │                                                                          │
      │            OIDC auth-code flow (discovery/authorize/token/userinfo)      │
      └───────────────────────────────▶  tiny-idp  ◀───────────────────────────┘
```

The **trust boundary** that matters: Prosody trusts the **adapter** (via the shared HS256 `app_secret`),
and the adapter trusts **tiny-idp** (via the OIDC code exchange). Prosody never talks to tiny-idp.

---

## 7. Data / Claim-Mapping Design

This is the heart of the integration: turning tiny-idp's OIDC claims into a valid Jitsi JWT. Demonstrated
live by `scripts/02-oidc-to-jitsi-jwt.py` (output in `scripts/02-oidc-to-jitsi-jwt.output.txt`).

### 7.1 Mapping table

| Jitsi JWT field | Source (tiny-idp) | Rule |
|-----------------|-------------------|------|
| `iss` | (config) `JWT_APP_ID` | Constant; must match Prosody `app_id`/`asap_accepted_issuers`. |
| `aud` | (config) `JWT_APP_ID` | Constant; must match `asap_accepted_audiences`. |
| `sub` | Jitsi `state` (tenant) or base domain | Injected from the room URL, **not** from the OIDC `sub`. `*` for all. |
| `room` | Jitsi `state.room` | The room the user is entering. `*` for all. |
| `iat`/`nbf`/`exp` | adapter clock | `exp = now + JWT_EXP_SECOND` (default 10800). |
| `context.user.id` | OIDC `sub` | Coerce to string. |
| `context.user.name` | OIDC `name` ?? `preferred_username` | Coerce to string; never null. |
| `context.user.email` | OIDC `email` | Coerce to string. |
| `context.user.avatar` | OIDC `picture` (absent in tiny-idp) | Empty string; optional. |
| `context.user.moderator` | (policy) e.g. OIDC `roles`/`groups` | `"true"` only if you enable role logic (§7.3). |

> Note the two different `sub`s. The OIDC `sub` identifies the *person*; it becomes `context.user.id`. The
> Jitsi top-level `sub` identifies the *tenant/domain scope* and comes from the room URL. Conflating them is
> the classic first-integration bug.

### 7.2 Reference pseudocode (mirrors adapter `context.ts`, → 14)

```python
def oidc_to_jitsi(claims, *, app_id, room, tenant_sub, ttl, moderator, now):
    user = {
        "id":     str(claims.get("sub", "")),
        "name":   str(claims.get("name") or claims.get("preferred_username") or ""),
        "email":  str(claims.get("email", "")),
        "avatar": str(claims.get("picture", "")),
    }
    if moderator:
        user["moderator"] = "true"          # honored iff enableUserRolesBasedOnToken
    return {
        "iss": app_id, "aud": app_id, "sub": tenant_sub, "room": room,
        "iat": now, "nbf": now, "exp": now + ttl,
        "context": {"user": user},
    }
# sign HS256 with the secret shared with Prosody
```

A concrete run (`--room standup --sub personal --moderator`) produced a valid Jitsi JWT whose payload is:

```json
{ "iss":"jitsi","aud":"jitsi","sub":"personal","room":"standup",
  "iat":…, "nbf":…, "exp":…,
  "context":{"user":{"id":"user-alice-fixed","name":"Alice Inbox",
                     "email":"alice@example.test","avatar":"","moderator":"true"}}}
```

### 7.3 Moderator, room-scoping, guests

- **Room scoping** is enforced by `mod_token_verification` comparing the token `room`/`sub` to reality.
  Because the adapter injects `room` from Jitsi's `state`, the token is naturally scoped to the room the
  user is entering.
- **Moderator:** `context.user.moderator: "true"` grants host rights **only** with
  `config.enableUserRolesBasedOnToken = true`. The stock adapter does *not* set it (it sets
  `lobby_bypass`/`security_bypass` instead, → 14); to grant moderator you either edit the adapter's
  `createContext()` or install the community **`token_affiliation`** Prosody plugin and emit an
  `affiliation: "owner"` claim (→ 05).
- **Guests:** set `allow_empty_token = true` on the token VirtualHost + a `guest.<host>` anonymous
  VirtualHost (`config.hosts.anonymousdomain`) to let tokenless users in; add `muc_wait_for_host` +
  `persistent_lobby` to make guests wait for a token-authenticated moderator (→ 06, 08).

---

## 8. API Design (interfaces you will touch)

### 8.1 tiny-idp (unchanged — reference only)

```
GET  /.well-known/openid-configuration  → discovery JSON
GET  /jwks                               → JWKS (unused by Jitsi)
GET  /authorize?response_type=code&client_id&redirect_uri&scope&state[&nonce][&code_challenge…]
                                          → login form → 302 redirect_uri?code&state
POST /token   grant_type=authorization_code&code&redirect_uri&client_id[&code_verifier]
                                          → { id_token, access_token, token_type, expires_in }
GET  /userinfo   Authorization: Bearer <access_token>
                                          → { sub, name, preferred_username, email, … }
```

Register the adapter as a client:

```bash
tinyidp serve \
  --issuer http://127.0.0.1:5556 \
  --client-id jitsi-adapter \
  --client-secret adapter-secret \       # or omit for a public client
  --redirect-uris https://<jitsi-host>/oidc/tokenize \
  --users-file ./users.yaml
```

### 8.2 jitsi-oidc-adapter (deploy target)

```
GET /oidc/auth?state={state}      # entry from Jitsi tokenAuthUrl.
                                  # → 302 to tiny-idp /authorize (starts OIDC), stashes state.
GET /oidc/tokenize?code&state     # OIDC redirect_uri.
                                  # → exchanges code at /token, calls /userinfo, mints Jitsi JWT,
                                  #   302 to https://<host>/<tenant>/<room>?jwt=<jwt>
GET /oidc/health                  # liveness.
```

Adapter environment (→ 12, 15):

```
OIDC_ISSUER_URL   = http://127.0.0.1:5556       # tiny-idp issuer
OIDC_CLIENT_ID    = jitsi-adapter
OIDC_CLIENT_SECRET= adapter-secret               # empty for public client
OIDC_SCOPES       = openid profile email
JWT_APP_ID        = jitsi                         # == Prosody app_id
JWT_APP_SECRET    = <shared HS256 secret>         # == Prosody app_secret
JWT_ALG           = HS256
JWT_EXP_SECOND    = 10800
```

### 8.3 Prosody + Jitsi config

```lua
-- /etc/prosody/conf.d/<host>.cfg.lua
asap_accepted_issuers   = { "jitsi" }
asap_accepted_audiences = { "jitsi" }
VirtualHost "jitsi.example.com"
    authentication = "token";
    app_id     = "jitsi";
    app_secret = "<shared HS256 secret>";   -- MUST equal adapter JWT_APP_SECRET
    allow_empty_token = false;
Component "conference.jitsi.example.com" "muc"
    modules_enabled = { "token_verification" }
```

```js
// /etc/jitsi/meet/<host>-config.js
config.tokenAuthUrl = 'https://jitsi.example.com/oidc/auth?state={state}';
config.tokenAuthUrlAutoRedirect = true;
config.enableUserRolesBasedOnToken = true;
```

> **Do NOT** enable authentication in `jicofo.conf` when token auth is active — Jicofo is an admin user and
> must not be forced to present a token (→ 06). This is the #1 misconfiguration.

---

## 9. Decision Records

### ADR-1 — Use a translation adapter, not a direct wire-up
- **Context:** Jitsi validates a Jitsi-shaped JWT delivered via `?jwt=`; it has no native OIDC and no
  claim-mapping config (§4.2–4.5).
- **Options:** (a) point Jitsi at tiny-idp directly; (b) insert an OIDC→Jitsi-JWT adapter.
- **Decision:** (b). (a) is technically impossible today.
- **Consequences:** one extra small service to run; complete IdP-independence (swap tiny-idp for Keycloak
  with zero adapter changes). **Status: accepted.**

### ADR-2 — HS256 shared-secret to Prosody, not RS256/ASAP
- **Context:** Prosody RS256 uses PEM-by-kid, not JWKS (§4.4); tiny-idp only serves JWKS.
- **Options:** (a) teach tiny-idp to serve `{sha256hex(kid)}.pem` and mint Jitsi-shaped RS256 tokens;
  (b) let the adapter mint HS256 tokens with a secret shared with Prosody.
- **Decision:** (b). It is the adapter default and avoids Jitsi's JWKS gap entirely.
- **Consequences:** the secret must be identical in the adapter and Prosody; rotate both together.
  **Status: accepted.**

### ADR-3 — Deploy `jitsi-contrib/jitsi-oidc-adapter` rather than build our own
- **Context:** the mapping is ~40 lines but the OIDC plumbing, state handling, and Jitsi quirks are already
  solved and provider-agnostic (→ 13).
- **Options:** (a) deploy the contrib adapter; (b) MarcelCoding/jitsi-openid (Rust); (c) build in-house
  (e.g., a small Go service reusing tiny-idp libs).
- **Decision:** (a) for the first integration; keep (c) as a future option if we want a single Go binary.
- **Consequences:** adds a Deno runtime dependency. **Status: accepted (revisit for production).**

### ADR-4 — Prototype against tiny-idp; keep the design IdP-agnostic
- **Context:** the adapter consumes only standard OIDC discovery/authorize/token/userinfo (§4.6, §5.1).
- **Decision:** use tiny-idp for dev/integration; the same wiring works for any compliant OIDC IdP.
- **Consequences:** tiny-idp's non-production posture is acceptable for the prototype; production may swap
  the IdP without touching Jitsi/adapter config. **Status: accepted.**

---

## 10. Core Flow (end-to-end)

```
User          Jitsi web         nginx        adapter            tiny-idp        Prosody
 │ open room     │                │             │                  │              │
 │──────────────▶│ tokenAuthUrl   │             │                  │              │
 │◀──────────────┤ 302 /oidc/auth?state={room,tenant}             │              │
 │ GET /oidc/auth ───────────────▶│────────────▶│ build OIDC authz req           │
 │◀───────────────────────────────┤◀────────────┤ 302 tiny-idp /authorize?code…  │
 │ GET /authorize ───────────────────────────────────────────────▶│ login form  │
 │ POST login+password ──────────────────────────────────────────▶│ check creds │
 │◀────────────────────────────────────────────  302 /oidc/tokenize?code&state ──┤
 │ GET /oidc/tokenize ───────────▶│────────────▶│ POST /token(code) ───────────▶│ id/access tok
 │                                 │             │ GET /userinfo ───────────────▶│ user claims
 │                                 │             │ map → mint HS256 Jitsi JWT     │
 │◀────────────────────────────────┤◀────────────┤ 302 /<tenant>/<room>?jwt=…     │
 │ join room?jwt=… (BOSH/WS token) ─────────────────────────────────────────────▶│ validate JWT
 │◀──────────────────────────────────────────────────────────────────────────────┤ joined ✅
```

---

## 11. Implementation Plan (phased, intern-executable)

**Phase 0 — Local OIDC sanity (no Jitsi yet).** *Done in this ticket.*
Run `scripts/01-oidc-smoke.sh` (use `TINYIDP_BIN=$(go build …)` for speed). Confirm discovery, code
exchange, id_token, userinfo. Run `scripts/02-oidc-to-jitsi-jwt.py` to see the mapping. **Exit criteria:**
both scripts print a valid token/JWT.

**Phase 1 — Stand up Jitsi in token mode.** Deploy Jitsi (docker `jitsi/docker-jitsi-meet` is easiest).
Set Prosody `authentication="token"`, `app_id`/`app_secret`, enable `token_verification`. Verify with a
*hand-minted* HS256 token (from script 02, using the same secret) appended as `?jwt=` — you should join.
**Exit criteria:** a hand-minted token joins a room; a wrong-secret token is rejected.

**Phase 2 — Deploy the adapter.** Run `jitsi-contrib/jitsi-oidc-adapter` with the env from §8.2
(`OIDC_ISSUER_URL` → tiny-idp, `JWT_APP_SECRET` == Prosody secret). Proxy `/oidc/*` in nginx. Register the
adapter as a tiny-idp client with redirect `https://<host>/oidc/tokenize`. **Exit criteria:**
`GET /oidc/health` OK; `GET /oidc/auth?state=…` 302s to tiny-idp.

**Phase 3 — Wire Jitsi to the adapter.** Set `config.tokenAuthUrl`, `tokenAuthUrlAutoRedirect`,
`enableUserRolesBasedOnToken`. **Exit criteria:** opening a room redirects to tiny-idp; logging in as
`alice` lands you in the room with name "Alice Inbox".

**Phase 4 — Roles & guests (optional).** Decide moderator policy: edit `createContext()` to set
`moderator:"true"` from a tiny-idp `roles`/`groups` claim, or install `token_affiliation` + emit
`affiliation`. Configure guest VirtualHost if needed. **Exit criteria:** a "writer"/"owner" user is
moderator; a "reader" is not.

**Phase 5 — (Future, optional) In-house Go adapter.** If we want a single Go binary and to drop the Deno
dependency, port the ~40-line mapping (§7.2) into a small Go service reusing tiny-idp's `keys`/`domain`
packages. Track separately.

---

## 12. Testing and Validation Strategy

- **Unit (mapping):** table-test `oidc_to_jitsi` — null/missing `name` falls back to `preferred_username`;
  every `context.user` value is a string; `exp = now + ttl`. (Script 02 is the executable spec.)
- **Integration (OIDC):** `scripts/01-oidc-smoke.sh` in CI against a throwaway tiny-idp — assert discovery
  fields, a code is issued, id_token has `sub`/`email`, userinfo matches.
- **Integration (Jitsi):** Phase-1 hand-minted-token join; Phase-3 full browser flow (drive with Playwright:
  open room → expect redirect to tiny-idp → submit `alice`/`alice-password` → expect participant name).
- **Negative:** wrong `app_secret` → rejected; expired `exp` → rejected; `room` mismatch → rejected; guest
  with `allow_empty_token=false` → blocked.
- **Security review:** confirm Jicofo auth is OFF; confirm the shared secret is not logged; confirm
  `asap_accepted_issuers`/`audiences` are set (not `*`); confirm redirect URIs are allowlisted in tiny-idp.
- **Logout (optional):** tiny-idp has `/end-session`; wiring Jitsi "logout" to it is a separate task —
  Jitsi's own session is the JWT lifetime, so IdP logout only affects the *next* login.

---

## 13. Risks, Alternatives, Open Questions

**Risks.**
- *tiny-idp is not production-grade* (in-memory keys/sessions; loopback intent). Fine for dev/integration;
  for production, swap the IdP or complete the hardening tickets.
- *Shared-secret coupling:* adapter and Prosody must hold the identical `app_secret`; rotation is manual
  and requires restarts.
- *Deno runtime* dependency for the stock adapter (mitigated by ADR-3 future Go port).
- *Moderator mapping is not out-of-the-box* — requires an adapter edit or a Prosody plugin (§7.3).

**Alternatives.**
- *MarcelCoding/jitsi-openid* (Rust) as the adapter — same pattern, different runtime, passes `affiliation`
  through natively (→ 05).
- *Direct-ish RS256/ASAP path* — teach tiny-idp to mint Jitsi-shaped RS256 tokens AND serve
  `{sha256hex(kid)}.pem`, plus a `?jwt=` redirect endpoint. This is just embedding the adapter into
  tiny-idp; more coupling, same work. Not recommended for a first cut.

**Open questions.**
- Do we want moderator rights driven by tiny-idp `roles` (e.g. `owner` → moderator)? Needs a policy call.
- Multi-tenant? If we use `TENANT/ROOM` URLs, confirm `sub`=tenant mapping in the adapter matches
  tiny-idp's `tenant` claim usage.
- Production IdP: keep tiny-idp (after hardening) or adopt Keycloak? The Jitsi side is unaffected.

---

## 14. Intern Onboarding Checklist

**Read first (30 min):** this doc §§1–4; then `sources/web/01-lib-jitsi-meet-tokens.md` (the token spec)
and `sources/web/03-jitsi-native-oidc-issue-16576.md` (why an adapter is needed).

**Run first (15 min):**
```bash
cd tiny-idp
go build -o /tmp/tinyidp ./cmd/tinyidp
T=ttmp/2026/07/09/TINYIDP-JITSI-001--*/
TINYIDP_BIN=/tmp/tinyidp USERS_FILE=$PWD/examples/users/personal-inbox-users.yaml \
  LOGIN=alice PASSWORD=alice-password bash $T/scripts/01-oidc-smoke.sh
python3 $T/scripts/02-oidc-to-jitsi-jwt.py --room standup --sub personal --moderator
```
You should see a full OIDC flow, then a minted Jitsi JWT.

**Edit first (when building):** the adapter's `createContext()` (claim→`context.user` mapping) is the only
place you normally touch. On the tiny-idp side, seeded users live in a `--users-file`
(`examples/users/personal-inbox-users.yaml`); claim assembly is `internal/domain/claims.go`.

**Mental model:** *tiny-idp says who you are (OIDC); the adapter restates that as a Jitsi JWT; Prosody
checks the JWT with a shared secret; Jitsi lets you in.*

---

## 15. References

**Local — this ticket**
- `scripts/01-oidc-smoke.sh` (+ `.output.txt`) — verified OIDC flow against tiny-idp.
- `scripts/02-oidc-to-jitsi-jwt.py` (+ `.output.txt`) — verified OIDC→Jitsi-JWT mapping.
- `sources/web/01-lib-jitsi-meet-tokens.md` — canonical `mod_auth_token` spec (claims, ASAP).
- `sources/web/03-jitsi-native-oidc-issue-16576.md` — no native OIDC (adapter required).
- `sources/web/06,07` — Jitsi handbook token & secure-domain.
- `sources/web/08,09,10,12,13,14,15` — adapter setup, README, and source.
- `sources/web/05` — MarcelCoding/jitsi-openid (alternative adapter).
- `reference/01-investigation-diary.md` — chronological investigation record.

**tiny-idp source**
- `internal/oidcmeta/discovery.go` — discovery metadata.
- `internal/keys/keys.go` — RS256 signing + JWKS.
- `internal/domain/claims.go` — `ClaimsForScopes`.
- `internal/cmds/serve.go` — engines, client registry, users file.
- `internal/server/authorize.go`, `internal/server/server.go` — authorize/routes (mock engine).

**External**
- lib-jitsi-meet `doc/tokens.md`; Jitsi handbook (token-authentication, secure-domain).
- jitsi-contrib/jitsi-oidc-adapter; nordeck/jitsi-keycloak-adapter; MarcelCoding/jitsi-openid.
- jitsi/jitsi-meet issues #16576 (native OIDC) and #15182 (JWKS support).
