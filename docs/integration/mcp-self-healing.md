# MCP Self-Healing — Guide complet

## Réponses rapides

| Question | Réponse |
|----------|---------|
| Où est le MCP server ? | **Intégré dans QA Capsule**, même process, même port — endpoint `POST /mcp` |
| Comment le configurer ? | Variable d'env `QACAPSULE_MCP_TOKEN` + config Cursor `~/.cursor/mcp.json` |
| Quel modèle AI choisir ? | **Ton modèle dans Cursor** (Claude, GPT-4…) — QA Capsule n'a aucun LLM interne |
| Outil MCP externe ? | Dans Cursor, tu combines QA Capsule MCP + `filesystem` MCP + `github` MCP |

---

## 1. Où est le MCP server ?

Le MCP server est **déjà dans QA Capsule**. Rien à installer séparément.

```
QA Capsule (port 9000)
├── GET  /              → UI web
├── POST /api/webhooks/upload  → ingest JUnit XML
├── GET  /api/healing/insights → Self-Healing Hub
└── POST /mcp           → ← MCP server JSON-RPC 2.0
                              (accessible dès le lancement de l'app)
```

Quand tu lances QA Capsule (`docker compose up` ou `go run`), le MCP server
démarre automatiquement. Il n'y a pas de commande séparée.

---

## 2. Configuration côté serveur

### Étape 1 — Générer un token MCP

```bash
# Linux/macOS
openssl rand -hex 32
# → ex: a3f8c2d1e7b9044f6a2d0e5c8b1f3a7d9e2c4b6a8d0f2e4c6a8b0d2f4e6c8a0b

# Windows PowerShell
[System.Web.Security.Membership]::GeneratePassword(48, 8)
```

### Étape 2 — Injecter le token dans QA Capsule

**Via Docker (recommandé) :**

Crée/modifie `.env` à la racine du repo :
```env
QACAPSULE_JWT_SECRET=ton-secret-jwt-fort
QACAPSULE_MCP_TOKEN=a3f8c2d1e7b9044f6a2d0e5c8b1f3a7d9e2c4b6a8d0f2e4c6a8b0d2f4e6c8a0b
GITHUB_TOKEN=ghp_xxxxxxxxxxxx   # pour create_remediation_pr
```

Le `docker-compose.yml` injecte ces variables automatiquement :
```bash
docker compose up -d --build
```

**Via `config.yaml` (dev local) :**
```yaml
telemetry:
    webhook_token: "ton-token-mcp-ici"
```

**Priorité de lecture :**
1. Variable d'env `QACAPSULE_MCP_TOKEN` (prioritaire)
2. Fallback sur `config.yaml → telemetry.webhook_token`
3. Si les deux sont vides → **aucune auth** (dev local uniquement)

### Vérification

```bash
# Sans token (dev)
curl -X POST http://localhost:9000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'

# Avec token (prod)
curl -X POST http://localhost:9000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ton-token" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Réponse attendue :
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {"name": "detect_locator_failures", ...},
      {"name": "record_healing_intervention", ...},
      ...
    ]
  }
}
```

---

## 3. Configuration côté Cursor (client MCP)

### Étape 1 — Ouvrir la config MCP Cursor

Deux options :

