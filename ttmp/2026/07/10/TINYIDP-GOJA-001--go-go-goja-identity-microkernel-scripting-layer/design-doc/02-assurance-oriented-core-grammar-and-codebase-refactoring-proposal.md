---
Title: Assurance-Oriented Core Grammar and Codebase Refactoring Proposal
Ticket: TINYIDP-GOJA-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Current native authorization control flow and artifact issuance boundary
    - Path: repo://internal/securitytrace/trace.go
      Note: Existing runtime event vocabulary and parametric monitor
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Existing one-time and named atomic transition authority contracts
    - Path: repo://pkg/idpstore/types.go
      Note: Existing interaction resources, obligations, outcomes, sessions, and protocol state
    - Path: repo://pkg/verifyplan/plan.go
      Note: Existing data-only scenario grammar to evolve toward typed step codecs
    - Path: repo://ttmp/2026/07/11/TINYIDP-MODEL-001--serious-model-checking-and-formal-state-assurance-for-tiny-idp/design-doc/02-serious-model-checking-system-architecture-and-implementation-plan.md
      Note: Formal-model evidence architecture synthesized by this proposal
    - Path: repo://ttmp/2026/07/11/TINYIDP-STATIC-001--static-analysis-and-implementation-verification-for-tiny-idp/design-doc/01-static-analysis-and-implementation-verification-research-design-and-implementation-guide.md
      Note: Static authority, effect, abstract-domain, and mutation architecture synthesized by this proposal
ExternalSources: []
Summary: Synthesis design for reorganizing tiny-idp around stable resources, facts, obligations, native transitions, configuration graphs, scenarios, and traces that improve scriptability, static analysis, model checking, and runtime verification without replacing native OAuth security semantics with a generic workflow engine.
LastUpdated: 2026-07-12T01:55:00Z
WhatFor: Evaluates the present architecture and proposes a staged assurance-oriented intermediate representation shared by Go, Goja compilation, analyzers, formal models, scenario drivers, and runtime traces.
WhenToUse: Read before refactoring provider control flow, implementing the identity graph, introducing scriptable steps, consolidating verification vocabularies, or generating model/static-analysis artifacts.
---


# Assurance-Oriented Core Grammar and Codebase Refactoring Proposal

## Executive summary

tiny-idp is already in a strong intermediate state. The current code has a
server-owned `InteractionRecord`, typed required actions and terminal outcomes,
named atomic store operations, deterministic clocks, structured security events,
a parametric runtime monitor, a Rapid reference state machine, Porcupine
linearizability checks, typed verification plans, an isolated Goja plan compiler,
and a strict-provider scenario driver. These are exactly the artifacts from which
an assurance-oriented architecture can be built.

The problem is not an absence of state-machine concepts. The problem is that the
same concepts are represented several times:

- interaction actions are private integers in the Rapid test;
- scenario steps are open strings plus raw JSON;
- security events use a separate set of string constants;
- the proposed Goja identity graph defines another node and effect vocabulary;
- future TLA+ models need action, resource, and property identifiers;
- static analyzers need their own transition-authority and effect catalog; and
- provider control flow remains encoded directly in HTTP handler branches.

The recommended refactor is not to convert the OAuth provider into a general
scripted workflow engine. That would move trusted protocol semantics into a more
dynamic execution layer, enlarge the trusted computing base, complicate Fosite
integration, and make static analysis less precise. Instead, introduce a small,
data-only assurance grammar with stable identifiers and three related schemas:

1. **Configuration graph.** Describes which native blocks, policies, resources,
   and flows are selected for an activated identity realm. Goja may compile into
   this graph.
2. **Native transition catalog.** Describes the fixed Go-owned steps, resource
   kinds, required facts, produced facts, obligations, effects, atomicity class,
   and emitted observations. This is metadata for analysis and instrumentation;
   it is not initially an interpreter.
3. **Scenario and trace records.** Describe requested test actions and observed
   runtime transitions using the same stable step, resource, fact, and property
   identifiers.

```text
JavaScript policy/configuration
        |
        v
configuration graph -------> validator/materializer -------> native Go plan
                                    |
native transition catalog ---------+
        |               |           |
        v               v           v
static analyzers   model exporters  automatic trace instrumentation
        |               |           |
        +---------------+-----------+
                        v
              scenarios and evidence
```

The initial catalog should describe existing code rather than drive it. Once the
catalog, trace, model, and analyzer mappings are stable, selected authorization
logic can move into a pure typed transition kernel. This staged approach obtains
most analysis benefits before risking a rewrite of Fosite orchestration.

The first vertical slice should be authorization interaction. It already has the
best state representation and the richest assurance evidence. The slice should
make `begin`, `authenticate`, `approve_consent`, `deny`, `expire`, and
`commit_approval` share stable identities across Go, VerificationPlan, security
events, model specifications, and static analyzer rules. It should preserve the
existing HTTP/Fosite/store behavior and demonstrate that the new grammar detects
the historical forced-reauthentication and password-change defects.

