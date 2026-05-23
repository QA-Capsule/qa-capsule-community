---
icon: material/tune
---

# Guide de configuration (deux côtés)

Chaque intégration QA Capsule se configure en **deux endroits** : la plateforme QA Capsule et le **fournisseur** (Slack, Jira, PagerDuty, etc.).

<div align="center" class="integration-hero">
  <img src="../assets/integrations/slack.png" alt="Slack" title="Exemple logo">
  <img src="../assets/integrations/jira.png" alt="Jira" style="margin-left:12px">
  <img src="../assets/integrations/pagerduty.png" alt="PagerDuty" style="margin-left:12px">
</div>

---

## Vue d’ensemble

```mermaid
flowchart LR
  subgraph provider [Fournisseur]
    API[Compte / API / Webhook]
  end
  subgraph qac [QA Capsule]
    ENV[Secrets serveur]
    PE[Plugin Engine]
    GW[CI/CD Gateway routing]
    WH[Webhook ingestion]
  end
  WH --> PE
  PE --> API
  GW --> PE
  ENV --> PE
  provider --> ENV
```

| Côté | Qui configure | Où | Quoi |
|------|---------------|-----|------|
| **Fournisseur** | Admin outil (Slack, Atlassian, …) | Console du fournisseur | Compte service, token, webhook URL, clés API |
| **QA Capsule** | Manager / Lead | UI + variables d’environnement | Secrets globaux, AUTO-RUN, routage par pipeline |

---

## Côté QA Capsule (détail)

### 1. Secrets globaux (recommandé)

Sur le **processus Go** qui exécute QA Capsule :

```bash
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
export JIRA_API_TOKEN=...
```

Priorité : **variable d’environnement** > champ `env` dans le manifest JSON.

Ne commitez **jamais** de tokens dans Git.

### 2. Plugin Engine

| Action | Rôle | Description |
|--------|------|-------------|
| **Configure** | Manager / Lead | Valeurs par défaut dans `plugins/.../*.json` (non secrets en prod) |
| **AUTO-RUN ON/OFF** | Manager / Admin | Si OFF : aucun déclenchement auto sur échec CI |
| **Execute** | Lead+ | Test manuel sans attendre un incident |

Manifest exemple :

```json
{
  "integration": "slack",
  "name": "Smart Slack Routing",
  "status": "Active",
  "auto_run": true,
  "trigger_on": ["CRITICAL", "FLAKY", "Timeout"],
  "config": {}
}
```

### 3. CI/CD Gateway — SRE Routing

Pour chaque **pipeline** :

1. **Add configuration**
2. Choisir une intégration **Active** (logo dans la liste)
3. Remplir les champs projet (ex. `#alerts-checkout`, clé Jira `PAY`)

Seules les intégrations listées sur ce gateway sont déclenchées automatiquement (si AUTO-RUN est ON).

Exemple stocké en base (`sre_routing_json` sur le projet) :

```json
[
  {
    "integration": "slack",
    "file_path": "slack/slack-notifier.json",
    "name": "Smart Slack Routing",
    "values": { "SLACK_CHANNEL": "#alerts-checkout" }
  },
  {
    "integration": "jira",
    "file_path": "jira/jira-ticket.json",
    "name": "Jira Auto Ticket",
    "values": { "JIRA_PROJECT_KEY": "PAY" }
  }
]
```

L’UI **Add configuration** remplit `integration`, `file_path`, `name` et les `values` selon le schéma ci-dessous.

### 4. Champs dynamiques du gateway (par intégration)

Ces champs apparaissent dans l’UI après sélection d’un plugin **Active** (avec logo). Ils sont injectés dans `ProjectRouting.Values` au moment du run.

| Logo | Type `integration` | Clé technique | Libellé UI | Obligatoire |
|:----:|------------------|---------------|------------|-------------|
| ![Slack](../assets/integrations/slack.png){ width="22" } | `slack` | `SLACK_CHANNEL` | Slack Channel | Recommandé |
| ![Teams](../assets/integrations/teams.png){ width="22" } | `teams` | `TEAMS_WEBHOOK_URL` | MS Teams Webhook URL | Oui si pas d’env global |
| ![Jira](../assets/integrations/jira.png){ width="22" } | `jira` | `JIRA_PROJECT_KEY` | Jira Project Key | Oui |
| ![PagerDuty](../assets/integrations/pagerduty.png){ width="22" } | `pagerduty` | `PAGERDUTY_ROUTING_KEY` | PagerDuty Routing Key | Oui* |
| ![Opsgenie](../assets/integrations/opsgenie.png){ width="22" } | `opsgenie` | `OPSGENIE_TEAM` | Opsgenie Team | Non |
| ![VictorOps](../assets/integrations/victorops.png){ width="22" } | `victorops` | `VICTOROPS_ROUTING_URL` | VictorOps Routing URL | Oui* |
| ![Datadog](../assets/integrations/datadog.png){ width="22" } | `datadog` | `DATADOG_TAGS` | Datadog Tags | Non |
| ![Webhook](../assets/integrations/webhook.png){ width="22" } | `webhook` | `WEBHOOK_URL` | Custom Webhook URL | Oui* |
| ![GitHub](../assets/integrations/github.png){ width="22" } | `github` | `GITHUB_OWNER`, `GITHUB_REPO`, `GITHUB_WORKFLOW_ID` | Owner / Repo / Workflow | Oui |
| ![Email](../assets/integrations/email.png){ width="22" } | `sendgrid` | `SENDGRID_TO` | Alert Email To | Oui |
| ![Email](../assets/integrations/email.png){ width="22" } | `smtp` | `SMTP_TO` | SMTP Alert To | Oui |
| ![TestRail](../assets/integrations/testrail.png){ width="22" } | `testrail` / `zephyr` / `xray` | `WEBHOOK_URL` | Webhook URL | Oui |
| ![K8s](../assets/integrations/k8s.png){ width="22" } | `k8s` | `WEBHOOK_URL` | GitOps Webhook URL | Roadmap |

