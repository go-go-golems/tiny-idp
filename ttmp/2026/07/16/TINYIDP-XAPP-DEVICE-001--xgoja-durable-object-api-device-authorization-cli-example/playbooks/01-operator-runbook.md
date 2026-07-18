---
Title: tinyidp-xapp Device API Operator Runbook
Ticket: TINYIDP-XAPP-DEVICE-001
Status: active
Topics:
    - operations
    - security
    - oidc
    - oauth2
    - durable-objects
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp-xapp/device_api.go
      Note: Bearer outcomes, audit event names, and durable-object dispatch boundary.
    - Path: cmd/tinyidp-xapp/device_cli.go
      Note: Device token cache and operator-facing terminal commands.
    - Path: cmd/tinyidp-xapp/init.go
      Note: Initialization command and owner-only password-file contract.
    - Path: cmd/tinyidp-xapp/serve.go
      Note: Development lifecycle and local readiness behavior.
    - Path: cmd/tinyidp-xapp/serve_initialized.go
      Note: TLS startup, health/readiness, maintenance, and graceful shutdown.
    - Path: cmd/tinyidp-xapp/state.go
      Note: State-root layout and permission validation.
ExternalSources: []
Summary: Procedure for safely starting, validating, monitoring, and responding to incidents in the self-contained xapp, embedded tiny-idp, and device bearer API.
LastUpdated: 2026-07-16T00:00:00Z
WhatFor: Ensure operations preserve browser, device, secret, audit, and durable-object security boundaries.
WhenToUse: Before demos or deployment, during release checks, and during authentication, audit, or token incidents.
---


# tinyidp-xapp Device API Operator Runbook

## Scope

`tinyidp-xapp` is one Go process with an embedded tiny-idp at `/idp/`, xgoja
browser application routes at `/`, and host-owned bearer endpoints at
`/api/device/`. Browser mutations use the OIDC application session plus CSRF.
Terminals use a distinct public device client. Its opaque bearer token is
introspected by Go before a durable-object BBS request is dispatched.

```text
browser ──OIDC/session──> xapp routes ──CSRF──> BBS durable object
terminal ──device flow───> tiny-idp ──opaque token──> /api/device/*
                                                    │
                           RFC 7662 Basic auth <── host-only resourceauth
                                                    │
                                      verified sub ─┘
```

The server is the security boundary. JavaScript never receives the resource
client secret or a raw bearer token, and `device_api.go` derives durable-object
actor identity from the verified OIDC subject instead of request input.

This is a single-node product example. State, audit persistence, backup,
reverse-proxy policy, and monitoring need a deployment-specific operating
design before any high-availability or distributed-protection claim is made.

## Preconditions and invariants

Before starting the application:

- Run `go test ./...` for the exact source revision.
- Choose the browser-visible public origin before initialization. Production
  initialization accepts only an absolute HTTPS origin with no path, query,
  fragment, or userinfo.
- Use a service-owned state root with mode `0700` and no group/other access.
- Supply the initial password through a regular mode-`0600` password file.
  Do not place it in command history, source, logs, tickets, or environment.
- Ensure TLS key/certificate access follows the deployment secret policy.
- Do not rely on unreviewed forwarded headers. `serve-initialized` explicitly
  does not trust them.

Do not reuse development HTTP, development credentials, or development state
in a real deployment.

## Development operation

Run development mode in tmux from the repository root:

```sh
tmux new-session -s tinyidp-xapp-dev
go run ./cmd/tinyidp-xapp serve \
  --listen 127.0.0.1:18878 \
  --public-base-url http://127.0.0.1:18878 \
  --state-root /tmp/tinyidp-xapp-dev-18878 \
  --login alice \
  --password 'correct horse battery staple' \
  --second-login bob \
  --second-password 'correct horse battery staple'
```

Use discovery as the development readiness probe:

```sh
curl -fsS http://127.0.0.1:18878/idp/.well-known/openid-configuration
```

