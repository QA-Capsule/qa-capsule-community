---
icon: material/robot
---

# Robot Framework

| | |
|---|---|
| **Upload param** | `?framework=RobotFramework` |
| **Canonical upload report** | `tests/results/robot-junit.xml` |
| **Repo workflow** | `.github/workflows/e2e-tests-robot.yml` |
| **Runner script** | `scripts/run-tests.sh` |

## Suites in this repository

| File | Result in CI |
|------|----------------|
| `smoke_tests.robot` | Pass |
| `api_health.robot` | Pass |
| `demo_failure.robot` | Fail (demo alert) |
| `ui_navigation.robot` | Pass (Selenium + Chrome) |

## 1. Local run

```bash
export QA_CAPSULE_URL="https://qa-capsule.example.com"
export QA_CAPSULE_API_KEY="your-key"
export SELENIUM_ENABLED=true
chmod +x scripts/run-tests.sh && ./scripts/run-tests.sh
```

## 2. JUnit (rebot)

```bash
chmod +x scripts/run-tests.sh scripts/quarantine-ci-gate.sh
export QA_CAPSULE_URL=https://your-capsule.example.com
export QA_CAPSULE_API_KEY=your-project-key
./scripts/run-tests.sh
```

!!! warning "`--xunit` path"
    Use **basename only** (`robot-junit.xml`), not `tests/results/robot-junit.xml` — rebot resolves paths relative to `--outputdir`.

!!! note "Canonical file only (no fallback discovery)"
    CI upload must target only `tests/results/robot-junit.xml`. Do not fallback to `find ... robot-junit.xml` paths, because stale or duplicated files can introduce repeated testcases.

## 3. Upload

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=RobotFramework" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${GITHUB_RUN_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@tests/results/robot-junit.xml"
```

## 4. GitHub Actions

Secrets: `QA_CAPSULE_URL`, `QA_CAPSULE_API_ROBOT_KEY`

Uses `scripts/run-tests.sh` + dedicated upload step with `if: always()`.

← [All frameworks](../test-frameworks.md)
