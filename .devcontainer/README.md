# Devcontainer

この devcontainer は、Go バックエンド、Svelte フロントエンド、TypeSpec を前提にした作業用の土台です。

## 含まれるもの

- Node.js 24 と pnpm
- Go 1.26.4 以上
- `gopls` `goimports` `dlv` `golangci-lint` `air`
- `wrangler` `golang-migrate` `oapi-codegen` `openspec` `opencode` `agent-browser`
- Playwright 実行に必要な Linux 依存
- PostgreSQL 18、Valkey 9、OpenSearch 3、MinIO、Mailpit、SigNoz のローカルサービス
- `docker` と `docker compose` をコンテナ内から利用可能

## コンテナ起動後の状態

- `postCreateCommand` で `pnpm install` を実行
- Playwright のブラウザをインストール
- `agent-browser` と Chrome for Testing を利用可能
- `packages/backend/go.mod` が存在すれば依存取得を実行
- `pnpm migrate:up` を実行し、backend migration と Admin Console 用 login role 作成を適用

## サービス接続先

- PostgreSQL: `postgres:5432`
- Valkey: `valkey:6379`
- OpenSearch: `http://opensearch:9200`
- MinIO API: `http://minio:9000`
- MinIO Console: `http://localhost:9001`
- Mailpit SMTP: `mailpit:1025`
- Mailpit UI: `http://localhost:8025`
- SigNoz UI: `http://localhost:3301`
- SigNoz OTLP gRPC: `signoz-otel-collector:4317` / `http://localhost:4317`
- SigNoz OTLP HTTP: `http://localhost:4318`

## Agent Browser

- バージョン: `agent-browser@0.27.0`
- 起動確認: `agent-browser doctor --offline --quick`
- 基本操作: `agent-browser open http://www.localhost:5173` の後に `agent-browser snapshot`
- Dashboard: `agent-browser dashboard start` の後に `http://localhost:4848`
- 認証 state や profile は Cookie やセッショントークンを含む可能性があるため、repo にはコミットしないでください

## 主要な環境変数

- `DATABASE_URL=postgres://www-template:www-template@postgres:5432/www-template?sslmode=disable`
- `ADMIN_CONFIG_PATH=/workspaces/www-template/.config/local.admin.toml`
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
- `PUBLIC_OTEL_COLLECTOR_URL=http://localhost:4318/v1/traces`

## メモ

- frontend、Go API、SigNoz 向けの主要ポートを forward しています
- 今後の Go API や OpenNext アプリ向けに `3001` `8080` `8081` も forward しています
- PostgreSQL と OpenSearch は major 更新時のローカルデータ衝突を避けるため versioned named volume を使っています
- SigNoz の永続データは `signoz-*` named volume に分離し、repo 配下へ runtime data を書き込みません
