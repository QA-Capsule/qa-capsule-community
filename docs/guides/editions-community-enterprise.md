---
icon: material/package-variant
---

# Community vs Enterprise editions

QA Capsule ships two **compile-time** editions using Go build tags.

---

## Community (default)

- Build: `go build ./cmd/qacapsule` (no tags)
- Login: **Community Edition** badge, no SSO paywall
- SSO section hidden in UI
- `GET /api/sso/status` returns `"edition": "community"`
- All core SRE features: ingest, dashboard, plugins (Go), workflow, RCA, quarantine, FinOps, DORA

---

## Enterprise

- Build: `go build -tags enterprise ./cmd/qacapsule`
- License key stored in `enterprise_config.license_key`
- When license is active:
  - SSO block visible on login (Google Workspace flow)
  - `enterprise_active: true` from `/api/sso/status`
- Same codebase; gating via `core.EditionActive()`

---

## Choosing an edition

| Need | Edition |
|---|---|
| Open-source self-host, full control plane | **Community** |
| SSO + commercial license enforcement | **Enterprise** |

Docker images in this repository default to **Community**.

---

## API discovery

```bash
curl -s http://localhost:9000/api/sso/status | jq .
```

```json
{
  "edition": "community",
  "enterprise_active": false,
  "sso_available": false
}
```

Response header `X-QA-Capsule-Edition` is also set on static assets.
