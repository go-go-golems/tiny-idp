#!/usr/bin/env bash
set -euo pipefail

# Exercise password-login context creation, opt-in label-policy validation,
# and RP-initiated global browser-context logout.
repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$repo_root"

go test ./internal/fositeadapter -run 'TestPersistBrowserSessionRefreshesSubjectAndBoundsRememberedAccounts|TestOptInPasswordLoginCreatesRememberedBrowserSession|TestAccountChooserRememberingRequiresLabelPolicy|TestEndSessionRevokesBrowserContextAndClearsItsCookie' -count=1 -v
go test ./pkg/embeddedidp -run TestAccountChooserConfigurationValidation -count=1 -v
