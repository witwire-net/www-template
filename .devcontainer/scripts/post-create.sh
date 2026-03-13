#!/usr/bin/env bash

set -euo pipefail

corepack enable
corepack prepare pnpm@latest --activate
pnpm install

if pnpm exec playwright --version >/dev/null 2>&1; then
  pnpm exec playwright install chromium firefox webkit
fi

if [ -f go.work ]; then
  go work sync
fi

if [ -f packages/backend/go.mod ]; then
  go -C packages/backend mod download
fi
