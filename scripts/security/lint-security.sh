#!/usr/bin/env bash

set -euo pipefail

readonly SECURITY_GO_MEM_LIMIT="${SECURITY_GO_MEM_LIMIT:-2GiB}"
readonly SECURITY_GO_GC="${SECURITY_GO_GC:-50}"

export GOMEMLIMIT="${GOMEMLIMIT:-$SECURITY_GO_MEM_LIMIT}"
export GOGC="${GOGC:-$SECURITY_GO_GC}"

bash scripts/security/govulncheck.sh
bash scripts/security/gitleaks.sh
bash scripts/security/osv-scanner.sh
