#!/usr/bin/env bash

set -euo pipefail

bash scripts/security/govulncheck.sh
bash scripts/security/gitleaks.sh
bash scripts/security/osv-scanner.sh
