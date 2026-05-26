---
icon: material/robot
---

# MCP, self-healing, UI & CLI testing guide

End-to-end validation of **Self-Healing**, **MCP**, the **web UI**, **REST API**, and **qacapsule CLI**.

---

## What you can test where

| Capability | Web UI | REST API | MCP (`/mcp`) | CLI |
|---|---|---|---|---|
| Ingest failures | Refresh after webhook | Webhook | — | Agent / curl |
| Dashboard / KPIs | Yes | `GET /api/metrics` | — | — |
| Self-healing insights | Self-Healing Hub | `GET /api/healing/insights` | `list_failed_incidents` + `get_incident_context` | — |
| Flaky / quarantine | Quarantine page | Quarantine API | `get_flaky_tests` | `qacapsule run` warning |
| Self-healing patch | Not yet (button) | `POST .../healing/propose` | — | — |
| Health | — | `GET /healthz` | — | curl |

---

## Part A — Web UI

### Start the server

```bash
docker compose up -d --build
# or: go run ./cmd/qacapsule/main.go
```

Open `http://localhost:9000` — sign in **`admin`** / **`admin`**, then set a new password.

Optional MCP protection:

```bash
export QACAPSULE_MCP_TOKEN=dev-mcp-secret
```

### Create a gateway

1. **CI/CD Gateways** → create project `demo-ci`.
2. Copy **API key** and webhook URL `http://localhost:9000/api/webhooks/demo-ci`.

### Ingest a failure (webhook)

```powershell
$headers = @{
  "X-API-Key" = "YOUR_KEY"
  "X-Run-Id" = "run-$(Get-Date -Format yyyyMMddHHmmss)"
  "Content-Type" = "application/json"
}
$body = @{
  name = "checkout payment"
  status = "CRITICAL"
  error = "Timeout waiting for upstream"
  project = "demo-ci"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:9000/api/webhooks/demo-ci" -Method POST -Headers $headers -Body $body
```

Expect `{"status":"queued"}`. Wait 2–3 seconds, refresh the **Dashboard**.

### Self-Healing Hub

1. Open **Self-Healing Hub**.
2. Verify open failures are categorized (`timeout`, `locator`, `assertion`, ...).
3. Use **Copy MCP setup** and **Copy MCP prompt** from an incident card.

### Flaky detection

1. Resolve the incident on the dashboard.
2. Re-send the same webhook (same test name + error, new `X-Run-Id`).
3. Expect `[FLAKY]` prefix on the test name.

---

## Part B — qacapsule CLI

### Build

```bash
go build -o bin/qacapsule ./cmd/cli
```

### Commands

```bash
qacapsule run --api http://localhost:9000 --api-key YOUR_KEY \
  --test-name "checkout payment" --test-error "timeout" \
  npx playwright test
```

| Flag / env | Description |
|---|---|
| `--api` / `QACAPSULE_API_URL` | Control plane URL |
| `--api-key` / `QACAPSULE_API_KEY` | Project API key |
| `--test-name` / `QACAPSULE_TEST_NAME` | Fingerprint name |
| `--test-error` / `QACAPSULE_TEST_ERROR` | Fingerprint error text |

On failure, CLI calls `GET /api/incidents/check-flaky/{sha256}` and prints a **yellow `[FLAKY]` warning** when applicable.

### Other binaries

See [Utility binaries](../reference/utility-binaries.md) (`agent`, `listusers`, `resetpass`).

---

## Part C — Self-healing API

```http
POST /api/incidents/{id}/healing/propose
Authorization: Bearer <jwt>
Content-Type: application/json
```

Requires Lead+. Response includes framework-agnostic guidance (`error_category`, `suggested_actions`, `mcp_prompt`).

```http
POST /api/incidents/{id}/healing/pr
```

GitHub PR creation (when integration configured).

---

## Part D — MCP (Cursor)

1. Add server URL `http://localhost:9000/mcp`.
2. If `QACAPSULE_MCP_TOKEN` is set, add header `Authorization: Bearer <token>`.
3. Verify tools:
   - `list_failed_incidents`
   - `get_incident_context`
   - `propose_healing`
   - `submit_healing_patch`
   - `create_remediation_pr`
   - `resolve_incident`
   - `get_flaky_tests`

### MCP tool quick contract

- `list_failed_incidents` → optional `{ project, limit }`
- `get_incident_context` → `{ incident_id }`
- `propose_healing` → `{ incident_id, file_content? }`
- `submit_healing_patch` → `{ incident_id, repo, file_path, code, explanation?, agent_source? }` (persists audit record)
- `create_remediation_pr` → `{ repo, file_path, code, incident_id? }`
- `resolve_incident` → `{ incident_id, resolved_by? }`

Example prompt in Cursor:

> List failed incidents for project demo-ci via list_failed_incidents.  
> Use get_incident_context for incident 1 and propose a minimal cross-framework fix via propose_healing.  
> Submit patch metadata with submit_healing_patch, then open a PR with create_remediation_pr.  
> After green rerun, mark incident resolved with resolve_incident.

---

## Part E — Health endpoints

```bash
curl -s http://localhost:9000/healthz
curl -s http://localhost:9000/readyz
curl -s http://localhost:9000/metrics
```

---

## Validation checklist

| # | Test | UI | API/CLI |
|---|---|---|---|
| 1 | Login + password change | Yes | — |
| 2 | Create gateway | Yes | — |
| 3 | Webhook → incident | Refresh | curl / agent |
| 4 | Self-healing context | Yes | GET/POST healing endpoints |
| 5 | Flaky tag | Yes | Same webhook twice |
| 6 | CLI flaky warn | — | `qacapsule run` |
| 7 | MCP tools | — | Cursor |
| 8 | healthz/readyz | — | curl |
