---
Title: Strict Device Authorization Operations and Alert Runbook
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - oauth2
    - oidc
    - security
    - operations
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/fositeadapter/provider.go
      Note: Device authorization creation and generic token endpoint audit events.
    - Path: internal/fositeadapter/device_verification.go
      Note: Browser verification audit events and rate limiting.
    - Path: ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/scripts/01-device-audit-metrics.go
      Note: Bounded JSONL audit to low-cardinality Prometheus metrics exporter.
ExternalSources:
    - sources/rfc-8628-oauth-device-authorization-grant.md
    - sources/rfc-9700-oauth-security-bcp.md
Summary: Concrete observability contract, dashboard queries, alert thresholds, and incident actions for the strict RFC 8628 implementation.
LastUpdated: 2026-07-16T00:00:00Z
WhatFor: Lets an operator distinguish normal pending polling from abuse, browser verification failure, token persistence failure, or audit-health failure without exposing bearer credentials.
WhenToUse: Before enabling device clients in production, when wiring audit log collection, and during a device-login incident.
---

# Strict Device Authorization Operations and Alert Runbook

## Purpose and trust boundary

The strict provider's device flow has three independently observable stages:

```text
device client                 browser                    provider
-------------                 -------                    --------
POST device_authorization  ->                           device.authorization.*
                              GET/POST /device      ->  device.verification.*
POST /token (device_code) ->                           token.request.*
```

The audit stream records the stage, a reviewed reason code, and normal
non-secret identifiers where needed for forensic retrieval. It deliberately
does not contain `device_code`, `user_code`, access token, refresh token,
password, authorization header, or raw form. The metrics exporter removes
even client and subject labels, so dashboards remain bounded and do not turn
identity identifiers into metrics data.

This runbook concerns the strict embedded provider. The old mock device flow
is not an operations reference and must not be used as production evidence.

## Required deployment wiring

1. Run `tinyidp serve-production` with a durable `--audit-path`; check its
   `GET /readyz` endpoint through the issuer mount. A not-ready provider must
   block rollout because production mode requires durable audit delivery.
2. Ship the JSONL audit file to durable, access-controlled log storage. Preserve
   the file's `0600` permissions at its origin and do not use a world-readable
   node-exporter directory for the raw log.
3. Run the ticket exporter on a copy/read-only view of the audit stream and
   publish only its numeric output to a Prometheus textfile collector:

   ```sh
   go run ./ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/scripts/01-device-audit-metrics.go \
     -audit /var/log/tinyidp/audit.jsonl -since 15m
   ```

   The command writes metrics to standard output. The deployment wrapper must
   write a temporary textfile and atomically rename it; it must not append to
   a public metrics file while another process reads it.
4. Record the exporter invocation, audit retention period, scrape cadence, and
   alert receiver in the service deployment record. The exporter is a batch
   evidence tool, not a replacement for log shipping.

## Metric contract

The exporter has exactly these label dimensions:

| Metric | Labels | Interpretation |
| --- | --- | --- |
| `tinyidp_device_authorizations_total` | `stage`, `outcome` | Device authorization allocations. |
| `tinyidp_device_verifications_total` | `stage`, `outcome` | Browser approvals and denials. |
| `tinyidp_device_token_requests_total` | `stage`, `outcome`, `reason` | Device-code token completions or protocol rejections. |
| `tinyidp_device_rejections_total` | `stage`, `outcome`, `reason` | Creation/verification rejections. |
| `tinyidp_device_audit_invalid_lines_total` | none | Audit input parser health. |

`reason` is allow-listed by the exporter. Unknown values become `other`.
No metric label is derived from a client ID, user, user code, device code,
token, remote address, request ID, or free-form error text.

## Dashboard panels

The exporter reports a selected time window as counters. A collection job
should publish a fresh scrape file each interval; use the job's regular
collection cadence consistently when interpreting rates.

```promql
# Device flow creation volume
sum(tinyidp_device_authorizations_total{outcome="created"})

# Browser approval/denial mix
sum by (outcome) (tinyidp_device_verifications_total)

# Token protocol outcomes (pending and slow_down are expected while waiting)
sum by (outcome, reason) (tinyidp_device_token_requests_total)

# Rejections by hardened boundary and reason
sum by (stage, reason) (tinyidp_device_rejections_total)

# Input/exporter health: must remain zero
tinyidp_device_audit_invalid_lines_total
```

