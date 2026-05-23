/**
* web/app.js
* Main controller for QA Capsule Control Plane
*/

// IMPORT MODULES
import { notify, showConfirmModal, showPromptModal, closeModal, toggleTheme, initSidebar, toggleSidebar } from './js/ui.js';
import { parseJwt, performLogout, showLoginScreen, fetchWithAuth, parseApiJson, isApiOffline, pingApiServer, resetApiOfflineBanner } from './js/api.js';
import * as iam from './js/iam.js';
import * as settings from './js/settings.js';
import * as profile from './js/profile.js';
import * as finops from './js/finops.js';
import * as about from './js/about.js';
import * as analyticsLayout from './js/analytics-layout.js';
import { applyRoleVisibility, canAccessFinOps, canAccessPlugins, canResolveIncidents, canDeleteIncidents, hasMinRole, roleLabel, canManageTeams, canManageIAM, isAdmin, canAccessView, accessDeniedMessage, defaultViewForRole, canManagePluginAutoRun } from './js/roles.js';
import * as workflowEditor from './js/workflow-editor.js';
<<<<<<< HEAD
import * as rca from './js/rca.js';
import * as quarantine from './js/quarantine.js';
import * as runbooks from './js/runbooks.js';
import * as dora from './js/dora.js';
=======
>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
import { setupAutocomplete } from './js/autocomplete.js';
import { initTheme } from './js/ui.js';

// EXPORT GLOBALLY FOR HTML INLINE HANDLERS
Object.assign(window, { notify, showConfirmModal, showPromptModal, closeModal, toggleTheme, initSidebar, toggleSidebar, parseJwt, performLogout, fetchWithAuth });

// Bind all module functions to the window so HTML 'onclick' can find them
for (const [key, value] of Object.entries(iam)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(settings)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(profile)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(finops)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(about)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(analyticsLayout)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(workflowEditor)) {
    if (typeof value === 'function') window[key] = value;
}
<<<<<<< HEAD
for (const [key, value] of Object.entries(rca)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(quarantine)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(runbooks)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(dora)) {
    if (typeof value === 'function') window[key] = value;
}
=======

>>>>>>> 70a3559fb4d4fbfe14293d19734d53e04a1553fb
// ==========================================
// VARIABLES GLOBALES & FINOPS SRE
// ==========================================
window.currentIncidents = [];
window.selectedIncidents = new Set();
window.pausePollingUntil = 0;
window.statusFilter = 'all';
window.pendingResolvedIds = new Map();
window._resolveRetryInFlight = false;
window.groupedIncidents = {};

// FINOPS GLOBALS
window.isEnterpriseActive = false;
window.currencySymbols = { "USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥", "AUD": "A$", "CAD": "C$", "CHF": "Fr", "INR": "₹", "CNY": "¥", "MXN": "$", "SGD": "S$", "NZD": "NZ$" };
window.selectedCurrency = "USD"; 

// ==========================================
// INCIDENTS LOGIC
// ==========================================
function loadPendingResolvedFromStorage() {
    try {
        const raw = sessionStorage.getItem('qacapsule-pending-resolved');
        if (!raw) return;
        window.pendingResolvedIds = new Map(JSON.parse(raw));
    } catch (_) {
        window.pendingResolvedIds = new Map();
    }
}

function savePendingResolvedToStorage() {
    try {
        sessionStorage.setItem('qacapsule-pending-resolved', JSON.stringify([...window.pendingResolvedIds]));
    } catch (_) { }
}

function normalizeIsResolved(inc) {
    return inc.is_resolved === true || inc.is_resolved === 1 || inc.is_resolved === '1';
}

window.markIncidentsPendingResolve = function (ids, resolvedBy) {
    ids.forEach(id => {
        window.pendingResolvedIds.set(String(id), { resolvedBy: resolvedBy || 'You', since: Date.now() });
    });
    savePendingResolvedToStorage();
};

window.applyOptimisticResolve = function (ids, resolvedBy) {
    const idSet = new Set(ids.map(id => String(id)));
    window.currentIncidents.forEach(inc => {
        if (idSet.has(String(inc.id))) {
            inc.is_resolved = true;
            inc.status = 'resolved';
            inc.resolved_by = resolvedBy;
        }
    });
};

window.mergePendingResolvedState = function (incidents) {
    if (!incidents || window.pendingResolvedIds.size === 0) return incidents;
    return incidents.map(inc => {
        const pending = window.pendingResolvedIds.get(String(inc.id));
        if (!pending) return inc;
        if (normalizeIsResolved(inc)) {
            window.pendingResolvedIds.delete(String(inc.id));
            savePendingResolvedToStorage();
            return inc;
        }
        return { ...inc, is_resolved: true, status: 'resolved', resolved_by: inc.resolved_by || pending.resolvedBy };
    });
};

window.confirmPendingResolvedFromServer = function (rawIncidents) {
    const needsRetry = [];
    (rawIncidents || []).forEach(inc => {
        const key = String(inc.id);
        if (!window.pendingResolvedIds.has(key)) return;
        if (normalizeIsResolved(inc)) {
            window.pendingResolvedIds.delete(key);
        } else {
            const pending = window.pendingResolvedIds.get(key);
            if (pending && Date.now() - pending.since >= 2000) {
                const numId = parseInt(inc.id, 10);
                if (!isNaN(numId)) needsRetry.push(numId);
            }
        }
    });
    savePendingResolvedToStorage();
    return [...new Set(needsRetry)];
};

window.resolveIncidentsByIds = async function (ids, successMessage) {
    const uniqueIds = [...new Set(ids.map(id => parseInt(id, 10)).filter(id => !isNaN(id)))];
    if (uniqueIds.length === 0) return false;

    const currentUser = parseJwt(localStorage.getItem('sre-jwt')).username || 'You';
    window.pausePollingUntil = Date.now() + 30000;

    window.markIncidentsPendingResolve(uniqueIds, currentUser);
    window.applyOptimisticResolve(uniqueIds, currentUser);
    window.renderIncidentsList();

    try {
        const res = await window.fetchWithAuth('/api/incidents', { method: 'PUT', body: JSON.stringify({ ids: uniqueIds }) });
        if (!res.ok) throw new Error('Update failed on server side');

        notify(successMessage || 'Alerts resolved', 'success');

        for (let attempt = 0; attempt < 8; attempt++) {
            await new Promise(r => setTimeout(r, attempt === 0 ? 600 : 800));
            await window.fetchIncidents(true, { skipPauseCheck: true });
            const allConfirmed = uniqueIds.every(id => !window.pendingResolvedIds.has(String(id)));
            if (allConfirmed) break;
        }

        if (uniqueIds.some(id => window.pendingResolvedIds.has(String(id))) && !window._resolveRetryInFlight) {
            window._resolveRetryInFlight = true;
            try {
                await window.fetchWithAuth('/api/incidents', { method: 'PUT', body: JSON.stringify({ ids: uniqueIds }) });
                await new Promise(r => setTimeout(r, 800));
                await window.fetchIncidents(true, { skipPauseCheck: true });
            } finally { window._resolveRetryInFlight = false; }
        }

        window.pausePollingUntil = Date.now() + 5000;
        await window.fetchMetricsOnly();
        return true;
    } catch (e) {
        notify('Failed to resolve incident(s)', 'error');
        window.pausePollingUntil = Date.now() + 5000;
        await window.fetchIncidents(true, { skipPauseCheck: true });
        return false;
    }
};