## 1. The design question

The central question is not whether tiny-idp can be expressed as steps and
resources. Any program can be described that way after the fact. The useful
question is whether a grammar can expose exactly the security-relevant structure
without hiding essential semantics behind generic callbacks or turning protocol
code into dynamically typed configuration.

The grammar must satisfy four consumers with different needs:

- Go execution needs typed values, explicit errors, contexts, transactions, and
  Fosite integration.
- Goja configuration needs serializable graph nodes and narrow callbacks without
  raw stores, tokens, keys, or provider objects.
- static analysis needs declared transition authorities, effects, trust classes,
  obligations, and sinks that map to actual Go symbols;
- model checking and runtime verification need finite state, stable action IDs,
  explicit nondeterminism, atomic boundaries, and secret-free observations.

A universal `Step.Execute(map[string]any)` interface would superficially serve
all four. It would also discard the type and effect information the assurance
program needs. The proposal therefore treats the grammar as a typed schema and
catalog, not as an untyped plugin protocol.

## 2. Current-state assessment

### 2.1 What is already structurally strong

#### Server-owned interaction continuation

`pkg/idpstore/types.go:175-224` defines `InteractionRequiredAction`,
`InteractionOutcome`, and `InteractionRecord`. The record persists canonical
request data, request digest, client and redirect binding, required actions,
browser/session binding, configuration generation, time bounds, and terminal
state. `pkg/idpstore/interfaces.go:83-89` exposes create, get, and atomic consume.

This is already a domain state machine. It is not merely an HTTP form record.
The proposed grammar should describe and reuse it rather than replace it with a
generic graph token.

#### Named atomic persistence operations

`pkg/idpstore/interfaces.go:154-166` exposes transaction callbacks and named
operations such as `RecordSuccessfulLogin`, `RevokeUserSecurityArtifacts`, and
`RotateSigningKey`. These methods encode business-level atomicity better than a
script-visible database transaction would.

The grammar should identify these operations as native effects with atomicity
classes. Scripts may request a policy decision that eventually selects an
effect; scripts must not invoke transaction primitives directly.

#### Verification plans as data

`pkg/verifyplan/plan.go:13-158` defines bounded, immutable plans, steps,
assertions, observations, a driver, and a runner. `internal/gojaverify/compiler.go`
compiles JavaScript into the data-only plan inside an isolated runtime.
`internal/gojamodules/verify/module.go` exposes only a plan constructor and no
live provider capabilities.

This is a successful instance of the proposed pattern: JavaScript authors data;
Go validates and executes it. The weakness is that `Step.Kind`,
`Observation.Kind`, assertion IDs, and parameter JSON remain open strings without
a shared catalog.

#### Runtime security events and monitor

`internal/securitytrace/trace.go` defines versioned secret-free events and a
per-interaction monitor. The monitor checks creation order, required
authentication and consent, one terminal outcome, and artifact ordering.

The event vocabulary overlaps the desired transition grammar, but required
action bits are duplicated locally at `trace.go:68-72`. This is evidence that the
package boundary prevents a shared domain definition or that the definition has
not yet been factored into a dependency-neutral package.

#### Executable state models

`internal/fositeadapter/state_model_test.go` defines private model actions and a
reference transition function. It supports Rapid generation, minimized replay,
and native fuzzing. `linearizability_test.go` supplies a Porcupine sequential
model for one-time consumption.

These tests prove the value of a compact action grammar. Their private action
types, however, cannot be reused by VerificationPlan, trace monitoring, formal
models, or analyzer metadata.

### 2.2 Where the architecture resists analysis

#### Issue 1: security vocabulary is duplicated

**Problem:** Required actions, actions, event kinds, scenario steps, assertions,
and model operations use independent identifiers and representations.

**Where to look:**

- `pkg/idpstore/types.go:175-224`;
- `internal/securitytrace/trace.go:13-145`;
- `pkg/verifyplan/plan.go:28-44`;
- `internal/fositeadapter/verification_scenario_test.go:61-137`;
- `internal/fositeadapter/state_model_test.go:14-73`.

**Example:**

```go
type Step struct {
    Kind       string
    Parameters json.RawMessage
}

type Event struct {
    Kind            securitytrace.Kind
    RequiredActions uint32
}
```

**Why it matters:** A model counterexample cannot be replayed automatically
without an adapter. Static rules cannot reference the same action IDs as runtime
events. Vocabulary drift can make one evidence layer green while another checks
a subtly different property.

**Cleanup sketch:** Introduce dependency-neutral IDs and generated codecs:

```go
type StepID string
type ResourceKind string
type FactID string
type PropertyID string

const StepAuthenticate StepID = "authn.password.verify@v1"
```

#### Issue 2: provider control flow carries implicit facts

**Problem:** `beginAuthorize`, `resumeAuthorize`, and `finishAuthorize` encode
validation, obligation, authentication, consent, client/user/key freshness,
terminal consumption, and artifact issuance through local variables and branch
order.

