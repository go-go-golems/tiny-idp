---
Title: tiny-idp runtime probe summary
Ticket: TINYIDP-PROD-IMPL-001
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
LastUpdated: 2026-07-10T00:46:46Z
WhatFor: "Providing a bounded runtime regression baseline for release candidate 2930981."
WhenToUse: "Use when changing authentication, Fosite, SQLite, runtime limits, or observability."
---

# tiny-idp runtime probe summary

Source: `ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/various/phase5-runtime-load.ndjson`

Candidate source: `29309814f1fcdad3a5134674fc27a8938cb39c6a`.

Requests observed: **5125**. Audit events emitted: **129**.

## HTTP observations

| Operation | Count | Statuses | p50 | p95 | p99 | Mean bytes | Errors |
|---|---:|---|---:|---:|---:|---:|---:|
| `GET /.well-known/openid-configuration (load_.well-known_openid-configuration)` | 1250 | 200×1250 | 105 µs | 236 µs | 848 µs | 790 | 0 |
| `GET /authorize (authorize_get)` | 25 | 200×25 | 505 µs | 1.72 ms | 2.06 ms | 1040 | 0 |
| `GET /jwks (load_jwks)` | 1250 | 200×1250 | 2.51 ms | 10.46 ms | 15.75 ms | 433 | 0 |
| `GET /readyz (load_readyz)` | 1250 | 200×1250 | 12.76 ms | 26.12 ms | 34.67 ms | 834 | 0 |
| `GET /userinfo (load_userinfo)` | 1250 | 200×1250 | 5.68 ms | 15.73 ms | 22.53 ms | 143 | 0 |
| `GET /userinfo (userinfo)` | 25 | 200×25 | 491 µs | 695 µs | 801 µs | 143 | 0 |
| `POST /authorize (authorize_post)` | 25 | 303×25 | 492.43 ms | 546.74 ms | 631.01 ms | 0 | 0 |
| `POST /token (token_exchange)` | 25 | 200×25 | 8.15 ms | 10.90 ms | 20.57 ms | 1227 | 0 |
| `POST /token (token_refresh)` | 25 | 200×25 | 6.30 ms | 11.39 ms | 15.80 ms | 1227 | 0 |

## Go runtime deltas

| Metric | Before | After | Delta |
|---|---:|---:|---:|
| `/cpu/classes/gc/total:cpu-seconds` | 0.007 | 0.285 | +0.279 |
| `/gc/cycles/total:gc-cycles` | 4.000 | 59.000 | +55.000 |
| `/gc/heap/allocs:bytes` | 204717896.000 | 2047019312.000 | +1842301416.000 |
| `/gc/heap/live:bytes` | 69245616.000 | 137099696.000 | +67854080.000 |
| `/gc/heap/objects:objects` | 3924.000 | 10301.000 | +6377.000 |
| `/memory/classes/heap/objects:bytes` | 69311920.000 | 137564720.000 | +68252800.000 |
| `/sched/goroutines:goroutines` | 19.000 | 19.000 | +0.000 |

## SQLite pool snapshots

| Phase | Open | In use | Idle | Wait count | Wait time | Max-idle closed | Max-lifetime closed |
|---|---:|---:|---:|---:|---:|---:|---:|
| before | 1 | 0 | 1 | 0 | 0 µs | 0 | 0 |
| after | 1 | 0 | 1 | 8827 | 28413.84 ms | 0 | 0 |

## Password-work admission

| Capacity | Completed | Saturations | Rejected | Total wait | Total Argon2 duration |
|---:|---:|---:|---:|---:|---:|
| 2 | 25 | 22 | 0 | 8001.04 ms | 3459.06 ms |

## Interpretation limits

This is a bounded, in-process probe using `httptest` and a temporary local SQLite database. It validates real strict-handler and persistence code paths, but it does not model reverse-proxy behavior, network TLS, multi-process SQLite contention, production disks, or sustained traffic. Treat the values as diagnostic evidence and regression baselines, not capacity numbers.