loadPendingResolvedFromStorage();

window.toggleIncidentSelection = function (id, checked) {
    window.pausePollingUntil = Date.now() + 15000;
    const strId = String(id);
    if (checked) window.selectedIncidents.add(strId);
    else window.selectedIncidents.delete(strId);

    const masterCb = document.getElementById('select-all-cb');
    if (masterCb) masterCb.checked = window.currentIncidents.length > 0 && window.currentIncidents.every(inc => window.selectedIncidents.has(String(inc.id)));
    window.updateBulkActionUI();
};

window.toggleGroupSelection = function (groupId, checked) {
    window.pausePollingUntil = Date.now() + 15000;
    const group = window.groupedIncidents[groupId];
    if (!group) return;

    group.incidents.forEach(inc => {
        const strId = String(inc.id);
        const cb = document.getElementById(`cb-inc-${inc.id}`);
        if (cb) cb.checked = checked;
        if (checked) window.selectedIncidents.add(strId);
        else window.selectedIncidents.delete(strId);
    });

    const masterCb = document.getElementById('select-all-cb');
    if (masterCb) masterCb.checked = window.currentIncidents.length > 0 && window.currentIncidents.every(inc => window.selectedIncidents.has(String(inc.id)));
    window.updateBulkActionUI();
};

window.toggleSelectAll = function (checked) {
    window.pausePollingUntil = Date.now() + 15000;
    window.currentIncidents.forEach(inc => {
        const strId = String(inc.id);
        if (checked) window.selectedIncidents.add(strId);
        else window.selectedIncidents.delete(strId);
        const cb = document.getElementById(`cb-inc-${inc.id}`);
        if (cb) cb.checked = checked;
    });
    document.querySelectorAll('[id^="cb-group-"]').forEach(cb => cb.checked = checked);
    window.updateBulkActionUI();
};

window.updateBulkActionUI = function () {
    const count = window.selectedIncidents.size;
    const banner = document.getElementById('bulk-action-banner');
    const label = document.getElementById('bulk-count-label');

    if (count === 0) {
        if (banner) banner.style.display = 'none';
    } else {
        if (banner) {
            banner.style.display = 'flex';
            if (label) label.innerText = `${count} Test(s) Selected`;
        }
    }
};

window.resolveSelected = async function () {
    if (window.selectedIncidents.size === 0) return;
    const ids = Array.from(window.selectedIncidents).map(id => parseInt(id, 10)).filter(id => !isNaN(id));
    if (ids.length === 0) return;
    window.selectedIncidents.clear();
    window.updateBulkActionUI();
    await window.resolveIncidentsByIds(ids, 'Alerts resolved');
};

window.deleteSelected = async function () {
    if (window.selectedIncidents.size === 0) return notify('Select at least one alert.', 'error');
    const role = parseJwt(localStorage.getItem('sre-jwt')).role;
    if (!canDeleteIncidents(role)) return notify('Manager or Lead role required to delete alerts.', 'error');

    showConfirmModal('Delete Selected?', `Permanently delete ${window.selectedIncidents.size} alert(s)?`, 'danger', async function () {
        window.pausePollingUntil = Date.now() + 15000;
        const ids = Array.from(window.selectedIncidents).map(id => parseInt(id, 10)).filter(id => !isNaN(id));
        window.currentIncidents = window.currentIncidents.filter(inc => !ids.includes(parseInt(inc.id, 10)));
        window.selectedIncidents.clear();
        window.renderIncidentsList();
        notify('Deleting alerts…', 'success');

        try {
            const res = await window.fetchWithAuth(`/api/incidents?ids=${ids.join(',')}`, { method: 'DELETE' });
            if (!res.ok) {
                const err = await res.text();
                notify(err || 'Delete failed', 'error');
                window.pausePollingUntil = 0;
                window.fetchIncidents(true);
                return;
            }
            setTimeout(() => { window.pausePollingUntil = 0; window.fetchIncidents(true); window.fetchMetricsOnly(); }, 800);
        } catch (e) {
            window.pausePollingUntil = 0;
            window.fetchIncidents(true);
            notify('Delete failed', 'error');
        }
    });
};

window.resolveGroup = async function (groupId, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    const dropdown = document.getElementById(`action-dropdown-${groupId}`);
    if (dropdown) dropdown.style.display = 'none';

    const group = window.groupedIncidents[groupId];
    if (!group) return;

    const selectedInGroup = group.incidents.filter(inc => window.selectedIncidents.has(String(inc.id)));
    const targetIncidents = selectedInGroup.length > 0 ? selectedInGroup : group.incidents;

    const ids = targetIncidents.filter(inc => !normalizeIsResolved(inc)).map(inc => parseInt(inc.id, 10)).filter(id => !isNaN(id));
    if (ids.length === 0) return;

    targetIncidents.forEach(i => window.selectedIncidents.delete(String(i.id)));
    window.updateBulkActionUI();
    await window.resolveIncidentsByIds(ids, ids.length === 1 ? 'Sub-alert resolved.' : 'Pipelines marked as resolved.');
}

window.deleteGroup = async function (groupId, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    const role = parseJwt(localStorage.getItem('sre-jwt')).role;
    if (!canDeleteIncidents(role)) return notify('Manager or Lead role required to delete alerts.', 'error');

    const dropdown = document.getElementById(`action-dropdown-${groupId}`);
    if (dropdown) dropdown.style.display = 'none';

    const group = window.groupedIncidents[groupId];
    if (!group) return;

    showConfirmModal('Delete Pipeline Execution?', 'Permanently delete this pipeline execution?', 'danger', async function () {
        window.pausePollingUntil = Date.now() + 15000;
        const ids = group.incidents.map(i => parseInt(i.id, 10)).filter(id => !isNaN(id));

        window.currentIncidents = window.currentIncidents.filter(inc => !ids.includes(parseInt(inc.id, 10)));
        group.incidents.forEach(i => window.selectedIncidents.delete(String(i.id)));

        window.renderIncidentsList();
        notify("Suppression du pipeline en cours...", "success");

        try {
            const res = await window.fetchWithAuth(`/api/incidents?ids=${ids.join(',')}`, { method: 'DELETE' });
            if (!res.ok) {
                notify('Delete failed', 'error');
                window.pausePollingUntil = 0;
                window.fetchIncidents(true);
                return;
            }
            setTimeout(() => { window.pausePollingUntil = 0; window.fetchIncidents(true); window.fetchMetricsOnly(); }, 800);
        } catch (e) {
            window.pausePollingUntil = 0;
            window.fetchIncidents(true);
            notify('Delete failed', 'error');
        }
    });
};

