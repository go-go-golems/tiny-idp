---
Title: Production administrator bootstrap design and implementation guide
Ticket: TINYIDP-INVITES-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-shared-two-apps/bootstrap.sql
      Note: Raw local bootstrap mechanism superseded by Phase 6
    - Path: ws://go-go-goja/pkg/gojahttp/auth/appauth/appauth.go
      Note: Defines the immutable OIDC identity key and deterministic application user ID
    - Path: ws://go-go-goja/pkg/gojahttp/auth/appauth/sqlstore/schema.go
      Note: Defines the application authorization records reconciled by bootstrap
    - Path: ws://go-go-goja/pkg/gojahttp/auth/audit/sqlstore/schema.go
      Note: Defines the durable audit row committed by bootstrap
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-21T16:52:09.428709821-04:00
WhatFor: ""
WhenToUse: ""
---


# Production administrator bootstrap design and implementation guide

## Purpose and scope

This follow-on replaces the local deployment's raw `bootstrap.sql` with a narrow operator command that establishes the first administrator of a generated go-go-goja application. TinyIDP owns authentication and produces an OIDC issuer and subject. The application owns its user record, organization, resource, and membership. The command reconciles the latter records; it does not create a TinyIDP account, verify a password, issue invitations, or provide general user administration.

The initial implementation belongs in go-go-goja because the affected tables and authorization rules are go-go-goja `appauth` contracts. The shared TinyIDP Compose example consumes the resulting generated-host command and supplies its local fixture identity. A later k3s deployment can run the same command as a controlled Job with its database DSN supplied through a mounted secret.

## Existing system

The generated host normalizes every successful OIDC identity into an application user. That intentionally grants no organization role. An application therefore begins with no principal authorized to issue organization invitations or inspect organization audit data.

The local stack currently bypasses this first-administrator problem with `examples/tinyidp-shared-two-apps/bootstrap.sql`. The script knows table names, repeats application normalization logic, and hardcodes the derived user ID. It works as a local fixture, but it is not an adequate production operation because application invariants are not represented as code and conflicts can be overwritten silently.

```text
TinyIDP                         generated goja application
--------                        --------------------------
issuer + immutable subject ---> app user + external identity
                                      |
operator bootstrap ------------------+--> tenant / org resource
                                           |
                                           +--> admin membership
                                           +--> audit record
```

## Command contract

The generated binary exposes this Glazed command when its runtime plan mounts the hostauth `operator` command set:

```text
generated-oidc-host-auth operator bootstrap-admin \
  --db-driver postgres \
  --db-dsn-file /run/secrets/appauth-dsn \
  --issuer https://idp.example.test \
  --subject 01J... \
  --email operator@example.test \
  --display-name "Deployment Operator" \
  --organization-id primary \
  --organization-slug primary \
  --organization-name "Primary Organization"
```

The database secret is read from a file. The command must not require a DSN containing credentials in process arguments. SQLite is supported for focused tests and single-process development; PostgreSQL is the production path. `--apply-schema` is explicit and defaults to false in the operator command.

The immutable identity key is `(issuer, subject)`. Email and display name are mutable metadata. The resulting application user ID uses the same deterministic `appauth.OIDCUserID` calculation as the normal callback path. The only role created by this command is the literal `admin` role for the requested organization.

## Reconciliation invariants

One database transaction performs all authoritative changes and appends the audit record.

- Empty database state creates the user, external-identity binding, tenant, organization resource, and admin membership.
- Repeating the same request restores disabled or revoked desired records and otherwise produces the same state without duplicates.
- Existing mutable email, display name, organization slug, and organization name are reconciled to the requested values.
- An issuer/subject already bound to a different application user is a conflict.
- A deterministic user ID already assigned to a different issuer/subject is a conflict.
- An organization ID already represented by a resource belonging to another tenant is a conflict.
- A slug already owned by another tenant is a conflict surfaced by the database constraint.
- No existing identity binding is reassigned and no unrelated membership is removed.
- Invalid or incomplete input fails before opening a transaction.

Pseudocode:

