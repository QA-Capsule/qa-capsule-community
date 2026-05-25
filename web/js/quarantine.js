/**
 * Quality / Quarantine (DenyList) view
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canManageQuarantine } from './roles.js';

export function loadQuarantineView() {
    const projectSel = document.getElementById('quarantine-project-filter');
    const listEl = document.getElementById('quarantine-list-body');
    if (projectSel && !projectSel.dataset.bound) {
        projectSel.dataset.bound = '1';
        projectSel.addEventListener('change', () => loadQuarantineList());
    }
    if (listEl && !listEl.dataset.bound) {
        listEl.dataset.bound = '1';
        listEl.addEventListener('click', (e) => {
            const btn = e.target.closest('.btn-lift-quarantine');
            if (!btn) return;
            e.preventDefault();
            liftQuarantine(btn.dataset.project, btn.dataset.fingerprint);
        });
    }
    loadQuarantineProjects(projectSel);
    loadQuarantineList();

    const addBtn = document.getElementById('quarantine-add-btn');
    if (addBtn && !addBtn.dataset.bound) {
        addBtn.dataset.bound = '1';
        addBtn.addEventListener('click', () => manualQuarantine());
    }
    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    if (addBtn) addBtn.style.display = canManageQuarantine(role) ? '' : 'none';
}

async function loadQuarantineProjects(select) {
    if (!select) return;
    const res = await fetchWithAuth('/api/my-projects');
    const { ok, data } = await parseApiJson(res);
    select.innerHTML = '';
    if (!ok) return;
    const projects = Array.isArray(data) ? data : [];
    if (!projects.length) {
        select.innerHTML = '<option value="">No project</option>';
        return;
    }
    projects.forEach((p, i) => {
        select.innerHTML += `<option value="${escapeAttr(p.name)}"${i === 0 ? ' selected' : ''}>${escapeHtml(p.name)}</option>`;
    });
}

function reasonBadge(reason) {
    const r = (reason || '').toLowerCase();
    let cls = 'intel-badge--default';
    if (r === 'flaky') cls = 'intel-badge--flaky';
    else if (r === 'manual') cls = 'intel-badge--manual';
    else if (r === 'policy') cls = 'intel-badge--policy';
    return `<span class="intel-badge ${cls}">${escapeHtml(reason || '—')}</span>`;
}

function updateQuarantineStats(tests) {
    const set = (id, v) => {
        const el = document.getElementById(id);
        if (el) el.textContent = String(v);
    };
    const active = tests.length;
    const auto = tests.filter(t => {
        const r = (t.reason || '').toLowerCase();
        return r === 'flaky' || r === 'policy';
    }).length;
    const manual = tests.filter(t => (t.reason || '').toLowerCase() === 'manual').length;
    set('quarantine-stat-active', active);
    set('quarantine-stat-auto', auto);
    set('quarantine-stat-manual', manual);
}

export async function loadQuarantineList() {
    const listEl = document.getElementById('quarantine-list-body');
    const project = document.getElementById('quarantine-project-filter')?.value;
    if (!listEl || !project) return;
    listEl.innerHTML = '<p class="modern-data-grid__loading">Loading quarantine list…</p>';
    const res = await fetchWithAuth(`/api/quarantine?project=${encodeURIComponent(project)}`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        listEl.innerHTML = '<p class="intelligence-empty">Failed to load quarantine list.</p>';
        updateQuarantineStats([]);
        return;
    }
    const tests = data?.tests || [];
    updateQuarantineStats(tests);
    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    const canEdit = canManageQuarantine(role);
    if (!tests.length) {
        listEl.innerHTML = '<p class="modern-data-grid__empty">No quarantined tests for this gateway.</p>';
        return;
    }
    listEl.innerHTML = tests.map(t => `
        <div class="modern-data-grid__row" role="listitem">
            <div class="modern-data-grid__cell modern-data-grid__cell--primary">${escapeHtml(t.test_name)}</div>
            <div class="modern-data-grid__cell modern-data-grid__cell--muted">
                <code title="${escapeAttr(t.fingerprint || '')}">${escapeHtml((t.fingerprint || '').slice(0, 12))}…</code>
            </div>
            <div class="modern-data-grid__cell">${reasonBadge(t.reason)}</div>
            <div class="modern-data-grid__cell modern-data-grid__cell--muted">${escapeHtml(t.since || '')}</div>
            <div class="modern-data-grid__cell modern-data-grid__cell--actions">
                ${canEdit
        ? `<button type="button" class="btn-ghost btn-sm btn-lift-quarantine" data-project="${escapeAttr(project)}" data-fingerprint="${escapeAttr(t.fingerprint)}">Release</button>`
        : '<span class="modern-data-grid__cell--muted">—</span>'}
            </div>
        </div>
    `).join('');
}

export async function liftQuarantine(project, fingerprint) {
    const res = await fetchWithAuth(`/api/quarantine?project=${encodeURIComponent(project)}&fingerprint=${encodeURIComponent(fingerprint)}`, {
        method: 'DELETE'
    });
    if (res.ok) {
        notify('Test removed from quarantine.', 'success');
        loadQuarantineList();
    } else {
        notify('Failed to lift quarantine.', 'error');
    }
}

async function manualQuarantine() {
    const project = document.getElementById('quarantine-project-filter')?.value;
    const testName = document.getElementById('quarantine-test-name')?.value?.trim();
    if (!project || !testName) {
        notify('Project and test name required.', 'error');
        return;
    }
    const res = await fetchWithAuth(`/api/quarantine?project=${encodeURIComponent(project)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ test_name: testName, project })
    });
    if (res.ok) {
        notify('Test quarantined.', 'success');
        document.getElementById('quarantine-test-name').value = '';
        loadQuarantineList();
    } else {
        notify('Failed to quarantine test.', 'error');
    }
}

function escapeHtml(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}

function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, '&#39;');
}
