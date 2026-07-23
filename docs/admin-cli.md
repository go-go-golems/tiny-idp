# Admin CLI

`tinyidp admin` provides the first operational command surface for SQLite-backed product deployments.

All commands currently use an explicit database path:

```bash
tinyidp admin --db ./tinyidp.db <command>
```

The structured production configuration ticket will later replace or supplement `--db` with config-backed runtime loading.

## Initialize and migrate

```bash
tinyidp admin --db ./tinyidp.db init --generate-signing-key --kid initial-rsa-1
tinyidp admin --db ./tinyidp.db migrate
tinyidp admin --db ./tinyidp.db migrate --dry-run
tinyidp admin --db ./tinyidp.db doctor
```

`init` opens the SQLite database, applies embedded migrations, and can create an initial active RSA signing key. `doctor` checks clients and signing keys using production validation rules.

## Clients

```bash
tinyidp admin --db ./tinyidp.db client create \
  --id web-app \
  --generate-secret \
  --redirect-uri https://app.example.test/callback \
  --scope openid --scope profile --scope email --scope offline_access \
  --grant-type authorization_code --grant-type refresh_token \
  --require-pkce

tinyidp admin --db ./tinyidp.db client list
tinyidp admin --db ./tinyidp.db client get --id web-app
tinyidp admin --db ./tinyidp.db client disable --id web-app
tinyidp admin --db ./tinyidp.db client enable --id web-app
tinyidp admin --db ./tinyidp.db client rotate-secret --id web-app
```

Client command output redacts stored secret hashes. Generated secrets are returned once in command output. For operator-managed or container-mounted secrets, use `--secret-file /run/secrets/<name>` instead of placing a value in argv. `--secret`, `--secret-file`, and `--generate-secret` are mutually exclusive. Secret files must be regular, non-symlink files no larger than 4096 bytes; surrounding whitespace is removed, empty files fail closed, and the resulting client secret must not exceed bcrypt's 72-byte input limit.

## Users

See `docs/users-and-passwords.md` for user/password semantics. The admin CLI includes:

```bash
tinyidp admin --db ./tinyidp.db user create --login alice --password-from-stdin
tinyidp admin --db ./tinyidp.db user set-password --login alice --password-from-stdin
tinyidp admin --db ./tinyidp.db user get --login alice
tinyidp admin --db ./tinyidp.db user disable --login alice
tinyidp admin --db ./tinyidp.db user enable --login alice
```

Prefer `--password-from-stdin` over `--password` outside throwaway local testing.

## Signing keys

```bash
tinyidp admin --db ./tinyidp.db keys generate --kid rsa-1 --active
tinyidp admin --db ./tinyidp.db keys rotate --kid rsa-2
tinyidp admin --db ./tinyidp.db keys list
tinyidp admin --db ./tinyidp.db keys retire --kid rsa-1
```

Key command output never includes private key PEM material.

## Backups and diagnostics

```bash
tinyidp admin --db ./tinyidp.db backup create --out ./tinyidp-backup.db
tinyidp admin backup verify --path ./tinyidp-backup.db
tinyidp admin --db ./tinyidp.db export diagnostics
```

Diagnostics are sanitized: clients show whether a secret exists, not the secret hash, and keys omit private key PEM bytes.

## Validation

```bash
go test ./...
scripts/run-conformance.sh
```
