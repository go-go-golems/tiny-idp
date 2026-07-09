---
Title: Use tiny-idp as an OIDC Identity Provider for Jitsi Meet
Ticket: TINYIDP-JITSI-001
Status: active
Topics:
    - oidc
    - jitsi
    - authentication
    - research
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-09T12:17:47.798688144-04:00
WhatFor: ""
WhenToUse: ""
---

# Use tiny-idp as an OIDC Identity Provider for Jitsi Meet

## Overview

Evaluates whether **tiny-idp** (our minimal fosite-shaped OIDC provider) can serve as the identity
provider for **Jitsi Meet**, and specifies how.

**Verdict:** feasible — but Jitsi has **no native OIDC login**, so a small **OIDC→Jitsi-JWT translation
adapter** (recommended: `jitsi-contrib/jitsi-oidc-adapter`) sits between the IdP and Jitsi. tiny-idp
already exposes exactly the OIDC surface the adapter needs (discovery + authorize + token + userinfo),
verified live; **no new tiny-idp features are required.** Prosody validates the adapter-minted **HS256**
Jitsi JWT via a shared `app_secret`, so tiny-idp's RS256/JWKS are never used by Jitsi (which cannot read
JWKS anyway).

**Deliverables:** the design/implementation guide (`design-doc/01-...md`), two verified experiments
(`scripts/01`, `scripts/02`), 15 captured sources (`sources/web/`), and the investigation diary
(`reference/01-...md`). Status: **research & design complete**; implementation phases are listed as
follow-ups in `tasks.md`.

## Key Links

- **Design/implementation guide:** [design-doc/01-...](./design-doc/01-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet-analysis-design-and-implementation-guide.md)
- **Investigation diary:** [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- **Experiments:** [scripts/01-oidc-smoke.sh](./scripts/01-oidc-smoke.sh), [scripts/02-oidc-to-jitsi-jwt.py](./scripts/02-oidc-to-jitsi-jwt.py)
- **Sources:** [sources/web/](./sources/web/) (Jitsi token spec, handbook, adapters)

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- oidc
- jitsi
- authentication
- research
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
