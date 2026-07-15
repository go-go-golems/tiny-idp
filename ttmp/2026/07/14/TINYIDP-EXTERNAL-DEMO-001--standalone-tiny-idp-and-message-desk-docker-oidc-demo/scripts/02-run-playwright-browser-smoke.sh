#!/usr/bin/env sh
set -eu

# Run after `01-compose-health-smoke.sh` or `docker compose up --build -d`.
# The suite uses the public development seed from demo-seed.json; it never
# accepts production credentials. Override URLs/account labels only for a
# deliberately equivalent development fixture.
root=$(CDPATH= cd -- "$(dirname -- "$0")/../../../../../../" && pwd)
scripts_dir="$root/ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts"

cd "$scripts_dir"
pnpm install --frozen-lockfile
pnpm test:browser
