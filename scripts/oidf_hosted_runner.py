#!/usr/bin/env python3
"""Hybrid runner for the hosted OpenID Foundation conformance suite.

The certification suite exposes JSON APIs for creating test instances and
polling status, but individual tests still export browser URLs that must be
visited as the OP user.  This script drives the API with an authenticated
JSESSIONID cookie and uses a plain HTTP session to follow the exported
authorization URLs through tiny-idp login/consent pages and back to the suite.

It is intentionally conservative: it stops on failed tests, on tests that need
manual review with no actionable browser URL, and when it cannot understand a
browser interaction.  The suite session cookie can be copied from an already
logged-in browser, or supplied via OIDF_JSESSIONID.
"""

from __future__ import annotations

import argparse
import html.parser
import json
import os
import re
import sys
import time
import urllib.parse
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import requests

DEFAULT_BASE_URL = "https://www.certification.openid.net"
TERMINAL_STATUSES = {"FINISHED", "INTERRUPTED"}
PASS_RESULTS = {"PASSED", "WARNING", "SKIPPED"}


class FormParser(html.parser.HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.forms: list[dict[str, Any]] = []
        self._current: dict[str, Any] | None = None

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        attrs_dict = {k: (v or "") for k, v in attrs}
        if tag.lower() == "form":
            self._current = {
                "action": attrs_dict.get("action", ""),
                "method": attrs_dict.get("method", "get").lower(),
                "inputs": [],
            }
            self.forms.append(self._current)
        elif tag.lower() == "input" and self._current is not None:
            self._current["inputs"].append(attrs_dict)

    def handle_endtag(self, tag: str) -> None:
        if tag.lower() == "form":
            self._current = None


@dataclass
class SuiteClient:
    base_url: str
    session: requests.Session

    @classmethod
    def from_cookie(cls, base_url: str, jsessionid: str) -> "SuiteClient":
        sess = requests.Session()
        host = urllib.parse.urlparse(base_url).hostname or "www.certification.openid.net"
        sess.cookies.set("JSESSIONID", jsessionid, domain=host, path="/")
        sess.headers.update({"Accept": "application/json"})
        return cls(base_url=base_url.rstrip("/"), session=sess)

    def api(self, method: str, path: str, **kwargs: Any) -> Any:
        url = self.base_url + path
        resp = self.session.request(method, url, timeout=60, **kwargs)
        if resp.status_code >= 400:
            raise RuntimeError(f"{method} {url} failed: {resp.status_code} {resp.text[:1000]}")
        if resp.text.strip():
            return resp.json()
        return None

    def current_user(self) -> Any:
        return self.api("GET", "/api/currentuser")

    def plan(self, plan_id: str) -> dict[str, Any]:
        return self.api("GET", f"/api/plan/{plan_id}")

    def info(self, test_id: str) -> dict[str, Any]:
        return self.api("GET", f"/api/info/{test_id}")

    def runner(self, test_id: str) -> dict[str, Any]:
        return self.api("GET", f"/api/runner/{test_id}")

    def log(self, test_id: str) -> list[dict[str, Any]]:
        return self.api("GET", f"/api/log/{test_id}", params={"public": "false"})

    def start_test(self, plan_id: str, test_name: str, variant: dict[str, Any]) -> str:
        data = self.api(
            "POST",
            "/api/runner",
            params={"test": test_name, "plan": plan_id, "variant": json.dumps(variant, separators=(",", ":"))},
        )
        test_id = data.get("id") or data.get("testId") or data.get("_id")
        if not test_id:
            raise RuntimeError(f"runner creation response did not contain test id: {data}")
        return test_id


class BrowserDriver:
    def __init__(
        self,
        login: str,
        password: str = "",
        verbose: bool = False,
        suite_base_url: str | None = None,
        suite_jsessionid: str | None = None,
    ) -> None:
        self.session = requests.Session()
        if suite_base_url and suite_jsessionid:
            host = urllib.parse.urlparse(suite_base_url).hostname or "www.certification.openid.net"
            self.session.cookies.set("JSESSIONID", suite_jsessionid, domain=host, path="/")
        self.login = login
        self.password = password
        self.verbose = verbose

    def drive(self, url: str, method: str = "GET", body: str | None = None) -> None:
        self._debug(f"browser {method} {url}")
        if method.upper() == "POST":
            headers = {"Content-Type": "application/x-www-form-urlencoded"}
            resp = self.session.post(url, data=body or "", headers=headers, timeout=60, allow_redirects=True)
        else:
            resp = self.session.get(url, timeout=60, allow_redirects=True)
        self._follow_interaction(resp)

    def _follow_interaction(self, resp: requests.Response, depth: int = 0) -> None:
        if depth > 8:
            raise RuntimeError("too many nested browser interactions")
        self._debug(f"browser response {resp.status_code} {resp.url} {resp.headers.get('content-type','')}")
        text = resp.text or ""

        # The suite's implicit callback page uses JavaScript to POST to an
        # implicit submission URL. Reproduce that POST explicitly.
        implicit = self._find_implicit_submit_url(text, resp.url)
        if implicit:
            self._debug(f"implicit submit POST {implicit}")
            follow = self.session.post(implicit, timeout=60, allow_redirects=True)
            self._follow_interaction(follow, depth + 1)
            return

        form = self._find_actionable_form(text)
        if form is None:
            return

        action = urllib.parse.urljoin(resp.url, form["action"] or resp.url)
        method = form["method"].upper()
        fields: dict[str, str] = {}
        for inp in form["inputs"]:
            name = inp.get("name")
            if not name:
                continue
            typ = inp.get("type", "text").lower()
            value = inp.get("value", "")
            if name == "login":
                value = self.login
            elif name == "password":
                value = self.password
            elif name == "consent_approved":
                value = value or "true"
            elif typ in {"checkbox", "radio"} and name != "consent_approved" and "checked" not in inp:
                continue
            fields[name] = value

        self._debug(f"submit form {method} {action} fields={sorted(fields)}")
        if method == "POST":
            follow = self.session.post(action, data=fields, timeout=60, allow_redirects=True)
        else:
            follow = self.session.get(action, params=fields, timeout=60, allow_redirects=True)
        self._follow_interaction(follow, depth + 1)

    def _find_actionable_form(self, text: str) -> dict[str, Any] | None:
        parser = FormParser()
        parser.feed(text)
        for form in parser.forms:
            names = {inp.get("name", "") for inp in form["inputs"]}
            if {"csrf_token"} & names or {"login", "consent_approved"} & names:
                return form
        return None

    def _find_implicit_submit_url(self, text: str, base_url: str) -> str | None:
        unescaped = text.replace("\\/", "/")
        patterns = [
            r"implicitSubmitUrl\s*[=:]\s*['\"]([^'\"]+)",
            r"xhr\.open\(['\"]POST['\"],\s*['\"]([^'\"]+)",
            r"/test/a/[^'\"<>]+/implicit/[^'\"<>]+",
        ]
        for candidate in (text, unescaped):
            for pattern in patterns:
                m = re.search(pattern, candidate)
                if m:
                    value = m.group(1) if m.lastindex else m.group(0)
                    value = value.replace("\\/", "/").replace("&amp;", "&")
                    return urllib.parse.urljoin(base_url, value)
        return None

    def _debug(self, msg: str) -> None:
        if self.verbose:
            print(f"[browser] {msg}", file=sys.stderr)


def module_variant(module: dict[str, Any]) -> dict[str, Any]:
    variant = dict(module.get("variant") or {})
    # Older UI calls sometimes submit only the module variant.  The runner API
    # also accepts plan-level parameters; including them makes created instances
    # self-describing in /api/info.
    return variant


def choose_modules(plan: dict[str, Any], only: set[str] | None, remaining: bool) -> list[dict[str, Any]]:
    modules = list(plan.get("modules", []))
    if only:
        modules = [m for m in modules if m.get("testModule") in only]
    if remaining:
        modules = [m for m in modules if not m.get("instances")]
    return modules


def latest_instance(module: dict[str, Any]) -> str | None:
    instances = module.get("instances") or []
    return instances[-1] if instances else None


def pending_browser_actions(runner: dict[str, Any]) -> list[tuple[str, str, str | None]]:
    browser = runner.get("browser") or {}
    actions: list[tuple[str, str, str | None]] = []
    visited = set(browser.get("visited") or [])
    visited_with_method = {
        (item.get("method", "GET").upper(), item.get("url"))
        for item in browser.get("visitedUrlsWithMethod") or []
        if item.get("url")
    }
    for item in browser.get("urlsWithMethod") or []:
        url = item.get("url")
        if not url:
            continue
        method = item.get("method", "GET").upper()
        if (method, url) not in visited_with_method:
            actions.append((url, method, item.get("body") or item.get("requestBody")))
    if not actions:
        for url in browser.get("urls") or []:
            if url not in visited:
                actions.append((url, "GET", None))
    return actions


def run_one(
    suite: SuiteClient,
    browser: BrowserDriver,
    plan_id: str,
    test_name: str,
    variant: dict[str, Any],
    poll_seconds: float,
    timeout_seconds: float,
    artifacts: Path | None,
    reuse_test_id: str | None = None,
) -> tuple[str, str, str | None]:
    test_id = reuse_test_id or suite.start_test(plan_id, test_name, variant)
    print(f"==> {test_name}: {test_id}")
    started = time.monotonic()
    driven: set[tuple[str, str]] = set()

    while True:
        info = suite.info(test_id)
        status = info.get("status") or "UNKNOWN"
        result = info.get("result") or info.get("testmodule_result")
        print(f"    status={status} result={result or '-'}")

        if artifacts:
            artifacts.mkdir(parents=True, exist_ok=True)
            (artifacts / f"{test_id}.info.json").write_text(json.dumps(info, indent=2, sort_keys=True))
            try:
                (artifacts / f"{test_id}.log.json").write_text(json.dumps(suite.log(test_id), indent=2, sort_keys=True))
            except Exception as e:  # keep polling even if log fetch transiently fails
                print(f"    warning: could not save log: {e}", file=sys.stderr)

        if status in TERMINAL_STATUSES:
            return test_id, status, result

        runner = suite.runner(test_id)
        actions = pending_browser_actions(runner)
        remaining_actions = [(url, method, body) for url, method, body in actions if (method, url) not in driven]
        progressed = False
        for url, method, body in remaining_actions:
            browser.drive(url, method=method, body=body)
            driven.add((method, url))
            progressed = True

        if not progressed and status in {"REVIEW", "WAITING", "RUNNING"}:
            logs = suite.log(test_id)
            review = [e for e in logs if e.get("result") == "REVIEW"]
            if review and not remaining_actions:
                msg = review[-1].get("msg") or "manual review required"
                print(f"    manual review required: {msg}")
                return test_id, status, "REVIEW"

        if time.monotonic() - started > timeout_seconds:
            return test_id, "TIMEOUT", None
        time.sleep(poll_seconds)


def main() -> int:
    parser = argparse.ArgumentParser(description="Run hosted OIDF conformance plan modules with Python automation")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--cookie", default=os.environ.get("OIDF_JSESSIONID"), help="JSESSIONID value or 'JSESSIONID=...' string")
    parser.add_argument("--plan", required=True, help="Hosted suite test plan id")
    parser.add_argument("--only", action="append", help="Run only this test module name; may be repeated")
    parser.add_argument("--remaining", action="store_true", help="Run only plan modules with no instances yet")
    parser.add_argument("--resume", action="store_true", help="Poll/drive latest existing instances instead of creating new ones")
    parser.add_argument("--login", default=os.environ.get("TINYIDP_LOGIN", "alice"))
    parser.add_argument("--password", default=os.environ.get("TINYIDP_PASSWORD", ""))
    parser.add_argument("--poll", type=float, default=2.0)
    parser.add_argument("--timeout", type=float, default=300.0)
    parser.add_argument("--artifacts", type=Path, default=None, help="Directory for info/log JSON artifacts")
    parser.add_argument("--keep-going", action="store_true", help="Continue after failures/manual-review results")
    parser.add_argument("--dry-run", action="store_true", help="List selected modules without starting tests")
    parser.add_argument("--verbose", action="store_true")
    args = parser.parse_args()

    if not args.cookie:
        print("error: pass --cookie or set OIDF_JSESSIONID", file=sys.stderr)
        return 2
    cookie = args.cookie.split("=", 1)[1] if args.cookie.startswith("JSESSIONID=") else args.cookie

    suite = SuiteClient.from_cookie(args.base_url, cookie)
    try:
        user = suite.current_user()
    except Exception as e:
        print(f"error: could not authenticate to suite API: {e}", file=sys.stderr)
        return 2
    print(f"Authenticated as: {json.dumps(user, sort_keys=True)}")

    plan = suite.plan(args.plan)
    modules = choose_modules(plan, set(args.only or []) or None, args.remaining)
    if not modules:
        print("No modules selected")
        return 0

    print(f"Plan {args.plan}: {plan.get('planName')} ({len(modules)} selected)")
    for i, m in enumerate(modules, 1):
        print(f"  {i:02d}. {m.get('testModule')} variant={json.dumps(module_variant(m), sort_keys=True)} instances={m.get('instances') or []}")
    if args.dry_run:
        return 0

    browser = BrowserDriver(
        login=args.login,
        password=args.password,
        verbose=args.verbose,
        suite_base_url=args.base_url,
        suite_jsessionid=cookie,
    )
    failures = 0
    for m in modules:
        name = m["testModule"]
        reuse = latest_instance(m) if args.resume else None
        if args.resume and not reuse:
            print(f"==> {name}: skipped; no existing instance to resume")
            continue
        test_id, status, result = run_one(
            suite,
            browser,
            args.plan,
            name,
            module_variant(m),
            poll_seconds=args.poll,
            timeout_seconds=args.timeout,
            artifacts=args.artifacts,
            reuse_test_id=reuse,
        )
        ok = status == "FINISHED" and result in PASS_RESULTS
        if not ok:
            failures += 1
            print(f"FAILED/STOP: {name} {test_id} status={status} result={result}")
            if not args.keep_going:
                break

    if failures:
        print(f"Completed with {failures} failing/manual/timeout module(s)")
        return 1
    print("Completed selected modules successfully")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
