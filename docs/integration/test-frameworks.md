---
icon: material/test-tube
---

# All test frameworks — catalog

QA Capsule ingests **any runner that produces JUnit XML** (recommended) or **JSON** via webhooks.

| Method | Endpoint | When to use |
|--------|----------|-------------|
| **JUnit upload** | `POST /api/webhooks/upload?framework={Name}` | Playwright, Cypress, Pytest, Robot, Newman, JUnit, NUnit, Jest, Go, PHPUnit, … |
| **JSON webhook** | `POST /api/webhooks/` | K6 thresholds, custom scripts, pipeline-level alerts |
| **Playwright reporter** | Real-time per test | Traces + live failures — [Playwright Reporter](playwright-reporter.md) |

---

## Pipelines in this repository

| Pipeline file | Workflow name | Framework | Test folder |
|---|---|---|---|
| `api-tests-pytest.yml` | `API Tests \| Pytest` | `Pytest` | `tests/pytest/` |
| `api-tests-newman.yml` | `API Tests \| Newman (Postman)` | `Postman` | `tests/newman/` |
| `api-tests-junit-java.yml` | `API Tests \| JUnit Java` | `JUnit` | `tests/junit-java/src/` |
| `e2e-tests-cypress.yml` | `E2E Tests \| Cypress` | `Cypress` | `tests/cypress/` |
| `e2e-tests-playwright.yml` | `E2E Tests \| Playwright` | `Playwright` | `tests/playwright/` |
| `e2e-tests-robot.yml` | `E2E Tests \| Robot Framework` | `RobotFramework` | `tests/robotframework/` |
| `e2e-tests-selenium-pytest.yml` | `E2E Tests \| Selenium + Pytest` | `Pytest` | `tests/selenium-pytest/` |

**Naming pattern:** `{Scope} Tests | {Framework}` — scope is `API` or `E2E`.

---

## Master reference table

| Framework | Guide | `framework` param | Report file |
|-----------|-------|-------------------|-------------|
| **Playwright** | [Playwright](frameworks/playwright.md) | `Playwright` | `playwright-results.xml` |
| **Cypress** | [Cypress](frameworks/cypress.md) | `Cypress` | `cypress-results.xml` |
| **Pytest** | [Pytest](frameworks/pytest.md) | `Pytest` | `pytest-results.xml` |
| **Robot Framework** | [Robot Framework](frameworks/robot-framework.md) | `RobotFramework` | `robot-junit.xml` |
| **Selenium (Python)** | [Selenium](frameworks/selenium.md) | `Pytest` | `selenium-results.xml` |
| **Selenium (Java)** | [Selenium](frameworks/selenium.md) | `JUnit` | Surefire `TEST-*.xml` |
| **Postman / Newman** | [Newman](frameworks/postman-newman.md) | `Postman` | `newman-results.xml` |
| **JUnit 5 / Java** | [JUnit (Java)](frameworks/junit-java.md) | `JUnit` | `TEST-junit-jupiter.xml` |
| **TestNG** | [TestNG](frameworks/testng.md) | `JUnit` | `testng-results.xml` |
| **Jest** | [Jest / Mocha](frameworks/jest-mocha.md) | `Jest` | `junit.xml` |
| **Mocha** | [Jest / Mocha](frameworks/jest-mocha.md) | `Mocha` | `junit.xml` |
| **NUnit** | [.NET tests](frameworks/dotnet.md) | `NUnit` | `TestResults.xml` |
| **xUnit** | [.NET tests](frameworks/dotnet.md) | `xUnit` | `TestResults.xml` |
| **Go `testing`** | [Go](frameworks/golang.md) | `Go` | `report.xml` |
| **PHPUnit** | [PHPUnit](frameworks/phpunit.md) | `PHPUnit` | `junit.xml` |
| **RSpec** | [RSpec](frameworks/rspec.md) | `RSpec` | `rspec.xml` |
| **Karate** | [Karate](frameworks/karate.md) | `Karate` | `karate-reports/*.xml` |
| **REST Assured** | [REST Assured](frameworks/rest-assured.md) | `JUnit` | Surefire XML |
| **Cucumber** | [Cucumber](frameworks/cucumber.md) | `Cucumber` | `cucumber.xml` |
| **Appium** | [Appium](frameworks/appium.md) | `Appium` | JUnit from driver |
| **K6** | [K6](frameworks/k6.md) | — (JSON) | N/A |

---

## Universal upload pattern (all JUnit-based frameworks)

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@results.xml"
```

| Header | Purpose |
|--------|---------|
| `X-API-Key` | Project key from **CI/CD Gateways** |
| `X-Run-Id` | `github.run_id`, `$CI_PIPELINE_ID`, `$BUILD_NUMBER`, … |
| `X-Execution-Env` | `PROD`, `STAGING`, `DEV`, `INTEGRATION` |
| `X-Execution-Type` | `TEST-RUN`, `SMOKE`, `NIGHTLY`, `REAL` |

!!! tip "Always upload after tests"
    GitHub: `if: always()` · GitLab: `after_script` · Jenkins: `post { always { ... } }`

---

## Secrets reference

| Secret | Used by |
|--------|---------|
| `QA_CAPSULE_URL` | All pipelines (base URL, no trailing slash) |
| `QA_CAPSULE_API_PYTEST_KEY` | `api-tests-pytest.yml` |
| `QA_CAPSULE_API_NEWMAN_KEY` | `api-tests-newman.yml` |
| `QA_CAPSULE_API_JUNIT_JAVA_KEY` | `api-tests-junit-java.yml` |
| `QA_CAPSULE_API_CYPRESS_KEY` | `e2e-tests-cypress.yml` |
| `QA_CAPSULE_API_PLAYWRIGHT_KEY` | `e2e-tests-playwright.yml` |
| `QA_CAPSULE_API_ROBOT_KEY` | `e2e-tests-robot.yml` |
| `QA_CAPSULE_API_SELENIUM_KEY` | `e2e-tests-selenium-pytest.yml` |

---

## CI/CD platforms

Platform-specific steps: [CI/CD providers](cicd-providers.md) (GitHub, GitLab, Jenkins, Azure DevOps, CircleCI, Bitbucket, …).

---

## Where results appear

1. **Operations → Telemetry Stream** — group by `X-Run-Id`
2. **Dashboard** — incidents per failed test
3. **View Full Report** — pass / fail / skip matrix

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Upload OK, nothing in UI | XML must contain `<testcase>` elements |
| 401 Unauthorized | Re-copy API key from CI/CD Gateways |
| 404 Not Found | Base URL must not include `/api/webhooks/...` |
| Upload skipped on failure | Add `if: always()` / `after_script` |
| Cypress result not found | Use `working-directory` + full path in `curl` |

---

## Related

- [JUnit XML Upload API](junit-xml-upload.md)
- [CI/CD Overview](cicd-overview.md)
- [Webhooks API](../api/webhooks.md)
