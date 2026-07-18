---
Title: Production Trust Boundaries and Release Security
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - architecture
    - auth
    - identity
    - oidc
    - research
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://.github/workflows/release-gates.yml
      Note: |-
        Exact artifact and hosted conformance gates
        External release evidence
    - Path: repo://internal/cmds/serve_production.go
      Note: |-
        TLS host, limits, proxy trust, maintenance, and shutdown
        Production host
    - Path: repo://pkg/embeddedidp/options.go
      Note: |-
        Production validation and readiness ownership
        Production validation
    - Path: repo://pkg/idp/contracts.go
      Note: |-
        Audit, limiter, and trusted proxy contracts
        Proxy audit and limiter contracts
ExternalSources: []
Summary: Host, proxy, audit, key, recovery, dependency, artifact, conformance, and human approval requirements that complete the provider's production security case.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Teaching why a correct handler is necessary but insufficient for a production identity service.
WhenToUse: Before deployment, configuration changes, incident response, key rotation, release approval, or claims about production readiness.
---


# Production Trust Boundaries and Release Security

## The host owns security properties

An embeddable provider cannot choose the listener, TLS configuration, HTTP
timeouts, body limit, filesystem permissions, process signals, reverse proxy,
secret delivery, or deployment topology. `serve-production` demonstrates one
valid host contract; applications embedding the library must supply equivalent
controls.

Production validation rejects missing durable audit, limiter, address resolver,
secure cookies, strong token secret, password controls, valid signing key, and
maintenance configuration. Fail-closed construction is preferable to silently
installing development defaults.

## Proxy trust

`X-Forwarded-For` is attacker-controlled unless the immediate peer is a trusted
proxy. The resolver walks the chain from the peer toward the client, accepts only
configured CIDRs and bounded hops, and ignores forwarding headers from untrusted
peers. The proxy must sanitize inbound forwarding headers and protect its link to
the application.

Client address is an abuse-control input, not identity. Login limiting combines
normalized address with client/account dimensions while avoiding unbounded keys
derived solely from unauthenticated claimed client IDs.

## Audit, security events, and readiness

Audit records support accountability and incident response. Security events
support machine temporal verification. Logs support diagnosis. Metrics support
aggregate operations. They are separate schemas and failure modes.

The production audit sink appends synchronously and fsyncs. Delivery failure is
observable. Readiness includes lifecycle, store, schema, active signing key,
token secret, audit, limiter, and maintenance recency. Liveness answers whether
the process functions; readiness answers whether it should receive security
traffic.

## Keys and recovery

Planned signing rotation retains retired verification keys through the maximum
ID-token lifetime plus skew. Emergency purge deliberately removes that overlap
after compromise. Token-secret rotation invalidates opaque tokens immediately.
These operations have different availability goals and must not share one vague
“rotate” procedure.

Recovery requires verified online backup, manifest/checksum validation, offline
restore, rollback preservation, doctor checks, and a fresh post-recovery backup.
A restore drill is evidence; an unexecuted runbook is not.

## Release claim graph

```text
source commit
  -> reproducible binary hash
  -> tests/static/race/fuzz/fault/recovery
  -> SBOM + module graph + vulnerability result
  -> signed checksum + provenance
  -> deployed exact hash
  -> hosted OIDF + proxy/TLS + generic web checks
  -> independent review
  -> release-owner approval
```

Later nodes do not replace earlier ones. Hosted conformance does not prove audit
durability. A signature does not prove protocol correctness. Local tests do not
authorize production release.

The exact candidate ledger intentionally leaves hosted OIDF, generic scanning,
signing/provenance, independent review, and owner approval open. Honest missing
evidence is a security control because it prevents local success from being
reinterpreted as organizational approval.

## Exercises

1. Classify each readiness component as provider, store, or host responsibility.
2. Explain why accepting the leftmost XFF value is unsafe.
3. Compare planned key rotation, emergency key purge, and token-secret rotation.
4. Explain what hosted OIDF does not test.
5. Given a binary hash and green local suite, list every remaining authority
   required before production approval.

## Retrospective: production readiness changed the library boundary

The first production review found that tiny-idp's useful strict provider lived
behind internal packages and exposed internal interfaces. An external host could
not construct it without importing forbidden implementation details. The review
therefore began with API ownership, not deployment YAML.

