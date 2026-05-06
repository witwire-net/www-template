## 1. TypeSpec 契約変更（API コントラクトのソース）

- [ ] 1.1 `packages/typespec/src/models/auth.tsp` から `PasskeyOtpResponse`, `PasskeyAddByOtpStartRequest`, `PasskeyAddByOtpFinishRequest` を削除し、`TokenKind` enum (`"recovery"` | `"device-link"`) を追加、`RecoveryConsumeResponse` に `kind: TokenKind` を追加
- [ ] 1.2 `packages/typespec/src/routes/v1/auth.tsp` から `startPasskeyAdditionByOtp` (L122-139), `finishPasskeyAdditionByOtp` (L141-157), `issuePasskeyOtp` (L365-395) を削除
- [ ] 1.3 `packages/typespec/src/routes/v1/auth.tsp` の `issuePasskeyOtp` と同じ namespace に `sendDeviceLink` エンドポイント（`POST /api/v1/passkeys/send-device-link`, bearer + `X-Reauth-Session` 必須, `operationId: sendDeviceLink`）を追加。レスポンスは `PasskeyOtpResponse` と同じ `{requestId, issued}` を返す

## 2. コード生成

- [ ] 2.1 `pnpm gen` を実行し TypeSpec → OpenAPI → Go bindings → フロントエンド SDK を再生成
- [ ] 2.2 `pnpm check:codegen` でコード生成ドリフトが無いことを確認

## 3. Go ドメイン層の変更

- [ ] 3.1 `packages/backend/internal/auth/domain/device_login_handoff.go` を削除（`DeviceLoginHandoff` 型、`NewDeviceLoginHandoff`、全メソッド）
- [ ] 3.2 `packages/backend/internal/auth/domain/recovery_token.go` に `TokenKind` 型（`type TokenKind string`, `const TokenKindRecovery TokenKind = "recovery"`, `const TokenKindDeviceLink TokenKind = "device-link"`）を追加
- [ ] 3.3 `RecoveryToken` 構造体に `kind TokenKind` フィールドを追加、`NewRecoveryToken` のシグネチャに `kind` パラメータ追加、`kind` が空でないことをバリデーション
- [ ] 3.4 `RecoverySession` 構造体に `kind TokenKind` フィールドを追加、`NewRecoverySession` のシグネチャに `kind` パラメータ追加
- [ ] 3.5 `ReconstituteRecoveryToken` / `ReconstituteRecoverySession` に `kind` パラメータ追加
- [ ] 3.6 `RecoveryToken` / `RecoverySession` に `Kind() TokenKind` getter 追加
- [ ] 3.7 ドメインエラー `ErrInvalidTokenKind` を追加

## 4. Go アプリケーション層の変更 — 削除

- [ ] 4.1 `packages/backend/internal/auth/application/auth_service.go` の `hashSecret()` (L748-755) を削除（OTP 削除に伴う、ペッパー付き HMAC は Valkey 層へ移動）
- [ ] 4.2 `IssuePasskeyOtp()` (L757-802) を削除
- [ ] 4.3 `StartAddPasskeyByOtp()` (L804-872) を削除
- [ ] 4.4 `FinishAddPasskeyByOtp()` (L874-933) を削除
- [ ] 4.5 `checkHandoffRateLimits()` (L935-960) を削除
- [ ] 4.6 `resolveAndConsumeHandoff()` (L962-991) を削除
- [ ] 4.7 `packages/backend/internal/auth/application/auth_errors.go` から `ErrInvalidOtp`, `ErrOtpExpiredOrConsumed` を削除
- [ ] 4.8 `packages/backend/internal/auth/application/auth_contracts.go` から `PasskeyOtpSender` インターフェース、`UsePasskeyOtpSender` メソッド、`passkeyOtpSender` フィールドを削除
- [ ] 4.9 `AuthService` 構造体から `passkeyOtpSender` フィールドを削除

## 5. Go アプリケーション層の変更 — kind-aware 修正

