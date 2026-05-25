---
icon: material/test-tube
---

# Guide de test — Self-Healing, MCP, UI et CLI

Ce document décrit **exactement** comment valider les fonctionnalités Self-Healing et MCP, ce qui est possible **dans l’interface web**, et ce qui passe par **API**, **Cursor (MCP)** ou **CLI**.

---

## Résumé : UI vs API vs CLI vs MCP

| Fonctionnalité | Interface web (`http://localhost:9000`) | CLI `qacapsule` (`cmd/cli`) | MCP (`POST /mcp`) | API REST directe |
|----------------|----------------------------------------|-----------------------------|-------------------|------------------|
| Ingestion CI / incidents | Oui (Dashboard, Gateways) | Non | Non | Oui (`POST /api/webhooks/...`) |
| RCA / analyse IA | Oui (vue **RCA & AI Insights**) | Non | Non | Oui (`GET/POST .../rca`) |
| Quarantaine / flaky | Oui (vue **Quality / Quarantine**) | Oui (`run` + warning flaky) | Oui (`get_flaky_tests`) | Oui (`/api/ci/quarantine`, `check-flaky`) |
| Contexte incident brut (MCP) | **Non** (pas d’écran dédié) | Non | Oui (`get_incident_context`) | Oui (même données via MCP) |
| Self-healing — proposition de patch | **Non** (pas encore dans l’UI) | Non | Non | Oui (`POST .../healing/propose`) |
| Self-healing — PR GitHub | **Non** | Non | Non | Oui (`POST .../healing/pr`) |
| Santé SRE (`/healthz`, `/readyz`, `/metrics`) | Non | Non | Non | Oui (curl / navigateur) |

**Conclusion :** tu peux tester **une grande partie du parcours** dans l’UI (incidents, RCA, flaky, quarantaine). Le **self-healing** et le **serveur MCP** se testent aujourd’hui surtout via **API** ou **Cursor** — pas via un bouton dans le dashboard.

---

## Prérequis communs

### 1. Démarrer le serveur

```powershell
cd C:\Users\khaba\Desktop\qa-capsule\qa-capsule-community
go run .\cmd\qacapsule\main.go
```

Attendu : `[SERVER] Started on port 9000`.

### 2. Ouvrir l’UI

Navigateur : **http://localhost:9000**

| Compte par défaut | Mot de passe initial |
|-------------------|----------------------|
| `admin` | `admin` |

À la première connexion : l’UI impose le **changement de mot de passe**.

### 3. Variables utiles (optionnel)

```powershell
$env:QACAPSULE_MCP_TOKEN = "dev-mcp-secret"   # pour sécuriser /mcp
$env:OPENAI_API_KEY = "..."                    # RCA + self-healing réels
$env:GITHUB_TOKEN = "..."                      # PR GitHub (healing/pr)
```

---

# Partie A — Tests dans l’interface web

## A.1 — Checklist navigation

| Étape | Menu | Rôle minimum | Critère de succès |
|-------|------|--------------|-------------------|
| Login | Page login | — | JWT stocké, dashboard visible |
| Créer un gateway | **CI/CD Gateways** | Manager / Lead | Projet + **API Key** + URL webhook |
| Voir un échec | **Dashboard** | Observer+ | Carte incident rouge / CRITICAL |
| RCA IA | **RCA & AI Insights** | Observer+ (lecture), Manager (config IA) | Résumé IA ou statut `pending` |
| Flaky / quarantaine | **Quality / Quarantine** | Lead+ (actions) | Liste deny-list, KPIs |
| Plugins | **Plugin Engine** | Lead+ (execute), Manager (auto-run) | Intégrations listées |

---

## A.2 — Préparer des données (100 % UI + webhook)

### Créer un projet (Gateway)

1. **CI/CD Gateways** → **Add configuration** (ou équivalent).
2. Renseigner :
   - **Name** : `demo-ci`
   - **Team** : Root Organization
   - **CI system** : `github` (ou autre, agnostique)
