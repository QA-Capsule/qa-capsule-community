---
icon: material/webhook
---

# Webhook HTTP personnalisé

<div align="center" class="integration-hero">
  <img src="../assets/integrations/webhook.png" alt="Webhook">
</div>

`POST` JSON générique vers votre API interne (ServiceNow, runbook, orchestrateur maison).

| | |
|---|---|
| **Manifest** | `plugins/webhook/custom-webhook.json` |
| **Type** | `webhook` |

Utilisé aussi pour **QA flaky report**, **TestRail**, **Zephyr**, **Xray** (même runner HTTP).

---

=== "Côté QA Capsule"

    | Variable | Obligatoire | Description |
    |----------|-------------|-------------|
    | `WEBHOOK_URL` | **Oui** | URL HTTPS cible |
    | `WEBHOOK_AUTH_HEADER` | Non | Ex. `Bearer xxx` (header Authorization) |

    **Gateway** : **Custom Webhook URL** par projet.

    ## Payload par défaut

    ```json
    {
      "source": "qa-capsule",
      "event": "incident.detected",
      "incident": "test name",
      "error": "message",
      "status": "CRITICAL",
      "action": "AUTO_EVENT:..."
    }
    ```

    Votre API doit répondre **2xx** pour un succès côté Plugin Engine.

=== "Côté fournisseur (votre API)"

    ## 1. Endpoint récepteur

    - Méthode **POST**, `Content-Type: application/json`
    - TLS recommandé (HTTPS)
    - Auth : API key, mTLS, ou IP allowlist du serveur QA Capsule

    ## 2. Idempotence

    Prévoir `fingerprint` ou `incident_id` dans une évolution custom pour éviter doublons.

    ## 3. Test

    ```bash
    curl -X POST https://votre-api.internal/hooks/qa-capsule \
      -H "Content-Type: application/json" \
      -d '{"source":"qa-capsule","event":"test"}'
    ```

---

- [Catalogue](integrations-catalog.md)
