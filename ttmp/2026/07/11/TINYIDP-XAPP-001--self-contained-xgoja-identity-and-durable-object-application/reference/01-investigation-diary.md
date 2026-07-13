---
Title: Investigation and Implementation Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/frontend/public/app.js
      Note: CSRF-bearing logout UI and explicit retained-IdP-session state committed in 99505d1
    - Path: repo://cmd/tinyidp-xapp/production_app.go
      Note: Persistent product composition and application-versus-IdP cookie configuration exercised by the real server
    - Path: repo://cmd/tinyidp-xapp/serve_initialized.go
      Note: Production-shaped TLS listener readiness maintenance and shutdown lifecycle used by the checkpoint
    - Path: repo://ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/scripts/01_real_browser_e2e.py
      Note: Draft real-Chromium two-user TLS/OIDC/CSRF/object-isolation harness described by the handoff
    - Path: ws://go-go-goja/pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Native OIDC callback session creation and current GET/POST logout semantics flagged for lifecycle review
    - Path: ws://go-go-goja/pkg/xgoja/hostauth/builder.go
      Note: Native auth route registration and session endpoint surface used by the composed XAPP
ExternalSources: []
Summary: Chronological record of the research, design, implementation, failures, decisions, verification, and review procedure for the self-contained xgoja identity and Durable Objects application.
LastUpdated: 2026-07-11T19:05:00-04:00
WhatFor: Reconstruct why the architecture and implementation took their present form, reproduce validation, and help a new contributor resume work safely.
WhenToUse: Read before changing the product host, OIDC integration, subject-to-object binding, persistence layout, or generated xgoja package.
---





# Investigation and Implementation Diary

## Step 1 — Record the requested product and delivery contract

The initiating request was:

```text
ok, now let's assess the current state of the project. I know we can add the scripting and all the model checking and all the static analysis, but is this is a usable project?

 One thing I'd be interested in is bundling tinyidp + go-go-goja express + go-go-objects (durable objects) to build a full self contained solution wher eI can log in and access my durable object + provide a HTML +JS frontend to interact with things. I want to do that with xgoja, one thing I'm not necessarily clear about is how to integrate the tinyidp in there. It could be a goja provider but maybe it also makes sense to use xgoja to generate some go code and do the actual bring up of the application and the serving of the idp and its configuration through  my own go host.

Create a new docmgr ticket to build this, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
```

The scope was then explicitly widened:

```text
this might need changes to go-go-goja express / go-go-objects potentially, that's totally possible
```

Finally, the ticket was promoted from a design artifact to an implementation program:

```text
Then create detailed phases + detgailed  tasks per phase (if necessary), and then implement step by step, keeping a detailed diary as you work, committing at appropriate intervals.
```

This means the work is allowed to modify all three repositories. It must not treat their current APIs as immutable when a smaller, safer cross-repository contract is possible. It must also leave resumable task and diary state after each implementation interval.

## Step 2 — Create the ticket before investigation artifacts spread

Created docmgr ticket `TINYIDP-XAPP-001`, titled “Self-contained xgoja identity and durable object application.” The ticket root is:

```text
ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/
```

Created a design document, this diary, tasks, changelog, ticket index, README, and `sources/` packet. This kept architectural conclusions and the evidence that motivated them in one searchable unit.

## Step 3 — Read the integration surfaces as code, not only as README claims

Inspected the tiny-idp embedded-provider API, especially `pkg/embeddedidp/options.go` and `pkg/embeddedidp/provider.go`. Important findings:

- `embeddedidp.Options` already exposes production-relevant store, cookie, token, audit, consent, limiter, address-resolution, authenticator, password, and maintenance configuration.
- `embeddedidp.New` returns an ordinary `http.Handler` and provider lifecycle object.
- An issuer with a path component, such as `https://host.example/idp`, is supported; the provider derives its endpoint prefix from the issuer.
- The provider exposes liveness, readiness, maintenance, and close operations suitable for a custom host.
- Production-mode validation is intentionally stricter than development mode and should remain the authority for identity configuration.

Inspected go-go-goja Express, host authentication, provider APIs, host services, generated runtime-package examples, and generated host-auth examples. Important findings:

- Express planned routes can consume a host-authenticated actor through `ctx.actor` and express authorization/CSRF/audit policy declaratively.
- The OIDC adapter is behaviorally generic but remains named `keycloakauth` in its package and some application fields.
- OIDC discovery runs synchronously in the auth builder, before the outer host starts serving.
- xgoja can generate an importable Go runtime package, TypeScript declarations, embedded JavaScript, and embedded frontend assets.
- xgoja HostServices are the intended injection point for Go-owned services.

Inspected go-go-objects manager, gateway, bundle, manifest, storage, alarms, idle eviction, xgoja provider, TypeScript surface, and examples. Important findings:

- The xgoja provider and embedded bundle path already exist.
- A manager serializes dispatch per object and uses SQLite-backed storage per object.
- Promise-aware JavaScript dispatch, CPU deadlines, alarms, and idle eviction are implemented and tested.
- The generic gateway accepts caller-supplied namespace and object name. It is a developer/low-level API, not an end-user authorization boundary.
- The project describes its runtime as experimental and does not yet provide storage quotas, distributed ownership, or mature schema migrations.

## Step 4 — Preserve the primary local reference packet

Copied nine directly relevant, locally maintained primary references into `sources/`:

1. xgoja v2 reference.
2. Express auth host integration guide.
3. Provider runtime configuration and HostServices guide.
4. Generated host-auth example guide.
5. Generated host-auth xgoja specification.
6. go-go-objects README.
7. Durable Objects xgoja specification.
8. tiny-idp developer guide.
9. tiny-idp reference.

The packet contains approximately 2,600 lines. It intentionally favors repository-local API and design sources over secondary web summaries because the implementation must match the checked-out versions in the shared `go.work`.

## Step 5 — Establish the baseline before proposing changes

Initially attempted to orchestrate the three complete test suites in parallel. The command wrapper yielded before all child processes returned and therefore did not preserve trustworthy final exit statuses. This was an orchestration deficiency, not a software failure. I did not count this run as evidence.

Reran each suite conclusively under the existing shared `go.work`, without creating a private module cache:

```text
cd tiny-idp && go test ./... -count=1
PASS

cd go-go-goja && go test ./... -count=1
PASS

cd go-go-objects && go test ./... -count=1
PASS
```

The tiny-idp suite completed in approximately 16.5 seconds. The other two included xgoja generation/runtime, Express/hostauth, provider, example, manager, storage, gateway, and JavaScript integration coverage. This baseline matters: new failures can now be attributed to this ticket's changes rather than assumed pre-existing breakage.

## Step 6 — Identify the first security-critical integration defect

The app and IdP are intended to share one externally visible origin. The OIDC relying party performs discovery during construction. If the public issuer is `https://host.example/idp`, and the server owning that origin is not listening until after construction, normal network discovery introduces a startup cycle:

```text
construct app auth
  -> discover https://host.example/idp/.well-known/openid-configuration
  -> outer server is not yet serving
  -> construction cannot complete
  -> outer server never starts
```

Starting a partially configured public server merely to satisfy discovery would create fragile readiness and race behavior. Rewriting the issuer to a loopback URL would change the security identity of the issuer and break external token verification assumptions.

Decision: add an injectable HTTP client to the generic OIDC adapter, and provide an origin-restricted `RoundTripper` that routes only the configured public issuer origin to the in-process tiny-idp handler. Authorization redirects remain public browser URLs. Discovery, JWKS retrieval, and back-channel token exchange use the injected transport without making a socket connection.

Review invariant: the transport must compare normalized scheme and authority, reject userinfo/fragments/relative URLs, preserve the public request URL seen by the handler, and fail closed for every other origin. It must not become an unrestricted SSRF proxy or a general URL rewriter.

## Step 7 — Identify the durable-object authorization boundary

The low-level durable-object API accepts `(namespace, name)`. Passing either value through from the browser would allow an authenticated user to guess or select another user's object. Authentication of the HTTP request does not make a caller-supplied object identifier trustworthy.

Decision: define an actor-bound dispatcher. It accepts the authenticated application actor and an allowlisted logical namespace, derives the physical object name in Go, and only then calls the manager.

The v1 derivation is conceptually:

```text
principal = canonical(public_issuer) || NUL || oidc_subject
appUserID = base64url(SHA-256(principal))
objectName = "v1_" || base64url(HMAC-SHA-256(binding_key,
    namespace || NUL || appUserID))
```

The binding key is owner-only persistent state. Raw `/rpc/:namespace/:name` and `/fetch/:namespace/:name` gateways remain disabled in product mode. The browser receives neither the physical name nor the binding key.

## Step 8 — Choose the composition architecture

Compared two broad options:

- Put tiny-idp entirely behind an xgoja provider.
- Generate an importable xgoja runtime package and let a small handwritten Go host own process lifecycle.

Selected the second option for the first product slice. Identity initialization, persistent stores, maintenance, readiness, shutdown ordering, backup/restore, and native route ownership are process-level concerns. Hiding them behind JavaScript module initialization would make partial startup harder to reason about. The generated package remains valuable for module registration, TypeScript declarations, scripts, route assets, and repeatable runtime assembly.

The intended boundary is:

```text
handwritten Go host
  owns configuration, stores, tiny-idp, app sessions, OIDC RP,
  bound object service, mux, maintenance, readiness, shutdown

generated xgoja runtime package
  owns selected providers, JS route/object bundles, embedded assets,
  module/DTS registry, and validated runtime construction
```

After a working host proves the lifecycle, the reusable tiny-idp portion can be extracted into an xgoja provider. That sequencing ensures the provider API is derived from concrete lifecycle requirements.

## Step 9 — Write the intern-ready design and implementation guide

Wrote a 1,122-line design guide. It includes the product contract, current-state assessment, API maps, trust boundaries, session model, startup-cycle analysis, actor/object binding, proposed cross-repository APIs, xgoja specification, JavaScript examples, persistence, threat model, operational model, phased implementation, test matrix, decision records, risks, file map, and acceptance criteria.

