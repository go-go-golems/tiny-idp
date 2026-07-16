#!/home/manuel/.pyenv/versions/3.11.3/bin/python
"""Exercise browser login, CSRF-protected BBS mutation, and logout.

The script intentionally uses the locally installed Playwright package and the
system Chromium executable. It accepts only a local xapp base URL and seeded
development credentials; it never prints cookies, CSRF tokens, or OAuth data.
"""

import argparse
from playwright.sync_api import sync_playwright


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--login", default="alice")
    parser.add_argument("--password", default="correct horse battery staple")
    args = parser.parse_args()
    base = args.base_url.rstrip("/")

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            headless=True, executable_path="/usr/bin/chromium-browser"
        )
        page = browser.new_page()
        page.goto(base + "/auth/login", wait_until="networkidle")
        page.locator('input[name="login"]').fill(args.login)
        page.locator('input[name="password"]').fill(args.password)
        page.locator('[name="action"][value="approve"]').click()
        page.wait_for_url(base + "/", wait_until="networkidle")
        page.locator("#post-title").fill("Playwright browser dispatch")
        page.locator("#post-body").fill("Created through the browser CSRF route.")
        page.locator("#post-category").select_option("notes")
        page.get_by_role("button", name="Post dispatch").click()
        page.get_by_text("Playwright browser dispatch").wait_for()
        page.get_by_role("button", name="Log out of Local Loop", exact=True).click()
        page.get_by_text("You are logged out of the application.").wait_for()
        browser.close()


if __name__ == "__main__":
    main()
