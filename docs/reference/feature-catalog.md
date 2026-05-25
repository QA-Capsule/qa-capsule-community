---
icon: material/view-list
---

# Feature catalog

Complete map of **QA Capsule Community** capabilities: UI, API, CLI, and background workers.

Typography in this site uses the shared compact theme (`docs/stylesheets/extra.css`): **Roboto 13px body**, smaller code blocks, consistent tables.

---

## Editions

| Edition | Build tag | SSO | License UI |
|---|---|---|---|
| **Community** | default (`!enterprise`) | Hidden | No paywall on login |
| **Enterprise** | `-tags enterprise` | When license key set | License API `/api/license` |

See [Editions](../guides/editions-community-enterprise.md).

---

## Control plane (web UI)

| Area | Role (min.) | Capability |
|---|---|---|
| **Dashboard** | Observer | Live incidents, filters, time range, KPI sync, auto-refresh |
| **Execution Hub** | Observer | Pipeline runs, env tags (PROD, STAGING, INTEGRATION, DEV) |
| **Incidents** | Observer | Detail, resolve, logs, RCA trigger, artifacts |
| **RCA & AI Insights** | Observer | AI summaries, provider config (Manager) |
| **Quarantine** | Lead | Deny-list, flaky stats, CI gate alignment |
| **CI/CD Gateways** | Manager | Projects, API keys, webhook URLs, routing matrix |
| **Plugin Engine** | Lead | Native integrations (Slack, Jira, …), test run |
| **Visual Workflow** | Lead | Drawflow DAG per gateway, simulate, save |
| **Runbooks** | Lead | Template apply (502, flaky, OOM, …) |
| **FinOps** | Manager | Cost KPIs, evolution charts, PDF export |
| **Analytics** | Observer | Custom charts (QCL), pinned widgets |
| **DORA** | Manager | Deployment frequency, lead time, CFR, MTTR |
| **IAM** | Admin | Users, roles, domain policy, password reset |
| **System Settings** | Admin | SMTP, security policy, global config |
| **Help Center** | All | In-app guides (workflows, FinOps formulas, architecture) |

---

## Ingestion and intelligence

| Feature | Entry | Notes |
|---|---|---|
| JSON webhook | `POST /api/webhooks/{project}` | `X-API-Key`, optional `X-Run-Id`, `X-Run-Attempt` |
| JUnit upload | `POST /api/webhooks/upload` | Batch XML |
| Async queue | Internal | `202 queued`; workers: `QACAPSULE_INGEST_WORKERS`, `QACAPSULE_INGEST_QUEUE_SIZE` |
| Deduplication | Fingerprint + run | Same test + same run + **open** incident only |
| Flaky tag | `[FLAKY]` prefix | Re-fail after resolve within policy window |
| Perf tag | `[PERF]` | Passed test slower than 150% of 30-day avg |
| Quarantine gate | `GET /api/ci/quarantine/status` | CI skips unstable tests |
| AI RCA | `POST /api/incidents/{id}/rca` | OpenAI / Ollama / disabled |
| Self-healing propose | `POST /api/incidents/{id}/healing/propose` | API / MCP (no UI button yet) |
| Prometheus signals | `POST /api/webhooks/prometheus` | Correlate external alerts |

---

## Authentication and security

| Mechanism | Doc |
|---|---|
| JWT (UI sessions) | [Security & authentication](../setup/security-authentication.md) |
| Project API keys | Webhooks, CLI, artifacts |
| MCP bearer token | `QACAPSULE_MCP_TOKEN` |
| Optional webhook token | `telemetry.webhook_token` in config |
| RBAC | [RBAC & teams](../setup/rbac-teams.md) |

---

## Developer tools

| Tool | Path | Purpose |
|---|---|---|
| **qacapsule CLI** | `cmd/cli` | `qacapsule run` + flaky warning |
| **JUnit agent** | `cmd/agent` | Push XML to webhook |
| **listusers** | `cmd/listusers` | List DB users |
| **resetpass** | `cmd/resetpass` | Reset password offline |
| **Playwright reporter** | `examples/playwright-reporter` | Real-time failure POST |

Details: [Utility binaries](utility-binaries.md), [Artifacts & CLI](../guides/artifacts-and-cli.md).

---

## Operations

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Liveness |
| `GET /readyz` | DB + storage readiness |
| `GET /metrics` | Prometheus-style process metrics |

See [Operations & health API](../api/operations-health.md).

---

## Data and storage

| Path | Content |
|---|---|
| `QACAPSULE_DATA_DIR` / `./data` | `qacapsule.db` (SQLite WAL) |
| `./data/artifacts/` | Incident uploads (local provider) |
| `plugins/` | JSON integration manifests |
| `reports/` | Telemetry JSON for `/ws` demo |

---

## Native integrations (plugin engine)

Slack, Jira, Microsoft Teams, PagerDuty, Opsgenie, VictorOps, Datadog, email, GitHub, custom webhook, test-management stubs, K8s stub. Catalog: [Integrations catalog](../plugins/integrations-catalog.md).

Shell-based plugin execution is **not** supported (RCE removed).

---

## Related guides

- [Platform user guide](../guides/platform-user-guide.md)
- [MCP & self-healing testing](../guides/mcp-self-healing-testing.md)
- [System architecture](../guides/architecture.md)
