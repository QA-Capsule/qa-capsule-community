#!/bin/bash
INCIDENT_NAME="${1:-manual trigger}"

for var in GITHUB_TOKEN GITHUB_OWNER GITHUB_REPO GITHUB_WORKFLOW_ID; do
  if [ -z "${!var}" ]; then
    echo "[ERROR] $var is not configured."
    exit 1
  fi
done

REF="${GITHUB_REF:-main}"
echo "[GITHUB] Dispatching workflow ${GITHUB_WORKFLOW_ID} on ${GITHUB_OWNER}/${GITHUB_REPO}@${REF} for: ${INCIDENT_NAME}"

PAYLOAD=$(cat <<EOF
{
  "ref": "${REF}",
  "inputs": {
    "triggered_by": "qa-capsule",
    "incident": "${INCIDENT_NAME}"
  }
}
EOF
)

HTTP=$(curl -s -o /tmp/gh_resp.txt -w "%{http_code}" -X POST \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/workflows/${GITHUB_WORKFLOW_ID}/dispatches")

if [ "$HTTP" = "204" ]; then
  echo "[GITHUB] Workflow dispatch accepted."
  exit 0
fi

echo "[ERROR] GitHub API returned HTTP $HTTP"
cat /tmp/gh_resp.txt
exit 1