The guide's core usability conclusion is deliberately narrow: the three codebases contain usable and tested building blocks, but they do not yet constitute a turnkey production product. The ticket exists to build and verify the missing composition and operational layer.

## Step 10 — Convert the design into long-running implementation state

Created phases 0 through 8 with 69 detailed tasks covering:

- product seam and architecture approval;
- provider-neutral OIDC and in-process issuer transport;
- tiny-idp application embedding;
- subject-bound durable objects;
- custom host and generated runtime composition;
- frontend product loop;
- persistence, backup, restore, and runbooks;
- later tiny-idp provider extraction;
- assurance and release evidence.

Ran `docmgr task migrate --ticket TINYIDP-XAPP-001`, which stamped every task with a stable identifier. These IDs allow diary entries and commits to refer to tasks even if wording or ordering changes.

## Step 11 — Begin implementation planning after explicit authorization

Inspected the precise implementation sites for the first two cross-repository seams:

```text
go-go-goja/pkg/gojahttp/auth/keycloakauth/keycloakauth.go
go-go-goja/pkg/gojahttp/auth/keycloakauth/keycloakauth_test.go
go-go-goja/pkg/xgoja/hostauth/builder.go
go-go-objects/pkg/durableobjects/manager.go
go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go
```

Confirmed that `oidc.NewProvider` can receive an HTTP client through `oidc.ClientContext`, and that OAuth token exchange and remote-key verification must also run with that client-bearing context. Merely injecting the client during discovery would be incomplete: later callback requests could still dial the public origin.

The first implementation interval will therefore:

1. Introduce provider-neutral OIDC naming without a compatibility adapter.
2. Add an explicit OIDC HTTP client dependency.
3. Implement and unit-test the fail-closed in-process issuer transport.
4. Thread the same client through discovery, token exchange, and JWKS verification.
5. Update hostauth and its tests.
6. Run targeted tests, the complete go-go-goja suite, formatting, and static checks.
7. Update this diary and commit the dependency phase before beginning object binding.

## Current review procedure

At any resume point, use this sequence:

```text
1. Read tasks.md and find the first unchecked stable task ID.
2. Read the latest diary step and its explicit review invariant.
3. Inspect git status in tiny-idp, go-go-goja, and go-go-objects.
4. Preserve unrelated user changes; never stage them with ticket work.
5. Make one cohesive implementation increment.
6. Run targeted tests first, then the affected repository's full suite.
7. Update the diary with commands, failures, decisions, and remaining risks.
8. Mark only acceptance-proven tasks complete.
9. Review the diff and commit only the cohesive increment.
```

## Current state

- Research packet: complete for the initial architecture.
- Design guide: drafted and internally consistent.
- Task graph: 69 stable-ID tasks across nine phases.
- Baseline: all three repository test suites pass.
- Product code changed by this ticket so far: cross-repository foundation implemented and committed.
- Next task: create the generated product skeleton and custom tiny-idp host.

## Step 12 — Implement provider-neutral OIDC and same-process back-channel transport

Renamed `pkg/gojahttp/auth/keycloakauth` to `pkg/gojahttp/auth/oidcauth` without retaining an adapter package. Renamed the app/session API and fresh schema from `KeycloakSub`/`keycloak_sub` to `OIDCSubject`/`oidc_subject`. Updated hostauth, examples, tests, and current documentation.

Added `Config.HTTPClient` to `oidcauth`. The same client-bearing context is now used for all three back-channel stages:

```text
oidc.NewProvider       discovery
oauth2.Config.Exchange token exchange
IDTokenVerifier.Verify remote JWKS lookup
```

Added `NewInProcessIssuerTransport`. It accepts one validated absolute HTTP(S) issuer URL and one `http.Handler`. It rejects nil handlers, relative URLs, non-HTTP schemes, userinfo, issuer query/fragment components, a different scheme or authority, and paths outside the issuer prefix.

Added a complete no-dial test. The fake issuer advertises `https://identity.example.test/idp`; no server owns that address. Discovery, authorization, token exchange, and JWKS verification still complete because only the OIDC back-channel client uses the in-process transport. Existing tests continue to reject bad state, nonce, audience, and expiry.

Targeted validation:

```text
go test ./pkg/gojahttp/auth/keycloakauth -count=1
ok (before the package rename)

go test ./pkg/gojahttp/auth/... ./pkg/xgoja/hostauth \
  ./examples/xgoja/19-express-keycloak-auth-host/cmd/host -count=1
ok

go test ./... -count=1
ok
```

The first commit invocation displayed only the lefthook startup banner because the tool returned before the long hook finished. The staged set remained intact. A controlled retry with a longer terminal window showed the real hook progress. Generation, the full test suite, golangci-lint, Go vet, and glazed-lint passed.

Committed in go-go-goja:

```text
ebc5600 Auth: generalize OIDC and add in-process transport
```

## Step 13 — Add an authenticated actor context for trusted native services

The desired JavaScript API must not accept `actorID` because route/body/query data could be substituted accidentally. Added `gojahttp.ContextWithActor` and `ActorFromContext`. The planned dispatcher installs the actor only after authentication, CSRF/resource checks, and authorization have succeeded, then passes that context through the initial handler call and Promise settlement calls.

Added an integration test whose native Goja handler reads `runtimebridge.CurrentOwnerContext(vm)` and verifies that it contains `context-user`. The first test draft omitted the action/authorizer required by planned user routes and failed at registration with:

```text
planned user route GET /context-actor requires .allow(action)
```

The test was corrected to use the same deny-by-default route contract as production. Targeted and complete go-go-goja suites then passed. The pre-commit hook also passed generation, tests, lint, vet, and glazed-lint.

Committed in go-go-goja:

```text
ce36bf8 HTTP: expose authenticated actor to native route services
```

## Step 14 — Implement subject-bound Durable Objects

Added `pkg/durableobjects/bound_dispatcher.go`. Its constructor requires:

- a non-nil dispatcher;
- at least 32 bytes of binding key material;
- at least one explicitly allowlisted, syntactically valid namespace.

For a trusted actor ID and namespace it derives:

```text
v1_ + base64url(HMAC-SHA-256(binding_key, namespace || NUL || actor_id))
```

The dispatcher copies the key, never emits the actor ID in the physical name, and rejects every caller-supplied `ObjectID`, including partially populated values. Tests prove deterministic mapping, different actor mappings, namespace denial, object-name injection denial, short-key denial, key-copy behavior, and actual two-user isolation using separate SQLite-backed counter actors.

Extended the xgoja provider with `BoundDispatcherService`, `rpcForActor`, `fetchForActor`, and matching TypeScript declarations. The JavaScript functions contain no actor-ID or object-name parameter. A host-supplied `ActorID(context.Context)` resolver reads identity from the trusted route context.

Also corrected a pre-existing Go-to-JavaScript codec mismatch: `FetchRequest`/`FetchResponse` TypeScript used JSON-style lowercase fields, while direct `vm.ToValue` struct conversion exposed Go field names. Fetch inputs and outputs now traverse JSON normalization, so `path`, `status`, and `body` match the declaration.

The raw caller-named `gateway()`/`handler()` and automatic `/rpc`/`/fetch` mounts are now disabled by default. Low-level examples opt in with `enableRawGateway: true`. Product hosts use the bound service.

Targeted and shared-workspace validation passed:

```text
go test ./pkg/durableobjects ./pkg/xgoja/providers/durableobjects -count=1
ok

go test ./... -count=1
ok
```

## Step 15 — Let the standalone hook improve the dependency boundary

The first go-go-objects commit attempt failed in its `GOWORK=off` hook:

```text
undefined: gojahttp.ActorFromContext
undefined: gojahttp.ContextWithActor
```

The shared workspace saw the new go-go-goja commit, but go-go-objects' released dependency did not. Updating to an unpublished pseudo-version or relying on `go.work` would make the library commit non-reproducible.

Changed the design instead. `BoundDispatcherService` now carries an `ActorID func(context.Context) (string, error)` resolver. go-go-objects remains ignorant of go-go-goja's context key. The product host, which imports both repositories, will implement the resolver with `gojahttp.ActorFromContext`. This is a cleaner inversion boundary and allows alternate authenticated hosts.

Conclusive standalone validation:

```text
GOWORK=off go test ./... -count=1
ok
```

The retrying commit hook passed standalone tests, golangci-lint, Go vet, glazed-lint, and logcopter generation checks.

Committed in go-go-objects:

```text
ec73ddd Objects: add actor-bound private dispatch
```

## Step 16 — Scope application identity by issuer and subject

OIDC `sub` is unique only within one issuer. The prior app store indexed only the subject, so reusing a database across issuer changes or multiple issuers could merge distinct people.

Added `OIDCClaims.Issuer`, populated from the already verified ID token. App users now store both `OIDCIssuer` and `OIDCSubject`. `OIDCUserID` rejects empty/NUL-containing components and derives an opaque stable ID from SHA-256 over `issuer || NUL || subject`. Memory and SQL stores index the full tuple. SQLite and PostgreSQL fresh schemas use a unique `(oidc_issuer, oidc_subject)` index.

Added memory and SQLite tests proving that one subject string from two issuers creates two identities. Updated hostauth to retain `oidcIssuer` and `oidcSubject` as session claims.

The initial commit accidentally omitted the sibling internal store-contract file from the staged directory. The working tree and hook tests included it, but the commit did not. Staged that exact file and amended the commit; the hook passed again.

Committed in go-go-goja:

```text
43a69d7 Auth: scope application users by OIDC issuer
```

## Checkpoint after cross-repository foundations

Acceptance-proven task IDs completed in this interval:

```text
jam5  77da  5mkm  paaq  g7o1  2zhc  sypf
d5yg  71id  eg3d  bbyu
```

Deliberately still open:

