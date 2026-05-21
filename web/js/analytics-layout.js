/**
 * Customizable "System analytics & quality" layout — metrics, charts, colors, positions.
 */
import { fetchWithAuth, parseApiJson } from './api.js';
import { notify } from './ui.js';
import { CHART_PALETTE, defaultWidgetColors } from './chart-palette.js';

export const METRIC_CATALOG = {
    total_incidents: { label: 'Total incidents', hint: 'All failures in range' },
    active_backlog: { label: 'Active backlog', hint: 'Unresolved incidents' },
    resolved_incidents: { label: 'Resolved', hint: 'Marked resolved' },
    flaky_tests: { label: 'Flaky tests', hint: 'Tagged [FLAKY]' },
    stable_failures: { label: 'Stable failures', hint: 'Non-flaky failures' },
    resolution_rate: { label: 'Resolution rate', hint: 'Resolved / total %' },
    flaky_ratio: { label: 'Flaky ratio', hint: 'Flaky / total %' },
    mttr_minutes: { label: 'MTTR', hint: 'Mean time to resolve', suffix: ' min' },
    mttf_minutes: { label: 'MTTF', hint: 'Mean time to failure', suffix: ' min' }
};

export const DEFAULT_ANALYTICS_LAYOUT = [
    { id: 'm1', type: 'metric', metric: 'total_incidents', title: 'Total incidents', color: CHART_PALETTE.semantic.info, span: 1 },
    { id: 'm2', type: 'metric', metric: 'active_backlog', title: 'Active backlog', color: CHART_PALETTE.semantic.danger, span: 1 },
    { id: 'm3', type: 'metric', metric: 'resolution_rate', title: 'Resolution rate', color: CHART_PALETTE.semantic.success, span: 1 },
    { id: 'm4', type: 'metric', metric: 'flaky_ratio', title: 'Flaky ratio', color: CHART_PALETTE.semantic.warning, span: 1 },
    { id: 'c1', type: 'doughnut', title: 'Failure taxonomy', color: CHART_PALETTE.doughnut[0], color2: CHART_PALETTE.doughnut[1], span: 2 },
    { id: 'c2', type: 'evolution', title: 'Incident evolution', color: CHART_PALETTE.series[0], color2: CHART_PALETTE.series[2], color3: CHART_PALETTE.series[1], span: 4 }
];

let currentLayout = [...DEFAULT_ANALYTICS_LAYOUT];
let draftLayout = null;
let editMode = false;
let dragWidgetId = null;

function getActiveLayout() {
    return (editMode && draftLayout) ? draftLayout : currentLayout;
}

function setActiveLayout(layout) {
    if (editMode && draftLayout) draftLayout = layout;
    else currentLayout = layout;
}
let chartInstances = {};
let lastMetricsData = null;

export function getMetricDisplayValue(data, key) {
    return metricValue(data, key);
}

function metricValue(data, key) {
    if (!data) return '—';
    const total = data.total_incidents ?? 0;
    const resolved = data.resolved_incidents ?? 0;
    const flaky = data.flaky_tests ?? 0;
    switch (key) {
        case 'total_incidents': return String(total);
        case 'active_backlog': return String(Math.max(0, total - resolved));
        case 'resolved_incidents': return String(resolved);
        case 'flaky_tests': return String(flaky);
        case 'stable_failures': return String(data.stable_failures ?? Math.max(0, total - flaky));
        case 'resolution_rate': return total > 0 ? `${Math.round((resolved / total) * 100)}%` : 'N/A';
        case 'flaky_ratio': return total > 0 ? `${Math.round((flaky / total) * 100)}%` : 'N/A';
        case 'mttr_minutes': return `${data.mttr_minutes ?? 0} min`;
        case 'mttf_minutes': return (data.mttf_minutes > 0) ? `${data.mttf_minutes} min` : 'N/A';
        default: return '—';
    }
}

function spanClass(span) {
    const s = Math.min(4, Math.max(1, parseInt(span, 10) || 1));
    return `analytics-widget span-${s}`;
}

function destroyLayoutCharts() {
    Object.values(chartInstances).forEach(ch => {
        if (ch?.destroy) ch.destroy();
    });
    Object.keys(chartInstances).forEach(k => delete chartInstances[k]);
}

export function getAnalyticsLayout() {
    return [...currentLayout];
}