3. **Save** → copier l’**API Key** (ex. `test-api-key-local`).
4. Noter l’URL webhook affichée : `http://localhost:9000/api/webhooks/demo-ci` (le segment final = nom/id du projet selon votre config).

### Ingérer un échec (webhook — hors UI mais nécessaire pour avoir un incident)

L’UI ne simule pas encore un webhook ; utilise PowerShell **une fois** :

```powershell
$body = @{
  tests = @(
    @{
      name   = "checkout should complete"
      status = "failed"
      error  = "TimeoutError: waiting for selector '#pay-button'"
      stack  = "at step (checkout.spec.ts:42:12)"
    }
  )
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Method POST -Uri "http://localhost:9000/api/webhooks/demo-ci" `
  -Headers @{
    "X-API-Key" = "test-api-key-local"
    "X-Run-Id"  = "ui-run-001"
    "X-Execution-Env" = "INTEGRATION"
  } -ContentType "application/json" -Body $body
```

Attendu : `"status": "queued"` (ingestion asynchrone). Attendre **2–3 secondes**, puis rafraîchir le dashboard.

### Vérifier sur le Dashboard

1. **Dashboard** → filtre projet `demo-ci`.
2. Vérifier :
   - Nom du test
   - Message d’erreur
   - Logs (console / error)
   - Groupe par `pipeline_run_id` = `ui-run-001` si affiché

**Critère :** l’incident est visible sans appeler l’API manuellement pour la lecture.

---

## A.3 — Tester la RCA (proche du self-healing, côté UI)

Le self-healing utilise le **même moteur LLM** que la RCA, mais **pas la même page**.

### Activer l’IA (Manager / Platform Admin)

1. **RCA & AI Insights**.
2. Panneau **AI Provider** :
   - Provider : `OpenAI` (ou Ollama, etc.)
   - Cocher **Enabled**
   - **Save configuration**
3. (Optionnel) Profil utilisateur : **Enable AI RCA on new failures**.

### Déclencher une analyse

- **Automatique** : après un nouvel échec (si préférence activée).
- **Manuel** : depuis le détail incident (si bouton RCA / trigger) ou API `POST /api/incidents/{id}/rca` (Lead+).

### Vérifier dans **RCA & AI Insights**

1. Filtre projet : `demo-ci`.
2. **Refresh**.
3. Attendu dans la grille :
   - `rca_status` : `pending` → `ready` (ou `failed`)
   - Colonnes **summary** / root cause quand prêt

| Résultat | Signification |
|----------|----------------|
| `pending` / vide | Job en file |
| `ready` + texte | LLM OK |
| Stub / message d’erreur provider | Clé API manquante ou provider `disabled` |

**Ce que l’UI ne fait pas encore :** bouton « Proposer un patch » ou « Ouvrir une PR » — uniquement via API (partie C).

---

## A.4 — Tester flaky & quarantaine (lié à `get_flaky_tests` MCP)

### Scénario flaky (UI)

1. Sur le **Dashboard**, **résoudre** l’incident du test.
2. Renvoyer le **même** webhook (même `name` + `error`, nouveau `X-Run-Id`).
3. Rafraîchir le dashboard.

**Critère :** le nom du test commence par **`[FLAKY]`**.

### Vue Quarantaine

1. **Quality / Quarantine**.
2. Sélectionner le gateway `demo-ci`.
3. Vérifier KPIs (actifs, auto, manuel).
4. (Lead+) Ajouter un test dans la deny-list → vérifier qu’il apparaît.

**Lien MCP :** `get_flaky_tests` renvoie les mêmes familles de tests `[FLAKY]` avec empreinte SHA-256 et taux d’échec — données cohérentes avec cette vue, mais exposées pour Cursor, pas comme écran séparé.

---

## A.5 — Execution Hub & tags CI (contexte MCP)

1. **Dashboard** ou vue exécutions (selon version) → ouvrir un groupe de run.
2. Vérifier badges **ENV** (ex. `INTEGRATION`) et **TYPE** si envoyés en headers webhook.

Ces infos alimentent `ci_tags` dans `get_incident_context` (API/MCP), pas un écran « MCP » dédié.

---

## A.6 — Ce que tu ne peux pas tester dans l’UI aujourd’hui

| Action | Alternative |
|--------|-------------|
| Proposer un correctif (self-healing) | `POST /api/incidents/{id}/healing/propose` |
| Créer une PR GitHub | `POST /api/incidents/{id}/healing/pr` |
| Lister outils MCP | Cursor ou `POST /mcp` method `tools/list` |
| `get_incident_context` | Cursor ou `POST /mcp` |
| `get_flaky_tests` | Cursor ou `POST /mcp` |
| Métriques Prometheus | `GET http://localhost:9000/metrics` |

