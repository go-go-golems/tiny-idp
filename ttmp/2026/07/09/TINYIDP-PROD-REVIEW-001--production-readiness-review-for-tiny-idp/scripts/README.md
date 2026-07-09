# Production review tools

These tools make the production-readiness findings reproducible without
changing the product implementation.

## Static surface audit

```bash
./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/static-surface-audit.sh
```

The script inventories packages, routes, HTTP server construction, security-
sensitive defaults, persistence operations, environment access, tests, and
automation. It is intentionally based on `rg` and standard Go tooling so it can
run in CI without extra dependencies.

## Repository-specific Go analyzer

```bash
go run \
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint \
  ./cmd/... ./internal/... ./pkg/...
```

`auditlint` is a `go/analysis` multichecker. Its analyzers use type information
and AST structure rather than text matching. Stable diagnostic categories
cover public APIs that expose `internal/` types, ignored `crypto/rand` errors,
raw SQLite file backup, implicit allow-all/no-op production controls, remote
addresses (including ephemeral ports) used as rate-limit keys, unused public
configuration fields, ignored audit-delivery errors, `http.ListenAndServe`
without explicit server hardening, and multi-mutation persistence functions
without a transaction boundary.

## External API smoke probe

```bash
./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-api-smoke.sh
```

The script creates a temporary external module and attempts to compile the
production embedding example. It succeeds as an audit probe only when the Go
compiler reproduces the expected `use of internal package ... not allowed`
failure. Once the public storage boundary is fixed, this probe should be
replaced by a positive external-consumer integration test.

## SQLite live-backup probe

```bash
go run \
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/sqlite-backup-probe.go \
  --log-level info
```

This puts the source database into WAL mode, commits a client that remains in
the WAL, calls the product's file-copy backup function, and demonstrates that
the resulting backup can open successfully while silently omitting committed
data.

## Parser fuzz harnesses

```bash
go test \
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts \
  -run '^$' -fuzz FuzzIssuerParsing -fuzztime 10s
```

The package also contains fuzz targets for redirect URI validation and encoded
Argon2id hash parsing. Run one fuzz target at a time, as required by `go test`.

## Strict runtime instrumentation

```bash
mkdir -p ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime
go run \
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/runtime-probe \
  --requests 40 --concurrency 4 \
  --output ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/events.ndjson \
  --cpu-profile ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/cpu.pprof \
  --heap-profile ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/heap.pprof

go run \
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/runtime-analyze \
  --input ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/events.ndjson \
  --output ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/summary.md
```

The probe provisions a temporary production-mode SQLite provider, performs a
complete Authorization Code + PKCE login/token/userinfo/refresh flow, then
applies bounded concurrent read traffic. It emits NDJSON request events,
runtime metric snapshots, SQL-pool snapshots, and audit counts. Optional CPU
and heap profiles support deeper `go tool pprof` investigation. The analyzer
turns the NDJSON into route latency percentiles, status distributions, runtime
deltas, and database pool observations.
