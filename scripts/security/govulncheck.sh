#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

bash scripts/go/verify-module.sh

readonly GOVULNCHECK_MAX_ATTEMPTS="${GOVULNCHECK_MAX_ATTEMPTS:-3}"
readonly GOVULNCHECK_RETRY_DELAY_SECONDS="${GOVULNCHECK_RETRY_DELAY_SECONDS:-5}"

is_retryable_govulncheck_failure() {
  local output=$1

  case "$output" in
    *"lookup vuln.go.dev"* | *"Temporary failure in name resolution"* | *"no such host"* | *"i/o timeout"* | *"dial tcp"*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

run_govulncheck() {
  (
    cd packages/backend
    go run -mod=readonly "$GOVULNCHECK_PKG" ./...
  )
}

attempt=1

while true; do
  set +e
  output="$(run_govulncheck 2>&1)"
  status=$?
  set -e

  printf '%s\n' "$output"

  if [ "$status" -eq 0 ]; then
    exit 0
  fi

  if [ "$attempt" -ge "$GOVULNCHECK_MAX_ATTEMPTS" ] || ! is_retryable_govulncheck_failure "$output"; then
    exit "$status"
  fi

  printf 'govulncheck transient network failure; retrying (%s/%s) in %ss...\n' "$attempt" "$GOVULNCHECK_MAX_ATTEMPTS" "$GOVULNCHECK_RETRY_DELAY_SECONDS" >&2
  attempt=$((attempt + 1))
  sleep "$GOVULNCHECK_RETRY_DELAY_SECONDS"
done
