/**
 * Runbooks & Automations — apply curated workflow DAGs to CI gateways.
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canManageWorkflow } from './roles.js';

export function loadRunbooksView() {
    loadRunbookTemplates();
    loadRunbookProjects();
}

async function loadRunbookTemplates() {
    const grid = document.getElementById('runbooks-template-grid');
    if (!grid) return;
    grid.innerHTML = '<p class="runbooks-status">Loading templates…</p>';
    const res = await fetchWithAuth('/api/runbooks/templates');
    const { ok, data } = await parseApiJson(res);
    if (!ok || !data?.templates?.length) {
        grid.innerHTML = '<p class="runbooks-status">No templates available.</p>';
        return;
    }
    grid.innerHTML = '';
    data.templates.forEach(t => {
        const card = document.createElement('div');
        card.className = 'data-card runbook-card';
        const tags = (t.tags || []).map(x => `<span class="tag-pill">${escapeHtml(x)}</span>`).join(' ');
        const nodes = t.node_count != null ? t.node_count : '—';
        card.innerHTML = `
            <h3 class="runbook-card__title">${escapeHtml(t.name)}</h3>
            <p class="runbook-card__desc">${escapeHtml(t.description || '')}</p>
            <div class="runbook-card__tags">${tags}</div>
            <p class="runbook-card__plugins">${nodes} steps · Plugins: ${escapeHtml((t.required_plugins || []).join(', '))}</p>
            <button type="button" class="btn-primary btn-apply-runbook" data-template-id="${escapeHtml(t.id)}">Apply to gateway</button>
        `;
        grid.appendChild(card);
    });
    grid.querySelectorAll('.btn-apply-runbook').forEach(btn => {
        btn.addEventListener('click', () => applyRunbookTemplate(btn.dataset.templateId));
    });
}

async function loadRunbookProjects() {
    const sel = document.getElementById('runbooks-project-select');
    if (!sel) return;
    const res = await fetchWithAuth('/api/my-projects');
    const { ok, data } = await parseApiJson(res);
    if (!ok) return;
    const projects = Array.isArray(data) ? data : data?.projects || [];
    sel.innerHTML = '<option value="">Select CI/CD gateway…</option>';
    projects.forEach(p => {
        const opt = document.createElement('option');
        opt.value = p.id ?? p.ID ?? '';
        opt.textContent = p.name ?? p.Name ?? opt.value;
        sel.appendChild(opt);
    });
}

async function applyRunbookTemplate(templateId) {
    const payload = parseJwt(localStorage.getItem('sre-jwt'));
    if (!canManageWorkflow(payload?.role)) {
        notify('Lead role required to apply runbooks', 'error');
        return;
    }
    const projectId = document.getElementById('runbooks-project-select')?.value;
    if (!projectId) {
        notify('Select a CI/CD gateway first', 'error');
        return;
    }
    const res = await fetchWithAuth('/api/runbooks/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project_id: String(projectId), template_id: templateId, enable: true })
    });
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        notify(data?.error || 'Failed to apply runbook', 'error');
        return;
    }
    notify('Runbook applied — workflow enabled on gateway', 'success');
}

function escapeHtml(s) {
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
