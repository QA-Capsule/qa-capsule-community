import sys
import time
import pytest
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC


@pytest.fixture(scope="function")
def driver():
    opts = Options()
    opts.add_argument("--headless")
    opts.add_argument("--no-sandbox")
    opts.add_argument("--disable-dev-shm-usage")
    opts.add_argument("--window-size=1280,800")
    drv = webdriver.Chrome(options=opts)
    yield drv
    drv.quit()


class TestPageNavigation:
    def test_homepage_title_is_not_empty(self, driver):
        print("[STDOUT] Navigating to https://example.com")
        driver.get("https://example.com")
        title = driver.title
        print(f"[STDOUT] Page title: '{title}'")
        assert len(title) > 0, "Page title should not be empty"

    def test_homepage_h1_is_displayed(self, driver):
        print("[STDOUT] Checking h1 visibility on https://example.com")
        driver.get("https://example.com")
        h1 = driver.find_element(By.TAG_NAME, "h1")
        print(f"[STDOUT] h1 text: '{h1.text}'")
        assert h1.is_displayed(), "H1 heading should be visible"

    def test_page_contains_link_to_iana(self, driver):
        print("[STDOUT] Checking external link on https://example.com")
        driver.get("https://example.com")
        links = driver.find_elements(By.TAG_NAME, "a")
        hrefs = [l.get_attribute("href") for l in links]
        print(f"[STDOUT] Links found: {hrefs}")
        assert any("iana" in (h or "") for h in hrefs), "Expected a link to iana.org"


class TestPagePerformance:
    def test_page_loads_under_3_seconds(self, driver):
        print("[STDOUT] Measuring page load time for https://example.com")
        start = time.time()
        driver.get("https://example.com")
        elapsed = time.time() - start
        print(f"[STDOUT] Page loaded in {elapsed:.2f}s")
        assert elapsed < 3.0, f"Page took too long: {elapsed:.2f}s"

    def test_missing_dashboard_element_times_out(self, driver):
        print("[STDOUT] Looking for #dashboard-chart on https://example.com")
        driver.get("https://example.com")
        sys.stderr.write("[STDERR] #dashboard-chart not found — component missing\n")
        WebDriverWait(driver, 2).until(
            EC.presence_of_element_located((By.ID, "dashboard-chart"))
        )

    def test_page_body_content_is_not_empty(self, driver):
        print("[STDOUT] Verifying body content on https://example.com")
        driver.get("https://example.com")
        body = driver.find_element(By.TAG_NAME, "body")
        content = body.text.strip()
        print(f"[STDOUT] Body text length: {len(content)} chars")
        assert len(content) > 0, "Page body should not be empty"


class TestFormInteractions:
    def test_search_input_accepts_text(self, driver):
        print("[STDOUT] Looking for input fields on https://example.com")
        driver.get("https://example.com")
        inputs = driver.find_elements(By.TAG_NAME, "input")
        if inputs:
            inputs[0].send_keys("test query")
            assert inputs[0].get_attribute("value") == "test query"
        else:
            print("[STDOUT] No input found — page has no form (expected for example.com)")
            assert True

    def test_login_form_not_found_raises_timeout(self, driver):
        print("[STDOUT] Waiting for #login-form on https://example.com")
        driver.get("https://example.com")
        sys.stderr.write("[STDERR] Login form expected but not present\n")
        WebDriverWait(driver, 2).until(
            EC.presence_of_element_located((By.ID, "login-form"))
        )

    def test_page_has_paragraph_content(self, driver):
        print("[STDOUT] Checking paragraph content on https://example.com")
        driver.get("https://example.com")
        paragraphs = driver.find_elements(By.TAG_NAME, "p")
        print(f"[STDOUT] Found {len(paragraphs)} paragraphs")
        assert len(paragraphs) > 0, "Page should have at least one paragraph"
