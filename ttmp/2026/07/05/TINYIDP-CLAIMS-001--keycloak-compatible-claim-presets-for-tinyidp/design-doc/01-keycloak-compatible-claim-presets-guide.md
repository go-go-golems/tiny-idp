---
Title: Keycloak-compatible Claim Presets Guide
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
    - Path: internal/scenario/scenario.go
      Note: |-
        Current scenario model and extra-claim hooks
        Scenario ExtraClaims surface
    - Path: internal/scenario/seeded_users.go
      Note: |-
        Current seeded-user claims map that can already express nested claims manually
        Seeded-user keycloak preset schema target
    - Path: internal/server/jwt.go
      Note: |-
        ID token claim construction and scenario claim mutation
        ID token claim construction
    - Path: internal/server/userinfo.go
      Note: |-
        UserInfo claim construction from scenario/user state
        UserInfo claim construction
ExternalSources: []
Summary: Design and implementation guide for optional Keycloak-shaped authorization claim presets in tinyidp.
LastUpdated: 2026-07-05T17:45:00-04:00
WhatFor: Use when implementing optional Keycloak-compatible realm/client role claim helpers for tests that need authorization-claim compatibility.
WhenToUse: Read before changing scenario claim construction, seeded-user schemas, claim presets, or xgoja/appauth examples that consume Keycloak roles/groups.
---


# Keycloak-compatible Claim Presets Guide

## Executive summary

`tinyidp` does not need Keycloak claim presets for basic OIDC login. The current personal-inbox smokes can work with standard claims such as `sub`, `email`, `email_verified`, and `name`, plus simple custom claims in seeded users.

Keycloak-compatible claim presets become useful when a relying party tests authorization logic that expects Keycloak-shaped claims. Common examples are `realm_access.roles`, `resource_access.<client_id>.roles`, and group paths. Writing these nested structures by hand in every users file is possible today through `claims`, but it is repetitive and error-prone.

This ticket designs optional claim presets that expand concise seeded-user config into Keycloak-shaped nested claims. The feature should stay opt-in. It should not make tinyidp pretend to be Keycloak. It should provide a convenient compatibility fixture for apps whose claim-mapping code already expects Keycloak structures.

## Problem statement and scope

Current seeded users support arbitrary `claims` maps. That means this is already possible:

```yaml
claims:
  realm_access:
    roles: [admin, user]
  resource_access:
    personal-inbox-local:
      roles: [writer]
  groups:
    - /inbox-users
```

The problem is ergonomics and consistency. Different tests may encode the same Keycloak shapes differently. Some may put roles in `roles`, some in `groups`, some in `realm_access`, and some in `resource_access`. If an app specifically tests Keycloak role mapping, fixtures should be obvious and consistent.

This ticket covers claim preset expansion only. It does not add Keycloak protocol endpoints, admin APIs, import of realm JSON, or full Keycloak emulation.

## Current-state analysis

### Scenario extra claims

`internal/scenario/seeded_users.go` accepts a `Claims map[string]any` field and copies it into `Scenario.ExtraClaims`. This already supports arbitrary nested YAML objects.

### ID token and userinfo

`internal/server/jwt.go` and `internal/server/userinfo.go` merge scenario/user claims into token and userinfo responses. The exact helper names should be confirmed during implementation, but the important current behavior is that `Scenario.ExtraClaims` reaches both ID token and userinfo.

### Existing limitation

There is no reusable vocabulary for Keycloak-like role shapes. Users must know the exact nested JSON shape and repeat it in each fixture.

## Proposed model

Add an optional `keycloak` section to `SeededUser`:

```go
type SeededUser struct {
    Login string `json:"login" yaml:"login"`
    Sub   string `json:"sub" yaml:"sub"`
    Email string `json:"email" yaml:"email"`
    Name  string `json:"name" yaml:"name"`

    Claims   map[string]any      `json:"claims" yaml:"claims"`
    Keycloak *KeycloakClaimPreset `json:"keycloak" yaml:"keycloak"`
}

type KeycloakClaimPreset struct {
    RealmRoles   []string            `json:"realm_roles" yaml:"realm_roles"`
    ClientRoles  map[string][]string `json:"client_roles" yaml:"client_roles"`
    Groups       []string            `json:"groups" yaml:"groups"`
    PreferredUsername string         `json:"preferred_username" yaml:"preferred_username"`
}
```

Expanded claims:

```json
{
  "preferred_username": "alice",
  "groups": ["/inbox-users"],
  "realm_access": {
    "roles": ["user", "admin"]
  },
  "resource_access": {
    "personal-inbox-local": {
      "roles": ["writer"]
    }
  }
}
```

### YAML example

```yaml
users:
  - login: alice
    sub: user-alice-fixed
    email: alice@example.test
    name: Alice Inbox
    keycloak:
      preferred_username: alice
      realm_roles: [user, inbox-admin]
      client_roles:
        personal-inbox-local: [writer]
      groups:
        - /inbox-users
        - /engineering
```

### Merge rules

1. Start with user-derived base claims.
2. Expand `keycloak` preset claims into a map.
3. Merge explicit `claims` on top of preset claims.
4. Apply `omit_claims` last.

This lets users use the preset for common shape and override exact fields when needed.

Pseudocode:

```go
extra := map[string]any{}
if su.Keycloak != nil {
    merge(extra, expandKeycloakPreset(su.Login, su.Keycloak))
}
merge(extra, su.Claims)
applyEmailVerified(extra)
scenario.ExtraClaims = extra
scenario.OmitClaims = su.OmitClaims
```

### Conflict behavior

