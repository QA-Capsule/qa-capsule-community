---
icon: material/language-python
---

# Pytest

| | |
|---|---|
| **Upload param** | `?framework=Pytest` |
| **Report** | `pytest-results.xml` |
| **Repo workflow** | `.github/workflows/api-tests-pytest.yml` |
| **Secret** | `QA_CAPSULE_API_PYTEST_KEY` |

## Test suites in this repository

| Class | Tests | Expected result |
|---|---|---|
| `TestAuthenticationSuite` | login valid, login invalid password, token expiry | 1 pass · 2 fail |
| `TestAPIValidationSuite` | create user 201, missing field 422, rate limit 429 | 1 pass · 2 fail |
| `TestPerformanceSuite` | response time, DB query timeout, cache hit | 2 pass · 1 error |

## 1. Generate JUnit XML

```bash
pip install pytest
pytest tests/ -v --junitxml=pytest-results.xml
```

`pytest.ini` (recommended):

```ini
[pytest]
junit_family = xunit2
```

## 2. Upload to QA Capsule

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Pytest" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@pytest-results.xml"
```

## 3. GitHub Actions

```yaml
- name: Run Pytest
  run: pytest test_suite.py -v --junitxml=pytest-results.xml
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_PYTEST_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=Pytest" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@pytest-results.xml"
```

!!! note "Headers"
    - `X-Run-Id` groups all test results under the same pipeline run in the Execution Hub.
    - `X-Execution-Env` accepts `PROD`, `STAGING`, `INTEGRATION`, `DEV`.
    - `X-Execution-Type` accepts `TEST-RUN`, `NIGHTLY`, `SMOKE`, `REAL`.

## With Selenium

Use `pytest-selenium` or plain `selenium`; keep the same `--junitxml` flag and `?framework=Pytest` upload param.

← [All frameworks](../test-frameworks.md)
