#!/bin/bash
# ==============================================================================
# QA Flight Recorder - CI/CD Telemetry Agent
# ==============================================================================

# Variables injectées par la CI ou le Control Plane
WEBHOOK_URL="${SRE_WEBHOOK_URL}"
API_KEY="${SRE_API_KEY}"

# Variables par défaut (GitLab CI injecte automatiquement CI_JOB_NAME)
JOB_NAME="${CI_JOB_NAME:-"Pipeline Inconnu"}"
ERROR_MSG="${1:-"[FATAL] Le pipeline CI/CD a échoué silencieusement."}"
LOG_DETAILS="Branche: ${CI_COMMIT_BRANCH:-"N/A"} | Commit: ${CI_COMMIT_SHORT_SHA:-"N/A"}"

echo "[SRE Agent] Initialisation de l'envoi de télémétrie..."

if [ -z "$WEBHOOK_URL" ] || [ -z "$API_KEY" ]; then
    echo "[SRE Agent] ERREUR : Les variables SRE_WEBHOOK_URL et SRE_API_KEY sont manquantes !"
    exit 1
fi

# Construction du payload JSON (On utilise jq pour formater proprement le JSON)
PAYLOAD=$(jq -n \
  --arg name "$JOB_NAME" \
  --arg err "$ERROR_MSG" \
  --arg status "CRITICAL" \
  --arg logs "$LOG_DETAILS" \
  '{name: $name, error: $err, status: $status, console_logs: $logs}')

echo "[SRE Agent] Transmission au Control Plane : $WEBHOOK_URL"

# Envoi de la requête POST au Control Plane
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d "$PAYLOAD")

# Le Control Plane renvoie 202 (Accepted) dans notre route webhooks
if [ "$HTTP_STATUS" -eq 202 ] || [ "$HTTP_STATUS" -eq 200 ]; then
    echo "[SRE Agent] Alerte reçue avec succès par le SRE Control Plane (HTTP $HTTP_STATUS)."
else
    echo "[SRE Agent] Échec de l'envoi (HTTP $HTTP_STATUS). Vérifiez l'URL ou l'API Key."
    exit 1
fi