/**
* web/app.js
* Main controller for QA Capsule Control Plane
*/

let allUsers = [];
let allProjects = [];
let editingProjectId = null;

// Global state management
window.currentIncidents = [];
window.selectedIncidents = new Set();
window.pausePollingUntil = 0;
window.isFirstLoad = true;
window.statusFilter = 'all'; // 'all', 'active', 'resolved'
window.selectedCurrency = 'USD'; // Default currency
// Persist resolved sub-alerts until the server confirms (survives polling/refresh)
window.pendingResolvedIds = new Map();
window._resolveRetryInFlight = false;

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
        window.pendingResolvedIds.set(String(id), {
            resolvedBy: resolvedBy || 'You',
            since: Date.now()
        });
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
        return {
            ...inc,
            is_resolved: true,
            status: 'resolved',
            resolved_by: inc.resolved_by || pending.resolvedBy
        };
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
        const res = await window.fetchWithAuth('/api/incidents', {
            method: 'PUT',
            body: JSON.stringify({ ids: uniqueIds })
        });
        if (!res.ok) throw new Error('Update failed on server side');

        notify(successMessage || 'Alerts resolved', 'success');

        for (let attempt = 0; attempt < 8; attempt++) {
            await new Promise(r => setTimeout(r, attempt === 0 ? 600 : 800));
            await window.fetchIncidents(true, { skipPauseCheck: true });

            // Pending entries are cleared only when the server returns is_resolved=true
            const allConfirmed = uniqueIds.every(id => !window.pendingResolvedIds.has(String(id)));
            if (allConfirmed) break;
        }

        if (uniqueIds.some(id => window.pendingResolvedIds.has(String(id))) && !window._resolveRetryInFlight) {
            window._resolveRetryInFlight = true;
            try {
                await window.fetchWithAuth('/api/incidents', {
                    method: 'PUT',
                    body: JSON.stringify({ ids: uniqueIds })
                });
                await new Promise(r => setTimeout(r, 800));
                await window.fetchIncidents(true, { skipPauseCheck: true });
            } finally {
                window._resolveRetryInFlight = false;
            }
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

// Currency symbols
window.currencySymbols = {
    'USD': '$', 'EUR': '€', 'GBP': '£', 'JPY': '¥', 'AUD': 'A$',
    'CAD': 'C$', 'CHF': 'Fr', 'INR': '₹', 'CNY': '¥', 'MXN': '$',
    'SGD': 'S$', 'NZD': 'NZ$'
};

// ==========================================
// NOTIFICATIONS & THEME
// ==========================================

/**
 * Displays a toast notification
 * @param {string} message - Text to display
 * @param {string} type - 'success' or 'error'
 */
function notify(message, type = 'success') {
    const container = document.getElementById('notification-container');
    if (!container) return alert(message);

    const toast = document.createElement('div');
    const bgColor = type === 'error' ? '#ff4444' : '#00C851';

    toast.style.cssText = `background-color: ${bgColor}; color: white; padding: 15px 20px; border-radius: 4px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); font-weight: bold; font-size: 14px; transition: opacity 0.5s ease; margin-top: 10px; z-index: 99999;`;
    toast.innerHTML = `<span>${message}</span>`;
    container.appendChild(toast);
    setTimeout(() => { toast.style.opacity = '0'; setTimeout(() => toast.remove(), 500); }, 4000);
}

function initTheme() { if (localStorage.getItem('sre-theme') === 'dark') document.body.setAttribute('data-theme', 'dark'); }
function toggleTheme() {
    const isDark = document.body.hasAttribute('data-theme');
    isDark ? document.body.removeAttribute('data-theme') : document.body.setAttribute('data-theme', 'dark');
    localStorage.setItem('sre-theme', isDark ? 'light' : 'dark');
}
initTheme();

function parseJwt(token) { try { return JSON.parse(atob(token.split('.')[1])); } catch (e) { return {}; } }

// ==========================================
// CUSTOM MODAL LOGIC
// ==========================================

window.showConfirmModal = function (title, message, type, confirmCallback) {
    const modal = document.getElementById('custom-modal');
    const box = document.getElementById('custom-modal-box');
    const titleEl = document.getElementById('custom-modal-title');

    titleEl.innerText = title;
    document.getElementById('custom-modal-message').innerText = message;
    document.getElementById('custom-modal-input').style.display = 'none';

    if (type === 'danger') {
        box.style.borderColor = 'var(--log-fatal)';
        titleEl.style.color = 'var(--log-fatal)';
    } else if (type === 'warning') {
        box.style.borderColor = 'var(--log-warn)';
        titleEl.style.color = 'var(--log-warn)';
    } else {
        box.style.borderColor = 'var(--text-main)';
        titleEl.style.color = 'var(--text-main)';
    }

    const confirmBtn = document.getElementById('custom-modal-confirm');
    confirmBtn.onclick = function () {
        closeModal();
        confirmCallback();
    };
    modal.style.display = 'flex';
};

window.showPromptModal = function (title, message, placeholder, confirmCallback, isPassword = false) {
    const modal = document.getElementById('custom-modal');
    const box = document.getElementById('custom-modal-box');
    const titleEl = document.getElementById('custom-modal-title');
    const inputEl = document.getElementById('custom-modal-input');

    titleEl.innerText = title;
    document.getElementById('custom-modal-message').innerText = message;

    inputEl.style.display = 'block';
    inputEl.type = isPassword ? 'password' : 'text';
    inputEl.placeholder = placeholder;
    inputEl.value = '';

    box.style.borderColor = 'var(--text-main)';
    titleEl.style.color = 'var(--text-main)';

    const confirmBtn = document.getElementById('custom-modal-confirm');
    confirmBtn.onclick = function () {
        const val = inputEl.value;
        if (val) {
            closeModal();
            confirmCallback(val);
        }
    };
    modal.style.display = 'flex';
    inputEl.focus();
};

window.closeModal = function () {
    document.getElementById('custom-modal').style.display = 'none';
};

// ==========================================
// CI/CD INGESTION GATEWAY LOGIC
// ==========================================

let currentSelectedCI = 'gitlab';

window.selectCI = function (ciName, element) {
    document.querySelectorAll('.ci-card').forEach(card => card.classList.remove('active'));
    element.classList.add('active');
    currentSelectedCI = ciName;

    const dynamicField = document.getElementById('dynamic-ci-field');
    const specificInput = document.getElementById('ci-specific');

    if (ciName === 'gitlab') {
        dynamicField.querySelector('label').innerText = 'GitLab Repository Path';
        specificInput.placeholder = 'group/project';
    } else if (ciName === 'github') {
        dynamicField.querySelector('label').innerText = 'GitHub Action Workflow ID';
        specificInput.placeholder = 'build-deploy.yml';
    } else if (ciName === 'jenkins') {
        dynamicField.querySelector('label').innerText = 'Jenkins Job Name';
        specificInput.placeholder = 'PROD-Deployment-Job';
    }

    const baseUrl = window.location.origin;
    document.getElementById('ci-webhook-url').value = `${baseUrl}/api/webhooks/${ciName}`;

    if (!editingProjectId) {
        const apiKeyInput = document.getElementById('ci-api-key');
        if (apiKeyInput) {
            apiKeyInput.type = 'text';
            const randomHex = Math.random().toString(16).substring(2, 10);
            apiKeyInput.value = `sre_pk_${ciName}_${randomHex}`;
        }
    }
}

window.loadGatewaysData = function () {
    window.fetchWithAuth(`/api/my-projects?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(projects => {
            allProjects = projects || [];
            const tbody = document.getElementById('projects-table-body');
            if (!allProjects || allProjects.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" style="text-align:center; padding:20px; opacity:0.5;">No pipelines provisioned yet.</td></tr>';
                return;
            }
            tbody.innerHTML = allProjects.map(p => `
                <tr style="border-bottom: 1px solid var(--border-main);">
                    <td style="padding: 10px;"><strong>${p.name}</strong></td>
                    <td style="padding: 10px; text-transform: uppercase;">${p.ci_system}</td>
                    <td style="padding: 10px;"><code style="color: #ff7b72; font-family: monospace;">••••••••••••</code></td>
                    <td style="padding: 10px; text-align: right;">
                        <button class="btn-secondary" style="font-size:10px; padding:4px 8px; color:#58a6ff; border-color:#58a6ff; margin-right:5px;" onclick="window.editProject('${p.id}')">EDIT</button>
                        <button class="btn-secondary" style="font-size:10px; padding:4px 8px; color:#ff7b72; border-color:#ff7b72;" onclick="window.deleteProject('${p.id}')">DELETE</button>
                    </td>
                </tr>
            `).join('');
        });

    window.fetchWithAuth(`/api/teams?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(teams => {
            const select = document.getElementById('ci-team-id');
            select.innerHTML = '<option value="" disabled selected>Select a Team</option>';
            if (teams) teams.forEach(t => select.innerHTML += `<option value="${t.id}">${t.name}</option>`);
        });
}

window.editProject = function (projectId) {
    const p = allProjects.find(item => item.id === projectId);
    if (!p) return;

    editingProjectId = projectId;
    document.getElementById('ci-project-name').value = p.name || "";
    document.getElementById('ci-team-id').value = p.team_id || "";
    document.getElementById('ci-specific').value = p.repo_path || "";

    const routing = p.routing || {};
    document.getElementById('ci-slack-channel').value = p.slack_channel || routing.slack_channel || "";
    document.getElementById('ci-jira-key').value = p.jira_project_key || routing.jira_project_key || "";
    document.getElementById('ci-teams-webhook').value = p.teams_webhook || routing.teams_webhook || "";

    const ciCard = document.querySelector(`.ci-card[onclick*="'${p.ci_system}'"]`);
    if (ciCard) window.selectCI(p.ci_system, ciCard);

    const apiKeyInput = document.getElementById('ci-api-key');
    if (apiKeyInput) {
        apiKeyInput.value = p.api_key || "";
        apiKeyInput.type = 'password';
    }

    const btn = document.querySelector("button[onclick='saveCIConfig()']");
    if (btn) {
        btn.innerText = "Update Project Endpoint";
        btn.style.borderColor = "#58a6ff";
        btn.style.color = "#58a6ff";
    }

    document.getElementById('ci-project-name').focus();
    notify("Editing project: " + p.name, "success");
}

window.deleteProject = function (projectId) {
    showConfirmModal("Delete Pipeline Endpoint?", "Are you sure? This will permanently delete this pipeline endpoint and all its routing configurations.", "danger", function () {
        window.fetchWithAuth(`/api/config/projects?id=${projectId}`, { method: 'DELETE' })
            .then(res => {
                if (res.ok) { notify("Pipeline deleted successfully.", "success"); window.loadGatewaysData(); window.loadDashboardFilters(); }
                else notify("Failed to delete pipeline.", "error");
            });
    });
}

window.saveCIConfig = function () {
    const projectName = document.getElementById('ci-project-name').value;
    const teamId = document.getElementById('ci-team-id').value;

    if (!projectName) return notify("Error: Project Name is required.", "error");
    if (!teamId) return notify("Error: You must assign this pipeline to a Team.", "error");

    const payload = {
        id: editingProjectId || projectName.toLowerCase().replace(/\s+/g, '-'),
        name: projectName,
        team_id: parseInt(teamId),
        ci_system: currentSelectedCI,
        repo_path: document.getElementById('ci-specific').value,
        api_key: document.getElementById('ci-api-key').value,
        routing: {
            slack_channel: document.getElementById('ci-slack-channel').value,
            jira_project_key: document.getElementById('ci-jira-key').value,
            teams_webhook: document.getElementById('ci-teams-webhook').value
        }
    };

    const method = editingProjectId ? 'PUT' : 'POST';

    window.fetchWithAuth('/api/config/projects', { method: method, body: JSON.stringify(payload) })
        .then(res => {
            if (res.ok) {
                notify(editingProjectId ? "Project updated!" : "Endpoint Provisioned!", "success");

                editingProjectId = null;
                document.getElementById('ci-project-name').value = "";
                document.getElementById('ci-specific').value = "";
                document.getElementById('ci-slack-channel').value = "";
                document.getElementById('ci-jira-key').value = "";
                document.getElementById('ci-teams-webhook').value = "";

                const apiKeyInput = document.getElementById('ci-api-key');
                if (apiKeyInput) {
                    apiKeyInput.type = 'text';
                    const randomHex = Math.random().toString(16).substring(2, 10);
                    apiKeyInput.value = `sre_pk_${currentSelectedCI}_${randomHex}`;
                }

                const btn = document.querySelector("button[onclick='saveCIConfig()']");
                if (btn) {
                    btn.innerText = "Provision Project Endpoint";
                    btn.style.borderColor = "";
                    btn.style.color = "";
                }

                window.loadGatewaysData();
                window.loadDashboardFilters();
            } else {
                notify("Operation failed", "error");
            }
        }).catch(() => notify("Network error", "error"));
}

window.revealApiKey = function () {
    const apiKeyInput = document.getElementById('ci-api-key');
    if (!apiKeyInput) return;

    if (apiKeyInput.type === 'text') {
        apiKeyInput.type = 'password';
        return;
    }

    const token = localStorage.getItem('sre-jwt');
    if (!token) return;
    const currentUser = parseJwt(token).username;

    showPromptModal("Security Verification", "Please enter your password to reveal this secret API Key.", "Password...", function (password) {
        fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username: currentUser, password: password })
        })
            .then(res => {
                if (res.ok) {
                    apiKeyInput.type = 'text';
                    notify("Secret revealed. It will be masked on reload.", "success");
                } else {
                    notify("Access Denied: Incorrect password", "error");
                }
            }).catch(() => notify("Network error", "error"));
    }, true);
};