- persistent OIDC transaction storage;
- creation and persistence of the product binding key;
- disabled-user end-to-end object denial;
- request/value/nesting limits and quotas;
- the generated runtime package and actual tiny-idp product host.

## Step 17 — Validate, commit, and publish the checkpoint

Related the design to eight concrete implementation files across the three repositories. The first docmgr relation attempt interpreted sibling `../go-go-goja` and `../go-go-objects` paths relative to the ticket directory and produced six missing-file warnings. Replaced those generated paths with explicit `ws:///go-go-goja/...` and `ws:///go-go-objects/...` anchors.

Conclusive documentation validation:

```text
docmgr doctor --ticket TINYIDP-XAPP-001 --fail-on warning
All checks passed

git diff --check
clean
```

Committed the ticket and preserved source packet in tiny-idp:

```text
e93d304 Docs: design self-contained identity object app
```

Uploaded one table-of-contents bundle containing the ticket index, 1,139-line design guide, 400-line diary, phased tasks, and changelog:

```text
TINYIDP-XAPP-001 Self Contained Identity Objects e93d304.pdf
/ai/2026/07/11/TINYIDP-XAPP-001
```

The uploader returned `OK: uploaded`; no redundant cloud listing was performed.

## Step 18 — Create and generate the product skeleton

Created `cmd/tinyidp-xapp` inside the existing top-level tiny-idp module. No nested `go.mod` or private module cache was introduced.

The product source now contains:

```text
cmd/tinyidp-xapp/
  main.go                         Glazed/Cobra root with shared logging/help
  doctor.go                       source-layout diagnostic command
  xgoja.yaml                      runtime-package specification
  app/routes/site.js              trusted Express planned routes
  app/objects/objects.js          USER_STATE Durable Object bundle
  app/frontend/                   pnpm + pinned Bootstrap minimal shell
  app/types/xgoja-modules.d.ts    generated declarations
  internal/xgojaruntime/          generated importable runtime and assets
```

The route script serves the index explicitly at `/`, static assets only under `/static`, session identity at `/api/me`, and actor-bound reads/writes at `/api/object`. Writes require host CSRF enforcement. The script calls `fetchForActor("USER_STATE", ...)` and contains no actor-ID or object-name parameter.

The object bundle validates documents before SQLite persistence with explicit encoded-size, total-key, and nesting bounds. The UI is intentionally the permitted initial minimal shell rather than a premature React application. It uses pnpm to pin Bootstrap and copies the minified CSS into the embedded public tree; `node_modules` is ignored.

The first `gen-dts` invocation failed with the expected actionable message:

```text
Error: --out is required unless the v2 spec has a dts artifact with output
```

Added an explicit strict `dts` artifact. The first successful generation wrote artifacts relative to the process working directory, which put them at root-level `app/` and `internal/xgojaruntime/`. Deleted those generated files, changed outputs to repository-root-qualified `cmd/tinyidp-xapp/...` paths, and regenerated.

The initial runtime-package artifact did not embed the `application-routes` jsverbs source. Added all three source IDs to the runtime-package artifact. The regenerated runtime plan now uses:

```text
xgoja_embed/jsverbs/application_routes
xgoja_embed/assets/frontend_assets
xgoja_embed/assets/object_bundle
```

Added `go:generate` directives that deliberately change to the repository root before invoking the sibling xgoja command, preserving the root-relative artifact contract.

Validation:

```text
pnpm install && pnpm run build
ok

go run ../go-go-goja/cmd/xgoja doctor -f cmd/tinyidp-xapp/xgoja.yaml
schema, module resolution, and all three source plans: ok

go generate ./cmd/tinyidp-xapp
ok

go test ./cmd/tinyidp-xapp/... -count=1
ok

go run ./cmd/tinyidp-xapp doctor --output json
all ten required source/generated files: ok

go test ./... -count=1
ok
```

The generated bundle test verifies that provider registration succeeds, declarations contain `express`, `fs:assets`, `rpcForActor`, and `fetchForActor`, and the embedded plan contains the four selected runtime modules.

Committed in tiny-idp:

```text
5176052 App: scaffold generated identity object runtime
```

This completes the generation seam, not the product lifecycle. The next open task is the custom host that owns persistent tiny-idp, application auth/session stores, the binding key and manager, HostServices injection, route loading, listener, readiness, maintenance, and shutdown.

## Step 19 — Make IdP cookies explicit at the combined-host boundary

Continued with Phase 2 task `o3fe`, because a combined origin has at least three
independent browser-state domains: the tiny-idp login session, the tiny-idp CSRF
nonce, and the application's own session. Fixed literal cookie names in the
Fosite adapter would make independently configured applications collide and
would prevent a host from stating its cookie ownership policy explicitly.

Extended `embeddedidp.CookieConfig` with `SessionName`, `CSRFName`, and `Path`.
The embedded provider passes these values into the adapter, and every read and
write of either cookie now uses provider-owned configuration. Defaults preserve
the existing names and derive the path from the issuer URL. A host may broaden
the path (for example, `/`) when necessary, but configuration validation rejects:

- equal session and CSRF names;
- whitespace and HTTP separator/control characters in names;
- relative or header-unsafe paths;
- a configured path that does not contain the issuer path.

The adapter repeats the direct-caller validation rather than relying only on the
public embedding layer. This protects internal tests and any future package-level
composition from constructing an unusable cookie policy.

Initial formatting and focused tests passed before the new scenario was added:

```text
go test ./internal/fositeadapter ./pkg/embeddedidp -count=1
ok
```

Added a full cookie coexistence scenario. It configures issuer `/idp`, names the
IdP cookies `xapp_idp_session` and `xapp_idp_csrf`, supplies an unrelated
`xapp_session=host-owned-session`, completes password login, and performs silent
authorization with both the application and IdP session cookies. It also asserts
that no default tiny-idp names are emitted and both configured cookies carry the
requested path.

The first run returned `404 page not found` while fetching CSRF. This was useful
evidence: direct adapter routes are mounted under the configured issuer path, so
the test had incorrectly requested `/authorize` instead of `/idp/authorize`.
Changed the scenario base URL to include `/idp`.

The second run issued a valid authorization code with HTTP 303, while the test
only admitted HTTP 302. Updated the acceptance condition to match the provider's
existing successful redirect contract (`Found` or `See Other`). The focused
suite then passed:

```text
ok github.com/manuel/tinyidp/internal/fositeadapter
ok github.com/manuel/tinyidp/pkg/embeddedidp
```

## Step 20 — Remove a stale generation side effect without losing embedded assets

`git status --short` revealed an untracked root-level
`internal/xgojaruntime/xgoja_embed/assets` tree. This was a stale artifact from
the earlier generation attempt that ran before artifact outputs were made
repository-root-qualified. It duplicated the checked-in generated package under
`cmd/tinyidp-xapp/internal/xgojaruntime`.

I first removed the `embedded-assets` artifact marker to test whether the runtime
package's own source list was sufficient. Regeneration proved that this was not
correct: xgoja intentionally uses the separate marker to decide whether to copy
asset sources and inject `EmbeddedAssets` into `app.HostOptions`. Restored the
marker, added a comment explaining the non-obvious two-part contract, removed
only the known stale generated files, and regenerated.

The final reproducibility check established all three conditions:

```text
go generate ./cmd/tinyidp-xapp
xgoja generate ok: cmd/tinyidp-xapp/internal/xgojaruntime

find internal/xgojaruntime -type f
(no output)

git diff --exit-code -- \
  cmd/tinyidp-xapp/internal/xgojaruntime/xgoja_runtime.gen.go \
  cmd/tinyidp-xapp/app/types/xgoja-modules.d.ts
(exit 0)
```

The generated package still contains `//go:embed all:xgoja_embed/assets/*` and
passes `embeddedAssets` to the host. The correction therefore removes accidental
root output without weakening the single-binary asset contract.

## Step 21 — Run the repository-wide gate for the cookie checkpoint

Ran the entire tiny-idp Go suite through the shared `go.work` after the focused
tests and regeneration check:

```text
go test ./... -count=1
```

All command, adapter, authn, persistence, scenario, generated runtime, static
analysis helper, and external-consumer packages passed. No private module cache
or nested module was introduced. The two pre-existing untracked OIDF source
trees under the unrelated production-review ticket remain deliberately
unstaged.

## Step 22 — Commit the cookie boundary checkpoint

Reviewed the implementation diff, ran `git diff --check`, staged only the eleven
implementation, focused-test, and xgoja-spec files, and committed:

```text
577f253 IDP: configure embedded cookie ownership
```

The ticket documents and unrelated OIDF research trees were not included in the
implementation commit. Task `o3fe` is complete based on the focused coexistence
test, invalid-configuration table, deterministic generation check, and full
repository test gate above.

## Step 23 — Build the first custom development host

Started Phase 0 task `w8ei` and the Phase 4 composition seam by implementing a
Go-owned development application rather than delegating lifecycle to the
generated `serve` command. The custom host now constructs and owns:

- an in-memory tiny-idp store with an exact public PKCE relying-party client;
- one password-backed development user and an ephemeral RSA signing key;
- an `embeddedidp.Provider` mounted below `/idp`;
- an origin/path-restricted in-process OIDC transport for discovery, token
  exchange, and JWKS retrieval;
- application auth/session/audit/authorization services from `hostauth`;
- one external `gojahttp.Host` with raw routes rejected;
- the generated xgoja runtime package and its embedded assets;
- a Go-owned Durable Objects server using SQLite under the selected state root;
- an HMAC `BoundDispatcher` restricted to `USER_STATE`;
- trusted route loading before the HTTP handler becomes available;
- exact native `/auth/*` handlers, the IdP prefix, and the Express fallback in
  one outer Go 1.22+ `ServeMux`;
- reverse-order runtime, object, auth-store, and IdP shutdown.

Added a Glazed `serve` command with explicit `--listen`, `--public-base-url`,
`--state-root`, `--login`, and development-only `--password` fields. It starts a
bounded `http.Server` with read-header, read, write, and idle timeouts and uses
`errgroup` to coordinate serving with graceful context-driven shutdown. No
environment lookup, nested module, or private Go cache was introduced.

