---
Title: Assurance Methods and Evidence Interpretation
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - research
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: |-
        Property and fuzz models
        Generated testing
    - Path: repo://internal/securitytrace/trace.go
      Note: |-
        Runtime verification monitor
        Runtime verification
    - Path: repo://pkg/verifyplan/plan.go
      Note: |-
        Native verification plan execution boundary
        Native verdict boundary
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: |-
        Repository-specific Go AST analyzers
        Go AST analyzers
ExternalSources: []
Summary: How static analysis, examples, property testing, fuzzing, fault injection, race detection, linearizability, runtime monitoring, conformance, and review support different claim classes.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Preventing overclaiming and helping contributors select the cheapest technique capable of falsifying a security hypothesis.
WhenToUse: During test design, security review, analyzer development, evidence collection, or release decisions.
---


# Assurance Methods and Evidence Interpretation

## Start with the claim

Tool selection follows the property. Static analysis is effective for forbidden
dataflow or missing error checks. A state model is effective for legal operation
sequences. Linearizability is effective for concurrent histories. Hosted
conformance is effective for standardized external protocol behavior. None is a
universal security oracle.

| Method | Strongest supported claim | Important limitation |
|---|---|---|
| Example test | named behavior works | sparse input/history coverage |
| `go/analysis` | syntactic/typed pattern absent or present | bounded by rule precision |
| Rapid property | generated sequences satisfy model relation | model may be incomplete |
| Native fuzzing | no counterexample in explored input space/time | weak semantic oracle |
| Metamorphic test | controlled transform preserves/changes relation | relation must be correct |
| Failpoints | named intermediate failures preserve invariant | unenumerated failures remain |
| Race detector | executed code has no observed Go data race | not semantic concurrency proof |
| Porcupine | history admits legal sequential explanation | model and observed output are abstractions |
| Trace monitor | emitted facts satisfy temporal property | missing instrumentation is invisible |
| OIDF conformance | external standardized cases pass | not application policy or operations |
| Human review | design assumptions and omissions challenged | reviewer scope and independence matter |

## Static analysis

The custom analyzers use Go AST, types, and local control structure. They detect
browser continuation reads, ignored security errors, insecure bearer transport,
fail-open parsing, direct security wall clocks, limiter identity risks, and
protocol writes outside lifecycle helpers.

Every analyzer needs positive fixtures, negative fixtures, a repository scan,
and a precision statement. The lifecycle analyzer is not a whole-program proof;
it checks concrete adapter methods for required helper calls. This limitation is
part of the tool's contract, not an embarrassment to omit.

## Generated testing and fuzzing

Property testing generates meaningful actions and compares them with a reference
model. Fuzzing mutates bytes and is most effective when the harness has a strong
oracle: no panic, strict parse result, terminal uniqueness, or monitor verdict.
Shrinking and committed replay histories convert discovery into maintainable
regressions.

The seed, plan hash, source commit, action sequence, observations, trace schema,
and assertion versions should eventually form one evidence envelope. A bare
statement that “fuzzing passed” cannot be reproduced.

## Runtime verification and scripting

Security events describe authoritative native transitions. The native monitor
owns verdicts. Goja scripts compile data-only `VerificationPlan` values and may
select scenarios, but they receive no provider, store, network, clock, or
assertion authority. Otherwise the same script could create, conceal, and judge
its own evidence.

The in-process compiler has source, time, and output limits, not a hard heap
quota. Trusted reviewed scripts are the accepted threat model. Hostile tenant
scripts would require process isolation.

## Evidence discipline

Use this template for every claim:

```text
Claim:
Threat/counterexample:
Authoritative state:
Falsification method:
Observed result:
Coverage boundary:
Residual uncertainty:
```

## Exercises

1. Choose a technique for “browser fields cannot replace canonical state” and
   explain why fuzzing alone is insufficient.
2. State the false-negative boundary of one auditlint analyzer.
3. Explain what a monitor cannot detect when an event is never emitted.
4. Distinguish a passing race run from a passing Porcupine history.
5. Write an evidence statement for a 30-second fuzz campaign without claiming
   exhaustive correctness.

## Research packet

Read `paper-ifds-dataflow-analysis`, `paper-model-based-security-testing`,
`paper-stateful-greybox-fuzzing`, `paper-chess-systematic-concurrency-testing`,
`paper-lineage-driven-fault-injection`, and
`paper-runtime-verification-brief-account` under `sources/`.
