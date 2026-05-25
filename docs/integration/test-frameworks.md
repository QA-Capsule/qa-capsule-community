---
icon: material/test-tube
---

# All test frameworks — catalog

QA Capsule ingests **any runner that produces JUnit XML** (recommended) or **JSON** via webhooks. This catalog lists **every supported stack** with the same integration pattern.

| Method | Endpoint | When to use |
|--------|----------|-------------|
| **JUnit upload** | `POST /api/webhooks/upload?framework={Name}` | Playwright, Cypress, Pytest, Robot, Newman, Surefire, NUnit, Jest, Go, PHPUnit, … |
| **JSON webhook** | `POST /api/webhooks/` | K6 thresholds, custom scripts, pipeline-level alerts |
| **Playwright reporter** | Real-time per test | Traces + live failures — [Playwright Reporter](playwright-reporter.md) |

---

## Master reference table

| Framework | Guide | `framework` param | Report file |
|-----------|-------|-------------------|-------------|
| **Playwright** | [Playwright](frameworks/playwright.md) | `Playwright` | `playwright-results.xml` |
| **Cypress** | [Cypress](frameworks/cypress.md) | `Cypress` | `cypress-results.xml` |
| **Pytest** | [Pytest](frameworks/pytest.md) | `Pytest` | `pytest-results.xml` |
| **Robot Framework** | [Robot Framework](frameworks/robot-framework.md) | `RobotFramework` | `robot-junit.xml` |
| **Selenium (Java)** | [Selenium](frameworks/selenium.md) | `JUnit` | Surefire `TEST-*.xml` |
| **Selenium (Python)** | [Selenium](frameworks/selenium.md) | `Pytest` | `pytest-results.xml` |
| **Postman / Newman** | [Newman](frameworks/postman-newman.md) | `Postman` | `newman-results.xml` |
| **JUnit 5 / Maven / Gradle** | [JUnit (Java)](frameworks/junit-java.md) | `JUnit` | `surefire-reports/*.xml` |
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
| **Gatling** | [K6](frameworks/k6.md#gatling) | JSON | N/A |

**Example workflows in this repo:** `e2e-tests-robot.yml`, `e2e-tests-playwright.yml`, `e2e-tests-cypress.yml`, `api-tests-pytest.yml`

---

## Universal upload (all JUnit-based frameworks)

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@playwright-results.xml"
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

## Secrets (per pipeline or shared)

| Secret | Framework |
|--------|-----------|
| `QA_CAPSULE_URL` | All (base URL, no trailing slash) |
| `QA_CAPSULE_API_KEY` | Shared |
| `QA_CAPSULE_API_PLAYWRIGHT_KEY` | Playwright |
| `QA_CAPSULE_API_CYPRESS_KEY` | Cypress |
| `QA_CAPSULE_API_PYTEST_KEY` | Pytest |
| `QA_CAPSULE_API_ROBOT_KEY` | Robot Framework |

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
| Upload OK, nothing in UI | XML must contain `<testcase>`; Robot nested suites need current QA Capsule |
| 401 | Re-copy API key from CI/CD Gateways |
| 404 | Base URL without duplicate `/api/webhooks/...` |
| Upload skipped on fail | `if: always()` / `after_script` |

---

## Related

- [JUnit XML Upload API](junit-xml-upload.md)
- [CI/CD Overview](cicd-overview.md)
- [Webhooks API](../api/webhooks.md)
