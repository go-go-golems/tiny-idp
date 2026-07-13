---
Title: Browser, Accessibility, and Canary Evidence
Ticket: TINYIDP-UI-001
Status: active
Topics:
    - oidc
    - identity
    - security
    - go
    - architecture
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp-xapp/internal/loginui/renderer.go
      Note: Renderer exercised by the real-browser probe
    - Path: cmd/tinyidp-xapp/internal/loginui/static/login.css
      Note: Theme measured for focus, reflow, and contrast
    - Path: ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/scripts/browser_probe.py
      Note: Reproducible sanitized Chromium assurance probe
ExternalSources: []
Summary: Sanitized real-browser assurance evidence for the customized interaction renderer.
LastUpdated: 2026-07-13T18:38:01.463770747-04:00
WhatFor: ""
WhenToUse: ""
---


# Browser, Accessibility, and Canary Evidence

## Goal

Record reproducible and sanitized browser evidence for the xapp interaction UI.
This document reports only structural facts, status codes, origin classes, and
aggregate accessibility measurements. It does not contain credentials, cookies,
hidden input values, OAuth request values, authorization codes, or page source.

## Context

The evidence was collected on 2026-07-13 against a disposable local canary at
`http://127.0.0.1:8790`. The server ran in tmux from the current branch and used
a fresh temporary state directory. Chromium 150 was controlled through
Playwright 1.50. Two dedicated nonproduction accounts were seeded so browser
contexts could prove they received distinct authenticated subjects.

The executable probe is `scripts/browser_probe.py`. It uses the real xapp route
chain:

```text
/auth/login
    -> /idp/authorize
        -> POST /idp/authorize
            -> /auth/callback
                -> /
```

The same browser loaded the real embedded CSS from
`/static/tinyidp/login.css`. No mock HTML or isolated component story was used.

## Quick Reference

### Result summary

| Check | Result |
| --- | --- |
| Authorization document status | 200 |
| Stylesheet status and type | 200, `text/css; charset=utf-8` |
| Request origins | only the configured loopback origin |
| Script elements | 0 |
| Inline style elements/attributes | 0 |
| Event-handler attributes | 0 |
| Password inputs | exactly 1 |
| Password `value` attributes | 0 |
| Username autocomplete | `username` |
| Password autocomplete | `current-password` |
| Actions | exactly `approve` and `deny` |
| Frame embedding | blocked; child became `chrome-error:` |
| Keyboard order | username, password, approve, deny |
| Visible focus | solid 4 px outline |
| 320 px horizontal overflow | false |
| 200% zoom horizontal overflow | false |
| Login/OIDC callback | completed to `/` |
| Application session | user and CSRF fields present, values not recorded |
| Two-account isolation | distinct authenticated subjects |

### CSP

The browser received exactly:

```text
default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'
```

The stylesheet was the only subresource. The recorded origin set contained one
origin, the disposable canary itself. This gives positive evidence that the page
did not load analytics, remote fonts, images, scripts, or third-party CSS.

### Contrast measurements

Computed foreground/background colors were measured in the browser using WCAG
relative luminance. All sampled normal text exceeded 4.5:1:

| Sample | Ratio |
| --- | ---: |
| Body text | 14.12:1 |
| Approve button | 10.34:1 |
| Deny button | 8.99:1 |
| Identity eyebrow | 10.34:1 |
| Footer | 6.94:1 |
| Explanatory lede | 6.94:1 |
| Scope label | 9.92:1 |

### Manual visual review

The captured screenshot was inspected from `/tmp`, not committed. It showed:

- a centered, bounded identity panel without menu bar or window chrome;
- a monochrome paper/ink foundation;
- restrained pastel mint, rose, blue, and gold accents;
- visible labels and requested scopes;
- square controls and focusable approve/deny actions;
- no credential, token, cookie, or hidden-field value visible in the image.

The screenshot confirmed the requested retro monochrome direction and did not
introduce a Chicago font dependency.

## Usage Examples

Start a disposable two-user development canary in tmux, using dedicated test
credentials and a fresh state directory. Then run:

```bash
PYENV_VERSION=3.11.4 python \
  ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/scripts/browser_probe.py \
  --base-url http://127.0.0.1:8790 \
  --login '<test-login-1>' \
  --password '<test-password-1>' \
  --second-login '<test-login-2>' \
  --second-password '<test-password-2>' \
  --screenshot /tmp/tinyidp-ui-login.png
```

The probe deliberately accepts credentials as arguments because the existing
development server does. Do not use production credentials, do not paste the
command into a ticket, and do not retain shell history from a sensitive host.

Kill the canary with the required port-aware command:

```bash
lsof-who -p 8790 -k
tmux kill-session -t tinyidp-xapp-ui-probe
```

Expected output is bounded JSON. It contains booleans, counts, paths, status,
content type, origin, CSP, focus geometry, and contrast ratios. It does not emit
page HTML or any submitted/returned secret.

### Limits of this evidence

- Chromium and Playwright do not prove equivalent behavior in every assistive
  technology or browser engine.
- Autocomplete attributes establish password-manager compatibility, but the
  probe does not automate a specific password-manager extension.
- The frame test proves enforcement in the tested Chromium build and is backed
  by both CSP `frame-ancestors 'none'` and `X-Frame-Options: DENY`.
- The canary was local and disposable. A production deployment still requires
  observation in its real proxy, TLS, and asset-serving topology.
- A screen-reader review remains a human release-sign-off activity even though
  parsed-DOM labels, landmarks, alerts, and keyboard order are automated.

## Related

- `design-doc/01-secure-interaction-rendering-analysis-design-and-implementation-guide.md`
- `reference/01-investigation-diary.md`
- `reference/03-interaction-ui-release-and-rollback-runbook.md`
- `docs/interaction-rendering.md`
