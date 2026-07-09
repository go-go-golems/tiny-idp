#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT

mkdir -p "$probe_dir/consumer"
printf 'module example.test/tinyidp-consumer\n\ngo 1.25.11\n\nrequire github.com/manuel/tinyidp v0.0.0\n\nreplace github.com/manuel/tinyidp => %s\n' "$repo_root" > "$probe_dir/go.mod"
cp "$repo_root/go.sum" "$probe_dir/go.sum"
printf '%s\n' \
  'package consumer' \
  '' \
  'import (' \
  '  "github.com/manuel/tinyidp/internal/store/sqlite"' \
  '  "github.com/manuel/tinyidp/pkg/embeddedidp"' \
  ')' \
  '' \
  'func Build() error {' \
  '  store, err := sqlite.Open("idp.db")' \
  '  if err != nil { return err }' \
  '  defer store.Close()' \
  '  _, err = embeddedidp.New(embeddedidp.Options{Store: store})' \
  '  return err' \
  '}' > "$probe_dir/consumer/consumer.go"

set +e
output="$(cd "$probe_dir" && GOWORK=off go test -mod=mod ./consumer 2>&1)"
status=$?
set -e
printf '%s\n' "$output"

if [[ $status -eq 0 ]]; then
  printf 'UNEXPECTED: external production embedding compiled successfully\n' >&2
  exit 1
fi
if [[ "$output" != *"use of internal package"*"not allowed"* ]]; then
  printf 'UNEXPECTED: compilation failed for a reason other than the public/internal API boundary\n' >&2
  exit 1
fi
printf 'EXPECTED: external production embedding is blocked by Go internal-package visibility\n'
