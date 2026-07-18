---
Title: New Intern Handoff and Continuation Playbook
Ticket: TINYIDP-XAPP-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - identity
    - oidc
    - research
    - testing
    - xgoja
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/objects/objects.js
      Note: Bound USER_STATE object behavior and document validation limits
    - Path: repo://cmd/tinyidp-xapp/app/routes/site.js
      Note: Trusted Express authentication authorization CSRF audit and private-object route contract
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: Shared generated-runtime route object-binding mux and resource-lifecycle composition
    - Path: repo://cmd/tinyidp-xapp/production_app.go
      Note: Production identity application-session and persistent-store composition explained in the handoff
    - Path: repo://cmd/tinyidp-xapp/serve_initialized.go
      Note: Real TLS readiness maintenance limits and shutdown procedure
    - Path: repo://ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/scripts/01_real_browser_e2e.py
      Note: Draft executable checkpoint and intern's first continuation target
    - Path: ws://go-go-goja/pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Generic application OIDC callback and logout semantics identified for lifecycle review
ExternalSources: []
Summary: Intern-ready system map, security model, executable checkpoint, and first contribution sequence for continuing the self-contained tiny-idp, xgoja Express, and Durable Objects product.
LastUpdated: 2026-07-12T20:24:27-04:00
WhatFor: Bring a new contributor from repository orientation to a reviewable real-browser security contribution without requiring them to reconstruct the project history.
WhenToUse: Read this before resuming TINYIDP-XAPP-001, changing identity or session semantics, running the initialized TLS application, or promoting browser and fault tests into release gates.
---


# New Intern Handoff and Continuation Playbook

## 1. Purpose and immediate assignment

This document is the operational handoff for the self-contained identity and
Durable Object application tracked by `TINYIDP-XAPP-001`. It is written for a
new intern who must become productive in the codebase without first reading the
entire historical diary or every research report. It explains what the product
does, which repository owns each subsystem, what has already been proven, what
has not been proven, and how to make the next contribution safely.

Your first assignment is deliberately concrete:

1. run the initialized product over a real TLS listener;
2. make the existing Chromium harness reach the application;
3. preserve structured evidence for two-user identity and object isolation;
4. characterize logout, disablement, expiry, and forced reauthentication;
5. turn each security expectation into a named automated invariant;
6. only then add deterministic failure injection.

Do not begin by extracting a generic tiny-idp xgoja provider, redesigning the
frontend, or implementing backup. Those are valid later phases, but they do not
replace the missing release evidence at the current checkpoint.

## 2. What the product is

The product is one Go process and one public HTTPS origin. It combines:

- tiny-idp as an embedded OpenID Connect identity provider;
- go-go-goja host authentication as the OIDC relying party and application
  session manager;
- xgoja-generated Go code containing trusted Express routes, frontend assets,
  and the Durable Object JavaScript bundle;
- go-go-objects as the persistent per-actor object runtime;
- a small HTML and JavaScript frontend for reading and writing one private JSON
  document.

The browser-visible route contract is:

```text
https://host.example/
├── /                         public HTML shell
├── /static/*                 embedded frontend assets
├── /auth/login               application OIDC login start
├── /auth/callback            application OIDC callback
├── /auth/session             application-session bootstrap
├── /auth/logout              application-session logout
├── /idp/.well-known/*        embedded issuer discovery
├── /idp/authorize            login, consent, and authorization interaction
├── /idp/token                authorization-code exchange
├── /idp/jwks.json            signing-key publication
├── /api/me                   authenticated actor projection
├── /api/object               subject-bound private Durable Object API
├── /healthz                  process liveness
└── /readyz                   aggregate readiness
```

There is no network hop between the relying party and the embedded issuer for
discovery, JWKS, or token exchange. The browser still traverses ordinary HTTPS
OIDC endpoints. Server-side OIDC requests use an origin-restricted in-process
HTTP transport. This preserves protocol boundaries while avoiding a startup
cycle in which a server would need to call its own listener before binding it.

## 3. Repository map and ownership

The workspace root is:

```text
/home/manuel/workspaces/2026-07-07/prod-tiny-idp
```

It contains a shared `go.work` and three relevant repositories.

### 3.1 tiny-idp

Path:

```text
/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp
```

This repository owns:

- the identity provider and its production store;
- password authentication and password lifecycle policy;
- authorization interactions, CSRF, consent, sessions, signing keys, tokens,
  audit, rate limiting, and maintenance;
- the composed product command under `cmd/tinyidp-xapp`;
- the XAPP ticket and all handoff artifacts.

