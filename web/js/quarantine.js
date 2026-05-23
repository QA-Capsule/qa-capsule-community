/**
 * Quality / Quarantine (DenyList) view
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify } from './ui.js';
import { canManageQuarantine } from './roles.js';

export function loadQuarantineView() {
    const projectSel = document.getElementById('quarantine-project-filter');
    const tbody = document.getElementById('quarantine-list-body');
    if (projectSel && !projectSel.dataset.bound) {
        projectSel.dataset.bound = '1';
        projectSel.addEventListener('change', () => loadQuarantineList());
    }
    if (tbody && !tbody.dataset.bound) {
        tbody.dataset.bound = '1';
        tbody.addEventListener('click', (e) => {
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
        select.innerHTML += `<option value="${escapeHtml(p.name)}"${i === 0 ? ' selected' : ''}>${escapeHtml(p.name)}</option>`;
    });
}

export async function loadQuarantineList() {
    const tbody = document.getElementById('quarantine-list-body');
    const project = document.getElementById('quarantine-project-filter')?.value;
    if (!tbody || !project) return;
    tbody.innerHTML = '<tr><td colspan="5" style="padding:20px;opacity:0.5;">Loading…</td></tr>';
    const res = await fetchWithAuth(`/api/quarantine?project=${encodeURIComponent(project)}`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        tbody.innerHTML = '<tr><td colspan="5">Failed to load quarantine list.</td></tr>';
        return;
    }
    const tests = data?.tests || [];
    const role = parseJwt(localStorage.getItem('sre-jwt'))?.role;
    const canEdit = canManageQuarantine(role);
    if (!tests.length) {
        tbody.innerHTML = '<tr><td colspan="5" style="padding:20px;opacity:0.5;">No quarantined tests for this gateway.</td></tr>';
        return;
    }
    tbody.innerHTML = tests.map(t => `
        <tr style="border-bottom:1px solid var(--border-main);">
            <td style="padding:10px;">${escapeHtml(t.test_name)}</td>
            <td style="padding:10px;"><code style="font-size:10px;">${escapeHtml(t.fingerprint?.slice(0, 12) || '')}…</code></td>
            <td style="padding:10px;">${escapeHtml(t.reason)}</td>
            <td style="padding:10px;font-size:11px;">${escapeHtml(t.since || '')}</td>
            <td style="padding:10px;text-align:right;">
                ${canEdit ? `<button type="button" class="btn-secondary btn-sm btn-lift-quarantine" data-project="${escapeAttr(project)}" data-fingerprint="${escapeAttr(t.fingerprint)}">Lift</button>` : '—'}
            </td>
        </tr>
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
