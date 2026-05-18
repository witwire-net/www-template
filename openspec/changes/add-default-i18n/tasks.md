## 1. 契約と永続化

- [ ] 1.1 `packages/typespec/src/models/localization.tsp` と `packages/typespec/src/routes/v1/account_settings.tsp` を追加し、`ja` / `en` の locale enum、account locale 取得・更新 request/response、認証済み `/api/v1/account/settings` 系 route を定義する。
- [ ] 1.2 `packages/typespec/main.tsp` に localization model と account settings route を読み込み、`pnpm gen` で OpenAPI、frontend SDK、Go bindings を再生成する。
- [ ] 1.3 `packages/backend/db/migrations/000007_add_account_locale.up.sql` と `.down.sql` を追加し、`accounts.locale` の既定値、対応値制約、rollback を用意する。
- [ ] 1.4 `packages/admin/prisma/admin/schema.prisma` と `packages/admin/prisma/admin/migrations/000002_add_operator_locale/migration.sql` を更新し、`admin.operators.locale` の既定値と対応値制約を用意する。
- [ ] 1.5 Product と Admin の locale 型・辞書で対応ロケール一覧を重複なく参照できるよう、各層の `ja` / `en` 定義を揃える。

## 2. Product BE 実装

- [ ] 2.1 `packages/backend/internal/auth/domain/locale.go` を追加し、locale の検証、正規化、既定値処理を実装する。
- [ ] 2.2 `packages/backend/internal/auth/domain/auth_account.go` と関連テストを更新し、AuthAccount が locale を保持・返却できるようにする。
- [ ] 2.3 `packages/backend/internal/auth/application/account_settings.go` と `auth_contracts.go` を更新し、認証済み account locale の取得・更新 use case と repository 契約を追加する。
- [ ] 2.4 `packages/backend/internal/adapters/persistence/postgres/auth_account_repository.go` と関連テストを更新し、`accounts.locale` の読み込み、既定値補完、更新、未対応値拒否を実装する。
- [ ] 2.5 `packages/backend/internal/adapters/http/router.go` と `account_settings_test.go` を更新し、account locale 取得・更新 handler、no-store response、auth failure、validation error を実装する。
- [ ] 2.6 Go HTTP テストを追加し、テスト名に `[LOCALIZATION-BE-S001]`、`[LOCALIZATION-BE-S002]`、`[LOCALIZATION-BE-S003]`、`[LOCALIZATION-BE-S004]`、`[LOCALIZATION-BE-S013]` を含めて account locale API の成功、更新、未対応 locale、未認証拒否、refresh response の DB client settings locale を確認する。
- [ ] 2.7 `packages/backend/internal/auth/application/auth_service.go` と `auth_contracts.go` を更新し、recovery/device-link delivery と完了メール送信に account locale を渡す。
- [ ] 2.8 `packages/backend/internal/adapters/mailer/localized_messages.go` を追加し、復旧、デバイスリンク、復旧完了、デバイス追加完了メールの日本語・英語件名と本文を実装する。
- [ ] 2.9 `packages/backend/internal/adapters/mailer/account_recovery_sender.go` と関連テストを更新し、保存済み account locale でメール文面が選択され、token が log/trace/error に出ないことを確認する。
- [ ] 2.10 Go domain、repository、mailer テストを追加し、テスト名に `[LOCALIZATION-BE-S005]`、`[LOCALIZATION-BE-S006]`、`[LOCALIZATION-BE-S007]`、`[LOCALIZATION-BE-S008]` を含めて既定 locale、復旧メール、デバイスリンク完了メール、DB 制約を確認する。

## 3. Frontend API と認証済みアプリ

- [ ] 3.1 `packages/frontend/api/src/sdk.ts` と `packages/frontend/api/src/api/client.ts` を更新し、生成 SDK の account locale 取得・更新 method を domain から使える形で公開する。
- [ ] 3.2 `packages/frontend/domain/src/localization/*` と domain export を追加し、`useAccountLocalization`、locale 型、account settings API wrapper、load/update action を実装する。
- [ ] 3.3 `packages/frontend/domain` と `packages/frontend/app` のテストを追加し、テスト名に `[LOCALIZATION-FE-S004]` を含めて保存済み account locale が state に反映され、認証済み app の navigation、heading、操作 label が保存済み locale で表示されることを確認する。
- [ ] 3.4 `packages/frontend/app/src/lib/i18n/*` を追加し、認証前 fallback、`localStorage` 優先 locale、browser/OS locale resolver、認証済み account locale の両方で使う `ja` / `en` 辞書を実装する。
- [ ] 3.5 `packages/frontend/app/src/routes/login/+page.svelte` を辞書文言に置き換え、テスト名に `[LOCALIZATION-FE-S006]` を含む component test で account API なしに `localStorage` locale を優先し、存在しない場合は browser/OS locale から fallback 文言を表示することを確認する。
- [ ] 3.6 `packages/frontend/app/src/routes/(protected)/+layout.svelte` と既存 protected routes を更新し、account locale 読み込み、localized navigation、localized overview copy を実装する。
- [ ] 3.7 `packages/frontend/app/src/routes/(protected)/settings/+page.svelte` を追加し、account locale の表示・更新 UI、成功表示、失敗表示を実装する。
- [ ] 3.8 認証済みアプリの component または E2E テストを追加し、テスト名に `[LOCALIZATION-FE-S005]` と `[LOCALIZATION-FE-S012]` を含めて account locale 更新後の表示切り替えと、refresh 後に DB client settings locale が fallback 表示を置き換えることを確認する。

