#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
ticket="$repo_root/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening"

docs=(
  "$ticket/design-doc/05-intern-accelerated-curriculum-and-code-reading-map.md"
  "$ticket/design-doc/06-oauth-oidc-protocol-security-foundations.md"
  "$ticket/design-doc/07-security-state-machines-and-temporal-invariants.md"
  "$ticket/design-doc/08-durable-security-state-transactions-and-concurrency.md"
  "$ticket/design-doc/09-assurance-methods-and-evidence-interpretation.md"
  "$ticket/design-doc/10-production-trust-boundaries-and-release-security.md"
  "$ticket/reference/07-intern-security-review-labs.md"
)

minimum_lines=1000
total=0
for doc in "${docs[@]}"; do
  test -f "$doc"
  lines="$(wc -l < "$doc")"
  if (( lines < minimum_lines )); then
    printf 'FAIL: %s has %d lines; minimum is %d\n' "$doc" "$lines" "$minimum_lines" >&2
    exit 1
  fi
  rg -q '^# ' "$doc"
  rg -qi 'research|standard|source' "$doc"
  rg -qi 'code|symbol|file' "$doc"
  rg -qi 'exercise|lab|question' "$doc"
  total=$((total + lines))
  printf 'PASS: %4d lines %s\n' "$lines" "${doc#"$repo_root/"}"
done

cd "$repo_root"
git diff --check
docmgr doctor --ticket TINYIDP-PROD-IMPL-001
printf 'PASS: %d textbook lines across %d documents\n' "$total" "${#docs[@]}"
