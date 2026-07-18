---
Title: Investigation Diary
Ticket: TINYIDP-JITSI-001
Status: active
Topics:
    - oidc
    - jitsi
    - authentication
    - research
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological investigation diary for evaluating tiny-idp as an OIDC identity provider for Jitsi Meet."
LastUpdated: 2026-07-09T12:20:00-04:00
WhatFor: "Record the step-by-step research/design journey, commands, sources, experiments, failures, and decisions."
WhenToUse: "When continuing the investigation or reviewing how the design conclusions were reached."
---

# Investigation Diary — tiny-idp as an OIDC IdP for Jitsi Meet

## Original prompt (verbatim intent)

> Analyze tiny-idp and create a docmgr ticket to analyze (and test?) whether we can use tiny-idp as an identity
> provider for Jitsi Meet. Create a detailed analysis / design / implementation guide for a new intern, explaining
> all the parts of the system, with prose, bullet points, pseudocode, diagrams, API references, and file references.
> Store it in the ticket and upload to reMarkable. Search the web and save interesting documents to `sources/` with
> defuddle; store scripts in `scripts/`. Keep a detailed diary and commit at appropriate intervals. A colleague is
> also working on tiny-idp, so if things don't compile, be patient and retry later.

## Step 0 — Orientation (2026-07-09)

**What I did.** Located the workspace at `/home/manuel/workspaces/2026-07-07/prod-tiny-idp`. It is a Go
`go.work` workspace containing `glazed`, `go-go-goja`, and `tiny-idp`. Confirmed `docmgr`, `defuddle`,
`remarquee`, and `surf` are all installed and on PATH.

**tiny-idp at a glance (from `tiny-idp/README.md`).** A minimal *mock* OpenID Connect provider built on the
Glazed CLI framework, explicitly labelled **"not production-grade"**, intended to replace Keycloak-in-Docker for
local dev/integration testing. It issues **RS256-signed ID tokens** and implements a surprisingly complete OIDC/OAuth2
surface.

**Endpoints observed** (`README.md` "Endpoints" table; `internal/server/*`):

- `GET /.well-known/openid-configuration` — discovery
- `GET /jwks` — public signing keys (JWKS)
- `GET /authorize` — login form → authorization code
- `POST /device_authorization`, `GET/POST /device` — device-code grant
- `POST /token` — `authorization_code`, `refresh_token`, device-code grants
- `GET /userinfo` — bearer/DPoP access token → claims
- `GET /end-session` — RP-initiated logout
- `GET /healthz`, `GET/POST /debug/*` (loopback only)

**Signing internals** (`internal/keys/keys.go`): RSA-2048, `RS256`, header `{"typ":"JWT","alg":"RS256","kid":<id>}`.
JWKS is standard `{kty:"RSA",use:"sig",kid,alg:"RS256",n,e}` (`PublicJWK`). Discovery advertises
`id_token_signing_alg_values_supported: ["RS256"]`, `response_types: ["code"]`,
`grant_types: ["authorization_code","refresh_token"]`, `code_challenge_methods: ["S256"]`
(`internal/oidcmeta/discovery.go`).

**User claim shape** (`internal/domain/claims.go`, `ClaimsForScopes`): `sub`; with `email` scope →
`email`, `email_verified`; with `profile` scope → `name`, `preferred_username`, `groups`, `roles`, `tenant`, `locale`.
Seeded users (`README.md`, `internal/user/user.go`) can pin `sub`/`email`/`name`/`password`/`groups`/`roles` and add
arbitrary raw `claims`.

**Early key insight.** tiny-idp is a *standards-shaped* OIDC provider (discovery + JWKS + code/PKCE + userinfo). But
its ID token is a **generic OIDC ID token**, not a **Jitsi-shaped JWT** (which needs `room`, and a `context.user`
object). And its public keys are served as **JWKS**, whereas Jitsi's classic token auth fetches **PEM-by-kid** from an
`asap_key_server`. So the interesting design question is not "is tiny-idp a valid OIDC IdP" (it is) but "what glue is
needed between a generic OIDC IdP and Jitsi's token model." That is what the research agent is chasing on the Jitsi side.

## Step 1 — Ticket + scaffolding

**What I did.**
- `docmgr ticket create --ticket TINYIDP-JITSI-001 --title "Use tiny-idp as an OIDC Identity Provider for Jitsi Meet" --topics oidc,jitsi,authentication,research,architecture`
- Added a `design-doc` and this `reference` diary via `docmgr doc add`.
- Created `sources/web/` and `scripts/` subfolders.
- Launched a background research agent to gather Jitsi authentication evidence and save `sources/web/NN-*.md` captures with `defuddle`.

**Commands run.**
```bash
docmgr ticket create --ticket TINYIDP-JITSI-001 \
  --title "Use tiny-idp as an OIDC Identity Provider for Jitsi Meet" \
  --topics oidc,jitsi,authentication,research,architecture
docmgr doc add --ticket TINYIDP-JITSI-001 --doc-type design-doc --title "tiny-idp as an OIDC ... Guide"
docmgr doc add --ticket TINYIDP-JITSI-001 --doc-type reference --title "Investigation Diary"
```

## Step 2 — Jitsi-side research (complete)

Background research agent captured **15 sources** into `sources/web/` (`01`–`15`, no `11`). Key findings:

