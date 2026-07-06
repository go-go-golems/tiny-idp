# Tasks

## Phase 0 — Ticket and design package

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing device authorization grant design and implementation guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-DEVICE-001 --stale-after 30`
- [x] Commit ticket design package

## Phase 1 — Data model and route skeleton

- [x] Add `deviceGrant` type and device-grant constants
- [x] Add `deviceGrants map[string]deviceGrant` to `server.Server`
- [x] Initialize `deviceGrants` in `server.New`
- [x] Initialize `deviceGrants` in server test helpers
- [x] Register `/device_authorization` under root and issuer path prefixes
- [x] Register `/device` under root and issuer path prefixes
- [x] Add method-not-allowed handling for new endpoints

## Phase 2 — Device authorization endpoint

- [x] Implement `POST /device_authorization`
- [x] Validate `client_id`
- [x] Validate requested scope against the client
- [x] Generate high-entropy `device_code`
- [x] Generate human-friendly `user_code`
- [x] Store pending device grant with expiry and interval
- [x] Return `device_code`, `user_code`, `verification_uri`, `verification_uri_complete`, `expires_in`, and `interval`
- [x] Add no-store response headers

## Phase 3 — User code helpers

- [x] Implement user-code generation helper
- [x] Implement user-code normalization helper
- [x] Add tests for user-code format
- [x] Add tests for user-code normalization
- [x] Decide whether to add a secondary user-code index or scan map under lock

## Phase 4 — Verification UI and approval

- [x] Add `/device` GET approval page
- [x] Prefill `user_code` from query parameter when present
- [x] Add `/device` POST approval handling
- [x] Add approve and deny actions
- [x] Validate unknown and expired user codes
- [x] Resolve login through scenario registry
- [x] Validate fixture password using existing semantics
- [x] Mark grant approved with user, scenario, and auth time
- [x] Mark grant denied without user state
- [x] Ensure wrong password leaves grant pending

## Phase 5 — Token polling grant

- [x] Add device-code grant type constant
- [x] Add `urn:ietf:params:oauth:grant-type:device_code` dispatch in `/token`
- [x] Implement `tokenDeviceCode`
- [x] Return `authorization_pending` for pending grants
- [x] Return `slow_down` for too-frequent polling
- [x] Return `expired_token` for expired grants
- [x] Return `access_denied` for denied grants
- [x] Return `invalid_grant` for unknown or client-mismatched device codes
- [x] Issue access token after approval
- [x] Issue ID token when scope includes `openid`
- [x] Issue refresh token when scope includes `offline_access`
- [x] Delete device grant after successful token issuance

## Phase 6 — Discovery, debug, and docs

- [x] Add `device_authorization_endpoint` to discovery
- [x] Add device-code grant type to `grant_types_supported`
- [x] Add `/debug/device-grants` with redacted device-code prefixes
- [x] Update README
- [x] Update `cmd/tinyidp/doc/pages/user-guide.md`
- [x] Update `cmd/tinyidp/doc/pages/developer-guide.md`
- [x] Update `cmd/tinyidp/doc/pages/reference.md`
- [x] Add `cmd/tinyidp/doc/pages/tutorial-device-authorization.md`

## Phase 7 — Tests

- [x] Test discovery metadata advertises device support
- [x] Test device authorization rejects unknown clients
- [x] Test device authorization rejects disallowed scopes
- [x] Test device authorization success response shape
- [x] Test initial token poll returns `authorization_pending`
- [x] Test aggressive polling returns `slow_down`
- [x] Test expired grants return `expired_token`
- [x] Test denied grants return `access_denied`
- [x] Test approved grants return access token and ID token
- [x] Test `offline_access` returns refresh token
- [x] Test successful device code is one-time use
- [x] Test client mismatch returns `invalid_grant`
- [x] Test wrong fixture password leaves grant pending
- [x] Test seeded Alice approval yields fixed subject and claims
- [x] Test path-based issuer routes include device endpoints

## Phase 8 — Validation and handoff

- [x] Run `go test ./internal/server -count=1`
- [x] Run `GOWORK=off go test ./... -count=1`
- [x] Run `GOWORK=off go build ./cmd/tinyidp`
- [x] Smoke-render new help page with `go run ./cmd/tinyidp help tutorial-device-authorization`
- [x] Run manual `curl` device flow smoke
- [x] Update implementation diary with exact command output
- [x] Update changelog and doc relations
- [x] Run `docmgr doctor --ticket TINYIDP-DEVICE-001 --stale-after 30`
- [ ] Commit implementation and docs
