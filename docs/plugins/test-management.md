# Test management plugins (TestRail, Zephyr, Xray)

## TestRail

| Variable | Description |
|----------|-------------|
| `TESTRAIL_URL` | Instance URL (e.g. `https://company.testrail.io`) |
| `TESTRAIL_USER` | API user email |
| `TESTRAIL_API_KEY` | API key from TestRail profile |
| `TESTRAIL_RUN_ID` | Active test run ID |
| `TESTRAIL_CASE_ID` | Case ID to mark failed |

Per-gateway overrides: set `TESTRAIL_RUN_ID` / `TESTRAIL_CASE_ID` in CI/CD Gateway routing fields (injected as env vars).

## Zephyr Scale

| Variable | Description |
|----------|-------------|
| `ZEPHYR_API_TOKEN` | Bearer token from Zephyr Scale API keys |
| `ZEPHYR_PROJECT_KEY` | Jira project key |
| `ZEPHYR_TEST_CASE_KEY` | Test case key (e.g. `PROJ-T42`) |

## Xray Cloud

| Variable | Description |
|----------|-------------|
| `XRAY_CLIENT_ID` | OAuth client ID |
| `XRAY_CLIENT_SECRET` | OAuth client secret |
| `XRAY_TEST_KEY` | Test or execution issue key |
| `XRAY_PROJECT_KEY` | Jira project key (optional) |

Configure each plugin, click **Execute** to validate, then set `status: Active` for auto-trigger.
