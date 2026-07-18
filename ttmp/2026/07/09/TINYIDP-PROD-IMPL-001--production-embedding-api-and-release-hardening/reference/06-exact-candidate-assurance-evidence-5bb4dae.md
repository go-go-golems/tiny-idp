---
Title: Exact Candidate Assurance Evidence for 5bb4dae
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://.github/workflows/ci.yml
      Note: |-
        Always-on build, test, static analysis, vulnerability, external-consumer, and recovery gates
        Always-on local gates
    - Path: repo://.github/workflows/release-gates.yml
      Note: |-
        Exact-hash race, fuzz, fault, recovery, and hosted-conformance workflow
        Exact hash and hosted OIDF workflow
    - Path: repo://scripts/run-conformance.sh
      Note: |-
        Local strict-engine protocol and external-consumer gate
        Local strict-engine conformance
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/scripts/02-release-drills.sh
      Note: |-
        Migration, backup, restore, rollback, key, and token-secret drills
        Recovery and rotation drills
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-api-smoke.sh
      Note: |-
        Standalone-module production OIDC flow and toolchain compatibility probe
        Standalone consumer gate
ExternalSources: []
Summary: Reproducible local evidence and explicit missing external rows for code candidate 5bb4dae and its deterministic Linux binary hash.
LastUpdated: 2026-07-10T21:20:00-04:00
WhatFor: Deciding which production-release claims are supported for the post-assurance candidate and which gates still require hosted systems or human authority.
WhenToUse: Before deploying, signing, running hosted OIDF, requesting independent review, or approving the release.
---


# Exact Candidate Assurance Evidence for `5bb4dae`

## Decision

The local candidate passed its build, full race, static analysis, lint,
vulnerability, fuzz, fault-injection, recovery, external-module, local OIDC,
reverse-proxy resolver, and production-host smoke gates. It is **not approved for
production release** because the new exact binary has not completed hosted OIDF
conformance, an installed generic web scanner was unavailable, the artifact has
not been signed/attested by release CI, and independent security/release-owner
approval has not been recorded.

This document freezes evidence for the last code/tooling commit before this
documentation update:

```text
source commit: 5bb4dae6961b23c5bb9e40678316cf15dd3d07b7
build command: GOWORK=off go build -trimpath -buildvcs=false -o /tmp/tinyidp-exact-candidate ./cmd/tinyidp
binary SHA-256: cf43cae64de3c1ac9610eb2bd723eb09189df751a6da422b2f8b80dbf86f43dd
toolchain directive: go1.26.5
module minimum: go1.26.1
```

The two untracked hosted-conformance evidence directories under the older
`TINYIDP-PROD-001` ticket were present before this interval and were not added,
edited, or interpreted as evidence for this candidate.

## Evidence hierarchy

The gate structure follows the research architecture developed in design docs
02–04. Different tools support different claims:

- compilation, vet, lint, and custom AST analyzers support structural claims;
- unit, integration, model, and metamorphic tests support named behavioral
  examples and relations;
- fuzzing supports bounded robustness over generated inputs, not correctness for
  all inputs;
- race detection supports observed Go memory-race freedom for executed paths;
- failpoints support named crash/failure boundaries;
- recovery drills support executable operator procedures;
- local conformance supports the selected strict-engine profile;
- hosted OIDF supports standardized externally orchestrated protocol cases;
- generic web scanning supports common HTTP weakness detection but has weak
  knowledge of OIDC temporal invariants;
- independent review and release-owner approval are human authority, not test
  results.

No single green row is promoted into a claim that belongs to another row.

## Gate ledger

