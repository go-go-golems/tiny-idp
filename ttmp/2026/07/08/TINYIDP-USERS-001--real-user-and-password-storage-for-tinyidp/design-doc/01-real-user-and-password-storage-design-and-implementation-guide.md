---
Title: Real User and Password Storage Design and Implementation Guide
Ticket: TINYIDP-USERS-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/domain/types.go
      Note: Current user model has profile/account fields but no credential hash
    - Path: repo://internal/fositeadapter/provider.go
      Note: Strict login path that must call password authenticator
    - Path: repo://internal/scenario/scenario.go
      Note: Mock scenario password semantics that must stay non-production
    - Path: repo://internal/scenario/seeded_users.go
      Note: Dev fixture password model to preserve outside production
    - Path: repo://internal/storage/interfaces.go
      Note: Current user store lacks password credential methods
    - Path: repo://internal/store/sqlite/migrations/001_schema.sql
      Note: Current users table lacks credential/account-security tables
ExternalSources: []
Summary: Design for replacing dev-only seeded-login behavior with durable users, password credentials, lockout, reset, and strict authentication.
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: Use this when implementing production user accounts and password credential verification for tinyidp.
WhenToUse: Read before changing login handling, user storage, password hashing, account lockout, seeded users, or admin user commands.
---


# Real User and Password Storage Design and Implementation Guide

## Executive Summary

`tinyidp` has a durable `UserStore` and a strict Fosite-backed authorization flow, but strict login currently accepts a submitted login if the user exists in the store. Password fields exist in the rendered form and in seeded test fixtures, yet production password credentials are not modeled or verified. This guide designs the missing account system: durable user records, separate password credential records, password hashing, verification, lockout, reset, audit events, and admin integration.

The central design choice is to keep identity profile data separate from credential secrets. `domain.User` should remain the OIDC subject/profile record. Password hashes, hash parameters, password lifecycle timestamps, failure counters, and reset requirements should live in separate credential types and storage tables. That separation keeps token claims clean, prevents accidental password-hash exposure through userinfo/profile paths, and allows future non-password authenticators.

## Problem Statement

The current system has user and login concepts, but not production authentication.

Evidence from the current codebase:

- `internal/domain/types.go:31-47` defines `domain.User` with subject/profile fields, disabled state, lock timestamp, and timestamps. It does not include a password hash or credential lifecycle fields.
- `internal/storage/interfaces.go:25-29` defines `UserStore` with `GetUser`, `GetUserByLogin`, and `PutUser`, but no password credential methods.
- `internal/store/sqlite/migrations/001_schema.sql:2` stores users as a serialized blob keyed by user ID with unique login. There is no credential table.
- `internal/scenario/seeded_users.go:35-41` accepts a plaintext `Password` field for seeded fixtures. This is useful for tests, but it is not a production credential model.
- `internal/scenario/scenario.go:28-30` states that an empty scenario password remains permissive and any submitted password is accepted. That is correct for mock scenarios, not for production strict login.
- `internal/fositeadapter/provider.go:342-355` lowercases the posted login, calls `p.store.GetUserByLogin`, creates a browser session, and emits `login.success`. It does not read or verify the posted password.
- `internal/fositeadapter/provider.go:613-623` renders both username and password fields in the strict login form, so the UI already exposes a credential boundary that the backend must implement.

The product risk is obvious: a production strict provider cannot accept a login name alone. It must verify a secret or delegate to another authenticator. This ticket covers local password credentials.

## Scope

In scope:

- Durable user account model and password credential model.
- Password hashing and verification.
- Login verification in the strict Fosite adapter.
- Account disabled/locked/reset-required behavior.
- Audit events for success, failure, lockout, disabled account, and password changes.
- Admin-service integration for user creation and password operations.
- SQLite and memory store implementations.
- Tests for credential behavior and security-sensitive edge cases.

Out of scope:

- Web self-service registration.
- Email delivery for password reset.
- MFA/WebAuthn.
- External identity federation.
- Passwordless login.