// ==========================================
// AUTHENTICATION & NAVIGATION
// ==========================================

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

    if (payload.role === 'admin') {
        window.loadUsers();
    }

    if (document.getElementById('view-dashboard').classList.contains('active')) {
        window.loadDashboardFilters();
        window.pausePollingUntil = 0;
        window.fetchIncidents(true);
    }
    if (document.getElementById('view-organizations').classList.contains('active')) window.loadOrganizations();
    if (document.getElementById('view-management').classList.contains('active')) {
        if (payload.role === 'admin') window.renderUserTable(allUsers);
    }
    if (document.getElementById('view-settings').classList.contains('active')) {
        window.loadConfig();
        window.loadFinOps();
    }
    if (document.getElementById('view-plugins').classList.contains('active')) window.loadPlugins();
    if (document.getElementById('view-ingestion').classList.contains('active')) {
        if (payload.role === 'admin') window.loadGatewaysData();
    }
}

window.performLogout = function () { localStorage.removeItem('sre-jwt'); location.reload(); }

window.fetchWithAuth = function (url, opts = {}) {
    const token = localStorage.getItem('sre-jwt');
    opts.headers = { ...opts.headers, 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' };
    return fetch(url, opts).then(res => { if (res.status === 401) window.performLogout(); return res; });
}

window.submitNewPassword = function () {
    const p1 = document.getElementById('force-new-pass').value;
    const p2 = document.getElementById('force-new-pass-confirm').value;
    if (!p1 || p1 !== p2) return notify("Passwords do not match", "error");

    window.fetchWithAuth('/api/users/change-password', { method: 'POST', body: JSON.stringify({ new_password: p1 }) })
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
    if (id === 'management' && payload.role === 'admin') window.renderUserTable(allUsers);
    if (id === 'plugins') window.loadPlugins();
    if (id === 'settings') {
        window.loadConfig();
        window.loadFinOps();
    }
    if (id === 'ingestion' && payload.role === 'admin') window.loadGatewaysData();
}

window.applyPermissions = function () {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    document.querySelectorAll('.admin-only').forEach(x => x.style.display = payload.role === 'admin' ? '' : 'none');
}

// ==========================================
// ANALYTICS DASHBOARD LOGIC 
// ==========================================

let analyticsChart = null;

window.toggleAnalytics = function () {
    const view = document.getElementById('analytics-view');
    if (view.style.display === 'none') {
        view.style.display = 'block';
        window.loadAnalytics();
    } else {
        view.style.display = 'none';
    }
};

window.loadAnalytics = function () {
    window.fetchWithAuth(`/api/metrics?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            document.getElementById('mttr-value').innerText = data.mttr_minutes + " min";

            if (analyticsChart) {
                analyticsChart.destroy();
            }

            const ctx = document.getElementById('flakyChart').getContext('2d');
            analyticsChart = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Stable Failures (Real Bugs)', 'Flaky Tests (Unstable)'],
                    datasets: [{
                        data: [data.stable_failures, data.flaky_tests],
                        backgroundColor: ['#ff7b72', '#d29922'],
                        borderColor: '#0d1117',
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: { position: 'bottom', labels: { color: '#c9d1d9' } },
                        title: { display: true, text: 'Failure Quality Assessment', color: '#c9d1d9' }
                    }
                }
            });
        })
        .catch(err => console.error("Error loading analytics:", err));
};

// ==========================================
// WEEKLY REPORT GENERATOR (CSV & PDF)
// ==========================================

window.downloadWeeklyReportCSV = function () {
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';

    window.fetchWithAuth(`/api/reports/weekly?project=${encodeURIComponent(projectFilter)}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            if (!data || data.length === 0) {
                return notify("Aucun incident enregistré ces 7 derniers jours.", "warning");
            }

            let csvContent = "Pipeline Name,Total Alerts,Resolved Alerts,Flaky Tests,Health Score (%)\n";

            data.forEach(row => {
                csvContent += `${row.pipeline},${row.total_alerts},${row.resolved_alerts},${row.flaky_tests},${row.health_score}%\n`;
            });

            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            const dateStr = new Date().toISOString().split('T')[0];
            let suffix = projectFilter === 'all' ? 'Global' : projectFilter;
            a.download = `QA_Capsule_Weekly_Report_${suffix}_${dateStr}.csv`;

            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);

            notify("Rapport CSV généré avec succès !", "success");
        })
        .catch(() => notify("Erreur lors de la génération du rapport CSV", "error"));
};

