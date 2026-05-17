/**
* web/app.js
* Main controller for QA Capsule Control Plane
*/

// IMPORT MODULES
import { notify, showConfirmModal, showPromptModal, closeModal, toggleTheme } from './js/ui.js';
import { parseJwt, performLogout, fetchWithAuth } from './js/api.js';
import * as iam from './js/iam.js';
import * as settings from './js/settings.js';

// EXPORT GLOBALLY FOR HTML INLINE HANDLERS
Object.assign(window, { notify, showConfirmModal, showPromptModal, closeModal, toggleTheme, parseJwt, performLogout, fetchWithAuth });

// Bind all module functions to the window so HTML 'onclick' can find them
for (const [key, value] of Object.entries(iam)) {
    if (typeof value === 'function') window[key] = value;
}
for (const [key, value] of Object.entries(settings)) {
    if (typeof value === 'function') window[key] = value;
}

// Global state management for Incidents
window.currentIncidents = [];
window.selectedIncidents = new Set();
window.pausePollingUntil = 0;
window.statusFilter = 'all'; 
window.pendingResolvedIds = new Map();
window._resolveRetryInFlight = false;
window.groupedIncidents = {};

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
    } catch (_) {}
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
        notify('Erreur lors de la résolution', 'error');
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
    if (window.selectedIncidents.size === 0) return notify("Sélectionnez au moins une alerte.", "error");
    if (parseJwt(localStorage.getItem('sre-jwt')).role !== 'admin') return notify("Seuls les admins peuvent supprimer.", "error");

    showConfirmModal("Delete Selected?", `Supprimer définitivement ${window.selectedIncidents.size} alertes ?`, "danger", async function () {
        window.pausePollingUntil = Date.now() + 15000;
        const ids = Array.from(window.selectedIncidents).map(id => parseInt(id, 10)).filter(id => !isNaN(id));
        window.currentIncidents = window.currentIncidents.filter(inc => !ids.includes(parseInt(inc.id, 10)));
        window.selectedIncidents.clear();
        window.renderIncidentsList();
        notify("Suppression en cours...", "success");

        try {
            const promises = ids.map(id => window.fetchWithAuth(`/api/incidents?id=${id}&ids=${id}`, { method: 'DELETE' }));
            await Promise.all(promises);
            setTimeout(() => { window.pausePollingUntil = 0; window.fetchIncidents(true); window.fetchMetricsOnly(); }, 1200);
        } catch (e) {
            window.pausePollingUntil = 0; window.fetchIncidents(true);
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
    await window.resolveIncidentsByIds(ids, ids.length === 1 ? 'Sub-alert résolue.' : 'Pipelines marqués comme résolus.');
}

window.deleteGroup = async function (groupId, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    const dropdown = document.getElementById(`action-dropdown-${groupId}`);
    if (dropdown) dropdown.style.display = 'none';

    const group = window.groupedIncidents[groupId];
    if (!group) return;

    showConfirmModal("Delete Pipeline Execution?", `Supprimer définitivement l'exécution ?`, "danger", async function () {
        window.pausePollingUntil = Date.now() + 15000;
        const ids = group.incidents.map(i => parseInt(i.id, 10)).filter(id => !isNaN(id));

        window.currentIncidents = window.currentIncidents.filter(inc => !ids.includes(parseInt(inc.id, 10)));
        group.incidents.forEach(i => window.selectedIncidents.delete(String(i.id)));

        window.renderIncidentsList();
        notify("Suppression du pipeline en cours...", "success");

        try {
            const promises = ids.map(id => window.fetchWithAuth(`/api/incidents?id=${id}&ids=${id}`, { method: 'DELETE' }));
            await Promise.all(promises);
            setTimeout(() => { window.pausePollingUntil = 0; window.fetchIncidents(true); window.fetchMetricsOnly(); }, 1200);
        } catch (e) {
            window.pausePollingUntil = 0; window.fetchIncidents(true);
        }
    });
}

window.toggleActionDropdown = function (id, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    const dropdown = document.getElementById(`action-dropdown-${id}`);
    if (dropdown) {
        const isCurrentlyHidden = dropdown.style.display === 'none';
        document.querySelectorAll('[id^="action-dropdown-"]').forEach(el => el.style.display = 'none');
        if (isCurrentlyHidden) { dropdown.style.display = 'block'; window.pausePollingUntil = Date.now() + 15000; }
        else { dropdown.style.display = 'none'; window.pausePollingUntil = 0; }
    }
}

document.addEventListener('click', function (event) {
    if (!event.target.closest('.action-dropdown-container')) {
        document.querySelectorAll('[id^="action-dropdown-"]').forEach(el => el.style.display = 'none');
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

window.loadDashboardFilters = function () {
    fetchWithAuth(`/api/my-projects?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(projects => {
            const filter = document.getElementById('project-filter');
            const currentVal = filter.value;
            filter.innerHTML = '<option value="all">All My Projects</option>';
            if (projects) projects.forEach(p => filter.innerHTML += `<option value="${p.name}">${p.name}</option>`);
            if (currentVal) filter.value = currentVal;
        }).catch(e => console.log("Error loading filter"));
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
    fetchWithAuth(`/api/metrics?_ts=${Date.now()}`)
        .then(r => r.json())
        .then(metrics => {
            if (metrics) {
                document.getElementById('kpi-active').innerText = metrics.total_incidents - metrics.resolved_incidents;
                document.getElementById('kpi-resolved').innerText = metrics.resolved_incidents;
                document.getElementById('kpi-health').innerText = `${metrics.total_incidents > 0 ? Math.round((metrics.resolved_incidents / metrics.total_incidents) * 100) : 100}%`;
            }
        });
};

window.fetchIncidents = function (forceRender = false, opts = {}) {
    if (!opts.skipPauseCheck && !forceRender && Date.now() < window.pausePollingUntil) return;
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';
    
    return fetchWithAuth(`/api/incidents?project=${encodeURIComponent(projectFilter)}&_ts=${Date.now()}&_bust=${Math.random()}`, {
        headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate', 'Pragma': 'no-cache', 'Expires': '0' }
    })
        .then(res => res.json())
        .then(data => {
            if (!opts.skipPauseCheck && !forceRender && Date.now() < window.pausePollingUntil) return;
            const rawData = [...(data || [])].sort((a, b) => a.id - b.id);
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
            window.fetchMetricsOnly();
            return safeData;
        })
        .catch(err => {
            const listEl = document.getElementById('incident-list');
            if (listEl && listEl.innerHTML.includes('Loading')) listEl.innerHTML = `<div style="text-align:center; padding: 40px; color: #8b949e;">No incidents found or database empty.</div>`;
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

    if (filteredData.length === 0) { listEl.innerHTML = '<div style="text-align:center; padding: 40px; opacity: 0.5;">No results match your search.</div>'; return; }

    const activeCount = filteredData.filter(i => !normalizeIsResolved(i)).length;
    const resolvedCount = filteredData.filter(i => normalizeIsResolved(i)).length;
    const health = filteredData.length > 0 ? Math.round((resolvedCount / filteredData.length) * 100) : 100;

    document.getElementById('kpi-active').innerText = activeCount; document.getElementById('kpi-active').style.color = activeCount > 0 ? '#ff7b72' : '#3fb950';
    document.getElementById('kpi-resolved').innerText = resolvedCount;
    document.getElementById('kpi-health').innerText = `${health}%`; document.getElementById('kpi-health').style.color = health < 80 ? '#d29922' : '#58a6ff';

    const sortedData = [...filteredData].sort((a, b) => a.id - b.id);
    const groupsArray = []; let currentGroup = null;

    sortedData.forEach(inc => {
        const safeDateStr = inc.created_at ? inc.created_at.replace(' ', 'T') + 'Z' : '';
        const incTime = new Date(safeDateStr).getTime() || 0;
        if (!currentGroup) {
            currentGroup = { id: inc.id, project_name: inc.project_name, created_at: inc.created_at, is_resolved: true, incidents: [], lastTime: incTime, lastId: inc.id };
            groupsArray.push(currentGroup);
        } else {
            const timeDiffSec = Math.abs(incTime - currentGroup.lastTime) / 1000;
            const idDiff = Math.abs(inc.id - currentGroup.lastId);
            if (inc.project_name === currentGroup.project_name && timeDiffSec <= 120 && idDiff <= 100) {
                currentGroup.lastTime = incTime; currentGroup.lastId = inc.id;
            } else {
                currentGroup = { id: inc.id, project_name: inc.project_name, created_at: inc.created_at, is_resolved: true, incidents: [], lastTime: incTime, lastId: inc.id };
                groupsArray.push(currentGroup);
            }
        }
        currentGroup.incidents.push(inc);
        if (!normalizeIsResolved(inc)) currentGroup.is_resolved = false;
    });

    groupsArray.reverse(); window.groupedIncidents = {};
    groupsArray.forEach(g => { window.groupedIncidents[g.id] = g; });

    const userRole = parseJwt(localStorage.getItem('sre-jwt')).role;
    const canResolve = userRole !== 'viewer'; const isAdmin = userRole === 'admin';

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
    <div id="bulk-action-banner" style="display: ${window.selectedIncidents.size > 0 ? 'flex' : 'none'}; margin-bottom: 20px; justify-content: space-between; align-items: center; background: #161b22; padding: 12px 20px; border: 1px solid #58a6ff; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.5); position: sticky; top: 10px; z-index: 100;">
        <div style="display:flex; align-items:center; gap: 10px;">
            <input type="checkbox" id="select-all-cb" onclick="toggleSelectAll(this.checked)" ${globalAllSelected ? 'checked' : ''} style="width: 16px; height: 16px; cursor: pointer;">
            <label for="select-all-cb" id="bulk-count-label" style="color: #58a6ff; font-size: 13px; font-weight: bold; cursor: pointer;">${window.selectedIncidents.size} Test(s) Selected</label>
        </div>
        <div style="display: flex; gap: 10px;">
            ${canResolve ? `<button class="btn-primary" style="font-size: 12px; padding: 6px 12px; border-color:#3fb950; background:rgba(63, 185, 80, 0.1); color:#3fb950;" onclick="resolveSelected()">${iconCheck} Resolve</button>` : ''}
            ${isAdmin ? `<button class="btn-secondary" style="font-size: 12px; padding: 6px 12px; color: #ff7b72; border-color: #ff7b72;" onclick="deleteSelected()">${iconTrash} Delete</button>` : ''}
        </div>
    </div>`;

    htmlContent += groupsArray.map(group => {
        const activeSubAlerts = group.incidents.filter(i => !normalizeIsResolved(i)).length;
        let severityColor = activeSubAlerts > 0 ? '#ff7b72' : '#3fb950';

        let resolvedBadge = activeSubAlerts === 0
            ? `<span style="display:inline-flex; align-items:center; background: rgba(63, 185, 80, 0.1); color: #3fb950; padding: 4px 10px; border-radius: 12px; font-size: 11px; font-weight: bold;">${iconCheck} EXECUTION RESOLVED</span>`
            : `<span style="display:inline-flex; align-items:center; background: rgba(255, 123, 114, 0.1); color: #ff7b72; padding: 4px 10px; border-radius: 12px; font-size: 11px; font-weight: bold;">${iconAlert} ${activeSubAlerts} ACTIVE / ${group.incidents.length} TOTAL</span>`;

        let subAlertsHTML = group.incidents.map(inc => {
            const isFlaky = inc.name.includes("[FLAKY]");
            const flakyBadge = isFlaky ? `<span style="color: #d29922; margin-right: 8px;">${iconWarning} FLAKY</span>` : ``;
            const cleanName = inc.name.replace("[FLAKY] ", "");
            const displayLog = inc.error_logs || inc.error_message || "No logs available.";
            const isResolved = normalizeIsResolved(inc);
            const textDecoration = isResolved ? 'text-decoration: line-through;' : '';
            const textColor = isResolved ? '#3fb950' : 'var(--text-main)';
            const isChecked = window.selectedIncidents.has(String(inc.id)) ? 'checked' : '';
            const bgStyle = isResolved ? 'background: rgba(63, 185, 80, 0.05); border-left: 3px solid #3fb950;' : 'background: #0d1117; border-left: 3px solid #30363d;';

            const subAlertFlag = isResolved
                ? `<span style="display:inline-flex; align-items:center; background: rgba(63, 185, 80, 0.1); color: #3fb950; padding: 2px 6px; border-radius: 4px; font-size: 9px; font-weight: bold; margin-left: 10px;">${iconCheck} RESOLVED BY ${inc.resolved_by || 'SYSTEM'}</span>`
                : `<span style="display:inline-flex; align-items:center; background: rgba(255, 123, 114, 0.1); color: #ff7b72; padding: 2px 6px; border-radius: 4px; font-size: 9px; font-weight: bold; margin-left: 10px;">${iconAlert} ACTIVE TEST</span>`;

            return `
            <div style="${bgStyle} border: 1px solid #30363d; border-radius: 6px; margin-top: 10px; padding: 12px; transition: all 0.3s ease;">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                    <div style="display:flex; align-items:center; gap: 10px;">
                        <input type="checkbox" id="cb-inc-${inc.id}" onclick="toggleIncidentSelection('${inc.id}', this.checked)" ${isChecked} style="width: 14px; height: 14px; cursor: pointer;">
                        <strong style="font-size: 13px; color: ${textColor}; ${textDecoration}">${flakyBadge}${cleanName}</strong>
                        ${subAlertFlag}
                    </div>
                </div>
                <div style="font-family: monospace; font-size: 11px; color: #8b949e; background: #161b22; padding: 10px; border-radius: 4px; overflow-x: auto; white-space: pre-wrap; max-height: 150px; overflow-y: auto;">${displayLog}</div>
            </div>`;
        }).join('');

        let actionDropdown = `
            <div style="position: relative; display: inline-block;" class="action-dropdown-container">
                <button class="btn-secondary" style="font-size: 11px; padding: 5px 10px; display: flex; align-items: center;" onclick="toggleActionDropdown('${group.id}', event)">${iconGear} Actions</button>
                <div id="action-dropdown-${group.id}" style="display: none; position: absolute; right: 0; top: 30px; background: #161b22; border: 1px solid #30363d; z-index: 10; border-radius: 6px; min-width: 180px; box-shadow: 0 4px 12px rgba(0,0,0,0.5); overflow: hidden;">
                    ${(!group.is_resolved && canResolve) ? `<a href="#" onclick="resolveGroup('${group.id}', event)" style="display: flex; align-items: center; padding: 10px 12px; color: #3fb950; text-decoration: none; font-size: 12px; border-bottom: 1px solid #30363d; transition: background 0.2s;"><span style="flex-grow:1;">${iconCheck} Resolve Execution</span></a>` : ''}
                    ${isAdmin ? `<a href="#" onclick="deleteGroup('${group.id}', event)" style="display: flex; align-items: center; padding: 10px 12px; color: #ff7b72; text-decoration: none; font-size: 12px; border-bottom: 1px solid #30363d; transition: background 0.2s;"><span style="flex-grow:1;">${iconTrash} Delete Execution</span></a>` : ''}
                    <a href="#" onclick="downloadGroupLog('${group.id}', 'error', event)" style="display: flex; align-items: center; padding: 10px 12px; color: #ff7b72; text-decoration: none; font-size: 12px; border-bottom: 1px solid #30363d; transition: background 0.2s;"><span style="flex-grow:1;">${iconAlert} Export Errors</span></a>
                    <a href="#" onclick="downloadGroupLog('${group.id}', 'full', event)" style="display: flex; align-items: center; padding: 10px 12px; color: #c9d1d9; text-decoration: none; font-size: 12px; border-bottom: 1px solid #30363d; transition: background 0.2s;"><span style="flex-grow:1;">${iconFile} Export Full Logs</span></a>
                    <a href="#" onclick="downloadGroupLog('${group.id}', 'xml', event)" style="display: flex; align-items: center; padding: 10px 12px; color: #58a6ff; text-decoration: none; font-size: 12px; transition: background 0.2s;"><span style="flex-grow:1;">${iconCode} Generate JUnit XML</span></a>
                </div>
            </div>`;

        const groupAllSelected = group.incidents.length > 0 && group.incidents.every(inc => window.selectedIncidents.has(String(inc.id)));
        const groupChecked = groupAllSelected ? 'checked' : '';

        return `
        <div class="data-card" style="border-left: 4px solid ${severityColor}; margin-bottom: 20px; opacity: ${group.is_resolved ? '0.7' : '1'}; transition: 0.3s; padding: 20px;">
            <div style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #30363d; padding-bottom: 15px;">
                <div style="display: flex; align-items: center; gap: 15px;">
                    <input type="checkbox" id="cb-group-${group.id}" onclick="toggleGroupSelection('${group.id}', this.checked)" ${groupChecked} style="width: 16px; height: 16px; cursor: pointer;">
                    <div>
                        <div style="font-size: 11px; color: #8b949e; margin-bottom: 4px; text-transform: uppercase; letter-spacing: 1px;">Pipeline Execution</div>
                        <strong style="font-size: 16px; color: var(--text-main);">${group.project_name}</strong>
                        <span style="font-size: 12px; color: #8b949e; margin-left: 10px;">• ${group.created_at}</span>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 10px;">
                    ${resolvedBadge}
                    ${actionDropdown}
                    <button class="btn-secondary" style="border: none; background: #161b22; padding: 6px 12px; display: flex; align-items: center; gap: 6px;" onclick="toggleSubAlerts('${group.id}', event)">
                        ${group.incidents.length} Alert(s) <span id="toggle-icon-${group.id}" style="display: inline-block;">${iconChevron}</span>
                    </button>
                </div>
            </div>
            <div id="sub-alerts-${group.id}" style="display: none; padding-top: 10px;">${subAlertsHTML}</div>
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

window.connectWebSocket = function () {
    setInterval(() => {
        const dashboard = document.getElementById('view-dashboard');
        if (Date.now() < window.pausePollingUntil) return;
        if (dashboard && dashboard.classList.contains('active')) window.fetchIncidents();
    }, 3000);
}

window.performLogin = function () {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    fetch('/api/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) })
        .then(res => { if (!res.ok) throw new Error(); return res.json(); })
        .then(data => { localStorage.setItem('sre-jwt', data.token); window.checkAuth(); })
        .catch(() => { document.getElementById('login-error').style.display = 'block'; notify("Invalid credentials", "error"); });
}

window.checkAuth = function () {
    const token = localStorage.getItem('sre-jwt');
    if (!token) {
        document.getElementById('login-screen').style.display = 'flex';
        document.getElementById('app-container').style.display = 'none';
        return;
    }
    const payload = parseJwt(token);

    if (payload.require_password_change) {
        document.getElementById('login-screen').style.display = 'none';
        document.getElementById('app-container').style.display = 'none';
        document.getElementById('force-password-screen').style.display = 'flex';
        return;
    }

    document.getElementById('login-screen').style.display = 'none';
    document.getElementById('force-password-screen').style.display = 'none';
    document.getElementById('app-container').style.display = 'flex';

    window.applyPermissions();
    window.connectWebSocket();

    if (payload.role === 'admin') window.loadUsers();

    if (document.getElementById('view-dashboard').classList.contains('active')) {
        window.loadDashboardFilters(); window.pausePollingUntil = 0; window.fetchIncidents(true);
    }
    if (document.getElementById('view-organizations').classList.contains('active')) window.loadOrganizations();
    if (document.getElementById('view-management').classList.contains('active')) {
        if (payload.role === 'admin') window.renderUserTable(iam.allUsers);
    }
    if (document.getElementById('view-settings').classList.contains('active')) {
        window.loadConfig(); window.loadFinOps();
    }
    if (document.getElementById('view-plugins').classList.contains('active')) window.loadPlugins();
    if (document.getElementById('view-ingestion').classList.contains('active')) {
        if (payload.role === 'admin') window.loadGatewaysData();
    }
}

window.submitNewPassword = function () {
    const p1 = document.getElementById('force-new-pass').value;
    const p2 = document.getElementById('force-new-pass-confirm').value;
    if (!p1 || p1 !== p2) return notify("Passwords do not match", "error");

    fetchWithAuth('/api/users/change-password', { method: 'POST', body: JSON.stringify({ new_password: p1 }) })
        .then(res => {
            if (res.ok) { notify("Password secured. Please login again.", "success"); setTimeout(() => window.performLogout(), 2000); }
            else notify("Error updating password", "error");
        }).catch(() => notify("Network error", "error"));
}

window.switchView = function (id, el) {
    document.querySelectorAll('.view-section').forEach(x => x.classList.remove('active'));
    document.querySelectorAll('.nav-item').forEach(x => x.classList.remove('active'));
    document.getElementById('view-' + id).classList.add('active');
    el.classList.add('active');

    const payload = parseJwt(localStorage.getItem('sre-jwt'));

    if (id === 'dashboard') { window.loadDashboardFilters(); window.pausePollingUntil = 0; window.fetchIncidents(true); }
    if (id === 'organizations') window.loadOrganizations();
    if (id === 'management' && payload.role === 'admin') window.renderUserTable(iam.allUsers);
    if (id === 'plugins') window.loadPlugins();
    if (id === 'settings') { window.loadConfig(); window.loadFinOps(); }
    if (id === 'ingestion' && payload.role === 'admin') window.loadGatewaysData();
}

window.applyPermissions = function () {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    document.querySelectorAll('.admin-only').forEach(x => x.style.display = payload.role === 'admin' ? '' : 'none');
}

window.onload = function () {
    window.checkAuth();
    window.checkSSOStatus();
    const savedCurrency = localStorage.getItem('selected-currency');
    if (savedCurrency) { window.setSelectedCurrency(savedCurrency); }
    window.updateCurrencyDisplay();
    setTimeout(() => window.setStatusFilter(window.statusFilter || 'all'), 100);
};