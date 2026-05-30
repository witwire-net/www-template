## Why

現在の Product Web 認証は、ログイン後に browser-readable な `accessToken` を frontend memory state に保持し、`Authorization: Bearer` header として API へ送信する前提になっている。refreshToken はすでに HttpOnly Cookie として扱われているが、accessToken が JavaScript 可読である限り、XSS や debugging surface から session credential が漏えいするリスクが残る。

一方で、API / mobile / CLI / SDK 利用者には Bearer token による明示的な認証方式が必要である。顧客にとって安全な Web 体験と、外部 client にとって扱いやすい API 体験を両立するには、Web Cookie と Bearer を曖昧に混在させず、credential mode を明示したうえで fail-close できる認証契約が必要である。

## What Changes

- Product Web は HttpOnly Cookie を使って authenticated API を利用し、browser-readable accessToken を保持しない。
- API / mobile / CLI / SDK は `credentialMode="bearer"` を明示したときだけ Bearer accessToken を response body で受け取れる。
- Web client は `credentialMode="web-cookie"` を明示し、access credential と refresh credential を HttpOnly Cookie として受け取る。
- Cookie credential と `Authorization: Bearer` credential が同じ protected request に同時提示された場合、backend は credential ambiguity として request を拒否する。
- Cookie credential を使う state-changing request は CSRF / Origin 境界で保護される。
- Web Cookie mode の refreshToken は HttpOnly Cookie 専用 secret として扱い、request / response body や JavaScript 可読 state に戻さない。Bearer mode の refreshToken は API / mobile / CLI / SDK 用 credential として body で返す。
- **BREAKING** Product Web frontend の auth state は `accessToken` 依存をやめ、既存の `Authorization` header 前提の domain API 呼び出し契約を Cookie + CSRF 送信へ変更する。
- **BREAKING** Product auth session / refresh response は credential mode ごとに shape を分け、Web Cookie mode では accessToken を body に返さない。

## Spec Units

### New Spec Units

- なし。既存の `auth-fe` と `auth-be` が Product 認証の永続責務をすでに所有しているため、新しい Spec Unit は追加しない。

### Modified Spec Units

- `auth-fe`: Product Web の session state、login / recovery registration / refresh / logout / authenticated API 呼び出しを、browser-readable Bearer token ではなく HttpOnly Cookie と CSRF token を使う契約へ変更する。
- `auth-be`: Product API の session issuance、refresh、protected route authorization、Cookie/Bearer ambiguity rejection、CSRF / Origin enforcement、credential-mode response shape を定義する。

## Naming

- 変更対象 Spec Unit は既存の `auth-fe` と `auth-be` を使用する。
- Frontend scenario は `AUTH-FE-S###`、backend scenario は `AUTH-BE-S###` の既存 prefix を継続し、FE / BE の scenario ID は別系列として扱う。

## Impact

- Impacted packages: `packages/typespec`, `packages/backend`, `packages/frontend/api`, `packages/frontend/domain`, `packages/frontend/app`。
- API contract: `packages/typespec/main.tsp` を source of truth として credential mode、response union、CSRF header を更新し、`pnpm gen` で OpenAPI / SDK / Go bindings を再生成する。
- Security: browser-readable accessToken exposure をなくし、Cookie/Bearer 同時提示拒否、CSRF / Origin enforcement、no-store response を必須にする。
- Persistence: 既存 session / refresh token store の session metadata に Cookie CSRF binding を追加する可能性がある。永続 DB migration は不要な見込みだが、Valkey auth state schema は変更対象になる。
- Operations: Web auth cookie の SameSite / Secure / HttpOnly / Path / Max-Age 設定、allowed origin 判定、CORS header の見直しが必要になる。
- Tests: backend auth middleware / handler / token rotation tests、frontend auth session / passkey login / passkey management / account settings tests、codegen drift check が影響を受ける。
