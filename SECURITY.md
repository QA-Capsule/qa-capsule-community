# Security Policy

## Reporting a vulnerability

If you discover a security issue, please report it privately to the repository maintainers. Do not open a public issue for sensitive findings.

## Production deployment checklist

- Set `QACAPSULE_JWT_SECRET` to a strong random value (never use defaults).
- Set `QACAPSULE_MCP_TOKEN` — required outside `APP_ENV=development`.
- Change the default `admin` password immediately after first login.
- Terminate TLS at a reverse proxy (nginx, Traefik, cloud load balancer).
- Restrict network access to `/metrics` and `/mcp` at the edge.
- Store integration secrets (Jira, Slack, GitHub) in environment variables, not in committed plugin JSON files.
- Use Docker named volumes and schedule regular backups of `qacapsule.db` and artifact storage.

## Supported versions

| Version        | Supported |
|----------------|-----------|
| 1.0.17-beta    | Yes       |