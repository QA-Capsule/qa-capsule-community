/**
 * Self-Healing Hub
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canManageHealing, canDeleteIncidents } from './roles.js';

let currentRole = '';
const selected = new Set(); // incident ids currently checked

/* ── Framework / category metadata ─────────────────────────────────────── */
const FW = {
    robotframework: { label: 'Robot',      color: '#e67e22' },
    playwright:     { label: 'Playwright', color: '#7c5cfc' },
    cypress:        { label: 'Cypress',    color: '#17a974' },
    selenium:       { label: 'Selenium',   color: '#2e86c1' },
    pytest:         { label: 'Pytest',     color: '#0097a7' },
    newman:         { label: 'Newman',     color: '#c0392b' },
    jest:           { label: 'Jest',       color: '#c77b00' },
    junit:          { label: 'JUnit',      color: '#6d4c41' },
};
const FW_DEF = { label: 'Unknown', color: '#7f8c8d' };

const CAT = {
    timeout:        { label: 'Timeout',   color: '#d4a017' },
    locator:        { label: 'Locator',   color: '#c0392b' },
    assertion:      { label: 'Assertion', color: '#e67e22' },
    network:        { label: 'Network',   color: '#2980b9' },
    script_failure: { label: 'Script',    color: '#8e44ad' },
    unknown:        { label: 'Unknown',   color: '#95a5a6' },
};

const fw  = r => FW[(r || '').toLowerCase().replace(/[^a-z]/g, '')] || FW_DEF;
const cat = r => CAT[(r || '').toLowerCase().replace(/[^a-z_]/g, '')] || CAT.unknown;
const esc = s => String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
const ago = s => { if (!s) return ''; const m = Math.floor((Date.now() - new Date(s)) / 60000); return m < 2 ? 'just now' : m < 60 ? `${m}m ago` : Math.floor(m / 60) < 24 ? `${Math.floor(m/60)}h ago` : `${Math.floor(m/1440)}d ago`; };

/* ── Icons SVG ────────────────────────────────────────────────────────── */
const ICON = {
    fix:     `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>`,
    context: `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`,
    copy:    `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`,
    trash:   `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/><path d="M9 6V4h6v2"/></svg>`,
    refresh: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-.08-1.96"/></svg>`,
    arrow:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>`,
    close:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
    check:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="20 6 9 17 4 12"/></svg>`,
    star:    `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z"/></svg>`,
};

/* ── Init ─────────────────────────────────────────────────────────────── */
window.loadHealingView = loadHealingView;
export function loadHealingView() {
    if (!document.getElementById('healing-insights-body')) return;
    currentRole = parseJwt(localStorage.getItem('sre-jwt'))?.role || '';
    const sel = document.getElementById('healing-project-filter');
    loadHealingProjects(sel);
    loadHealingInsights();
    loadLocatorInterventions();
    _loadAIChip();
    if (sel && !sel.dataset.bound) {
        sel.dataset.bound = '1';
        sel.addEventListener('change', () => { loadHealingInsights(); loadLocatorInterventions(); });
    }
}

async function _loadAIChip() {
    const chip = document.getElementById('healing-ai-chip');
    if (!chip) return;
    try {
        const { ok, data } = await parseApiJson(await fetchWithAuth('/api/ai/status'));
        if (ok && data?.enabled && data.provider !== 'disabled') {
            chip.innerHTML = `<span class="hc-dot hc-dot--live"></span>${esc(data.provider.toUpperCase())} · ${esc(data.model || '—')}`;
            chip.className = 'hc-chip hc-chip--on';
        } else {
            chip.innerHTML = `<span class="hc-dot"></span>AI off`;
            chip.className = 'hc-chip hc-chip--off';
        }
    } catch { if (chip) { chip.textContent = 'AI —'; chip.className = 'hc-chip hc-chip--off'; } }
}

async function loadHealingProjects(sel) {
    if (!sel) return;
    const { ok, data } = await parseApiJson(await fetchWithAuth('/api/my-projects'));
    sel.innerHTML = '<option value="">All gateways</option>';
    if (ok) (Array.isArray(data) ? data : []).forEach(p =>
        sel.innerHTML += `<option value="${esc(p.name)}">${esc(p.name)}</option>`
    );
}

