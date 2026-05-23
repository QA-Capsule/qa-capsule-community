/**
 * Visual SRE Workflow Builder (Drawflow + canonical DAG JSON).
 */
import { fetchWithAuth, parseApiJson, parseJwt } from './api.js';
import { notify, showConfirmModal } from './ui.js';
import { canManageWorkflow, normalizeRole } from './roles.js';

let editor = null;
let currentProjectId = null;
let canEditWorkflow = false;
let workflowCatalog = [];
let canonicalWorkflow = null;

const NODE_TEMPLATES = {
    trigger: {
        html: `<div class="wf-node wf-trigger"><div class="wf-title">Trigger</div><div class="wf-desc">On CI alert ingested</div></div>`,
        inputs: 0,
        outputs: 1,
        data: { label: 'On CI alert', nodeType: 'trigger' }
    },
    condition: {
        html: `<div class="wf-node wf-condition"><div class="wf-title">Condition</div>
            <select class="wf-cond-op login-input" style="margin-top:6px;font-size:11px;width:100%;">
                <option value="tag:prefix:[FLAKY]">Tag: [FLAKY]</option>
                <option value="tag:prefix:[PERF]">Tag: [PERF]</option>
                <option value="status:eq:CRITICAL">Status: CRITICAL</option>
                <option value="status:eq:PERF_DEGRADATION">Status: PERF</option>
                <option value="text:contains:timeout">Text contains timeout</option>
            </select></div>`,
        inputs: 1,
        outputs: 2,
        data: { label: 'Condition', nodeType: 'condition', when: { op: 'tag', match: 'prefix', value: '[FLAKY]', field: 'incident.name' } }
    },
    action: {
        html: `<div class="wf-node wf-action"><div class="wf-title">Action</div>
            <select class="wf-action-path login-input" style="margin-top:6px;font-size:11px;width:100%;"><option value="">Select integration…</option></select></div>`,
        inputs: 1,
        outputs: 0,
        data: { label: 'Action', nodeType: 'action', file_path: '', integration: '' }
    }
};

/** Drawflow typenode: false = HTML template (required for custom node HTML). */
const HTML_NODE = false;

export function openWorkflowEditor(projectId, projectName) {
    if (!projectId) {
        notify('Invalid project id.', 'error');
        return;
    }
    currentProjectId = projectId;
    const modal = document.getElementById('workflow-editor-modal');
    if (!modal) {
        notify('Workflow editor UI is missing. Refresh the page.', 'error');
        return;
    }
    const titleEl = document.getElementById('workflow-editor-title');
    if (titleEl) {
        titleEl.textContent = `Visual Workflow — ${projectName || projectId}`;
    }
    modal.style.display = 'flex';
    bindWorkflowToolbar();
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            void loadWorkflow(projectId).catch((err) => {
                console.error('[workflow] load failed', err);
                notify('Failed to open workflow editor.', 'error');
            });
        });
    });
}

function bindWorkflowToolbar() {
    const toggle = document.getElementById('workflow-enabled-toggle');
    if (toggle && !toggle.dataset.bound) {
        toggle.dataset.bound = '1';
        toggle.addEventListener('change', () => updateWorkflowStatusPill());
    }
}

export function closeWorkflowEditor() {
    const modal = document.getElementById('workflow-editor-modal');
    if (modal) modal.style.display = 'none';
    destroyDrawflowEditor();
    currentProjectId = null;
}

function destroyDrawflowEditor() {
    const container = document.getElementById('drawflow');
    if (container) {
        container.innerHTML = '';
    }
    editor = null;
}

function initDrawflow() {
    if (typeof Drawflow === 'undefined') {
        notify('Drawflow library not loaded. Check your network or refresh.', 'error');
        return null;
    }
    const container = document.getElementById('drawflow');
    if (!container) return null;

    destroyDrawflowEditor();
    container.innerHTML = '';

    editor = new Drawflow(container);
    editor.reroute = true;
    editor.reroute_fix_curvature = true;
    editor.editor_mode = canEditWorkflow ? 'edit' : 'fixed';
    editor.start();

    const canvas = container.querySelector('.drawflow');
    if (canvas) {
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        canvas.style.minHeight = '420px';
    }
    refreshDrawflowViewport();
    return editor;
}

function refreshDrawflowViewport() {
    if (!editor) return;
    requestAnimationFrame(() => {
        try {
            if (typeof editor.zoom_refresh === 'function') {
                editor.zoom_refresh();
            }
            if (typeof editor.zoom_reset === 'function') {
                editor.zoom_reset();
            }
        } catch (_) { /* ignore */ }
        updateCanvasEmptyState();
    });
}

function drawflowNodeCount() {
    if (!editor?.drawflow?.drawflow?.Home?.data) return 0;
    return Object.keys(editor.drawflow.drawflow.Home.data).length;
}