Diary Steps 7 through 10 moved identity, policy, and persistence contracts into
public packages, introduced a context-aware provider lifecycle, and built a
standalone external consumer. This established a concrete definition of
embeddability: a separate Go module can provision public SQLite, construct the
production provider, obtain an `http.Handler`, inspect readiness, complete OIDC,
and close it without importing `internal/`.

The production boundary then expanded. Store operations, backup, password work,
proxy trust, audit, readiness, maintenance, host TLS, release evidence, and
incident procedures were added because a handler alone cannot own them.

## Public package ownership

### `pkg/embeddedidp`

This package is the host-facing construction boundary. `Options` contains
issuer, mode, store, cookie policy, token secret, audit, consent, rate limiter,
client-address resolver, authenticator, password policy/work limits, and
maintenance settings.

`New(ctx, opts)` validates the complete production preflight, constructs the
internal strict adapter, runs initial maintenance, and returns a lifecycle
object. The returned provider exposes handler, readiness, liveness, maintenance,
password-work stats, and close.

The package deliberately does not export Fosite types. Fosite remains the
protocol engine behind the adapter and can evolve without becoming the public
host contract.

### `pkg/idp`

This package defines host policy and operational contracts: audit sink/reporting,
consent, rate limiting, client-address resolution, authentication, password
acceptance, password-work reporting, readiness reports, and maintenance status.

Interfaces include production-readiness reporters so construction can reject a
control that is present but explicitly development-only.

### `pkg/idpstore`

This package defines durable domain types, narrow store interfaces, transaction
views, atomic security operations, persistence/schema reporting, and maintenance.
It contains no SQLite dependency.

### `pkg/sqlitestore`

This is the supported durable implementation. It owns migrations, filesystem
mode, WAL/connection topology, transactions, atomic transitions, online backup,
verification, restore, and maintenance.

## Context-aware lifecycle

Construction accepts `context.Context`. A nil or already-canceled context is
rejected. Store queries and initial validation use the caller's cancellation
boundary. `Close` is explicit, even where current cleanup is small, so future
background resources do not require an API break.

The host owns process lifetime. The provider does not call `ListenAndServe`,
install signal handlers, or terminate the process.

## Production preflight in exact checks

`Options.Validate` normalizes mode and then checks the issuer against production
rules. It requires a store and loads all clients so each client can be validated
under the same mode.

Maintenance retention derives from client token lifetimes. Protocol state must
outlive the maximum refresh-token lifetime plus expired-record retention.
Signing-key retention must outlive the maximum ID-token lifetime plus clock skew.

Production mode requires:

- token secret of at least 32 bytes;
- secure cookies and a supported SameSite policy;
- a durable production-ready audit reporter whose health is ready;
- a configured production-ready rate limiter;
- a configured production-ready client-address resolver;
- NIST-aligned password acceptance bounds and blocklist;
- bounded concurrent password work;
- production-ready custom authenticator with password-work reporting when
  supplied;
- valid client redirect, scope, TTL, and PKCE settings;
- an active signing key and supported schema through provider readiness.

Development defaults remain available only in development mode. The custom
`tinyidpsecuritydefault` analyzer requires explicit directives where no-op or
allow-all controls are installed behind that boundary.

## Issuer and TLS semantics

The issuer is an identity and URL origin, not just a base string. Discovery,
JWKS, authorization, token, and UserInfo metadata derive from it. Production
requires HTTPS and canonical parsing. Redirects and token `iss` claims must agree
with the deployed externally reachable issuer.

`serve-production` configures TLS 1.2 minimum and requires certificate/key paths.
The example can terminate TLS directly. An embedding behind a reverse proxy may
serve plain HTTP internally only when the external issuer, secure cookie
behavior, sanitized forwarding metadata, and protected proxy link remain
correct.

## HTTP server hardening

The production command builds an explicit `http.Server` with:

- read-header timeout;
- full read timeout;
- write timeout;
- idle timeout;
- maximum header bytes;
- `MaxBytesHandler` request-body bound;
- TLS configuration;
- graceful shutdown deadline.

These controls limit slow clients, oversized requests, resource retention, and
unbounded shutdown. They do not replace upstream connection limits, DDoS
protection, or platform quotas.

The exact-candidate host returned 400 for a streamed 1.1 MiB oversized form,
demonstrating bounded parsing. A reviewer may prefer normalized 413 semantics;
the security property currently evidenced is bounded work, not the status code.

## Request routing and generic web controls

The strict handler uses `http.ServeMux` and mounts discovery, JWKS, authorize,
token, UserInfo, health, and readiness. It has no debug route.