- **Jitsi has no native OIDC login.** (`sources/web/03-jitsi-native-oidc-issue-16576.md`, issue #16576, open.)
  Jitsi authenticates with a **Jitsi-shaped JWT** validated by Prosody's `mod_auth_token`, delivered as a
  `?jwt=` **query param** — not an OIDC fragment. A translation adapter is therefore mandatory.
- **Jitsi JWT claim contract** (`sources/web/01-lib-jitsi-meet-tokens.md`): validated `iss`/`aud`/`sub`/`room`/
  `exp`; display-only `context.user{id,name,email,avatar}` (all must be strings). `sub` = tenant/domain (or `*`),
  `room` = room name (or `*`).
- **No JWKS support** (issue #15182, open). RS256 only via `asap_key_server` PEM-by-kid:
  `{server}/{sha256hex(kid)}.pem`. So tiny-idp's JWKS cannot be handed to Jitsi directly.
- **All adapters converge** on: run OIDC auth-code flow → read userinfo → mint **HS256** Jitsi JWT with a secret
  shared with Prosody → redirect `?jwt=`. Recommended: **`jitsi-contrib/jitsi-oidc-adapter`** (generic, Deno).
  It uses only **discovery + authorize + token + userinfo** and does **not** check the IdP's ID-token signature
  or JWKS (`sources/web/13-jitsi-oidc-adapter-adapter-ts.txt`, `14`, `15`).

**Decision recorded (ADR-2 in the design doc):** use HS256 shared-secret to Prosody, not RS256/ASAP — sidesteps
the JWKS gap. **Verdict: feasible via adapter; tiny-idp needs zero new features.**

## Step 3 — Experiments (both pass)

**Experiment A — `scripts/01-oidc-smoke.sh`.** Starts tiny-idp (mock engine) and drives the full OIDC auth-code
flow with curl + a cookie jar. First run failed: `REPO_ROOT` path math was off by one (`.../cmd/tinyidp: directory
not found`) — fixed from 7 `../` to 6. Second run (via a prebuilt `TINYIDP_BIN` to skip the slow `go run` of the
whole `go.work`) succeeded end to end: discovery, JWKS (3 keys), `302 …?code=…`, `/token` → `id_token`+`access_token`,
decoded claims (`sub=user-alice-fixed`, `name=Alice Inbox`, `email`, `groups`, `roles`, `tenant`), and `/userinfo`.
Output saved to `scripts/01-oidc-smoke.output.txt`. **This proves tiny-idp already exposes everything the adapter
consumes.**

- Command: `TINYIDP_BIN=<built> USERS_FILE=examples/users/personal-inbox-users.yaml LOGIN=alice PASSWORD=alice-password bash scripts/01-oidc-smoke.sh`
- Gotcha: default `serve` engine is `mock` (`internal/cmds/serve.go:99`), whose `/authorize` has **no CSRF**
  (simple `login`+`password`+hidden fields) — so a curl flow needs no CSRF token. The `fosite` engine *does* add CSRF.

**Experiment B — `scripts/02-oidc-to-jitsi-jwt.py`.** Reproduces the adapter's claim mapping (`context.ts`) in ~40
dependency-free lines and mints a valid **HS256 Jitsi JWT** from tiny-idp claims. Run with `--room standup --sub
personal --moderator --now <fixed>` for reproducible output (`scripts/02-oidc-to-jitsi-jwt.output.txt`). Confirms the
two-`sub` subtlety: OIDC `sub` → `context.user.id`; Jitsi top-level `sub` = tenant from the room URL.

## Step 4 — Design doc + bookkeeping

Wrote `design-doc/01-...md` (15 sections: primer, current-state evidence for both sides with file/source refs,
architecture + diagrams, claim-mapping table + pseudocode, API references, 4 ADRs, end-to-end sequence diagram,
phased implementation plan, testing strategy, risks/alternatives, intern onboarding). Committed research + probes as
`85b3fde` (ticket dir only; the 60 MB built binary is git-ignored, never committed).

## Step 5 — Finalize

`docmgr doctor --ticket TINYIDP-JITSI-001` initially warned about (a) unknown vocab topics
(`architecture`/`authentication`/`jitsi`/`research`) and (b) four `missing_related_file` entries. Cause of (b):
I prefixed the relate paths with `$ABS/tiny-idp/internal/...` but `$ABS` already ended in `tiny-idp`, so the stored
`repo://` paths were doubled. Fixed by adding the vocab topics and rewriting the four RelatedFiles paths to
`repo://internal/...`. Doctor now passes clean (all checks passed). Uploaded the bundle (index, design guide, diary,
tasks, changelog) to reMarkable at `/ai/2026/07/09/TINYIDP-JITSI-001` via `remarquee upload bundle`.

## Open questions being tracked

- Does any Jitsi component accept a **JWKS URL** directly, or is PEM-by-kid the only RS256 path? (Affects whether a shim is mandatory.)
- Does native `tokenAuthUrl` OIDC require the IdP to mint a Jitsi JWT, or does Jitsi mint it after an OIDC login?
- Minimal viable path: adapter (nordeck) vs. teaching tiny-idp to emit a Jitsi-shaped JWT directly.

## Follow-ups / needs a second pair of eyes

- Confirm exact `context.user` field names Jitsi consumes (moderator flag, avatar, id vs sub).
- Confirm ASAP PEM path template `{asap_key_server}/{sha256hex(kid)}.pem`.
