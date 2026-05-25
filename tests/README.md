# Automated tests (Robot Framework)

Sample suite runnable locally or in CI/CD, with optional JUnit XML upload to **QA Capsule**.

**Docs:** [Test frameworks catalog](https://qa-capsule.github.io/qa-capsule-community/integration/test-frameworks/) · [CI/CD providers](https://qa-capsule.github.io/qa-capsule-community/integration/cicd-providers/)

## Prerequisites

- Python 3.10+
- `bash` (Git Bash on Windows, or Linux/macOS)
- For `ui_navigation.robot`: Chrome/Chromium + WebDriver and `SELENIUM_ENABLED=true`

## Layout

```
tests/
├── requirements.txt
├── robotframework/
│   ├── resources/common.robot
│   ├── smoke_tests.robot
│   ├── api_health.robot
│   ├── demo_failure.robot   # intentional failure (QA Capsule alert in CI)
│   └── ui_navigation.robot
└── results/          # generated (gitignored)
```

## Run locally

From the repository root:

```bash
chmod +x scripts/run-tests.sh
./scripts/run-tests.sh
```

Without QA Capsule env vars, only Robot tests run; upload is skipped.

### Upload to QA Capsule

```bash
export QA_CAPSULE_URL="http://localhost:9000"
export QA_CAPSULE_API_KEY="your-project-api-key"
export QA_CAPSULE_EXEC_ENV="DEV"
export QA_CAPSULE_EXEC_TYPE="TEST-RUN"

./scripts/run-tests.sh
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

Entry point: **`scripts/run-tests.sh`** — installs deps, runs Robot, converts to JUnit, uploads when secrets are set.

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

Workflow: [`.github/workflows/e2e-tests-robot.yml`](../.github/workflows/e2e-tests-robot.yml).  
Quarantine gate: `scripts/quarantine-ci-gate.sh` (when URL + API key are set).

Add secrets:

- `QA_CAPSULE_URL`
- `QA_CAPSULE_API_ROBOT_KEY` (or `QA_CAPSULE_API_KEY`)

Run **Actions → Robot Framework Pipeline → Run workflow**.

Suites executed: `smoke_tests.robot`, `api_health.robot`, `demo_failure.robot` (intentional fail), `ui_navigation.robot`. `resources/common.robot` is a shared resource file only.
