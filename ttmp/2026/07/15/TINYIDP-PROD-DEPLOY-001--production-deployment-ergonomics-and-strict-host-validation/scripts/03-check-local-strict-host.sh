#!/bin/sh
# Probe a running local strict host without disabling TLS verification.
set -eu

usage() {
  echo "usage: $0 --work-dir ABSOLUTE_PATH [--base-url https://localhost:9443]" >&2
  exit 64
}

work_dir=
base_url=https://localhost:9443
while [ "$#" -gt 0 ]; do
  case "$1" in
    --work-dir) [ "$#" -ge 2 ] || usage; work_dir=$2; shift 2 ;;
    --base-url) [ "$#" -ge 2 ] || usage; base_url=$2; shift 2 ;;
    *) usage ;;
  esac
done
[ -n "$work_dir" ] || usage
[ -f "$work_dir/tls.crt" ] || { echo "missing $work_dir/tls.crt" >&2; exit 1; }

curl --fail --silent --show-error --cacert "$work_dir/tls.crt" "$base_url/healthz"
curl --fail --silent --show-error --cacert "$work_dir/tls.crt" "$base_url/readyz"
curl --fail --silent --show-error --cacert "$work_dir/tls.crt" "$base_url/.well-known/openid-configuration"