**Where to look:** `internal/fositeadapter/provider.go:384-617` and
`:938-999`; `interaction.go:66-170`.

**Why it matters:** The intended facts are visible to a human reviewer but not
represented as typed inputs to an issuance boundary. Static analysis must infer
them from control flow, and model/refinement work must reconstruct them manually.

**Cleanup sketch:** Add a native, non-scriptable authorization proof object:

```go
type AuthorizationProof struct {
    Interaction    InteractionRef
    Request        ValidatedRequestRef
    Principal      CurrentPrincipalRef
    Authentication AuthenticationProof
    Consent        ConsentProof
    Signing        SigningCapabilityRef
}

func (p *Provider) commitAuthorization(
    ctx context.Context,
    proof AuthorizationProof,
) (AuthorizationCommit, error)
```

Construction remains internal and typed. Scripts can influence a narrow policy
decision but cannot construct proofs.

#### Issue 3: effects are discoverable only from implementation calls

**Problem:** The fact that a step reads a session, writes an interaction,
consumes a one-time capability, or emits an artifact is not declared in one
machine-readable place.

**Where to look:** provider/store methods and the auditlint protocol-lifecycle
heuristics under the production-review ticket.

**Why it matters:** Static analyzers infer effects from named calls; model authors
manually choose state variables; Goja graph validation cannot reject an effect in
the wrong slot without a descriptor; traces can omit an important boundary.

**Cleanup sketch:** Maintain a native descriptor registry with code symbol links,
required/produced facts, and effects. Test descriptor-to-code mappings rather
than pretending metadata proves the implementation follows it.

#### Issue 4: scenario parameters are stringly typed

**Problem:** Verification plan steps accept arbitrary raw JSON and drivers decode
it in switch cases.

**Where to look:** `pkg/verifyplan/plan.go:34-37` and
`verification_scenario_test.go:61-152`.

**Why it matters:** Unknown step kinds fail at execution rather than plan
materialization. Parameter schemas live in unexported driver structs. Tooling
cannot enumerate or generate valid scenarios from the plan package alone.

**Cleanup sketch:** Keep serialized JSON but register typed codecs:

```go
RegisterStep(StepDescriptor[BeginAuthorization]{
    ID: StepAuthorizeBegin,
    Decode: StrictJSON[BeginAuthorization],
})
```

#### Issue 5: tracing is manual and semantically partial

**Problem:** Security events are emitted explicitly from selected provider paths.

**Why it matters:** A new terminal or error path can omit an event. Runtime
verification then reasons over an incomplete trace without knowing that coverage
is incomplete.

**Cleanup sketch:** Wrap selected native transition functions with a recorder that
emits attempted, committed, rejected, and failed outcomes from the transition
result. Keep audit events separate because audit has different durability and
human semantics.

#### Issue 6: the proposed Goja graph is broader than the current native kernel

**Problem:** The original scripting design includes protocols, flows, nodes,
callbacks, resources, effects, and challenges before a shared native transition
catalog exists.

**Why it matters:** The graph risks defining semantics that the provider later
implements inconsistently. It can become a second identity architecture rather
than a configuration language for the native one.

**Cleanup sketch:** Make graph nodes reference registered native block and policy
descriptors. Reject unknown, incompatible, or overly powerful effects during
materialization. The graph selects composition; the registry defines semantics.

## 3. The proposed core concepts

The grammar uses six concepts. Each concept has a precise role.

### 3.1 Resources

A resource is a typed identity for state owned by a trusted subsystem. It is not
the state itself and never carries a raw secret.

```go
type ResourceKind string

const (
    ResourceInteraction ResourceKind = "interaction@v1"
    ResourceSession     ResourceKind = "browser_session@v1"
    ResourceClient      ResourceKind = "oauth_client@v1"
    ResourcePrincipal   ResourceKind = "principal@v1"
    ResourceGrant       ResourceKind = "grant@v1"
    ResourceCode        ResourceKind = "authorization_code@v1"
    ResourceTokenFamily ResourceKind = "refresh_family@v1"
    ResourceSigning     ResourceKind = "signing_capability@v1"
    ResourceGeneration  ResourceKind = "policy_generation@v1"
)

type ResourceRef struct {
    Kind ResourceKind `json:"kind"`
    ID   string       `json:"id"` // opaque correlation ID or keyed hash label
}
```

Runtime traces may use bounded correlation identifiers. Scripts receive redacted
descriptors. Formal models replace identifiers with finite atoms. Static analysis
uses resource kinds and code symbols rather than runtime IDs.

### 3.2 Facts

A fact is a proposition established for a resource at a point in a transition
history.

```text
request.validated(interaction)
browser.bound(interaction, browser)
client.current(interaction, clientGeneration)
principal.current(interaction, userGeneration)
authentication.satisfied(interaction, authTime, amr)
consent.satisfied(interaction, scopes)
signing.available(keyGeneration)
interaction.pending(interaction)
```

