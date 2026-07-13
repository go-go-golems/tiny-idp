---
Title: BBS Verification and Operations Playbook
Ticket: TINYIDP-BBS-001
Status: active
Topics:
    - architecture
    - xgoja
    - identity
    - security
    - testing
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Executable commands and expected evidence for building, testing, running, restarting, and reviewing the shared BBS."
LastUpdated: 2026-07-13T16:27:18.465430152-04:00
WhatFor: "Provide one reproducible path from frontend dependencies through backend tests, browser verification, restart, and final tmux operation."
WhenToUse: "Use during implementation checkpoints, code review, release verification, incident reproduction, and handoff."
---

# BBS Verification and Operations Playbook

## Purpose

This playbook verifies the feature at four layers: the Durable Object domain,
the authenticated HTTP API, the React build, and the real browser. Commands run
from the `tiny-idp` repository unless a section says otherwise.

## 1. Workspace prerequisites

Confirm the top-level workspace resolves local modules:

```bash
go env GOWORK
go list -m github.com/go-go-golems/go-go-goja
go list -m github.com/go-go-golems/go-go-objects
```

Do not create another `go.mod` or private Go cache.

## 2. Frontend build

```bash
pnpm --dir cmd/tinyidp-xapp/app/frontend install --frozen-lockfile
pnpm --dir cmd/tinyidp-xapp/app/frontend run typecheck
pnpm --dir cmd/tinyidp-xapp/app/frontend run build
```

Inspect the output:

```bash
find cmd/tinyidp-xapp/app/frontend/dist -maxdepth 3 -type f -print
rg -n 'src="/static/|href="/static/' \
  cmd/tinyidp-xapp/app/frontend/dist/index.html
```

All built static assets must use `/static/`.

## 3. Generated runtime

```bash
go generate ./cmd/tinyidp-xapp
git status --short cmd/tinyidp-xapp/internal/xgojaruntime
```

Run generation twice after implementation. The second run must be stable.

## 4. Focused checks

```bash
go test ./cmd/tinyidp-xapp -count=1
go test ./cmd/tinyidp-xapp -run 'TestBBS' -count=1 -v
go vet ./cmd/tinyidp-xapp/...
```

Expected coverage includes unauthenticated reads, CSRF rejection, creation,
reply, validation, identity spoofing resistance, author-only deletion, and
restart persistence.

## 5. Full checks

```bash
go test ./... -count=1
go build ./...
```

If a legitimate software error repeats twice, stop under the workspace
debugging rule rather than continuing speculative changes.

## 6. TLS application in tmux

Before starting the server:

```bash
lsof-who -p 19443 -k
tmux kill-session -t tinyidp-xapp 2>/dev/null || true
```

Use the established explicit XAPP command and state root. Do not introduce a
new environment variable for credentials without notifying the user.

```bash
tmux new-session -d -s tinyidp-xapp \
  '<existing explicit tinyidp-xapp serve command>'
tmux capture-pane -pt tinyidp-xapp -S -120
```

## 7. Negative API matrix

```text
GET    /api/bbs                         no session       → 401
POST   /api/bbs/posts                   no CSRF          → 403
POST   /api/bbs/posts                   blank title      → 400
POST   /api/bbs/posts/:id/replies       missing post     → 404
DELETE /api/bbs/posts/:id               different actor  → 403
```

Submit `actorId`, `actorName`, `namespace`, and `objectName` fields in public
JSON. The resulting content must still use the session actor and shared board.

## 8. Browser scenario

The ticket browser script creates separate Alice and Bob contexts. It must not
print cookies, CSRF tokens, passwords, OIDC subjects, or physical object IDs.

1. Alice signs in and creates a uniquely titled project post.
2. Bob signs in separately and sees it attributed to Alice.
3. Bob replies and cannot delete Alice's post.
4. Bob's direct deletion receives 403.
5. Hostile markup renders as literal text.
6. The server restarts with the same state root.
7. Alice sees the post and Bob's reply after restart.
8. Alice deletes the thread and counters update.
9. Logout invalidates the application session.

## 9. Persistence evidence

Stop the server through tmux before inspecting or copying SQLite files. The
restart proof is behavioral:

```text
create → close application → construct application with same state root
→ read board → assert same post and reply
```

Back up the complete Durable Objects storage root rather than copying one live
SQLite file without its WAL.

## 10. Documentation and upload

```bash
docmgr doctor --ticket TINYIDP-BBS-001 --stale-after 30
remarquee upload bundle --dry-run \
  ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/index.md \
  ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/design-doc/01-shared-durable-object-bbs-analysis-design-and-implementation-guide.md \
  ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/reference/01-implementation-diary.md \
  ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/playbook/01-bbs-verification-and-operations-playbook.md \
  --name "TINYIDP BBS 001 Shared Durable Object Bulletin Board" \
  --remote-dir "/ai/2026/07/13/TINYIDP-BBS-001" \
  --toc-depth 2 --non-interactive
```

After the dry run, repeat without `--dry-run`. The successful upload message is
delivery evidence; a routine cloud listing is unnecessary.

## 11. Review checklist

- The Go host contains no BBS-specific HTTP handler.
- Every mutation declares authentication, CSRF, authorization, and audit.
- Every named object call uses fixed literals.
- Browser fields cannot overwrite trusted actor fields.
- Public JSON contains no actor IDs or object details.
- React renders board content as text.
- Static assets are served under `/static/`.
- Two users share state and have distinct deletion permissions.
- State survives process restart.
- Generated output is reproducible.

## Purpose

<!-- What does this playbook accomplish? -->

## Environment Assumptions

<!-- What environment or setup is required? -->

## Commands

<!-- List of commands to execute -->

```bash
# Command sequence
```

## Exit Criteria

<!-- What indicates success or completion? -->

## Notes

<!-- Additional context or warnings -->
