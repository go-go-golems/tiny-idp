#!/usr/bin/env python3
"""Real-browser assurance probe for a running tinyidp-xapp development server.

The probe never prints credentials, cookies, hidden field values, or page HTML.
It emits a bounded JSON summary suitable for release evidence.
"""

from __future__ import annotations

import argparse
import json
import time
from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Browser, Page, sync_playwright


EXPECTED_CSP = "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://127.0.0.1:8790")
    parser.add_argument("--login", required=True)
    parser.add_argument("--password", required=True)
    parser.add_argument("--second-login")
    parser.add_argument("--second-password")
    parser.add_argument("--screenshot", type=Path)
    args = parser.parse_args()
    if bool(args.second_login) != bool(args.second_password):
        parser.error("--second-login and --second-password must be supplied together")
    return args


def interaction_page(browser: Browser, base_url: str) -> tuple[Page, dict]:
    context = browser.new_context(viewport={"width": 1280, "height": 900})
    page = context.new_page()
    requests: list[str] = []
    responses: list[dict] = []
    page.on("request", lambda request: requests.append(request.url))
    page.on("response", lambda response: responses.append({"url": response.url, "status": response.status, "content_type": response.headers.get("content-type", "")}))
    response = page.goto(base_url + "/auth/login", wait_until="networkidle")
    assert response is not None
    assert page.url.startswith(base_url + "/idp/authorize"), page.url
    assert response.headers.get("content-security-policy") == EXPECTED_CSP
    metadata = page.evaluate(
        """() => ({
          title: document.title,
          scriptCount: document.scripts.length,
          inlineStyleCount: document.querySelectorAll('[style], style').length,
          eventAttributeCount: [...document.querySelectorAll('*')].reduce(
            (count, element) => count + [...element.attributes].filter(a => a.name.toLowerCase().startsWith('on')).length, 0),
          passwordCount: document.querySelectorAll('input[type=password]').length,
          passwordValueAttributeCount: document.querySelectorAll('input[type=password][value]').length,
          usernameAutocomplete: document.querySelector('input[name=login]')?.autocomplete,
          passwordAutocomplete: document.querySelector('input[type=password]')?.autocomplete,
          actionValues: [...document.querySelectorAll('button[name=action]')].map(button => button.value).sort(),
          stylesheetHrefs: [...document.querySelectorAll('link[rel=stylesheet]')].map(link => link.getAttribute('href')),
          horizontalOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth
        })"""
    )
    origin = origin_of(base_url)
    request_origins = sorted({origin_of(request_url) for request_url in requests})
    assert request_origins == [origin], request_origins
    assert metadata["scriptCount"] == 0
    assert metadata["inlineStyleCount"] == 0
    assert metadata["eventAttributeCount"] == 0
    assert metadata["passwordCount"] == 1
    assert metadata["passwordValueAttributeCount"] == 0
    assert metadata["usernameAutocomplete"] == "username"
    assert metadata["passwordAutocomplete"] == "current-password"
    assert metadata["actionValues"] == ["approve", "deny"]
    assert metadata["stylesheetHrefs"] == ["/static/tinyidp/login.css"]
    stylesheet = next(item for item in responses if item["url"] == base_url + "/static/tinyidp/login.css")
    assert stylesheet["status"] == 200 and stylesheet["content_type"].startswith("text/css")
    return page, {
        "csp": response.headers.get("content-security-policy"),
        "request_origins": request_origins,
        "stylesheet": stylesheet,
        "dom": metadata,
    }