export async function loadAnalyticsLayoutFromPrefs() {
    try {
        const res = await fetchWithAuth('/api/me/preferences');
        const { ok, data } = await parseApiJson(res);
        if (ok && Array.isArray(data?.analytics_layout) && data.analytics_layout.length > 0) {
            currentLayout = data.analytics_layout;
        } else {
            currentLayout = [...DEFAULT_ANALYTICS_LAYOUT];
        }
    } catch {
        currentLayout = [...DEFAULT_ANALYTICS_LAYOUT];
    }
    return getAnalyticsLayout();
}

export async function saveAnalyticsLayoutToPrefs() {
    const res = await fetchWithAuth('/api/me/preferences', {
        method: 'PUT',
        body: JSON.stringify({ analytics_layout: currentLayout })
    });
    const { ok } = await parseApiJson(res);
    if (!ok) notify('Could not save analytics layout.', 'error');
}

export function toggleAnalyticsCustomize() {
    openAnalyticsLayoutModal();
}

export function openAnalyticsLayoutModal() {
    draftLayout = JSON.parse(JSON.stringify(currentLayout));
    editMode = true;
    window.pausePollingUntil = Date.now() + 3600000;
    const modal = document.getElementById('analytics-layout-modal');
    if (modal) modal.style.display = 'flex';
    syncAnalyticsWidgetForm();
    if (lastMetricsData) renderAnalyticsGrid(lastMetricsData, { isExport: false });
    else if (typeof window.loadAnalytics === 'function') window.loadAnalytics(false);
}

export function closeAnalyticsLayoutModal(discard = true) {
    editMode = false;
    draftLayout = null;
    dragWidgetId = null;
    window.pausePollingUntil = 0;
    const modal = document.getElementById('analytics-layout-modal');
    if (modal) modal.style.display = 'none';
    const modalGrid = document.getElementById('analytics-grid-modal');
    if (modalGrid) modalGrid.innerHTML = '';
    refreshMainAnalyticsGrid();
    if (discard) notify('Layout changes discarded.', 'info');
}

export function saveAnalyticsLayoutFromModal() {
    if (!draftLayout) return;
    currentLayout = JSON.parse(JSON.stringify(draftLayout));
    editMode = false;
    draftLayout = null;
    dragWidgetId = null;
    window.pausePollingUntil = 0;
    const modal = document.getElementById('analytics-layout-modal');
    if (modal) modal.style.display = 'none';
    const modalGrid = document.getElementById('analytics-grid-modal');
    if (modalGrid) modalGrid.innerHTML = '';
    saveAnalyticsLayoutToPrefs().then(() => {
        notify('Analytics layout saved.', 'success');
        refreshMainAnalyticsGrid();
    });
}

/** Re-render visible dashboard analytics after PDF export or modal close (main grid only). */
export function refreshMainAnalyticsGrid() {
    if (typeof window.loadAnalytics === 'function') {
        const view = document.getElementById('analytics-view');
        if (view && view.style.display !== 'none') {
            window.loadAnalytics(false);
            return;
        }
    }
    if (lastMetricsData) renderAnalyticsGrid(lastMetricsData, { isExport: false });
}

export function applyAnalyticsColorPreset(which) {
    const preset = document.getElementById(which === 2 ? 'aw-color2-preset' : 'aw-color-preset')?.value;
    if (!preset) return;
    const input = document.getElementById(which === 2 ? 'aw-color2' : 'aw-color');
    if (input) input.value = preset;
}

export function addAnalyticsWidget() {
    return addAnalyticsWidgetFromForm();
}

export function addAnalyticsWidgetFromForm() {
    if (!editMode || !draftLayout) {
        notify('Open Customize layout to add widgets.', 'warning');
        return;
    }
    const type = document.getElementById('aw-type')?.value || 'metric';
    const title = document.getElementById('aw-title')?.value?.trim() || 'New widget';
    const metric = document.getElementById('aw-metric')?.value || 'total_incidents';
    const color = document.getElementById('aw-color')?.value || CHART_PALETTE.semantic.info;
    const color2 = document.getElementById('aw-color2')?.value || CHART_PALETTE.semantic.warning;
    const span = parseInt(document.getElementById('aw-span')?.value || '1', 10);
    const defaults = defaultWidgetColors(type, draftLayout.length);
    const w = {
        id: `w-${Date.now()}`,
        type,
        title,
        color: color || defaults.color,
        color2: color2 || defaults.color2,
        color3: defaults.color3,
        span,
        metric: type === 'metric' ? metric : undefined
    };
    draftLayout.push(w);
    if (lastMetricsData) renderAnalyticsGrid(lastMetricsData, { isExport: false });
    notify('Widget added — click Save layout to keep changes.', 'info');
}

