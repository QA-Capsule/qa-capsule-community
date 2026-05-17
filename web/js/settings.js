/**
 * web/js/settings.js
 * Technical settings, FinOps, Plugins, Analytics and Reports
 */
import { fetchWithAuth, parseJwt } from './api.js';
import { notify, showConfirmModal, showPromptModal } from './ui.js';

export let allProjects = [];
let editingProjectId = null;
let currentSelectedCI = 'gitlab';
export let selectedCurrency = 'USD';
let analyticsChart = null;

export const currencySymbols = {
    'USD': '$', 'EUR': '€', 'GBP': '£', 'JPY': '¥', 'AUD': 'A$',
    'CAD': 'C$', 'CHF': 'Fr', 'INR': '₹', 'CNY': '¥', 'MXN': '$',
    'SGD': 'S$', 'NZD': 'NZ$'
};

export function setSelectedCurrency(val) { selectedCurrency = val; }

export function selectCI(ciName, element) {
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

export function loadGatewaysData() {
    fetchWithAuth(`/api/my-projects?_ts=${Date.now()}`)
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

    fetchWithAuth(`/api/teams?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(teams => {
            const select = document.getElementById('ci-team-id');
            select.innerHTML = '<option value="" disabled selected>Select a Team</option>';
            if (teams) teams.forEach(t => select.innerHTML += `<option value="${t.id}">${t.name}</option>`);
        });
}

export function editProject(projectId) {
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
    if (ciCard) selectCI(p.ci_system, ciCard);

    const apiKeyInput = document.getElementById('ci-api-key');
    if (apiKeyInput) {
        apiKeyInput.value = p.api_key || "";
        apiKeyInput.type = 'password';
    }

    const btn = document.querySelector("button[onclick='window.saveCIConfig()']");
    if (btn) {
        btn.innerText = "Update Project Endpoint";
        btn.style.borderColor = "#58a6ff";
        btn.style.color = "#58a6ff";
    }

    document.getElementById('ci-project-name').focus();
    notify("Editing project: " + p.name, "success");
}

export function deleteProject(projectId) {
    showConfirmModal("Delete Pipeline Endpoint?", "Are you sure? This will permanently delete this pipeline endpoint and all its routing configurations.", "danger", function () {
        fetchWithAuth(`/api/config/projects?id=${projectId}`, { method: 'DELETE' })
            .then(res => {
                if (res.ok) { notify("Pipeline deleted successfully.", "success"); loadGatewaysData(); window.loadDashboardFilters(); }
                else notify("Failed to delete pipeline.", "error");
            });
    });
}

export function saveCIConfig() {
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

    fetchWithAuth('/api/config/projects', { method: method, body: JSON.stringify(payload) })
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

                const btn = document.querySelector("button[onclick='window.saveCIConfig()']");
                if (btn) {
                    btn.innerText = "Provision Project Endpoint";
                    btn.style.borderColor = "";
                    btn.style.color = "";
                }

                loadGatewaysData();
                window.loadDashboardFilters();
            } else {
                notify("Operation failed", "error");
            }
        }).catch(() => notify("Network error", "error"));
}

export function revealApiKey() {
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
}

export function toggleAnalytics() {
    const view = document.getElementById('analytics-view');
    if (view.style.display === 'none') {
        view.style.display = 'block';
        loadAnalytics();
    } else {
        view.style.display = 'none';
    }
}

export function loadAnalytics() {
    fetchWithAuth(`/api/metrics?_ts=${Date.now()}`)
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
}

export function downloadWeeklyReportCSV() {
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';

    fetchWithAuth(`/api/reports/weekly?project=${encodeURIComponent(projectFilter)}&_ts=${Date.now()}`)
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
}

export function downloadWeeklyReportPDF() {
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';

    fetchWithAuth(`/api/reports/weekly?project=${encodeURIComponent(projectFilter)}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            if (!data || data.length === 0) {
                return notify("Aucun incident enregistré ces 7 derniers jours.", "warning");
            }

            const { jsPDF } = window.jspdf;
            const doc = new jsPDF();

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
}

export function loadPlugins() {
    fetchWithAuth(`/api/plugins?_ts=${Date.now()}`).then(res => res.json()).then(data => {
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

export function togglePluginConfig(id) { const el = document.getElementById(id); el.style.display = el.style.display === 'none' ? 'block' : 'none'; }

export function savePluginConfig(path, idx, keysStr) {
    const env = {};
    keysStr.split(',').filter(k => k).forEach(k => env[k] = document.getElementById(`env-${idx}-${k}`).value);
    fetchWithAuth('/api/plugins/config', { method: 'POST', body: JSON.stringify({ file_path: path, env: env }) })
        .then(res => { if (res.ok) notify("Configuration saved", "success"); else notify("Failed to save", "error"); })
}

export function runPlugin(path, idx) {
    const btn = document.getElementById(`btn-run-${idx}`);
    const logsContainer = document.getElementById(`logs-container-${idx}`);
    const logsEl = document.getElementById(`logs-${idx}`);

    btn.innerHTML = `Running...`;
    btn.disabled = true; btn.style.opacity = '0.7';
    logsContainer.style.display = 'block'; logsEl.style.color = '#8b949e'; logsEl.innerHTML = 'Init...';

    fetchWithAuth('/api/plugins/run', { method: 'POST', body: JSON.stringify({ file_path: path }) }).then(async (res) => {
        const data = await res.json();
        btn.innerHTML = `Execute`; btn.disabled = false; btn.style.opacity = '1';
        if (res.ok) { notify("Success", "success"); logsEl.style.color = '#00ff00'; }
        else { notify("Error", "error"); logsEl.style.color = '#ff7b72'; }
        logsEl.innerText = data.logs || "No output.";
    });
}

export function loadConfig() {
    fetchWithAuth(`/api/config?_ts=${Date.now()}`).then(res => res.json()).then(c => {
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

export function saveSecurityPolicy() { fetchWithAuth('/api/config/policy', { method: 'POST', body: JSON.stringify({ allowed_domain: document.getElementById('policy-domain').value }) }).then(res => { if (res.ok) notify("Policy updated", "success"); }); }

export function saveSMTPConfig() {
    const payload = { host: document.getElementById('smtp-host').value, port: parseInt(document.getElementById('smtp-port').value) || 0, user: document.getElementById('smtp-user').value, password: document.getElementById('smtp-pass').value, from: document.getElementById('smtp-from').value };
    fetchWithAuth('/api/config/smtp', { method: 'POST', body: JSON.stringify(payload) }).then(res => { if (res.ok) notify("SMTP saved", "success"); });
}

export function loadFinOps() {
    fetchWithAuth(`/api/finops?_ts=${Date.now()}`)
        .then(res => { if (res.ok) return res.json(); throw new Error(); })
        .then(data => {
            const devRateInput = document.getElementById('finops-dev-rate');
            if (devRateInput) devRateInput.value = data.dev_hourly_rate;

            const ciCostInput = document.getElementById('finops-ci-cost');
            if (ciCostInput) ciCostInput.value = data.ci_minute_cost;

            const durationInput = document.getElementById('finops-duration');
            if (durationInput) durationInput.value = data.avg_pipeline_duration;

            const invTimeInput = document.getElementById('finops-investigation');
            if (invTimeInput) invTimeInput.value = data.avg_investigation_time;

            const currencySelector = document.getElementById('finops-currency');
            if (currencySelector && data.currency) {
                currencySelector.value = data.currency;
                selectedCurrency = data.currency;
            }
        })
        .catch(e => console.error("Could not load FinOps settings"));
}

export function saveCurrencyPreference() {
    const currency = document.getElementById('finops-currency').value;
    selectedCurrency = currency;
    localStorage.setItem('selected-currency', currency);
    notify(`Currency changed to ${currency}`, 'success');
}

export function saveFinOps() {
    const payload = {
        dev_hourly_rate: parseFloat(document.getElementById('finops-dev-rate').value) || 50,
        ci_minute_cost: parseFloat(document.getElementById('finops-ci-cost').value) || 0.008,
        avg_pipeline_duration: parseFloat(document.getElementById('finops-duration').value) || 15,
        avg_investigation_time: parseFloat(document.getElementById('finops-investigation').value) || 30,
        currency: document.getElementById('finops-currency').value || 'USD'
    };

    fetchWithAuth('/api/finops', { method: 'PUT', body: JSON.stringify(payload) })
        .then(res => {
            if (res.ok) {
                notify('FinOps settings updated. Dashboard metrics will recalculate.', 'success');
                if (document.getElementById('view-dashboard').classList.contains('active')) {
                    window.fetchIncidents(true);
                }
            } else { notify('Failed to update FinOps settings', 'error'); }
        }).catch(e => notify('Network error saving FinOps', 'error'));
}

export function updateCurrencyDisplay() {
    const currencySymbol = currencySymbols[selectedCurrency] || '$';
    const symbolElement = document.getElementById('kpi-cost-symbol');
    if (symbolElement) {
        symbolElement.textContent = currencySymbol;
    }
}

export function checkSSOStatus() {
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
}

export function triggerSSO() {
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
}