---

# Partie B — CLI QA Capsule (tous les binaires)

## B.1 — Vue d’ensemble des commandes

| Binaire | Chemin | Rôle |
|---------|--------|------|
| **Serveur** | `go run ./cmd/qacapsule/main.go` | Lance API + UI statique |
| **CLI développeur** | `go build -o bin/qacapsule-cli ./cmd/cli` | Wrapper tests locaux + check flaky |
| **Agent JUnit** | `go run ./cmd/agent/main.go` | Envoie un XML JUnit vers le webhook |
| **listusers** | `go run ./cmd/listusers/main.go` | Liste utilisateurs SQLite |
| **resetpass** | `go run ./cmd/resetpass/main.go` | Réinitialise un mot de passe admin |

Le CLI **n’expose pas encore** de sous-commandes `mcp` ou `healing` — seulement `run`.

---

## B.2 — CLI développeur : `qacapsule run`

### Build

```powershell
go build -o bin\qacapsule-cli .\cmd\cli
```

### Aide

```powershell
bin\qacapsule-cli --help
bin\qacapsule-cli run --help
```

### Flags et variables

| Flag | Variable d’environnement | Défaut | Description |
|------|------------------------|--------|-------------|
| `--api` | `QACAPSULE_API_URL` | `http://localhost:9000` | URL du control plane |
| `--api-key` | `QACAPSULE_API_KEY` | (vide) | Clé projet (`X-API-Key`) |
| `--test-name` | `QACAPSULE_TEST_NAME` | nom de la commande | Nom du test pour l’empreinte |
| `--test-error` | `QACAPSULE_TEST_ERROR` | message d’erreur du process | Texte d’erreur pour l’empreinte |

### Test 1 — Commande qui réussit

```powershell
$env:QACAPSULE_API_URL = "http://localhost:9000"
$env:QACAPSULE_API_KEY = "test-api-key-local"

bin\qacapsule-cli run echo hello
```

**Critère :** exit code 0, pas de message flaky.

### Test 2 — Commande qui échoue (sans flaky en base)

```powershell
bin\qacapsule-cli run --test-name "checkout should complete" `
  --test-error "TimeoutError: waiting for selector" `
  cmd /c "exit 1"
```

**Critère :** exit code 1, **pas** de bandeau jaune flaky (si le test n’est pas encore marqué flaky).

### Test 3 — Commande qui échoue (avec flaky connu)

**Prérequis :** avoir fait le scénario flaky (Partie A.4) pour le même couple nom + erreur.

```powershell
bin\qacapsule-cli run --test-name "[FLAKY] checkout should complete" `
  --test-error "TimeoutError: waiting for selector" `
  cmd /c "exit 1"
```

**Critère :** message jaune du type *« Ce test a échoué, mais il est instable en CI… »*.

### Test 4 — API injoignable

```powershell
bin\qacapsule-cli --api http://localhost:9999 run cmd /c "exit 1"
```