export function removeAnalyticsWidget(id) {
    const layout = getActiveLayout();
    setActiveLayout(layout.filter(w => w.id !== id));
    if (lastMetricsData) renderAnalyticsGrid(lastMetricsData, { isExport: false });
}

export function moveAnalyticsWidget(id, dir) {
    const layout = [...getActiveLayout()];
    const i = layout.findIndex(w => w.id === id);
    if (i < 0) return;
    const j = i + dir;
    if (j < 0 || j >= layout.length) return;
    const tmp = layout[i];
    layout[i] = layout[j];
    layout[j] = tmp;
    setActiveLayout(layout);
    if (lastMetricsData) renderAnalyticsGrid(lastMetricsData, { isExport: false });
}

function reorderLayoutByDrag(fromId, toId) {
    const layout = [...getActiveLayout()];
    const fromIdx = layout.findIndex(w => w.id === fromId);
    const toIdx = layout.findIndex(w => w.id === toId);
    if (fromIdx < 0 || toIdx < 0 || fromIdx === toIdx) return;
    const [item] = layout.splice(fromIdx, 1);
    layout.splice(toIdx, 0, item);
    setActiveLayout(layout);

    const grid = getAnalyticsGridElement();
    if (!grid) return;
    const fromEl = grid.querySelector(`[data-id="${fromId}"]`);
    const toEl = grid.querySelector(`[data-id="${toId}"]`);
    if (fromEl && toEl && fromEl !== toEl) {
        if (fromIdx < toIdx) toEl.after(fromEl);
        else toEl.before(fromEl);
    }
}

function bindAnalyticsDragDrop(grid) {
    if (!grid || grid.dataset.dragBound === '1') return;
    grid.dataset.dragBound = '1';

    grid.addEventListener('dragstart', (e) => {
        const widget = e.target.closest('.analytics-widget');
        const handle = e.target.closest('.analytics-drag-hint');
        if (!widget || !handle) {
            e.preventDefault();
            return;
        }
        dragWidgetId = widget.dataset.id;
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', dragWidgetId);
        widget.classList.add('is-dragging');
    }, true);

    grid.addEventListener('dragend', (e) => {
        const widget = e.target.closest('.analytics-widget');
        if (widget) widget.classList.remove('is-dragging');
        grid.querySelectorAll('.analytics-widget').forEach(n => n.classList.remove('drag-over'));
        dragWidgetId = null;
    }, true);

    grid.addEventListener('dragover', (e) => {
        const widget = e.target.closest('.analytics-widget');
        if (!widget) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        grid.querySelectorAll('.analytics-widget').forEach(n => n.classList.remove('drag-over'));
        widget.classList.add('drag-over');
    });

    grid.addEventListener('dragleave', (e) => {
        const widget = e.target.closest('.analytics-widget');
        if (widget) widget.classList.remove('drag-over');
    });

    grid.addEventListener('drop', (e) => {
        const widget = e.target.closest('.analytics-widget');
        if (!widget) return;
        e.preventDefault();
        widget.classList.remove('drag-over');
        const fromId = dragWidgetId || e.dataTransfer.getData('text/plain');
        if (fromId && widget.dataset.id) reorderLayoutByDrag(fromId, widget.dataset.id);
        dragWidgetId = null;
    });
}

