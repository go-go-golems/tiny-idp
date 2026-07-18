#!/usr/bin/env python3
"""Drive the initialized tinyidp-xapp through a real Chromium browser.

The server is intentionally external to this script. Start it with the real
``serve-initialized`` command in tmux so startup, TLS, logs, shutdown, and port
ownership remain observable independently from browser assertions.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

from playwright.sync_api import BrowserContext, Page, sync_playwright


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def read_secret(path: Path) -> str:
    value = path.read_text(encoding="utf-8").strip()
    require(bool(value), f"empty password file: {path}")
    return value


def sign_in(page: Page, base_url: str, login: str, password: str) -> None:
    page.goto(base_url + "/", wait_until="domcontentloaded")
    page.wait_for_selector('input[name="login"]')
    page.fill('input[name="login"]', login)
    page.fill('input[name="password"]', password)
    page.get_by_role("button", name="Approve").click()
    page.wait_for_url(base_url + "/")
    page.wait_for_selector("#app:not(.d-none)")


def session(page: Page) -> dict[str, Any]:
    response = page.request.get(page.url.rstrip("/") + "/auth/session")
    require(response.status == 200, f"session status={response.status}")
    return response.json()


def save_document(page: Page, value: dict[str, Any]) -> None:
    page.locator("#document").fill(json.dumps(value))
    page.get_by_role("button", name="Save").click()
    page.wait_for_function(
        "document.querySelector('#status').textContent === 'Saved'"
    )


def loaded_document(page: Page) -> dict[str, Any]:
    with page.expect_response(
        lambda response: response.url.endswith("/api/object")
        and response.request.method == "GET"
    ) as response_info:
        page.get_by_role("button", name="Reload").click()
    response = response_info.value
    require(response.status == 200, f"object reload status={response.status}")
    return response.json()


def cookie_summary(context: BrowserContext) -> list[dict[str, Any]]:
    keys = ("name", "path", "secure", "httpOnly", "sameSite")
    return [{key: cookie[key] for key in keys} for cookie in context.cookies()]


def run(args: argparse.Namespace) -> dict[str, Any]:
    base_url = args.base_url.rstrip("/")
    alice_value = {"owner": "alice", "sequence": 1}
    bob_value = {"owner": "bob", "sequence": 2}

    with sync_playwright() as playwright:
        # Playwright 1.50 is installed without its version-matched cached
        # Chromium artifacts on this workstation. Use the installed system
        # Chrome explicitly so the checkpoint does not perform a network
        # download or mutate the Playwright cache.
        browser = playwright.chromium.launch(
            executable_path="/usr/bin/google-chrome",
            headless=not args.headed,
        )
        alice_context = browser.new_context(ignore_https_errors=True)
        bob_context = browser.new_context(ignore_https_errors=True)
        alice = alice_context.new_page()
        bob = bob_context.new_page()

        sign_in(alice, base_url, "alice", read_secret(args.alice_password_file))
        alice_session = session(alice)
        alice_cookies = cookie_summary(alice_context)
        cookie_by_name = {cookie["name"]: cookie for cookie in alice_cookies}
        expected_cookie_names = {"xapp_session", "xapp_idp_session", "xapp_idp_csrf"}
        require(
            set(cookie_by_name) == expected_cookie_names,
            f"unexpected browser cookies: {sorted(set(cookie_by_name) - expected_cookie_names)}",
        )
        for name in expected_cookie_names:
            require(name in cookie_by_name, f"missing browser cookie {name}")
            require(cookie_by_name[name]["secure"], f"cookie {name} is not Secure")
            require(cookie_by_name[name]["httpOnly"], f"cookie {name} is not HttpOnly")
        require(
            cookie_by_name["xapp_session"]["path"] == "/",
            "application session cookie must cover the application",
        )
        require(
            cookie_by_name["xapp_idp_session"]["path"] == "/idp",
            "IdP session cookie must be issuer-scoped",
        )

        no_csrf = alice.request.post(
            base_url + "/api/object",
            data=json.dumps({"must": "fail"}),
            headers={"Content-Type": "application/json"},
        )
        require(no_csrf.status == 403, f"missing-CSRF write status={no_csrf.status}")
        alice_initial_document = loaded_document(alice)
        if args.expect_existing:
            require(
                alice_initial_document == alice_value,
                "Alice value did not survive the process restart",
            )
        save_document(alice, alice_value)
        alice.reload(wait_until="domcontentloaded")
        alice.wait_for_selector("#app:not(.d-none)")
        require(loaded_document(alice) == alice_value, "Alice value did not persist")

        sign_in(bob, base_url, "bob", read_secret(args.bob_password_file))
        bob_session = session(bob)
        require(
            bob_session["userId"] != alice_session["userId"],
            "distinct OIDC subjects collapsed to one application user",
        )
        bob_initial_document = loaded_document(bob)
        require(bob_initial_document != alice_value, "Bob read Alice's private object")
        if args.expect_existing:
            require(
                bob_initial_document == bob_value,
                "Bob value did not survive the process restart",
            )
        save_document(bob, bob_value)
        require(loaded_document(bob) == bob_value, "Bob value did not persist")
        require(loaded_document(alice) == alice_value, "Bob overwrote Alice's object")

        logout_without_csrf = alice.request.post(base_url + "/auth/logout")
        require(
            logout_without_csrf.status == 403,
            f"logout without csrf status={logout_without_csrf.status}",
        )
        session_after_rejected_logout = alice.request.get(base_url + "/auth/session")
        require(
            session_after_rejected_logout.status == 200,
            "csrf-rejected logout revoked the application session",
        )
        with alice.expect_response(
            lambda response: response.url.endswith("/auth/logout")
            and response.request.method == "POST"
        ) as logout_response_info:
            alice.get_by_role("button", name="Log out").click()
        logout_with_csrf = logout_response_info.value
        require(
            logout_with_csrf.status == 204,
            f"logout with csrf status={logout_with_csrf.status}",
        )
        alice.wait_for_function(
            "document.querySelector('#status').textContent.includes('Application session ended')"
        )
        require(alice.locator("#login").is_visible(), "post-logout sign-in link is hidden")
        post_logout = alice.request.get(base_url + "/auth/session")
        require(post_logout.status == 401, "logout did not revoke application session")

        alice.get_by_role("link", name="Sign in again").click()
        alice.wait_for_url(base_url + "/")
        alice.wait_for_selector("#app:not(.d-none)")
        silent_reauthentication_session = session(alice)
        cleanup_logout = alice.request.post(
            base_url + "/auth/logout",
            headers={
                "X-CSRF-Token": silent_reauthentication_session["csrfToken"]
            },
        )
        require(cleanup_logout.status == 204, "cleanup logout failed")
        final_session = alice.request.get(base_url + "/auth/session")
        require(final_session.status == 401, "cleanup logout left a live session")

        result = {
            "baseUrl": base_url,
            "distinctApplicationUsers": bob_session["userId"]
            != alice_session["userId"],
            "cookies": alice_cookies,
            "missingCsrfObjectWriteStatus": no_csrf.status,
            "aliceDocument": alice_value,
            "bobDocument": bob_value,
            "aliceValuePresentBeforeWrite": alice_initial_document == alice_value,
            "bobValuePresentBeforeWrite": bob_initial_document == bob_value,
            "logoutWithoutCsrfStatus": logout_without_csrf.status,
            "sessionAfterRejectedLogoutStatus": session_after_rejected_logout.status,
            "logoutWithCsrfStatus": logout_with_csrf.status,
            "sessionAfterLogoutStatus": post_logout.status,
            "postLogoutUIVisible": True,
            "retainedIdPSessionCompletedExplicitSignIn": True,
            "finalSessionStatus": final_session.status,
        }
        browser.close()
        return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--alice-password-file", type=Path, required=True)
    parser.add_argument("--bob-password-file", type=Path, required=True)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--expect-existing", action="store_true")
    parser.add_argument("--headed", action="store_true")
    return parser.parse_args()


if __name__ == "__main__":
    arguments = parse_args()
    rendered = json.dumps(run(arguments), indent=2, sort_keys=True) + "\n"
    if arguments.output is not None:
        arguments.output.write_text(rendered, encoding="utf-8")
    print(rendered, end="")