function updateCanvasEmptyState() {
    const el = document.getElementById('workflow-canvas-empty');
    if (!el) return;
    el.classList.toggle('visible', drawflowNodeCount() === 0);
}

function addDrawflowNode(type, posX, posY, data, html) {
    const tpl = NODE_TEMPLATES[type];
    if (!editor || !tpl) return null;
    const nodeData = { ...tpl.data, ...data };
    return editor.addNode(
        type,
        tpl.inputs,
        tpl.outputs,
        posX,
        posY,
        type,
        nodeData,
        html || tpl.html,
        HTML_NODE
    );
}

async function loadWorkflow(projectId) {
    const res = await fetchWithAuth(`/api/projects/${encodeURIComponent(projectId)}/workflow`);
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        notify(data?.error || 'Failed to load workflow.', 'error');
        return;
    }
    canEditWorkflow = !!data.can_edit;
    workflowCatalog = Array.isArray(data.catalog) ? data.catalog : [];
    canonicalWorkflow = data.workflow || null;

    applyEditorRBAC();
    if (!initDrawflow()) return;

    populateActionSelects();
    syncEnableToggle(data.workflow, data.workflow_enabled, data.has_workflow);
    importWorkflowToCanvas(data.workflow, data.workflow?.ui);
    refreshDrawflowViewport();
    updateWorkflowStatusPill();
}

function syncEnableToggle(workflow, workflowEnabled, hasWorkflow) {
    const toggle = document.getElementById('workflow-enabled-toggle');
    if (!toggle) return;
    if (workflow && typeof workflow.enabled === 'boolean') {
        toggle.checked = workflow.enabled;
    } else if (hasWorkflow) {
        toggle.checked = !!workflowEnabled;
    } else {
        toggle.checked = false;
    }
}

export function isWorkflowEnabled() {
    const toggle = document.getElementById('workflow-enabled-toggle');
    return toggle ? toggle.checked : false;
}

export function updateWorkflowStatusPill() {
    const pill = document.getElementById('workflow-status-pill');
    if (!pill) return;
    const enabled = isWorkflowEnabled();
    const hasGraph = drawflowNodeCount() > 0;
    const hasStored = canonicalWorkflow && (canonicalWorkflow.entry || Object.keys(canonicalWorkflow.nodes || {}).length > 0);

    pill.classList.remove('state-active', 'state-draft', 'state-legacy');
    if (enabled && (hasGraph || hasStored)) {
        pill.textContent = 'DAG active';
        pill.classList.add('state-active');
        pill.title = 'Visual workflow drives remediation on alert ingest';
    } else if (hasGraph || hasStored) {
        pill.textContent = 'Draft';
        pill.classList.add('state-draft');
        pill.title = 'Workflow saved but disabled — legacy AUTO-RUN still applies';
    } else {
        pill.textContent = 'Legacy';
        pill.classList.add('state-legacy');
        pill.title = 'No visual workflow — legacy AUTO-RUN and trigger_on rules';
    }
}

function applyEditorRBAC() {
    const role = normalizeRole(parseJwt(localStorage.getItem('sre-jwt'))?.role);
    canEditWorkflow = canManageWorkflow(role) && canEditWorkflow;
    document.querySelectorAll('.wf-edit-only').forEach(el => {
        el.style.display = canEditWorkflow ? '' : 'none';
    });
    const badge = document.getElementById('workflow-readonly-badge');
    if (badge) badge.style.display = canEditWorkflow ? 'none' : 'block';
    const toggle = document.getElementById('workflow-enabled-toggle');
    if (toggle) toggle.disabled = !canEditWorkflow;
    if (editor) {
        editor.editor_mode = canEditWorkflow ? 'edit' : 'fixed';
    }
}

function populateActionSelects() {
    const options = workflowCatalog.map(p =>
        `<option value="${escapeAttr(p.file_path)}">${escapeHtml(p.name || p.file_path)}</option>`
    ).join('');
    document.querySelectorAll('.wf-action-path').forEach(sel => {
        const cur = sel.value;
        sel.innerHTML = `<option value="">Select integration…</option>${options}`;
        if (cur) sel.value = cur;
        if (!canEditWorkflow) sel.disabled = true;
    });
    document.querySelectorAll('.wf-cond-op').forEach(sel => {
        if (!canEditWorkflow) sel.disabled = true;
    });
}

