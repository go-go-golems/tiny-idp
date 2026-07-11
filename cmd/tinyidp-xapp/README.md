# tinyidp-xapp

This directory is the product source root for the self-contained tiny-idp,
xgoja Express, and actor-bound Durable Objects application tracked by
`TINYIDP-XAPP-001`.

The current checkpoint provides the generated-package seam and trusted assets;
the production `init`/`serve` lifecycle is intentionally still under
implementation.

## Reproduce generated files

```bash
pnpm --dir cmd/tinyidp-xapp/app/frontend install
pnpm --dir cmd/tinyidp-xapp/app/frontend run build
go generate ./cmd/tinyidp-xapp
```

## Validate the skeleton

```bash
go run ./cmd/tinyidp-xapp doctor --output table
go run ../go-go-goja/cmd/xgoja doctor -f cmd/tinyidp-xapp/xgoja.yaml
go test ./cmd/tinyidp-xapp/... -count=1
```

`xgoja.yaml` keeps `enableRawGateway: false`. Product routes call only
`fetchForActor`/`rpcForActor`; the future custom host injects the persistent
binding key, actor resolver, manager, identity provider, app sessions, and
readiness/shutdown lifecycle.
