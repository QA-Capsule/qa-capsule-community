#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TESTS_DIR="${ROOT_DIR}/tests"
OUT_DIR="${OUT_DIR:-${TESTS_DIR}/results/robotframework}"
mkdir -p "${OUT_DIR}"

export OUTPUT_DIR="${OUT_DIR}"
export QA_CAPSULE_SKIP_UPLOAD="${QA_CAPSULE_SKIP_UPLOAD:-true}"

chmod +x "${ROOT_DIR}/scripts/run-tests.sh"
"${ROOT_DIR}/scripts/run-tests.sh" || true

JUNIT_FILE="${OUT_DIR}/robot-junit.xml"
if [[ -f "${JUNIT_FILE}" ]]; then
  chmod +x "${TESTS_DIR}/upload-junit.sh"
  "${TESTS_DIR}/upload-junit.sh" "RobotFramework" "${JUNIT_FILE}" || true
else
  echo "WARN: Robot JUnit file missing: ${JUNIT_FILE}"
fi
