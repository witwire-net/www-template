# www-template

TypeSpec を API 契約の正とし、Svelte フロントエンドと Go バックエンドを同じリポジトリで運用するモノレポテンプレートです。

- API 契約は `packages/typespec/main.tsp` が唯一の正です
- OpenAPI・フロントエンド SDK・Go バインディングは契約から生成します（手編集禁止）
- `web` は公開面 LP であり `domain` / `api` に依存しません（`ui` のみ利用可）
- `app` は `app -> domain -> api` の依存方向を守ります（`app` は `ui` にも依存）
- バックエンドは `cmd/api -> internal/app -> (adapter/*|application|platform) -> domain` の依存方向を守ります

## 目次

- [技術スタック](#技術スタック)
- [リポジトリ構成](#リポジトリ構成)
- [アーキテクチャ](#アーキテクチャ)
- [ローカル開発環境セットアップ](#ローカル開発環境セットアップ)
- [コマンド一覧](#コマンド一覧)
- [標準の検証順（CI と同じ）](#標準の検証順ci-と同じ)
- [API 契約と生成物](#api-契約と生成物)
- [現在の API surface](#現在の-api-surface)
- [Auth surface と認証仕様](#auth-surface-と認証仕様)
- [環境変数リファレンス](#環境変数リファレンス)
- [データベースマイグレーション](#データベースマイグレーション)
- [CI/CD](#cicd)
- [Git hooks とコミット規約](#git-hooks-とコミット規約)
- [Guardrails と静的解析](#guardrails-と静的解析)
- [関連ドキュメント](#関連ドキュメント)

---

## 技術スタック

| 分類               | 内容                                                      |
| ------------------ | --------------------------------------------------------- |
| フロントエンド     | SvelteKit 2, Svelte 5, TypeScript, Vite                   |
| テスト（フロント） | Vitest, Playwright                                        |
| API 契約           | TypeSpec 1.8, OpenAPI 3, Spectral, Orval                  |
| コード生成         | Orval（frontend SDK）, oapi-codegen 2.4（Go bindings）    |
| バックエンド       | Go 1.26.4, Gin 1.11, GORM 1.31, golang-migrate 4.18       |
| DB / KVS / Search  | PostgreSQL 18, Valkey 9 (Redis 互換), OpenSearch 3        |
| Object Storage     | Cloudflare R2 / MinIO（S3 互換）                          |
| メール             | SMTP（開発時は Mailpit）                                  |
| ツール             | pnpm 10.33, ESLint 9, Prettier, golangci-lint 1.64, Husky |
| CI                 | GitHub Actions                                            |
| 開発環境           | Dev Container（Docker Compose）                           |

---

## リポジトリ構成

```text
.
├── .devcontainer/               # Dev Container 定義（compose.yaml, devcontainer.json, Dockerfile）
├── .github/workflows/ci.yml     # CI パイプライン
├── .husky/                      # Git hooks（pre-commit, commit-msg）
├── scripts/                     # CI / lint / codegen / migration ヘルパースクリプト
│   ├── codegen/check.sh         # codegen drift check
│   ├── go/                      # Go build / lint / format / test / migrate
│   ├── hooks/                   # pre-commit 内で呼ばれるフックスクリプト
│   └── security/                # govulncheck / gitleaks / osv-scanner
├── tests/                       # Playwright E2E テスト
├── packages/
│   ├── typespec/                # API 契約（唯一の正）
│   │   ├── main.tsp             # エントリポイント
│   │   ├── src/
│   │   │   ├── common/          # 共通エラー定義
│   │   │   ├── models/          # データモデル
│   │   │   └── routes/v1/       # API ルート定義
│   │   ├── openapi/openapi.json # 生成 OpenAPI（手編集禁止）
│   │   └── .spectral.yaml       # Spectral lint ルールセット
│   ├── frontend/
│   │   ├── api/                 # 生成 SDK + API ヘルパー（手編集禁止）
│   │   │   └── src/generated/client.ts
│   │   ├── domain/              # ドメインフック / ドメイン型
│   │   │   └── src/hooks/       # use* 形式の Svelte 5 stateful hook
│   │   ├── app/                 # SvelteKit SPA アプリ（認証面・アプリ機能）
│   │   │   └── src/routes/      # CSR ルート
│   │   └── ui/                  # 共有 UI コンポーネント（atoms / molecules / organisms）
│   ├── web/                     # SvelteKit 公開面 LP（domain / api 非依存）
│   └── backend/
│       ├── cmd/api/             # Go API エントリポイント
│       ├── db/migrations/       # golang-migrate SQL ファイル（*.up.sql / *.down.sql）
│       ├── internal/
│       │   ├── app/             # ランタイムコンテナ・依存注入（Composition Root）
│       │   ├── domain/          # ドメインモデル・値オブジェクト（Account/Auth を flat package で表現）
│       │   ├── application/     # アプリケーションサービス（ユースケース・ポート）
│       │   ├── adapter/
│       │   │   ├── http/        # Gin ルーター・ハンドラアダプタ
│       │   │   ├── postgres/    # GORM リポジトリ実装
│       │   │   ├── valkey/      # Valkey 状態リポジトリ実装
│       │   │   ├── webauthn/    # WebAuthn プロバイダ実装
│       │   │   └── mailer/      # SMTP メール送信実装
│       │   ├── platform/
│       │   │   ├── config/      # 設定・共有型
│       │   │   ├── id/          # ID ポリシー・ULID 生成
│       │   │   ├── observability/ # ロガー・メトリクス・トレーサー
│       │   │   └── health/      # インフラ健全性チェック
│       │   └── generated/openapi/openapi.gen.go  # 生成 Go バインディング（手編集禁止）
│       └── tools/analyzers/     # カスタム静的解析ツール（guardrails）
└── openspec/                    # OpenSpec 仕様（現在は lint / CI 対象外）
```

---

## アーキテクチャ

### フロントエンド依存方向

```
packages/web  ──────────────────────────────►  packages/frontend/ui
                                                （公開面 LP は api / domain に依存しない）

packages/frontend/app  ──►  packages/frontend/domain  ──►  packages/frontend/api
         │                                                          │
         └──────────────────►  packages/frontend/ui                └── 生成 SDK

```

- `web` は `@www-template/ui` のみ利用可。`@www-template/domain` / `@www-template/api` の import は ESLint で禁止。
- `app` はページ・コンポーネントで直接 API を呼ばず、`domain` の `use*` フックを経由します。
- `domain` フックは `{ data, actions }` を返す `use*` export に統一します（`.svelte.ts` に配置）。
- ページ・コンポーネントは副作用（`onMount`, `$effect` による I/O）を書かず、フックの `actions` を呼ぶだけにします。

### バックエンド依存方向

```
cmd/api
  └── internal/app              （DI コンテナ・ランタイム）
        ├── internal/adapter/http        （Gin + 生成アダプタ）
        │     └── internal/application
        ├── internal/adapter/postgres    （GORM リポジトリ実装）
        │     └── internal/domain
        ├── internal/adapter/valkey      （Valkey 状態リポジトリ実装）
        │     └── internal/domain
        ├── internal/adapter/webauthn    （WebAuthn プロバイダ）
        │     └── internal/domain
        ├── internal/adapter/mailer      （SMTP メール送信）
        │     └── internal/application
        ├── internal/application         （ユースケース・ポート）
        │     └── internal/domain
        ├── internal/domain              （ドメインモデル・値オブジェクト）
        └── internal/platform/*           （クロスカッティング基盤）
```

**禁止事項（守護ルール）**

| レイヤー                 | 禁止                                                                    |
| ------------------------ | ----------------------------------------------------------------------- |
| `domain` / `application` | Gin, GORM, generated, HTTP infra の import                              |
| `adapter/http`           | `domain.<Entity>` / `domain.Err\*` の直接使用（application DTO を経由） |
| `adapter/postgres` 以外  | GORM / Postgres driver の import                                        |
| `domain` / `application` | `time.Now`, `os.Getenv`, `log`, `math/rand` の直接使用                  |

### API 契約フロー

```
packages/typespec/main.tsp
        │  pnpm gen
        ├──► packages/typespec/openapi/openapi.json
        │           │  Orval
        │           └──► packages/frontend/api/src/generated/client.ts
        │           │  oapi-codegen
        │           └──► packages/backend/internal/generated/openapi/openapi.gen.go
        │
        └── pnpm check:codegen（drift があれば CI 失敗）
```

---

## ローカル開発環境セットアップ

### オプション A: Dev Container（推奨）

VS Code または GitHub Codespaces で `.devcontainer/` を使います。起動すると以下のサービスが自動で立ち上がります。

| サービス                 | 接続先                                                  |
| ------------------------ | ------------------------------------------------------- |
| PostgreSQL 18            | `postgres:5432`                                         |
| Valkey 9（Redis 互換）   | `valkey:6379`                                           |
| OpenSearch 3             | `http://opensearch:9200`                                |
| MinIO（S3 互換）         | `http://minio:9000`（Console: `http://localhost:9001`） |
| Mailpit（SMTP + Web UI） | SMTP: `mailpit:1025` / UI: `http://localhost:8025`      |

環境変数はコンテナ内に自動注入されます。`postCreateCommand` で `pnpm install` と Playwright ブラウザのインストールも自動実行されます。

### オプション B: ローカルセットアップ（手動）

**前提ツール**

- Node.js 24.12+
- pnpm 10.27+（`corepack enable` で有効化）
- Go 1.26.4+
- PostgreSQL 18
- Valkey 9（または Redis 7+）
- OpenSearch 3
- MinIO または S3 互換ストレージ
- SMTP サーバー（開発時は Mailpit 推奨）

**手順**

```bash
# 1. 依存インストール
corepack enable
pnpm install

# 2. 生成物をそろえる（TypeSpec -> OpenAPI -> SDK -> Go bindings）
pnpm gen

# 3. 環境変数を設定（.env.example を参考に）
cp .env.example .env
# .env を編集して DATABASE_URL, VALKEY_URL 等を設定

# 4. DB マイグレーションを実行
pnpm db:migrate:up

# 5. 全サービスを起動
pnpm dev:all
```

起動後のアクセス先:

| サービス               | URL                           |
| ---------------------- | ----------------------------- |
| Product Go API         | `http://localhost:8080`       |
| Admin Go API           | `http://localhost:8081`       |
| 公開面 LP（web）       | `http://www.localhost:5173`   |
| SPA アプリ（app）      | `http://app.localhost:5174`   |
| Admin Console（admin） | `http://admin.localhost:5176` |

---

## コマンド一覧

### 開発

```bash
pnpm dev:all          # Product/Admin Go API + web + app + admin を並列起動
pnpm dev:server       # Product Go API のみ（http://localhost:8080）
pnpm dev:admin-server # Admin Go API のみ（http://localhost:8081）
pnpm dev:web          # 公開面 LP のみ（http://www.localhost:5173）
pnpm dev:app          # SPA アプリのみ（http://app.localhost:5174）
pnpm dev:admin        # Admin Console のみ（http://admin.localhost:5176）
pnpm dev:client       # dev:web のエイリアス
```

### コード生成

```bash
pnpm gen              # TypeSpec -> OpenAPI -> frontend SDK -> Go bindings（フル生成）
pnpm gen:openapi      # TypeSpec -> OpenAPI のみ
pnpm gen:api-sdk      # TypeSpec -> OpenAPI -> frontend SDK
pnpm gen:backend      # OpenAPI -> Go bindings のみ
pnpm check:codegen    # 生成物に未コミットの差分があれば失敗（CI 確認用）
```

### 検証

```bash
pnpm format:check     # Prettier + tsp format + gofmt/goimports のフォーマット確認
pnpm lint             # Spectral + ESLint + golangci-lint + custom guardrails + security + codegen drift
pnpm check            # TypeSpec compile + frontend 型チェック + Go build
pnpm test:run         # web + app + ui + Go ユニットテスト（全て）
pnpm test:server      # Go ユニットテストのみ
pnpm test:client      # web + app フロントテストのみ
pnpm test:e2e         # Playwright E2E テスト
pnpm build            # Go backend + frontend をビルド
```

### フォーマット

```bash
pnpm format           # 全ファイルをフォーマット（Prettier + tsp format + gofmt/goimports）
pnpm format:check     # フォーマット確認のみ（変更なし）
```

### DB マイグレーション

```bash
pnpm db:migrate:create <name>   # 新規マイグレーションファイルを作成（up + down のペア）
pnpm db:migrate:up              # 未適用の全マイグレーションを適用
pnpm db:migrate:down            # 直近 1 つのマイグレーションをロールバック
```

---

## 標準の検証順（CI と同じ）

CI は以下の順番で実行します。ローカルで問題が疑われる場合はこの順番で確認してください。

```bash
pnpm format:check     # 1. フォーマット確認
pnpm gen              # 2. 生成物を最新化
pnpm lint             # 3. 全 lint（Spectral / ESLint / Go lint / security / codegen drift）
pnpm check            # 4. 型チェック + Go build
pnpm test:run         # 5. 全ユニットテスト
pnpm check:codegen    # 6. 生成物 drift 確認（pnpm gen 後の差分ゼロ確認）
pnpm build            # 7. 本番ビルド
```

---

## API 契約と生成物

### ファイル対応

| 役割              | パス                                                         |
| ----------------- | ------------------------------------------------------------ |
| 契約（唯一の正）  | `packages/typespec/main.tsp`                                 |
| 生成 OpenAPI      | `packages/typespec/openapi/openapi.json`                     |
| 生成 frontend SDK | `packages/frontend/api/src/generated/client.ts`              |
| 生成 Go bindings  | `packages/backend/internal/generated/openapi/openapi.gen.go` |

**生成物は手編集禁止です。** 契約を変更したら `pnpm gen` を実行し、生成物をまとめてコミットしてください。`pnpm check:codegen` は生成物に差分が残っていると失敗します。

### API 変更の手順

1. `packages/typespec/main.tsp`（または `src/` 配下の `.tsp`）を編集
2. `pnpm gen` を実行して生成物を更新
3. `pnpm lint` と `pnpm check` を通す
4. 生成物とソースをまとめてコミット

### Product/Admin デプロイルーティング

- Product domain と Admin domain は一致させません。どちらも同じ `/api/v1/*` path 空間を使いますが、別 domain / 別 Go binary / 別 TypeSpec service / 別 OpenAPI / 別 SDK / 別 Go bindings で分離します。
- Product domain では、Cloudflare route が `/api/v1/*` を Product GoServer（`packages/backend/cmd/api`）へ送り、それ以外の公開面・アプリ shell は Product frontend 側で配信します。Product frontend からの API 呼び出しは同一 Product domain の `/api/v1/*` を使います。
- Admin domain では、Cloudflare route が `/api/v1/*` を Admin GoServer（`packages/backend/cmd/admin-api`）へ送り、それ以外の path は Admin static frontend を配信します。Admin frontend からの API 呼び出しは同一 Admin domain の `/api/v1/*` だけを使い、Product domain や `/api/admin/*` BFF route は使いません。
- `packages/web/wrangler.toml` と `packages/frontend/app/wrangler.toml` は配備設定であり、API contract の canonical source ではありません。API の正は常に `packages/typespec/main.tsp` です。

### Spectral lint ルール

OpenAPI に対して以下の Spectral ルールが適用されます（`pnpm lint` / CI に含まれます）。

| ルール        | 内容                                                                           |
| ------------- | ------------------------------------------------------------------------------ |
| path-policy   | OpenAPI path は `/api/v1/*` のみ許可                                           |
| app-security  | `/api/v1/auth/*` と `/api/v1/status` 以外の operation は `BearerAuth` 宣言必須 |
| bearer-scheme | `BearerAuth` は `type=http` + `scheme=bearer` に限定                           |

---

## 現在の API surface

### public（Bearer 不要）

| メソッド | パス                                  | 説明                           |
| -------- | ------------------------------------- | ------------------------------ |
| `GET`    | `/api/v1/status`                      | ヘルスチェック                 |
| `POST`   | `/api/v1/auth/passkey/start`          | パスキー認証開始               |
| `POST`   | `/api/v1/auth/passkey/finish`         | パスキー認証完了               |
| `POST`   | `/api/v1/auth/passkey/register/start` | パスキー登録開始               |
| `POST`   | `/api/v1/auth/passkey/register`       | パスキー登録完了               |
| `POST`   | `/api/v1/auth/recovery`               | アカウントリカバリー開始       |
| `POST`   | `/api/v1/auth/recovery/consume`       | リカバリートークン消費         |
| `POST`   | `/api/v1/auth/passkey/add/start`      | パスキー追加開始（OTP フロー） |
| `POST`   | `/api/v1/auth/passkey/add/finish`     | パスキー追加完了（OTP フロー） |

### Bearer 必須

| メソッド | パス                      | 説明                               |
| -------- | ------------------------- | ---------------------------------- |
| `POST`   | `/api/v1/auth/logout`     | ログアウト                         |
| `GET`    | `/api/v1/passkeys`        | パスキー一覧取得                   |
| `POST`   | `/api/v1/passkeys/start`  | パスキー追加開始（認証済みフロー） |
| `POST`   | `/api/v1/passkeys/finish` | パスキー追加完了（認証済みフロー） |
| `POST`   | `/api/v1/passkeys/otp`    | OTP 発行                           |
| `DELETE` | `/api/v1/passkeys/{id}`   | パスキー削除                       |

### OpenAPI 契約外（router.go 直書き）

| メソッド | パス      | 説明                           |
| -------- | --------- | ------------------------------ |
| `GET`    | `/health` | インフラレベルのヘルスチェック |

---

## Auth surface と認証仕様

### フロントエンドルート

| パス                       | 説明                                            |
| -------------------------- | ----------------------------------------------- |
| `/login`                   | パスキー専用の認証面                            |
| `/login/recovery`          | 既存アカウント向けリカバリー導線                |
| `/login/recovery/sent`     | リカバリーメール送信完了                        |
| `/login/recovery/consume`  | リカバリートークン消費                          |
| `/login/recovery/register` | リカバリー後のパスキー再登録                    |
| `/logout`                  | ログアウト（実行は `POST /api/v1/auth/logout`） |

auth routes (`/login*`, `/logout`) と auth endpoints は `Cache-Control: no-store` 前提で扱います。

### Bearer セッション契約

- ログイン / リカバリー登録成功後、クライアントは `Authorization: Bearer <session token>` を `/api/v1/passkeys/*` 等に付与します
- bearer token はフロントエンドの **in-memory state にのみ保持**し、`localStorage` / `sessionStorage` 等の永続ストレージには書き込みません
- セッション不在 → 通常の `/login` 導線へ戻す
- セッション期限切れ / 失効 → `/session-expired` へ分岐

### システム所有 ID ポリシー

以下の ID は canonical ULID string を使用します。

`accountId`, `sessionId`, `passkeyCredentialId`, `recoveryTokenId`, `recoverySessionId`, `requestId`

例外（ULID 対象外）: opaque bearer token, recovery link token, rate-limit bucket key, WebAuthn RP ID

### Auth のレート制限・TTL デフォルト値

| 設定                                   | 値                                                       |
| -------------------------------------- | -------------------------------------------------------- |
| challenge TTL                          | 5 分                                                     |
| recovery token TTL                     | 30 分                                                    |
| recovery session TTL                   | 15 分                                                    |
| session idle TTL                       | 12 時間                                                  |
| session absolute TTL                   | 14 日                                                    |
| passkey start throttle                 | 5 req / 5 分                                             |
| recovery throttle                      | 3 req / 時（メールアドレスごと）, 10 req / 時（IP ごと） |
| finish / consume / register 失敗ロック | 10 失敗 / 15 分 → 15 分ロック                            |

---

## 環境変数リファレンス

`.env.example` をコピーして `.env` として使用してください。

```bash
cp .env.example .env
```

### 必須

| 変数                   | 説明                                                                               |
| ---------------------- | ---------------------------------------------------------------------------------- |
| `DATABASE_URL`         | PostgreSQL 接続先（例: `postgres://user:pass@localhost:5432/app?sslmode=disable`） |
| `VALKEY_URL`           | Valkey 接続先（例: `redis://localhost:6379/0`）                                    |
| `OPENSEARCH_URL`       | OpenSearch 接続先（例: `http://localhost:9200`）                                   |
| `R2_ENDPOINT`          | R2/S3 互換 object storage endpoint                                                 |
| `R2_REGION`            | object storage region                                                              |
| `R2_BUCKET`            | object storage bucket 名                                                           |
| `R2_ACCESS_KEY_ID`     | object storage アクセスキー                                                        |
| `R2_SECRET_ACCESS_KEY` | object storage シークレットキー                                                    |
| `SMTP_HOST`            | SMTP サーバーホスト                                                                |
| `MAIL_FROM_ADDRESS`    | メールの From アドレス                                                             |

`APP_ENV!=development` の場合は `APP_BEARER_TOKEN` も必須です（未設定では起動しません）。

### オプション / デフォルトあり

| 変数                        | デフォルト                                         | 説明                                             |
| --------------------------- | -------------------------------------------------- | ------------------------------------------------ |
| `APP_ENV`                   | `development`                                      | 実行環境（`development` 以外では厳格モード）     |
| `APP_BEARER_TOKEN`          | `dev-app-auth`（dev のみ）                         | app API 用 Bearer token                          |
| `PORT`                      | `8080`                                             | backend listen port                              |
| `ALLOWED_ORIGINS`           | `http://www.localhost:5173,...`                    | CORS 許可オリジン（カンマ区切り）                |
| `VALKEY_KEY_PREFIX`         | `www-template`                                     | Valkey key の共通プレフィックス                  |
| `R2_USE_PATH_STYLE`         | `false`                                            | MinIO 等 path-style endpoint を使う場合は `true` |
| `WEBAUTHN_RP_ID`            | `app.localhost`                                    | WebAuthn の Relying Party ID                     |
| `ACCOUNT_RECOVERY_URL_BASE` | `http://app.localhost:5174/login/recovery/consume` | recovery リンクのベース URL                      |
| `SMTP_PORT`                 | `587`                                              | SMTP ポート（Mailpit の場合は `1025`）           |
| `SMTP_USERNAME`             | （空）                                             | SMTP ユーザー名                                  |
| `SMTP_PASSWORD`             | （空）                                             | SMTP パスワード                                  |

### 重要な起動条件

- backend は起動時に PostgreSQL / Valkey / OpenSearch / object storage の疎通確認を行います。接続失敗時は起動しません
- `APP_ENV!=development` では `APP_BEARER_TOKEN` 未設定のまま起動できません

---

## データベースマイグレーション

マイグレーションは `packages/backend/db/migrations/` に配置します。`AutoMigrate` は禁止です。

### 命名規則

```
000001_description_here.up.sql
000001_description_here.down.sql
```

- 6 桁の連番 + アンダースコア + lowercase 英数字
- `up` / `down` のペアが必須
- ネストディレクトリ禁止

### 操作コマンド

```bash
# 新規マイグレーションファイルを作成（up + down のペア自動生成）
pnpm db:migrate:create add_auth_tables

# 未適用のマイグレーションを全て適用
pnpm db:migrate:up

# 直近 1 つのマイグレーションをロールバック
pnpm db:migrate:down
```

GORM の import は `packages/backend/internal/adapter/postgres/**` のみに許可されています。

---

## CI/CD

GitHub Actions の `ci.yml` が以下の順番で実行されます（`main` / `develop` への push と PR が対象）。

```
Checkout
└── Setup pnpm 11.5.0
└── Setup Node 24
└── Setup Go 1.26.4
└── pnpm install --frozen-lockfile
└── pnpm format:check          # フォーマット確認
└── pnpm gen                   # 生成物の最新化
└── pnpm lint                  # 全 lint
└── pnpm check                 # 型チェック + Go build
└── pnpm test:run              # ユニットテスト
└── pnpm check:codegen         # codegen drift 確認
└── pnpm build                 # ビルド
```

タイムアウト: 15 分。`pnpm install` は `--frozen-lockfile` で実行するため、`pnpm-lock.yaml` を常に最新にしてコミットしてください。

---

## Git hooks とコミット規約

### pre-commit（`pnpm lint-staged`）

staged ファイルに対して以下を自動適用します。

| 対象パターン                           | 処理内容                                                                |
| -------------------------------------- | ----------------------------------------------------------------------- |
| `*.{ts,tsx,js,jsx}`                    | `eslint --fix --no-inline-config --max-warnings 0` → `prettier --write` |
| `*.{json,md,yml,yaml}`                 | `prettier --write`                                                      |
| `*.go`                                 | `gofmt -w` + `goimports -local www-template -w`                         |
| `packages/backend/db/migrations/*.sql` | migration ファイル名 / ペアポリシーの検証                               |

codegen drift check は pre-commit には含まれず、`pnpm lint` と CI で実行されます。

### commit-msg

Conventional Commits 形式を強制します（`commitlint`）。

```
<type>: <subject>
```

使用可能な type: `feat` | `fix` | `docs` | `style` | `refactor` | `perf` | `test` | `build` | `ci` | `chore` | `revert`

**NG 例**: `update stuff`  
**OK 例**: `fix: prevent write application services from bypassing domain validation`

---

## Guardrails と静的解析

`pnpm lint` は以下の検証をすべて実行します。

### フロントエンド（ESLint）

- `eslint-plugin-boundaries`: パッケージ間依存方向の強制
- `no-restricted-imports`: `app` からの `@www-template/api` 直接 import 禁止など
- `frontend-svelte5/no-legacy-syntax`: Svelte 5 記法の強制（`on:click` 等の旧記法禁止）
- `sveltekit-app-policy`: サーバー面 route・サーバー import の禁止
- `hooks-domain/require-domain-structure`: `use*` + `{ data, actions }` 形式の強制
- `export-tsdoc/require-export-tsdoc`: export に TSDoc コメント必須
- `eslint-comments/no-use`: `eslint-disable` コメント全面禁止
- `max-lines` / `max-lines-per-function`: ファイル 500 行・関数 100 行以内

### バックエンド（golangci-lint + カスタム guardrails）

- `depguard`: レイヤー間の外部依存強制（GORM は adapter/postgres のみ等）
- カスタム静的解析（`tools/analyzers/cmd/guardrails/main.go`）:
  - レイヤー配置ポリシー・import 方向チェック
  - `adapter/http` が `domain` 型を直接使用していないか
  - application の exported API が domain 型を露出していないか
  - domain entity を `{}` リテラルで直接構築していないか
  - write application service が domain を経由しているか
  - `time.Now`, `os.Getenv` 等の副作用源の直接使用禁止
  - `AutoMigrate` 禁止
  - migration ファイル名・ペアポリシー

### セキュリティスキャン

- `govulncheck`: Go の既知脆弱性チェック
- `osv-scanner`: 依存関係の OSV チェック
- `gitleaks --no-git --config .gitleaks.toml`: シークレットスキャン

---

## 関連ドキュメント

| ドキュメント                  | 内容                                                       |
| ----------------------------- | ---------------------------------------------------------- |
| `CONTRIBUTING.md`             | コントリビューター向けの最短フロー                         |
| `CODING_STANDARDS.md`         | 機械的に fail するルールの完全一覧（guardrail の解説付き） |
| `AGENTS.md`                   | AI コーディングエージェント向けの実行方針                  |
| `.devcontainer/README.md`     | Dev Container の詳細（サービス接続先・環境変数）           |
| `packages/typespec/README.md` | TypeSpec 契約の詳細                                        |
