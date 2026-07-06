---
Title: "Scenario catalog"
Slug: scenarios
Short: "Every scenario tinyidp can simulate, grouped by category, and the model that selects them."
Topics:
- scenarios
- oidc
- testing
Commands:
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

tinyidp's scenarios are the reason to use it over a real IdP. A scenario
is a named bundle of behavior attached to a synthetic user: it decides
what happens at each OIDC endpoint, so you can reproduce a specific
failure or claim shape by changing one input. This page is the complete
catalog of the thirty-one builtin scenarios and the model behind them.

## How a scenario is selected

You select a scenario by logging in as its name. When a login is
submitted, tinyidp's registry looks up the login string. If it matches a
scenario `Name`, that scenario drives the flow. If it matches nothing,
tinyidp derives a normal user from the login, so any arbitrary username
still works. The login page renders every scenario as a quick-pick
button grouped by category, so you do not have to memorize names.

## The model

A scenario is a struct with one field per endpoint it can influence. A
field is empty by default, meaning "behave normally"; setting it selects
a failure. The fields are:

| Field | Affects | What it does |
|-------|---------|--------------|
| `AuthError` | `/authorize` | Returns the OAuth error code via redirect (e.g. `access_denied`). No code is issued. |
| `TokenError` | `/token` | Selects a token-endpoint failure: `invalid_grant`, `server_error`, or `slow` (sleep 10s, then succeed). |
| `UserInfoError` | `/userinfo` | Selects a userinfo failure: `401`, `500`, or `sub_mismatch` (200 with a different `sub`). |
| `MutateClaims` | ID token | A function that mutates ID-token claims after they are built (e.g. set `exp` in the past, change `aud`). ID-token only. |
| `ExtraClaims` | ID token + userinfo | A map of claims merged into both responses (groups, roles, tenant). The declarative way to model a user's attributes. |
| `OmitClaims` | ID token + userinfo | A list of claims deleted from both responses (e.g. drop `email`). |
| `SignKey` | ID token signature | Selects the signing key: `rotated`, `unknown-kid`, or `bad-sig`. Corrupts the signature/key binding, not claim values. |

The distinction between `MutateClaims` and `ExtraClaims`/`OmitClaims`
matters. `MutateClaims` is a failure-injection hook that corrupts the
ID token to test RP validation; it affects the ID token only.
`ExtraClaims` and `OmitClaims` describe a user's real attributes and
are honored by both the ID token and the userinfo response, so the two
endpoints agree. If a scenario's claims appear in the ID token but not
in userinfo, the abstraction has regressed.

`MutateClaims` runs last, after `ExtraClaims` and `OmitClaims`, so a
failure mutator can override a declarative value.

## The catalog

### Normal users

| Name | Description |
|------|-------------|
| `alice` | Normal user. |
| `bob` | Normal user. |

### Claim variants

These model positive claim shapes — real attributes a relying party's
authorization logic might depend on — using `ExtraClaims` and
`OmitClaims`.

| Name | Description |
|------|-------------|
| `admin` | `groups:[admin, engineering]`, `roles:[owner]`, `preferred_username:admin`. |
| `viewer` | `groups:[viewer]`, `roles:[reader]`. |
| `no-groups` | No `groups` or `roles` claims. |
| `many-groups` | Eight groups (stress claim parsing). |
| `tenant-a-admin` | `groups:[admin]`, `tenant:tenant-a`. |
| `tenant-b-viewer` | `groups:[viewer]`, `tenant:tenant-b`. |
| `unicode-name` | `name:"Müller Frédéric"`, `locale:de-DE`. |
| `no-email` | `email` and `email_verified` omitted. |
| `unverified-email` | `email_verified:false`. |

### Authorization failures

Selected by `AuthError`. The failure surfaces at `/authorize` via a
redirect back to the RP with an `error` parameter; no code is issued.

| Name | Error |
|------|-------|
| `fail-access-denied` | `access_denied` |
| `fail-login-required` | `login_required` |
| `fail-consent-required` | `consent_required` |
| `fail-server-error` | `server_error` |

### Token failures

Selected by `TokenError`. The authorize step succeeds; the failure
surfaces at `/token`.

| Name | Behavior |
|------|----------|
| `token-invalid-grant` | 400 `invalid_grant`. |
| `token-server-error` | 500 `server_error`. |
| `token-slow` | Sleeps 10s, then succeeds normally. |

### ID token failures

Selected by `MutateClaims`. The token is signed with a valid key; a
claim value is corrupted. These test RP ID-token validation.

| Name | What is wrong |
|------|---------------|
| `id-expired` | `exp` in the past. |
| `id-wrong-aud` | Wrong `aud`. |
| `id-wrong-iss` | Wrong `iss`. |
| `id-missing-email` | `email` claims omitted. |
| `id-email-unverified` | `email_verified:false`. |
| `id-bad-nonce` | `nonce` mismatch. |
| `id-future-iat` | `iat`/`auth_time` in the future. |

### UserInfo failures

Selected by `UserInfoError`. The ID token is valid; the userinfo
response is broken.

| Name | Behavior |
|------|----------|
| `userinfo-401` | 401. |
| `userinfo-500` | 500. |
| `userinfo-sub-mismatch` | 200 with a `sub` different from the ID token. |

### JWKS / key rotation

Selected by `SignKey`. These corrupt the signature or key binding, not
claim values. JWKS publishes three kids (`dev-key-1`, `rotated-key-2`,
`bad-sig-key`); each scenario picks a different signing behavior.

| Name | What happens |
|------|--------------|
| `key-rotated` | Signed with the second published key (`rotated-key-2`); verifies, testing that the RP looks up the kid. |
| `kid-not-found` | Signed with a kid not present in JWKS; the RP cannot find a key. |
| `bad-signature` | The kid is in JWKS but the signature was made with a different key; verification fails. |

JWKS-level failures (HTTP 500, slow, empty key set) are not scenarios —
they are server-level modes toggled through the debug UI, because
`/jwks` is global and not tied to a login. See `tinyidp help reference`.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Scenario does not trigger. | The login did not match a scenario `Name` exactly. | Log in with the exact name (e.g. `id-expired`, not `expired`). Unknown logins derive a normal user. |
| Failure surfaces at the wrong endpoint. | Each field selects one endpoint. | `AuthError` → `/authorize`; `TokenError` → `/token`; `UserInfoError` → `/userinfo`; `MutateClaims`/`SignKey` → ID token. |
| ID token and userinfo disagree on claims. | A hook leaked across the boundary. | Use `ExtraClaims`/`OmitClaims` for attributes (both endpoints); reserve `MutateClaims` for ID-token failure injection. |
| `token-slow` hangs the test. | It sleeps a fixed 10s. | Exclude it from CI, or accept the delay; it models a real slow IdP. |

## See also

- `tinyidp help tutorial` — a guided walkthrough of four scenarios.
- `tinyidp help tutorial-seeded-users-and-claims` — deterministic users and custom claims before scenario failures.
- `tinyidp help developer-guide` — how scenarios are implemented and extended.
- `tinyidp help reference` — endpoints, the debug UI, and the JWKS mode toggle.
