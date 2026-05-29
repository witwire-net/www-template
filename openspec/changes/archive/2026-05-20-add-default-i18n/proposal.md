## Why

このテンプレートは、公開サイト、認証済みアプリ、管理コンソールを分離して提供します。多言語対応を後付けすると、表示文言、メール文面、Account 設定、Auth 導線、管理画面の責務が分散し、利用者が別端末で同じ Account を使っても言語が安定しません。

この変更では、Product を利用する主体を `Account` として明確に定義します。`Account` には、表示・通知に使う `AccountSetting` と、本人確認・セッション・認証器を扱う `Auth` がぶら下がります。locale は `AccountSetting.locale` として扱い、Auth の属性、端末ローカル状態、または UI 都合の一時設定として扱いません。

メール送信も Account の言語で一貫させます。復旧メールやデバイスリンク通知が UI と別言語で届くと、認証導線の信頼性が下がります。AccountSetting を Product BE の永続状態として扱い、画面、API、メール、lint の各層で同じ Account 中心の語彙を守れる状態にします。

## What Changes

- 公開サイトはパスベースのロケールルートを提供し、`/ja` と `/en` で表示言語を明示できるようにします。
- 公開サイト、認証済みアプリ、Admin Console の SvelteKit 面は、保守性を重視した `packages/frontend/i18n` の共有 i18n 実装を使います。外部 i18n dependency に頼らず、locale 定義、loader/config、typed translator、辞書 key coverage の共通処理を集約します。ただし locale JSON files は文言を表示する `packages/web`、`packages/frontend/app`、`packages/admin` がそれぞれ所有し、別 surface の辞書を共有または import しません。
- 認証済みアプリは、Account に属する `AccountSetting.locale` を Product BE に保存し、別端末でも同じ Account の標準言語で表示されるようにします。
- 未ログイン時の認証済みアプリは、AccountSetting をまだ取得できないため、端末に保存された対応 locale を優先し、存在しない場合はアクセス時のブラウザまたは OS 言語から対応ロケールへ解決します。
- refresh token の rotation 成功時は、Auth が token pair と AccountID を確定した後、HTTP composition が AccountSetting snapshot を読み込み、client が認証後の表示状態を Account の保存済み設定へ同期できるようにします。
- Product API 契約は Product Account と AccountSetting だけを表し、Admin operator locale を TypeSpec、OpenAPI、Product SDK、Go bindings に含めません。API model 名は `AccountSetting` / `AccountSettingSnapshot` / `AccountLocale` のように Account 所有を明示します。
- Product BE は Clean Architecture を徹底し、`packages/backend/internal/domain` が Account root、AccountSetting、Account.Auth projection を flat package で所有します。`packages/backend/internal/application` は AccountSetting use case と Auth use case を分離し、Auth 側は本人確認、session、token、passkey/recovery 認証フローだけを扱います。Auth は AccountSetting を所有せず、Account を代替する `AuthAccount` / `AuthSubject` も作りません。
- `AuthAccount` / `AuthAccountRepository` は許容せず、Account.Auth の認証用 projection である `AccountAuth` / `AccountAuthRepository` に置き換えます。`AccountAuth` は AccountSetting を持たず、Auth が必要とする AccountID、認証 identifier、email、status、session revoked boundary、passkey credentials だけを扱います。
- database は Account root から組み直し、`accounts`、`account_settings`、`account_passkey_credentials` を正規 table とします。locale は `account_settings.locale` に永続化します。Auth repository は `account_settings` を読んではならず、旧 table 名 `passkey_credentials` も残しません。
- 管理コンソールはオペレーター単位の言語設定を Admin-owned schema に保存し、別端末でも同じ言語で表示されるようにします。Admin operator locale は `packages/admin` 内で package-local に定義し、Product AccountSetting を共有しません。
- `packages/frontend/ui` は表示言語も i18n 実装も所有しません。i18n import が必要なコンポーネントは UI package に置かず、利用面の app/web/admin 側に配置します。`DeviceManager` は認証済みアプリの device/session 文言を持つ concrete component として `packages/frontend/app` 側へ置きます。
- Product BE の認証関連メールは、Auth が生成した配送 intent と AccountSetting.locale を composition して件名と本文を選択します。Auth domain/application は locale 値オブジェクトや AccountSetting mutation を所有しません。
- lint で、対象パッケージの多言語対応を迂回するハードコード文言、未登録ロケールキー、AccountSetting と Auth の境界違反を検知できるようにします。
- Account root、AccountSetting、Account.Auth、DB table、FE domain root を Account 中心へ揃えます。公開サイトの `/` はロケール付き URL へ誘導する入口として扱い、app 固有の `DeviceManager` は shared UI package から削除し、認証済み app 側へ移します。