Added owner-only binding-key initialization at
`<state-root>/secrets/object-binding.key`. Creation uses 32 random bytes,
`O_CREATE|O_EXCL`, directory mode `0700`, and file mode `0600`. A second load
returns the same bytes. This is the first persistent security root in the
product host and prevents a restart from silently remapping every actor to a
new physical object.

The first compile exposed three mechanical issues: the standard and pkg/errors
packages had the same import name, `cmds.WithFlags` accepts multiple
`*fields.Definition` values rather than one variadic `fields.New` call, and the
xgoja app package alias was missing. Corrected those directly. The focused
command package then compiled and its prior tests passed.

## Step 24 — Find and repair multi-provider HTTP-host composition

The first real construction test failed while registering the Express module:

```text
http host service "go-go-goja-http.host" must be ExternalHostService,
got []interface {}
```

This was a cross-provider contract failure. The custom host contributed its
external HTTP host, and the Durable Objects capability also contributed an HTTP
host value. `app.HostServices` correctly represents service keys as ordered
multi-values, but the HTTP provider called the singular `HostService` accessor;
that accessor returns the complete slice when more than one value exists.

Changed go-go-goja's HTTP provider to consume `HostServiceValues`, validate each
contribution, and select the first non-nil host deterministically. Added a
regression test with two composed contributions and proved the custom host can
then construct its runtime.

The initial handler-level host test now proves:

- `/` returns the generated HTML;
- `/static/app.js` is served from embedded assets;
- `/api/me` and `/api/object` reject unauthenticated access;
- `/rpc/USER_STATE/injected` remains unavailable;
- `/idp/.well-known/openid-configuration` publishes the exact combined-host
  issuer.

Ran the actual server under tmux on port `18787`, captured its pane, and queried
the live listener. Results were `root=200`, `discovery=200`, and `private=401`.
`/auth/login` returned a 302 to the embedded issuer with an exact callback,
state, nonce, S256 PKCE challenge, and `openid profile email` scopes. Sent
Ctrl-C through tmux, checked/killed the port with `lsof-who`, and removed the
tmux session.

## Step 25 — Extend the smoke test through login and stop on a repeated token failure

Added an end-to-end `httptest` scenario with a real cookie jar. It follows
`/auth/login` to the embedded credential form, extracts the interaction and
CSRF values, submits Alice's password, follows the authorization callback,
reads the application session, and is intended to write and reread the private
object.

The first run reached the callback with `error=access_denied`. The form helper
had captured both submit-button values and retained the final `deny` value.
Explicitly selecting `action=approve` corrected the harness and produced a real
authorization code.

The callback then failed at token exchange:

```text
401 oidc token exchange failed
```

The first protocol hypothesis was OAuth2 client-auth probing. Go's OAuth2
client can probe HTTP Basic before retrying body parameters; a strict provider
may consume a one-time code during a malformed first request. Changed
go-go-goja's OIDC adapter to set `AuthStyleInParams` when `ClientSecret` is
empty, with a focused public-client configuration test. Both the OIDC and HTTP
provider suites passed, but the integrated tiny-idp exchange still returned the
same generic 401.

At that point two consecutive protocol-level corrections had not resolved the
same failure. In accordance with the repository debugging rule, stopped without
committing the incomplete host and reported:

```text
I think I'm stuck, let's TOUCH GRASS
```

The next authorized step is narrow observation of the failed token endpoint
response. It must preserve the response body for the OAuth client, avoid logging
tokens or credentials, and distinguish client authentication, redirect/PKCE,
code lifecycle, and persistence errors before another implementation change.

## Step 26 — Instrument the failed back channel and repair server metadata

Added a development-only observing RoundTripper around the already restricted
in-process transport. It buffers only failed responses, restores the body for
the OAuth client, caps observation at 64 KiB, and records method, issuer-relative
path, status, and error body. It never records request bodies, authorization
codes, PKCE verifiers, cookies, or headers.

The next vertical test produced the exact failure:

```text
POST /idp/token
500
{"error":"server_error","error_description":"resolve client address failed"}
```

The transport had supplied the outbound client-side `http.Request` directly to
an `http.Handler`. Client-side requests do not carry server connection metadata,
so `RemoteAddr` was empty and tiny-idp's fail-closed client-address resolver
rejected the token request. This was unrelated to client authentication, PKCE,
or authorization-code consumption.

Changed `InProcessIssuerTransport` to clone the request into a server-facing
view, set `RequestURI` from the absolute URL, and provide the explicit loopback
peer `127.0.0.1:0` when no peer exists. The exact-origin and issuer-path checks
still run before dispatch. Added a test that verifies both server metadata
fields. The focused OIDC and HTTP-provider suites passed.

The token exchange then succeeded and the test created an application session.

## Step 27 — Define implicit-self authorization for actor-bound writes

The authenticated object read returned `{}`, but the CSRF-protected write was
denied with 403. The route uses action `user.self.update` and deliberately has no
request-selected resource: its physical Durable Object identity is derived from
the authenticated actor after authorization. The generic app authorizer had
required an explicit `user` resource even for the self action.

Refined the action contract as follows:

```text
user.self.update + no resource
    => authenticated actor's implicit self; allow

user.self.update + explicit resource
    => require type=user and resource.id == actor.id
```

This does not allow selection of another object. Explicit resources retain the
same equality check, while the no-resource form is appropriate only for trusted
routes whose downstream service derives ownership from the enforced actor.
Added the implicit-self positive case without weakening the existing
other-user negative case.

The complete vertical scenario then passed:

```text
/auth/login
  -> /idp/authorize credential + CSRF interaction
  -> /auth/callback with code
  -> in-process /idp/token and /idp/jwks
  -> xapp_session
  -> GET /api/object                    200 {}
  -> POST /api/object + X-CSRF-Token    200 {"message":"private"}
  -> GET /api/object                    200 {"message":"private"}
```

## Step 28 — Eliminate the shadow Durable Objects manager

After the successful path, `git status` showed an unexpected untracked
`var/durable-objects/alarms.sqlite`. The provider constructed the manager from
module configuration before looking up the host-supplied manager, then replaced
the configured manager during module initialization. The losing manager still
created storage and background resources.

Reordered construction so external gateway service lookup happens first. When
a host manager exists, module configuration is not allowed to instantiate a
second manager. Added a regression test with an invalid configured bundle path
and a sentinel storage root; loader construction succeeds via the external
manager and the sentinel directory remains nonexistent. Removed the known
generated `var/` artifact and verified subsequent product tests do not recreate
it.

Also strengthened the persistent binding-key initializer. Existing keys must
be exactly 32 bytes and mode `0600`; new keys are synced before close; short or
failed writes remove the incomplete file. Tests cover stable reload, exact
length, owner-only creation, and rejection of loose existing permissions.

## Step 29 — Validate and commit the first vertical product

Conclusive full-suite gates:

```text
go-go-goja:    go test ./... -count=1    PASS
go-go-objects: go test ./... -count=1    PASS
tiny-idp:      go test ./... -count=1    PASS
```

The go-go-goja pre-commit hook additionally ran generation, golangci-lint,
Glazed lint/vet, and its complete test suite. The go-go-objects pre-commit hook
ran `GOWORK=off go test ./...`, golangci-lint, Glazed lint/vet, and logcopter
validation. Product-focused `go vet ./cmd/tinyidp-xapp/...` also passed.

Committed independently in dependency order:

```text
2d7878d  go-go-goja     HTTP: harden composed in-process OIDC hosts
46ba195  go-go-objects  Objects: prefer host-owned manager without side effects
3ca71e5  tiny-idp       App: run embedded identity object vertical slice
```

The implementation commits exclude ticket documents and the unrelated OIDF
source trees. The product is now usable as a development vertical slice. It is
not yet a production service: identity and application auth stores remain
in-memory, initialization is not an operator command, aggregate readiness and
backup/restore are absent, and the default development credential must not be
used outside the explicitly labeled command.

## Step 30 — Publish the vertical-slice checkpoint

Ran the reMarkable bundle workflow as a dry run first, explicitly ordering the
ticket index, design guide, diary, tasks, and changelog and selecting ToC depth
2. The preflight resolved all five inputs and the intended remote directory.
Uploaded the resulting bundle non-interactively:

```text
TINYIDP-XAPP-001 Vertical Slice 86595c6.pdf
/ai/2026/07/11/TINYIDP-XAPP-001
OK: uploaded
```

No overwrite was required because the checkpoint has a unique commit-qualified
name; prior ticket bundles and annotations remain intact.

## Step 31 — Implement idempotent persistent-state initialization

Implemented the `tinyidp-xapp init` Glazed command and a reusable
`InitializeState` reconciler. The command accepts an owner-only password file,
not a password value or environment variable, and clears the loaded byte slice
after use. Production initialization requires a canonical HTTPS origin with no
path, query, fragment, or userinfo.

The state-root layout is now explicit:

```text
<root>/
  state.json                         completion manifest, mode 0600
  identity/tinyidp.sqlite            migrated tiny-idp store, mode 0600
  audit/tinyidp.jsonl                durable initialization/audit stream
  secrets/token.key                  32-byte token/CSRF root, mode 0600
  secrets/object-binding.key         32-byte actor binding root, mode 0600
  application/auth.sqlite            reserved persistent app-auth store
  objects/                           per-object SQLite state
```

Initialization performs migrations and reconciles the exact public PKCE client,
first password credential, active RSA signing key, token secret, and object
binding key. `state.json` is written last, so its presence is the completion
marker rather than evidence that an earlier partial step merely started. The
manifest temporary file is owner-only, fully written, synced, closed, and then
renamed.

Reruns preserve existing secrets, signing keys, and credential hashes. They
reject a conflicting public origin, client redirect/scope contract, disabled or
conflicting first user, corrupt key length, or loose permissions. The
initialization tests run the reconciler twice with a different second password
and prove that no root or credential changes. Additional tests prove incomplete
state refusal, HTTPS validation before manifest mutation, password-file newline
handling, and rejection of a mode-0644 password file.