function importWorkflowToCanvas(workflow, ui) {
    if (!editor) return;
    editor.clear();

    let imported = false;
    if (ui && ui.drawflow && drawflowImportHasNodes(ui)) {
        try {
            editor.import(ui);
            imported = drawflowNodeCount() > 0;
        } catch (e) {
            console.warn('[workflow] drawflow import failed', e);
        }
    }

    if (imported) {
        bindNodeControls();
        syncControlsFromNodeData();
        updateCanvasEmptyState();
        return;
    }

    if (workflow?.nodes && Object.keys(workflow.nodes).length > 0) {
        buildCanvasFromCanonical(workflow);
        return;
    }

    addDefaultStarterGraph();
}

function drawflowImportHasNodes(ui) {
    const data = ui?.drawflow?.Home?.data;
    return data && Object.keys(data).length > 0;
}

function buildCanvasFromCanonical(workflow) {
    const idMap = {};
    let y = 80;
    let x = 80;
    for (const [cid, node] of Object.entries(workflow.nodes)) {
        const tpl = NODE_TEMPLATES[node.type];
        if (!tpl) continue;
        const data = { ...tpl.data, label: node.label || tpl.data.label, nodeType: node.type };
        if (node.type === 'condition' && node.when) data.when = node.when;
        if (node.type === 'action') {
            data.file_path = node.file_path || '';
            data.integration = node.integration || '';
        }
        const numId = addDrawflowNode(node.type, x, y, data, tpl.html);
        idMap[cid] = numId;
        y += 130;
    }
    (workflow.edges || []).forEach((e) => {
        const from = idMap[e.from];
        const to = idMap[e.to];
        if (!from || !to) return;
        let outPort = 'output_1';
        if (e.when === 'false') outPort = 'output_2';
        editor.addConnection(from, to, outPort, 'input_1');
    });
    bindNodeControls();
    syncControlsFromNodeData();
    updateCanvasEmptyState();
}

function addDefaultStarterGraph() {
    if (!editor) return;
    const t = addDrawflowNode('trigger', 120, 120, {}, null);
    const c = addDrawflowNode('condition', 380, 120, {}, null);
    if (t && c) {
        editor.addConnection(t, c, 'output_1', 'input_1');
    }
    if (canEditWorkflow && workflowCatalog.length > 0) {
        const plugin = workflowCatalog[0];
        const a = addDrawflowNode('action', 640, 60, {
            file_path: plugin.file_path,
            integration: plugin.integration,
            label: plugin.name || 'Action'
        }, null);
        if (c && a) editor.addConnection(c, a, 'output_1', 'input_1');
    }
    bindNodeControls();
    syncControlsFromNodeData();
    updateCanvasEmptyState();
}

function syncControlsFromNodeData() {
    if (!editor) return;
    Object.values(editor.drawflow.drawflow.Home.data).forEach((node) => {
        const id = node.id;
        const d = node.data || {};
        const el = document.querySelector(`#node-${id}`);
        if (!el) return;
        if (d.nodeType === 'condition' && d.when) {
            const sel = el.querySelector('.wf-cond-op');
            if (sel) sel.value = conditionToPreset(d.when);
        }
        if (d.nodeType === 'action' && d.file_path) {
            const sel = el.querySelector('.wf-action-path');
            if (sel) sel.value = d.file_path;
        }
    });
}

function conditionToPreset(when) {
    if (!when) return 'tag:prefix:[FLAKY]';
    const { op, match, value, field } = when;
    if (op === 'tag') return `tag:${match || 'prefix'}:${value || '[FLAKY]'}`;
    if (op === 'status') return `status:${match || 'eq'}:${value || 'CRITICAL'}`;
    if (op === 'text') return `text:${match || 'contains'}:${value || 'timeout'}`;
    return 'tag:prefix:[FLAKY]';
}

function bindNodeControls() {
    document.querySelectorAll('.wf-cond-op').forEach(sel => {
        sel.onchange = () => syncConditionData(sel);
        if (!canEditWorkflow) sel.disabled = true;
    });
    document.querySelectorAll('.wf-action-path').forEach(sel => {
        sel.onchange = () => syncActionData(sel);
        if (!canEditWorkflow) sel.disabled = true;
    });
}

function syncConditionData(sel) {
    const nodeEl = sel.closest('.drawflow-node');
    if (!nodeEl || !editor) return;
    const id = nodeEl.id.replace('node-', '');
    const when = parseConditionPreset(sel.value);
    const data = editor.getNodeFromId(id)?.data || {};
    data.when = when;
    editor.updateNodeDataFromId(id, data);
}

function syncActionData(sel) {
    const nodeEl = sel.closest('.drawflow-node');
    if (!nodeEl || !editor) return;
    const id = nodeEl.id.replace('node-', '');
    const path = sel.value;
    const plugin = workflowCatalog.find(p => p.file_path === path);
    const data = editor.getNodeFromId(id)?.data || {};
    data.file_path = path;
    data.integration = plugin?.integration || '';
    data.label = plugin?.name || 'Action';
    editor.updateNodeDataFromId(id, data);
}

