## 1. API 契約とコード生成

- [ ] 1.1 `packages/typespec/src/models/auth.tsp` に `credentialMode`、Web Cookie session response、Bearer session response、CSRF token、mode 別 refresh DTO を追加・更新する。完了条件: TypeSpec model が暗黙的に body token を返す挙動に依存せず、`web-cookie` と `bearer` の両 flow を表現できる。
- [ ] 1.2 `packages/typespec/src/routes/v1/auth.tsp` を更新し、passkey finish、recovery register、refresh、logout、protected mutation の契約で新しい request/response shape と必要箇所の `X-CSRF-Token` を公開する。完了条件: 生成される route signature が Cookie mode と Bearer mode を分離して表現できる。
- [ ] 1.3 `pnpm gen` を実行する。完了条件: `packages/typespec/openapi/openapi.json`、`packages/backend/internal/generated/openapi/openapi.gen.go`、`packages/frontend/api/src/generated/client.ts` が TypeSpec 変更を反映している。
- [ ] 1.4 `pnpm check:codegen` を実行する。完了条件: 生成 artifact に drift がない。

## 2. Backend Session と Credential Model

- [ ] 2.1 `packages/backend/internal/application/auth_contracts.go` を更新し、mode 別 session result、CSRF token output、Bearer refresh token output、session metadata の CSRF hash を表現する。完了条件: application DTO がすべての Product session response に `AccessToken` を強制しない。
- [ ] 2.2 `packages/backend/internal/application/token_service.go` を更新し、Web Cookie mode では access credential、refresh credential、CSRF token を発行し、Bearer mode では response body の accessToken / refreshToken を発行する。完了条件: issue/refresh 呼び出し元がどちらの mode も明示的に要求できる。
- [ ] 2.3 `packages/backend/internal/application/auth_service.go` を更新し、passkey finish、recovery register、refresh、logout が `credentialMode` と session credential data を正しく伝搬する。完了条件: handler が response body field から Web/Bearer を推測する必要がない。
- [ ] 2.4 `packages/backend/internal/adapter/valkey/session_store.go` を更新し、session metadata の CSRF hash を保存・読み込みする。完了条件: CSRF hash を持たない session は Cookie mutation で fail-close する。
- [ ] 2.5 `[AUTH-BE-S044]`、`[AUTH-BE-S045]`、`[AUTH-BE-S046]`、`[AUTH-BE-S060]`、`[AUTH-BE-S062]`、`[AUTH-BE-S063]` の backend application test を追加または更新する。完了条件: TokenService の issue/refresh/expiry/theft test が mode 別 credential を網羅する。

## 3. Backend HTTP 境界