**Critère :** échec du test local, **sans** crash du CLI (warning flaky silencieusement ignoré).

### Test 5 — Empreinte cohérente avec CI

L’empreinte utilisée par le CLI est la même que le serveur :

```text
SHA256( testName + "|" + errorMessage )
```

Vérifie avec l’API (remplace par ton hash) :

```powershell
# Après calcul côté serveur, ou via incident fingerprint en base
Invoke-RestMethod "http://localhost:9000/api/incidents/check-flaky/<64-char-hex>" `
  -Headers @{ "X-API-Key" = "test-api-key-local" }
```

---

## B.3 — Agent JUnit (`cmd/agent`)

Envoie un fichier XML vers le webhook (agnostique framework si sortie JUnit).

```powershell
go run .\cmd\agent\main.go `
  -api http://localhost:9000 `
  -key test-api-key-local `
  -project demo-ci `
  -file .\reports\junit.xml
```

(Adapte les flags avec `go run .\cmd\agent\main.go -h` si la version locale diffère.)

**Critère :** réponse webhook `queued`, incidents visibles dans l’UI.

---

## B.4 — Utilitaires admin

### Lister les utilisateurs

```powershell
go run .\cmd\listusers\main.go
# ou avec chemin DB explicite :
go run .\cmd\listusers\main.go .\data\qacapsule.db
```

### Réinitialiser un mot de passe

```powershell
go run .\cmd\resetpass\main.go
# Suivre l’invite (username + nouveau password)
```

---

## B.5 — Matrice CLI ↔ fonctionnalités Self-Healing / MCP

| Fonction | `qacapsule run` | Futur CLI (non implémenté) |
|----------|-----------------|----------------------------|
| Check flaky local | Oui | — |
| Ingestion webhook | Non (utiliser agent ou curl) | `qacapsule ingest` |
| MCP tools | Non | `qacapsule mcp call ...` |
| Healing propose | Non | `qacapsule heal propose` |
| Healing PR | Non | `qacapsule heal pr` |

---

# Partie C — API REST (self-healing & compléments)

## C.1 — Authentification (PowerShell)

```powershell
$base = "http://localhost:9000"
$login = Invoke-RestMethod -Method POST -Uri "$base/api/login" `
  -ContentType "application/json" -Body '{"username":"admin","password":"Admin123!"}'
$h = @{ Authorization = "Bearer $($login.token)" }
```

## C.2 — Self-healing : proposition

```powershell
$id = 1   # ID incident du dashboard
$propose = @{
  file_content = "export async function clickPay(page) { await page.click('#pay-button'); }"
} | ConvertTo-Json

Invoke-RestMethod -Method POST -Uri "$base/api/incidents/$id/healing/propose" `
  -Headers $h -ContentType "application/json" -Body $propose
```

| Champ réponse | Attendu |
|---------------|---------|
| `code` | Fichier proposé (stub ou LLM) |
| `explanation` | Texte explicatif |

## C.3 — Self-healing : PR GitHub

```powershell
$pr = @{
  repo      = "mon-org/mon-repo"
  file_path = "tests/checkout.spec.ts"
  code      = "<contenu renvoyé par propose>"
} | ConvertTo-Json

Invoke-RestMethod -Method POST -Uri "$base/api/incidents/$id/healing/pr" `
  -Headers $h -ContentType "application/json" -Body $pr
```

**Prérequis :** `$env:GITHUB_TOKEN` côté **processus serveur**, droits `contents` + `pull_requests`.

---

# Partie D — Serveur MCP (Cursor ou HTTP)

## D.1 — Configurer Cursor

Fichier MCP utilisateur (ex. `%USERPROFILE%\.cursor\mcp.json`) :

```json
{
  "mcpServers": {
    "qa-capsule": {
      "url": "http://localhost:9000/mcp",
      "headers": {
        "Authorization": "Bearer dev-mcp-secret"
      }
    }
  }
}
```

