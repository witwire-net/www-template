#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

bash scripts/go/verify-module.sh

go run -mod=readonly "$OSV_SCANNER_PKG" scan source packages/backend
