#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "== tiny-idp strict-engine local conformance checks =="

echo "-- full Go test suite"
go test ./...

echo "-- strict Fosite protocol/security tests"
go test ./internal/fositeadapter -run 'TestStrictAuthorizationCodeFlow|TestFositeSQLiteStoreSurvivesProviderRestart|TestFositeSQLiteRefreshTokenReuseIsRejected|TestBrowserSessionSilentAuthorizeAndPromptNone|TestAuthorizeRequiresCSRFAndEmitsAudit|TestAuditReasonsUseStableCodes|TestSecurityHeadersOnDiscovery|TestStrictProviderHasNoDebugRoute' -count=1

echo "-- durable storage/key invariants"
go test ./internal/storage ./internal/store/memory ./internal/store/sqlite ./internal/keys -count=1

echo "OK: local strict-engine conformance checks passed"
