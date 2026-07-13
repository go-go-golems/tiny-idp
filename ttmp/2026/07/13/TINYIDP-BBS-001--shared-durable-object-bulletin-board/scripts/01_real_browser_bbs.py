#!/usr/bin/env python3
"""Exercise the shared BBS with two independent system-Chrome contexts.

Run mode ``create`` before restarting the external tmux server. Run mode
``verify-restart`` after restarting with the same state root; it verifies the
persisted thread and removes it through Alice's UI.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

from playwright.sync_api import BrowserContext, Page, TimeoutError as PlaywrightTimeoutError, sync_playwright


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def read_secret(path: Path) -> str:
    value = path.read_text(encoding="utf-8").strip()
    require(bool(value), f"empty password file: {path}")
    return value


def install_diagnostics(page: Page) -> list[str]:
    messages: list[str] = []
    page.on("console", lambda message: messages.append(f"console[{message.type}] {message.text}"))
    page.on("pageerror", lambda error: messages.append(f"pageerror {error}"))
    page.on("requestfailed", lambda request: messages.append(f"requestfailed {request.method} {request.url}: {request.failure}"))
    return messages


def sign_in(page: Page, base_url: str, login: str, password: str, diagnostics: list[str]) -> None:
    page.goto(base_url + "/", wait_until="domcontentloaded")
    try:
        page.wait_for_selector('a[href="/auth/login?return_to=/"], [data-testid="current-user"]')
    except PlaywrightTimeoutError as error:
        body = page.locator("body").inner_text()
        raise AssertionError(
            f"application did not render session state; url={page.url}; "
            f"body={body!r}; diagnostics={diagnostics!r}"
        ) from error
    if page.locator('[data-testid="current-user"]').is_visible():
        return
    sign_in_link = page.get_by_role("link", name="Sign in", exact=True)
    sign_in_link.click()
    page.wait_for_selector('input[name="login"]')
    page.fill('input[name="login"]', login)
    page.fill('input[name="password"]', password)
    page.get_by_role("button", name="Approve").click()
    page.wait_for_url(base_url + "/")
    page.wait_for_selector('[data-testid="current-user"]')


def session(page: Page, base_url: str) -> dict[str, Any]:
    response = page.request.get(base_url + "/auth/session")
    require(response.status == 200, f"session status={response.status}")
    return response.json()


def board(page: Page, base_url: str) -> dict[str, Any]:
    response = page.request.get(base_url + "/api/bbs")
    require(response.status == 200, f"board status={response.status}")
    return response.json()


def actor_display_name(page: Page, base_url: str) -> str:
    response = page.request.get(base_url + "/api/me")
    require(response.status == 200, f"current-user status={response.status}")
    current_user = response.json()
    claims = current_user.get("claims") or {}
    candidate = claims.get("name") or claims.get("preferredUsername") or "Member"
    display_name = str(candidate).strip()[:80] or "Member"
    require(bool(display_name), "trusted actor display name is empty")
    return display_name


def find_post(document: dict[str, Any], title: str) -> dict[str, Any] | None:
    return next((post for post in document["posts"] if post["title"] == title), None)


def cookie_summary(context: BrowserContext) -> list[dict[str, Any]]:
    keys = ("name", "path", "secure", "httpOnly", "sameSite")
    return [{key: cookie[key] for key in keys} for cookie in context.cookies()]


def assert_cookie_security(context: BrowserContext) -> list[dict[str, Any]]:
    cookies = cookie_summary(context)
    by_name = {cookie["name"]: cookie for cookie in cookies}
    expected = {"xapp_session", "xapp_idp_session", "xapp_idp_csrf"}
    require(set(by_name) == expected, f"unexpected cookie names: {sorted(by_name)}")
    for name in expected:
        require(by_name[name]["secure"], f"cookie {name} is not Secure")
        require(by_name[name]["httpOnly"], f"cookie {name} is not HttpOnly")
    require(by_name["xapp_session"]["path"] == "/", "app cookie path is not /")
    require(by_name["xapp_idp_session"]["path"] == "/idp", "IdP cookie path is not /idp")
    return cookies


def create_post_through_ui(page: Page, title: str, content: str) -> None:
    page.get_by_label("Title").fill(title)
    page.get_by_label("Category").select_option("projects")
    page.get_by_label("Message").fill(content)
    with page.expect_response(
        lambda response: response.url.endswith("/api/bbs/posts")
        and response.request.method == "POST"
    ) as response_info:
        page.get_by_role("button", name="Post dispatch").click()
    response = response_info.value
    require(response.status == 201, f"post create status={response.status}")
    page.get_by_text(title, exact=True).wait_for()


def reply_through_ui(page: Page, title: str, content: str) -> None:
    thread = page.locator("article.thread").filter(has_text=title)
    require(thread.count() == 1, "expected exactly one target thread")
    thread.get_by_label("Add a reply").fill(content)
    with page.expect_response(
        lambda response: "/replies" in response.url
        and response.request.method == "POST"
    ) as response_info:
        thread.get_by_role("button", name="Reply").click()
    response = response_info.value
    require(response.status == 201, f"reply status={response.status}")
    thread.get_by_text(content, exact=True).wait_for()


def delete_through_ui(page: Page, title: str) -> None:
    thread = page.locator("article.thread").filter(has_text=title)
    require(thread.count() == 1, "target thread missing before deletion")
    with page.expect_response(
        lambda response: "/api/bbs/posts/" in response.url
        and response.request.method == "DELETE"
    ) as response_info:
        thread.get_by_role("button", name="Delete thread and replies").click()
    response = response_info.value
    require(response.status == 200, f"owner delete status={response.status}")
    thread.wait_for(state="detached")


def assert_logout_lifecycle(page: Page, base_url: str) -> None:
    rejected = page.request.post(base_url + "/auth/logout")
    require(rejected.status == 403, f"logout without CSRF status={rejected.status}")
    with page.expect_response(
        lambda response: response.url.endswith("/auth/logout")
        and response.request.method == "POST"
    ) as response_info:
        page.get_by_role("button", name="Log out").click()
    require(response_info.value.status == 204, "valid logout did not return 204")
    page.wait_for_selector('[data-testid="session-ended"]')
    require(page.request.get(base_url + "/auth/session").status == 401, "session survived logout")
    page.get_by_role("link", name="Sign in again").click()
    page.wait_for_url(base_url + "/")
    page.wait_for_selector('[data-testid="current-user"]')


def run(args: argparse.Namespace) -> dict[str, Any]:
    base_url = args.base_url.rstrip("/")
    title = f"BBS browser checkpoint {args.marker}"
    hostile_content = "Stored as text: <img src=x onerror=alert(1)>"
    reply_content = f"Bob verified shared state for {args.marker}."

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            executable_path="/usr/bin/google-chrome",
            headless=not args.headed,
        )
        alice_context = browser.new_context(ignore_https_errors=True)
        bob_context = browser.new_context(ignore_https_errors=True)
        alice = alice_context.new_page()
        bob = bob_context.new_page()

        alice_diagnostics = install_diagnostics(alice)
        bob_diagnostics = install_diagnostics(bob)

        sign_in(alice, base_url, "alice", read_secret(args.alice_password_file), alice_diagnostics)
        alice_cookies = assert_cookie_security(alice_context)
        alice_session = session(alice, base_url)
        alice_display_name = actor_display_name(alice, base_url)

        if args.mode == "create":
            existing = find_post(board(alice, base_url), title)
            if existing is not None:
                cleanup = alice.request.delete(
                    base_url + f"/api/bbs/posts/{existing['id']}",
                    headers={"X-CSRF-Token": alice_session["csrfToken"]},
                )
                require(cleanup.status == 200, "could not remove stale checkpoint")

            no_csrf = alice.request.post(
                base_url + "/api/bbs/posts",
                data=json.dumps({"title": "Denied", "body": "No token", "category": "general"}),
                headers={"Content-Type": "application/json"},
            )
            require(no_csrf.status == 403, f"missing-CSRF post status={no_csrf.status}")
            create_post_through_ui(alice, title, hostile_content)
            require(alice.locator('img[src="x"]').count() == 0, "stored markup created an img element")

            created = find_post(board(alice, base_url), title)
            require(created is not None, "created post missing from API board")
            require(
                created["author"] == alice_display_name,
                f"stored author {created['author']!r} did not match trusted actor label {alice_display_name!r}",
            )
            require(created["canDelete"], "Alice cannot delete her own post")

            sign_in(bob, base_url, "bob", read_secret(args.bob_password_file), bob_diagnostics)
            bob_session = session(bob, base_url)
            require(bob_session["userId"] != alice_session["userId"], "Alice and Bob share an app user")
            bob.get_by_text(title, exact=True).wait_for()
            bob_thread = bob.locator("article.thread").filter(has_text=title)
            require(bob_thread.get_by_role("button", name="Delete thread and replies").count() == 0, "Bob sees owner delete control")
            reply_through_ui(bob, title, reply_content)

            denied_delete = bob.request.delete(
                base_url + f"/api/bbs/posts/{created['id']}",
                headers={"X-CSRF-Token": bob_session["csrfToken"]},
            )
            require(denied_delete.status == 403, f"Bob delete status={denied_delete.status}")
            alice.reload(wait_until="domcontentloaded")
            alice.get_by_text(reply_content, exact=True).wait_for()
            if args.screenshot is not None:
                args.screenshot.parent.mkdir(parents=True, exist_ok=True)
                alice.screenshot(path=str(args.screenshot), full_page=True)
            assert_logout_lifecycle(alice, base_url)
            result = {
                "mode": args.mode,
                "title": title,
                "missingCsrfPostStatus": no_csrf.status,
                "distinctApplicationUsers": True,
                "bobDeleteStatus": denied_delete.status,
                "storedMarkupRenderedAsText": True,
                "trustedAuthorLabel": alice_display_name,
                "postLeftForRestart": True,
                "cookieSecurity": alice_cookies,
                "logoutLifecyclePassed": True,
            }
        else:
            persisted = find_post(board(alice, base_url), title)
            require(persisted is not None, "checkpoint post did not survive restart")
            require(any(reply["body"] == reply_content for reply in persisted["replies"]), "checkpoint reply did not survive restart")
            alice.get_by_text(title, exact=True).wait_for()
            delete_through_ui(alice, title)
            require(find_post(board(alice, base_url), title) is None, "checkpoint survived owner deletion")
            if args.screenshot is not None:
                args.screenshot.parent.mkdir(parents=True, exist_ok=True)
                alice.screenshot(path=str(args.screenshot), full_page=True)
            result = {
                "mode": args.mode,
                "title": title,
                "postSurvivedRestart": True,
                "replySurvivedRestart": True,
                "ownerDeletionPassed": True,
            }

        browser.close()
        return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--alice-password-file", type=Path, required=True)
    parser.add_argument("--bob-password-file", type=Path, required=True)
    parser.add_argument("--mode", choices=("create", "verify-restart"), required=True)
    parser.add_argument("--marker", default="TINYIDP-BBS-001")
    parser.add_argument("--screenshot", type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--headed", action="store_true")
    return parser.parse_args()


if __name__ == "__main__":
    arguments = parse_args()
    rendered = json.dumps(run(arguments), indent=2, sort_keys=True) + "\n"
    if arguments.output is not None:
        arguments.output.parent.mkdir(parents=True, exist_ok=True)
        arguments.output.write_text(rendered, encoding="utf-8")
    print(rendered, end="")
