## Why

現在のシステムは 1 アカウントにつき 1 件のパスキーしか保持できない設計になっており、`ReplacePasskey` が既存パスキーを全削除して新規 1 件に置き換える。このため、複数デバイスで認証したいユーザーがデバイスを追加するたびに既存のパスキーを失い、他のデバイスでのログインが即座に不可能になる。

パスキーは本来「デバイスごとに登録する」認証方式であり、MacBook・iPhone・セキュリティキーをそれぞれ登録して使い分けることがユーザーにとって自然な使い方だ。しかし現状の実装では複数デバイスを使う利用者が安全にアカウントを維持できず、信頼性の高い認証体験を提供できていない。

## What Changes

- 認証済みユーザーが追加のパスキーを登録できる新しい API エンドポイントを追加する。
- 認証済みユーザーが登録済みパスキーの一覧を取得できる API エンドポイントを追加する。
- 認証済みユーザーが指定したパスキーを削除できる API エンドポイントを追加する（最後の 1 件は削除不可）。
- ドメインモデル・永続化層を複数パスキー対応に変更する。
- アプリ内にパスキー管理画面を追加し、登録済みパスキーの一覧・追加・削除操作を提供する。
- **BREAKING**: `ReplacePasskey` の全削除挙動を廃止し、`AddPasskey` に置き換える。
- **新規**: OTP ハンドオフによる新端末へのパスキー追加フローを導入する。ログイン済み端末で既存パスキーによる再認証を行い 6 桁 OTP（有効期限 5 分）を発行。新端末は OTP を入力して公開エンドポイント（`POST /api/v1/auth/passkey/add/*`）経由でパスキーを登録する。

## Spec Units

### New Spec Units

なし

### Modified Spec Units

- `auth-be`: 認証済みアカウントによるパスキー管理 API（一覧・追加・削除）の追加。最終 1 件削除防止制約の追加。OTP ハンドオフによる新端末パスキー登録フローの追加（`POST /api/v1/passkeys/otp` 発行、`POST /api/v1/auth/passkey/add/start`・`POST /api/v1/auth/passkey/add/finish` 公開エンドポイント）。
- `auth-fe`: 認証済みアプリ内パスキー管理画面（一覧・追加・削除 UI）の追加。

## Naming

- BE の Scenario ID プレフィックス: `AUTH-BE`（例: `AUTH-BE-S014`）
- FE の Scenario ID プレフィックス: `AUTH-FE`（例: `AUTH-FE-S010`）
- 既存の `AUTH-BE-S001〜S013`、`AUTH-FE-S001〜S009` に続けて採番する。

## Impact

- **API**: TypeSpec に `GET /api/v1/passkeys`、`POST /api/v1/passkeys/start`、`POST /api/v1/passkeys/finish`、`DELETE /api/v1/passkeys/{id}`、`POST /api/v1/passkeys/otp`（OTP 発行）、`POST /api/v1/auth/passkey/add/start`、`POST /api/v1/auth/passkey/add/finish`（新端末向け公開エンドポイント）を追加。
- **DB**: `passkey_credentials` テーブルは既存のままだが、ドメイン・永続化層が複数件を扱えるよう変更。既存データは無変換で利用可能（後方互換）。
- **Backend**: `domain.AuthAccount` の単一パスキーフィールドを `[]PasskeyCredential` に変更。`persistence` 層の `ReplacePasskey` を廃止し、`AddPasskey` / `ListPasskeys` / `DeletePasskey` に置き換える。
- **Frontend**: `packages/frontend/api` に新規エンドポイントのクライアント生成。`packages/frontend/domain` に passkey 管理ユースケース追加。`packages/frontend/app` にパスキー管理ページ追加。
- **Security**: 削除操作は bearer session による本人確認を要する。最終 1 件のパスキーは削除不可とし、ロックアウトを防止する。
- **Migration**: スキーマ変更なし。アプリ層の変更のみ。
