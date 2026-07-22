---
Title: Shared TinyIDP two-application Mailpit deployment design
Ticket: TINYIDP-GITOPS-MAILPIT-001
Status: active
Topics:
    - identity
    - oidc
    - security
    - operations
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-shared-two-apps/compose.yaml
      Note: Locally proven Mailpit and SMTP topology
    - Path: repo://examples/tinyidp-shared-two-apps/open-signup.js
      Note: Source shared signup workflow to promote
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/deployment.yaml
      Note: Live TinyIDP workload to update
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/network-policy.yaml
      Note: Live network boundary to extend
ExternalSources: []
Summary: Promote the locally proven shared TinyIDP signup workflow to GitOps with private Mailpit delivery, Vault-backed keys, and operator-driven acceptance.
LastUpdated: 2026-07-22T19:09:53.243960887-04:00
WhatFor: Implement and review the live two-application verified-signup deployment.
WhenToUse: When changing the tiny-message-desk or goja-auth-host-demo GitOps applications and validating signup delivery.
---


# Shared TinyIDP two-application Mailpit deployment design

## Executive Summary

Message Desk and Goja Auth already use the same live TinyIDP issuer, but the
deployed signup program is the older single-form workflow. This change promotes
the locally tested shared program: Message Desk remains open admission, Goja
Auth requires a one-time signup invitation, and both applications require an
email verification code before account creation.

Mailpit is a temporary operator-only delivery sink. It receives SMTP inside the
cluster and has no Ingress. An operator reaches its web UI through an explicit
`kubectl port-forward`; TinyIDP remains responsible for generating, hashing,
expiring, and verifying every code.

## Problem Statement

The current live pod has neither email-challenge flags nor invitation keys, and
its ConfigMap creates identities immediately after collecting a password. That
does not exercise the professional admission and verification paths proven by
the local two-app stack. Publishing Mailpit on the Internet would expose active
verification codes, while omitting durable keys would make restarts invalidate
or weaken workflow state.

## Proposed Solution

Deploy four coordinated changes in the `tiny-message-desk` namespace:

- Mount the reviewed shared two-application JavaScript program.
- Supply independent Vault-managed HMAC keys for invitation lookup and email
  challenges, copied into an owner-only runtime volume by the existing init
  container.
- Configure TinyIDP to deliver its fixed `signup-code` template to
  `mailpit:1025` using private plaintext SMTP within the namespace.
- Run pinned Mailpit behind a ClusterIP Service. Permit SMTP from TinyIDP and
  operator UI access only through Kubernetes port-forwarding.

```text
Browser -> Message Desk -----+
                             +-> shared TinyIDP -> Mailpit SMTP :1025
Browser -> Goja Auth --------+         |                |
                                       |                +-> port-forward :8025
                                       +-> SQLite             -> operator
```

| Client | Admission | Verification |
|---|---|---|
| `tinyidp-message-app` | Open signup | Email code required |
| `goja-auth-host-demo` | One-time TinyIDP invitation | Email code required |

## Design Decisions

- Mailpit has no public Ingress and no NodePort. Verification codes are bearer
  secrets and must stay inside the operator boundary.
- SMTP is plaintext only across the namespace-local service path. NetworkPolicy
  limits both source and destination; Internet SMTP is out of scope.
- Raw keys remain in Vault. Kubernetes receives them through the existing
  `VaultStaticSecret`, and the init container creates mode `0600` files because
  TinyIDP rejects permissive key files.
- Invitation consumption stays in the same native transaction as identity and
  credential creation. JavaScript selects the workflow but cannot mutate the
  invitation store directly.
- Program, image, flags, keys, and Mailpit roll out together. TinyIDP startup
  validation fails closed when a required service is missing.

## Alternatives Considered

- Public Mailpit with Basic authentication was rejected because an ingress or
  authentication mistake would disclose active codes.
- The personal SMTP server remains deferred to the dedicated email transport
  ticket; it adds credentials, abuse controls, and deliverability concerns.
- Logging codes was rejected because logs have broader retention and access
  than the deliberate operator outbox.

## Implementation Plan

1. Add Vault keys and strict runtime file preparation.
2. Add Mailpit Deployment, Service, and NetworkPolicy rules.
3. Promote the shared program and TinyIDP flags/image.
4. Render Kustomize, run policy checks, and synchronize through Argo CD.
5. Port-forward Mailpit and exercise open signup, invited signup, invalid code,
   invite reuse, shared login, and logout.

## Open Questions

The final outbound SMTP provider is deferred. Mailpit proves the IDP workflow,
not Internet email deliverability.

## References

- `examples/tinyidp-shared-two-apps/open-signup.js`
- `examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts`
- `gitops/kustomize/tiny-message-desk/`
