## Why

現在の Product Web 認証と Admin Console 認証は、ログイン後に browser-readable な `accessToken` を frontend memory state に保持し、`Authorization: Bearer` header として API へ送信する前提になっている。refreshToken はすでに HttpOnly Cookie として扱われているが、accessToken が JavaScript 可読である限り、XSS や debugging surface から session credential が漏えいするリスクが残る。

一方で、Product API / mobile / CLI / SDK 利用者には Bearer token による明示的な認証方式が必要である。顧客と運営者にとって安全な browser 体験と、外部 client にとって扱いやすい Product API 体験を両立するには、Cookie と Bearer を credential transport の違いとして分離し、実際に処理する account/session context は server-issued `X-Auth-Context-Id` selector で明示したうえで fail-close できる認証契約が必要である。

## What Changes

- Product Web と Admin Console は HttpOnly Cookie を credential transport として authenticated API を利用し、browser-readable accessToken を保持しない。
- API / mobile / CLI / SDK は `credentialMode="bearer"` を明示したときだけ Bearer accessToken を response body で受け取れる。
- Admin API / automation client は Admin 用 `credentialMode="bearer"` を明示したときだけ operator Bearer accessToken を response body で受け取れる。
- Product Web クライアントは `credentialMode="cookie"` を明示し、credential container と refresh credential を HttpOnly Cookie として受け取る。
- Admin Console は `credentialMode="cookie"` を明示し、operator credential container と refresh credential を HttpOnly Cookie として受け取る。
- Cookie mode の refresh credential は Cookie `Path` で refresh endpoint へ送信範囲を限定する。browser JavaScript は送信 Cookie を個別選択できないため、refresh endpoint は同送されうる access credential Cookie を認可材料にも credential ambiguity 判定にも使用しない。
- Access credential は Product/Admin とも短命 TTL を発行時に固定し、通常の protected request では延命しない。session 継続は refresh rotation による新しい access credential 発行だけで行う。
- Product protected route は Cookie / Bearer のどちらの transport でも `X-Auth-Context-Id` を auth context selector として要求し、credential がその selector を利用できる場合だけ request context に account/session を束縛する。
- Admin protected route は Cookie / Bearer のどちらの transport でも `X-Auth-Context-Id` を auth context selector として要求し、credential がその selector を利用できる場合だけ request context に operator/session を束縛する。
- Product Web の複数アカウント切り替えは browser-readable token list ではなく、Cookie credential container に紐づく server-owned auth context registry と `X-Auth-Context-Id` の選択で提供する。
- Admin Console の operator context 切り替えは browser-readable token list ではなく、Cookie credential container に紐づく server-owned auth context registry と `X-Auth-Context-Id` の選択で提供する。
- Admin Console は Admin operator session の credential と refresh credential を HttpOnly Cookie として受け取り、response body には operator accessToken / refreshToken 平文を含めない。
- Protected route で access credential Cookie と `Authorization: Bearer <accessToken>` が同時提示された場合、backend は credential ambiguity として request を拒否する。refresh credential は protected route の exactly-one credential 判定対象ではない。
- Refresh endpoint を除く protected Cookie access-credential state-changing request は CSRF / Origin 境界で保護される。Cookie refresh は既存の `X-Auth-Context-Id` / CSRF token を要求せず、成功後に新しい `authContextId` / CSRF token を返す。
- Cookie mode の refreshToken は HttpOnly Cookie 専用 secret として扱い、request / response body や JavaScript 可読 state に戻さない。Bearer mode の refreshToken は API / mobile / CLI / SDK 用 credential として body で返す。
- **BREAKING** Product Web frontend と Admin Console の auth state は `accessToken` 依存をやめ、既存の `Authorization` header 前提の domain API 呼び出し契約を Cookie + `X-Auth-Context-Id` + CSRF 送信へ変更する。
- **BREAKING** Product auth session / refresh response は credential mode ごとに shape を分け、Cookie mode では accessToken を body に返さない。
- **BREAKING** Admin auth session / refresh response は credential mode ごとに shape を分け、Admin Console の Cookie mode では browser-readable operator accessToken を body に返さず、Admin protected routes は Admin credential transport + `X-Auth-Context-Id` + CSRF / Origin で認可する。