## Current Login Flow

```text
GET /authorize
  render login form
       |
       v
POST /authorize
  parse form
  validate CSRF
  NewAuthorizeRequest
  read browser session
  login := form["login"]
  GetUserByLogin(login)
  create browser session
  finish authorize
```

The missing step is between `GetUserByLogin` and `create browser session`:

```text
VerifyPassword(login, password, remote address, now)
```

That verification must handle nonexistent users and wrong passwords in a way that does not leak which logins exist.

## Proposed Domain Model

Keep `domain.User` focused on OIDC subject/profile data. Add credential-specific models.

```go
package domain

type PasswordCredential struct {
    UserID string
    Login string
    PasswordHash []byte
    HashAlgorithm string // argon2id-v1 initially
    HashParams PasswordHashParams
    CreatedAt time.Time
    UpdatedAt time.Time
    PasswordChangedAt time.Time
    MustChangeAtLogin bool
    Disabled bool
}

type PasswordHashParams struct {
    MemoryKiB uint32
    Iterations uint32
    Parallelism uint8
    SaltLength uint8
    KeyLength uint32
}

type AccountSecurityState struct {
    UserID string
    FailedLoginCount int
    FirstFailedLoginAt *time.Time
    LastFailedLoginAt *time.Time
    LockedUntil *time.Time
    LastSuccessfulLoginAt *time.Time
}
```

The exact storage representation can combine credential and security state into one table or split them. The important part is that password hash data is not stored in `domain.User`.

## Proposed Storage Interfaces

Add a credential store interface instead of expanding `UserStore` too broadly.

```go
type PasswordCredentialStore interface {
    PutPasswordCredential(ctx context.Context, c domain.PasswordCredential) error
    GetPasswordCredentialByLogin(ctx context.Context, login string) (domain.PasswordCredential, error)
    GetPasswordCredentialByUserID(ctx context.Context, userID string) (domain.PasswordCredential, error)
    DeletePasswordCredential(ctx context.Context, userID string) error
}

type AccountSecurityStore interface {
    GetAccountSecurityState(ctx context.Context, userID string) (domain.AccountSecurityState, error)
    PutAccountSecurityState(ctx context.Context, s domain.AccountSecurityState) error
    ResetAccountSecurityState(ctx context.Context, userID string, now time.Time) error
}
```

Then update the aggregate store:

```go
type Store interface {
    ClientStore
    UserStore
    PasswordCredentialStore
    AccountSecurityStore
    GrantStore
    AuthorizationCodeStore
    AccessTokenStore
    RefreshTokenStore
    ConsentStore
    SessionStore
    KeyStore
}
```

If that aggregate change is too disruptive, introduce a narrower optional interface on `fositeadapter.Options` first:

```go
type PasswordAuthenticator interface {
    AuthenticatePassword(ctx context.Context, login, password string, meta LoginMetadata) (domain.User, AuthResult, error)
}
```

The provider can depend on `PasswordAuthenticator` while stores evolve underneath.

## Password Hashing

Use Argon2id through `golang.org/x/crypto/argon2` with a versioned encoded hash format. The encoded hash should contain algorithm, parameters, salt, and derived key.

Example encoded value:

```text
$argon2id$v=19$m=65536,t=3,p=2$base64salt$base64hash
```

Recommended initial parameters:

- memory: 64 MiB;
- iterations: 3;
- parallelism: 2;
- salt length: 16 bytes;
- key length: 32 bytes.

Make parameters configurable only through a small policy struct, not arbitrary per-request flags.

```go
type PasswordHasher interface {
    HashPassword(password []byte) (EncodedPasswordHash, error)
    VerifyPassword(password []byte, encoded EncodedPasswordHash) (needsRehash bool, err error)
}
```

Verification must use constant-time comparison for derived keys.

## Authentication Service

Create `internal/authn/password.go` or `internal/account/password.go`.