## Spec Units

### New Spec Units

- `localization-fe`: 公開サイト、認証済みアプリ、管理コンソールのロケール選択、表示文言、AccountSetting 同期、設定 UI、lint 強制を扱います。横断懸念として、SvelteKit の公開・認証・管理境界、アクセシビリティ、SEO、文字列管理の保守性を含みます。
- `localization-be`: Product BE と Admin server 側の AccountSetting API、永続化、DB 制約、メール文言選択、Account と Auth の境界を扱います。横断懸念として、TypeSpec 契約、DB migration、セキュリティ、監査可能性を含みます。

### Modified Spec Units

- `admin-console-be`: database の Account.Auth child table を `account_passkey_credentials` に揃え、Admin view が禁止 table 名 `passkey_credentials` を参照しないことを保証します。

## Naming

- シナリオ ID のドメイン接頭辞は `LOCALIZATION` を使用します。フロントエンド要件は `LOCALIZATION-FE-S###`、バックエンド要件は `LOCALIZATION-BE-S###` とし、FE と BE で異なる接頭辞を使います。
- Product 利用主体は `Account` と呼びます。
- Account の言語設定は `AccountSetting.locale` と呼びます。
- Auth 側の認証用 projection は `AccountAuth` と呼びます。`AuthAccount`、`AuthSubject`、`AccountClientSettings` は新規実装に残してはなりません。

## Impact

- 影響パッケージ: `packages/web`、`packages/frontend/app`、`packages/frontend/domain`、`packages/frontend/ui`、`packages/frontend/i18n`、`packages/frontend/api`、`packages/admin`、`packages/backend`、`packages/typespec`。
- API 影響: Product API に認証済み AccountSetting の取得・更新エンドポイントを追加します。refresh response は Auth の token pair に AccountSetting snapshot を composition して返します。Product API 契約は Admin operator locale を含めません。
- DB 影響: database の初期 schema を Account root から組み直し、`accounts`、`account_settings`、`account_passkey_credentials`、Account 中心の admin views/functions を正とします。Admin-owned schema の `admin.operators` には operator locale を永続化します。
- 生成物影響: TypeSpec 変更により OpenAPI、frontend API SDK、Go server bindings の再生成が必要です。
- メール影響: 復旧、デバイスリンク、復旧完了、デバイス追加完了メールの言語選択が AccountSetting.locale に依存します。
- lint 影響: 対象パッケージに i18n 強制ルールを追加し、対象外にする文字列や例外条件を明確化します。`packages/frontend/ui` と `packages/frontend/domain` では `packages/frontend/i18n` と app/web/admin の i18n module import を禁止します。`packages/web`、`packages/frontend/app`、`packages/admin` は各自の locale JSON files だけを import し、互いの辞書を参照しません。
- セキュリティ影響: AccountSetting 更新は認証済み本人に限定し、Admin operator locale 更新は認証済みオペレーター本人に限定します。未知ロケールは fail-closed で拒否します。
- アーキテクチャ影響: Product Account root、AccountSetting、Account.Auth projection の境界を flat backend 構造で整理します。`internal/domain` は Account、AccountSetting、AccountAuth を所有し、`internal/application` は AccountSetting use case と Auth の credential/session/token/recovery use case を分離して所有します。
- フロントエンド境界影響: `frontend/i18n` は共通翻訳実装を担当し、locale JSON files は `web`、`frontend/app`、`admin` が表示面ごとに所有します。`frontend/app` は未認証 fallback と Account 同期を担当し、`frontend/domain` は `packages/frontend/domain/src/account` を Account domain root として AccountSetting API 協調を担当し、`frontend/ui` は言語、i18n import、固定 locale formatter を所有しません。
