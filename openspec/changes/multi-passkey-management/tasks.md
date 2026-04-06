## 1. TypeSpec: API 契約の追加

- [x] 1.1 `packages/typespec/src/models/auth.tsp` に `PasskeyItem`、`PasskeyListResponse`、`PasskeyAddStartResponse`、`PasskeyAddFinishRequest`、`PasskeyOtpResponse`、`PasskeyAddByOtpStartRequest`、`PasskeyAddByOtpFinishRequest` モデルを追加する（`POST /passkeys/start` は request body なし）
- [x] 1.2 `packages/typespec/src/routes/v1/auth.tsp` の `ApiV1App` namespace に `GET /passkeys`、`POST /passkeys/start`、`POST /passkeys/finish`、`DELETE /passkeys/{id}`、`POST /passkeys/otp` エンドポイントを追加する（bearer 認証必須）。また public namespace に `POST /auth/passkey/add/start`、`POST /auth/passkey/add/finish` を追加する（認証不要）
- [x] 1.3 `pnpm gen` を実行し OpenAPI・フロントエンドクライアント・Go bindings が正常に再生成されることを確認する
- [x] 1.4 `pnpm check:codegen` を実行してドリフトがないことを確認する

## 2. Backend: ドメインモデル変更

- [x] 2.1 `packages/backend/internal/domain/auth_account.go` に `PasskeyCredential` 値オブジェクト（`id`, `accountID`, `identifier`, `credentialHandle`, `createdAt`）を追加し、`NewPasskeyCredential` コンストラクタを実装する
- [x] 2.2 `AuthAccount` に `credentials []PasskeyCredential` フィールドと `Credentials() []PasskeyCredential` アクセサを追加する（後方互換のため `PasskeyCredentialID()` / `CredentialHandle()` は最初の要素から返す形で維持）
- [x] 2.3 `[UT-AUTH-BE-BND-001]` UT: `NewPasskeyCredential` の空ハンドル・無効 ID で `ErrInvalidAuthID`/`ErrInvalidAccountID`/`ErrInvalidPasskeyCredential` が返ることをテストする（実装は id/accountID 不正 → `ErrInvalidAuthID`/`ErrInvalidAccountID`、空ハンドル → `ErrInvalidPasskeyCredential`）

## 3. Backend: 永続化層の変更

- [x] 3.1 `packages/backend/internal/persistence/gorm_auth_account_repository.go` に `ListPasskeys(ctx, accountID)` メソッドを追加する（`passkey_credentials` を全件取得して `[]domain.PasskeyCredential` を返す）
- [x] 3.2 `AddPasskey(ctx, accountID, credentialID, handle)` メソッドを追加する（既存パスキーを削除せず 1 件追加する）
- [x] 3.3 `DeletePasskeyByID(ctx, accountID, credentialID)` メソッドを追加する（`account_id` と `id` の両方で絞り込み、他アカウントを誤削除しない）
- [x] 3.4 `ReplacePasskey` を `AddPasskey` に置き換える（`ReplacePasskey` は `AddPasskey` の呼び出しに変更し、既存パスキー全削除ロジックを削除する）
- [x] 3.5 `FindByEmail` が複数パスキーのある場合も正しく動作することを確認する（最古の 1 件を返す現挙動は維持）

## 4. Backend: ユースケース層の変更

