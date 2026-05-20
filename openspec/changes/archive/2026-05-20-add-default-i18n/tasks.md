## 1. 契約と永続化

- [x] 1.1 `packages/typespec/src/models/account_settings.tsp` と `packages/typespec/src/routes/v1/account_settings.tsp` で、Product API 専用の `AccountLocale`、`AccountSetting`、`AccountSettingSnapshot`、AccountSetting 取得・更新 request/response、認証済み `/api/v1/account/settings` 系 route を定義する。TypeSpec model/doc/tag/operation は Admin operator locale を表現してはならず、Product Account と AccountSetting だけを表す。`AccountClientSettings` は作らない。
- [x] 1.2 `packages/typespec/src/models/auth.tsp` と `packages/typespec/src/routes/v1/auth.tsp` を更新し、refresh response が Auth の token pair と AccountSetting snapshot を返せるようにする。refresh operation 自体は Auth の token rotation であり、AccountSetting snapshot は transport/application composition の結果として表現する。
- [x] 1.3 `packages/typespec/main.tsp` に AccountSetting model と account settings route を読み込み、`pnpm gen` で OpenAPI、frontend SDK、Go bindings を再生成する。guard 名に `ARCH-BE-PRODUCT-API-CONTRACT` を含め、Product TypeSpec、OpenAPI、frontend SDK、Go bindings に Admin operator locale、operator settings、`/api/admin/**`、Admin 向け generated SDK symbols、`AccountClientSettings` が含まれないことを確認する。
- [x] 1.4 Product DB migration を Account root 前提で作り直し、`000001_create_accounts.*.sql`、`000002_create_account_settings.*.sql`、`000003_create_account_passkey_credentials.*.sql`、`000004_add_account_status.*.sql`、`000005_create_admin_views.*.sql`、`000006_create_admin_functions.*.sql` に揃える。Product DB の正規形は `accounts`、`account_settings`、`account_passkey_credentials`、Account 中心の admin views/functions とする。`passkey_credentials`、`auth_accounts`、`accounts.locale`、旧 schema 併存 path は作らない。migration/admin view/function tests を更新し、テスト名に `[ADMIN-CONSOLE-BE-S007]`、`[ADMIN-CONSOLE-BE-S008]`、`[ADMIN-CONSOLE-BE-S009]`、`[ADMIN-CONSOLE-BE-S010]`、`[ADMIN-CONSOLE-BE-S011]`、`[ADMIN-CONSOLE-BE-S012]`、`[ADMIN-CONSOLE-BE-S013]`、`[ADMIN-CONSOLE-BE-S037]`、`[ADMIN-CONSOLE-BE-S038]`、`[ADMIN-CONSOLE-BE-S042]`、`[ADMIN-CONSOLE-BE-S043]`、`[ADMIN-CONSOLE-BE-S044]` を含める。
- [x] 1.5 `packages/admin/prisma/admin/schema.prisma` と `packages/admin/prisma/admin/migrations/000002_add_operator_locale/migration.sql` を更新し、`admin.operators.locale` の既定値と対応値制約を用意する。Admin operator locale は Product AccountSetting を参照してはならない。
- [x] 1.6 Product Account、AccountSetting、Product TypeSpec、frontend app/domain、frontend/ui、public web、Admin server の対応ロケール値を `ja` / `en` に揃える。ただし runtime 型や実装 module は所有境界ごとに分け、Admin operator locale が Product TypeSpec/generated SDK や Product AccountSetting model を import しないこと、Product BE の AccountSetting 型が `packages/backend/internal/domain` にあり Auth use case / repository の所有物になっていないことを guard で確認する。
- [x] 1.7 `packages/frontend/i18n` を共有 frontend i18n 実装として構築し、locale 定義、loader/config、JSON catalog loader、typed translator、formatter、key coverage utility を実装する。`packages/frontend/i18n` は app/web/admin の locale JSON files を持ってはならない。`packages/web`、`packages/frontend/app`、`packages/admin` は各自の `src/lib/i18n/messages/{locale}/{namespace}.json` を所有し、それを `@www-template/i18n` に渡して使う。巨大な単一辞書や場当たり的な翻訳関数を作ってはならない。外部 i18n dependency は追加せず、`packages/frontend/ui` と `packages/frontend/domain` は `@www-template/i18n` や app/web/admin の i18n module を import してはならない。

