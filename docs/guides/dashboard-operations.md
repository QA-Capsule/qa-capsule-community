# Dashboard Operations Guide

The **Telemetry Stream** dashboard is the primary interface for monitoring, filtering, resolving, and exporting pipeline failures.

---

## Dashboard Layout

| Area | Description |
|---|---|
| **KPI Cards** | Active alerts, resolved (48h), pipeline health %, MTTR |
| **Toolbar** | Search, project filter, status filter, action buttons |
| **Pipeline cards** | Grouped executions with sub-alerts inside |
| **Analytics panel** | Toggle via **Analytics** button — charts and weekly reports |

---

## Toolbar Controls

All toolbar controls share a unified height and dark-theme styling for visual consistency.

| Control | Function |
|---|---|
| **Search** | Filter by test name, project name, or error message (client-side, instant) |
| **Filter** | Restrict to a specific provisioned project |
| **Status: All / Active / Resolved** | Filter incidents by resolution state |
| **Refresh** | Force-fetch incidents from the API (bypasses poll cache) |
| **Analytics** | Show/hide the analytics charts panel |
| **CSV / PDF** | Download weekly health report |

---

## Pipeline Execution Groups

Incidents are automatically grouped into **Pipeline Executions** when they share:

- The same `project_name`
- Created within **120 seconds** of each other
- Sequential incident IDs within a range of 100

Each group card shows:

- **Project name** and timestamp
- **Badge** — `EXECUTION RESOLVED` (green) or `N ACTIVE / M TOTAL` (red)
- **Actions** dropdown — resolve, delete, export logs
- **Alert(s)** toggle — expand/collapse sub-alerts

---

## Sub-Alerts

Each failed test case appears as a **sub-alert** inside its pipeline group.

| Element | Meaning |
|---|---|
| Green left border + `RESOLVED BY {user}` | Test acknowledged and resolved |
| Red left border + `ACTIVE TEST` | Open failure requiring attention |
| `[FLAKY]` badge | Test failed after being resolved within 48h |
| Checkbox | Select for bulk actions |
| Log panel | `error_logs` or `error_message` monospace display |

---

## Resolving Incidents

### Resolve a single sub-alert

1. Expand the pipeline group (**Alert(s)** button).
2. Check the checkbox on the sub-alert you want to resolve.
3. Click **Actions → Resolve Execution** (resolves only selected sub-alerts).
4. Or use the sticky **Resolve** banner button when sub-alerts are selected.

### Resolve an entire pipeline execution

1. Do **not** select any checkboxes.
2. Click **Actions → Resolve Execution** on the pipeline card.
3. All active sub-alerts in that group are resolved.

### What happens on resolve

1. UI updates immediately (optimistic green state).
2. `PUT /api/incidents` persists `is_resolved = 1` in SQLite.
3. Polling confirms server state; resolution survives page refresh.
4. Future CI uploads with the same fingerprint are **suppressed** (no duplicate re-opening).

!!! info "Viewer role cannot resolve"
    Users with the `viewer` role can read incidents but cannot resolve or delete them.

---

## Bulk Actions

When one or more sub-alerts are checked, a sticky banner appears at the top:

| Button | Action |
|---|---|
| **Resolve** | Resolves all selected sub-alerts |
| **Delete** | Permanently deletes selected incidents (Admin only) |

The **select-all** checkbox in the banner selects every incident on the current filtered view.

---

## Exporting Logs

From the **Actions** dropdown on any pipeline card:

| Export | Content |
|---|---|
| **Export Errors** | Error messages only |
| **Export Full Logs** | Complete console + error logs |
| **Generate JUnit XML** | Reconstruct JUnit XML from stored incident data |

---

## Status Filter Behavior

| Filter | Shows |
|---|---|
| **All** | Every incident (active + resolved) |
| **Active** | Only `is_resolved = false` |
| **Resolved** | Only `is_resolved = true` |

KPI counters update to reflect the filtered dataset, not the global database totals.

---

## Real-Time Polling

The dashboard polls `/api/incidents` every **3 seconds** when the Telemetry Stream view is active. Polling is automatically paused for 30 seconds during resolve/delete operations to prevent UI flicker.

---

## Role-Based Visibility

| Role | Visible incidents |
|---|---|
| **Admin** | All projects |
| **Operator / Viewer** | Only projects linked to the user's teams |

Use **Organizations** in the sidebar to assign users to teams and link projects to teams via CI/CD Gateways.
