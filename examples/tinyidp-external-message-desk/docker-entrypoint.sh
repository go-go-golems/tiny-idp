#!/bin/sh
set -eu

# Docker creates named volumes as root. Fix the durable-state boundary once,
# then run both services as the dedicated unprivileged identity.
if [ "$(id -u)" = "0" ]; then
  mkdir -p /state
  if [ -d /run/secrets ]; then
    mkdir -p /state/.secrets
    for secret in /run/secrets/*; do
      [ -f "$secret" ] || continue
      cp "$secret" "/state/.secrets/$(basename "$secret")"
    done
    chmod 0700 /state/.secrets
    find /state/.secrets -type f -exec chmod 0400 {} \;
  fi
  chown -R tinyidp:tinyidp /state
  exec setpriv --reuid=tinyidp --regid=tinyidp --init-groups "$@"
fi

exec "$@"
