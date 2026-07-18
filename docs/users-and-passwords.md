# Users, Passwords, and Login Abuse Controls

`tinyidp` strict login uses durable user records, separate password
credentials, atomic account-security state, and bounded Argon2id work.

## Data model

- `idpstore.User` stores OIDC subject/profile/account state only.
- `idpstore.PasswordCredential` stores the encoded Argon2id verifier and
  password timestamps.
- `idpstore.AccountSecurityState` stores failed-login counters, lockout
  timestamps, and last successful login time.

Password verifiers are never stored on `idpstore.User`. The former
`MustChangeAtLogin` field was removed because no safe protocol flow existed to
complete the forced change; tiny-idp does not publish state it cannot enforce.

## Password establishment policy

Password acceptance is public and independent from Argon2 encoding parameters:

```go
policy := idp.DefaultPasswordAcceptancePolicy()
```

The production default follows NIST SP 800-63B-4 for a single-factor password:

- minimum 15 Unicode code points;
- at least 64 characters permitted (tiny-idp permits 1024);
- NFC normalization before hashing and verification;
- no character-class composition rules;
- complete-password blocklist plus login/user context checks;
- a 4096-byte ceiling before Argon2 to bound resource use.

The same policy is applied by `PasswordService.HashCredential`, so user create,
password reset, and password change cannot drift. Authentication normalizes in
the same way but does not re-run establishment-only length/blocklist checks
against an already issued credential.

The built-in blocklist is a baseline for self-contained deployments. Hosts can
inject a larger breach/common-password implementation through
`idp.PasswordBlocklist`; implementations must not log or retain supplied
passwords.

## Bounded Argon2id work

`idp.PasswordWorkConfig.MaxConcurrent` bounds both hashing and verification.
The default capacity is two concurrent 64 MiB Argon2id operations. Additional
requests wait on a context-aware semaphore; cancellation rejects the work and
does not start Argon2.

```go
stats, ok := provider.PasswordWorkStats()
```

`PasswordWorkStats` exports capacity, in-flight/waiting counts, saturations,
context rejections, completions, total wait nanoseconds, and total Argon2
duration nanoseconds. It contains no usernames, passwords, or hashes.

## Login behavior and fail-closed storage

The strict adapter authenticates `POST /authorize` through an
`idp.PasswordAuthenticator` before creating a browser session.

- Browser-facing credential failures use the generic message
  `invalid login or password`.
- Unknown accounts perform the same bounded dummy Argon2 verification.
- Account-disabled and account-locked outcomes use stable internal audit codes.
- A failed counter write, successful-login reset, credential load, malformed
  stored verifier, or rehash persistence failure returns authentication
  unavailable; it never accepts the login or silently skips security state.
- Production custom authenticators must report production readiness and expose
  password-work metrics.

## Abuse-control keys and trusted proxies

Every submitted password login consumes three independent limiter keys:

```text
login:account:<sha256(normalized login)>
login:client:<oauth client id>
login:address:<resolved client IP>
```

All three are evaluated even when one rejects, and the account key does not
contain the login. Production construction requires a limiter implementing
`idp.ProductionReadyReporter`. `idp.FixedWindowRateLimiter` is the supported
in-process implementation for the single-node SQLite topology.

`idp.DirectClientAddressResolver` ignores forwarding headers and uses the TCP
peer. When a known reverse proxy is present, use `NewTrustedProxyResolver` with
explicit CIDRs. It accepts `X-Forwarded-For` only when the immediate peer is
trusted, walks right-to-left through trusted intermediaries, rejects malformed
or overlong chains, and returns the first untrusted address.

## Password change revocation

Password replacement atomically writes the credential, resets lockout state,
and invalidates security artifacts for the user:

- browser sessions;
- domain authorization codes and grants;
- domain access and refresh tokens;
- Fosite authorization-code, PKCE, OIDC session, access-token, and refresh-token
  rows indexed by subject.

Migration 004 adds subject columns and indexes to protocol tables so this is a
bounded SQL operation rather than JSON scanning at password-change time.

## Admin commands

```bash
printf '%s\n' 'a sufficiently long password phrase' | \
  tinyidp admin --db ./tinyidp.db user create \
    --login alice --email alice@example.test --email-verified \
    --name 'Alice Example' --password-from-stdin

printf '%s\n' 'a different long password phrase' | \
  tinyidp admin --db ./tinyidp.db user set-password \
    --login alice --password-from-stdin
```

Prefer `--password-from-stdin` so passwords do not land in shell history. The
`--password` flag exists for throwaway local tests only.

## Validation

```bash
go test -race ./internal/authn ./internal/admin ./internal/fositeadapter ./pkg/idp ./pkg/sqlitestore
go run ./ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/scripts/01-password-work-load \
  --workers 8 --attempts 24 --max-concurrent 2
```

The load command deliberately uses production Argon2id parameters and reports
runtime memory plus password-work saturation metrics as JSON.
