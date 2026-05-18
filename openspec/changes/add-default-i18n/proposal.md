## Why

このテンプレートは、公開サイト、認証済みアプリ、管理コンソールを分離して提供します。そのため、初期状態から多言語対応の責務境界を明確にしないと、画面文言、メール文面、言語切り替え、テスト観点が後から分散します。特に認証済みアプリと管理コンソールでは、端末ごとのブラウザ設定や一時的なローカル状態だけに依存すると、別端末からアクセスした利用者が毎回言語を切り替える必要があります。

メール送信も、利用者またはオペレーターが選択した言語に従う必要があります。画面だけを翻訳しても、復旧メールやデバイスリンク通知が別言語で届くと、認証導線の信頼性が下がります。言語設定はバックエンド側の永続設定として扱い、画面、API、メール、lint の各層で一貫して守れる状態にします。

## What Changes

- 公開サイトはパスベースのロケールルートを提供し、`/ja` と `/en` で表示言語を明示できるようにします。
- 公開サイトの `/` は、安全な既定言語またはブラウザ言語に基づくロケールルートへ誘導します。
- 認証済みアプリはアカウント単位の言語設定を Product BE に保存し、別端末でも同じ言語で表示されるようにします。
- 未ログイン時の認証済みアプリは、端末に保存されたローカル言語設定を優先し、存在しない場合はアクセス時のブラウザまたは OS 言語から対応ロケールへ解決します。
- アクセストークンリフレッシュ時は、言語設定を含むクライアント設定を DB から読み込み、client が認証後の表示状態を保存済み設定へ同期できるようにします。
- Product API 契約は Product account locale だけを表し、Admin operator locale を TypeSpec、OpenAPI、Product SDK、Go bindings に含めません。API model 名も `AccountLocale` / `AccountClientSettings` のように Product account 所有を明示します。
- Product BE は Clean Architecture を徹底し、account locale、account settings、client settings を `packages/backend/internal/account` の責務に分離します。`packages/backend/internal/auth` は本人確認、session、token、passkey/recovery 認証フローだけを扱い、locale や account settings を所有しません。既存の `AuthAccount` のように Product account aggregate と誤読される auth domain model は、認証主体だけを表す `AuthSubject` と credential/session モデルへリファクタリングします。
- 管理コンソールはオペレーター単位の言語設定を Admin DB に保存し、別端末でも同じ言語で表示されるようにします。
- Admin operator locale は `packages/admin` 内で package-local に定義し、Product TypeSpec/generated SDK や Product account locale model を共有しません。`ja` / `en` の値整合は lint/test で確認します。
- `packages/frontend/ui` は表示言語を所有せず、再利用コンポーネントは localized label、aria label、日時 formatter を呼び出し側から受け取るようにします。
- Product BE は、アカウント言語設定を取得・更新する認証済み API を提供します。
- Product BE の認証関連メールは、保存済みアカウント言語に基づいて件名と本文を選択します。
- Admin Console はログイン済みオペレーターの言語設定を読み込み、設定画面から更新できるようにします。
- lint で、対象パッケージの多言語対応を迂回するハードコード文言や未登録ロケールキーを検知できるようにします。
- **破壊的変更なし**。既存の主要導線は維持し、公開サイトの `/` はロケール付き URL へ誘導する入口として扱います。

## Spec Units

### New Spec Units

- `localization-fe`: 公開サイト、認証済みアプリ、管理コンソールのロケール選択、表示文言、設定 UI、クライアント状態、lint 強制を扱います。横断懸念として、SvelteKit の公開・認証・管理境界、アクセシビリティ、SEO、文字列管理の保守性を含みます。
- `localization-be`: Product BE と Admin server 側の言語設定 API、永続化、DB 制約、メール文言選択、認証境界を扱います。横断懸念として、TypeSpec 契約、DB migration、セキュリティ、監査可能性を含みます。

### Modified Spec Units

- なし。多言語対応は新しい横断責務として `localization-fe` と `localization-be` に集約します。

## Naming

シナリオ ID のドメイン接頭辞は `LOCALIZATION` を使用します。フロントエンド要件は `LOCALIZATION-FE-S###`、バックエンド要件は `LOCALIZATION-BE-S###` とし、FE と BE で異なる接頭辞を使います。

## Impact

- 影響パッケージ: `packages/web`、`packages/frontend/app`、`packages/frontend/domain`、`packages/frontend/ui`、`packages/frontend/api`、`packages/admin`、`packages/backend`、`packages/typespec`。
- API 影響: Product API に認証済みアカウント言語設定の取得・更新エンドポイントを追加します。Product API 契約は Admin operator locale を含めません。
- DB 影響: Product DB の `accounts` と Admin DB の `admin.operators` に言語設定を永続化する migration が必要です。
- 生成物影響: TypeSpec 変更により OpenAPI、frontend API SDK、Go server bindings の再生成が必要です。
- メール影響: 復旧、デバイスリンク、復旧完了、デバイス追加完了メールの言語選択が保存済み設定に依存します。
- lint 影響: 対象パッケージに i18n 強制ルールを追加し、対象外にする文字列や例外条件を明確化します。
- セキュリティ影響: 言語設定更新は認証済み本人または認証済みオペレーター本人に限定し、未知ロケールは fail-closed で拒否します。
- アーキテクチャ影響: Product account 設定のドメイン・ユースケース・repository port を `internal/account` に追加し、`internal/auth` から locale/account settings の所有を排除します。`AuthAccount` / `AuthAccountRepository` のような中途半端な account 命名は `AuthSubject` / `AuthSubjectRepository` へ整理します。
- フロントエンド境界影響: `frontend/app` は未認証 fallback と辞書適用を担当し、`frontend/domain` は Product account settings API 協調だけを担当し、`frontend/ui` は言語や固定 locale formatter を所有しません。
