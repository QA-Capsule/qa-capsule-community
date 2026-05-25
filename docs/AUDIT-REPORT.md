---
icon: material/clipboard-check
---

# Code audit report (Community)

Summary of the repository audit: cleanup, language, security, and documentation.

---

## Files removed or untracked

| Item | Reason |
|------|--------|
| `site/` (MkDocs build) | Generated output — belongs in CI, not git (~120 HTML/JS files) |
| `data/qacapsule.db-wal`, `data/qacapsule.db-shm` | Runtime SQLite sidecars |
| `docs/guides/testing-selfheal-mcp-ui-cli.md` | French-only duplicate — replaced by English guide |

Recommended: keep `file.txt` / `env.txt` out of git (listed in `.gitignore`).

---

## Language and UI hygiene

| Area | Action |
|------|--------|
| Go comments (`db.go`, `auth_handlers.go`) | Translated to English |
| CLI flaky message (`cmd/cli`) | English, no emoji |
| `tests/README.md` | Rewritten in English |
| Web UI | Replaced decorative symbols on buttons (Close / Refresh text) |
| Login | Community edition (no PRO / license paywall on login) |

---

## Security findings

| Finding | Severity | Remediation |
|---------|----------|-------------|
| Real SMTP credentials in `config.yaml` | **High** | Cleared; use `config.yaml.example` + UI/env |
| Default JWT dev secret when `APP_ENV=development` | Medium | Documented; require `QACAPSULE_JWT_SECRET` in prod |
| WebSocket `CheckOrigin: true` (all origins) | Medium | Terminate TLS and restrict origins at reverse proxy |
| `/metrics` without auth | Low | Internal network only |
| `/api/login` no rate limit | Low | Add proxy rate limiting |
| Shell plugins disabled | Positive | Go integrations only |

Details: [Security & authentication](setup/security-authentication.md).

---

## Documentation updates

| New / updated | Path |
|---------------|------|
| Feature catalog | `reference/feature-catalog.md` |
| Security & JWT | `setup/security-authentication.md` |
| Utility binaries | `reference/utility-binaries.md` |
| MCP & self-healing testing | `guides/mcp-self-healing-testing.md` |
| Editions | `guides/editions-community-enterprise.md` |
| Health API | `api/operations-health.md` |
| README (root) | English, full quick start + doc map |
| `mkdocs.yml` | Navigation extended |

Typography: unchanged compact theme in `docs/stylesheets/extra.css` (Roboto 13px body, smaller code).

---

## Build verification

```bash
go build ./cmd/qacapsule/...
go build ./cmd/cli/...
```

Regenerate published docs:

```bash
mkdocs build   # output: site/ (gitignored)
```
