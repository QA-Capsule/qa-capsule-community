---
icon: material/book-open-page-variant
---

# Platform User Guide (Complete)

End-to-end guide for **every major feature** in QA Capsule: Operations dashboard, CI/CD Gateways, Plugin Engine, Visual Workflows, AI RCA, Quarantine, Runbooks, DORA, FinOps, and Help Center.

For internal mechanics, see [System Architecture](architecture.md).

---

## 1. Roles and navigation

| Role | Code | Typical navigation |
|------|------|-------------------|
| Platform Admin | `admin` | Workspaces, IAM, Settings, Help Center |
| Manager | `manager` | Operations, Gateways, Plugins, FinOps, DORA, Runbooks |
| Lead | `lead` | Operations, Gateways, Plugins, Workflow, Quarantine, RCA |
| Observer | `observer` | Operations (read-only), RCA read, Quarantine read |

After login, the sidebar shows only views your role may access. Observers cannot resolve, delete, edit gateways, or save workflows.

---

## 2. Operations dashboard (Telemetry Stream)

### Purpose

Monitor live CI failures, grouped by **pipeline execution**, resolve or delete alerts, export logs.

### Time range

Use the toolbar preset (5m → 30d, Today, Custom). All KPIs and the incident list respect the selected window. Auto-refresh interval adapts to the range.

### KPI cards

| KPI | Meaning |
|-----|---------|
| Active | Unresolved incidents in range |
| Resolved | Resolved incidents |
| Health % | Resolved / total in current filter |
| Flaky | Count with `[FLAKY]` prefix |

### Pipeline execution card

One card = one `pipeline_run_id` (or legacy time bucket) with multiple failed tests inside.

| Action | Who | Effect |
|--------|-----|--------|
| Resolve execution | Lead+ | Marks all active tests in group resolved |
| Delete execution | Manager+ | Permanently removes incidents |
| Export errors / full / JUnit | All with access | Downloads logs for the group |
| Expand alerts | All | Shows per-test rows and log panels |

### Bulk selection

Check individual tests or use **select all** → **Resolve** or **Delete** (role-gated).

### Search and project filter

Client-side search on name, project, error text. Project dropdown limits API fetch to one gateway.

### Analytics panel

**Analytics** toggles charts (incidents, flaky ratio, MTTR, etc.). **Customize layout** saves per-user tile order and chart types — uses dashboard time filter, not a separate query language.

---

## 3. CI/CD Gateways (Ingestion)

**Path:** Settings area → **CI/CD Gateways** (Manager/Lead).

### Provision a gateway

1. Create project row: name, team, CI system, repo path.
2. Copy generated **API key** — store in CI secrets as `QACAPSULE_API_KEY`.
3. Configure webhook URL: `POST https://<host>:9000/api/webhooks/<optional-suffix>` with header `X-API-Key`.

### Recommended headers

| Header | Purpose |
|--------|---------|
| `X-API-Key` | Required authentication |
| `X-Run-Id` | Correlate all tests in one pipeline run |
| `X-Commit-Sha` / `X-Git-Commit` | Commit for quarantine + DORA |
| `X-Branch` / `X-Git-Branch` | Branch label on `pipeline_runs` |

### SRE routing matrix

**Add configuration** per integration:

- Select plugin from **active** registry list (Manager enabled routing on plugin).
- Fill routing fields (`SLACK_CHANNEL`, `JIRA_PROJECT_KEY`, `TEAMS_WEBHOOK_URL`, …).
- Saved in `sre_routing_json` — defines **allowed** plugin paths for auto-run and workflow actions.

### Workflow column badges

| Badge | Meaning |
|-------|---------|
| Legacy | No saved DAG — linear AUTO-RUN only |
| Draft | DAG saved but `enabled: false` |
| Active | DAG enabled — replaces linear auto-run |

### WORKFLOW button

Opens the Visual Workflow Editor for that project (Lead+ edit, Observer read-only).

---

## 4. Plugin Engine

**Path:** Plugins (Manager/Lead).

### Lifecycle

1. **Manager** opens Plugin Engine → sees all manifests from `plugins/`.
2. **Configure & Save** — writes env template to manifest (secrets should live in runner env, not committed).
3. **Enable for gateways** — sets `routing_enabled` so the plugin appears in gateway routing dropdown.
4. **AUTO-RUN toggle** — legacy linear trigger on keyword match (when workflow not active).

### Manual test

**Execute** sends `POST /api/plugins/run` with `file_path` — useful to validate Slack/Jira tokens before enabling automation.

### Categories

Slack, Teams, Jira, PagerDuty, Opsgenie, VictorOps, Datadog, email, GitHub Actions, generic webhook, Kubernetes rollout, test management tools — see [Integrations catalog](../plugins/integrations-catalog.md).

---

## 5. Visual Workflow Builder (step-by-step)

**Access:** Gateways table → **WORKFLOW** (Lead, Manager, Admin).

### Step 1 — Open editor

Modal shows Drawflow canvas, toolbar, and right help/simulate panel.

### Step 2 — Build the DAG

1. Click **+ Trigger** (exactly one trigger per workflow).
2. Click **+ Condition** — choose preset (e.g. Tag `[FLAKY]`, status CRITICAL, error contains `timeout`).
   - Connect from trigger **output** to condition **input**.
   - **Top output** = true branch; **bottom output** = false branch.
3. Click **+ Action** — pick integration from dropdown (only gateway-allowed, routing-active plugins).
4. Wire condition outputs to actions.

### Step 3 — Example workflow