Security middleware sets content-security policy, frame denial, no-referrer, and
nosniff where appropriate. UserInfo responses are no-store. Unsupported methods
return method errors. TRACE to authorization returned 405 in the production
host smoke.

Generic web scanners can detect missing headers and common HTTP weaknesses. They
have weak knowledge of OIDC state histories and cannot replace semantic tests.
The exact-candidate ledger leaves the scanner row open because no installed ZAP
or equivalent was run.

## Reverse proxy trust in current code

`DirectClientAddressResolver` parses the immediate peer and is correct when no
trusted proxy exists. It ignores forwarded headers.

`TrustedProxyResolver` parses configured CIDR ranges and maximum hops. On each
request it:

1. parses the immediate `RemoteAddr` host;
2. returns that address if the peer is untrusted;
3. splits X-Forwarded-For only for a trusted peer;
4. validates every address;
5. walks right-to-left through trusted proxy hops;
6. returns the first untrusted address as the client;
7. rejects malformed or excessive chains.

Taking the leftmost value is unsafe because an attacker can prepend values and a
proxy may append rather than replace. Trust must begin at the known immediate
peer.

The production host accepts repeatable trusted CIDRs and max hops. An empty CIDR
list selects direct resolution instead of trusting all proxies.

## Rate limiting as host policy

The fixed-window production limiter bounds login attempts across account,
verified client, and normalized address dimensions. Pre-authentication token
limiting uses address before client authentication and verified client identity
afterward.

The original code used `RemoteAddr` including ephemeral port, allowing each
connection to create a new bucket. Later review removed claimed-client
pre-authentication buckets to avoid attacker-controlled cardinality.

Rate limiting is abuse resistance, not authentication. It must not turn
attacker-controlled fields into unbounded memory and must expose production
readiness.

## Password acceptance and password work

Password acceptance policy governs minimum/maximum characters and blocklist at
creation/change. Password hashing uses Argon2id parameters stored with each
credential. Authentication work is memory-hard and can exhaust a host even when
credentials are invalid.

The password-work controller bounds concurrent hash operations, records waits,
saturation, rejection, completions, and Argon duration, and accepts context
cancellation. Production preflight requires a positive capacity and compatible
reporting from custom authenticators.

The runtime probe observed capacity two, saturation, admission wait, and no
rejections in the recorded load. This supports capacity planning for that host,
not a universal safe rate.

NIST SP 800-63B-4 informed acceptance policy and verifier behavior. The local
implementation documents which recommendations are directly implemented and
which deployment/account-recovery controls belong outside the IdP core.

## Audit design

The production sink appends one JSON object per line and calls `fsync` before
returning. This supplies backpressure and avoids an in-memory queue whose
acknowledged events can be lost on process failure.

Events contain stable names, result, reason, client, subject where permitted,
and timestamp. Authentication failures use stable reason codes rather than raw
errors. Credentials, codes, tokens, secret hashes, and raw interaction handles
are excluded.

Audit delivery can fail before or after a durable domain transition. Before a
mutation, failure may block safe progress depending on policy. After commit,
rollback may be impossible; admin operations expose typed post-commit ambiguity
and provider paths increment delivery-failure counters.

The `tinyidpauditdelivery` analyzer reports ignored errors. Readiness checks the
audit reporter's health. This does not prove every security transition emits the
right event; event-completeness review remains separate.

## Security events, logs, and metrics

Security events are versioned machine facts for temporal monitoring. Audit is an
accountability record. Logs diagnose components. Metrics aggregate operations.

The separation prevents one sink from accumulating incompatible privacy,
durability, sampling, and schema requirements. Security event delivery failure
has its own counter. The current open question is whether production readiness
should fail on that counter or treat the monitor as optional assurance.

## Liveness

`Provider.Liveness` answers whether the provider lifecycle is functioning. It is
not a database or dependency check. An orchestrator should restart on persistent
liveness failure, not every temporary readiness failure.

The endpoint returns structured checks so operators and automation do not infer
health from HTTP status alone.

## Readiness in exact components

`Provider.Readiness` evaluates:

1. lifecycle state;
2. store persistence and access;
3. supported schema;
4. active and verification signing keys;
5. token-secret policy;
6. audit health and delivery failures;
7. limiter/address production readiness;
8. maintenance recency.

Each component has ready/degraded state, reason, and checked timestamp. The
production host smoke observed all eight ready over HTTP/2.