def accessibility_probe(page: Page) -> dict:
    page.locator("input[name=login]").focus()
    focus = page.evaluate(
        """() => {
          const style = getComputedStyle(document.activeElement);
          return {tag: document.activeElement.tagName, name: document.activeElement.getAttribute('name'), outlineStyle: style.outlineStyle, outlineWidth: style.outlineWidth};
        }"""
    )
    assert focus["name"] == "login"
    assert focus["outlineStyle"] != "none" and float(focus["outlineWidth"].replace("px", "")) >= 2
    keyboard_order = []
    for _ in range(4):
        keyboard_order.append(page.evaluate("() => ({tag: document.activeElement.tagName, name: document.activeElement.getAttribute('name'), value: document.activeElement.tagName === 'BUTTON' ? document.activeElement.value : null})"))
        page.keyboard.press("Tab")
    assert [item["name"] for item in keyboard_order[:4]] == ["login", "password", "action", "action"]

    contrast = page.evaluate(
        """() => {
          const rgb = value => value.match(/[0-9.]+/g).slice(0, 3).map(Number);
          const luminance = value => {
            const channels = rgb(value).map(channel => {
              channel /= 255;
              return channel <= 0.04045 ? channel / 12.92 : Math.pow((channel + 0.055) / 1.055, 2.4);
            });
            return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
          };
          const ratio = (foreground, background) => {
            const values = [luminance(foreground), luminance(background)].sort((a, b) => b - a);
            return (values[0] + 0.05) / (values[1] + 0.05);
          };
          const cardBackground = getComputedStyle(document.querySelector('.identity-card')).backgroundColor;
          const samples = [
            ['body', 'body', null],
            ['eyebrow', '.eyebrow', null],
            ['approve', '.action-approve', null],
            ['deny', '.action-deny', null],
            ['scope', 'code', null],
            ['lede', '.lede', cardBackground],
            ['footer', 'footer', cardBackground]
          ];
          return Object.fromEntries(samples.map(([name, selector, inheritedBackground]) => {
            const style = getComputedStyle(document.querySelector(selector));
            const background = inheritedBackground || style.backgroundColor;
            return [name, Math.round(ratio(style.color, background) * 100) / 100];
          }));
        }"""
    )
    assert min(contrast.values()) >= 4.5, contrast

    narrow = page.context.browser.new_context(viewport={"width": 320, "height": 640})
    narrow_page = narrow.new_page()
    narrow_page.goto(page.url, wait_until="networkidle")
    narrow_overflow = narrow_page.evaluate("() => document.documentElement.scrollWidth > document.documentElement.clientWidth")
    narrow_page.set_viewport_size({"width": 1280, "height": 900})
    narrow_page.evaluate("() => { document.documentElement.style.zoom = '2'; }")
    zoom_overflow = narrow_page.evaluate("() => document.documentElement.scrollWidth > document.documentElement.clientWidth")
    narrow.close()
    assert not narrow_overflow
    assert not zoom_overflow
    return {"focused_control": focus, "keyboard_order": keyboard_order, "contrast_ratios": contrast, "narrow_horizontal_overflow": narrow_overflow, "zoom_200_observed_overflow": zoom_overflow}


def framing_probe(browser: Browser, base_url: str) -> dict:
    context = browser.new_context()
    page = context.new_page()
    page.set_content(f'<iframe id="target" src="{base_url}/auth/login"></iframe>')
    time.sleep(1)
    frame_urls = [frame.url for frame in page.frames]
    loaded_target = any(url.startswith(base_url) for url in frame_urls[1:])
    context.close()
    assert not loaded_target, frame_urls
    return {"blocked": not loaded_target, "frame_url_classes": [urlparse(url).scheme or "empty" for url in frame_urls]}


def login_probe(page: Page, base_url: str, login: str, password: str) -> tuple[dict, str]:
    page.locator("input[name=login]").fill(login)
    page.locator("input[name=password]").fill(password)
    page.locator('button[name=action][value=approve]').click()
    page.wait_for_url(base_url + "/", wait_until="networkidle")
    session_response = page.request.get(base_url + "/auth/session")
    assert session_response.status == 200
    session = session_response.json()
    assert session.get("userId") and session.get("csrfToken")
    return {"completed": True, "landing_path": urlparse(page.url).path, "session_has_user": bool(session.get("userId")), "session_has_csrf": bool(session.get("csrfToken"))}, session["userId"]


def origin_of(raw: str) -> str:
    parsed = urlparse(raw)
    return f"{parsed.scheme}://{parsed.netloc}"


def main() -> None:
    args = parse_args()
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True, executable_path="/usr/bin/chromium-browser", args=["--no-sandbox"])
        page, interaction = interaction_page(browser, args.base_url)
        accessibility = accessibility_probe(page)
        framing = framing_probe(browser, args.base_url)
        if args.screenshot:
            args.screenshot.parent.mkdir(parents=True, exist_ok=True)
            page.screenshot(path=str(args.screenshot), full_page=True)
        login, primary_user_id = login_probe(page, args.base_url, args.login, args.password)
        account_isolation = {"tested": False}
        if args.second_login:
            second_page, _ = interaction_page(browser, args.base_url)
            second_login, second_user_id = login_probe(second_page, args.base_url, args.second_login, args.second_password)
            assert second_login["completed"] and second_user_id != primary_user_id
            account_isolation = {"tested": True, "distinct_authenticated_subjects": True}
        browser.close()
    print(json.dumps({"interaction": interaction, "accessibility": accessibility, "framing": framing, "login": login, "account_isolation": account_isolation}, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
