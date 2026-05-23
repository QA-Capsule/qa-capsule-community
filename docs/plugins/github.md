---
icon: fontawesome/brands/github
---

# GitHub Actions

<div align="center" class="integration-hero">
  <img src="../assets/integrations/github.png" alt="GitHub logo">
</div>

Déclenche un workflow GitHub via `POST /repos/{owner}/{repo}/actions/workflows/{id}/dispatches`.

| | |
|---|---|
| **Manifest** | `plugins/github/github-rerun.json` |
| **Type** | `github` |

---

=== "Côté QA Capsule"

    | Variable | Obligatoire | Gateway |
    |----------|-------------|---------|
    | `GITHUB_TOKEN` | **Oui** | PAT fine-grained ou classic |
    | `GITHUB_OWNER` | **Oui** | Owner |
    | `GITHUB_REPO` | **Oui** | Repository |
    | `GITHUB_WORKFLOW_ID` | **Oui** | ID numérique ou nom fichier `ci.yml` |
    | `GITHUB_REF` | Non | Branche (défaut `main`) |

    Inputs dispatch : `triggered_by: qa-capsule`, `incident: <résumé>`.

    Succès : HTTP **204**.

=== "Côté fournisseur (GitHub)"

    ## 1. Personal Access Token

    1. GitHub → **Settings** → **Developer settings** → **Fine-grained tokens** (recommandé)
    2. Repository access : repo cible uniquement
    3. Permissions : **Actions: Read and write**, **Contents: Read** (selon workflow)

    ## 2. Workflow déclenchable

    Le workflow cible doit avoir :

    ```yaml
    on:
      workflow_dispatch:
        inputs:
          triggered_by:
            required: false
          incident:
            required: false
    ```

    ## 3. Trouver WORKFLOW_ID

    - URL Actions du workflow → ID dans l’API
    - Ou utiliser le nom du fichier : `ci.yml`

    ## 4. Test

    ```bash
    curl -X POST -H "Authorization: Bearer TOKEN" \
      -H "Accept: application/vnd.github+json" \
      https://api.github.com/repos/OWNER/REPO/actions/workflows/WORKFLOW_ID/dispatches \
      -d '{"ref":"main","inputs":{"triggered_by":"test"}}'
    ```

---

- [Catalogue](integrations-catalog.md)
