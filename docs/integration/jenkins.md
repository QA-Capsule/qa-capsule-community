---
icon: fontawesome/brands/jenkins
---

# Jenkins Integration

Jenkins remains the workhorse of enterprise CI/CD. By integrating Jenkins with QA Capsule, you can bring modern observability and automated incident routing to your legacy or highly customized Jenkins pipelines.

This guide will walk you through provisioning the endpoint in QA Capsule and securely injecting the telemetry agent into your `Jenkinsfile`.

!!! tip "Recommended: JUnit XML Upload"
    Publish JUnit test results in Jenkins, then `curl` the XML file to `/api/webhooks/upload`.
    See [JUnit XML Upload](junit-xml-upload.md).

---

## Prerequisites

Before starting, ensure you have:

1. **QA Capsule Access:** `Operator` or `Administrator` permissions to create a new Webhook.
2. **Jenkins Access:** Permissions to create global/folder credentials and edit pipeline configurations.
3. **HTTP Request Capabilities:** Your Jenkins build nodes must have `curl` installed, or your Jenkins controller must have the `HTTP Request Plugin` installed. (This guide uses standard `curl` as it is the most universal method).

---

## Step 1: Provision the Endpoint in QA Capsule

Because Jenkins does not have "Repository Paths" in the same way GitHub or GitLab do, QA Capsule tracks Jenkins telemetry using the **Jenkins Job Name**.

1. Log in to your **QA Capsule Dashboard**.
2. Navigate to the **CI/CD Gateways** module.
3. Click on the **Jenkins** provider card.
4. **Fill out the Provisioning Form:**
   * **Project Name:** A human-readable name (e.g., `Billing Service Pipeline`).
   * **Routing Variables:** Define where alerts should go (e.g., `SLACK_CHANNEL: #alerts-billing`, `JIRA_PROJECT_KEY: BILL`).
   * **Jenkins Job Name:** Enter the exact name of your job as it appears in the Jenkins dashboard (e.g., `PROD-Billing-Deployment`). If you use Folders, include the folder path (e.g., `Backend/PROD-Billing-Deployment`).
5. Click **Provision Project Endpoint**.

**CRITICAL:** Copy the generated **Webhook URL** and the **API Key (Secret)** immediately. You will need both for the next step.

---

## Step 2: Configure Jenkins Credentials

Never hardcode your `X-API-Key` into your `Jenkinsfile` or shell scripts. Jenkins provides a secure Credentials store that encrypts secrets.

1. Open your Jenkins Dashboard.
2. Navigate to **Manage Jenkins > Credentials**.
3. Click on **System**, then click on **Global credentials (unrestricted)**.
4. Click **Add Credentials** on the left menu.
5. Fill out the form:
   * **Kind:** `Secret text`
   * **Scope:** `Global`
   * **Secret:** Paste the QA Capsule API Key (e.g., `sre_pk_jenkins_123456789`).
   * **ID:** `QA_CAPSULE_API_KEY` *(This is the variable name you will use in your Jenkinsfile)*.
   * **Description:** API Key for QA Capsule Telemetry.
6. Click **Create**.

---

## Step 3: Update your Declarative `Jenkinsfile`

In a Declarative Pipeline, the best place to put the QA Capsule telemetry agent is inside the `post` block. This guarantees that the script executes after all your tests are finished, allowing you to catch the `failure` state.

Open your `Jenkinsfile` and add the following configuration. Note how we use Jenkins environment variables (`${env.JOB_NAME}`, `${env.BUILD_URL}`) to dynamically build the payload.

```groovy
pipeline {
    agent any

    // 1. Bind the QA Capsule Secret to an environment variable
    environment {
        // You can hardcode the URL, as it is not a secret
        QA_CAPSULE_URL = '[https://sre.yourcompany.com/api/webhooks/jenkins](https://sre.yourcompany.com/api/webhooks/jenkins)'
        // Securely inject the API key from the Credentials store
        QA_CAPSULE_API_KEY = credentials('QA_CAPSULE_API_KEY')
    }

    stages {
        stage('Install Dependencies') {
            steps {
                sh 'npm ci'
            }
        }
        
        stage('Run E2E Tests') {
            steps {
                // If this fails, the pipeline jumps immediately to the 'post' block
                sh 'npx playwright test'
            }
        }
    }

    // ========================================================================
    // QA CAPSULE TELEMETRY AGENT
    // ========================================================================
    post {
        failure {
            script {
                echo "Pipeline failed. Dispatching telemetry to QA Capsule..."
                
                // Construct a JSON payload using Jenkins built-in variables
                def payload = """
                {
                    "name": "Jenkins Job: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
                    "status": "CRITICAL",
                    "browser": "Jenkins Linux Node",
                    "error": "Build failed during execution.",
                    "console_logs": "[FATAL] Pipeline failed. Please review the Jenkins console output.\\n\\nArtifacts and logs available at: ${env.BUILD_URL}"
                }
                """
                
                // Execute curl to push the payload
                sh """
                curl --silent --show-error -X POST "${QA_CAPSULE_URL}" \\
                  -H "Content-Type: application/json" \\
                  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \\
                  -d '${payload}'
                """
            }
        }
        success {
            echo "Pipeline passed successfully. No critical alerts sent."
        }
    }
}
```

---

## Alternative: Legacy Freestyle Jobs

If you are not using Jenkins Pipelines (`Jenkinsfile`) and are still using classic "Freestyle" jobs via the Jenkins UI:

1. Open your Job configuration.
2. Check the box **Use secret text(s) or file(s)** under the "Build Environment" section.
3. Under "Bindings", add a **Secret text**.
   * Variable: `QA_CAPSULE_API_KEY`
   * Credentials: Select the credential you created in Step 2.
4. Scroll down to **Post-build Actions** and add an **Execute shell** step.
5. Paste the following bash script:

```bash
#!/bin/bash
# Only run if the build failed
if [ "$GIT_PREVIOUS_SUCCESSFUL_COMMIT" != "$GIT_COMMIT" ]; then
    curl -X POST "[https://sre.yourcompany.com/api/webhooks/jenkins](https://sre.yourcompany.com/api/webhooks/jenkins)" \
      -H "Content-Type: application/json" \
      -H "X-API-Key: $QA_CAPSULE_API_KEY" \
      -d '{
            "name": "Freestyle Job Failed",
            "status": "CRITICAL",
            "error": "Jenkins UI Build Failure",
            "console_logs": "[FATAL] Review Jenkins UI for details."
          }'
fi
```

---

## Step 4: Verification & Troubleshooting

1. Trigger a build in Jenkins that is designed to fail.
2. Open the **Console Output** for that specific Jenkins build.
3. Scroll to the bottom to find the `post { failure { ... } }` execution block.
4. Look for the `curl` output.

### Common Jenkins Issues:
* **`curl: not found`:** The Docker container or virtual machine running your Jenkins agent does not have `curl` installed. You must install it (`apt-get install curl`) or use the Jenkins HTTP Request plugin instead.
* **`401 Unauthorized`:** Jenkins injected the secret, but it was incorrect. Verify that the ID `QA_CAPSULE_API_KEY` matches exactly in both the Jenkins Credential Manager and the `Jenkinsfile`.
* **Missing Variables:** If your JSON payload breaks, it is usually because variables like `${env.BUILD_URL}` are not configured in your Jenkins Global Settings. Ensure your "Jenkins URL" is correctly set in `Manage Jenkins > System`.