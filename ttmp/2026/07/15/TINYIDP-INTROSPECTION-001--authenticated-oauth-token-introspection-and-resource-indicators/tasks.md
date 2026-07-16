# Tasks

## Phase A — Contract, client model, and resource indicators

- [x] Add a documented discovery contract for authenticated `POST /introspect` and an explicit bearer-only/DPOP-rejection policy.
- [ ] Extend client persistence, validation, admin/config/bootstrap input, and safe output with `AllowedAudiences` and `CanIntrospect`.
- [x] Wire allowed audiences into Fosite clients and prove authorization-code/device issuance and refresh preserve only granted audiences.

## Phase B — Endpoint implementation and provider hardening

- [x] Mount `/introspect` for root and path issuers; enforce method/form transport and confidential resource-client Basic authentication.
- [x] Validate opaque access tokens through Fosite, require resource-audience intersection, and return constrained RFC 7662 metadata or exactly inactive.
- [ ] Add bounded audit/security events and endpoint-specific rate limiting without recording token values.

## Phase C — Verification and operator usability

- [ ] Add memory and SQL lifecycle tests: active, unknown, malformed, expired, revoked, refresh rotation, wrong audience, and audience persistence.
- [ ] Add negative tests for public/disabled/wrong resource-client authentication, token oracle resistance, root/path discovery, bearer-only policy, and redaction.
- [ ] Add a strict TLS fixture/smoke and an operator guide for registering a resource server and calling introspection.

## Phase D — xgoja consumer handoff

- [ ] Publish the stable response/configuration contract and handoff checklist for xgoja `oidcresource`.
- [ ] Keep application-owned `programauth` device tokens documented as an independent optional credential system.

## TODO