function parseConditionPreset(val) {
    const [op, match, ...rest] = val.split(':');
    const value = rest.join(':');
    if (op === 'tag') return { op: 'tag', match, value, field: 'incident.name' };
    if (op === 'status') return { op: 'status', match, value };
    if (op === 'text') return { op: 'text', match, field: 'incident.name', value };
    return { op: 'tag', match: 'prefix', value: '[FLAKY]', field: 'incident.name' };
}

export function addWorkflowNode(type) {
    if (!canEditWorkflow || !editor) return;
    const count = drawflowNodeCount();
    const y = 100 + count * 100;
    addDrawflowNode(type, 200, y, {}, null);
    populateActionSelects();
    bindNodeControls();
    updateCanvasEmptyState();
}

export async function saveWorkflow() {
    if (!canEditWorkflow || !currentProjectId || !editor) return;
    const enabled = isWorkflowEnabled();
    const exported = editor.export();
    let doc = canonicalFromDrawflow(exported);

    if (!doc) {
        if (enabled) {
            notify('Workflow must include a Trigger node before enabling.', 'error');
            return;
        }
        doc = {
            version: 1,
            enabled: false,
            entry: '',
            nodes: {},
            edges: [],
            meta: { name: 'Remediation workflow (draft)' }
        };
    }

    doc.ui = exported;
    doc.enabled = enabled;
    doc.version = 1;
    doc.meta = { ...(doc.meta || {}), name: doc.meta?.name || 'Remediation workflow' };

    const res = await fetchWithAuth(`/api/projects/${encodeURIComponent(currentProjectId)}/workflow`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(doc)
    });
    const { ok, data } = await parseApiJson(res);
    if (!ok) {
        notify(data?.error || 'Failed to save workflow.', 'error');
        return;
    }
    canonicalWorkflow = doc;
    updateWorkflowStatusPill();
    if (enabled) {
        notify('Workflow saved and enabled — DAG is active for this pipeline.', 'success');
    } else {
        notify('Workflow saved as draft — legacy AUTO-RUN remains active.', 'success');
    }
    if (typeof window.loadGatewaysData === 'function') {
        window.loadGatewaysData();
    }
}

export async function clearWorkflow() {
    if (!canEditWorkflow || !currentProjectId) return;
    showConfirmModal(
        'Reset visual workflow?',
        'This deletes the DAG for this pipeline and restores legacy AUTO-RUN behavior.',
        'danger',
        async () => {
            const res = await fetchWithAuth(`/api/projects/${encodeURIComponent(currentProjectId)}/workflow`, {
                method: 'DELETE'
            });
            if (res.ok) {
                notify('Workflow cleared — legacy AUTO-RUN restored.', 'success');
                canonicalWorkflow = null;
                const toggle = document.getElementById('workflow-enabled-toggle');
                if (toggle) toggle.checked = false;
                if (initDrawflow()) {
                    addDefaultStarterGraph();
                }
                updateWorkflowStatusPill();
                if (typeof window.loadGatewaysData === 'function') {
                    window.loadGatewaysData();
                }
            } else {
                notify('Failed to clear workflow.', 'error');
            }
        }
    );
}

function canonicalFromDrawflow(exported) {
    const home = exported?.drawflow?.Home?.data;
    if (!home) return null;
    const nodes = {};
    const idMap = {};
    let entry = '';

    Object.entries(home).forEach(([numId, node]) => {
        const cid = `n_${numId}`;
        idMap[numId] = cid;
        const d = node.data || {};
        const type = d.nodeType || node.name;
        const out = { type, label: d.label || type };
        if (type === 'condition') out.when = d.when || NODE_TEMPLATES.condition.data.when;
        if (type === 'action') {
            out.file_path = d.file_path || '';
            out.integration = d.integration || '';
        }
        nodes[cid] = out;
        if (type === 'trigger' && !entry) entry = cid;
    });
    if (!entry) return null;

    const edges = [];
    Object.entries(home).forEach(([numId, node]) => {
        const outs = node.outputs || {};
        Object.entries(outs).forEach(([outKey, out]) => {
            const conns = out.connections || [];
            conns.forEach(c => {
                let when = '';
                if (outKey === 'output_2') when = 'false';
                if (outKey === 'output_1' && (node.name === 'condition' || node.data?.nodeType === 'condition')) {
                    when = 'true';
                }
                edges.push({
                    from: idMap[numId],
                    to: idMap[String(c.node)],
                    when
                });
            });
        });
    });

    return { version: 1, enabled: false, entry, nodes, edges };
}

function escapeHtml(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}
function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, '&#39;');
}

export function toggleWorkflowHelp() {
    const panel = document.getElementById('workflow-help-panel');
    if (panel) panel.classList.toggle('open');
}
