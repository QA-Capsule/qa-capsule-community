---
icon: fontawesome/brands/github
---

# GitHub Actions Integration

Integrating QA Capsule with GitHub Actions allows you to instantly capture test failures from your Pull Requests, nightly builds, and deployment pipelines. 

By pushing this telemetry to your QA Capsule Control Plane, your engineering team can diagnose flaky tests and pipeline crashes without ever having to dig through raw GitHub Actions logs.

---

## Prerequisites

Before proceeding, ensure you have the following:

1. **QA Capsule Access:** You must have the `Operator` or `Administrator` role to provision a new Webhook endpoint.
2. **GitHub Access:** You must have `Admin` or `Maintainer` permissions on the GitHub repository to manage Action Secrets.
3. **A Test Pipeline:** An existing `.github/workflows/*.yml` file that runs your automated tests.

---

## Step 1: Identify your Workflow ID

QA Capsule tracks failures on a per-pipeline basis. To do this, it needs to know the exact identifier of the GitHub workflow it is monitoring.

In GitHub Actions, the **Workflow ID** is simply the filename of the YAML file located in your repository's `.github/workflows/` directory.

**How to find it:**

1. Open your repository on GitHub.
2. Navigate to the `.github/workflows/` folder.
3. Look at the filename of the pipeline you want to monitor (e.g., `playwright-e2e.yml`, `nightly-build.yaml`, or `ci.yml`).
4. **Your Workflow ID is exactly that filename** (including the `.yml` or `.yaml` extension).

*Note: QA Capsule uses this ID for deep-linking. In future updates, this will allow the QA Capsule dashboard to link you directly to the exact failing GitHub run.*

---

## Step 2: Provision the Endpoint in QA Capsule

Now that you know your Workflow ID, you must create a dedicated, secure entry point for it in the QA Capsule database.

1. Log in to your **QA Capsule Dashboard**.
2. Navigate to the **CI/CD Gateways** module using the left-hand sidebar.
3. Click on the **GitHub Actions** provider card. The form will dynamically update.
4. **Fill out the Provisioning Form:**

   * **Project Name:** Give it a human-readable name (e.g., `Frontend Web App - E2E`).
   * **Routing Variables:** Define where alerts for this specific project should go if they fail (e.g., `SLACK_CHANNEL: #alerts-frontend`, `JIRA_PROJECT_KEY: WEB`).
   * **GitHub Action Workflow ID:** Paste the filename you identified in Step 1 (e.g., `playwright-e2e.yml`).

5. Click **Provision Project Endpoint**.

**CRITICAL:** The system will generate a **Webhook URL** and a **Generated API Key (Secret)**. Copy both of these values immediately. For security reasons, the raw API Key will never be displayed in the UI again.

---

## Step 3: Configure GitHub Repository Secrets

You must never hardcode your QA Capsule API Key directly into your workflow YAML files. Instead, we will use GitHub's secure secret manager.

1. Open your repository on GitHub.
2. Click on the **Settings** tab.
3. In the left sidebar, scroll down to the **Security** section, expand **Secrets and variables**, and click on **Actions**.
4. Click the green **New repository secret** button.
5. Add the Webhook URL:

   * **Name:** `QA_CAPSULE_URL`
   * **Secret:** Paste the generated URL (e.g., `https://sre.yourcompany.com/api/webhooks/github`). *Do not add a trailing slash.*
   * Click **Add secret**.

6. Click **New repository secret** again to add the API Key:

   * **Name:** `QA_CAPSULE_API_KEY`
   * **Secret:** Paste the generated API Key (e.g., `sre_pk_github_a1b2c3d4e5f6`).
   * Click **Add secret**.

---

## Step 4: Update your Workflow YAML

The final step is to instruct your GitHub Actions runner to push data to QA Capsule when a test fails.

Open your workflow file (e.g., `.github/workflows/playwright-e2e.yml`) and add a new step at the very end of your job. 

### The `if: always()` condition

It is absolutely critical that you add `if: always()` or `if: failure()` to this step. By default, GitHub stops executing subsequent steps if a previous step (like a test) fails. Using this condition ensures the telemetry agent runs regardless of the pipeline's failure status.

### Example Configuration:

```yaml
name: Frontend E2E Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  run-playwright:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: 20

      - name: Install Dependencies
        run: npm ci

      - name: Install Playwright Browsers
        run: npx playwright install --with-deps

      - name: Run E2E Tests
        run: npx playwright test
        continue-on-error: true # Optional: Allows pipeline to reach the agent step gracefully

      # ========================================================================
      # QA CAPSULE TELEMETRY AGENT
      # ========================================================================
      - name: Push Telemetry to QA Capsule
        if: always() 
        env:
          # Securely inject the secrets from GitHub environment
          WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
          API_KEY: ${{ secrets.QA_CAPSULE_API_KEY }}
        run: |
          echo "Sending test results to QA Capsule Control Plane..."
          
          # In a production environment, you would use the 'sre-agent.sh' 
          # to parse your JUnit XML here. For demonstration, we send a static payload:
          
          curl --silent --show-error -X POST "$WEBHOOK_URL/api/webhooks/github" \
            -H "Content-Type: application/json" \
            -H "X-API-Key: $API_KEY" \
            -d "{
                  \"name\": \"GitHub Actions: ${GITHUB_WORKFLOW}\",
                  \"status\": \"CRITICAL\",
                  \"browser\": \"Ubuntu / GitHub Runner\",
                  \"error\": \"Test suite failed on branch ${GITHUB_REF_NAME}\",
                  \"console_logs\": \"[FATAL] E2E Tests failed during CI execution. See GitHub run ID: ${GITHUB_RUN_ID} for full artifacts.\"
                }"
```

## Step 5: Verify the Integration

To ensure the connection is established:

1. Commit and push the changes to your .github/workflows/*.yml file.
2. Intentionally cause a test to fail (e.g., by changing an assertion from true to false).
3. Go to the Actions tab in GitHub and watch the run.
4. Expand the Push Telemetry to QA Capsule step in the GitHub logs. You should see no curl errors.
5. Open your QA Capsule Dashboard. You should see the new incident appear instantly in the Active Incidents log.

### Troubleshooting

1. `Error 401 Unauthorized:` The `X-API-Key` sent by GitHub does not match the database. Ensure you copied the secret correctly into GitHub without any hidden spaces.

2. `Error 404 Not Found:` Ensure your `QA_CAPSULE_URL` secret ends exactly with `/api/webhooks/github` and does not have a trailing slash.

3. `No alert on Dashboard:` Check if your GitHub Action step actually executed. If you forgot the `if: always()` clause, the step was skipped when the tests failed.