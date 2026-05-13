## Why

現在、新端末へのパスキー追加には **OTP ハンドオフ** と **メールリカバリー** の2系統の実装が存在する。両者は本質的に「メールを信頼基点として新パスキーを登録する」という同一の操作であり、以下の問題がある：

1. **保守性の二重化**: `DeviceLoginHandoff` と `RecoveryToken` / `RecoverySession` という2つのドメイン型、2系統の Valkey 永続化、2系統の API エンドポイント、2系統のフロントエンドページが存在する
2. **セキュリティ強度の不整合**: リカバリートークンは SHA-256（ペッパーなし）だが OTP は HMAC-SHA256（ペッパーあり）。また OTP は 6 桁（~20bit）の低エントロピーで理論上総当たり可能
3. **UX の歪み**: OTP 方式では6桁のコードをメールから手入力する必要があり、URL クリックで完結するリカバリー方式より劣る

OTP を廃止し、URL トークン方式を **`kind` パラメータで分岐させる単一のフロー** に一本化することで、保守性・セキュリティ・UX のすべてを改善する。

## What Changes

- **REMOVED**: `POST /api/v1/passkeys/otp` エンドポイントと OTP ハンドオフ全機構（**BREAKING**）
- **REMOVED**: `POST /api/v1/auth/passkey/add/start` / `POST /api/v1/auth/passkey/add/finish` エンドポイント（**BREAKING**）
- **REMOVED**: フロントエンドの `/passkeys/add` ルートと OTP 入力 UI（**BREAKING**）
- **CHANGED**: `RecoveryToken` / `RecoverySession` に `kind` フィールド（`"recovery"` | `"device-link"`）を追加
- **CHANGED**: `hashSecret` をペッパー付き HMAC-SHA256 に統一
- **ADDED**: `POST /api/v1/passkeys/send-device-link` — 認証済み端末から新端末追加用の URL トークンを発行（bearer + reauth 必須）
- **ADDED**: パスキー登録完了時の後処理 — `kind=recovery` 時は全セッション強制失効、いずれの kind でも完了通知メールを送信
- **ADDED**: `SessionStore.RevokeAllForAccount` — アカウント単位のセッション全失効

## Spec Units

### Modified Spec Units

- **`auth-be`**: OTP ハンドオフ要件（`Requirement: OTP ハンドオフによる新端末へのパスキー追加`）を削除し、URL トークン方式による新端末追加（`POST /api/v1/passkeys/send-device-link`）要件を追加。RecoveryToken/RecoverySession に `kind` 追加、`hashSecret` 強化、パスキー登録完了時の kind 別後処理（セッション失効＋通知メール）を追加。
- **`auth-fe`**: OTP 発行・入力 UI 要件（`Requirement: パスキー管理ページから OTP を発行して新端末にパスキーを追加できる`）を削除し、URL トークン方式の新端末追加 UI（「新しい端末でログインを有効にする」→ reauth → リンク送信確認表示）要件を追加。既存のリカバリー消費・登録 UI は kind による分岐に対応。

## Naming

- 本変更は既存の Spec Unit `auth-be` / `auth-fe` を修正するため、Scenario ID の DOMAIN prefix は既存の `AUTH-BE-*` / `AUTH-FE-*` を継続使用する

## Impact

| 領域               | 影響                                                                                                                                                                             |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **API (TypeSpec)** | OTP 3 エンドポイント削除、send-device-link 1 エンドポイント追加、RecoveryConsumeResponse に `kind` 追加                                                                          |
| **Backend (Go)**   | `DeviceLoginHandoff` ドメイン型・永続化削除、`issueRecoveryDelivery` に `kind` 分岐追加、`RegisterPasskey` に後処理追加、`SessionStore.RevokeAllForAccount` 追加                 |
| **Valkey**         | `auth:handoff:*` キー廃止、`auth:recovery-token:*` スキーマに `kind` 追加                                                                                                        |
| **Mailer**         | `SendPasskeyOtp` 削除、`SendDeviceLink` 追加、`SendRecoveryComplete` / `SendDeviceLinkComplete` 通知追加                                                                         |
| **Frontend**       | `addByOtp/` ドメイン削除、`(auth)/passkeys/add/` ルート削除、管理画面の「新しい端末でログインを有効にする」ボタンを reauth → send-device-link に変更、消費ページに kind 分岐追加 |
| **Security**       | OTP の低エントロピー脆弱性解消、`hashSecret` のペッパー統一、復旧時のセッション強制失効追加                                                                                      |
| **Migration**      | Valkey キー構造変更のため既存 OTP handoff state は無効化                                                                                                                         |
