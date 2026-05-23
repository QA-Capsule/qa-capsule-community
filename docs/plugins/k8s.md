---
icon: material/kubernetes
---

# Kubernetes

<div align="center" class="integration-hero">
  <img src="../assets/integrations/k8s.png" alt="Kubernetes logo">
</div>

| | |
|---|---|
| **Manifest** | `plugins/k8s/k8s-restart.json` |
| **Type** | `k8s` |
| **État** | **Stub** — pas d’accès cluster en community (sécurité) |

---

=== "Côté QA Capsule"

    L’intégration `k8s` retourne un message explicite invitant à utiliser un **webhook** vers GitOps / opérateur.

    ## Pattern recommandé

    1. Activer l’intégration **Custom Webhook** ou **GitHub Actions**
    2. Gateway : URL vers votre contrôleur (ex. restart deployment, sync Argo CD)
    3. Payload standard QA Capsule (voir [webhook.md](webhook.md))

    ## Gateway k8s (optionnel)

    Champ **GitOps / Operator Webhook URL** → même runner que `webhook`.

=== "Côté fournisseur (Kubernetes / GitOps)"

    ## 1. Ne pas exposer kubeconfig à QA Capsule

    Préférer un service intermédiaire qui :

    - Valide un token webhook
    - Applique une action limitée (rollout restart, scale, sync)

    ## 2. Exemples d’outils

    | Outil | Rôle |
    |-------|------|
    | Argo CD | Sync application après incident |
    | Flux | Reconcile GitOps |
    | API interne | `kubectl rollout restart` encapsulé |

    ## 3. RBAC cluster

    ServiceAccount dédié avec droits minimaux sur un namespace cible.

---

- [Webhook](webhook.md) · [Catalogue](integrations-catalog.md)
