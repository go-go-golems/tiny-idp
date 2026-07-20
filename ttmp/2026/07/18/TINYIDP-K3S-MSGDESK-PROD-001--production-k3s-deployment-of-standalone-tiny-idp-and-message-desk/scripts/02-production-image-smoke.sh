#!/bin/sh
# Build and exercise the two production images without using any production
# secret. This is intentionally a local Docker check; Phase 5 supplies the
# Kubernetes volume, secret, and proxy topology.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../../../../../.." && pwd)
cd "$repo_root"

idp_image="tinyidp:phase4-smoke"
message_desk_image="tinyidp-message-desk:phase4-smoke"
idp_container="tinyidp-phase4-smoke-$$"
message_desk_container="tinyidp-message-desk-phase4-smoke-$$"

cleanup() {
  docker logs "$idp_container" 2>/dev/null || true
  docker logs "$message_desk_container" 2>/dev/null || true
  docker rm -f "$idp_container" "$message_desk_container" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

if lsof -n -iTCP:18080 -sTCP:LISTEN >/dev/null 2>&1 || lsof -n -iTCP:18443 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "port 18080 or 18443 is already in use; image smoke requires both loopback ports" >&2
  exit 1
fi

make TINYIDP_IMAGE="$idp_image" MESSAGE_DESK_IMAGE="$message_desk_image" IMAGE_VERSION=phase4-smoke image-build

for image in "$idp_image" "$message_desk_image"; do
  test "$(docker image inspect "$image" --format '{{.Config.User}}')" = "65532:65532"
  test "$(docker image inspect "$image" --format '{{.Config.StopSignal}}')" = "SIGTERM"
  for label in org.opencontainers.image.source org.opencontainers.image.revision org.opencontainers.image.version; do
    test -n "$(docker image inspect "$image" --format "{{index .Config.Labels \"$label\"}}")"
  done
done

docker run --rm --read-only --tmpfs /tmp:rw,noexec,nosuid,size=16m "$idp_image" --help >/dev/null
docker run --rm --read-only --tmpfs /tmp:rw,noexec,nosuid,size=16m "$message_desk_image" --help >/dev/null

docker run --rm --read-only \
  --tmpfs /var/lib/tinyidp:uid=65532,gid=65532,mode=0750 \
  --tmpfs /var/log/tinyidp:uid=65532,gid=65532,mode=0750 \
  --entrypoint /bin/sh "$idp_image" -ec '
    test "$(id -u)" = 65532
    test -w /var/lib/tinyidp
    test -w /var/log/tinyidp
    test ! -e /etc/tinyidp/signup/open-signup.js
    test -z "$(find /run/tinyidp-secrets -type f -print -quit)"
  '

docker run --rm --read-only \
  --tmpfs /var/lib/tinyidp-message-desk:uid=65532,gid=65532,mode=0750 \
  --entrypoint /bin/sh "$message_desk_image" -ec '
    test "$(id -u)" = 65532
    test -w /var/lib/tinyidp-message-desk
    test -z "$(find /run/tinyidp-secrets -type f -print -quit)"
  '

docker run -d --name "$idp_container" --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --tmpfs /var/lib/tinyidp:uid=65532,gid=65532,mode=0750 \
  --tmpfs /var/log/tinyidp:uid=65532,gid=65532,mode=0750 \
  --tmpfs /run/tinyidp-secrets:uid=65532,gid=65532,mode=0700 \
  -v "$repo_root/pkg/idpsignup/open_signup.js:/etc/tinyidp/signup/open-signup.js:ro" \
  -p 127.0.0.1:18443:8443 --entrypoint /bin/sh "$idp_image" -ec '
    umask 077
    head -c 48 /dev/urandom > /run/tinyidp-secrets/token.key
    openssl req -x509 -newkey rsa:2048 -sha256 -nodes \
      -keyout /run/tinyidp-secrets/tls.key -out /run/tinyidp-secrets/tls.crt -days 1 \
      -subj "/CN=idp.example.test" >/dev/null 2>&1
    exec tinyidp serve-production \
      --addr :8443 --listener-mode direct-tls --issuer https://idp.example.test \
      --message-desk-origin https://message-desk.example.test \
      --signup-program-file /etc/tinyidp/signup/open-signup.js \
      --db /var/lib/tinyidp/tinyidp.sqlite --audit-path /var/log/tinyidp/audit.jsonl \
      --token-secret-file /run/tinyidp-secrets/token.key \
      --tls-cert /run/tinyidp-secrets/tls.crt --tls-key /run/tinyidp-secrets/tls.key
  ' >/dev/null

idp_port=18443
attempt=0
until curl --noproxy '*' --fail --silent --show-error --insecure \
  --resolve "idp.example.test:$idp_port:127.0.0.1" \
  "https://idp.example.test:$idp_port/readyz" >/dev/null; do
  attempt=$((attempt + 1))
  test "$attempt" -lt 30
  sleep 1
done

docker run -d --name "$message_desk_container" --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --tmpfs /var/lib/tinyidp-message-desk:uid=65532,gid=65532,mode=0750 \
  -p 127.0.0.1:18080:8080 --entrypoint /bin/sh "$message_desk_image" -ec '
    tinyidp-message-desk init --state-root /var/lib/tinyidp-message-desk --public-base-url http://localhost:18080
    exec tinyidp-message-desk serve --state-root /var/lib/tinyidp-message-desk --addr :8080 --listener-mode development-http
  ' >/dev/null

attempt=0
until curl --noproxy '*' --fail --silent --show-error http://127.0.0.1:18080/readyz >/dev/null; do
  attempt=$((attempt + 1))
  test "$attempt" -lt 30
  sleep 1
done

echo "production image smoke passed"
