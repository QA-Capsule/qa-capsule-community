/**
 * FinOps metrics theory — Help Center documentation
 */
export const FINOPS_METRICS_DOC_HTML = `
<article class="about-doc">
  <h2>About FinOps Metrics</h2>
  <p class="about-lead">
    QA Capsule translates SRE activity (incidents, CI pipelines, investigation time) into financial and
    operational indicators. This document describes the theory, assumptions, and exact formulas
    implemented in <code>/api/metrics</code> and <code>/api/finops</code>.
  </p>

  <h3>1. Baseline configuration variables</h3>
  <p>These parameters are set by <strong>Managers</strong> in FinOps Intelligence:</p>
  <table class="about-table">
    <thead><tr><th>Symbol</th><th>API field</th><th>Unit</th><th>Description</th></tr></thead>
    <tbody>
      <tr><td><em>R<sub>dev</sub></em></td><td><code>dev_hourly_rate</code></td><td>currency/h</td><td>Loaded hourly cost of a QA/SRE engineer.</td></tr>
      <tr><td><em>C<sub>ci</sub></em></td><td><code>ci_minute_cost</code></td><td>currency/min</td><td>Marginal cost of one CI runner minute.</td></tr>
      <tr><td><em>T<sub>pipe</sub></em></td><td><code>avg_pipeline_duration</code></td><td>minutes</td><td>Average pipeline duration per recorded failure.</td></tr>
      <tr><td><em>T<sub>inv</sub></em></td><td><code>avg_investigation_time</code></td><td>minutes</td><td>Average human time to diagnose before resolution.</td></tr>
    </tbody>
  </table>

  <h3>2. Incident-derived counts</h3>
  <ul>
    <li><strong>N</strong> = total incidents (<code>total_incidents</code>)</li>
    <li><strong>N<sub>f</sub></strong> = flaky-tagged incidents (<code>flaky_tests</code>)</li>
    <li><strong>N<sub>s</sub></strong> = stable failures: N − N<sub>f</sub></li>
    <li><strong>N<sub>r</sub></strong> = resolved incidents</li>
  </ul>

  <h3>3. Investigation cost per incident</h3>
  <div class="about-formula">C<sub>invest</sub> = (R<sub>dev</sub> / 60) × T<sub>inv</sub></div>
  <p>Example: R<sub>dev</sub> = 50 USD/h, T<sub>inv</sub> = 30 min → C<sub>invest</sub> = 25 USD per incident.</p>

  <h3>4. CI minutes lost</h3>
  <div class="about-formula">M<sub>total</sub> = N × T<sub>pipe</sub></div>
  <div class="about-formula">M<sub>flaky</sub> = N<sub>f</sub> × T<sub>pipe</sub></div>

  <h3>5. Total financial impact</h3>
  <div class="about-formula">Impact<sub>total</sub> = (M<sub>total</sub> × C<sub>ci</sub>) + (N × C<sub>invest</sub>)</div>

  <h3>6. Flaky waste</h3>
  <div class="about-formula">Impact<sub>flaky</sub> = (M<sub>flaky</sub> × C<sub>ci</sub>) + (N<sub>f</sub> × C<sub>invest</sub>)</div>

  <h3>7. MTTR</h3>
  <p>Average minutes between <code>created_at</code> and <code>resolved_at</code> for resolved incidents (minimum 1 minute).</p>

  <h3>8. Weekly FinOps evolution</h3>
  <p><code>/api/finops/evolution</code> returns per-week: incident volume, flaky count, MTTR, estimated cost, and flaky waste cost.</p>

  <h3>9. Gateway export reports</h3>
  <p><code>/api/finops/export?period=week|month|year</code> produces a CSV per CI gateway (project) with failures, resolution, flaky counts, and USD estimates.</p>
</article>
`;
