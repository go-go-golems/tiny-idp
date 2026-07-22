#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
secret_dir="$example_dir/runtime/secrets"
caddy_volume="tinyidp-local-caddy-pki"

if ! docker volume inspect "$caddy_volume" >/dev/null 2>&1; then
  docker volume create \
    --label dev.wesen.purpose=local-caddy-pki \
    --label dev.wesen.retention=manual-delete-only \
    "$caddy_volume" >/dev/null
  printf 'Created persistent local Caddy PKI volume %s\n' "$caddy_volume"
fi

mkdir -p "$secret_dir"
chmod 0700 "$secret_dir"

if [ ! -s "$secret_dir/local-admin-password.txt" ]; then
  printf '%s\n' 'local-admin-password-2026!' >"$secret_dir/local-admin-password.txt"
fi
if [ ! -s "$secret_dir/local-invitee-password.txt" ]; then
  printf '%s\n' 'local-invitee-password-2026!' >"$secret_dir/local-invitee-password.txt"
fi
if [ ! -s "$secret_dir/invitation-lookup.key" ]; then
  dd if=/dev/urandom of="$secret_dir/invitation-lookup.key" bs=32 count=1 status=none
fi
if [ ! -s "$secret_dir/email-challenge.key" ]; then
  dd if=/dev/urandom of="$secret_dir/email-challenge.key" bs=32 count=1 status=none
fi
if [ ! -s "$secret_dir/goja-appauth-dsn.txt" ]; then
  printf '%s\n' 'postgres://goja:goja-local-development-only@postgres:5432/goja_auth?sslmode=disable' >"$secret_dir/goja-appauth-dsn.txt"
fi
chmod 0600 "$secret_dir/local-admin-password.txt" "$secret_dir/local-invitee-password.txt" "$secret_dir/invitation-lookup.key" "$secret_dir/email-challenge.key" "$secret_dir/goja-appauth-dsn.txt"
printf 'Initialized local-only secrets in %s\n' "$secret_dir"