Readiness is a routing decision, not a formal proof. It answers whether known
operational prerequisites currently hold.

## Maintenance

Maintenance removes expired domain records, old protocol state, and retired
signing keys according to derived retention. The host runs initial maintenance
and schedules later runs with cancellation.

Readiness becomes false when maintenance has not completed within its expected
interval budget. This makes a stopped scheduler observable before unbounded state
growth or stale keys become invisible operational debt.

Deletion policies preserve replay and verification windows. Shortening retention
without considering maximum token lifetimes can turn cleanup into a protocol
failure.

## Signing-key lifecycle

An active RSA key signs ID tokens. Verification keys include active and recently
retired keys. Planned rotation atomically activates a new key and retires the
old, setting `NotAfter` at retirement.

Retention derives from maximum ID-token lifetime plus clock skew so relying
parties can verify already-issued tokens. Maintenance purges only after overlap.

Emergency purge is separate. It refuses active or never-retired staged keys but
allows compromised retired trust to be removed immediately after rotation. This
can invalidate outstanding tokens and is therefore an incident action, not
ordinary maintenance.

## Token-secret lifecycle

The HMAC token secret protects opaque Fosite artifacts. `serve-production` reads
at least 32 bytes from an owner-only file. It is intentionally not accepted as a
command-line literal or environment variable, reducing process-list and broad
environment exposure.

Rotation is immediate: a provider with the new secret rejects opaque access and
refresh tokens minted with the old secret. There is no dual-secret grace period
in the current design. Operators must plan the availability impact.

## Filesystem controls

SQLite, audit, token secret, TLS private key, backups, and rollback files contain
different sensitive data but all require owner-restricted permissions. The
security-invariants probe runs under permissive umask and confirms the store
creates owner-only files.

Filesystem mode does not protect against a compromised process owner or root.
Target deployment should add volume encryption, backup access control, and
host-level monitoring as required.

## Backup and restore operations

The admin CLI creates online verified backups through the public store API. The
manifest binds schema, migration checksums, table counts, and active keys.

Restore is an offline operation. The runbook requires graceful stop, verification,
rollback preservation, restore, doctor, smoke, and a new backup. The release
drill executes this sequence and proves post-backup state disappears as expected.

The rollback path is evidence and recovery capacity, not a substitute for tested
restore. Backup success without restore testing is incomplete.

## Administrative audit

The admin CLI opens a synchronous `<db>.audit.jsonl` sink. Schema initialization,
migrations, client/user/credential mutation, backup, restore, signing rotation,
and emergency purge emit records.

Administrative commands can commit before audit delivery failure. Their output
must make ambiguity visible so automation reconciles state rather than repeating
a non-idempotent operation blindly.

## Production host command

`serve-production` is implemented with Glazed fields rather than direct
environment access. Typed decoding provides help, schema, config-file support,
and consistent logging.

The command:

- validates required paths and durations;
- opens SQLite and synchronous audit;
- selects direct or trusted-proxy resolver;
- constructs bounded limiter and password work;
- creates `embeddedidp.Provider`;
- runs initial maintenance;
- starts scheduled maintenance in an `errgroup`;
- starts TLS with explicit server limits;
- handles SIGINT/SIGTERM through context cancellation;
- calls graceful shutdown with a deadline;
- waits for all goroutines and closes resources.

The use of `errgroup` ensures one failing owned goroutine cancels the host rather
than leaving partial service.

## Production smoke in tmux

The exact candidate was run in tmux per project process guidelines. The smoke
generated an ephemeral localhost certificate and owner-only token secret,
provisioned SQLite and an active key, started the production command, and used
`capture-pane` for logs.

Observed evidence:

- TLS and HTTP/2 listener;
- discovery and security headers;
- liveness 200;
- readiness 200 with eight ready checks;
- request-body bound;
- TRACE rejection;
- query bearer rejection;
- graceful SIGINT;
- port unbound after shutdown.

The smoke is tied to a local host and generated certificate. It does not prove
target proxy, DNS, certificate automation, filesystem, cgroup, or network policy.

## Runtime load and profiles

The production runtime probe completed authorization, token, UserInfo, refresh,
and bounded concurrent password flows. It wrote NDJSON events and CPU/heap
profiles, then generated a Markdown summary.

The recorded run had 5,125 HTTP operations, 129 durable audit events, zero HTTP
errors, and bounded password capacity statistics. Profiles help locate CPU and
heap pressure. They do not predict every production workload.