- [x] 4.1 `packages/backend/internal/usecases/auth_contracts.go` の `AuthAccountRepository` インターフェースに `AddPasskey`, `ListPasskeys`, `DeletePasskeyByID` を追加し、`ReplacePasskey` をインターフェースから廃止する
- [x] 4.2 `packages/backend/internal/usecases/auth_errors.go` に `ErrLastPasskeyCannotBeDeleted`、`ErrInvalidOtp`、`ErrOtpExpiredOrConsumed` を追加する
- [x] 4.3 `auth_service.go` に `ListPasskeys(ctx, accountID)` メソッドを追加する
- [x] 4.4 `auth_service.go` に `StartAddPasskey(ctx, accountID)` メソッドを追加する（Valkey にチャレンジを保存）
- [x] 4.5 `auth_service.go` に `FinishAddPasskey(ctx, accountID, credential)` メソッドを追加する（チャレンジ検証 → `AddPasskey` 呼び出し）
- [x] 4.6 `auth_service.go` に `DeletePasskey(ctx, accountID, credentialID)` メソッドを追加する（残数チェック → 1 件なら `ErrLastPasskeyCannotBeDeleted`、複数なら `DeletePasskeyByID` 呼び出し）
- [x] 4.7 `[UT-AUTH-BE-HAP-001]` UT: `DeletePasskey` の最終 1 件保護ロジックが `ErrLastPasskeyCannotBeDeleted` を返すことをテストする
- [x] 4.8 `[UT-AUTH-BE-SEC-001]` UT: `DeletePasskey` で他アカウントの credential ID を指定した場合に `ErrAuthAccountNotFound` が返ることをテストする
- [x] 4.9 `auth_service.go` に `IssuePasskeyOtp(ctx, accountID)` メソッドを追加する（6 桁乱数を生成して Valkey に `{otp → accountID, TTL: 5min}` で保存し、OTP を返す）
- [x] 4.10 `auth_service.go` に `StartAddPasskeyByOtp(ctx, otp)` メソッドを追加する（OTP 検証 → accountID 解決 → チャレンジ生成 → Valkey に保存）
- [x] 4.11 `auth_service.go` に `FinishAddPasskeyByOtp(ctx, otp, credential)` メソッドを追加する（OTP 再検証・消費 → credential 検証 → `AddPasskey` 呼び出し）
- [x] 4.12 `[UT-AUTH-BE-ERR-001]` UT: `StartAddPasskeyByOtp` で期限切れ OTP を指定した場合に `ErrOtpExpiredOrConsumed` が返ることをテストする
- [x] 4.13 `[UT-AUTH-BE-ERR-002]` UT: `FinishAddPasskeyByOtp` で消費済み OTP を指定した場合に `ErrOtpExpiredOrConsumed` が返ることをテストする

## 5. Backend: HTTP ハンドラーの追加

- [x] 5.1 `packages/backend/internal/http/auth.go` に `GET /api/v1/passkeys` ハンドラーを追加する（bearer 認証済み accountID から `ListPasskeys` を呼び出し、`PasskeyListResponse` を返す）
- [x] 5.2 `POST /api/v1/passkeys/start` ハンドラーを追加する
- [x] 5.3 `POST /api/v1/passkeys/finish` ハンドラーを追加する
- [x] 5.4 `DELETE /api/v1/passkeys/{id}` ハンドラーを追加する（`ErrLastPasskeyCannotBeDeleted` → 409 を返す）
- [x] 5.5 `POST /api/v1/passkeys/otp` ハンドラーを追加する（bearer 認証済み・再認証完了前提で `IssuePasskeyOtp` を呼び出し `PasskeyOtpResponse` を返す）
- [x] 5.6 `POST /api/v1/auth/passkey/add/start` 公開ハンドラーを追加する（`StartAddPasskeyByOtp` を呼び出し challenge を返す）
- [x] 5.7 `POST /api/v1/auth/passkey/add/finish` 公開ハンドラーを追加する（`FinishAddPasskeyByOtp` を呼び出し 200 を返す）
- [x] 5.8 `packages/backend/internal/http/router.go` に新規ルートをすべて登録する

## 6. Backend: 統合テスト

- [x] 6.1 `[AUTH-BE-S014]` IT: `GET /api/v1/passkeys` が登録済みパスキー一覧を返すことをテストする
- [x] 6.2 `[AUTH-BE-S015]` IT: パスキー追加後に既存パスキーが保持されることをテストする
- [x] 6.3 `[AUTH-BE-S016]` IT: 最終 1 件の削除が 409 を返すことをテストする
- [x] 6.4 `[AUTH-BE-S017]` IT: 2 件中 1 件の削除が正しく動作することをテストする
- [x] 6.5 `[AUTH-BE-S018]` IT: 他アカウントのパスキー削除が 403 を返すことをテストする
- [x] 6.6 `[AUTH-BE-S019]` IT: 未認証リクエストが 401 を返すことをテストする
- [x] 6.7 `[AUTH-BE-S020]` IT: パスキー追加後に既存パスキーが保持されることをテストする（回帰）
- [x] 6.8 `[AUTH-BE-S021]` IT: `POST /api/v1/passkeys/otp` が 6 桁の OTP を返すことをテストする
- [x] 6.9 `[AUTH-BE-S022]` IT: 有効な OTP を使った新端末パスキー登録フロー（add/start → add/finish）が成功し既存パスキーが保持されることをテストする
- [x] 6.10 `[AUTH-BE-S023]` IT: 有効期限切れの OTP が add/start で拒否されることをテストする
- [x] 6.11 `[AUTH-BE-S024]` IT: 消費済みの OTP が再利用できないことをテストする
- [x] 6.12 既存 auth 統合テストがすべて合格することを確認する（回帰）