- [ ] 5.1 `issueRecoveryDelivery()` (L557-579) に `kind TokenKind` パラメータを追加し、`RecoveryDelivery` に `Kind` を設定、`NewRecoveryToken` に `kind` を伝播。RecoveryURL は recovery も device-link も同一の `AccountRecoveryURLBase` を使用
- [ ] 5.2 `RequestPasskeyRecovery()` (L257-297) で `issueRecoveryDelivery` に `TokenKindRecovery` を渡すよう修正
- [ ] 5.3 `ConsumeRecoveryToken()` (L299-341) で consumed `RecoveryToken.Kind()` を `NewRecoverySession(kind)` に伝播し、`RecoverySession` レスポンスに `Kind` を含める
- [ ] 5.4 `RecoveryDelivery` DTO に `Kind TokenKind` フィールドを追加
- [ ] 5.5 `RecoverySession` DTO に `Kind TokenKind` フィールドを追加

## 6. Go アプリケーション層の変更 — 新規追加

- [ ] 6.1 `executeDeviceLink(ctx, accountID, sessionID string) (DeviceLinkIssued, error)` メソッドを追加。`VerifyReauthSession(kind="device-link")` → `issueRecoveryDelivery(kind=device-link)` → `deviceLinkSender.SendDeviceLink(...)` → `{Issued: true}` を返す
- [ ] 6.2 `DeviceLinkIssued` DTO を追加（`RequestID string, Issued bool`）
- [ ] 6.3 `auth_contracts.go` に `SendDeviceLinkSender` インターフェース追加（`SendDeviceLink(ctx, delivery RecoveryDelivery) error`）、`UseDeviceLinkSender` メソッド、`deviceLinkSender` フィールド追加
- [ ] 6.4 `auth_contracts.go` に `SendRecoveryCompleteSender` インターフェース追加（`SendRecoveryComplete(ctx, accountID, email string) error`）、`SendDeviceLinkCompleteSender` インターフェース追加（`SendDeviceLinkComplete(ctx, accountID, email string) error`）

## 7. Go アプリケーション層の変更 — RegisterPasskey 後処理

- [ ] 7.1 `RegisterPasskey()` (L398-460) の recovery session 消費後、パスキー登録成功後に kind を評価する後処理ブロックを追加
- [ ] 7.2 `kind=recovery` の場合：`sessionStore.RevokeAllForAccount(ctx, accountID)` を呼び出し、fire-and-forget で `recoveryCompleteSender.SendRecoveryComplete(ctx, accountID, email)` を実行
- [ ] 7.3 `kind=device-link` の場合：セッション失効は行わず、fire-and-forget で `deviceLinkCompleteSender.SendDeviceLinkComplete(ctx, accountID, email)` を実行
- [ ] 7.4 後処理のメール送信失敗は `slog.ErrorContext` で記録し、registration の成功レスポンスは妨げない
- [ ] 7.5 `sessionStore interface` に `RevokeAllForAccount(ctx context.Context, accountID string) error` を追加
- [ ] 7.6 `TokenService` に `RevokeAllForAccount(ctx context.Context, accountID string) error` メソッドを追加（sessionStore の全セッション削除 + refreshStore の全トークン失効）

## 8. Go Valkey 永続化層の変更

- [ ] 8.1 `packages/backend/internal/adapters/persistence/valkey/auth_state_repository.go` の `SaveDeviceLoginHandoff`, `FindDeviceLoginHandoffByEmailAndOtp`, `ConsumeDeviceLoginHandoff`, `GetDeviceLoginHandoff` メソッドを削除
- [ ] 8.2 `deviceLoginHandoffRecord` 構造体を削除
- [ ] 8.3 `hashSecret()` (L323-326) を raw SHA-256 から HMAC-SHA256 + `SecretHashKey` に変更。`SecretHashKey` はリポジトリ構築時のコンストラクタパラメータとして注入
- [ ] 8.4 `recoveryTokenRecord` に `Kind string` フィールド追加、`recoveryTokenRecordFromDomain` と `toDomain` で `Kind` をマッピング
- [ ] 8.5 `recoverySessionRecord` に `Kind string` フィールド追加、`recoverySessionRecordFromDomain` と `toDomain` で `Kind` をマッピング

