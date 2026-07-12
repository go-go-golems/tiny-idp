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
RelatedFiles: []
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
