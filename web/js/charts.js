/**
 * Chart Studio — QCL editor, saved charts library, pin to Dashboard / FinOps
 */
import { fetchWithAuth, parseApiJson, describeApiFailure } from './api.js';
import { notify, showConfirmModal } from './ui.js';
import { renderChartOnCanvas, destroyChartInstance } from './chart-widgets.js';
import { setupAutocomplete } from './autocomplete.js';
import { canAccessFinOps } from './roles.js';
import { parseJwt } from './api.js';

let chartLibraryAcInit = false;

const STUDIO_KEY = 'studio-preview';
let savedCharts = [];
let activeChartId = null;

const DEFAULT_QCL = `CHART line "Weekly incident volume"
METRIC incidents
RANGE 12w
GROUP week`;

export function loadChartsView() {
    const editor = document.getElementById('qcl-editor');
    if (editor && !editor.value.trim()) editor.value = DEFAULT_QCL;

    loadChartReference();
    initChartLibraryAutocomplete();
    applyChartStudioFinopsUi();
    refreshSavedChartsLibrary();
    setTimeout(() => runChartQuery(), 300);
}

function applyChartStudioFinopsUi() {
    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    if (canAccessFinOps(role)) return;
    const pinFin = document.getElementById('pin-finops');
    if (pinFin) pinFin.checked = false;
}

function initChartLibraryAutocomplete() {
    if (chartLibraryAcInit) return;
    const input = document.getElementById('chart-library-search');
    let list = document.getElementById('chart-library-search-ac');
    if (!input) return;
    if (!list) {
        const wrap = document.createElement('div');
        wrap.className = 'ac-wrap';
        input.parentNode.insertBefore(wrap, input);
        wrap.appendChild(input);
        list = document.createElement('div');
        list.id = 'chart-library-search-ac';
        list.className = 'autocomplete-items';
        list.style.display = 'none';
        wrap.appendChild(list);
    }
    chartLibraryAcInit = true;
    setupAutocomplete({
        input,
        list,
        minChars: 0,
        getSuggestions: q => {
            const v = q.toLowerCase();
            return savedCharts
                .filter(c => !v || c.name.toLowerCase().includes(v) || (c.description || '').toLowerCase().includes(v))
                .slice(0, 12)
                .map(c => ({ label: c.name, sublabel: c.description || 'Saved chart', value: c.name }));
        },
        onSelect: item => {
            input.value = item.value;
            filterChartLibrary();
        }
    });
}

function loadChartReference() {
    fetchWithAuth('/api/charts/reference')
        .then(r => parseApiJson(r))
        .then(({ ok, data: ref }) => {
            const el = document.getElementById('qcl-reference-body');
            if (!el || !ok || !ref?.directives) return;
            const metrics = Array.isArray(ref.metrics) ? ref.metrics : [];
            el.innerHTML = `
                <p class="qcl-ref-lead"><strong>QCL v${ref.version || '1'}</strong> — ${ref.directives.join(' · ')}</p>
                <pre class="qcl-ref-example">${escapeHtml(ref.example || '')}</pre>
                <ul class="qcl-ref-metrics">${metrics.map(m =>
                    `<li><code>${m.id}</code><span>${m.description}</span></li>`).join('')}</ul>`;
        })
        .catch(() => {});
}