Validation:

```text
go test ./cmd/tinyidp-xapp/... -count=1    PASS
go vet ./cmd/tinyidp-xapp/...              PASS
go test ./... -count=1                     PASS
```

Committed:

```text
acbf207 App: initialize persistent product security state
```

This completes initialization and state layout, but not production serving. The
next step must construct the combined host from this state, refuse an absent or
incomplete manifest, use persistent application auth/session/audit stores, and
aggregate readiness before opening a listener.

## Step 32 — Construct the initialized persistent runtime

Extracted the shared composition seam from the development constructor. Both
deployment modes now feed their chosen identity/auth services into the same
external HTTP host, host-owned object manager, bound dispatcher, generated
runtime, trusted route registration, and outer mux construction path.

Added `NewInitializedApplication`. Before allocating runtime resources it calls
`ValidateInitializedState`, so a missing manifest, database, token key, binding
key, or owner-only permission invariant refuses construction. It then opens:

- the migrated persistent tiny-idp SQLite store;
- the durable file audit sink;
- production password work, fixed-window rate limiting, and direct-peer address
  resolution;
- production secure IdP cookies and the initialized token secret;
- the in-process issuer transport;
- a shared SQLite application session/audit/user/capability store with schemas;
- the persistent object root and initialized binding key.

Initial retention maintenance runs synchronously before construction succeeds.
`Ready` requires every major component and delegates identity dependency checks
to `embeddedidp.Readiness`. Resource closure now includes the identity audit and
SQLite handles after runtime, object, auth, and provider shutdown.

Focused tests prove incomplete-state refusal, persistent application-auth DB
creation, object-root creation, production provider readiness, explicit route
availability, and clean shutdown. The first run exposed missing parent
directories for the application-auth database and lazily-created object root;
both are now created explicitly before the corresponding services start.

Validation:

```text
go test ./... -count=1                     PASS
go vet ./cmd/tinyidp-xapp/...              PASS
git diff --check                           PASS
```

Committed:

```text
a9b562e App: construct initialized persistent runtime
```

The listener is still deliberately absent. The next checkpoint adds TLS,
aggregate `/healthz` and `/readyz`, request-size bounds, periodic maintenance,
and cancellation-driven shutdown around this already-validated constructor.

## Step 33 — Add the initialized TLS serving lifecycle

Added the Glazed `serve-initialized` command. It validates duration and request
limits, calls the strict initialized constructor, requires aggregate readiness,
and only then invokes `ListenAndServeTLS`. Certificate and private-key paths are
required; this path does not silently downgrade to cleartext or trust forwarded
headers.

The outer production handler reserves native aggregate endpoints:

- `GET /healthz` reports process liveness without dependency detail;
- `GET /readyz` calls combined readiness and returns 503 when dependencies are
  degraded;
- all remaining paths are wrapped by `http.MaxBytesHandler` before entering the
  IdP/auth/Express mux.

The server config bounds header parsing, request reads, response writes,
keep-alive idleness, maximum header bytes, and graceful shutdown. An errgroup
owns the TLS server, periodic maintenance ticker, and cancellation-triggered
shutdown. Initial maintenance already runs in the constructor; subsequent
maintenance failures are logged and reflected by tiny-idp readiness.

Focused tests cover aggregate health/readiness on a fully persistent
application. The complete repository suite and product vet gate passed:

```text
go test ./... -count=1                     PASS
go vet ./cmd/tinyidp-xapp/...              PASS
```

Committed:

```text
568367b App: serve initialized runtime over TLS
```

This is the first production-shaped serve path. Remaining operational work
includes proxy-mode policy, backup/restore commands and drills, stronger
component-level readiness probes, browser E2E against a real TLS listener, and
release assurance/load/fault gates.

## Step 34 — Publish the initialized TLS checkpoint

Dry-ran and uploaded the refreshed five-document ticket bundle:

```text
TINYIDP-XAPP-001 Initialized TLS b1b22e3.pdf
/ai/2026/07/11/TINYIDP-XAPP-001
OK: uploaded
```

This version includes the persistent state contract, initialized constructor,
TLS lifecycle, current task state, and the complete debugging chronology.

## Step 35 — Park layered backup design and resume the real product

### Prompt Context

The user said:

> ok, create a ticket to keep track of that, so that we can build it later. then, let's do 2 - 3 - 4, which are actually running a real app

Here, “that” refers to the decision that tiny-idp should own a general,
component-level backup and restore contract while the composed XAPP owns
quiescence, manifesting, and whole-product recovery. Items 2, 3, and 4 refer to
the previously proposed real-browser end-to-end harness, identity/session
lifecycle closure, and injected failure testing.

### What I did and why

Created and committed `TINYIDP-BACKUP-001`, a separate docmgr ticket that
records the layered backup ownership model, archive contract, CLI surface,
consistency protocol, restore staging algorithm, phases, and stable tasks. This
keeps future recovery implementation precise without allowing it to displace
the immediate goal of exercising the running product.

The backup ticket passed `docmgr doctor --fail-on-warning` after replacing
unregistered vocabulary with the repository's controlled topics. It was
committed as:

```text
efd2c52 Docs: plan layered backup and restore
```

I then returned to this XAPP ticket and selected an evidence-first order for
the requested work:

1. initialize the persistent product state and run `serve-initialized` with a
   real TLS certificate in `tmux`;
2. drive the complete browser/OIDC/PKCE/session/object path with Chromium;
3. encode the observed lifecycle invariants, including two-user isolation,
   disablement, logout, and expiry;
4. add explicit failure seams and exercise dependency, persistence,
   maintenance, JavaScript, and shutdown failures against the composed app.

### What worked

The backup design is now independently searchable, reviewable, and deferred.
The existing XAPP task ledger already identifies the relevant incomplete
browser, two-user, disabled-user, and fault-injection work, so no speculative
parallel architecture is needed before running the product.

### What did not work

My first diary lookup used a remembered `diary/01-implementation-diary.md`
path. The actual ticket stores its chronology at
`reference/01-investigation-diary.md`. I recovered with `rg --files` rather
than making a second guessed lookup. No file was modified by the failed read.

### What I learned

The initialized TLS server, persistent stores, generated runtime, and minimal
frontend already form the correct test subject. The next useful evidence must
come from a browser crossing the external HTTP boundary, not another
constructor-only test.

### What was tricky

“Running a real app” must not collapse into an in-memory `httptest` success.
The harness needs a real listener, secure cookies, TLS, browser redirect and
form behavior, JavaScript execution, and persistent SQLite/object state. Unit
and in-process tests remain useful after this vertical observation, especially
for deterministic fault injection.

### How to review this step

Read the new backup ticket and verify its doctor result, then inspect the XAPP
commands and run the browser harness described by the next diary steps. Confirm
that the server is launched in `tmux`, its output is captured, and its port is
released with `lsof-who` after each run.

### Future work

Implement and preserve the real-browser harness, record each lifecycle gap it
reveals before fixing it, add focused regression tests, and then introduce
test-only failure controls at existing identity, app-auth, object, runtime, and
maintenance seams.

### Review instructions

Treat browser assertions as product-contract tests, not screenshot-only smoke
tests. A passing result must establish who authenticated, which cookies and
paths were used, which actor/object was reached, what survived reload/restart,
and what became inaccessible after logout, disablement, or expiry.

### Technical details

The backup ticket is located at
`ttmp/2026/07/12/TINYIDP-BACKUP-001--layered-tiny-idp-and-xapp-backup-restore`.
The runtime subject remains `cmd/tinyidp-xapp`, using its `init` and
`serve-initialized` Glazed commands. Browser tooling and captured results will
be stored under this ticket's `scripts/` and `reference/` directories so the
investigation can be replayed.

## Step 36 — Hand off the interrupted real-browser checkpoint

This step preserves the exact state of the real-application investigation so a
new engineer can continue without reconstructing the environment or mistaking
a browser-installation failure for a tiny-idp failure. The production-shaped
application successfully initialized, opened its persistent stores, served TLS,
and reported ready. Chromium never reached the application because the local
Playwright installation references browser artifacts that are absent.

The handoff is intentionally made before lifecycle or fault-injection code was
changed. The uncommitted browser script is an investigation harness, not yet a
passing product test. It expresses the intended assertions and should be
reviewed as draft code before being promoted into a release gate.

### Prompt Context

**User prompt (verbatim):** "write up your diary, we're going tohand it off"

**Assistant interpretation:** Record the complete executable-app checkpoint,
including failures and repository/runtime state, so another engineer can resume
immediately and safely.

**Inferred user intent:** Preserve engineering continuity and prevent the next
person from repeating environment discovery, losing security observations, or
assuming that phases 2–4 completed.

**Commit (documentation):** not yet committed at the time of this entry; the
last repository commit is `efd2c52` (`Docs: plan layered backup and restore`).

### What I did

- Verified Playwright availability under an explicit pyenv interpreter:

  ```text
  $ PYENV_VERSION=3.11.4 playwright --version
  Version 1.50.0
  $ PYENV_VERSION=3.11.4 python -c 'from playwright.sync_api import sync_playwright'
  python-playwright: import OK
  ```

- Created the isolated runtime root
  `/tmp/tinyidp-xapp-real-browser` with an owner-only operator directory,
  password files, one-day self-signed RSA certificate, and initialized product
  state. The certificate has `subjectAltName=IP:127.0.0.1`.
- Initialized the real product at public origin
  `https://127.0.0.1:19443` using:

  ```text
  go run ./cmd/tinyidp-xapp init \
    --state-root /tmp/tinyidp-xapp-real-browser/state \
    --public-base-url https://127.0.0.1:19443 \
    --login alice \
    --name 'Alice Operator' \
    --email alice@example.test \
    --password-file /tmp/tinyidp-xapp-real-browser/operator/alice-password
  ```

