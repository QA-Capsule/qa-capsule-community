# MCP Self-Healing — Guide complet

## Vue d'ensemble

QA Capsule fournit un système de **self-healing framework-agnostique** basé sur le protocole MCP (Model Context Protocol).  
Il fonctionne en **deux modes complémentaires** :

| Mode | Nécessite un LLM ? | Ce que ça fait |
|------|-------------------|----------------|
| **Automatique (Healing Gate)** | Non | Détecte les locator failures, enregistre les interventions, notifie Slack/Teams |
| **Agent AI (Cursor/Claude)** | Oui — ton modèle | Lit le code source, raisonne sur le DOM, applique le fix précis, crée la PR |

---

## Architecture

```
CI Pipeline (any framework)
│
├── 1. Run Tests           → JUnit XML produit
├── 2. Upload Results      → POST /api/webhooks/upload
│                             incidents créés en base
│
└── 3. MCP Healing Gate    → POST /api/healing/gate
          │
          ├── Détecte les failure locator dans les incidents du run
          ├── Enregistre dans locator_healings (original → healed)
          ├── Marque l'incident mcp_healed = 1
          └── Notifie Slack/Teams/Jira si configuré

                    ↕  MCP JSON-RPC  ↕

QA Capsule MCP Server (/mcp)
│
├── detect_locator_failures     ← Cursor appelle ça après /gate
├── record_healing_intervention ← Cursor enregistre son fix
├── get_incident_context        ← Contexte complet (logs, stack, selector)
├── propose_healing             ← Hints rule-based sans LLM
├── submit_healing_patch        ← Soumet le patch pour audit
└── create_remediation_pr       ← Crée la PR GitHub

                    ↕  IDE MCP  ↕

Agent AI externe (Cursor, Copilot, etc.)
     └── Modèle AI = celui de ton IDE (Claude, GPT-4, etc.)
         QA Capsule ne contient aucun LLM interne
```

---

## Mode 1 — Healing Gate (sans LLM, opérationnel immédiatement)

### Ce que fait le Healing Gate

À chaque run CI, après l'upload du JUnit XML, le step "MCP Healing Gate" appelle :

```
POST /api/healing/gate
X-API-Key: <project key>
X-Run-Id:  <github.run_id>
X-Framework: Playwright   # optionnel, améliore la détection
```

Le serveur :
1. Lit tous les incidents du run (`pipeline_run_id = run_id`)
2. Pour chaque incident dont l'`error_message` contient un pattern locator :
   - Extrait le sélecteur cassé (`#stripe-pay-button`, `[data-testid=...]`, etc.)
   - Génère une suggestion heuristique (`#id` → `[data-testid="id"]`)
   - Insère dans `locator_healings`
   - Met à jour `incidents.mcp_healed = 1`
3. Si au moins une intervention : déclenche la notification (Slack/Teams)
4. Retourne un JSON avec le rapport complet

### Détection multi-framework

| Framework | Pattern reconnu | Exemple |
|-----------|----------------|---------|
| **Playwright** | `locator('...')` | `locator('#pay-btn').click: Timeout 3000ms` |
| **Cypress** | `Expected to find element: '...'` | `Expected to find element: '#dashboard'` |
| **Selenium** | `NoSuchElementException`, `Unable to locate element` | `Unable to locate element: {"selector":"#login"}` |
| **Robot Framework** | `Element '...' not found` | `Element '#submit-form' not found` |
| **Générique** | Tout `#id` ou `[data-testid=...]` dans l'erreur | — |

### Format de la notification

Quand le gate détecte une intervention, la notification envoyée ressemble à :

> **[MCP] Locator healing — test_checkout_flow**  
> [MCP Healing Gate] 2 locator intervention(s) détectées dans le run 12345678.

---

## Mode 2 — Agent AI dans Cursor (avec LLM)

### Configuration (une seule fois)

Dans **Cursor → Settings → MCP** (ou `~/.cursor/mcp.json`) :

```json
{
  "mcpServers": {
    "qa-capsule": {
      "url": "https://ton-qa-capsule.example.com/mcp",
      "headers": {
        "Authorization": "Bearer TON_QACAPSULE_MCP_TOKEN"
      }
    }
  }
}
```

Le `QACAPSULE_MCP_TOKEN` est défini côté serveur QA Capsule :
- Variable d'environnement : `QACAPSULE_MCP_TOKEN`
- Ou `config.Telemetry.WebhookToken` dans la config YAML