| Gate | Result | Exact evidence | Claim boundary |
|---|---|---|---|
| Reproducible binary | PASS | SHA-256 above | local Linux/amd64 build only |
| Full race | PASS | `GOWORK=off go test -race ./... -count=1` | executed tests only |
| Vet | PASS | `GOWORK=off go vet ./...` | standard vet analyzers |
| Custom AST analysis | PASS | auditlint over `./pkg/... ./internal/...` | documented syntactic precision |
| Pinned lint | PASS after fix | golangci-lint + Glazed lint, zero issues | configured linters only |
| Reachable vulnerabilities | PASS | govulncheck: zero called vulnerabilities | database and call-graph snapshot |
| Parser fuzz | PASS | three 30-second campaigns | bounded local time |
| New invariant fuzz | PASS | max-age, action model, and monitor campaigns | bounded local time |
| Persistence failpoints | PASS | authorization, code exchange, refresh rotation | named failpoints only |
| Recovery drills | PASS | migration, backup, verify, restore, downgrade, key/token rotation | temporary local SQLite host |
| External module | PASS after probe fix | standalone module completed Authorization Code + S256 PKCE | local replace, not published tag |
| Local strict conformance | PASS | `scripts/run-conformance.sh` | repository-selected tests |
| Production TLS host | PASS | HTTP/2, liveness, eight readiness checks, graceful stop | ephemeral local certificate/host |
| Reverse proxy resolver | PASS | trusted/untrusted/malformed XFF unit cases | resolver, not deployed proxy chain |
| Manual generic HTTP probes | PASS | headers, TRACE 405, query bearer 400, bounded oversized body | not a ZAP-equivalent scan |
| Hosted OIDF | NOT RUN | requires deployed exact binary, plan ID, and suite authority | blocking |
| Generic web scanner | NOT RUN | no local ZAP image/binary or equivalent | blocking if required by release profile |
| Signed artifact/provenance | NOT RUN | release workflow not invoked | blocking |
| Independent review | NOT RUN | no reviewer sign-off | blocking |
| Release-owner approval | NOT RUN | no approval record | blocking |

## Detailed results

### Full race and static gates

The full race command passed every repository package, including the new Goja
compiler, verification plan, security trace, custom analyzer fixtures, external
consumer, and SQLite tests. Adapter race execution completed in 22.227 seconds.

Vet and the custom analyzer emitted no diagnostics. The first pinned lint run
found three concrete issues:

```text
internal/securitytrace/trace.go: missing TokenLifecycleDone exhaustive case
internal/securitytrace/trace.go: missing InteractionCreated/TokenLifecycleDone exhaustive cases
internal/fositeadapter/sqlstore.go: capitalized error string
```

Commit `68945c7` added explicit non-applicable event cases and lowercased the
error. The repeated pinned lint gate reported zero issues, and targeted adapter
and monitor tests passed. This was a useful release gate: the monitor already
returned correctly for token events, but explicit exhaustive cases make future
event-schema extension visible to code review.

Govulncheck reported:

```text
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
1 vulnerability exists in imported packages and 14 in required modules,
but current code does not appear to call them.
```

This result must be rerun when the module graph or vulnerability database
changes.

### Fuzz campaigns

| Target | Duration | Executions | New interesting | Result |
|---|---:|---:|---:|---|
| `FuzzIssuerParsing` | 30 s | 892,448 | 24 | PASS |
| `FuzzProductionRedirectURI` | 30 s | 860,573 | 17 | PASS |
| `FuzzArgon2idHashParsing` | 31 s | 685,423 | 7 | PASS |
| `FuzzParseMaxAgeAcceptsOnlyBoundedDecimal` | 10 s | 44,544 | 1 | PASS |
| `FuzzInteractionModelActionSequences` | 10 s | 44,309 | 0 | PASS |
| `FuzzMonitorEventSequences` | 11 s | 5,021 | 4 | PASS |

Two adapter fuzz commands were initially started concurrently. Go fuzz workers
for the same package stalled before returning results. They exited while the
termination command was being issued, and both targets were then rerun
sequentially with the successful counts above. The concurrent orchestration
attempt is not counted as test evidence.

### Fault injection and recovery

The exact-candidate targeted failpoint run passed:

- seven authorization persistence rollback points;
- eight code-redemption transaction points;
- ten refresh-rotation transaction points with retryability.

