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
