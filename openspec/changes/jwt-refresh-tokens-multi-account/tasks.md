## 1. Backend Domain & Configuration

- [ ] 1.1 `packages/backend/internal/auth/domain/jwt.go` を作成する。JWT claims 構造体 (`Claims`)、HS256 署名関数 (`SignAccessToken`)、検証関数 (`VerifyAccessToken`)、およびエラー型 (`ErrTokenExpired`, `ErrInvalidSignature`) を実装する。
- [ ] 1.2 `packages/backend/internal/app/config.go` に `AuthConfig.RefreshTokenTTL` (`time.Duration`, ポインタまたは omitempty) を追加し、24 時間未満の場合に `fail-close` で起動エラーを返すバリデーションを実装する。
- [ ] 1.3 `packages/backend/config.example.toml` に `auth.refresh_token_ttl` の設定例を追加する。
- [ ] 1.4 UT: `[AUTH-BE-S046] Expired JWT verification fails` — `jwt.go` の期限切れ検証をテストする。
- [ ] 1.5 UT: `[AUTH-BE-S040] Short TTL blocks startup` — 設定バリデーションのテストを追加する。
- [ ] 1.6 IT: `[AUTH-BE-S038] Unset refresh token TTL is unlimited` — 未設定時の無期限リフレッシュトークンをエンドポイントテストする。
- [ ] 1.7 IT: `[AUTH-BE-S039] 24h+ TTL is applied correctly` — 24 時間以上 TTL の適用をエンドポイントテストする。

## 2. Backend Application & Persistence

- [ ] 2.1 `packages/backend/internal/adapters/persistence/valkey/refresh_token_store.go` を作成する。`Save` (SET EX/NX)、`Consume` (GETDEL)、`RevokeAllForFingerprint` (SMEMBERS → DEL) を実装し、キースキーマ (`auth:refresh:{hash}`, `auth:refresh_index:{accountID}:{fingerprint}`) を定義する。
- [ ] 2.2 `packages/backend/internal/auth/application/token_service.go` を作成する。`Issue` (JWT + refresh token 発行)、`Refresh` (GETDEL 消費 → 新ペア生成)、`Revoke` (セッション失効) を実装する。盗難検出時は同一 fingerprint の全リフレッシュトークンを失効させる。
- [ ] 2.3 IT: `[AUTH-BE-S043] Refresh endpoint returns new pair` — `POST /api/v1/auth/refresh` の正常系をエンドポイントテストする。
- [ ] 2.4 IT: `[AUTH-BE-S044] Rotation failure revokes family` — 消費済みトークン再利用による盗難検出と失効をテストする。
- [ ] 2.5 IT: `[AUTH-BE-S045] Invalid refresh token rejected` — 不正トークンでのリフレッシュ拒否をテストする。
- [ ] 2.6 UT: `[AUTH-BE-S043] TokenService rotates refresh token` — `TokenService.Refresh` のローテーションロジックをユニットテストする。
- [ ] 2.7 UT: `[AUTH-BE-S044] TokenService detects theft` — 盗難検出ロジックをユニットテストする。
- [ ] 2.8 `packages/backend/internal/adapters/persistence/valkey/session_store.go` を作成する。`SaveSession` (SET + SADD)、`ListSessions` (SMEMBERS → MGET)、`RevokeSession` (DEL + SREM)、`RevokeOthers` (SMEMBERS → 現在 sessionID 以外を DEL + SREM) を実装し、キースキーマ (`auth:session:{sessionID}`, `auth:account-sessions:{accountID}`) を定義する。
- [ ] 2.9 `packages/backend/internal/auth/application/session_service.go` を作成する。`List` (accountID のセッション一覧)、`Revoke` (特定セッション無効化)、`RevokeOthers` (現在以外の全セッション無効化) を実装する。所有権検証（他アカウント操作拒否）を含める。
- [ ] 2.10 IT: `[AUTH-BE-S047] Session list returns owned sessions only` — `GET /api/v1/sessions` が自身のセッションのみ返すことをエンドポイントテストする。
- [ ] 2.11 IT: `[AUTH-BE-S048] Revoke session removes metadata and tokens` — `DELETE /api/v1/sessions/{id}` でメタデータとリフレッシュトークンが削除されることをテストする。
- [ ] 2.12 IT: `[AUTH-BE-S049] Revoke others leaves current session` — `DELETE /api/v1/sessions/others` で現在セッションのみ維持されることをテストする。
- [ ] 2.13 IT: `[AUTH-BE-S048] Revoking another account's session is forbidden` — 他アカウントのセッション無効化が `403` になることをテストする。
- [ ] 2.14 UT: `[AUTH-BE-S047] SessionStore lists sessions with metadata` — `SessionStore.ListSessions` の一覧取得をユニットテストする。
- [ ] 2.15 UT: `[AUTH-BE-S048] SessionStore revokes specific session` — `SessionStore.RevokeSession` の削除動作をユニットテストする。
- [ ] 2.16 UT: `[AUTH-BE-S049] SessionStore revokes others` — `SessionStore.RevokeOthers` の一括削除動作をユニットテストする。