## 9. Go Mailer 層の変更

- [ ] 9.1 `packages/backend/internal/adapters/mailer/account_recovery_sender.go` の `SendPasskeyOtp()` と `buildPasskeyOtpMessage()` を削除
- [ ] 9.2 `SendAccountRecovery()` は `delivery.Kind` を参照し、kind に応じてメール件名・文面を分岐するよう修正（既存の recovery 文面は維持、device-link は新規文面）
- [ ] 9.3 新規 `buildDeviceLinkMessage(from, email, url, requestID string) string` を追加。件名: "www-template device login link"、本文に device-link URL を含める
- [ ] 9.4 `AccountRecoverySender` に `sendRecoveryComplete` と `sendDeviceLinkComplete` の notification 送信メソッドを追加。件名例: "www-template passkey recovered" / "www-template passkey added on new device"

## 10. Go Router 層の変更

- [ ] 10.1 `packages/backend/internal/adapters/http/router.go` の `IssuePasskeyOtp` ハンドラ (L505-529) を削除
- [ ] 10.2 `StartPasskeyAdditionByOtp` ハンドラ (L532-553) を削除
- [ ] 10.3 `FinishPasskeyAdditionByOtp` ハンドラ (L556-569) を削除
- [ ] 10.4 新規 `SendDeviceLink` ハンドラを追加：`bearerTokenFromContext` → `AuthorizeSession` → `VerifyReauthSession(kind="device-link")` → `executeDeviceLink` → 200 応答
- [ ] 10.5 `RegisterPasskey` ハンドラのレスポンス生成で、session の `kind` 情報をレスポンスに含める（必要に応じて）

## 11. Go DI コンテナの変更

- [ ] 11.1 `packages/backend/internal/app/container.go` から `authSvc.UsePasskeyOtpSender(recoverySender)` を削除
- [ ] 11.2 `authSvc.UseDeviceLinkSender(recoverySender)` を追加（同一の `AccountRecoverySender` を使用）
- [ ] 11.3 `authSvc.UseRecoveryCompleteSender(recoverySender)` / `authSvc.UseDeviceLinkCompleteSender(recoverySender)` を追加（どちらも `AccountRecoverySender` に通知メソッドを実装）
- [ ] 11.4 `NewAuthStateRepository` に `SecretHashKey` を注入するよう修正

## 12. Go テスト更新

- [ ] 12.1 OTP 関連テスト（`TestIssuePasskeyOtp*`, `TestStartPasskeyAdditionByOtp*`, `TestFinishPasskeyAdditionByOtp*`, `TestHandoffRateLimit*`, OTP brute force シナリオテスト等）を `auth_service_test.go`, `auth_test.go` から削除
- [ ] 12.2 `stubStateRepo` から OTP 関連メソッド（`IssuePasskeyOtp`, `StartAddPasskeyByOtp` 等のモック）を削除
- [ ] 12.3 `stubAuditNotifier` から OTP 関連メソッドを削除
- [ ] 12.4 `[AUTH-BE-S047]` valid device-link token creates RecoverySession with kind=device-link の UT を追加 (`auth_service_test.go`)
- [ ] 12.5 `[AUTH-BE-S048]` RegisterPasskey with kind=device-link does not revoke sessions の UT を追加
- [ ] 12.6 `[AUTH-BE-S051]` RegisterPasskey with kind=recovery revokes all sessions の UT を追加
- [ ] 12.7 `[AUTH-BE-S052]` RegisterPasskey with kind=device-link preserves all sessions の UT を追加
- [ ] 12.8 `[AUTH-BE-S053]` notification failure does not block registration success の UT を追加
- [ ] 12.9 `[AUTH-BE-S049]` send-device-link with valid reauth returns 200 の IT を `auth_test.go` に追加
- [ ] 12.10 `[AUTH-BE-S050]` send-device-link without reauth returns 403 の IT を追加

