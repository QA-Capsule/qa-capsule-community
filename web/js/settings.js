/**
 * web/js/settings.js
 * Technical settings, FinOps, Plugins, Analytics and Reports
 */
import { fetchWithAuth, parseJwt, parseApiJson, asArray } from './api.js';
import { notify, showConfirmModal, showPromptModal } from './ui.js';
import * as analyticsLayout from './analytics-layout.js';

export let allProjects = [];
let editingProjectId = null;
let currentSelectedCI = 'gitlab';
export let selectedCurrency = 'USD';

export let analyticsChart = null;
export let evolutionChart = null;

export const currencySymbols = {
    'USD': '$', 'EUR': '€', 'GBP': '£', 'JPY': '¥', 'AUD': 'A$',
    'CAD': 'C$', 'CHF': 'Fr', 'INR': '₹', 'CNY': '¥', 'MXN': '$',
    'SGD': 'S$', 'NZD': 'NZ$'
};

// --- GLOBAL CHART.JS DESIGN DEFAULTS ---
Chart.defaults.font.family = "'DM Sans', system-ui, -apple-system, sans-serif";

function getChartTheme() {
    const dark = document.body.getAttribute('data-theme') === 'dark';
    return {
        legend: dark ? '#c9d1d9' : '#334155',
        title: dark ? '#94a3b8' : '#475569',
        tick: dark ? '#8b949e' : '#64748b',
        border: dark ? '#111827' : '#ffffff',
        grid: dark ? 'rgba(48, 54, 61, 0.45)' : 'rgba(203, 213, 225, 0.9)',
    };
}

function applyChartThemeDefaults() {
    const t = getChartTheme();
    Chart.defaults.color = t.tick;
}

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
    const tbody = document.getElementById('projects-table-body');
    fetchWithAuth(`/api/my-projects?_ts=${Date.now()}`)
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            allProjects = ok ? asArray(data) : [];
            if (!tbody) return;
            if (!ok) {
                tbody.innerHTML = '<tr><td colspan="4"><span class="load-error-msg">Unable to load gateways.</span></td></tr>';
                return;
            }
            if (allProjects.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" style="text-align:center; padding:20px; opacity:0.5;">No pipelines provisioned yet.</td></tr>';
                return;
            }
            tbody.innerHTML = allProjects.map(p => `
                <tr style="border-bottom: 1px solid var(--border-main);">
                    <td style="padding: 10px;"><strong>${p.name}</strong></td>
                    <td style="padding: 10px; text-transform: uppercase;">${p.ci_system}</td>
                    <td style="padding: 10px;"><code style="color: #ff7b72; font-family: monospace;">••••••••••••</code></td>
                    <td style="padding: 10px; text-align: right;">
                        <button type="button" class="btn btn-secondary btn-sm btn-info" onclick="window.editProject('${p.id}')">EDIT</button>
                        <button type="button" class="btn btn-secondary btn-sm btn-danger" onclick="window.deleteProject('${p.id}')">DELETE</button>
                    </td>
                </tr>
            `).join('');
        })
        .catch(() => {
            if (tbody) tbody.innerHTML = '<tr><td colspan="4"><span class="load-error-msg">Unable to load gateways.</span></td></tr>';
        });

    fetchWithAuth(`/api/teams?_ts=${Date.now()}`)
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            const select = document.getElementById('ci-team-id');
            if (!select || !ok) return;
            select.innerHTML = '<option value="" disabled selected>Select a Team</option>';
            asArray(data).forEach(t => select.innerHTML += `<option value="${t.id}">${t.name}</option>`);
        })
        .catch(() => {});
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

    const btn = document.getElementById('save-ci-btn');
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
        .then(res => parseApiJson(res))
        .then(({ ok, offline, status, data }) => {
            if (!ok) {
                const msg = offline ? 'Server unreachable.' : (data?.error || `Operation failed (${status || 'error'})`);
                notify(msg, 'error');
                return;
            }
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
            const btn = document.getElementById('save-ci-btn');
            if (btn) {
                btn.innerText = "Provision Project Endpoint";
                btn.style.borderColor = "";
                btn.style.color = "";
            }
            loadGatewaysData();
            if (window.loadDashboardFilters) window.loadDashboardFilters();
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
        analyticsLayout.loadAnalyticsLayoutFromPrefs().then(() => loadAnalytics(false));
    } else {
        view.style.display = 'none';
    }
}

