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

## Step 2 — Jitsi-side research (in progress)

Background research agent tasked with: Prosody auth modes, `mod_auth_token` JWT claims, ASAP/RS256 key server,
native OIDC (`tokenAuthUrl`/`tokenAuthUrlAutoRedirect`), and the `nordeck/jitsi-keycloak-adapter` bridge. Sources are
being captured into `sources/web/`. (Findings and citations recorded in the next diary update.)

## Open questions being tracked

- Does any Jitsi component accept a **JWKS URL** directly, or is PEM-by-kid the only RS256 path? (Affects whether a shim is mandatory.)
- Does native `tokenAuthUrl` OIDC require the IdP to mint a Jitsi JWT, or does Jitsi mint it after an OIDC login?
- Minimal viable path: adapter (nordeck) vs. teaching tiny-idp to emit a Jitsi-shaped JWT directly.

## Follow-ups / needs a second pair of eyes

- Confirm exact `context.user` field names Jitsi consumes (moderator flag, avatar, id vs sub).
- Confirm ASAP PEM path template `{asap_key_server}/{sha256hex(kid)}.pem`.
