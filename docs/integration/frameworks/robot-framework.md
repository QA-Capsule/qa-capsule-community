---
icon: material/robot
---

# Robot Framework

| | |
|---|---|
| **Upload param** | `?framework=RobotFramework` |
| **Report** | `tests/results/robot-junit.xml` |
| **Repo workflow** | `.github/workflows/e2e-tests-robot.yml` |
| **Secret** | `QA_CAPSULE_API_ROBOT_KEY` |

## Test suites in this repository

| File | Result in CI |
|------|----------------|
| `smoke_tests.robot` | Pass |
| `api_health.robot` | Pass |
| `demo_failure.robot` | Fail (demo alert) |
| `ui_navigation.robot` | Pass (Selenium + Chrome) |

## 1. Generate JUnit XML

```bash
pip install -r tests/requirements.txt
robot --outputdir tests/results --xunit robot-junit.xml tests/robotframework/
```

!!! warning "`--xunit` path"
    Use **basename only** (`robot-junit.xml`), not `tests/results/robot-junit.xml` — Robot resolves paths relative to `--outputdir`.

## 2. Upload to QA Capsule

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=RobotFramework" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@tests/results/robot-junit.xml"
```

## 3. GitHub Actions

```yaml
- name: Run Robot Tests
  run: robot --outputdir tests/results --xunit robot-junit.xml tests/robotframework/
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_ROBOT_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=RobotFramework" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@tests/results/robot-junit.xml"
```

!!! note "Headers"
    - `X-Run-Id` groups all test results under the same pipeline run in the Execution Hub.
    - `X-Execution-Env` accepts `PROD`, `STAGING`, `INTEGRATION`, `DEV`.
    - `X-Execution-Type` accepts `TEST-RUN`, `NIGHTLY`, `SMOKE`, `REAL`.

← [All frameworks](../test-frameworks.md)