/* ── Load failures ────────────────────────────────────────────────────── */
export async function loadHealingInsights() {
    const body = document.getElementById('healing-insights-body');
    if (!body) return;
    const project = document.getElementById('healing-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=100` : '?limit=100';

    body.innerHTML = `<div class="hc-loading">Loading…</div>`;

    const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/healing/insights${q}`));
    if (!ok) { body.innerHTML = `<div class="hc-empty">Failed to load failures.</div>`; updateStats([]); return; }
    const rows = Array.isArray(data) ? data : [];
    updateStats(rows);
    selected.clear();
    _updateBulkBar();

    if (!rows.length) {
        body.innerHTML = `<div class="hc-empty">No open failures — all tests are passing.</div>`;
        return;
    }
    body.innerHTML = rows.map(r => renderCard(r)).join('');
}
window.loadHealingInsights = loadHealingInsights;

function updateStats(rows) {
    _count('healing-stat-total',         rows.length);
    _count('healing-stat-categorized',   rows.filter(r => r.error_category && r.error_category !== 'unknown').length);
    _count('healing-stat-mcp-ready',     rows.length);
    _count('healing-stat-interventions', rows.filter(r => r.mcp_healed).length);
}

function _count(id, n) {
    const el = document.getElementById(id);
    if (!el) return;
    const from = parseInt(el.textContent) || 0;
    if (from === n) { el.textContent = n; return; }
    const t0 = performance.now();
    const tick = now => { const p = Math.min((now - t0) / 500, 1); el.textContent = Math.round(from + (n - from) * p); if (p < 1) requestAnimationFrame(tick); };
    requestAnimationFrame(tick);
}

/* ── Render one card ──────────────────────────────────────────────────── */
function renderCard(r) {
    const id     = Number(r.incident_id) || 0;
    const f      = fw(r.framework || r.project_name);
    const c      = cat(r.error_category);
    const healed = !!r.mcp_healed;
    const canMgr = canManageHealing(currentRole);
    const canDel = canDeleteIncidents(currentRole);

    // inline tinted badge style using the framework/category color
    const fwStyle  = `color:${f.color};border-color:${f.color}30;background:${f.color}18`;
    const catStyle = `color:${c.color};border-color:${c.color}30;background:${c.color}18`;

    return `
<article class="hc-card${healed ? ' hc-card--healed' : ''}" id="hc-card-${id}" data-id="${id}">
    <div class="hc-card__top">
        <!-- Checkbox -->
        <label class="hc-cb-cell" title="Select failure">
            <input type="checkbox" class="hc-cb" value="${id}" onchange="window._onCardCheck(${id},this.checked)">
        </label>

        <!-- Color stripe -->
        <div class="hc-stripe" style="background:${f.color}"></div>

        <!-- Main body -->
        <div class="hc-body">
            <!-- Meta row -->
            <div class="hc-row-meta">
                <span class="hc-badge" style="${fwStyle}">${esc(f.label)}</span>
                <span class="hc-badge" style="${catStyle}">${esc(c.label)}</span>
                ${healed ? `<span class="hc-badge hc-badge--healed">${ICON.check}&nbsp;Healed</span>` : ''}
                <span class="hc-id">#${id}</span>
            </div>

            <!-- Test name -->
            <h3 class="hc-test-name">${esc(r.test_name || 'Unknown test')}</h3>

            <!-- Error description -->
            <p class="hc-test-desc">${esc(r.summary || r.error_message || 'No description available.')}</p>

            <!-- Action bar -->
            <div class="hc-card__actions">
                ${canMgr ? `
                <button class="hc-btn-fix" title="AI scans live page and proposes the corrected selector" onclick="window.openProposeFix(${id})">
                    ${ICON.fix} Propose Fix
                </button>
                <div class="hc-icon-group">
                    <button class="hc-icon-btn" title="View context" onclick="window.openHealingContext(${id})">${ICON.context}</button>
                    <button class="hc-icon-btn" title="Copy MCP prompt" onclick="window.copyMCPPrompt(${id})">${ICON.copy}</button>
                    ${canDel ? `<button class="hc-icon-btn hc-icon-btn--danger" title="Delete failure" onclick="window.deleteHealingFailure(${id})">${ICON.trash}</button>` : ''}
                </div>` : (canDel ? `
                <div class="hc-icon-group">
                    <button class="hc-icon-btn hc-icon-btn--danger" title="Delete failure" onclick="window.deleteHealingFailure(${id})">${ICON.trash}</button>
                </div>` : '')}
            </div>
        </div>
    </div>

    <!-- Fix proposal panel (injected by openProposeFix) -->
    <div id="fix-proposal-${id}" class="hc-panel" style="display:none"></div>
</article>`;
}

/* ── Checkbox / bulk select ───────────────────────────────────────────── */
window._onCardCheck = (id, checked) => {
    if (checked) selected.add(id); else selected.delete(id);
    _updateBulkBar();
    _syncSelectAll();
};

window._selectAll = checked => {
    document.querySelectorAll('.hc-cb').forEach(cb => {
        cb.checked = checked;
        const id = parseInt(cb.value);
        if (checked) selected.add(id); else selected.delete(id);
    });
    _updateBulkBar();
};

function _updateBulkBar() {
    const bar = document.getElementById('hc-bulk-bar');
    const cnt = document.getElementById('hc-bulk-count');
    if (!bar) return;
    if (selected.size > 0) {
        bar.style.display = 'flex';
        if (cnt) cnt.textContent = `${selected.size} selected`;
    } else {
        bar.style.display = 'none';
    }
}

function _syncSelectAll() {
    const allCbs = document.querySelectorAll('.hc-cb');
    const sa = document.getElementById('hc-select-all');
    if (!sa || !allCbs.length) return;
    const allChecked = [...allCbs].every(cb => cb.checked);
    const someChecked = [...allCbs].some(cb => cb.checked);
    sa.checked = allChecked;
    sa.indeterminate = someChecked && !allChecked;
}

window.bulkDelete = async () => {
    if (!selected.size) return;
    const ids = [...selected].join(',');
    window.showConfirmModal('Delete failures?', `Delete ${selected.size} selected failure(s)?`, 'danger', async () => {
        const { ok } = await parseApiJson(await fetchWithAuth(`/api/incidents?ids=${ids}`, { method: 'DELETE' }));
        if (!ok) { notify('Failed to delete.', 'error'); return; }
        selected.forEach(id => document.getElementById(`hc-card-${id}`)?.remove());
        selected.clear();
        _updateBulkBar();
        notify(`${ids.split(',').length} failure(s) deleted.`, 'success');
        loadHealingInsights();
    });
};

/* ── Delete single ────────────────────────────────────────────────────── */
export function deleteHealingFailure(id) {
    if (!canDeleteIncidents(currentRole)) { notify('Manager or Lead role required.', 'error'); return; }
    window.showConfirmModal('Delete failure?', `Delete INC #${id}?`, 'danger', async () => {
        const { ok } = await parseApiJson(await fetchWithAuth(`/api/incidents?ids=${id}`, { method: 'DELETE' }));
        if (!ok) { notify('Failed to delete.', 'error'); return; }
        document.getElementById(`hc-card-${id}`)?.remove();
        selected.delete(id); _updateBulkBar();
        notify(`INC #${id} deleted.`, 'success');
        loadHealingInsights();
    });
}
window.deleteHealingFailure = deleteHealingFailure;

