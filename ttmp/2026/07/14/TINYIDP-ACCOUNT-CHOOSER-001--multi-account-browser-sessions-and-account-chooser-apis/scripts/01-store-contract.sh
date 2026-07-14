#!/usr/bin/env bash
set -euo pipefail

# Run the account-chooser persistence contract against both supported stores.
# This deliberately uses the repository's normal Go/work cache.
repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$repo_root"

go test ./internal/store/memory ./pkg/sqlitestore ./pkg/idpstore -count=1
