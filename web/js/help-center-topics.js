/**
 * Extended Help Center topics (workflows, Super-App, operations).
 * Imported by about.js — replaces legacy QCL v1.0 topic.
 */

export const EXTENDED_HELP_TOPICS = {
    workflow: {
        title: 'Visual Workflows',
        breadcrumb: 'Help Center / Visual Workflows',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>Visual Workflow Builder</h2>
          <p class="about-lead">
            Draw <strong>remediation DAGs</strong> per CI/CD gateway. When <em>Enable workflow</em> is saved,
            the graph replaces legacy keyword AUTO-RUN for that pipeline only.
          </p>
          <h3>Node types</h3>
          <table class="about-table">
            <thead><tr><th>Node</th><th>Role</th></tr></thead>
            <tbody>
              <tr><td><strong>Trigger</strong></td><td>Single entry when an alert is ingested (one per graph).</td></tr>
              <tr><td><strong>Condition</strong></td><td>Branches on <code>[FLAKY]</code>/<code>[PERF]</code> tags, status, or error text. <em>Top</em> output = true, <em>bottom</em> = false.</td></tr>
              <tr><td><strong>Action</strong></td><td>Runs one native plugin (Slack, Jira, K8s, …) from the registry — never shell.</td></tr>
            </tbody>
          </table>
          <h3>Typical setup (Lead+)</h3>
          <ol class="about-detail-list">
            <li><strong>Plugin Engine</strong> — Manager enables routing on integrations.</li>
            <li><strong>CI/CD Gateways</strong> — Add configuration (channels, Jira key, …).</li>
            <li><strong>WORKFLOW</strong> — Build graph: Trigger → Condition → Actions.</li>
            <li><strong>Simulate</strong> — Dry-run with sample incident (no external calls).</li>
            <li><strong>Enable workflow</strong> + <strong>Save</strong> — Activates DAG.</li>
          </ol>
          <h3>Toolbar</h3>
          <ul class="about-detail-list">
            <li><strong>Example</strong> — Flaky → Slack (true) / Jira (false).</li>
            <li><strong>Simulate / Run simulation</strong> — Shows path, plugins that would run, skipped reasons.</li>
            <li><strong>Draft</strong> — Save without Enable keeps legacy AUTO-RUN.</li>
            <li><strong>Reset</strong> — Deletes DAG; returns to Legacy mode.</li>
          </ul>
          <div class="about-callout">
            <strong>Allowed plugins</strong> — Only integrations listed in the gateway routing matrix run.
            An empty routing list blocks all workflow actions (by design).
          </div>
        </article>`
    },
    operations: {
        title: 'Operations dashboard',
        breadcrumb: 'Help Center / Operations',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>Telemetry Stream (Operations)</h2>
          <p class="about-lead">Live view of CI failures grouped by <strong>pipeline execution</strong> (<code>X-Run-Id</code>).</p>
          <h3>Toolbar</h3>
          <ul class="about-detail-list">
            <li><strong>Time range</strong> — Filters KPIs and list; auto-refresh adapts to window.</li>
            <li><strong>Search</strong> — Client filter on test name, project, error.</li>
            <li><strong>Project filter</strong> — Limits API fetch to one gateway.</li>
            <li><strong>Status</strong> — All / Active / Resolved.</li>
            <li><strong>Analytics</strong> — Charts panel; <strong>Customize layout</strong> saves your tiles (per user).</li>
          </ul>
          <h3>Pipeline card actions</h3>
          <table class="about-table">
            <thead><tr><th>Action</th><th>Role</th></tr></thead>
            <tbody>
              <tr><td>Resolve execution</td><td>Lead+</td></tr>
              <tr><td>Delete execution</td><td>Manager+</td></tr>
              <tr><td>Export logs / JUnit XML</td><td>Users with dashboard access</td></tr>
            </tbody>
          </table>
          <p>Sub-alerts show per-test logs, flaky badge (<code>[FLAKY]</code>), and individual checkboxes for bulk resolve.</p>
        </article>`
    },
    gateways: {
        title: 'CI/CD Gateways',
        breadcrumb: 'Help Center / Gateways',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>CI/CD Gateways (Ingestion)</h2>
          <p class="about-lead">Each gateway is a <strong>project</strong> with its own API key, routing, and optional workflow DAG.</p>
          <h3>Webhook</h3>
          <pre class="about-code">POST /api/webhooks/
X-API-Key: &lt;gateway-secret&gt;
X-Run-Id: pipeline-123
X-Commit-Sha: abcdef</pre>
          <h3>Workflow column</h3>
          <table class="about-table">
            <thead><tr><th>Badge</th><th>Meaning</th></tr></thead>
            <tbody>
              <tr><td>Legacy</td><td>No DAG — AUTO-RUN + <code>trigger_on</code> on plugins</td></tr>
              <tr><td>Draft</td><td>DAG saved, not enabled</td></tr>
              <tr><td>Active</td><td>DAG enabled — WorkflowEngine runs</td></tr>
            </tbody>
          </table>
          <h3>SRE routing matrix</h3>
          <p>Add configuration rows to set Slack channel, Jira project, Teams webhook, etc. This defines which plugins may run for this gateway.</p>
        </article>`
    },
    rca_quarantine: {
        title: 'AI RCA & Quarantine',
        breadcrumb: 'Help Center / RCA & Quarantine',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>AI Root Cause Analysis</h2>
          <p>Async LLM summaries (OpenAI or Ollama) per failure. Manager configures provider; Lead+ views insights. Does not block webhooks.</p>
          <h3>RCA &amp; AI Insights view</h3>
          <ul class="about-detail-list">
            <li>Browse recent summaries across projects you can access.</li>
            <li>Re-run analysis on an incident after log updates.</li>
            <li>Status: pending, running, completed, skipped, failed.</li>
          </ul>
          <h2>Smart Quarantine</h2>
          <p>Deny-list flaky or broken tests so they stop creating incidents and firing Slack/Jira.</p>
          <ul class="about-detail-list">
            <li><strong>Auto</strong> — Engine may quarantine after flaky tags or repeated failures.</li>
            <li><strong>Manual</strong> — Lead+ adds/lifts tests in Quarantine view.</li>
            <li><strong>CI API</strong> — <code>GET /api/ci/quarantine</code> with same <code>X-API-Key</code> to skip tests in pipeline.</li>
          </ul>
          <div class="about-callout">Quarantined ingests return <code>quarantined_skipped</code> in webhook JSON — no incident row.</div>
        </article>`
    },
    runbooks_dora: {
        title: 'Runbooks & DORA',
        breadcrumb: 'Help Center / Runbooks & DORA',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>Runbooks</h2>
          <p class="about-lead">One-click <strong>workflow templates</strong> (502 restart, flaky triage, OOM, perf, timeout) validated against the plugin registry.</p>
          <p>Lead+ selects gateway + template → Apply writes and enables <code>sre_workflow_json</code>. Customize afterward in WORKFLOW editor.</p>
          <h2>DORA dashboard (Manager)</h2>
          <table class="about-table">
            <thead><tr><th>Metric</th><th>Source</th></tr></thead>
            <tbody>
              <tr><td>Deployment frequency</td><td>Pipeline runs per day</td></tr>
              <tr><td>Lead time</td><td>Median run start → first incident</td></tr>
              <tr><td>Change failure rate</td><td>Failed runs / total runs</td></tr>
              <tr><td>MTTR</td><td>Mean time to resolve incidents</td></tr>
            </tbody>
          </table>
          <h3>Prometheus webhook</h3>
          <pre class="about-code">POST /api/webhooks/prometheus?project=&lt;gateway-name&gt;
X-API-Key: &lt;key&gt;</pre>
          <p>Correlates external alerts to incidents within ±15 minutes.</p>
        </article>`
    },
    analytics: {
        title: 'Dashboard analytics',
        breadcrumb: 'Help Center / Analytics',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>System analytics &amp; quality</h2>
          <p class="about-lead">
            Built-in KPI tiles and charts on the Operations dashboard — <strong>not</strong> a separate query language.
            Use <strong>Customize layout</strong> to pick metrics, chart types (line, bar, doughnut), colors, and order. Layout is saved per user.
          </p>
          <h3>Common metrics</h3>
          <table class="about-table">
            <thead><tr><th>Metric</th><th>Meaning</th></tr></thead>
            <tbody>
              <tr><td>incidents</td><td>Failure count in time bucket</td></tr>
              <tr><td>flaky / flaky_ratio</td><td>Tests tagged <code>[FLAKY]</code></td></tr>
              <tr><td>resolved / active</td><td>Backlog vs cleared</td></tr>
              <tr><td>mttr</td><td>Mean resolution time (minutes)</td></tr>
              <tr><td>finops_cost</td><td>Modeled USD (Manager FinOps baselines)</td></tr>
            </tbody>
          </table>
          <p>All analytics respect the dashboard <strong>time range</strong> filter and auto-refresh.</p>
          <div class="about-callout">For FinOps formula details see <strong>FinOps metrics</strong> in this Help Center.</div>
        </article>`
    },
    architecture: {
        title: 'Architecture (summary)',
        breadcrumb: 'Help Center / Architecture',
        html: `
        <article class="about-doc about-doc-wide">
          <h2>How QA Capsule works</h2>
          <ol class="about-detail-list">
            <li><strong>Ingest</strong> — Webhook validates API key → dedup by fingerprint + run → optional flaky/perf tags → SQLite insert.</li>
            <li><strong>Remediate</strong> — Visual workflow DAG <em>or</em> legacy AUTO-RUN plugins (async, max 32 concurrent).</li>
            <li><strong>Enrich</strong> — RCA job + quarantine stats (async hooks).</li>
            <li><strong>Observe</strong> — Dashboard, DORA, FinOps, artifacts.</li>
          </ol>
          <h3>Storage</h3>
          <p>SQLite (<code>data/qacapsule.db</code>), plugin manifests in <code>plugins/</code>, artifacts on disk. Single Go process serves API + static UI.</p>
          <h3>Security</h3>
          <ul class="about-detail-list">
            <li>UI: JWT after login (role in claims).</li>
            <li>CI: per-gateway <code>X-API-Key</code> only.</li>
            <li>Teams scope projects via <code>user_teams</code>.</li>
          </ul>
          <p>Full visual schemas (C4, sequences, ER, state machines): MkDocs → <em>Design Schemas &amp; Diagrams</em>. Narrative: <em>System Architecture</em>.</p>
        </article>`
    }
};
