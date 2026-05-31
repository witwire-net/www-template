## 1. TypeSpec 契約と生成 artifact

- [ ] 1.1 `packages/typespec/main.tsp`、`packages/typespec/src/models/auth.tsp`、`packages/typespec/src/models/admin.tsp` に Product/Admin の `credentialMode`、Cookie/Bearer response、`authContextId`、context refresh request/response、failure response、clear-cookie command list、context index update hints を追加・更新する。完了条件: Product/Admin browser Cookie mode と external Bearer mode の DTO が TypeSpec 上で分離し、`[API-CONTRACT-BE-S011]`、`[API-CONTRACT-BE-S012]` の contract tests がある。
- [ ] 1.2 `packages/typespec/src/routes/v1/auth.tsp` と `packages/typespec/src/routes/admin-v1/auth.tsp` に Product/Admin それぞれの `POST /api/v1/auth/contexts/{authContextId}/refresh`、login/setup/register/logout response shape、BearerAuth declaration を追加・更新する。完了条件: Product/Admin とも `/api/v1/*` のままで `/api/admin/*` がない。
- [ ] 1.3 Product/Admin surface separation lint と codegen boundary を更新する。対象: `packages/typespec/.spectral.yaml`、`packages/typespec/spectral/*.js`、`packages/typespec/scripts/check-surface-boundaries.mjs`。完了条件: `[API-CONTRACT-BE-S001]`、`[API-CONTRACT-BE-S002]`、`[API-CONTRACT-BE-S003]`、`[API-CONTRACT-BE-S010]`、`[API-CONTRACT-BE-S006]`、`[API-CONTRACT-BE-S007]`、`[API-CONTRACT-BE-S008]`、`[API-CONTRACT-BE-S009]` を検出できる contract/codegen tests がある。
- [ ] 1.4 `pnpm gen` を実行する。完了条件: Product OpenAPI / SDK / Go bindings と Admin OpenAPI / SDK / Go bindings が TypeSpec から再生成され、生成物を手編集していない。
- [ ] 1.5 `pnpm check:codegen` を実行する。完了条件: Product/Admin generated artifacts の drift と surface contamination がない。

## 2. Shared backend auth primitives

- [ ] 2.1 `packages/backend/internal/application/shared/authprimitive` などの中立 package を追加・更新し、Cookie path construction、Cookie clear command、TTL validation、opaque token hash、failure normalization を実装する。完了条件: Product/Admin domain enum、RBAC、status 判定、operator active 判定を含まない。
- [ ] 2.2 shared primitive の unit/import-boundary tests を追加する。test title に `[AUTH-BE-S064]`、`[AUTH-BE-S065]`、`[AUTH-BE-S068]`、`[AUTH-BE-S082]` を含める。完了条件: Cookie TTL 上限、TTL validation、中立 primitive 境界、protected request での accessToken TTL 非延長が repository script 経由で検証される。
- [ ] 2.3 Product/Admin application import-boundary tests を追加・更新する。test title に `[AUTH-BE-S067]`、`[AUTH-BE-S069]`、`[AUTH-BE-S070]`、`[AUTH-BE-S071]`、`[AUTH-BE-S072]` を含める。完了条件: Product/Admin domain eligibility ownership、Product application と Admin application の相互 import、単一 `identityDomain` switch が検出される。

## 3. Product backend 実装

- [ ] 3.1 Product session issuance を更新し、Cookie mode で body accessToken/authContextId/metadata と path-scoped refresh Cookie、Bearer mode で body accessToken/refreshToken を返す。対象: `packages/backend/internal/application/auth_service.go`、`token_service.go`、Product DTO。完了条件: `[AUTH-BE-S060]`、`[AUTH-BE-S063]` の tests が通る。
- [ ] 3.2 Product context refresh を `POST /api/v1/auth/contexts/{authContextId}/refresh` に実装し、Cookie mode と Bearer mode の exactly-one refresh credential、Authorization header rejection、path ownership validation、atomic rotation、Cookie Path 非認可境界、同時 refresh race handling を実装する。完了条件: `[AUTH-BE-S062]`、`[AUTH-BE-S066]`、`[AUTH-BE-S083]`、`[AUTH-BE-S086]`、`[AUTH-BE-S087]`、`[AUTH-BE-S080]`、`[AUTH-BE-S090]`、`[AUTH-BE-S091]`、`[AUTH-BE-S044]`、`[AUTH-BE-S045]` の endpoint/application tests がある。
- [ ] 3.3 Product protected route middleware を Bearer-only に更新し、Cookie credential、refresh Cookie、`X-Auth-Context-Id`、CSRF を認可材料にしない。完了条件: `[AUTH-BE-S084]`、`[AUTH-BE-S085]`、`[AUTH-BE-S009]`、`[AUTH-BE-S046]` の tests がある。
- [ ] 3.4 Product logout / revoke / suspend / restore flow を accessToken claims と refresh token family に接続し、対象 refresh Cookie path の clear command を返す。完了条件: `[AUTH-BE-S042]`、`[AUTH-BE-S092]`、`[AUTH-BE-S054]`、`[AUTH-BE-S055]`、`[AUTH-BE-S058]`、`[AUTH-BE-S056]`、`[AUTH-BE-S057]`、`[AUTH-BE-S059]` の tests がある。
- [ ] 3.5 Product auth state store unavailable、allowed Origin、Fetch Metadata、CORS credential policy、no-store / security header handling を更新する。完了条件: `[AUTH-BE-S010]`、`[AUTH-BE-S088]`、`[AUTH-BE-S089]` を含む fail-close endpoint tests がある。
- [ ] 3.6 Product backend tests を `packages/backend/internal/adapter/http/product/*_test.go`、`packages/backend/internal/application/*_test.go` に追加・更新し、test title に 3.1〜3.5 の Scenario ID を角括弧で入れる。