If both `keycloak.realm_roles` and `claims.realm_access` are set, explicit `claims.realm_access` wins. This is simpler than trying to merge nested role arrays. It also gives fixture authors an escape hatch.

## Proposed API details

### Go types

Add to `internal/scenario/seeded_users.go` or a sibling `keycloak_claims.go`:

```go
type KeycloakClaims struct {
    PreferredUsername string              `json:"preferred_username" yaml:"preferred_username"`
    RealmRoles        []string            `json:"realm_roles" yaml:"realm_roles"`
    ClientRoles       map[string][]string `json:"client_roles" yaml:"client_roles"`
    Groups            []string            `json:"groups" yaml:"groups"`
}

func ExpandKeycloakClaims(login string, kc *KeycloakClaims) map[string]any
```

### Expansion pseudocode

```go
func ExpandKeycloakClaims(login string, kc *KeycloakClaims) map[string]any {
    out := map[string]any{}

    preferred := strings.TrimSpace(kc.PreferredUsername)
    if preferred == "" {
        preferred = user.Normalize(login)
    }
    if preferred != "" {
        out["preferred_username"] = preferred
    }

    if len(kc.Groups) > 0 {
        out["groups"] = normalizeGroups(kc.Groups)
    }

    if len(kc.RealmRoles) > 0 {
        out["realm_access"] = map[string]any{
            "roles": uniqueSorted(kc.RealmRoles),
        }
    }

    if len(kc.ClientRoles) > 0 {
        ra := map[string]any{}
        for clientID, roles := range kc.ClientRoles {
            ra[clientID] = map[string]any{"roles": uniqueSorted(roles)}
        }
        out["resource_access"] = ra
    }

    return out
}
```

Do not sort if preserving author order matters for tests. If deterministic JSON snapshots are important, use first-seen de-duplication rather than alphabetical sort.

## Decision records

### Decision: Presets are opt-in seeded-user config, not global server mode

- **Context:** Not every app wants Keycloak-shaped claims. Basic OIDC tests should remain generic.
- **Options considered:** Global `--keycloak-claims` mode; seeded-user `keycloak:` block; only raw `claims` maps.
- **Decision:** Add an optional `keycloak:` block per seeded user.
- **Rationale:** Authorization roles are usually user-specific. Per-user config keeps fixtures explicit.
- **Consequences:** A user file can mix generic users and Keycloak-shaped users.
- **Status:** proposed

### Decision: Explicit `claims` override preset output

- **Context:** Fixture authors need escape hatches for unusual claim shapes.
- **Options considered:** Preset wins; explicit claims win; deep merge everything.
- **Decision:** Expand preset first, then apply explicit `claims`.
- **Rationale:** Explicit config should be the final authority.
- **Consequences:** Docs must explain that `claims.realm_access` replaces preset-generated `realm_access` if both are set.
- **Status:** proposed

### Decision: Do not import Keycloak realm JSON in this ticket

- **Context:** The first compatibility need is claim shape, not full Keycloak realm import.
- **Options considered:** Parse Keycloak realm JSON; define small tinyidp-native preset schema.
- **Decision:** Use a small tinyidp-native `keycloak:` schema.
- **Rationale:** Realm JSON is large, versioned, and includes many features tinyidp does not implement.
- **Consequences:** Users must translate realm roles into tinyidp config manually, but the target config is much smaller.
- **Status:** proposed

## Implementation phases

### Phase 1: model and expansion helper

- Add `KeycloakClaims` struct.
- Add `Keycloak *KeycloakClaims` to `SeededUser`.
- Implement `ExpandKeycloakClaims`.
- Unit-test empty, realm roles, client roles, groups, preferred username fallback, and mixed claims.

### Phase 2: seeded-user integration

- In `seededUserToScenario`, expand preset claims before `su.Claims`.
- Preserve existing `email_verified` behavior.
- Ensure `omit_claims` can omit preset-generated fields.

### Phase 3: docs and examples

- Add docs to README seeded-user section.
- Add `examples/users/roles-demo-users.yaml`.
- Add a help/reference section explaining the exact emitted JSON.

### Phase 4: integration validation

- Add a server test that logs in as a Keycloak-preset seeded user and verifies ID token and userinfo include:
  - `realm_access.roles`;
  - `resource_access.<client>.roles`;
  - `groups`;
  - `preferred_username`.
- Add a negative/override test proving explicit `claims` wins.

## Testing strategy

Unit tests:

```text
TestExpandKeycloakClaimsRealmRoles
TestExpandKeycloakClaimsClientRoles
TestExpandKeycloakClaimsGroups
TestSeededUserClaimsOverrideKeycloakPreset
TestOmitClaimsCanRemoveKeycloakPresetField
```

Server integration tests:

1. Start test server with registry containing a seeded Keycloak-preset user.
2. Complete authorize/token flow.
3. Decode ID token.
4. Call `/userinfo`.
5. Assert both surfaces carry expected nested claims.

Optional xgoja tests:

- Only add xgoja/appauth tests if a current example actually consumes Keycloak roles. Do not add preset usage to simple login smokes.

## Risks and open questions

- The word “Keycloak” can imply broader compatibility than this feature provides. Docs should say “Keycloak-shaped claims,” not “Keycloak emulation.”
- Some apps expect `groups` to be simple names, while Keycloak often emits path-like groups. The config should accept exactly what the fixture author writes.
- Some apps expect roles in `roles` or `scope` instead of Keycloak nested objects. Those should remain raw `claims` config, not part of this preset.
- If future tests need realm JSON import, that should be a separate ticket.

## References

- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/scenario/seeded_users.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/jwt.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/userinfo.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/keycloak/realm-personal-inbox.json`