function formatBucketLabel(raw, bucket) {
    if (!raw) return raw;
    const d = new Date(String(raw).replace(' ', 'T'));
    if (isNaN(d.getTime())) return raw;
    if (bucket === 'hour') return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    if (bucket === 'day') return d.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' });
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

export function getExportChartTheme() {
    return {
        legend: '#334155',
        title: '#0f172a',
        tick: '#64748b',
        border: '#ffffff',
        grid: 'rgba(203, 213, 225, 0.55)',
        background: '#ffffff'
    };
}

function renderDoughnut(canvas, data, widget, theme, key, { forPdf = false } = {}) {
    const stable = data.stable_failures ?? 0;
    const flaky = data.flaky_tests ?? 0;
    return new Chart(canvas.getContext('2d'), {
        type: 'doughnut',
        data: {
            labels: ['Stable Failures', 'Flaky Tests'],
            datasets: [{
                data: [stable, flaky],
                backgroundColor: [widget.color || CHART_PALETTE.doughnut[0], widget.color2 || CHART_PALETTE.doughnut[1]],
                borderColor: '#ffffff',
                borderWidth: 2
            }]
        },
        options: {
            responsive: !forPdf,
            maintainAspectRatio: false,
            animation: !forPdf,
            cutout: forPdf ? '62%' : '70%',
            layout: { padding: forPdf ? 8 : 0 },
            interaction: forPdf ? undefined : { mode: 'nearest', intersect: true },
            plugins: {
                tooltip: forPdf ? { enabled: false } : { enabled: true },
                legend: {
                    position: forPdf ? 'right' : 'bottom',
                    align: 'center',
                    labels: {
                        color: theme.legend,
                        usePointStyle: true,
                        font: { size: forPdf ? 11 : 12 },
                        padding: forPdf ? 14 : 12
                    }
                },
                title: {
                    display: !forPdf,
                    text: widget.title || 'Failure taxonomy',
                    color: theme.title,
                    font: { size: 12, weight: '600' }
                }
            }
        }
    });
}

function renderEvolution(canvas, data, widget, theme, key, { forPdf = false } = {}) {
    const evo = data.evolution || [];
    const bucket = data.evolution_bucket || 'week';
    const labels = evo.map(e => formatBucketLabel(e.week_start, bucket));
    return new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {
            labels,
            datasets: [
                { label: 'Total', type: 'bar', data: evo.map(e => e.total_failures), backgroundColor: widget.color || 'rgba(59,130,246,0.75)', yAxisID: 'y', borderRadius: forPdf ? 3 : 0 },
                { label: 'Flaky', type: 'bar', data: evo.map(e => e.flaky_count), backgroundColor: widget.color2 || 'rgba(217,119,6,0.75)', yAxisID: 'y', borderRadius: forPdf ? 3 : 0 },
                { label: 'MTTR (min)', type: 'line', data: evo.map(e => Math.round(e.mttr)), borderColor: widget.color3 || '#059669', backgroundColor: 'rgba(5,150,105,0.08)', fill: forPdf, tension: 0.35, yAxisID: 'y1', borderWidth: 2.5, pointRadius: forPdf ? 4 : 0, pointHoverRadius: forPdf ? 4 : 5 }
            ]
        },
        options: {
            responsive: !forPdf,
            maintainAspectRatio: false,
            animation: false,
            interaction: { mode: 'index', intersect: false },
            plugins: {
                legend: {
                    position: 'top',
                    align: 'start',
                    labels: { color: theme.legend, usePointStyle: true, font: { size: forPdf ? 10 : 11 }, padding: 12 }
                }
            },
            scales: {
                x: {
                    ticks: { color: theme.tick, font: { size: forPdf ? 9 : 10 }, maxRotation: forPdf ? 25 : 0 },
                    grid: { display: false }
                },
                y: {
                    ticks: { color: theme.tick, font: { size: forPdf ? 9 : 10 }, stepSize: 1 },
                    grid: { color: theme.grid },
                    title: forPdf ? { display: true, text: 'Incidents', color: theme.tick, font: { size: 9 } } : undefined
                },
                y1: {
                    position: 'right',
                    ticks: { color: widget.color3 || '#059669', font: { size: forPdf ? 9 : 10 } },
                    grid: { drawOnChartArea: false },
                    title: forPdf ? { display: true, text: 'MTTR (min)', color: widget.color3 || '#059669', font: { size: 9 } } : undefined
                }
            }
        }
    });
}

