#!/bin/sh
# Provision an isolated, localhost-only strict tinyidp host.
set -eu

usage() {
  echo "usage: $0 --work-dir ABSOLUTE_PATH" >&2
  exit 64
}

work_dir=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --work-dir) [ "$#" -ge 2 ] || usage; work_dir=$2; shift 2 ;;
    *) usage ;;
  esac
done
[ -n "$work_dir" ] || usage
case "$work_dir" in /*) ;; *) echo "--work-dir must be absolute" >&2; exit 64 ;; esac

for path in "$work_dir/idp.db" "$work_dir/token-secret" "$work_dir/tls.crt" "$work_dir/tls.key"; do
  if [ -e "$path" ]; then
    echo "refusing to reuse existing fixture file: $path" >&2
    exit 1
  fi
done

umask 077
mkdir -p "$work_dir"
openssl rand -out "$work_dir/token-secret" 32
chmod 600 "$work_dir/token-secret"
openssl req -x509 -newkey rsa:2048 -sha256 -nodes -keyout "$work_dir/tls.key" -out "$work_dir/tls.crt" -days 1 -subj '/CN=localhost' -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1' >/dev/null 2>&1

go run ./cmd/tinyidp admin --db "$work_dir/idp.db" init --generate-signing-key --kid local-strict-smoke
printf '%s\n' 'Smoke Device Password 2026!' | go run ./cmd/tinyidp admin --db "$work_dir/idp.db" user create --login smoke --name 'Smoke User' --password-from-stdin
go run ./cmd/tinyidp admin --db "$work_dir/idp.db" client create --id device-smoke --public --grant-type urn:ietf:params:oauth:grant-type:device_code --scope openid --scope profile
go run ./cmd/tinyidp admin --db "$work_dir/idp.db" doctor

echo "fixture work directory: $work_dir"
echo "fixture login: smoke"
echo "fixture password: Smoke Device Password 2026!"
echo "next: scripts/02-run-local-strict-host.sh --work-dir $work_dir"
