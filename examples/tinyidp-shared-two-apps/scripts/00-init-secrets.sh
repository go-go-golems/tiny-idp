#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
secret_dir="$example_dir/runtime/secrets"
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
chmod 0600 "$secret_dir/local-admin-password.txt" "$secret_dir/local-invitee-password.txt" "$secret_dir/invitation-lookup.key" "$secret_dir/email-challenge.key"
printf 'Initialized local-only secrets in %s\n' "$secret_dir"
