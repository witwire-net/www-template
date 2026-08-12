# Contributing

## ドキュメント

- コーディング規則: `CODING_STANDARDS.md`
- 変更運用: `docs/change-operation.md`
- 永続的な振る舞い契約: `openspec/specs/**/spec.md`
- `pnpm lint` は活動中差分の構造、識別子、競合、主仕様と試験の追跡、作業パッケージの対象範囲を検査します

## 前提

- Node.js 24.12+
- pnpm 11.16.0
- Go 1.26.5+
- backend 実行には `DATABASE_URL`, `VALKEY_URL`, `OPENSEARCH_URL`, `R2_ENDPOINT`, `R2_REGION`, `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `MAIL_FROM_ADDRESS`

VSCode で作業する場合は repository root を Dev Container として開いてください。VSCode の terminal、task、LSP、formatter は Dev Container 内の toolchain を使います。

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
- Product API と Admin API はどちらも `/api/v1/*` だけを許可し、origin / Go binary / TypeSpec service / OpenAPI artifact / SDK package / Go bindings で分離する
- `/api/admin/*` は Admin BFF 逃げ道として使わない
- 生成物は手編集しない
  - Product OpenAPI: `packages/typespec/openapi/openapi.json`
  - Admin OpenAPI: `packages/typespec/openapi/admin.openapi.json`
  - Product SDK: `packages/frontend/api/src/generated/client.ts`
  - Admin SDK: `packages/admin/api/src/generated/client.ts`
  - Product Go bindings: `packages/backend/internal/generated/openapi/openapi.gen.go`
  - Admin Go bindings: `packages/backend/internal/generated/adminopenapi/openapi.gen.go`
- 契約変更後は必ず `pnpm gen` と `pnpm check:codegen`

## Go backend ルール

- Product public surface は `/api/v1/auth/*`（`/api/v1/auth/logout` を除く）および `/api/v1/status`
- runtime public surface baseline は `/api/v1/status`, `/api/v1/auth/passkey/start`, `/api/v1/auth/passkey/finish`, `/api/v1/auth/passkey/register/start`, `/api/v1/auth/passkey/register`, `/api/v1/auth/recovery`, `/api/v1/auth/recovery/consume`, `/api/v1/auth/passkey/add/start`, `/api/v1/auth/passkey/add/finish`
- app surface（bearer 必須）は `/api/v1/passkeys/*` および `/api/v1/auth/logout`
- app surface は `Authorization: Bearer <token>` 境界を必須にする
- Admin surface も Admin origin の `/api/v1/*` として提供し、Product origin / Product binary / Product OpenAPI / Product SDK / Product Go bindings へ混入させない
- `APP_ENV!=development` では `APP_BEARER_TOKEN` を必須にする
- OpenAPI は Spectral lint で path policy と bearer security declaration を検証する
- backend の依存方向は `cmd/api -> internal/app -> (internal/adapter/* | internal/application | internal/platform/*) -> internal/domain` を守る
- GORM は `packages/backend/internal/adapter/postgres/**` のみ
- `AutoMigrate` は禁止。`golang-migrate` 用 SQL を `packages/backend/db/migrations/**` に置く
- `internal/domain` / `internal/application` は Gin, GORM, generated, HTTP infra に依存しない
- `internal/adapter/http` は `internal/adapter/postgres` / `internal/adapter/valkey` などの永続化 adapter を直 import しない

## Hooks

- `pre-commit`: `pnpm lint-staged` のみ。codegen drift check は `pnpm lint` と CI の `pnpm check:codegen` で実行する
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

変更を始める前に、`docs/change-operation.md` に従って三軸を独立に決めます。

| 軸               | 値                                     | 判断内容                           |
| ---------------- | -------------------------------------- | ---------------------------------- |
| `Operation Lane` | `DIRECT` / `BEHAVIOR` / `ARCHITECTURE` | 振る舞い・構造をどの運用で扱うか   |
| `UX Mode`        | `NONE` / `CONTINUITY` / `SHAPE`        | 利用者に見える体験をどう扱うか     |
| `Review Depth`   | `STANDARD` / `DEEP`                    | 独立レビューをどの深さで実施するか |

- `DIRECT`: 観測可能な振る舞いも物質的な内部構造も変えません。OpenSpec Change は不要です。
- `BEHAVIOR`: 観測可能な振る舞いを変更します。`behavior-change` の OpenSpec Change が必要です。
- `ARCHITECTURE`: 物質的な内部構造を変更します。`architecture-change` の OpenSpec Change が必要です。
- `SHAPE` は UX の方向付けが必要な場合だけ使用し、実際の UI 変更にはプロダクトデザイナーの関与とデスクトップ・モバイル双方の実ブラウザ確認が必要です。
- `STANDARD` を既定とし、重要なセキュリティ、データ、外部契約、移行、領域横断の構造、活動中 Change との相互作用に危険がある場合は `DEEP` を選びます。

Change は次のコマンドで作成し、`openspec/changes/**` を手作業で作りません。

```bash
scripts/devcontainer/run.sh pnpm openspec new change <change-id> --schema behavior-change
scripts/devcontainer/run.sh pnpm openspec new change <change-id> --schema architecture-change
```

OpenSpec `1.8.0` は設定ファイルの既定スキーマを Change 作成時に参照しないため、`--schema` を省略しません。OpenCode の公式コア定義は `pnpm gen:openspec` で再生成し、`.opencode/commands/opsx-*.md` と `.opencode/skills/openspec-*/SKILL.md` を手編集しません。

`tasks.md` は粗い作業パッケージ台帳です。ファイル、補助処理、試験階層の詳細は、現在の作業パッケージと検証結果に基づき実装時に決めます。

```bash
scripts/devcontainer/run.sh pnpm lint:openspec:scenario -- --change <change-id>
scripts/devcontainer/run.sh pnpm lint:openspec:scenario -- --change <change-id> --require-test-references
scripts/devcontainer/run.sh pnpm lint:openspec:scenario
```
