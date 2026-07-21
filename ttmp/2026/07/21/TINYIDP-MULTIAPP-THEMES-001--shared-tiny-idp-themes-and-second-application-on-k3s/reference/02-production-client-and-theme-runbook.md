---
Title: Production client and theme operations runbook
Ticket: TINYIDP-MULTIAPP-THEMES-001
Status: active
Topics:
    - oidc
    - operations
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/admin_client.go
      Note: Administrative disable and enable operations
    - Path: repo://internal/productionconfig/clients.go
      Note: Strict desired browser-client catalog
    - Path: repo://pkg/embeddedidp/bootstrap.go
      Note: Additive bootstrap and durable conflict semantics
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/kustomization.yaml
      Note: Content-hashed catalog and theme packaging
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/themes/clients.json
      Note: Desired public-client catalog
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/themes/themes.json
      Note: Client-to-theme mapping
ExternalSources: []
Summary: Add, change, disable, roll back, and recover TinyIDP clients and themes without bypassing GitOps or weakening the public issuer topology.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Give an operator an exact, safe sequence for routine and emergency changes to the shared production TinyIDP.
WhenToUse: Use for every production client or IdP theme change after the two-app rollout.
---


# Production client and theme operations runbook

## 1. The invariant to preserve

The browser speaks to the canonical HTTPS issuer
`https://idp-message-desk.yolo.scapegoat.dev`. Application Pods also use that
issuer and reach it through Traefik's TLS listener. Do not replace it with a
Service DNS name or a plain HTTP URL. Issuer equality is part of token
validation, not cosmetic configuration.

Git owns desired client registration and presentation. TinyIDP validates the
catalogs once at startup and preloads only the named CSS files. A request may
select a theme only indirectly: TinyIDP first validates `client_id`, then maps
that trusted ID through `themes.json`. Never add a query parameter, callback
field, or arbitrary URL that chooses a stylesheet.

```text
reviewed JSON/CSS
      |
      v
Kustomize content hash -----> Deployment Pod-template reference
      |                                  |
      v                                  v
immutable ConfigMap                controlled rollout
      |
      v
TinyIDP startup validation -----> serve both clients or fail closed
```

## 2. Preflight for every change

Work in a branch of the Hetzner k3s repository. Do not edit a live ConfigMap or
Deployment. Record these observations before changing files:

```bash
kubectl cluster-info
kubectl -n argocd get application tiny-message-desk goja-auth-host-demo
kubectl -n tiny-message-desk get deployment,pod,pvc,certificate
kubectl -n goja-auth-host-demo get deployment,pod,certificate
```

Both Argo applications should begin `Synced` and `Healthy`. The two TinyIDP
PVC names and bound volume IDs are rollback evidence: a normal theme/client
rollout must not replace either application data PVC.

Render both affected trees before opening a PR:

```bash
kubectl kustomize gitops/kustomize/tiny-message-desk > /tmp/tiny-message-desk.yaml
kubectl apply --dry-run=client -f /tmp/tiny-message-desk.yaml
kubectl kustomize gitops/kustomize/goja-auth-host-demo > /tmp/goja-auth-host-demo.yaml
kubectl apply --dry-run=client -f /tmp/goja-auth-host-demo.yaml
```

Review the rendered Deployment, not only the source. Confirm that catalog
ConfigMap names have suffixes and are referenced by the TinyIDP Pod template.

## 3. Add a browser application

First define the application contract. It needs one stable client ID, one exact
HTTPS callback URL, one exact post-logout URL, and the smallest required scope
set. Browser clients are public; they have no secret and must use authorization
code plus PKCE.

1. Add one object to `themes/clients.json`. Keep arrays canonical and include
   `openid`.
2. Add reviewed CSS below `themes/`. CSS is presentation-only. It cannot change
   the provider form action, field names, CSP, or authorization decision.
3. Add the theme to `themes/themes.json` and map the exact client ID.
4. Configure the application with the canonical issuer and exact matching
   callback. Do not mount an OIDC client secret for a public client.
5. Give a separately deployed app its own namespace, ServiceAccount, Service,
   Ingress/certificate, data store, and NetworkPolicy. Its IdP backchannel
   should traverse Traefik's websecure port.
6. Render, dry-run, open a PR, merge, and wait for Argo. Do not use `kubectl
   apply` to bypass the Application.
7. Run the public acceptance harness or extend it with the new client. Assert
   the authorization query's PKCE fields, exact redirect, selected stylesheet,
   authenticated application session, and logout behavior.

