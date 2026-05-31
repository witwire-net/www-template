## Why

現行 Product frontend は複数 account/session を `accessToken` と `activeSessionId` で memory-only に保持し、利用者が AccountSwitcher で active session を切り替えられる。一方、backend の refresh credential は単一 `refresh_token` HttpOnly Cookie として最後の login / refresh で上書きされるため、短命 accessToken の切り替え対象と refresh 対象が一致しない。これは複数アカウントを同時に扱う顧客体験と、refreshToken を JavaScript へ露出しないセキュリティ目標の両方を損なう。

同じ問題は Admin Console、Product external API / mobile / CLI / SDK、Admin automation client を含む認証方式全体で解く必要がある。Product と Admin は origin / Go binary / TypeSpec service / OpenAPI artifact / SDK package / Go bindings を分離し続けるが、context-scoped refresh、short-lived access token、refresh token rotation、Cookie path construction、Cookie clear command、credential ambiguity rejection、TTL validation、no-store response、failure normalization は共通概念として揃える。refresh 対象は `POST /api/v1/auth/contexts/{authContextId}/refresh` の path と path-scoped Cookie で選択できるため、Product/Admin protected request に `X-Auth-Context-Id` header を要求しない。

## What Changes

- Product Web と Admin Console の `credentialMode="cookie"` は、response body に short-lived `accessToken`、`authContextId`、identity/session metadata を返し、`refreshToken` を body に返さない。
- Product Web と Admin Console は複数 account/operator/session context を memory-only に保持し、active session item の `accessToken` を `Authorization: Bearer <accessToken>` として protected API に送信する。
- Product/Admin browser refresh credential は `HttpOnly; Secure; SameSite=Lax` Cookie とし、各 origin の `/api/v1/auth/contexts/{authContextId}/refresh` に Path を限定する。
- Product/Admin context refresh endpoint は同じ relative path `POST /api/v1/auth/contexts/{authContextId}/refresh` を使うが、Product/Admin の origin、binary、TypeSpec service、OpenAPI artifact、SDK package、Go bindings は分離する。
- Cookie mode refresh は URL path の `authContextId` と path-scoped refresh Cookie の所属を検証して rotation し、新しい accessToken、同じ authContextId、session/identity metadata、path-scoped refresh Cookie を返す。body に refreshToken は返さない。
- External Bearer mode refresh は同じ endpoint の request body `refreshToken` を使い、成功時に body の `accessToken` と `refreshToken` を返し、Cookie を設定しない。
- Refresh endpoint は `Authorization` header を refresh credential として扱わず、Bearer refresh request に `Authorization` header がある場合、または body `refreshToken` と refresh Cookie が同時提示された場合は fail-close で拒否する。
- Product/Admin protected routes は `Authorization: Bearer <accessToken>` / `<operatorAccessToken>` のみを account/operator/session credential として使い、auth context は accessToken claims と server-side session record から束縛する。
- Product/Admin protected request に `X-Auth-Context-Id` header と CSRF token は導入しない。ambient refresh Cookie や legacy access Cookie は protected route の認可材料にしない。
- Browser frontend は `authContextId` を protected API header に使わず、context refresh URL construction、session metadata、UI selection、non-secret context index/bootstrap だけに使う。
- Browser memory が消えた後に path-scoped HttpOnly refresh Cookies だけでは context 一覧を JavaScript が発見できないため、Product/Admin それぞれの origin-local `localStorage` に token/secret を含まない context index を設け、tamper は refresh failure として fail-close する。
- Cookie `Path` は refresh credential の browser 送信先を選ぶための補助であり、認可境界ではない。backend は refresh token record に surface、authContextId、session、family、cookiePath を保存し、path と credential 所属の不一致を fail-close で拒否する。
- Product/Admin browser Cookie-setting flow と Cookie mode context refresh は allowed Origin、Fetch Metadata、CORS credential policy、SameSite=Lax、no-store/security headers を fail-close で検証する。
- Refresh rotation は family ID 単位で atomic consume + issue を行い、同時 refresh race、旧 token reuse、unknown/tampered token を replay/theft として扱い、定義済み範囲の refresh family を revoke する。
- Logout / revoke / suspend / operator deactivation は accessToken claims と refreshToken family の両方を失効し、対象 refresh Cookie path を clear する。
- **BREAKING** 古い単一 refresh Cookie 前提、Product/Admin protected Cookie auth 前提、protected request の `X-Auth-Context-Id` 前提、protected mutation の CSRF header 前提は残さない。

