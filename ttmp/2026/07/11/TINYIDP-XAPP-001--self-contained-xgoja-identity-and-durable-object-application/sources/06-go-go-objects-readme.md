# go-go-objects

`go-go-objects` is an experimental Durable Objects runtime built on top of [`goja`](https://github.com/dop251/goja) and the `go-go-goja` runtime owner/engine packages.

The runtime is intentionally small: each object identity maps to one live JavaScript actor, that actor owns one `goja.Runtime`, and each actor has private SQLite-backed durable storage.

## Current capabilities

- Stable object identity: namespace + name + hash
- Bundle-derived namespaces from `exports.objects` with `CamelCase` to `CAMEL_CASE` conversion
- Optional manifest aliases for custom namespace mappings
- CommonJS bundle loading with `exports.objects = { Counter }`
- Lazy actor startup through `Manager.Dispatch`
- One owned `goja` runtime per live actor
- Synchronous `state.storage` API backed by SQLite
- JSON-only RPC dispatch with returned Promise awaiting
- Plain-object fetch dispatch with returned Promise awaiting
- `/rpc/:namespace/:name/:method` and `/fetch/:namespace/:name/*` HTTP gateway
- Persistent alarms with a central SQLite due-alarm index
- Explicit alarm scheduler and idle evictor wrappers
- Idle actor eviction with durable state recovery

## Quick demo

Run the built-in counter demo server:

```bash
go run ./cmd/go-go-objects --addr 127.0.0.1:8787 --storage ./var/durable-objects
```

Increment a durable counter:

```bash
curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[1]'
```

Read through fetch dispatch:

```bash
curl http://127.0.0.1:8787/fetch/COUNTER/global/count
```

Stop and restart the server, then increment again. The count is recovered from SQLite.

To run your own bundle:

```bash
go run ./cmd/go-go-objects \
  --addr 127.0.0.1:8787 \
  --storage ./var/durable-objects \
  --bundle ./objects.js
```

Namespaces are derived from `exports.objects` keys with Cloudflare-style `CamelCase` to `CAMEL_CASE` conversion. For example, `exports.objects = { ChatRoom }` creates namespace `CHAT_ROOM`; `exports.objects = { Counter }` creates namespace `COUNTER`.

You can still provide an explicit JSON/YAML manifest when you need custom aliases:

```bash
go run ./cmd/go-go-objects \
  --addr 127.0.0.1:8787 \
  --storage ./var/durable-objects \
  --bundle ./objects.js \
  --manifest ./durableobjects.yaml
```

```yaml
objects:
  COUNTER: Counter
  CHAT_ROOM: ChatRoom
```

## JavaScript authoring model

The MVP authoring model is CommonJS:

```js
class Counter {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  async increment(by) {
    await Promise.resolve();
    const current = this.state.storage.get("count") || 0;
    const next = current + (by || 1);
    this.state.storage.put("count", next);
    return next;
  }

  async fetch(req) {
    if (req.path === "/count") {
      return { status: 200, body: String(this.state.storage.get("count") || 0) };
    }
    return { status: 404, body: "not found" };
  }
}

exports.objects = { Counter };
```

RPC methods, `fetch(req)`, and `alarm()` may return either plain values or Promises. The actor waits for returned Promises before returning to the HTTP gateway or xgoja caller. `state.storage` remains synchronous, and `state.storage.transaction(fn)` callbacks must remain synchronous; an async transaction callback is rejected so SQLite transaction lifetime stays bounded.

This is Promise-aware dispatch, not full Cloudflare input/output gate compatibility. While a Promise is pending, the object dispatch remains active and serialized; the runtime does not yet interleave another request into the same object during non-storage awaits.

## xgoja provider configuration

The xgoja provider can initialize a Durable Objects manager from filesystem paths:

```yaml
modules:
  - package: go-go-objects-durableobjects
    name: durableobjects
    config:
      storageRoot: ./var/durable-objects
      bundlePath: ./objects.js
```

For self-contained generated binaries, use embedded xgoja asset IDs instead:

```yaml
modules:
  - package: go-go-objects-durableobjects
    name: durableobjects
    config:
      storageRoot: ./var/durable-objects
      bundleAsset: durableobjects/objects.js
```

`manifestPath` and `manifestAsset` are optional. If omitted, namespaces are derived from `exports.objects`.

With xgoja/v2 and the shared mountable HTTP handler ABI, the recommended generated-server path is to let JavaScript compose the server:

```js
const express = require("express");
const durableobjects = require("durableobjects");

__package__({ name: "durableobjects" });
__verb__("site", { name: "site", output: "text" });
function site() {
  const app = express.app();
  const gateway = durableobjects.gateway();
  app.mount("/rpc", gateway);
  app.mount("/fetch", gateway);
}
module.exports = { site };
```

Then use the HTTP provider's `serve` command set in `xgoja.yaml`. A complete v2 example lives at:

```text
examples/counter/xgoja-buildspec.yaml
```

The provider also exposes a direct command provider mounted as `durableobjects serve` for generated apps that do not need Express/JS composition:

```yaml
commands:
  - id: durableobjects-serve
    type: provider.command-set
    provider: durableobjects
    name: serve
    mount: durableobjects
    config:
      storageRoot: ./var/durable-objects
      bundleAsset: counter-bundle
      bundleAssetPath: objects.js
```

The generated command runs a Durable Objects HTTP gateway directly:

```bash
./generated-app durableobjects serve --addr 127.0.0.1:8787
```

For existing Go HTTP servers, use xgoja `type: template` artifacts with `examples/templates/durableobjects_http_runtime.go.tmpl`. The generated package exposes `NewRuntime(ctx)`, `Runtime.Mount(mux)`, `Runtime.Handler()`, and `Runtime.Close(ctx)` so a host application can mount `/rpc/` and `/fetch/` on its own `http.Server`.

## Storage and operations

SQLite files are stored below the configured storage root:

```text
var/durable-objects/
  alarms.sqlite              # central due-alarm index
  objects/<prefix>/<hash>.sqlite
```

Each object database stores user key/value data, object metadata, and the local alarm record. The central alarm index is reconciled from object-local alarm records before due alarms are dispatched, so a missing or stale `alarms.sqlite` row can be repaired after a crash.

Back up the full storage root as one unit. The current schema is initialized with `CREATE TABLE IF NOT EXISTS`; future incompatible changes should add explicit schema-version migrations before release.

## Security and resource limits

- Object namespaces and names are validated as safe single path segments before storage paths are derived.
- Gateway request bodies are capped by `GatewayOptions.MaxRequestBytes` and default to 64 MiB.
- Production callers should set `DevErrors: false`; detailed errors are intended for local development.
- JavaScript bundles are trusted code. CPU timeout limits bound synchronous CPU work and returned Promise settlement time, but this is not a sandbox for hostile code.
- Storage quotas are not enforced yet; embedders should isolate storage roots and monitor disk usage.

## Examples

A runnable counter bundle lives in:

```text
examples/counter/objects.js
```

Run it with the CLI:

```bash
go run ./cmd/go-go-objects --bundle ./examples/counter/objects.js
```

An xgoja embedded-asset configuration sketch lives in:

```text
examples/counter/xgoja-runtime.yaml
```

## Development

```bash
go test ./... -count=1
```

For a release candidate, also run focused concurrency/async tests and `docmgr doctor --ticket GOJA-DO-001 --stale-after 30`. See `docs/release-notes.md` for the full release checklist and known limitations.

The design and implementation diary live in the docmgr ticket:

```text
ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja
```
