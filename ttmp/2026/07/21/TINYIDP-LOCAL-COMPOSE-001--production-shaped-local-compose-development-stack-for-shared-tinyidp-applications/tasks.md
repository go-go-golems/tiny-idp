# Tasks

## TODO

- [ ] Map the existing Compose demo, production manifests, and goja auth-host runtime contract. <!-- t:lera -->
- [ ] Implement a strict local HTTPS proxy topology with both client registrations and per-client themes. <!-- t:e5uc -->
- [ ] Document the local run, browser trust, reset, observability, and optional local goja build workflows. <!-- t:42sv -->
- [ ] Validate Compose configuration, build and start the full stack, then exercise both OIDC applications. <!-- t:zq12 -->
- [ ] Publish the intern guide to reMarkable and complete ticket bookkeeping. <!-- t:1f86 -->
- [x] Define a bounded theme-aware browser error presentation contract for registration rejections. <!-- t:04vh -->
- [x] Implement default and production browser error renderers with strict CSP and safe fallback behavior. <!-- t:dgyc -->
- [x] Route rejected registration POSTs through the client-specific themed error page without changing OAuth redirects. <!-- t:m3o9 -->
- [x] Add unit, renderer, and provider integration coverage for themed registration rejection pages. <!-- t:x755 -->
- [x] Rebuild the local stack and validate the themed rejection response over trusted HTTPS. <!-- t:141x -->
- [x] Define the browser-state, identity-state, application-state, and validation-path test matrix with explicit UI expectations. <!-- t:zpds -->
- [x] Add a Playwright test project for the real local HTTPS Compose stack with trusted CA, deterministic fixtures, screenshots, traces, and failure artifacts. <!-- t:c08j -->
- [ ] Cover signup validation paths: malformed email, duplicate email, short password, password mismatch, email code failures, and invite failures. <!-- t:m17w -->
- [x] Cover session navigation paths: first login, remembered-account selection, add-account signup, Message Desk-only logout, TinyIDP logout, and account switching. <!-- t:gix8 -->
- [x] Cover both relying applications and assert themed HTML for recoverable and terminal authentication failures. <!-- t:r9rq -->
- [ ] Classify browser-test failures in the defect ledger and implement the provider or application UX fixes without weakening security checks. <!-- t:8vbu -->
- [ ] Run the complete Playwright matrix against a fresh and retained local stack, document evidence, and close the discovered defects. <!-- t:1lta -->
- [x] Verify durable email-code exhaustion recovery: committed SQLite attempts, rotated resend code, and real-browser replacement-code success <!-- t:u4n1 -->
