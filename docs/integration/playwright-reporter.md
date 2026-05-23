---
icon: material/play-circle
---

# Playwright Reporter (Native)

Send failures to QA Capsule **as each test ends** (fire-and-forget), without waiting for the full suite. Optionally upload the Playwright trace zip to the artifact API.

Source: `examples/playwright-reporter/qacapsule-reporter.ts`

---

## Prerequisites

1. QA Capsule server running (port `9000` by default).
2. A **CI/CD Gateway** project with API key.
3. Environment variables on the runner:

```bash
export QACAPSULE_API_URL=http://localhost:9000
export QACAPSULE_API_KEY=your_project_api_key
export QACAPSULE_PROJECT=Frontend E2E   # optional label
```

---

## Install in Playwright

`playwright.config.ts`:

```typescript
import { defineConfig } from '@playwright/test';

export default defineConfig({
  reporter: [
    ['list'],
    ['./examples/playwright-reporter/qacapsule-reporter.ts', {
      apiUrl: process.env.QACAPSULE_API_URL || 'http://localhost:9000',
      apiKey: process.env.QACAPSULE_API_KEY,
      project: process.env.QACAPSULE_PROJECT,
    }],
  ],
});
```

For a published package, copy `qacapsule-reporter.ts` into your repo or reference it by relative path.

---

## Behavior

| Event | Action |
|---|---|
| Test **failed** / timed out | `POST /api/webhooks/` with Playwright metadata (async) |
| Test **passed** with duration | Same endpoint with `status: PASSED` (for perf regression) |
| Trace attachment present | After webhook, `POST /api/incidents/{id}/artifacts` with zipped trace |

Headers sent:

```http
X-API-Key: ...
X-Run-Id: pw-<timestamp>
Content-Type: application/json
```

Payload includes: `title`, `failure_reason`, `browser` (project name), `os`, `execution_time_ms`, `framework: playwright`.

---

## Trace upload

When Playwright records a trace (`trace: 'on-first-retry'` or `on`), the reporter:

1. Reads `last_incident_id` from the webhook response.
2. Zips the trace file (`zip` on Linux/macOS, `Compress-Archive` on Windows).
3. Uploads via multipart `file` field.

Requires trace size under **50 MB**.

---

## Comparison with JUnit upload

| Approach | When to use |
|---|---|
| **This reporter** | Real-time alerts, fastest feedback, trace per failure |
| [JUnit XML upload](junit-xml-upload.md) | Post-suite batch, legacy CI, all frameworks exporting XML |

You can use both: reporter for speed, JUnit for full suite archival.

---

## Troubleshooting

| Issue | Fix |
|---|---|
| No incidents | Check `QACAPSULE_API_KEY`, server logs |
| No artifact | `last_incident_id` missing — verify webhook returns 202 with ids |
| Zip fails on Windows | Ensure PowerShell available; path without special chars |
| Perf alerts | Send several fast `PASSED` runs then one slow run |

---

## Related

- [Webhooks API](../api/webhooks.md)
- [Artifacts & CLI](../guides/artifacts-and-cli.md)
- [CI/CD Overview](cicd-overview.md)
