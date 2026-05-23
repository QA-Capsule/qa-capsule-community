---
icon: material/phone-alert
---

# VictorOps / Splunk On-Call

<div align="center" class="integration-hero">
  <img src="../assets/integrations/victorops.png" alt="Splunk On-Call logo">
</div>

Envoie un incident **CRITICAL** vers l’URL de routage REST Splunk On-Call (ex-VictorOps).

| | |
|---|---|
| **Manifest** | `plugins/victorops/victorops-alert.json` |
| **Type** | `victorops` |

---

=== "Côté QA Capsule"

    | Variable | Obligatoire |
    |----------|-------------|
    | `VICTOROPS_ROUTING_URL` | **Oui** (URL REST complète fournie par Splunk) |

    Configurable par pipeline via gateway **VictorOps Routing URL**.

    Corps JSON : `message_type: CRITICAL`, `entity_display_name: QA Capsule`, `state_message` avec résumé incident.

=== "Côté fournisseur (Splunk On-Call)"

    ## 1. Obtenir l’URL de routage

    1. Splunk On-Call → **Settings** → intégration / endpoint REST
    2. Copier l’URL POST unique (souvent par équipe)

    ## 2. Politique d’alerte

    Associer l’endpoint à la rotation on-call ; ajuster seuils pour éviter flood CI.

    ## 3. Test

    Poster un JSON minimal `message_type` + `state_message` vers l’URL fournie (voir doc Splunk On-Call pour le schéma exact de votre tenant).

---

- [Catalogue](integrations-catalog.md)
