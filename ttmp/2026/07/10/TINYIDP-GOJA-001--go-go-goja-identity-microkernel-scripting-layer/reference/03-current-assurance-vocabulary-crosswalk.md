---
Title: Current assurance vocabulary crosswalk
Ticket: TINYIDP-GOJA-001
Status: active
Topics:
    - architecture
    - auth
    - oidc
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/idpstore/types.go
      Note: Persisted interaction obligations and outcomes
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: Current private interaction model actions
    - Path: repo://pkg/verifyplan/plan.go
      Note: Current stringly typed verification steps and observations
    - Path: repo://internal/fositeadapter/verification_scenario_test.go
      Note: Strict-provider scenario driver and assertion vocabulary
    - Path: repo://internal/securitytrace/trace.go
      Note: Secret-free monitored security-event vocabulary
    - Path: repo://internal/assurance/vocabulary.go
      Note: Shared versioned identifier foundation
ExternalSources: []
Summary: Exhaustive source crosswalk of the current persisted, model, verification, trace, and static-property vocabularies that must converge on the assurance grammar.
LastUpdated: 2026-07-20T00:00:00-04:00
WhatFor: Implementing and reviewing the shared assurance vocabulary, transition catalog, typed verification codecs, and trace instrumentation.
WhenToUse: Read before adding or changing an assurance identifier, interaction action, scenario step, security event, or static property.
---

# Current assurance vocabulary crosswalk

## Scope and method

This is the implementation baseline for Phase 9 task `agxk`. It inventories
every current finite vocabulary used by persisted interaction state, the private
reference model, verification plans and assertions, security traces, and the
static assurance surface. Ordinary test fixture client/user IDs are intentionally
excluded: they are test data, not a vocabulary contract.

The source scan is reproducible with:

```bash
rg -n 'InteractionRequire|interactionModelAction|Kind:|kind:|id:' \
  pkg/idpstore internal/fositeadapter pkg/verifyplan internal/securitytrace
```

## Persisted interaction obligations

`pkg/idpstore/types.go` stores a compact `InteractionRequiredAction` bit set.
Each bit becomes a versioned assurance obligation in the next implementation
step. Unknown bits must fail closed rather than being silently ignored.

| Persisted bit | Proposed obligation ID | Meaning |
|---|---|---|
| `InteractionRequireLogin` | `authn.login.required@v1` | Establish an authenticated principal. |
| `InteractionRequireFreshLogin` | `authn.fresh.required@v1` | Reauthenticate rather than reusing the session. |
| `InteractionRequireConsent` | `consent.required@v1` | Obtain a valid approval for requested scopes. |
| `InteractionRequireStepUp` | `authn.step_up.required@v1` | Establish stronger authentication evidence. |
| `InteractionRequireAccountSelection` | `account.selection.required@v1` | Select one server-bound remembered session. |
| `InteractionRequireRegistration` | `registration.required@v1` | Complete the native registration continuation. |

The current terminal interaction vocabulary is also finite:

| Current outcome | Proposed transition outcome |
|---|---|
| `approved` | `applied` |
| `denied` | `denied` |
| `rejected` | `rejected` |
| missing/expired/consumed store result | `not_found` / `expired` / `conflict` |

## Private model actions

`internal/fositeadapter/state_model_test.go` currently uses private action
names. They are valuable but cannot yet be replayed by `verifyplan` without an
adapter. The transition catalog will use these stable step IDs:

| Private model action | Proposed step ID |
|---|---|
| `modelCreate` | `interaction.create@v1` |
| `modelGet` | `interaction.load@v1` |
| `modelApprove` | `interaction.approve@v1` |
| `modelDeny` | `interaction.deny@v1` |
| `modelAdvancePastExpiry` | `clock.advance@v1` |
| `modelMutateReturnedCopy` | `interaction.copy_mutation@v1` |

## Verification plan and scenario vocabulary

`pkg/verifyplan/plan.go` currently permits arbitrary `Step.Kind` and
`Observation.Kind`; the strict-provider driver recognizes the following closed
set. Task `vpdm` will make this registry executable and reject unknown kinds at
plan materialization time.

| Category | Current string | Proposed stable ID |
|---|---|---|
| Scenario step | `session.login` | `session.login@v1` |
| Scenario step | `authorize.begin` | `authorize.begin@v1` |
| Scenario step | `interaction.submit` | `interaction.submit@v1` |
| Scenario step | `clock.advance` | `clock.advance@v1` |
| Observation | `session.established` | `session.established@v1` |
| Observation | `authorize.response` | `authorize.response@v1` |
| Observation | `interaction.response` | `interaction.response@v1` |
| Observation | `clock.advanced` | `clock.advanced@v1` |
| Assertion | `credentialFormShown@v1` | `ui.credential_form_shown@v1` |
| Assertion | `noAuthorizationCode@v1` | `artifact.no_authorization_code@v1` |
| Assertion test fixture | `freshAuthenticationBeforeIssuance@v1` | `authn.fresh_before_issuance@v1` |
| Assertion test fixture | `observedKind@v1` | `trace.observed_kind@v1` |

## Security trace events and monitored properties

`internal/securitytrace/trace.go` owns the existing secret-free event schema.
The catalog must preserve these event strings while introducing a versioned
identifier alias:

| Event kind | Proposed observation ID | Existing monitor property |
|---|---|---|
| `interaction.created` | `interaction.created@v1` | exactly one creation before interaction events |
| `authentication.satisfied` | `authentication.satisfied@v1` | required auth precedes approval |
| `consent.approved` | `consent.approved@v1` | required consent precedes approval |
| `consent.denied` | `consent.denied@v1` | denial does not count as consent |
| `interaction.terminal` | `interaction.terminal@v1` | one terminal outcome |
| `authorization.artifacts_committed` | `authorization.artifacts_committed@v1` | artifacts only after approved terminal state; at most once |
| `token.lifecycle_committed` | `token.lifecycle_committed@v1` | independent token lifecycle boundary |

The monitor's current static properties are therefore:

- `interaction.created_once@v1`
- `authorization.required_auth_before_approval@v1`
- `authorization.required_consent_before_approval@v1`
- `interaction.single_terminal@v1`
- `authorization.artifacts_after_approval@v1`
- `authorization.artifacts_once@v1`

## Diagnostic and scripting identifiers already in use

`pkg/idpprogram` provides bounded program/schema/lambda/capability/effect
diagnostics. `pkg/idp.AuthorizationDecision` already validates bounded stable
diagnostics and evidence such as `policy.member_required` and
`evidence.community_membership`. These are retained as `DiagnosticID` and
`EvidenceID` values; the following migration must validate them through the
shared vocabulary rather than a second local grammar.

## Deliberate non-mappings

- OAuth client IDs, user IDs, email addresses, redirect URIs, browser handles,
  device/user codes, token values, and source locations are data, not stable
  vocabulary identifiers. They must never become trace/metric dimensions.
- Script-defined workflow and handler IDs remain configuration identifiers. The
  compiler validates their bounded syntax, while the transition catalog contains
  only host-owned step IDs.
- The deprecated graph-first task list is historical evidence only and is not
  part of this crosswalk.