```go
type PasswordService struct {
    Users storage.UserStore
    Credentials storage.PasswordCredentialStore
    Security storage.AccountSecurityStore
    Hasher PasswordHasher
    Clock func() time.Time
    Audit audit.Sink
    Policy PasswordPolicy
}

type PasswordPolicy struct {
    MinLength int
    MaxLength int
    LockoutThreshold int
    LockoutWindow time.Duration
    LockoutDuration time.Duration
    RequireKnownUser bool // true in production
}

type LoginMetadata struct {
    RemoteAddr string
    UserAgent string
    ClientID string
}

func (s *PasswordService) AuthenticatePassword(ctx context.Context, login, password string, meta LoginMetadata) (domain.User, AuthResult, error)
```

### Pseudocode: authentication

```go
func (s *PasswordService) AuthenticatePassword(ctx context.Context, login, password string, meta LoginMetadata) (domain.User, AuthResult, error) {
    now := s.Clock().UTC()
    normalized := user.Normalize(login)
    if normalized == "" || password == "" {
        s.auditFailure(ctx, normalized, meta, "invalid_credentials")
        return domain.User{}, AuthResult{}, ErrInvalidCredentials
    }

    u, userErr := s.Users.GetUserByLogin(ctx, normalized)
    cred, credErr := s.Credentials.GetPasswordCredentialByLogin(ctx, normalized)

    // Always run a dummy hash verification on missing users to reduce timing leaks.
    if userErr != nil || credErr != nil {
        _ = s.Hasher.VerifyPassword([]byte(password), s.Policy.DummyHash)
        s.auditFailure(ctx, normalized, meta, "invalid_credentials")
        return domain.User{}, AuthResult{}, ErrInvalidCredentials
    }

    if u.Disabled || cred.Disabled {
        s.auditFailure(ctx, normalized, meta, "account_disabled")
        return domain.User{}, AuthResult{}, ErrAccountDisabled
    }

    state, _ := s.Security.GetAccountSecurityState(ctx, u.ID)
    if state.LockedUntil != nil && now.Before(*state.LockedUntil) {
        s.auditFailure(ctx, normalized, meta, "account_locked")
        return domain.User{}, AuthResult{}, ErrAccountLocked
    }

    needsRehash, err := s.Hasher.VerifyPassword([]byte(password), cred.PasswordHash)
    if err != nil {
        s.recordFailure(ctx, u.ID, now)
        s.auditFailure(ctx, normalized, meta, "invalid_credentials")
        return domain.User{}, AuthResult{}, ErrInvalidCredentials
    }

    s.Security.ResetAccountSecurityState(ctx, u.ID, now)
    if needsRehash { s.rehashAsyncOrInline(ctx, cred, password) }
    s.auditSuccess(ctx, u, meta)
    return u, AuthResult{MustChangePassword: cred.MustChangeAtLogin}, nil
}
```

The returned error to the browser should usually be `invalid login or password`, even if the internal audit reason is more specific.

## Strict Provider Integration

Update the provider login path around `internal/fositeadapter/provider.go:342-355`.

Current behavior:

```go
login := strings.ToLower(strings.TrimSpace(r.PostForm.Get("login")))
if login != "" {
    u, err = p.store.GetUserByLogin(r.Context(), login)
    ...
    createBrowserSession(...)
}
```

Target behavior:

```go
login := strings.TrimSpace(r.PostForm.Get("login"))
password := r.PostForm.Get("password")
if login != "" {
    result, err := p.authenticator.AuthenticatePassword(r.Context(), login, password, LoginMetadata{
        RemoteAddr: r.RemoteAddr,
        UserAgent: r.UserAgent(),
        ClientID: ar.GetClient().GetID(),
    })
    if err != nil {
        p.renderInteractionError(w, ar, "invalid login or password")
        return
    }
    u = result.User
    authTime = time.Now().UTC()
    createBrowserSession(...)
}
```

Development mode can use a permissive authenticator that preserves existing conformance/dev behavior. Production mode must require a password authenticator.

