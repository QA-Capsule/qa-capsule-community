#!/bin/bash
# Reçoit : K8S_NAMESPACE, DEPLOYMENT_NAME
# Nécessite un fichier de config k8s ou un service account dans le pod

echo "[K8S] Restarting deployment ${DEPLOYMENT_NAME} in namespace ${K8S_NAMESPACE}..."
kubectl rollout restart deployment/${DEPLOYMENT_NAME} -n ${K8S_NAMESPACE}