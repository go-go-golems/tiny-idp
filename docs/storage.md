# tiny-idp Strict Engine Storage Profile

The strict engine stores domain state through the public `pkg/idpstore.Store` contract. Production mode requires a persistent store unless tests explicitly override validation.

## SQLite store

`pkg/sqlitestore` is the initial durable embedded store. Its migration currently creates:

- clients
- users
- grants
- authorization codes
- access tokens
- refresh tokens
- consents
- sessions
- signing keys
- Fosite authorize-code, PKCE, OIDC, access-token, refresh-token, and JWT ID tables

The Fosite tables live in the SQLite migration, not adapter startup DDL, so schema ownership is centralized in the store package.

## Secret handling

The store keeps hashes for bearer-style handles where the domain owns the token/session value. Fosite protocol rows store Fosite request/session metadata needed to complete grants and refresh rotation; raw externally issued token values must not be written to audit logs.

## Required invariants

- Authorization codes are one-time use, including under parallel consumption.
- Refresh tokens rotate and old-token reuse revokes/rejects the family.
- Consent lookup normalizes scope order and duplicates.
- Sessions store only keyed handle hashes.
- Signing keys persist across restart.
- Retired signing keys remain available through `VerificationKeys` until retention cleanup.
