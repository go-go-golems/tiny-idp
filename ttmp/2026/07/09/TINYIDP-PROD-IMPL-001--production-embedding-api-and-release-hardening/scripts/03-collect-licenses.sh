#!/usr/bin/env bash
set -euo pipefail

out="${1:?usage: 03-collect-licenses.sh OUTPUT_DIRECTORY}"
mkdir -p "$out"

go list -m -f '{{printf "%s\t%s\t%s" .Path .Version .Dir}}' all | \
while IFS=$'\t' read -r module version directory; do
  test -n "$directory" || continue
  safe="$(printf '%s@%s' "$module" "$version" | tr '/:' '__')"
  found=0
  for pattern in LICENSE LICENSE.* COPYING COPYING.* NOTICE NOTICE.*; do
    for source in "$directory"/$pattern; do
      test -f "$source" || continue
      mkdir -p "$out/$safe"
      cp "$source" "$out/$safe/$(basename "$source")"
      found=1
    done
  done
  if test "$found" -eq 0; then
    printf '%s\t%s\tNO_TOP_LEVEL_LICENSE_FILE_FOUND\n' "$module" "$version" >> "$out/MISSING.tsv"
  fi
done
