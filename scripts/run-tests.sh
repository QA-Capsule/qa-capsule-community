#!/usr/bin/env bash
# Run Robot Framework suites and upload JUnit results to QA Capsule.
#
# Required in CI/CD (secrets / variables):
#   QA_CAPSULE_URL       Base URL, e.g. https://qa-capsule.example.com
#   QA_CAPSULE_API_KEY   Project API key (Settings → CI/CD Gateway)
#
# Optional:
#   QA_CAPSULE_EXEC_ENV   PROD | STAGING | CANARY | DEV (default: DEV)
#   QA_CAPSULE_EXEC_TYPE  REAL | TEST-RUN | NIGHTLY | SMOKE (default: TEST-RUN)
#   CI_PIPELINE_ID        Used as X-Run-Id when set
#   OUTPUT_DIR            Robot output directory (default: tests/results)
#   VENV_DIR              Python venv path (default: .venv-robot)
#   SELENIUM_ENABLED      true to run ui_navigation.robot in CI

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Aliases used by other QA Capsule workflows (Pytest, Playwright, Cypress)
QA_CAPSULE_URL="${QA_CAPSULE_URL:-${WEBHOOK_URL:-}}"
QA_CAPSULE_API_KEY="${QA_CAPSULE_API_KEY:-${API_KEY:-}}"

VENV_DIR="${VENV_DIR:-.venv-robot}"
OUTPUT_DIR="${OUTPUT_DIR:-tests/results}"
ROBOT_OUTPUT="${OUTPUT_DIR}/output.xml"
JUNIT_FILE="${JUNIT_FILE:-${OUTPUT_DIR}/robot-junit.xml}"
ROBOT_EXIT=0

echo "==> QA Capsule Robot test runner"
echo "    Root: ${ROOT_DIR}"

# --- Python environment (idempotent) ---
if [[ ! -d "${VENV_DIR}" ]]; then
  echo "==> Creating virtualenv: ${VENV_DIR}"
  python3 -m venv "${VENV_DIR}"
fi

# shellcheck source=/dev/null
source "${VENV_DIR}/bin/activate"

echo "==> Installing dependencies from tests/requirements.txt"
python -m pip install --upgrade pip --quiet
python -m pip install -r tests/requirements.txt --quiet

mkdir -p "${OUTPUT_DIR}"

# --- Execute Robot Framework ---
echo "==> Running Robot Framework suites in tests/robotframework"
set +e
robot \
  --outputdir "${OUTPUT_DIR}" \
  --loglevel INFO \
  tests/robotframework
ROBOT_EXIT=$?
set -e

# Robot writes native output.xml; QA Capsule expects JUnit XML → convert with rebot.
if [[ -f "${ROBOT_OUTPUT}" ]]; then
  echo "==> Converting Robot output to JUnit XML: ${JUNIT_FILE}"
  rebot \
    --xunit "${JUNIT_FILE}" \
    --outputdir "${OUTPUT_DIR}" \
    "${ROBOT_OUTPUT}"
else
  echo "WARN: Robot output.xml not found; skipping JUnit conversion."
fi

# --- Upload to QA Capsule ---
if [[ -n "${QA_CAPSULE_URL:-}" && -n "${QA_CAPSULE_API_KEY:-}" ]]; then
  if [[ -f "${JUNIT_FILE}" ]]; then
    RUN_ID="${CI_PIPELINE_ID:-${GITHUB_RUN_ID:-${GITLAB_CI_PIPELINE_ID:-local-$(date +%s)}}}"
    EXEC_ENV="${QA_CAPSULE_EXEC_ENV:-DEV}"
    EXEC_TYPE="${QA_CAPSULE_EXEC_TYPE:-TEST-RUN}"

    echo "==> Uploading JUnit report to QA Capsule (${QA_CAPSULE_URL})"
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=RobotFramework" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -H "X-Run-Id: ${RUN_ID}" \
      -H "X-Execution-Env: ${EXEC_ENV}" \
      -H "X-Execution-Type: ${EXEC_TYPE}" \
      -F "file=@${JUNIT_FILE}"

    echo "==> QA Capsule ingest accepted."
  else
    echo "WARN: JUnit file missing; upload skipped."
  fi
else
  echo "==> QA_CAPSULE_URL / QA_CAPSULE_API_KEY not set — upload skipped (OK for local runs)."
fi

echo "==> Robot exit code: ${ROBOT_EXIT}"
exit "${ROBOT_EXIT}"
