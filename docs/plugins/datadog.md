---
icon: material/chart-line
---

# Datadog

<div align="center" class="integration-hero">
  <img src="../assets/integrations/datadog.png" alt="Datadog logo">
</div>

Publie un **Event** Datadog (`alert_type: error`) sur l’API Events v1.

| | |
|---|---|
| **Manifest** | `plugins/datadog/datadog-event.json` |
| **Type** | `datadog` |

---

=== "Côté QA Capsule"

    | Variable | Obligatoire | Description |
    |----------|-------------|-------------|
    | `DD_API_KEY` | **Oui** | Clé API Datadog |
    | `DD_SITE` | Non | Défaut `datadoghq.com` (EU : `datadoghq.eu`) |

    Gateway : **Datadog Tags** optionnel (ex. `env:ci,team:checkout`) — enrichissement futur côté event.

    URL appelée : `https://api.{DD_SITE}/api/v1/events` avec header `DD-API-KEY`.

=== "Côté fournisseur (Datadog)"

    ## 1. Clé API

    1. Datadog → **Organization Settings** → **API Keys**
    2. Créer une clé dédiée `qa-capsule-integration`

    ## 2. Site / région

    | Région | `DD_SITE` |
    |--------|-----------|
    | US | `datadoghq.com` |
    | EU | `datadoghq.eu` |

    ## 3. Dashboards & monitors

    Les events apparaissent dans **Event Stream** ; optionnel : créer un monitor sur `source:qa-capsule`.

    ## 4. Test

    ```bash
    curl -X POST "https://api.datadoghq.com/api/v1/events" \
      -H "DD-API-KEY: VOTRE_CLE" -H "Content-Type: application/json" \
      -d '{"title":"QA Capsule test","text":"hello","alert_type":"error"}'
    ```

---

- [Catalogue](integrations-catalog.md)
