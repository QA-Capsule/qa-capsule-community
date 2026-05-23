---
icon: material/brain
---

# AI RCA & Quarantine

## AI Root Cause Analysis

- Configure provider under **RCA & AI Insights** (Manager: OpenAI or Ollama).
- On each non-flaky failure, analysis runs asynchronously (stub summary if AI is disabled).
- API: `GET /api/rca/insights`, `GET /api/incidents/{id}/rca`, `PUT /api/ai/config`.

Set `OPENAI_API_KEY` or run Ollama at `http://localhost:11434` when using local models.

## Smart Quarantine (DenyList)

- Flaky / repeated failures auto-add tests to `test_quarantine_entries`.
- CI pipeline: `GET /api/ci/quarantine` with header `X-API-Key` (same as webhooks).
- UI: **Quarantine** — manual add/lift (Lead+).

Optional webhook fields: `commit_sha`, `branch` (or headers `X-Commit-Sha`).
