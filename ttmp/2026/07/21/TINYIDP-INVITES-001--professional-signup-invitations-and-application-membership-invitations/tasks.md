# Tasks

## TODO

- [x] Phase 0 — Record the trust-boundary invariants separating TinyIDP identity creation from application membership <!-- t:umwx -->
- [x] Phase 0 — Inventory the existing TinyIDP scripting, continuation, invitation, transaction, and audit contracts <!-- t:7r1v -->
- [x] Phase 0 — Inventory go-go-goja capability, OIDC normalization, membership, and example-route contracts <!-- t:22oy -->
- [x] Phase 1 — Bind TinyIDP durable invitation lookup to declared invitation-provider lambdas in production <!-- t:h0xf -->
- [x] Phase 1 — Enable the consumeInvitation effect in the production signup-program validator <!-- t:06re -->
- [x] Phase 1 — Construct the durable invitation service with an operator-managed lookup key and auditable lifecycle <!-- t:7d89 -->
- [x] Phase 1 — Add operator commands to issue and revoke signup invitations without building an admin web UI <!-- t:5nvk -->
- [x] Phase 2 — Define and implement a concrete invite-required signup program using inspect-then-commit semantics <!-- t:mfoy -->
- [x] Phase 2 — Add denial rendering and stable error categories without exposing token validity or secrets <!-- t:9bej -->
- [x] Phase 2 — Test expiry, revocation, replay, audience mismatch, concurrent redemption, and transaction rollback <!-- t:cc0l -->
- [x] Phase 3 — Add a go-go-goja application operation that atomically consumes an org invite and creates membership <!-- t:aghi -->
- [x] Phase 3 — Require an authenticated actor and enforce verified-email or subject binding when accepting membership invitations <!-- t:osl2 -->
- [x] Phase 3 — Add deployment bootstrap for the initial tenant, resource, and administrator membership <!-- t:w3hu -->
- [x] Phase 4 — Preserve a pending application invite through signup, OIDC authorization, callback, and membership acceptance <!-- t:ucs8 -->
- [x] Phase 4 — Add a goja-host registration entry route that requests TinyIDP signup without conflating login and registration <!-- t:g2np -->
- [x] Phase 4 — Define retry behavior for the cross-database saga when identity creation succeeds but membership acceptance fails <!-- t:ys5g -->
- [x] Phase 5 — Validate both open-signup Message Desk and invite-gated goja application flows in the local shared Compose stack <!-- t:1jyp -->
- [x] Phase 5 — Add browser-level acceptance tests and operational audit/log checks for invitation issuance and redemption <!-- t:b5yp -->
- [x] Documentation — Write the intern-oriented analysis, design, API reference, pseudocode, diagrams, file map, and implementation plan <!-- t:crsf -->
- [x] Documentation — Relate the decisive source files, validate the ticket, and upload the guide to reMarkable <!-- t:rc3z -->
