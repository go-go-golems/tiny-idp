#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$example_dir/browser-tests"
exec pnpm test
