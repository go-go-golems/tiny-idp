#!/bin/sh
# Run a previously provisioned localhost-only strict tinyidp host in the foreground.
set -eu

usage() {
  echo "usage: $0 --work-dir ABSOLUTE_PATH [--addr 127.0.0.1:9443]" >&2
  exit 64
}

work_dir=
addr=127.0.0.1:9443
while [ "$#" -gt 0 ]; do
  case "$1" in
    --work-dir) [ "$#" -ge 2 ] || usage; work_dir=$2; shift 2 ;;
    --addr) [ "$#" -ge 2 ] || usage; addr=$2; shift 2 ;;
    *) usage ;;
  esac
done
[ -n "$work_dir" ] || usage
for path in "$work_dir/idp.db" "$work_dir/token-secret" "$work_dir/tls.crt" "$work_dir/tls.key"; do
  [ -f "$path" ] || { echo "missing provisioned file: $path" >&2; exit 1; }
done

exec go run ./cmd/tinyidp serve-production --addr "$addr" --issuer "https://localhost:${addr##*:}" --db "$work_dir/idp.db" --audit-path "$work_dir/audit.jsonl" --token-secret-file "$work_dir/token-secret" --tls-cert "$work_dir/tls.crt" --tls-key "$work_dir/tls.key"
