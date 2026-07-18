#!/usr/bin/env bash
set -euo pipefail

# Validate the chooser presentation contract and default renderer semantics.
repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$repo_root"

go test ./pkg/idpui ./pkg/idpui/idpuitest -count=1
