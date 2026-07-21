---
Title: 'Durable unique display names: analysis, design, and implementation guide'
Ticket: TINYIDP-UNIQUE-DISPLAY-NAMES-001
Status: active
Topics:
    - tiny-idp
    - oidc
    - goja
    - identity
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-shared-two-apps/open-signup.js
      Note: Shared demo signup policy.
    - Path: repo://internal/fositeadapter/scripted_signup.go
      Note: Native transactional signup and themed errors.
    - Path: repo://internal/gojamodules/tinyidp/module.go
      Note: Closed JavaScript signup effect boundary.
    - Path: repo://pkg/idpaccounts/accounts.go
      Note: Account preparation and atomic credential commit.
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Durable store contracts.
    - Path: repo://pkg/sqlitestore/store.go
      Note: SQLite user storage implementation.
ExternalSources: []
Summary: A script-selected signup policy that prevents duplicate display names with an atomic native reservation, bounded lookup capability, and themed recovery paths.
LastUpdated: 2026-07-21T19:39:02.099353707-04:00
WhatFor: Implement optional, production-safe unique display names in scripted TinyIDP signup.
WhenToUse: Use when a relying application needs display names to act as unique public handles without turning JavaScript into the authority for durable identity state.
---


# Durable unique display names: analysis, design, and implementation guide

## Executive Summary

TinyIDP currently stores a profile `User.Name`, but it does not define that
field as unique. The shared Message Desk and Goja Auth demo asks for a
different rule: a person must not be able to create a second account using an
already claimed display name. The rule must be correct under concurrent
browser requests, usable by a Goja signup workflow, and recoverable in the
browser with the application-selected TinyIDP theme.

This ticket adds a small, reusable primitive rather than a Message Desk
special case. A script opts in at `ctx.commit.signup(...)` with
`uniqueDisplayName: true`. Native Go validates the effect, normalizes the
display name, and reserves its normalized key in the same store transaction
that creates the account. A bounded capability lets the earlier form step
give a useful field-level answer, but it is explicitly advisory: the
transactional reservation decides the race.

## Problem Statement

`displayName` is collected by `examples/tinyidp-shared-two-apps/open-signup.js`
and becomes `idpstore.User.Name` through the scripted native commit boundary
in `internal/fositeadapter/scripted_signup.go`. Today the account service
trims the name and stores it alongside the user. The normal user indexes are
only login and OIDC subject. Consequently two accounts can use the same name.

Checking in JavaScript alone would be incorrect. Two requests can both ask
whether `Manuel` is available before either writes. A configuration flag alone
would also be too rigid: some relying applications deliberately allow shared
or cosmetic display names. The primitive must be optional per scripted signup
workflow, while the invariant must be native and durable once selected.

Display names and logins remain distinct concepts:

| Concept | Current TinyIDP field | Purpose | Uniqueness |
| --- | --- | --- | --- |
| Login | password credential login / email | authenticate an account | always unique |
| OIDC subject | `User.Sub` | stable protocol principal | always unique |
| Display name | `User.Name` | human-facing profile label | optional, chosen by signup policy |

This ticket does not redefine every existing `User.Name` as a global handle,
nor does it introduce an application organization-membership system.

## Proposed Solution

The implementation uses four cooperating layers.

```text
Browser form                 Goja lambda               native TinyIDP
-------------                -----------               -------------
displayName ──submit──> identity.displayName.lookup ─> read reservation
       ^                         │                               │
       │ field error             └── present same form <─────────┘
       │
       └── password submit ──> ctx.commit.signup({uniqueDisplayName:true})
                                      │
                                      v
                           native effect validation
                                      │
                                      v
                  transaction: reserve normalized key + user + credential
                                      │
                     duplicate ──────┴────── success
                         │                     │
                         v                     v
                themed retry page         browser session/OIDC continue
```

### 1. Canonical key and reservation

The account layer owns `NormalizeDisplayName`. It trims outer whitespace,
collapses internal Unicode whitespace, applies Unicode case folding and NFC
normalization, and rejects an empty result. Thus `" Manuel "` and `"manuel"`
refer to one key. The exact original `User.Name` remains the presentation
value; only the derived key is reserved.

`idpstore.DisplayNameStore` exposes two deliberate operations:

```go
LookupDisplayName(ctx, key string) (idpstore.DisplayNameClaim, error)
ReserveDisplayName(ctx, key, userID string) error // ErrDisplayNameTaken
```

The SQLite implementation stores claims in a separate `display_name_claims`
table keyed by the normalized key. The memory implementation uses the same
keyed map inside its transaction snapshot. The reservation occurs before the
user and credential write, inside the existing `Store.Update` transaction.
The database primary key—not an earlier query—is the concurrency authority.