// Re-export for inline handlers
if (typeof window !== 'undefined') {
    window.reloadDashboardAnalytics = window.reloadDashboardAnalytics || function () {
        const view = document.getElementById('analytics-view');
        if (view && view.style.display !== 'none') loadAnalytics(false);
    };
}

export function loadAnalytics(isExport = false) {
    applyChartThemeDefaults();
    const theme = getChartTheme();
    const rangeQ = typeof window.getDashboardRangeQuery === 'function' ? window.getDashboardRangeQuery() : 'range=7d';
    if (rangeQ === null) {
        notify('Select both From and To dates before loading analytics.', 'error');
        return Promise.resolve();
    }
    return fetchWithAuth(`/api/metrics?${rangeQ}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            window.__lastMetricsData = data;
            analyticsLayout.renderAnalyticsGrid(data, { isExport, theme });
            return data;
        })
        .catch(err => {
            console.error('Error loading analytics:', err);
            return null;
        });
}

function hexToRgb(hex) {
    const h = (hex || '#3b82f6').replace('#', '');
    if (h.length !== 6) return [59, 130, 246];
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
}

async function loadImageAsDataUrl(url) {
    const res = await fetch(url);
    if (!res.ok) throw new Error('logo');
    const blob = await res.blob();
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = reject;
        reader.readAsDataURL(blob);
    });
}

export function downloadWeeklyReportCSV() {
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';
    const rangeQ = typeof window.getDashboardRangeQuery === 'function' ? window.getDashboardRangeQuery() : 'range=7d';
    if (rangeQ === null) return notify('Select a valid time range first.', 'error');

    fetchWithAuth(`/api/reports/weekly?project=${encodeURIComponent(projectFilter)}&${rangeQ}&_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            if (!data || data.length === 0) {
                return notify('No incidents in the selected time range.', 'warning');
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

            notify("CSV report generated successfully.", "success");
        })
        .catch(() => notify("Failed to generate CSV report", "error"));
}

const PDF_PAGE = { w: 210, h: 297, margin: 14 };

function pdfEnsureSpace(doc, y, need, margin = PDF_PAGE.margin) {
    if (y + need > PDF_PAGE.h - margin) {
        doc.addPage();
        doc.setFillColor(248, 250, 252);
        doc.rect(0, 0, PDF_PAGE.w, PDF_PAGE.h, 'F');
        return margin + 4;
    }
    return y;
}

function pdfDrawHeader(doc, { logoData, titleScope, rangeLabel }) {
    const m = PDF_PAGE.margin;
    const headerH = 34;
    doc.setFillColor(248, 250, 252);
    doc.rect(0, 0, PDF_PAGE.w, headerH, 'F');
    doc.setFillColor(37, 99, 235);
    doc.rect(0, 0, 3, headerH, 'F');
    doc.setDrawColor(226, 232, 240);
    doc.setLineWidth(0.3);
    doc.line(0, headerH, PDF_PAGE.w, headerH);

    const textX = logoData ? m + 28 : m;
    if (logoData) doc.addImage(logoData, 'PNG', m, 7, 22, 22);

    doc.setTextColor(15, 23, 42);
    doc.setFontSize(16);
    doc.setFont('helvetica', 'bold');
    doc.text('QA Capsule — Flight Recorder', textX, 14);

    doc.setFontSize(9);
    doc.setFont('helvetica', 'normal');
    doc.setTextColor(71, 85, 105);
    doc.text(`Executive SRE Observability Report · ${titleScope}`, textX, 20);

    doc.setFontSize(8);
    doc.setTextColor(100, 116, 139);
    doc.text(`Period: ${rangeLabel}`, textX, 25);
    doc.text(`Generated ${new Date().toLocaleString()}`, textX, 29);

    return headerH + 8;
}

