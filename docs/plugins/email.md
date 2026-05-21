# Email plugins (SendGrid & SMTP)

## SendGrid

| Variable | Description |
|----------|-------------|
| `SENDGRID_API_KEY` | API key with Mail Send permission |
| `SENDGRID_FROM` | Verified sender address |
| `SENDGRID_TO` | Recipient email |

## SMTP

| Variable | Description |
|----------|-------------|
| `SMTP_HOST` | Relay hostname |
| `SMTP_PORT` | Usually `587` (STARTTLS) or `465` |
| `SMTP_USER` / `SMTP_PASS` | Optional auth |
| `SMTP_FROM` / `SMTP_TO` | Sender and recipient |

Requires `curl` with SMTP support on the QA Capsule host.