### Workflow agent AI (Cursor + Claude)

Une fois le MCP configuré, Cursor voit ces outils disponibles :

```
qa-capsule/detect_locator_failures
qa-capsule/record_healing_intervention
qa-capsule/get_incident_context
qa-capsule/propose_healing
qa-capsule/submit_healing_patch
qa-capsule/create_remediation_pr
qa-capsule/list_failed_incidents
qa-capsule/get_flaky_tests
qa-capsule/resolve_incident
```

**Exemple de session Cursor après un run CI qui échoue :**

```
Toi dans Cursor:
  "Le run 12345678 a des échecs — analyse et corrige les locators cassés"

Cursor (Claude) appelle automatiquement:
  1. detect_locator_failures(run_id="12345678")
     → Retourne: [{incident_id: 42, test: "checkout_test", locator: "#stripe-pay-button"}]

  2. get_incident_context(incident_id=42)
     → Retourne: stack trace, logs console, hint sélecteur, suggested_actions

  3. Claude lit tests/playwright/checkout.spec.js depuis le repo
     → Repère la ligne avec #stripe-pay-button
     → Comprend que le bouton a été renommé data-testid="checkout-cta"

  4. record_healing_intervention(
       incident_id=42,
       original_locator="#stripe-pay-button",
       healed_locator="[data-testid='checkout-cta']",
       explanation="ID renommé après refactor Stripe, data-testid stable",
       confidence=0.92
     )

  5. submit_healing_patch(incident_id=42, repo="org/repo", file_path="...", code="...")

  6. create_remediation_pr(repo="org/repo", file_path="...", code="...")
     → PR GitHub créée automatiquement
```

---

## UI — Self-Healing Hub

Dans QA Capsule → **Self-Healing Hub**, tu trouves :

### KPI bar (4 métriques)

| Métrique | Description |
|----------|-------------|
| **Open failures** | Incidents non résolus |
| **Categorized** | Failures avec catégorie détectée (locator, timeout, etc.) |
| **MCP-ready** | Toutes les failures sont MCP-ready (contexte disponible) |
| **MCP interventions** | Nombre d'incidents où le Healing Gate est intervenu |

### Liste des failures (insights)

Chaque carte montre :
- Nom du test
- Badge **"MCP Healed"** (vert) si le gate a déjà enregistré une intervention pour ce test
- Catégorie de l'erreur (`locator`, `timeout`, `assertion`, `network`)
- Bouton **"Context"** → charge le contexte complet (sélecteur, actions suggérées, prompt MCP)
- Bouton **"Copy MCP prompt"** → copie le prompt prêt à coller dans Cursor

### Section "Locator Interventions MCP"

Tableau de toutes les interventions enregistrées :

```
┌─────────────────────────────────────────────────────────────────┐
│  [MCP]  test_checkout_flow                   [mcp_gate]         │
│  INC #42  Playwright  confidence 60%  5m ago                    │
│                                                                 │
│  #stripe-pay-button  →  [data-testid="stripe-pay-button"]       │
│                                                                 │
│  ID selectors are fragile; prefer data-testid or ARIA role.     │
└─────────────────────────────────────────────────────────────────┘
```

- Locator **original** (barré, fond rouge) → Locator **suggéré** (fond vert)
- Score de confiance avec couleur (vert ≥ 70%, orange ≥ 40%, gris sinon)
- Source de l'intervention (`mcp_gate` = automatique, `cursor_mcp` = agent Cursor, etc.)

---

## Secrets GitHub Actions requis

| Secret | Utilisé par |
|--------|------------|
| `QA_CAPSULE_URL` | Tous les pipelines |
| `QA_CAPSULE_API_PLAYWRIGHT_KEY` | Playwright pipeline |
| `QA_CAPSULE_API_CYPRESS_KEY` | Cypress pipeline |
| `QA_CAPSULE_API_PYTEST_KEY` | Pytest pipeline |
| `QA_CAPSULE_API_ROBOT_KEY` | Robot Framework pipeline |
| `QA_CAPSULE_API_SELENIUM_KEY` | Selenium pipeline |
| `QA_CAPSULE_API_NEWMAN_KEY` | Newman pipeline |
| `QA_CAPSULE_API_JUNIT_JAVA_KEY` | JUnit Java pipeline |

> Les clés API de projet se trouvent dans **QA Capsule → Settings → Gateways**.

---

## Pipeline CI — Structure du step Healing Gate

Le step est identique pour tous les frameworks :