```text
validate and normalize request
userID = OIDCUserID(issuer, subject)

begin transaction
  lock identity binding and deterministic user rows
  reject conflicting ownership
  upsert desired application user
  insert immutable external identity binding
  upsert desired tenant
  upsert org resource with tenantID = organizationID
  upsert active admin membership
  insert operator.bootstrap_admin audit record
commit
return reconciled identifiers and whether each record changed
```

The audit event contains identifiers and requested non-secret metadata. It never contains the database DSN, cookies, OIDC tokens, passwords, invitation tokens, or capability tokens. Its actor is an operator identity supplied as `--operator-id`, defaulting to `deployment-operator`, because this operation runs outside an authenticated browser session.

## Package and API design

The core reconciler is a Go API under `pkg/gojahttp/auth/appauth/adminbootstrap`. It receives `*sql.DB`, a declared dialect, a clock, and a typed request. Keeping the transaction here ensures that the appauth rows and audit row commit together. The package uses the schemas owned by `appauth/sqlstore` and `audit/sqlstore`; it does not expose raw SQL to JavaScript.

```go
type Request struct {
    Issuer, Subject string
    Email, DisplayName string
    OrganizationID, OrganizationSlug, OrganizationName string
    OperatorID string
}

type Result struct {
    UserID, OrganizationID, Role string
}

func (r *Reconciler) BootstrapAdmin(ctx context.Context, request Request) (Result, error)
```

The hostauth provider registers an `operator` command-set provider beside its JavaScript module. The command owns database opening, optional schema application, secret-file reading, input decoding, cleanup, and Glazed result output. Adding it to an application remains explicit in `xgoja.yaml`; merely importing the hostauth provider does not expose an operational command.

## Failure and retry behavior

Validation errors and identity conflicts are permanent and return nonzero without mutation. Connection failures, serialization failures, and context cancellation also roll back. A process crash before commit leaves no partial bootstrap; a crash after commit can be retried safely. If the command reports success, the audit record and authorization state are in the same commit.

This command does not silently adapt an old schema. Schema creation is either performed through the existing current schema functions when `--apply-schema` is requested or assumed to have been completed by the serving application/migration job.

## Tests and acceptance

Focused SQLite tests cover creation, exact replay, revoked-membership restoration, disabled-user restoration, identity collision, organization collision, rollback, validation, and the audit record. PostgreSQL query construction is exercised by the local Compose acceptance path against PostgreSQL 17.

The shared stack replaces the `postgres:17-alpine` SQL utility job with the generated goja-auth image running `operator bootstrap-admin`. Acceptance runs the command twice and checks:

- both invocations exit successfully;
- one user and one external identity exist for the issuer/subject;
- one active admin membership exists;
- the normal login resolves to the same application user;
- the administrator can issue/accept the invitation flows already covered by Phase 5;
- no raw database credential appears in logs or committed files.

## Implementation sequence

1. Add the typed transactional reconciler and focused SQLite tests.
2. Add the Glazed command and hostauth command-set registration tests.
3. Mount `operator` in example 21's runtime plan and verify generated CLI help.
4. Change the shared Compose bootstrap service to run the generated command and remove `bootstrap.sql`.
5. Run focused tests, `go build ./...`, `go test ./...`, then the local Compose smoke and browser acceptance suites.
6. Record commits, failures, validation evidence, and review instructions in the ticket diary.

## File map

- `go-go-goja/pkg/gojahttp/auth/appauth/appauth.go` — identity model and deterministic OIDC user ID.
- `go-go-goja/pkg/gojahttp/auth/appauth/sqlstore/schema.go` — application authorization schema.
- `go-go-goja/pkg/gojahttp/auth/audit/sqlstore/schema.go` — durable audit schema.
- `go-go-goja/pkg/xgoja/providers/hostauth/hostauth.go` — hostauth provider registration seam.
- `go-go-goja/examples/xgoja/21-generated-host-auth/xgoja.yaml` — explicit generated-command mount.
- `tiny-idp/examples/tinyidp-shared-two-apps/compose.yaml` — local production-shaped invocation.
- `tiny-idp/examples/tinyidp-shared-two-apps/bootstrap.sql` — superseded raw mechanism to remove.

## Executive Summary

<!-- Provide a high-level overview of the design proposal -->

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