- [ ] 3.1 `packages/backend/internal/adapter/http/product/auth.go` を更新し、Cookie または Bearer から Product credential source を exactly one として抽出し、曖昧な credential を拒否する。完了条件: middleware が raw token を handler に渡さず、認可済み session context を束縛する。
- [ ] 3.2 unsafe な Cookie request に Product Origin validation を追加する。完了条件: 許可されていない Origin または欠落した Origin が protected state mutation 前に拒否される。
- [ ] 3.3 Cookie state-changing request に Product CSRF validation を追加する。完了条件: handler 実行前に `X-CSRF-Token` が session-bound metadata と比較される。
- [ ] 3.4 `packages/backend/internal/adapter/http/product/router.go` の `access_token` と `refresh_token` 用 Cookie helper を、Set-Cookie と clear behavior を含めて更新する。完了条件: Web Cookie login/refresh/logout が正しい HttpOnly Cookie を設定・削除する。
- [ ] 3.5 `router.go` の Product strict handler を更新し、passkey finish、register、refresh、logout、passkey management、account settings、session management で mode 別の generated DTO を使う。完了条件: handler が Web Cookie request のために `Authorization` を直接読まない。
- [ ] 3.6 recovery/device-link scenario `[AUTH-BE-S004]`、`[AUTH-BE-S005]`、`[AUTH-BE-S006]`、`[AUTH-BE-S030]`、`[AUTH-BE-S047]`、`[AUTH-BE-S073]` の backend endpoint test を追加または更新する。完了条件: token issuance/consume/device-link Cookie session の coverage がある。
- [ ] 3.7 recovery register scenario `[AUTH-BE-S007]`、`[AUTH-BE-S008]`、`[AUTH-BE-S048]` の backend endpoint test を追加または更新する。完了条件: register が mode 別 session を返し、recovery/device-link post-processing を維持する。
- [ ] 3.8 passkey management scenario `[AUTH-BE-S014]`、`[AUTH-BE-S015]`、`[AUTH-BE-S016]`、`[AUTH-BE-S017]`、`[AUTH-BE-S018]`、`[AUTH-BE-S019]`、`[AUTH-BE-S074]` の backend endpoint test を追加または更新する。完了条件: Cookie/Bearer session source と ambiguity rejection が網羅される。
- [ ] 3.9 WebAuthn reauthentication scenario `[AUTH-BE-S028]`、`[AUTH-BE-S029]`、`[AUTH-BE-S036]`、`[AUTH-BE-S037]` の backend endpoint test を追加または更新する。完了条件: high-risk operation が、受け入れ可能な session credential source のどちらでも reauthentication を要求する。
- [ ] 3.10 device-link scenario `[AUTH-BE-S049]` と `[AUTH-BE-S050]` の backend endpoint test を追加または更新する。完了条件: device-link delivery が有効なアプリケーションセッションと reauth で成功し、reauth なしで失敗する。
- [ ] 3.11 session issuance/authorization scenario `[AUTH-BE-S001]`、`[AUTH-BE-S002]`、`[AUTH-BE-S003]`、`[AUTH-BE-S009]`、`[AUTH-BE-S010]`、`[AUTH-BE-S054]`、`[AUTH-BE-S055]`、`[AUTH-BE-S058]`、`[AUTH-BE-S060]`、`[AUTH-BE-S063]`、`[AUTH-BE-S075]`、`[AUTH-BE-S076]`、`[AUTH-BE-S077]` の backend endpoint test を追加または更新する。完了条件: Bearer mode、Cookie mode、logout、missing/expired/suspended、CSRF、ambiguity behavior が網羅される。

## 4. Frontend Auth State と API Calls

- [ ] 4.1 `packages/frontend/domain/src/auth/types.ts` を更新し、Web auth state が `accessToken` なしで session metadata と CSRF token を保存するようにする。完了条件: TypeScript type 上、browser-readable な Product accessToken を Web hook から利用できない。
- [ ] 4.2 `packages/frontend/domain/src/auth/session/state.ts` を更新し、Authorization header generation を same-origin credential request helper と CSRF header helper に置き換える。完了条件: Product Web domain code に `Authorization: Bearer` 作成 path がない。
- [ ] 4.3 `packages/frontend/domain/src/auth/session/hook.svelte.ts` を更新し、bootstrap refresh、session-expired refresh-once retry、logout、session clearing、account-suspended routing、AccountSetting snapshot handling を扱う。完了条件: auth state が Cookie response と CSRF token rotation によって駆動される。
- [ ] 4.4 `packages/frontend/domain/src/auth/passkey/login/hook.svelte.ts` を更新し、`credentialMode="web-cookie"` を送信して Web Cookie session response を受け入れる。完了条件: login が accessToken を decode または保存しない。
- [ ] 4.5 `packages/frontend/domain/src/auth/recovery/hook.svelte.ts` を更新し、register で `credentialMode="web-cookie"` を送信して CSRF/session metadata を受け入れる。完了条件: recovery/device-link registration が token body なしで authenticated state に入る。
- [ ] 4.6 `packages/frontend/domain/src/auth/passkey/management/hook.svelte.ts` と `packages/frontend/domain/src/auth/session/session_api.ts` を更新し、mutation で same-origin credential と CSRF header を送信する。完了条件: passkey/device/session management が caller から Authorization header を受け取らない。
- [ ] 4.7 `packages/frontend/domain/src/account/hook.svelte.ts` と `packages/frontend/domain/src/account/localeSync.svelte.ts` を更新し、Cookie + CSRF request helper を使う。完了条件: AccountSetting load/update が bearer header なしで動作する。
- [ ] 4.8 `packages/frontend/app/src/tests/mocks/handlers.ts` と関連 app mock を更新し、Web Cookie mode response body を返す。完了条件: frontend test が Product Web 用の mock `accessToken` body に依存しない。

