# Contributing

## 前提

- Node.js 24.12+
- pnpm 10.27+
- Go 1.26.2+
- backend 実行には `DATABASE_URL`, `VALKEY_URL`, `OPENSEARCH_URL`, `R2_ENDPOINT`, `R2_REGION`, `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `MAIL_FROM_ADDRESS`

## 基本フロー

1. `corepack enable && pnpm install`
2. `pnpm gen`
3. 実装
4. `pnpm lint`
5. `pnpm test:run` (`frontend + Go`)
6. `pnpm build`

## API 契約

- 正は `packages/typespec/main.tsp`
- `packages/web/wrangler.toml` と `packages/frontend/app/wrangler.toml` は配備設定であり、API contract の canonical source ではない
- OpenAPI path は `/api/v1/*` だけを許可する
- 生成物は手編集しない
  - `packages/typespec/openapi/openapi.json`
  - `packages/frontend/api/src/generated/client.ts`
  - `packages/backend/internal/generated/openapi/openapi.gen.go`
- 契約変更後は必ず `pnpm gen` と `pnpm check:codegen`

## Go backend ルール

- public surface は `/api/v1/auth/*`（`/api/v1/auth/logout` を除く）および `/api/v1/status`
- runtime public surface baseline は `/api/v1/status`, `/api/v1/auth/passkey/start`, `/api/v1/auth/passkey/finish`, `/api/v1/auth/passkey/register/start`, `/api/v1/auth/passkey/register`, `/api/v1/auth/recovery`, `/api/v1/auth/recovery/consume`, `/api/v1/auth/passkey/add/start`, `/api/v1/auth/passkey/add/finish`
- app surface（bearer 必須）は `/api/v1/passkeys/*` および `/api/v1/auth/logout`
- app surface は `Authorization: Bearer <token>` 境界を必須にする
- `APP_ENV!=development` では `APP_BEARER_TOKEN` を必須にする
- OpenAPI は Spectral lint で path policy と bearer security declaration を検証する
- GORM は `packages/backend/internal/persistence/**` のみ
- `AutoMigrate` は禁止。`golang-migrate` 用 SQL を `packages/backend/db/migrations/**` に置く
- domain / usecases は Gin, GORM, generated, HTTP infra に依存しない
- http は persistence を直 import しない

## Hooks

- `pre-commit`: `pnpm lint-staged` + `pnpm check:codegen`
- staged `.go` は hook 内で `gofmt` + `goimports` を掛ける
- staged migration SQL は custom guardrail で filename / pair policy を検証する
- staged ESLint は inline suppression 無効・warning 失敗で実行する

## チェックコマンド

```bash
pnpm gen
pnpm check:codegen
pnpm lint
pnpm test:run
pnpm build
```

## OpenSpec

- `openspec/**` は default lint / CI から外しています
- 仕様の正は TypeSpec とテストです
