/**
 * Help Center — available to all roles
 */
import { FINOPS_METRICS_DOC_HTML } from './finops-metrics-doc.js';
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
        <article class="about-doc">
          <h2>QA Flight Recorder</h2>
          <p class="about-lead">An SRE control plane that ingests CI failures, correlates alerts, detects flaky tests, and connects quality signals to FinOps metrics.</p>
          <div class="about-callout">
            <strong>Typical workflow</strong>
            <ol>
              <li>Admin or Operator provisions a CI/CD gateway with an API key.</li>
              <li>Pipeline uploads JUnit XML or JSON webhooks on failure.</li>
              <li>Operators triage and resolve incidents from the dashboard.</li>
              <li>Managers review FinOps trends, exports, and custom charts.</li>
            </ol>
          </div>
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
        <article class="about-doc">
          <h2>QA Chart Language (QCL)</h2>
          <p class="about-lead">Managers can build custom charts in <strong>Chart Studio</strong>, save them to a personal library, and pin them to the <strong>Dashboard</strong> or <strong>FinOps</strong> views.</p>
          <pre class="about-formula">CHART line "Weekly FinOps cost"
METRIC finops_cost
RANGE 12w
GROUP week
PROJECT my-service</pre>
          <h3>Directives</h3>
          <ul>
            <li><code>CHART</code> — line | bar | doughnut (+ optional title in quotes)</li>
            <li><code>METRIC</code> — incidents, flaky, resolved, mttr, finops_cost, finops_flaky_cost, ci_minutes</li>
            <li><code>RANGE</code> — 7d, 12w, 1y</li>
            <li><code>GROUP</code> — week | project</li>
            <li><code>PROJECT</code> — optional gateway filter</li>
          </ul>
          <div class="about-callout">
            <strong>Saving &amp; pinning</strong> — Click <em>Save</em> in Chart Studio, then enable <em>Pin to Dashboard</em> or <em>Pin to FinOps</em>. Pinned charts refresh automatically when you open those sections.
          </div>
        </article>`
    },
    glossary: {
        title: 'SRE glossary',
        breadcrumb: 'Help Center / Glossary',
        html: `
        <article class="about-doc">
          <h2>Glossary</h2>
          <dl class="about-dl">
            <dt>MTTR</dt><dd>Mean Time To Resolution — average time to mark an incident resolved.</dd>
            <dt>MTTF</dt><dd>Mean Time To Failure — average interval between recorded failures.</dd>
            <dt>Flaky test</dt><dd>Intermittent failure tagged <code>[FLAKY]</code> after re-failing within 48h of resolution.</dd>
            <dt>Fingerprint</dt><dd>SHA-256 hash of test name and error message for deduplication.</dd>
            <dt>FinOps</dt><dd>Practice of aligning CI/cloud spend with engineering quality outcomes.</dd>
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
    }
}
