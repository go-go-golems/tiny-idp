# Tasks

## Phase 0 — Ticket and design package

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing device authorization grant design and implementation guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-DEVICE-001 --stale-after 30`
- [x] Commit ticket design package

## Phase 1 — Data model and route skeleton

- [ ] Add `deviceGrant` type and device-grant constants
- [ ] Add `deviceGrants map[string]deviceGrant` to `server.Server`
- [ ] Initialize `deviceGrants` in `server.New`
- [ ] Initialize `deviceGrants` in server test helpers
- [ ] Register `/device_authorization` under root and issuer path prefixes
- [ ] Register `/device` under root and issuer path prefixes
- [ ] Add method-not-allowed handling for new endpoints

## Phase 2 — Device authorization endpoint

- [ ] Implement `POST /device_authorization`
- [ ] Validate `client_id`
- [ ] Validate requested scope against the client
- [ ] Generate high-entropy `device_code`
- [ ] Generate human-friendly `user_code`
- [ ] Store pending device grant with expiry and interval
- [ ] Return `device_code`, `user_code`, `verification_uri`, `verification_uri_complete`, `expires_in`, and `interval`
- [ ] Add no-store response headers

## Phase 3 — User code helpers

- [ ] Implement user-code generation helper
- [ ] Implement user-code normalization helper
- [ ] Add tests for user-code format
- [ ] Add tests for user-code normalization
- [ ] Decide whether to add a secondary user-code index or scan map under lock

## Phase 4 — Verification UI and approval

- [ ] Add `/device` GET approval page
- [ ] Prefill `user_code` from query parameter when present
- [ ] Add `/device` POST approval handling
- [ ] Add approve and deny actions
- [ ] Validate unknown and expired user codes
- [ ] Resolve login through scenario registry
- [ ] Validate fixture password using existing semantics
- [ ] Mark grant approved with user, scenario, and auth time
- [ ] Mark grant denied without user state
- [ ] Ensure wrong password leaves grant pending

## Phase 5 — Token polling grant

- [ ] Add device-code grant type constant
- [ ] Add `urn:ietf:params:oauth:grant-type:device_code` dispatch in `/token`
- [ ] Implement `tokenDeviceCode`
- [ ] Return `authorization_pending` for pending grants
- [ ] Return `slow_down` for too-frequent polling
- [ ] Return `expired_token` for expired grants
- [ ] Return `access_denied` for denied grants
- [ ] Return `invalid_grant` for unknown or client-mismatched device codes
- [ ] Issue access token after approval
- [ ] Issue ID token when scope includes `openid`
- [ ] Issue refresh token when scope includes `offline_access`
- [ ] Delete device grant after successful token issuance

## Phase 6 — Discovery, debug, and docs

- [ ] Add `device_authorization_endpoint` to discovery
- [ ] Add device-code grant type to `grant_types_supported`
- [ ] Add `/debug/device-grants` with redacted device-code prefixes
- [ ] Update README
- [ ] Update `cmd/tinyidp/doc/pages/user-guide.md`
- [ ] Update `cmd/tinyidp/doc/pages/developer-guide.md`
- [ ] Update `cmd/tinyidp/doc/pages/reference.md`
- [ ] Add `cmd/tinyidp/doc/pages/tutorial-device-authorization.md`

## Phase 7 — Tests

- [ ] Test discovery metadata advertises device support
- [ ] Test device authorization rejects unknown clients
- [ ] Test device authorization rejects disallowed scopes
- [ ] Test device authorization success response shape
- [ ] Test initial token poll returns `authorization_pending`
- [ ] Test aggressive polling returns `slow_down`
- [ ] Test expired grants return `expired_token`
- [ ] Test denied grants return `access_denied`
- [ ] Test approved grants return access token and ID token
- [ ] Test `offline_access` returns refresh token
- [ ] Test successful device code is one-time use
- [ ] Test client mismatch returns `invalid_grant`
- [ ] Test wrong fixture password leaves grant pending
- [ ] Test seeded Alice approval yields fixed subject and claims
- [ ] Test path-based issuer routes include device endpoints

## Phase 8 — Validation and handoff

- [ ] Run `go test ./internal/server -count=1`
- [ ] Run `GOWORK=off go test ./... -count=1`
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`
- [ ] Smoke-render new help page with `go run ./cmd/tinyidp help tutorial-device-authorization`
- [ ] Run manual `curl` device flow smoke
- [ ] Update implementation diary with exact command output
- [ ] Update changelog and doc relations
- [ ] Run `docmgr doctor --ticket TINYIDP-DEVICE-001 --stale-after 30`
- [ ] Commit implementation and docs