TinyIDP bootstrap refuses an existing client whose durable registration does
not exactly match the desired catalog. That failure is intentional. Do not
delete the database to make a conflicting redirect URI appear to work.

## 4. Change a theme

A theme-only change edits the application-owned CSS source. It should change
the `tinyidp-theme-catalog-*` generated name while leaving the
`tinyidp-client-catalog-*` name unchanged. Run:

```bash
ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/scripts/02-kustomize-theme-rollout.sh \
  /absolute/path/to/hetzner-k3s/gitops/kustomize/tiny-message-desk
```

After merge, wait for the TinyIDP rollout and request the stylesheet through
the public issuer. It must return `Content-Type: text/css`,
`X-Content-Type-Options: nosniff`, and the expected content. Then begin an auth
flow for each client and inspect the HTML's root-relative stylesheet link.

## 5. Disable a client safely

Removing a client from `clients.json` does **not** delete its durable TinyIDP
record. That behavior prevents accidental deletion, but removal alone is not a
security disable. Conversely, using `tinyidp admin client disable` while the
old active declaration remains in the mounted catalog causes the next startup
to fail with a `disabled` bootstrap conflict.

Use this two-part sequence:

1. Prepare and review a GitOps PR that removes the client from `clients.json`
   and removes its mapping from `themes.json`. Do not merge it yet.
2. Record the current Argo and Pod state.
3. Run the TinyIDP administrative disable against the durable database from a
   controlled operator session. Capture only the redacted command result and
   audit event—never tokens or session cookies.
4. Immediately merge the prepared GitOps PR. The running provider rejects the
   disabled client, and the replacement Pod omits it from desired bootstrap,
   avoiding a disabled-state conflict.
5. Prove a new authorization request for the client is rejected, while the
   remaining client still completes its flow.
6. Confirm an `admin.client.disabled` audit event and healthy Argo state.

The exact administrative invocation depends on the operator access method and
must name `/var/lib/tinyidp/tinyidp.sqlite`. Do not open a copied database and
mistake that for disabling production. If the administrative command cannot be
completed, stop the application Ingress as a separately reviewed containment
measure; do not improvise database SQL.

To re-enable, reverse the ordering: enable the durable client while it is still
absent from the catalog, then merge a reviewed PR restoring the exact catalog
and theme mapping. A changed redirect URI is a new migration decision, not a
routine enable.

## 6. Roll back a theme or deployment

Revert the GitOps merge commit that introduced the bad catalog/theme/image and
merge the revert. Argo should reconcile the old ConfigMap suffix and Pod
template. Do not delete PVCs, Secrets, certificates, or the namespace.

After rollback, verify:

- the previous image and ConfigMap names are present in the Deployment;
- the TinyIDP and MessageDesk PVC volume IDs are unchanged;
- both certificates remain `Ready=True`;
- both Argo applications are `Synced` and `Healthy`;
- one public authorization flow per remaining client succeeds;
- the audit log contains accepted authorization/token events after rollback.

If the rollback also changes a client registration, remember that strict
bootstrap compares the desired declaration with durable state. A redirect or
scope change needs an explicit data migration; a Git revert alone cannot
silently mutate it.

## 7. Recover a broken catalog

A malformed JSON document, missing CSS file, unsafe basename, unknown theme,
or durable-client conflict makes the new TinyIDP Pod fail startup. With the
default rolling Deployment, Kubernetes should retain the previously healthy
Pod while the replacement is unavailable. Confirm that fact; do not assume it.

1. Inspect the new Pod status and startup log. Record the bounded validation
   error, but do not print Secret values.
2. Confirm the old Pod still serves `/readyz` and both public application
   frontends remain available.
3. Revert the bad GitOps commit or fix the catalog in a new PR.
4. Render and dry-run locally before merging.
5. Wait for Argo and the Deployment rollout; verify the old Pod terminates only
   after the corrected Pod is Ready.
6. Re-run the public two-app harness.

Do not repair a generated ConfigMap in place. Argo will overwrite it, its name
no longer represents its content, and no review record will explain the live
state.

## 8. Evidence that closes an operation

Attach the following to the ticket diary or PR without sensitive values:

- Git commit and PR/merge commit;
- Kustomize render and client-side dry-run result;
- old and new content-hashed ConfigMap names;
- Argo revision, sync, and health for both applications;
- Deployment image tags and Ready replica counts;
- certificate readiness and unchanged PVC identifiers;
- public acceptance harness PASS lines;
- audit `name`, `result`, and `client_id` only.

The evidence must never contain a password, cookie, authorization code, state,
nonce, CSRF token, client secret, Vault token, or full subject identifier.
