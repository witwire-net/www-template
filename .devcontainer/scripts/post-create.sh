#!/usr/bin/env bash

set -euo pipefail

# Devcontainer 内で指定 pnpm を必ず使い、host 側の Corepack 状態に依存しないようにする。
corepack enable
corepack prepare pnpm@11.6.0 --activate

# workspace の Node 依存を先に復元し、以降の pnpm script と TypeScript 系 tooling を利用可能にする。
pnpm install

# Playwright が依存に存在する場合だけ browser binary を導入し、frontend / E2E 検証をすぐ実行できる状態にする。
if pnpm exec playwright --version >/dev/null 2>&1; then
  # 旧 container layer や volume に残った壊れた symlink を消し、browser install の再実行を安定させる。
  find /ms-playwright -xtype l -delete 2>/dev/null || true

  # Chromium / Firefox / WebKit を共有 cache volume に置き、devcontainer 再作成後も browser download を再利用する。
  pnpm exec playwright install chromium firefox webkit
fi

# Go workspace がある場合は module view を同期し、backend tooling が最新の workspace 定義を参照できるようにする。
if [ -f go.work ]; then
  go work sync
fi

# backend module がある場合は Go 依存を事前取得し、初回の build / test / migration 実行時の待ち時間を減らす。
if [ -f packages/backend/go.mod ]; then
  go -C packages/backend mod download
fi

# PostgreSQL service が healthy になった後に schema migration を適用し、Admin runtime login role も migrate script に集約して作成する。
pnpm migrate:up
