# Automated tests (multi-framework)

**QA Capsule Community v1.0.17-beta** — sample projects runnable locally or in CI/CD, with optional JUnit XML upload to **QA Capsule**.

**Docs:** [Test frameworks catalog](https://qa-capsule.github.io/qa-capsule-community/integration/test-frameworks/) · [CI/CD providers](https://qa-capsule.github.io/qa-capsule-community/integration/cicd-providers/)

## Prerequisites

- Python 3.10+
- `bash` (Git Bash on Windows, or Linux/macOS)
- For `ui_navigation.robot`: Chrome/Chromium + WebDriver and `SELENIUM_ENABLED=true`

## Layout

```
tests/
├── run-framework.sh
├── upload-junit.sh
├── robotframework/        # full Robot project + run.sh
├── playwright/            # full Playwright project + run.sh
├── cypress/               # full Cypress project + run.sh
├── pytest/                # full Pytest project + run.sh
├── selenium-pytest/       # Selenium + Pytest sample + run.sh
├── newman/                # Postman/Newman sample + run.sh
└── junit-java/            # JUnit XML sample + run.sh
```

## Run locally

From the repository root:

```bash
chmod +x tests/run-framework.sh tests/*/run.sh tests/upload-junit.sh
./tests/run-framework.sh robot
./tests/run-framework.sh playwright
./tests/run-framework.sh cypress
./tests/run-framework.sh pytest
./tests/run-framework.sh selenium-py
./tests/run-framework.sh newman
./tests/run-framework.sh junit-java
```

Without QA Capsule env vars, tests run and upload is skipped.

### Upload to QA Capsule

```bash
export QA_CAPSULE_URL="http://localhost:9000"
export QA_CAPSULE_API_KEY="your-project-api-key"
export QA_CAPSULE_EXEC_ENV="DEV"
export QA_CAPSULE_EXEC_TYPE="TEST-RUN"

./tests/run-framework.sh robot
```

### API target (optional)

By default `api_health.robot` uses `jsonplaceholder.typicode.com`. For your API:

```bash
export API_HEALTH_HOST=api.example.com
./scripts/run-tests.sh
```

### UI tests (optional)

```bash
export SELENIUM_ENABLED=true
export SELENIUM_BROWSER=headlesschrome
./scripts/run-tests.sh
```

## CI/CD

Entry point can now be `tests/<framework>/run.sh` (framework-specific) or `tests/run-framework.sh`.

### QA Capsule requirements

1. Instance reachable from the runner (`localhost` only works on self-hosted runners).
2. Copy the project **API key** from **CI/CD Gateways**.
3. Upload URL: `{QA_CAPSULE_URL}/api/webhooks/upload?framework=RobotFramework`

### Pipeline variables

| Variable | Required | Example |
|----------|----------|---------|
| `QA_CAPSULE_URL` | For upload | `https://qa-capsule.example.com` |
| `QA_CAPSULE_API_KEY` | For upload | project key |
| `CI_PIPELINE_ID` | Recommended | job id (`X-Run-Id`) |
| `QA_CAPSULE_EXEC_ENV` | Optional | `STAGING`, `PROD`, `DEV` |
| `QA_CAPSULE_EXEC_TYPE` | Optional | `TEST-RUN`, `SMOKE`, `NIGHTLY` |
| `SELENIUM_ENABLED` | Optional | `true` in GitHub workflow |

Without `QA_CAPSULE_*`, tests still run; upload is skipped.

### GitHub Actions

All framework workflows are **manual** (`workflow_dispatch`) so they do not break CI on every push. They include intentional failures for self-healing demos.

| Workflow | Framework |
|----------|-----------|
| `e2e-tests-robot.yml` | Robot Framework |
| `e2e-tests-playwright.yml` | Playwright |
| `e2e-tests-cypress.yml` | Cypress |
| `e2e-tests-selenium-pytest.yml` | Selenium + Pytest |
| `api-tests-pytest.yml` | Pytest API |
| `api-tests-newman.yml` | Postman/Newman |
| `api-tests-junit-java.yml` | JUnit Java |

Shared upload scripts: [`.github/scripts/`](../.github/scripts/) (`upload-junit-to-qacapsule.sh`, `healing-gate.sh`).

Quarantine gate: `scripts/quarantine-ci-gate.sh` (when URL + API key are set).

Add secrets:

- `QA_CAPSULE_URL`
- `QA_CAPSULE_API_ROBOT_KEY` (or `QA_CAPSULE_API_KEY`)

Run **Actions → Robot Framework Pipeline → Run workflow**.

Suites executed: `smoke_tests.robot`, `api_health.robot`, `demo_failure.robot` (intentional fail), `ui_navigation.robot`. `resources/common.robot` is a shared resource file only.
