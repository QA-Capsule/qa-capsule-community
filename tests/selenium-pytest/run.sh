#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="${ROOT_DIR}/tests/selenium-pytest"
cd "${DIR}"

python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
pytest -v --junitxml=selenium-pytest-results.xml || true

JUNIT_FILE="${DIR}/selenium-pytest-results.xml"
chmod +x "${ROOT_DIR}/tests/upload-junit.sh"
"${ROOT_DIR}/tests/upload-junit.sh" "Pytest" "${JUNIT_FILE}" || true
