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

## References

- `playbook/01-production-operations-and-incident-response-runbook.md`
- `reference/03-release-candidate-evidence-packet-and-approval-ledger.md`
- `reference/06-exact-candidate-assurance-evidence-5bb4dae.md`
- `sources/rfc9700-oauth-security-bcp.md`
- `sources/nist-sp-800-63b-4-authenticators.md`