## 3. Backend HTTP Layer

- [ ] 3.1 `packages/backend/internal/adapters/http/middleware/auth_middleware.go` を更新する。Bearer ヘッダーから JWT を抽出し、`VerifyAccessToken` で署名と有効期限を検証する。失効セッションのチェックも維持する。
- [ ] 3.2 `packages/backend/internal/adapters/http/auth_handler.go` を更新する。`POST /api/v1/auth/refresh` ハンドラーを追加し、login / recovery register / passkey finish のレスポンスに `accessToken` と `refreshToken` を含めるよう変更する。また、`GET /api/v1/sessions`、`DELETE /api/v1/sessions/{id}`、`DELETE /api/v1/sessions/others` のハンドラーを追加する。
- [ ] 3.3 IT: `[AUTH-BE-S001] Passkey finish returns JWT and refresh token` — ログイン完了レスポンスに両トークンが含まれることを検証する。
- [ ] 3.4 IT: `[AUTH-BE-S002] Missing or inactive session is rejected` — JWT 検証ミドルウェアの拒否動作をテストする。
- [ ] 3.5 IT: `[AUTH-BE-S003] Logout revokes active session` — ログアウト後のセッション失効をテストする。
- [ ] 3.6 IT: `[AUTH-BE-S009] Request without session is unauthenticated` — トークン未提示時の分類をテストする。
- [ ] 3.7 IT: `[AUTH-BE-S010] Auth state store unavailable is internal-error` — Valkey Unavailable 時の fail-close をテストする。
- [ ] 3.8 IT: `[AUTH-BE-S042] Logout revokes only one session` — マルチセッション環境での単一セッションログアウトをテストする。
- [ ] 3.9 IT: `[AUTH-BE-S041] Multiple accounts hold independent sessions` — 複数アカウントの独立セッション保持をテストする。
- [ ] 3.10 IT: `[AUTH-BE-S046] Expired JWT rejected` — 期限切れ JWT での保護エンドポイントアクセス拒否をテストする。
- [ ] 3.11 IT: `[AUTH-BE-S047] Session list endpoint returns sessions` — `GET /api/v1/sessions` の正常系をテストする。
- [ ] 3.12 IT: `[AUTH-BE-S048] Revoke session endpoint invalidates session` — `DELETE /api/v1/sessions/{id}` の正常系をテストする。
- [ ] 3.13 IT: `[AUTH-BE-S049] Revoke others endpoint invalidates other sessions` — `DELETE /api/v1/sessions/others` の正常系をテストする。

## 4. API Contract (TypeSpec)

- [ ] 4.1 `packages/typespec/main.tsp` に `POST /api/v1/auth/refresh` エンドポイントを追加する。Request body に `refreshToken`、response に `accessToken` と `refreshToken` を含める。
- [ ] 4.2 `packages/typespec/main.tsp` の Bearer スキーマを JWT 形式であることを注釈する。
- [ ] 4.3 `packages/typespec/main.tsp` に `GET /api/v1/sessions` エンドポイントを追加する。Response にセッション一覧（`sessionId`, `deviceName`, `loginAt`, `lastActiveAt`, `ipHash`, `isCurrentSession`）を含める。
- [ ] 4.4 `packages/typespec/main.tsp` に `DELETE /api/v1/sessions/{id}` エンドポイントを追加する。Path parameter に `id` を含める。Response は `204 No Content` とする。
- [ ] 4.5 `packages/typespec/main.tsp` に `DELETE /api/v1/sessions/others` エンドポイントを追加する。Response は `204 No Content` とする。
- [ ] 4.6 `pnpm gen` を実行し、OpenAPI、フロントエンド SDK、Go bindings を再生成する。
- [ ] 4.7 `pnpm check:codegen` で codegen drift がないことを確認する。

## 5. Frontend Domain

