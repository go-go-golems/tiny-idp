# Changelog

## 2026-07-22

- Initial workspace created

## 2026-07-22

Step 1: Recorded the live two-application baseline and private Mailpit deployment boundary.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/deployment.yaml — Baseline workload evidence

## 2026-07-22

Step 2: Implemented private Mailpit, Vault-backed verified signup, and operator acceptance (infrastructure commit 78308e6).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/deployment.yaml — TinyIDP durable key and SMTP wiring
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/mailpit.yaml — Private delivery sink

## 2026-07-22

Step 3: Merged infrastructure PR 194, synchronized the healthy Argo deployment, passed the complete live Message Desk verified-signup journey, and isolated Goja Auth acceptance to an unpublished `/auth/register` route commit.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Compose/Kubernetes live acceptance configuration and invited Goja journey
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja/pkg/xgoja/hostauth/builder.go — Goja relying-party registration route pending publication
