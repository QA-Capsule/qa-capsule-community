---
icon: material/api
---

# Postman / Newman

| | |
|---|---|
| **Upload param** | `?framework=Postman` |
| **Report** | `newman-results.xml` |
| **Repo workflow** | `.github/workflows/api-tests-newman.yml` |
| **Test folder** | `tests/newman/` |
| **Secret** | `QA_CAPSULE_API_NEWMAN_KEY` |

## Test suites in this repository

Collection: `tests/newman/collection.json` — 3 folders × 3 requests against `jsonplaceholder.typicode.com`.

| Folder | Requests | Expected result |
|---|---|---|
| `Users API` | GET all users, GET by ID, GET non-existent (404) | 2 pass · 1 fail |
| `Posts API` | POST create, PUT update, DELETE | 3 pass |
| `Todos API` | GET list, GET completed filter, GET response time | 2 pass · 1 fail |

## 1. Install Newman

```bash
npm install -g newman newman-reporter-junitfull
```

!!! note "Reporter"
    Use `newman-reporter-junitfull` for richer JUnit XML output (includes request/response details per test).
    The built-in `junit` reporter produces a simpler format.

## 2. Run tests

```bash
newman run tests/newman/collection.json \
  --reporters cli,junitfull \
  --reporter-junitfull-export newman-results.xml
```

With an environment file:

```bash
newman run tests/newman/collection.json \
  -e staging.postman_environment.json \
  --reporters cli,junitfull \
  --reporter-junitfull-export newman-results.xml
```

## 3. Upload to QA Capsule

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Postman" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@newman-results.xml"
```

## 4. GitHub Actions

```yaml
- name: Install Newman
  run: npm install -g newman newman-reporter-junitfull

- name: Run Newman Tests
  run: |
    newman run tests/newman/collection.json \
      --reporters cli,junitfull \
      --reporter-junitfull-export newman-results.xml
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_NEWMAN_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=Postman" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@newman-results.xml"
```

!!! note "Headers"
    - `X-Run-Id` groups all test results under the same pipeline run in the Execution Hub.
    - `X-Execution-Env` accepts `PROD`, `STAGING`, `INTEGRATION`, `DEV`.
    - `X-Execution-Type` accepts `TEST-RUN`, `NIGHTLY`, `SMOKE`, `REAL`.

← [All frameworks](../test-frameworks.md)