## Always-on CI

The CI workflow uses exact Go 1.26.5 with `GOWORK=off`. It records toolchain/CGO,
builds, runs all tests and vet, CLI smoke, custom analyzers, fuzz seed corpus,
external consumer, backup/restore, pinned lint, and govulncheck.

`GOWORK=off` matters because a published consumer cannot rely on sibling modules
in the developer workspace. Normal local `go.work` testing remains valuable for
development integration; standalone CI tests packaging.

## Release gates

The manual release-gates workflow requires:

- hosted plan ID;
- expected candidate SHA-256;
- exact deterministic build and hash comparison;
- full race;
- 30-second parser fuzz campaigns;
- concurrency and fault suites;
- migration/rotation/backup/restore drills;
- hosted OIDF after local gates.

The hosted job uses protected environment secrets for suite authority and test
login. The runner preserves plan/test logs as artifacts.

## Release evidence workflow

The release workflow builds a trimpath, buildvcs-disabled Linux artifact and
records SHA-256, Go build info, and environment. It creates SPDX JSON SBOM,
module graph, dependency license notices, Sigstore bundles, and GitHub build
provenance.

Each artifact answers a separate question:

| Evidence | Question |
|---|---|
| SHA-256 | Which exact bytes? |
| build info | Which Go/module metadata is embedded? |
| SBOM | Which components are present? |
| module graph | Why are dependencies present? |
| vulnerability result | Which known reachable issues were found now? |
| license packet | Which redistribution obligations need review? |
| signature | Which identity signed these bytes? |
| provenance | Which build process produced them? |

## Go version and dependency boundary

Adding go-go-goja raised the module minimum to Go 1.26.1. The external consumer
probe still generated `go 1.25.11` and failed. Its script originally captured
stderr and exited before printing it; the exact-candidate gate exposed both
problems.

The fix prints captured failures and declares the actual minimum. This is not a
compatibility layer. A dependency requiring 1.26.1 cannot truthfully support a
1.25 consumer.

This incident demonstrates why external packaging tests belong in release
evidence even when repository tests pass.

## Vulnerability interpretation

Govulncheck reported zero vulnerabilities reached by current code, while one
imported package and fourteen required modules contained vulnerabilities not
called by analyzed paths.

This is a time-bound database and call-graph result. It must be rerun after code,
dependency, tool, or vulnerability database changes. Unreachable does not mean
irrelevant forever.

## Hosted OIDF

The repository contains a runner that authenticates to the hosted OpenID
Foundation suite, starts tests, follows exported browser interactions, submits
login/consent, polls terminal status, and saves info/log JSON.

Hosted evidence must bind:

- suite version;
- plan ID and configuration;
- each test and variant;
- deployed issuer and client settings;
- exact binary hash;
- start/end timestamps;
- pass/warning/failure results;
- warning adjudication;
- preserved logs.

Prior hosted result directories cannot establish results for a changed binary.
The latest exact-candidate ledger therefore leaves this blocking row open.

## Generic web scanning

ZAP or equivalent scanning should run through the real reverse proxy and TLS
topology. Results need triage because OAuth redirects, CSP, cache controls, and
browser forms can produce context-sensitive findings.

Manual header and method probes are useful but are not renamed as scanner
evidence. The ledger records the missing installed scanner honestly.

## Supply-chain research context

NIST SSDF organizes secure development practices across preparation, software
protection, secure production, and vulnerability response. SLSA focuses more
specifically on source-to-artifact integrity and provenance.

The current workflows implement pieces of both: pinned toolchain/actions,
reproducible build command, SBOM, dependency graph, vulnerability checking,
signatures, and provenance. The project does not claim a formal SLSA level in
the ticket; such a claim would require evaluating the complete build platform
and policy.

## Incident response

The runbook defines deployment checks, backup/restore, signing-key compromise,
token-secret compromise, database corruption, audit failure, dependency
vulnerability, and rollback. It names evidence to preserve and distinguishes
planned operations from emergency revocation.

An incident procedure must be executable under stress. Release drills validate
the mechanics before an incident. Target-environment drills remain necessary.

## Residual-risk register

The release ledger records risk, severity, mitigation, owner, expiry, and release
exception status. Important open risks include exact hosted conformance, target
topology validation, generic scanning, independent review, and signed artifact
publication.

