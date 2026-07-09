#!/usr/bin/env python3
"""
02-oidc-to-jitsi-jwt.py

Assumption under test: the ONLY thing standing between tiny-idp's OIDC claims and
a working Jitsi login is a small, mechanical claim-reshaping + HS256 re-signing
step -- exactly what jitsi-contrib/jitsi-oidc-adapter does.

This script reproduces that translation in ~40 lines, with no third-party deps,
so an intern can see the transform concretely:

  tiny-idp userinfo/id_token claims  --(map)-->  Jitsi-shaped JWT (HS256)

It does NOT talk to a network; feed it the userinfo JSON that 01-oidc-smoke.sh
prints (or pipe it in). It prints the minted Jitsi JWT and its decoded payload,
which is what Prosody's mod_auth_token validates.

Usage:
  # use the built-in Alice sample:
  ./02-oidc-to-jitsi-jwt.py --room standup --app-id jitsi --app-secret myappsecret

  # feed real userinfo from the smoke test:
  curl -s .../userinfo -H "Authorization: Bearer $AT" \
     | ./02-oidc-to-jitsi-jwt.py --room standup --app-secret myappsecret
"""
import argparse, base64, hashlib, hmac, json, sys, time

def b64url(b: bytes) -> str:
    return base64.urlsafe_b64encode(b).rstrip(b"=").decode()

def b64url_json(obj) -> str:
    return b64url(json.dumps(obj, separators=(",", ":"), sort_keys=True).encode())

def sign_hs256(header: dict, payload: dict, secret: str) -> str:
    signing_input = f"{b64url_json(header)}.{b64url_json(payload)}".encode()
    sig = hmac.new(secret.encode(), signing_input, hashlib.sha256).digest()
    return f"{signing_input.decode()}.{b64url(sig)}"

# --- the actual OIDC -> Jitsi mapping (mirrors adapter context.ts, source 14) --
def oidc_to_jitsi(claims: dict, *, app_id: str, room: str, sub: str,
                  ttl: int, moderator: bool, now: int) -> dict:
    # context.user: EVERY value must be a non-null string (lib-jitsi-meet throws
    # otherwise -- see sources/web/01-lib-jitsi-meet-tokens.md).
    user = {
        "id":     str(claims.get("sub", "")),
        "name":   str(claims.get("name") or claims.get("preferred_username") or ""),
        "email":  str(claims.get("email", "")),
        "avatar": str(claims.get("picture", "")),
    }
    if moderator:
        user["moderator"] = "true"   # honored only with enableUserRolesBasedOnToken
    return {
        "iss":  app_id,              # must match Prosody asap_accepted_issuers / app_id
        "aud":  app_id,              # must match asap_accepted_audiences
        "sub":  sub,                 # tenant or base domain (or "*")
        "room": room,                # the room being entered (or "*")
        "iat":  now,
        "nbf":  now,
        "exp":  now + ttl,
        "context": {"user": user},
    }

SAMPLE = {  # the id_token/userinfo tiny-idp emits for seeded Alice
    "sub": "user-alice-fixed", "name": "Alice Inbox",
    "preferred_username": "alice", "email": "alice@example.test",
    "email_verified": True, "groups": ["inbox-users"], "roles": ["writer"],
    "tenant": "personal", "locale": "en-US",
}

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--room", default="*")
    ap.add_argument("--sub", default="*", help="tenant/domain scope; * = all")
    ap.add_argument("--app-id", default="jitsi")
    ap.add_argument("--app-secret", default="myappsecret")
    ap.add_argument("--ttl", type=int, default=10800)
    ap.add_argument("--moderator", action="store_true")
    ap.add_argument("--now", type=int, default=None, help="fixed epoch (for reproducible output)")
    args = ap.parse_args()

    raw = sys.stdin.read().strip() if not sys.stdin.isatty() else ""
    claims = json.loads(raw) if raw else SAMPLE
    now = args.now if args.now is not None else int(time.time())

    payload = oidc_to_jitsi(claims, app_id=args.app_id, room=args.room,
                            sub=args.sub, ttl=args.ttl, moderator=args.moderator, now=now)
    header = {"alg": "HS256", "typ": "JWT"}
    jwt = sign_hs256(header, payload, args.app_secret)

    print("== INPUT OIDC claims (from tiny-idp) ==")
    print(json.dumps(claims, indent=2, sort_keys=True))
    print("\n== OUTPUT Jitsi JWT (HS256, what Prosody validates) ==")
    print(jwt)
    print("\n== decoded Jitsi payload ==")
    print(json.dumps(payload, indent=2, sort_keys=True))
    print("\nJitsi joins via:  https://<jitsi-host>/%s?jwt=%s" %
          (args.room if args.room != "*" else "<room>", "<jwt>"))

if __name__ == "__main__":
    main()
