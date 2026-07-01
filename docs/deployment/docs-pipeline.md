---
icon: material/rocket-launch
---

# Documentation Deployment Pipeline

This site is built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) and deployed automatically to **GitHub Pages** via GitHub Actions.

---

## Workflow File

`.github/workflows/docs.yml`

### Triggers

| Event | When |
|---|---|
| `push` to `main` | Changes under `docs/**`, `mkdocs.yml`, or the workflow file |
| `workflow_dispatch` | Manual run from the Actions tab |

### Jobs

1. **build** — Installs Python 3.12, runs `mkdocs build --strict`, uploads `site/` artifact.
2. **deploy** — Deploys artifact to GitHub Pages via `actions/deploy-pages@v4`.

---

## One-Time Repository Setup

For the workflow to succeed, enable GitHub Pages with the **GitHub Actions** source:

1. Open your repository on GitHub.
2. Go to **Settings → Pages**.
3. Under **Build and deployment → Source**, select **GitHub Actions**.
4. Save.

!!! warning "Do not use the gh-pages branch source"
    The old `mkdocs gh-deploy` method pushes to a `gh-pages` branch. The current workflow uses the official Pages artifact API and requires the **GitHub Actions** source.

---

## Local Development

```bash
# Install dependencies
pip install -r docs/requirements.txt

# Serve with live reload
mkdocs serve

# Open http://127.0.0.1:8000
```

### Strict build (same as CI)

```bash
mkdocs build --strict
```

Fix any broken links or missing files before pushing to `main`.

### Mermaid diagrams show as raw code

MkDocs Material only renders ` ```mermaid ` blocks when `mkdocs.yml` configures `pymdownx.superfences` with a `mermaid` custom fence (`class: mermaid`). This repo includes that setting. After changing `mkdocs.yml`, restart `mkdocs serve` and hard-refresh the browser (Ctrl+F5). Requires `mkdocs-material>=9.5`.

---

## Project Structure

```
docs/
├── index.md                    # Home page
├── setup/                      # Installation & config
├── integration/                # CI/CD guides
├── guides/                     # Dashboard & operations
├── plugins/                    # Plugin engine
├── api/                        # REST API reference
├── deployment/                 # This file
└── requirements.txt            # Python dependencies

mkdocs.yml                      # Site configuration & navigation
.github/workflows/docs.yml      # CI/CD deploy workflow
```

---

## Custom Domain (Optional)

To use a custom domain (e.g. `docs.yourcompany.com`):

1. Add a `CNAME` file in `docs/` with your domain.
2. Configure DNS `CNAME` record pointing to `qa-capsule.github.io`.
3. Enable **Enforce HTTPS** in GitHub Pages settings.

---

## Troubleshooting Failed Deployments

### Deploy job fails with HTTP 404 (most common)

If the **build** job succeeds but **deploy** fails with:

```text
Failed to create deployment (status: 404)
Ensure GitHub Pages has been enabled
```

GitHub Pages is **not enabled** on the repository (or the source is not **GitHub Actions**). The MkDocs build is fine; only the Pages API rejects the deployment.

**Fix (repository admin):**

1. Open **Settings → Pages** on GitHub.
2. Under **Build and deployment → Source**, choose **GitHub Actions** (not *Deploy from a branch*).
3. Save, then re-run **Deploy Documentation** from the Actions tab.

For organization repos, an org owner may also need **Organization settings → Member privileges → Pages** set to allow Pages for member repositories.

### Other errors

| Error | Cause | Fix |
|---|---|---|
| `Resource not accessible by integration` | Pages permissions missing | Workflow must have `pages: write` and `id-token: write` |
| `No GitHub Pages deployment found` | Pages source not set to Actions | Settings → Pages → Source → GitHub Actions |
| `mkdocs: command not found` | Dependencies not installed | Ensure `pip install -r docs/requirements.txt` runs before build |
| Strict mode warning | Broken internal link | Run `mkdocs build --strict` locally and fix links |
| Site not updating | Path filter skipped workflow | Push a change under `docs/` or run `workflow_dispatch` |
| 404 on site URL | Wrong `site_url` in mkdocs.yml | Match `site_url` to your actual Pages URL |

---

## Manual Deploy Trigger

If you need to redeploy without code changes:

1. Go to **Actions** tab on GitHub.
2. Select **Deploy Documentation**.
3. Click **Run workflow** → **Run workflow**.
