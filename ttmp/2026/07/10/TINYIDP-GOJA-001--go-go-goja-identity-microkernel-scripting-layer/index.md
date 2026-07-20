---
Title: Go go goja identity microkernel scripting layer
Ticket: TINYIDP-GOJA-001
Status: complete
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Provider-owned workflow renderer dependency.
    - Path: repo://internal/fositeadapter/rendering.go
      Note: Native workflow page response security boundary.
    - Path: repo://internal/fositeadapter/workflow_rendering_internal_test.go
      Note: Workflow CSP, cache, and hidden-control regression test.
    - Path: repo://internal/gojamodules/tinyidp/module.go
      Note: Branded field/action builders and data-only ctx.present.form bridge.
    - Path: repo://internal/gojamodules/tinyidp/typescript.go
      Note: TypeScript declarations for presentation and descriptor handles.
    - Path: repo://pkg/idpprogram/outcomes.go
      Note: VM-independent presentation transport envelope.
    - Path: repo://pkg/idpscript/invoke.go
      Note: Frozen per-invocation presentation context binding.
    - Path: repo://pkg/idpscript/invoke_test.go
      Note: End-to-end presentation outcome serialization coverage.
    - Path: repo://pkg/idpui/workflow.go
      Note: Validated redirect-origin workflow form contract.
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Deprecated historical graph-first analysis and implementation guide
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md
      Note: Normative lambda-first JavaScript API, explicit browser continuation, virtual resource, and implementation design
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/reference/01-investigation-diary.md
      Note: Chronological evidence and validation record
ExternalSources: []
Summary: Design and implementation ticket for a lambda-first Goja identity program with explicit browser continuations, virtual resources, native capability and effect boundaries, and a shared assurance grammar.
LastUpdated: 2026-07-20T00:14:05.351274378-04:00
WhatFor: Planning and implementing the tiny-idp identity-microkernel scripting layer.
WhenToUse: Start here when reviewing, implementing, or resuming TINYIDP-GOJA-001.
---






# Go-go-goja identity microkernel scripting layer

## Overview

This ticket defines a programmable identity kernel for the current tiny-idp and
go-go-goja repositories. Trusted JavaScript lambdas implement workflows,
virtual users, virtual invitations, routing, policy, and identity projection.
Go keeps Fosite protocols, browser security, secrets, credential and challenge
primitives, continuation persistence, atomic effects, sessions, and artifact
issuance inside the trusted computing base.

The normative API now uses explicit browser continuations. A lambda may await
bounded capabilities during one HTTP request. A form, email verification, or
other browser wait returns a presentation/challenge outcome naming the handler
that a later validated request resumes. No Goja heap or Promise is persisted.

## Key links

- [Normative lambda-first JavaScript API and explicit-continuation guide](design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md)
- [Deprecated graph-first analysis and implementation guide](design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md)
- [Assurance-oriented core grammar and refactoring proposal](design-doc/02-assurance-oriented-core-grammar-and-codebase-refactoring-proposal.md)
- [Security verification scripting-plane assessment](reference/02-security-verification-scripting-plane-assessment.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Colleague identity-microkernel research](sources/01-colleague-identity-microkernel-research.md)
- [Implementation tasks](tasks.md)
- [Changelog](changelog.md)

## Current status

Research, the deprecated graph-first design, the assurance refactoring design,
and the lambda-first API design are complete. Phases 0–3 of the normative
ledger are now implemented and gated: the isolated program compiler and owned
worker pool, durable generation-bound continuations, native workflow forms and
strict POST projection, and the checked-in open-signup JavaScript workflow with
one native atomic commit boundary. The remaining work begins at Phase 4:
virtual identity and invitation providers, then email challenge, activation,
and additional existing workflows.

The new refactoring proposal recommends consolidating stable resource, fact,
obligation, step, effect, outcome, observation, and property identifiers before
introducing a general graph executor. The first vertical slice is the existing
authorization interaction, preserving native Fosite and storage semantics.

## Key decisions

- Go remains the process host and trusted computing base.
- JavaScript registers typed named lambdas that implement workflows and virtual
  providers; the surrounding program graph constrains their inputs, outcomes,
  capabilities, effects, budgets, and continuation edges.
- `require("tinyidp").v1` is the primary API.
- Compiler and policy runtimes receive no ambient host modules.
- Production scripts reference host-owned resources by opaque name.
- Signup is the first production workflow slice; browser waits use explicit
  versioned continuations rather than suspended Goja Promises.
- Lambdas return effect plans; Go revalidates and commits irreversible changes.
- Policy infrastructure failure fails closed and unhealthy workers are replaced.
- Generated xgoja binaries require a provider package, TypeScript declarations,
  and generated-host smoke tests.

## Validation baseline

`go test ./... -count=1` passed from the current tiny-idp repository on
2026-07-10. The diary records exact commands, evidence, and investigation
failures.