## 2. Product Account / Auth BE 実装

- [x] 2.1 `packages/backend/internal/domain/account.go`、`account_setting.go`、`account_locale.go` を追加し、Product 利用主体としての Account、Account に属する AccountSetting、AccountSetting.locale の検証、正規化、既定値、snapshot を flat domain に実装する。
- [x] 2.2 `packages/backend/internal/application/contracts.go`、`account_setting_service.go`、`account_setting_snapshot.go` を追加し、認証済み AccountSetting の取得・更新 use case、refresh response 用 AccountSetting snapshot 読み込み、AccountSetting repository port を flat application に実装する。
- [x] 2.3 `packages/backend/internal/adapter/postgres/account_setting_repository.go` と関連テストを追加し、`account_settings.locale` の読み込み、作成、更新、未対応値拒否を AccountSetting repository adapter として実装する。Account 作成時は AccountSetting を同一 Account の必須 child として作り、AccountSetting 欠落を通常状態として扱わない。
- [x] 2.4 旧 AuthAccount model と関連テストを削除し、`packages/backend/internal/domain/account_auth.go` / `account_auth_test.go` を追加する。`AccountAuth` は Account にぶら下がる Auth projection として AccountID、identifier、email、status、session revoked boundary、passkey credentials だけを扱い、AccountSetting、locale、snapshot、単一 credential accessor を持ってはならない。`AuthSubject` は作らない。
- [x] 2.5 `packages/backend/internal/application/auth_contracts.go` と利用箇所を更新し、`AuthAccountRepository` を `AccountAuthRepository` に置き換える。Auth application の repository port は Account.Auth projection だけを扱い、AccountSetting DTO、locale mutation、AccountSetting snapshot を含めてはならない。
- [x] 2.6 `packages/backend/internal/adapter/postgres/auth_account_repository.go` / `auth_account_repository_test.go` を `account_auth_repository.go` / `account_auth_repository_test.go` に置き換える。Account.Auth repository は `accounts` と `account_passkey_credentials` から認証に必要な projection だけを復元し、`account_settings` table を読んではならない。禁止 table 名 `passkey_credentials` を参照してはならない。
- [x] 2.7 `packages/backend/internal/app/container.go` と `packages/backend/internal/adapter/http/router.go` を更新し、HTTP handler で Auth の bearer session 認可結果 AccountID を AccountSetting service へ渡す。AccountSetting endpoint と refresh response は Auth/AccountSetting use case を application 境界で合成する。
- [x] 2.8 Go HTTP テストを追加し、テスト名に `[LOCALIZATION-BE-S001]`、`[LOCALIZATION-BE-S002]`、`[LOCALIZATION-BE-S003]`、`[LOCALIZATION-BE-S004]`、`[LOCALIZATION-BE-S013]` を含めて AccountSetting API の成功、更新、未対応 locale、未認証拒否、refresh response の DB AccountSetting snapshot locale を確認する。Auth/AccountSetting 合成境界は guard 名 `ARCH-BE-REFRESH-COMPOSITION` で確認する。
- [x] 2.9 `packages/backend/internal/application/auth_service.go` と delivery DTO を更新し、recovery/device-link delivery と完了メール送信では Auth が AccountID、email、URL、token kind などの配送 intent だけを生成するようにする。Auth use case は AccountSetting reader、locale 値オブジェクト、AccountSetting mutation を所有しない。
- [x] 2.10 `packages/backend/internal/adapter/mailer/localized_messages.go` を追加し、復旧、デバイスリンク、復旧完了、デバイス追加完了メールの日本語・英語件名と本文を実装する。locale 正規化は AccountSetting.locale 定義に従う。
- [x] 2.11 `packages/backend/internal/adapter/mailer/account_recovery_sender.go` と関連テストを更新し、Auth delivery intent と AccountSetting.locale を composition してメール文面が選択され、token が log/trace/error に出ないことを確認する。
- [x] 2.12 Go account domain、AccountSetting repository、mailer テストを追加し、テスト名に `[LOCALIZATION-BE-S005]`、`[LOCALIZATION-BE-S006]`、`[LOCALIZATION-BE-S007]`、`[LOCALIZATION-BE-S008]` を含めて既定 locale、復旧メール、デバイスリンク完了メール、DB 制約を確認する。
- [x] 2.13 backend package boundary テストまたは grep guard を追加し、テスト名に `[LOCALIZATION-BE-S014]`、guard 名に `ARCH-BE-ACCOUNT-AUTH-SUBORDINATION` と `ARCH-BE-AUTH-NO-ACCOUNT-SETTING` を含めて、AccountSetting 値オブジェクトは `packages/backend/internal/domain`、AccountSetting use case、snapshot DTO、AccountSetting repository port は `packages/backend/internal/application` に存在し、Auth use case / repository が AccountSetting を所有しないこと、さらに `AuthAccount` / `AuthSubject` / `AuthAccountRepository` symbols が残らず `AccountAuth` / `AccountAuthRepository` が Account.Auth projection だけを扱うこと、Auth repository が `passkey_credentials` table を参照しないことを確認する。