```yaml
- name: MCP Healing Gate
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_XXX_KEY }}
  run: |
    curl -s -X POST "$WEBHOOK_URL/api/healing/gate" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Framework: Playwright" \
      | tee healing-report.json || true
    echo "--- MCP Healing Report ---"
    cat healing-report.json 2>/dev/null || true
```

**Pourquoi `|| true` ?** — Le gate ne doit jamais faire échouer le pipeline. Si QA Capsule est indisponible, le CI continue normalement.

**Pourquoi `if: always()` ?** — Le gate doit s'exécuter même si les tests ont échoué (c'est précisément dans ce cas qu'il y a du travail à faire).

### Exemple de réponse JSON

```json
{
  "project": "my-project",
  "run_id": "12345678",
  "locator_failures": 2,
  "interventions": 2,
  "healings": [
    {
      "id": 1,
      "incident_id": 42,
      "run_id": "12345678",
      "framework": "Playwright",
      "original_locator": "#stripe-pay-button",
      "healed_locator": "[data-testid=\"stripe-pay-button\"]",
      "confidence": 0.60,
      "explanation": "ID selectors are fragile; prefer data-testid or ARIA role.",
      "agent_source": "mcp_gate",
      "test_name": "test_checkout_flow",
      "created_at": "2026-05-27T11:42:00Z"
    }
  ],
  "status": "ok"
}
```

---

## API Reference

### `POST /api/healing/gate`

Endpoint CI — Auth par `X-API-Key`.

| Header | Requis | Description |
|--------|--------|-------------|
| `X-API-Key` | Oui | Clé API du projet |
| `X-Run-Id` | Oui | ID du run CI (ex: `${{ github.run_id }}`) |
| `X-Framework` | Non | Hint framework (`Playwright`, `Cypress`, `Selenium`, `RobotFramework`, `JUnit`, `Postman`) |

### `GET /api/healing/locator-interventions`

Endpoint UI — Auth JWT (Observer+).

| Query | Description |
|-------|-------------|
| `project` | Filtre par projet |
| `limit` | Nombre de résultats (défaut 50) |

### `GET /api/healing/insights`

Liste des failures avec `mcp_healed: true/false`.

### `POST /mcp` (JSON-RPC 2.0)

MCP tools disponibles :

| Tool | Description |
|------|-------------|
| `detect_locator_failures` | Détecte les failures locator pour un `run_id` |
| `record_healing_intervention` | Enregistre une intervention MCP (original → healed) |
| `get_incident_context` | Contexte complet d'un incident pour l'agent AI |
| `propose_healing` | Hints rule-based (sans LLM) |
| `submit_healing_patch` | Soumet un patch pour audit |
| `create_remediation_pr` | Crée une PR GitHub avec le fix |
| `list_failed_incidents` | Liste les incidents ouverts |
| `get_flaky_tests` | Liste les tests flaky |
| `resolve_incident` | Marque un incident résolu |

---

## FAQ

**Q : Je n'ai pas de LLM configuré. Le Healing Gate fonctionne quand même ?**  
R : Oui. Le gate fonctionne entièrement sans LLM — il utilise des heuristiques (règles regex) pour détecter et suggérer des corrections. La notification Slack/Teams est envoyée automatiquement.

**Q : Quel modèle AI le MCP utilise-t-il ?**  
R : Aucun. QA Capsule n'a pas de LLM interne. Le MCP *expose le contexte* ; c'est l'agent AI de ton IDE (Claude dans Cursor, GPT-4 dans Copilot, etc.) qui raisonne dessus.

**Q : Puis-je désactiver le Healing Gate pour un pipeline ?**  
R : Oui — retire simplement le step "MCP Healing Gate" du workflow YAML. L'upload normal continue de fonctionner.

**Q : Comment améliorer les suggestions de locator ?**  
R : Connecte Cursor au MCP avec `QACAPSULE_MCP_TOKEN`. L'agent AI lira le code source et proposera le vrai locator corrigé plutôt qu'une heuristique générique.

**Q : La suggestion automatique est-elle toujours correcte ?**  
R : Non — c'est une heuristique (confiance 30–60%). L'objectif est de te donner une direction immédiate et de notifier l'équipe. Pour une correction précise, utilise l'agent Cursor.

**Q : Comment voir toutes les interventions passées ?**  
R : Dans **Self-Healing Hub → Locator Interventions MCP** (section en bas de page). Filtre par gateway si nécessaire.
