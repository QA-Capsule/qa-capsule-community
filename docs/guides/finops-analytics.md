---
icon: material/calculator-variant
---

# Analytics & FinOps

QA Capsule includes built-in analytics to quantify the cost of test failures and track engineering health over time.

---

## Dashboard KPIs

| KPI | Source | Description |
|---|---|---|
| **Active Alerts** | Unresolved incident count | Real-time open failures |
| **Resolved (48h)** | Resolved in rolling window | Recently closed incidents |
| **Pipeline Health** | `resolved / total * 100` | Percentage of incidents resolved |
| **MTTR** | Average resolve time | Mean minutes from `created_at` to `resolved_at` |

---

## Analytics Panel

Click **Analytics** in the dashboard toolbar to reveal:

- **Failure trend chart** — failures over time by project
- **Flaky vs stable breakdown** — pie chart of `[FLAKY]` tagged tests
- **Weekly report table** — per-pipeline health scores

---

## FinOps Settings

Administrators configure cost parameters under **Settings → FinOps**:

| Parameter | Default | Purpose |
|---|---|---|
| `dev_hourly_rate` | Configurable | Engineer hourly cost (USD/EUR/etc.) |
| `ci_minute_cost` | Configurable | CI runner cost per minute |
| `avg_pipeline_duration` | Configurable | Average pipeline length in minutes |
| `avg_investigation_time` | Configurable | Average minutes spent investigating a failure |

### Calculated metrics

```
cost_per_investigation = (dev_hourly_rate / 60) * avg_investigation_time
total_ci_minutes_lost = total_incidents * avg_pipeline_duration
total_financial_impact = (total_ci_minutes_lost * ci_minute_cost) + (total_incidents * cost_per_investigation)
flaky_financial_impact = (flaky_count * avg_pipeline_duration * ci_minute_cost) + (flaky_count * cost_per_investigation)
```

---

## Weekly Reports API

```
GET /api/reports/weekly?project={name}
Authorization: Bearer {jwt}
```

Response per pipeline:

```json
{
  "pipeline": "Frontend E2E",
  "total_alerts": 24,
  "resolved_alerts": 20,
  "flaky_tests": 6,
  "health_score": 83
}
```

### Export from UI

- **CSV** — downloads spreadsheet-compatible report
- **PDF** — formatted executive summary via jsPDF

---

## Metrics API

```
GET /api/metrics
Authorization: Bearer {jwt}
```

```json
{
  "total_incidents": 150,
  "resolved_incidents": 120,
  "flaky_tests": 18,
  "stable_failures": 132,
  "mttr_minutes": 45,
  "sre_impact": {
    "ci_minutes_lost": 2250,
    "flaky_minutes_lost": 270,
    "estimated_cost_usd": 1840,
    "flaky_waste_cost_usd": 220
  }
}
```

Use this endpoint for Grafana/custom dashboards by polling from a script with a service account JWT.
