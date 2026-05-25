---
icon: material/shield-lock
---

# Security & authentication

How QA Capsule protects the control plane, CI ingestion, and optional MCP access.

---

## Threat model (summary)

| Asset | Risk | Mitigation |
|---|---|---|
| Admin UI | Session hijack | HTTPS in production, strong JWT secret, short-lived tokens |
| Webhooks | Spoofed ingest | Per-project `X-API-Key`, optional global `webhook_token` |
| SQLite DB | Tampering / theft | Filesystem permissions, Docker named volumes, backups |
| Plugins | RCE | **No shell execution** — Go HTTP integrations only |
| Artifacts | Malware upload | Extension allow-list, 50 MB cap, async scan hook point |
| MCP | Unauthorized reads | Optional `QACAPSULE_MCP_TOKEN` bearer |

---

## JWT (dashboard login)

### Flow

1. `POST /api/login` with `{ "username", "password" }`.
2. Server validates bcrypt hash in `users` table.
3. Response: `{ "token": "<JWT>", "require_password_change": bool }`.
4. UI stores token in `localStorage` key `sre-jwt` and sends `Authorization: Bearer <token>` on API calls.

### Signing key (required in production)

| Source | Priority |
|---|---|
| `security.jwt_secret` in `config.yaml` | 1 |
| `QACAPSULE_JWT_SECRET` environment variable | 2 |
| Built-in dev secret | Only if `APP_ENV=development` (or `dev` / `local`) |

```bash
export QACAPSULE_JWT_SECRET="$(openssl rand -hex 32)"
export APP_ENV=production
```

If neither secret nor development env is set, the server **exits at startup** (`InitJWT` fatal).

### Token contents

| Claim | Meaning |
|---|---|
| `username` | Login name |
| `role` | `admin`, `manager`, `lead`, `observer` (normalized) |
| `require_password_change` | Forces password screen before dashboard |
| `exp` | 24 hours from issue |

Algorithm: **HS256**. Rotate `QACAPSULE_JWT_SECRET` to invalidate all sessions.

### Password policy

- Default bootstrap user: `admin` / `admin` (first login must change password).
- Admin can reset users via UI or `cmd/resetpass`.
- Domain fencing: `security.allowed_domain` blocks new users outside `@company.com`.

---

## Project API keys (CI / CLI)

Each CI/CD gateway has a unique **API key** stored in `projects.api_key`.

```http
POST /api/webhooks/my-project HTTP/1.1
X-API-Key: <project-api-key>
Content-Type: application/json
```

Used for:

- Webhook JSON and JUnit upload
- `GET /api/incidents/check-flaky/{hash}` (CLI)
- Artifact upload without JWT

**Never commit API keys** — use CI secrets (`QACAPSULE_API_KEY`).

---

## Optional webhook shared secret

`telemetry.webhook_token` in `config.yaml` (if set) can be validated on inbound hooks (see deployment config). Prefer per-project keys for isolation.

---

## MCP server (`POST /mcp`)

JSON-RPC style tools for IDE agents (Cursor, etc.):

| Tool | Purpose |
|---|---|
| `get_incident_context` | Incident + CI tags for debugging |
| `get_flaky_tests` | Flaky/quarantine snapshot |

When `QACAPSULE_MCP_TOKEN` is set on the server:

```http
Authorization: Bearer <token>
```

Omit the env var only on **local development** networks.

---

## RBAC (high level)

| Role | IAM | Gateways | Resolve incidents | Plugins / workflow |
|---|---|---|---|---|
| admin | Full | Full | Yes | Full |
| manager | Teams + users (read) | Full | Yes | View FinOps/DORA |
| lead | Team scope | Edit routing | Yes | Plugins + workflow |
| observer | Read | Read | Yes | Read |

Details: [RBAC & teams](rbac-teams.md).

---

## Transport and headers

| Recommendation | Setting |
|---|---|
| TLS termination | Reverse proxy (nginx, Traefik) in production |
| Bind address | Default `:9000` — do not expose publicly without auth |
| WebSocket `/ws` | `CheckOrigin` allows all origins today — restrict at proxy in production |

---

## Secrets checklist (production)

```bash
QACAPSULE_JWT_SECRET=<random 32+ bytes>
QACAPSULE_MCP_TOKEN=<optional>
SMTP_USER / SMTP_PASSWORD via UI or env
SLACK_WEBHOOK_URL, JIRA_API_TOKEN, ... per integration
```

Copy `config.yaml.example` from the repository root to `config.yaml` — **do not** commit real SMTP or JWT values. See [System Configuration](config.md) for field descriptions.

---

## Security audit notes (Community)

| Finding | Severity | Status |
|---|---|---|
| Mailtrap credentials in old `config.yaml` samples | High | **Removed** — use empty SMTP + UI config |
| Dev JWT fallback when `APP_ENV=development` | Medium | Documented; disable in prod |
| WebSocket permissive origin | Medium | Mitigate with reverse proxy |
| SQLite file permissions on bind mounts | Medium | Use Docker named volume or `chmod` |
| No rate limit on `/api/login` | Low | Add reverse-proxy rate limiting |

Report vulnerabilities privately — see root `README.md` security section.
