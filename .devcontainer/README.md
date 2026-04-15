# Devcontainer

この devcontainer は、Go バックエンド、Svelte フロントエンド、TypeSpec を前提にした作業用の土台です。

## 含まれるもの

- Node.js 24 と pnpm
- Go 1.26.2 以上
- `gopls` `goimports` `dlv` `golangci-lint` `air`
- `wrangler` `golang-migrate` `oapi-codegen` `openspec` `opencode`
- Playwright 実行に必要な Linux 依存
- PostgreSQL 18 Valkey 9 OpenSearch 3 MinIO Mailpit のローカルサービス
- `docker` と `docker compose` をコンテナ内から利用可能

## コンテナ起動後の状態

- `postCreateCommand` で `pnpm install` を実行
- Playwright のブラウザをインストール
- `packages/backend/go.mod` が存在すれば依存取得を実行

## サービス接続先

- PostgreSQL: `postgres:5432`
- Valkey: `valkey:6379`
- OpenSearch: `http://opensearch:9200`
- MinIO API: `http://minio:9000`
- MinIO Console: `http://localhost:9001`
- Mailpit SMTP: `mailpit:1025`
- Mailpit UI: `http://localhost:8025`

## 主要な環境変数

- `DATABASE_URL=postgres://template:template@postgres:5432/template?sslmode=disable`
- `VALKEY_URL=redis://valkey:6379/0`
- `OPENSEARCH_URL=http://opensearch:9200`
- `R2_ENDPOINT=http://minio:9000`
- `R2_REGION=us-east-1`
- `R2_BUCKET=template`
- `R2_ACCESS_KEY_ID=minioadmin`
- `R2_SECRET_ACCESS_KEY=minioadmin`
- `R2_USE_PATH_STYLE=true`
- `SMTP_HOST=mailpit`
- `MAIL_FROM_ADDRESS=noreply@example.com`
- `SMTP_PORT=1025`

## メモ

- frontend と Go API 向けの主要ポートを forward しています
- 今後の Go API や OpenNext アプリ向けに `3000` `3001` `8080` `8081` も forward しています
- PostgreSQL と OpenSearch は major 更新時のローカルデータ衝突を避けるため versioned named volume を使っています