Development `serve` does not mount `/healthz`; a `404` there is expected.
For the maintained local setup, use
`scripts/run-xapp-device-smoke.sh`. It starts a dedicated tmux server and
prints non-secret device commands. Inspect logs with:

```sh
tmux capture-pane -pt tinyidp-xapp-device-smoke -S -100
```

## Initialized TLS operation

Initialized mode first creates durable state, then serves it. Re-running
`init` is allowed only when the public-origin and identity configuration match
the existing manifest. A conflict is a protective error; do not edit
`state.json` by hand.

Prepare the root and password file through the deployment's secret mechanism:

```sh
install -d -m 0700 /srv/tinyidp-xapp
install -m 0600 /dev/null /srv/tinyidp-xapp/initial-password
stat -c '%a %F %n' /srv/tinyidp-xapp /srv/tinyidp-xapp/initial-password
```

Initialize the exact public origin:

```sh
go run ./cmd/tinyidp-xapp init \
  --state-root /srv/tinyidp-xapp \
  --public-base-url https://app.example.test \
  --login administrator \
  --password-file /srv/tinyidp-xapp/initial-password \
  --email administrator@example.test \
  --name 'Application Administrator'
```

Start TLS serving. Bind directly to the visible TLS listener, or add a
separately reviewed proxy mode; do not silently assume proxy-header trust.

```sh
go run ./cmd/tinyidp-xapp serve-initialized \
  --state-root /srv/tinyidp-xapp \
  --listen :8443 \
  --tls-cert /run/secrets/tinyidp-xapp.crt \
  --tls-key /run/secrets/tinyidp-xapp.key \
  --maintenance-interval 15m
```

Verify liveness, readiness, and discovery:

```sh
curl --fail --silent --show-error https://app.example.test/healthz
curl --fail --silent --show-error https://app.example.test/readyz
curl --fail --silent --show-error https://app.example.test/idp/.well-known/openid-configuration
```

`/healthz` means the listener can answer. `/readyz` additionally requires the
provider, SQLite state, signing material, audit sink, rate limiter,
maintenance status, application auth services, durable objects, generated
runtime, and router to be usable. Remove an instance from traffic on readiness
failure rather than retrying mutations.

## State, secrets, and backup

The initialized root is a security boundary. The root must be `0700`; the
database and secret files are checked as `0600` by `ValidateInitializedState`.

| Relative path | Purpose | Handling |
| --- | --- | --- |
| `state.json` | Public origin, issuer, client IDs, API audience, creation time | Configuration record; do not hand edit. |
| `identity/tinyidp.sqlite` | Accounts, OAuth/Fosite state, clients, signing material | Confidential, integrity-critical SQLite state. |
| `application/auth.sqlite` | Browser application-auth/session state | Confidential persistent state. |
| `objects/` | Durable-object BBS state | Application data; restore consistently with related state. |
| `secrets/token.key` | tiny-idp token cryptographic secret | Credential material; never log or casually rotate. |
| `secrets/resource-client.key` | Derives RFC 7662 confidential resource-client credential | Credential material; never send to JavaScript or a CLI. |
| `secrets/object-binding.key` | Actor-binding key for durable objects | Credential material; a lifecycle dependency. |
| `audit/tinyidp.jsonl` | Durable security audit stream | Restrict access; retain/ship under policy. |

Back up state through an SQLite-consistent method while quiesced or otherwise
coordinated. Do not restore a single OAuth table or one durable-object file.
On restoration: stop service, preserve failed state for forensics, restore the
compatible complete set, restore permissions/ownership, start, and require
`/readyz` before traffic.

## Device client procedure

The browser user signs in normally. The device client is separate and needs
`openid`, requested BBS scopes, and the exact API audience. The current
tiny-idp device endpoint expects the request field named `audience`, not the
RFC 8707 `resource` field.

