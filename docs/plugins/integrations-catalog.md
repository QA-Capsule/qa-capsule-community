---
icon: material/view-grid
---

# Catalogue des intégrations

Toutes les intégrations sont implémentées en **Go** (HTTP natif). Référence rapide avec logos.

!!! tip "Configuration détaillée deux côtés"
    Lisez d’abord le [Guide de configuration (QA Capsule + fournisseur)](configuration-guide.md), puis la page de chaque outil.

---

## Tableau récapitulatif

| Logo | Intégration | Page | Secrets globaux (exemples) | Champs gateway |
|:----:|-------------|------|---------------------------|----------------|
| <img src="../assets/integrations/slack.png" width="36"> | Slack | [slack.md](slack.md) | `SLACK_WEBHOOK_URL` | Slack Channel |
| <img src="../assets/integrations/teams.png" width="36"> | Microsoft Teams | [teams.md](teams.md) | `TEAMS_WEBHOOK_URL` | MS Teams Webhook URL |
| <img src="../assets/integrations/jira.png" width="36"> | Jira | [jira.md](jira.md) | `JIRA_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN` | Jira Project Key |
| <img src="../assets/integrations/pagerduty.png" width="36"> | PagerDuty | [pagerduty.md](pagerduty.md) | `PAGERDUTY_ROUTING_KEY` | PagerDuty Routing Key |
| <img src="../assets/integrations/opsgenie.png" width="36"> | Opsgenie | [opsgenie.md](opsgenie.md) | `OPSGENIE_API_KEY` | Opsgenie Team |
| <img src="../assets/integrations/victorops.png" width="36"> | VictorOps | [victorops.md](victorops.md) | `VICTOROPS_ROUTING_URL` | VictorOps Routing URL |
| <img src="../assets/integrations/datadog.png" width="36"> | Datadog | [datadog.md](datadog.md) | `DD_API_KEY`, `DD_SITE` | Datadog Tags |
| <img src="../assets/integrations/webhook.png" width="36"> | Webhook | [webhook.md](webhook.md) | `WEBHOOK_URL` | Custom Webhook URL |
| <img src="../assets/integrations/github.png" width="36"> | GitHub Actions | [github.md](github.md) | `GITHUB_TOKEN` | Owner, Repo, Workflow ID |
| <img src="../assets/integrations/email.png" width="36"> | SendGrid | [email.md](email.md) | `SENDGRID_API_KEY`, `FROM`, `TO` | Alert Email To |
| <img src="../assets/integrations/email.png" width="36"> | SMTP | [email.md](email.md) | `SMTP_HOST`, `USER`, `PASS` | SMTP Alert To |
| <img src="../assets/integrations/testrail.png" width="36"> | TestRail | [test-management.md](test-management.md) | `WEBHOOK_URL` (+ API*) | Webhook URL |
| <img src="../assets/integrations/zephyr.png" width="36"> | Zephyr | [test-management.md](test-management.md) | `WEBHOOK_URL` | Webhook URL |
| <img src="../assets/integrations/xray.png" width="36"> | Xray | [test-management.md](test-management.md) | `WEBHOOK_URL` | Webhook URL |
| <img src="../assets/integrations/qa.png" width="36"> | QA Flaky | [webhook.md](webhook.md) | `WEBHOOK_URL` | Custom Webhook URL |
| <img src="../assets/integrations/k8s.png" width="36"> | Kubernetes | [k8s.md](k8s.md) | — (stub) | GitOps Webhook URL |

\* API natives TestRail/Zephyr/Xray : roadmap ; utiliser webhook en attendant.

---

## Matrice rôles QA Capsule

| Action | Platform Admin | Manager | Lead | Observer |
|--------|------------------|---------|------|----------|
| AUTO-RUN ON/OFF | Oui | Oui | Non | Non |
| Configure secrets (UI) | Oui | Oui | Oui | Non |
| Execute test | Oui | Oui | Oui | Non |
| Gateway Add configuration | Oui | Oui | Oui | Non |

---

## Fichiers manifest

```
plugins/
├── slack/slack-notifier.json
├── teams/teams.json
├── jira/jira-ticket.json
├── pagerduty/pagerduty.json
├── opsgenie/opsgenie-alert.json
├── victorops/victorops-alert.json
├── datadog/datadog-event.json
├── webhook/custom-webhook.json
├── github/github-rerun.json
├── email/sendgrid-alert.json
├── email/smtp-alert.json
├── testrail/testrail-result.json
├── zephyr/zephyr-execution.json
├── xray/xray-result.json
├── qa/flaky-report.json
└── k8s/k8s-restart.json
```

---

## Voir aussi

- [Vue d’ensemble Plugin Engine](overview.md)
- [Webhooks API](../api/webhooks.md)
