---
Title: Intern Security Review Labs
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - auth
    - go
    - identity
    - oidc
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/interaction_hardening_test.go
      Note: |-
        Browser protocol-state laboratory
        Browser security labs
    - Path: repo://internal/fositeadapter/linearizability_test.go
      Note: |-
        Concurrency laboratory
        Concurrency labs
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: |-
        Transaction and failpoint laboratory
        Failure atomicity labs
    - Path: repo://internal/fositeadapter/verification_scenario_test.go
      Note: |-
        Typed programmable verification laboratory
        Programmable verification lab
ExternalSources: []
Summary: Executable first-week exercises that connect the intern textbook chapters to real tiny-idp code, tests, traces, and review artifacts.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Measuring whether a new contributor can reason about the system rather than merely repeat documentation.
WhenToUse: Complete in order during onboarding and review answers with a maintainer.
---


# Intern Security Review Labs

## Lab 1: Trace authority

Run:

```bash
go test ./internal/fositeadapter -run TestStrictAuthorizationCodeFlow -count=1 -v
```

Produce a table for every request value showing its origin, validator,
server-owned representation, mutation point, and final artifact. Explain why
`state`, nonce, interaction handle, code, and access token are not interchangeable.

## Lab 2: Reproduce temporal defenses

```bash
go test ./internal/fositeadapter -run 'TestForcedPromptLoginCannotReuseExistingSession|TestExpiredMaxAgeCannotReuseExistingSession|TestAuthorizationInteractionRejectsSequentialReplay|TestAuthorizationInteractionIsOneTimeUnderConcurrency' -count=1 -v
```

Draw each history. Identify the event whose absence would permit the defect and
the authoritative store transition that rejects replay.

## Lab 3: Audit browser authority

Read `resumeAuthorize` and the `tinyidpinteractioncontinuation` analyzer. List all
POST fields read by the handler, classify them as opaque/native user input or
forbidden protocol continuation, and explain the analyzer's false-negative
boundary.

## Lab 4: Break transactions safely

```bash
go test ./internal/fositeadapter -run 'TestSQLiteAuthorizationCodeRedemptionFailpointsAreAtomic|TestSQLiteRefreshRotationFailpointsAreAtomicAndRetryable' -count=1 -v
```

Choose two failpoints. Predict database rows and security events before running.
Compare the prediction with assertions and explain the transaction owner.

## Lab 5: Interpret concurrency

```bash
go test ./internal/fositeadapter -run 'TestInteractionConsumeHistoryIsLinearizable|TestRefreshRotationHistoryIsLinearizable' -count=10 -v
```

Write the sequential model, identify the linearization point, and explain why the
refresh family can end revoked after one successful response.

## Lab 6: Compare assurance methods

Run the custom analyzer, one Rapid test, one fuzz target, and the security trace
monitor. For each, write one supported claim and one unsupported claim.

```bash
go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -- ./pkg/... ./internal/...
go test ./internal/fositeadapter -run TestInteractionStoreStateMachine -count=1
go test ./internal/securitytrace -run '^$' -fuzz FuzzMonitorEventSequences -fuzztime 2s -parallel=1
```

## Lab 7: Review the scripting boundary

Read `pkg/verifyplan`, `internal/gojaverify`, and
`verification_scenario_test.go`. Enumerate every object reachable by JavaScript.
Explain why the plan can select an assertion but cannot implement its verdict.
State the in-process memory limitation.

## Lab 8: Make a release decision

Read reference 06 and the operations playbook. Write a release recommendation
that separates local technical evidence, hosted evidence, artifact provenance,
independent review, and owner authority. The correct current decision is not
approval; the exercise is to justify every row precisely.

## Completion artifact

Submit a short review note with:

- one protocol invariant;
- one temporal counterexample;
- one transaction boundary;
- one concurrent history;
- one analyzer precision limit;
- one fuzz or property evidence limit;
- one production trust boundary;
- one open release blocker.

The mentor should challenge any statement that lacks a code symbol, test,
standard, trace, or explicit inference.
