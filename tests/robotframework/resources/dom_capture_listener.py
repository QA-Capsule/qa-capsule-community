"""
QA Capsule — DOM Capture Listener for Robot Framework Browser Library.

On every test FAIL, captures:
  - The full page HTML (for AI locator healing)
  - A screenshot path

Attach to your suite with:
    robot --listener resources/dom_capture_listener.py ...

The HTML is printed to stdout so it ends up in the JUnit XML console_logs
field, which QA Capsule stores and passes to Groq/Gemini for locator healing.
"""
import os
import sys

try:
    from robot.libraries.BuiltIn import BuiltIn
except ImportError:
    BuiltIn = None

CAPTURE_ENABLED = os.environ.get("QA_CAPSULE_DOM_CAPTURE", "1") != "0"
MAX_HTML_CHARS  = 24_000   # keep under token limits


class dom_capture_listener:
    ROBOT_LISTENER_API_VERSION = 3

    def end_test(self, data, result):
        if not CAPTURE_ENABLED:
            return
        if result.passed:
            return
        self._capture(result.name)

    # ── internal ──────────────────────────────────────────────────────────────

    def _capture(self, test_name: str):
        try:
            browser = BuiltIn().get_library_instance("Browser")
        except Exception:
            return  # Browser library not loaded — skip silently

        try:
            html = browser.get_page_source()
            if html:
                trimmed = html[:MAX_HTML_CHARS]
                print(
                    f"\n[QA_CAPSULE_DOM_SNAPSHOT_START]\n"
                    f"{trimmed}\n"
                    f"[QA_CAPSULE_DOM_SNAPSHOT_END]",
                    flush=True,
                )
        except Exception as e:
            print(f"[QA Capsule] DOM capture failed: {e}", file=sys.stderr)

        try:
            screenshot_dir = os.environ.get("ROBOT_OUTPUT_DIR", "tests/results")
            safe_name = "".join(c if c.isalnum() else "_" for c in test_name)[:60]
            screenshot_path = os.path.join(screenshot_dir, f"failure_{safe_name}.png")
            browser.take_screenshot(filename=screenshot_path)
            print(f"[QA_CAPSULE_SCREENSHOT]: {screenshot_path}", flush=True)
        except Exception:
            pass
