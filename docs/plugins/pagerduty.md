# PagerDuty plugin

Triggers a PagerDuty incident via [Events API v2](https://developer.pagerduty.com/docs/events-api/send-events-api-v2/).

## Configuration

| Variable | Description |
|----------|-------------|
| `PAGERDUTY_ROUTING_KEY` | Integration routing key from your PagerDuty service |

Set **status** to `Active` in the manifest to auto-trigger on `CRITICAL` / `FATAL` keywords in incident text.

## Manual test

Use **Execute** in Plugin Engine after saving the routing key.
