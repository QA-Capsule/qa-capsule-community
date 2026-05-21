/**

 * FinOps Intelligence — Manager role only

 */

import { fetchWithAuth, parseApiJson, asArray } from './api.js';

import { notify } from './ui.js';

import { currencySymbols } from './settings.js';

import { CHART_PALETTE } from './chart-palette.js';



let finopsWeeklyChart = null;

let finopsMetricsChart = null;



/** Same time range as the Operations dashboard when set. */

export function getFinOpsRangeQuery() {
    if (typeof window.getDashboardRangeQuery === 'function') {
        const q = window.getDashboardRangeQuery();
        if (q) return q;
    }
    return 'range=90d';
}



export function loadFinOpsView() {

    loadFinOpsBaselines();

    refreshFinOpsKPIs();

    loadFinOpsWeeklyTable();

    loadFinOpsWeeklyEvolution();

    loadFinOpsProjectFilter();

}



function loadFinOpsProjectFilter() {

    fetchWithAuth('/api/my-projects')

        .then(res => parseApiJson(res))

        .then(({ ok, data }) => {

            const sel = document.getElementById('finops-export-project');

            if (!sel || !ok) return;

            sel.innerHTML = '<option value="all">All gateways</option>';

            asArray(data).forEach(p => {

                sel.innerHTML += `<option value="${p.name}">${p.name}</option>`;

            });

        })

        .catch(() => {});

}