Facts are not all persisted. Some are derived from authoritative state during a
request. A fact descriptor states its authority and lifetime.

### 3.3 Obligations

An obligation is a fact that must be established before a selected sink or
terminal transition.

```go
type ObligationID string

const (
    ObligationLogin      ObligationID = "authn.login.required@v1"
    ObligationFreshLogin ObligationID = "authn.fresh.required@v1"
    ObligationConsent    ObligationID = "consent.required@v1"
    ObligationStepUp     ObligationID = "authn.step_up.required@v1"
    ObligationPwdChange  ObligationID = "password.change.required@v1"
)
```

`InteractionRequiredAction` can remain the compact storage representation. A
lossless codec maps it to obligation IDs. Unknown bits fail closed.

### 3.4 Steps

A step is a native domain transition with declared inputs, preconditions,
effects, outputs, observations, and failure classes.

```go
type StepDescriptor struct {
    ID              StepID
    Version         int
    InputSchema     SchemaRef
    OutputSchema    SchemaRef
    Reads           []ResourceKind
    Writes          []ResourceKind
    RequiresFacts   []FactID
    ProducesFacts   []FactID
    Discharges      []ObligationID
    MayCreate       []ObligationID
    Effects         []Effect
    Atomicity       AtomicityClass
    Idempotency     IdempotencyClass
    Scriptability   Scriptability
    CodeAuthorities []CodeAuthority
    EventKinds      []EventKind
}
```

The descriptor is immutable data. Native Go functions remain ordinary typed
functions. A registry binds the descriptor to typed codecs and, where useful, a
test/verification executor.

### 3.5 Effects

Effects describe security-relevant state interaction:

```text
read(resource-kind)
create(resource-kind)
update(resource-kind)
consume-once(resource-kind)
revoke-family(resource-kind)
issue-artifact(kind)
emit-security-event(kind)
invoke-policy(slot)
invoke-capability(name, effect-class)
```

Effects are not permissions by themselves. Scriptable graph slots state which
effect classes they may select or invoke. Static analyzers compare declared
effects with calls made by code authorities. Model exporters use effects to
choose state variables and atomic actions.

### 3.6 Outcomes and observations

A transition result separates domain outcome from execution failure:

```go
type Outcome string

const (
    OutcomeApplied  Outcome = "applied"
    OutcomeRejected Outcome = "rejected"
    OutcomeDenied   Outcome = "denied"
    OutcomeExpired  Outcome = "expired"
    OutcomeConflict Outcome = "conflict"
)

type TransitionResult[T any] struct {
    Step         StepID
    Resources    []ResourceRef
    Outcome      Outcome
    Value        T
    Produced     []Fact
    Discharged   []ObligationID
    Observations []Observation
}
```

Storage outage, context cancellation, and invariant violation remain errors, not
domain denials. This distinction matters for fail-closed behavior and trace
interpretation.

## 4. Three schemas, not one universal graph

### 4.1 Configuration graph

The configuration graph belongs to the Goja design. It answers:

- which native protocols and authentication blocks are enabled;
- which policy callbacks occupy approved slots;
- which opaque host resources and capabilities are required;
- which generation/source hash is active; and
- which embedded policy tests must pass before activation.

It must not describe low-level OAuth parsing, token issuance, SQL statements, or
cryptographic operations.

### 4.2 Native transition catalog

The catalog answers:

- what domain transitions exist;
- which code symbols are authorized to implement them;
- what state they read/write;
- which facts and obligations they require/produce;
- what atomicity and idempotency class they claim;
- which trace events must be emitted; and
- whether scripts may select, configure, invoke, or never observe them.

The catalog is initially descriptive and checked by tests/static analyzers. It
does not dynamically dispatch production requests.

### 4.3 Scenario and trace schemas

Scenarios request executable test steps. Traces record observed transitions.
They share stable IDs but are not identical.

```text
ScenarioStep:
  step ID + bounded public parameters + desired scheduling/failure choice

TraceEvent:
  step ID + resource correlations + outcome + fact/obligation deltas + time
  + configuration generation + implementation/evidence version
```

A trace must not claim a fact solely because the descriptor says it should have
been produced. The runtime transition result supplies actual produced facts; the
monitor compares them with the catalog and property rules.

## 5. Authorization vertical slice

### 5.1 Step catalog

```text
AUTH.BEGIN@v1
  reads: client, optional session, consent, signing capability
  creates: interaction
  produces: request.validated, interaction.pending, browser.bound
  may create: login, fresh-login, consent, step-up obligations

AUTH.AUTHENTICATE_PASSWORD@v1
  reads: interaction, credential, account-security state
  writes: account-security state, optional session
  produces: authentication.satisfied or password.change.required
  discharges: login, fresh-login when successful and not blocked

AUTH.APPROVE_CONSENT@v1
  reads: interaction, client, principal
  writes: consent
  produces: consent.satisfied
  discharges: consent

AUTH.DENY@v1
  consumes once: interaction
  outcome: denied terminal

AUTH.COMMIT_APPROVAL@v1
  requires: validated request, pending interaction, current client/principal,
            satisfied required obligations, active signing capability
  consumes once: interaction
  effects: authorization protocol records + response artifact
```