window.downloadWeeklyReportPDF = function () {
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';

    window.fetchWithAuth(`/api/reports/weekly?project=${encodeURIComponent(projectFilter)}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            if (!data || data.length === 0) {
                return notify("Aucun incident enregistré ces 7 derniers jours.", "warning");
            }

            const { jsPDF } = window.jspdf;
            const doc = new jsPDF();

            // Header Design
            doc.setFillColor(13, 17, 23);
            doc.rect(0, 0, 210, 40, 'F');

            doc.setTextColor(88, 166, 255);
            doc.setFontSize(22);
            doc.text("QA Flight Recorder", 14, 20);

            doc.setTextColor(201, 209, 217);
            doc.setFontSize(12);
            let titleScope = projectFilter === 'all' ? 'Global Organization' : `Project: ${projectFilter}`;
            doc.text(`Weekly SRE Metrics & Pipeline Health (${titleScope})`, 14, 28);

            doc.setFontSize(10);
            doc.setTextColor(139, 148, 158);
            doc.text(`Generated: ${new Date().toLocaleString()}`, 14, 34);

            const tableColumn = ["Pipeline Name", "Total Alerts", "Resolved", "Flaky Tests", "Health Score"];
            const tableRows = [];

            data.forEach(row => {
                tableRows.push([
                    row.pipeline,
                    row.total_alerts,
                    row.resolved_alerts,
                    row.flaky_tests,
                    row.health_score + "%"
                ]);
            });

            doc.autoTable({
                head: [tableColumn],
                body: tableRows,
                startY: 45,
                theme: 'grid',
                headStyles: { fillColor: [22, 27, 34], textColor: [201, 209, 217], lineColor: [48, 54, 61], lineWidth: 0.1 },
                bodyStyles: { fillColor: [13, 17, 23], textColor: [201, 209, 217], lineColor: [48, 54, 61], lineWidth: 0.1 },
                alternateRowStyles: { fillColor: [22, 27, 34] },
                styles: { font: 'helvetica', fontSize: 10, cellPadding: 5 }
            });

            const dateStr = new Date().toISOString().split('T')[0];
            let suffix = projectFilter === 'all' ? 'Global' : projectFilter;
            doc.save(`QA_Capsule_Weekly_Report_${suffix}_${dateStr}.pdf`);
            notify("Rapport PDF généré avec succès !", "success");
        })
        .catch(() => notify("Erreur lors de la génération du rapport PDF", "error"));
};


// ==========================================
// BULK ACTIONS & SELECTIONS LOGIC
// ==========================================

window.toggleIncidentSelection = function (id, checked) {
    window.pausePollingUntil = Date.now() + 15000;
    const strId = String(id);

    if (checked) window.selectedIncidents.add(strId);
    else window.selectedIncidents.delete(strId);

    const masterCb = document.getElementById('select-all-cb');
    if (masterCb) {
        const allChecked = window.currentIncidents.length > 0 && window.currentIncidents.every(inc => window.selectedIncidents.has(String(inc.id)));
        masterCb.checked = allChecked;
    }
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
    if (masterCb) {
        const allChecked = window.currentIncidents.length > 0 && window.currentIncidents.every(inc => window.selectedIncidents.has(String(inc.id)));
        masterCb.checked = allChecked;
    }
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

    const ids = Array.from(window.selectedIncidents)
        .map(id => parseInt(id, 10))
        .filter(id => !isNaN(id));
    if (ids.length === 0) return;

    window.selectedIncidents.clear();
    window.updateBulkActionUI();
    await window.resolveIncidentsByIds(ids, 'Alerts resolved');
};

window.deleteSelected = async function () {
    if (window.selectedIncidents.size === 0) return notify("Sélectionnez au moins une alerte.", "error");
    
    const userRole = parseJwt(localStorage.getItem('sre-jwt')).role;
    if (userRole !== 'admin') return notify("Seuls les admins peuvent supprimer.", "error");

    showConfirmModal("Delete Selected?", `Supprimer définitivement ${window.selectedIncidents.size} alertes ?`, "danger", async function () {
        window.pausePollingUntil = Date.now() + 15000;
        
        const ids = Array.from(window.selectedIncidents).map(id => parseInt(id, 10)).filter(id => !isNaN(id));

        window.currentIncidents = window.currentIncidents.filter(inc => !ids.includes(parseInt(inc.id, 10)));
        window.selectedIncidents.clear();
        window.renderIncidentsList();
        notify("Suppression en cours...", "success");

        try {
            // ENVOI INDIVIDUEL DE CHAQUE SUPPRESSION AVEC LE PARAMÈTRE "id="
            const promises = ids.map(id => {
                return window.fetchWithAuth(`/api/incidents?id=${id}&ids=${id}`, { method: 'DELETE' });
            });

            await Promise.all(promises);

            setTimeout(() => {
                window.pausePollingUntil = 0;
                window.fetchIncidents(true);
                window.fetchMetricsOnly();
            }, 1200);
        } catch (e) {
            console.error("Delete error", e);
            window.pausePollingUntil = 0;
            window.fetchIncidents(true);
        }
    });
};

// ==========================================
// GROUP ACTIONS
// ==========================================

window.resolveGroup = async function (groupId, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    const dropdown = document.getElementById(`action-dropdown-${groupId}`);
    if (dropdown) dropdown.style.display = 'none';

    const group = window.groupedIncidents[groupId];
    if (!group) return;

    const selectedInGroup = group.incidents.filter(inc => window.selectedIncidents.has(String(inc.id)));
    const targetIncidents = selectedInGroup.length > 0 ? selectedInGroup : group.incidents;

    const ids = targetIncidents
        .filter(inc => !normalizeIsResolved(inc))
        .map(inc => parseInt(inc.id, 10))
        .filter(id => !isNaN(id));

    if (ids.length === 0) return;

    targetIncidents.forEach(i => window.selectedIncidents.delete(String(i.id)));
    window.updateBulkActionUI();

    const msg = ids.length === 1 ? 'Sub-alert résolue.' : 'Pipelines marqués comme résolus.';
    await window.resolveIncidentsByIds(ids, msg);
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
            // ENVOI INDIVIDUEL DE LA SUPPRESSION AVEC "?id="
            const promises = ids.map(id => {
                return window.fetchWithAuth(`/api/incidents?id=${id}&ids=${id}`, { method: 'DELETE' });
            });

            await Promise.all(promises);

            setTimeout(() => {
                window.pausePollingUntil = 0;
                window.fetchIncidents(true);
                window.fetchMetricsOnly();
            }, 1200);
        } catch (e) {
            window.pausePollingUntil = 0;
            window.fetchIncidents(true);
        }
    });
}

// ==========================================
// ACTION DROPDOWN & LOG EXPORT
// ==========================================

window.toggleActionDropdown = function (id, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    const dropdown = document.getElementById(`action-dropdown-${id}`);
    if (dropdown) {
        const isCurrentlyHidden = dropdown.style.display === 'none';

        document.querySelectorAll('[id^="action-dropdown-"]').forEach(el => {
            el.style.display = 'none';
        });

        if (isCurrentlyHidden) {
            dropdown.style.display = 'block';
            window.pausePollingUntil = Date.now() + 15000;
        } else {
            dropdown.style.display = 'none';
            window.pausePollingUntil = 0;
        }
    }
}

document.addEventListener('click', function (event) {
    if (!event.target.closest('.action-dropdown-container')) {
        document.querySelectorAll('[id^="action-dropdown-"]').forEach(el => {
            el.style.display = 'none';
        });
    }
});

window.downloadGroupLog = function (groupId, type, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    document.getElementById(`action-dropdown-${groupId}`).style.display = 'none';
    window.pausePollingUntil = 0;

    const group = window.groupedIncidents[groupId];
    if (!group) return notify("Could not retrieve group data.", "error");

    let fileContent = "";
    let fileName = "";
    let blobType = "text/plain";

    if (type === 'error') {
        fileName = `execution_${groupId}_errors.log`;
        group.incidents.forEach(inc => {
            fileContent += `--- [TEST: ${inc.name}] ---\n`;
            fileContent += `${inc.error_logs || inc.error_message || "No error details."}\n\n`;
        });
    } else if (type === 'full') {
        fileName = `execution_${groupId}_full_logs.log`;
        group.incidents.forEach(inc => {
            fileContent += `=======================================\n`;
            fileContent += `TEST: ${inc.name}\n`;
            fileContent += `=======================================\n`;
            fileContent += `[ERROR LOGS]\n${inc.error_logs || inc.error_message || "N/A"}\n\n`;
            fileContent += `[STDOUT]\n${inc.console_logs || "N/A"}\n\n`;
        });
    } else if (type === 'xml') {
        fileName = `execution_${groupId}_results.xml`;
        blobType = "application/xml";
        fileContent = `<?xml version="1.0" encoding="UTF-8"?>\n<testsuites>\n  <testsuite name="QA_Capsule_Reconstructed_Suite" tests="${group.incidents.length}" failures="${group.incidents.length}">\n`;

        group.incidents.forEach(inc => {
            const safeName = (inc.name || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
            const safeErr = (inc.error_message || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            const safeLogs = (inc.error_logs || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            const safeOut = (inc.console_logs || "").replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

            fileContent += `    <testcase name="${safeName}">\n`;
            fileContent += `      <failure message="${safeErr}">${safeLogs}</failure>\n`;
            if (safeOut) fileContent += `      <system-out>${safeOut}</system-out>\n`;
            fileContent += `    </testcase>\n`;
        });

        fileContent += `  </testsuite>\n</testsuites>`;
    }

    const blob = new Blob([fileContent], { type: blobType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

window.toggleSubAlerts = function (groupId, event) {
    if (event) { event.preventDefault(); event.stopPropagation(); }
    window.pausePollingUntil = Date.now() + 15000;
    const container = document.getElementById(`sub-alerts-${groupId}`);
    const icon = document.getElementById(`toggle-icon-${groupId}`);
    if (container.style.display === 'none') {
        container.style.display = 'block';
        icon.style.transform = 'rotate(180deg)';
    } else {
        container.style.display = 'none';
        icon.style.transform = 'rotate(0deg)';
    }
}

// ==========================================
// DASHBOARD RENDER LOGIC 
// ==========================================

window.loadDashboardFilters = function () {
    window.fetchWithAuth(`/api/my-projects?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(projects => {
            const filter = document.getElementById('project-filter');
            const currentVal = filter.value;
            filter.innerHTML = '<option value="all">All My Projects</option>';
            if (projects) {
                projects.forEach(p => filter.innerHTML += `<option value="${p.name}">${p.name}</option>`);
            }
            if (currentVal) filter.value = currentVal;
        }).catch(e => console.log("Error loading filter"));
}

window.setStatusFilter = function (status) {
    window.statusFilter = status;

    const allBtn = document.getElementById('status-all-btn');
    const activeBtn = document.getElementById('status-active-btn');
    const resolvedBtn = document.getElementById('status-resolved-btn');

    [allBtn, activeBtn, resolvedBtn].forEach(btn => {
        if (!btn) return;
        btn.classList.remove('active-all', 'active-active', 'active-resolved');
    });

    if (status === 'all' && allBtn) allBtn.classList.add('active-all');
    else if (status === 'active' && activeBtn) activeBtn.classList.add('active-active');
    else if (status === 'resolved' && resolvedBtn) resolvedBtn.classList.add('active-resolved');

    window.fetchIncidents(true);
}

window.fetchMetricsOnly = function () {
    window.fetchWithAuth(`/api/metrics?_ts=${Date.now()}`)
        .then(r => r.json())
        .then(metrics => {
            if (metrics) {
                document.getElementById('kpi-active').innerText = metrics.total_incidents - metrics.resolved_incidents;
                document.getElementById('kpi-resolved').innerText = metrics.resolved_incidents;
                const health = metrics.total_incidents > 0 ? Math.round((metrics.resolved_incidents / metrics.total_incidents) * 100) : 100;
                document.getElementById('kpi-health').innerText = `${health}%`;
            }
        });
};

window.fetchIncidents = function (forceRender = false, opts = {}) {
    if (!opts.skipPauseCheck && !forceRender && Date.now() < window.pausePollingUntil) return;

    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';
    
    // ANTI-CACHE ULTRA AGRESSIF
    const url = `/api/incidents?project=${encodeURIComponent(projectFilter)}&_ts=${Date.now()}&_bust=${Math.random()}`;

    return window.fetchWithAuth(url, {
        headers: {
            'Cache-Control': 'no-cache, no-store, must-revalidate',
            'Pragma': 'no-cache',
            'Expires': '0'
        }
    })
        .then(res => res.json())
        .then(data => {
            if (!opts.skipPauseCheck && !forceRender && Date.now() < window.pausePollingUntil) return;
            if (!data) data = [];

            const rawData = [...data].sort((a, b) => a.id - b.id);
            const retryIds = window.confirmPendingResolvedFromServer(rawData);

            if (retryIds.length > 0 && !window._resolveRetryInFlight) {
                window._resolveRetryInFlight = true;
                window.fetchWithAuth('/api/incidents', {
                    method: 'PUT',
                    body: JSON.stringify({ ids: retryIds })
                }).finally(() => { window._resolveRetryInFlight = false; });
            }

            const safeData = window.mergePendingResolvedState(rawData);
            const currentData = window.currentIncidents || [];
            const safeCurrent = [...currentData].sort((a, b) => a.id - b.id);

            const oldSig = safeCurrent.map(i => `${i.id}:${normalizeIsResolved(i)}`).join('|');
            const newSig = safeData.map(i => `${i.id}:${normalizeIsResolved(i)}`).join('|');

            if (forceRender || oldSig !== newSig) {
                window.currentIncidents = safeData;
                window.renderIncidentsList();
            }
            window.fetchMetricsOnly();
            return safeData;
        })
        .catch(err => {
            const listEl = document.getElementById('incident-list');
            if (listEl && listEl.innerHTML.includes('Loading')) {
                listEl.innerHTML = `<div style="text-align:center; padding: 40px; color: #8b949e;">No incidents found or database empty.</div>`;
            }
        });
};

window.renderIncidentsList = function () {
    const listEl = document.getElementById('incident-list');
    const searchQuery = document.getElementById('incident-search') ? document.getElementById('incident-search').value.toLowerCase() : '';

    const openGroups = new Set();
    document.querySelectorAll('[id^="sub-alerts-"]').forEach(el => {
        if (el.style.display === 'block') openGroups.add(el.id.replace('sub-alerts-', ''));
    });

    let filteredData = window.currentIncidents || [];

    if (window.statusFilter === 'active') {
        filteredData = filteredData.filter(inc => !normalizeIsResolved(inc));
    } else if (window.statusFilter === 'resolved') {
        filteredData = filteredData.filter(inc => normalizeIsResolved(inc));
    }

    if (searchQuery) {
        filteredData = filteredData.filter(inc =>
            (inc.name && inc.name.toLowerCase().includes(searchQuery)) ||
            (inc.project_name && inc.project_name.toLowerCase().includes(searchQuery)) ||
            (inc.error_message && inc.error_message.toLowerCase().includes(searchQuery))
        );
    }

    if (filteredData.length === 0) {
        listEl.innerHTML = '<div style="text-align:center; padding: 40px; opacity: 0.5;">No results match your search.</div>';
        return;
    }

    const activeCount = filteredData.filter(i => !normalizeIsResolved(i)).length;
    const resolvedCount = filteredData.filter(i => normalizeIsResolved(i)).length;
    const health = filteredData.length > 0 ? Math.round((resolvedCount / filteredData.length) * 100) : 100;

    document.getElementById('kpi-active').innerText = activeCount;
    document.getElementById('kpi-active').style.color = activeCount > 0 ? '#ff7b72' : '#3fb950';
    document.getElementById('kpi-resolved').innerText = resolvedCount;
    document.getElementById('kpi-health').innerText = `${health}%`;
    document.getElementById('kpi-health').style.color = health < 80 ? '#d29922' : '#58a6ff';

    const sortedData = [...filteredData].sort((a, b) => a.id - b.id);
    const groupsArray = [];
    let currentGroup = null;

    sortedData.forEach(inc => {
        const safeDateStr = inc.created_at ? inc.created_at.replace(' ', 'T') + 'Z' : '';
        const incTime = new Date(safeDateStr).getTime() || 0;

        if (!currentGroup) {
            currentGroup = {
                id: inc.id,
                project_name: inc.project_name,
                created_at: inc.created_at,
                is_resolved: true,
                incidents: [],
                lastTime: incTime,
                lastId: inc.id
            };
            groupsArray.push(currentGroup);
        } else {
            const timeDiffSec = Math.abs(incTime - currentGroup.lastTime) / 1000;
            const idDiff = Math.abs(inc.id - currentGroup.lastId);

            if (inc.project_name === currentGroup.project_name && timeDiffSec <= 120 && idDiff <= 100) {
                currentGroup.lastTime = incTime;
                currentGroup.lastId = inc.id;
            } else {
                currentGroup = {
                    id: inc.id,
                    project_name: inc.project_name,
                    created_at: inc.created_at,
                    is_resolved: true,
                    incidents: [],
                    lastTime: incTime,
                    lastId: inc.id
                };
                groupsArray.push(currentGroup);
            }
        }

        currentGroup.incidents.push(inc);
        if (!normalizeIsResolved(inc)) currentGroup.is_resolved = false;
    });

    groupsArray.reverse();
    window.groupedIncidents = {};
    groupsArray.forEach(g => { window.groupedIncidents[g.id] = g; });

    const userRole = parseJwt(localStorage.getItem('sre-jwt')).role;
    const canResolve = userRole !== 'viewer';
    const isAdmin = userRole === 'admin';

    const iconCheck = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2.5"><polyline points="20 6 9 17 4 12"></polyline></svg>`;
    const iconAlert = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2.5"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>`;
    const iconWarning = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>`;
    const iconFile = `<svg style="width:12px;height:12px;margin-right:6px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>`;
    const iconCode = `<svg style="width:12px;height:12px;margin-right:6px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>`;
    const iconChevron = `<svg style="width:14px;height:14px;stroke:currentColor;fill:none;transition: transform 0.2s;" viewBox="0 0 24 24" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>`;
    const iconTrash = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>`;
    const iconGear = `<svg style="width:12px;height:12px;margin-right:4px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`;

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
    </div>
    `;

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
            // English Comment: Convert inc.id to String to safely test against Set populated via HTML
            const isChecked = window.selectedIncidents.has(String(inc.id)) ? 'checked' : '';

            const bgStyle = isResolved ? 'background: rgba(63, 185, 80, 0.05); border-left: 3px solid #3fb950;' : 'background: #0d1117; border-left: 3px solid #30363d;';

            const subAlertFlag = isResolved
                ? `<span style="display:inline-flex; align-items:center; background: rgba(63, 185, 80, 0.1); color: #3fb950; padding: 2px 6px; border-radius: 4px; font-size: 9px; font-weight: bold; margin-left: 10px;">${iconCheck} RESOLVED BY ${inc.resolved_by || 'SYSTEM'}</span>`
                : `<span style="display:inline-flex; align-items:center; background: rgba(255, 123, 114, 0.1); color: #ff7b72; padding: 2px 6px; border-radius: 4px; font-size: 9px; font-weight: bold; margin-left: 10px;">${iconAlert} ACTIVE TEST</span>`;

            // English Comment: CRITICAL BUG FIX - Added single quotes around '${inc.id}' to prevent ReferenceErrors 
            // when passing UUID strings to inline JS functions, preventing crashes that caused polling logic to fail.
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
            </div>
        `;

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

            <div id="sub-alerts-${group.id}" style="display: none; padding-top: 10px;">
                ${subAlertsHTML}
            </div>

        </div>`;
    }).join('');

    listEl.innerHTML = htmlContent;
    window.updateBulkActionUI();

    openGroups.forEach(groupId => {
        const el = document.getElementById(`sub-alerts-${groupId}`);
        const icon = document.getElementById(`toggle-icon-${groupId}`);
        if (el) {
            el.style.display = 'block';
            if (icon) icon.style.transform = 'rotate(180deg)';
        }
    });
};

// ==========================================
// AUTO-REFRESH PREVENTION (POLLING CONTROLLER)
// ==========================================

window.connectWebSocket = function () {
    setInterval(() => {
        const dashboard = document.getElementById('view-dashboard');
        if (Date.now() < window.pausePollingUntil) return;

        if (dashboard && dashboard.classList.contains('active')) {
            window.fetchIncidents();
        }
    }, 3000);
}

// ==========================================
// ORGANIZATIONS & TREE VIEW LOGIC
// ==========================================

let currentSelectedOrgId = null;

window.loadOrganizations = function () {
    window.fetchWithAuth(`/api/teams?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            const treeContainer = document.getElementById('organization-tree');
            if (!data || data.length === 0) {
                treeContainer.innerHTML = "<p>No organizations found.</p>";
                return;
            }

            const treeMap = {};
            const roots = [];
            data.forEach(node => { treeMap[node.id] = { ...node, children: [] }; });
            data.forEach(node => {
                if (node.parent_id === null || node.parent_id === 0) {
                    roots.push(treeMap[node.id]);
                } else if (treeMap[node.parent_id]) {
                    treeMap[node.parent_id].children.push(treeMap[node.id]);
                }
            });

            treeContainer.innerHTML = roots.map(root => renderTreeNode(root)).join('');
            if (roots.length > 0) window.selectOrg(roots[0].id, roots[0].name);
        })
        .catch(err => notify("Failed to load directory", "error"));
};

