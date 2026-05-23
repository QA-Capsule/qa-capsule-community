---
icon: fontawesome/brands/slack
---

# Slack

<div align="center" class="integration-hero">
  <img src="../assets/integrations/slack.png" alt="Slack logo">
</div>

Envoie une carte d’alerte formatée dans un canal Slack via **Incoming Webhook** (intégration Go native, sans script shell).

| | |
|---|---|
| **Manifest** | `plugins/slack/slack-notifier.json` |
| **Type** | `slack` |
| **API utilisée** | `POST` JSON sur l’URL Incoming Webhook |

---

=== "Côté QA Capsule"

    ## 1. Prérequis rôle

    | Action | Rôle minimum |
    |--------|----------------|
    | Configurer secrets / AUTO-RUN | **Manager** ou **Platform Admin** |
    | Routage canal par pipeline | **Manager** ou **Lead** |
    | Execute (test) | **Lead** |

    ## 2. Variables (serveur Go)

    | Variable | Obligatoire | Où la mettre | Description |
    |----------|-------------|--------------|-------------|
    | `SLACK_WEBHOOK_URL` | **Oui** | Env serveur ou Plugin **Configure** | URL `https://hooks.slack.com/services/...` |
    | `SLACK_CHANNEL` | Non | **CI/CD Gateway** uniquement | Canal par projet (ex. `#alerts-e2e`) |

    ```bash
    export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T00/B00/XXXX"
    ```

    ## 3. Plugin Engine

    1. Ouvrir **Plugin Engine** → carte **Smart Slack Routing**
    2. **Configure** : laisser `SLACK_WEBHOOK_URL` vide si déjà en env
    3. **AUTO-RUN** : laisser **OFF** tant que le webhook n’est pas testé
    4. **Execute** : doit afficher `[SLACK] Delivered to ... (HTTP 200)`

    Manifest actuel :

    ```json
    {
      "integration": "slack",
      "name": "Smart Slack Routing",
      "status": "Active",
      "auto_run": true,
      "trigger_on": ["CRITICAL", "Timeout", "ECONNREFUSED", "FLAKY"],
      "config": { "SLACK_WEBHOOK_URL": "" }
    }
    ```

    ## 4. CI/CD Gateway (routage dynamique)

    1. **CI/CD Gateways** → projet pipeline
    2. **+ Add configuration** → choisir **Smart Slack Routing** (logo Slack)
    3. Champ **Slack Channel** : `#alerts-frontend` ou ID membre `U01234567`
    4. Enregistrer le gateway

    Au runtime, QA Capsule envoie `"channel": "<valeur gateway>"` dans le JSON Slack.

    ## 5. Payload envoyé par QA Capsule

    ```json
    {
      "channel": "#alerts-frontend",
      "attachments": [{
        "color": "#ff4444",
        "title": "SRE Alert: [Playwright] checkout",
        "text": "Error detected by QA Capsule.\n\nTimeout 30000ms",
        "footer": "QA Capsule Remediation Engine"
      }]
    }
    ```

    ## 6. Dépannage QA Capsule

    | Symptôme | Cause | Solution |
    |----------|-------|----------|
    | `[ERROR] SLACK_WEBHOOK_URL not configured` | Secret absent | `export` ou Configure |
    | HTTP 404 | URL révoquée | Recréer webhook côté Slack |
    | Pas de message | AUTO-RUN OFF ou gateway sans Slack | Activer + Add configuration |
    | Mauvais canal | `SLACK_CHANNEL` vide | Remplir dans gateway |

=== "Côté fournisseur (Slack)"

    ## 1. Créer une Slack App

    1. [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From scratch**
    2. Nom : `QA Capsule Bot`
    3. Workspace : votre entreprise

    ## 2. Activer Incoming Webhooks

    1. Menu app → **Incoming Webhooks** → **On**
    2. **Add New Webhook to Workspace**
    3. Choisir un canal par défaut (ex. `#general`) — QA Capsule peut **surcharger** le canal via `SLACK_CHANNEL`
    4. Copier l’URL : `https://hooks.slack.com/services/T…/B…/…`

    ## 3. Permissions / bonnes pratiques

    | Recommandation | Détail |
    |----------------|--------|
    | Canal dédié | `#sre-qa-alerts` par équipe ou produit |
    | Pas de token dans Git | URL = secret ; rotation si fuite |
    | Inviter l’app | Le canal cible doit exister ; bot invité si canal privé |

    ## 4. Vérification côté Slack

    Test manuel :

    ```bash
    curl -X POST -H "Content-Type: application/json" \
      -d '{"text":"QA Capsule test"}' \
      "https://hooks.slack.com/services/VOTRE_URL"
    ```

    Réponse `ok` → Slack accepte le webhook.

    ## 5. Limites Slack

    - Incoming Webhooks : pas de threads avancés sans migrer vers l’API `chat.postMessage` + Bot token (hors scope community)
    - Rate limits Slack standards ; rafales CI importantes → regrouper côté corrélation QA Capsule

---

## Liens

- [Guide deux côtés](configuration-guide.md)
- [Catalogue intégrations](integrations-catalog.md)