## 13. フロントエンド領域（domain）の変更

- [ ] 13.1 `packages/frontend/domain/src/auth/passkey/addByOtp/` ディレクトリを削除（`hook.svelte.ts`, `state.ts`, `index.ts`）
- [ ] 13.2 `packages/frontend/domain/src/auth/passkey/index.ts` から `addByOtp` のエクスポートを削除
- [ ] 13.3 `packages/frontend/domain/src/auth/passkey/management/hook.svelte.ts` の `issueOtp` アクションを `sendDeviceLink` に改名し、`POST /api/v1/passkeys/send-device-link` を呼び出すよう変更
- [ ] 13.4 `packages/frontend/domain/src/auth/passkey/management/state.ts` の OTP 関連 state フィールドを device-link 用に改名（`otpIssued` → `deviceLinkSent`）
- [ ] 13.5 `packages/frontend/domain/src/auth/passkey/management/state.test.ts` 更新

## 14. フロントエンド領域（app UI）の変更

- [ ] 14.1 `packages/frontend/app/src/routes/(auth)/passkeys/add/` ディレクトリを削除
- [ ] 14.2 `packages/frontend/app/src/routes/(protected)/passkeys/+page.svelte` の `handleIssueOtp` を `handleSendDeviceLink` に改名し、`performReauth('device-link')` → `sendDeviceLink(reauthSession)` に変更。成功時は device-link 送信完了メッセージを表示
- [ ] 14.3 `packages/frontend/app/src/lib/profiles/PasskeyList.svelte` の `otpIssued` prop を `deviceLinkSent` に改名、表示文面をデバイスリンク用に変更（「ログイン有効化リンクを送信しました」「有効期限: 30分」）
- [ ] 14.4 `packages/frontend/app/src/routes/(auth)/login/recovery/consume/+page.svelte` で consume レスポンスの `kind` を参照し、遷移先を分岐（`kind=recovery` → `/login/recovery/register`、`kind=device-link` → 同一 register ページで device-link 用の完了メッセージを表示するために context を渡す）
- [ ] 14.5 `packages/frontend/domain/src/auth/recovery/hook.svelte.ts` の `consumeToken` 戻り値に `kind` を含めるよう変更

## 15. フロントエンド領域（API client）の変更

- [ ] 15.1 `pnpm gen` で再生成された SDK の API client 型・関数が正しいことを確認。OTP 関連メソッド（`issuePasskeyOtp`, `startPasskeyAdditionByOtp`, `finishPasskeyAdditionByOtp`）が削除されていること
- [ ] 15.2 生成された `sendDeviceLink` メソッドのシグネチャ（`X-Reauth-Session` ヘッダー必須）を確認
- [ ] 15.3 生成された `consumeRecoveryToken` レスポンス型に `kind` が含まれていることを確認

## 16. 統合テスト

- [ ] 16.1 `pnpm test:client` を実行しフロントエンドテストが全通過することを確認
- [ ] 16.2 OTP 関連の削除済みテストファイルが参照エラーなく排除されていることを確認
- [ ] 16.3 `pnpm test:server` を実行し Go バックエンドテストが全通過することを確認

## 17. コード品質チェック

- [ ] 17.1 `pnpm lint` を実行し lint エラーが無いことを確認
- [ ] 17.2 `pnpm check:codegen` を実行しコード生成ドリフトが無いことを確認
- [ ] 17.3 Go バックエンドの `go vet ./...` が通過することを確認

## 18. ドキュメント更新

- [ ] 18.1 `openspec/specs/auth-be/spec.md` に delta spec を適用（REMOVED: OTP 要件, MODIFIED: recovery token / register / throttle / webauthn 要件, ADDED: send-device-link / kind 後処理要件）
- [ ] 18.2 `openspec/specs/auth-fe/spec.md` に delta spec を適用（REMOVED: OTP UI 要件, MODIFIED: 管理画面 / 復旧導線 / no-store routes 要件, ADDED: デバイスリンク UI 要件）
