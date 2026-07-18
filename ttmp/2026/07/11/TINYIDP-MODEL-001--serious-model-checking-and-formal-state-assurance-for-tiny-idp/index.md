---
Title: Serious Model Checking and Formal State Assurance for tiny-idp
Ticket: TINYIDP-MODEL-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Dedicated architecture, specification, checker, counterexample, implementation-integration, research, and CI program for serious model checking of tiny-idp security state.
LastUpdated: 2026-07-12T00:35:00Z
WhatFor: Tracking the long-lived transition from generated state testing and finite-history checking to bounded exhaustive formal models tied back to Go regressions.
WhenToUse: Start here before research, modeling, checker experiments, implementation integration, or evidence review under TINYIDP-MODEL-001.
---

# Serious Model Checking and Formal State Assurance for tiny-idp

## Overview

tiny-idp already has Rapid reference models, Porcupine linearizability checks,
versioned runtime monitors, failpoints, deterministic clocks, and typed scenario
replay. This ticket adds the missing formal layer: reviewed abstractions and
bounded exhaustive specifications for authorization interactions, code
redemption, and refresh-token families.

The program is designed as an evidence-production pipeline: property catalog,
reviewed abstraction and assumption ledgers, three bounded formal models,
reproducible checker envelopes, normalized counterexamples, native Go replay,
and CI/release governance. Literature and theoretical qualification are one
early phase because the evidence is not credible unless the team understands
the selected semantics, bounds, fairness, and checker outcomes.

## Key Links

- [Primary system architecture and implementation plan](./design-doc/02-serious-model-checking-system-architecture-and-implementation-plan.md)
- [Companion theory, literature, and technical reader](./design-doc/01-model-checking-research-analysis-design-and-implementation-guide.md)
- [Chronological investigation diary](./reference/01-investigation-diary.md)
- [Detailed 84-task phase ledger](./tasks.md)
- [Research and primary-source packet](./sources/)

## Status

Current status: **active**

Current gate: ticket and research baseline complete; theoretical qualification,
abstraction review, and checker tutorial work remain open. No production formal
specification has been implemented yet.

## Topics

- architecture
- auth
- go
- oidc
- research
- testing

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Research, architecture, API, and implementation guides
- reference/ - Chronological diary and future focused reference material
- sources/ - Preserved official documentation, papers, and source captures
- scripts/ - Reproducible qualification experiments and temporary tooling