/** High-resolution chart renders for PDF (light theme, off-screen). Does not touch dashboard chart instances. */
export async function renderChartsForPdfExport(metricsData, layout) {
    const theme = getExportChartTheme();
    const chartWidgets = layout.filter(w => w.type !== 'metric');
    if (chartWidgets.length === 0) return {};

    let root = document.getElementById('pdf-chart-export-root');
    if (!root) {
        root = document.createElement('div');
        root.id = 'pdf-chart-export-root';
        root.setAttribute('aria-hidden', 'true');
        root.style.cssText = 'position:fixed;left:-10000px;top:0;width:0;height:0;overflow:hidden;';
        document.body.appendChild(root);
    }

    const sizes = { doughnut: { w: 640, h: 280 }, evolution: { w: 920, h: 320 } };
    root.innerHTML = chartWidgets.map(w => {
        const s = sizes[w.type] || sizes.evolution;
        return `<div style="width:${s.w}px;height:${s.h}px;background:#fff;"><canvas id="pdf-chart-${w.id}" width="${s.w}" height="${s.h}"></canvas></div>`;
    }).join('');

    await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));

    const images = {};
    chartWidgets.forEach(w => {
        const canvas = document.getElementById(`pdf-chart-${w.id}`);
        if (!canvas) return;
        const key = `analytics-${w.id}`;
        let chart;
        if (w.type === 'doughnut') chart = renderDoughnut(canvas, metricsData, w, theme, key, { forPdf: true });
        else if (w.type === 'evolution') chart = renderEvolution(canvas, metricsData, w, theme, key, { forPdf: true });
        if (chart?.toBase64Image) images[key] = chart.toBase64Image('image/png', 1.5);
        if (chart) chart.destroy();
    });

    root.innerHTML = '';
    return images;
}

function getAnalyticsGridElement() {
    if (editMode) {
        return document.getElementById('analytics-grid-modal') || document.getElementById('analytics-grid');
    }
    return document.getElementById('analytics-grid');
}

export function renderAnalyticsGrid(metricsData, { isExport = false, theme } = {}) {
    lastMetricsData = metricsData;
    const grid = getAnalyticsGridElement();
    if (!grid) return;

    destroyLayoutCharts();
    delete grid.dataset.dragBound;

    const layout = getActiveLayout();
    grid.innerHTML = layout.map(w => {
        const editControls = editMode ? `
            <div class="analytics-widget-edit">
                <span class="analytics-drag-hint" draggable="true" title="Drag to reorder">⠿</span>
                <button type="button" class="btn btn-secondary btn-sm btn-danger" onclick="removeAnalyticsWidget('${w.id}')">×</button>
            </div>` : '';

        if (w.type === 'metric') {
            const def = METRIC_CATALOG[w.metric] || { label: w.metric, hint: '' };
            return `<article class="${spanClass(w.span)} analytics-widget analytics-widget-metric" data-id="${w.id}">
                ${editControls}
                <h4>${w.title || def.label}</h4>
                <div class="analytics-metric-value" style="color:${w.color || 'var(--accent)'}">${metricValue(metricsData, w.metric)}</div>
                <small>${def.hint}</small>
            </article>`;
        }
        const h = w.type === 'evolution' ? 300 : 220;
        return `<article class="${spanClass(w.span)} analytics-widget analytics-widget-chart" data-id="${w.id}">
            ${editControls}
            <h4>${w.title || 'Chart'}</h4>
            <div class="analytics-chart-wrap" style="height:${h}px"><canvas id="chart-${w.id}"></canvas></div>
        </article>`;
    }).join('');

    layout.forEach(w => {
        if (w.type === 'metric') return;
        const canvas = document.getElementById(`chart-${w.id}`);
        if (!canvas) return;
        const key = `analytics-${w.id}`;
        let chart;
        if (w.type === 'doughnut') chart = renderDoughnut(canvas, metricsData, w, theme, key);
        else if (w.type === 'evolution') chart = renderEvolution(canvas, metricsData, w, theme, key);
        if (chart) chartInstances[key] = chart;
    });

    if (editMode) {
        grid.querySelectorAll('.analytics-widget').forEach(el => {
            el.classList.add('analytics-widget-draggable');
        });
        bindAnalyticsDragDrop(grid);
    }
}

/** Returns map widgetId -> base64 chart image for PDF export */
export function getAnalyticsChartImages() {
    const out = {};
    Object.entries(chartInstances).forEach(([key, chart]) => {
        if (chart?.toBase64Image) out[key] = chart.toBase64Image();
    });
    return out;
}

export function resetAnalyticsLayoutDefault() {
    const layout = JSON.parse(JSON.stringify(DEFAULT_ANALYTICS_LAYOUT));
    if (editMode && draftLayout) draftLayout = layout;
    else currentLayout = layout;
    if (lastMetricsData) renderAnalyticsGrid(lastMetricsData, { isExport: false });
}

export function syncAnalyticsWidgetForm() {
    const type = document.getElementById('aw-type')?.value || 'metric';
    const metricWrap = document.getElementById('aw-metric-wrap');
    if (metricWrap) metricWrap.style.display = type === 'metric' ? 'flex' : 'none';
}