Start with these files:

- `cmd/tinyidp-xapp/state.go`: initialized state layout and reconciliation;
- `cmd/tinyidp-xapp/production_app.go`: production dependency composition;
- `cmd/tinyidp-xapp/development_app.go`: generated runtime, routes, object
  binding, outer mux, and close order;
- `cmd/tinyidp-xapp/serve_initialized.go`: TLS listener, maintenance,
  readiness, limits, and shutdown;
- `cmd/tinyidp-xapp/xgoja.yaml`: generated runtime package contract;
- `cmd/tinyidp-xapp/app/routes/site.js`: authenticated route policy;
- `cmd/tinyidp-xapp/app/objects/objects.js`: private object behavior and
  document bounds;
- `cmd/tinyidp-xapp/app/frontend/public/`: current browser UI;
- `internal/fositeadapter/provider.go`: authorization interaction and login
  policy;
- `internal/fositeadapter/interaction_hardening_test.go`: forced-login,
  max-age, replay, CSRF, and consent regression tests.

### 3.2 go-go-goja

Path:

```text
/home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja
```

This repository owns:

- the Goja runtime and Express route planner;
- host authentication interfaces and persistent application sessions;
- the OIDC relying-party implementation;
- in-process issuer transport;
- native `/auth/*` handler construction;
- xgoja provider and generated-runtime infrastructure.

For lifecycle work, begin with:

- `pkg/gojahttp/auth/oidcauth/oidcauth.go`;
- `pkg/gojahttp/auth/sessionauth/sessionauth.go`;
- `pkg/xgoja/hostauth/builder.go`;
- `pkg/xgoja/hostauth/stores.go`;
- `pkg/gojahttp/enforcer.go`.

Do not patch logout only in tiny-idp-xapp if the behavior is a general hostauth
contract. The generic OIDC host owns application-session logout. The product
frontend owns how that contract is presented to a user.

### 3.3 go-go-objects

Path:

```text
/home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-objects
```

This repository owns:

- Durable Object instance management;
- per-object SQLite persistence;
- alarms, eviction, and CPU/request bounds;
- xgoja `durableobjects` integration;
- `BoundDispatcher`, which derives an object identity from the authenticated
  actor and a host secret.

The product disables raw object gateways. JavaScript may call
`fetchForActor("USER_STATE", ...)`, but it cannot select another actor's
object name. Review the bound dispatcher and server manager before changing
identity derivation or namespaces.

## 4. Runtime architecture

The request flow is:

