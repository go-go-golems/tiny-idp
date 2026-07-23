#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file="$example_dir/compose.yaml"
ca_file="$example_dir/runtime/caddy-local-root.crt"

if [ ! -s "$ca_file" ]; then
  "$example_dir/scripts/01-export-browser-ca.sh"
fi

docker compose --project-directory "$example_dir" -f "$compose_file" config --quiet

services="idp proxy meet-web prosody jicofo jvb"
for service in $services; do
  container_id=$(docker compose --project-directory "$example_dir" -f "$compose_file" ps -q "$service")
  if [ -z "$container_id" ] || [ "$(docker inspect -f '{{.State.Running}}' "$container_id")" != "true" ]; then
    printf 'Service %s is not running\n' "$service" >&2
    exit 1
  fi
done

curl --cacert "$ca_file" -fsS https://idp.localhost:8443/readyz |
  grep -q '"name":"plugin.jitsi","ready":true'
curl --cacert "$ca_file" -fsS https://meet.localhost:8443/ |
  grep -q "config.tokenAuthUrl = 'https://idp.localhost:8443/integrations/jitsi/start?room={room}'"

metrics=$(docker compose --project-directory "$example_dir" -f "$compose_file" \
  exec -T idp curl -fsS http://127.0.0.1:9090/metrics)
printf '%s\n' "$metrics" | grep -q 'tinyidp_plugin_requests_total'
printf '%s\n' "$metrics" | grep -q 'tinyidp_jitsi_tokens_issued_total'

prosody_config=$(docker compose --project-directory "$example_dir" -f "$compose_file" \
  exec -T prosody sh -ec 'grep -R -E "authentication = .token.|allow_empty_token" /config/conf.d /config/prosody.cfg.lua')
printf '%s\n' "$prosody_config" | grep -q 'authentication = "token"'

printf 'OK: TinyIDP, Jitsi, Prosody token mode, HTTPS, readiness, and metrics are available\n'