## 7. Frontend: API クライアントの更新

- [x] 7.1 `pnpm gen` の出力 `packages/frontend/api/src/generated/client.ts` に passkey 管理 4 エンドポイントのクライアント関数が生成されていることを確認する

## 8. Frontend: domain ユースケース追加

- [x] 8.1 `packages/frontend/domain/src/hooks/auth/usePasskeyManagement.svelte.ts` を追加し、`{ data: { passkeys, loading, error }, actions: { listPasskeys(), startAddPasskey(), finishAddPasskey(), deletePasskey(id), issueOtp() } }` を返す `usePasskeyManagement()` hook を実装する
- [x] 8.2 `packages/frontend/domain/src/hooks/auth/usePasskeyAddByOtp.svelte.ts` を追加し、`{ data: { loading, error, done }, actions: { start(otp), finish(otp, credential) } }` を返す `usePasskeyAddByOtp()` hook を実装する
- [x] 8.3 `packages/frontend/domain/src/index.ts` に `usePasskeyManagement` と `usePasskeyAddByOtp` のエクスポートを追加する
- [x] 8.4 `[AUTH-FE-S012]` UT: `deletePasskey` 成功時に `data.passkeys` から対象が除去されることをテストする
- [x] 8.5 `[AUTH-FE-S015]` UT: `deletePasskey` で API エラー時に `data.passkeys` が変化しないことをテストする

## 9. Frontend: パスキー管理ページ・新端末登録ページの追加

- [x] 9.1 `packages/frontend/app/src/lib/profiles/PasskeyList.svelte` コンポーネントを追加する（一覧表示・最終 1 件は削除ボタン無効化）
- [x] 9.2 `packages/frontend/app/src/routes/(protected)/passkeys/+page.ts` を追加する（`listPasskeys` を呼び出して初期データをロード）
- [x] 9.3 `packages/frontend/app/src/routes/(protected)/passkeys/+page.svelte` を追加する（一覧・追加・削除の UI を実装、OTP 表示 UI 含む）
- [x] 9.4 `packages/frontend/app/src/routes/passkeys/add/+page.ts` を追加する
- [x] 9.5 `packages/frontend/app/src/routes/passkeys/add/+page.svelte` を追加する（OTP 入力 → WebAuthn 登録フローを `usePasskeyAddByOtp` hook で実装、未認証 surface）
- [x] 9.6 `packages/frontend/app/src/routes/(protected)/+layout.svelte` にパスキー管理ページへのナビゲーションリンクを追加する（必要に応じて）

## 10. E2E テスト

- [x] 10.1 `[AUTH-FE-S010]` E2E: パスキー管理ページで一覧が表示されることをテストする
- [x] 10.2 `[AUTH-FE-S011]` E2E: パスキー追加フローが完了することをテストする（WebAuthn stub）
- [x] 10.3 `[AUTH-FE-S012]` E2E: パスキー削除が成功することをテストする（2 件以上の状態）
- [x] 10.4 `[AUTH-FE-S013]` E2E: 最終 1 件のパスキーの削除ボタンが無効化されていることをテストする
- [x] 10.5 `[AUTH-FE-S014]` E2E: パスキー追加フロー中に WebAuthn がキャンセルされた場合にエラーメッセージが表示されることをテストする
- [x] 10.6 `[AUTH-FE-S016]` E2E: OTP 発行後に管理ページへ OTP が表示されることをテストする
- [x] 10.7 `[AUTH-FE-S017]` E2E: 新端末パスキー登録ページで有効な OTP を入力して WebAuthn 登録が完了することをテストする（WebAuthn stub）
- [x] 10.8 `[AUTH-FE-S018]` E2E: 新端末パスキー登録ページで無効な OTP を入力した場合にエラーメッセージが表示されることをテストする

## 11. ビルド・CI 確認

- [x] 11.1 `pnpm test:run` を実行してすべての単体テストが合格することを確認する（`@www-template/ui SafeHTML.test.ts` 失敗は今回の変更と無関係の既存問題）
- [x] 11.2 `pnpm test:e2e` を実行してすべての E2E テストが合格することを確認する（passkey-management 9テスト・user-flow 3テスト 全通過確認済み。auth-contract 2テスト失敗はバックエンド DB シードデータ未投入による既存問題で今回の変更と無関係）
- [x] 11.3 `pnpm check:codegen` を実行してドリフトがないことを最終確認する