window.toggleActionDropdown = function (id, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    const dropdown = document.getElementById(`action-dropdown-${id}`);
    if (dropdown) {
        const isCurrentlyHidden = !dropdown.classList.contains('is-open');
        document.querySelectorAll('.action-dropdown-menu').forEach(el => el.classList.remove('is-open'));
        if (isCurrentlyHidden) { dropdown.classList.add('is-open'); window.pausePollingUntil = Date.now() + 15000; }
        else { window.pausePollingUntil = 0; }
    }
}

document.addEventListener('click', function (event) {
    if (!event.target.closest('.action-dropdown-container')) {
        document.querySelectorAll('.action-dropdown-menu').forEach(el => el.classList.remove('is-open'));
    }
});

window.downloadGroupLog = function (groupId, type, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    document.getElementById(`action-dropdown-${groupId}`).style.display = 'none';
    window.pausePollingUntil = 0;

    const group = window.groupedIncidents[groupId];
    if (!group) return notify("Could not retrieve group data.", "error");

    let fileContent = ""; let fileName = ""; let blobType = "text/plain";

    if (type === 'error') {
        fileName = `execution_${groupId}_errors.log`;
        group.incidents.forEach(inc => { fileContent += `--- [TEST: ${inc.name}] ---\n${inc.error_logs || inc.error_message || "No error details."}\n\n`; });
    } else if (type === 'full') {
        fileName = `execution_${groupId}_full_logs.log`;
        group.incidents.forEach(inc => { fileContent += `=======================================\nTEST: ${inc.name}\n=======================================\n[ERROR LOGS]\n${inc.error_logs || inc.error_message || "N/A"}\n\n[STDOUT]\n${inc.console_logs || "N/A"}\n\n`; });
    } else if (type === 'xml') {
        fileName = `execution_${groupId}_results.xml`; blobType = "application/xml";
        fileContent = `<?xml version="1.0" encoding="UTF-8"?>\n<testsuites>\n  <testsuite name="QA_Capsule_Reconstructed_Suite" tests="${group.incidents.length}" failures="${group.incidents.length}">\n`;
        group.incidents.forEach(inc => {
            const safeName = (inc.name || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
            const safeErr = (inc.error_message || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            const safeLogs = (inc.error_logs || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            const safeOut = (inc.console_logs || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            fileContent += `    <testcase name="${safeName}">\n      <failure message="${safeErr}">${safeLogs}</failure>\n`;
            if (safeOut) fileContent += `      <system-out>${safeOut}</system-out>\n`;
            fileContent += `    </testcase>\n`;
        });
        fileContent += `  </testsuite>\n</testsuites>`;
    }

    const blob = new Blob([fileContent], { type: blobType });
    const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = fileName;
    document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url);
}

window.toggleSubAlerts = function (groupId, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    window.pausePollingUntil = Date.now() + 15000;
    const container = document.getElementById(`sub-alerts-${groupId}`);
    const icon = document.getElementById(`toggle-icon-${groupId}`);
    if (container.style.display === 'none') { container.style.display = 'block'; icon.style.transform = 'rotate(180deg)'; }
    else { container.style.display = 'none'; icon.style.transform = 'rotate(0deg)'; }
}

window.__logsExpandAll = null;

window.expandAllIncidentLogs = function () {
    window.__logsExpandAll = true;
    window.pausePollingUntil = Date.now() + 15000;
    window.renderIncidentsList();
};

window.collapseAllIncidentLogs = function () {
    window.__logsExpandAll = false;
    window.pausePollingUntil = Date.now() + 15000;
    window.renderIncidentsList();
};

window.toggleIncidentLog = function (id, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    window.__logsExpandAll = null;
    window.pausePollingUntil = Date.now() + 15000;
    const logContent = document.getElementById(`log-content-${id}`);
    const logIcon = document.getElementById(`log-icon-${id}`);
    const logText = document.getElementById(`log-text-${id}`);

    if (logContent.style.display === 'none') {
        logContent.style.display = 'block';
        if (logIcon) logIcon.style.transform = 'rotate(180deg)';
        if (logText) logText.innerText = 'Collapse logs';
    } else {
        logContent.style.display = 'none';
        if (logIcon) logIcon.style.transform = 'rotate(0deg)';
        if (logText) logText.innerText = 'Show logs';
    }
};

window.loadDashboardFilters = function () {
    const filter = document.getElementById('project-filter');
    if (!filter) return;
    return fetchWithAuth(`/api/my-projects?_ts=${Date.now()}`)
        .then(res => parseApiJson(res))
        .then(({ ok, data: projects }) => {
            if (!ok) throw new Error('projects');
            const currentVal = filter.value;
            filter.innerHTML = '<option value="all">All My Projects</option>';
            const list = Array.isArray(projects) ? projects : [];
            list.forEach(p => filter.innerHTML += `<option value="${p.name}">${p.name}</option>`);
            if (currentVal) filter.value = currentVal;
        })
        .catch(() => {
            if (filter.options.length <= 1) {
                filter.innerHTML = '<option value="all">All My Projects</option>';
            }
        });
}

window.setStatusFilter = function (status) {
    window.statusFilter = status;
    const allBtn = document.getElementById('status-all-btn'); const activeBtn = document.getElementById('status-active-btn'); const resolvedBtn = document.getElementById('status-resolved-btn');
    [allBtn, activeBtn, resolvedBtn].forEach(btn => { if (btn) btn.classList.remove('active-all', 'active-active', 'active-resolved'); });
    if (status === 'all' && allBtn) allBtn.classList.add('active-all');
    else if (status === 'active' && activeBtn) activeBtn.classList.add('active-active');
    else if (status === 'resolved' && resolvedBtn) resolvedBtn.classList.add('active-resolved');
    if (localStorage.getItem('sre-jwt')) window.fetchIncidents(true);
}

window.fetchMetricsOnly = function () {
    if (isApiOffline()) return;
    fetchWithAuth(`/api/metrics?_ts=${Date.now()}`)
        .then(r => parseApiJson(r))
        .then(({ ok, data: metrics }) => {
            if (!ok || !metrics) return;
            if (metrics) {
                document.getElementById('kpi-active').innerText = metrics.total_incidents - metrics.resolved_incidents;
                document.getElementById('kpi-resolved').innerText = metrics.resolved_incidents;
                document.getElementById('kpi-health').innerText = `${metrics.total_incidents > 0 ? Math.round((metrics.resolved_incidents / metrics.total_incidents) * 100) : 100}%`;
                
                const flakyEl = document.getElementById('kpi-flaky');
                if (flakyEl) flakyEl.innerText = metrics.flaky_tests ?? 0;
            }
        })
        .catch(err => console.error("FetchMetrics error:", err));
};

const DASHBOARD_RANGE_LABELS = {
    '5m': 'Last 5 minutes',
    '15m': 'Last 15 minutes',
    '30m': 'Last 30 minutes',
    '1h': 'Last 1 hour',
    '6h': 'Last 6 hours',
    '24h': 'Last 24 hours',
    '7d': 'Last 7 days',
    '30d': 'Last 30 days',
    today: 'Today',
    yesterday: 'Yesterday',
    all: 'All time (1 year)',
    custom: 'Custom range'
};

let dashboardRangeDebounceTimer = null;
let dashboardRefreshTimer = null;

const DASHBOARD_REFRESH_MS = {
    '5m': 15000,
    '15m': 30000,
    '30m': 45000,
    '1h': 60000,
    '6h': 120000,
    '12h': 180000,
    '24h': 180000,
    'today': 120000,
    'yesterday': 300000,
    '7d': 300000,
    '30d': 600000,
    'all': 600000,
    'custom': 120000
};

function dashboardRefreshIntervalMs() {
    const preset = document.getElementById('dashboard-range-preset')?.value || '5m';
    return DASHBOARD_REFRESH_MS[preset] || 60000;
}

function updateDashboardRefreshHint() {
    const el = document.getElementById('dashboard-auto-refresh-hint');
    if (!el) return;
    const sec = Math.round(dashboardRefreshIntervalMs() / 1000);
    el.textContent = `Auto ↻ ${sec}s`;
}

window.stopDashboardAutoRefresh = function () {
    if (dashboardRefreshTimer) clearInterval(dashboardRefreshTimer);
    dashboardRefreshTimer = null;
};

window.startDashboardAutoRefresh = function () {
    window.stopDashboardAutoRefresh();
    updateDashboardRefreshHint();
    dashboardRefreshTimer = setInterval(() => {
        const dash = document.getElementById('view-dashboard');
        if (!dash || !dash.classList.contains('active')) return;
        if (Date.now() < window.pausePollingUntil) return;
        window.runDashboardRangeFilter();
    }, dashboardRefreshIntervalMs());
};

window.restartDashboardAutoRefresh = function () {
    window.startDashboardAutoRefresh();
};

function formatDateInput(d) {
    const p = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function formatTimeInput(d) {
    const p = (n) => String(n).padStart(2, '0');
    return `${p(d.getHours())}:${p(d.getMinutes())}`;
}

function setSplitDatetime(dateId, timeId, d) {
    const dateEl = document.getElementById(dateId);
    const timeEl = document.getElementById(timeId);
    if (dateEl) dateEl.value = formatDateInput(d);
    if (timeEl) timeEl.value = formatTimeInput(d);
}

function getCombinedCustomDatetime(dateId, timeId) {
    const date = document.getElementById(dateId)?.value;
    const time = document.getElementById(timeId)?.value || '00:00';
    if (!date) return null;
    return formatDatetimeForApi(`${date}T${time}`);
}

/** Sends naive local timestamps to match SQLite created_at (no UTC shift). */
function formatDatetimeForApi(value) {
    if (!value || !String(value).includes('T')) return null;
    const [date, time] = String(value).split('T');
    const [hh = '00', mm = '00', ss = '00'] = (time || '').split(':');
    return `${date} ${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}:${String(ss).padStart(2, '0')}`;
}

function formatRangeSummary(fromVal, toVal) {
    try {
        const f = new Date(fromVal);
        const t = new Date(toVal);
        if (isNaN(f.getTime()) || isNaN(t.getTime())) return 'Custom range';
        return `${f.toLocaleString()} → ${t.toLocaleString()}`;
    } catch {
        return 'Custom range';
    }
}

window.getDashboardRangeQuery = function () {
    const preset = document.getElementById('dashboard-range-preset')?.value || '5m';
    if (preset === 'custom') {
        const from = getCombinedCustomDatetime('dashboard-range-from-date', 'dashboard-range-from-time');
        const to = getCombinedCustomDatetime('dashboard-range-to-date', 'dashboard-range-to-time');
        if (!from || !to) return null;
        return `range=custom&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`;
    }
    return `range=${encodeURIComponent(preset)}`;
};

function syncDashboardRangeChips(preset) {
    document.querySelectorAll('#dashboard-range-quick .range-chip').forEach(chip => {
        chip.classList.toggle('active', chip.dataset.range === preset);
    });
}

function fillCustomRangeDefaults() {
    const to = new Date();
    const from = new Date(to);
    from.setHours(0, 0, 0, 0);
    setSplitDatetime('dashboard-range-from-date', 'dashboard-range-from-time', from);
    setSplitDatetime('dashboard-range-to-date', 'dashboard-range-to-time', to);
}

window.setDashboardRangePreset = function (preset) {
    const sel = document.getElementById('dashboard-range-preset');
    if (sel) sel.value = preset;
    window.onDashboardRangeChange();
};

window.setDashboardRangeEndToNow = function () {
    const to = new Date();
    setSplitDatetime('dashboard-range-to-date', 'dashboard-range-to-time', to);
    window.onDashboardCustomRangeInput();
};

window.onDashboardCustomRangeInput = function () {
    clearTimeout(dashboardRangeDebounceTimer);
    dashboardRangeDebounceTimer = setTimeout(() => window.runDashboardRangeFilter(), 350);
};

window.runDashboardRangeFilter = function () {
    const preset = document.getElementById('dashboard-range-preset')?.value || '5m';
    if (preset === 'custom') {
        const fromRaw = `${document.getElementById('dashboard-range-from-date')?.value || ''}T${document.getElementById('dashboard-range-from-time')?.value || ''}`;
        const toRaw = `${document.getElementById('dashboard-range-to-date')?.value || ''}T${document.getElementById('dashboard-range-to-time')?.value || ''}`;
        if (!fromRaw.includes('T') || !toRaw.includes('T') || fromRaw === 'T' || toRaw === 'T') return;
        const from = new Date(fromRaw);
        const to = new Date(toRaw);
        if (isNaN(from.getTime()) || isNaN(to.getTime())) return;
        if (from > to) {
            notify('"Start" must be before "End".', 'error');
            return;
        }
        const summary = document.getElementById('dashboard-range-summary');
        if (summary) summary.textContent = formatRangeSummary(fromRaw, toRaw);
    }
    const rangeQ = window.getDashboardRangeQuery();
    if (rangeQ === null) return;
    window.fetchIncidents(true);
    window.reloadDashboardAnalytics();
    if (typeof window.restartDashboardAutoRefresh === 'function') window.restartDashboardAutoRefresh();
    const finopsView = document.getElementById('view-finops');
    if (finopsView && finopsView.classList.contains('active') && window.loadFinOpsView) {
        window.loadFinOpsView();
    }
};

window.onDashboardRangeChange = function () {
    const preset = document.getElementById('dashboard-range-preset')?.value || '5m';
    const customEl = document.getElementById('dashboard-custom-range');
    const summary = document.getElementById('dashboard-range-summary');

    syncDashboardRangeChips(preset);

    if (preset === 'custom') {
        if (customEl) customEl.style.display = 'grid';
        fillCustomRangeDefaults();
        if (summary) summary.textContent = formatRangeSummary(
            `${document.getElementById('dashboard-range-from-date')?.value}T${document.getElementById('dashboard-range-from-time')?.value}`,
            `${document.getElementById('dashboard-range-to-date')?.value}T${document.getElementById('dashboard-range-to-time')?.value}`
        );
        window.runDashboardRangeFilter();
        return;
    }

    if (customEl) customEl.style.display = 'none';
    if (summary) summary.textContent = DASHBOARD_RANGE_LABELS[preset] || preset;
    window.runDashboardRangeFilter();
};

window.reloadDashboardAnalytics = function () {
    const view = document.getElementById('analytics-view');
    if (view && view.style.display !== 'none' && typeof window.loadAnalytics === 'function') {
        window.loadAnalytics(false);
    }
};

window.initDashboardTimeRange = function () {
    const sel = document.getElementById('dashboard-range-preset');
    if (sel) sel.value = '5m';
    fillCustomRangeDefaults();
    syncDashboardRangeChips('5m');
    const summary = document.getElementById('dashboard-range-summary');
    if (summary) summary.textContent = DASHBOARD_RANGE_LABELS['5m'];
    const customEl = document.getElementById('dashboard-custom-range');
    if (customEl) customEl.style.display = 'none';
};

window.fetchIncidents = function (forceRender = false, opts = {}) {
    if (!opts.skipPauseCheck && !forceRender && Date.now() < window.pausePollingUntil) return;
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';
    const rangeQ = window.getDashboardRangeQuery();
    if (rangeQ === null) return Promise.resolve();

    return fetchWithAuth(`/api/incidents?project=${encodeURIComponent(projectFilter)}&${rangeQ}&_ts=${Date.now()}&_bust=${Math.random()}`, {
        headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate', 'Pragma': 'no-cache', 'Expires': '0' }
    })
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            if (!ok) throw new Error('incidents');
            if (!opts.skipPauseCheck && !forceRender && Date.now() < window.pausePollingUntil) return;
            const rawData = [...(Array.isArray(data) ? data : [])].sort((a, b) => a.id - b.id);
            const retryIds = window.confirmPendingResolvedFromServer(rawData);

            if (retryIds.length > 0 && !window._resolveRetryInFlight) {
                window._resolveRetryInFlight = true;
                fetchWithAuth('/api/incidents', { method: 'PUT', body: JSON.stringify({ ids: retryIds }) }).finally(() => { window._resolveRetryInFlight = false; });
            }

            const safeData = window.mergePendingResolvedState(rawData);
            const safeCurrent = [...(window.currentIncidents || [])].sort((a, b) => a.id - b.id);
            const oldSig = safeCurrent.map(i => `${i.id}:${normalizeIsResolved(i)}`).join('|');
            const newSig = safeData.map(i => `${i.id}:${normalizeIsResolved(i)}`).join('|');

            if (forceRender || oldSig !== newSig) { window.currentIncidents = safeData; window.renderIncidentsList(); }
            return safeData;
        })
        .catch(() => {
            const listEl = document.getElementById('incident-list');
            if (listEl) {
                listEl.innerHTML = `<div class="load-error-msg">Unable to load incidents. Check that the server is running and refresh.</div>`;
            }
        });
};

window.renderIncidentsList = function () {
    const listEl = document.getElementById('incident-list');
    const searchQuery = document.getElementById('incident-search') ? document.getElementById('incident-search').value.toLowerCase() : '';

    const openGroups = new Set();
    document.querySelectorAll('[id^="sub-alerts-"]').forEach(el => { if (el.style.display === 'block') openGroups.add(el.id.replace('sub-alerts-', '')); });

    let filteredData = window.currentIncidents || [];
    if (window.statusFilter === 'active') filteredData = filteredData.filter(inc => !normalizeIsResolved(inc));
    else if (window.statusFilter === 'resolved') filteredData = filteredData.filter(inc => normalizeIsResolved(inc));

    if (searchQuery) filteredData = filteredData.filter(inc => (inc.name && inc.name.toLowerCase().includes(searchQuery)) || (inc.project_name && inc.project_name.toLowerCase().includes(searchQuery)) || (inc.error_message && inc.error_message.toLowerCase().includes(searchQuery)));

    const openLogs = new Set();
    if (window.__logsExpandAll === true) {
        filteredData.forEach(inc => openLogs.add(String(inc.id)));
    } else if (window.__logsExpandAll !== false) {
        document.querySelectorAll('[id^="log-content-"]').forEach(el => {
            if (el.style.display === 'block') openLogs.add(el.id.replace('log-content-', ''));
        });
    }

    if (filteredData.length === 0) {
        const total = (window.currentIncidents || []).length;
        let msg = 'No incidents yet. Run a pipeline and send results to your CI/CD gateway webhook.';
        if (total > 0 && searchQuery) msg = 'No incidents match your search. Clear the search box or change filters.';
        else if (total > 0 && window.statusFilter === 'active') msg = 'No active incidents. Switch status filter to "All" or "Resolved".';
        else if (total > 0 && window.statusFilter === 'resolved') msg = 'No resolved incidents yet.';
        else if (total > 0) msg = 'No incidents match the current project filter.';
        listEl.innerHTML = `<div class="incident-empty-state">${msg}</div>`;
        return;
    }

    const activeCount = filteredData.filter(i => !normalizeIsResolved(i)).length;
    const resolvedCount = filteredData.filter(i => normalizeIsResolved(i)).length;
    const health = filteredData.length > 0 ? Math.round((resolvedCount / filteredData.length) * 100) : 100;

    const kpiActive = document.getElementById('kpi-active');
    kpiActive.innerText = activeCount;
    kpiActive.className = `kpi-value ${activeCount > 0 ? 'kpi-danger' : 'kpi-success'}`;
    document.getElementById('kpi-resolved').innerText = resolvedCount;
    const kpiHealth = document.getElementById('kpi-health');
    kpiHealth.innerText = `${health}%`;
    kpiHealth.className = `kpi-value ${health < 80 ? 'kpi-warn' : 'kpi-info'}`;

    const sortedData = [...filteredData].sort((a, b) => a.id - b.id);
    const groupsArray = []; let currentGroup = null;

    sortedData.forEach(inc => {
        const safeDateStr = inc.created_at ? inc.created_at.replace(' ', 'T') + 'Z' : '';
        const incTime = new Date(safeDateStr).getTime() || 0;
        const runKey = (inc.pipeline_run_id && String(inc.pipeline_run_id).trim())
            ? String(inc.pipeline_run_id)
            : `legacy-${inc.project_name}-${Math.floor(incTime / 120000)}`;

        if (!currentGroup) {
            currentGroup = { id: inc.id, project_name: inc.project_name, created_at: inc.created_at, pipeline_run_id: runKey, is_resolved: true, incidents: [], firstTime: incTime, lastTime: incTime, lastId: inc.id };
            groupsArray.push(currentGroup);
        } else {
            const sameRun = inc.project_name === currentGroup.project_name && runKey === currentGroup.pipeline_run_id;
            if (sameRun) {
                currentGroup.lastTime = incTime; currentGroup.lastId = inc.id;
            } else {
                currentGroup = { id: inc.id, project_name: inc.project_name, created_at: inc.created_at, pipeline_run_id: runKey, is_resolved: true, incidents: [], firstTime: incTime, lastTime: incTime, lastId: inc.id };
                groupsArray.push(currentGroup);
            }
        }
        currentGroup.incidents.push(inc);
        if (!normalizeIsResolved(inc)) currentGroup.is_resolved = false;
    });

    groupsArray.reverse(); window.groupedIncidents = {};
    groupsArray.forEach(g => {
        window.groupedIncidents[g.id] = g;
        openGroups.add(String(g.id));
    });

    const userRole = parseJwt(localStorage.getItem('sre-jwt')).role;
    const canResolve = canResolveIncidents(userRole);
    const canDelete = canDeleteIncidents(userRole);

    const iconCheck = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2.5"><polyline points="20 6 9 17 4 12"></polyline></svg>`;
    const iconAlert = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2.5"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>`;
    const iconWarning = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>`;
    const iconFile = `<svg style="width:12px;height:12px;margin-right:6px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>`;
    const iconCode = `<svg style="width:12px;height:12px;margin-right:6px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>`;
    const iconChevron = `<svg style="width:14px;height:14px;stroke:currentColor;fill:none;transition: transform 0.2s;" viewBox="0 0 24 24" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>`;
    const iconTrash = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>`;
    const iconGear = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1 0-2.83 2 2 0 0 1 0-2.83l.06.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`;

    const globalAllSelected = window.currentIncidents.length > 0 && window.currentIncidents.every(inc => window.selectedIncidents.has(String(inc.id)));

    let htmlContent = `
    <div id="bulk-action-banner" class="bulk-action-bar" style="display: ${window.selectedIncidents.size > 0 ? 'flex' : 'none'};">
        <div style="display:flex; align-items:center; gap: 10px;">
            <input type="checkbox" id="select-all-cb" onclick="toggleSelectAll(this.checked)" ${globalAllSelected ? 'checked' : ''} style="width: 16px; height: 16px; cursor: pointer;">
            <label for="select-all-cb" id="bulk-count-label" class="bulk-action-label">${window.selectedIncidents.size} Test(s) Selected</label>
        </div>
        <div style="display: flex; gap: 10px; margin-left: auto;">
            ${canResolve ? `<button type="button" class="btn btn-secondary btn-success btn-sm" onclick="resolveSelected()">${iconCheck} Resolve</button>` : ''}
            ${canDelete ? `<button type="button" class="btn btn-secondary btn-danger btn-sm" onclick="deleteSelected()">${iconTrash} Delete</button>` : ''}
        </div>
    </div>`;

    htmlContent += groupsArray.map(group => {
        const activeSubAlerts = group.incidents.filter(i => !normalizeIsResolved(i)).length;
        const cardStateClass = activeSubAlerts > 0 ? 'pipeline-exec-card--active' : 'pipeline-exec-card--resolved';

        const resolvedBadge = activeSubAlerts === 0
            ? `<span class="exec-status-badge exec-status-badge--resolved">${iconCheck} EXECUTION RESOLVED</span>`
            : `<span class="exec-status-badge exec-status-badge--active">${iconAlert} ${activeSubAlerts} ACTIVE / ${group.incidents.length} TOTAL</span>`;

        const subAlertsHTML = group.incidents.map(inc => {
            const isFlaky = inc.name.includes("[FLAKY]");
            const flakyBadge = isFlaky ? `<span class="kpi-warn" style="margin-right:8px;">${iconWarning} FLAKY</span>` : '';
            const cleanName = inc.name.replace("[FLAKY] ", "");
            const rawLog = inc.error_logs || inc.error_message || "No logs available.";
            const displayLog = String(rawLog)
                .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

            const isResolved = normalizeIsResolved(inc);
            const nameClass = isResolved ? 'kpi-success' : '';
            const nameStyle = isResolved ? 'text-decoration:line-through;' : '';
            const isChecked = window.selectedIncidents.has(String(inc.id)) ? 'checked' : '';
            const rowClass = isResolved ? 'is-resolved' : 'is-active';

            const subAlertFlag = isResolved
                ? `<span class="incident-test-badge exec-status-badge exec-status-badge--resolved">${iconCheck} RESOLVED BY ${inc.resolved_by || 'SYSTEM'}</span>`
                : `<span class="incident-test-badge exec-status-badge exec-status-badge--active">${iconAlert} ACTIVE TEST</span>`;

            const isLogOpen = openLogs.has(String(inc.id));
            const logDisplay = isLogOpen ? 'block' : 'none';
            const iconTransform = isLogOpen ? 'rotate(180deg)' : 'rotate(0deg)';
            const textLabel = isLogOpen ? 'Collapse logs' : 'Show logs';

            return `
            <div class="incident-test-row ${rowClass}">
                <div class="incident-test-row-header">
                    <div class="incident-test-row-title">
                        <input type="checkbox" id="cb-inc-${inc.id}" onclick="toggleIncidentSelection('${inc.id}', this.checked)" ${isChecked} style="width: 14px; height: 14px; cursor: pointer; flex-shrink: 0;">
                        <strong class="${nameClass}" style="font-size: 13px; ${nameStyle}">${flakyBadge}${cleanName}</strong>
                    </div>
                    ${subAlertFlag}
                </div>
                <div style="margin-top: 8px; text-align: left;">
                    <button type="button" class="log-toggle-btn" onclick="window.toggleIncidentLog('${inc.id}', event)">
                        <svg id="log-icon-${inc.id}" style="width:12px;height:12px;stroke:currentColor;fill:none;transition: transform 0.2s; transform: ${iconTransform};" viewBox="0 0 24 24" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>
                        <span id="log-text-${inc.id}">${textLabel}</span>
                    </button>
                    <div id="log-content-${inc.id}" class="incident-log-panel" style="display: ${logDisplay};">${displayLog}</div>
                </div>
            </div>`;
        }).join('');

        const actionDropdown = `
            <div class="action-dropdown-container">
                <button type="button" class="btn btn-secondary btn-sm" onclick="toggleActionDropdown('${group.id}', event)">${iconGear} Actions</button>
                <div id="action-dropdown-${group.id}" class="action-dropdown-menu">
                    ${(!group.is_resolved && canResolve) ? `<a href="#" class="action-resolve" onclick="resolveGroup('${group.id}', event)">${iconCheck} Resolve Execution</a>` : ''}
                    ${canDelete ? `<a href="#" class="action-delete" onclick="deleteGroup('${group.id}', event)">${iconTrash} Delete Execution</a>` : ''}
                    <a href="#" class="action-export-err" onclick="downloadGroupLog('${group.id}', 'error', event)">${iconAlert} Export Errors</a>
                    <a href="#" class="action-export-full" onclick="downloadGroupLog('${group.id}', 'full', event)">${iconFile} Export Full Logs</a>
                    <a href="#" class="action-export-xml" onclick="downloadGroupLog('${group.id}', 'xml', event)">${iconCode} Generate JUnit XML</a>
                </div>
            </div>`;

        const groupAllSelected = group.incidents.length > 0 && group.incidents.every(inc => window.selectedIncidents.has(String(inc.id)));
        const groupChecked = groupAllSelected ? 'checked' : '';

        return `
        <div class="data-card pipeline-exec-card ${cardStateClass}">
            <div class="pipeline-exec-header">
                <div class="pipeline-exec-meta">
                    <input type="checkbox" id="cb-group-${group.id}" onclick="toggleGroupSelection('${group.id}', this.checked)" ${groupChecked} style="width: 16px; height: 16px; cursor: pointer; flex-shrink: 0; margin-top: 4px;">
                    <div>
                        <span class="pipeline-exec-label">Pipeline execution</span>
                        <span class="pipeline-exec-title">${group.project_name}</span>
                        <span class="pipeline-exec-submeta">${group.created_at} · ${group.incidents.length} failed test(s)</span>
                    </div>
                </div>
                <div class="pipeline-exec-toolbar">
                    ${resolvedBadge}
                    ${actionDropdown}
                    <button type="button" class="btn btn-secondary btn-sm" onclick="toggleSubAlerts('${group.id}', event)">
                        ${group.incidents.length} Alert(s) <span id="toggle-icon-${group.id}" style="display: inline-block;">${iconChevron}</span>
                    </button>
                </div>
            </div>
            <div id="sub-alerts-${group.id}" class="pipeline-exec-body" style="display: none;">${subAlertsHTML}</div>
        </div>`;
    }).join('');

    listEl.innerHTML = htmlContent;
    window.updateBulkActionUI();

    openGroups.forEach(groupId => {
        const el = document.getElementById(`sub-alerts-${groupId}`);
        const icon = document.getElementById(`toggle-icon-${groupId}`);
        if (el) { el.style.display = 'block'; if (icon) icon.style.transform = 'rotate(180deg)'; }
    });
};

window.initDashboardScrollHelpers = function () {
    const main = document.querySelector('.main-content');
    const btn = document.getElementById('scroll-top-btn');
    if (!main || !btn || main.dataset.scrollBound) return;
    main.dataset.scrollBound = '1';
    main.addEventListener('scroll', () => {
        btn.classList.toggle('visible', main.scrollTop > 280);
    }, { passive: true });
    btn.addEventListener('click', () => main.scrollTo({ top: 0, behavior: 'smooth' }));
};

window.scrollDashboardToTop = function () {
    const main = document.querySelector('.main-content');
    if (main) main.scrollTo({ top: 0, behavior: 'smooth' });
};

window.connectWebSocket = function () {
    if (window.__incidentPollTimer) clearInterval(window.__incidentPollTimer);
    window.__incidentPollTimer = setInterval(() => {
        if (isApiOffline()) return;
        const dashboard = document.getElementById('view-dashboard');
        if (Date.now() < window.pausePollingUntil) return;
        if (dashboard && dashboard.classList.contains('active')) window.fetchIncidents();
    }, 5000);
}

// ==========================================
// AUTH & ROUTING
// ==========================================
window.performLogin = function () {
    const username = document.getElementById('login-username').value.trim();
    const password = document.getElementById('login-password').value;
    const errEl = document.getElementById('login-error');
    if (!username || !password) {
        if (errEl) { errEl.textContent = 'Enter username and password.'; errEl.style.display = 'block'; }
        return notify("Enter username and password", "error");
    }
    if (errEl) errEl.style.display = 'none';

    const loginBtn = document.querySelector('#login-screen .btn-primary');
    if (loginBtn) { loginBtn.disabled = true; loginBtn.textContent = 'SIGNING IN…'; }

    const controller = new AbortController();
    const loginTimeout = setTimeout(() => controller.abort(), 30000);

    fetch('/api/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }), signal: controller.signal })
        .then(async res => {
            let body = null;
            try { body = await res.json(); } catch (_) { /* plain text */ }
            if (res.ok) return body;
            const msg = (body && body.error) ? body.error
                : res.status === 401 ? 'Invalid username or password.'
                : res.status === 403 ? (body?.error || 'Account disabled.')
                : `Login failed (${res.status}).`;
            throw new Error(msg);
        })
        .then(data => {
            localStorage.setItem('sre-jwt', data.token);
            if (errEl) errEl.style.display = 'none';
            resetApiOfflineBanner();
            if (data.require_password_change) {
                notify('Sign-in OK. Set a new password below to enter the app.', 'info');
            }
            window.checkAuth();
        })
        .catch(err => {
            const isAbort = err?.name === 'AbortError';
            const isNetwork = err instanceof TypeError || isAbort || (err.message && /failed|network/i.test(err.message));
            const text = isAbort
                ? 'Login timed out. The server may be overloaded — wait and try again.'
                : isNetwork
                ? 'Cannot reach the server. Open http://localhost:9000 and start the backend (go run ./cmd/qacapsule).'
                : (err.message || 'Invalid username or password.');
            if (errEl) { errEl.textContent = text.toUpperCase(); errEl.style.display = 'block'; }
            notify(text, "error");
        })
        .finally(() => {
            clearTimeout(loginTimeout);
            if (loginBtn) { loginBtn.disabled = false; loginBtn.textContent = 'INITIALIZE SESSION'; }
        });
}

window.probeLoginServer = function () {
    const hint = document.getElementById('login-server-hint');
    if (!hint) return;
    pingApiServer().then(ok => {
        if (ok) hint.style.display = 'none';
        else {
            hint.style.display = 'block';
            hint.textContent = 'Backend not reachable. Use http://localhost:9000 (run: go run ./cmd/qacapsule).';
        }
    });
}

window.checkAuth = function () {
    const token = localStorage.getItem('sre-jwt');
    if (!token) {
        showLoginScreen();
        return;
    }
    const payload = parseJwt(token);
    if (!payload || !payload.role) {
        localStorage.removeItem('sre-jwt');
        showLoginScreen();
        notify('Invalid session. Please sign in again.', 'error');
        return;
    }

    if (payload.require_password_change) {
        document.getElementById('login-screen').style.display = 'none';
        document.getElementById('app-container').style.display = 'none';
        document.getElementById('force-password-screen').style.display = 'flex';
        return;
    }

    document.getElementById('login-screen').style.display = 'none';
    document.getElementById('force-password-screen').style.display = 'none';
    document.getElementById('app-container').style.display = 'flex';

    pingApiServer().then(() => {
        window.applyPermissions();
        if (window.loadUserPreferences) window.loadUserPreferences();

        if (payload.role === 'admin') window.loadUsers();

        if (document.getElementById('view-dashboard').classList.contains('active')) {
            window.initDashboardTimeRange();
            window.loadDashboardFilters();
            window.pausePollingUntil = 0;
            if (window.loadAnalyticsLayoutFromPrefs) window.loadAnalyticsLayoutFromPrefs();
            window.fetchIncidents(true);
            if (window.startDashboardAutoRefresh) window.startDashboardAutoRefresh();
            window.initDashboardScrollHelpers();
        }
        if (document.getElementById('view-organizations').classList.contains('active') && canManageTeams(payload.role)) {
            window.loadOrganizations();
        }
        if (document.getElementById('view-management').classList.contains('active') && payload.role === 'admin') {
            if (window.renderUserTable) window.renderUserTable(iam.allUsers);
        }
        if (document.getElementById('view-settings').classList.contains('active') && window.loadConfig) {
            window.loadConfig();
        }
        if (document.getElementById('view-plugins').classList.contains('active') && hasMinRole(payload.role, 'lead')) {
            if (window.loadPlugins) window.loadPlugins();
        }
        if (document.getElementById('view-ingestion').classList.contains('active') && hasMinRole(payload.role, 'lead')) {
            if (window.loadGatewaysData) window.loadGatewaysData();
        }
        window.connectWebSocket();
    });
}

window.submitNewPassword = function () {
    const p1 = document.getElementById('force-new-pass').value;
    const p2 = document.getElementById('force-new-pass-confirm').value;
    if (!p1 || p1.length < 6) return notify("Password must be at least 6 characters", "error");
    if (p1 !== p2) return notify("Passwords do not match", "error");

    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    const username = payload?.username || document.getElementById('login-username')?.value?.trim();
    if (!username) return notify("Session expired. Sign in again.", "error");

    fetchWithAuth('/api/users/change-password', { method: 'POST', body: JSON.stringify({ new_password: p1 }) })
        .then(res => parseApiJson(res))
        .then(async ({ ok }) => {
            if (!ok) {
                notify("Error updating password", "error");
                return;
            }
            localStorage.removeItem('sre-jwt');
            const loginRes = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password: p1 })
            });
            const loginBody = await loginRes.json().catch(() => ({}));
            if (!loginRes.ok) {
                notify(loginBody.error || 'Password saved. Sign in with your new password.', 'info');
                showLoginScreen();
                return;
            }
            localStorage.setItem('sre-jwt', loginBody.token);
            notify("Password secured. Welcome!", "success");
            document.getElementById('force-new-pass').value = '';
            document.getElementById('force-new-pass-confirm').value = '';
            window.checkAuth();
        }).catch(() => notify("Network error", "error"));
}

