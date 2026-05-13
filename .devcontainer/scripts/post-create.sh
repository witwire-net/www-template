#!/usr/bin/env bash

set -euo pipefail

corepack enable
corepack prepare pnpm@11.1.1 --activate
pnpm install

if pnpm exec playwright --version >/dev/null 2>&1; then
  find /ms-playwright -xtype l -delete 2>/dev/null || true
  pnpm exec playwright install chromium firefox webkit
fi

if [ -f go.work ]; then
  go work sync
fi

if [ -f packages/backend/go.mod ]; then
  go -C packages/backend mod download
fi