function renderTreeNode(node) {
    const hasChildren = node.children && node.children.length > 0;
    const toggleIcon = hasChildren ? `<span class="tree-toggle" onclick="toggleTree(event, 'tree-children-${node.id}')">▼</span>` : `<span style="width:20px; display:inline-block;"></span>`;

    let html = `
    <div class="tree-node" id="node-container-${node.id}" style="${!node.parent_id ? 'margin-left: 0; border-left: none; padding-left: 0;' : ''}">
        <div style="display:flex; align-items:center;">
            ${toggleIcon}
            <div class="tree-item" id="org-node-${node.id}" onclick="window.selectOrg(${node.id}, '${node.name}')">
                <svg style="width:16px;height:16px;stroke:currentColor;fill:none;flex-shrink:0;" viewBox="0 0 24 24" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
                ${node.name}
            </div>
        </div>`;

    if (hasChildren) {
        html += `<div id="tree-children-${node.id}" style="display:block;">` + node.children.map(child => renderTreeNode(child)).join('') + `</div>`;
    }
    html += `</div>`;
    return html;
}

window.toggleTree = function (e, id) {
    e.stopPropagation();
    const el = document.getElementById(id);
    const icon = e.target;
    if (el.style.display === 'none') {
        el.style.display = 'block';
        icon.innerText = '▼';
    } else {
        el.style.display = 'none';
        icon.innerText = '▶';
    }
}