```go
type Authenticator interface {
    AuthenticatePassword(ctx context.Context, login, password string, meta LoginMetadata) (AuthResult, error)
}

type AuthResult struct {
    User domain.User
    MustChangePassword bool
    AMR []string // e.g. ["pwd"]
}
```

The provider can then set `AMR` on the server-side browser session or OIDC session when that support is added.

## SQLite Schema

Add migration `002_password_credentials.sql`.

```sql
CREATE TABLE IF NOT EXISTS password_credentials (
  user_id TEXT PRIMARY KEY,
  login TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  hash_algorithm TEXT NOT NULL,
  hash_params_json BLOB NOT NULL,
  must_change_at_login INTEGER NOT NULL DEFAULT 0,
  disabled INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  password_changed_at TIMESTAMP NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS account_security_states (
  user_id TEXT PRIMARY KEY,
  failed_login_count INTEGER NOT NULL DEFAULT 0,
  first_failed_login_at TIMESTAMP,
  last_failed_login_at TIMESTAMP,
  locked_until TIMESTAMP,
  last_successful_login_at TIMESTAMP,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_password_credentials_login ON password_credentials(login);
CREATE INDEX IF NOT EXISTS idx_account_security_locked_until ON account_security_states(locked_until);
```

Use text for the encoded password hash because the modular string format is self-describing. Use timestamps in the same style as existing stores.

## Admin Integration

The admin CLI from `TINYIDP-ADMIN-001` should expose these operations:

```text
tinyidp admin user create --login alice --email alice@example.com --password-from-stdin
tinyidp admin user set-password --login alice --password-from-stdin
tinyidp admin user require-password-reset --login alice
tinyidp admin user lock --login alice --duration 30m
tinyidp admin user unlock --login alice
tinyidp admin user disable --login alice
tinyidp admin user enable --login alice
```

Rules:

- Passwords are accepted via stdin, terminal prompt, or env reference; avoid direct flags that land in shell history.
- User creation writes both `domain.User` and `domain.PasswordCredential` transactionally where supported.
- Password reset invalidates active browser sessions and optionally refresh-token grants for that user.
- Disabling a user blocks login and should revoke sessions immediately.

## Mock and Seeded User Compatibility

Do not remove scenario password behavior from the mock engine. It is useful for integration tests and failure simulation.

Instead, define explicit authenticators:

```text
mock engine:
  scenario authenticator; keeps seeded user password semantics and arbitrary fallback users

strict dev mode:
  seeded-user authenticator allowed only when Mode=dev

strict production mode:
  password service authenticator required
```

This preserves the previous testing model while making the production path safe.

## Implementation Plan

### Phase 1: Password hashing package

1. Add `internal/passwordhash` with Argon2id hash and verify functions.
2. Add parser for encoded hash strings.
3. Add tests for successful verify, wrong password, malformed hash, parameter upgrade detection, and constant-time compare path.

### Phase 2: Domain and storage interfaces

1. Add `domain.PasswordCredential`, `domain.PasswordHashParams`, and `domain.AccountSecurityState`.
2. Add credential/security store interfaces.
3. Implement memory store support.
4. Add SQLite migration and store methods.
5. Add store conformance tests similar to existing store tests.

### Phase 3: Password authentication service

1. Add `internal/authn.PasswordService`.
2. Implement normalization, missing-user dummy verify, disabled checks, lockout, failure recording, success reset, and audit events.
3. Add unit tests for each outcome.
4. Add redaction tests for logs and errors.

### Phase 4: Strict provider wiring

1. Add authenticator option to `fositeadapter.Options`.
2. In dev mode, default to a seeded/permissive authenticator only when explicitly allowed.
3. In production mode, fail provider construction if no password authenticator is supplied.
4. Update `provider.go` login POST handling to verify password.
5. Update browser tests to submit valid and invalid passwords.

### Phase 5: Admin/user commands

