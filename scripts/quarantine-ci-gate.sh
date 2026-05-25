#!/usr/bin/env bash
# QA Capsule CI quarantine gate — fetch deny-list and match Robot test names.
#
# Env (same aliases as run-tests.sh):
#   QA_CAPSULE_URL / WEBHOOK_URL
#   QA_CAPSULE_API_KEY / API_KEY
#
# Usage:
#   quarantine-ci-gate.sh list              # print quarantined test names (one per line)
#   quarantine-ci-gate.sh should-skip NAME  # exit 0 if skip, 1 if run
#   quarantine-ci-gate.sh robot-tests FILE  # print runnable test names for a .robot suite

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CACHE_FILE="${TMPDIR:-/tmp}/qacapsule-quarantine-$$.json"

QA_CAPSULE_URL="${QA_CAPSULE_URL:-${WEBHOOK_URL:-}}"
QA_CAPSULE_API_KEY="${QA_CAPSULE_API_KEY:-${API_KEY:-}}"

quarantine_enabled() {
  [[ -n "${QA_CAPSULE_URL}" && -n "${QA_CAPSULE_API_KEY}" ]]
}

quarantine_fetch() {
  if ! quarantine_enabled; then
    echo '{}' >"${CACHE_FILE}"
    return 0
  fi
  local url="${QA_CAPSULE_URL%/}/api/ci/quarantine"
  echo "==> Fetching CI quarantine deny-list: ${url}"
  curl -fsS -H "X-API-Key: ${QA_CAPSULE_API_KEY}" "${url}" -o "${CACHE_FILE}"
}

quarantine_python() {
  python3 - "$@" <<'PY'
import json
import os
import sys

path = os.environ.get("QACAPSULE_QUARANTINE_CACHE", "")
try:
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
except (OSError, json.JSONDecodeError):
    data = {}

tests = data.get("tests") or []
names = set()
for t in tests:
    n = (t.get("test_name") or "").strip()
    if n:
        names.add(n)

cmd = sys.argv[1]

if cmd == "list":
    for n in sorted(names):
        print(n)
elif cmd == "should-skip":
    name = " ".join(sys.argv[2:]).strip()
    for p in ("[FLAKY] ", "[PERF] "):
        if name.startswith(p):
            name = name[len(p):].strip()
    print("skip" if name in names else "run")
    sys.exit(0 if name in names else 1)
elif cmd == "robot-tests":
    suite = sys.argv[2]
    in_cases = False
    found = []
    with open(suite, encoding="utf-8") as f:
        for line in f:
            s = line.rstrip("\n")
            if s.strip().startswith("*** Test Cases ***"):
                in_cases = True
                continue
            if in_cases and s.strip().startswith("***"):
                break
            if not in_cases:
                continue
            if not s or s[0] in " \t#":
                continue
            found.append(s.strip())
    runnable = []
    skipped = []
    for t in found:
        norm = t
        for p in ("[FLAKY] ", "[PERF] "):
            if norm.startswith(p):
                norm = norm[len(p):].strip()
        if norm in names:
            skipped.append(t)
        else:
            runnable.append(t)
    for t in runnable:
        print(t)
    if skipped:
        print("::quarantine-skipped::" + "|".join(skipped), file=sys.stderr)
    sys.exit(0 if runnable else 1)
else:
    print("unknown command", file=sys.stderr)
    sys.exit(2)
PY
}

export QACAPSULE_QUARANTINE_CACHE="${CACHE_FILE}"

main() {
  local cmd="${1:-list}"
  shift || true
  quarantine_fetch
  case "${cmd}" in
    list) quarantine_python list ;;
    should-skip) quarantine_python should-skip "$@" ;;
    robot-tests) quarantine_python robot-tests "$@" ;;
    *)
      echo "Usage: $0 {list|should-skip NAME|robot-tests FILE.robot}" >&2
      exit 2
      ;;
  esac
}

trap 'rm -f "${CACHE_FILE}"' EXIT
main "$@"
