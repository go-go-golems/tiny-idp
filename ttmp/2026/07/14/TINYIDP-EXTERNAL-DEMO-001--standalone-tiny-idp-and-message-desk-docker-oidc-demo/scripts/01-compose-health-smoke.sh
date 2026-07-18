#!/usr/bin/env sh
set -eu

# Starts (or rebuilds) the documented two-container topology and verifies only
# its unauthenticated health boundary. Browser login/logout scenarios remain
# recorded in the implementation diary and should use Playwright.
root=$(CDPATH= cd -- "$(dirname -- "$0")/../../../../../../examples/tinyidp-external-message-desk" && pwd)
cd "$root"

docker compose up --build -d
curl --fail --silent --show-error http://localhost:8081/readyz >/dev/null
curl --fail --silent --show-error http://localhost:8080/readyz >/dev/null
docker compose ps