### 5.2 Typed proof boundary

`finishAuthorize` currently accepts `AuthorizeRequester`, user, authentication
time, consent boolean, and optional interaction. Those values do not make the
required evidence explicit. Refactor toward internal proof types with
unexported constructors:

```go
type authorizationProof struct {
    interaction pendingInteraction
    request     validatedAuthorizeRequest
    principal   currentPrincipal
    authn       authenticationProof
    consent     consentProof
    signer      signingCapability
}

func (p *Provider) commitAuthorization(
    ctx context.Context,
    proof authorizationProof,
) (authorizationCommit, error)
```

This is not intended as cryptographic proof. It is a compile-time construction
discipline and a clear static-analysis sink. The analyzer can enforce that only
approved constructors create these values and only `commitAuthorization` issues
artifacts.

### 5.3 Pseudocode flow

```text
beginAuthorize(request):
    validated = fositeValidate(request)
    snapshot = loadCurrentResources(validated)
    obligations = deriveObligations(validated, snapshot)
    interaction = createInteraction(validated, snapshot, obligations)
    record AUTH.BEGIN result

resumeAuthorize(handle, form):
    interaction = loadAndBind(handle, browser)
    request = reconstructValidatedRequest(interaction)
    state = evaluateCurrentResources(interaction)

    if denial requested:
        atomically consume as denied
        record AUTH.DENY
        return protocol denial

    if authentication obligation remains:
        auth = authenticate(form.credentials)
        if auth.mustChangePassword:
            persist password-change obligation
            reject or start native password-change challenge
        establish authentication proof
        record AUTH.AUTHENTICATE_PASSWORD

    if consent obligation remains:
        establish consent proof
        record AUTH.APPROVE_CONSENT

    proof = assembleAuthorizationProof(...)
    commit = commitAuthorization(proof)
    record AUTH.COMMIT_APPROVAL from commit result
```

### 5.4 What stays outside the grammar

- Fosite request and response objects;
- password and token bytes;
- signing key material;
- CSRF implementation details;
- SQL transaction handles;
- Goja runtime objects and callbacks;
- HTTP writers and requests; and
- cryptographic verification algorithms.

The grammar refers to their trusted results and effect boundaries.

## 6. Static-analysis integration

### 6.1 Generated analyzer knowledge

The transition catalog can generate or feed:

- approved code-authority sets;
- artifact and terminal sinks;
- required proof/fact types;
- declared read/write/effect summaries;
- step and event ID validation;
- forbidden scriptability/effect combinations; and
- missing descriptor/event coverage diagnostics.

The catalog must not generate the entire analyzer. Path-sensitive rules still
need explicit abstract domains and transfer semantics. Generated metadata reduces
duplicated symbol lists and identifier drift.

### 6.2 Descriptor conformance analyzer

```text
for each registered transition descriptor D:
    resolve every CodeAuthority to a Go object
    collect calls and typed effects in the implementation
    report undeclared security-critical effects
    report declared mandatory events with no reachable emitter
    report production artifact sinks outside any registered authority
```

This analysis is approximate. A matching descriptor does not prove semantic
correctness. Its claim is architectural: security-sensitive operations remain in
known transition authorities and declared effects do not silently expand.

### 6.3 Proof-type analyzer

The typed authorization proof creates a stronger, simpler rule:

```text
sink: commitAuthorization(authorizationProof)
rule: proof values may originate only from approved unexported constructors
rule: constructor arguments must dominate construction on all paths
rule: MustChangePassword blocks authenticationProof construction
```

This is easier to analyze than inferring the entire protocol from arbitrary
booleans and requester values.

## 7. Model-checking integration

### 7.1 Catalog-to-model skeleton

The catalog can generate a model skeleton, not a finished formal model:

```text
VARIABLES resourceStates, facts, obligations, outcomes

AUTH_BEGIN ==
    /\ preconditions from descriptor
    /\ resources/facts/obligations updated nondeterministically within bounds

AUTH_COMMIT_APPROVAL ==
    /\ required facts present
    /\ required obligations absent
    /\ interaction pending
    /\ interaction becomes approved
```

Human model authors must still choose abstraction, finite domains,
nondeterminism, environment actions, fairness, and invariants. Descriptor
generation prevents spelling drift and supplies a reviewable starting point.

### 7.2 Counterexample replay

Because model actions and scenario steps share `StepID`, a normalized
counterexample can map to VerificationPlan codecs:

