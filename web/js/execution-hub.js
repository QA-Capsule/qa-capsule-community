/**
 * Unified Execution Hub — pipeline flags, test matrix, detail side-sheet.
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canPatchExecutionFlags, canViewRCA } from './roles.js';

const ENV_OPTIONS = ['UNKNOWN', 'PROD', 'STAGING', 'CANARY', 'DEV'];
const TYPE_OPTIONS = ['REAL', 'TEST-RUN', 'NIGHTLY', 'SMOKE'];
const MATRIX_MAX = 144;
const MATRIX_COLS = 12;

const reportCache = new Map();

function cacheKey(project, runId) {
    return `${project}|${runId}`;
}

function escapeHtml(s) {
    return String(s || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function isRealRunId(runId) {
    return runId && !String(runId).startsWith('legacy-');
}

function envBadgeClass(env) {
    const e = String(env || 'UNKNOWN').toUpperCase();
    if (e === 'PROD') return 'exec-flag--prod';
    if (e === 'STAGING') return 'exec-flag--staging';
    if (e === 'CANARY') return 'exec-flag--canary';
    if (e === 'DEV') return 'exec-flag--dev';
    return 'exec-flag--unknown';
}

function typeBadgeClass(typ) {
    const t = String(typ || 'REAL').toUpperCase();
    if (t === 'TEST-RUN') return 'exec-flag--testrun';
    if (t === 'NIGHTLY') return 'exec-flag--nightly';
    if (t === 'SMOKE') return 'exec-flag--smoke';
    return 'exec-flag--real';
}

export function enrichExecutionGroup(group) {
    const first = group.incidents && group.incidents[0];
    group.execution_env = (first && first.execution_env) || 'UNKNOWN';
    group.execution_type = (first && first.execution_type) || 'REAL';
    group.execution_summary = (first && first.execution_summary) || {};
    group.has_real_run = isRealRunId(group.pipeline_run_id);
    return group;
}

export function renderGroupExecFlagsHtml(group, canPatch) {
    const env = group.execution_env || 'UNKNOWN';
    const typ = group.execution_type || 'REAL';
    const sum = group.execution_summary || {};
    let html = `<div class="exec-hub-flags">`;
    html += `<span class="exec-flag-badge ${envBadgeClass(env)}" title="Execution environment">${escapeHtml(env)}</span>`;
    html += `<span class="exec-flag-badge ${typeBadgeClass(typ)}" title="Execution type">${escapeHtml(typ)}</span>`;
    if (sum.total > 0) {
        html += `<span class="exec-hub-summary">${sum.passed}/${sum.total} pass · ${sum.failed} fail · ${sum.skipped} skip</span>`;
    }
    if (canPatch && group.has_real_run) {
        const envOpts = ENV_OPTIONS.map(v =>
            `<option value="${v}" ${v === env ? 'selected' : ''}>${v}</option>`).join('');
        const typeOpts = TYPE_OPTIONS.map(v =>
            `<option value="${v}" ${v === typ ? 'selected' : ''}>${v}</option>`).join('');
        html += `<div class="exec-flag-editors">
            <label class="exec-flag-edit-label">ENV</label>
            <select class="login-input exec-flag-select" onchange="patchGroupExecFlags('${group.id}', this.value, null)">${envOpts}</select>
            <label class="exec-flag-edit-label">TYPE</label>
            <select class="login-input exec-flag-select" onchange="patchGroupExecFlags('${group.id}', null, this.value)">${typeOpts}</select>
        </div>`;
    }
    html += `</div>`;
    return html;
}

export function renderMatrixSectionHtml(group) {
    if (!group.has_real_run) {
        return `<div class="exec-matrix-section exec-matrix-section--empty"><span class="exec-matrix-hint">No pipeline run id — matrix available after CI webhook with X-Run-Id.</span></div>`;
    }
    return `<div class="exec-matrix-section">
        <div class="exec-matrix-head">
            <span class="exec-matrix-title">TEST MATRIX</span>
            <span class="exec-matrix-hint">12×12 · click cell for stacktrace / RCA</span>
        </div>
        <div id="exec-matrix-${group.id}" class="exec-test-matrix" data-project="${escapeHtml(group.project_name)}" data-run-id="${escapeHtml(group.pipeline_run_id)}">Awaiting expand…</div>
    </div>`;
}

function statusCellClass(status) {
    const s = String(status || '').toLowerCase();
    if (s === 'pass') return 'exec-matrix-cell--pass';
    if (s === 'skip') return 'exec-matrix-cell--skip';
    if (s === 'flaky') return 'exec-matrix-cell--flaky';
    return 'exec-matrix-cell--fail';
}

export function renderTestMatrixHtml(tests) {
    const slice = (tests || []).slice(0, MATRIX_MAX);
    if (!slice.length) {
        return `<div class="exec-matrix-empty">No test cases in report. Upload JUnit or send tests[] in webhook JSON.</div>`;
    }
    const cells = slice.map((t, idx) => {
        const short = (t.name || `T${idx + 1}`).replace(/^\[[^\]]+\]\s*/, '').slice(0, 24);
        const title = escapeHtml(t.name || '');
        const inc = t.incident_id ? String(t.incident_id) : '';
        return `<button type="button" class="exec-matrix-cell ${statusCellClass(t.status)}"
            title="${title}"
            data-cell-idx="${idx}"
            data-incident-id="${inc}"
            onclick="openExecutionSheetFromCell(this)">${escapeHtml(short)}</button>`;
    }).join('');
    const overflow = (tests || []).length > MATRIX_MAX
        ? `<div class="exec-matrix-overflow">+${tests.length - MATRIX_MAX} more tests not shown</div>` : '';
    return `<div class="exec-test-matrix-grid" style="--matrix-cols:${MATRIX_COLS}">${cells}</div>${overflow}`;
}

