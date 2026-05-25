# QA Flight Recorder (QA Capsule)

[![Version](https://img.shields.io/badge/version-v1.0.12--beta-blue.svg)](https://ashraf-khabar.github.io/qa-capsule/)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)
[![Doc](https://img.shields.io/badge/docs-available-brightgreen.svg)](https://qa-capsule.github.io/qa-capsule-community/)


**QA Flight Recorder** is an enterprise-grade, SRE-oriented control plane designed to monitor CI/CD test failures, detect flaky and slow tests, store visual evidence, and automate incident response via native Go integrations.

<p align="center">
  <img src="./images/ui-1.png" width="800px" alt="Dashboard View">
</p>
<p align="center">
  <img src="./images/ui-2.png" width="800px" alt="Analytics View">
</p>
<p align="center">
  <img src="./images/ui-3.png" width="800px" alt="Plugin Engine">
</p>
<p align="center">
  <img src="./images/ui-4.png" width="800px" alt="Plugin Engine">
</p>
<p align="center">
  <img src="./images/ui-5.png" width="800px" alt="Plugin Engine">
</p>

---

## Why QA Capsule?

Modern CI/CD pipelines generate too much noise. When a database goes down, 100 tests fail, generating 100 identical alerts. **QA Capsule reduces alert fatigue** by correlating identical events, tagging flaky and performance regressions, and triggering Slack/Jira/Teams actions without shell scripts.

## Key Features

### Smart Event Correlation
* **Deduplication:** SHA-256 fingerprinting per test + pipeline run (`X-Run-Id`).
* **Flakiness:** `[FLAKY]` tag when a resolved test fails again within 48h; CLI can query flaky status by hash.
* **Performance:** `[PERF]` alerts when a passed test exceeds 150% of its 30-day average duration.
* **Analytics & MTTR:** Dashboards and FinOps (Chart.js, QCL, PDF export).

### Test Artifacts
* Upload Playwright traces, screenshots, videos: `POST /api/incidents/{id}/artifacts` (max 50MB).
* Local storage under `data/artifacts/`; S3 provider stub for enterprise.

### Native Integration Engine (no shell RCE)
* **Go HTTP integrations:** Slack, Jira, Teams, PagerDuty, Opsgenie, webhooks, email, GitHub.
* JSON manifests in `plugins/` with `"integration": "slack"` (legacy `.sh` commands are ignored).
* Secrets via environment variables; loaded once at startup.

### Developer Experience
* **CLI:** `qacapsule run` wraps local tests and warns on known flaky fingerprints.
* **Playwright reporter:** Real-time failure POST + optional trace zip upload.

### Multi-Tenancy & IAM
* Hierarchical teams and RBAC: Platform Admin, Manager, Lead, Observer.
* Domain lock and forced password reset.

### Universal CI/CD Gateways
* JSON webhooks with `browser`, `os`, `viewport`, `execution_time_ms`, batch `tests[]`.
* JUnit XML upload for Playwright, Cypress, Pytest, and more.

---

## Documentation

Full official docs (MkDocs): **[docs/index.md](docs/index.md)** — published at [qa-capsule.github.io/qa-capsule-community](https://qa-capsule.github.io/qa-capsule-community/)

| Topic | Link |
|---|---|
| Webhooks & enriched payloads | [docs/api/webhooks.md](docs/api/webhooks.md) |
| Artifacts & CLI | [docs/guides/artifacts-and-cli.md](docs/guides/artifacts-and-cli.md) |
| Playwright reporter | [docs/integration/playwright-reporter.md](docs/integration/playwright-reporter.md) |
| Plugin engine (Go) | [docs/plugins/overview.md](docs/plugins/overview.md) |
| Configuration deux côtés (QA Capsule + fournisseur) | [docs/plugins/configuration-guide.md](docs/plugins/configuration-guide.md) |
| Catalogue intégrations (logos) | [docs/plugins/integrations-catalog.md](docs/plugins/integrations-catalog.md) |

---

## Technology Stack

* **Backend:** Go 1.25+ — `pkg/integrations`, `pkg/storage`, `pkg/service`
* **CLI:** Cobra (`cmd/cli`)
* **Database:** SQLite (`modernc.org/sqlite`)
* **Frontend:** Vanilla JavaScript (ES6+)
* **Reporter:** TypeScript (`examples/playwright-reporter/`)

---

## Quick Start (Docker)

```bash
git clone https://github.com/QA-Capsule/qa-capsule-community.git
cd qa-capsule-community
docker compose up -d --build
```

Open **http://localhost:9000** — login `admin` / `admin` (change password on first login).

### Local development

```bash
go run ./cmd/qacapsule/main.go
go build -o bin/qacapsule-cli ./cmd/cli
```

---

## Integration manifest example

```json
{
  "integration": "slack",
  "name": "Smart Slack Routing",
  "version": "1.2",
  "description": "Alerts the project Slack channel on critical failures.",
  "status": "Active",
  "trigger_on": ["CRITICAL", "Timeout", "FLAKY"],
  "env": {}
}
```

Set `SLACK_WEBHOOK_URL` in the server environment — not in git.

---

## Security

If you discover a security vulnerability, please do not open a public issue. Contact the maintainers directly.

Shell-based plugin execution was removed to prevent remote code execution; use native integrations or outbound webhooks only.

## License

MIT License — see `LICENSE`.
