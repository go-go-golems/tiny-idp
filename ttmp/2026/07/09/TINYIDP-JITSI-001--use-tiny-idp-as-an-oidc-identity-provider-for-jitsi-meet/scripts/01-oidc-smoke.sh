#!/usr/bin/env bash
# 01-oidc-smoke.sh
#
# Assumption under test: tiny-idp exposes a standards-shaped OIDC surface
# (discovery + JWKS + authorization-code flow + userinfo) that an OIDC relying
# party (e.g. a Jitsi keycloak-style adapter) can consume unmodified.
#
# What it does, end to end, against a freshly started tiny-idp (mock engine):
#   1. GET /.well-known/openid-configuration      -> discovery metadata
#   2. GET /jwks                                   -> RS256 public keys
#   3. GET /authorize (renders login form)  + POST -> authorization code
#   4. POST /token (authorization_code grant)      -> id_token + access_token
#   5. Decode the id_token header + payload (the claims an adapter maps to Jitsi)
#   6. GET /userinfo with the access token         -> the same user claims
#
# Requires: go, curl, jq, python3 (for base64url JWT decode). Loopback only.
#
# Usage:
#   ./01-oidc-smoke.sh                       # ephemeral synthetic user, any password
#   USERS_FILE=/abs/personal-inbox-users.yaml LOGIN=alice PASSWORD=alice-password ./01-oidc-smoke.sh
set -euo pipefail

# ---- config knobs -----------------------------------------------------------
PORT="${PORT:-15556}"
ISSUER="${ISSUER:-http://127.0.0.1:${PORT}}"
CLIENT_ID="${CLIENT_ID:-dev-client}"
REDIRECT_URI="${REDIRECT_URI:-http://127.0.0.1:3000/callback}"
SCOPE="${SCOPE:-openid profile email}"
LOGIN="${LOGIN:-alice}"
PASSWORD="${PASSWORD:-whatever}"     # ignored for ephemeral users; enforced for seeded ones
USERS_FILE="${USERS_FILE:-}"
# tiny-idp repo root is six levels up from scripts/:
#   scripts/ -> TICKET -> 09 -> 07 -> 2026 -> ttmp -> <tiny-idp root>
REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)}"

COOKIES="$(mktemp)"
trap 'rm -f "$COOKIES"' EXIT

jwt_decode() { # $1 = compact JWT, prints "HEADER<newline>PAYLOAD" pretty JSON
  python3 - "$1" <<'PY'
import sys, json, base64
def d(seg):
    seg += "=" * (-len(seg) % 4)
    return json.loads(base64.urlsafe_b64decode(seg))
h, p, _ = sys.argv[1].split(".")
print("HEADER:",  json.dumps(d(h), indent=2))
print("PAYLOAD:", json.dumps(d(p), indent=2))
PY
}

# ---- 0. start tiny-idp ------------------------------------------------------
echo "== starting tiny-idp on ${ISSUER} (mock engine) =="
ARGS=(serve --issuer "$ISSUER" --addr "127.0.0.1:${PORT}"
      --client-id "$CLIENT_ID" --redirect-uris "$REDIRECT_URI")
[[ -n "$USERS_FILE" ]] && ARGS+=(--users-file "$USERS_FILE")

# TINYIDP_BIN lets you skip `go run` (which recompiles the whole go.work) for a
# fast prebuilt binary: `go build -o /tmp/tinyidp ./cmd/tinyidp`.
if [[ -n "${TINYIDP_BIN:-}" ]]; then
  "$TINYIDP_BIN" "${ARGS[@]}" &
else
  ( cd "$REPO_ROOT" && go run ./cmd/tinyidp "${ARGS[@]}" ) &
fi
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null; rm -f "$COOKIES"' EXIT

# wait for readiness
for _ in $(seq 1 60); do
  curl -fsS "${ISSUER}/healthz" >/dev/null 2>&1 && break
  sleep 0.5
done

# ---- 1. discovery -----------------------------------------------------------
echo; echo "== 1. discovery =="
curl -fsS "${ISSUER}/.well-known/openid-configuration" | jq .

# ---- 2. jwks ----------------------------------------------------------------
echo; echo "== 2. jwks (RS256 public keys) =="
curl -fsS "${ISSUER}/jwks" | jq .

# ---- 3. authorize: GET form, then POST login -> code ------------------------
echo; echo "== 3. authorization-code flow =="
AUTHZ="response_type=code&client_id=${CLIENT_ID}&redirect_uri=$(python3 -c 'import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))' "$REDIRECT_URI")&scope=$(python3 -c 'import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))' "$SCOPE")&state=st-123&nonce=nonce-xyz"

# GET renders the login form (and may set a session cookie); we don't need to parse it.
curl -fsS -c "$COOKIES" "${ISSUER}/authorize?${AUTHZ}" >/dev/null

# POST submits login + password + echoes the hidden authorize params.
LOCATION="$(curl -sS -b "$COOKIES" -c "$COOKIES" -o /dev/null -D - \
  --data-urlencode "login=${LOGIN}" \
  --data-urlencode "password=${PASSWORD}" \
  --data-urlencode "response_type=code" \
  --data-urlencode "client_id=${CLIENT_ID}" \
  --data-urlencode "redirect_uri=${REDIRECT_URI}" \
  --data-urlencode "scope=${SCOPE}" \
  --data-urlencode "state=st-123" \
  --data-urlencode "nonce=nonce-xyz" \
  "${ISSUER}/authorize" | tr -d '\r' | awk 'tolower($1)=="location:"{print $2}')"

echo "redirect Location: ${LOCATION}"
CODE="$(printf '%s' "$LOCATION" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')"
if [[ -z "$CODE" ]]; then echo "!! no authorization code (check LOGIN/PASSWORD/redirect)"; exit 1; fi
echo "authorization code: ${CODE}"

# ---- 4. token ---------------------------------------------------------------
echo; echo "== 4. token exchange =="
TOKENS="$(curl -fsS -X POST "${ISSUER}/token" \
  --data-urlencode "grant_type=authorization_code" \
  --data-urlencode "code=${CODE}" \
  --data-urlencode "redirect_uri=${REDIRECT_URI}" \
  --data-urlencode "client_id=${CLIENT_ID}")"
echo "$TOKENS" | jq '{token_type, expires_in, scope, has_id_token:(.id_token!=null), has_access_token:(.access_token!=null), has_refresh_token:(.refresh_token!=null)}'

ID_TOKEN="$(echo "$TOKENS" | jq -r .id_token)"
ACCESS_TOKEN="$(echo "$TOKENS" | jq -r .access_token)"

# ---- 5. decode id_token -----------------------------------------------------
echo; echo "== 5. id_token (the claims a Jitsi adapter would map) =="
jwt_decode "$ID_TOKEN"

# ---- 6. userinfo ------------------------------------------------------------
echo; echo "== 6. userinfo =="
curl -fsS "${ISSUER}/userinfo" -H "Authorization: Bearer ${ACCESS_TOKEN}" | jq .

echo; echo "== OK: tiny-idp served a complete OIDC authorization-code flow =="
