# Tasks

## Phase 1 — Establish the production configuration boundary

- [ ] Replace `--message-desk-origin` with a bounded, read-only `--clients-file` containing all desired browser clients.
- [ ] Define and validate the versioned client catalog schema, including duplicate-ID, HTTPS-origin, redirect URI, scope, and profile checks.
- [ ] Preserve startup conflict detection: a catalog change must never silently widen a stored client registration.
- [ ] Add focused unit tests for two browser clients, a duplicate client ID, malformed JSON, invalid origins, and a stored-client conflict.

## Phase 2 — Add IdP-owned, mounted theme assets

- [ ] Add a versioned theme catalog with `defaultTheme`, client-to-theme mapping, display name, and a root-relative stylesheet filename.
- [ ] Implement an immutable-at-startup catalog loader rooted at `--theme-dir`; reject traversal, non-CSS files, missing files, duplicate names, and unassigned client references.
- [ ] Implement a theme-selecting interaction renderer that consumes only the validated `InteractionPage.ClientID` supplied by TinyIDP.
- [ ] Implement an allowlisted same-origin asset handler below `/static/themes/`; do not proxy or redirect to application URLs.
- [ ] Update the interaction template to use the selected product name and stylesheet path while retaining every provider-owned form field and the existing CSP contract.
- [ ] Add renderer, loader, route, escaping, and CSP tests for the default and each named theme.

## Phase 3 — Make MessageDesk a GitOps-provided theme

- [ ] Move the MessageDesk IdP CSS source into an application-owned theme source directory and document its review ownership.
- [ ] Add a GitOps Kustomize `configMapGenerator` that packages `themes.json` and reviewed CSS with a content-hashed name.
- [ ] Mount the generated ConfigMap read-only at `/etc/tinyidp/themes`; pass `--theme-dir` and `--theme-catalog-file` to TinyIDP.
- [ ] Remove the embedded MessageDesk stylesheet route from the production host after the mounted asset route is verified.
- [ ] Render the shared IdP manifest and prove a theme source change alters the generated ConfigMap reference and therefore the Pod template.

## Phase 4 — Register and deploy the second application

- [ ] Choose the concrete second example and record its public hostname, callback path, logout path, client ID, required scopes, and UI theme owner.
- [ ] Add that client to the shared IdP client catalog; use a distinct, immutable public client ID.
- [ ] Create a new Argo CD Application and a separate kustomization/namespace for the second app; it must not share the MessageDesk PVC or ServiceAccount.
- [ ] Give the second app an HTTPS Ingress, certificate, Service, restricted NetworkPolicy, trusted-Traefik listener configuration, and canonical-issuer backchannel route through Traefik.
- [ ] Add its theme CSS and catalog mapping through the same reviewed GitOps change.
- [ ] Verify MessageDesk remains functional while the second app completes signup/login/consent/logout against the same IdP.

## Phase 5 — Acceptance, operations, and rollback

- [ ] Extend the public acceptance harness with one independent authorization-code-with-PKCE flow for each client.
- [ ] Assert theme selection from HTML and stylesheet headers without logging cookies, authorization codes, or passwords.
- [ ] Test invalid client/theme mappings, path traversal, a direct untrusted forwarded request, and CSP rejection of off-origin assets.
- [ ] Write an operator runbook for adding a client, changing a theme, disabling a client, rolling back a theme, and recovering a broken catalog.
- [ ] Capture Argo health, certificate readiness, PVC preservation, and audit events for both clients.

## Research branch — TinyIDP Kubernetes operator

- [ ] Define a minimal `TinyIDP` custom resource and the exact objects the controller may own.
- [ ] Decide whether a separate `TinyIDPClient` resource is needed, and define cross-namespace ownership and namespace-selection rules.
- [ ] Prototype only reconciliation planning and status conditions before granting the controller secret, ingress, or Vault authority.
- [ ] Compare operator complexity and failure modes against the GitOps-only design after two applications are operating.
