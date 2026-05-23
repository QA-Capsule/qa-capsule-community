---
icon: fontawesome/brands/jira
---

# Jira Software

<div align="center" class="integration-hero">
  <img src="../assets/integrations/jira.png" alt="Jira logo">
</div>

Crée automatiquement un ticket Jira (Bug / Task) via l’API REST `POST /rest/api/2/issue`.

| | |
|---|---|
| **Manifest** | `plugins/jira/jira-ticket.json` |
| **Type** | `jira` |
| **Auth** | Basic (email + API token Atlassian) |

---

=== "Côté QA Capsule"

    ## Variables serveur

    | Variable | Obligatoire | Source QA Capsule |
    |----------|-------------|-------------------|
    | `JIRA_URL` | **Oui** | Env / Configure (ex. `https://company.atlassian.net`) |
    | `JIRA_EMAIL` | **Oui** | Compte service Atlassian |
    | `JIRA_API_TOKEN` | **Oui** | Token Atlassian (jamais en clair dans Git) |
    | `JIRA_ISSUE_TYPE` | Non | Défaut `Bug` |
    | `JIRA_PROJECT_KEY` | **Oui** pour auto | **CI/CD Gateway** ou tag `@SCRUM-42` dans le nom du test |

    ```bash
    export JIRA_URL="https://votre-domaine.atlassian.net"
    export JIRA_EMAIL="sre-bot@company.com"
    export JIRA_API_TOKEN="xxxxxxxx"
    ```

    ## Plugin Engine

    1. **Configure** : optionnel si tout est en env
    2. **Execute** sans `JIRA_PROJECT_KEY` → erreur explicite (comportement voulu)
    3. **AUTO-RUN ON** seulement après un Execute réussi

    ## CI/CD Gateway

    - **Add configuration** → **Jira Auto-Ticketing**
    - Champ **Jira Project Key** : `SCRUM`, `PAY`, etc.

    ## Extraction automatique depuis le test

    Nom de test contenant `@jira-SCRUM-99` ou `@SCRUM-99` → clé issue / projet injectés à l’ingestion.

    ## Payload créé

    ```json
    {
      "fields": {
        "project": { "key": "SCRUM" },
        "summary": "[QA Capsule] checkout payment",
        "description": "Incident from QA Capsule.\n\nTimeout...",
        "issuetype": { "name": "Bug" }
      }
    }
    ```

    ## Dépannage

    | Erreur | Solution |
    |--------|----------|
    | HTTP 401 | Token ou email incorrect |
    | HTTP 400 project | `JIRA_PROJECT_KEY` invalide |
    | Pas de ticket | AUTO-RUN off ou Jira absent du gateway |

=== "Côté fournisseur (Atlassian Jira)"

    ## 1. Compte service

    1. Créer un utilisateur dédié (ex. `sre-bot@company.com`) ou utiliser un compte bot approuvé par l’admin Jira
    2. L’inviter aux projets à surveiller avec permission **Create issues**

    ## 2. API Token

    1. [id.atlassian.com](https://id.atlassian.com) → **Security** → **API tokens**
    2. **Create API token** → copier une seule fois
    3. Stocker dans le secret manager / `export JIRA_API_TOKEN` sur le serveur QA Capsule

    ## 3. Projet Jira

    | Élément | Où le trouver |
    |---------|----------------|
    | **Project Key** | Paramètres projet → ex. `SCRUM` |
    | **Issue type** | Schéma projet (Bug, Task, Story) → aligner `JIRA_ISSUE_TYPE` |

    ## 4. Test API Atlassian

    ```bash
    curl -u "email:API_TOKEN" -H "Content-Type: application/json" \
      -d '{"fields":{"project":{"key":"SCRUM"},"summary":"QA Capsule test","issuetype":{"name":"Task"}}}' \
      "https://VOTRE-DOMAINE.atlassian.net/rest/api/2/issue"
    ```

    HTTP **201** + `id` → configuration fournisseur OK.

---

- [Guide configuration](configuration-guide.md) · [Catalogue](integrations-catalog.md)
