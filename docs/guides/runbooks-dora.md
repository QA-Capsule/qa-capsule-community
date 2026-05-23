# Runbooks & DORA Metrics

## Runbook templates

Built-in templates live in `pkg/integrations/runbooks.go`. Each template is a validated workflow DAG using **registry plugins only**.

| Template ID | Use case |
|-------------|----------|
| `502-restart-pod` | HTTP 502 in error → K8s rollout restart + Slack |
| `flaky-triage` | `[FLAKY]` prefix → Slack + Jira |
| `perf-regression` | `[PERF]` prefix → Datadog + Slack |
| `oom-restart` | OOM / CrashLoop in error → K8s restart |
| `timeout-cache-flush` | Timeout in error → custom webhook |

### API

- `GET /api/runbooks/templates` — list templates
- `GET /api/runbooks/templates?id=502-restart-pod` — preview workflow JSON
- `POST /api/runbooks/apply` — body: `{ "project_id": "1", "template_id": "502-restart-pod", "enable": true }` (Lead+)

Applying a template writes `projects.sre_workflow_json` and enables the visual workflow engine for that gateway.

## DORA dashboard

Managers see **DORA & Executive Dashboard** with:

- Deployment frequency (pipeline runs / day)
- Lead time (median minutes from pipeline start to first incident)
- Change failure rate (failed runs / total runs)
- MTTR (resolved incidents)

Pipeline runs are recorded on each CI webhook ingest (`X-Run-Id`, optional `X-Branch`, `X-Commit-Sha`).

### Prometheus webhook

```http
POST /api/webhooks/prometheus?project=my-e2e-suite
X-API-Key: <gateway-api-key>
Content-Type: application/json

{ "alerts": [{ "labels": { "alertname": "HighErrorRate" }, "annotations": { "summary": "5xx spike" }, "startsAt": "2026-05-23T10:00:00Z" }] }
```

Signals are stored in `external_signals` and correlated to incidents in the same project within **±15 minutes**.

### API

- `GET /api/dora/metrics?range=30d&project=` (Manager JWT)
- `GET /api/dora/signals?project=` (Manager JWT)
