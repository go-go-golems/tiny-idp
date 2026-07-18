---
Title: Hosted OIDF Basic OP conformance summary
Ticket: TINYIDP-PROD-001
Status: active
Topics:
    - auth
    - go
    - identity
    - oidc
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://docs/conformance.md
      Note: Hosted OIDF runner and distinct-client runbook
    - Path: repo://internal/cmds/serve.go
      Note: Strict CLI extra-client support used by final plan
    - Path: repo://internal/fositeadapter/provider.go
      Note: Strict prompt/max-age/request-object behavior validated by hosted plan
    - Path: repo://scripts/oidf_hosted_runner.py
      Note: Hosted suite automation used for final plan
ExternalSources: []
Summary: Hosted OpenID Foundation Basic OP plan results for tiny-idp strict Fosite engine.
LastUpdated: 2026-07-07T23:58:06.620545927-04:00
WhatFor: Use this as the sanitized evidence summary for hosted OIDF Basic OP runs.
WhenToUse: Before publishing conformance evidence, rerunning hosted tests, or explaining remaining REVIEW/WARNING/SKIPPED statuses.
---


# Hosted OIDF Basic OP conformance summary

## Goal

Record the hosted OpenID Foundation Basic OP conformance results for the tiny-idp strict Fosite engine without committing raw suite logs that may contain transient authorization codes, tokens, or client secrets.

## Final distinct-client plan

- Suite: `https://www.certification.openid.net`
- Suite version observed earlier in the run: `5.2.0`
- Plan ID: `Geeb9MBn659ah`
- Alias: `tinyidp-basic-20260708b`
- Variant: `server_metadata=discovery`, `client_registration=static_client`
- Public issuer used during run: `https://2853-2600-8805-9398-8a00-a781-feaf-fcbd-986c.ngrok-free.app`
- Static clients:
  - `client`: `web-app`
  - `client2`: `web-app-2`
- Result count: `PASSED=21`, `WARNING=6`, `SKIPPED=4`, `REVIEW=4`, hard failures `0`.

`REVIEW` entries are terminal hosted-suite review outcomes after screenshot upload. `SKIPPED` entries are optional-scope or unsupported-request-object cases where discovery did not advertise the optional capability. `WARNING` entries are suite warnings, not interrupted failures.

## Final module results

| Module | Test ID | Status | Result |
| --- | --- | --- | --- |
| `oidcc-server` | `apLPI3h92bMdxfM` | FINISHED | PASSED |
| `oidcc-response-type-missing` | `b7aQqvWO38aHih9` | FINISHED | PASSED |
| `oidcc-userinfo-get` | `ZjUC1wdIcCaroWW` | FINISHED | PASSED |
| `oidcc-userinfo-post-header` | `UPyLe44BjjOdFeO` | FINISHED | PASSED |
| `oidcc-userinfo-post-body` | `ppVd78bAwWaaHyc` | FINISHED | PASSED |
| `oidcc-ensure-request-without-nonce-succeeds-for-code-flow` | `MBvunOt1s2K3cwA` | FINISHED | PASSED |
| `oidcc-scope-profile` | `SdUuUA2mkGFsElG` | FINISHED | WARNING |
| `oidcc-scope-email` | `aF4eBc3wQ7nuD6p` | FINISHED | WARNING |
| `oidcc-scope-address` | `wxUR94rekiW4HlB` | FINISHED | SKIPPED |
| `oidcc-scope-phone` | `gglclEBINYkaEOv` | FINISHED | SKIPPED |
| `oidcc-scope-all` | `4LOhLM9HLxyECBc` | FINISHED | SKIPPED |
| `oidcc-alternate-happy-flow` | `tTVJzDXT9qSs2ZW` | FINISHED | WARNING |
| `oidcc-display-page` | `YlVO0bjPEn0WQyZ` | FINISHED | PASSED |
| `oidcc-display-popup` | `2ror6O7hNKej75X` | FINISHED | PASSED |
| `oidcc-prompt-login` | `RINDRaXQon20ej4` | FINISHED | REVIEW |
| `oidcc-prompt-none-not-logged-in` | `Cb0HqWshL2KaPXc` | FINISHED | PASSED |
| `oidcc-prompt-none-logged-in` | `1ynNsXWpbQJNAXI` | FINISHED | PASSED |
| `oidcc-max-age-1` | `GUAbWQ4hOFAC6m9` | FINISHED | REVIEW |
| `oidcc-max-age-10000` | `RCaqGCcxg0vxBDF` | FINISHED | PASSED |
| `oidcc-ensure-request-with-unknown-parameter-succeeds` | `8BIDl3tdxJuGXeh` | FINISHED | PASSED |
| `oidcc-id-token-hint` | `9OZ82OMEkVm9CSe` | FINISHED | PASSED |
| `oidcc-login-hint` | `ADmyJy9dPJXvRMh` | FINISHED | PASSED |
| `oidcc-ui-locales` | `5QbOR1iXUzshkuL` | FINISHED | PASSED |
| `oidcc-claims-locales` | `XEcEnywYSKpfH30` | FINISHED | PASSED |
| `oidcc-ensure-request-with-acr-values-succeeds` | `DNHtjuy4gtOUPJw` | FINISHED | WARNING |
| `oidcc-codereuse` | `LzQkZlN06YU2lNn` | FINISHED | PASSED |
| `oidcc-codereuse-30seconds` | `9xjnkBuP4ConVTl` | FINISHED | PASSED |
| `oidcc-ensure-registered-redirect-uri` | `u4MOTlfQh54gcas` | FINISHED | REVIEW |
| `oidcc-ensure-post-request-succeeds` | `odynIrYfhEr5mLr` | FINISHED | WARNING |
| `oidcc-server-client-secret-post` | `UyqH6q9k3kIQjQO` | FINISHED | PASSED |
| `oidcc-unsigned-request-object-supported-correctly-or-rejected-as-unsupported` | `PS6Kgxi20WMdukq` | FINISHED | SKIPPED |
| `oidcc-claims-essential` | `e0arVhgiw779uH3` | FINISHED | WARNING |
| `oidcc-ensure-request-object-with-redirect-uri` | `InwzCwvyFBavjbQ` | FINISHED | REVIEW |
| `oidcc-refresh-token` | `s6Wy9BgOnvhsEG5` | FINISHED | PASSED |
| `oidcc-ensure-request-with-valid-pkce-succeeds` | `OxFwwCixcKaDdHu` | FINISHED | PASSED |

## Operational notes

The strict server for the final run was started with a second static client so the refresh-token cross-client misuse test could prove client binding:

```bash
CB='https://www.certification.openid.net/test/a/tinyidp-basic-20260708b/callback'
tinyidp serve --engine fosite \
  --issuer 'https://2853-2600-8805-9398-8a00-a781-feaf-fcbd-986c.ngrok-free.app' \
  --addr 127.0.0.1:5556 \
  --client-id web-app \
  --client-secret dev-secret \
  --redirect-uris "$CB" \
  --redirect-uris "$CB?dummy1=lorem&dummy2=ipsum" \
  --extra-clients "web-app-2|dev-secret-2|$CB|$CB?dummy1=lorem&dummy2=ipsum"
```

The Python runner command used for the final plan was:

```bash
scripts/oidf_hosted_runner.py \
  --plan Geeb9MBn659ah \
  --remaining \
  --artifacts ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/oidf-hosted-python-distinct-clients
```

## Related

- `docs/conformance.md`
- `scripts/oidf_hosted_runner.py`
- `internal/cmds/serve.go`
- `internal/fositeadapter/provider.go`