An unassigned owner is itself visible risk. Expiry prevents a temporary exception
from becoming permanent undocumented policy.

## Human authority

Independent reviewer and release owner have separate signature sections. The
reviewer evaluates technical adequacy and residual risks. The owner decides
whether organizational deployment conditions and risk acceptance are satisfied.

The build, test suite, agent, or document cannot self-sign these roles. This is a
security boundary against evidence inflation.

## Exact-candidate case study

Code candidate `5bb4dae` produced deterministic Linux binary SHA-256
`cf43cae64de3c1ac9610eb2bd723eb09189df751a6da422b2f8b80dbf86f43dd`.

Local results included full race, vet, analyzers, repeated lint, zero reachable
vulnerabilities, six fuzz campaigns, all protocol failpoints, SQLite/recovery
drills, external-module OIDC, local conformance, proxy tests, and production TLS
host smoke.

The decision remained not approved because hosted OIDF, generic scanner, signed
artifact/provenance execution, independent review, and owner approval were
missing. This is the intended final lesson: evidence quantity does not erase a
missing authority class.

## Deployment review method

For a proposed production deployment:

1. Record exact source commit and binary digest.
2. Verify issuer, DNS, certificate, and redirect consistency.
3. Verify token secret, TLS key, database, audit, backup, and rollback file
   permissions.
4. Verify one active local SQLite writer and supported filesystem.
5. Verify proxy CIDRs, XFF sanitation, hop limit, and protected upstream link.
6. Verify HTTP timeouts, body/header limits, connection/platform quotas.
7. Verify audit health, shipping, retention, and alert ownership.
8. Verify password-work and rate-limit capacity from load evidence.
9. Verify maintenance scheduling and readiness behavior.
10. Execute backup/restore and key/secret rotation drills.
11. Run external OIDC, hosted OIDF, and generic web tests against deployed bytes.
12. Preserve SBOM, vulnerability, license, signature, and provenance evidence.
13. Obtain independent review and owner approval.
14. Define rollback thresholds and on-call ownership.

## Common production errors

### Relying on development defaults

No-op audit and allow-all limiting make a process easy to start but cannot be
silently accepted in production mode.

### Trusting forwarding headers globally

Forwarded identity is valid only through configured trusted peers and sanitized
chains.

### Treating readiness as liveness

Restarting on temporary dependency unready can create loops and destroy evidence.

### Rotating keys without overlap

Immediate deletion of an old signing key invalidates outstanding ID tokens.
Emergency purge must be explicit.

### Copying SQLite files

WAL state can be omitted from a valid-looking copy. Use online backup and verify.

### Storing secrets in arguments or logs

Process listings, shell history, audit, and traces expand exposure. Use protected
files and secret-free evidence.

### Equating local tests with deployment proof

Proxy, TLS, filesystem, cgroup, DNS, and hosted suite behavior remain external.

### Signing an unbound artifact

Conformance, signature, and deployed digest must refer to the same bytes.

## Decision records

### PR-1: public library, host-owned process

- **Decision:** expose handler/lifecycle; host owns listener and signals.
- **Reason:** deployment controls cannot be safely hidden in a library.
- **Consequence:** every embedding host must satisfy the production contract.

### PR-2: fail-closed production preflight

- **Decision:** reject missing or development-only controls.
- **Reason:** silent defaults create false readiness.
- **Consequence:** production construction is intentionally demanding.

### PR-3: explicit trusted-proxy resolver

- **Decision:** accept forwarding only from configured CIDRs with bounded chain.
- **Reason:** headers are otherwise attacker-controlled.
- **Consequence:** deployments must inventory proxy topology.

### PR-4: synchronous durable audit

- **Decision:** append and fsync before success reporting where possible.
- **Reason:** acknowledged audit loss is unacceptable for current scale.
- **Consequence:** audit latency applies backpressure and must be capacity-tested.

### PR-5: derived retention

- **Decision:** calculate protocol/key retention from maximum token lifetimes.
- **Reason:** cleanup must preserve replay and verification windows.
- **Consequence:** client TTL changes affect maintenance validation.

### PR-6: exact artifact ledger

- **Decision:** bind gates, signatures, deployment, and hosted plan to SHA-256.
- **Reason:** evidence for different bytes is not composable.
- **Consequence:** code changes require a new candidate evidence pass.

### PR-7: human release authority remains external

- **Decision:** keep reviewer and owner rows explicit and unsigned until humans
  act.
