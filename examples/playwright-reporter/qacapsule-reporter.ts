/**
 * QA Capsule Playwright reporter — fire-and-forget failure posts + trace zip upload.
 *
 * playwright.config.ts:
 *   reporter: [['./examples/playwright-reporter/qacapsule-reporter.ts', {
 *     apiUrl: 'http://localhost:9000',
 *     apiKey: process.env.QACAPSULE_API_KEY,
 *   }]],
 */
import type {
  FullConfig,
  FullResult,
  Reporter,
  TestCase,
  TestResult,
} from '@playwright/test/reporter';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';

type Options = {
  apiUrl?: string;
  apiKey?: string;
  project?: string;
};

export default class QACapsuleReporter implements Reporter {
  private apiUrl: string;
  private apiKey: string;
  private project: string;
  private runId: string;

  constructor(options: Options = {}) {
    this.apiUrl = (options.apiUrl || process.env.QACAPSULE_API_URL || 'http://localhost:9000').replace(/\/$/, '');
    this.apiKey = options.apiKey || process.env.QACAPSULE_API_KEY || '';
    this.project = options.project || process.env.QACAPSULE_PROJECT || 'default';
    this.runId = `pw-${Date.now()}`;
  }

  onBegin(_config: FullConfig) {
    if (!this.apiKey) {
      console.warn('[QA Capsule] Missing QACAPSULE_API_KEY — events disabled.');
    }
  }

  onTestEnd(test: TestCase, result: TestResult) {
    if (!this.apiKey) return;
    const status =
      result.status === 'passed' ? 'PASSED' : result.status === 'skipped' ? 'SKIPPED' : 'FAILED';
    if (status === 'SKIPPED') return;
    if (status === 'PASSED' && result.duration <= 0) return;

    const title = test.titlePath().join(' > ');
    const errMsg = result.error?.message || result.errors?.[0]?.message || '';

    if (status === 'FAILED') {
      void this.fireAndForget(test, result, title, errMsg, 'CRITICAL', (id) => {
        const trace = result.attachments?.find((a) => a.name === 'trace')?.path;
        if (id && trace && fs.existsSync(trace)) {
          void this.uploadTrace(id, trace);
        }
      });
      return;
    }
    void this.fireAndForget(test, result, title, errMsg, 'PASSED');
  }

  private fireAndForget(
    test: TestCase,
    result: TestResult,
    title: string,
    errMsg: string,
    status: string,
    onIncident?: (id: number) => void,
  ) {
    const payload = {
      framework: 'playwright',
      title,
      name: title,
      failure_reason: errMsg,
      error: errMsg,
      status,
      project: this.project,
      browser: test.parent?.project()?.name || '',
      os: process.platform,
      viewport: '',
      execution_time_ms: result.duration,
    };
    fetch(`${this.apiUrl}/api/webhooks/`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': this.apiKey,
        'X-Run-Id': this.runId,
      },
      body: JSON.stringify(payload),
    })
      .then((r) => r.json())
      .then((body: { last_incident_id?: number }) => {
        if (onIncident && body?.last_incident_id) {
          onIncident(body.last_incident_id);
        }
      })
      .catch((e) => console.warn('[QA Capsule] ingest failed', e));
  }

  private async uploadTrace(incidentId: number, tracePath: string) {
    const zipPath = `${tracePath}.qacapsule.zip`;
    try {
      if (process.platform === 'win32') {
        execSync(
          `powershell -NoProfile -Command "Compress-Archive -Path '${tracePath}' -DestinationPath '${zipPath}' -Force"`,
        );
      } else {
        execSync(`zip -j "${zipPath}" "${tracePath}"`);
      }
      const buf = fs.readFileSync(zipPath);
      const form = new FormData();
      form.append('file', new Blob([buf]), path.basename(zipPath));
      fetch(`${this.apiUrl}/api/incidents/${incidentId}/artifacts`, {
        method: 'POST',
        headers: { 'X-API-Key': this.apiKey },
        body: form,
      }).catch((e) => console.warn('[QA Capsule] artifact upload failed', e));
    } catch (e) {
      console.warn('[QA Capsule] trace zip failed', e);
    }
  }

  onEnd(_result: FullResult) {}
}