function escapeHtml(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export async function refreshSavedChartsLibrary() {
    const list = document.getElementById('chart-library-list');
    if (!list) return;

    try {
        const res = await fetchWithAuth('/api/charts/saved');
        const { ok, data } = await parseApiJson(res);
        savedCharts = ok && data?.charts ? data.charts : [];
    } catch {
        savedCharts = [];
    }

    const q = (document.getElementById('chart-library-search')?.value || '').toLowerCase();
    const filtered = savedCharts.filter(c =>
        !q || c.name.toLowerCase().includes(q) || (c.description || '').toLowerCase().includes(q));

    if (filtered.length === 0) {
        list.innerHTML = `<div class="chart-library-empty">No saved charts yet.<br>Write a query and click <strong>Save</strong>.</div>`;
        return;
    }

    list.innerHTML = filtered.map(c => `
        <button type="button" class="chart-library-item ${activeChartId === c.id ? 'active' : ''}"
            onclick="loadSavedChart(${c.id})">
            <span class="chart-library-item-title">${escapeHtml(c.name)}</span>
            <span class="chart-library-item-meta">
                ${c.pin_dashboard ? '<span class="chart-pin-badge">Dashboard</span>' : ''}
                ${c.pin_finops ? '<span class="chart-pin-badge finops">FinOps</span>' : ''}
            </span>
        </button>
    `).join('');
}

export function filterChartLibrary() {
    refreshSavedChartsLibrary();
}

export function newChartDraft() {
    activeChartId = null;
    document.getElementById('chart-save-name').value = '';
    document.getElementById('chart-save-desc').value = '';
    document.getElementById('pin-dashboard').checked = false;
    document.getElementById('pin-finops').checked = false;
    const editor = document.getElementById('qcl-editor');
    if (editor) editor.value = DEFAULT_QCL;
    document.getElementById('chart-editor-title').textContent = 'New chart';
    refreshSavedChartsLibrary();
    runChartQuery();
}

export function loadSavedChart(id) {
    const chart = savedCharts.find(c => c.id === id);
    if (!chart) return;
    activeChartId = chart.id;
    document.getElementById('chart-save-name').value = chart.name;
    document.getElementById('chart-save-desc').value = chart.description || '';
    document.getElementById('pin-dashboard').checked = !!chart.pin_dashboard;
    document.getElementById('pin-finops').checked = !!chart.pin_finops;
    document.getElementById('qcl-editor').value = chart.qcl_query;
    document.getElementById('chart-editor-title').textContent = chart.name;
    refreshSavedChartsLibrary();
    runChartQuery();
}

export function runChartQuery() {
    const query = document.getElementById('qcl-editor')?.value?.trim();
    if (!query) return notify('Enter a QCL query', 'error');

    const statusEl = document.getElementById('chart-preview-status');
    if (statusEl) statusEl.textContent = 'Running query…';

    fetchWithAuth('/api/charts/evaluate', { method: 'POST', body: JSON.stringify({ query }) })
        .then(res => parseApiJson(res))
        .then(({ ok, offline, status: httpStatus, data }) => {
            if (!ok) throw new Error(describeApiFailure(httpStatus, offline));
            if (!data) throw new Error('Query failed');
            const canvas = document.getElementById('chart-studio-canvas');
            renderChartOnCanvas(canvas, data, STUDIO_KEY);
            if (statusEl) statusEl.textContent = data.title || 'Preview ready';
        })
        .catch(e => {
            if (statusEl) statusEl.textContent = 'Query failed';
            notify(e.message || 'Chart evaluation failed', 'error');
        });
}

export function saveCurrentChart() {
    const name = document.getElementById('chart-save-name')?.value?.trim();
    const qcl_query = document.getElementById('qcl-editor')?.value?.trim();
    if (!name) return notify('Enter a chart name', 'error');
    if (!qcl_query) return notify('Enter a QCL query', 'error');

    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    const payload = {
        id: activeChartId || 0,
        name,
        description: document.getElementById('chart-save-desc')?.value?.trim() || '',
        qcl_query,
        pin_dashboard: document.getElementById('pin-dashboard')?.checked || false,
        pin_finops: canAccessFinOps(role) && (document.getElementById('pin-finops')?.checked || false)
    };

    const method = activeChartId ? 'PUT' : 'POST';
    fetchWithAuth('/api/charts/saved', { method, body: JSON.stringify(payload) })
        .then(res => parseApiJson(res))
        .then(({ ok, data }) => {
            if (!ok) throw new Error(data?.error || 'Save failed');
            if (!activeChartId && data.id) activeChartId = data.id;
            notify(activeChartId ? 'Chart updated' : 'Chart saved', 'success');
            document.getElementById('chart-editor-title').textContent = name;
            refreshSavedChartsLibrary();
            if (payload.pin_dashboard && window.refreshDashboardPinnedCharts) {
                window.refreshDashboardPinnedCharts();
            }
            if (payload.pin_finops && window.refreshFinOpsPinnedCharts) {
                window.refreshFinOpsPinnedCharts();
            }
        })
        .catch(e => notify(e.message || 'Failed to save chart', 'error'));
}

export function deleteCurrentChart() {
    if (!activeChartId) return notify('Select a saved chart to delete', 'error');
    showConfirmModal('Delete chart?', 'This removes the chart from your library and unpins it everywhere.', 'danger', () => {
        fetchWithAuth(`/api/charts/saved?id=${activeChartId}`, { method: 'DELETE' })
            .then(res => {
                if (!res.ok) throw new Error();
                notify('Chart deleted', 'success');
                activeChartId = null;
                newChartDraft();
                if (window.refreshDashboardPinnedCharts) window.refreshDashboardPinnedCharts();
                if (window.refreshFinOpsPinnedCharts) window.refreshFinOpsPinnedCharts();
            })
            .catch(() => notify('Delete failed', 'error'));
    });
}

export function toggleQclReference() {
    const body = document.getElementById('qcl-reference-body');
    const btn = document.getElementById('qcl-reference-toggle');
    if (!body) return;
    const open = body.style.display !== 'none';
    body.style.display = open ? 'none' : 'block';
    if (btn) btn.textContent = open ? 'Show reference' : 'Hide reference';
}

export function insertQclSnippet(snippet) {
    const editor = document.getElementById('qcl-editor');
    if (editor) {
        editor.value = snippet;
        editor.focus();
        runChartQuery();
    }
}
