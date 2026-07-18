---
Title: Multi-Account Browser Sessions and Account Chooser Analysis Design and Implementation Guide
Ticket: TINYIDP-ACCOUNT-CHOOSER-001
Status: active
Topics:
    - identity
    - oidc
    - security
    - architecture
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Authorization prompt state machine to extend
    - Path: repo://internal/fositeadapter/session.go
      Note: Current active browser-session creation and validation
    - Path: repo://pkg/idpstore/types.go
      Note: Durable session model and future browser-context types
    - Path: repo://pkg/idpui/types.go
      Note: Presentation contract for chooser extension
ExternalSources:
    - https://openid.net/specs/openid-connect-core-1_0-18.html
    - https://openid.net/specs/openid-connect-rpinitiated-1_0.html
Summary: Design for a standard OIDC account chooser and provider-owned multiple browser sessions.
LastUpdated: 2026-07-14T22:00:00Z
WhatFor: Makes tiny-idp a reusable, secure multi-account identity toolbox.
WhenToUse: Before changing sessions, prompts, renderers, embedding, or logout.
---


# Multi-Account Browser Sessions and Account Chooser

## Purpose

tiny-idp currently maps one random HttpOnly cookie handle to one durable
`idpstore.Session`. `internal/fositeadapter/session.go:28-68` creates and
validates that mapping; `provider.go:453-503` silently uses it unless
`prompt=login` or `max_age` requires new authentication. A Message Desk logout
ended only its RP session, so a subsequent authorization reused the prior IdP
session. The provider needs a general account-selection feature, not an
application-specific workaround.

The target is the experience expected from a mature identity provider: select
among accounts with current sessions, use another account, remove a remembered
account, sign out of one application, or sign out globally. The target is not
to turn a browser-visible account list into authentication evidence.

## Foundation

| Entity | Meaning | Evidence of authentication? |
| --- | --- | --- |
| RP session | Application session after ID-token verification | Only for that RP |
| IdP session | Durable provider session for one user | Yes, after server validation |
| Browser context | Opaque selector cookie for a browser profile | No |
| Remembered entry | Context-to-session membership with safe label | No |
| Active session | Fresh active cookie bound to a chosen valid session | Yes |

OpenID Connect Core defines `prompt=select_account`: the authorization server
asks the user to select among accounts with current sessions. A silent request
that cannot obtain a choice returns `account_selection_required`. Core also
requires fresh authentication for `prompt=login`, and no UI for `prompt=none`.
Fosite already admits `select_account` as an allowed prompt. RP-Initiated
Logout is still the correct operation for complete provider logout.

## Existing boundaries

`pkg/idpstore/types.go:161-173` has server-side user, authentication time,
expiry, AMR, and revocation data. `SessionStore` has only Create/Get/Revoke
(`interfaces.go:77-81`), and SQLite stores serialized session rows in
`migrations/001_schema.sql:10`; it lacks browser contexts and membership.

`pkg/idpui/types.go` keeps renderer code presentation-only. Its interaction
model currently has login and consent prompts. Extend this public model rather
than letting a renderer construct redirect state or choose a subject.

## Architecture

```text
opaque context cookie -> browser_contexts
                         -> remembered_entries -> idp_sessions -> users

authorize(prompt=select_account)
  -> context-bound InteractionRecord -> provider UI chooser
  -> opaque entry POST -> atomic server validation
  -> fresh active session handle -> existing consent/code flow
```

The context cookie contains only a random handle. Context, entry, and session
handles are stored as keyed hashes. A cookie list of account IDs is rejected:
signing prevents mutation but cannot prove a selected account is current,
enabled, unrevoked, or fresh enough.

## Proposed public APIs

```go
type BrowserContextStore interface {
 CreateBrowserContext(context.Context, BrowserContext) error
 GetBrowserContext(context.Context, contextHash []byte) (BrowserContext, error)
 ListRememberedBrowserSessions(context.Context, contextHash []byte, now time.Time) ([]RememberedBrowserSession, error)
 ActivateRememberedSession(context.Context, contextHash, entryHash, newHandleHash []byte, now time.Time) (Session, User, error)
 RemoveRememberedBrowserSession(context.Context, contextHash, entryHash []byte, at time.Time) error
 RevokeBrowserContext(context.Context, contextHash []byte, at time.Time) error
}

type AccountChooserConfig struct {
 Enabled bool
 ContextCookieName string
 ContextTTL time.Duration
 MaxRememberedAccounts int
 RememberOnPasswordLogin bool
 DisplayLabel func(idpstore.User) (string, error)
}
```