## Spec Units

### New Spec Units

- なし。既存の Product/Admin 認証 Spec Unit が永続責務をすでに所有しているため、新しい Spec Unit は追加しない。

### Modified Spec Units

- `auth-be`: Product backend の session issuance、context-scoped refresh、Bearer-only protected authorization、refresh credential ambiguity rejection、logout/revoke/suspend、共有中立 primitive 境界を変更する。
- `auth-fe`: Product Web の multi-account memory session、active accessToken request、path-scoped refresh URL、context index/bootstrap、logout/expiry handling、secret 非保存を変更する。
- `admin-auth-be`: Admin backend の operator session issuance、context-scoped refresh、Bearer-only protected authorization、setup/operator-setup/login refresh、RBAC 前段の operator/session validation、共有中立 primitive 境界を変更する。
- `admin-auth-fe`: Admin Console の multi-operator memory session、active operator accessToken request、path-scoped refresh URL、context index/bootstrap、setup/login/logout/expiry handling、secret 非保存を変更する。
- `admin-console-be`: Admin 管理 API surface と RBAC controller 境界を、Bearer operator accessToken claims と server-side operator/session record 検証後の application authorization に揃え、Product bearer / Cookie / `/api/admin/*` 混入を防ぐ。
- `api-contract-be`: TypeSpec service separation、Product/Admin OpenAPI・SDK・Go bindings の surface isolation、context refresh route の両 surface 分離生成、codegen drift / contamination 検証を変更する。

## Naming

- Product frontend scenario は `AUTH-FE-S###`、Product backend scenario は `AUTH-BE-S###` の既存 prefix を使う。
- Admin auth frontend scenario は `ADMIN-AUTH-FE-S###`、Admin auth backend scenario は `ADMIN-AUTH-BE-S###` の既存 prefix を使う。
- Admin Console backend scenario は `ADMIN-CONSOLE-BE-S###` の既存 prefix を使う。
- API contract scenario は `API-CONTRACT-BE-S###` の既存 prefix を使う。
- FE / BE scenario ID は別 prefix として扱い、Product と Admin の scenario ID を混在させない。

## Impact

- Impacted packages: `packages/typespec`, `packages/backend`, `packages/frontend/api`, `packages/frontend/domain`, `packages/frontend/app`, `packages/admin/api`, `packages/admin/domain`, `packages/admin/app`。
- API contract: `packages/typespec/main.tsp` を source of truth とし、Product/Admin の Cookie/Bearer credential mode、context refresh endpoint、response shape、BearerAuth、surface separation を更新する。
- Generated artifacts: `pnpm gen` で Product/Admin OpenAPI、SDK、Go bindings を再生成し、`pnpm check:codegen` で drift と surface contamination を検証する。生成物は手編集しない。
- Security: refreshToken は browser-readable state/storage/log/telemetry に置かず、Cookie/body refresh credential の exactly-one、Authorization-on-refresh rejection、authContextId 所属検証、no-store、Origin / SameSite / Fetch Metadata fail-close を適用する。
- Frontend state: Product Web と Admin Console は複数 session item を memory-only に保持し、active item の accessToken だけを protected request に使う。
- Persistence/operations: server-side refresh token family、authContextId、Cookie Path、Cookie clear command、context index integrity、Valkey namespace、token TTL validation、logout/revoke/suspend の失効境界が影響を受ける。
