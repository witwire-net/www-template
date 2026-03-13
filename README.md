# www-template

TypeSpec を API 契約の正に据え、Svelte フロントエンドと Go バックエンドを同じリポジトリで運用するためのモノレポです。

- API 契約は `packages/typespec/main.tsp` を正とします
- OpenAPI、frontend SDK、Go bindings は契約から生成します
- フロントエンドは `app -> domain -> api` の依存方向を守ります
- バックエンドは `cmd/api -> internal/app -> (http|persistence|usecases) -> domain -> types` の依存方向を守ります

## 技術スタック

- Frontend: SvelteKit, Svelte 5, Vitest, Playwright
- Contract: TypeSpec, OpenAPI, Spectral, Orval, `oapi-codegen`
- Backend: Go 1.26.1, Gin, GORM, golang-migrate
- Tooling: pnpm, ESLint, Prettier, golangci-lint, Husky

## リポジトリ構成

```text
packages/
├── backend/
│   ├── cmd/api                          # Go API entrypoint
│   ├── db/migrations/                   # golang-migrate SQL
│   ├── internal/app/                    # runtime / container
│   ├── internal/http/                   # Gin + generated handler adapter
│   ├── internal/persistence/            # GORM / memory repository
│   ├── internal/usecases/               # application services
│   ├── internal/domain/                 # domain model / invariants
│   ├── internal/types/                  # config / shared backend types
│   └── internal/generated/openapi/      # generated Go bindings
├── frontend/
│   ├── api/                             # generated SDK + API helpers
│   ├── domain/                          # domain hooks / domain types
│   ├── app/                             # SvelteKit app
│   └── ui/                              # shared UI package
└── typespec/                            # API contract source of truth
```

## クイックスタート

```bash
corepack enable
pnpm install
pnpm gen
pnpm dev:all
```

個別起動:

```bash
pnpm dev:server
pnpm dev:client
```

- Go API: `http://localhost:8080`
- Frontend: `http://localhost:5173`

## よく使うコマンド

```bash
pnpm gen               # TypeSpec -> OpenAPI -> frontend SDK -> Go bindings
pnpm check:codegen     # 生成物 drift check
pnpm lint              # Spectral + ESLint + Go lint + custom guardrails + security + codegen drift
pnpm check             # TypeSpec check + frontend type check + Go build
pnpm test:run          # frontend app + frontend ui + Go unit tests
pnpm build             # Go backend と frontend app を build
pnpm test:e2e          # Playwright E2E
pnpm db:migrate:create add_profiles
pnpm db:migrate:up
pnpm db:migrate:down
```

## 標準の検証順

CI は次の順番で実行されます。

```bash
pnpm format:check
pnpm gen
pnpm lint
pnpm check
pnpm test:run
pnpm check:codegen
pnpm build
```

迷ったらこの順にローカルでも確認すると安全です。

## API 契約と生成物

- Source of truth: `packages/typespec/main.tsp`
- Generated OpenAPI: `packages/typespec/openapi/openapi.json`
- Generated frontend SDK: `packages/frontend/api/src/generated/client.ts`
- Generated Go bindings: `packages/backend/internal/generated/openapi/openapi.gen.go`

契約を変更したら、生成物は手編集せず `pnpm gen` を実行してください。`pnpm check:codegen` は生成物の差分が残っていると fail します。

OpenAPI には Spectral lint を掛けています。

- path policy: `/api/v1/*` と `/api/v1/app/*` だけを許可
- app endpoint: `BearerAuth` 宣言を必須化
- `BearerAuth`: `type=http` / `scheme=bearer` を必須化

## 現在の API surface

- `GET /health`
- `GET /api/v1/status`
- `GET /api/v1/profiles`
- `POST /api/v1/profiles`
- `GET /api/v1/profiles/{id}`
- `GET /api/v1/app/profiles`
- `GET /api/v1/app/profiles/{id}`

`/api/v1/app/*` は bearer token が必須です。

## 環境変数

主に使うもの:

- `APP_ENV` - 既定値は `development`
- `APP_BEARER_TOKEN` - app API 用 token。`APP_ENV=development` かつ未設定のときだけ既定値 `dev-app-auth`
- `APP_PROFILE_STORE` - 既定値は `memory`。DB を使うときは `gorm`
- `DATABASE_URL` - `APP_PROFILE_STORE=gorm` や migration 実行時に必要
- `ALLOWED_ORIGINS` - 既定値は `http://localhost:5173,http://127.0.0.1:5173`
- `PORT` - backend listen port。既定値は `8080`

重要:

- `APP_ENV!=development` では `APP_BEARER_TOKEN` 未設定のまま起動できません
- `APP_PROFILE_STORE=gorm` のときは `DATABASE_URL` が必要です

## PostgreSQL を使う場合

```bash
export APP_PROFILE_STORE=gorm
export DATABASE_URL='postgres://user:pass@localhost:5432/app?sslmode=disable'
pnpm db:migrate:up
pnpm dev:server
```

GORM は `packages/backend/internal/persistence/**` に限定され、`AutoMigrate` は禁止です。schema 変更は `packages/backend/db/migrations/*.sql` で管理します。

## Guardrails と Git hooks

- `pnpm lint` は Spectral、ESLint、golangci-lint、custom Go guardrails、`govulncheck`、`gitleaks`、`osv-scanner`、`pnpm check:codegen` を実行します
- `pre-commit` は `pnpm lint-staged` の後に `pnpm check:codegen` を実行します
- `commit-msg` は `pnpm commitlint --edit $1` を実行します

`lint-staged` の実行内容:

- `*.{ts,tsx,js,jsx}` -> `eslint --fix --no-inline-config --max-warnings 0` + `prettier --write`
- `*.{json,md,yml,yaml}` -> `prettier --write`
- `*.go` -> `gofmt -w` + `goimports -w`
- `packages/backend/db/migrations/*.sql` -> migration filename / pair policy を guardrail で検証

## アーキテクチャメモ

- Frontend dependency direction: `frontend/app -> frontend/domain -> frontend/api`
- Backend dependency direction: `backend/cmd/api -> backend/internal/app -> (backend/internal/http|backend/internal/persistence|backend/internal/usecases) -> backend/internal/domain -> backend/internal/types`
- public routes は `/api/v1/*`
- app routes は `/api/v1/app/*`
- OpenAPI は TypeSpec から生成し、server route から逆生成しません

より厳密な機械ルールは `CODING_STANDARDS.md` を見てください。

## 関連ドキュメント

- `CONTRIBUTING.md` - contributor 向けの最短フロー
- `CODING_STANDARDS.md` - 実際に fail するルールだけをまとめた一覧
- `AGENTS.md` - coding agent 向けの実行方針

## OpenSpec

- `openspec/**` は現在の default `pnpm lint` / Git hooks / CI の対象外です
- 仕様と実装の整合は、今は主に TypeSpec とテストで保っています
