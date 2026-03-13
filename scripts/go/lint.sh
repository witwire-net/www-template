#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

bash scripts/go/verify-module.sh
bash scripts/go/format-check.sh

(
  cd packages/backend
  go run -mod=readonly "$GOLANGCI_LINT_PKG" run --config .golangci.yml ./...
)

bash scripts/go/guardrails.sh
