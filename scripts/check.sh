#!/usr/bin/env sh
# Run the same lint/test checks as ci.yml locally.
# Usage: ./scripts/check.sh [frontend|backend|all]
set -e

TARGET="${1:-all}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

run_frontend() {
  echo "=== Frontend: lint + test ==="
  cd "$REPO_ROOT/web"
  pnpm install --frozen-lockfile --silent
  pnpm lint
  pnpm test
  echo "Frontend OK"
}

run_backend() {
  echo "=== Backend: go mod tidy ==="
  cd "$REPO_ROOT"
  go mod tidy
  git diff --exit-code go.mod go.sum || { echo "go.mod/go.sum not tidy"; exit 1; }

  echo "=== Backend: golangci-lint ==="
  if command -v golangci-lint > /dev/null 2>&1; then
    golangci-lint run --timeout=3m
  else
    echo "golangci-lint not found, skipping (install: https://golangci-lint.run/welcome/install/)"
  fi

  echo "=== Backend: tests ==="
  go test -race -count=1 ./internal/... ./cmd/... ./proto/...
  echo "Backend OK"
}

case "$TARGET" in
  frontend) run_frontend ;;
  backend)  run_backend  ;;
  all)      run_frontend; run_backend ;;
  *)
    echo "Usage: $0 [frontend|backend|all]"
    exit 1
    ;;
esac

echo ""
echo "All checks passed."
