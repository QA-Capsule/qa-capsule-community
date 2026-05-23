---
icon: material/bell-alert
---

# Opsgenie (Atlassian)

<div align="center" class="integration-hero">
  <img src="../assets/integrations/opsgenie.png" alt="Opsgenie logo">
</div>

Crée une alerte via `POST /v2/alerts` (header `Authorization: GenieKey …`).

| | |
|---|---|
| **Manifest** | `plugins/opsgenie/opsgenie-alert.json` |
| **Type** | `opsgenie` |

---

=== "Côté QA Capsule"

    | Variable | Obligatoire | Défaut |
    |----------|-------------|--------|
    | `OPSGENIE_API_KEY` | **Oui** | — |
    | `OPSGENIE_API_URL` | Non | `https://api.opsgenie.com` |

    Gateway : **Opsgenie Team** (optionnel, métadonnée future).

    Priorité envoyée : `P1`. Message = résumé incident + description = erreur test.

=== "Côté fournisseur (Opsgenie)"

    ## 1. Clé API

    1. Opsgenie → **Settings** → **Integrations** → **API**
    2. Créer une clé avec droit **Create and Update Alerts**
    3. Copier la clé `GenieKey …`

    ## 2. Équipes et routage

    Associer la clé à l’équipe on-call ; définir politiques d’escalade dans Opsgenie.

    ## 3. Test

    ```bash
    curl -X POST https://api.opsgenie.com/v2/alerts \
      -H "Authorization: GenieKey VOTRE_CLE" \
      -H "Content-Type: application/json" \
      -d '{"message":"QA Capsule test","description":"test","priority":"P3"}'
    ```

---

- [Catalogue](integrations-catalog.md)