- Added a second identity to the same identity database through tiny-idp's real
  admin surface:

  ```text
  go run ./cmd/tinyidp admin \
    --db /tmp/tinyidp-xapp-real-browser/state/identity/tinyidp.sqlite \
    user create \
    --login bob \
    --name 'Bob Operator' \
    --email bob@example.test \
    --email-verified \
    --password-from-stdin \
    < /tmp/tinyidp-xapp-real-browser/operator/bob-password
  ```

- Launched the actual initialized TLS command in tmux session
  `tinyidp-xapp-e2e`:

  ```text
  go run ./cmd/tinyidp-xapp --log-level debug serve-initialized \
    --state-root /tmp/tinyidp-xapp-real-browser/state \
    --listen 127.0.0.1:19443 \
    --tls-cert /tmp/tinyidp-xapp-real-browser/operator/tls.crt \
    --tls-key /tmp/tinyidp-xapp-real-browser/operator/tls.key \
    --maintenance-interval 10s
  ```

- Captured the tmux log and queried the external listener. The final readiness
  response was:

  ```text
  HTTP/2 200
  content-type: application/json

  {"status":"ready"}
  ```

- Added the draft replayable harness at
  `scripts/01_real_browser_e2e.py`. It is designed to cover two independent
  browser contexts, the password/OIDC/PKCE flow, application and IdP cookies,
  missing-CSRF denial, per-user durable-object persistence, two-user isolation,
  logout, and post-logout session state.
- After the two permitted browser-launch attempts failed, stopped the tmux
  session, killed any listener through `lsof-who -p 19443 -k`, and verified
  that port 19443 was no longer reachable.
- Audited the handoff state: no `tinyidp-xapp-e2e` tmux session exists, no
  process listens on port 19443, and the `/tmp` fixture remains available.

### Why

- A real browser and TLS listener observe redirects, Secure/HttpOnly/SameSite
  cookie behavior, form interactions, JavaScript, route composition, and
  persistent storage boundaries that constructor or `httptest` tests cannot.
- Two users in one initialized product are necessary to establish subject-bound
  object isolation; two separate installations would not exercise the shared
  application database and object manager.
- The server was kept external to the browser script so startup, readiness,
  logs, maintenance, shutdown, and port ownership remain independently
  observable.

### What worked

- `tinyidp-xapp init` reconciled a complete persistent state root.
- The general tiny-idp admin command opened that identity database and created
  Bob without a separate schema or compatibility path.
- `serve-initialized` opened the composed identity, application-auth, generated
  Goja runtime, and Durable Object application over real TLS.
- Aggregate readiness returned HTTP 200 from the actual TCP listener.
- The machine has usable browser launchers on PATH:
  `/usr/bin/google-chrome`, `/snap/bin/chromium`, and
  `/usr/bin/chromium-browser`. These were discovered during handoff inspection
  but deliberately not tried after the two-failure stop threshold.
- Cleanup completed. The message was:

  ```text
  tinyidp-xapp-e2e stopped; port 19443 released
  ```

### What didn't work

- Invoking `playwright` under the default pyenv selection failed before the
  real run because the command is installed only in Python 3.11.3/3.11.4:

  ```text
  pyenv: playwright: command not found

  The `playwright' command exists in these Python versions:
    3.11.3
    3.11.4
  ```

  Pinning `PYENV_VERSION=3.11.4` resolved this discovery issue.
- Browser launch attempt one used Playwright's default executable selection:

  ```text
  PYENV_VERSION=3.11.4 python \
    ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/scripts/01_real_browser_e2e.py \
    --base-url https://127.0.0.1:19443 \
    --alice-password-file /tmp/tinyidp-xapp-real-browser/operator/alice-password \
    --bob-password-file /tmp/tinyidp-xapp-real-browser/operator/bob-password
  ```

  It failed verbatim with:

  ```text
  BrowserType.launch: Executable doesn't exist at /home/manuel/.cache/ms-playwright/chromium_headless_shell-1155/chrome-linux/headless_shell
  Looks like Playwright was just installed or updated.
  Please run the following command to download new browsers:
      playwright install
  ```

- Browser launch attempt two set
  `executable_path=playwright.chromium.executable_path`. Playwright had reported
  that location during metadata inspection, but the file is absent. It failed
  verbatim with:

  ```text
  BrowserType.launch: Failed to launch chromium because executable doesn't exist at /home/manuel/.cache/ms-playwright/chromium-1155/chrome-linux/chrome
  ```

- No browser page was opened, so none of the harness's identity, cookie, CSRF,
  persistence, isolation, or logout assertions have executed. There is no
  results JSON under `reference/results/` yet.
- A later read-only executable audit used `path` as a zsh loop variable. In zsh,
  `path` is tied to `PATH`, so the loop accidentally made `ls` unavailable and
  printed `zsh:4: command not found: ls`. The `command -v` results and explicit
  missing Playwright paths remain valid; future shell snippets must not use
  `path` as a zsh variable name.

### What I learned

- The running application passed its first external production-shaped startup
  checkpoint. The current blocker is local browser artifact selection, not
  application initialization, TLS binding, or aggregate readiness.
- Playwright package installation and Playwright browser installation are
  distinct. `playwright --version` and Python import success do not establish
  that the version-matched browser executable exists.
- A system Chrome is available at `/usr/bin/google-chrome`; the smallest next
  experiment is to point `chromium.launch(executable_path=...)` there. An
  alternative is `PYENV_VERSION=3.11.4 playwright install chromium`, but that
  performs a network download and changes the workstation cache.
- The app logout implementation in go-go-goja currently accepts POST without a
  CSRF check and also exposes GET logout. The frontend renders a plain POST
  form, while object mutations use `X-CSRF-Token`. This is an observed review
  target, not yet a fixed or browser-proven vulnerability in this checkpoint.
- The preserved state root has mode `0700`, while
  `state/application/auth.sqlite` and `state/objects/alarms.sqlite` are mode
  `0644`. Directory traversal confines access today, but this conflicts with
  earlier owner-only-store wording and could become unsafe if files are moved,
  copied, or backed up incorrectly. The intended per-file permission contract
  must be decided and tested.

### What was tricky to build

- The real flow spans two session domains: `xapp_session` for the application
  and issuer-scoped `xapp_idp_session`/`xapp_idp_csrf` cookies for tiny-idp.
  Logging out of the app may leave the IdP session valid, so a later `/auth/login`
  can silently create a new app session. The harness must distinguish correct
  two-layer behavior from a logout failure.
- The browser script intentionally treats Alice's and Bob's application user
  identifiers as opaque. Isolation must be established by different normalized
  user IDs and different private object values, not by assuming an ID format.
- Async frontend writes and reloads need response- or state-based waits. The
  draft currently uses one fixed 100 ms wait in `loaded_document`; this may be
  flaky and should be replaced with a response predicate or observable value
  transition before calling the harness stable.
- Playwright API request calls should be reviewed to ensure they use the
  browser context's cookie-bearing request context. If this Playwright version
  does not expose `page.request`, use `page.context.request` or execute `fetch`
  in the page; do not create an unrelated API request context that bypasses the
  browser cookie jar.

### What warrants a second pair of eyes

- Decide whether POST `/auth/logout` must enforce the current app-session CSRF
  token and whether GET `/auth/logout` should exist at all. Review
  `pkg/gojahttp/auth/oidcauth/oidcauth.go` and
  `pkg/xgoja/hostauth/builder.go` in go-go-goja before changing the frontend.
- Review the exact two-layer logout contract: app logout only, IdP logout only,
  and product-wide logout should be distinct and visible to the user.
- Confirm whether disabling a tiny-idp user must immediately invalidate an
  already-issued XAPP session. Currently the app session is a locally persisted
  projection; upstream disablement may only be observed at the next OIDC login.
- Review permissions for all SQLite databases, WAL/SHM sidecars, audit logs,
  secrets, certificate/key material, and future backup archives.
- Review `scripts/01_real_browser_e2e.py` before trusting it as a gate,
  especially cookie sharing for API requests, navigation waits, response waits,
  cleanup on assertion failure, and whether logout-without-CSRF should be an
  expected failure rather than merely an observation.
- Preserve the unrelated untracked OIDF source directories under
  `TINYIDP-PROD-001`; they predate this work and must not be staged or deleted.

### What should be done in the future

1. Change only the browser executable selection to
   `/usr/bin/google-chrome`, restart the same real server in tmux, and rerun the
   harness. If policy prefers a hermetic Playwright browser, install the exact
   1.50 Chromium bundle instead; do not do both before learning which contract
   the project wants.
2. Fix any harness API/wait issues exposed after the browser actually launches,
   preserving each exact error in the next diary step.
3. Save the passing structured JSON under `reference/results/`, capture server
   logs, and repeat after a full process restart to prove persistent object and
   session behavior.
4. Turn observed lifecycle behavior into explicit product contracts and
   regression tests: CSRF-safe logout, app-versus-IdP logout, expiry, forced
   reauthentication, password-change-required, and user disablement.
5. Add deterministic failure seams and scenarios only after the successful
   browser baseline: IdP/app/object SQLite failures, maintenance failure, Goja
   timeout, readiness degradation, and cancellation/shutdown races.
6. Decide and enforce the file-mode contract before backup work consumes these
   files.
7. Remove `/tmp/tinyidp-xapp-real-browser` after the handoff engineer either
   completes the replay or deliberately creates a fresh fixture. Its passwords
   are throwaway but known and must never be reused outside this local test.

### Code review instructions

- Start with this diary step, then review:
  - `scripts/01_real_browser_e2e.py` for intended browser assertions;
  - `cmd/tinyidp-xapp/production_app.go` for persistent composition and cookie
    names;
  - `cmd/tinyidp-xapp/serve_initialized.go` for the TLS lifecycle;
  - `cmd/tinyidp-xapp/app/frontend/public/app.js` and `index.html` for CSRF and
    logout behavior;
  - go-go-goja's `pkg/gojahttp/auth/oidcauth/oidcauth.go` and
    `pkg/xgoja/hostauth/builder.go` for native session endpoints.
- Before restart, run:

  ```text
  lsof-who -p 19443 -k
  tmux kill-session -t tinyidp-xapp-e2e  # ignore absent-session error
  ```

- Launch the server with the exact command under `What I did`, then inspect:

  ```text
  tmux capture-pane -p -t tinyidp-xapp-e2e -S -200
  curl -ksi https://127.0.0.1:19443/readyz
  ```

- For the next controlled browser attempt, edit the draft launch to use:

  ```python
  browser = playwright.chromium.launch(
      executable_path="/usr/bin/google-chrome",
      headless=not args.headed,
  )
  ```

- Do not mark the Phase 3 two-user task or Phase 5 browser task complete until
  the browser reaches the app and emits passing structured evidence.
- After any implementation changes, run focused tests, `go test ./... -count=1`,
  `go vet ./cmd/tinyidp-xapp/...`, `git diff --check`, and docmgr doctor.

### Technical details

- Repository: `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp`
- Ticket: `TINYIDP-XAPP-001`
- Ticket directory:
  `ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application`
- Draft harness:
  `scripts/01_real_browser_e2e.py`
- Runtime fixture: `/tmp/tinyidp-xapp-real-browser`
- Public origin: `https://127.0.0.1:19443`
- tmux session when running: `tinyidp-xapp-e2e`
- Playwright Python: pyenv `3.11.4`, Playwright `1.50.0`
- Absent Playwright executables:
  - `/home/manuel/.cache/ms-playwright/chromium_headless_shell-1155/chrome-linux/headless_shell`
  - `/home/manuel/.cache/ms-playwright/chromium-1155/chrome-linux/chrome`
