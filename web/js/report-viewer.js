/**
 * Unified pipeline report viewer — full HTML report from GET /api/reports/{runId}
 */
import { fetchWithAuth, parseApiJson } from './api.js';
import { notify } from './ui.js';

function escapeHtml(s) {
    return String(s || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function formatDuration(ms) {
    const n = Number(ms) || 0;
    if (n <= 0) return '—';
    if (n < 1000) return `${n}ms`;
    const sec = Math.floor(n / 1000);
    const min = Math.floor(sec / 60);
    const remSec = sec % 60;
    if (min > 0) return `${min}m ${remSec}s`;
    return `${sec}s`;
}

function formatSummaryDuration(ms) {
    return formatDuration(ms);
}

function statusLabel(status) {
    const s = String(status || '').toLowerCase();
    if (s === 'pass') return 'PASS';
    if (s === 'skip') return 'SKIP';
    if (s === 'flaky') return 'FLAKY';
    return 'FAIL';
}

function statusRowClass(status) {
    const s = String(status || '').toLowerCase();
    if (s === 'pass') return 'ur-test-row--pass';
    if (s === 'skip') return 'ur-test-row--skip';
    if (s === 'flaky') return 'ur-test-row--flaky';
    return 'ur-test-row--fail';
}

function logBlockContent(test) {
    const parts = [];
    if (test.error_message) parts.push(test.error_message);
    if (test.error_logs) parts.push(test.error_logs);
    if (test.console_logs) parts.push(test.console_logs);
    return parts.join('\n\n').trim() || 'No error details captured.';
}

function hasLogContent(test) {
    return !!(test.error_message || test.error_logs || test.console_logs);
}

/** Strip framework prefix and long unified name for display. */
function displayTestName(test) {
    let n = String(test.name || '').replace(/^\[[^\]]+\]\s*/, '');
    if (test.class_name) {
        const shortName = test.name && test.name.includes('>')
            ? test.name.split('>').pop().trim()
            : n.split('>').pop().trim();
        return `${test.class_name} › ${shortName}`;
    }
    if (n.includes(' > ')) {
        return n.replace(/\s*>\s*/g, ' › ');
    }
    return n || '—';
}

function suiteKeyFor(test) {
    if (test.suite) return test.suite;
    if (test.class_name) return test.class_name;
    const n = String(test.name || '').replace(/^\[[^\]]+\]\s*/, '');
    const idx = n.indexOf(' > ');
    if (idx > 0) return n.slice(0, idx).trim();
    return 'Test Suite';
}

function buildSuiteTree(tests) {
    const tree = new Map();
    for (const t of tests) {
        const key = suiteKeyFor(t);
        if (!tree.has(key)) {
            tree.set(key, { name: key, tests: [], passed: 0, failed: 0, skipped: 0, flaky: 0, durationMs: 0 });
        }
        const node = tree.get(key);
        node.tests.push(t);
        node.durationMs += Number(t.duration_ms) || 0;
        const st = String(t.status || '').toLowerCase();
        if (st === 'pass') node.passed++;
        else if (st === 'skip') node.skipped++;
        else if (st === 'flaky') { node.flaky++; node.failed++; }
        else node.failed++;
    }
    return [...tree.values()].sort((a, b) => a.name.localeCompare(b.name));
}

