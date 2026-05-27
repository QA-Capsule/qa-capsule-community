---
icon: material/web-clock
---

# Selenium

Selenium is a **driver library**, not a reporter. Pair it with a runner that exports JUnit XML.

| Stack | Runner | Upload param |
|-------|--------|--------------|
| Java + JUnit 5 | JUnit Platform Console / Maven Surefire | `JUnit` |
| Python + Pytest | Pytest | `Pytest` |
| BDD | Cucumber + Selenium | `Cucumber` |
| Robot | SeleniumLibrary | `RobotFramework` |
| Mobile | Appium | `Appium` |

---

## Python + Pytest (this repository)

| | |
|---|---|
| **Upload param** | `?framework=Pytest` |
| **Report** | `selenium-results.xml` |
| **Repo workflow** | `.github/workflows/e2e-tests-selenium-pytest.yml` |
| **Test folder** | `tests/selenium-pytest/` |
| **Secret** | `QA_CAPSULE_API_SELENIUM_KEY` |

### Test suites in this repository

| Class | Tests | Expected result |
|---|---|---|
| `TestPageNavigation` | homepage title, h1 visible, link to iana | 3 pass |
| `TestPagePerformance` | load time, missing element timeout, body content | 2 pass · 1 fail |
| `TestFormInteractions` | search input, login form timeout, paragraph content | 2 pass · 1 fail |

### Install

```bash
pip install pytest selenium webdriver-manager
```

### Run

```bash
pytest tests/selenium-pytest/ -v --junitxml=selenium-results.xml
```

### GitHub Actions

```yaml
- name: Setup Chrome
  uses: browser-actions/setup-chrome@v1

- name: Install Dependencies
  run: pip install -r tests/selenium-pytest/requirements.txt

- name: Run Selenium Tests
  run: pytest tests/selenium-pytest/ -v --junitxml=selenium-results.xml
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_SELENIUM_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=Pytest" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@selenium-results.xml"
```

---

## Java (Maven + Surefire)

```xml
<dependency>
  <groupId>org.seleniumhq.selenium</groupId>
  <artifactId>selenium-java</artifactId>
  <version>4.20.0</version>
</dependency>
```

```bash
mvn test
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=JUnit" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${BUILD_NUMBER}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@target/surefire-reports/TEST-com.example.UITest.xml"
```

---

## Robot Framework + SeleniumLibrary

```bash
export SELENIUM_ENABLED=true
export SELENIUM_BROWSER=headlesschrome
robot --outputdir tests/results --xunit robot-junit.xml tests/robotframework/
```

See [Robot Framework](robot-framework.md) for the full pipeline.

← [All frameworks](../test-frameworks.md)