## 3. Frontend API と認証済みアプリ

- [x] 3.1 `packages/frontend/api/src/sdk.ts` と `packages/frontend/api/src/api/client.ts` を更新し、生成 SDK の AccountSetting 取得・更新 method を Account API package の public wrapper として公開する。`packages/frontend/api/src/api/account_settings.ts` などの feature-specific wrapper file は作らない。
- [x] 3.2 `packages/frontend/domain/src/account/*` と domain export を実装し、`useAccount`、Account 型、AccountSetting 型、AccountSetting snapshot、load/update action を Account domain として扱う。Domain 配下に `account_setting_api.ts`、`account_settings_api.ts`、`*_api.ts` などの API wrapper file を作ってはならず、generated SDK 直接 import も禁止する。FE domain は `account-settings` という root を持ってはならない。
- [x] 3.3 `packages/frontend/domain` と `packages/frontend/app` のテストを追加し、テスト名に `[LOCALIZATION-FE-S004]` を含めて保存済み Account.setting.locale が `useAccount` の Account state に反映され、認証済み app の navigation、heading、操作 label が保存済み locale で表示されることを確認する。domain/API 配置境界は guard 名 `ARCH-FE-DOMAIN-API-BOUNDARY` で、domain Account が API wrapper file や generated SDK import を持たないこと、API wrapper が `packages/frontend/api/src/api/client.ts` に集約されていること、`packages/frontend/domain/src/account-settings` と `useAccountSetting` root entrypoint が存在しないことを確認する。
- [x] 3.4 `packages/frontend/app` を `@www-template/i18n` に接続し、app-owned locale JSON files、typed translator、認証前 fallback、`localStorage` 優先 locale、browser/OS locale resolver、認証済み AccountSetting.locale 同期を実装する。app route 内に独自辞書や ad hoc translator を作ってはならない。
- [x] 3.5 `packages/frontend/app/src/routes/login/+page.svelte` を辞書文言に置き換え、テスト名に `[LOCALIZATION-FE-S006]` を含む component test で AccountSetting API なしに `localStorage` locale を優先し、存在しない場合は browser/OS locale から fallback 文言を表示することを確認する。
- [x] 3.6 `packages/frontend/app/src/routes/(protected)/+layout.svelte` と protected routes を更新し、Account state 読み込み、localized navigation、localized overview copy を実装する。
- [x] 3.7 `packages/frontend/app/src/routes/(protected)/settings/+page.svelte` を追加し、AccountSetting.locale の表示・更新 UI、成功表示、失敗表示を実装する。
- [x] 3.8 認証済みアプリの component または E2E テストを追加し、テスト名に `[LOCALIZATION-FE-S005]` と `[LOCALIZATION-FE-S012]` を含めて AccountSetting.locale 更新後の表示切り替えと、refresh 後に DB AccountSetting snapshot locale が fallback 表示を置き換えることを確認する。
- [x] 3.9 `packages/frontend/ui` の再利用コンポーネントから固定言語文言、`ja-JP` / `en-US` などの固定 locale formatter、app 固有の認証文言を除去する。`packages/frontend/ui/src/components/device-manager/device-manager.svelte` は削除し、`packages/frontend/app/src/components/device-manager/device-manager.svelte` へ移す。移動後の `DeviceManager` は app-owned locale JSON files と `@www-template/i18n` から作った label / aria label / date-time formatter を使い、UI package は reusable primitive だけを提供する。`@www-template/i18n` や app/web/admin i18n module import が必要な concrete component は `packages/frontend/app`、`packages/web`、または `packages/admin` 側へ移す。テスト名に `[LOCALIZATION-FE-S013]` を含めて DeviceManager が app-owned component として表示されることを確認する。guard 名に `ARCH-FE-UI-LOCALIZED-PROPS` と `ARCH-FE-UI-NO-I18N-IMPORT` を含めて UI package が locale/i18n を所有せず、`device-manager` を含まないことを確認する。

