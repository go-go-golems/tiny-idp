---
Title: Research diary
Ticket: TINYIDP-PLUGIN-001
Status: active
Topics:
    - architecture
    - auth
    - jitsi
    - operations
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp/main.go
      Note: Production parser middleware composition introduced in commit 294e343
    - Path: repo://internal/cmds/serve_production.go
      Note: Production host now decodes the reusable section in commit 294e343
    - Path: repo://internal/sections/production/section.go
      Note: Canonical production Glazed section introduced in commit 294e343
ExternalSources: []
Summary: Chronological record of the preliminary TinyIDP plugin architecture research.
LastUpdated: 2026-07-23T16:32:32.222501884-04:00
WhatFor: ""
WhenToUse: ""
---


# Research diary

## Goal

Identify the configuration, browser identity, Goja, lifecycle, deployment, and
observability decisions required before writing the full plugin system design.

## 2026-07-23 — Protocol and prior-ticket review

- Reviewed `TINYIDP-JITSI-001` and `TINYIDP-JITSI-K3S-001`.
- Confirmed that an embedded plugin can replace the standalone OIDC adapter,
  but not Prosody.
- Rechecked current Jitsi token documentation. Prosody validates shared-secret
  or public-key JWTs on the XMPP connection and again against the MUC room.
- Rechecked the current `jitsi-contrib/jitsi-oidc-adapter`; it remains a useful
  behavioral reference and separately deployable fallback.

## 2026-07-23 — Configuration inspection

- Read the TinyIDP command construction, reusable OIDC section, profile
  middleware, config plan, and production server wiring.
- Confirmed the intended precedence:
  `defaults < profiles < config < environment < arguments < flags`.
- Verified from local Glazed source tests that a section prefix participates in
  environment names, allowing `TINYIDP_JITSI_*`.
- Found that `serve-production` does not currently use the parser/profile
  configuration used by `serve-dev` and `print-config`.
- Confirmed that production already uses protected secret-file references,
  which is the appropriate precedent for plugin secrets.

## 2026-07-23 — Scriptability and runtime inspection

- Read `pkg/idpscript`, `pkg/idpsignup`, `pkg/embeddedidp`, and the internal
  browser-session implementation.
- Confirmed that bounded, versioned Goja capabilities fit plugin policy.
- Identified browser identity as the principal missing host service: a handler
  mounted next to the provider cannot currently resolve the active TinyIDP
  browser identity or initiate/resume login through a public API.
- Recorded native browser-identity and embedded OIDC/PKCE approaches.

## 2026-07-23 — Loading and operations

- Compared compiled-in Go registration, standard-library shared objects,
  HashiCorp subprocess plugins, JavaScript-only plugins, and standalone
  adapters.
- Selected compiled-in registration as the exploratory recommendation.
- Confirmed structured logging, durable audit, readiness, and internal atomic
  metric snapshots, but no general production metrics exporter.
- Drafted scoped routing, readiness, secret resolution, Goja policy, and
  observability boundaries for review before the full design.

## Current conclusion

The first full-design decision should be the browser identity seam. The rest of
the plugin structure can be built cleanly around a compiled-in registry and
plugin-owned Glazed sections.

## 2026-07-23 — Full design

- Promoted the exploratory findings into an intern-facing analysis, design,
  and implementation guide.
- Selected a host-owned OIDC relying-party broker for version one instead of
  exposing browser session internals. The broker uses authorization code,
  PKCE S256, nonce, durable one-time transactions, ID-token validation, and
  userinfo.
- Selected an in-process provider-backed HTTP transport for server-side token
  and userinfo calls so the pod does not depend on its own public ingress.
- Defined the compiled-in descriptor, definition, prepared, runtime, registry,
  route, service, readiness, and lifecycle boundaries.
- Defined the Jitsi configuration, claim, Goja policy, secret, audit,
  observability, deployment, failure, and test contracts.
- Added task-level implementation phases through GitOps and media-connected
  browser validation.

## 2026-07-23 — Validation and delivery

- Validated the guide frontmatter with `docmgr validate frontmatter`.
- Validated the complete ticket with
  `docmgr doctor --ticket TINYIDP-PLUGIN-001`; all checks passed.
- The first upload attempt failed because the sandbox could not resolve the
  reMarkable cloud hosts. Repeated the same upload with approved network
  access.
