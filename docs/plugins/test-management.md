---
icon: material/clipboard-check
---

# Test management (TestRail, Zephyr, Xray)

<div align="center" class="integration-hero">
  <img src="../assets/integrations/testrail.png" alt="TestRail" style="margin-right:16px">
  <img src="../assets/integrations/zephyr.png" alt="Zephyr" style="margin-right:16px">
  <img src="../assets/integrations/xray.png" alt="Xray">
</div>

| Outil | Manifest | Type |
|-------|----------|------|
| TestRail | `testrail/testrail-result.json` | `testrail` |
| Zephyr Scale | `zephyr/zephyr-execution.json` | `zephyr` |
| Xray Cloud | `xray/xray-result.json` | `xray` |

Le moteur community utilise le **runner webhook** : configurez une URL réceptrice par outil.

---

=== "TestRail"

    === "Côté QA Capsule"

        | Variable | Usage |
        |----------|--------|
        | `WEBHOOK_URL` | URL de votre bridge ou middleware |
        | Gateway | **TestRail Webhook URL** |

        Variables documentées pour une future API native :

        | Variable | Description |
        |----------|-------------|
        | `TESTRAIL_URL` | Instance |
        | `TESTRAIL_USER` | Utilisateur API |
        | `TESTRAIL_API_KEY` | Clé API |
        | `TESTRAIL_RUN_ID` | Run actif |
        | `TESTRAIL_CASE_ID` | Cas en échec |

    === "Côté fournisseur (TestRail)"

        1. TestRail → profil utilisateur → **API Key**
        2. Créer un run de test ; noter **Run ID** et **Case ID**
        3. Option : middleware (Azure Function, small service) qui reçoit le JSON QA Capsule et appelle l’API TestRail `add_result_for_case`

=== "Zephyr Scale"

    === "Côté QA Capsule"

        Gateway : **Zephyr Webhook URL** → `WEBHOOK_URL` effectif.

        Variables futures : `ZEPHYR_API_TOKEN`, `ZEPHYR_PROJECT_KEY`, `ZEPHYR_TEST_CASE_KEY`.

    === "Côté fournisseur (Zephyr)"

        1. Jira → Zephyr Scale → **API Keys**
        2. Documenter clés de cas de test (`PROJ-T42`)
        3. Webhook ou automation Jira pour marquer exécution en échec

=== "Xray Cloud"

    === "Côté QA Capsule"

        Gateway : **Xray Webhook URL**.

        Variables futures : `XRAY_CLIENT_ID`, `XRAY_CLIENT_SECRET`, `XRAY_TEST_KEY`.

    === "Côté fournisseur (Xray)"

        1. Xray Cloud → **API Keys** (OAuth client)
        2. Associer tests Jira ; utiliser clés d’exécution
        3. Middleware recommandé jusqu’à runner natif Xray

---

## Payload QA Capsule reçu par votre webhook

```json
{
  "source": "qa-capsule",
  "event": "incident.detected",
  "incident": "[Playwright] login",
  "error": "assertion failed",
  "status": "CRITICAL"
}
```

---

- [Webhook](webhook.md) · [Catalogue](integrations-catalog.md)
