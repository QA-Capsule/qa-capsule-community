---
icon: material/api
---

# Incidents REST API

Authenticated endpoints for incidents, artifacts, flaky lookup, FinOps, and reports.

**Base URL:** `http://localhost:9000` or your production host

**Authentication:** JWT Bearer from `POST /api/login` (except flaky check and artifact upload with `X-API-Key`)

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

---

## List incidents

```
GET /api/incidents?project={name}
```

### Response fields (subset)

| Field | Description |
|---|---|
| `id` | Incident ID |
| `name` | May include `[FLAKY]` or `[PERF]` prefix |
| `browser`, `os`, `viewport` | Dimensional tags (when sent by webhook) |
| `execution_time_ms` | Last reported duration |
| `pipeline_run_id` | CI run scope |
| `is_resolved` | Boolean |

### Visibility

| Role | Scope |
|---|---|
| `admin`, `manager` | All projects |
| `lead`, `observer` | Projects linked via teams |

---

## Resolve incidents

```
PUT /api/incidents
Content-Type: application/json
```

```json
{ "ids": [42, 43] }
```

| Role | Can resolve? |
|---|---|
| `admin`, `manager`, `lead` | Yes |
| `observer` | No |

---

## Delete incidents

```
DELETE /api/incidents?ids=42,43
```

| Role | Can delete? |
|---|---|
| `admin`, `manager`, `lead` | Yes (per RBAC UI rules) |
| `observer` | No |

---

## Check flaky fingerprint

Used by the **developer CLI** and custom tooling.

```
GET /api/incidents/check-flaky/{fingerprint}
```

- `{fingerprint}` — 64-character hex SHA-256 of `name|error` (same algorithm as server).
- No JWT required; optional `X-API-Key`.

### Response `200 OK`

```json
{
  "fingerprint": "a1b2c3...",
  "flaky": true,
  "label": "[FLAKY]",
  "message": "This test is known as unstable in CI (resolved and failed again within 30 days)."
}
```

### Compute fingerprint (examples)

=== "Go"

```go
import "github.com/QA-Capsule/qa-capsule-community/pkg/core"

hash := core.IncidentFingerprint("Login test", "assert 1==2")
```

=== "PowerShell"

```powershell
$raw = "Login test|assert 1==2"
$bytes = [Text.Encoding]::UTF8.GetBytes($raw)
$hash = ([BitConverter]::ToString([Security.Cryptography.SHA256]::Create().ComputeHash($bytes)) -replace '-','').ToLower()
```

---

## Upload artifact

Attach Playwright traces, screenshots, or videos to an incident.

```
POST /api/incidents/{incident_id}/artifacts
Content-Type: multipart/form-data
```

| Auth | Header |
|---|---|
| CI / reporter | `X-API-Key: <project_key>` |
| UI automation | `Authorization: Bearer <jwt>` |

Form field: **`file`**

| Constraint | Value |
|---|---|
| Max size | 50 MB |
| Allowed extensions | `.zip`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.webm`, `.mp4`, `.trace` |

### Response `202 Accepted`

```json
{
  "status": "accepted",
  "incident_id": 42,
  "message": "Artifact upload queued for storage"
}
```

Files are written asynchronously to `./data/artifacts/incident_{id}/` (or `STORAGE_PROVIDER` config).

---

## List artifacts

```
GET /api/incidents/{incident_id}/artifacts
Authorization: Bearer <jwt>
```

Returns JSON array of `incident_artifacts` records.

---

## Weekly report & metrics

```
GET /api/reports/weekly?project={name}
GET /api/metrics
GET /api/finops
```

See [Analytics & FinOps](../guides/finops-analytics.md).

---

## Example workflow

```bash
# 1. Login
TOKEN=$(curl -s -X POST "http://localhost:9000/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# 2. Ingest failure (get incident id)
curl -s -X POST "http://localhost:9000/api/webhooks/" \
  -H "X-API-Key: YOUR_KEY" -H "Content-Type: application/json" \
  -d '{"name":"Demo","status":"FAILED","error":"fail"}' | jq .

# 3. Upload trace zip
curl -X POST "http://localhost:9000/api/incidents/42/artifacts" \
  -H "X-API-Key: YOUR_KEY" \
  -F "file=@trace.zip"

# 4. Check flaky
curl "http://localhost:9000/api/incidents/check-flaky/${HASH}"
```

---

## Error codes

| Code | Meaning |
|---|---|
| `401` | Missing or invalid JWT / API key |
| `403` | Insufficient role |
| `413` | Artifact > 50 MB |
| `400` | Invalid extension or incident id |
