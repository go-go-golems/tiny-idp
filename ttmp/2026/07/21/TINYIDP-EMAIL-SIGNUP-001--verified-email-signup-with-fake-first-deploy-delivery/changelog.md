# Changelog

## 2026-07-21

- Initial workspace created


## 2026-07-21

Created the intern-oriented verified-email signup design. The first deployment uses a private operator SMTP outbox while retaining TinyIDP native challenge generation, hashing, binding, verification, and truthful email_verified claims; real SMTP and cluster delivery remain deferred.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpemailchallenge/service.go — Existing native security boundary reused by the design

## 2026-07-21

Implemented the native SMTP adapter, fail-closed production email-challenge construction, combined open/invite-gated verified signup program, private authenticated Mailpit outbox, and full browser acceptance including restart, wrong-code retry, new-user membership, replay, audit, and log-redaction checks (commits 9a5605f, b79f77d, ef597a1).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py — Executable completion evidence


## 2026-07-21

Ticket closed
