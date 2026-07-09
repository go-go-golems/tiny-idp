#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

section() {
  printf '\n## %s\n\n' "$1"
}

section "Repository"
printf 'commit: '
git rev-parse HEAD
printf 'go: '
go version
printf 'go source files: '
find cmd internal pkg -type f -name '*.go' | wc -l
printf 'go source lines: '
find cmd internal pkg -type f -name '*.go' -print0 | xargs -0 wc -l | tail -1

section "Packages"
GOWORK=off go list -buildvcs=false -f '{{.ImportPath}}' ./...

section "HTTP routes and server construction"
rg -n 'HandleFunc|Handle\(|ListenAndServe|http\.Server|Shutdown\(|ReadHeaderTimeout|ReadTimeout|WriteTimeout|IdleTimeout|MaxHeaderBytes' cmd internal pkg -S || true

section "Production defaults and trust controls"
rg -n 'ProductionMode|NoopSink|AllowAllRateLimiter|RemoteAddr|X-Forwarded|Cookie|SameSite|Secure|GlobalSecret|SecretKey|ClientSecrets' cmd internal pkg -S || true

section "Persistence and transaction boundaries"
rg -n 'BeginTx|BEGIN|COMMIT|ROLLBACK|INSERT OR REPLACE|ExecContext|QueryContext|QueryRowContext|io\.Copy|PRAGMA|VACUUM|Backup' internal/store internal/fositeadapter internal/admin -S || true

section "Potential process and randomness hazards"
rg -n 'panic\(|log\.Fatal|os\.Exit|rand\.Read|_, _ = rand\.Read|go func\(' cmd internal pkg -S || true

section "Environment access"
rg -n 'Getenv|LookupEnv|Environ|AutomaticEnv|SetEnv|TINYIDP_' cmd internal pkg -S || true

section "Tests, fuzzing, and benchmarks"
printf 'unit/integration tests: '
rg -n '^func Test' --glob '*_test.go' cmd internal pkg | wc -l
printf 'fuzz targets in product tree: '
count="$(rg -n '^func Fuzz' --glob '*_test.go' cmd internal pkg | wc -l || true)"
printf '%s\n' "$count"
printf 'benchmarks in product tree: '
count="$(rg -n '^func Benchmark' --glob '*_test.go' cmd internal pkg | wc -l || true)"
printf '%s\n' "$count"

section "CI and release automation"
find .github -maxdepth 3 -type f -print 2>/dev/null || true
rg -n 'govulncheck|gosec|staticcheck|golangci|go test|go build|race|fuzz|conformance' Makefile .github scripts -S 2>/dev/null || true

section "Worktree state"
git status --short