window.switchView = function (id, el) {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    if (payload?.role && !canAccessView(payload.role, id)) {
        notify(accessDeniedMessage(id), 'error');
        return;
    }

    document.querySelectorAll('.view-section').forEach(x => { 
        x.style.display = 'none'; 
        x.classList.remove('active'); 
    });
    document.querySelectorAll('.nav-item').forEach(x => x.classList.remove('active'));
    
    const targetSection = document.getElementById('view-' + id);
    if (targetSection) {
        targetSection.style.display = 'block';
        targetSection.classList.add('active');
    }
    if (el) el.classList.add('active');

    if (id === 'dashboard') {
        window.initDashboardTimeRange();
        window.loadDashboardFilters();
        window.pausePollingUntil = 0;
        if (window.loadAnalyticsLayoutFromPrefs) window.loadAnalyticsLayoutFromPrefs();
        window.fetchIncidents(true);
        if (window.startDashboardAutoRefresh) window.startDashboardAutoRefresh();
        window.initDashboardScrollHelpers();
    } else if (typeof window.stopDashboardAutoRefresh === 'function') {
        window.stopDashboardAutoRefresh();
    }
    if (id === 'organizations' && payload && canManageTeams(payload.role)) {
        if (window.loadOrganizations) window.loadOrganizations();
    }
    if (id === 'management' && payload && payload.role === 'admin') if (window.renderUserTable) window.renderUserTable(iam.allUsers);
    if (id === 'plugins' && payload && canAccessPlugins(payload.role)) {
        if (window.loadPlugins) window.loadPlugins();
    }
    if (id === 'settings' && payload && canManageIAM(payload.role)) {
        if (window.loadConfig) window.loadConfig();
    }
    if (id === 'ingestion' && payload && hasMinRole(payload.role, 'lead')) {
        if (window.loadGatewaysData) window.loadGatewaysData();
    }
    if (id === 'finops' && payload && canAccessFinOps(payload.role)) {
        if (window.loadFinOpsView) window.loadFinOpsView();
    }
    if (id === 'rca' && payload && canAccessView(payload.role, 'rca')) {
        if (window.loadRCAView) window.loadRCAView();
        if (window.loadAIConfigPanel) window.loadAIConfigPanel();
    }
    if (id === 'quarantine' && payload && canAccessView(payload.role, 'quarantine')) {
        if (window.loadQuarantineView) window.loadQuarantineView();
    }
    if (id === 'runbooks' && payload && canAccessView(payload.role, 'runbooks')) {
        if (window.loadRunbooksView) window.loadRunbooksView();
    }
    if (id === 'dora' && payload && canAccessView(payload.role, 'dora')) {
        if (window.loadDORAView) window.loadDORAView();
    } else if (typeof window.destroyDORAChart === 'function') {
        window.destroyDORAChart();
    }
    if (id === 'about') {
        if (window.loadAboutView) window.loadAboutView();
    }
    if (id === 'profile' && window.loadProfileView) window.loadProfileView();
}

