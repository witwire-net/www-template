#!/usr/bin/env bash

set -euo pipefail

bash scripts/go/verify-module.sh

(
  cd packages/backend
  go build -mod=readonly ./...
)
