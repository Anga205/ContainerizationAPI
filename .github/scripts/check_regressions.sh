#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "[regression-check] $1" >&2
  exit 1
}

# 1) Sandbox runtime bind mounts must be remounted read-only.
if ! grep -q 'MS_REMOUNT' executor/workspace.go; then
  fail "executor/workspace.go must remount runtime bind mounts with MS_REMOUNT"
fi
if ! grep -q 'MS_RDONLY' executor/workspace.go; then
  fail "executor/workspace.go must remount runtime bind mounts with MS_RDONLY"
fi

# 2) Test reset helper must reset queue worker state.
if ! grep -q 'workerActive = false' dispatcher/main.go; then
  fail "dispatcher/resetQueueStateForTests must reset workerActive=false"
fi

# 3) Avoid printf logging in request handler and startup config dump.
if grep -q 'fmt.Printf("Error during dispatch' routes/simple-execute.go; then
  fail "routes/simple-execute.go must not use fmt.Printf for dispatch errors"
fi
if grep -q 'Configuration loaded: %+v' main.go; then
  fail "main.go must not print full config at startup"
fi

# 4) Supported language comments should not mention javascript.
if grep -q 'javascript' models/models.go; then
  fail "models/models.go comments should reflect supported languages only"
fi

# 5) README quality checks for known issues.
if grep -q '\[The Alcoding Club\](github.com/thealcodingclub)' README.md; then
  fail "README.md must use https:// in The Alcoding Club link"
fi
if grep -qi 'wether\|gonic-gin\|reccomended' README.md; then
  fail "README.md still contains known typos in GIN_MODE section"
fi

echo "[regression-check] all checks passed"
