---
Title: Static Analysis and Implementation Verification for tiny-idp
Ticket: TINYIDP-STATIC-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - security
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Long-lived program for repository-specific Go static analysis, interprocedural security dataflow, mutation evaluation, and selected deductive verification of tiny-idp."
LastUpdated: 2026-07-12T01:30:00Z
WhatFor: "Tracks the transition from a valuable ticket-local analyzer prototype to maintained, measurable, release-governed implementation assurance."
WhenToUse: "Start here before researching, implementing, promoting, suppressing, or interpreting tiny-idp static-analysis and verification rules."
---

# Static Analysis and Implementation Verification for tiny-idp

## Overview

tiny-idp already has fifteen custom `go/analysis` rules. This ticket turns that
prototype into a serious assurance program with stable invariant/rule IDs,
explicit AST/CFG/SSA/interprocedural semantics, taint and typestate analyses,
historical and seeded mutation benchmarks, evidence envelopes, CI governance,
and a bounded deductive-verification experiment.

The first vertical milestone is a path-sensitive authorization property covering
both persisted forced reauthentication and mandatory password change. Model
checking, runtime verification, and Go regressions retain separate evidence
claims while sharing the same canonical invariant catalog.

## Key Links

- [Primary research, architecture, and implementation guide](./design-doc/01-static-analysis-and-implementation-verification-research-design-and-implementation-guide.md)
- [Chronological investigation diary](./reference/01-investigation-diary.md)
- [Detailed 101-task phase ledger](./tasks.md)
- [Preserved research and tool documentation](./sources/)

## Status

Current status: **active**

Current gate: research/design baseline complete. Stable evidence semantics, tool
qualification, analyzer promotion, SSA infrastructure, mutation evaluation,
deductive experiments, and CI integration remain open. No production code has
been changed by this ticket.

## Topics

- architecture
- auth
- go
- oidc
- research
- security
- testing

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Primary architecture and implementation guide
- reference/ - Chronological diary and focused reference documents
- sources/ - Official tool documentation and foundational papers
- scripts/ - Reproducible experiments and temporary analysis tooling