```text
TLC AUTH.BEGIN(tab1, promptLogin)
  -> ScenarioStep{Step: AUTH.BEGIN@v1, Parameters: ...}

TLC AUTH.AUTHENTICATE_PASSWORD(tab1, blank)
  -> ScenarioStep{Step: AUTH.AUTHENTICATE_PASSWORD@v1, Parameters: result fixture}
```

Some model actions, such as crash, scheduler choice, or administrative mutation,
map to harness-only steps. The registry marks execution profile explicitly.

### 7.3 Trace refinement evidence

Runtime traces can be projected into model observations. The comparison supports
a bounded conformance claim for executed traces; it does not prove general Go-to-
TLA+ refinement.

```text
native trace
  -> erase implementation-only fields
  -> normalize resources and logical time
  -> replay action sequence through abstract model
  -> compare abstract and concrete outcome/fact deltas
```

## 8. Scriptability integration

### 8.1 Scriptability classes

```go
type Scriptability string

const (
    ScriptNeverObserve Scriptability = "native-only"
    ScriptSelect       Scriptability = "selectable"
    ScriptConfigure    Scriptability = "configurable"
    ScriptPolicy       Scriptability = "policy-callback"
    ScriptVerify       Scriptability = "verification-only"
)
```

Examples:

| Transition/effect | Class | Reason |
|---|---|---|
| Validate redirect URI/PKCE | native-only | protocol trust boundary |
| Select password vs passkey block | selectable | graph composition |
| Password hash verification | native-only | credential secret and timing |
| Application allow/deny | policy-callback | narrow post-validation decision |
| Computed application claims | policy-callback | validated bounded output |
| Token/code creation | native-only | protocol artifact authority |
| Clock advance/crash injection | verification-only | test capability |

### 8.2 Graph validation through descriptors

The Goja compiler may create graph nodes referencing stable block IDs. The
materializer resolves each node against the native catalog and validates input/
output types, effect class, capability requirements, and allowed slot.

```text
script node "password-primary"
  kind: authn.password@v1
  config: {credentialStore: "primary"}

materializer:
  resolve authn.password@v1 descriptor
  reject if used in claims slot
  resolve opaque store reference in host registry
  construct native password block
```

JavaScript never supplies a function that replaces the native password
transition.

### 8.3 Verification scripting

The existing `tinyidp/verify` compiler is the correct safety pattern. Refactor it
to enumerate registered verification step codecs and assertion descriptors.
Keep live fake clocks, stores, failpoints, and providers in the Go driver. The
script compiles a plan; it does not receive capabilities.

## 9. Tracing, audit, and observability

### 9.1 Distinct records

- A transition trace is machine-oriented evidence of step execution.
- An audit event is a durable human/security accountability record.
- A metric is an aggregate bounded measurement.
- A debug log is implementation diagnostics.

One transition result may produce all four through separate adapters. They
should not share an unrestricted attribute map.

### 9.2 Transition envelope

```go
type TransitionEvent struct {
    SchemaVersion    int
    CatalogHash      string
    Step             StepID
    Attempt          uint64
    Time             time.Time
    Generation       string
    Resources        []ResourceRef
    Outcome          Outcome
    ProducedFacts    []FactID
    Discharged       []ObligationID
    Created          []ObligationID
    ErrorClass       string
}
```

No subject email, token, cookie, password, raw code, arbitrary policy error, or
unbounded client string is included. A separate redaction-reviewed projection
may create audit records.

### 9.3 Attempt versus commit

For crash and transaction analysis, one `step.completed` event is insufficient.
Selected transitions need an attempt identifier and explicit commit observation:

```text
transition.attempted
storage.commit_started
storage.commit_succeeded
response.delivery_attempted
transition.observed
```

These should be introduced only where the model distinguishes the boundaries.
Excess events create cost and ambiguity.

## 10. Package organization

The first implementation should remain internal while vocabulary changes:

```text
internal/assurance/
  ids/             stable IDs and parsing
  catalog/         resource/fact/obligation/step descriptors
  transition/      result and observation envelopes
  trace/           recorder and projections
  verifycodec/     typed VerificationPlan codecs
  modelexport/     skeleton/export helpers

internal/authorization/
  state.go         typed interaction snapshot
  obligations.go   derive and discharge rules
  proof.go         unexported proof types/constructors
  transition.go    pure or nearly pure transition decisions

pkg/idpgraph/
  graph.go         configuration graph DTOs
  descriptor.go    references to native block IDs
  validate.go

internal/idpgraphruntime/
  materialize.go
  registry.go
```

Do not create `pkg/idpgraph` until the first graph schema is approved as a public
contract. A first spike can use `internal/idpgraph` and move it deliberately;
there is no need for a compatibility adapter before release.

`internal/securitytrace` should either consume the dependency-neutral assurance
IDs or be folded into `internal/assurance/trace`. Avoid a circular dependency on
`fositeadapter` or `idpstore`.

## 11. Migration plan

### Phase 0: vocabulary and mapping only

- Inventory every existing action, event, obligation, assertion, and model step.
- Assign stable IDs and produce a crosswalk.
- Create descriptor schemas and validate uniqueness/reference integrity.
- Make no provider behavior changes.