## 4. 公開 Web 実装

- [x] 4.1 `packages/web` を `@www-template/i18n` に接続し、web-owned locale JSON files、public locale validator、既定 locale、typed translator、metadata 文言を定義する。web route 内に独自辞書や ad hoc translator を作ってはならない。
- [x] 4.2 `packages/web/src/routes/+page.ts` を追加し、`/` から対応ロケール URL へ誘導する処理を実装する。
- [x] 4.3 `packages/web/src/routes/+page.svelte` の公開ページ本体を `packages/web/src/routes/[locale]/+page.svelte` へ移し、URL locale に対応した文言と metadata を表示する。
- [x] 4.4 `packages/web/src/routes/+layout.svelte` を更新し、公開 navigation と言語切り替えリンクを辞書または locale 定義から表示する。
- [x] 4.5 Playwright または web package test を追加し、テスト名に `[LOCALIZATION-FE-S001]`、`[LOCALIZATION-FE-S002]`、`[LOCALIZATION-FE-S003]` を含めて root 誘導、locale 別表示、未対応 locale 処理を確認する。

## 5. Admin 実装

- [x] 5.1 `packages/admin/src/lib/server/models/operator_locale.ts`、`types.ts`、`operators.ts` を更新し、Admin package-local の `OperatorLocale`、Operator 型、Prisma mapping に locale を追加する。Admin operator locale 型や validator は Product TypeSpec/generated SDK/Product AccountSetting model を import してはならない。
- [x] 5.2 `packages/admin/src/app.d.ts` と `packages/admin/src/hooks.server.ts` を更新し、認証済み operator context に保存済み locale を含める。
- [x] 5.3 `packages/admin/src/lib/server/services/operators/locale.ts` を追加し、認証済み本人の operator locale 更新、未対応 locale 拒否、他 operator 非変更を実装する。未知の永続 locale は既定値へ黙って丸めず、DB 制約違反または server error として fail-closed に扱う。
- [x] 5.4 Admin server/model テストを追加し、テスト名に `[LOCALIZATION-BE-S009]`、`[LOCALIZATION-BE-S010]`、`[LOCALIZATION-BE-S011]`、`[LOCALIZATION-BE-S012]` を含めて context 読み込み、本人更新、未対応値拒否、管理操作で locale が変わらないことを確認する。
- [x] 5.5 `packages/admin` を `@www-template/i18n` に接続し、admin-owned locale JSON files、typed translator、代替言語、resolver を実装する。Admin route 内に独自辞書や ad hoc translator を作ってはならない。
- [x] 5.6 `packages/admin/src/routes/+layout.server.ts` と `+layout.svelte` を更新し、operator locale と localized navigation/layout data を画面に渡す。
- [x] 5.7 `packages/admin/src/routes/settings/+page.server.ts` と `+page.svelte` を更新し、operator locale 設定 form、action、成功/失敗表示、operator 管理概要の辞書化を実装する。
- [x] 5.8 Admin component または route test を追加し、テスト名に `[LOCALIZATION-FE-S007]`、`[LOCALIZATION-FE-S008]`、`[LOCALIZATION-FE-S009]` を含めて保存済み operator locale 表示、本人更新、認証前 fallback を確認する。
- [x] 5.9 Admin boundary guard を追加し、guard 名に `ARCH-ADMIN-LOCALE-INDEPENDENCE` を含めて `packages/admin` が operator locale のために Product TypeSpec/generated SDK、`@www-template/api`、Product AccountSetting model を import しないことを確認する。