- [ ] 5.1 `packages/frontend/domain/src/auth/session/token_state.ts` を作成する。JWT ペイロードのデコード関数、`exp` 監視、1 分未満判定、メモリ上の pure トークン保持を実装する。
- [ ] 5.2 `packages/frontend/domain/src/auth/session/state.ts` を更新する。`sessions`（配列）、`activeSession`、ログイン追加、切り替え、アクティブセッションのみの除去を管理する。セッションが空になった場合は未認証イベントを発火する。
- [ ] 5.3 UT: `[AUTH-FE-S023] TokenManager schedules refresh before expiry` — 期限前リフレッシュ判定をテストする。
- [ ] 5.4 UT: `[AUTH-FE-S024] Expired access token triggers refresh` — 期限切れ時のリフレッシュ呼び出しをテストする。
- [ ] 5.5 UT: `[AUTH-FE-S025] Tokens are not persisted to storage` — メモリ保持のみであることをテストする。
- [ ] 5.6 UT: `[AUTH-FE-S026] Browser revisit normalizes to unauthenticated` — 再起動時の未認証正規化をテストする。
- [ ] 5.7 UT: `[AUTH-FE-S027] SessionStore adds new session on login` — ログイン時のセッション追加をテストする。
- [ ] 5.8 UT: `[AUTH-FE-S028] SessionStore switches active session` — アクティブセッション切り替えをテストする。
- [ ] 5.9 UT: `[AUTH-FE-S030] SessionStore logout clears only active` — 部分ログアウトをテストする。
- [ ] 5.10 UT: `[AUTH-FE-S031] SessionStore redirects when empty` — 全セッション消失時の遷移をテストする。
- [ ] 5.11 UT: `[AUTH-FE-S006] Session expiry routes to session-expired` — セッション失効時のリダイレクト動作をテストする。
- [ ] 5.12 UT: `[AUTH-FE-S007] Logout returns to unauthenticated route` — ログアウト後の未認証導線復帰をテストする。
- [ ] 5.13 UT: `[AUTH-FE-S008] Missing session stays on normal login flow` — トークン不在時に通常ログイン導線に留まることをテストする。
- [ ] 5.14 `packages/frontend/domain/src/auth/session/session_api.ts` を作成する。`GET /api/v1/sessions` のラッパー、`DELETE /api/v1/sessions/{id}` のラッパー、`DELETE /api/v1/sessions/others` のラッパーを実装する。エラー時は汎用メッセージを返す。
- [ ] 5.15 `packages/frontend/domain/src/auth/session/hook.svelte.ts` を更新する。`listDevices()`、`revokeDevice(sessionId)`、`revokeOtherDevices()` を `useAuthSession()` 戻り値に追加する。
- [ ] 5.16 UT: `[AUTH-FE-S034] DeviceManager renders session list` — デバイス一覧コンポーネントのレンダリングをテストする。
- [ ] 5.17 UT: `[AUTH-FE-S035] DeviceManager triggers revoke on click` — 個別ログアウトクリックの動作をテストする。
- [ ] 5.18 UT: `[AUTH-FE-S036] DeviceManager triggers revoke-others on click` — 一括ログアウトクリックの動作をテストする。

## 6. Frontend App & UI

- [ ] 6.1 `packages/frontend/app/src/lib/components/AccountSwitcher.svelte` を作成する。複数セッション存在時にアカウント一覧を表示し、クリックで `switchSession` を呼び出すコンポーネントを実装する。
- [ ] 6.2 認証済みレイアウトまたはヘッダーに `AccountSwitcher` を組み込む。
- [ ] 6.3 ログアウトボタンが `logoutActive()` を呼び出し、対象セッションのみを除去するよう更新する。
- [ ] 6.4 E2E: `[AUTH-FE-S029] Account switcher UI is visible` — 複数セッション時の UI 表示を Playwright で検証する。
- [ ] 6.5 E2E: `[AUTH-FE-S028] Account switching changes active token` — 切り替え後の API ヘッダー変更を Playwright で検証する。
- [ ] 6.6 E2E: `[AUTH-FE-S030] Logout affects only active session` — 部分ログアウトの E2E を Playwright で検証する。
- [ ] 6.7 E2E: `[AUTH-FE-S032] Proactive refresh continues session` — 期限切れ前の自動リフレッシュ継続を Playwright で検証する。
- [ ] 6.8 E2E: `[AUTH-FE-S033] Refresh failure redirects to session-expired` — リフレッシュ失敗時の遷移を Playwright で検証する。
- [ ] 6.9 `packages/frontend/app/src/lib/components/DeviceManager.svelte` を作成する。デバイス一覧を表示し、各デバイスにログアウトボタンを配置、「他のすべてのデバイスをログアウト」ボタンを配置する。現在のデバイスにはインジケーターを表示する。
- [ ] 6.10 `packages/frontend/app/src/routes/sessions/+page.svelte` を作成する。認証済みレイアウト内で `DeviceManager` をレンダリングし、ドメインフック経由でデータ取得・操作を行う。
- [ ] 6.11 認証済みレイアウトまたは設定メニューにデバイス管理ページへのリンクを追加する。
- [ ] 6.12 E2E: `[AUTH-FE-S034] Device manager page shows sessions` — デバイス管理ページの表示を Playwright で検証する。
- [ ] 6.13 E2E: `[AUTH-FE-S035] Device manager revokes specific device` — 特定デバイスログアウトの E2E を Playwright で検証する。
- [ ] 6.14 E2E: `[AUTH-FE-S036] Device manager revokes all other devices` — 他デバイス一括ログアウトの E2E を Playwright で検証する。

## 7. Validation & Documentation

- [ ] 7.1 `pnpm lint` を実行し、エラーがないことを確認する。
- [ ] 7.2 `pnpm test:run` を実行し、すべてのテストが通過することを確認する。
- [ ] 7.3 `pnpm check:codegen` を実行し、コード生成の整合性を確認する。
- [ ] 7.4 `openspec validate --type change jwt-refresh-tokens-multi-account --strict --no-interactive` を実行し、OpenSpec アーティファクトが有効であることを確認する。
- [ ] 7.5 必要に応じて `openspec/changes/jwt-refresh-tokens-multi-account/` 以下の design.md または spec.md に実装中の意思決定を反映する。