export async function fetchExecutionReport(projectName, runId) {
    const key = cacheKey(projectName, runId);
    if (reportCache.has(key)) return reportCache.get(key);
    const res = await fetchWithAuth(
        `/api/executions/${encodeURIComponent(runId)}/report?project=${encodeURIComponent(projectName)}`
    );
    const data = await parseApiJson(res);
    if (!res.ok) throw new Error(data?.error || 'Failed to load execution report');
    reportCache.set(key, data);
    return data;
}

export async function loadGroupMatrix(groupId) {
    const el = document.getElementById(`exec-matrix-${groupId}`);
    if (!el || el.dataset.loaded === '1') return;
    const project = el.dataset.project;
    const runId = el.dataset.runId;
    if (!project || !runId) return;
    el.textContent = 'Loading matrix…';
    try {
        const report = await fetchExecutionReport(project, runId);
        el.__reportTests = report.tests || [];
        el.innerHTML = renderTestMatrixHtml(el.__reportTests);
        el.dataset.loaded = '1';
    } catch (e) {
        el.innerHTML = `<div class="exec-matrix-empty">${escapeHtml(e.message || 'Matrix load failed')}</div>`;
    }
}

export function onExecGroupExpanded(groupId) {
    const container = document.getElementById(`sub-alerts-${groupId}`);
    if (container && container.style.display !== 'none') {
        loadGroupMatrix(groupId);
    }
}