function pdfDrawKpiRow(doc, startY, metricWidgets, metricsData) {
    if (!metricWidgets.length) return startY;
    const m = PDF_PAGE.margin;
    const contentW = PDF_PAGE.w - m * 2;
    const cols = Math.min(4, metricWidgets.length);
    const gap = 5;
    const cardW = (contentW - gap * (cols - 1)) / cols;
    const cardH = 26;
    let y = startY;

    metricWidgets.forEach((w, index) => {
        const col = index % cols;
        const row = Math.floor(index / cols);
        const x = m + col * (cardW + gap);
        const cy = y + row * (cardH + gap);
        const rgb = hexToRgb(w.color);
        const label = w.title || analyticsLayout.METRIC_CATALOG[w.metric]?.label || w.metric || 'Metric';
        const val = analyticsLayout.getMetricDisplayValue(metricsData, w.metric);

        doc.setFillColor(255, 255, 255);
        doc.setDrawColor(226, 232, 240);
        doc.setLineWidth(0.25);
        if (typeof doc.roundedRect === 'function') {
            doc.roundedRect(x, cy, cardW, cardH, 2, 2, 'FD');
        } else {
            doc.rect(x, cy, cardW, cardH, 'FD');
        }
        doc.setFillColor(rgb[0], rgb[1], rgb[2]);
        doc.rect(x, cy, cardW, 1.2, 'F');

        doc.setFontSize(7);
        doc.setTextColor(100, 116, 139);
        doc.setFont('helvetica', 'bold');
        const labelLines = doc.splitTextToSize(label.toUpperCase(), cardW - 8);
        doc.text(labelLines, x + 4, cy + 8);

        doc.setFontSize(16);
        doc.setTextColor(rgb[0], rgb[1], rgb[2]);
        doc.setFont('helvetica', 'bold');
        doc.text(String(val), x + 4, cy + 19);
    });

    const rows = Math.ceil(metricWidgets.length / cols);
    return y + rows * (cardH + gap) + 6;
}

function pdfDrawSectionTitle(doc, y, title) {
    const m = PDF_PAGE.margin;
    doc.setFontSize(11);
    doc.setFont('helvetica', 'bold');
    doc.setTextColor(30, 41, 59);
    doc.text(title, m, y);
    doc.setDrawColor(226, 232, 240);
    doc.setLineWidth(0.2);
    doc.line(m, y + 2, PDF_PAGE.w - m, y + 2);
    return y + 8;
}

function pdfDrawChartBlock(doc, y, title, img, chartType) {
    const m = PDF_PAGE.margin;
    const contentW = PDF_PAGE.w - m * 2;
    const blockH = chartType === 'evolution' ? 78 : 58;
    y = pdfEnsureSpace(doc, y, blockH + 14, m);

    y = pdfDrawSectionTitle(doc, y, title);

    doc.setFillColor(255, 255, 255);
    doc.setDrawColor(226, 232, 240);
    doc.setLineWidth(0.25);
    if (typeof doc.roundedRect === 'function') {
        doc.roundedRect(m, y, contentW, blockH, 2, 2, 'FD');
    } else {
        doc.rect(m, y, contentW, blockH, 'FD');
    }

    const pad = 4;
    const imgW = contentW - pad * 2;
    const imgH = blockH - pad * 2;
    doc.addImage(img, 'PNG', m + pad, y + pad, imgW, imgH);
    return y + blockH + 10;
}

