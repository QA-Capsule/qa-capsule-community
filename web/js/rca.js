/**
 * RCA & AI Insights view
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canConfigureAI } from './roles.js';
import {
    AI_PROVIDERS,
    iconAiInsight,
    logoForProvider,
    getProviderMeta
} from './ai-provider-icons.js';

let providerPickerBound = false;

export function loadRCAView() {
    const listEl = document.getElementById('rca-insights-body');
    const projectSel = document.getElementById('rca-project-filter');
    if (!listEl) return;

    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    const cfgPanel = document.getElementById('rca-ai-config-panel');
    if (cfgPanel) cfgPanel.style.display = canConfigureAI(role) ? 'block' : 'none';

    initRCAProviderUI();
    loadRCAProjects(projectSel);
    loadRCAInsights();
    loadAIConfigPanel();

    const cfgBtn = document.getElementById('rca-save-ai-config');
    if (cfgBtn && !cfgBtn.dataset.bound) {
        cfgBtn.dataset.bound = '1';
        cfgBtn.addEventListener('click', () => saveAIConfig());
    }
    if (projectSel && !projectSel.dataset.bound) {
        projectSel.dataset.bound = '1';
        projectSel.addEventListener('change', () => loadRCAInsights());
    }
}

function initRCAProviderUI() {
    const headingIcon = document.getElementById('rca-ai-heading-icon');
    if (headingIcon) headingIcon.innerHTML = iconAiInsight();

    const picker = document.getElementById('rca-ai-provider-picker');
    const select = document.getElementById('rca-ai-provider');
    if (!picker || !select) return;

    if (!picker.dataset.rendered) {
        picker.innerHTML = AI_PROVIDERS.map(p => `
            <button type="button" class="ai-provider-option" data-provider="${p.id}" role="radio" aria-checked="false">
                <span class="ai-provider-option__logo">${logoForProvider(p.id)}</span>
                <span>${p.label}</span>
            </button>
        `).join('');
        picker.dataset.rendered = '1';
    }

    if (!providerPickerBound) {
        providerPickerBound = true;
        picker.addEventListener('click', (e) => {
            const btn = e.target.closest('.ai-provider-option');
            if (!btn) return;
            const prev = select.value;
            const next = btn.dataset.provider;
            const applyDefaults = next !== prev;
            setActiveProvider(next, { applyDefaults });
        });
    }

    setActiveProvider(select.value || 'disabled', { applyDefaults: false });
}

export function setActiveProvider(providerId, opts = {}) {
    const select = document.getElementById('rca-ai-provider');
    const picker = document.getElementById('rca-ai-provider-picker');
    const meta = getProviderMeta(providerId);
    const id = meta.id;

    if (select) select.value = id;

    picker?.querySelectorAll('.ai-provider-option').forEach(btn => {
        const active = btn.dataset.provider === id;
        btn.classList.toggle('is-active', active);
        btn.setAttribute('aria-checked', active ? 'true' : 'false');
    });

    const logoEl = document.getElementById('rca-ai-provider-logo');
    if (logoEl) logoEl.innerHTML = logoForProvider(id);

    const modelInput = document.getElementById('rca-ai-model');
    const baseInput = document.getElementById('rca-ai-base-url');
    const keyInput = document.getElementById('rca-ai-key-env');

    if (opts.applyDefaults) {
        if (modelInput && meta.defaultModel) modelInput.value = meta.defaultModel;
        if (baseInput) baseInput.value = meta.defaultBaseUrl;
        if (keyInput) keyInput.value = meta.defaultKeyEnv;
    }

    if (modelInput) {
        modelInput.placeholder = meta.defaultModel || 'Model name';
        modelInput.disabled = id === 'disabled';
    }
    if (baseInput) {
        baseInput.placeholder = meta.defaultBaseUrl || 'Base URL';
        baseInput.disabled = id === 'disabled';
    }
}

async function loadRCAProjects(select) {
    if (!select) return;
    const res = await fetchWithAuth('/api/my-projects');
    const { ok, data } = await parseApiJson(res);
    select.innerHTML = '<option value="">All gateways</option>';
    if (!ok) return;
    (Array.isArray(data) ? data : []).forEach(p => {
        select.innerHTML += `<option value="${escapeHtml(p.name)}">${escapeHtml(p.name)}</option>`;
    });
}

export async function loadRCAInsights() {
    const listEl = document.getElementById('rca-insights-body');
    if (!listEl) return;
    const project = document.getElementById('rca-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=100` : '?limit=100';
    listEl.innerHTML = '<p class="modern-data-grid__loading">Loading AI insights…</p>';
    const res = await fetchWithAuth(`/api/rca/insights${q}`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        listEl.innerHTML = '<p class="modern-data-grid__empty">Failed to load insights.</p>';
        updateRCAStats([]);
        return;
    }
    const rows = Array.isArray(data) ? data : [];
    updateRCAStats(rows);
    if (!rows.length) {
        listEl.innerHTML = '<p class="modern-data-grid__empty">No AI summaries yet. Failures enqueue RCA when AI is enabled.</p>';
        return;
    }
    listEl.innerHTML = rows.map(r => renderRCAInsightCard(r)).join('');
}

function updateRCAStats(rows) {
    const set = (id, v) => {
        const el = document.getElementById(id);
        if (el) el.textContent = String(v);
    };
    const total = rows.length;
    const completed = rows.filter(r => (r.rca_status || '').toLowerCase() === 'completed').length;
    const pending = total - completed;
    set('rca-stat-total', total);
    set('rca-stat-completed', completed);
    set('rca-stat-pending', pending);
}

function renderRCAInsightCard(r) {
    const status = escapeHtml(r.rca_status || 'pending');
    return `
        <article class="data-card rca-insight-card" role="listitem">
            <header class="rca-insight-card__head">
                <div class="rca-insight-card__title-wrap">
                    <h3 class="rca-insight-card__title">${escapeHtml(r.test_name || 'Unknown test')}</h3>
                    <p class="rca-insight-card__meta">
                        ${escapeHtml(r.project_name || '—')}
                        <span class="rca-insight-card__status">${status}</span>
                    </p>
                </div>
                <button type="button" class="btn-secondary btn-sm" onclick="window.triggerRCA(${Number(r.incident_id) || 0})">Re-run</button>
            </header>
            <p class="rca-insight-card__body">${escapeHtml(r.summary || 'No summary generated yet.')}</p>
        </article>
    `;
}

window.loadRCAInsights = loadRCAInsights;

export async function triggerRCA(incidentId) {
    const res = await fetchWithAuth(`/api/incidents/${incidentId}/rca`, { method: 'POST' });
    if (res.ok) {
        notify('RCA analysis queued.', 'success');
        setTimeout(() => loadRCAInsights(), 2000);
    } else {
        notify('Failed to queue RCA.', 'error');
    }
}

async function saveAIConfig() {
    const payload = {
        provider: document.getElementById('rca-ai-provider')?.value || 'disabled',
        model: document.getElementById('rca-ai-model')?.value || '',
        base_url: document.getElementById('rca-ai-base-url')?.value || '',
        api_key_env: document.getElementById('rca-ai-key-env')?.value || 'OPENAI_API_KEY',
        max_tokens: parseInt(document.getElementById('rca-ai-max-tokens')?.value || '1024', 10),
        timeout_seconds: parseInt(document.getElementById('rca-ai-timeout')?.value || '45', 10),
        enabled: document.getElementById('rca-ai-enabled')?.checked || false
    };
    const res = await fetchWithAuth('/api/ai/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });
    if (res.ok) notify('AI provider config saved.', 'success');
    else notify('Failed to save AI config.', 'error');
}

export async function loadAIConfigPanel() {
    const res = await fetchWithAuth('/api/ai/config');
    const { ok, data } = await parseApiJson(res);
    if (!ok || !data) return;
    const set = (id, v) => { const el = document.getElementById(id); if (el) el.value = v ?? ''; };
    set('rca-ai-provider', data.provider);
    set('rca-ai-model', data.model);
    set('rca-ai-base-url', data.base_url);
    set('rca-ai-key-env', data.api_key_env);
    set('rca-ai-max-tokens', data.max_tokens);
    set('rca-ai-timeout', data.timeout_seconds);
    const en = document.getElementById('rca-ai-enabled');
    if (en) en.checked = !!data.enabled;
    setActiveProvider(data.provider || 'disabled', { applyDefaults: false });
}

function escapeHtml(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}
