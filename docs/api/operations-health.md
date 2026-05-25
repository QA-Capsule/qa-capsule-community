---
icon: material/heart-pulse
---

# Operations & health API

SRE endpoints for orchestrators (Kubernetes, Docker Compose health checks, Prometheus).

---

## `GET /healthz`

**Liveness** — process is running.

```json
{"status":"ok"}
```

- No authentication
- Supports `HEAD`

---

## `GET /readyz`

**Readiness** — database and artifact storage are usable.

Success (`200`):

```json
{"status":"ready","checks":{"database":"ok","storage":"ok"}}
```

Failure (`503`):

```json
{"status":"not_ready","checks":{"database":"...","storage":"..."}}
```

Docker Compose example:

```yaml
healthcheck:
  test: ["CMD", "curl", "-fsS", "http://127.0.0.1:9000/healthz"]
```

Use `/readyz` when you need DB availability before routing traffic.

---

## `GET /metrics`

Prometheus-compatible text metrics (process uptime, ingest queue depth, etc.).

- No authentication today — **restrict at network edge** in production
- Scrape from internal network only

---

## Related

- [Docker deployment](../setup/docker.md)
- [Feature catalog](../reference/feature-catalog.md)