Existing users are intentionally not silently converted to claims by a schema
migration. Enabling this optional policy must call an explicit startup
reconciliation which computes keys, detects historical collisions, and fails
with actionable diagnostics rather than arbitrarily assigning a name. The
initial shared demo has controlled seed data and will reconcile at startup.

### 2. Script-selected native effect

`tinyidp.v1` gains an optional boolean `uniqueDisplayName` on
`ctx.commit.signup`:

```javascript
return ctx.commit.signup({
  login: ctx.input.email,
  displayName: ctx.input.displayName,
  uniqueDisplayName: true,
  password: ctx.secret.password,
  passwordConfirmation: ctx.secret.passwordConfirmation,
});
```

The module emits this as part of the closed `createLocalIdentity` effect
payload. `commitScriptedSignup` decodes only that known boolean and passes it
to `idpaccounts.PrepareCreate`. JavaScript cannot name a table, supply a
normalized key, or invoke a store method. Programs that omit the boolean keep
the current non-unique display-name behavior.

### 3. Bounded preflight capability

The Goja capability is named `identity.displayName.lookup@v1`. It accepts
only `{displayName: string}` and returns only `{available: boolean}`. It does
not reveal user IDs, emails, disabled state, or a directory of names.

The shared signup program invokes it in `signup.submitted`. If unavailable,
it returns `ctx.present.form(...)` for the same handler and attaches the
standard display-name field error. If available, it starts the email challenge.
The capability is registered in the program contract and rejected at
production startup unless the provider has the native identity lookup service.

### 4. Browser recovery

Preflight failure is attached to the display-name field, preserving public
values and allowing correction. A commit-time collision can happen only after
the workflow has advanced to the password page, so the native provider renders
a specific themed global error. It explains that the display name was claimed
while signup was in progress and directs the user to restart with another
name. It must not falsely call a password or email problem.

## Design Decisions

1. **Native reservation is authoritative.** JavaScript is a policy author and
   UI coordinator; it is never a database transaction boundary.
2. **The policy is opt-in at the commit effect.** This keeps profile names
   flexible for applications that do not need handles.
3. **Lookup is deliberately coarse.** A boolean supports a good signup UI
   without exposing an account-enumeration API.
4. **Use a separate claims table.** A unique column on `users` would make the
   restriction global and would make later policy changes unsafe.
5. **Historical data requires an explicit reconciliation.** Failing on legacy
   collisions is safer than picking a winner automatically.

## Alternatives Considered

| Alternative | Rejection reason |
| --- | --- |
| JavaScript checks only | has a check-then-write race and is bypassable by another workflow |
| unique index on `users.name` | makes the optional policy mandatory for all applications |
| configuration-only allowlist/regex | cannot express application-specific workflow choices and does not itself persist a claim |
| application-side uniqueness only | splits one identity namespace across relying applications and cannot protect direct TinyIDP signup |
| treat display name as email/login | changes profile semantics and excludes non-email public handles |

## Implementation Plan

1. Map the present flow and commit the guide/diary.
2. Add normalization, typed duplicate error, claim store contract, memory and
   SQLite implementations, migration, and atomic account tests.
3. Add `identity.displayName.lookup@v1`, executor capability injection, and
   production validation.
4. Extend the closed Goja signup effect and wire the shared program to opt in.
5. Render both error paths through the workflow renderer.
6. Exercise unit, SQLite, provider, and Playwright journeys; record exact
   results and publish the completed guide.

### Pseudocode: authoritative commit

```go
prepared := accounts.PrepareCreate(request)
store.Update(ctx, func(tx TxStore) error {
    if prepared.RequireUniqueDisplayName {
        key := accounts.NormalizeDisplayName(prepared.User.Name)
        if err := tx.ReserveDisplayName(ctx, key, prepared.User.ID); err != nil {
            return err // ErrDisplayNameTaken rolls the entire transaction back
        }
    }
    tx.PutUser(ctx, prepared.Login, prepared.User)
    tx.PutPasswordCredential(ctx, prepared.Credential)
    return nil
})
```

## Open Questions

1. Should the product eventually expose a separate immutable `handle` instead
   of making a mutable display name act as a public unique name? This ticket
   deliberately does not add that new user concept.
2. Reconciliation needs a clear production operator command and diagnostic
   format before a deployment with pre-existing users enables the policy.
3. We should rate-limit or otherwise monitor repeated lookup requests because
   even a boolean can be used for low-grade name enumeration.

## References

- `examples/tinyidp-shared-two-apps/open-signup.js` — shared demo policy.
- `internal/gojamodules/tinyidp/module.go` — closed JavaScript commit API.
- `internal/fositeadapter/scripted_signup.go` — native effect transaction and
  themed error rendering boundary.
- `pkg/idpaccounts/accounts.go` — account preparation and credential creation.
- `pkg/idpstore/interfaces.go` and `pkg/sqlitestore/store.go` — durable store
  contract and SQLite implementation.
