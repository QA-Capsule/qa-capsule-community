---
icon: material/api
---

# Incidents REST API

Authenticated endpoints for reading, resolving, and deleting incidents. Used by the dashboard and available for automation.

**Base URL:** `https://sre.yourcompany.com` (or `http://localhost:9000`)

**Authentication:** JWT Bearer token from `POST /api/login`

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

---

## List Incidents

```
GET /api/incidents?project={name}&_ts={timestamp}
```

| Query param | Description |
|---|---|
| `project` | Filter by project name, or `all` for every project (admin) |
| `_ts` | Cache-buster timestamp (optional) |

### Response `200 OK`

```json
[
  {
    "id": 42,
    "project_name": "Frontend E2E",
    "name": "checkout.spec - payment button visible",
    "status": "CRITICAL",
    "error_message": "Timeout 1500ms exceeded",
    "console_logs": "Navigating to checkout...",
    "error_logs": "locator.click: Timeout...",
    "is_resolved": false,
    "resolved_by": "",
    "created_at": "2026-05-17 14:32:01",
    "resolved_at": ""
  }
]
```

### Visibility rules

| Role | Scope |
|---|---|
| `admin` | All incidents |
| `operator`, `viewer` | Incidents for projects linked to user's teams only |

---

## Resolve Incidents

```
PUT /api/incidents
Content-Type: application/json
```

### Request body

```json
{
  "ids": [42, 43, 44]
}
```

Or single ID:

```json
{
  "id": 42,
  "ids": [42]
}
```

### Response `200 OK`

Empty body on success.

### Side effects

```sql
UPDATE incidents SET
  is_resolved = 1,
  status = 'resolved',
  resolved_by = '{username}',
  resolved_at = CURRENT_TIMESTAMP
WHERE id IN (...)
```

| Role | Can resolve? |
|---|---|
| `admin` | Yes |
| `operator` | Yes |
| `viewer` | **No** (403) |

---

## Delete Incidents

```
DELETE /api/incidents?ids=42,43,44
```

| Role | Can delete? |
|---|---|
| `admin` | Yes |
| `operator`, `viewer` | **No** (403) |

### Response `200 OK`

Records permanently removed from SQLite.

---

## Weekly Report

```
GET /api/reports/weekly?project={name}
```

See [Analytics & FinOps](../guides/finops-analytics.md).

---

## Metrics

```
GET /api/metrics
```

See [Analytics & FinOps](../guides/finops-analytics.md).

---

## FinOps Settings

```
GET  /api/finops
PUT  /api/finops   (admin only)
```

### PUT body

```json
{
  "dev_hourly_rate": 75.0,
  "ci_minute_cost": 0.008,
  "avg_pipeline_duration": 15.0,
  "avg_investigation_time": 30.0,
  "currency": "USD"
}
```

---

## Example: Resolve via curl

```bash
# 1. Login
TOKEN=$(curl -s -X POST "http://localhost:9000/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"yourpassword"}' | jq -r '.token')

# 2. Resolve incidents 10 and 11
curl -X PUT "http://localhost:9000/api/incidents" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"ids": [10, 11]}'
```

---

## Error Responses

| Code | Meaning |
|---|---|
| `401` | Missing or expired JWT |
| `403` | Insufficient role (e.g. viewer trying to delete) |
| `400` | Malformed JSON or invalid ID format |
| `500` | SQLite lock contention (automatic retry up to 5 attempts) |
