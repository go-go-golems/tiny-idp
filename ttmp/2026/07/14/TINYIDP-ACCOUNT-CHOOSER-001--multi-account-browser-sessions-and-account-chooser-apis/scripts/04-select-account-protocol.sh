#!/usr/bin/env bash
set -euo pipefail

# Exercise selection, fresh-session activation, consent continuation, and the
# explicit credential path for “Use another account.”
repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$repo_root"

go test ./internal/fositeadapter -run '^TestPromptSelectAccount' -count=1 -v