The SQLite fault suite passed concurrent failed-login accounting, failed
migration recording, checksum mismatch refusal, canceled backup publication,
busy-context deadlines, backup during concurrent writes, and disk-full atomic
publication.

The release drill created a schema-v6 database, provisioned a client/user/key,
ran doctor, created and verified an online backup, rotated the signing key,
created a post-backup client, restored the backup, proved the later client did
not survive restore, reran doctor, and passed downgrade, rollback, signing-key,
and token-secret rotation tests.

### External module packaging

The first exact external-consumer run exited without visible output because the
script captured `go test` stderr in a command substitution and `set -e` exited
before printing it. The harness was changed to print captured output on failure.
The revealed error was:

```text
module tiny-idp requires go >= 1.26.1 (running go 1.25.11)
```

Adding go-go-goja legitimately raised tiny-idp's module minimum to 1.26.1, while
the probe still generated a 1.25.11 consumer module. Commit `5bb4dae` updates the
probe minimum to 1.26.1 and preserves failure output. The repeated standalone
module test compiled and completed Authorization Code + S256 PKCE.

This is not a backward-compatibility adapter. A dependency requiring Go 1.26.1
cannot be supported truthfully by a generated 1.25.11 consumer declaration.

### Production host and HTTP surface

The production host ran in tmux on `127.0.0.1:18443` with:

- a generated RSA certificate whose SAN covered localhost and loopback;
- owner-only TLS key and 32-byte token-secret files;
- provisioned schema-v6 SQLite and signing key;
- synchronous audit path;
- trusted loopback proxy CIDR;
- scheduled maintenance and explicit production settings.

Observed results:

- HTTP/2 liveness: 200 and ready;
- HTTP/2 readiness: 200 with lifecycle, store, schema, signing key, token secret,
  audit, limiter, and maintenance checks ready;
- discovery: 200 JSON with CSP, no-referrer, nosniff, and frame denial;
- TRACE authorization request: 405;
- UserInfo query bearer: 400;
- a 1.1 MB streamed POST was bounded and returned 400;
- trusted, untrusted, and malformed forwarding cases passed resolver tests;
- SIGINT stopped the tmux-hosted server and the port was no longer bound.

The first oversized-body command attempted to place 1.1 MB in one shell
argument and failed with `argument list too long`. Streaming from stdin tested
the intended host behavior successfully.

### Hosted and generic web gaps

Hosted OIDF was not invoked. The workflow requires a deployed issuer backed by
this exact binary, a plan ID, login credentials, and either suite API token or
session authority. This interval did not read environment variables or reuse
older hosted result directories as if they belonged to the new hash.

No local ZAP binary/image, Nikto, Nuclei, testssl, or sslscan installation was
available. Pulling and introducing a new scanner was not necessary to establish
the already-defined blocking row. Manual HTTP probes are useful but are not
renamed as a generic scanner pass.

## Release continuation

1. Build and deploy commit `5bb4dae` with the recorded deterministic command and
   verify SHA-256 `cf43cae…f43dd` at the target.
2. Run the manual release workflow with that exact hash and an authorized hosted
   plan.
3. Preserve plan metadata, per-test results/logs, suite version, deployed hash,
   timestamps, and any warning adjudication.
4. Run the selected generic web scanner through the actual reverse-proxy/TLS
   topology and triage results without weakening OIDC semantics.
5. Produce signed checksums, SBOM, provenance, module graph, and license packet.
6. Obtain independent security/code review.
7. Record release-owner approval only after all blocking rows are green or have
   explicit, owner-signed, time-bounded exceptions.

## Final status

```text
local exact-candidate gates: PASS
hosted OIDF exact-candidate gate: MISSING / BLOCKING
generic web scanner: MISSING if required / NOT SUBSTITUTED
signed artifact and provenance: MISSING / BLOCKING
independent security review: MISSING / BLOCKING
release-owner approval: MISSING / BLOCKING
production release decision: NOT APPROVED
```
