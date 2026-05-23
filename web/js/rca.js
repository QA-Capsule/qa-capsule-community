/**
 * RCA & AI Insights view
 */
import { fetchWithAuth, parseApiJson } from './api.js';
import { notify } from './ui.js';
import { canConfigureAI } from './roles.js';
import { parseJwt } from './api.js';

export function loadRCAView() {
    const tbody = document.getElementById('rca-insights-body');
    const projectSel = document.getElementById('rca-project-filter');
    if (!tbody) return;

    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    const cfgPanel = document.getElementById('rca-ai-config-panel');
    if (cfgPanel) cfgPanel.style.display = canConfigureAI(role) ? 'block' : 'none';

    loadRCAProjects(projectSel);
    loadRCAInsights();

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
    const tbody = document.getElementById('rca-insights-body');
    if (!tbody) return;
    const project = document.getElementById('rca-project-filter')?.value || '';
    const q = project ? `?project=${encodeURIComponent(project)}&limit=100` : '?limit=100';
    tbody.innerHTML = '<tr><td colspan="5" style="padding:20px;opacity:0.5;">Loading…</td></tr>';
    const res = await fetchWithAuth(`/api/rca/insights${q}`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        tbody.innerHTML = '<tr><td colspan="5">Failed to load insights.</td></tr>';
        return;
    }
    const rows = Array.isArray(data) ? data : [];
    if (!rows.length) {
        tbody.innerHTML = '<tr><td colspan="5" style="padding:20px;opacity:0.5;">No AI summaries yet. Failures enqueue RCA when AI is enabled.</td></tr>';
        return;
    }
    tbody.innerHTML = rows.map(r => `
        <tr style="border-bottom:1px solid var(--border-main);">
            <td style="padding:10px;">${escapeHtml(r.project_name)}</td>
            <td style="padding:10px;">${escapeHtml(r.test_name)}</td>
            <td style="padding:10px;">${escapeHtml(r.rca_status || '—')}</td>
            <td style="padding:10px;font-size:12px;">${escapeHtml(r.summary || '—')}</td>
            <td style="padding:10px;text-align:right;">
                <button type="button" class="btn-secondary btn-sm" onclick="window.triggerRCA(${r.incident_id})">Re-run</button>
            </td>
        </tr>
    `).join('');
}

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
}

function escapeHtml(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}
