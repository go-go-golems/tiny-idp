---
Title: Generic Authorization Claim Presets Guide
Ticket: TINYIDP-CLAIMS-001
Status: active
Topics:
    - oidc
    - testing
    - identity
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Public seeded-user documentation for generic claim fields
    - Path: cmd/tinyidp/doc/pages/reference.md
      Note: Glazed help reference for generic claim fields
    - Path: examples/users/generic-claims-users.yaml
      Note: Provider-neutral seeded-user example
    - Path: internal/scenario/scenario.go
      Note: Current scenario model and generic ExtraClaims/OmitClaims hooks
    - Path: internal/scenario/seeded_users.go
      Note: Current seeded-user schema and conversion target for generic preset fields
    - Path: internal/server/token.go
      Note: ID token claim construction and ExtraClaims merge behavior
    - Path: internal/server/userinfo.go
      Note: UserInfo claim construction from scenario/user state
ExternalSources: []
Summary: Design and implementation guide for generic seeded-user authorization claim presets in tinyidp, without Keycloak-specific realm/client role shapes.
LastUpdated: 2026-07-05T18:20:00-04:00
WhatFor: Use when implementing generic seeded-user claim helpers such as groups, roles, tenant, locale, and preferred_username.
WhenToUse: Read before changing seeded-user schemas, claim preset fields, or appauth examples that consume role/group claims.
---


# Generic Authorization Claim Presets Guide

## Executive summary

This ticket is intentionally generic. `tinyidp` should not grow Keycloak-specific realm/client role structures for this work. The goal is to make common authorization claims easy to express in seeded users while preserving the existing raw `claims` map for advanced or provider-specific shapes.

The proposed feature adds short, provider-neutral fields to seeded users:

- `groups`
- `roles`
- `tenant`
- `preferred_username`
- `locale`

These fields expand into normal top-level OIDC/custom claims through the existing `Scenario.ExtraClaims` mechanism. They do not create nested provider-specific structures such as realm access maps. If a test needs unusual shapes, it can still use the explicit `claims` map.

## Problem statement and scope

Seeded users already support arbitrary claims:

```yaml
users:
  - login: alice
    claims:
      groups: [inbox-users]
      roles: [writer]
      tenant: personal
```

That is flexible, but it makes the most common role/group fixtures look more verbose than necessary. It also hides important authorization-relevant values under a generic map when scanning a users file.

This ticket introduces convenience fields only. It does not add:

- Keycloak realm/client role claim structures;
- provider-specific import/export;
- full authorization policy evaluation;
- server-side access control;
- new token endpoints or admin APIs.

## Current-state analysis

`internal/scenario/seeded_users.go` converts a `SeededUser` into a `Scenario`. During conversion it builds an `extra` map from `su.Claims`, then adds `email_verified` if configured. That `extra` map becomes `Scenario.ExtraClaims`.

`internal/server/token.go` merges `Scenario.ExtraClaims` into ID token claims. `internal/server/userinfo.go` mirrors those user-facing claims in `/userinfo`. This is the correct path for generic preset fields because no token-layer special case is required.

## Proposed seeded-user schema

Extend `SeededUser` with provider-neutral fields:

```go
type SeededUser struct {
    Login string `json:"login" yaml:"login"`
    Sub   string `json:"sub" yaml:"sub"`
    Email string `json:"email" yaml:"email"`
    Name  string `json:"name" yaml:"name"`

    Groups            []string `json:"groups" yaml:"groups"`
    Roles             []string `json:"roles" yaml:"roles"`
    Tenant            string   `json:"tenant" yaml:"tenant"`
    PreferredUsername string   `json:"preferred_username" yaml:"preferred_username"`
    Locale            string   `json:"locale" yaml:"locale"`

    Claims     map[string]any `json:"claims" yaml:"claims"`
    OmitClaims []string       `json:"omit_claims" yaml:"omit_claims"`
}
```

YAML example:

```yaml
users:
  - login: alice
    sub: user-alice-fixed
    email: alice@example.test
    name: Alice Inbox
    groups: [inbox-users, engineering]
    roles: [writer]
    tenant: personal
    preferred_username: alice
    locale: en-US
```

Expanded claims:

```json
{
  "groups": ["inbox-users", "engineering"],
  "roles": ["writer"],
  "tenant": "personal",
  "preferred_username": "alice",
  "locale": "en-US"
}
```

## Merge rules

The merge order should be explicit and test-backed:

1. Start with convenience fields (`groups`, `roles`, `tenant`, `preferred_username`, `locale`).
2. Apply explicit `claims` on top of convenience fields.
3. Apply `email_verified` compatibility fields.
4. Apply `omit_claims` later in token/userinfo construction as it already does.

This means explicit `claims` win over convenience fields. For example:

```yaml
users:
  - login: alice
    roles: [writer]
    claims:
      roles: [owner]
```

The emitted `roles` claim is `["owner"]`.

## Pseudocode

