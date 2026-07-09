# Users and Passwords

`tinyidp` production-style strict login uses durable user records plus separate password credentials.

## Data model

- `domain.User` stores OIDC subject/profile/account state only.
- `domain.PasswordCredential` stores the encoded password hash and password lifecycle flags.
- `domain.AccountSecurityState` stores failed-login counters, lockout timestamps, and last successful login time.

Password hashes are encoded Argon2id strings and are never stored on `domain.User`.

## Strict login behavior

The strict Fosite adapter authenticates `POST /authorize` through a `PasswordAuthenticator` before creating a browser session.

- Production/default credential path requires a stored password credential.
- Dev-mode strict runs preserve legacy seeded-user ergonomics by allowing passwordless login only when no credential exists for that user.
- If a credential exists, the password is verified even in dev mode.
- Browser-facing failures use the generic message `invalid login or password`.
- Audit reasons use stable codes such as `invalid_credentials`, `account_disabled`, and `account_locked`.

## Admin commands

The first user/password admin commands operate on a SQLite database directly:

```bash
# Create a user and password credential.
printf '%s\n' 'alice-password' | \
  tinyidp admin --db ./tinyidp.db user create \
    --login alice \
    --email alice@example.test \
    --email-verified \
    --name 'Alice Example' \
    --password-from-stdin

# Replace a password and require a change at next login.
printf '%s\n' 'new-password' | \
  tinyidp admin --db ./tinyidp.db user set-password \
    --login alice \
    --must-change \
    --password-from-stdin

# Inspect or disable the user.
tinyidp admin --db ./tinyidp.db user get --login alice
tinyidp admin --db ./tinyidp.db user disable --login alice
tinyidp admin --db ./tinyidp.db user enable --login alice
```

Prefer `--password-from-stdin` so passwords do not land in shell history. The `--password` flag exists for throwaway local tests only.

## Validation

Use:

```bash
go test ./internal/passwordhash ./internal/authn ./internal/admin ./internal/fositeadapter ./internal/store/memory ./pkg/sqlitestore ./pkg/embeddedidp
```