- **Reason:** technical automation cannot accept organizational risk.
- **Consequence:** “all local gates pass” still permits NOT APPROVED.

## Extended exercises

1. Trace every field in `embeddedidp.Options` to validation and runtime use.
2. Explain why a production-ready interface marker is needed in addition to
   non-nil controls.
3. Review the issuer parser against a reverse-proxy deployment.
4. Construct a malicious XFF chain and walk the resolver right-to-left.
5. Calculate password-work memory at several capacities.
6. Identify an audit event that occurs after commit and define reconciliation.
7. Explain every readiness component and its operator response.
8. Calculate signing-key retention for two clients with different ID-token TTLs.
9. Compare planned signing rotation with emergency purge.
10. Run the release drill and annotate each state boundary.
11. Explain why `GOWORK=off` is a packaging gate.
12. Reproduce the external consumer's former Go-version failure conceptually.
13. Map SBOM, signature, and provenance to separate threat questions.
14. Define the metadata needed to accept hosted OIDF evidence.
15. Explain why manual HTTP probes do not close the scanner row.
16. Review the exact-candidate decision and defend NOT APPROVED.
17. Draft a target-environment canary and rollback checklist.
18. Define alert thresholds for audit failure and maintenance staleness.
19. Identify a host control that cannot be validated by the embedded provider.
20. Write a release exception with owner and expiry, then explain when it is
    unacceptable.

## Chapter review checklist

- Can the reader explain public package ownership and external embeddability?
- Can the reader enumerate production preflight requirements?
- Can the reader review TLS, HTTP, proxy, limiter, and password-work controls?
- Can the reader distinguish audit, security events, logs, and metrics?
- Can the reader explain liveness, readiness, and maintenance?
- Can the reader plan key, token-secret, backup, and restore operations?
- Can the reader describe CI, release gates, and evidence artifacts?
- Can the reader interpret vulnerability and hosted-conformance results?
- Can the reader bind evidence to exact bytes?
- Can the reader identify human authority that automation cannot provide?

## Production configuration review table

| Control | Configuration | Validation | Runtime owner | Readiness/evidence |
|---|---|---|---|---|
| issuer | `Options.Issuer` / CLI | production HTTPS parser | provider/host | discovery + host smoke |
| store | `Options.Store` / DB path | non-nil, persistent/schema | SQLite | store/schema checks |
| cookie secure | `Cookie.Secure` | required production | adapter/browser | session tests |
| SameSite | `Cookie.SameSite` | supported explicit mode | adapter/browser | cookie tests |
| token secret | protected file -> `Token.SecretKey` | length + file mode | host/Fosite | token-secret check/rotation |
| audit | path + sink | production-ready + health | host/sink | audit readiness/counter |
| consent | policy | production default/explicit | adapter/policy | consent tests |
| limiter | rate/window | production-ready | host/provider | limiter readiness/tests |
| client address | direct or trusted CIDRs | production-ready | resolver | proxy tests |
| authenticator | native/custom | production-ready/reporting | authn | password stats/readiness |
| password policy | min/max/blocklist | NIST-aligned bounds | admin/authn | acceptance tests |
| password work | max concurrent | positive bounded | authn | stats/load probe |
| maintenance | interval/retentions | derived lifetime minima | provider/host | recency readiness |
| signing key | DB active key | current/supported/overlap | store/adapter | signing readiness/JWKS |
| TLS cert/key | CLI paths | required/readable | HTTP host | TLS/HTTP2 smoke |
| request bounds | CLI sizes/timeouts | positive parse | HTTP server | manual/load probes |
| shutdown | duration/signals | positive duration | host errgroup | tmux graceful stop |

## Target deployment evidence worksheet

Record values rather than checking generic boxes.

```text
environment/region
deployment owner
source commit
binary SHA-256
signature/provenance locations
issuer and DNS
certificate issuer/SAN/expiry/renewal owner
proxy product/config revision
trusted proxy CIDRs and max hops
upstream link protection
listener/network policy
CPU/memory/process/file limits
SQLite filesystem and mount options
single-writer enforcement
database/audit/secret/key modes and owners
backup destination/encryption/retention
last restore drill and result
last key rotation drill
last token-secret rotation drill
audit shipping/retention/alert
password-work capacity evidence
rate-limit policy and alert
maintenance interval/last success
readiness/liveness routing policy
hosted OIDF plan/result
generic scanner/result/adjudication
on-call and rollback owner
independent reviewer
release owner
```

