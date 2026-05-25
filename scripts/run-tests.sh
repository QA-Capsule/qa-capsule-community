#!/usr/bin/env bash
# Run Robot Framework suites and upload JUnit results to QA Capsule.
#
# Required in CI/CD (secrets / variables):
#   QA_CAPSULE_URL       Base URL, e.g. https://qa-capsule.example.com
#   QA_CAPSULE_API_KEY   Project API key (Settings → CI/CD Gateway)
#
# Optional:
#   QA_CAPSULE_EXEC_ENV   PROD | STAGING | INTEGRATION | DEV (default: DEV)
#   QA_CAPSULE_EXEC_TYPE  REAL | TEST-RUN | NIGHTLY | SMOKE (default: TEST-RUN)
#   CI_PIPELINE_ID        Used as X-Run-Id when set
#   OUTPUT_DIR            Robot output directory (default: tests/results)
#   VENV_DIR              Python venv path (default: .venv-robot)
#   ROBOT_SUITES_DIR      Directory scanned for *.robot suites (default: tests/robotframework)
#   SELENIUM_ENABLED      true to run ui_navigation.robot (default: true in CI when CHROME_BIN set)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Aliases used by other QA Capsule workflows (Pytest, Playwright, Cypress)
QA_CAPSULE_URL="${QA_CAPSULE_URL:-${WEBHOOK_URL:-}}"
QA_CAPSULE_API_KEY="${QA_CAPSULE_API_KEY:-${API_KEY:-}}"

VENV_DIR="${VENV_DIR:-.venv-robot}"
OUTPUT_DIR="${OUTPUT_DIR:-tests/results}"
ROBOT_OUTPUT="${OUTPUT_DIR}/output.xml"
# rebot --xunit path is relative to --outputdir (not repo root); use basename only.
JUNIT_BASENAME="${JUNIT_BASENAME:-robot-junit.xml}"
JUNIT_FILE="${JUNIT_FILE:-${OUTPUT_DIR}/${JUNIT_BASENAME}}"
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

# --- Discover all suite files (*.robot), excluding resources/ ---
ROBOT_SUITES_DIR="${ROBOT_SUITES_DIR:-tests/robotframework}"
mapfile -d '' -t ROBOT_SUITE_FILES < <(
  find "${ROBOT_SUITES_DIR}" -name '*.robot' ! -path '*/resources/*' -print0 | sort -z
)
if [[ "${#ROBOT_SUITE_FILES[@]}" -eq 0 ]]; then
  echo "ERROR: No .robot suite files found under ${ROBOT_SUITES_DIR}"
  exit 1
fi

GATE_SCRIPT="${ROOT_DIR}/scripts/quarantine-ci-gate.sh"
QUARANTINE_GATE=false
if [[ -x "${GATE_SCRIPT}" ]] && [[ -n "${QA_CAPSULE_URL:-}" && -n "${QA_CAPSULE_API_KEY:-}" ]]; then
  QUARANTINE_GATE=true
  echo "==> CI quarantine gate enabled (${QA_CAPSULE_URL})"
fi

echo "==> Running all Robot suites (${#ROBOT_SUITE_FILES[@]} files):"
for suite in "${ROBOT_SUITE_FILES[@]}"; do
  echo "    - ${suite}"
done

