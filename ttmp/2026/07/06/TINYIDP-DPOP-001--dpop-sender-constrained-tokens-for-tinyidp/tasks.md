---
Title: Tasks
Ticket: TINYIDP-DPOP-001
Status: active
Topics:
  - oidc
  - auth
  - identity
  - testing
  - go
DocType: reference
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation checklist for DPoP sender-constrained token support in tinyidp.
LastUpdated: 2026-07-06T00:00:00-04:00
WhatFor: Track design, implementation, validation, docs, upload, and closure for TINYIDP-DPOP-001.
WhenToUse: Use while implementing or reviewing tinyidp DPoP support.
---

# Tasks

- [x] Create DPoP ticket and implementation guide
- [x] Add implementation diary
- [x] Relate design-relevant source files
- [x] Upload design bundle to reMarkable
- [x] Add `DPoPJKT` to access-token state
- [x] Add `DPoPJKT` to refresh-token state
- [x] Add `dpopReplay` replay cache to `Server`
- [x] Initialize replay cache in constructors and tests
- [x] Clear replay cache in debug reset
- [x] Advertise `dpop_signing_alg_values_supported`
- [x] Implement `internal/server/dpop.go`
- [x] Parse compact proof JWTs
- [x] Validate proof header `typ=dpop+jwt`
- [x] Reject unsupported or unsafe proof algorithms
- [x] Parse ES256 public JWKs
- [x] Parse RS256 public JWKs
- [x] Reject private JWK members
- [x] Verify ES256 proof signatures
- [x] Verify RS256 proof signatures
- [x] Compute RFC 7638 JWK thumbprints
- [x] Validate `htm`
- [x] Validate `htu`
- [x] Validate `iat` freshness
- [x] Validate non-empty `jti`
- [x] Record and reject replayed proofs
- [x] Validate `ath` when an access token is supplied
- [x] Add proof helper tests
- [x] Bind authorization-code token responses when `DPoP` header is present
- [x] Bind device-code token responses when `DPoP` header is present
- [x] Return `token_type: DPoP` for bound access tokens
- [x] Preserve bearer response behavior without `DPoP` header
- [x] Bind refresh tokens issued from DPoP flows
- [x] Require matching DPoP proof for bound refresh tokens
- [x] Allow unbound refresh tokens to stay bearer unless a DPoP proof upgrades them
- [x] Enforce `Authorization: DPoP` for DPoP-bound userinfo requests
- [x] Require `ath` for DPoP-bound userinfo requests
- [x] Preserve bearer userinfo behavior for unbound access tokens
- [x] Add token endpoint DPoP tests
- [x] Add userinfo DPoP tests
- [ ] Add device-code DPoP test coverage
- [ ] Update README
- [ ] Update Glazed reference docs
- [ ] Add or update DPoP tutorial/help page
- [x] Run `go test ./internal/server -run 'TestDPoP|TestDevice' -count=1`
- [ ] Run `GOWORK=off go test ./... -count=1`
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`
- [ ] Smoke-render DPoP help page
- [ ] Run manual DPoP curl or Go smoke
- [ ] Update diary with exact validation output
- [ ] Run `docmgr doctor --ticket TINYIDP-DPOP-001 --stale-after 30`
- [ ] Commit implementation and docs
- [ ] Upload final implementation bundle to reMarkable
- [ ] Close ticket
