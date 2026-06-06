"""
QA Capsule DOM Capture Listener for Robot Framework Browser Library.

On every test failure this listener:
  1. Captures the full page HTML from the Browser library.
  2. Wraps it in QA_CAPSULE_DOM_SNAPSHOT_START / END markers.
  3. Logs it via Robot Framework's built-in logging system so it appears in
     both the HTML report AND the per-test <system-out> element of the xunit
     XML output that QA Capsule ingests.
  4. Takes a screenshot and logs its path.

Attach to a test suite with:
    robot --listener path/to/dom_capture_listener.py ...

QA Capsule parses the <system-out> field of each failing <testcase> in the
uploaded JUnit XML, extracts the HTML between the markers, and passes it to
Groq / Gemini to find the correct replacement locator.

Environment variables:
  QA_CAPSULE_DOM_CAPTURE   Set to "0" to disable capture (default: enabled).
  ROBOT_OUTPUT_DIR         Directory for screenshots (default: tests/results).
"""
import os
import sys

try:
    from robot.libraries.BuiltIn import BuiltIn
    from robot.api import logger as robot_logger
except ImportError:
    BuiltIn = None
    robot_logger = None

CAPTURE_ENABLED = os.environ.get("QA_CAPSULE_DOM_CAPTURE", "1") != "0"
# Keep well under LLM token limits while preserving the full DOM structure.
MAX_HTML_CHARS = 24_000


class dom_capture_listener:
    """Robot Framework listener (API v3) that captures DOM + screenshot on failure."""

    ROBOT_LISTENER_API_VERSION = 3

    def end_test(self, data, result):
        """Called by Robot Framework after each test completes."""
        if not CAPTURE_ENABLED or result.passed:
            return
        self._capture(result.name)

    # ── internal helpers ──────────────────────────────────────────────────────

    def _capture(self, test_name: str) -> None:
        """Capture page HTML and screenshot for a failing test."""
        try:
            browser = BuiltIn().get_library_instance("Browser")
        except Exception:
            # Browser library is not loaded in this suite — nothing to capture.
            return

        self._capture_html(browser, test_name)
        self._capture_screenshot(browser, test_name)

    def _capture_html(self, browser, test_name: str) -> None:
        """Write the page HTML between QA Capsule DOM snapshot markers.

        We write directly to sys.stdout so that Robot Framework's xunit
        reporter captures the output in the <system-out> element of the
        failing <testcase>.  robot.api.logger.info() inside a listener's
        end_test hook is NOT guaranteed to appear in the per-test system-out;
        it may be written to the suite-level block or the log.html file only.

        sys.stdout.write() is the most reliable cross-version approach.
        """
        try:
            html = browser.get_page_source()
            if not html:
                return
            trimmed = html[:MAX_HTML_CHARS]
            snapshot = (
                f"\n[QA_CAPSULE_DOM_SNAPSHOT_START]\n"
                f"{trimmed}\n"
                f"[QA_CAPSULE_DOM_SNAPSHOT_END]\n"
            )
            # Direct stdout write — always captured in xunit <system-out>.
            sys.stdout.write(snapshot)
            sys.stdout.flush()
            # Also write to Robot Framework log.html for human inspection.
            if robot_logger is not None:
                robot_logger.info(f"DOM snapshot captured ({len(trimmed)} chars)", also_console=False)
        except Exception as exc:
            _warn(f"DOM capture failed for '{test_name}': {exc}")

    def _capture_screenshot(self, browser, test_name: str) -> None:
        """Take a screenshot and log its path."""
        try:
            output_dir = os.environ.get("ROBOT_OUTPUT_DIR", "tests/results")
            os.makedirs(output_dir, exist_ok=True)
            safe_name = "".join(c if c.isalnum() else "_" for c in test_name)[:60]
            screenshot_path = os.path.join(output_dir, f"failure_{safe_name}.png")
            browser.take_screenshot(filename=screenshot_path)
            msg = f"[QA_CAPSULE_SCREENSHOT]: {screenshot_path}"
            if robot_logger is not None:
                robot_logger.info(msg, also_console=True)
            else:
                print(msg, flush=True)
        except Exception:
            # Screenshot is optional — never let this fail the test run.
            pass


def _warn(message: str) -> None:
    """Write a warning to stderr without raising."""
    print(f"[QA Capsule] {message}", file=sys.stderr, flush=True)
