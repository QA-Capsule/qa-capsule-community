---
icon: material/email
---

# Email (SendGrid & SMTP)

<div align="center" class="integration-hero">
  <img src="../assets/integrations/email.png" alt="Email">
</div>

Deux manifests distincts sous `plugins/email/` :

| Manifest | Type | Protocole |
|----------|------|-----------|
| `sendgrid-alert.json` | `sendgrid` | API SendGrid v3 |
| `smtp-alert.json` | `smtp` | SMTP (STARTTLS) |

---

=== "SendGrid"

    === "Côté QA Capsule"

        | Variable | Obligatoire |
        |----------|-------------|
        | `SENDGRID_API_KEY` | **Oui** |
        | `SENDGRID_FROM` | **Oui** |
        | `SENDGRID_TO` | **Oui** (ou gateway **Alert Email To**) |

        Succès API : HTTP **202**.

    === "Côté fournisseur (SendGrid)"

        1. [sendgrid.com](https://sendgrid.com) → **Settings** → **API Keys** → créer clé **Mail Send**
        2. **Sender Authentication** : domaine ou Single Sender vérifié → adresse `SENDGRID_FROM`
        3. Destinataire `SENDGRID_TO` = liste on-call / mailing list

=== "SMTP"

    === "Côté QA Capsule"

        | Variable | Défaut | Description |
        |----------|--------|-------------|
        | `SMTP_HOST` | — | Serveur |
        | `SMTP_PORT` | `587` | Port |
        | `SMTP_USER` / `SMTP_PASS` | — | Auth |
        | `SMTP_FROM` | — | Expéditeur |
        | `SMTP_TO` | — | Destinataire (gateway possible) |

    === "Côté fournisseur (SMTP)"

        1. Obtenir relais corporate (Exchange, Postfix, Mailtrap pour dev)
        2. Autoriser l’IP du serveur QA Capsule en relay si nécessaire
        3. SPF/DKIM alignés sur `SMTP_FROM`

---

- [Catalogue](integrations-catalog.md)