function loadFinOpsBaselines() {

    fetchWithAuth(`/api/finops?_ts=${Date.now()}`)

        .then(res => { if (!res.ok) throw new Error(); return res.json(); })

        .then(data => {

            const set = (id, val) => { const el = document.getElementById(id); if (el) el.value = val; };

            set('finops-dev-rate', data.dev_hourly_rate ?? 50);

            set('finops-ci-cost', data.ci_minute_cost ?? 0.008);

            set('finops-duration', data.avg_pipeline_duration ?? 15);

            set('finops-investigation', data.avg_investigation_time ?? 30);

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

    const rangeQ = getFinOpsRangeQuery();

    fetchWithAuth(`/api/metrics?${rangeQ}&_ts=${Date.now()}`)

        .then(res => parseApiJson(res))

        .then(({ ok, data }) => {

            if (!ok || !data) return;

            const sym = currencySymbols[document.getElementById('finops-currency')?.value || 'USD'] || '$';

            const impact = data.sre_impact || {};

            const setText = (id, text) => { const el = document.getElementById(id); if (el) el.textContent = text; };



            setText('finops-kpi-total-cost', sym + (impact.estimated_cost_usd ?? '0'));

            setText('finops-kpi-flaky-cost', sym + (impact.flaky_waste_cost_usd ?? '0'));

            setText('finops-kpi-ci-minutes', (impact.ci_minutes_lost ?? 0) + ' min');

            setText('finops-kpi-mttr', (data.mttr_minutes ?? 0) + ' min');

            setText('finops-kpi-incidents', String(data.total_incidents ?? 0));

            setText('finops-kpi-flaky-count', String(data.flaky_tests ?? 0));



            const total = impact.estimated_cost_usd || 0;

            const flaky = impact.flaky_waste_cost_usd || 0;

            const pct = total > 0 ? Math.round((flaky / total) * 100) : 0;

            setText('finops-kpi-waste-pct', pct + '% of total');

        })

        .catch(() => notify('Could not load FinOps KPIs', 'error'));

}



function loadFinOpsWeeklyTable() {

    const rangeQ = getFinOpsRangeQuery();

    fetchWithAuth(`/api/reports/weekly?${rangeQ}&_ts=${Date.now()}`)

        .then(res => parseApiJson(res))

        .then(({ ok, data }) => {

            const tbody = document.getElementById('finops-weekly-body');

            if (!tbody) return;

            if (!ok) {

                tbody.innerHTML = '<tr><td colspan="5" class="load-error-msg">Unable to load weekly report.</td></tr>';

                return;

            }

            const rows = asArray(data);

            if (rows.length === 0) {

                tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;padding:20px;opacity:0.5;">No incidents in the last 90 days.</td></tr>';

                return;

            }

            tbody.innerHTML = rows.map(r => `

                <tr style="border-bottom:1px solid var(--border-main);">

                    <td style="padding:10px;"><strong>${r.pipeline}</strong></td>

                    <td style="padding:10px;">${r.total_alerts}</td>

                    <td style="padding:10px;color:#3fb950;">${r.resolved_alerts}</td>

                    <td style="padding:10px;color:#d97706;">${r.flaky_tests}</td>

                    <td style="padding:10px;">${r.health_score}%</td>

                </tr>`).join('');

        })

        .catch(() => {});

}



function showFinOpsChartEmpty(canvasId, message) {

    const canvas = document.getElementById(canvasId);

    if (!canvas) return;

    const wrap = canvas.parentElement;

    if (wrap) {

        wrap.innerHTML = `<div class="finops-chart-empty">${message}</div>`;

    }

}



function formatFinOpsLabel(raw, bucket) {
    if (!raw) return raw;
    const d = new Date(String(raw).replace(' ', 'T'));
    if (isNaN(d.getTime())) return raw;
    if (bucket === 'hour') return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    if (bucket === 'day') return d.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' });
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function loadFinOpsWeeklyEvolution() {
    const rangeQ = getFinOpsRangeQuery();
    fetchWithAuth(`/api/finops/evolution?${rangeQ}&_ts=${Date.now()}`)
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            if (!ok || !data?.series?.length) {
                const hint = 'No data for the selected time range. Widen the dashboard period or ingest more incidents.';
                showFinOpsChartEmpty('finops-evolution-chart', hint);
                showFinOpsChartEmpty('finops-cost-evolution-chart', hint);
                return;
            }
            renderFinOpsMetricsChart(data.series, data.evolution_bucket);
            renderFinOpsCostChart(data.series, data.evolution_bucket);
        })
        .catch(() => {
            showFinOpsChartEmpty('finops-evolution-chart', 'Failed to load FinOps evolution.');
        });
}



function renderFinOpsMetricsChart(series, bucket = 'week') {
    const wrap = document.getElementById('finops-evolution-chart')?.parentElement;
    if (wrap?.querySelector('.finops-chart-empty')) {
        wrap.innerHTML = '<canvas id="finops-evolution-chart"></canvas>';
    }
    const canvas = document.getElementById('finops-evolution-chart');
    if (!canvas || !series.length) return;
    if (finopsWeeklyChart) finopsWeeklyChart.destroy();

    finopsWeeklyChart = new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {
            labels: series.map(s => formatFinOpsLabel(s.week_start, bucket)),

            datasets: [

                { label: 'Incidents', data: series.map(s => s.total_incidents), backgroundColor: CHART_PALETTE.series[0] },

                { label: 'Flaky', data: series.map(s => s.flaky_count), backgroundColor: CHART_PALETTE.series[2] },

                { label: 'MTTR (min)', type: 'line', data: series.map(s => Math.round(s.mttr_minutes)), borderColor: CHART_PALETTE.series[1], yAxisID: 'y1', tension: 0.3 }

            ]

        },

        options: {

            responsive: true,

            maintainAspectRatio: false,

            plugins: { legend: { position: 'top' } },

            scales: {

                y: { position: 'left', beginAtZero: true },

                y1: { position: 'right', grid: { drawOnChartArea: false }, beginAtZero: true }

            }

        }

    });

}



function renderFinOpsCostChart(series, bucket = 'week') {
    const wrap = document.getElementById('finops-cost-evolution-chart')?.parentElement;
    if (wrap?.querySelector('.finops-chart-empty')) {
        wrap.innerHTML = '<canvas id="finops-cost-evolution-chart"></canvas>';
    }
    const canvas = document.getElementById('finops-cost-evolution-chart');
    if (!canvas || !series.length) return;
    if (finopsMetricsChart) finopsMetricsChart.destroy();

    const sym = currencySymbols[document.getElementById('finops-currency')?.value || 'USD'] || '$';

    finopsMetricsChart = new Chart(canvas.getContext('2d'), {
        type: 'line',
        data: {
            labels: series.map(s => formatFinOpsLabel(s.week_start, bucket)),

            datasets: [

                { label: `Total cost (${sym})`, data: series.map(s => s.estimated_cost_usd), borderColor: CHART_PALETTE.series[0], backgroundColor: 'rgba(37,99,235,0.12)', fill: true, tension: 0.35 },

                { label: `Flaky waste (${sym})`, data: series.map(s => s.flaky_cost_usd), borderColor: CHART_PALETTE.series[2], tension: 0.35 }

            ]

        },

        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'top' } } }

    });

}