Sans token : omettre `headers` si `QACAPSULE_MCP_TOKEN` n’est pas défini sur le serveur.

Redémarrer Cursor → vérifier que les outils **`get_flaky_tests`** et **`get_incident_context`** apparaissent.

## D.2 — Tests manuels HTTP

Headers :

```powershell
$mcpH = @{ "Content-Type" = "application/json" }
if ($env:QACAPSULE_MCP_TOKEN) { $mcpH["Authorization"] = "Bearer $($env:QACAPSULE_MCP_TOKEN)" }
```

| # | method | params | Critère |
|---|--------|--------|---------|
| 1 | `initialize` | `protocolVersion`, `clientInfo` | `result.protocolVersion` |
| 2 | `tools/list` | `{}` | 2 outils listés |
| 3 | `tools/call` | `name: get_flaky_tests`, `arguments: { project: "demo-ci" }` | JSON avec `identity_fingerprint_sha256`, `failure_rate` |
| 4 | `tools/call` | `name: get_incident_context`, `arguments: { incident_id: 1 }` | `stack_trace`, `ci_tags`, `test_name` |

Exemple `tools/call` :

```powershell
$call = @{
  jsonrpc = "2.0"; id = 3; method = "tools/call"
  params = @{
    name = "get_incident_context"
    arguments = @{ incident_id = 1 }
  }
} | ConvertTo-Json -Depth 5
(Invoke-RestMethod -Method POST -Uri "$base/mcp" -Headers $mcpH -Body $call).result.content[0].text
```

## D.3 — Scénario bout-en-bout avec Cursor (IDE)

1. Serveur + données (Partie A.2).
2. Config MCP (D.1).
3. Dans Cursor, demander :
   - *« Utilise get_incident_context pour l’incident 1 et explique l’échec sans supposer le langage. »*
   - *« Liste les tests flaky du projet demo-ci via get_flaky_tests. »*
4. **Critère :** Cursor s’appuie sur le JSON brut ; le lien avec ton repo local (Java, TS, etc.) est fait **dans l’IDE**, pas par QA Capsule.

---

# Partie E — Santé & observabilité (hors UI)

```powershell
Invoke-RestMethod http://localhost:9000/healthz
Invoke-RestMethod http://localhost:9000/readyz
(Invoke-WebRequest http://localhost:9000/metrics).Content
```

---

# Partie F — Grille de validation finale

| # | Test | Via UI | Via CLI | Via MCP/API |
|---|------|--------|---------|-------------|
| 1 | Serveur démarre | — | `qacapsule` main | — |
| 2 | Login + dashboard | Oui | — | — |
| 3 | Créer gateway | Oui | — | — |
| 4 | Webhook → incident | Partiel (refresh UI) | agent / curl | POST webhook |
| 5 | RCA IA | Oui | — | POST/GET rca |
| 6 | Flaky `[FLAKY]` | Oui | — | — |
| 7 | Quarantaine | Oui | — | GET quarantine |
| 8 | CLI warning flaky | — | `qacapsule run` | check-flaky API |
| 9 | MCP flaky list | Non | — | `get_flaky_tests` |
| 10 | MCP incident ctx | Non | — | `get_incident_context` |
| 11 | Healing propose | Non | — | healing/propose |
| 12 | Healing PR | Non | — | healing/pr |
| 13 | healthz/readyz/metrics | Non | — | GET |

---

# Documents connexes

- [End-to-End Testing](testing.md) — parcours ingestion / dashboard / flaky classique
- [Artifacts & CLI](artifacts-and-cli.md) — artefacts + détails `qacapsule run`
- [Platform User Guide](platform-user-guide.md) — rôles et menus
- [Intelligence & Quarantine](intelligence-quarantine.md) — quarantaine CI

---

**Prochaine évolution possible (non livrée) :** panneau Self-Healing dans le détail incident (propose + PR) et sous-commandes CLI `healing` / `mcp` pour éviter curl.