Exit criterion: the forced-login property can be traced across store bits,
provider symbols, trace events, scenario steps, tests, analyzers, and proposed
model actions from one catalog entry.

### Phase 1: typed VerificationPlan codecs

- Register `session.login`, `authorize.begin`, `interaction.submit`, and
  `clock.advance` under stable step IDs.
- Generate JSON schema and documentation from codecs.
- Keep the serialized v1 plan only if compatibility is explicitly required;
  otherwise update fixtures and remove old open-string paths.
- Reject unknown steps at validation/materialization time.

Exit criterion: Goja compilation, native construction, and model trace import all
produce the same validated step form.

### Phase 2: transition trace envelope

- Map current security events to stable steps and properties.
- Introduce transition results around authorization interaction boundaries.
- Generate monitor input from results rather than scattered event calls.
- Test secret absence and event coverage.

Exit criterion: every authorization terminal path produces a complete cataloged
trace or an explicit instrumentation-failure signal.

### Phase 3: authorization proof boundary

- Extract current-resource snapshot and obligation derivation.
- Introduce unexported proof types and one artifact issuance sink.
- Preserve Fosite and HTTP behavior with characterization tests.
- Add static rules for constructor and sink authority.

Exit criterion: forced-login and `MustChangePassword` bypass mutations fail at
tests/static analysis, and only the approved sink can issue artifacts.

### Phase 4: formal model and counterexample integration

- Generate the authorization model vocabulary/skeleton.
- Maintain human-authored abstraction and invariants.
- Normalize checker traces to registered scenario steps.
- Compare native traces against abstract observations.

Exit criterion: one historical model counterexample replays without a handwritten
action-name adapter.

### Phase 5: configuration graph materializer

- Implement descriptors for native password, consent, authorization policy, and
  computed claims slots.
- Compile Goja configuration to graph references.
- Materialize only registered compatible native blocks/resources.
- Bind graph generation/hash into interactions and traces.

Exit criterion: the local-web preset materializes to the existing strict provider
without moving protocol validation or issuance into JavaScript.

### Phase 6: selective pure kernels

- Extract only transition decisions that become clearer as pure functions.
- Use the same kernel in model-based tests, runtime executor, and deductive
  verification experiments where practical.
- Keep I/O orchestration and dependency integration explicit.

Exit criterion: extraction reduces duplicated semantics and improves evidence;
it is not accepted merely for architectural uniformity.

## 12. Alternatives

### Full generic workflow interpreter

This would make every node implement a common execute interface and route all
authorization through a graph engine. It improves uniform logging and extension
at the cost of type erasure, a larger trusted interpreter, harder Fosite
alignment, and dynamic effect authority. It is not recommended for the first
several phases.

### Code annotations only

Comments or directives could attach step/effect IDs to existing functions. This
is useful for a Phase 0 spike and static mapping, but it cannot provide typed
scenario codecs, runtime result envelopes, or graph validation by itself.

### Generate Go from the graph

Generated Go could restore static types while allowing composition. It adds a
build/deployment pipeline and does not solve request-time policy callbacks. It
may be reconsidered for fixed appliances after the native catalog stabilizes.

### Keep current architecture unchanged

This remains a valid near-term choice. The present system is testable and its
security boundaries are improving. If the graph is not implemented, the lowest
risk work is still stable identifiers, typed VerificationPlan codecs, and
consolidated trace/property catalogs. Those changes have value independently.

## 13. Decision records

### Decision: use three linked schemas

- **Context:** Configuration, executable transitions, scenarios, and traces have
  related identities but different trust and semantic requirements.
- **Options considered:** one universal graph; independent schemas; linked
  configuration, catalog, and scenario/trace schemas.
- **Decision:** Use three linked schemas sharing stable IDs.
- **Rationale:** This preserves boundaries while eliminating vocabulary drift.
- **Consequences:** Cross-schema validators and catalog ownership are required.
- **Status:** proposed

### Decision: catalog before interpreter

- **Context:** Replacing provider branches with a workflow runtime would be a
  large security-sensitive rewrite.
- **Decision:** First describe and instrument existing native transitions.
- **Rationale:** Static/model/trace benefits arrive before execution changes.
- **Consequences:** Descriptor conformance is approximate until selected kernels
  are structurally extracted.
- **Status:** proposed

### Decision: scripts configure and decide within slots

- **Context:** JavaScript improves composition and application policy but is an
  unsuitable owner for protocol validation, credentials, atomicity, and issuance.
- **Decision:** Scripts compile configuration graphs and bounded policy results;
  native descriptors define blocks/effects.
- **Consequences:** Novel script-only protocols require new reviewed native
  blocks rather than arbitrary callbacks.
- **Status:** consistent with accepted Goja ticket decisions

### Decision: typed proofs at irreversible sinks

