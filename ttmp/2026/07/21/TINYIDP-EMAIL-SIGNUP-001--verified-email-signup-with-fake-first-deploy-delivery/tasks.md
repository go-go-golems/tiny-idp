# Tasks

## TODO

- [ ] Phase 0 — Record existing mailer, durable challenge, continuation, verified-evidence, and account-commit contracts
- [ ] Phase 0 — Add a focused test preserving the current fail-closed production rejection before activation
- [ ] Phase 1 — Implement a narrow SMTP email-challenge mailer with a fixed native template catalog
- [ ] Phase 1 — Add TLS modes, timeouts, address validation, retry classification, and secret-safe diagnostics
- [ ] Phase 1 — Test the mailer against an in-process SMTP server without external credentials
- [ ] Phase 2 — Add Glazed production fields and file-backed challenge/SMTP secrets
- [ ] Phase 2 — Construct the durable email-challenge service from the production SQLite store
- [ ] Phase 2 — Permit challenge outcomes only when every native production dependency is configured
- [ ] Phase 2 — Add fail-closed startup and production command tests
- [ ] Phase 3 — Compose open Message Desk signup with mandatory email verification
- [ ] Phase 3 — Compose goja invite-gated signup with invitation inspection before email delivery
- [ ] Phase 3 — Preserve verified evidence into the native password/account commit
- [ ] Phase 3 — Prove optional signup-invite redemption remains atomic with verified account creation
- [ ] Phase 4 — Add a private first-deploy SMTP catcher with no public ingress
- [ ] Phase 4 — Document authenticated operator retrieval, manual relay, retention, and removal
- [ ] Phase 5 — Extend browser acceptance to retrieve and submit codes through the operator outbox
- [ ] Phase 5 — Prove both applications create email-verified accounts and preserve restart/retry behavior
- [ ] Phase 5 — Prove a newly registered goja user can accept its email-bound organization invitation
- [ ] Phase 5 — Assert raw codes and invitation tokens are absent from public responses, URLs, audit, and logs
- [ ] Documentation — Relate decisive source files, validate the ticket, and maintain the implementation diary
