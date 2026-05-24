---
icon: material/email
---

# Email (SendGrid & SMTP)

<div align="center" class="integration-hero">
  <img src="../assets/integrations/email.png" alt="Email">
</div>

Two separate manifests under `plugins/email/`:

| Manifest | Type | Protocol |
|----------|------|-----------|
| `sendgrid-alert.json` | `sendgrid` | SendGrid API v3 |
| `smtp-alert.json` | `smtp` | SMTP (STARTTLS) |

---

=== "SendGrid"

    === "QA Capsule Side"

        | Variable | Required |
        |----------|-------------|
        | `SENDGRID_API_KEY` | **Yes** |
        | `SENDGRID_FROM` | **Yes** |
        | `SENDGRID_TO` | **Yes** (or gateway **Alert Email To**) |

        API success: HTTP **202**.

    === "Provider Side (SendGrid)"

        1. [sendgrid.com](https://sendgrid.com) → **Settings** → **API Keys** → create **Mail Send** key
        2. **Sender Authentication**: verified domain or Single Sender → `SENDGRID_FROM` address
        3. `SENDGRID_TO` recipient = on-call list / mailing list

=== "SMTP"

    === "QA Capsule Side"

        | Variable | Default | Description |
        |----------|--------|-------------|
        | `SMTP_HOST` | — | Server |
        | `SMTP_PORT` | `587` | Port |
        | `SMTP_USER` / `SMTP_PASS` | — | Auth |
        | `SMTP_FROM` | — | Sender |
        | `SMTP_TO` | — | Recipient (gateway optional) |

    === "Provider Side (SMTP)"

        1. Obtain corporate relay (Exchange, Postfix, Mailtrap for dev)
        2. Allow QA Capsule server IP as relay if required
        3. SPF/DKIM aligned with `SMTP_FROM`

---

- [Catalog](integrations-catalog.md)
