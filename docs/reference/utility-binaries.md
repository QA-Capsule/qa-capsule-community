---
icon: material/console
---

# Utility binaries

Besides the main server (`cmd/qacapsule`) and developer CLI (`cmd/cli`), the repository ships small **operator tools**.

---

## Server (`qacapsule`)

```bash
go run ./cmd/qacapsule/main.go
# or Docker: docker compose up -d --build
```

| Env | Purpose |
|---|---|
| `QACAPSULE_DATA_DIR` | SQLite + artifacts directory |
| `QACAPSULE_JWT_SECRET` | JWT signing |
| `APP_ENV=development` | Dev JWT fallback (not for production) |
| `QACAPSULE_INGEST_WORKERS` | Async ingest worker count |
| `QACAPSULE_MCP_TOKEN` | Protect `/mcp` |

---

## Developer CLI (`qacapsule` / `cmd/cli`)

```bash
go build -o bin/qacapsule ./cmd/cli
bin/qacapsule run [flags] -- <your test command>
```

Documented in [Artifacts & CLI](../guides/artifacts-and-cli.md).

---

## JUnit agent (`cmd/agent`)

Uploads a JUnit XML file to the webhook upload endpoint.

```bash
go run ./cmd/agent/main.go -h
```

Typical CI usage: produce `results.xml`, then agent POSTs to `POST /api/webhooks/upload` with project API key.

See [JUnit XML upload](../integration/junit-xml-upload.md).

---

## listusers (`cmd/listusers`)

Lists users from the SQLite database (offline admin tool).

```bash
go run ./cmd/listusers/main.go
go run ./cmd/listusers/main.go /path/to/qacapsule.db
```

Respects `QACAPSULE_DATA_DIR` and project root detection via `core.EnsureProjectRoot()`.

---

## resetpass (`cmd/resetpass`)

Resets a user password when locked out of the UI.

```bash
go run ./cmd/resetpass/main.go admin 'NewSecurePass123!'
go run ./cmd/resetpass/main.go --no-change-required admin 'TempPass1!'
```

Flags:

| Flag | Effect |
|---|---|
| `--no-change-required` / `-n` | Skip forced password change on next login |

---

## Security notes

- Utility commands read **`qacapsule.db` directly** — run only on trusted hosts.
- Do not expose the database file in shared backups without encryption.
- Prefer UI admin reset when SMTP and audit trail matter.
