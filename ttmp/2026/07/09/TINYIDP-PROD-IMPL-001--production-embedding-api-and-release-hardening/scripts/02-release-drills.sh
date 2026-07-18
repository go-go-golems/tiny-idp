#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

db="$work_dir/idp.db"
backup="$work_dir/backup/idp.db"

cd "$repo_root"

go run ./cmd/tinyidp admin --db "$db" init --generate-signing-key --kid initial
go run ./cmd/tinyidp admin --db "$db" client create \
  --id drill-spa --public --redirect-uri https://client.example.test/callback \
  --scope openid --scope profile --id-token-ttl 1h --refresh-token-ttl 24h
printf '%s\n' 'release drill password has sufficient length' | \
  go run ./cmd/tinyidp admin --db "$db" user create \
    --login release-drill --password-from-stdin
go run ./cmd/tinyidp admin --db "$db" doctor
go run ./cmd/tinyidp admin --db "$db" backup create --out "$backup"
go run ./cmd/tinyidp admin backup verify --path "$backup"

go run ./cmd/tinyidp admin --db "$db" keys rotate --kid rotated
go run ./cmd/tinyidp admin --db "$db" client create \
  --id post-backup --public --redirect-uri https://client.example.test/post-backup \
  --scope openid
go run ./cmd/tinyidp admin --db "$db" doctor

go run ./cmd/tinyidp admin --db "$db" backup restore --path "$backup"
go run ./cmd/tinyidp admin --db "$db" doctor
if go run ./cmd/tinyidp admin --db "$db" client get --id post-backup >/dev/null 2>&1; then
  printf 'post-backup client unexpectedly survived restore\n' >&2
  exit 1
fi

go test ./pkg/sqlitestore -run 'TestOnlineBackupVerifyAndRestore|TestRestorePreservesRollbackAndRejectsCorruption|TestNewerSchemaVersionRefusesOpen|TestSigningKeyRotationPersistsRetiredVerificationKey' -count=1
go test ./internal/fositeadapter -run 'TestTokenSecretRotationInvalidatesPriorOpaqueTokens' -count=1

printf 'release drills passed: migration, downgrade refusal, backup, restore, rollback preservation, signing-key rotation, and token-secret rotation\n'