export async function downloadWeeklyReportPDF() {
    const filterEl = document.getElementById('project-filter');
    const projectFilter = filterEl ? filterEl.value : 'all';
    const rangeQ = typeof window.getDashboardRangeQuery === 'function' ? window.getDashboardRangeQuery() : 'range=7d';
    if (rangeQ === null) return notify('Select a valid time range first.', 'error');

    try {
        const metricsRes = await fetchWithAuth(`/api/metrics?${rangeQ}&_ts=${Date.now()}`);
        const metricsData = await metricsRes.json();
        if (!metricsData) return notify('Could not load metrics for export.', 'error');

        const res = await fetchWithAuth(`/api/reports/weekly?project=${encodeURIComponent(projectFilter)}&${rangeQ}&_ts=${Date.now()}`);
        const tableData = await res.json();

        if (!tableData || tableData.length === 0) {
            return notify('No incidents in the selected time range.', 'warning');
        }

        const layout = analyticsLayout.getAnalyticsLayout();
        const chartImages = await analyticsLayout.renderChartsForPdfExport(metricsData, layout);
        const rangeLabel = document.getElementById('dashboard-range-summary')?.textContent || rangeQ;
        const titleScope = projectFilter === 'all' ? 'Global Organization' : `Project: ${projectFilter}`;

        let logoData = null;
        try { logoData = await loadImageAsDataUrl('/assets/logo.png'); } catch { /* optional */ }

        const { jsPDF } = window.jspdf;
        const doc = new jsPDF({ unit: 'mm', format: 'a4' });

        const paintPageBg = () => {
            doc.setFillColor(248, 250, 252);
            doc.rect(0, 0, PDF_PAGE.w, PDF_PAGE.h, 'F');
        };
        paintPageBg();

        let y = pdfDrawHeader(doc, { logoData, titleScope, rangeLabel });
        y = pdfDrawKpiRow(doc, y, layout.filter(w => w.type === 'metric'), metricsData);

        layout.filter(w => w.type !== 'metric').forEach(w => {
            const img = chartImages[`analytics-${w.id}`];
            if (!img) return;
            y = pdfDrawChartBlock(doc, y, w.title || 'Chart', img, w.type);
        });

        y = pdfEnsureSpace(doc, y, 40);
        y = pdfDrawSectionTitle(doc, y, 'Pipeline health summary');

        doc.autoTable({
            head: [['Pipeline', 'Total alerts', 'Resolved', 'Flaky tests', 'Health']],
            body: tableData.map(row => [
                row.pipeline,
                row.total_alerts,
                row.resolved_alerts,
                row.flaky_tests,
                `${row.health_score}%`
            ]),
            startY: y,
            margin: { left: PDF_PAGE.margin, right: PDF_PAGE.margin },
            theme: 'plain',
            headStyles: {
                fillColor: [241, 245, 249],
                textColor: [51, 65, 85],
                fontStyle: 'bold',
                fontSize: 9,
                cellPadding: 4
            },
            bodyStyles: {
                fillColor: [255, 255, 255],
                textColor: [30, 41, 59],
                fontSize: 9,
                cellPadding: 4
            },
            alternateRowStyles: { fillColor: [248, 250, 252] },
            styles: {
                font: 'helvetica',
                lineColor: [226, 232, 240],
                lineWidth: 0.2
            }
        });

        const pageCount = doc.getNumberOfPages();
        for (let p = 1; p <= pageCount; p++) {
            doc.setPage(p);
            doc.setFontSize(7);
            doc.setTextColor(148, 163, 184);
            doc.text(
                `QA Capsule · Confidential · Page ${p} / ${pageCount}`,
                PDF_PAGE.w / 2,
                PDF_PAGE.h - 8,
                { align: 'center' }
            );
        }

        const dateStr = new Date().toISOString().split('T')[0];
        const suffix = projectFilter === 'all' ? 'Global' : projectFilter;
        doc.save(`QA_Capsule_Executive_Report_${suffix}_${dateStr}.pdf`);
        notify('PDF report generated successfully.', 'success');
    } catch (err) {
        console.error('PDF Gen Error:', err);
        notify('Failed to generate PDF report', 'error');
    } finally {
        analyticsLayout.refreshMainAnalyticsGrid();
    }
}

export function saveAnalyticsLayout() {
    const modal = document.getElementById('analytics-layout-modal');
    if (modal && modal.style.display === 'flex') {
        return analyticsLayout.saveAnalyticsLayoutFromModal();
    }
    return analyticsLayout.saveAnalyticsLayoutToPrefs();
}