## Spec Units

### New Spec Units

- なし。既存の `auth-fe` と `auth-be` が Product 認証の永続責務をすでに所有しているため、新しい Spec Unit は追加しない。

### Modified Spec Units

- `auth-fe`: Product Web の session state、login / recovery registration / refresh / logout / account switching / authenticated API 呼び出しを、browser-readable Bearer token ではなく HttpOnly Cookie、`X-Auth-Context-Id`、CSRF token を使う契約へ変更する。
- `auth-be`: Product API の session issuance、refresh、protected route authorization、protected access credential の Cookie/Bearer ambiguity rejection、`X-Auth-Context-Id` selector 検証、CSRF / Origin enforcement、credential-mode response shape を定義する。
- `admin-auth-fe`: Admin Console の session state、login / setup / refresh / logout / protected route verification / passkey management / operator context switching を、browser-readable operator Bearer token ではなく HttpOnly Cookie、`X-Auth-Context-Id`、CSRF token を使う契約へ変更する。
- `admin-auth-be`: Admin backend の operator session issuance、refresh、protected route authorization、protected access credential の Cookie/Bearer ambiguity rejection、`X-Auth-Context-Id` selector 検証、CSRF / Origin enforcement、Admin credential response shape、Admin OpenAPI / SDK / Go bindings 分離を定義する。
- `admin-console-be`: Admin 管理 API surface と RBAC controller 境界を、browser-readable operator accessToken ではなく Admin credential transport、operator session record、`X-Auth-Context-Id`、CSRF binding、RBAC authorization use case に揃える。

## Naming

- 変更対象 Spec Unit は既存の `auth-fe`、`auth-be`、`admin-auth-fe`、`admin-auth-be` を使用する。
- Frontend scenario は `AUTH-FE-S###`、backend scenario は `AUTH-BE-S###` の既存 prefix を継続し、FE / BE の scenario ID は別系列として扱う。
- Admin scenario は既存の `ADMIN-AUTH-FE-S###` と `ADMIN-AUTH-BE-S###` prefix を継続し、Product scenario と混在させない。
- Admin Console backend scenario は既存の `ADMIN-CONSOLE-BE-S###` prefix を継続し、Admin auth scenario と役割境界を分ける。

## Impact

- Impacted packages: `packages/typespec`, `packages/backend`, `packages/frontend/api`, `packages/frontend/domain`, `packages/frontend/app`, `packages/admin/api`, `packages/admin/domain`, `packages/admin/app`。
- API contract: `packages/typespec/main.tsp` を source of truth として Product/Admin credential mode、Cookie session response、Bearer session response、`X-Auth-Context-Id`、CSRF header、Cookie/Bearer security scheme を更新し、`pnpm gen` で Product / Admin の OpenAPI、SDK、Go bindings を同じ生成入口から再生成・検証する。
- Admin artifacts: `packages/typespec/openapi/admin.openapi.json`、`packages/admin/api/src/generated/client.ts`、`packages/backend/internal/generated/adminopenapi/openapi.gen.go` は Admin Cookie/Bearer session response と `X-Auth-Context-Id` / CSRF header を反映し、Product auth operation を混入させず、Admin origin の `/api/v1/*` contract と operator auth SDK / Go bindings の分離を保つ。
- Security: browser-readable accessToken exposure をなくし、protected access credential の Cookie/Bearer 同時提示拒否、auth context selector の credential 所属検証、CSRF / Origin enforcement、no-store response を必須にする。
- Persistence: 既存 session / refresh token store の session metadata に Cookie CSRF binding と auth context registry を追加する可能性がある。永続 DB migration は不要な見込みだが、Valkey auth state schema は変更対象になる。
- Operations: Web auth cookie の SameSite / Secure / HttpOnly / Path / Max-Age 設定、allowed origin 判定、CORS header の見直しが必要になる。
- Tests: Product backend auth middleware / handler / token rotation tests、Product frontend auth session / passkey login / passkey management / account settings tests、Admin backend auth middleware / handler / token rotation tests、Admin frontend auth session / setup / passkey management tests、Product/Admin codegen drift / contamination check が影響を受ける。
