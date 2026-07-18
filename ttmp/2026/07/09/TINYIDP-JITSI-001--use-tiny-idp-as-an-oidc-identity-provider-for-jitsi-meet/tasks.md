# Tasks

## Done (research & design)

- [x] Analyze tiny-idp OIDC surface (endpoints, flows, token/claim shapes, RS256/JWKS, seeded users).
- [x] Research Jitsi Meet authentication (Prosody token auth, `mod_auth_token`, ASAP, adapters); 15 sources captured.
- [x] Experiment A: `scripts/01-oidc-smoke.sh` — full OIDC auth-code flow against tiny-idp (verified).
- [x] Experiment B: `scripts/02-oidc-to-jitsi-jwt.py` — OIDC→Jitsi-JWT claim mapping (verified).
- [x] Write intern-ready design/implementation guide (`design-doc/01-...md`).
- [x] Investigation diary (`reference/01-investigation-diary.md`).

## Follow-up (implementation — separate execution)

- [ ] Phase 1: stand up Jitsi in Prosody `token` mode; join with a hand-minted HS256 token.
- [ ] Phase 2: deploy `jitsi-contrib/jitsi-oidc-adapter`; point `OIDC_ISSUER_URL` at tiny-idp; register adapter client.
- [ ] Phase 3: set `config.tokenAuthUrl` + `tokenAuthUrlAutoRedirect`; verify full browser login as `alice`.
- [ ] Phase 4 (optional): moderator mapping (adapter `createContext()` or `token_affiliation`); guest VirtualHost.
- [ ] Decide moderator policy driven by tiny-idp `roles`/`groups`.
- [ ] (Future) evaluate an in-house Go adapter to drop the Deno dependency (ADR-3).
- [ ] (Optional) wire Jitsi logout → tiny-idp `/end-session`.

## Open questions

- Multi-tenant `TENANT/ROOM` URL scoping vs tiny-idp `tenant` claim.
- Production IdP choice (hardened tiny-idp vs Keycloak) — Jitsi side is unaffected.
