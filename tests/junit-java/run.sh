#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="${ROOT_DIR}/tests/junit-java"
mkdir -p "${DIR}"
cd "${DIR}"

cat > junit-sample.xml <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="2" failures="1" skipped="0">
  <testsuite name="JUnit Java Sample" tests="2" failures="1">
    <testcase classname="com.example.ApiHealthTest" name="shouldReturn200" time="0.05"/>
    <testcase classname="com.example.DemoFailureTest" name="shouldFailIntentionally" time="0.03">
      <failure message="Expected 201 got 400">AssertionError stacktrace sample</failure>
    </testcase>
  </testsuite>
</testsuites>
EOF

JUNIT_FILE="${DIR}/junit-sample.xml"
chmod +x "${ROOT_DIR}/tests/upload-junit.sh"
"${ROOT_DIR}/tests/upload-junit.sh" "JUnit" "${JUNIT_FILE}" || true