- **Context:** Artifact issuance currently depends on implicit local facts.
- **Decision:** Introduce unexported proof types for selected irreversible sinks.
- **Rationale:** This improves code review and static authority analysis without
  claiming formal proof.
- **Consequences:** Provider refactoring and constructor discipline are required.
- **Status:** proposed

## 14. Risks and controls

- **Grammar becomes a second implementation.** Keep descriptors declarative and
  test code-authority/effect mappings.
- **Identifier churn.** Version IDs and approve semantic changes explicitly.
- **Over-generalization.** Begin with authorization interaction, not every route.
- **Trace overclaim.** Record coverage and instrumentation failures; traces prove
  only observed executions.
- **Model generation overclaim.** Generate skeletons/vocabulary, not invariants or
  abstraction decisions.
- **Script capability creep.** Scriptability is an explicit descriptor field;
  native-only is the default.
- **Performance overhead.** Use compact typed results and selective traces; measure
  provider latency and allocations.
- **Fosite mismatch.** Keep Fosite validation/orchestration native and add
  characterization tests before refactors.
- **Package instability.** Keep the grammar internal until exercised by at least
  two consumers and one complete vertical slice.

## 15. Validation strategy

The refactor is accepted only if it preserves or improves:

- hosted and native OIDC conformance behavior;
- existing provider and store tests;
- forced reauthentication and password-change regressions;
- interaction Rapid/fuzz histories;
- Porcupine linearizability results;
- SQL failpoint atomicity results;
- runtime monitor verdicts;
- VerificationPlan execution;
- static analyzer mutation sensitivity; and
- provider latency/allocation budgets.

Add differential tests that run the old and refactored authorization flows over a
fixed scenario corpus and compare HTTP outcome, redirect/error, durable rows,
audit projection, transition trace, and security monitor result. Do not compare
opaque random token bytes.

## 16. Intern implementation entry point

The first contribution should be vocabulary-only:

1. inventory current action/event/assertion strings and required-action bits;
2. define stable IDs in a dependency-neutral internal package;
3. create an authorization crosswalk with code symbols and property IDs;
4. add codecs between `InteractionRequiredAction` and obligation IDs;
5. update the runtime monitor to consume the shared obligation representation;
6. add tests for unknown IDs/bits and stable JSON encoding;
7. do not change provider control flow yet.

This contribution proves the package dependency shape and creates immediate
drift detection. The next contribution can add typed VerificationPlan codecs.

## 17. Final assessment

tiny-idp does not need a wholesale reorganization before model checking, static
analysis, or scripting can proceed. Its recent hardening work has already created
the right domain objects and evidence surfaces. A large workflow rewrite now
would spend that advantage.

The highest-leverage refactor is vocabulary and authority consolidation followed
by typed irreversible boundaries. A small core grammar of resources, facts,
obligations, steps, effects, and outcomes can make the system easier to analyze,
model, script, trace, and review. Its success depends on remaining descriptive
and typed where the current system is native and security-critical. Configuration
graphs should select reviewed native semantics, not replace them.

## 18. Local references

- `pkg/idpstore/types.go:175-224` — required actions and interaction state.
- `pkg/idpstore/interfaces.go:83-89,154-166` — one-time interaction and named
  atomic operations.
- `internal/fositeadapter/provider.go:384-617,938-999` — authorization control
  flow and irreversible issuance boundary.
- `internal/fositeadapter/interaction.go:66-170` — server-owned continuation.
- `pkg/verifyplan/plan.go:13-158` — current data-only scenario grammar.
- `internal/gojaverify/compiler.go` — isolated compile-only Goja path.
- `internal/gojamodules/verify/module.go` — capability-free plan authoring API.
- `internal/securitytrace/trace.go` — runtime transition events and monitor.
- `internal/fositeadapter/state_model_test.go` — Rapid model/action vocabulary.
- `internal/fositeadapter/linearizability_test.go` — Porcupine sequential model.
- `TINYIDP-GOJA-001/design-doc/01-...md` — configuration graph and scripting
  architecture.
- `TINYIDP-GOJA-001/reference/02-...md` — separate verification scripting plane.
- `TINYIDP-MODEL-001/design-doc/02-...md` — model-checking evidence architecture.
- `TINYIDP-STATIC-001/design-doc/01-...md` — static-analysis architecture,
  abstract domains, mutation evaluation, and deductive-verification boundary.

## 19. Research foundations used in this synthesis

The design relies on the research packets preserved under the model-checking and
static-analysis tickets: formal OAuth/OIDC analysis for protocol/attacker
boundaries; security automata and runtime verification for observable bad
prefixes; typestate for state-dependent operation authority; IFDS/dataflow and
abstract interpretation for implementation analysis; model-based testing and
linearizability for executable transition semantics; CHESS and fault injection
for scheduling/crash boundaries; and Gobra for the limits and opportunities of
deductive Go verification. The Goja ticket's original microkernel research
supplies the graph, block, slot, resource, capability, and activation concepts.