- Candidate installed executable: `/usr/bin/google-chrome`
- Current Git state relevant to this checkpoint:
  - modified: `reference/01-investigation-diary.md`;
  - untracked: `scripts/01_real_browser_e2e.py`;
  - unrelated and untracked: two OIDF source directories under
    `TINYIDP-PROD-001/sources/`.
- No implementation source file was changed in this checkpoint.

## Step 37 — Give the incoming intern a bounded first contribution

The prior diary step is a complete chronological handoff, but chronology alone
does not tell a new contributor where to invest attention. I created a separate
intern playbook that reorganizes the same evidence around system ownership,
security boundaries, executable procedures, review points, and a sequence of
small contributions. The goal is to let the intern establish competence by
producing real browser evidence before they are asked to change architecture.

The playbook explicitly distinguishes facts already proven by the initialized
TLS checkpoint from assertions that exist only in the draft browser harness. It
also identifies decisions that require review, especially logout CSRF,
disablement propagation, actor identity, proxy trust, and file permissions.

### Prompt Context

**User prompt (verbatim):** "make a document in the ticket so we can hand this over to a new intern that needs to find their mark, and then continue where you left off"

**Assistant interpretation:** Create a self-contained intern onboarding and
continuation document inside `TINYIDP-XAPP-001`, then resume the real TLS/browser
work from the exact checkpoint recorded in Step 36.

**Inferred user intent:** Let a new engineer become useful quickly, make a
reviewable first security contribution, and preserve forward momentum rather
than ending at documentation.

**Commit (documentation):** pending at the time this step was written.

### What I did

- Added the docmgr playbook `New Intern Handoff and Continuation Playbook`.
- Re-read the ticket design, task ledger, recent diary, production constructor,
  TLS server, shared composition code, generated runtime specification,
  JavaScript routes, object bundle, frontend, and draft browser harness.
- Wrote an ownership map for tiny-idp, go-go-goja, and go-go-objects.
- Documented the route tree, startup sequence, two-session model,
  issuer/subject identity projection, actor-bound object derivation, and state
  layout.
- Preserved exact real-server evidence and separated it from browser assertions
  that have not run.
- Defined a first-day command sequence using tmux, `serve-initialized`, curl,
  Python 3.11.4, system Chrome, cleanup, and validation.
- Defined scenario contracts for logout CSRF, disabled users, forced
  reauthentication, password-change-required, expiry, and two-user isolation.
- Defined the later fault matrix and the exit criteria for browser/lifecycle and
  fault-injection phases.
- Related the playbook to seven material implementation and harness files using
  absolute docmgr file notes.

### Why

- A chronological diary is optimized for reconstructing decisions; an intern
  playbook must instead optimize reading order, repository navigation, first
  commands, and review boundaries.
- The next engineer needs a meaningful first contribution that does not require
  broad architectural authority. Completing the real-browser baseline and
  freezing one lifecycle invariant satisfies that requirement.
- Explicitly separating observation, desired behavior, and open decisions
  prevents a test harness from accidentally defining product policy.

### What worked

- docmgr created the playbook under the existing ticket with the controlled
  `playbook` document type and ticket vocabulary.
- All referenced paths were verified against the current repositories before
  being included.
- The handoff now provides both forms of continuity: Step 36 preserves exact
  history, while the playbook provides a task-oriented entry point.

### What didn't work

- An exploratory read initially guessed `cmd/tinyidp-xapp/application.go` and
  `app/routes/main.js`; neither exists. `rg --files` established the actual
  ownership in `development_app.go` and `app/routes/site.js`. No modification
  depended on the guessed paths.

### What I learned

- The current product is compact enough that an intern can trace the complete
  request path without treating generated code as the primary source. The
  decisive files are the state reconciler, two composition files, TLS command,
  xgoja specification, site routes, object bundle, and generic hostauth OIDC
  handler.
- The best first mark is not new feature volume. It is a passing, sanitized,
  reproducible proof across the real browser boundary, followed by one explicit
  lifecycle invariant.

### What was tricky to build

- The playbook must be actionable without freezing unresolved policy. Logout
  CSRF and disablement propagation are therefore written as scenario questions
  and review gates rather than instructions to implement a predetermined
  mechanism.
- Repository ownership crosses three modules. The document distinguishes
  product presentation in tiny-idp-xapp from generic OIDC/session semantics in
  go-go-goja and actor/object semantics in go-go-objects, reducing the chance of
  a product-only patch to a shared contract.
- The preserved `/tmp` fixture is useful but contains known throwaway
  credentials. The playbook permits one local replay, forbids reuse, and
  requires eventual deletion without copying secret values into documentation.

### What warrants a second pair of eyes

- Review whether the recommended first invariant should be logout CSRF or
  disabled-user propagation after the baseline passes.
- Confirm that system Chrome is acceptable for local evidence or whether the
  final CI gate must pin and install a Playwright-managed browser.
- Review the stated per-file SQLite permission concern and the eventual
  WAL/SHM/backup contract.
- Confirm the playbook's contribution boundaries before assigning the intern
  cross-repository write access.

### What should be done in the future

- Keep the playbook current when commands, route ownership, session semantics,
  or the next incomplete phase changes.
- Replace the handoff's “browser not yet proven” wording after preserving a
  passing structured result.
- Add links to focused lifecycle decision documents when logout and
  disablement contracts are approved.

### Code review instructions

- Read the playbook from Purpose through First-Day Continuation Procedure, then
  verify each path and command against the related files.
- Run:

  ```text
  docmgr doctor --ticket TINYIDP-XAPP-001 --stale-after 30 --fail-on warning
  git diff --check
  ```

- Confirm the new document contains no password, cookie value, token,
  authorization code, subject, or binding-key material.

### Technical details

- Document:
  `playbook/01-new-intern-handoff-and-continuation-playbook.md`
- Next runtime experiment: select `/usr/bin/google-chrome` in
  `scripts/01_real_browser_e2e.py`, start `serve-initialized` in tmux, and run
  the harness against `https://127.0.0.1:19443`.
- The documentation work does not complete browser task `j5ba`, isolation task
  `ihzp`, or fault task `uvhr`.

## Step 38 — Complete the first real-Chromium product baseline

The resumed browser run crossed the boundary that the previous checkpoint did
not: system Chrome completed the rendered tiny-idp login, OIDC authorization
code flow, application callback, frontend bootstrap, CSRF-protected object
writes, persistent reloads, and two independent users' private objects against
the actual TLS listener. No application behavior was changed to obtain this
result; only Playwright's executable path changed.

The successful run also produced two evidence-backed lifecycle findings. The
generic OIDC logout endpoint revoked an application session on a POST without
CSRF, and the Express host emitted an unused non-Secure
`go_go_goja_session` cookie in addition to the intended secure application
session. These observations now justify focused implementation work.

### Prompt Context

**User prompt (verbatim):** (same as Step 37)

**Assistant interpretation:** Continue from the stopped browser checkpoint and
use the running product to determine the next concrete hardening work.

**Inferred user intent:** Replace design-only confidence with real application
evidence, then robustify security behavior revealed by that evidence.

**Commit (code):** pending at the time this step was written.

### What I did

- Changed the draft harness's Chromium executable from the absent
  Playwright-managed path to `/usr/bin/google-chrome`.
- Verified the preserved initialized fixture, certificate, and two password
  files still existed.
- Killed any prior listener, removed any stale tmux session, and started the
  real `serve-initialized` command in `tinyidp-xapp-e2e`.
- Captured the server startup log and confirmed `/readyz` returned HTTP 200.
- Ran the Python harness under `PYENV_VERSION=3.11.4` and Playwright 1.50.0.
- Observed a complete passing harness result for its current assertions.
- Traced the unexpected cookie to `pkg/gojahttp/session.go`: the Express host
  creates a lightweight opaque request session by default for every planned
  route, independently from hostauth's `xapp_session`.
- Traced logout behavior to `pkg/gojahttp/auth/oidcauth/oidcauth.go` and native
  route construction in `pkg/xgoja/hostauth/builder.go`.
- Stored a sanitized pre-hardening result at
  `reference/results/01-real-browser-baseline-before-lifecycle-hardening.json`.

### Why

- Selecting system Chrome was the single untried variable identified by Step
  36. Keeping product code unchanged made the successful result interpretable.