Keep raw audit exploration separate from this dashboard. It is the approved
place to filter a specific client or subject after an incident has a ticket
and access-control justification.

## Alerts and triage

Thresholds below are starting values. Baseline the service for at least one
release window, then tune per deployment size without adding identity labels.

| Signal | Initial alert | First interpretation | Immediate action |
| --- | --- | --- | --- |
| `device.authorization.rejected` with `rate_limited_or_invalid_client` | sustained nonzero for 10 minutes above normal baseline | malformed clients, scanning, or a rollout client-config mismatch | inspect deployment change; inspect rate limiter/audit; block suspicious sources at edge if confirmed. |
| verification `invalid_csrf` or `state_transition_failed` | any sustained burst; page at high severity if paired with successful tokens | broken UI integration, cross-site attempt, or concurrent/replay behavior | preserve audit interval; verify UI renderer/version; do not disable CSRF or named transitions. |
| token `invalid_grant` after successful approval | material rise above replay baseline | duplicate device polling, client retry bug, or replay attempt | compare client polling implementation with RFC interval; inspect whether the first successful response was lost. |
| token `slow_down` | high proportion of device token requests | client violates polling interval or bot traffic | contact client owner; do not reduce server backoff. |
| token `server_error` / audit failures / `/readyz` failure | immediate page | persistence, signing, audit, or store availability failure | stop rollout, keep database/audit evidence, check readiness dependency reason, fail over only according to database runbook. |
| `tinyidp_device_audit_invalid_lines_total > 0` | warning immediately | truncated/corrupt input or incompatible audit schema | verify log rotation/shipper atomicity; do not trust dashboard totals until repaired. |

## Incident procedures

### Suspected device-code replay

1. Preserve the audit range and current database backup before changing traffic.
2. Find the approved `device.verification.approved` and one subsequent token
   result using approved log-access procedures. Do **not** search or record
   raw device codes; they are intentionally absent from audit data.
3. Confirm whether the device client received the first token response. A
   network loss after commit produces a legitimate client retry that returns
   `invalid_grant`; issuing a second token would be a security failure.
4. Validate current binary with the transactional failpoint and replay suite:

   ```sh
   go test ./internal/fositeadapter -run 'Test(SQLiteDeviceTokenRedemptionFailpointsRollbackGrantAndTokens|SQLiteApprovedDeviceGrantSurvivesRestartAndRejectsReplay)' -count=1
   ```

5. If duplicate successful issuance is observed, remove the client from the
   device grant capability set, preserve the database and logs, rotate affected
   signing/token material according to the general incident runbook, and open
   a security incident. Do not patch around the conditional consume check.

### Browser verification abuse or user-code guessing

1. Inspect the rejection panel and audit reasons `rate_limited`, `missing_login`,
   and invalid-credential outcomes.
2. Verify proxy trust and client-address resolution; a bad proxy configuration
   collapses rate-limit buckets or trusts attacker-controlled headers.
3. Keep the existing code-entry and login limiter enabled. Adjust edge rules
   or capacity only after confirming a legitimate traffic increase.
4. Review grant expiry and maintenance. Do not increase user-code lifetime to
   compensate for a client polling defect.

### Audit or exporter health failure

1. Treat `/readyz` failure as deployment unhealthy. Liveness alone is not
   sufficient because it intentionally ignores durable dependencies.
2. Check file ownership, disk capacity, fsync errors, log rotation, and the
   shipper. Preserve the failed file and its permissions for forensics.
3. A nonzero invalid-line counter means metrics are incomplete. Fix the
   producer/rotation boundary, then compare raw retained audit volume before
   using the dashboard for incident conclusions.

## Release evidence

Before enabling a new device client, attach to its release record:

- a successful readiness probe after audit initialization;
- one sanitized exporter output from a controlled approve/deny/pending run;
- a screenshot or export of the five panels above with no identity labels;
- an owner and rotation policy for the audit file and metrics textfile;
- the exact strict-provider commit and focused test command results.

This operational evidence complements, but cannot replace, an independent
security review of the provider and its deployment.