`ActivateRememberedSession` is intentionally atomic. It verifies context
membership, removal, session expiry/revocation, and user state, then writes a
fresh active-cookie handle. Existing storage never retains an old raw handle,
so it must not be reissued. `embeddedidp.Options` gains this configuration and
advertises `select_account` in discovery only when enabled.

`idpui.InteractionPage` gains `AccountChooserPrompt`, holding unique opaque
entry values, bounded safe labels, a selection field, and the provider-defined
**Use another account** action. Renderers submit fields; they cannot decide
membership or write cookies. `idpuitest` gains conformance checks for labels,
keyboard interaction, denial, and forged choices.

## Authorization algorithm

```text
read active session and browser context
if prompt=none and selection is needed: return account_selection_required
if prompt=login or max_age fails: render password login
else if prompt includes select_account:
  list valid remembered entries
  if empty: render login / use-another
  else: persist interaction bound to browser context; render chooser
on choice POST:
  validate CSRF, interaction digest, context binding, action, opaque entry
  atomically activate valid remembered session with fresh handle
  run existing consent and authorization-code completion
```

`prompt=login` always wins over chooser reuse. `prompt=none` never renders a
chooser. A disabled, expired, revoked, or removed entry never yields a token.

## Decisions

### Decision: standard `prompt=select_account`

- **Context:** RPs need a portable account-selection request.
- **Options considered:** private parameter, always chooser, OIDC Core prompt.
- **Decision:** implement and conditionally advertise the Core prompt.
- **Rationale:** Core defines both selection and silent error behavior.
- **Consequences:** test combinations with `none`, `login`, consent, and `max_age`.
- **Status:** proposed.

### Decision: durable context records

- **Context:** Labels are privacy-sensitive and sessions change state.
- **Options considered:** signed account-list cookie, one cookie per account, server records.
- **Decision:** opaque context cookie plus durable context/entry records.
- **Rationale:** permits expiry, revocation, removal, privacy policy, and atomic validation.
- **Consequences:** new migrations, retention, audit events, and backups.
- **Status:** proposed.

## Phases and tasks

1. **Research/contracts:** threat model; store parity matrix; label/privacy, cookie, retention, audit, migration decisions.
2. **Storage:** public types; SQLite migration; memory store; atomic activation; expiry/removal/race tests.
3. **Lifecycle:** context creation/rotation on password login, attach sessions, removal/global revocation, maintenance and audit events.
4. **OIDC:** standard prompt parsing, context-bound interactions, standard errors, discovery metadata, prompt combination tests.
5. **Toolbox/UI:** `idpui` chooser model, renderer conformance, default renderer, `embeddedidp` config, stylable example host.
6. **Assurance:** scenarios for two accounts, use-another, forged entry, copied context, disablement, replay, fuzzing, analyzers, browser accessibility, backup/rollback.

## Test matrix

| Scenario | Required behavior |
| --- | --- |
| Two valid remembered sessions | Choice issues a code for selected user. |
| `select_account none` | `account_selection_required`, no UI or cookie change. |
| `login select_account` | Password reauthentication wins. |
| Forged/cross-context entry | No subject disclosure or code. |
| Concurrent select/remove | One terminal outcome, no resurrection. |
| Global logout | Context/active session revoke; exact registered redirect. |

## Risks and questions

Remembered labels can disclose prior use on a shared browser. Default to opt-in
remembering, short TTL, bounded safe labels, explicit removal, and no login or
email display unless the host deliberately chooses it. Decide context restart
behavior, default label, whether removal revokes membership only or IdP
session, and which front/back-channel logout work belongs in the first release.

## References

- `sources/01-openid-connect-core.md` — `select_account`, prompt, errors.
- `sources/02-openid-connect-rp-initiated-logout.md` — logout/redirect rules.
- `sources/03-openid-connect-session-management.md` — session lifecycle.
- `sources/04-openid-connect-prompt-create.md` — related prompt extension.