```go
func seededUserToScenario(su SeededUser) (Scenario, error) {
    login := user.Normalize(su.Login)
    if login == "" {
        return Scenario{}, fmt.Errorf("login is required")
    }

    extra := map[string]any{}
    if len(su.Groups) > 0 {
        extra["groups"] = cleanStringList(su.Groups)
    }
    if len(su.Roles) > 0 {
        extra["roles"] = cleanStringList(su.Roles)
    }
    if strings.TrimSpace(su.Tenant) != "" {
        extra["tenant"] = strings.TrimSpace(su.Tenant)
    }
    if strings.TrimSpace(su.PreferredUsername) != "" {
        extra["preferred_username"] = strings.TrimSpace(su.PreferredUsername)
    }
    if strings.TrimSpace(su.Locale) != "" {
        extra["locale"] = strings.TrimSpace(su.Locale)
    }

    for k, v := range su.Claims {
        extra[k] = v // explicit claims override presets
    }

    if ev := firstBool(su.EmailVerified, su.EmailVerifiedKebab); ev != nil {
        extra["email_verified"] = *ev
    }

    return Scenario{ExtraClaims: extra, OmitClaims: su.OmitClaims}, nil
}
```

`cleanStringList` should trim whitespace and drop empty values. It should preserve author order. Do not sort unless tests require deterministic reordering; preserving config order is easier to reason about.

## Decision records

### Decision: Use generic top-level claim fields only

- **Context:** The user explicitly requested no Keycloak realm-specific work and wants tinyidp kept generic.
- **Options considered:** Provider-specific nested claims; generic top-level fields; raw `claims` only.
- **Decision:** Add generic convenience fields only.
- **Rationale:** This supports common authorization tests without coupling tinyidp to one provider's vocabulary.
- **Consequences:** Users who need provider-specific structures can still write them under `claims` manually.
- **Status:** accepted

### Decision: Explicit `claims` override convenience fields

- **Context:** Convenience fields should reduce boilerplate, not remove control.
- **Options considered:** Convenience fields win; explicit claims win; reject conflicts.
- **Decision:** Explicit `claims` win.
- **Rationale:** The raw map is the escape hatch for unusual test fixtures.
- **Consequences:** Conflicting values are allowed and deterministic.
- **Status:** proposed

### Decision: Preserve author order for groups and roles

- **Context:** Some tests snapshot token/userinfo JSON or display ordered groups.
- **Options considered:** Sort values; preserve order; deduplicate.
- **Decision:** Trim/drop empty entries but preserve author order and duplicates initially.
- **Rationale:** Changing list contents can surprise fixture authors. Validation should be minimal.
- **Consequences:** If duplicate handling is desired later, add it as a separate explicit behavior.
- **Status:** proposed

## Detailed implementation task list

### Phase 0 — Scope correction and bookkeeping

- [x] Replace Keycloak-specific design language with generic claim preset language.
- [x] Record the user correction in the diary.
- [x] Update this task list to be precise enough for step-by-step execution.
- [ ] Commit the scope-correction docs.

### Phase 1 — Seeded-user schema

- [ ] Add `Groups []string` to `SeededUser` with JSON/YAML tags.
- [ ] Add `Roles []string` to `SeededUser` with JSON/YAML tags.
- [ ] Add `Tenant string` to `SeededUser` with JSON/YAML tags.
- [ ] Add `PreferredUsername string` to `SeededUser` with JSON/YAML tags.
- [ ] Add `Locale string` to `SeededUser` with JSON/YAML tags.
- [ ] Keep the existing `Claims map[string]any` field unchanged.

### Phase 2 — Claim expansion helper

- [ ] Add a helper to trim string claim values.
- [ ] Add a helper to trim string-list values while preserving order.
- [ ] Expand non-empty `groups` into `extra["groups"]`.
- [ ] Expand non-empty `roles` into `extra["roles"]`.
- [ ] Expand non-empty `tenant` into `extra["tenant"]`.
- [ ] Expand non-empty `preferred_username` into `extra["preferred_username"]`.
- [ ] Expand non-empty `locale` into `extra["locale"]`.

### Phase 3 — Merge semantics

- [ ] Apply convenience fields before explicit `Claims`.
- [ ] Preserve explicit `Claims` override behavior with a test.
- [ ] Preserve `email_verified` handling after explicit `Claims`.
- [ ] Preserve `OmitClaims` behavior without changing token/userinfo code.

### Phase 4 — Unit tests

- [ ] Add a test for top-level groups/roles/tenant/preferred_username/locale.
- [ ] Add a test proving explicit `claims` override top-level groups/roles.
- [ ] Add a test proving empty/whitespace list entries are dropped.
- [ ] Add a YAML load test covering generic top-level fields.
- [ ] Run `go test ./internal/scenario -count=1`.

### Phase 5 — Server-flow tests

- [ ] Add or update server flow test proving generic preset claims appear in ID token.
- [ ] Assert the same claims appear in `/userinfo`.
- [ ] Run `go test ./internal/server -count=1`.

### Phase 6 — Docs and examples

- [ ] Update README seeded-user documentation.
- [ ] Update Glazed reference page seeded-user documentation.
- [ ] Add or update an example users file with generic top-level fields.
- [ ] Avoid provider-specific realm/client role examples.

### Phase 7 — Final validation and diary

- [ ] Run `GOWORK=off go test ./... -count=1`.
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`.
- [ ] Update diary with exact command output.
- [ ] Update changelog and doc relations.
- [ ] Run `docmgr doctor --ticket TINYIDP-CLAIMS-001 --stale-after 30`.
- [ ] Commit implementation and docs.

## Risks and open questions

- Top-level `roles` and `groups` are common but not universal. The raw `claims` map remains the compatibility escape hatch.
- Trimming and dropping empty list entries is useful, but deduplication is not included to avoid surprising authors.
- `email_verified` currently overrides any `claims.email_verified`; this guide preserves that behavior for compatibility.

## References

- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/scenario/seeded_users.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/token.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/userinfo.go`
