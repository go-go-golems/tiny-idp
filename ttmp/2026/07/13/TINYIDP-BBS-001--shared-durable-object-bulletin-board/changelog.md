# Changelog

## 2026-07-13

- Initial workspace created


## 2026-07-13

Phase 1: created the feature ticket, documented the xgoja shared-object boundary, specified the API and state contracts, and added a 25-task implementation ledger.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/design-doc/01-shared-durable-object-bbs-analysis-design-and-implementation-guide.md — Accepted architecture and intern implementation guide


## 2026-07-13

Phase 2: implemented planned xgoja BBS routes, the shared persistent object, explicit host policy actions, and direct plus full-application boundary tests (go-go-goja f9dbf36; tiny-idp 0f5b907).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/app/objects/objects.js — Persistent shared board state machine
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/app/routes/site.js — Trusted fixed-object BBS API


## 2026-07-13

Phase 3: replaced the placeholder page with the typed React/Redux/RTK Query application, implemented the early-Mac monochrome visual system, made hashed asset generation deterministic, and passed the two-user initialized TLS browser scenario across a process restart.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/app/frontend/src/App.tsx — Session-aware BBS presentation and interactions
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/app/frontend/src/api.ts — Typed BBS API and CSRF-aware transport
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/app/frontend/src/styles.css — Early-Mac monochrome visual system
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/scripts/01_real_browser_bbs.py — Two-user browser and restart harness


## 2026-07-13

Phase 4: added the BBS validation matrix, strict RP-initiated current-browser logout, distinct application/IdP logout UX, responsive and keyboard browser assertions, audit/storage inspection, race checks, full repository test/build, and workspace-aware lint gates.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/end_session.go — Strict current-browser RP-initiated logout
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/end_session_test.go — Redirect, revocation, cookie, and audit invariants
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/sources/01-openid-connect-rp-initiated-logout-1-0.md — Defuddled official standard


## 2026-07-13

Verification checkpoint committed as `110b008`; final initialized TLS handoff restarted cleanly in tmux and returned ready status.

Uploaded all eight ticket Markdown documents as one reMarkable PDF bundle to `/ai/2026/07/13/TINYIDP-BBS-001`.

## 2026-07-13

Implementation, verification, tmux handoff, and reMarkable bundle delivery complete.