## 5. Frontend Tests

- [ ] 5.1 login scenario `[AUTH-FE-S001]`、`[AUTH-FE-S002]`、`[AUTH-FE-S051]` の frontend test を追加または更新する。完了条件: passkey login が accessToken を保存せず、CSRF/session metadata から authenticated state に入る。
- [ ] 5.2 recovery/device-link scenario `[AUTH-FE-S003]`、`[AUTH-FE-S004]`、`[AUTH-FE-S005]`、`[AUTH-FE-S038]`、`[AUTH-FE-S052]` の frontend test を追加または更新する。完了条件: recovery registration が Web Cookie session response を受け入れる。
- [ ] 5.3 refresh/session continuation scenario `[AUTH-FE-S045]`、`[AUTH-FE-S024]`、`[AUTH-FE-S046]`、`[AUTH-FE-S026]`、`[AUTH-FE-S047]` の frontend test を追加または更新する。完了条件: bootstrap refresh、retry-on-session-expired、persistent token storage 不使用が網羅される。
- [ ] 5.4 single active Cookie session scenario `[AUTH-FE-S048]`、`[AUTH-FE-S028]`、`[AUTH-FE-S049]`、`[AUTH-FE-S050]`、`[AUTH-FE-S031]` の frontend test を追加または更新する。完了条件: account switching 用 token-list behavior が Product Web から削除される。
- [ ] 5.5 expiry/logout routing scenario `[AUTH-FE-S006]`、`[AUTH-FE-S007]`、`[AUTH-FE-S008]` の frontend test を追加または更新する。完了条件: missing session、expired session、logout route intent が区別される。
- [ ] 5.6 passkey management scenario `[AUTH-FE-S010]`、`[AUTH-FE-S011]`、`[AUTH-FE-S012]`、`[AUTH-FE-S013]`、`[AUTH-FE-S014]`、`[AUTH-FE-S015]`、`[AUTH-FE-S035]`、`[AUTH-FE-S037]` の frontend test を追加または更新する。完了条件: management API call が期待どおり credential と CSRF を含む。
- [ ] 5.7 security presentation scenario `[AUTH-FE-S019]`、`[AUTH-FE-S020]`、`[AUTH-FE-S054]` の frontend test を追加または更新する。完了条件: token/Cookie value が state/storage に存在せず、auth route が no-store/security behavior を維持する。
- [ ] 5.8 suspended account scenario `[AUTH-FE-S041]`、`[AUTH-FE-S042]`、`[AUTH-FE-S043]`、`[AUTH-FE-S044]` の frontend test を追加または更新する。完了条件: suspended handling が Cookie session state を消去し、public enumeration を避ける。
- [ ] 5.9 device management scenario `[AUTH-FE-S034]`、`[AUTH-FE-S053]`、`[AUTH-FE-S036]` の frontend test を追加または更新する。完了条件: session list/revoke/revoke-others が Cookie + CSRF request を使う。

## 6. Verification と Artifact Maintenance

- [ ] 6.1 `pnpm lint` を実行する。完了条件: repository script 経由の lint が通る。
- [ ] 6.2 `pnpm check` を実行する。完了条件: TypeScript/Svelte/Go の type check が repository script 経由で通る。
- [ ] 6.3 `pnpm test:server` を実行する。完了条件: `[AUTH-BE-*]` scenario を網羅する backend test が通る。
- [ ] 6.4 `pnpm test:client` を実行する。完了条件: `[AUTH-FE-*]` scenario を網羅する frontend test が通る。
- [ ] 6.5 すべての implementation edit 後に `pnpm check:codegen` を実行する。完了条件: TypeSpec generated artifact が同期している。
- [ ] 6.6 実装中に contract mismatch が見つかった場合、OpenSpec delta specs/design/tasks を更新する。完了条件: archive または sync 前に artifact と code が一致している。
