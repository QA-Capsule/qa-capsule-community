#!/bin/bash
INCIDENT_NAME="${1:-manual}"

NS="${K8S_NAMESPACE:-default}"
DEPLOY="${DEPLOYMENT_NAME:-}"

if [ -z "$DEPLOY" ]; then
  echo "[ERROR] DEPLOYMENT_NAME is not configured."
  exit 1
fi

echo "[K8S] Restarting deployment ${DEPLOY} in namespace ${NS} (trigger: ${INCIDENT_NAME})..."
kubectl rollout restart "deployment/${DEPLOY}" -n "${NS}"