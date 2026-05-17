---
icon: fontawesome/brands/gitlab
---

# GitLab CI/CD Integration

Integrating QA Capsule with GitLab CI allows you to capture test failures directly from your GitLab Runners. Whether you are using shared SaaS runners or self-hosted runners, QA Capsule can ingest your telemetry instantly.

This guide will walk you through provisioning the endpoint, securing your credentials, and updating your `.gitlab-ci.yml` file.

!!! tip "Recommended: JUnit XML Upload"
    For structured per-test sub-alerts, use `POST /api/webhooks/upload` with your JUnit XML report.
    See [JUnit XML Upload](junit-xml-upload.md) and the [CI/CD Overview](cicd-overview.md).

---

## Prerequisites

Before proceeding, ensure you have the following:

1. **QA Capsule Access:** You must have the `Operator` or `Administrator` role to provision a new Webhook endpoint.
2. **GitLab Access:** You must have the `Maintainer` or `Owner` role in your GitLab project to manage CI/CD Variables.
3. **A Test Pipeline:** An existing `.gitlab-ci.yml` file that runs your automated test suites.

---

## Step 1: Identify your Repository Path

Unlike GitHub (which uses Workflow IDs) or Jenkins (which uses Job Names), QA Capsule tracks GitLab pipelines using the **GitLab Repository Path** (also known as the project path with its namespace).

**How to find it:**

1. Open your project in GitLab.
2. Look at the URL in your browser. 
3. If your URL is `https://gitlab.com/acme-corp/backend/payment-api`, your Repository Path is **`acme-corp/backend/payment-api`**.
4. *(Alternatively, this is the exact value of the `$CI_PROJECT_PATH` environment variable inside your runner).*

---

## Step 2: Provision the Endpoint in QA Capsule

Now that you have your Repository Path, you must create a dedicated secure entry point in the QA Capsule database.

1. Log in to your **QA Capsule Dashboard**.
2. Navigate to the **CI/CD Gateways** module using the left-hand sidebar.
3. Click on the **GitLab CI** provider card.
4. **Fill out the Provisioning Form:**
   * **Project Name:** Give it a human-readable name (e.g., `Payment API Core Tests`).
   * **Routing Variables:** Define where alerts for this specific project should go (e.g., `SLACK_CHANNEL: #alerts-backend`, `MS_TEAMS_WEBHOOK: https://...`).
   * **GitLab Repository Path:** Paste the path you identified in Step 1 (e.g., `acme-corp/backend/payment-api`).
5. Click **Provision Project Endpoint**.

**CRITICAL:** The system will generate a **Webhook URL** and a **Generated API Key (Secret)**. Copy both of these values immediately.

---

## Step 3: Configure GitLab CI/CD Variables

You must never commit your QA Capsule API Key into your `.gitlab-ci.yml` file. We will use GitLab's CI/CD Variables to securely inject them into the runner.

1. Open your project on GitLab.
2. Navigate to **Settings > CI/CD** in the left sidebar.
3. Expand the **Variables** section.
4. Click **Add variable** to add the Webhook URL:
   * **Key:** `QA_CAPSULE_URL`
   * **Value:** Paste the generated Webhook URL (e.g., `https://sre.yourcompany.com/api/webhooks/gitlab`).
   * **Type:** Variable
   * **Visibility:** Masked (Checked) - *Optional for URL, but recommended.*
   * Click **Add variable**.
5. Click **Add variable** again for the API Key:
   * **Key:** `QA_CAPSULE_API_KEY`
   * **Value:** Paste the generated API Key (e.g., `sre_pk_gitlab_a1b2c3d4e5f6`).
   * **Type:** Variable
   * **Visibility:** Masked (Checked) - **CRITICAL**
   * **Protect variable:** Check this if you only want telemetry sent from protected branches (like `main`). Otherwise, leave it unchecked so feature branches can report errors too.
   * Click **Add variable**.

---

## Step 4: Update your `.gitlab-ci.yml`

The most elegant way to handle telemetry in GitLab CI is by using the `after_script` block. This block executes after the main `script` block, regardless of whether the main script succeeded or failed.

Open your `.gitlab-ci.yml` file and add the `after_script` to the job that runs your tests.

### Example Configuration:

```yaml
stages:
  - build
  - test

run_e2e_tests:
  stage: test
  image: [mcr.microsoft.com/playwright:v1.40.0-jammy](https://mcr.microsoft.com/playwright:v1.40.0-jammy)
  script:
    - echo "Installing dependencies..."
    - npm ci
    - echo "Running E2E tests..."
    - npx playwright test
  
  # ========================================================================
  # QA CAPSULE TELEMETRY AGENT
  # ========================================================================
  after_script:
    - >
      if [ "$CI_JOB_STATUS" == "failed" ]; then
        echo "Test failure detected. Sending telemetry to QA Capsule..."
        
        # Build the JSON Payload dynamically using GitLab predefined variables
        PAYLOAD=$(cat <<EOF
        {
          "name": "GitLab Job: $CI_JOB_NAME",
          "status": "CRITICAL",
          "browser": "GitLab Runner (Docker)",
          "error": "Pipeline failed on branch $CI_COMMIT_REF_NAME",
          "console_logs": "[FATAL] Job $CI_JOB_ID failed by user $CI_GITLAB_USER_LOGIN.\n\nReview artifacts at: $CI_JOB_URL"
        }
        EOF
        )
        
        # Dispatch the telemetry via cURL
        curl --silent --show-error -X POST "$QA_CAPSULE_URL" \
          -H "Content-Type: application/json" \
          -H "X-API-Key: $QA_CAPSULE_API_KEY" \
          -d "$PAYLOAD"
      else
        echo "Tests passed successfully. No alert sent."
      fi
```

Understanding the Logic:

* We use the predefined variable `$CI_JOB_STATUS` to check if the script block failed.
* If it failed, we construct a JSON payload using rich context variables like `$CI_JOB_URL` (which gives the exact link to the failed job) and `$CI_COMMIT_REF_NAME` (the branch name).
* We execute curl using the Masked Variables we defined in Step 3.

## Step 5: Verify the Integration

1. Commit and push the updated .gitlab-ci.yml to your repository.

2. Trigger a pipeline that is designed to fail.

3. Open the GitLab Job logs and scroll to the bottom to view the after_script execution.

4. If successful, you will see Test failure detected. Sending telemetry to QA Capsule... and no curl errors.

5. Open your QA Capsule Dashboard; the incident will appear instantly.

Troubleshooting

* **`CI_JOB_STATUS` is empty :** Note that `$CI_JOB_STATUS` is only available in GitLab Runner v13.5 and newer. If you are using an ancient runner, use the `rules: - when: on_failure `block instead of an `after_script` condition.

* **401 Unauthorized Error :** GitLab injected the secret, but it was incorrect. Double-check that you didn't accidentally copy a trailing whitespace when pasting the `QA_CAPSULE_API_KEY` into the GitLab Variables UI.