```sh
go run ./cmd/tinyidp-xapp device-login \
  --issuer https://app.example.test/idp \
  --audience https://app.example.test/api \
  --scopes 'openid bbs.read bbs.post.create' \
  --token-cache ./tinyidp-xapp-device-token.json

go run ./cmd/tinyidp-xapp bbs-post \
  --api-base-url https://app.example.test \
  --token-cache ./tinyidp-xapp-device-token.json \
  --title 'Terminal dispatch' \
  --body 'Approved through device authorization.' \
  --category notes

go run ./cmd/tinyidp-xapp bbs-get \
  --api-base-url https://app.example.test \
  --token-cache ./tinyidp-xapp-device-token.json
```

The terminal prints the verification URL/code. The human completes approval in
a browser. The terminal must never receive a browser password. Token caches
must be regular mode-`0600` files; delete them on shared terminals or after an
incident.

## Audit and monitoring

Initialized mode writes to `audit/tinyidp.jsonl`. It should contain decisions,
not credentials. Expected events include tiny-idp
`introspection.accepted`/`introspection.inactive`/`introspection.rejected`,
identity password-change events, and xapp
`xapp.api.auth.accepted`/`xapp.api.auth.rejected`/
`xapp.api.auth.unavailable`/`xapp.api.bbs.read`/
`xapp.api.bbs.posted` events. xapp events record credential kind
`oidc_bearer`; raw tokens, secrets, device codes, passwords, and cookies must
not appear.

Monitor readiness, audit delivery/open failures, SQLite and filesystem errors,
TLS expiry, maintenance failures, and sustained increases in inactive/rejected
introspection. `503` from bearer routes means provider/dispatch unavailability
and is intentionally fail-closed. The current production path uses an
in-process fixed-window rate limiter; use a deliberate shared limiter design
before operating multiple replicas.

## Incident response

### Suspected device-token exposure

1. Preserve relevant redacted logs/audit evidence; never copy the token into a
   ticket.
2. Change the affected user's password through the approved lifecycle path.
   The tested password-security transition removes that subject's Fosite access
   token sessions, making a later introspection inactive.
3. Delete the affected terminal's local token cache.
4. Verify the old token gets `401` from `/api/device/bbs`; record only the
   redacted audit result.
5. Investigate the cache escape. Cache deletion alone is not server revocation.

### Suspected resource-client or token-secret exposure

1. Remove the instance from traffic and preserve its state root for forensics.
2. Do not change one secret in place. Secret rotation affects persisted OAuth
   state and active sessions and needs a separately tested migration plan.
3. Escalate to controlled rotation/reinitialization with backup/restore and
   user-impact review.

### Readiness or audit failure

1. Remove the instance from traffic when `/readyz` fails.
2. Capture process logs, recent audit lines, disk space, file modes, and SQLite
   errors; do not run broad cleanup against state.
3. Restore the failed dependency before restart. Do not bypass startup state
   validation or operate without required audit evidence.

### Repeated bearer failures

Classify before changing configuration: `401` is missing/malformed/inactive;
`403` is valid but lacks scope; `503` is an unavailable provider/dispatch.
For `403`, request correct scopes in a new device flow. For `401`, check issuer,
exact `/api` audience, cache expiry, and client setup. Never add a cookie
fallback or caller-provided actor field to "fix" a device route.

## Release checklist

- [ ] `go test ./...` passes from the released revision.
- [ ] Root is `0700`; secrets, password file, and databases are `0600`.
- [ ] Public origin, certificate SAN, issuer discovery, `/healthz`, and
      `/readyz` are correct.
- [ ] One browser login/post/logout and one device login/post/read succeed.
- [ ] Audit contains redacted success and rejection decisions.
- [ ] Backup and restore are exercised outside production.
- [ ] Operators accept the known limits: single node, in-process rate limiter,
      no operator token-revocation endpoint, and `audience` request-field
      interoperability constraint.

## Review references

- `cmd/tinyidp-xapp/state.go`
- `cmd/tinyidp-xapp/serve_initialized.go`
- `cmd/tinyidp-xapp/device_api.go`
- `cmd/tinyidp-xapp/device_cli.go`
- `reference/01-implementation-diary.md`
