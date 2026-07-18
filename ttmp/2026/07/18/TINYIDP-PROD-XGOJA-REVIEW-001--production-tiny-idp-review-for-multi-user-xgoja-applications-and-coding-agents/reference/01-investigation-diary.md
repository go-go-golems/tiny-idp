---
Title: Investigation diary
Ticket: TINYIDP-PROD-XGOJA-REVIEW-001
Status: active
Topics:
    - architecture
    - auth
    - identity
    - oauth2
    - oidc
    - operations
    - research
    - security
    - testing
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological evidence, commands, findings, decisions, validation results, and delivery record for the production IdP review."
LastUpdated: 2026-07-18T13:58:42.125849394-04:00
WhatFor: "Preserve how the review was conducted and make its evidence and validation reproducible."
WhenToUse: "When reviewing the final design, reproducing findings, or continuing the investigation."
---

# Diary

## Goal

This diary records the evidence-first review of `tiny-idp` as the production
identity plane for multi-user xgoja applications and coding agents. It preserves
the commands, conclusions, failures, review risks, and delivery steps needed for
another engineer to reproduce or continue the work.

## Step 1: Establish the ticket and research contract

The first step created a dedicated docmgr workspace and fixed the scope before
reading implementation details. The review must connect the identity provider,
browser-facing user lifecycle, xgoja Express authentication syntax, resource
server enforcement, and device-authorized coding-agent access without treating
them as unrelated features.

The deliverable is documentation and review guidance, not a compatibility layer
or an implementation patch. Existing focused tickets are evidence inputs; this
ticket synthesizes the end-to-end production model and names unresolved gaps.

### Prompt Context

**User prompt (verbatim):** "Creat ea new docmgr ticket to do a review of tiny-idp/ in the axis of using it as a production idp for allowing a set of multi-user xgoja apps using the go-go-goja express auth syntax to be deployed, allowing users to signup, use the site, but also use the device auth path to register coding agents to use the API.

Create  a detailed analysis / design / code review guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

You can write and run experiments in the docmgr ticket's scripts/ directory. 

Keep a detailed diary as you work, commit at appropriat eintervals."

**Assistant interpretation:** Create a new ticket containing an evidence-backed,
intern-facing production-readiness review and target design for tiny-idp-backed
multi-user xgoja applications, including browser signup/login and coding-agent
OAuth device authorization, then validate, commit, and upload it to reMarkable.

**Inferred user intent:** Establish a technically precise shared model that a new
engineer can use to understand the system, review its security and production
gaps, and implement the remaining work without rediscovering the architecture.

### What I did

- Read the ticket research, docmgr, diary, textbook-authoring, and reMarkable upload instructions.
- Confirmed the `tiny-idp` worktree was clean on `task/prod-tiny-idp`.
- Inspected the existing ticket registry and vocabulary.
- Created `TINYIDP-PROD-XGOJA-REVIEW-001`, its design document, this diary, and eight research/delivery tasks.
- Defined the writing contract: direct technical prose, conceptual foundations first, no analogies, complete-sentence bullets, and explicit observed-versus-proposed labels.

### Why

- A separate synthesis ticket preserves the boundaries of existing focused tickets while providing one end-to-end production review.
- Recording the scope first prevents later code reading from narrowing the analysis to whichever package is easiest to inspect.

### What worked

- `docmgr status --summary-only` located the workspace at `tiny-idp/ttmp` and reported a healthy existing documentation corpus.
- `docmgr ticket create-ticket` and both `docmgr doc add` calls created the expected index, task list, changelog, design document, and diary.
- Existing vocabulary already covered every required topic.

### What didn't work

- The first `git status --short --branch` ran from the multi-repository workspace root and returned exactly: `fatal: not a git repository (or any of the parent directories): .git`. Re-running it from `tiny-idp/` succeeded.
- The first ticket-file search targeted `ttmp` at the workspace root, but `.ttmp.yaml` points docmgr at `tiny-idp/ttmp`; the corrected search used that repository root.

### What I learned

- The workspace contains neighboring checkouts for `tiny-idp`, `go-go-goja`, and `go-go-objects`; identity-provider documentation belongs in the `tiny-idp` worktree even when it cites integration code from a neighboring repository.
- Existing tickets cover device authorization, browser account selection, deployment validation, token introspection, and an xgoja device client. This ticket must synthesize them and verify current code rather than copy their conclusions.

### What was tricky to build

- The scope crosses repository boundaries but only one repository owns the ticket. The solution is to store the review in `tiny-idp/ttmp`, cite neighboring files by absolute path where they materially shape the design, and avoid editing those repositories during a documentation-only review.

### What warrants a second pair of eyes

- Review whether the final target architecture draws the issuer/resource-server boundary correctly when several xgoja applications share one IdP but retain distinct clients, audiences, scopes, and application data stores.
- Check every security conclusion against current code rather than assuming a focused ticket was fully implemented.

### What should be done in the future

- Keep focused implementation work in follow-up tickets after this review establishes priorities and ownership boundaries.

### Code review instructions

- Start with this diary, then the ticket index and primary design document.
- Reproduce ticket discovery with `docmgr ticket list --ticket TINYIDP-PROD-XGOJA-REVIEW-001`.
- Confirm no unrelated source changes with `git status --short` from `tiny-idp/`.

### Technical details

Initial commands:

```text
docmgr status --summary-only
docmgr ticket create-ticket --ticket TINYIDP-PROD-XGOJA-REVIEW-001 ...
docmgr doc add --ticket TINYIDP-PROD-XGOJA-REVIEW-001 --doc-type design-doc ...
docmgr doc add --ticket TINYIDP-PROD-XGOJA-REVIEW-001 --doc-type reference --title "Investigation diary"
```