\* Peut être fourni uniquement en variable d’environnement serveur ; le champ gateway **surcharge** la valeur globale pour ce pipeline.

### 5. Priorité des secrets et paramètres

```mermaid
flowchart TB
  ENV["1. Env serveur Go\n(os.Getenv)"]
  GW["2. CI/CD Gateway\n(sre_routing + legacy)"]
  MF["3. Manifest plugins/*.json\n(config / env)"]
  RUN["Runner HTTP"]
  ENV --> RUN
  GW --> RUN
  MF --> RUN
```

| Exemple | Où le mettre côté QA Capsule | Où le mettre côté fournisseur |
|---------|------------------------------|-------------------------------|
| URL webhook Slack | `SLACK_WEBHOOK_URL` en env | Slack → Incoming Webhooks → URL |
| Canal par équipe | Gateway **Slack Channel** | Créer le canal `#alerts-*` dans le workspace |
| Token Jira | `JIRA_API_TOKEN` en env | Atlassian → API token (compte technique) |
| Clé projet Jira | Gateway **Jira Project Key** | Projet Jira existant (`PAY`, `SCRUM`, …) |

### 6. Déclenchement

- Ingestion : `POST /api/webhooks/` avec `X-API-Key`
- Moteur Go : match `trigger_on` + fingerprint + `auto_run`
- Pas de script shell (sécurité RCE)
- Timeout HTTP : **30 secondes** par appel intégration

---

## Côté fournisseur (détail)

Dépend de chaque outil — voir la page dédiée :

| Logo | Intégration | Page |
|------|-------------|------|
| ![Slack](../assets/integrations/slack.png){ width="28" } | Slack | [slack.md](slack.md) |
| ![Teams](../assets/integrations/teams.png){ width="28" } | Microsoft Teams | [teams.md](teams.md) |
| ![Jira](../assets/integrations/jira.png){ width="28" } | Jira | [jira.md](jira.md) |
| ![PagerDuty](../assets/integrations/pagerduty.png){ width="28" } | PagerDuty | [pagerduty.md](pagerduty.md) |
| ![Opsgenie](../assets/integrations/opsgenie.png){ width="28" } | Opsgenie | [opsgenie.md](opsgenie.md) |
| ![VictorOps](../assets/integrations/victorops.png){ width="28" } | VictorOps | [victorops.md](victorops.md) |
| ![Datadog](../assets/integrations/datadog.png){ width="28" } | Datadog | [datadog.md](datadog.md) |
| ![GitHub](../assets/integrations/github.png){ width="28" } | GitHub Actions | [github.md](github.md) |
| ![Email](../assets/integrations/email.png){ width="28" } | SendGrid / SMTP | [email.md](email.md) |
| ![Webhook](../assets/integrations/webhook.png){ width="28" } | Webhook HTTP | [webhook.md](webhook.md) |
| ![TestRail](../assets/integrations/testrail.png){ width="28" } | TestRail | [test-management.md](test-management.md) |
| ![Zephyr](../assets/integrations/zephyr.png){ width="28" } | Zephyr | [test-management.md](test-management.md) |
| ![Xray](../assets/integrations/xray.png){ width="28" } | Xray | [test-management.md](test-management.md) |
| ![K8s](../assets/integrations/k8s.png){ width="28" } | Kubernetes | [k8s.md](k8s.md) |

---

## Checklist de mise en production

- [ ] Compte / app créé chez le fournisseur
- [ ] Token ou URL copié dans les secrets serveur QA Capsule
- [ ] **Execute** réussi dans Plugin Engine
- [ ] AUTO-RUN activé uniquement quand prêt
- [ ] Routage ajouté sur le bon gateway CI/CD
- [ ] Test webhook réel depuis la pipeline
