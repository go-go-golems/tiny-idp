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
- [ ] Add `DPoPJKT` to access-token state
- [ ] Add `DPoPJKT` to refresh-token state
- [ ] Add `dpopReplay` replay cache to `Server`
- [ ] Initialize replay cache in constructors and tests
- [ ] Clear replay cache in debug reset
- [ ] Advertise `dpop_signing_alg_values_supported`
- [ ] Implement `internal/server/dpop.go`
- [ ] Parse compact proof JWTs
- [ ] Validate proof header `typ=dpop+jwt`
- [ ] Reject unsupported or unsafe proof algorithms
- [ ] Parse ES256 public JWKs
- [ ] Parse RS256 public JWKs
- [ ] Reject private JWK members
- [ ] Verify ES256 proof signatures
- [ ] Verify RS256 proof signatures
- [ ] Compute RFC 7638 JWK thumbprints
- [ ] Validate `htm`
- [ ] Validate `htu`
- [ ] Validate `iat` freshness
- [ ] Validate non-empty `jti`
- [ ] Record and reject replayed proofs
- [ ] Validate `ath` when an access token is supplied
- [ ] Add proof helper tests
- [ ] Bind authorization-code token responses when `DPoP` header is present
- [ ] Bind device-code token responses when `DPoP` header is present
- [ ] Return `token_type: DPoP` for bound access tokens
- [ ] Preserve bearer response behavior without `DPoP` header
- [ ] Bind refresh tokens issued from DPoP flows
- [ ] Require matching DPoP proof for bound refresh tokens
- [ ] Allow unbound refresh tokens to stay bearer unless a DPoP proof upgrades them
- [ ] Enforce `Authorization: DPoP` for DPoP-bound userinfo requests
- [ ] Require `ath` for DPoP-bound userinfo requests
- [ ] Preserve bearer userinfo behavior for unbound access tokens
- [ ] Add token endpoint DPoP tests
- [ ] Add userinfo DPoP tests
- [ ] Add device-code DPoP test coverage
- [ ] Update README
- [ ] Update Glazed reference docs
- [ ] Add or update DPoP tutorial/help page
- [ ] Run `go test ./internal/server -run 'TestDPoP|TestDevice' -count=1`
- [ ] Run `GOWORK=off go test ./... -count=1`
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`
- [ ] Smoke-render DPoP help page
- [ ] Run manual DPoP curl or Go smoke
- [ ] Update diary with exact validation output
- [ ] Run `docmgr doctor --ticket TINYIDP-DPOP-001 --stale-after 30`
- [ ] Commit implementation and docs
- [ ] Upload final implementation bundle to reMarkable
- [ ] Close ticket