- Pre-fix evidence is necessary to show that later lifecycle tests detect real
  behavior changes rather than merely encoding assumptions.
- The structured result excludes secrets and stable identity material while
  preserving statuses, cookie attributes, and equality/isolation facts needed
  for review.

### What worked

- Alice and Bob normalized to distinct application users.
- Alice's and Bob's JSON documents persisted and remained isolated.
- A POST to `/api/object` without the application CSRF token returned 403.
- The intended production cookies were Secure and HttpOnly with Lax SameSite:
  `xapp_session` at `/`, and `xapp_idp_session` plus `xapp_idp_csrf` at
  `/idp`.
- POST `/auth/logout` revoked Alice's app session; `/auth/session` then returned
  401.
- The actual server remained ready during the browser flow.

### What didn't work

- Logout without a CSRF header returned 204 instead of failing closed.
- GET logout remains registered as a state-changing native endpoint.
- The browser received `go_go_goja_session` with `Secure=false`, path `/`,
  HttpOnly, and SameSite=Lax. The XAPP does not use this separate lightweight
  session for authentication or object identity.
- The first readiness poll immediately after tmux launch printed
  `curl: (7) Failed to connect`; the bounded loop succeeded on the next poll.
  This is the expected process startup race, not a readiness failure.
- The harness output still included opaque application user IDs on stdout even
  though the persisted result removed them. The harness must stop emitting
  those IDs before its output is captured directly as a release artifact.

### What I learned

- The composed product is usable for its central v1 story: two real users can
  authenticate and operate isolated persistent objects through the embedded
  frontend.
- Hostauth application sessions and gojahttp lightweight request sessions are
  separate mechanisms. Leaving both enabled created a redundant cookie with a
  weaker transport attribute.
- Logout is a generic hostauth lifecycle contract, so the fix belongs primarily
  in go-go-goja and must be consumed by the XAPP frontend, not patched only at
  the product mux.

### What was tricky to build

- Browser cookie inspection must preserve attributes without values. The
  current result stores only names, paths, booleans, and SameSite modes.
- Two-user isolation evidence must prove distinctness without publishing raw
  subjects or object binding IDs. The sanitized artifact records boolean
  distinctness and per-user document outcomes.
- The unexpected cookie was not the insecure form of `xapp_session`; it was a
  second session mechanism. Fixing the Secure bit alone would retain needless
  state and obscure which session is authoritative. The product should disable
  the unused lightweight session explicitly.

### What warrants a second pair of eyes

- Confirm that no trusted XAPP route consumes `ctx.request.session` before
  disabling gojahttp lightweight sessions.
- Review removal of GET logout as an intentional breaking security correction,
  not a compatibility concern.
- Review whether logout revocation errors should return 500 without clearing
  the cookie, clear the browser cookie while reporting failure, or trigger a
  stronger audit/incident response. The existing code ignores the error.
- Confirm the frontend's logout transition after switching from a native form
  POST to a CSRF-bearing fetch.

### What should be done in the future

- Disable the unused gojahttp session in XAPP composition and assert no
  `go_go_goja_session` cookie is emitted.
- Require current application-session CSRF on POST logout, remove GET logout,
  handle revocation failure explicitly, and update the frontend.
- Sanitize harness stdout, replace fixed waits, and persist the post-fix result.
- Repeat after process restart before completing browser task `j5ba`.

### Code review instructions

- Reproduce with the exact server and browser commands in the intern playbook.
- Inspect the sanitized JSON artifact; verify it contains no values for cookies,
  passwords, codes, tokens, subjects, user IDs, or binding IDs.
- Review `gojahttp.NewHost` construction in `composeApplication`, then read
  `gojahttp/session.go`, `oidcauth.handleLogout`, and hostauth native route
  registration.

### Technical details

- Browser: system Google Chrome, headless user-agent major 138.
- Python: pyenv 3.11.4, Playwright 1.50.0.
- Origin: `https://127.0.0.1:19443`.
- Baseline statuses: readiness 200, missing-CSRF object POST 403,
  missing-CSRF logout POST 204, post-logout session 401.
- The server remains in tmux session `tinyidp-xapp-e2e` while lifecycle changes
  are prepared; it must be restarted before validating compiled changes.

## Step 39 — Commit the browser lifecycle hardening after misleading hook output

The lifecycle hardening is now committed in both affected repositories. The
go-go-goja pre-commit hook caused a misleading checkpoint: its large parallel
lint/test output ended before the Git success line was displayed, so the turn
reported that the commit had not been created. A later direct status check
showed that the second commit attempt had in fact completed as `73fd871`.

The remaining tiny-idp commit was intentionally made with `LEFTHOOK=0` after
the user authorized skipping the redundant hook. This did not skip validation:
the focused tests, generated-code check, complete tiny-idp suite, complete
go-go-goja suite, vet, lint, browser baseline, and restart browser run had all
already passed before the commit.

### Prompt Context

**User prompt (verbatim):** "why did it not create the commit? just LEFTHOOK=0 I guess to skip the checks"

**Assistant interpretation:** Determine whether the hook actually prevented the
commit, use the supported lefthook bypass for the already-validated remaining
checkpoint, and preserve the explanation.

**Inferred user intent:** Avoid repeating expensive successful checks merely
because hook output handling was confusing, while still ending with durable,
reviewable commits.

**Commit (code):** `73fd871` — "HTTP: require CSRF for OIDC logout"

**Commit (product and evidence):** `99505d1` — "App: harden browser session lifecycle"

### What I did

- Inspected `go-go-goja/lefthook.yml`. Its pre-commit hook runs `make lint` and
  `make test` in parallel for staged Go changes.
- Checked go-go-goja status and discovered that the staged set was clean and
  HEAD was already `73fd871`.
- Verified the commit contains the intended four generic OIDC/hostauth files.
- Revalidated the tiny-idp browser harness syntax and all three structured JSON
  evidence files.
- Searched the harness and results for known password phrases, raw Set-Cookie
  headers, bearer headers, and the old raw user-ID output shape.
- Staged only XAPP source, generated assets, focused test, harness, and three
  sanitized results. The unrelated OIDF source directories and the diary were
  excluded.
- Committed the already-validated tiny-idp staged set with:

  ```text
  LEFTHOOK=0 git commit -m "App: harden browser session lifecycle"
  ```

### Why

- Repeating the same repository-wide hook after both full suites and lint had
  passed would add time without new evidence.
- Separating the generic hostauth commit from the product integration commit
  preserves ownership: go-go-goja defines logout semantics; tiny-idp-xapp
  consumes them, disables unused request sessions, presents UI state, and
  stores browser evidence.

### What worked

- go-go-goja commit `73fd871` exists and contains 80 insertions and 8 deletions
  across the OIDC handler and hostauth builder with focused tests.
- tiny-idp commit `99505d1` was created immediately with hooks disabled and
  contains the XAPP behavior, generated assets, test, harness, and sanitized
  before/after/restart evidence.
- No unrelated untracked source directory was staged.

### What didn't work

- The first go-go-goja commit attempt ran the lefthook lint stream but left the
  changes staged. The captured output did not include a final error or exit
  explanation, so the precise first-attempt cause is unknown.
- The second attempt did create the commit, but the tool capture again ended at
  the hook output and omitted Git's final commit line. The prior response
  therefore incorrectly treated the second attempt as another failure.
- A verification command run from tiny-idp attempted `git show 73fd871`; that
  hash belongs to the sibling go-go-goja repository and was correctly reported
  as unknown there. Using `git -C ../go-go-goja show 73fd871` verified it.

### What I learned

- Hook process output is not authoritative evidence of Git state when commands
  are long and parallel. Always check `git status`, `git log -1`, and
  `git rev-parse HEAD` after the hook process exits.
- `LEFTHOOK=0` is appropriate when the same indexed content has already passed
  the hook-equivalent checks and the bypass is explicit. It must not be used to
  conceal a failing or unrun validation gate.

### What was tricky to build

- Three state machines overlapped: the tool cell, lefthook's parallel child
  commands, and Git's commit transaction. A completed/truncated output cell did
  not clearly show whether Git had advanced HEAD. Repository state, rather than
  the last visible log line, resolved the ambiguity.
- The product commit includes generated embedded assets. Both their source
  files and generated copies had to be staged together after `go generate` so
  the committed binary behavior matches the reviewed frontend source.

### What warrants a second pair of eyes

- Review `73fd871` for the policy choice that GET logout is removed, POST logout
  requires the current session's CSRF token, and revocation failure returns 500
  without clearing the cookie.
- Review `99505d1` for the choice to disable the unused gojahttp lightweight
  session completely rather than merely set its Secure bit.
- Confirm the frontend's retained-IdP-session explanation is sufficient product
  language and accessibility behavior.
- Consider configuring lefthook or the execution wrapper to retain the final
  exit status and commit tail for long parallel hooks.

### What should be done in the future

- Characterize disabled-user behavior with an already-active XAPP session.
- Add expiry and forced-reauthentication browser scenarios before completing
  browser task `j5ba`.
- Begin deterministic IdP/app/object persistence and maintenance failure
  scenarios only after recording the disabled-user contract.

### Code review instructions

- In go-go-goja:

  ```text
  git show 73fd871
  go test ./pkg/gojahttp/auth/oidcauth ./pkg/xgoja/hostauth -count=1
  ```

- In tiny-idp:

  ```text
  git show 99505d1
  go test ./cmd/tinyidp-xapp/... -count=1
  ```

- Re-run the browser harness with `--expect-existing` against the initialized
  TLS tmux server and compare the output to
  `reference/results/03-real-browser-after-process-restart.json`.

### Technical details

- Full suites passed before both commits:
  - `go-go-goja: go test ./... -count=1`;
  - `tiny-idp: go test ./... -count=1`.
- Focused vet passed in both affected package groups.
- The go-go-goja hook also reported golangci-lint `0 issues`.
- The XAPP server was stopped and port 19443 released before committing.
