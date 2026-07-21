# Tasks

## TODO

- [x] Phase 0 — Record existing mailer, durable challenge, continuation, verified-evidence, and account-commit contracts
- [x] Phase 0 — Add a focused test preserving the current fail-closed production rejection before activation
- [x] Phase 1 — Implement a narrow SMTP email-challenge mailer with a fixed native template catalog
- [x] Phase 1 — Add TLS modes, timeouts, address validation, retry classification, and secret-safe diagnostics
- [x] Phase 1 — Test the mailer against an in-process SMTP server without external credentials
- [x] Phase 2 — Add Glazed production fields and file-backed challenge/SMTP secrets
- [x] Phase 2 — Construct the durable email-challenge service from the production SQLite store
- [x] Phase 2 — Permit challenge outcomes only when every native production dependency is configured
- [x] Phase 2 — Add fail-closed startup and production command tests
- [x] Phase 3 — Compose open Message Desk signup with mandatory email verification
- [x] Phase 3 — Compose goja invite-gated signup with invitation inspection before email delivery
- [x] Phase 3 — Preserve verified evidence into the native password/account commit
- [x] Phase 3 — Prove optional signup-invite redemption remains atomic with verified account creation
- [x] Phase 4 — Add a private first-deploy SMTP catcher with no public ingress
- [x] Phase 4 — Document authenticated operator retrieval, manual relay, retention, and removal
- [x] Phase 5 — Extend browser acceptance to retrieve and submit codes through the operator outbox
- [x] Phase 5 — Prove both applications create email-verified accounts and preserve restart/retry behavior
- [x] Phase 5 — Prove a newly registered goja user can accept its email-bound organization invitation
- [x] Phase 5 — Assert raw codes and invitation tokens are absent from public responses, URLs, audit, and logs
- [x] Documentation — Relate decisive source files, validate the ticket, and maintain the implementation diary
