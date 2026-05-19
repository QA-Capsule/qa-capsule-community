/**
 * FinOps Intelligence — Manager role only
 */
import { fetchWithAuth } from './api.js';
import { notify } from './ui.js';
import { currencySymbols } from './settings.js';
import { mountPinnedCharts } from './chart-widgets.js';

let finopsWeeklyChart = null;
let finopsMetricsChart = null;

export function loadFinOpsView() {
    loadFinOpsBaselines();
    refreshFinOpsKPIs();
    loadFinOpsWeeklyTable();
    loadFinOpsWeeklyEvolution();
    loadFinOpsProjectFilter();
    refreshFinOpsPinnedCharts();
}

export function refreshFinOpsPinnedCharts() {
    return mountPinnedCharts('finops', 'finops-pinned-charts', 'finops-pinned-wrap');
}

function loadFinOpsProjectFilter() {
    fetchWithAuth('/api/my-projects')
        .then(r => r.json())
        .then(projects => {
            const sel = document.getElementById('finops-export-project');
            if (!sel) return;
            sel.innerHTML = '<option value="all">All gateways</option>';
            (projects || []).forEach(p => {
                sel.innerHTML += `<option value="${p.name}">${p.name}</option>`;
            });
        });
}

function loadFinOpsBaselines() {
    fetchWithAuth(`/api/finops?_ts=${Date.now()}`)
        .then(res => { if (!res.ok) throw new Error(); return res.json(); })
        .then(data => {
            const set = (id, val) => { const el = document.getElementById(id); if (el) el.value = val; };
            set('finops-dev-rate', data.dev_hourly_rate);
            set('finops-ci-cost', data.ci_minute_cost);
            set('finops-duration', data.avg_pipeline_duration);
            set('finops-investigation', data.avg_investigation_time);
            const cur = document.getElementById('finops-currency');
            if (cur && data.currency) cur.value = data.currency;
        })
        .catch(() => notify('Could not load FinOps settings', 'error'));
}

export function saveFinOpsBaselines() {
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
                notify('FinOps baseline saved. KPIs will recalculate.', 'success');
                refreshFinOpsKPIs();
                loadFinOpsWeeklyEvolution();
                loadFinOpsWeeklyTable();
            } else notify('Failed to save FinOps baseline', 'error');
        });
}

export function exportFinOpsReport() {
    const period = document.getElementById('finops-export-period')?.value || 'week';
    const project = document.getElementById('finops-export-project')?.value || 'all';
    const url = `/api/finops/export?period=${encodeURIComponent(period)}&project=${encodeURIComponent(project)}&format=csv&_ts=${Date.now()}`;

    fetchWithAuth(url)
        .then(res => {
            if (!res.ok) throw new Error();
            return res.blob();
        })
        .then(blob => {
            const a = document.createElement('a');
            a.href = URL.createObjectURL(blob);
            a.download = `finops-report-${period}-${project}-${new Date().toISOString().slice(0, 10)}.csv`;
            a.click();
            notify('FinOps report exported', 'success');
        })
        .catch(() => notify('Export failed', 'error'));
}

function refreshFinOpsKPIs() {
    fetchWithAuth(`/api/metrics?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(data => {
            const sym = currencySymbols[document.getElementById('finops-currency')?.value || 'USD'] || '$';
            const impact = data.sre_impact || {};
            const setText = (id, text) => { const el = document.getElementById(id); if (el) el.textContent = text; };

            setText('finops-kpi-total-cost', sym + (impact.estimated_cost_usd ?? '—'));
            setText('finops-kpi-flaky-cost', sym + (impact.flaky_waste_cost_usd ?? '—'));
            setText('finops-kpi-ci-minutes', (impact.ci_minutes_lost ?? 0) + ' min');
            setText('finops-kpi-mttr', (data.mttr_minutes ?? 0) + ' min');
            setText('finops-kpi-incidents', String(data.total_incidents ?? 0));
            setText('finops-kpi-flaky-count', String(data.flaky_tests ?? 0));

            const total = impact.estimated_cost_usd || 0;
            const flaky = impact.flaky_waste_cost_usd || 0;
            const pct = total > 0 ? Math.round((flaky / total) * 100) : 0;
            setText('finops-kpi-waste-pct', pct + '% of total');
        })
        .catch(() => {});
}

function loadFinOpsWeeklyTable() {
    fetchWithAuth(`/api/reports/weekly?_ts=${Date.now()}`)
        .then(res => res.json())
        .then(rows => {
            const tbody = document.getElementById('finops-weekly-body');
            if (!tbody) return;
            if (!rows || rows.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;padding:20px;opacity:0.5;">No data in the last 7 days.</td></tr>';
                return;
            }
            tbody.innerHTML = rows.map(r => `
                <tr style="border-bottom:1px solid var(--border-main);">
                    <td style="padding:10px;"><strong>${r.pipeline}</strong></td>
                    <td style="padding:10px;">${r.total_alerts}</td>
                    <td style="padding:10px;color:#3fb950;">${r.resolved_alerts}</td>
                    <td style="padding:10px;color:#d29922;">${r.flaky_tests}</td>
                    <td style="padding:10px;">${r.health_score}%</td>
                </tr>`).join('');
        });
}

function loadFinOpsWeeklyEvolution() {
    fetchWithAuth('/api/finops/evolution?weeks=12')
        .then(res => res.json())
        .then(data => {
            if (!data.series) return;
            renderFinOpsMetricsChart(data.series);
            renderFinOpsCostChart(data.series);
        });
}

function renderFinOpsMetricsChart(series) {
    const canvas = document.getElementById('finops-evolution-chart');
    if (!canvas || !series.length) return;
    if (finopsWeeklyChart) finopsWeeklyChart.destroy();

    finopsWeeklyChart = new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {
            labels: series.map(s => s.week_start),
            datasets: [
                { label: 'Incidents', data: series.map(s => s.total_incidents), backgroundColor: 'rgba(88, 166, 255, 0.65)' },
                { label: 'Flaky', data: series.map(s => s.flaky_count), backgroundColor: 'rgba(210, 153, 34, 0.65)' },
                { label: 'MTTR (min)', type: 'line', data: series.map(s => Math.round(s.mttr_minutes)), borderColor: '#3fb950', yAxisID: 'y1', tension: 0.3 }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: { position: 'left' },
                y1: { position: 'right', grid: { drawOnChartArea: false } }
            }
        }
    });
}

function renderFinOpsCostChart(series) {
    const canvas = document.getElementById('finops-cost-evolution-chart');
    if (!canvas || !series.length) return;
    if (finopsMetricsChart) finopsMetricsChart.destroy();

    finopsMetricsChart = new Chart(canvas.getContext('2d'), {
        type: 'line',
        data: {
            labels: series.map(s => s.week_start),
            datasets: [
                { label: 'Total FinOps cost (USD)', data: series.map(s => s.estimated_cost_usd), borderColor: '#58a6ff', fill: true, backgroundColor: 'rgba(88,166,255,0.1)', tension: 0.35 },
                { label: 'Flaky waste (USD)', data: series.map(s => s.flaky_cost_usd), borderColor: '#d29922', tension: 0.35 }
            ]
        },
        options: { responsive: true, maintainAspectRatio: false }
    });
}
