---
icon: material/robot-industrial
---

# The SRE Agent Architecture

To seamlessly connect your CI/CD infrastructure to the QA Capsule Control Plane, you need to understand the concept of the **SRE Agent**. 

Unlike legacy monitoring systems that require you to install heavy, resource-intensive daemons on your build servers, QA Capsule relies on a modern, event-driven **Push Architecture**.

---

## 1. The "No-Daemon" Philosophy

QA Capsule is entirely passive. It does not constantly poll your GitHub or GitLab repositories looking for failures. Instead, it waits securely behind its Webhook API.

**What is the SRE Agent?**

The SRE Agent is simply a lightweight script (often just a few lines of Bash or a simple Node.js script) that runs as the very last step of your CI/CD pipeline. Its only job is to:

1. Detect if the preceding test steps failed.
2. Read the generated test report (usually a JUnit XML file).
3. Send an authenticated HTTP POST request to QA Capsule.

Because it relies on standard Unix tools like `curl`, the Agent has zero dependencies, making it compatible with any CI/CD runner in the world (Alpine Linux, Ubuntu, Windows Server, etc.).

---

## 2. The Universal Language: JUnit XML

To build a truly agnostic system, QA Capsule needs to understand errors regardless of the language or framework your engineers are using. Whether your team writes tests in JavaScript (Playwright, Cypress), Python (PyTest, Robot Framework), or Java (Selenium, JUnit), QA Capsule understands them all.

How? By relying on the **JUnit XML Standard**.

When a test fails, your framework generates an XML file. The SRE Agent pushes this XML (or parsed components of it) to the QA Capsule parser.

### What the Parser Extracts:

When the QA Capsule backend receives the payload, it looks for specific XML tags to build a rich Incident Card on your dashboard:

* `<testcase name="...">`: Identifies the exact test that failed.
* `<failure message="...">`: The human-readable reason for the crash (e.g., "Timeout 5000ms exceeded").
* `CDATA` inside `<failure>`: The raw **StackTrace**, pointing to the exact line of code in your repository.
* `<system-out>`: Everything your code printed to `console.log()` or `print()` right before it crashed.
* `<system-err>`: System-level exceptions or network latency warnings.

---

## 3. Authentication & Security (X-API-Key)

Because QA Capsule can trigger external plugins (like posting to Jira or Slack), the ingestion endpoints are heavily secured. The SRE Agent must authenticate every payload it sends.

When you provision a project in the **CI/CD Gateways** UI, the system generates a unique API Key (e.g., `sre_pk_github_8f92a1b`). 

### The Security Contract

1. The SRE Agent must include this key in the HTTP Headers: `-H "X-API-Key: <YOUR_KEY>"`.
2. **Never hardcode this key in your repository.** You must inject it securely using your CI/CD provider's secret management system (e.g., GitHub Secrets, GitLab CI/CD Variables, Jenkins Credentials).
3. If the key is missing or invalid, QA Capsule drops the payload immediately and returns a `401 Unauthorized` error to prevent abuse.

---

## 4. The Reference Implementation: `sre-agent.sh`

While you can write the agent logic directly in your pipeline YAML files, enterprise teams often prefer to keep their pipelines clean by downloading a centralized bash script.

Below is a robust, production-ready reference script. You can save this as `sre-agent.sh` in your repository or host it centrally for all your teams to use.

```bash
#!/bin/bash
# ==============================================================================
# QA Capsule - SRE Telemetry Agent
# Description: Pushes CI/CD test failures to the QA Capsule Control Plane.
# ==============================================================================

set -e

# 1. Validate required environment variables
if [ -z "$QA_CAPSULE_URL" ]; then
  echo "[QA-Agent] ERROR: QA_CAPSULE_URL is not set."
  exit 1
fi

if [ -z "$QA_CAPSULE_API_KEY" ]; then
  echo "[QA-Agent] ERROR: QA_CAPSULE_API_KEY is not set."
  exit 1
fi

# 2. Extract context from the CI environment
# This example uses GitLab CI variables, but can be adapted for GitHub Actions
PROJECT_NAME="${CI_PROJECT_NAME:-"Unknown Project"}"
JOB_NAME="${CI_JOB_NAME:-"Automated Test Suite"}"
PIPELINE_URL="${CI_PIPELINE_URL:-"N/A"}"
BRANCH_NAME="${CI_COMMIT_REF_NAME:-"main"}"

echo "[QA-Agent] Initializing telemetry for project: $PROJECT_NAME"

# 3. Build the JSON Payload
# We use standard Bash heredocs to construct the Unified Alert Schema
PAYLOAD=$(cat <<EOF
{
  "name": "Pipeline Job Failed: $JOB_NAME",
  "status": "CRITICAL",
  "browser": "CI Runner ($BRANCH_NAME)",
  "error": "Pipeline failure on branch $BRANCH_NAME",
  "console_logs": "[INFO] Triggered by SRE Agent\n[FATAL] Job failed. Review artifacts at: $PIPELINE_URL"
}
EOF
)

# 4. Dispatch the Telemetry via cURL
echo "[QA-Agent] Dispatching telemetry to $QA_CAPSULE_URL..."

HTTP_RESPONSE=$(curl --silent --write-out "HTTPSTATUS:%{http_code}" \
  -X POST "$QA_CAPSULE_URL" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $QA_CAPSULE_API_KEY" \
  -d "$PAYLOAD")

# 5. Handle the Response
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

if [ "$HTTP_STATUS" -eq 202 ]; then
  echo "[QA-Agent] SUCCESS: Incident recorded by QA Capsule."
  exit 0
elif [ "$HTTP_STATUS" -eq 401 ]; then
  echo "[QA-Agent] FATAL: Authentication failed (401). Check your X-API-Key."
  exit 1
else
  echo "[QA-Agent] ERROR: Failed to reach QA Capsule. HTTP $HTTP_STATUS"
  echo "Response: $HTTP_BODY"
  exit 1
fi
```

**How to use this script in your pipeline:**

Make the script executable and run it only if the previous test step failed.

```bash
chmod +x sre-agent.sh
./sre-agent.sh
```

## 5. Troubleshooting the Agent

If your **CI/CD pipeline** is running but incidents are not appearing on your QA Capsule Dashboard, check the logs of your CI runner for the Agent's HTTP response code :

1. **202 Accepted** : The connection is perfect. The incident was saved, and background plugins are executing. If you don't see it on the dashboard, refresh the page.

2. **401 Unauthorized** : The connection works, but QA Capsule rejected the request. Ensure that the API Key in your CI Secrets perfectly matches the one generated in the UI. Ensure you haven't deleted and re-created the SQLite database without generating a new key.

3. **400 Bad Request** : The JSON payload sent by the agent is malformed (e.g., missing a comma, or unescaped quotes in the console logs).

4. **Connection Refused / Timeout** : Your `CI/CD` runner cannot reach your QA Capsule instance. If QA Capsule is hosted internally, ensure your firewall rules allow inbound traffic from your `CI/CD` provider's IP addresses.