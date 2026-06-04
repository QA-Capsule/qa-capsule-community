"""
QA Capsule — Selenium Python DOM Capture
Captures page HTML on test failure via a pytest fixture.

Usage:
  pip install pytest selenium
  Add this conftest to your test directory OR import the fixture:

  from qa_capsule_listener import qa_capsule_driver

  def test_login(qa_capsule_driver):
      driver = qa_capsule_driver
      driver.get("https://example.com")
      driver.find_element("css selector", "#broken-selector").click()
"""
import sys
import pytest

MAX_HTML = 24_000


@pytest.fixture
def qa_capsule_driver(driver, request):
    """
    Wraps the Selenium driver with automatic DOM capture on failure.
    Requires pytest-selenium (pip install pytest-selenium).
    """
    yield driver

    if request.node.rep_call.failed if hasattr(request.node, 'rep_call') else False:
        _capture(driver)


@pytest.fixture(autouse=True)
def _capture_on_failure(request, driver=None):
    yield
    if request.node.rep_call is not None and request.node.rep_call.failed:
        # Try to get driver from funcargs
        drv = request.node.funcargs.get('driver') or request.node.funcargs.get('selenium')
        if drv:
            _capture(drv)


@pytest.hookimpl(tryfirst=True, hookwrapper=True)
def pytest_runtest_makereport(item, call):
    outcome = yield
    rep = outcome.get_result()
    setattr(item, f"rep_{rep.when}", rep)


def _capture(driver):
    try:
        html = driver.page_source
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
        driver.save_screenshot("qa_capsule_failure.png")
        print("[QA_CAPSULE_SCREENSHOT]: qa_capsule_failure.png", flush=True)
    except Exception:
        pass
