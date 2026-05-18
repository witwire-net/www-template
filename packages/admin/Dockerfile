# Admin Console マルチステージ Docker ビルド
# builder ステージで依存解決・Prisma generate・Vite build を実行
# runtime ステージで本番用 Node.js ランタイムのみを保持し軽量化する

# ── Builder Stage ──────────────────────────────────────────────────────────
FROM node:24-alpine AS builder

# pnpm を有効化
RUN corepack enable && corepack prepare pnpm@11.1.2 --activate

WORKDIR /app

# モノレポルートの lockfile と workspace 定義を先にコピー（キャッシュ最適化）
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY packages/admin/package.json ./packages/admin/
COPY packages/frontend/ui/package.json ./packages/frontend/ui/

# 依存インストール（frozen-lockfile で再現性を保証）
RUN pnpm install --frozen-lockfile

# Admin ソース・Prisma schema をコピー
COPY packages/admin ./packages/admin

# Prisma Client を生成（Admin DB / Product DB 両方）
RUN pnpm --filter @www-template/admin prisma:admin:generate
RUN pnpm --filter @www-template/admin prisma:product:generate

# SvelteKit アプリをビルド
RUN pnpm --filter @www-template/admin build

# ── Runtime Stage ──────────────────────────────────────────────────────────
FROM node:24-alpine AS runtime

RUN corepack enable && corepack prepare pnpm@11.1.2 --activate

WORKDIR /app

# 本番実行に必要なファイルのみコピー
COPY --from=builder /app/packages/admin/build ./build
COPY --from=builder /app/packages/admin/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/packages/admin/node_modules/.prisma ./node_modules/.prisma
COPY --from=builder /app/packages/admin/prisma ./prisma

# ポート開放（SvelteKit adapter-node のデフォルト）
EXPOSE 3000

# 本番サーバー起動
CMD ["node", "build"]