# --- Execute Robot Framework (per suite; respects quarantine deny-list) ---
PARTIAL_OUTPUTS=()
ROBOT_EXIT=0
set +e
for suite in "${ROBOT_SUITE_FILES[@]}"; do
  suite_base="$(basename "${suite}" .robot)"
  partial_output="${suite_base}-output.xml"
  robot_args=(
    --outputdir "${OUTPUT_DIR}"
    --output "${partial_output}"
    --log NONE
    --report NONE
    --loglevel INFO
  )

  if [[ "${QUARANTINE_GATE}" == "true" ]]; then
    if ! mapfile -t RUNNABLE_TESTS < <("${GATE_SCRIPT}" robot-tests "${suite}" 2>/dev/null); then
      echo "==> SKIP suite (all tests quarantined): ${suite}"
      continue
    fi
    for t in "${RUNNABLE_TESTS[@]}"; do
      robot_args+=(--test "${t}")
    done
    echo "==> Running ${suite} (${#RUNNABLE_TESTS[@]} test(s) after quarantine filter)"
  else
    echo "==> Running ${suite}"
  fi

  robot "${robot_args[@]}" "${suite}"
  suite_exit=$?
  if [[ "${suite_exit}" -ne 0 ]]; then
    ROBOT_EXIT="${suite_exit}"
  fi
  if [[ -f "${OUTPUT_DIR}/${partial_output}" ]]; then
    PARTIAL_OUTPUTS+=("${OUTPUT_DIR}/${partial_output}")
  fi
done
set -e

# Robot writes native output.xml per suite; merge then convert to JUnit for QA Capsule.
if [[ "${#PARTIAL_OUTPUTS[@]}" -gt 0 ]]; then
  echo "==> Merging ${#PARTIAL_OUTPUTS[@]} Robot output(s) → ${ROBOT_OUTPUT}"
  set +e
  rebot --outputdir "${OUTPUT_DIR}" --output output.xml "${PARTIAL_OUTPUTS[@]}"
  set -e
fi

if [[ -f "${ROBOT_OUTPUT}" ]]; then
  echo "==> Converting Robot output to JUnit XML: ${JUNIT_FILE}"
  set +e
  rebot \
    --xunit "${JUNIT_BASENAME}" \
    --outputdir "${OUTPUT_DIR}" \
    "${ROBOT_OUTPUT}"
  REBOT_EXIT=$?
  set -e
  if [[ "${REBOT_EXIT}" -ne 0 ]]; then
    echo "WARN: rebot exit code ${REBOT_EXIT} (expected when tests failed); JUnit may still be valid."
  fi
else
  echo "WARN: Robot output.xml not found; skipping JUnit conversion."
fi

# --- Upload to QA Capsule (skip when CI workflow uploads in a dedicated step) ---
SKIP_UPLOAD="${QA_CAPSULE_SKIP_UPLOAD:-false}"
if [[ "${SKIP_UPLOAD}" == "true" || "${SKIP_UPLOAD}" == "1" ]]; then
  echo "==> QA_CAPSULE_SKIP_UPLOAD set — upload deferred to CI workflow step."
elif [[ -n "${QA_CAPSULE_URL:-}" && -n "${QA_CAPSULE_API_KEY:-}" ]]; then
  if [[ -f "${JUNIT_FILE}" ]]; then
    RUN_ID="${CI_PIPELINE_ID:-${GITHUB_RUN_ID:-${GITLAB_CI_PIPELINE_ID:-local-$(date +%s)}}}"
    EXEC_ENV="${QA_CAPSULE_EXEC_ENV:-DEV}"
    EXEC_TYPE="${QA_CAPSULE_EXEC_TYPE:-TEST-RUN}"

    echo "==> Uploading JUnit report to QA Capsule (${QA_CAPSULE_URL})"
    set +e
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=RobotFramework" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -H "X-Run-Id: ${RUN_ID}" \
      -H "X-Execution-Env: ${EXEC_ENV}" \
      -H "X-Execution-Type: ${EXEC_TYPE}" \
      -F "file=@${JUNIT_FILE}"
    CURL_EXIT=$?
    set -e
    if [[ "${CURL_EXIT}" -ne 0 ]]; then
      echo "ERROR: QA Capsule upload failed (curl exit ${CURL_EXIT})."
    else
      echo "==> QA Capsule ingest accepted."
    fi
  else
    echo "WARN: JUnit file missing; upload skipped."
  fi
else
  echo "==> QA_CAPSULE_URL / QA_CAPSULE_API_KEY not set — upload skipped (OK for local runs)."
fi

echo "==> Robot exit code: ${ROBOT_EXIT}"
exit "${ROBOT_EXIT}"
