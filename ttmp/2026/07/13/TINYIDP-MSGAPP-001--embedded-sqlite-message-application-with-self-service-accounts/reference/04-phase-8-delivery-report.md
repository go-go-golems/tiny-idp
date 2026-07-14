---
Title: Phase 8 Delivery Report
Ticket: TINYIDP-MSGAPP-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-message-app/main.go
      Note: Operator executable and Glazed root composition.
    - Path: repo://examples/tinyidp-message-app/commands.go
      Note: Durable state construction, commands, maintenance, and shutdown.
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: Application health/readiness and registration audit delivery semantics.
    - Path: repo://examples/tinyidp-message-app/fuzz_test.go
      Note: Short security fuzz-smoke targets.
    - Path: repo://ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/reference/03-phase-8-operations-and-verification-guide.md
      Note: Detailed operator runbook.
ExternalSources: []
Summary: Completion report for the executable, lifecycle, operational documentation, and verification work in Phase 8.
LastUpdated: 2026-07-14T21:15:00Z
WhatFor: Gives a reviewer or incoming intern a concise, evidence-backed Phase 8 handoff.
WhenToUse: Read before reviewing the Phase 8 commit, testing the live demo, or choosing the next task.
---

# Phase 8 Delivery Report

## Outcome

Phase 8 turns `examples/tinyidp-message-app` into a runnable single-process
example that embeds tiny-idp and a SQLite-backed relying-party application on
one origin. The executable exposes Glazed `init`, `serve`, and `doctor`
commands; initializes only durable state rather than an invented first user;
serves a real browser UI; and reports liveness/readiness independently.

The local demo is currently running at:

```text
http://127.0.0.1:8090/
tmux session: tinyidp-message-app
state root: /tmp/tinyidp-message-app-phase8
```

This loopback HTTP instance is a development canary. It is not a claim that
cleartext public authentication is production-ready. The state validator
allows HTTP only for loopback origins. HTTPS manifests require TLS certificate
and key arguments and secure provider cookies.

## Delivered architecture

```text
Glazed init
    -> state.json + owner-only secret files

Glazed serve
    -> identity SQLite + application SQLite + audit stream
    -> account service + conservative browser-client/signing-key bootstrap
    -> embedded tiny-idp with Message Desk UI renderer
    -> issuer-only in-process OIDC RP client
    -> one public HTTP server

GET /healthz
    -> process/provider liveness
GET /readyz
    -> provider schema/key/audit/maintenance checks + application SQLite ping
```

The server runs three coordinated goroutines under `errgroup`: the HTTP
listener, periodic provider/application retention work, and graceful shutdown.
SIGINT/SIGTERM cancels the common context. Shutdown uses a bounded
`http.Server.Shutdown`, then closes provider, audit, and both databases.

## Security-relevant decisions delivered

- The browser client is public, exact-origin, and PKCE S256-required.
- The provider is mounted at `/idp/`; the application keeps its RP routes and
  static prefixes distinct.
- The server-side OIDC client reaches only the exact configured issuer through
  `NewInProcessIssuerTransport`; it has no host-network fallback.
- Account registration does not auto-login. Browser authentication still
  crosses tiny-idp's password, interaction, consent, authorization-code, and
  callback boundaries.
- Runtime uses a synchronous file audit sink. A registration audit delivery
  failure is logged, counted, and makes `/readyz` unhealthy.
- HTTPS manifests cannot accidentally start an HTTP listener. Conversely,
  local HTTP cannot silently carry TLS configuration.
- Fuzz coverage now continuously checks the local return-path acceptance
  invariant and JSON decoder panic resistance in short smoke runs.

## Verification evidence

| Gate | Result | Evidence |
| --- | --- | --- |
| Focused example tests | pass | `go test ./examples/tinyidp-message-app` |
| Race detector | pass | `go test -race ./examples/tinyidp-message-app` |
| Go vet | pass | `go vet ./examples/tinyidp-message-app` |
| Frontend build | pass | `pnpm -C examples/tinyidp-message-app/ui run build` |
| Fuzz smoke | pass | 3-second runs for return path, JSON decoder, and idpui renderer fuzz target |
| IdP interaction analyzer | pass | explicit analyzer run over `examples/tinyidp-message-app/loginui` |
| Audit analyzer | pass after remediation | `make auditlint` found then verified audit-delivery behavior |
| Repository lint | pass | `make lint` |
| Repository tests | pass | `go test ./...` |
| tmux / health / readiness | pass | captured listener and `/readyz` with all provider and app checks ready |
| Browser canary | awaiting human check | live local process is intentionally left running |

The audit analyzer finding is important evidence, not an incidental log item.
It identified an unmarked no-op audit fallback and a discarded audit write
error. The implementation now has explicit development-only fallback semantics
for test construction and conservatively degrades production runtime readiness
on a registration audit delivery failure.

## How to perform the remaining browser canary

1. Open `http://127.0.0.1:8090/`.
2. Register a fresh account. Confirm registration does not log you in.
3. Choose **Sign in** and verify the styled tiny-idp page appears.
4. Submit valid credentials, complete consent if shown, and return to the
   Message Desk authenticated.
5. Post a message, refresh, and confirm it persists as literal text.
6. Log out and verify a new message cannot be posted until another login.

Support commands:

```sh
tmux capture-pane -pt tinyidp-message-app:0.0 -S -100
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
```

## Remaining tracked work

Phase 8 operational implementation is complete except for the deliberate
manual browser canary task. Separate earlier tasks remain open:

- Phase 4's explicit callback mismatch, expiry, and open-redirect regression
  coverage task;
- Phase 7's component, keyboard, reflow, and accessibility coverage task;
- longer CI browser testing and online two-database backup automation.

The next engineering step should be to turn the manual browser canary into a
repeatable browser test, while preserving the existing full embedded OIDC
integration test as a fast protocol-level regression.
