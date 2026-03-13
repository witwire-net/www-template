#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

bash scripts/go/verify-module.sh

(
  cd packages/backend
  go run -mod=readonly "$OAPI_CODEGEN_PKG" --config oapi-codegen.yaml ../typespec/openapi/openapi.json
)