1. Add user creation and password setting through the admin service.
2. Wire password-from-stdin and generated temporary passwords.
3. Ensure disabling/locking users affects login immediately.
4. Add smoke tests that create a user, start strict provider, and complete auth code flow.

### Phase 6: Documentation and migration guidance

1. Add `docs/users-and-passwords.md`.
2. Add example config/admin commands for first admin bootstrap.
3. Document migration from seeded users to real users.
4. Document password policy and lockout behavior.

## Decision Records

### Decision 1: Keep credentials separate from `domain.User`

Status: proposed.

Decision: add separate password credential and account security models instead of adding `PasswordHash` to `domain.User`.

Rationale: user records are used for OIDC profile claims and may be passed through userinfo-related code. Credential data should have a smaller access surface and its own lifecycle.

### Decision 2: Use Argon2id encoded hashes

Status: proposed.

Decision: use Argon2id with encoded hash strings containing parameters.

Rationale: parameterized hashes let the product detect when a password needs rehashing after policy changes. The encoded form avoids schema changes for parameter updates.

### Decision 3: Preserve mock permissiveness outside production

Status: proposed.

Decision: keep the scenario/seeded-user model for mock and development fixtures; require real password verification for production strict mode.

Rationale: `tinyidp` remains valuable as a mock IdP. Removing arbitrary scenario logins would break the local testing use case. The safety boundary should be engine/mode, not a global behavior removal.

## Testing Strategy

Unit tests:

- password hash/verify success;
- wrong password rejection;
- malformed encoded hash rejection;
- parameter upgrade marks `needsRehash`;
- missing user and wrong password produce the same public error;
- disabled user cannot authenticate;
- locked user cannot authenticate;
- lockout threshold is enforced;
- successful login resets failure counters;
- audit events use stable reason codes.

Store tests:

- put/get credential by login and user ID;
- unique login constraint;
- update password hash;
- delete credential;
- put/get/reset account security state;
- SQLite migration creates expected tables and indexes.

Provider tests:

- valid strict login with password completes authorization;
- wrong password returns unauthorized and no code;
- missing password returns generic invalid credentials;
- disabled user cannot create browser session;
- prompt=none existing session behavior remains unchanged after authentication;
- production provider construction fails without authenticator.

Integration tests:

```bash
go test ./... -count=1
scripts/run-conformance.sh
```

Additional smoke:

```bash
tinyidp admin user create --config-file ./dev.yaml --login alice --password-from-stdin
tinyidp serve --engine fosite --config-file ./dev.yaml
# complete Authorization Code + PKCE flow as alice with the configured password
```

## Risks and Mitigations

- Risk: timing side channels reveal whether a user exists. Mitigation: dummy hash verification for missing users and generic public errors.
- Risk: password hashes leak through user rendering. Mitigation: credentials are not fields on `domain.User` and are never included in userinfo.
- Risk: lockout enables denial-of-service against known accounts. Mitigation: combine account lockout with IP/client rate limiting and bounded lock durations.
- Risk: breaking hosted conformance automation. Mitigation: keep dev-mode seeded authenticator for conformance plans and production authenticator for production mode.
- Risk: accidental plaintext password storage. Mitigation: never add plaintext password fields to domain/store models; tests search serialized records for raw password strings.

## Intern Implementation Notes

Start with password hashing. It is isolated, easy to test, and creates the foundation for everything else. Then add storage interfaces and conformance tests. Only after the service works should you touch the provider login path.

When changing the provider, be careful not to break the existing browser-session behavior: if a valid session exists and `prompt=login` is not requested, the provider can still silently reuse it. Password verification is required when creating a new authenticated browser session, not on every authorization request.

## References

- `internal/domain/types.go:31-47`
- `internal/storage/interfaces.go:25-29`
- `internal/store/sqlite/migrations/001_schema.sql:2`
- `internal/scenario/seeded_users.go:35-41`
- `internal/scenario/scenario.go:28-30`
- `internal/fositeadapter/provider.go:342-355`
- `internal/fositeadapter/provider.go:613-623`