- Uploaded the guide as
  `TINYIDP Plugin System and Jitsi Integration Guide.pdf` to
  `/ai/2026/07/23/TINYIDP-PLUGIN-001` on reMarkable.

## Step 1: Compose production configuration through Glazed

This step removed the production server's exceptional configuration path.
Production settings now live in a reusable `production` section, and
`serve-production` resolves profiles, configuration files, environment values,
arguments, and flags through the same ordered middleware chain as the other
primary commands.

The change also makes the security boundary inspectable: Glazed sees secret
file paths but has no fields capable of accepting the corresponding secret
contents. Focused tests prove value precedence and retain the winning source in
the field provenance log.

### Prompt Context

**User prompt (verbatim):** "Implement TINYIDP-PLUGIN-001, run locally with docker compose to confirm. Keep a detailed diary as you work."

**Assistant interpretation:** Implement every ticket phase in focused commits, preserve a detailed implementation record, and prove the resulting Jitsi integration in a local Compose environment.

**Inferred user intent:** Turn the reviewed plugin design into a professional, locally reproducible TinyIDP integration rather than leaving it as architectural documentation.

**Commit (code):** `294e343` — "feat(config): compose production through Glazed sources"

### What I did

- Added `internal/sections/production` with the canonical Glazed section,
  typed settings, decoder, and tests.
- Changed `serve-production` to compose and decode that section.
- Wired `serve-production` through `ProfileMiddlewaresFunc`,
  `ConfigFilePlanBuilder`, `AppName: "tinyidp"`, and the profile settings
  section.
- Changed the supported configuration-file shape to the documented
  `production:` section; no flat legacy adapter was added.
- Checked ticket tasks `p1s1` through `p1s4`.

### Why

- Plugin sections must exist before Glazed parses values, and production must
  use the same source chain for core and plugin configuration.
- Keeping secrets as file references prevents raw signing material from
  entering flags, environment values, parsed-field output, or provenance logs.

### What worked

- `go test ./internal/sections/production -count=1` passed.
- `go test ./internal/cmds -run 'Production|OwnerOnly' -count=1` passed.
- The pre-commit test, lint, Glazed analyzer, and UI analyzer gates passed.

### What didn't work

- The first focused test command failed before compilation:
  `open /home/manuel/.cache/go-build/...: read-only file system`. It succeeded
  after setting `GOCACHE=/tmp/tinyidp-plugin-go-cache`.
- The broad `go test ./internal/cmds` run reached an unrelated existing
  listener test and failed with
  `httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted`.
  The production-only test selection avoided claiming coverage from a test the
  sandbox could not execute.
- The first `git add` failed with
  `Unable to create '/home/manuel/code/wesen/go-go-golems/tiny-idp/.git/worktrees/tiny-idp/index.lock': Read-only file system`.
  The same scoped operation succeeded with approved repository access.

### What I learned

- The repository already had the correct source middleware; production simply
  bypassed it.
- Glazed records provenance on each `FieldValue.Log`, so source inspection does
  not require a parallel configuration system.

### What was tricky to build

- Section extraction changes configuration-file nesting while preserving
  command-line flag names. The decoder must use `production.Slug`, not the
  default section, or values silently decode as zero values.
- Required fields cannot be exercised by defaults-only tests, so the section
  tests focus on optional/defaulted fields and schema shape while production
  command tests retain the required-field assertions.

### What warrants a second pair of eyes

- Review any existing deployment configuration for flat production keys. The
  intended new contract is a `production:` mapping, with no compatibility
  loader.
- Confirm the Glazed built-in parsed-field output presents secret file paths
  at the desired operational sensitivity; it never reads file contents.

### What should be done in the future

- Plugin sections must be passed to the same command constructor before
  parsing; they must not read environment variables independently.

### Code review instructions

- Start at `internal/sections/production/section.go`, then inspect the
  `serve-production` construction in `cmd/tinyidp/main.go`.
- Run:
  `GOCACHE=/tmp/tinyidp-plugin-go-cache go test ./internal/sections/production ./internal/cmds -run 'Production|OwnerOnly' -count=1`.

### Technical details

```text
defaults < profiles < config < environment < arguments < flags

config key: production.addr
environment: TINYIDP_ADDR
flag:        --addr
```