window.selectOrg = function (id, name) {
    currentSelectedOrgId = id;
    document.querySelectorAll('.tree-item').forEach(el => el.classList.remove('active'));
    const selectedEl = document.getElementById(`org-node-${id}`);
    if (selectedEl) selectedEl.classList.add('active');

    document.getElementById('org-management-panel').style.display = 'block';
    document.getElementById('org-selected-name').innerText = name;
    document.getElementById('org-selected-id').innerText = `Internal DB ID: ${id}`;

    window.loadOrgMembers(id);
};

window.promptRenameGroup = function () {
    if (!currentSelectedOrgId) return;
    const currentName = document.getElementById('org-selected-name').innerText;

    showPromptModal("Rename Group", `Enter a new name for '${currentName}':`, currentName, function (newName) {
        if (newName === currentName || !newName) return;

        window.fetchWithAuth('/api/teams', {
            method: 'PUT',
            body: JSON.stringify({ id: currentSelectedOrgId, name: newName })
        }).then(res => {
            if (res.ok) { notify("Group renamed successfully!", "success"); window.loadOrganizations(); }
            else notify("Failed to rename group. Name might exist.", "error");
        });
    });
};

window.loadOrgMembers = function (orgId) {
    window.fetchWithAuth(`/api/teams/members?team_id=${orgId}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(members => {
            const tbody = document.getElementById('org-members-list');
            if (!members || members.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" style="text-align:center; padding:20px; opacity:0.5;">No members assigned.</td></tr>';
                return;
            }
            tbody.innerHTML = members.map(m => {
                let roleColor = m.team_role === 'team_admin' ? '#d33833' : (m.team_role === 'team_operator' ? '#d29922' : '#58a6ff');
                return `
                <tr style="border-bottom:1px solid var(--border-main);">
                    <td style="padding:12px 10px;"><strong>${m.fullname}</strong><br><small style="opacity:0.6;">${m.username}</small></td>
                    <td><span style="border: 1px solid ${roleColor}; color: ${roleColor}; padding: 3px 8px; border-radius: 12px; font-size: 10px; font-weight: bold; text-transform:uppercase;">${m.team_role.replace('team_', '')}</span></td>
                    <td><span style="opacity:0.5; font-size: 11px; text-transform:uppercase;">${m.global_role}</span></td>
                    <td style="text-align:right;">
                        <button class="btn-secondary" style="font-size:10px; padding:3px 6px; color:#ff7b72; border-color:#ff7b72;" onclick="window.removeUserFromOrg('${m.username}', ${orgId})">REMOVE</button>
                    </td>
                </tr>`;
            }).join('');
        });
};

window.removeUserFromOrg = function (username, teamId) {
    showConfirmModal("Remove User?", `Are you sure you want to remove '${username}' from this group?`, "warning", function () {
        window.fetchWithAuth(`/api/teams/members?username=${username}&team_id=${teamId}`, { method: 'DELETE' })
            .then(res => {
                if (res.ok) { notify("User removed successfully", "success"); window.loadOrgMembers(teamId); }
                else notify("Failed to remove user", "error");
            });
    });
}

window.promptCreateSubGroup = function () {
    if (!currentSelectedOrgId) return notify("Please select a parent group first.", "error");
    showPromptModal("Create Sub-Group", "Enter the name of the new sub-group:", "e.g. Backend Squad", function (name) {
        window.fetchWithAuth('/api/teams', { method: 'POST', body: JSON.stringify({ name: name, parent_id: currentSelectedOrgId }) })
            .then(res => {
                if (res.ok) { notify("Sub-group created!", "success"); window.loadOrganizations(); }
                else notify("Failed to create group. Name might exist.", "error");
            });
    });
};

window.promptDeleteGroup = function () {
    if (!currentSelectedOrgId) return;
    showConfirmModal("Delete Branch?", "WARNING: Deleting this group will ALSO DELETE ALL of its Sub-Groups. Are you absolutely sure?", "danger", function () {
        window.fetchWithAuth(`/api/teams?id=${currentSelectedOrgId}`, { method: 'DELETE' })
            .then(res => {
                if (res.ok) { notify("Branch deleted successfully.", "success"); document.getElementById('org-management-panel').style.display = 'none'; window.loadOrganizations(); }
                else notify("Cannot delete Root Organization or error occurred.", "error");
            });
    });
};

window.handleUserSearch = function () {
    const val = document.getElementById('org-add-user-email').value.toLowerCase();
    const list = document.getElementById('user-autocomplete-list');
    list.innerHTML = '';
    if (!val) { list.style.display = 'none'; return; }

    const matches = allUsers.filter(u => u.username.toLowerCase().includes(val) || (u.fullname && u.fullname.toLowerCase().includes(val)));
    if (matches.length === 0) { list.style.display = 'none'; return; }

    matches.forEach(m => {
        const div = document.createElement('div');
        div.innerHTML = `<strong>${m.fullname}</strong> <span style="opacity:0.6">(${m.username})</span>`;
        div.onclick = () => {
            document.getElementById('org-add-user-email').value = m.username;
            list.style.display = 'none';
        };
        list.appendChild(div);
    });
    list.style.display = 'block';
};

window.assignUserToGroup = function () {
    if (!currentSelectedOrgId) return;
    const username = document.getElementById('org-add-user-email').value;
    const role = document.getElementById('org-add-user-role').value;
    if (!username) return notify("Please enter an email", "error");

    window.fetchWithAuth('/api/teams/members', { method: 'POST', body: JSON.stringify({ username: username.trim(), team_id: currentSelectedOrgId, team_role: role }) })
        .then(res => {
            if (res.ok) { notify("User assigned successfully!", "success"); document.getElementById('org-add-user-email').value = ''; window.loadOrgMembers(currentSelectedOrgId); }
            else if (res.status === 404) notify("User does not exist in the system.", "error");
            else notify("Assignment failed.", "error");
        });
};


// ==========================================
// USER -> TEAMS MODAL LOGIC (IAM TAB)
// ==========================================

let currentlyManagedUser = null;
let allTeamsFlatList = [];

window.openUserTeamsModal = function (username) {
    currentlyManagedUser = username;
    document.getElementById('manage-teams-username').innerText = username;
    document.getElementById('user-add-team-input').value = '';
    document.getElementById('user-teams-modal').style.display = 'flex';

    window.fetchWithAuth(`/api/teams?_ts=${Date.now()}`).then(res => res.json()).then(data => { allTeamsFlatList = data || []; });
    window.refreshUserTeamsModal();
}

window.closeUserTeamsModal = function () {
    document.getElementById('user-teams-modal').style.display = 'none';
}

window.refreshUserTeamsModal = function () {
    window.fetchWithAuth(`/api/users/teams?username=${currentlyManagedUser}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(teams => {
            const tbody = document.getElementById('user-teams-list');
            if (!teams || teams.length === 0) {
                tbody.innerHTML = '<tr><td style="text-align:center; padding:20px; opacity:0.5; color:#8b949e;">User is not assigned to any groups.</td></tr>';
                return;
            }
            tbody.innerHTML = teams.map(t => {
                let roleColor = t.role === 'team_admin' ? '#d33833' : (t.role === 'team_operator' ? '#d29922' : '#58a6ff');
                return `
            <tr style="border-bottom:1px solid #30363d;">
                <td style="padding:10px 15px; font-size:14px; font-weight:bold; color:#c9d1d9;">${t.name}</td>
                <td style="padding:10px 15px;"><span style="border: 1px solid ${roleColor}; color: ${roleColor}; padding: 2px 6px; border-radius: 12px; font-size: 9px; font-weight: bold; text-transform:uppercase;">${t.role.replace('team_', '')}</span></td>
                <td style="padding:10px 15px; text-align:right;">
                    <button class="btn-secondary" style="font-size:10px; padding:4px 8px; color:#ff7b72; border-color:#ff7b72;" onclick="window.removeUserFromTeamModal(${t.id})">REMOVE</button>
                </td>
            </tr>`;
            }).join('');
        });
}

window.removeUserFromTeamModal = function (teamId) {
    window.fetchWithAuth(`/api/teams/members?username=${currentlyManagedUser}&team_id=${teamId}`, { method: 'DELETE' })
        .then(res => {
            if (res.ok) { notify("Removed from group", "success"); window.refreshUserTeamsModal(); }
            else notify("Failed to remove", "error");
        });
}

window.handleTeamSearch = function () {
    const val = document.getElementById('user-add-team-input').value.toLowerCase();
    const list = document.getElementById('team-autocomplete-list');
    list.innerHTML = '';

    if (!val) { list.style.display = 'none'; return; }

    const matches = allTeamsFlatList.filter(t => t.name.toLowerCase().includes(val));
    if (matches.length === 0) { list.style.display = 'none'; return; }

    matches.forEach(m => {
        const div = document.createElement('div');
        div.innerHTML = `<span style="color:#58a6ff; margin-right:5px;">#${m.id}</span> <strong>${m.name}</strong>`;
        div.onclick = () => {
            document.getElementById('user-add-team-input').value = '';
            list.style.display = 'none';
            window.assignUserToTeamFromModal(m.id);
        };
        list.appendChild(div);
    });
    list.style.display = 'block';
}

window.assignUserToTeamFromModal = function (teamId) {
    window.fetchWithAuth('/api/teams/members', { method: 'POST', body: JSON.stringify({ username: currentlyManagedUser, team_id: teamId }) })
        .then(res => {
            if (res.ok) { notify("Added to group!", "success"); window.refreshUserTeamsModal(); }
            else notify("Failed to assign group", "error");
        });
}

// ==========================================
// IAM GLOBAL USERS
// ==========================================

window.loadUsers = function () {
    window.fetchWithAuth(`/api/users?_ts=${Date.now()}`)
        .then(res => { if (!res.ok) throw new Error(); return res.json(); })
        .then(users => {
            allUsers = users || [];
            if (document.getElementById('view-management').classList.contains('active')) {
                window.renderUserTable(allUsers);
            }
        })
        .catch(() => notify("Failed to load users", "error"));
}

window.renderUserTable = function (users) {
    const tbody = document.getElementById('user-list-body');
    if (!tbody) return;

    const currentUser = parseJwt(localStorage.getItem('sre-jwt')).username;
    if (users.length === 0) { tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;">No identities found.</td></tr>'; return; }

    tbody.innerHTML = users.map(u => {
        const isMe = u.username === currentUser;
        const disabledState = isMe ? 'disabled style="opacity:0.4; cursor:not-allowed;" title="You cannot modify yourself"' : '';
        return `
            <tr style="border-bottom:1px solid var(--border-main);">
                <td style="padding:15px 10px; color:${u.is_active ? 'var(--log-pass)' : 'var(--log-fatal)'}; font-weight:bold;">${u.is_active ? '● ACTIVE' : '○ DISABLED'}</td>
                <td><strong>${u.fullname || 'N/A'}</strong> ${isMe ? '<small>(YOU)</small>' : ''}<br><small style="opacity:0.6;">${u.username}</small></td>
                <td><small style="text-transform:uppercase;">${u.role}</small></td>
                <td style="text-align:right;">
                    <button class="btn-secondary" style="font-size:10px; border-color:#58a6ff; color:#58a6ff;" onclick="window.openUserTeamsModal('${u.username}')">TEAMS</button>
                    <button class="btn-secondary" style="font-size:10px;" onclick="window.adminResetPassword('${u.username}')" ${disabledState}>RESET PWD</button>
                    <button class="btn-secondary" style="font-size:10px; color:${u.is_active ? 'var(--log-fatal)' : 'var(--log-pass)'}" onclick="window.toggleUserStatus('${u.username}', ${u.is_active})" ${disabledState}>${u.is_active ? 'Disable' : 'Enable'}</button>
                    <button class="btn-primary" style="color:var(--log-fatal); border-color:var(--log-fatal); font-size:10px;" onclick="window.deleteUser('${u.username}')" ${disabledState}>DEL</button>
                </td></tr>`;
    }).join('');
}

window.filterUsers = function () {
    const q = document.getElementById('user-search').value.toLowerCase();
    const r = document.getElementById('role-filter').value;
    window.renderUserTable(allUsers.filter(u => (u.username.toLowerCase().includes(q) || (u.fullname && u.fullname.toLowerCase().includes(q))) && (r === 'all' || u.role === r)));
}

window.createUser = function () {
    const name = document.getElementById('new-user-fullname').value;
    const email = document.getElementById('new-user-name').value;
    if (!name || !email) return notify("Name and Email are required", "error");

    window.fetchWithAuth('/api/users', { method: 'POST', body: JSON.stringify({ username: email, fullname: name, role: document.getElementById('new-user-role').value }) })
        .then(res => {
            if (res.ok) {
                notify("Identity deployed!", "success");
                document.getElementById('new-user-fullname').value = '';
                document.getElementById('new-user-name').value = '';
                window.loadUsers();
            } else { notify("Failed to provision identity.", "error"); }
        });
}

window.toggleUserStatus = function (username, current) {
    window.fetchWithAuth('/api/users/status', { method: 'POST', body: JSON.stringify({ username, is_active: !current }) })
        .then(res => {
            if (res.ok) { notify("Status updated", "success"); window.loadUsers(); }
            else notify("Update failed", "error");
        });
}

window.adminResetPassword = function (username) {
    showConfirmModal("Reset Password?", `Are you sure you want to force a password reset for '${username}'?`, "warning", function () {
        window.fetchWithAuth('/api/users/reset-password', { method: 'POST', body: JSON.stringify({ username }) })
            .then(res => {
                if (res.ok) notify("New password emailed", "success");
                else notify("Reset failed", "error");
            });
    });
}

window.deleteUser = function (username) {
    showConfirmModal("Delete Identity?", `Are you absolutely sure you want to completely remove the user '${username}'? This action cannot be undone.`, "danger", function () {
        window.fetchWithAuth(`/api/users/delete?username=${username}`, { method: 'DELETE' }).then(res => {
            if (res.ok) { notify("Identity deleted", "success"); window.loadUsers(); }
            else notify("Deletion failed", "error");
        });
    });
}

// ==========================================
// PLUGINS & CONFIG 
// ==========================================

window.loadPlugins = function () {
    window.fetchWithAuth(`/api/plugins?_ts=${Date.now()}`).then(res => res.json()).then(data => {
        if (!data || data.length === 0) { document.getElementById('plugin-list').innerHTML = "<div style='text-align:center; padding:40px; opacity:0.5;'>No modules detected in the plugins directory.</div>"; return; }
        const configIcon = `<svg style="width:14px;height:14px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1 0-2.83 2 2 0 0 1 0-2.83l.06.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`;
        const runIcon = `<svg style="width:14px;height:14px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>`;
        const saveIcon = `<svg style="width:14px;height:14px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>`;
        const consoleIcon = `<svg style="width:12px;height:12px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line></svg>`;
        const pluginIconSVG = `<svg style="width:24px;height:24px;stroke:#8b949e;fill:none;" viewBox="0 0 24 24" stroke-width="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>`;

        const groupedPlugins = {};
        data.forEach(p => { const parts = p.file_path.split(/[/\\]/); const folderName = parts.length > 1 ? parts[0] : 'general'; if (!groupedPlugins[folderName]) groupedPlugins[folderName] = []; groupedPlugins[folderName].push(p); });

        let html = '';
        let globalIdx = 0;

        for (const [folder, plugins] of Object.entries(groupedPlugins)) {
            html += `<h3 style="margin: 30px 0 15px 0; padding-bottom: 8px; border-bottom: 1px solid var(--border-main); color: var(--text-main); font-size: 14px; letter-spacing: 1px; text-transform: uppercase; opacity: 0.8;">Category: ${folder}</h3>`;
            plugins.forEach(p => {
                const safePath = p.file_path.replace(/\\/g, '/');
                let envs = '';
                for (let k in (p.env || {})) { envs += `<div style="margin-bottom: 12px;"><label style="font-size:11px; color:#8b949e; display:block; margin-bottom:5px; text-transform:uppercase; font-weight:bold; letter-spacing:0.5px;">${k}</label><input type="text" id="env-${globalIdx}-${k}" class="login-input" style="padding:10px; margin:0; width:100%; box-sizing:border-box; background:#0d1117;" value="${p.env[k]}"></div>`; }
                let desc = p.description || "No description provided.";
                const logoUrl = `/assets/${folder.toLowerCase()}.png`;

                html += `
                <div class="data-card" style="border-left: 4px solid #58a6ff; margin-bottom: 20px; padding: 20px; transition: all 0.2s ease;">
                    <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap: wrap; gap: 20px;">
                        <div style="display:flex; align-items:center; gap: 15px; flex-grow: 1;">
                            <div style="position:relative; width:50px; height:50px; flex-shrink:0;">
                                <div style="position:absolute; top:0; left:0; width:100%; height:100%; border-radius:10px; background:#21262d; border:1px solid #30363d; display:flex; align-items:center; justify-content:center; z-index:1;">${pluginIconSVG}</div>
                                <img src="${logoUrl}" onerror="this.style.display='none'" style="position:absolute; top:0; left:0; width:100%; height:100%; object-fit:contain; border-radius:10px; background:#0d1117; padding:5px; border:1px solid #30363d; box-sizing:border-box; z-index:2;" />
                            </div>
                            <div>
                                <h3 style="margin:0 0 5px 0; color:var(--text-main); font-size: 16px; display:flex; align-items:center; gap:10px;">
                                    ${p.name} <span style="font-size:10px; background:#238636; color:white; padding:3px 8px; border-radius:12px;">v${p.version || '1.0'}</span>
                                    ${p.status === 'Active' ? '<span style="font-size:10px; color:#3fb950; border:1px solid #3fb950; padding:2px 6px; border-radius:4px; letter-spacing: 0.5px;">AUTO-RUN ON</span>' : ''}
                                </h3>
                                <p style="font-size:13px; color:#8b949e; margin:0; line-height: 1.4;">${desc}</p>
                            </div>
                        </div>
                        <div style="display:flex; gap:12px; flex-shrink: 0;">
                            <button class="btn-secondary" style="display:flex; align-items:center; padding: 8px 16px;" onclick="window.togglePluginConfig('config-${globalIdx}')">${configIcon} Configure</button>
                            <button class="btn-primary" id="btn-run-${globalIdx}" onclick="window.runPlugin('${safePath}', ${globalIdx})" style="background-color:#58a6ff; border-color:#58a6ff; color: #0d1117; font-weight: bold; padding: 8px 16px; display:flex; align-items:center; justify-content:center; min-width: 120px;">${runIcon} Execute</button>
                        </div>
                    </div>
                    <div id="config-${globalIdx}" style="display:none; margin-top:20px; background:var(--bg-main); padding:20px; border-radius:8px; border:1px solid var(--border-main);">
                        <h4 style="margin:0 0 15px 0; font-size:13px; text-transform:uppercase; color:#58a6ff; letter-spacing:1px;">Environment Variables</h4>
                        ${envs || '<p style="font-size:13px; color:#8b949e;">No external variables required.</p>'}
                        ${envs ? `<button class="btn-primary" style="font-size:12px; width:100%; margin-top:15px; padding:10px; display:flex; align-items:center; justify-content:center;" onclick="window.savePluginConfig('${safePath}', ${globalIdx}, '${Object.keys(p.env || {}).join(',')}')">${saveIcon} Save</button>` : ''}
                    </div>
                    <div id="logs-container-${globalIdx}" style="display:none; margin-top:20px;">
                        <div style="font-size:12px; color:#8b949e; margin-bottom:8px; text-transform:uppercase; font-weight:bold; display:flex; align-items:center;">${consoleIcon} STDOUT Console</div>
                        <pre id="logs-${globalIdx}" style="background:#0d1117; color:#00ff00; padding:15px; border-radius:8px; border:1px solid #30363d; font-family:monospace; font-size:13px; overflow-x:auto; white-space:pre-wrap; margin:0; max-height:400px; overflow-y:auto;"></pre>
                    </div>
                </div>`;
                globalIdx++;
            });
        }
        document.getElementById('plugin-list').innerHTML = html;
    }).catch(() => notify("Network error", "error"));
}

window.togglePluginConfig = function (id) { const el = document.getElementById(id); el.style.display = el.style.display === 'none' ? 'block' : 'none'; }

window.savePluginConfig = function (path, idx, keysStr) {
    const env = {};
    keysStr.split(',').filter(k => k).forEach(k => env[k] = document.getElementById(`env-${idx}-${k}`).value);
    window.fetchWithAuth('/api/plugins/config', { method: 'POST', body: JSON.stringify({ file_path: path, env: env }) })
        .then(res => { if (res.ok) notify("Configuration saved", "success"); else notify("Failed to save", "error"); })
}

window.runPlugin = function (path, idx) {
    const btn = document.getElementById(`btn-run-${idx}`);
    const logsContainer = document.getElementById(`logs-container-${idx}`);
    const logsEl = document.getElementById(`logs-${idx}`);

    btn.innerHTML = `Running...`;
    btn.disabled = true; btn.style.opacity = '0.7';
    logsContainer.style.display = 'block'; logsEl.style.color = '#8b949e'; logsEl.innerHTML = 'Init...';

    window.fetchWithAuth('/api/plugins/run', { method: 'POST', body: JSON.stringify({ file_path: path }) }).then(async (res) => {
        const data = await res.json();
        btn.innerHTML = `Execute`; btn.disabled = false; btn.style.opacity = '1';
        if (res.ok) { notify("Success", "success"); logsEl.style.color = '#00ff00'; }
        else { notify("Error", "error"); logsEl.style.color = '#ff7b72'; }
        logsEl.innerText = data.logs || "No output.";
    });
}

window.loadConfig = function () {
    window.fetchWithAuth(`/api/config?_ts=${Date.now()}`).then(res => res.json()).then(c => {
        document.getElementById('policy-domain').value = c.security.allowed_domain || "";
        if (c.smtp) {
            document.getElementById('smtp-host').value = c.smtp.host || "";
            document.getElementById('smtp-port').value = c.smtp.port || "";
            document.getElementById('smtp-user').value = c.smtp.user || "";
            document.getElementById('smtp-pass').value = c.smtp.password || "";
            document.getElementById('smtp-from').value = c.smtp.from || "";
        }
        document.getElementById('config-container').innerHTML = `<pre style="font-size:11px; opacity:0.5;">${JSON.stringify(c, null, 2)}</pre>`;
    });
}

window.saveSecurityPolicy = function () { window.fetchWithAuth('/api/config/policy', { method: 'POST', body: JSON.stringify({ allowed_domain: document.getElementById('policy-domain').value }) }).then(res => { if (res.ok) notify("Policy updated", "success"); }); }
window.saveSMTPConfig = function () {
    const payload = { host: document.getElementById('smtp-host').value, port: parseInt(document.getElementById('smtp-port').value) || 0, user: document.getElementById('smtp-user').value, password: document.getElementById('smtp-pass').value, from: document.getElementById('smtp-from').value };
    window.fetchWithAuth('/api/config/smtp', { method: 'POST', body: JSON.stringify(payload) }).then(res => { if (res.ok) notify("SMTP saved", "success"); });
}

// ==========================================
// FINOPS CONTROLLER
// ==========================================

window.loadFinOps = async function () {
    try {
        const res = await window.fetchWithAuth(`/api/finops?_ts=${Date.now()}`);
        if (res.ok) {
            const data = await res.json();
            const devRateInput = document.getElementById('finops-dev-rate');
            if (devRateInput) devRateInput.value = data.dev_hourly_rate;

            const ciCostInput = document.getElementById('finops-ci-cost');
            if (ciCostInput) ciCostInput.value = data.ci_minute_cost;

            const durationInput = document.getElementById('finops-duration');
            if (durationInput) durationInput.value = data.avg_pipeline_duration;

            const invTimeInput = document.getElementById('finops-investigation');
            if (invTimeInput) invTimeInput.value = data.avg_investigation_time;

            // Load currency preference
            const currencySelector = document.getElementById('finops-currency');
            if (currencySelector && data.currency) {
                currencySelector.value = data.currency;
                window.selectedCurrency = data.currency;
            }
        }
    } catch (e) {
        console.error("Could not load FinOps settings");
    }
}

window.saveCurrencyPreference = function () {
    const currency = document.getElementById('finops-currency').value;
    window.selectedCurrency = currency;
    localStorage.setItem('selected-currency', currency);
    notify(`Currency changed to ${currency}`, 'success');
}

window.saveFinOps = async function () {
    const payload = {
        dev_hourly_rate: parseFloat(document.getElementById('finops-dev-rate').value) || 50,
        ci_minute_cost: parseFloat(document.getElementById('finops-ci-cost').value) || 0.008,
        avg_pipeline_duration: parseFloat(document.getElementById('finops-duration').value) || 15,
        avg_investigation_time: parseFloat(document.getElementById('finops-investigation').value) || 30,
        currency: document.getElementById('finops-currency').value || 'USD'
    };

    try {
        const res = await window.fetchWithAuth('/api/finops', {
            method: 'PUT',
            body: JSON.stringify(payload)
        });

        if (res.ok) {
            notify('FinOps settings updated. Dashboard metrics will recalculate.', 'success');
            if (document.getElementById('view-dashboard').classList.contains('active')) {
                window.fetchIncidents(true);
            }
        } else {
            notify('Failed to update FinOps settings', 'error');
        }
    } catch (e) {
        notify('Network error saving FinOps', 'error');
    }
}

window.updateCurrencyDisplay = function () {
    const currencySymbol = window.currencySymbols[window.selectedCurrency] || '$';
    const symbolElement = document.getElementById('kpi-cost-symbol');
    if (symbolElement) {
        symbolElement.textContent = currencySymbol;
    }
}

// ==========================================
// ENTERPRISE LICENSE & SSO CONTROLLER
// ==========================================

window.checkSSOStatus = function () {
    fetch('/api/sso/status')
        .then(res => res.json())
        .then(data => {
            const ssoContainer = document.getElementById('sso-container');
            const ssoLocked = document.getElementById('sso-locked');

            if (ssoContainer && ssoLocked) {
                if (data.enterprise_active) {
                    ssoContainer.style.display = 'block';
                    ssoLocked.style.display = 'none';
                } else {
                    ssoContainer.style.display = 'none';
                    ssoLocked.style.display = 'block';
                }
            }
        }).catch(err => console.log("Enterprise check failed"));
};

window.triggerSSO = function () {
    fetch('/api/sso/login')
        .then(res => {
            if (res.status === 402) {
                notify("QA Capsule PRO License is missing or expired.", "error");
                return null;
            }
            return res.json();
        })
        .then(data => {
            if (data && data.sso_url) {
                notify(data.message, "success");
                setTimeout(() => { alert("Redirecting to: " + data.sso_url); }, 1500);
            }
        }).catch(err => notify("SSO failed. Please try again.", "error"));
};

// ==========================================
// INITIALIZATION
// ==========================================

window.onload = function () {
    window.checkAuth();
    window.checkSSOStatus();
    // Initialize currency from localStorage
    const savedCurrency = localStorage.getItem('selected-currency');
    if (savedCurrency) {
        window.selectedCurrency = savedCurrency;
    }
    window.updateCurrencyDisplay();
    // Initialize status filter button styles
    setTimeout(() => window.setStatusFilter(window.statusFilter || 'all'), 100);
};