## 4. 公開 Web 実装

- [ ] 4.1 `packages/web/src/lib/i18n.ts` を追加し、公開 Web 用 locale validator、既定 locale、`ja` / `en` 辞書、metadata 文言を定義する。
- [ ] 4.2 `packages/web/src/routes/+page.ts` を追加し、`/` から対応ロケール URL へ誘導する処理を実装する。
- [ ] 4.3 `packages/web/src/routes/+page.svelte` の公開ページ本体を `packages/web/src/routes/[locale]/+page.svelte` へ移し、URL locale に対応した文言と metadata を表示する。
- [ ] 4.4 `packages/web/src/routes/+layout.svelte` を更新し、公開 navigation と言語切り替えリンクを辞書または locale 定義から表示する。
- [ ] 4.5 Playwright または web package test を追加し、テスト名に `[LOCALIZATION-FE-S001]`、`[LOCALIZATION-FE-S002]`、`[LOCALIZATION-FE-S003]` を含めて root 誘導、locale 別表示、未対応 locale 処理を確認する。

## 5. Admin 実装

- [ ] 5.1 `packages/admin/src/lib/server/models/types.ts` と `operators.ts` を更新し、Operator 型と Prisma mapping に locale を追加する。
- [ ] 5.2 `packages/admin/src/app.d.ts` と `packages/admin/src/hooks.server.ts` を更新し、認証済み operator context に保存済み locale を含める。
- [ ] 5.3 `packages/admin/src/lib/server/services/operators/locale.ts` を追加し、認証済み本人の operator locale 更新、未対応 locale 拒否、他 operator 非変更を実装する。
- [ ] 5.4 Admin server/model テストを追加し、テスト名に `[LOCALIZATION-BE-S009]`、`[LOCALIZATION-BE-S010]`、`[LOCALIZATION-BE-S011]`、`[LOCALIZATION-BE-S012]` を含めて context 読み込み、本人更新、未対応値拒否、管理操作で locale が変わらないことを確認する。
- [ ] 5.5 `packages/admin/src/lib/i18n/*` を追加し、Admin 用 `ja` / `en` 辞書、代替言語、resolver を実装する。
- [ ] 5.6 `packages/admin/src/routes/+layout.server.ts` と `+layout.svelte` を更新し、operator locale と localized navigation/layout data を画面に渡す。
- [ ] 5.7 `packages/admin/src/routes/settings/+page.server.ts` と `+page.svelte` を更新し、operator locale 設定 form、action、成功/失敗表示、既存 operator 管理概要の辞書化を実装する。
- [ ] 5.8 Admin component または route test を追加し、テスト名に `[LOCALIZATION-FE-S007]`、`[LOCALIZATION-FE-S008]`、`[LOCALIZATION-FE-S009]` を含めて保存済み operator locale 表示、本人更新、認証前 fallback を確認する。

## 6. i18n lint 強制

- [ ] 6.1 `eslint.config.js` に対象 UI ソースのユーザー向け直書き文言検出ルールを追加し、許可する非翻訳 literal を狭く定義する。
- [ ] 6.2 `scripts/i18n/check-locales.ts` を追加し、`packages/web`、`packages/frontend/app`、`packages/admin` の辞書で `ja` / `en` の key 差分がないことを検証する。
- [ ] 6.3 `package.json` の `pnpm lint` 系 script に i18n 辞書網羅性チェックを組み込む。
- [ ] 6.4 `tests/i18n-lint.test.ts` を追加し、テスト名に `[LOCALIZATION-FE-S010]` と `[LOCALIZATION-FE-S011]` を含めて直書き文言と辞書欠落 key が失敗することを確認する。
- [ ] 6.5 既存 UI 文字列を辞書化し、lint 対象外にする必要がある技術識別子、route path、product name を最小限の例外として整理する。

## 7. 検証と仕上げ

- [ ] 7.1 `pnpm gen` を実行し、生成物を確認する。
- [ ] 7.2 `pnpm check:codegen` を実行し、契約と生成物の差分がないことを確認する。
- [ ] 7.3 `pnpm lint` を実行し、i18n lint、frontend 境界、backend guardrails が通ることを確認する。
- [ ] 7.4 `pnpm test:run` を実行し、Go、frontend、admin、ui のテストが通ることを確認する。
- [ ] 7.5 必要に応じて `pnpm build` を実行し、公開 Web、認証済みアプリ、Admin、Go backend の build を確認する。
- [ ] 7.6 実装中に仕様や設計から逸脱した場合、`openspec/changes/add-default-i18n` 配下の仕様・設計・タスクを更新し、シナリオ ID とテスト名の対応を維持する。
