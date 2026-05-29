# Admin Console static asset build
# builder ステージは workspace 依存を再現し、SvelteKit adapter-static の dist だけを生成する。

FROM node:24-alpine AS builder

# pnpm は root packageManager と同じ版に固定し、CI と Docker build の依存解決差分を避ける。
RUN corepack enable && corepack prepare pnpm@11.1.2 --activate

WORKDIR /app

# lockfile と workspace manifest を先にコピーし、Admin split package の依存 graph だけを解決できるようにする。
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY packages/admin/app/package.json ./packages/admin/app/
COPY packages/admin/domain/package.json ./packages/admin/domain/
COPY packages/admin/api/package.json ./packages/admin/api/
COPY packages/frontend/i18n/package.json ./packages/frontend/i18n/
COPY packages/frontend/ui/package.json ./packages/frontend/ui/

# frozen-lockfile で、package split 後の workspace import を lockfile と一致させる。
RUN pnpm install --frozen-lockfile

# Admin app/domain/api と共有 UI/i18n だけを build context へ取り込み、旧 Prisma/BFF runtime を持ち込まない。
COPY packages/admin ./packages/admin
COPY packages/frontend/i18n ./packages/frontend/i18n
COPY packages/frontend/ui ./packages/frontend/ui
COPY tsconfig.base.json ./tsconfig.base.json

# Admin Console は静的 SPA なので、出力は packages/admin/app/dist に限定する。
RUN pnpm --filter @www-template/admin build

FROM nginx:1.27-alpine AS runtime

# Go Admin API と同一 origin で配信される静的 asset だけを収め、Node/SvelteKit server runtime は含めない。
COPY --from=builder /app/packages/admin/app/dist /usr/share/nginx/html

# SPA fallback / cache policy / API path 非フォールバックを nginx 側でも固定し、Cloudflare `_headers` 依存にしない。
COPY packages/admin/app/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
