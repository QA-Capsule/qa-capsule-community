/**
 * Self-Healing Hub — framework-agnostic failure context for MCP agents.
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canManageHealing } from './roles.js';

let currentRole = '';

export function loadHealingView() {
    const listEl = document.getElementById('healing-insights-body');
    const projectSel = document.getElementById('healing-project-filter');
    if (!listEl) return;

    currentRole = parseJwt(localStorage.getItem('sre-jwt'))?.role || '';

    loadHealingProjects(projectSel);
    loadHealingInsights();
    loadLocatorInterventions();

    if (projectSel && !projectSel.dataset.bound) {
        projectSel.dataset.bound = '1';
        projectSel.addEventListener('change', () => {
            loadHealingInsights();
            loadLocatorInterventions();
        });
    }

    const copyBtn = document.getElementById('healing-copy-mcp-url');
    if (copyBtn && !copyBtn.dataset.bound) {
        copyBtn.dataset.bound = '1';
        copyBtn.addEventListener('click', () => copyMCPSetup());
    }
}

async function loadHealingProjects(select) {
    if (!select) return;
    const res = await fetchWithAuth('/api/my-projects');
    const { ok, data } = await parseApiJson(res);
    select.innerHTML = '<option value="">All gateways</option>';
    if (!ok) return;
    (Array.isArray(data) ? data : []).forEach(p => {
        select.innerHTML += `<option value="${escapeHtml(p.name)}">${escapeHtml(p.name)}</option>`;
    });
}

export async function loadHealingInsights() {
    const listEl = document.getElementById('healing-insights-body');
    if (!listEl) return;
    const project = document.getElementById('healing-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=100` : '?limit=100';
    listEl.innerHTML = '<p class="modern-data-grid__loading">Loading failures…</p>';
    const res = await fetchWithAuth(`/api/healing/insights${q}`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        listEl.innerHTML = '<p class="modern-data-grid__empty">Failed to load healing insights.</p>';
        updateHealingStats([]);
        return;
    }
    const rows = Array.isArray(data) ? data : [];
    updateHealingStats(rows);
    if (!rows.length) {
        listEl.innerHTML = '<p class="modern-data-grid__empty">No open failures. Ingest test results via webhook or JUnit upload.</p>';
        return;
    }
    listEl.innerHTML = rows.map(r => renderHealingCard(r)).join('');
}

function updateHealingStats(rows) {
    const set = (id, v) => {
        const el = document.getElementById(id);
        if (el) el.textContent = String(v);
    };
    const total = rows.length;
    const categorized = rows.filter(r => r.error_category && r.error_category !== 'unknown').length;
    const healed = rows.filter(r => r.mcp_healed).length;
    set('healing-stat-total', total);
    set('healing-stat-categorized', categorized);
    set('healing-stat-mcp-ready', total);
    set('healing-stat-interventions', healed);
}

function renderHealingCard(r) {
    const category = escapeHtml(r.error_category || 'unknown');
    const incidentId = Number(r.incident_id) || 0;
    const mcpBadge = r.mcp_healed
        ? `<span class="intel-badge intel-badge--healed" title="Le MCP Healing Gate a enregistré une intervention locator pour ce test">MCP Healed</span>`
        : '';
    const actions = canManageHealing(currentRole)
        ? `<button type="button" class="btn-secondary btn-sm" onclick="window.openHealingContext(${incidentId})">Context</button>
           <button type="button" class="btn-secondary btn-sm" onclick="window.copyMCPPrompt(${incidentId})">Copy MCP prompt</button>`
        : `<button type="button" class="btn-secondary btn-sm" onclick="window.openHealingContext(${incidentId})">Context</button>`;
    return `
        <article class="data-card rca-insight-card${r.mcp_healed ? ' rca-insight-card--healed' : ''}" role="listitem">
            <header class="rca-insight-card__head">
                <div class="rca-insight-card__title-wrap">
                    <h3 class="rca-insight-card__title">${escapeHtml(r.test_name || 'Unknown test')} ${mcpBadge}</h3>
                    <p class="rca-insight-card__meta">
                        ${escapeHtml(r.project_name || '—')}
                        <span class="rca-insight-card__status">${category}</span>
                        <span class="rca-insight-card__status">INC #${incidentId}</span>
                    </p>
                </div>
                <div style="display:flex;gap:6px;flex-shrink:0;">${actions}</div>
            </header>
            <p class="rca-insight-card__body">${escapeHtml(r.summary || 'Open failure — use MCP get_incident_context to self-heal.')}</p>
        </article>
    `;
}

window.loadHealingInsights = loadHealingInsights;

// ── Locator Interventions ─────────────────────────────────────────────────────

export async function loadLocatorInterventions() {
    const el = document.getElementById('healing-interventions-body');
    if (!el) return;
    const project = document.getElementById('healing-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=50` : '?limit=50';
    el.innerHTML = '<p class="modern-data-grid__loading">Loading interventions…</p>';
    const res = await fetchWithAuth(`/api/healing/locator-interventions${q}`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        el.innerHTML = '<p class="modern-data-grid__empty">Impossible de charger les interventions locator.</p>';
        return;
    }
    const rows = Array.isArray(data) ? data : [];
    // Update KPI with precise count from DB
    const kpiEl = document.getElementById('healing-stat-interventions');
    if (kpiEl) kpiEl.textContent = String(rows.length);
    if (!rows.length) {
        el.innerHTML = '<p class="modern-data-grid__empty">Aucune intervention locator. Le MCP Healing Gate notifiera ici lors de la prochaine failure de sélecteur.</p>';
        return;
    }
    el.innerHTML = rows.map(h => renderLocatorInterventionCard(h)).join('');
}

function renderLocatorInterventionCard(h) {
    const fw = escapeHtml(h.framework || 'unknown');
    const confidence = h.confidence ? Math.round(h.confidence * 100) + '%' : '—';
    const confidenceClass = h.confidence >= 0.7 ? 'healed' : h.confidence >= 0.4 ? 'warn' : 'unknown';
    const ago = formatRelativeTime(h.created_at);
    const agentLabel = escapeHtml(h.agent_source || 'mcp_gate');
    return `
        <article class="data-card rca-insight-card locator-intervention-card" role="listitem">
            <header class="rca-insight-card__head">
                <div class="rca-insight-card__title-wrap">
                    <h3 class="rca-insight-card__title">
                        <span class="intel-badge intel-badge--healed" style="margin-right:6px;vertical-align:middle;">MCP</span>
                        ${escapeHtml(h.test_name || 'INC #' + h.incident_id)}
                    </h3>
                    <p class="rca-insight-card__meta">
                        INC #${Number(h.incident_id) || '—'}
                        <span class="rca-insight-card__status">${fw}</span>
                        <span class="rca-insight-card__status locator-confidence--${confidenceClass}">confidence ${confidence}</span>
                        <span style="margin-left:8px;">${ago}</span>
                    </p>
                </div>
                <span class="intel-badge intel-badge--manual" style="flex-shrink:0;">${agentLabel}</span>
            </header>
            <div class="locator-intervention-card__locators">
                <code class="locator-intervention-card__original" title="Locator original (cassé)">${escapeHtml(h.original_locator || '—')}</code>
                <span class="locator-intervention-card__arrow" aria-hidden="true">→</span>
                <code class="locator-intervention-card__healed" title="Locator suggéré par le MCP">${escapeHtml(h.healed_locator || 'à définir par l\'agent')}</code>
            </div>
            ${h.explanation ? `<p class="rca-insight-card__body" style="margin-top:8px;">${escapeHtml(h.explanation)}</p>` : ''}
        </article>
    `;
}

function formatRelativeTime(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (isNaN(d)) return '';
    const diffMs = Date.now() - d.getTime();
    const diffMin = Math.floor(diffMs / 60000);
    if (diffMin < 2) return 'just now';
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffH = Math.floor(diffMin / 60);
    if (diffH < 24) return `${diffH}h ago`;
    return `${Math.floor(diffH / 24)}d ago`;
}

window.loadLocatorInterventions = loadLocatorInterventions;

export async function openHealingContext(incidentId) {
    const res = await fetchWithAuth(`/api/incidents/${incidentId}/healing/context`);
    const { ok, data } = await parseApiJson(res);
    if (!ok || !data) {
        notify('Failed to load healing context.', 'error');
        return;
    }
    const lines = [
        data.error_category ? `Category: ${data.error_category}` : '',
        data.selector_hint ? `Selector: ${data.selector_hint}` : '',
        ...(Array.isArray(data.suggested_actions) ? data.suggested_actions.map(a => `• ${a}`) : []),
        data.mcp_prompt || ''
    ].filter(Boolean);
    notify(lines.slice(0, 2).join(' — ') || 'Context loaded.', 'success');
    window.__lastHealingContext = data;
}

export async function copyMCPPrompt(incidentId) {
    let ctx = window.__lastHealingContext;
    if (!ctx || ctx.incident_id !== incidentId) {
        const res = await fetchWithAuth(`/api/incidents/${incidentId}/healing/context`);
        const { ok, data } = await parseApiJson(res);
        if (!ok || !data) {
            notify('Failed to build MCP prompt.', 'error');
            return;
        }
        ctx = data;
    }
    const text = ctx.mcp_prompt || `Use get_incident_context with incident_id=${incidentId} and propose a minimal fix.`;
    try {
        await navigator.clipboard.writeText(text);
        notify('MCP prompt copied to clipboard.', 'success');
    } catch {
        notify(text, 'info');
    }
}

function copyMCPSetup() {
    const origin = window.location.origin;
    const text = [
        'QA Capsule MCP setup (Cursor):',
        `URL: ${origin}/mcp`,
        'Header: Authorization: Bearer <QACAPSULE_MCP_TOKEN>',
        'Tools: get_incident_context, get_flaky_tests',
        'Workflow: ingest failure → get_incident_context → fix in repo → re-run CI'
    ].join('\n');
    navigator.clipboard.writeText(text).then(
        () => notify('MCP setup copied.', 'success'),
        () => notify(text, 'info')
    );
}

function escapeHtml(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}
