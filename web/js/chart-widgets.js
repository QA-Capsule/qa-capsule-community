/**
 * Shared Chart.js rendering and pinned chart mounts (Dashboard, FinOps, …)
 */
import { fetchWithAuth } from './api.js';

const chartInstances = new Map();

const CHART_COLORS = [
    { bg: 'rgba(88, 166, 255, 0.55)', border: '#58a6ff' },
    { bg: 'rgba(63, 185, 80, 0.55)', border: '#3fb950' },
    { bg: 'rgba(210, 153, 34, 0.55)', border: '#d29922' },
    { bg: 'rgba(255, 123, 114, 0.55)', border: '#ff7b72' },
    { bg: 'rgba(163, 113, 247, 0.55)', border: '#a371f7' }
];

function escapeHtml(s) {
    return String(s || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

export function destroyChartInstance(key) {
    const inst = chartInstances.get(key);
    if (inst) {
        inst.destroy();
        chartInstances.delete(key);
    }
}

export function renderChartOnCanvas(canvas, spec, instanceKey) {
    if (!canvas || !spec) return null;
    destroyChartInstance(instanceKey);

    const datasets = (spec.datasets || []).map((ds, i) => {
        const c = CHART_COLORS[i % CHART_COLORS.length];
        return {
            label: ds.label,
            data: ds.data,
            backgroundColor: c.bg,
            borderColor: c.border,
            borderWidth: 2,
            tension: spec.chart_type === 'line' ? 0.35 : 0,
            fill: spec.chart_type === 'line'
        };
    });

    const inst = new Chart(canvas.getContext('2d'), {
        type: spec.chart_type || 'line',
        data: { labels: spec.labels || [], datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: datasets.length > 1 || spec.chart_type === 'doughnut',
                    labels: { color: '#c9d1d9', boxWidth: 12 }
                },
                title: {
                    display: !!spec.title,
                    text: spec.title || '',
                    color: '#c9d1d9',
                    font: { size: 14, weight: '600' }
                }
            },
            scales: spec.chart_type === 'doughnut' ? {} : {
                x: { ticks: { color: '#8b949e', maxRotation: 45 }, grid: { color: 'rgba(48,54,61,0.5)' } },
                y: { ticks: { color: '#8b949e' }, grid: { color: 'rgba(48,54,61,0.5)' } }
            }
        }
    });
    chartInstances.set(instanceKey, inst);
    return inst;
}

/**
 * Load and render charts pinned to a section (dashboard | finops).
 */
export async function mountPinnedCharts(location, containerId, wrapId) {
    const container = document.getElementById(containerId);
    const wrap = wrapId ? document.getElementById(wrapId) : container;
    if (!container) return;

    try {
        const res = await fetchWithAuth(`/api/charts/pinned?location=${encodeURIComponent(location)}&_ts=${Date.now()}`);
        if (!res.ok) throw new Error();
        const data = await res.json();
        const charts = data.charts || [];

        if (charts.length === 0) {
            container.innerHTML = '';
            if (wrap && wrap !== container) wrap.style.display = 'none';
            return;
        }

        if (wrap && wrap !== container) wrap.style.display = '';
        container.innerHTML = charts.map(c => `
            <article class="pinned-chart-card" data-chart-id="${c.id}">
                <header class="pinned-chart-header">
                    <h4>${escapeHtml(c.name)}</h4>
                    ${c.description ? `<p>${escapeHtml(c.description)}</p>` : ''}
                </header>
                <div class="pinned-chart-canvas-wrap">
                    <canvas id="pinned-${location}-chart-${c.id}"></canvas>
                </div>
            </article>
        `).join('');

        charts.forEach(c => {
            const canvas = document.getElementById(`pinned-${location}-chart-${c.id}`);
            if (canvas) renderChartOnCanvas(canvas, c.spec, `pinned-${location}-${c.id}`);
        });
    } catch {
        container.innerHTML = '';
    }
}

export function clearPinnedCharts(location) {
    [...chartInstances.keys()].filter(k => k.startsWith(`pinned-${location}-`)).forEach(destroyChartInstance);
}
