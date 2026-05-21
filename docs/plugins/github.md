# GitHub Actions plugin

Dispatches a workflow using the GitHub REST API (`workflow_dispatch`).

## Configuration

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | PAT or GitHub App token with `actions:write` |
| `GITHUB_OWNER` | Org or user |
| `GITHUB_REPO` | Repository name |
| `GITHUB_WORKFLOW_ID` | Workflow file name or numeric ID |
| `GITHUB_REF` | Branch or tag (default `main`) |

Use for re-running E2E suites or smoke tests after QA Capsule detects a critical failure.
