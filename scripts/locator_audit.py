#!/usr/bin/env python3
"""
QA Capsule — Proactive Locator Audit
=====================================
Reads ALL test files in the repo, extracts every locator/selector,
opens the target app URL in a headless browser and checks which ones
no longer exist on the page.

Broken locators are sent to QA Capsule as incidents so Groq/Gemini
can propose fixes BEFORE a test fails in CI.

Usage:
  python scripts/locator_audit.py \
      --app-url  https://your-app.com \
      --tests-dir tests/ \
      --qac-url  https://your-qacapsule.com \
      --api-key  sre_pk_your_project_key

Environment variables (alternative to CLI args):
  APP_URL          Target application URL
  TESTS_DIR        Root folder of test files (default: tests/)
  QA_CAPSULE_URL   QA Capsule base URL
  QA_CAPSULE_API_KEY  Project API key

Dependencies:
  pip install playwright beautifulsoup4 requests
  playwright install chromium
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import NamedTuple

import requests

try:
    from playwright.sync_api import sync_playwright, TimeoutError as PWTimeout
except ImportError:
    print("ERROR: pip install playwright && playwright install chromium", file=sys.stderr)
    sys.exit(1)


# ── Locator extraction patterns per framework ─────────────────────────────────

PATTERNS = [
    # Robot Framework Browser / SeleniumLibrary
    (r"button\[data-[^\]]+\]",          "robot"),
    (r"input\[data-[^\]]+\]",           "robot"),
    (r"\[data-qa=['\"]([^'\"]+)['\"]\]","robot"),
    (r"\[data-testid=['\"]([^'\"]+)['\"]\]", "robot"),
    (r"id=(\w[\w-]*)",                  "robot"),
    (r"css=([^\s\n,]+)",                "robot"),
    (r"xpath=([^\s\n]+)",               "robot"),
    # Playwright / generic CSS
    (r"getByRole\(['\"]([^'\"]+)['\"]", "playwright"),
    (r"getByText\(['\"]([^'\"]+)['\"]", "playwright"),
    (r"getByLabel\(['\"]([^'\"]+)['\"]","playwright"),
    (r"locator\(['\"]([^'\"]+)['\"]\)", "playwright"),
    # Selenium Python/Java
    (r'By\.CSS_SELECTOR,\s*["\']([^"\']+)["\']', "selenium"),
    (r'By\.XPATH,\s*["\']([^"\']+)["\']',         "selenium"),
    (r'By\.ID,\s*["\']([^"\']+)["\']',            "selenium"),
    (r'find_element\([^,]+,\s*["\']([^"\']+)["\']',"selenium"),
    # Cypress
    (r"cy\.get\(['\"]([^'\"]+)['\"]\)", "cypress"),
    (r"cy\.contains\(['\"]([^'\"]+)['\"]","cypress"),
]

EXTENSIONS = {".robot", ".py", ".js", ".ts", ".java", ".feature", ".cs", ".rb"}

MAX_HTML = 30_000


class LocatorCheck(NamedTuple):
    locator:   str
    framework: str
    source_file: str
    source_line: int
    found:     bool
    page_url:  str


# ── File scanning ─────────────────────────────────────────────────────────────

def extract_locators_from_file(path: Path) -> list[tuple[str, str, int]]:
    """Returns list of (locator, framework, line_number)."""
    found = []
    try:
        text = path.read_text(encoding="utf-8", errors="ignore")
    except Exception:
        return []

    for lineno, line in enumerate(text.splitlines(), 1):
        for pattern, fw in PATTERNS:
            for m in re.finditer(pattern, line):
                loc = m.group(0).strip()
                if len(loc) > 3:
                    found.append((loc, fw, lineno))
    return found


def scan_test_files(root: str) -> dict[str, list[tuple[str, str, int]]]:
    """Returns {file_path: [(locator, framework, lineno), ...]}"""
    result = {}
    for p in Path(root).rglob("*"):
        if p.suffix in EXTENSIONS and ".git" not in p.parts:
            locs = extract_locators_from_file(p)
            if locs:
                result[str(p)] = locs
    return result


# ── Browser checking ──────────────────────────────────────────────────────────

def check_locators_on_page(page, url: str, locators: list[str]) -> dict[str, bool]:
    """Navigate to URL and check which locators exist in the DOM."""
    try:
        page.goto(url, timeout=15_000, wait_until="domcontentloaded")
    except Exception as e:
        print(f"  [WARN] Could not load {url}: {e}", file=sys.stderr)
        return {loc: None for loc in locators}  # None = unknown

    results = {}
    for loc in locators:
        try:
            # Normalize locator for Playwright
            if loc.startswith("id="):
                selector = f"#{loc[3:]}"
            elif loc.startswith("css="):
                selector = loc[4:]
            elif loc.startswith("xpath="):
                selector = f"xpath={loc[6:]}"
            else:
                selector = loc

            count = page.locator(selector).count()
            results[loc] = count > 0
        except Exception:
            results[loc] = False
    return results


def get_page_html(page) -> str:
    try:
        return page.content()[:MAX_HTML]
    except Exception:
        return ""


# ── QA Capsule reporting ──────────────────────────────────────────────────────

def report_broken_locator(qac_url: str, api_key: str, check: LocatorCheck, page_html: str):
    """Creates an incident in QA Capsule for a broken locator."""
    payload = {
        "results": [{
            "test_name":      f"[LocatorAudit] {Path(check.source_file).name}:{check.source_line}",
            "status":         "FAILED",
            "error_message":  (
                f"Locator '{check.locator}' not found on {check.page_url}.\n"
                f"Framework: {check.framework}\n"
                f"Source: {check.source_file} line {check.source_line}\n"
                f"[QA_CAPSULE_DOM_SNAPSHOT_START]\n{page_html}\n[QA_CAPSULE_DOM_SNAPSHOT_END]"
            ),
            "console_logs":   f"[QA_CAPSULE_DOM_SNAPSHOT_START]\n{page_html}\n[QA_CAPSULE_DOM_SNAPSHOT_END]",
            "browser":        "chromium",
            "pipeline_run_id": f"locator-audit-{os.environ.get('GITHUB_RUN_ID', 'local')}",
        }]
    }
    try:
        r = requests.post(
            f"{qac_url.rstrip('/')}/api/webhooks/",
            json=payload,
            headers={"X-API-Key": api_key, "Content-Type": "application/json"},
            timeout=10,
        )
        r.raise_for_status()
        return True
    except Exception as e:
        print(f"  [WARN] Failed to report to QA Capsule: {e}", file=sys.stderr)
        return False


# ── Main ─────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(description="QA Capsule Proactive Locator Audit")
    parser.add_argument("--app-url",   default=os.environ.get("APP_URL", ""))
    parser.add_argument("--tests-dir", default=os.environ.get("TESTS_DIR", "tests/"))
    parser.add_argument("--qac-url",   default=os.environ.get("QA_CAPSULE_URL", ""))
    parser.add_argument("--api-key",   default=os.environ.get("QA_CAPSULE_API_KEY", ""))
    parser.add_argument("--dry-run",   action="store_true", help="Print results, don't send to QA Capsule")
    args = parser.parse_args()

    if not args.app_url:
        print("ERROR: --app-url or APP_URL required", file=sys.stderr)
        sys.exit(1)

    print(f"🔍 Scanning test files in '{args.tests_dir}'…")
    all_files = scan_test_files(args.tests_dir)
    total_locators = sum(len(v) for v in all_files.values())
    print(f"   Found {len(all_files)} files, {total_locators} locator references")

    broken = []
    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=True)
        context = browser.new_context()
        page    = context.new_page()

        for file_path, locators in all_files.items():
            unique = {loc for loc, fw, ln in locators}
            print(f"\n📄 {Path(file_path).name} — checking {len(unique)} unique locators on {args.app_url}")

            results  = check_locators_on_page(page, args.app_url, list(unique))
            html     = get_page_html(page)

            for loc, fw, lineno in locators:
                found = results.get(loc)
                if found is False:
                    check = LocatorCheck(loc, fw, file_path, lineno, False, args.app_url)
                    broken.append((check, html))
                    print(f"  ❌ BROKEN  {loc}  ({fw})  line {lineno}")
                elif found is True:
                    print(f"  ✅ ok      {loc}")
                else:
                    print(f"  ⚠️  unknown {loc}")

        browser.close()

    print(f"\n{'='*60}")
    print(f"  AUDIT COMPLETE — {len(broken)} broken locator(s) found")
    print(f"{'='*60}\n")

    if not broken:
        print("✅ All locators are healthy.")
        sys.exit(0)

    if args.dry_run or not args.qac_url or not args.api_key:
        print("ℹ️  --dry-run or missing QA Capsule credentials — not reporting incidents.")
        for check, _ in broken:
            print(f"  • {check.locator}  [{check.source_file}:{check.source_line}]")
        sys.exit(1)

    print(f"📤 Reporting {len(broken)} broken locators to QA Capsule…")
    reported = 0
    for check, html in broken:
        if report_broken_locator(args.qac_url, args.api_key, check, html):
            reported += 1
            print(f"  ✅ Reported: {check.locator}")

    print(f"\n✅ {reported}/{len(broken)} incidents created in QA Capsule.")
    print("   Open Self-Healing Hub → Groq/Gemini will propose fixes automatically.")
    sys.exit(1 if broken else 0)


if __name__ == "__main__":
    main()