window.applyPermissions = function () {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    if (!payload) return;
    applyRoleVisibility(payload.role);
    window.initSmartSearchFields();

    const activeSection = document.querySelector('.view-section.active');
    const activeId = activeSection?.id?.replace('view-', '');
    if (activeId && !canAccessView(payload.role, activeId)) {
        const fallback = defaultViewForRole(payload.role);
        const nav = document.querySelector(`.nav-item[onclick*="switchView('${fallback}'"]`);
        window.switchView(fallback, nav);
    } else if (isAdmin(payload.role)) {
        const onAdminView = ['organizations', 'management', 'settings', 'profile'].includes(activeId);
        if (!onAdminView) {
            window.switchView('organizations', document.querySelector('.nav-item.role-workspace'));
        }
    }

    const roleEl = document.getElementById('profile-role');
    if (roleEl) roleEl.textContent = roleLabel(payload.role);
}

window.initSmartSearchFields = function () {
    const incidentInput = document.getElementById('incident-search');
    const incidentList = document.getElementById('incident-search-ac');
    if (incidentInput && incidentList && !incidentInput.dataset.acBound) {
        incidentInput.dataset.acBound = '1';
        setupAutocomplete({
            input: incidentInput,
            list: incidentList,
            minChars: 1,
            getSuggestions: q => {
                const v = q.toLowerCase();
                const hints = new Map();
                (window.currentIncidents || []).forEach(inc => {
                    if (inc.name) hints.set(inc.name, { label: inc.name, sublabel: inc.project_name || '', value: inc.name });
                    if (inc.project_name) hints.set(inc.project_name, { label: inc.project_name, sublabel: 'Project', value: inc.project_name });
                });
                document.querySelectorAll('#project-filter option').forEach(opt => {
                    if (opt.value && opt.value !== 'all' && opt.text.toLowerCase().includes(v)) {
                        hints.set(opt.value, { label: opt.text, sublabel: 'Project', value: opt.value });
                    }
                });
                return [...hints.values()].filter(h => h.label.toLowerCase().includes(v)).slice(0, 12);
            },
            onSelect: item => { incidentInput.value = item.value; window.fetchIncidents(true); }
        });
    }
    if (window.initIamUserSearchAutocomplete) window.initIamUserSearchAutocomplete();
};

// ==========================================
// STARTUP
// ==========================================
window.addEventListener('unhandledrejection', (e) => {
    console.error('[QA Capsule] unhandled promise rejection', e.reason);
});

window.onload = function () {
    initTheme();
    initSidebar();
    pingApiServer();
    window.checkAuth();
    if (!localStorage.getItem('sre-jwt') && window.probeLoginServer) window.probeLoginServer();
    if (window.checkSSOStatus) window.checkSSOStatus();
};