export async function patchGroupExecFlags(groupId, env, type) {
    const group = window.groupedIncidents && window.groupedIncidents[groupId];
    if (!group || !group.has_real_run) return;
    const body = {};
    if (env != null) body.env = env;
    if (type != null) body.type = type;
    const res = await fetchWithAuth(
        `/api/executions/${encodeURIComponent(group.pipeline_run_id)}/flag?project=${encodeURIComponent(group.project_name)}`,
        { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
    );
    const data = await parseApiJson(res);
    if (!res.ok) {
        notify(data?.error || 'Failed to update execution flags', 'error');
        return;
    }
    reportCache.delete(cacheKey(group.project_name, group.pipeline_run_id));
    notify('Execution flags updated', 'success');
    if (typeof window.fetchIncidents === 'function') window.fetchIncidents(true);
}

export function openExecutionSheetFromCell(btn) {
    if (!btn) return;
    const incidentId = parseInt(btn.dataset.incidentId, 10) || 0;
    const matrixEl = btn.closest('.exec-test-matrix');
    const idx = parseInt(btn.dataset.cellIdx, 10);
    const tests = matrixEl && matrixEl.__reportTests ? matrixEl.__reportTests : [];
    const tc = Number.isFinite(idx) ? tests[idx] : null;

    if (incidentId > 0) {
        openExecutionSheetFromIncident(incidentId);
        return;
    }
    openExecutionSheet({
        incidentId: 0,
        name: tc ? tc.name : 'Test case',
        error_message: tc ? tc.error_message : '',
        console_logs: tc ? tc.console_logs : '',
        error_logs: tc ? tc.error_logs : '',
        project_name: matrixEl ? matrixEl.dataset.project : ''
    });
}

export function openExecutionSheetFromIncident(incidentId, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }
    const inc = (window.currentIncidents || []).find(i => String(i.id) === String(incidentId));
    if (!inc) return;
    openExecutionSheet({
        incidentId: inc.id,
        name: inc.name,
        error_message: inc.error_message,
        console_logs: inc.console_logs,
        error_logs: inc.error_logs,
        project_name: inc.project_name
    });
}

export async function openExecutionSheet(detail) {
    const sheet = document.getElementById('exec-side-sheet');
    if (!sheet) return;
    const title = document.getElementById('exec-sheet-title');
    const meta = document.getElementById('exec-sheet-meta');
    const logs = document.getElementById('exec-sheet-logs');
    const rcaEl = document.getElementById('exec-sheet-rca');
    const actions = document.getElementById('exec-sheet-actions');
    if (!title || !meta || !logs || !actions) return;

    title.textContent = (detail.name || 'Test detail').replace(/^\[FLAKY\]\s*/, '');
    meta.innerHTML = detail.project_name
        ? `<span class="exec-sheet-meta-item">${escapeHtml(detail.project_name)}</span>` : '';
    if (detail.incidentId > 0) {
        meta.innerHTML += `<span class="exec-sheet-meta-item">INC #${detail.incidentId}</span>`;
    }

    const logBody = detail.error_logs || detail.error_message || detail.console_logs || 'No logs captured.';
    logs.textContent = logBody;

    if (rcaEl) {
        rcaEl.hidden = true;
        rcaEl.innerHTML = '';
    }

    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    let actionHtml = '';
    if (detail.incidentId > 0 && canViewRCA(role)) {
        actionHtml += `<button type="button" class="btn btn-secondary btn-sm" onclick="triggerRCA(${detail.incidentId})">Run AI RCA</button>`;
    }
    actionHtml += `<button type="button" class="btn btn-secondary btn-sm" onclick="closeExecutionSheet()">Close</button>`;
    actions.innerHTML = actionHtml;

    sheet.hidden = false;
    sheet.classList.add('is-open');
    document.body.classList.add('exec-sheet-open');

    if (detail.incidentId > 0 && canViewRCA(role) && rcaEl) {
        try {
            const res = await fetchWithAuth(`/api/incidents/${detail.incidentId}/rca`);
            const rep = await parseApiJson(res);
            if (res.ok && rep && rep.summary) {
                rcaEl.hidden = false;
                rcaEl.innerHTML = `<strong>RCA</strong><p>${escapeHtml(rep.summary)}</p>`;
            }
        } catch {
            /* optional */
        }
    }
}

export function closeExecutionSheet() {
    const sheet = document.getElementById('exec-side-sheet');
    if (!sheet) return;
    sheet.hidden = true;
    sheet.classList.remove('is-open');
    document.body.classList.remove('exec-sheet-open');
}

export function clearExecutionReportCache() {
    reportCache.clear();
}
