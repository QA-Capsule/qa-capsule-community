"""
QA Capsule — Playwright Python DOM Capture (pytest conftest)
Captures page HTML + screenshot on every test failure.

Usage:
  Copy or import this file as conftest.py in your test directory.
  Works with playwright-pytest (pip install pytest-playwright).
"""
import sys

MAX_HTML = 24_000


def pytest_runtest_makereport(item, call):
    """Hook: after each test phase, capture DOM if failed."""
    if call.when != "call":
        return
    if call.excinfo is None:
        return

    page = item.funcargs.get("page")
    if page is None:
        return

    try:
        html = page.content()
        trimmed = html[:MAX_HTML]
        print(
            f"\n[QA_CAPSULE_DOM_SNAPSHOT_START]\n"
            f"{trimmed}\n"
            f"[QA_CAPSULE_DOM_SNAPSHOT_END]",
            flush=True,
        )
    except Exception as e:
        print(f"[QA Capsule] DOM capture failed: {e}", file=sys.stderr)

    try:
        page.screenshot(path="qa_capsule_failure.png", full_page=True)
        print("[QA_CAPSULE_SCREENSHOT]: qa_capsule_failure.png", flush=True)
    except Exception:
        pass
