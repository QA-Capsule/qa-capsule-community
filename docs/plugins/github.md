---
icon: fontawesome/brands/github
---

# GitHub Actions

<div align="center" class="integration-hero">
  <img src="../assets/integrations/github.png" alt="GitHub logo">
</div>

Triggers a GitHub workflow via `POST /repos/{owner}/{repo}/actions/workflows/{id}/dispatches`.

| | |
|---|---|
| **Manifest** | `plugins/github/github-rerun.json` |
| **Type** | `github` |

---

=== "QA Capsule Side"

    | Variable | Required | Gateway |
    |----------|-------------|---------|
    | `GITHUB_TOKEN` | **Yes** | Fine-grained or classic PAT |
    | `GITHUB_OWNER` | **Yes** | Owner |
    | `GITHUB_REPO` | **Yes** | Repository |
    | `GITHUB_WORKFLOW_ID` | **Yes** | Numeric ID or filename `ci.yml` |
    | `GITHUB_REF` | No | Branch (default `main`) |

    Dispatch inputs: `triggered_by: qa-capsule`, `incident: <summary>`.

    Success: HTTP **204**.

=== "Provider Side (GitHub)"

    ## 1. Personal Access Token

    1. GitHub → **Settings** → **Developer settings** → **Fine-grained tokens** (recommended)
    2. Repository access: target repo only
    3. Permissions: **Actions: Read and write**, **Contents: Read** (depending on workflow)

    ## 2. Dispatchable workflow

    The target workflow must have:

    ```yaml
    on:
      workflow_dispatch:
        inputs:
          triggered_by:
            required: false
          incident:
            required: false
    ```

    ## 3. Find WORKFLOW_ID

    - Workflow Actions URL → ID in the API
    - Or use the filename: `ci.yml`

    ## 4. Test

    ```bash
    curl -X POST -H "Authorization: Bearer TOKEN" \
      -H "Accept: application/vnd.github+json" \
      https://api.github.com/repos/OWNER/REPO/actions/workflows/WORKFLOW_ID/dispatches \
      -d '{"ref":"main","inputs":{"triggered_by":"test"}}'
    ```

---

- [Catalog](integrations-catalog.md)