function renderTestDetailBlock(test, idx) {
    const st = String(test.status || '').toLowerCase();
    const isFail = st === 'fail' || st === 'flaky';
    const openAttr = isFail ? ' open' : '';
    const logs = logBlockContent(test);
    const showLogs = isFail || hasLogContent(test);
    return `
    <details class="ur-test-block ${statusRowClass(test.status)}"${openAttr}>
        <summary class="ur-test-summary">
            <span class="ur-status-pill ur-status-pill--${escapeHtml(st)}">${statusLabel(test.status)}</span>
            <span class="ur-test-name">${escapeHtml(displayTestName(test))}</span>
            <span class="ur-test-duration">${formatDuration(test.duration_ms)}</span>
            ${test.incident_id ? `<span class="ur-test-inc">INC #${test.incident_id}</span>` : ''}
        </summary>
        ${showLogs ? `<pre class="log-block ur-test-log">${escapeHtml(logs)}</pre>` : ''}
    </details>`;
}

function renderSuiteTreeHtml(tests) {
    const suites = buildSuiteTree(tests);
    if (!suites.length) {
        return '<p class="ur-empty">No test cases in this report. Upload JUnit/XML via CI webhook <code>/api/webhooks/upload</code>.</p>';
    }
    return suites.map(suite => {
        const total = suite.tests.length;
        const openSuite = suite.failed > 0 ? ' open' : '';
        const testsHtml = suite.tests.map((t, i) => renderTestDetailBlock(t, i)).join('');
        return `
        <details class="ur-suite-block"${openSuite}>
            <summary class="ur-suite-summary">
                <span class="ur-suite-chevron" aria-hidden="true"></span>
                <span class="ur-suite-name">${escapeHtml(suite.name)}</span>
                <span class="ur-suite-stats">
                    <span class="ur-suite-stat ur-suite-stat--pass">${suite.passed} pass</span>
                    <span class="ur-suite-stat ur-suite-stat--fail">${suite.failed} fail</span>
                    ${suite.skipped ? `<span class="ur-suite-stat">${suite.skipped} skip</span>` : ''}
                    <span class="ur-suite-stat">${total} tests · ${formatDuration(suite.durationMs)}</span>
                </span>
            </summary>
            <div class="ur-suite-body">${testsHtml}</div>
        </details>`;
    }).join('');
}

function renderReportHtml(data) {
    const passed = data.status === 'passed';
    const headerClass = passed ? 'ur-header ur-header--passed' : 'ur-header ur-header--failed';
    const statusText = passed ? 'ALL TESTS PASSED' : 'FAILURES DETECTED';
    const sum = data.summary || {};
    const tests = Array.isArray(data.tests) ? data.tests : [];

    const testRows = tests.map(t => {
        const isFail = t.status === 'fail' || t.status === 'flaky';
        const logRow = isFail
            ? `<tr class="ur-test-log-row">
                <td colspan="4"><pre class="log-block">${escapeHtml(logBlockContent(t))}</pre></td>
               </tr>` : '';
        return `
        <tr class="ur-test-row ${statusRowClass(t.status)}">
            <td class="ur-col-status"><span class="ur-status-pill ur-status-pill--${escapeHtml(t.status)}">${statusLabel(t.status)}</span></td>
            <td class="ur-col-name">${escapeHtml(t.name || '—')}</td>
            <td class="ur-col-duration">${formatDuration(t.duration_ms)}</td>
            <td class="ur-col-inc">${t.incident_id ? `#${t.incident_id}` : '—'}</td>
        </tr>${logRow}`;
    }).join('');

    const matrixCells = tests.slice(0, 288).map(t =>
        `<span class="ur-matrix-dot ur-matrix-dot--${escapeHtml(t.status)}" title="${escapeHtml(t.name)}"></span>`
    ).join('');

    return `
    <article class="unified-report-article">
        <header class="${headerClass}">
            <div class="ur-header-top">
                <span class="ur-header-status">${statusText}</span>
                <span class="ur-header-outcome">${escapeHtml(String(data.outcome || '').toUpperCase())}</span>
            </div>
            <h1 class="ur-header-project">${escapeHtml(data.project_name)}</h1>
            <p class="ur-header-meta">
                <span>Run <code>${escapeHtml(data.pipeline_run_id)}</code></span>
                ${data.branch ? `<span>Branch ${escapeHtml(data.branch)}</span>` : ''}
                ${data.commit_sha ? `<span>SHA ${escapeHtml(String(data.commit_sha).slice(0, 12))}</span>` : ''}
            </p>
            <p class="ur-header-meta">
                ${data.started_at ? `<span>${escapeHtml(data.started_at)}</span>` : ''}
                ${data.finished_at ? `<span>→ ${escapeHtml(data.finished_at)}</span>` : ''}
                <span class="ur-flag">${escapeHtml(data.execution_env || 'UNKNOWN')}</span>
                <span class="ur-flag">${escapeHtml(data.execution_type || 'REAL')}</span>
                ${data.framework ? `<span>${escapeHtml(data.framework)}</span>` : ''}
            </p>
        </header>

        <div class="ur-summary-bar">
            <div class="ur-kpi"><span class="ur-kpi-label">Total</span><span class="ur-kpi-value">${sum.total || 0}</span></div>
            <div class="ur-kpi ur-kpi--pass"><span class="ur-kpi-label">Passed</span><span class="ur-kpi-value">${sum.passed || 0}</span></div>
            <div class="ur-kpi ur-kpi--fail"><span class="ur-kpi-label">Failed</span><span class="ur-kpi-value">${sum.failed || 0}</span></div>
            <div class="ur-kpi ur-kpi--skip"><span class="ur-kpi-label">Skipped</span><span class="ur-kpi-value">${sum.skipped || 0}</span></div>
            ${sum.flaky ? `<div class="ur-kpi ur-kpi--flaky"><span class="ur-kpi-label">Flaky</span><span class="ur-kpi-value">${sum.flaky}</span></div>` : ''}
            <div class="ur-kpi"><span class="ur-kpi-label">Duration</span><span class="ur-kpi-value">${formatSummaryDuration(data.duration_ms || sum.duration_ms)}</span></div>
        </div>

        ${tests.length ? `
        <section class="ur-matrix-section">
            <h2 class="ur-section-title">Test matrix</h2>
            <div class="ur-matrix-grid">${matrixCells}</div>
        </section>` : ''}

        <section class="ur-suite-tree-section">
            <h2 class="ur-section-title">Execution details (suite hierarchy)</h2>
            <p class="ur-report-hint">Structured like Robot Framework / JUnit reports: suites → tests → logs. Expand failed tests for stack traces and <code>system-out</code> / <code>system-err</code>.</p>
            <div class="ur-suite-tree">${renderSuiteTreeHtml(tests)}</div>
        </section>

        <section class="ur-table-section ur-table-section--compact">
            <details>
                <summary class="ur-section-title ur-flat-toggle">Flat table view (${tests.length} tests)</summary>
            <div class="ur-table-wrap">
                <table class="ur-tests-table">
                    <thead>
                        <tr>
                            <th>Status</th>
                            <th>Test</th>
                            <th>Duration</th>
                            <th>Incident</th>
                        </tr>
                    </thead>
                    <tbody>${testRows || '<tr><td colspan="4" class="ur-empty">No test cases in report.</td></tr>'}</tbody>
                </table>
            </div>
            </details>
        </section>
    </article>`;
}

function showReportView() {
    if (typeof window.stopDashboardAutoRefresh === 'function') {
        window.stopDashboardAutoRefresh();
    }
    document.querySelectorAll('.view-section').forEach(x => {
        x.style.display = 'none';
        x.classList.remove('active');
    });
    document.querySelectorAll('.nav-item').forEach(x => x.classList.remove('active'));
    const section = document.getElementById('view-unified-report');
    if (section) {
        section.style.display = 'block';
        section.classList.add('active');
    }
}

export async function openReport(groupId, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }
    const group = window.groupedIncidents && window.groupedIncidents[groupId];
    if (!group) {
        notify('Execution group not found', 'error');
        return;
    }
    if (!group.has_real_run) {
        notify('Full report requires a pipeline run id from CI webhook (X-Run-Id)', 'warning');
        return;
    }

    const root = document.getElementById('unified-report-root');
    if (!root) return;

    showReportView();
    root.innerHTML = '<div class="unified-report-loading">Loading unified report…</div>';

    try {
        const res = await fetchWithAuth(
            `/api/reports/${encodeURIComponent(group.pipeline_run_id)}?project=${encodeURIComponent(group.project_name)}`
        );
        const { data } = await parseApiJson(res);
        if (!res.ok) {
            throw new Error((data && data.error) || 'Failed to load report');
        }
        root.innerHTML = renderReportHtml(data);
    } catch (e) {
        root.innerHTML = `<div class="unified-report-error">
            <p>${escapeHtml(e.message || 'Report load failed')}</p>
            <button type="button" class="btn btn-secondary" onclick="closeUnifiedReport()">Back to Dashboard</button>
        </div>`;
        notify(e.message || 'Report load failed', 'error');
    }
}

export function closeUnifiedReport() {
    const dashNav = document.querySelector('.nav-item[onclick*="switchView(\'dashboard\'"]');
    if (typeof window.switchView === 'function') {
        window.switchView('dashboard', dashNav);
    }
}
