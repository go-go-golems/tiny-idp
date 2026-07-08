---
Title: Investigation diary
Ticket: TINYIDP-PROD-CONFIG-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological notes for the structured production configuration design ticket."
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: "Use this to resume or review the structured configuration design work."
WhenToUse: "Read before implementing TINYIDP-PROD-CONFIG-001 or updating its design guide."
---

# Diary

## Goal

Capture the investigation and design-writing work for turning tinyidp's flat development configuration into a structured production configuration contract.

## Step 1: Create the ticket and write the configuration guide

I created a dedicated docmgr ticket for structured production configuration and wrote an implementation-oriented design guide. The guide anchors the proposed schema and startup flow in the existing Glazed config chain, strict provider startup, embedded provider validation, domain client model, and SQLite store schema.

This step did not change production code. It produced an implementation plan that future code work can follow without rediscovering the current configuration boundaries.

### Prompt Context

**User prompt (verbatim):** "Create three docmgr tickets for:
1. structured configuration,
2. admin CLI,
3. real user/password storage.

For each ticket, write a detailed intern-oriented analysis/design/implementation guide with prose, bullets, pseudocode, diagrams/API/file references, store it in the ticket, and upload it to reMarkable."

**Assistant interpretation:** Create a separate docmgr ticket and long-form design guide for structured production configuration, with enough codebase evidence and implementation detail for an intern to execute it.

**Inferred user intent:** Turn the productization follow-up list from the strict Fosite/OIDF work into concrete, reviewable implementation tickets.

### What I did
- Created ticket `TINYIDP-PROD-CONFIG-001`.
- Added `design-doc/01-structured-production-configuration-design-and-implementation-guide.md`.
- Added this investigation diary.
- Inspected current config, serve, embedded options, domain, storage, and SQLite schema files.

### Why
- Production strict mode needs a typed configuration contract rather than ad hoc dev flags.
- The ticket should be executable independently from admin CLI and password-storage work.

### What worked
- Existing Glazed config/profile code provides a useful precedence model to preserve.
- Existing `embeddedidp.Options.Validate` is already a good final runtime validation boundary.
- Existing domain clients and key storage already support many production concepts.

### What didn't work
- N/A; this was documentation and ticket setup only.

### What I learned
- The current `oidc` section is intentionally flat and dev-oriented.
- Strict serve currently builds an in-memory store and development signing key, so product config needs a separate startup path.

### What was tricky to build
- The main design tension is compatibility: mock/local behavior should remain simple while strict production mode needs a richer schema. The guide resolves this by keeping the legacy `oidc` section and adding a separate `internal/appconfig` product model.

### What warrants a second pair of eyes
- The proposed split between config-owned bootstrap state and admin-owned operational state should be reviewed before implementation.

### What should be done in the future
- Implement `internal/appconfig` in parsing/validation/runtime phases.
- Add operator docs and config-backed conformance examples.

### Code review instructions
- Start with `design-doc/01-structured-production-configuration-design-and-implementation-guide.md`.
- Cross-check evidence references against `internal/sections/oidc`, `internal/cmds/serve.go`, and `pkg/embeddedidp/options.go`.
- Validate docs with `docmgr doctor --ticket TINYIDP-PROD-CONFIG-001 --stale-after 30`.

### Technical details
- No code changed in this step.
- Key references: `internal/sections/oidc/section.go`, `internal/cmds/profiles.go`, `internal/cmds/serve.go`, `pkg/embeddedidp/options.go`, `internal/domain/types.go`.
