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

write_password() {
  target=$1
  value=$2
  if [ ! -s "$target" ]; then
    printf '%s\n' "$value" >"$target"
  fi
}

write_printable_secret() {
  target=$1
  if [ -s "$target" ] && ! LC_ALL=C grep -Eq '^[0-9a-f]{64}$' "$target"; then
    backup="$target.pre-printable-backup"
    if [ -e "$backup" ]; then
      printf 'Refusing to replace %s because backup %s already exists\n' "$target" "$backup" >&2
      exit 1
    fi
    mv "$target" "$backup"
    chmod 0600 "$backup"
    printf 'Preserved non-printable secret as %s\n' "$backup"
  fi
  if [ ! -s "$target" ]; then
    od -An -N32 -tx1 /dev/urandom | tr -d ' \n' >"$target"
    printf '\n' >>"$target"
  fi
}

write_password "$secret_dir/local-admin-password.txt" "local-jitsi-admin-password-2026!"
write_password "$secret_dir/local-policy-denied-password.txt" "local-jitsi-policy-denied-password-2026!"
write_printable_secret "$secret_dir/jitsi-shared-secret.key"
write_printable_secret "$secret_dir/jicofo-auth-password.key"
write_printable_secret "$secret_dir/jvb-auth-password.key"

chmod 0600 "$secret_dir"/*
printf 'Initialized local-only Jitsi secrets in %s\n' "$secret_dir"