export function resetAnalyticsLayout() {
    analyticsLayout.resetAnalyticsLayoutDefault();
    notify('Analytics layout reset to default. Click Save layout to persist.', 'info');
}

const PLUGIN_FOLDER_LABELS = {
    pagerduty: 'Paging (PagerDuty)',
    opsgenie: 'Paging (Opsgenie)',
    victorops: 'Paging (Splunk On-Call / VictorOps)',
    github: 'CI/CD (GitHub Actions)',
    datadog: 'Observability (Datadog)',
    webhook: 'Integrations (HTTP webhook)',
    qa: 'QA workflows',
    testrail: 'Test management (TestRail)',
    zephyr: 'Test management (Zephyr Scale)',
    xray: 'Test management (Xray Cloud)',
    email: 'Email (SendGrid / SMTP)',
    jira: 'Ticketing (Jira)',
    slack: 'Chat (Slack)',
    teams: 'Chat (Microsoft Teams)',
    k8s: 'Kubernetes remediation'
};

function escapePluginHtml(s) {
    return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function pluginFolderName(filePath) {
    const parts = (filePath || '').split(/[/\\]/);
    return parts.length > 1 ? parts[0] : 'general';
}

function pluginSearchHaystack(p, folder, folderLabel) {
    return [
        p.name,
        p.description,
        p.status,
        p.version,
        p.file_path,
        folder,
        folderLabel,
        ...(p.trigger_on || []),
        ...Object.keys(p.env || {})
    ].join(' ').toLowerCase();
}

function pluginMatchesFilters(p, folder, folderLabel, query, category) {
    if (category && category !== 'all' && folder.toLowerCase() !== category) return false;
    if (!query) return true;
    return pluginSearchHaystack(p, folder, folderLabel).includes(query);
}

function populatePluginCategoryFilter(plugins) {
    const sel = document.getElementById('plugin-category-filter');
    if (!sel) return;
    const folders = [...new Set(plugins.map(p => pluginFolderName(p.file_path).toLowerCase()))].sort();
    const current = sel.value || 'all';
    sel.innerHTML = '<option value="all">All categories</option>';
    folders.forEach(f => {
        const label = PLUGIN_FOLDER_LABELS[f] || f;
        sel.innerHTML += `<option value="${escapePluginHtml(f)}">${escapePluginHtml(label)}</option>`;
    });
    if ([...sel.options].some(o => o.value === current)) sel.value = current;
}

function updatePluginSearchMeta(shown, total, query, category) {
    const meta = document.getElementById('plugin-search-meta');
    if (!meta) return;
    if (total === 0) {
        meta.textContent = 'No modules detected in the plugins directory.';
        return;
    }
    let text = `Showing ${shown} of ${total} plugin${total === 1 ? '' : 's'}`;
    if (query || (category && category !== 'all')) text += ' (filtered)';
    meta.textContent = text;
}

function renderPluginList(plugins, query = '', category = 'all') {
    const listEl = document.getElementById('plugin-list');
    if (!listEl) return;

    const q = (query || '').trim().toLowerCase();
    if (!plugins || plugins.length === 0) {
        listEl.innerHTML = "<div style='text-align:center; padding:40px; opacity:0.5;'>No modules detected in the plugins directory.</div>";
        updatePluginSearchMeta(0, 0, q, category);
        return;
    }

    const configIcon = `<svg style="width:14px;height:14px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`;
    const runIcon = `<svg style="width:14px;height:14px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>`;
    const saveIcon = `<svg style="width:14px;height:14px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>`;
    const consoleIcon = `<svg style="width:12px;height:12px;margin-right:5px;vertical-align:middle;stroke:currentColor;fill:none;" viewBox="0 0 24 24" stroke-width="2"><polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line></svg>`;
    const pluginIconSVG = `<svg style="width:24px;height:24px;stroke:#8b949e;fill:none;" viewBox="0 0 24 24" stroke-width="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>`;

    const groupedPlugins = {};
    plugins.forEach(p => {
        const folder = pluginFolderName(p.file_path);
        if (!groupedPlugins[folder]) groupedPlugins[folder] = [];
        groupedPlugins[folder].push(p);
    });

    let html = '';
    let globalIdx = 0;
    let shown = 0;

    const sortedFolders = Object.keys(groupedPlugins).sort((a, b) => {
        const la = PLUGIN_FOLDER_LABELS[a.toLowerCase()] || a;
        const lb = PLUGIN_FOLDER_LABELS[b.toLowerCase()] || b;
        return la.localeCompare(lb);
    });

    for (const folder of sortedFolders) {
        const folderLabel = PLUGIN_FOLDER_LABELS[folder.toLowerCase()] || folder;
        const visible = groupedPlugins[folder].filter(p => pluginMatchesFilters(p, folder, folderLabel, q, category));
        if (visible.length === 0) continue;

        html += `<h3 class="plugin-category-heading" data-folder="${escapePluginHtml(folder.toLowerCase())}" style="margin: 30px 0 15px 0; padding-bottom: 8px; border-bottom: 1px solid var(--border-main); color: var(--text-main); font-size: 14px; letter-spacing: 1px; text-transform: uppercase; opacity: 0.8;">${escapePluginHtml(folderLabel)} <span style="font-weight:400; opacity:0.65;">(${visible.length})</span></h3>`;

        visible.forEach(p => {
            shown++;
            const safePath = p.file_path.replace(/\\/g, '/');
            const envKeys = Object.keys(p.env || {});
            let envs = '';
            envKeys.forEach(k => {
                envs += `<div style="margin-bottom: 12px;"><label style="font-size:11px; color:#8b949e; display:block; margin-bottom:5px; text-transform:uppercase; font-weight:bold; letter-spacing:0.5px;">${escapePluginHtml(k)}</label><input type="text" id="env-${globalIdx}-${k}" class="login-input" style="padding:10px; margin:0; width:100%; box-sizing:border-box; background:#0d1117;" value="${escapePluginHtml(p.env[k])}"></div>`;
            });
            const desc = p.description || 'No description provided.';
            const logoUrl = `/assets/${folder.toLowerCase()}.png`;
            const triggers = (p.trigger_on || []).join(', ');
            const autoBadge = p.status === 'Active'
                ? '<span style="font-size:10px; color:#3fb950; border:1px solid #3fb950; padding:2px 6px; border-radius:4px; letter-spacing: 0.5px;">AUTO-RUN ON</span>'
                : '';
            const envKeysAttr = envKeys.join(',');

            html += `
                <div class="data-card plugin-card" data-plugin-idx="${globalIdx}" data-folder="${escapePluginHtml(folder.toLowerCase())}" style="border-left: 4px solid #58a6ff; margin-bottom: 20px; padding: 20px; transition: all 0.2s ease;">
                    <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap: wrap; gap: 20px;">
                        <div style="display:flex; align-items:center; gap: 15px; flex-grow: 1;">
                            <div style="position:relative; width:50px; height:50px; flex-shrink:0;">
                                <div style="position:absolute; top:0; left:0; width:100%; height:100%; border-radius:10px; background:#21262d; border:1px solid #30363d; display:flex; align-items:center; justify-content:center; z-index:1;">${pluginIconSVG}</div>
                                <img src="${logoUrl}" onerror="this.style.display='none'" alt="" style="position:absolute; top:0; left:0; width:100%; height:100%; object-fit:contain; border-radius:10px; background:#0d1117; padding:5px; border:1px solid #30363d; box-sizing:border-box; z-index:2;" />
                            </div>
                            <div>
                                <h3 style="margin:0 0 5px 0; color:var(--text-main); font-size: 16px; display:flex; align-items:center; gap:10px; flex-wrap:wrap;">
                                    ${escapePluginHtml(p.name)} <span style="font-size:10px; background:#238636; color:white; padding:3px 8px; border-radius:12px;">v${escapePluginHtml(p.version || '1.0')}</span>
                                    ${autoBadge}
                                </h3>
                                <p style="font-size:13px; color:#8b949e; margin:0 0 4px; line-height: 1.4;">${escapePluginHtml(desc)}</p>
                                ${triggers ? `<p style="font-size:11px; color:#6e7681; margin:0;">Triggers: ${escapePluginHtml(triggers)}</p>` : ''}
                            </div>
                        </div>
                        <div style="display:flex; gap:12px; flex-shrink: 0;">
                            <button type="button" class="btn-secondary" style="display:flex; align-items:center; padding: 8px 16px;" onclick="window.togglePluginConfig('config-${globalIdx}')">${configIcon} Configure</button>
                            <button type="button" class="btn-primary" id="btn-run-${globalIdx}" onclick="window.runPlugin('${safePath}', ${globalIdx})" style="background-color:#58a6ff; border-color:#58a6ff; color: #0d1117; font-weight: bold; padding: 8px 16px; display:flex; align-items:center; justify-content:center; min-width: 120px;">${runIcon} Execute</button>
                        </div>
                    </div>
                    <div id="config-${globalIdx}" style="display:none; margin-top:20px; background:var(--bg-main); padding:20px; border-radius:8px; border:1px solid var(--border-main);">
                        <h4 style="margin:0 0 15px 0; font-size:13px; text-transform:uppercase; color:#58a6ff; letter-spacing:1px;">Environment Variables</h4>
                        ${envs || '<p style="font-size:13px; color:#8b949e;">No external variables required.</p>'}
                        ${envs ? `<button type="button" class="btn-primary" style="font-size:12px; width:100%; margin-top:15px; padding:10px; display:flex; align-items:center; justify-content:center;" onclick="window.savePluginConfig('${safePath}', ${globalIdx}, '${envKeysAttr}')">${saveIcon} Save</button>` : ''}
                    </div>
                    <div id="logs-container-${globalIdx}" style="display:none; margin-top:20px;">
                        <div style="font-size:12px; color:#8b949e; margin-bottom:8px; text-transform:uppercase; font-weight:bold; display:flex; align-items:center;">${consoleIcon} STDOUT Console</div>
                        <pre id="logs-${globalIdx}" style="background:#0d1117; color:#00ff00; padding:15px; border-radius:8px; border:1px solid #30363d; font-family:monospace; font-size:13px; overflow-x:auto; white-space:pre-wrap; margin:0; max-height:400px; overflow-y:auto;"></pre>
                    </div>
                </div>`;
            globalIdx++;
        });
    }

    if (!html) {
        html = `<div style="text-align:center; padding:48px 24px; opacity:0.65;">
            <p style="margin:0 0 8px; font-size:15px;">No plugins match your search.</p>
            <p style="margin:0; font-size:13px;">Try another keyword or reset filters.</p>
        </div>`;
    }

    listEl.innerHTML = html;
    updatePluginSearchMeta(shown, plugins.length, q, category);
}

let pluginFilterDebounce = null;

export function filterPlugins() {
    if (!window.__pluginsCache) return;
    clearTimeout(pluginFilterDebounce);
    pluginFilterDebounce = setTimeout(() => {
        const query = document.getElementById('plugin-search')?.value || '';
        const category = document.getElementById('plugin-category-filter')?.value || 'all';
        renderPluginList(window.__pluginsCache, query, category);
    }, 120);
}

export function resetPluginFilters() {
    const search = document.getElementById('plugin-search');
    const cat = document.getElementById('plugin-category-filter');
    if (search) search.value = '';
    if (cat) cat.value = 'all';
    filterPlugins();
}

export function loadPlugins() {
    const meta = document.getElementById('plugin-search-meta');
    if (meta) meta.textContent = 'Loading modules…';

    fetchWithAuth(`/api/plugins?_ts=${Date.now()}`).then(res => res.json()).then(data => {
        window.__pluginsCache = data || [];
        populatePluginCategoryFilter(window.__pluginsCache);
        const query = document.getElementById('plugin-search')?.value || '';
        const category = document.getElementById('plugin-category-filter')?.value || 'all';
        renderPluginList(window.__pluginsCache, query, category);
    }).catch(() => {
        notify('Network error', 'error');
        if (meta) meta.textContent = 'Failed to load plugins.';
    });
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