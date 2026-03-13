#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

go run "$GITLEAKS_PKG" detect --source . --no-git --config .gitleaks.toml --redact --exit-code 1
