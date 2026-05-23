---
icon: material/paperclip
---

# Test Artifacts, Developer CLI & Local Workflow

Store visual evidence per incident, wrap local test runs, and integrate with CI reporters.

---

## Test artifacts (Phase 1)

### Storage model

| Table | Purpose |
|---|---|
| `incident_artifacts` | Metadata: filename, size, provider, path |
| Files on disk | `./data/artifacts/incident_{id}/` (default) |

### Storage providers

| Provider | Config | Status |
|---|---|---|
| `local` | `storage.local_path` in `config.yaml` or default `./data/artifacts` | **Implemented** |
| `s3` | `STORAGE_PROVIDER=s3`, `STORAGE_S3_BUCKET`, … | Stub (enterprise) |

Environment override:

```bash
STORAGE_PROVIDER=local
```

### Upload API

```
POST /api/incidents/{incident_id}/artifacts
Content-Type: multipart/form-data
```

| Item | Value |
|---|---|
| Form field | `file` |
| Max size | 50 MB |
| Extensions | `.zip`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.webm`, `.mp4`, `.trace` |
| Auth | `X-API-Key` (CI) or JWT (UI) |
| Response | `202 Accepted` — write runs in background goroutine |

### Example (PowerShell)

```powershell
$apiKey = "YOUR_PROJECT_KEY"
$incidentId = 42

Invoke-RestMethod -Uri "http://localhost:9000/api/incidents/$incidentId/artifacts" `
  -Method POST -Headers @{ "X-API-Key" = $apiKey } `
  -Form @{ file = Get-Item ".\trace.zip" }
```

Verify:

```text
data/artifacts/incident_42/trace.zip
```

---

## Developer CLI (`qacapsule`)

Binary: `cmd/cli` (Cobra)

### Build

```bash
go build -o bin/qacapsule-cli ./cmd/cli
```

### Usage

```bash
qacapsule run --test-name "Login test" --test-error "assert 1==2" npx playwright test
```

| Flag / env | Description |
|---|---|
| `--api` / `QACAPSULE_API_URL` | Server base URL (default `http://localhost:9000`) |
| `--api-key` / `QACAPSULE_API_KEY` | Project API key |
| `--test-name` | Test name for fingerprint (should match CI) |
| `--test-error` | Error text for fingerprint |

### Behavior

1. Runs the wrapped command (stdout/stderr passed through).
2. On non-zero exit, computes `SHA256(name|error)`.
3. Calls `GET /api/incidents/check-flaky/{hash}`.
4. If `flaky: true`, prints a **yellow warning** that the failure may be ignorable in CI.

---

## Enriched ingestion (Phase 2)

Send dimensional metadata with failures or passed perf samples:

```json
{
  "name": "Checkout @jira-SCRUM-42",
  "status": "FAILED",
  "error": "Timeout",
  "browser": "chromium",
  "os": "linux",
  "viewport": "1280x720",
  "execution_time_ms": 4200
}
```

Batch:

```json
{ "tests": [ { "name": "...", "status": "FAILED", "error": "..." } ] }
```

Details: [Webhooks API](../api/webhooks.md), [Incident Lifecycle](incident-lifecycle.md).

---

## Playwright reporter (Phase 4)

Real-time reporter: [Playwright Reporter](../integration/playwright-reporter.md)

---

## End-to-end test checklist

| Step | Action |
|---|---|
| 1 | Start server: `go run ./cmd/qacapsule/main.go` |
| 2 | Copy project `api_key` from UI |
| 3 | POST webhook → note `last_incident_id` |
| 4 | POST artifact zip to `/api/incidents/{id}/artifacts` |
| 5 | Resolve incident → re-POST same failure → see `[FLAKY]` |
| 6 | Run CLI with matching `--test-name` / `--test-error` |

---

## Related

- [Incidents API](../api/incidents-api.md)
- [System Configuration](../setup/config.md) — `storage` block
