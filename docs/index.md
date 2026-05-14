---
icon: material/home
---

# QA Capsule

**QA Flight Recorder** (QA Capsule) is your dedicated SRE (Site Reliability Engineering) Control Plane. It is designed to ingest, analyze, and intelligently route failures from your CI/CD pipelines and End-to-End (E2E) tests in real-time.

## The Problem It Solves

In modern software development, test failures (Cypress, Playwright, Selenium) happen frequently. When a pipeline crashes, developers usually have to:
1. Log into the CI/CD platform (GitHub/GitLab).
2. Dig through thousands of lines of terminal logs to find the exact error.
3. Manually create a Jira ticket.
4. Manually notify the right team on Slack or Teams.

**QA Capsule automates this entire lifecycle.**

## Core Architecture

QA Capsule is built on a highly efficient Go backend with an embedded SQLite database and a vanilla JavaScript frontend.

1. **Ingestion (Webhooks):** QA Capsule provides secure, project-specific endpoints. Your CI/CD pipeline pushes test reports (JSON or JUnit XML) to these endpoints.
2. **Analysis (Parser):** The internal engine parses the reports, extracting the exact `StackTrace`, `StdOut`, and `StdErr`.
3. **Dynamic Routing:** QA Capsule identifies which project failed using the `X-API-Key`. It queries the SQLite database to find the specific Slack Channel, MS Teams Webhook, or Jira Project Key associated with that exact project.
4. **Remediation (Plugins):** The built-in Plugin Engine automatically executes Bash/Python scripts (like the Jira ticket creator or Teams notifier) injecting the dynamic routing variables on the fly.

Navigate through the sidebar to set up your instance, configure your CI/CD gateways, and activate your first plugins!