/* ── Context / copy prompt ────────────────────────────────────────────── */
export async function openHealingContext(id) {
    const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/incidents/${id}/healing/context`));
    if (!ok || !data) { notify('Failed to load context.', 'error'); return; }
    window.__lastHealingContext = data;
    const info = [data.error_category && `Category: ${data.error_category}`, data.selector_hint && `Selector: ${data.selector_hint}`].filter(Boolean).join(' · ');
    notify(info || 'Context loaded.', 'info');
}
window.openHealingContext = openHealingContext;

export async function copyMCPPrompt(id) {
    let ctx = window.__lastHealingContext;
    if (!ctx || ctx.incident_id !== id) {
        const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/incidents/${id}/healing/context`));
        if (!ok || !data) { notify('Failed to build prompt.', 'error'); return; }
        ctx = data;
    }
    const text = ctx.mcp_prompt || `Use get_incident_context with incident_id=${id} and propose a minimal fix.`;
    try { await navigator.clipboard.writeText(text); notify('Prompt copied.', 'success'); } catch { notify(text, 'info'); }
}
window.copyMCPPrompt = copyMCPPrompt;

/* ── Propose fix ──────────────────────────────────────────────────────── */
export async function openProposeFix(id) {
    const panel = document.getElementById(`fix-proposal-${id}`);
    if (!panel) return;

    // Toggle
    if (panel.style.display !== 'none') { panel.style.display = 'none'; return; }
    panel.style.display = 'block';
    panel.innerHTML = `<div class="hc-panel__loading">${ICON.refresh} <span>Fetching live page DOM and querying AI…</span></div>`;

    try {
        const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/incidents/${id}/healing/propose`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ file_content: '' }),
        }));

        if (!ok || !data) {
            panel.innerHTML = `<div class="hc-panel__error">Failed to get a fix proposal. Verify AI configuration.</div>`;
            return;
        }

        const pct   = data.confidence ? Math.round(data.confidence * 100) : 0;
        const conf  = pct >= 70 ? 'high' : pct >= 40 ? 'mid' : 'low';
        const domOk = data.dom_used;

        // Locator diff row
        let locRow = '';
        if (data.original_locator) {
            locRow = `
            <div class="hc-locrow">
                <div class="hc-locrow__col">
                    <span class="hc-loclbl">Broken selector</span>
                    <code class="hc-loc hc-loc--bad">${esc(data.original_locator)}</code>
                </div>
                <span class="hc-arrow">${ICON.arrow}</span>
                <div class="hc-locrow__col">
                    <span class="hc-loclbl">Fixed selector${domOk ? ' (from live DOM)' : ' (estimated)'}</span>
                    <code class="hc-loc hc-loc--ok">${esc(data.healed_locator || '—')}</code>
                </div>
                <span class="hc-conf hc-conf--${conf}">${pct}% confidence</span>
            </div>`;
        }

        // Explanation
        const expl = data.explanation
            ? `<div class="hc-panel__expl"><strong>Explanation</strong><p>${esc(data.explanation)}</p></div>`
            : '';

        // Code
        const code = data.code
            ? `<div class="hc-panel__code"><div class="hc-panel__code-head"><strong>Code fix</strong><button class="hc-panel__copy" onclick="window._copyFix(${id})">${ICON.copy} Copy</button></div><pre>${esc(data.code)}</pre></div>`
            : '';

        panel.innerHTML = `
        <div class="hc-panel__inner">
            <div class="hc-panel__head">
                <span class="hc-panel__title">${ICON.star} AI Fix Proposal</span>
                <button class="hc-panel__close" onclick="document.getElementById('fix-proposal-${id}').style.display='none'">${ICON.close}</button>
            </div>
            ${locRow}
            ${expl}
            ${code}
        </div>`;

        window[`__fixCode_${id}`] = data.code || '';
    } catch (e) {
        panel.innerHTML = `<div class="hc-panel__error">${esc(String(e))}</div>`;
    }
}
window.openProposeFix = openProposeFix;
window._copyFix = id => {
    const c = window[`__fixCode_${id}`] || '';
    navigator.clipboard.writeText(c).then(() => notify('Copied.', 'success'), () => notify(c, 'info'));
};

/* ── Locator interventions ────────────────────────────────────────────── */
export async function loadLocatorInterventions() {
    const el = document.getElementById('healing-interventions-body');
    if (!el) return;
    const project = document.getElementById('healing-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=50` : '?limit=50';
    el.innerHTML = `<div class="hc-loading">Loading…</div>`;

    const { ok, data } = await parseApiJson(await fetchWithAuth(`/api/healing/locator-interventions${q}`));
    if (!ok) { el.innerHTML = `<div class="hc-empty">Failed to load.</div>`; return; }
    const rows = Array.isArray(data) ? data : [];
    _count('healing-stat-interventions', rows.length);

    if (!rows.length) { el.innerHTML = `<div class="hc-empty">No locator interventions yet. They appear here after the CI Healing Gate runs.</div>`; return; }
    el.innerHTML = rows.map(h => renderItv(h)).join('');
}
window.loadLocatorInterventions = loadLocatorInterventions;

function renderItv(h) {
    const f   = fw(h.framework);
    const pct = h.confidence ? Math.round(h.confidence * 100) : 0;
    const conf = pct >= 70 ? 'high' : pct >= 40 ? 'mid' : 'low';

    return `
<article class="hc-itv">
    <div class="hc-itv__head">
        <span class="hc-badge" style="color:${f.color};border-color:${f.color}20;background:${f.color}14">${esc(f.label)}</span>
        <span class="hc-itv__name">${esc(h.test_name || `INC #${h.incident_id}`)}</span>
        <span class="hc-conf hc-conf--${conf}">${pct}% confidence</span>
        <span class="hc-ago">${ago(h.created_at)}</span>
    </div>
    <div class="hc-locrow">
        <div class="hc-locrow__col">
            <span class="hc-loclbl">Broken</span>
            <code class="hc-loc hc-loc--bad">${esc(h.original_locator || '—')}</code>
        </div>
        <span class="hc-arrow">${ICON.arrow}</span>
        <div class="hc-locrow__col">
            <span class="hc-loclbl">Healed</span>
            <code class="hc-loc hc-loc--ok">${esc(h.healed_locator || 'pending')}</code>
        </div>
    </div>
    ${h.explanation ? `<p class="hc-itv__note">${esc(h.explanation)}</p>` : ''}
</article>`;
}
