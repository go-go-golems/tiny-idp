---
Title: Production acceptance evidence for the shared two-app TinyIDP
Ticket: TINYIDP-MULTIAPP-THEMES-001
Status: active
Topics:
    - oidc
    - testing
    - operations
    - security
DocType: reference
Intent: ticket-specific
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/scripts/01-public-two-app-flow/main.go
      Note: |-
        Secret-safe public two-client acceptance harness
        Public two-client OIDC acceptance harness
    - Path: repo://ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/scripts/02-kustomize-theme-rollout.sh
      Note: |-
        Content-hash rollout proof
        Theme content-hash rollout proof
ExternalSources: []
Summary: Exact non-secret delivery and runtime evidence captured after GitOps PR 191 reached the k3s cluster.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Allow a reviewer to distinguish source completion from a proven production rollout.
WhenToUse: Use when reviewing or repeating the shared-TinyIDP deployment.
---


# Production acceptance evidence

## Delivery chain

- TinyIDP PR #12 merged as
  `78997ec08629d3f420b89c2de73c59d9b12cd5f8`.
- go-go-goja PR #100 merged as
  `cd1429f95e24156a98ff976fa63b484ee2d35e9c`.
- The source workflows published TinyIDP and MessageDesk image tag
  `sha-78997ec` and goja-auth image tag `sha-cd1429f` successfully.
- Combined GitOps PR #191 merged as
  `68209d0f6a426d01b7cf42dd5322051fb91d51ca`.

## Argo and workload state

Both `tiny-message-desk` and `goja-auth-host-demo` reported:

```text
SYNC=Synced  HEALTH=Healthy
REVISION=68209d0f6a426d01b7cf42dd5322051fb91d51ca
OP=Succeeded
```

The Ready Deployments used exactly:

```text
message-desk          ghcr.io/go-go-golems/tiny-idp-message-desk:sha-78997ec
tinyidp               ghcr.io/go-go-golems/tiny-idp:sha-78997ec
goja-auth-host-demo   ghcr.io/go-go-golems/go-go-goja-auth-host:sha-cd1429f
```

The two pre-existing application PVCs remained Bound after rollout:

```text
message-desk-data -> pvc-bbc479b1-34dd-40e0-af00-2ac242f3fdf0
tinyidp-data      -> pvc-0df35ec6-54cb-482c-a4d9-206514fbe144
```

`message-desk-tls`, `tinyidp-tls`, and `goja-auth-host-demo-tls` all reported
`Ready=True` from `letsencrypt-prod`.

## Public protocol and UI acceptance

Command:

```bash
go run ./ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/scripts/01-public-two-app-flow
```

Result:

```text
PASS public readiness for TinyIDP, MessageDesk, and goja-auth
PASS MessageDesk signup used its client-selected same-origin theme
PASS the same identity logged into goja-auth using its distinct theme
PASS goja-auth session, protected /me, and CSRF logout contracts
```

The harness uses separate cookie jars for the two relying parties. It validates
exact client IDs/callbacks, `state`, `nonce`, and S256 PKCE presence; it does not
print their values. It also checks the same-origin CSS content type and
`nosniff` header for each selected theme.

## Content-hash rollout proof

The rollout probe copied the Kustomize source to a temporary directory, changed
only `goja-auth.css`, rendered both versions, and observed:

```text
PASS theme ConfigMap reference changed:
  tinyidp-theme-catalog-9tc86ct4tg -> tinyidp-theme-catalog-969t5mb6g6
PASS independent client catalog reference remained:
  tinyidp-client-catalog-9t78t4hkh2
```

The temporary tree was deleted after the test; no production source was
modified by the probe.

## Security-negative test evidence

The focused test command passed:

```bash
go test ./internal/productionconfig ./internal/productionui ./internal/fositeadapter ./pkg/idp
```

Those packages cover malformed/unknown client and theme declarations,
stylesheet traversal and allowlisting, escaped presentation, the restrictive
same-origin CSP, and trusted-proxy rejection behavior. The public harness adds
positive evidence through the real certificate and Traefik listener.

## Audit evidence

The persisted TinyIDP audit file contained accepted events for both exact
client IDs after the public run. The extracted fields were restricted to
`name`, `result`, and `client_id`:

```text
account.self_registration   accepted   tinyidp-message-app
consent.granted             accepted   tinyidp-message-app
authorize.request.accepted  accepted   tinyidp-message-app
token.request.accepted      accepted   tinyidp-message-app
password.login.success      accepted   goja-auth-host-demo
login.success               accepted   goja-auth-host-demo
consent.granted             accepted   goja-auth-host-demo
authorize.request.accepted  accepted   goja-auth-host-demo
token.request.accepted      accepted   goja-auth-host-demo
```

No subject, email, password, cookie, authorization code, state, nonce, CSRF
token, client secret, or Vault material was copied into this document.
