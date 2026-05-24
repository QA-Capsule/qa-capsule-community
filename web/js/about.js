/**
 * Help Center — available to all roles
 */
import { FINOPS_METRICS_DOC_HTML } from './finops-metrics-doc.js';
import { EXTENDED_HELP_TOPICS } from './help-center-topics.js';
import { ROLE_LABELS, ROLE_DESCRIPTIONS } from './roles.js';

const TOPIC_ICONS = {
    overview: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
    architecture: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>',
    roles: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>',
    operations: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M9 21V9"/></svg>',
    gateways: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>',
    workflow: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="6" cy="6" r="3"/><circle cx="18" cy="18" r="3"/><path d="M8.5 8.5L15.5 15.5"/></svg>',
    plugins: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>',
    rca_quarantine: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2a7 7 0 0 1 7 7c0 5-7 13-7 13S5 14 5 9a7 7 0 0 1 7-7z"/><circle cx="12" cy="9" r="2"/></svg>',
    runbooks_dora: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>',
    analytics: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>',
    finops_metrics: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',
    glossary: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>'
};

const ABOUT_TOPICS = {
    overview: {
        title: 'Product overview',
        breadcrumb: 'Help Center / Overview',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>QA Capsule — SRE control plane</h2>
          <p class="about-lead">
            QA Capsule is a <strong>flight recorder for CI quality</strong>: signed gateways ingest failures,
            fingerprints deduplicate noise, flaky tests are tagged, and remediation runs through
            <strong>visual workflows</strong> or legacy plugins — plus AI summaries, quarantine, runbooks, and DORA.
          </p>
          <h3>Major capabilities</h3>
          <ul class="about-detail-list">
            <li><strong>Operations dashboard</strong> — Pipeline executions, resolve/delete, log export.</li>
            <li><strong>CI/CD Gateways</strong> — API keys, routing matrix, workflow badges.</li>
            <li><strong>Plugin Engine</strong> — Native Go integrations (Slack, Jira, PagerDuty, …).</li>
            <li><strong>Visual Workflows</strong> — DAG editor with simulate dry-run.</li>
            <li><strong>AI RCA</strong> — Async LLM incident summaries.</li>
            <li><strong>Quarantine</strong> — Deny-list + CI skip API.</li>
            <li><strong>Runbooks</strong> — Template workflows in one click.</li>
            <li><strong>DORA</strong> — Deployment metrics + Prometheus correlation.</li>
            <li><strong>FinOps</strong> — Cost of failures and flaky waste (Manager).</li>
          </ul>
          <div class="about-callout">
            <strong>Typical flow</strong>
            <ol>
              <li>Admin/Manager provisions gateway + team access.</li>
              <li>CI posts failures with <code>X-API-Key</code> and <code>X-Run-Id</code>.</li>
              <li>Lead triages on Operations; workflow or plugins notify chat/tickets.</li>
              <li>Manager reviews DORA and FinOps.</li>
            </ol>
          </div>
        </article>`
    },
    ...EXTENDED_HELP_TOPICS,
    roles: {
        title: 'Role hierarchy',
        breadcrumb: 'Help Center / Roles',
        html: buildRolesDoc()
    },
    finops_metrics: {
        title: 'About FinOps metrics',
        breadcrumb: 'Help Center / FinOps metrics',
        html: FINOPS_METRICS_DOC_HTML
    },
    plugins: {
        title: 'Plugin Engine',
        breadcrumb: 'Help Center / Plugins',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>Native integration engine</h2>
          <p class="about-lead">
            Plugins are JSON manifests under <code>plugins/</code> loaded at server start.
            Remediation runs in <strong>Go</strong> (HTTP clients) — not shell scripts in production paths.
          </p>
          <table class="about-table">
            <thead><tr><th>Category</th><th>Use case</th></tr></thead>
            <tbody>
              <tr><td>Slack / Teams</td><td>Chat alerts with per-gateway channel routing</td></tr>
              <tr><td>Jira</td><td>Create/update issues from failures</td></tr>
              <tr><td>PagerDuty / Opsgenie / VictorOps</td><td>On-call paging</td></tr>
              <tr><td>Datadog</td><td>Events on the SRE timeline</td></tr>
              <tr><td>Webhook / GitHub Actions</td><td>Custom automation</td></tr>
              <tr><td>Kubernetes</td><td>Rollout restart remediation</td></tr>
              <tr><td>Test management</td><td>TestRail, Zephyr, Xray</td></tr>
            </tbody>
          </table>
          <h3>Manager vs Lead</h3>
          <ul class="about-detail-list">
            <li><strong>Manager</strong> — Enable routing on plugin, toggle AUTO-RUN globally per manifest.</li>
            <li><strong>Lead</strong> — Configure secrets, manual Execute test, use in workflows.</li>
          </ul>
          <p>Workflow <strong>Action</strong> nodes only list plugins that are routing-active and allowed on the gateway.</p>
        </article>`
    },
    glossary: {
        title: 'SRE glossary',
        breadcrumb: 'Help Center / Glossary',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>SRE &amp; quality engineering glossary</h2>
          <p class="about-lead">Definitions as used in QA Capsule UI, APIs, and documentation.</p>
          <dl class="about-dl">
            <dt>MTTR (Mean Time To Resolution)</dt>
            <dd>Average minutes from <code>created_at</code> to <code>resolved_at</code> for resolved incidents.</dd>
            <dt>Flaky test</dt>
            <dd>Re-failure within 48h after resolution; prefixed <code>[FLAKY]</code> in test name.</dd>
            <dt>Fingerprint</dt>
            <dd>SHA-256(name ∥ error) for dedup per pipeline run.</dd>
            <dt>Pipeline execution</dt>
            <dd>UI group keyed by <code>pipeline_run_id</code> (header <code>X-Run-Id</code>).</dd>
            <dt>Gateway</dt>
            <dd>CI/CD project with API key, routing, and optional workflow DAG.</dd>
            <dt>Legacy mode</dt>
            <dd>No enabled workflow — linear AUTO-RUN on plugin <code>trigger_on</code> keywords.</dd>
            <dt>DAG / Visual workflow</dt>
            <dd>Enabled <code>sre_workflow_json</code> executed by WorkflowEngine.</dd>
            <dt>Quarantine</dt>
            <dd>Test on deny-list — ingest suppressed, CI may skip via API.</dd>
            <dt>Runbook</dt>
            <dd>Pre-built workflow template applied to a gateway.</dd>
            <dt>RCA</dt>
            <dd>AI-generated root cause summary stored per incident.</dd>
            <dt>Analytics layout</dt>
            <dd>Per-user dashboard chart configuration (not a separate DSL).</dd>
          </dl>
        </article>`
    }
};

function buildRolesDoc() {
    const rows = Object.keys(ROLE_LABELS).map(code => `
      <tr>
        <td><code>${code}</code></td>
        <td><strong>${ROLE_LABELS[code]}</strong></td>
        <td>${ROLE_DESCRIPTIONS[code]}</td>
      </tr>`).join('');

    return `
    <article class="about-doc">
      <h2>Global roles</h2>
      <p class="about-lead">Roles gate views and mutations. Observer is read-only on operations; Manager owns FinOps and DORA.</p>
      <table class="about-table">
        <thead><tr><th>Code</th><th>UI label</th><th>Scope</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
      <div class="about-callout" style="margin-top:20px;">
        <strong>Platform Admin</strong> — Workspaces, IAM, Settings.<br>
        <strong>Manager</strong> — FinOps, DORA, gateways, plugins, full triage.<br>
        <strong>Lead</strong> — Operations, gateways, plugins, workflows, quarantine.<br>
        <strong>Observer</strong> — Read-only dashboard and RCA/quarantine lists.
      </div>
    </article>`;
}

export function loadAboutView() {
    const nav = document.getElementById('about-topic-nav');
    if (!nav) return;
    if (!nav.dataset.built) {
        nav.dataset.built = '1';
        const order = [
            'overview', 'architecture', 'roles', 'operations', 'gateways',
            'workflow', 'plugins', 'rca_quarantine', 'runbooks_dora',
            'analytics', 'finops_metrics', 'glossary'
        ];
        nav.innerHTML = order
            .filter(key => ABOUT_TOPICS[key])
            .map(key => {
                const topic = ABOUT_TOPICS[key];
                return `
            <button type="button" class="about-topic-btn" data-topic="${key}" onclick="window.showAboutTopic('${key}')">
                ${TOPIC_ICONS[key] || ''}<span>${topic.title}</span>
            </button>`;
            }).join('');
    }
    showAboutTopic('overview');
}

export function showAboutTopic(key) {
    const topic = ABOUT_TOPICS[key];
    if (!topic) return;

    document.querySelectorAll('.about-topic-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.topic === key);
    });

    const content = document.getElementById('about-topic-content');
    if (content) {
        content.innerHTML = `
            <div class="about-breadcrumb">${topic.breadcrumb}</div>
            ${topic.html}`;
    }
}

/** Open Help Center from workflow editor (?) */
export function openWorkflowHelpTopic() {
    if (typeof window.switchView === 'function') {
        window.switchView('about', document.querySelector('.nav-item[onclick*="about"]'));
    }
    showAboutTopic('workflow');
}
