# Embedded provider example

This directory shows the shape of a host Go application embedding `pkg/embeddedidp`.

The example is marked `//go:build ignore` because it uses internal packages while the production API is still being phased in. It is intentionally small: create a store, configure a client and user, create a signing key, build `embeddedidp.New`, and mount the provider's handler.

Run manually from the repository root while developing the API:

```bash
go run ./examples/embedded/main.go
```
