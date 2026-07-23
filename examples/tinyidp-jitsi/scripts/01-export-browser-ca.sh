#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
docker compose --project-directory "$example_dir" \
  -f "$example_dir/compose.yaml" \
  cp ca-export:/trust/caddy-local-root.crt "$example_dir/runtime/caddy-local-root.crt"
chmod 0644 "$example_dir/runtime/caddy-local-root.crt"
printf 'Exported public local CA to %s\n' "$example_dir/runtime/caddy-local-root.crt"
