---
Title: Go go goja identity microkernel scripting layer
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
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Primary intern-oriented analysis and implementation guide
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/reference/01-investigation-diary.md
      Note: Chronological evidence and validation record
ExternalSources: []
Summary: Design ticket for adding a safe graph-compiled go-go-goja scripting and policy layer to tiny-idp while preserving Go ownership of identity security invariants.
LastUpdated: 2026-07-10T11:11:55.240944351-04:00
WhatFor: Planning and implementing the tiny-idp identity-microkernel scripting layer.
WhenToUse: Start here when reviewing, implementing, or resuming TINYIDP-GOJA-001.
---



# Go-go-goja identity microkernel scripting layer

## Overview

This ticket translates the colleague research into an implementation plan for
the current tiny-idp and go-go-goja repositories. The core direction is to
compile trusted JavaScript configuration into an immutable Go graph, keep
Fosite/protocols/cryptography/storage/challenges in Go, and run only named,
bounded authorization and claims callbacks in single-owner policy runtimes.

The initial release is deliberately narrower than the research vision. It
covers the current strict OIDC provider, password/session authentication,
stored consent, static and computed claims, allow/deny policy, capabilities,
policy tests, atomic activation, and xgoja/v2 packaging. General challenge
composition, step-up, passkeys, token exchange, CIBA, workload identity, and
multi-actor flows follow as native block phases.

## Key links

- [Analysis, design, and implementation guide](design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Colleague identity-microkernel research](sources/01-colleague-identity-microkernel-research.md)
- [Implementation tasks](tasks.md)
- [Changelog](changelog.md)

## Current status

Research, design, docmgr validation, and reMarkable delivery are complete.
Implementation has not started. The review bundle is available at
`/ai/2026/07/10/TINYIDP-GOJA-001` as `TINYIDP GOJA 001 Identity Microkernel
Scripting Design.pdf`.

The first owner decision is whether tiny-idp may raise its minimum Go version to
match the current go-go-goja module. Phase 0 should then prove explicit module
isolation, program reuse, runtime ownership, and deadline interruption before
any public JavaScript API is frozen.

## Key decisions

- Go remains the process host and trusted computing base.
- JavaScript compiles to a serializable graph and returns structured decisions.
- `require("tinyidp").v1` is the primary API.
- Compiler and policy runtimes receive no ambient host modules.
- Production scripts reference host-owned resources by opaque name.
- Authorization and claims ship before general challenges and step-up.
- Policy infrastructure failure fails closed and unhealthy workers are replaced.
- Generated xgoja binaries require a provider package, TypeScript declarations,
  and generated-host smoke tests.

## Validation baseline

`go test ./... -count=1` passed from the current tiny-idp repository on
2026-07-10. The diary records exact commands, evidence, and investigation
failures.
