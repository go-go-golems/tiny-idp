# Changelog

## 2026-07-21

- Initial workspace created


## 2026-07-21

Created durable optional unique display-name design, task plan, and implementation diary.

### Related Files

- design-doc/01-durable-unique-display-names-analysis-design-and-implementation-guide.md — Design and implementation guide.
- reference/01-implementation-diary.md — Chronological planning record.

## 2026-07-21

Added optional atomic display-name claim storage, normalization, and account-service enforcement.

### Related Files

- internal/store/memory/store.go — Transactional memory claim implementation.
- pkg/idpaccounts/accounts.go — Policy-aware account preparation and transactional reservation.
- pkg/idpstore/interfaces.go — Display-name claim contract and typed collision.
- pkg/sqlitestore/migrations/014_display_name_claims.sql — Durable SQLite claim table.
