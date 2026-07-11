#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT

mkdir -p "$probe_dir/consumer"
printf 'module example.test/tinyidp-consumer\n\ngo 1.26.1\n\nrequire github.com/manuel/tinyidp v0.0.0\n\nreplace github.com/manuel/tinyidp => %s\n' "$repo_root" > "$probe_dir/go.mod"
cp "$repo_root/go.sum" "$probe_dir/go.sum"
printf '%s\n' \
  'package consumer' \
  '' \
  'import (' \
  '  "context"' \
  '' \
  '  "github.com/manuel/tinyidp/pkg/idp"' \
  '  "github.com/manuel/tinyidp/pkg/idpstore"' \
  '  "github.com/manuel/tinyidp/pkg/embeddedidp"' \
  '  "github.com/manuel/tinyidp/pkg/sqlitestore"' \
  ')' \
  '' \
  'type limiter struct{}' \
  'func (limiter) Allow(context.Context, string) bool { return true }' \
  'var _ idp.RateLimiter = limiter{}' \
  '' \
  'func Build(ctx context.Context, path string) error {' \
  '  store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))' \
  '  if err != nil { return err }' \
  '  defer store.Close()' \
  '  var _ idpstore.Store = store' \
  '  provider, err := embeddedidp.New(ctx, embeddedidp.Options{' \
  '    Issuer: "https://issuer.example.test",' \
  '    Mode: embeddedidp.ProductionMode,' \
  '    Store: store,' \
  '    Cookie: embeddedidp.CookieConfig{Secure: true},' \
  '    Token: embeddedidp.TokenConfig{SecretKey: []byte("external-consumer-secret-32-bytes")},' \
  '    Audit: idp.NewMemorySink(),' \
  '    RateLimiter: limiter{},' \
  '    ClientAddress: idp.DirectClientAddressResolver{},' \
  '  })' \
  '  if err != nil { return err }' \
  '  _ = provider.Handler()' \
  '  _ = provider.Readiness(ctx)' \
  '  err = provider.Close(ctx)' \
  '  return err' \
  '}' > "$probe_dir/consumer/consumer.go"
cp "$repo_root/ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-consumer/flow_test.go" "$probe_dir/consumer/flow_test.go"

if ! output="$(cd "$probe_dir" && GOWORK=off go test -mod=mod ./consumer 2>&1)"; then
	printf '%s\n' "$output" >&2
	exit 1
fi
printf '%s\n' "$output"
printf 'OK: external production embedding compiles and completes Authorization Code + PKCE\n'
