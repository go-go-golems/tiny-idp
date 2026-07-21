# Tasks

## Phase 1 — Establish the production configuration boundary

- [x] Replace `--message-desk-origin` with a bounded, read-only `--clients-file` containing all desired browser clients. <!-- t:8ysd -->
- [x] Define and validate the versioned client catalog schema, including duplicate-ID, HTTPS-origin, redirect URI, scope, and profile checks. <!-- t:xrvj -->
- [x] Preserve startup conflict detection: a catalog change must never silently widen a stored client registration. <!-- t:eqhw -->
- [x] Add focused unit tests for two browser clients, a duplicate client ID, malformed JSON, invalid origins, and a stored-client conflict. <!-- t:uc9n -->

## Phase 2 — Add IdP-owned, mounted theme assets

- [x] Add a versioned theme catalog with `defaultTheme`, client-to-theme mapping, display name, and a root-relative stylesheet filename. <!-- t:as9r -->
- [x] Implement an immutable-at-startup catalog loader rooted at `--theme-dir`; reject traversal, non-CSS files, missing files, duplicate names, and unassigned client references. <!-- t:cl9s -->
- [x] Implement a theme-selecting interaction renderer that consumes only the validated `InteractionPage.ClientID` supplied by TinyIDP. <!-- t:cy2q -->
- [x] Implement an allowlisted same-origin asset handler below `/static/themes/`; do not proxy or redirect to application URLs. <!-- t:4g5l -->
- [x] Update the interaction template to use the selected product name and stylesheet path while retaining every provider-owned form field and the existing CSP contract. <!-- t:aen5 -->
- [x] Add renderer, loader, route, escaping, and CSP tests for the default and each named theme. <!-- t:86e2 -->

## Phase 3 — Make MessageDesk a GitOps-provided theme

- [x] Move the MessageDesk IdP CSS source into an application-owned theme source directory and document its review ownership. <!-- t:f9bm -->
- [x] Add a GitOps Kustomize `configMapGenerator` that packages `themes.json` and reviewed CSS with a content-hashed name. <!-- t:g1uy -->
- [x] Mount the generated ConfigMap read-only at `/etc/tinyidp/themes`; pass `--theme-dir` and `--theme-catalog-file` to TinyIDP. <!-- t:493j -->
- [x] Remove the embedded MessageDesk stylesheet route from the production host after the mounted asset route is verified. <!-- t:4wvf -->
- [x] Render the shared IdP manifest and prove a theme source change alters the generated ConfigMap reference and therefore the Pod template. <!-- t:5eqb -->

## Phase 4 — Register and deploy the second application

- [x] Choose the concrete second example and record its public hostname, callback path, logout path, client ID, required scopes, and UI theme owner. <!-- t:1zs0 -->
- [x] Add that client to the shared IdP client catalog; use a distinct, immutable public client ID. <!-- t:eo20 -->
- [x] Reuse the existing independent goja-auth Argo Application and separate kustomization/namespace; it does not share the MessageDesk PVC or ServiceAccount. <!-- t:3fis -->
- [x] Give the second app an HTTPS Ingress, certificate, Service, restricted NetworkPolicy, trusted-Traefik listener configuration, and canonical-issuer backchannel route through Traefik. <!-- t:s33j -->
- [x] Add its theme CSS and catalog mapping through the same reviewed GitOps change. <!-- t:jh9j -->
- [x] Verify MessageDesk remains functional while the second app completes login/consent/logout with an account created through MessageDesk against the same IdP. <!-- t:2qy7 -->

## Phase 5 — Acceptance, operations, and rollback

- [x] Extend the public acceptance harness with one independent authorization-code-with-PKCE flow for each client. <!-- t:42so -->
- [x] Assert theme selection from HTML and stylesheet headers without logging cookies, authorization codes, or passwords. <!-- t:l77l -->
- [x] Test invalid client/theme mappings, path traversal, a direct untrusted forwarded request, and CSP rejection of off-origin assets. <!-- t:inv5 -->
- [x] Write an operator runbook for adding a client, changing a theme, disabling a client, rolling back a theme, and recovering a broken catalog. <!-- t:3kjh -->
- [x] Capture Argo health, certificate readiness, PVC preservation, and audit events for both clients. <!-- t:rwm3 -->

## Research branch — TinyIDP Kubernetes operator

- [ ] Define a minimal `TinyIDP` custom resource and the exact objects the controller may own. <!-- t:g92w -->
- [ ] Decide whether a separate `TinyIDPClient` resource is needed, and define cross-namespace ownership and namespace-selection rules. <!-- t:4z1g -->
- [ ] Prototype only reconciliation planning and status conditions before granting the controller secret, ingress, or Vault authority. <!-- t:upo1 -->
- [ ] Compare operator complexity and failure modes against the GitOps-only design after two applications are operating. <!-- t:jfaa -->