## 4. Admin backend 実装

- [ ] 4.1 Admin login/setup/operator-setup issuance を更新し、Cookie mode で body operatorAccessToken/authContextId/metadata と path-scoped refresh Cookie、Bearer mode で body operatorAccessToken/refreshToken を返す。完了条件: `[ADMIN-AUTH-BE-S074]`、`[ADMIN-AUTH-BE-S079]`、`[ADMIN-AUTH-BE-S064]`、`[ADMIN-AUTH-BE-S065]` の tests がある。
- [ ] 4.2 Admin context refresh を Product と同じ relative path で Admin service に実装し、Cookie/body exactly-one、Authorization header rejection、path ownership validation、atomic rotation、Bearer refresh success、reuse/theft family revocation を実装する。完了条件: `[ADMIN-AUTH-BE-S081]`、`[ADMIN-AUTH-BE-S082]`、`[ADMIN-AUTH-BE-S083]`、`[ADMIN-AUTH-BE-S084]`、`[ADMIN-AUTH-BE-S085]`、`[ADMIN-AUTH-BE-S078]` の tests がある。
- [ ] 4.3 Admin protected route middleware を Bearer-only に更新し、Product bearer、Product Cookie、Admin refresh Cookie、legacy access Cookie、`X-Auth-Context-Id`、CSRF を認可材料にしない。完了条件: `[ADMIN-AUTH-BE-S057]`、`[ADMIN-AUTH-BE-S058]`、`[ADMIN-AUTH-BE-S080]`、`[ADMIN-AUTH-BE-S060]` の tests がある。
- [ ] 4.4 Admin Cookie-setting / pre-auth flows に allowed Origin、Fetch Metadata、CORS、no-store、security headers を適用する。完了条件: `[ADMIN-AUTH-BE-S059]`、`[ADMIN-AUTH-BE-S061]` の endpoint tests がある。
- [ ] 4.5 Admin domain eligibility と Valkey namespace tests を維持・更新する。test title に `[ADMIN-AUTH-BE-S056]` を含め、既存 canonical regression として `[ADMIN-AUTH-BE-S062]`、`[ADMIN-AUTH-BE-S063]` も継続検証する。
- [ ] 4.6 Admin Console backend surface / RBAC / logout を Bearer operator accessToken + server-side session record 前提へ更新する。完了条件: `[ADMIN-CONSOLE-BE-S056]`、`[ADMIN-CONSOLE-BE-S057]`、`[ADMIN-CONSOLE-BE-S058]`、`[ADMIN-CONSOLE-BE-S094]`、`[ADMIN-CONSOLE-BE-S068]`、`[ADMIN-CONSOLE-BE-S069]`、`[ADMIN-CONSOLE-BE-S095]`、`[ADMIN-AUTH-BE-S086]` の tests がある。
- [ ] 4.7 Admin backend tests を `packages/backend/internal/adapter/http/admin/*_test.go`、`packages/backend/internal/application/admin/**/*_test.go`、boundary tests に追加・更新し、4.1〜4.6 の Scenario ID を test title に入れる。

## 5. Product frontend 実装

