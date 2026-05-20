/**
 * Help Center — available to all roles
 */
import { FINOPS_METRICS_DOC_HTML } from './finops-metrics-doc.js';
import { formulaBlock, bindAboutFormulaCopy } from './about-formula.js';
import { ROLE_LABELS, ROLE_DESCRIPTIONS } from './roles.js';

const TOPIC_ICONS = {
    overview: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
    roles: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>',
    finops_metrics: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',
    glossary: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>',
    qcl: '<svg class="about-topic-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>'
};

const ABOUT_TOPICS = {
    overview: {
        title: 'Product overview',
        breadcrumb: 'Help Center / Overview',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>QA Capsule — SRE control plane</h2>
          <p class="about-lead">
            QA Capsule is a <strong>flight recorder for CI quality</strong>: it ingests pipeline failures through signed gateways,
            deduplicates alerts via cryptographic fingerprints, classifies flaky regressions, and surfaces both
            <em>operational</em> (MTTR, backlog) and <em>economic</em> (FinOps) indicators in a single pane of glass.
          </p>
          <h3>Architecture layers</h3>
          <ul class="about-detail-list">
            <li><strong>Ingestion plane</strong> — Webhooks accept JUnit XML / JSON payloads; optional <code>X-Run-Id</code> correlates multi-test failures into one pipeline execution group.</li>
            <li><strong>Correlation engine</strong> — SHA-256 fingerprint over test name + error message; spam guard per <code>pipeline_run_id</code>.</li>
            <li><strong>Triage workspace</strong> — Role-aware dashboard with bulk resolve, log export, JUnit regeneration.</li>
            <li><strong>Analytics &amp; QCL</strong> — Chart Studio evaluates declarative queries; pinned charts hydrate Dashboard and FinOps views.</li>
            <li><strong>FinOps intelligence</strong> — Manager-only cost modeling from configurable baselines (developer rate, CI minute cost, investigation time).</li>
          </ul>
          <div class="about-callout">
            <strong>End-to-end workflow</strong>
            <ol>
              <li><strong>Admin</strong> creates workspaces, IAM users, and CI/CD gateways (API keys).</li>
              <li><strong>CI pipeline</strong> posts failures to <code>/api/webhook/ingest</code> on red builds.</li>
              <li><strong>Operator / QA Lead</strong> triages grouped pipeline executions, resolves or bulk-resolves tests.</li>
              <li><strong>Manager</strong> reviews FinOps KPIs, weekly evolution, CSV exports, and custom QCL charts.</li>
              <li><strong>Viewer</strong> consumes dashboards read-only; Chart Studio access without FinOps write paths.</li>
            </ol>
          </div>
          <h3>Dashboard analytics (built-in)</h3>
          <p>
            Toggle <em>System analytics &amp; quality</em> on the Operations dashboard to reveal MTTR, MTTF, failure-quality
            doughnut (stable vs flaky), and a 5-week multi-axis evolution chart (volume + MTTR). Extended stat tiles show
            resolution rate, active backlog, and flaky ratio updated on each refresh.
          </p>
        </article>`
    },
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
    qcl: {
        title: 'Chart language (QCL)',
        breadcrumb: 'Help Center / QCL',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>QA Chart Language (QCL) v1.0</h2>
          <p class="about-lead">
            QCL is a <strong>declarative time-series DSL</strong> for quality metrics. Queries compile to Chart.js specs via
            <code>POST /api/charts/evaluate</code>. Saved definitions persist in <code>saved_charts</code> and can be pinned
            to Dashboard or FinOps surfaces.
          </p>
          <h3>Canonical example</h3>
          ${formulaBlock('CHART line "Weekly composite FinOps exposure"\nMETRIC finops_cost\nRANGE 12w\nGROUP week\nPROJECT QA-CAP-FRONT-PIPELINE')}
          <h3>Directive reference</h3>
          <table class="about-table">
            <thead><tr><th>Directive</th><th>Syntax</th><th>Semantics</th></tr></thead>
            <tbody>
              <tr><td><code>CHART</code></td><td><code>line | bar | doughnut</code> + optional title</td><td>Visualization primitive; doughnut requires a single-series metric.</td></tr>
              <tr><td><code>METRIC</code></td><td>see table below</td><td>Scalar field aggregated per GROUP bucket.</td></tr>
              <tr><td><code>RANGE</code></td><td><code>Nd</code>, <code>Nw</code>, <code>Ny</code></td><td>Lookback window (1–730 days). Examples: <code>35d</code>, <code>12w</code>, <code>1y</code>.</td></tr>
              <tr><td><code>GROUP</code></td><td><code>week | project</code></td><td>Temporal bucket (ISO week start) or gateway dimension.</td></tr>
              <tr><td><code>PROJECT</code></td><td>gateway name</td><td>Optional filter on <code>project_name</code> (CI gateway).</td></tr>
            </tbody>
          </table>
          <h3>Supported METRIC tokens</h3>
          <table class="about-table">
            <thead><tr><th>METRIC</th><th>Unit</th><th>Description</th></tr></thead>
            <tbody>
              <tr><td><code>incidents</code></td><td>count</td><td>Total failures in bucket.</td></tr>
              <tr><td><code>flaky</code></td><td>count</td><td>Failures tagged <code>[FLAKY]</code>.</td></tr>
              <tr><td><code>stable</code></td><td>count</td><td>Non-flaky failures (N − N<sub>f</sub>).</td></tr>
              <tr><td><code>resolved</code></td><td>count</td><td>Incidents with <code>is_resolved = 1</code>.</td></tr>
              <tr><td><code>active</code></td><td>count</td><td>Unresolved backlog in bucket.</td></tr>
              <tr><td><code>mttr</code></td><td>minutes</td><td>Mean resolution latency for resolved rows.</td></tr>
              <tr><td><code>resolution_rate</code></td><td>%</td><td>100 × resolved / total.</td></tr>
              <tr><td><code>flaky_ratio</code></td><td>%</td><td>100 × flaky / total.</td></tr>
              <tr><td><code>finops_cost</code></td><td>USD</td><td>Full loaded cost (CI + investigation).</td></tr>
              <tr><td><code>finops_flaky_cost</code></td><td>USD</td><td>Loaded cost attributed to flaky subset.</td></tr>
              <tr><td><code>ci_minutes</code></td><td>minutes</td><td>Runner minutes (incidents × T<sub>pipe</sub>).</td></tr>
              <tr><td><code>ci_cost</code></td><td>USD</td><td>CI spend only.</td></tr>
              <tr><td><code>invest_cost</code></td><td>USD</td><td>Investigation spend only.</td></tr>
            </tbody>
          </table>
          <h3>Advanced composition patterns</h3>
          <p><strong>Multi-gateway comparison</strong> — omit PROJECT, set GROUP project, use bar chart:</p>
          ${formulaBlock('CHART bar "Incident density by gateway"\nMETRIC incidents\nRANGE 30d\nGROUP project')}
          <p><strong>Quality vs cost</strong> — pin two charts (flaky_ratio + finops_flaky_cost) to FinOps for executive review.</p>
          ${formulaBlock('CHART line "Resolution efficiency"\nMETRIC resolution_rate\nRANGE 8w\nGROUP week')}
          <div class="about-callout">
            <strong>Persistence model</strong> — Charts are private to the authenticated author unless shared via exports.
            Pin flags (<code>pin_dashboard</code>, <code>pin_finops</code>) control lazy hydration when parent views mount.
            Comments in QCL start with <code>#</code>.
          </div>
        </article>`
    },
    glossary: {
        title: 'SRE glossary',
        breadcrumb: 'Help Center / Glossary',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>SRE &amp; quality engineering glossary</h2>
          <p class="about-lead">Canonical definitions as used inside QA Capsule telemetry, APIs, and Help Center documentation.</p>
          <dl class="about-dl">
            <dt>MTTR (Mean Time To Resolution)</dt>
            <dd>Average minutes from <code>created_at</code> to <code>resolved_at</code> for resolved incidents. Lagging indicator of triage efficiency.</dd>
            <dt>MTTF (Mean Time To Failure)</dt>
            <dd>Mean inter-arrival time between incident timestamps: (t<sub>max</sub> − t<sub>min</sub>) / (N − 1). Surrogate for CI stability frequency.</dd>
            <dt>Flaky test</dt>
            <dd>Non-deterministic failure re-detected within 48 hours of a prior resolution; prefixed <code>[FLAKY]</code> in test name.</dd>
            <dt>Stable failure</dt>
            <dd>Structural regression not classified as flaky; appears in failure-quality analytics as the stable slice.</dd>
            <dt>Fingerprint</dt>
            <dd>SHA-256(test name ∥ error message) used for deduplication and spam control per pipeline run.</dd>
            <dt>Pipeline execution group</dt>
            <dd>UI aggregation keyed by <code>pipeline_run_id</code> (or legacy 2-minute window) representing one CI run with multiple failing tests.</dd>
            <dt>FinOps</dt>
            <dd>Financial operations for engineering — maps CI minutes and investigation time into USD (or selected currency).</dd>
            <dt>Flaky waste</dt>
            <dd>Subset of FinOps impact attributable to flaky-tagged incidents; key metric for deflaking ROI.</dd>
            <dt>QCL</dt>
            <dd>QA Chart Language — declarative chart specification (CHART / METRIC / RANGE / GROUP / PROJECT).</dd>
            <dt>Gateway</dt>
            <dd>CI/CD integration endpoint (project) with API key authentication for webhook ingest.</dd>
            <dt>Resolution rate</dt>
            <dd>Percentage of incidents marked resolved in a cohort; available in dashboard tiles and QCL <code>resolution_rate</code> metric.</dd>
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
      <p class="about-lead">Roles are <strong>not</strong> fully cumulative: Admin and Manager own separate areas. Operators inherit triage permissions; Viewers are read-only.</p>
      <table class="about-table">
        <thead><tr><th>Code</th><th>UI label</th><th>Scope</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
      <div class="about-callout" style="margin-top:20px;">
        <strong>Note:</strong> <code>admin</code> manages <strong>Workspaces</strong>, <strong>IAM</strong>, and <strong>Settings</strong> only. <code>manager</code> has FinOps, workspaces, Chart Studio, gateways, and plugins. <code>operator</code> and <code>viewer</code> use Chart Studio and the dashboard but not FinOps.
      </div>
    </article>`;
}

export function loadAboutView() {
    const nav = document.getElementById('about-topic-nav');
    if (!nav) return;
    if (!nav.dataset.built) {
        nav.dataset.built = '1';
        nav.innerHTML = Object.entries(ABOUT_TOPICS).map(([key, topic]) => `
            <button type="button" class="about-topic-btn" data-topic="${key}" onclick="window.showAboutTopic('${key}')">
                ${TOPIC_ICONS[key] || ''}<span>${topic.title}</span>
            </button>
        `).join('');
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
        bindAboutFormulaCopy(content);
    }
}
