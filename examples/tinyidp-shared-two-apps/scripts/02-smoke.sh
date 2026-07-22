#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
trust_file="$example_dir/runtime/caddy-local-root.crt"

if [ ! -r "$trust_file" ]; then
  "$example_dir/scripts/01-export-browser-ca.sh" >/dev/null
fi

check_200() {
  url=$1
  attempt=1
  while [ "$attempt" -le 30 ]; do
    response_code=$(curl --cacert "$trust_file" -sS -o /dev/null -w '%{http_code}' "$url" || true)
    if [ "$response_code" = 200 ]; then
      printf 'OK %s\n' "$url"
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 1
  done
  printf 'expected 200 from %s, last response was %s\n' "$url" "$response_code" >&2
  return 1
}

check_login_redirect() {
  url=$1
  client_id=$2
  headers=$(mktemp)
  trap 'rm -f "$headers"' EXIT HUP INT TERM
  curl --cacert "$trust_file" -sS -o /dev/null -D "$headers" "$url"
  if ! grep -F "location: https://idp.localhost:8443/authorize?client_id=$client_id" "$headers" >/dev/null; then
    printf 'missing TinyIDP redirect for %s from %s\n' "$client_id" "$url" >&2
    sed -n '1,20p' "$headers" >&2
    exit 1
  fi
  rm -f "$headers"
  trap - EXIT HUP INT TERM
  printf 'OK login redirect %s\n' "$client_id"
}

check_200 https://idp.localhost:8443/readyz
check_200 https://message.localhost:8443/readyz
check_200 https://goja.localhost:8443/auth/readyz
check_login_redirect https://message.localhost:8443/auth/login tinyidp-message-app
check_login_redirect https://goja.localhost:8443/auth/login goja-auth-host-demo
