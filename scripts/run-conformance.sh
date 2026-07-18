#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "== tiny-idp strict-engine local conformance checks =="

echo "-- full Go test suite"
go test ./...

echo "-- strict Fosite protocol/security tests"
go test ./internal/fositeadapter -run 'TestStrictAuthorizationCodeFlow|TestFositeSQLiteStoreSurvivesProviderRestart|TestFositeSQLiteRefreshTokenReuseIsRejected|TestBrowserSessionSilentAuthorizeAndPromptNone|TestAuthorizeRequiresCSRFAndEmitsAudit|TestAuditReasonsUseStableCodes|TestSecurityHeadersOnDiscovery|TestStrictProviderHasNoDebugRoute' -count=1

echo "-- durable storage/key invariants"
go test ./pkg/sqlitestore ./internal/store/memory ./internal/keys -count=1

echo "-- repository-specific Go AST/analysis checks"
go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -- ./pkg/... ./internal/...

echo "-- external public-API consumer flow"
bash ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-api-smoke.sh

echo "OK: local strict-engine conformance checks passed"
