---
icon: material/account-group
---

# RBAC, Teams & Multi-Tenancy

QA Capsule uses hierarchical teams and role-based access control (RBAC) to isolate projects and enforce least-privilege access.

---

## Global Roles

| Role | Dashboard | Resolve | Provision CI | Plugins config | IAM | System settings |
|---|---|---|---|---|---|---|
| **admin** | All projects | Yes | Yes | Yes | Yes | Yes |
| **operator** | Team projects | Yes | Yes | No | No | No |
| **viewer** | Team projects | No | No | No | No | No |

Assign roles when provisioning users under **Users (IAM)**.

---

## Team Hierarchy

Teams are organized as a tree under **Organizations**:

```
Acme Corp (root)
├── Platform Engineering
│   ├── Backend Squad
│   └── Frontend Squad
└── QA Guild
```

### Team roles (per membership)

| Team role | Permissions within team |
|---|---|
| `team_admin` | Manage members, assign projects |
| `team_operator` | View and resolve team incidents |
| `team_member` | View team incidents |

---

## Linking Projects to Teams

When you provision a CI/CD project under **CI/CD Gateways**, you assign a **Team**. This controls:

1. Which operators/viewers can see incidents for that project.
2. Which Slack/Jira/Teams routing variables apply.

Admins see all projects regardless of team membership.

---

## Provisioning Workflow

### 1. Create organization structure

**Organizations** sidebar → create teams and sub-teams.

### 2. Add members

Search users by email → assign team role.

### 3. Provision CI/CD project

**Ingestion** → select provider → assign team → configure routing variables.

### 4. Verify access

Log in as a non-admin user on that team → confirm incidents appear in dashboard.

---

## Domain Fencing

Under **Settings → Network & Security Policy**, restrict login emails:

```
@acme-corp.com
```

Users with other domains are rejected at provisioning time with `403 Forbidden`.

---

## Security Best Practices

1. **Never share API keys** between projects — provision one endpoint per pipeline.
2. **Use `viewer` for developers** who only need read access to failures.
3. **Use `operator` for QA leads** who resolve and provision endpoints.
4. **Reserve `admin` for SRE/DevOps** — maximum 2–3 admin accounts.
5. **Enable domain fencing** in production to block personal email accounts.
6. **Rotate API keys** by deleting and re-provisioning the project endpoint if a key leaks.

---

## JWT Authentication

Dashboard and Incidents API use JWT tokens from:

```
POST /api/login
{ "username": "user@acme.com", "password": "..." }
```

Response:

```json
{ "token": "eyJhbGciOiJIUzI1NiIs..." }
```

Tokens embed `username` and `role` claims. Webhook endpoints use **API keys**, not JWT.