```text
Browser
  |
  | HTTPS
  v
serve-initialized http.Server
  |
  +-- GET /healthz, GET /readyz        native aggregate endpoints
  |
  `-- MaxBytesHandler
       |
       v
     outer ServeMux
       |
       +-- /idp/*                      embedded tiny-idp handler
       +-- /auth/*                     native go-go-goja hostauth handlers
       `-- /*                          generated Express host
                                           |
                                           +-- public asset/index routes
                                           +-- actor authentication
                                           +-- capability policy
                                           +-- CSRF policy
                                           +-- audit policy
                                           `-- BoundDispatcher
                                                   |
                                                   v
                                             USER_STATE object
                                                   |
                                                   v
                                              SQLite storage
```

The startup flow is:

```text
Validate state.json and required files
  -> open tiny-idp SQLite store and audit sink
  -> construct production password service and embedded IdP
  -> run initial tiny-idp maintenance
  -> construct origin-restricted in-process OIDC transport
  -> open persistent application auth/session stores
  -> create host-owned Durable Object server
  -> load object-binding key and construct BoundDispatcher
  -> construct generated xgoja runtime
  -> evaluate trusted site.js and register routes
  -> check aggregate readiness
  -> bind TLS listener
  -> run periodic maintenance and graceful shutdown under errgroup
```

The order is security-relevant. The listener must not open before state
validation, trusted route registration, persistent store construction, and
readiness succeed.

## 5. Identity and session model

There are two independent browser sessions.

### 5.1 Identity-provider session

Cookie names:

```text
xapp_idp_session
xapp_idp_csrf
```

The production cookies are Secure, HttpOnly, SameSite=Lax, and scoped to the
issuer path `/idp`. The IdP session states that a browser authenticated a
particular tiny-idp user at an `auth_time`.

### 5.2 Application session

Cookie name:

```text
xapp_session
```

This cookie is scoped to `/`. It refers to an application session persisted by
go-go-goja. The application user is normalized from the OIDC issuer and
subject. The session contains a CSRF token used by unsafe application routes.

The important consequence is:

```text
application logout != identity-provider logout
```

Revoking `xapp_session` may leave `xapp_idp_session` valid. A subsequent OIDC
login can therefore complete without a password unless `prompt=login`,
`max_age`, password lifecycle, or another policy requires fresh
authentication. This can be correct, but it must be explicit in the UI and in
tests.

### 5.3 Subject-bound object identity

The application normalizes OIDC identity using the stable pair:

```text
(issuer, subject)
```

The object dispatcher derives a non-user-selectable object identity using the
authenticated actor, namespace, and host-owned binding key. Conceptually:

```text
actor = applicationSession.authenticatedActor

require actor.id != ""
require namespace in {"USER_STATE"}

objectID = HMAC(bindingKey, namespace || actor.id)
dispatch(namespace, objectID, request)
```

The browser and route JavaScript never receive the raw binding key and never
choose an arbitrary actor ID. Two-user tests must prove that Alice and Bob see
different application user IDs and different persistent object values.

## 6. Persistent state

The initialized state root currently has this shape:

```text
<state-root>/
├── state.json
├── identity/tinyidp.sqlite
├── audit/tinyidp.jsonl
├── secrets/token.key
├── secrets/object-binding.key
├── application/auth.sqlite
└── objects/
    ├── alarms.sqlite
    └── object databases created by the manager
```

`state.json` is written last. Its existence means initialization completed; it
is not written as an early progress marker. Initialization is idempotent and
rejects conflicting origin, client, user, or secret state rather than silently
rewriting security identity.

The current handoff fixture is:

```text
/tmp/tinyidp-xapp-real-browser
```

It contains throwaway local credentials for Alice and Bob, a self-signed
certificate, and initialized state for `https://127.0.0.1:19443`. Never reuse
these credentials outside the test. Delete the fixture when browser work is
complete.

One unresolved contract requires attention: the state root and security
directories are mode `0700`, but observed `application/auth.sqlite` and
`objects/alarms.sqlite` files are mode `0644`. Directory traversal currently
confines them, but per-file owner-only permissions may be required for safe
copying and backup. Decide and test the contract; do not silently change modes
without reviewing SQLite WAL and SHM sidecars.

## 7. What is already proven

The following implementation checkpoints exist and were committed:

- `3ca71e5`: working development login-to-private-object vertical slice;
- `acbf207`: idempotent persistent product initialization;
- `a9b562e`: initialized persistent runtime construction;
- `568367b`: initialized TLS serving lifecycle;
- `efd2c52`: layered tiny-idp/XAPP backup design ticket;
- `c849264`: detailed real-browser handoff and draft harness.

The latest real-server checkpoint established:

- a persistent state root initialized successfully;
- Bob was added to the same tiny-idp database through the real admin CLI;
- `serve-initialized` opened a TLS listener in tmux;
- `/readyz` returned HTTP 200 and `{"status":"ready"}`;
- cleanup stopped the tmux session and released port 19443.

The repository also contains focused unit, integration, property, fuzz, and
model-oriented tests for several identity interaction invariants. Read the
ticket design guide and diary before assuming a missing browser test means no
lower-level coverage exists.

## 8. What is not yet proven

The real browser has not reached the application in the handoff run. The Python
Playwright package was installed, but both version-matched cached browser paths
were absent. Therefore none of these draft browser assertions has passed yet:

- password login through the rendered IdP form;
- OIDC authorization-code plus PKCE completion in a browser;
- production cookie attributes and paths;
- CSRF rejection for missing application token;
- Alice object persistence across reload;
- Bob object separation from Alice;
- app-session logout behavior;
- restart persistence;
- user disablement effects;
- session expiry and forced reauthentication.

Do not mark task `j5ba` or `ihzp` complete until structured browser or
equivalent full-boundary evidence exists.

## 9. The current harness

The draft harness is:

```text
ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/scripts/01_real_browser_e2e.py
```

It uses two independent Chromium browser contexts, one for Alice and one for
Bob. It intends to:

1. navigate to `/` and follow the app's login redirect;
2. fill tiny-idp's login and password fields;
3. approve the OIDC interaction;
4. wait for the authenticated frontend;
5. inspect session and IdP cookies;
6. prove a write without CSRF returns 403;
7. write and reload Alice's document;
8. log Bob in and prove he cannot read Alice's value;
9. write and reload Bob's value;
10. prove Alice's value remains unchanged;
11. submit app logout without CSRF and record the result;
12. query the session endpoint after logout.

The harness is draft code. Review these points before treating it as a release
gate:

- use `/usr/bin/google-chrome` explicitly for the next launch attempt;
- verify API calls share the browser context's cookies;
- replace the fixed 100 ms object reload delay with a response- or state-based
  wait;
- close both contexts and the browser in `finally` logic on assertion failure;
- save structured JSON even when a late assertion fails;
- record expected security behavior separately from observed behavior;
- never print passwords, tokens, full cookies, or raw object binding IDs.

## 10. First-day continuation procedure

### 10.1 Establish a clean process boundary

From the workspace root:

```bash
cd /home/manuel/workspaces/2026-07-07/prod-tiny-idp
lsof-who -p 19443 -k
tmux kill-session -t tinyidp-xapp-e2e 2>/dev/null || true
```

The project requires servers to run in tmux. Do not leave `go run` attached to
an opaque background shell.

### 10.2 Inspect or recreate the fixture

The existing fixture may be reused once. Confirm owner-only password/key files
and the manifest:

```bash
find /tmp/tinyidp-xapp-real-browser -maxdepth 3 -printf '%M %p\n' | sort
sed -n '1,120p' /tmp/tinyidp-xapp-real-browser/state/state.json
```

If it is missing or suspect, recreate it using Step 36 of the investigation
diary. Do not improvise a second origin: the OIDC client redirect URI and
issuer are reconciled to `https://127.0.0.1:19443`.

### 10.3 Start the real initialized product

```bash
tmux new-session -d \
  -s tinyidp-xapp-e2e \
  -c /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp \
  "go run ./cmd/tinyidp-xapp --log-level debug serve-initialized \
    --state-root /tmp/tinyidp-xapp-real-browser/state \
    --listen 127.0.0.1:19443 \
    --tls-cert /tmp/tinyidp-xapp-real-browser/operator/tls.crt \
    --tls-key /tmp/tinyidp-xapp-real-browser/operator/tls.key \
    --maintenance-interval 10s"
```

Observe rather than assume startup:

```bash
tmux capture-pane -p -t tinyidp-xapp-e2e -S -200
curl -ksi https://127.0.0.1:19443/readyz
```

Expected readiness:

```text
HTTP/2 200
{"status":"ready"}
```

### 10.4 Make the narrow browser change

In `01_real_browser_e2e.py`, select the installed system browser:

```python
browser = playwright.chromium.launch(
    executable_path="/usr/bin/google-chrome",
    headless=not args.headed,
)
```

This is the smallest next experiment. Do not simultaneously download another
browser bundle, change application code, and rewrite the harness. One changed
variable makes the result interpretable.

### 10.5 Run the harness

```bash
cd /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp
PYENV_VERSION=3.11.4 python \
  ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/scripts/01_real_browser_e2e.py \
  --base-url https://127.0.0.1:19443 \
  --alice-password-file /tmp/tinyidp-xapp-real-browser/operator/alice-password \
  --bob-password-file /tmp/tinyidp-xapp-real-browser/operator/bob-password
```

Preserve the first failure verbatim in the diary. Fix at most one causal issue
at a time. If two legitimate fixes fail consecutively and the cause is no
longer understood, follow the repository rule and stop with:

```text
I think I'm stuck, let's TOUCH GRASS
```

### 10.6 Always clean up

```bash
lsof-who -p 19443 -k
tmux kill-session -t tinyidp-xapp-e2e 2>/dev/null || true
```

Verify no listener remains before ending the work session.

## 11. Security scenarios to characterize next

Each scenario should have four parts:

```text
precondition -> action -> expected observation -> durable evidence
```

### 11.1 App logout CSRF

Precondition: an authenticated application session exists.

Actions:

- POST `/auth/logout` without `X-CSRF-Token`;
- POST with the current session token;
- GET `/auth/logout` from an authenticated browser.

Questions:

- Must unsafe logout require CSRF?
- Should GET mutate session state?
- Does the frontend visibly transition to unauthenticated state?
- Does the IdP session intentionally remain active?

The current generic OIDC handler accepts GET and POST and does not visibly call
the session CSRF verifier. Treat that as a review finding, not a predetermined
patch. Freeze the desired contract with tests before implementation.

### 11.2 Disabled identity with an existing app session

Precondition: Alice has an active IdP session and an active XAPP session.

Action: disable Alice through the tiny-idp admin CLI while the product is
running, then access `/api/me` and `/api/object` with the existing app session.

Questions:

- Is disablement intended to be immediate or effective only at next login?
- Which component can authoritatively answer without coupling every app request
  to the IdP database?
- Should an operator action revoke both IdP and app sessions?
- How is the event audited?

Do not add an ad hoc cross-store lookup until the product contract is decided.
Immediate revocation, bounded propagation, and next-login enforcement are
different designs.

### 11.3 Forced reauthentication

Precondition: a valid IdP session exists.

Actions:

- begin authorization with `prompt=login`;
- begin authorization with expired `max_age`;
- submit an empty login;
- submit a crafted interaction form;
- submit the correct fresh password.

Expected invariant:

```text
required fresh authentication cannot be satisfied by reusing old auth_time
```

Lower-level regression tests already exist. The browser scenario proves the
rendered form and OIDC relying party preserve the parameters across redirects.

### 11.4 Password change required

Precondition: an operator assigns a password marked `must-change`.

Action: authenticate through the product.

Expected invariant:

```text
MustChangePassword => no app session and no authorization code
```

If no password-change UI exists, fail closed with a clear user-visible result.
Do not issue tokens and defer enforcement to a nonexistent later step.

### 11.5 Session expiry

Test both:

- application idle and absolute expiry;
- IdP session expiry and `auth_time` freshness.

Use an injected clock in focused tests. Do not make CI sleep for production
durations. The real browser can use short configured test durations if those
settings are exposed through an explicit test constructor or command flag.

### 11.6 Two-user isolation

Minimum invariant:

```text
write(Alice, valueA)
write(Bob, valueB)
read(Alice) == valueA
read(Bob) == valueB
actorID(Alice) != actorID(Bob)
```

Add negative cases for namespace confusion, object-name injection, missing
actor context, unknown namespace, oversized JSON, excessive keys, and excessive
nesting.

## 12. Fault-injection phase

Begin fault work only after the passing baseline. A failure test without a
known-good path cannot distinguish the injection from an unrelated broken
harness.

Use existing seams where possible:

- tiny-idp authorization and token persistence hooks;
- tiny-idp maintenance result and readiness degradation;
- go-go-objects event hooks and scheduler error handler;
- Goja CPU timeout;
- store wrappers or test-only constructors for app-session failures;
- cancellation and server shutdown contexts.

The useful failure matrix is:

| Injection | Required behavior | Evidence |
|---|---|---|
| authorize SQLite failure | no code, no consumed partial interaction | response, table counts, audit |
| token SQLite failure | no partial active token family | response, table counts, audit |
| app-session write failure | callback fails closed, no cookie | browser/network response, store |
| object write failure | explicit error, no corrupt prior value | API response, subsequent read |
| maintenance failure | readiness degrades, liveness remains | `/readyz`, `/healthz`, log |
| Goja timeout | bounded request failure, process survives | latency, status, next request |
| cancellation race | listener closes and resources close once | race test, tmux exit, port |

Prefer typed, test-only injection controls over magic environment variables,
special production routes, filesystem permission races, or string-matched log
triggers. Production code may expose a narrow dependency interface when that
improves design, but it must not expose a fault command to untrusted callers.

## 13. Diary and evidence discipline

For each meaningful step:

1. append to `reference/01-investigation-diary.md` before context is lost;
2. include the exact user prompt the first time it drives a step;
3. record exact commands and errors;
4. distinguish observation from inference and desired contract;
5. store replay scripts under the ticket's numbered `scripts/` directory;
6. store sanitized structured results under `reference/results/`;
7. relate material files with absolute paths using `docmgr doc relate`;
8. update tasks and changelog;
9. run `docmgr doctor --fail-on warning` and `git diff --check`;
10. commit focused code and documentation checkpoints.

Never store raw passwords, authorization codes, tokens, session IDs, cookie
values, binding keys, or unredacted personal identity data in ticket results.
Cookie names, paths, Secure/HttpOnly/SameSite attributes, opaque user ID
inequality, statuses, timings, and bounded hashes are sufficient evidence.

## 14. Validation commands

Run focused tests while iterating, then repository gates:

```bash
cd /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp
go test ./cmd/tinyidp-xapp/... -count=1
go test ./... -count=1
go vet ./cmd/tinyidp-xapp/...
git diff --check
docmgr doctor --ticket TINYIDP-XAPP-001 --stale-after 30 --fail-on warning
```

When a change touches go-go-goja:

```bash
cd /home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja
go test ./pkg/gojahttp/auth/... ./pkg/xgoja/hostauth/... -count=1
go test ./... -count=1
```

When a change touches go-go-objects:

```bash
cd /home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-objects
go test ./pkg/durableobjects/... ./pkg/xgoja/providers/durableobjects/... -count=1
go test ./... -count=1
```

Use the shared top-level `go.work`. Do not create another module or private Go
cache.

## 15. How to make a good first contribution

A strong first contribution is small enough to review and significant enough
to establish a security contract. The recommended sequence is:

### Contribution A: passing real-browser baseline

- select system Chrome explicitly;
- fix deterministic harness issues;
- save sanitized JSON evidence;
- add a one-command playbook script if repetition remains manual;
- document the result and commit.

### Contribution B: CSRF-safe application logout

- write failing focused tests in go-go-goja;
- decide POST and GET semantics with a reviewer;
- enforce the current session's CSRF token on unsafe logout;
- update the product frontend to send it;
- add browser assertions for app-session and retained IdP-session behavior;
- document the behavior change without a compatibility adapter.

### Contribution C: two-user and disablement contract

- promote the isolation assertions into a stable test;
- characterize disablement with an existing session;
- write a short decision note before coupling stores;
- implement only the approved propagation/revocation behavior.

Each contribution should have a clear invariant in its test name. Avoid names
such as `TestLoginWorks`; prefer names such as
`TestForcedReauthenticationCannotReuseExistingIDPSession` or
`TestLogoutWithoutCurrentSessionCSRFIsRejected`.

## 16. Exit criteria for the current continuation phase

The current browser/lifecycle phase is complete only when:

- the product is started by `serve-initialized` on a real TLS listener;
- a real browser completes OIDC password login;
- cookie attributes and paths are asserted without recording values;
- missing CSRF is rejected for every unsafe product mutation;
- Alice and Bob retain isolated persistent objects;
- the behavior survives a process restart where the contract requires it;
- app logout and IdP session retention are explicit and tested;
- disabled-user and password-change-required behavior are explicit and tested;
- results and server observations are sanitized and stored in the ticket;
- all affected repository tests and doctor validation pass;
- diary, tasks, and changelog identify the commits and remaining risks.

Fault injection can then be considered complete when each dependency failure
has an expected HTTP/readiness/audit outcome, no tested failure creates partial
security state, shutdown passes race testing, and recovery to the healthy path
is demonstrated.

## 17. Where to ask for review

Request review before making any of these decisions:

- changing app-versus-IdP logout semantics;
- introducing cross-store disablement checks;
- changing actor identity derivation or binding-key inputs;
- exposing new raw JavaScript or object gateways;
- weakening cookie security for browser convenience;
- adding proxy-header trust;
- changing production file-permission contracts;
- adding compatibility adapters between old and new auth APIs;
- changing persistent schemas or backup boundaries.

You do not need approval to improve deterministic test waits, sanitize evidence,
add focused failing regression tests, document exact observed behavior, or use
the already-approved system Chrome executable for the local harness.

## 18. Reading order

Use this order rather than reading the ticket chronologically:

1. this playbook;
2. `reference/01-investigation-diary.md`, Steps 31–36;
3. design guide sections 3–6, 10, 12, 15, and 18;
4. `cmd/tinyidp-xapp/production_app.go`;
5. `cmd/tinyidp-xapp/development_app.go`, especially
   `composeApplication`;
6. `cmd/tinyidp-xapp/app/routes/site.js`;
7. `cmd/tinyidp-xapp/app/objects/objects.js`;
8. the draft browser harness;
9. go-go-goja OIDC and session auth handlers;
10. tiny-idp interaction-hardening tests.

After that sequence, you should be able to explain:

- why two sessions exist;
- why the issuer transport is in-process but still HTTP-shaped;
- why the object name is host-derived;
- why the generated runtime is embedded in a custom Go host;
- why browser evidence is still missing;
- what the next smallest safe change is.

## 19. Final handoff state

At the time this playbook was created:

- Git branch: `task/prod-tiny-idp`;
- last durable checkpoint: `c849264`;
- no XAPP server or tmux session was running;
- port 19443 was released;
- the initialized `/tmp` fixture was preserved;
- system Chrome was available at `/usr/bin/google-chrome`;
- Playwright Python was available through pyenv 3.11.4, version 1.50.0;
- the draft harness still selected a missing Playwright-managed executable;
- the unrelated OIDF source directories under `TINYIDP-PROD-001` were
  untracked and must remain untouched.

The next engineer should change the executable selection, restart the server,
run the harness, and write the next diary step from direct evidence.
