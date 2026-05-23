---
icon: material/bell-ring
---

# PagerDuty

<div align="center" class="integration-hero">
  <img src="../assets/integrations/pagerduty.png" alt="PagerDuty logo">
</div>

Déclenche un événement **Events API v2** (`event_action: trigger`) vers PagerDuty.

| | |
|---|---|
| **Manifest** | `plugins/pagerduty/pagerduty.json` |
| **Type** | `pagerduty` |

---

=== "Côté QA Capsule"

    | Variable | Obligatoire | Description |
    |----------|-------------|-------------|
    | `PAGERDUTY_ROUTING_KEY` | **Oui** | Integration / routing key de l’événement |
    | `PAGERDUTY_API_URL` | Non | Défaut `https://events.pagerduty.com/v2/enqueue` |

    **Gateway** : champ **PagerDuty Routing Key** (surcharge par pipeline).

    **Execute** doit retourner `[PAGERDUTY] Queued (HTTP 200)`.

=== "Côté fournisseur (PagerDuty)"

    ## 1. Service et intégration Events API v2

    1. PagerDuty → **Services** → service on-call cible
    2. **Integrations** → ajouter **Events API V2**
    3. Copier la **Integration Key** (= routing key)

    ## 2. Escalade

    Configurer politiques d’escalade, horaires, et filtres sur le service pour éviter le bruit (QA Capsule envoie `severity: critical`).

    ## 3. Test

    ```bash
    curl -X POST https://events.pagerduty.com/v2/enqueue \
      -H "Content-Type: application/json" \
      -d '{"routing_key":"VOTRE_CLE","event_action":"trigger","payload":{"summary":"QA Capsule test","source":"qa-capsule","severity":"critical"}}'
    ```

---

- [Catalogue](integrations-catalog.md)
