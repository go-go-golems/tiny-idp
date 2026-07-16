#!/usr/bin/env sh
# Run a strict HTTPS RFC 7662 smoke against a non-production token.
# Usage: 01-introspection-smoke.sh ISSUER RESOURCE_CLIENT RESOURCE_SECRET ACCESS_TOKEN AUDIENCE
set -eu

issuer=${1:?issuer URL is required}
resource_client=${2:?resource client ID is required}
resource_secret=${3:?resource client secret is required}
access_token=${4:?access token is required}
audience=${5:?expected audience is required}

case "$issuer" in
  https://*) ;;
  *) echo "issuer must use https" >&2; exit 64 ;;
esac

discovery=$(curl --fail --silent --show-error --proto '=https' --tlsv1.2 "$issuer/.well-known/openid-configuration")
endpoint=$(printf '%s' "$discovery" | jq -er '.introspection_endpoint')
methods=$(printf '%s' "$discovery" | jq -er '.introspection_endpoint_auth_methods_supported | index("client_secret_basic")')
[ "$methods" = "0" ] || [ "$methods" = "1" ] || [ "$methods" = "2" ] || {
  echo "discovery does not advertise client_secret_basic" >&2
  exit 65
}

response=$(curl --fail --silent --show-error --proto '=https' --tlsv1.2 \
  --user "$resource_client:$resource_secret" \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "token=$access_token" \
  "$endpoint")

printf '%s' "$response" | jq -e --arg audience "$audience" \
  '.active == true and .token_type == "Bearer" and (.aud | index($audience) != null)' >/dev/null
echo "introspection smoke passed for $endpoint"
