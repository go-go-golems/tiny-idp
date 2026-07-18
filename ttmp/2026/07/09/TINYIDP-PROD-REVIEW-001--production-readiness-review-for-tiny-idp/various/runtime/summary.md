---
Title: tiny-idp runtime probe summary
Ticket: TINYIDP-PROD-REVIEW-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Measured strict production-mode login, token, refresh, read-path, Go runtime, SQLite pool, and audit behavior."
LastUpdated: 2026-07-09T21:27:54Z
WhatFor: "Providing a bounded runtime regression baseline for the production review."
WhenToUse: "Use when changing authentication, Fosite, SQLite, runtime limits, or observability."
---

# tiny-idp runtime probe summary

Source: `ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/events.ndjson`

Requests observed: **45**. Audit events emitted: **9**.

## HTTP observations

| Operation | Count | Statuses | p50 | p95 | p99 | Mean bytes | Errors |
|---|---:|---|---:|---:|---:|---:|---:|
| `GET /.well-known/openid-configuration (load_.well-known_openid-configuration)` | 10 | 200×10 | 70 µs | 145 µs | 145 µs | 790 | 0 |
| `GET /authorize (authorize_get)` | 1 | 200×1 | 527 µs | 527 µs | 527 µs | 1040 | 0 |
| `GET /jwks (load_jwks)` | 10 | 200×10 | 293 µs | 382 µs | 382 µs | 433 | 0 |
| `GET /readyz (load_readyz)` | 10 | 200×10 | 131 µs | 221 µs | 221 µs | 6 | 0 |
| `GET /userinfo (load_userinfo)` | 10 | 200×10 | 159 µs | 487 µs | 487 µs | 143 | 0 |
| `GET /userinfo (userinfo)` | 1 | 200×1 | 199 µs | 199 µs | 199 µs | 143 | 0 |
| `POST /authorize (authorize_post)` | 1 | 303×1 | 81.94 ms | 81.94 ms | 81.94 ms | 0 | 0 |
| `POST /token (token_exchange)` | 1 | 200×1 | 6.28 ms | 6.28 ms | 6.28 ms | 1227 | 0 |
| `POST /token (token_refresh)` | 1 | 200×1 | 4.81 ms | 4.81 ms | 4.81 ms | 1227 | 0 |

## Go runtime deltas

| Metric | Before | After | Delta |
|---|---:|---:|---:|
| `/cpu/classes/gc/total:cpu-seconds` | 0.003 | 0.005 | +0.001 |
| `/gc/cycles/total:gc-cycles` | 3.000 | 4.000 | +1.000 |
| `/gc/heap/allocs:bytes` | 204550696.000 | 272452032.000 | +67901336.000 |
| `/gc/heap/live:bytes` | 69238352.000 | 69388336.000 | +149984.000 |
| `/gc/heap/objects:objects` | 3860.000 | 7243.000 | +3383.000 |
| `/memory/classes/heap/objects:bytes` | 69303312.000 | 69894016.000 | +590704.000 |
| `/sched/goroutines:goroutines` | 19.000 | 19.000 | +0.000 |

## SQLite pool snapshots

| Phase | Open | In use | Idle | Wait count | Wait time | Max-idle closed | Max-lifetime closed |
|---|---:|---:|---:|---:|---:|---:|---:|
| before | 1 | 0 | 1 | 0 | 0 µs | 0 | 0 |
| after | 2 | 0 | 2 | 0 | 0 µs | 3 | 0 |

## Interpretation limits

This is a bounded, in-process probe using `httptest` and a temporary local SQLite database. It validates real strict-handler and persistence code paths, but it does not model reverse-proxy behavior, network TLS, multi-process SQLite contention, production disks, or sustained traffic. Treat the values as diagnostic evidence and regression baselines, not capacity numbers.
