/**
 * DORA & Executive dashboard (Manager).
 */
import { fetchWithAuth, parseApiJson } from './api.js';
import { CHART_PALETTE } from './chart-palette.js';
import { getChartTheme, withChartTheme } from './chart-theme.js';
import { setPremiumKpi } from './kpi-premium.js';

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
            const cfr = m.change_failure_rate;
            const cfrPct = cfr != null ? `${(cfr * 100).toFixed(1)}%` : '—';
            const deployFreq = m.deployment_frequency_per_day?.toFixed(2) ?? '—';

            setPremiumKpi('dora-kpi-deploy-freq', deployFreq, {
                tone: 'info',
                trend: (m.deployment_frequency_per_day ?? 0) >= 0.5 ? 'up' : '',
                trendText: '↑ Delivery cadence'
            });

            const leadMin = m.lead_time_minutes_median;
            setPremiumKpi('dora-kpi-lead-time', formatMinutes(leadMin), {
                tone: leadMin != null && leadMin > 120 ? 'warn' : 'success',
                trend: leadMin != null && leadMin > 120 ? 'down' : 'up',
                trendText: leadMin != null && leadMin > 120 ? '↓ Slow lead time' : '↑ On track'
            });

            setPremiumKpi('dora-kpi-cfr', cfrPct, {
                tone: cfr != null && cfr > 0.15 ? 'danger' : 'success',
                trend: cfr != null && cfr > 0.15 ? 'down' : 'up',
                trendText: cfr != null && cfr > 0.15 ? '↓ Above target' : '↑ Within target'
            });

            const mttr = m.mttr_minutes;
            setPremiumKpi('dora-kpi-mttr', formatMinutes(mttr), {
                tone: mttr != null && mttr > 60 ? 'warn' : 'success',
                trend: mttr != null && mttr > 60 ? 'down' : 'up',
                trendText: mttr != null && mttr > 60 ? '↓ Recovery slow' : '↑ Healthy MTTR'
            });

            setPremiumKpi('dora-kpi-deployments', String(m.deployments ?? '—'), {
                tone: 'neutral',
                trend: (m.deployments ?? 0) > 0 ? 'up' : '',
                trendText: '↑ Pipeline activity'
            });

            setPremiumKpi('dora-kpi-signals', String(m.external_signals ?? 0), {
                tone: 'info',
                trend: (m.external_signals ?? 0) > 0 ? 'up' : '',
                trendText: '↑ Observability linked'
            });

            setPremiumKpi('dora-kpi-correlated', String(m.correlated_incidents ?? 0), {
                tone: (m.correlated_incidents ?? 0) > 0 ? 'warn' : 'neutral',
                trend: (m.correlated_incidents ?? 0) > 0 ? 'down' : 'up',
                trendText: (m.correlated_incidents ?? 0) > 0 ? '↓ Correlations found' : '↑ No correlations'
            });

            renderDORATrend(m.series || []);
            renderCorrelations(data.correlations || []);
        });
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
    const theme = getChartTheme();
    doraTrendChart = new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {
            labels,
            datasets: [
                {
                    label: 'Deployments',
                    data: series.map(s => s.deployments),
                    backgroundColor: CHART_PALETTE.series[0]
                },
                {
                    label: 'Failed',
                    data: series.map(s => s.failed_deployments),
                    backgroundColor: CHART_PALETTE.series[2]
                }
            ]
        },
        options: withChartTheme({
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { position: 'bottom' } },
            scales: {
                x: { stacked: true, grid: { display: false } },
                y: { stacked: true, beginAtZero: true }
            }
        }, theme)
    });
}

function renderCorrelations(rows) {
    const tbody = document.getElementById('dora-correlations-body');
    if (!tbody) return;
    if (!rows.length) {
        tbody.innerHTML = '<tr><td colspan="4" class="modern-data-grid__empty">No Prometheus ↔ incident correlations in range.</td></tr>';
        return;
    }
    tbody.innerHTML = rows.map(r => `
        <tr>
            <td>${escapeHtml(r.signal_name)}</td>
            <td>#${r.incident_id}</td>
            <td>${escapeHtml(r.incident_name || '')}</td>
            <td>${escapeHtml(r.fired_at || '')}</td>
        </tr>
    `).join('');
}

function escapeHtml(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
