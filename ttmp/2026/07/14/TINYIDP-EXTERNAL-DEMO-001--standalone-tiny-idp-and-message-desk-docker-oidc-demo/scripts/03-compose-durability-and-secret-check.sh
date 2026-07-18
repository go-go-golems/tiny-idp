#!/usr/bin/env sh
set -eu

# Validate the properties that are observable from the running development
# topology. This script deliberately knows the public fixture value only to
# prove it was not copied into Compose configuration or emitted to logs. It is
# not a production-secret scanner and must never be supplied real credentials.
root=$(CDPATH= cd -- "$(dirname -- "$0")/../../../../../../examples/tinyidp-external-message-desk" && pwd)
fixture_password='dev-only-not-a-secret-12345'
cd "$root"

wait_ready() {
  attempts=0
  until curl --fail --silent --show-error http://localhost:8081/readyz >/dev/null && \
    curl --fail --silent --show-error http://localhost:8080/readyz >/dev/null; do
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 30 ]; then
      echo "Compose services did not become ready after restart" >&2
      return 1
    fi
    sleep 1
  done
}

wait_ready

# `docker compose exec` itself enters as Docker's diagnostic user, commonly
# root. Inspect PID 1 instead: it is the actual server process after the image
# entrypoint's ownership initialization and `setpriv` transition.
idp_uid=$(docker compose exec -T idp sh -ec 'awk "/^Uid:/ {print \$2}" /proc/1/status')
desk_uid=$(docker compose exec -T message-desk sh -ec 'awk "/^Uid:/ {print \$2}" /proc/1/status')
if [ "$idp_uid" = 0 ] || [ "$desk_uid" = 0 ]; then
  echo "a service is running as root: idp=$idp_uid message-desk=$desk_uid" >&2
  exit 1
fi

if docker compose config | grep -F -- "$fixture_password" >/dev/null; then
  echo "the fixture password appeared in rendered Compose configuration" >&2
  exit 1
fi
if docker compose logs --no-color | grep -F -- "$fixture_password" >/dev/null; then
  echo "the fixture password appeared in service logs" >&2
  exit 1
fi

idp_key_before=$(docker compose exec -T idp sha256sum /state/token.key | awk '{print $1}')
desk_manifest_before=$(docker compose exec -T message-desk sha256sum /state/state.json | awk '{print $1}')
feed_before=$(mktemp)
feed_after=$(mktemp)
trap 'rm -f "$feed_before" "$feed_after"' EXIT
curl --fail --silent --show-error http://localhost:8080/api/messages >"$feed_before"

# Restart the identity authority first and wait for it before restarting its
# depending RP. This verifies persisted signing state and avoids manufacturing
# a startup race that the Compose initial dependency condition already avoids.
docker compose restart idp
attempts=0
until curl --fail --silent --show-error http://localhost:8081/readyz >/dev/null; do
  attempts=$((attempts + 1))
  if [ "$attempts" -ge 30 ]; then
    echo "tiny-idp did not recover after restart" >&2
    exit 1
  fi
  sleep 1
done
docker compose restart message-desk
wait_ready

idp_key_after=$(docker compose exec -T idp sha256sum /state/token.key | awk '{print $1}')
desk_manifest_after=$(docker compose exec -T message-desk sha256sum /state/state.json | awk '{print $1}')
curl --fail --silent --show-error http://localhost:8080/api/messages >"$feed_after"

test "$idp_key_before" = "$idp_key_after"
test "$desk_manifest_before" = "$desk_manifest_after"
cmp -s "$feed_before" "$feed_after"

printf '%s\n' 'durability and development-fixture exposure checks passed'
