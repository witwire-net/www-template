#!/usr/bin/env bash

set -euo pipefail

bash scripts/go/verify-module.sh

(
  cd packages/backend
  go run -mod=readonly ./tools/analyzers/cmd/guardrails .
)
