#!/bin/sh
set -eu

# Docker creates named volumes as root. Fix the durable-state boundary once,
# then run both services as the dedicated unprivileged identity.
if [ "$(id -u)" = "0" ]; then
  mkdir -p /state
  chown -R tinyidp:tinyidp /state
  exec setpriv --reuid=tinyidp --regid=tinyidp --init-groups "$@"
fi

exec "$@"