## 6. i18n lint と境界強制

- [x] 6.1 `eslint.config.js` に対象 UI ソースのユーザー向け直書き文言検出ルールと frontend package boundary 更新を追加し、`packages/web`、`packages/frontend/app`、`packages/admin` から `@www-template/i18n` を利用できるようにしつつ、許可する非翻訳 literal を狭く定義する。対象には `packages/frontend/ui` も含め、再利用 UI が app/admin/web の表示言語を所有しないこと、`packages/frontend/ui` と `packages/frontend/domain` が `@www-template/i18n` や app/web/admin の i18n module を import しないこと、app/web/admin が互いの locale JSON files を import しないことを検出する。自動テスト名に `[LOCALIZATION-FE-S011]` を含める。
- [x] 6.2 `scripts/i18n/check-locales.ts` を追加し、`packages/web`、`packages/frontend/app`、`packages/admin` が所有する locale JSON files と `packages/frontend/ui` の UI label contract で `ja` / `en` の key 差分がないことを検証する。`packages/frontend/i18n` 配下に surface-specific locale JSON files が存在しないことも確認する。自動テスト名に `[LOCALIZATION-FE-S010]` を含める。
- [x] 6.3 `AGENTS.md`、`CODING_STANDARDS.md`、`eslint.config.js` の frontend dependency boundary を更新し、`packages/web`、`packages/frontend/app`、`packages/admin` が `@www-template/i18n` を利用できることを正式に許可する。同時に `packages/frontend/ui` と `packages/frontend/domain` が `@www-template/i18n`、app/web/admin の i18n module、surface-owned locale JSON files を import できないことを明記し、機械検証にも反映する。
- [x] 6.4 `package.json` の `pnpm lint` 系 script に i18n 辞書網羅性チェックを組み込み、ESLint 対象に `packages/frontend/i18n` を含める。
- [x] 6.5 `package.json` の `pnpm check` と `pnpm test:run` に `@www-template/i18n` の typecheck と unit test を含める。i18n package に実行すべき test がない状態で完了扱いにしてはならない。
- [x] 6.6 `tests/i18n-lint.test.ts` を追加し、テスト名に `[LOCALIZATION-FE-S010]` と `[LOCALIZATION-FE-S011]` を含め、guard 名に `ARCH-I18N-LITERAL-GUARD` と `ARCH-I18N-DICTIONARY-COVERAGE` を含めて直書き文言と辞書欠落 key が失敗することを確認する。
- [x] 6.7 UI 文字列と固定 locale formatter を辞書化または呼び出し側注入に置き換え、lint 対象外にする必要がある技術識別子、route path、product name、非表示テスト fixture を最小限の例外として整理する。
- [x] 6.8 backend source guard を追加し、`AuthAccount`、`AuthSubject`、`AccountClientSettings`、Auth use case / repository による AccountSetting/locale 所有、Auth repository の `account_settings` 読み取り、Auth repository の `passkey_credentials` table 参照を拒否する。guard 名に `ARCH-BE-ACCOUNT-AUTH-SUBORDINATION` と `ARCH-BE-AUTH-NO-ACCOUNT-SETTING` を含める。

## 7. 検証と仕上げ

- [x] 7.1 `pnpm gen` を実行し、生成物を確認する。
- [x] 7.2 `pnpm check:codegen` を実行し、契約と生成物の差分がないことを確認する。
- [x] 7.3 `pnpm lint` を実行し、i18n lint、frontend 境界、backend guardrails、Account/Auth boundary guard が通ることを確認する。
- [x] 7.4 `pnpm check` を実行し、TypeScript/Svelte/TypeSpec/Go build 検証が通ることを確認する。
- [x] 7.5 `pnpm test:run` を実行し、Go、frontend、admin、ui、i18n package のテストが通ることを確認する。
- [x] 7.6 必要に応じて `pnpm build` を実行し、公開 Web、認証済みアプリ、Admin、Go backend の build を確認する。
- [x] 7.7 実装中に仕様や設計から逸脱した場合、`openspec/changes/add-default-i18n` 配下の仕様・設計・タスクを更新し、仕様シナリオ ID、設計 guard 名、テスト名の対応を維持する。
