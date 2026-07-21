# Changelog

## 2026-07-21

- Initial workspace created


## 2026-07-21

Created the intern-oriented verified-email signup design. The first deployment uses a private operator SMTP outbox while retaining TinyIDP native challenge generation, hashing, binding, verification, and truthful email_verified claims; real SMTP and cluster delivery remain deferred.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpemailchallenge/service.go — Existing native security boundary reused by the design
