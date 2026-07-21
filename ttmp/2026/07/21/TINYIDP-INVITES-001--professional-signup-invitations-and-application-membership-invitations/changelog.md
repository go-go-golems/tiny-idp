# Changelog

## 2026-07-21

- Initial workspace created


## 2026-07-21

Completed the code-backed intern design for separate TinyIDP account-creation and go-go-goja organization-membership invitations; documented existing primitives, production gaps, trust boundaries, APIs, transactions, browser saga, phases, and acceptance tests.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpinvite/durable.go — Existing professional TinyIDP invitation primitive that anchors the proposed scope

## 2026-07-21

Validated the ticket, committed the design as ae1637d, and uploaded the design plus diary bundle to reMarkable at /ai/2026/07/21/TINYIDP-INVITES-001.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/21/TINYIDP-INVITES-001--professional-signup-invitations-and-application-membership-invitations/design-doc/01-professional-invitation-core-and-application-membership-invitation-design-and-implementation-guide.md — Published intern-oriented invitation architecture and implementation guide

## 2026-07-21

Completed Phases 1–5: activated TinyIDP durable signup invitations, added atomic and identity-bound application membership acceptance, preserved opaque pending invitations through OIDC registration, bootstrapped the local application authority explicitly, and passed the seven-stage local HTTPS browser acceptance suite. No k3s or GitOps resources were changed.

### Commits

- tiny-idp `c984bfd` — production durable signup invitation activation and policy
- go-go-goja `7761bdd` — atomic membership invitation acceptance
- go-go-goja `41cc3f6` — pending invitation continuation through OIDC registration
- go-go-goja `c19969b` — route, identity-binding, and tenant-audit hardening
- tiny-idp `63dfc5f` — shared local Compose bootstrap and browser acceptance

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py — Executable seven-stage local product contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/compose.yaml — Shared HTTPS local deployment topology
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja/pkg/gojahttp/auth/membershipinvite/sqlstore/sqlstore.go — Atomic application membership and invitation transaction

## 2026-07-21

Phase 6 design: specified the transactional, idempotent, conflict-safe, audited production administrator bootstrap and generated-host operator CLI contract.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/21/TINYIDP-INVITES-001--professional-signup-invitations-and-application-membership-invitations/design-doc/02-production-administrator-bootstrap-design-and-implementation-guide.md — Follow-on design and implementation sequence
