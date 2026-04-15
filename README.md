# www-template

TypeSpec を API 契約の正に据え、Svelte フロントエンドと Go バックエンドを同じリポジトリで運用するためのモノレポです。

- API 契約は `packages/typespec/main.tsp` を正とします
- OpenAPI、frontend SDK、Go bindings は契約から生成します
- フロントエンドは `web -> domain -> api` および `app -> domain -> api`（`app` は `ui` にも依存）の依存方向を守ります
- バックエンドは `cmd/api -> internal/app -> (http|persistence|usecases) -> domain -> types` の依存方向を守ります

## 技術スタック

- Frontend: SvelteKit, Svelte 5, Vitest, Playwright
- Contract: TypeSpec, OpenAPI, Spectral, Orval, `oapi-codegen`
- Backend: Go 1.26.2, Gin, GORM, golang-migrate
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
pnpm db:migrate:create add_auth_tables
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

- path policy: `/api/v1/*` だけを許可
- app endpoint（`/api/v1/auth/*` 除く）: `BearerAuth` 宣言を必須化
- `BearerAuth`: `type=http` / `scheme=bearer` を必須化

## 現在の API surface

OpenAPI 契約（`packages/typespec/openapi/openapi.json`）に基づく正確な一覧です。

**public（bearer 不要）**

- `GET /api/v1/status`
- `POST /api/v1/auth/passkey/start`
- `POST /api/v1/auth/passkey/finish`
- `POST /api/v1/auth/passkey/register/start`
- `POST /api/v1/auth/passkey/register`
- `POST /api/v1/auth/recovery`
- `POST /api/v1/auth/recovery/consume`
- `POST /api/v1/auth/passkey/add/start`（OTP フロー）
- `POST /api/v1/auth/passkey/add/finish`（OTP フロー）

**bearer 必須**

- `POST /api/v1/auth/logout`
- `GET /api/v1/passkeys`
- `POST /api/v1/passkeys/start`
- `POST /api/v1/passkeys/finish`
- `POST /api/v1/passkeys/otp`
- `DELETE /api/v1/passkeys/{id}`

**OpenAPI 契約外（router.go 直書き）**

- `GET /health`

## Auth surface

- `/login` は passkey-only の認証面です
- `/login/recovery`, `/login/recovery/sent`, `/login/recovery/consume`, `/login/recovery/register` は既存アカウント向けの recovery-only 導線です
- `/logout` は utility route ですが、logout 実行は canonical な `POST /api/v1/auth/logout` を使います
- auth routes (`/login*`, `/logout`) と auth endpoints は no-store 前提で扱います

bearer session contract:

- login / recovery register 成功後、client は `Authorization: Bearer <session token>` で `/api/v1/passkeys/*` 等を利用します
- bearer token は frontend の in-memory state にのみ保持し、永続 storage に復元しません
- missing session は通常の `/login` 導線へ戻し、expired / revoked session は `/session-expired` へ分岐します

auth-owned identifier policy:

- `accountId`, `sessionId`, `passkeyCredentialId`, `recoveryTokenId`, `recoverySessionId`, `requestId` などの system-owned ID は canonical ULID string を使います
- 例外として、opaque bearer token、recovery link token、rate-limit bucket key、WebAuthn RP ID は ULID 対象外です

auth runtime dependencies:

- 短命 auth state は Valkey を第一実装として扱います
- recovery mail delivery は SMTP 設定を利用します

## 環境変数

主に使うもの:

- `APP_ENV` - 既定値は `development`
- `APP_BEARER_TOKEN` - app API 用 token。`APP_ENV=development` かつ未設定のときだけ既定値 `dev-app-auth`
- `DATABASE_URL` - PostgreSQL 接続先。backend 起動と migration 実行に必須
- `ALLOWED_ORIGINS` - 既定値は `http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174`
- `PORT` - backend listen port。既定値は `8080`
- `VALKEY_URL` - Valkey 接続先。backend 起動に必須
- `VALKEY_KEY_PREFIX` - shared Valkey key prefix。既定値は `www-template`
- `OPENSEARCH_URL` - OpenSearch 接続先。backend 起動に必須
- `R2_ENDPOINT` - R2/S3 互換 object storage endpoint。backend 起動に必須
- `R2_REGION` - object storage region。backend 起動に必須
- `R2_BUCKET` - object storage bucket 名。backend 起動に必須
- `R2_ACCESS_KEY_ID` - object storage access key。backend 起動に必須
- `R2_SECRET_ACCESS_KEY` - object storage secret key。backend 起動に必須
- `R2_USE_PATH_STYLE` - MinIO 等の path-style endpoint を使う場合に `true`
- `WEBAUTHN_RP_ID` - passkey/WebAuthn の RP ID。既定値は `localhost`
- `ACCOUNT_RECOVERY_URL_BASE` - account recovery link の base URL。既定値は `http://localhost:5174/login/recovery/consume`
- `SMTP_HOST` - shared SMTP host。backend 起動に必須
- `SMTP_PORT` - shared SMTP port。既定値は `587`
- `SMTP_USERNAME` - shared SMTP username
- `SMTP_PASSWORD` - shared SMTP password
- `MAIL_FROM_ADDRESS` - mail の From address。backend 起動に必須

重要:

- `APP_ENV!=development` では `APP_BEARER_TOKEN` 未設定のまま起動できません
- backend は PostgreSQL / Valkey / OpenSearch / object storage の設定が揃っていないと起動しません
- backend 起動時に PostgreSQL / Valkey / OpenSearch / object storage の疎通確認を行い、失敗したら起動しません

auth config defaults:

- challenge TTL: 5 minutes
- recovery token TTL: 30 minutes
- recovery session TTL: 15 minutes
- session idle TTL: 12 hours
- session absolute TTL: 14 days
- passkey start throttle: 5 requests / 5 minutes
- recovery throttle: 3 requests / hour per email, 10 requests / hour per IP
- finish / consume / register failure lock: 10 failures / 15 minutes -> 15 minute lock

## PostgreSQL を使う場合

```bash
export DATABASE_URL='postgres://user:pass@localhost:5432/app?sslmode=disable'
export VALKEY_URL='redis://localhost:6379/0'
export OPENSEARCH_URL='http://localhost:9200'
export R2_ENDPOINT='http://localhost:9000'
export R2_REGION='us-east-1'
export R2_BUCKET='template'
export R2_ACCESS_KEY_ID='minioadmin'
export R2_SECRET_ACCESS_KEY='minioadmin'
export R2_USE_PATH_STYLE='true'
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

- Frontend dependency direction: `frontend/web -> frontend/domain -> frontend/api` and `frontend/app -> frontend/domain -> frontend/api` (also `frontend/app -> frontend/ui`)
- Backend dependency direction: `backend/cmd/api -> backend/internal/app -> (backend/internal/http|backend/internal/persistence|backend/internal/usecases) -> backend/internal/domain -> backend/internal/types`
- public routes は `/api/v1/auth/*` および `/api/v1/status`
- app routes（bearer 必須）は `/api/v1/passkeys/*` および `/api/v1/auth/logout`
- OpenAPI は TypeSpec から生成し、server route から逆生成しません

より厳密な機械ルールは `CODING_STANDARDS.md` を見てください。

## 関連ドキュメント

- `CONTRIBUTING.md` - contributor 向けの最短フロー
- `CODING_STANDARDS.md` - 実際に fail するルールだけをまとめた一覧
- `AGENTS.md` - coding agent 向けの実行方針

## OpenSpec

- `openspec/**` は現在の default `pnpm lint` / Git hooks / CI の対象外です
- 仕様と実装の整合は、今は主に TypeSpec とテストで保っています