## Incident evidence worksheet

For suspected compromise preserve:

- detection timestamp and source;
- exact deployed artifact identity;
- configuration revision;
- audit segment and integrity metadata;
- security-event segment/schema;
- relevant application/proxy/system logs;
- database and verified backup identity;
- active/verification key IDs;
- token-secret generation identifier without secret bytes;
- affected clients/users/scopes/time window;
- rotation/revocation commands and results;
- readiness changes;
- restore/rollback decisions;
- external notifications and owner approvals.

Do not collect raw passwords or bearer tokens merely because an incident is in
progress. Evidence handling remains least-privilege.

## Release-gate failure interpretation

### Build hash mismatch

Stop. The workflow is not testing the artifact proposed for deployment. Rebuild
or correct expected identity; do not update expected hash without explaining the
source change.

### Race/lint/analyzer failure

Treat as code/evidence defect. Fix, commit, and restart exact-candidate identity.

### Fuzz counterexample

Preserve seed/corpus and shrunk input. Convert to deterministic regression before
fix. Rerun affected and full gates.

### Failpoint/recovery failure

Assume authority or backup integrity risk. Do not release under a generic flaky
test exception without root cause.

### Hosted conformance warning

Preserve logs and adjudicate against plan/spec. Warning is not automatically pass
or fail; decision needs named reviewer.

### Scanner finding

Reproduce through actual topology, classify context, fix or document explicit
risk. Do not disable scanner rule globally to produce green output.

### Signature/provenance failure

Artifact origin is unestablished. Do not distribute as approved release.

### Missing reviewer/owner

Technical work may be complete; authorization is not. Keep NOT APPROVED.

## Production change review questions

1. Does the change alter public host contract?
2. Does it add a secret or protected file?
3. Does it alter issuer, endpoint, redirect, or cookie behavior?
4. Does it change proxy/address/limiter trust?
5. Does it change password CPU/memory capacity?
6. Does it add/alter audit or security events?
7. Does it affect readiness/liveness/maintenance?
8. Does it affect key or token-secret rotation?
9. Does it change schema, backup, restore, or rollback?
10. Does it add a dependency or raise Go/toolchain minimum?
11. Does it change reproducible build bytes?
12. Which local and external gates must be repeated?
13. Which runbook/residual risk changes?
14. Who reviews and who approves?

## Final production competence test

The reader passes when they can review a concrete deployment and release packet,
identify missing controls without confusing them with code defects, bind every
piece of evidence to exact bytes and environment, and preserve NOT APPROVED until
the required technical, external, and human authority classes are present.

## Final production scenario

Assume a deployment has:

- green local tests and race;
- correct binary SHA-256;
- valid public certificate;
- a reverse proxy whose CIDR was not configured;
- healthy SQLite and signing key;
- audit disk at capacity;
- no hosted OIDF for the current bytes;
- a signed checksum;
- no independent reviewer.

The deployment is not ready and the release is not approved.

Required analysis:

1. Forwarded addresses are untrusted or misresolved until proxy CIDR and
   sanitation are verified.
2. Audit health failure should make production readiness false and requires
   capacity/retention response.
3. The signed checksum authenticates bytes but does not supply hosted protocol
   evidence.
4. Earlier OIDF results cannot bind changed bytes.
5. Independent review and owner authority remain missing.

Write the stop, remediation, evidence-preservation, retest, and approval sequence
without changing any control to make the dashboard green.

## Final source trace

For the scenario above cite `TrustedProxyResolver`, audit readiness, release-gate
workflow, exact-candidate ledger, incident runbook, RFC 9700 proxy guidance, and
the approval algorithm. This exercise confirms that production reasoning spans
code, deployment, standards, operations, artifacts, and human authority.

The reviewer must finish by naming the first safe reversible action and the
actions that require new authority. In this scenario, preserving evidence,
stopping traffic, restoring audit capacity, and correcting proxy trust are
operational remediation; accepting missing hosted review or approving release
requires designated owners and cannot be inferred by the software.

Record the owner and expiry of every exception.

## References

- `playbook/01-production-operations-and-incident-response-runbook.md`
- `reference/03-release-candidate-evidence-packet-and-approval-ledger.md`
- `reference/06-exact-candidate-assurance-evidence-5bb4dae.md`
- `sources/rfc9700-oauth-security-bcp.md`
- `sources/nist-sp-800-63b-4-authenticators.md`