Click **Example** — loads: Trigger → `[FLAKY]` condition → Slack (true) / Jira (false).

### Step 4 — Simulate (dry-run)

1. Fill **Test name**, **Status**, **Error** in the side panel.
2. Click **Simulate** or **Run simulation**.
3. Read path: which nodes visited, which plugins **would** run, which **skipped** (not on gateway, empty path, etc.).
4. Simulation uses **canvas JSON** — save not required.

### Step 5 — Save and enable

| Control | Behavior |
|---------|----------|
| **Enable workflow** checked + **Save** | Validates DAG (no cycles, known paths, gateway allow-list) → active remediation |
| **Enable** unchecked + **Save** | Draft only — legacy AUTO-RUN still runs |
| **Reset** | Deletes DAG from DB — returns to Legacy mode |
| **Fit** | Re-centers zoom on graph |

### Step 6 — Verify in CI

Send a webhook failure matching a branch — check logs for `workflow action` or `workflow action skipped`.

Full reference: [Visual Workflow Builder](../plugins/visual-workflow.md).

---

## 6. AI RCA & Insights

**Path:** RCA & AI Insights (Lead+ read; Manager configures AI).

### Configure provider (Manager+)

1. Open AI config panel.
2. Choose provider: OpenAI, Ollama, or Disabled.
3. Set model name, base URL (Ollama), API key env var name.
4. Save — stored in `ai_provider_config`.

### Automatic analysis

On each **new failure** (not quarantined-only path), async job:

- Builds prompt from test name, error logs, console.
- Stores summary in **RCA report** linked to incident.
- UI shows status: pending, running, completed, skipped, failed.

### Manual re-run

From incident detail or RCA view — trigger analysis again.

### Insights list

`GET /api/rca/insights` powers the table of recent summaries across projects you can access.

Details: [AI RCA & Quarantine](intelligence-quarantine.md).

---

## 7. Smart Quarantine

**Path:** Quarantine (Lead+ manage; Observer read).

### Why

Stop alert fatigue on chronically flaky tests and let CI skip known-bad tests.

### Auto-quarantine

Engine increments stats on each ingest transition. May auto-add when:

- Test already tagged `[FLAKY]`, or
- Flaky count / consecutive failures exceed policy.

### Manual quarantine

Add test by name under a project → creates deny-list entry.

### Lift

Remove test from deny-list when fixed — **Lift** button.

### Ingest behavior

Quarantined tests:

- Do **not** create incidents.
- Do **not** trigger Slack/Jira/workflow.
- Webhook JSON includes `quarantined_skipped`.

### CI integration

```http
GET /api/ci/quarantine?project=my-suite
X-API-Key: <gateway-key>
```

Returns list of `test_name` + fingerprint for pipeline skip logic.

---

## 8. Runbooks

**Path:** Runbooks (Lead+).

### What they are

Pre-built **workflow templates** (502 restart, flaky triage, OOM, perf regression, …) maintained in Go — not arbitrary shell.

### Apply a runbook

1. Select gateway (project).
2. Pick template (e.g. `flaky-triage`).
3. **Apply** — writes validated `sre_workflow_json` and enables workflow.

Overrides manual canvas for that project until you edit or reset.

API details: [Runbooks & DORA](runbooks-dora.md).

---

## 9. DORA & Executive Dashboard

**Path:** DORA (Manager).

### Metrics

| Metric | Source |
|--------|--------|
| Deployment frequency | `pipeline_runs` / day |
| Lead time | Median minutes from run start → first incident |
| Change failure rate | Failed runs / total runs |
| MTTR | Mean minutes to resolve incidents |

Filtered by time range and optional project.

### Prometheus correlation

Ingest Alertmanager payloads via `POST /api/webhooks/prometheus?project=…`. Signals appear in DORA view and may correlate to incidents ±15 min.

---

## 10. FinOps

**Path:** FinOps (Manager).

Models cost from:

- Configurable developer hourly rate.
- CI minute cost.
- Investigation time per incident.

Shows total cost, flaky-attributed waste, export PDF. See Help Center → **FinOps metrics** for formulas.

---

## 11. Artifacts & CLI

- Upload Playwright traces / screenshots: `POST /api/incidents/{id}/artifacts`.
- Local CLI wraps tests and checks flaky fingerprint: [Artifacts & CLI](artifacts-and-cli.md).

---

## 12. Help Center (in-app)

**Path:** Sidebar → Help Center (all roles).

Topics cover overview, roles, plugins, **visual workflows**, RCA/quarantine, runbooks/DORA, operations, architecture summary, FinOps formulas, glossary.

Use **?** in the workflow editor for quick remediation help.

---

## Quick troubleshooting

| Symptom | Check |
|---------|-------|
| No Slack on failure | Workflow enabled? Plugin on gateway? Allowed path? Legacy AUTO-RUN + trigger keyword? |
| Duplicate alerts | Same `X-Run-Id`? Dedup uses fingerprint+run. |
| `[FLAKY]` prefix | Resolved same test in last 48h. |
| Simulation ≠ production | Save workflow; gateway allow-list same as simulate body. |
| Login works but empty projects | Team membership for Lead/Observer. |

---

## Documentation index

| Topic | Document |
|-------|----------|
| Architecture | [architecture.md](architecture.md) |
| Webhooks | [webhooks.md](../api/webhooks.md) |
| Incident lifecycle | [incident-lifecycle.md](incident-lifecycle.md) |
| RBAC | [rbac-teams.md](../setup/rbac-teams.md) |