- **Config globale** : `~/.cursor/mcp.json` (s'applique à tous tes projets)
- **Config projet** : `.cursor/mcp.json` dans ce repo (s'applique à ce projet uniquement)

Un fichier template est fourni à `.cursor/mcp.json.example`.

### Étape 2 — Coller la configuration

```json
{
  "mcpServers": {
    "qa-capsule": {
      "url": "http://localhost:9000/mcp",
      "headers": {
        "Authorization": "Bearer ton-token-mcp"
      }
    }
  }
}
```

Pour une instance distante (staging, prod) :
```json
{
  "mcpServers": {
    "qa-capsule": {
      "url": "https://qa-capsule.ton-domaine.com/mcp",
      "headers": {
        "Authorization": "Bearer ton-token-mcp"
      }
    }
  }
}
```

### Étape 3 — Vérifier dans Cursor

`Cursor → Settings (⌘,) → MCP` — tu dois voir `qa-capsule` avec un point vert et les 9 outils listés :

```
qa-capsule  ●  connected
  detect_locator_failures
  record_healing_intervention
  get_incident_context
  propose_healing
  submit_healing_patch
  create_remediation_pr
  list_failed_incidents
  get_flaky_tests
  resolve_incident
```

---

## 4. Choisir le modèle AI

QA Capsule **n'a aucun LLM interne**. Il expose un contexte structuré via MCP.
C'est **ton modèle dans Cursor** qui raisonne dessus.

```
Cursor (Claude Sonnet 4.5)  ←── ton modèle
        │
        │  MCP JSON-RPC
        ▼
QA Capsule /mcp              ←── données structurées, pas d'IA
  get_incident_context()     → retourne logs, stack, sélecteur
  detect_locator_failures()  → retourne la liste des locators cassés
```

### Quel modèle choisir ?

Dans `Cursor → Settings → Models` :

| Modèle | Recommandé pour | Note |
|--------|----------------|------|
| **Claude Sonnet 4.5** | Healing quotidien | Bon équilibre vitesse/qualité |
| **Claude Opus 4** | Cas complexes, multi-fichiers | Plus lent mais plus précis sur le code |
| **GPT-4o** | Alternative | Fonctionne bien avec les outils MCP |

> **Conseil** : commence avec Claude Sonnet 4.5. Si le locator est dans un
> composant complexe (shadow DOM, iframe, microfrontend), passe sur Opus.

---

## 5. Utiliser des outils MCP externes (multi-MCP)

Pour qu'un agent AI réalise un healing complet, il a besoin de **plusieurs outils MCP** :

```
Cursor (agent AI)
│
├── qa-capsule MCP   → contexte de l'incident, enregistrement
├── filesystem MCP   → lit les fichiers de test dans le repo
└── github MCP       → crée la PR avec le fix (alternatif à create_remediation_pr)
```

### Configuration multi-MCP complète

```json
{
  "mcpServers": {

    "qa-capsule": {
      "url": "http://localhost:9000/mcp",
      "headers": {
        "Authorization": "Bearer ton-token-mcp"
      }
    },

    "filesystem": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-filesystem",
        "/chemin/absolu/vers/ton/repo/de/tests"
      ]
    },

    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxxxxxxxxxxx"
      }
    }

  }
}
```

**Installation des serveurs MCP locaux :**

```bash
# Vérifie que Node.js est installé
node --version   # >= 18

# Les serveurs sont téléchargés automatiquement par npx
# Pas de npm install global nécessaire
```

### Ce que chaque MCP apporte

| MCP | Outils utilisés | Rôle dans le healing |
|-----|----------------|---------------------|
| `qa-capsule` | `detect_locator_failures`, `get_incident_context`, `record_healing_intervention`, `create_remediation_pr` | Détecte le problème, enregistre le fix |
| `filesystem` | `read_file`, `list_directory` | Lit le fichier test pour trouver le locator à changer |
| `github` | `create_pull_request`, `get_file_contents` | Crée la PR (alternative au `create_remediation_pr` de QA Capsule) |

---

## 6. Session de healing complète — exemple concret

Voici comment se déroule une vraie session dans Cursor après un run CI qui échoue.

### Situation

```
Run GitHub Actions #12345678
❌ test_checkout_flow     — locator('#stripe-pay-button') timeout
❌ test_add_to_cart       — locator('#add-cart-btn') not found
✅ test_product_listing   — OK
```

### Session Cursor

**Toi :**
> Le run CI 12345678 a des échecs de locator. Analyse et corrige les sélecteurs cassés.

**Claude (via QA Capsule MCP) :**

```
Appel : detect_locator_failures(run_id="12345678")
← Retourne :
  [
    { incident_id: 42, test: "test_checkout_flow", locator: "#stripe-pay-button" },
    { incident_id: 43, test: "test_add_to_cart",   locator: "#add-cart-btn"      }
  ]
```

```
Appel : get_incident_context(incident_id=42)
← Retourne :
  error_category: "locator"
  selector_hint:  "locator('#stripe-pay-button')"
  suggested_actions: ["Replace brittle CSS/XPath with role-based selectors..."]
  mcp_prompt: "1. Call get_incident_context... 2. Read test source... 3. Apply fix..."
```

```
Appel : filesystem/read_file("tests/playwright/checkout.spec.js")
← Claude lit le fichier et trouve :
  ligne 34: await page.locator('#stripe-pay-button').click();
  → Il inspecte le HTML du composant et voit que l'ID a changé en data-testid="checkout-cta"
```

**Claude :**
> J'ai trouvé le problème. Le bouton Stripe a été renommé lors du refactor de la page checkout.
> 
> **Fix à appliquer :**
> - `#stripe-pay-button` → `[data-testid="checkout-cta"]` (confiance: 92%)
> - `#add-cart-btn` → `[data-testid="add-to-cart"]` (confiance: 88%)
>
> Je crée la PR ?

**Toi :** Oui

```
Appel : record_healing_intervention(
  incident_id=42,
  original_locator="#stripe-pay-button",
  healed_locator="[data-testid='checkout-cta']",
  explanation="ID renommé lors du refactor Stripe checkout, data-testid stable",
  confidence=0.92,
  agent_source="cursor_claude"
)

Appel : create_remediation_pr(
  repo="mon-org/mon-repo",
  file_path="tests/playwright/checkout.spec.js",
  code="...fichier complet avec les deux locators corrigés..."
)
← Retourne: { pr_url: "https://github.com/mon-org/mon-repo/pull/456" }
```

**Claude :**
> PR créée : https://github.com/mon-org/mon-repo/pull/456
> 
> Les deux incidents sont marqués "MCP Healed" dans QA Capsule.

---

## 7. Accès depuis GitHub Actions (MCP dans CI)

Tu peux aussi appeler le MCP directement depuis un pipeline CI pour des
workflows avancés (ex: un agent AI qui corrige automatiquement les tests après chaque échec).

```yaml
- name: AI Healing — analyse et PR automatique
  if: failure()
  env:
    QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
    MCP_TOKEN: ${{ secrets.QA_CAPSULE_MCP_TOKEN }}
  run: |
    # Appel JSON-RPC direct au MCP
    curl -s -X POST "$QA_CAPSULE_URL/mcp" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $MCP_TOKEN" \
      -d '{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
          "name": "detect_locator_failures",
          "arguments": {
            "run_id": "${{ github.run_id }}"
          }
        }
      }' | tee mcp-failures.json || true

    echo "MCP locator failures detected:"
    cat mcp-failures.json
```

> Ce pattern est utile pour des pipelines qui notifient un système externe
> ou alimentent un tableau de bord. La correction automatique par LLM en CI
> nécessite un agent externe (ex: Cursor SDK, LangChain, etc.) — pas natif ici.

---

## 8. Référence — Variables d'environnement

| Variable | Où ? | Rôle |
|----------|------|------|
| `QACAPSULE_MCP_TOKEN` | Serveur QA Capsule | Token d'auth pour `POST /mcp`. Vide = pas d'auth (dev). |
| `QACAPSULE_JWT_SECRET` | Serveur QA Capsule | Signature JWT pour l'UI web (obligatoire en prod). |
| `GITHUB_TOKEN` / `GITHUB_PAT` | Serveur QA Capsule | Nécessaire pour `create_remediation_pr`. |
| `QACAPSULE_DATA_DIR` | Serveur QA Capsule | Dossier SQLite + artifacts (défaut : `./data`). |

Côté CI (GitHub Actions secrets) :

| Secret | Utilisé par |
|--------|------------|
| `QA_CAPSULE_URL` | Tous les pipelines |
| `QA_CAPSULE_MCP_TOKEN` | Step "AI Healing" avancé (optionnel) |
| `QA_CAPSULE_API_*_KEY` | Chaque pipeline framework (upload + healing gate) |

---

## 9. Référence — Outils MCP disponibles

| Outil | Auth | Description |
|-------|------|-------------|
| `detect_locator_failures` | Token MCP | Trouve les sélecteurs cassés d'un run CI |
| `record_healing_intervention` | Token MCP | Enregistre un fix MCP (original → healed) |
| `get_incident_context` | Token MCP | Contexte complet : logs, stack, selector, actions suggérées |
| `propose_healing` | Token MCP | Hints rule-based sans LLM + prompt prêt pour l'agent |
| `submit_healing_patch` | Token MCP | Enregistre un patch pour audit et traçabilité |
| `create_remediation_pr` | Token MCP + `GITHUB_TOKEN` | Crée une PR GitHub avec le fix |
| `list_failed_incidents` | Token MCP | Liste les incidents ouverts (filtrable par projet) |
| `get_flaky_tests` | Token MCP | Tests flaky avec fingerprint et taux d'échec |
| `resolve_incident` | Token MCP | Marque un incident résolu |

---

## 10. FAQ

**Q : Le MCP server nécessite-t-il un process séparé ?**  
R : Non. Il tourne dans le même process que QA Capsule, sur le même port.

**Q : Puis-je utiliser le MCP sans Cursor ?**  
R : Oui. Tout client MCP compatible JSON-RPC 2.0 fonctionne : VS Code Copilot,
Continue.dev, ou même un script curl/Python. Le protocole est standard.

**Q : QA Capsule consomme-t-il des tokens LLM ?**  
R : Non. QA Capsule ne fait aucun appel LLM. Les tokens sont consommés côté
client (Cursor/Claude) quand le modèle traite les réponses MCP.

**Q : Le token MCP est-il le même que la clé API projet ?**  
R : Non. Ce sont deux choses distinctes :
- **Clé API projet** (`X-API-Key`) → upload JUnit XML, Healing Gate CI
- **Token MCP** (`Authorization: Bearer`) → accès aux outils MCP depuis l'IDE

**Q : Que se passe-t-il si `QACAPSULE_MCP_TOKEN` est vide ?**  
R : Le endpoint `/mcp` accepte toutes les requêtes sans vérifier l'auth.
C'est acceptable en développement local, dangereux en production.

**Q : La PR créée par `create_remediation_pr` est-elle mergée automatiquement ?**  
R : Non. La PR est ouverte, le développeur la review et merge manuellement.
C'est intentionnel — l'humain garde le contrôle final.