- [ ] 5.1 `packages/frontend/domain/src/auth/types.ts` と session state を、session item = `authContextId` + identity/session metadata + short-lived `accessToken` の memory-only model に更新する。完了条件: refreshToken / Cookie value が型にも state にも存在しない。
- [ ] 5.2 Product API request helpers を active accessToken の `Authorization` header に統一し、protected API で `X-Auth-Context-Id` と CSRF header を生成しないようにする。完了条件: `[AUTH-FE-S057]` の unit/component test がある。
- [ ] 5.3 Product context refresh URL builder、refresh-once retry、in-flight aggregation、target-session-only failure handling を実装する。完了条件: `[AUTH-FE-S045]`、`[AUTH-FE-S047]`、`[AUTH-FE-S055]` の tests がある。
- [ ] 5.4 Product context index/bootstrap を origin-local `localStorage` で実装し、schema/version/namespace/expiry/cleanup、token/secret 非保存、tamper fail-close、multi-tab propagation を検証する。完了条件: `[AUTH-FE-S056]`、`[AUTH-FE-S058]`、`[AUTH-FE-S059]` の unit/e2e tests がある。
- [ ] 5.5 Product AccountSwitcher と login/recovery/register/logout flows を multi-session item model に更新する。完了条件: `[AUTH-FE-S048]`、`[AUTH-FE-S049]`、`[AUTH-FE-S050]`、`[AUTH-FE-S060]`、`[AUTH-FE-S006]`、`[AUTH-FE-S007]`、`[AUTH-FE-S008]` の tests がある。
- [ ] 5.6 Product secret leakage tests を追加・更新する。test title に `[AUTH-FE-S046]` を含め、memory/localStorage/sessionStorage/IndexedDB/URL/telemetry/console に refreshToken と Cookie value がないことを確認する。

## 6. Admin frontend 実装

- [ ] 6.1 `packages/admin/api/src/client.ts` を Admin Console wrapper と Admin automation Bearer wrapper に分離する。完了条件: Console wrapper は active operator accessToken の Authorization を使い、refreshToken/Cookie value を受け取らない。
- [ ] 6.2 `packages/admin/domain/src/auth.ts` と `useAdminSession.svelte.ts` を operator session item = `authContextId` + operator/session metadata + short-lived operator accessToken の memory-only model に更新する。完了条件: `[ADMIN-AUTH-FE-S033]` の tests がある。
- [ ] 6.3 Admin current/protected route verification を active operator accessToken の Authorization header に統一し、Product SDK / Product Cookie / `X-Auth-Context-Id` / CSRF を使わない。完了条件: `[ADMIN-AUTH-FE-S027]`、`[ADMIN-AUTH-FE-S028]`、`[ADMIN-AUTH-FE-S030]`、`[ADMIN-AUTH-FE-S031]`、`[ADMIN-AUTH-FE-S034]` の tests がある。
- [ ] 6.4 Admin context refresh URL builder、refresh-once retry、session-expiry cleanup を実装する。完了条件: `[ADMIN-AUTH-FE-S035]`、`[ADMIN-AUTH-FE-S036]`、`[ADMIN-AUTH-FE-S037]` の tests がある。
- [ ] 6.5 Admin context index/bootstrap を origin-local `localStorage` で実装し、schema/version/namespace/expiry/cleanup、token/secret 非保存、tamper fail-close、multi-tab propagation を検証する。完了条件: `[ADMIN-AUTH-FE-S041]`、`[ADMIN-AUTH-FE-S042]`、`[ADMIN-AUTH-FE-S043]` の unit/e2e tests がある。
- [ ] 6.6 Admin setup/operator-setup login completion を updated session model に接続し、setup token / refreshToken / Cookie value を保存しない。完了条件: `[ADMIN-AUTH-FE-S038]`、`[ADMIN-AUTH-FE-S039]`、`[ADMIN-AUTH-FE-S040]`、`[ADMIN-AUTH-FE-S029]` の tests がある。

## 7. Repository verification

- [ ] 7.1 `pnpm lint` を実行する。完了条件: repository script 経由の lint が成功する。
- [ ] 7.2 `pnpm check` を実行する。完了条件: TypeScript / Svelte / Go typecheck が成功する。
- [ ] 7.3 `pnpm test:server` を実行する。完了条件: Product/Admin backend Scenario ID 付き tests が成功する。
- [ ] 7.4 `pnpm test:client` を実行する。完了条件: Product frontend Scenario ID 付き tests が成功する。
- [ ] 7.5 `pnpm test:admin` を実行する。完了条件: Admin API/domain/app Scenario ID 付き tests が成功する。
- [ ] 7.6 `pnpm test:e2e` を実行する。完了条件: Product Web と Admin Console の login / refresh / protected request / logout smoke が成功する。
- [ ] 7.7 `pnpm build` を実行する。完了条件: release build が成功する。
- [ ] 7.8 すべての実装差分後に `pnpm check:codegen` を再実行する。完了条件: Product/Admin generated artifacts に drift と contamination がない。
- [ ] 7.9 実装中に仕様差分が見つかった場合、この OpenSpec change の `proposal.md`、`design.md`、`specs/*/spec.md`、`tasks.md` を更新し、`openspec validate "unified-context-scoped-auth-refresh" --type change --strict --no-interactive` を実行する。
