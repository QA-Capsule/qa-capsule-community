/**
 * DORA & Executive dashboard (Manager).
 */
import { fetchWithAuth, parseApiJson } from './api.js';
import { CHART_PALETTE } from './chart-palette.js';

let doraTrendChart = null;

export function loadDORAView() {
    loadDORAProjectFilter();
    refreshDORAMetrics();
}

export function destroyDORAChart() {
    if (doraTrendChart) {
        doraTrendChart.destroy();
        doraTrendChart = null;
    }
}

function loadDORAProjectFilter() {
    const sel = document.getElementById('dora-project-filter');
    if (!sel) return;
    fetchWithAuth('/api/my-projects')
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            if (!ok) return;
            const projects = Array.isArray(data) ? data : data?.projects || [];
            sel.innerHTML = '<option value="">All projects</option>';
            projects.forEach(p => {
                const opt = document.createElement('option');
                opt.value = p.name ?? p.Name ?? '';
                opt.textContent = opt.value;
                sel.appendChild(opt);
            });
        });
}

export function refreshDORAMetrics() {
    const project = document.getElementById('dora-project-filter')?.value || '';
    const range = document.getElementById('dora-range-select')?.value || '30d';
    const q = new URLSearchParams({ range, project }).toString();
    fetchWithAuth(`/api/dora/metrics?${q}`)
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            if (!ok || !data?.metrics) return;
            const m = data.metrics;
            setText('dora-kpi-deploy-freq', m.deployment_frequency_per_day?.toFixed(2) ?? '—');
            setText('dora-kpi-lead-time', formatMinutes(m.lead_time_minutes_median));
            setText('dora-kpi-cfr', m.change_failure_rate != null ? `${(m.change_failure_rate * 100).toFixed(1)}%` : '—');
            setText('dora-kpi-mttr', formatMinutes(m.mttr_minutes));
            setText('dora-kpi-deployments', String(m.deployments ?? '—'));
            setText('dora-kpi-signals', String(m.external_signals ?? 0));
            setText('dora-kpi-correlated', String(m.correlated_incidents ?? 0));
            renderDORATrend(m.series || []);
            renderCorrelations(data.correlations || []);
        });
}

function setText(id, val) {
    const el = document.getElementById(id);
    if (el) el.textContent = val;
}

function formatMinutes(v) {
    if (v == null || Number.isNaN(v)) return '—';
    if (v < 60) return `${Math.round(v)}m`;
    return `${(v / 60).toFixed(1)}h`;
}

function renderDORATrend(series) {
    const canvas = document.getElementById('dora-trend-chart');
    if (!canvas || typeof Chart === 'undefined') return;
    if (doraTrendChart) doraTrendChart.destroy();
    const labels = series.map(s => s.period_start || '');
    doraTrendChart = new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {
            labels,
            datasets: [
                {
                    label: 'Deployments',
                    data: series.map(s => s.deployments),
                    backgroundColor: CHART_PALETTE[0]
                },
                {
                    label: 'Failed',
                    data: series.map(s => s.failed_deployments),
                    backgroundColor: CHART_PALETTE[2]
                }
            ]
        },
        options: {
            responsive: true,
            plugins: { legend: { position: 'bottom' } },
            scales: { x: { stacked: true }, y: { stacked: true, beginAtZero: true } }
        }
    });
}

function renderCorrelations(rows) {
    const tbody = document.getElementById('dora-correlations-body');
    if (!tbody) return;
    if (!rows.length) {
        tbody.innerHTML = '<tr><td colspan="4" style="padding:12px;opacity:0.6;">No Prometheus ↔ incident correlations in range.</td></tr>';
        return;
    }
    tbody.innerHTML = rows.map(r => `
        <tr>
            <td style="padding:8px;">${escapeHtml(r.signal_name)}</td>
            <td style="padding:8px;">#${r.incident_id}</td>
            <td style="padding:8px;">${escapeHtml(r.incident_name || '')}</td>
            <td style="padding:8px;">${escapeHtml(r.fired_at || '')}</td>
        </tr>
    `).join('');
}

function escapeHtml(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
