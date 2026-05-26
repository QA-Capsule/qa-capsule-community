#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_DIR="${ROOT_DIR}/tests"
FRAMEWORK="${1:-}"

if [[ -z "${FRAMEWORK}" ]]; then
  echo "Usage: tests/run-framework.sh <robot|playwright|cypress|pytest|newman|selenium-py|junit-java>"
  exit 1
fi

case "${FRAMEWORK}" in
  robot)
    exec "${TESTS_DIR}/robotframework/run.sh"
    ;;
  playwright)
    exec "${TESTS_DIR}/playwright/run.sh"
    ;;
  cypress)
    exec "${TESTS_DIR}/cypress/run.sh"
    ;;
  pytest)
    exec "${TESTS_DIR}/pytest/run.sh"
    ;;
  selenium-py)
    exec "${TESTS_DIR}/selenium-pytest/run.sh"
    ;;
  newman)
    exec "${TESTS_DIR}/newman/run.sh"
    ;;
  junit-java)
    exec "${TESTS_DIR}/junit-java/run.sh"
    ;;
  *)
    echo "Unknown framework: ${FRAMEWORK}"
    exit 1
    ;;
esac
