## 1. 契約と生成物の分離

- [ ] 1.1 `packages/typespec` に Product service と Admin service を分離して定義し、両 surface が `/api/v1/*` path policy を維持する状態にする。
- [ ] 1.2 Admin account / auth operation を `packages/typespec/src/routes/admin-v1/**` に追加し、Product route namespace から import されない状態にする。
- [ ] 1.3 Product OpenAPI と Admin OpenAPI を別 artifact として生成する `packages/typespec` scripts / config を更新する。
- [ ] 1.4 Product SDK は `packages/frontend/api`、Admin SDK は `packages/admin/api` に生成されるように Orval / package scripts を分離する。
- [ ] 1.5 Product Go bindings と Admin Go bindings を別 package に生成するように `scripts/go/gen-backend.sh` と backend codegen config を更新する。
- [ ] 1.6 `scripts/codegen/check.sh` に Product/Admin artifact の drift と operation/tag/export 混入検査を追加する。
- [ ] 1.7 `AGENTS.md`、`CODING_STANDARDS.md`、`CONTRIBUTING.md` に Product/Admin とも `/api/v1/*` を使い、origin / binary / artifact で分離するルールを反映する。
- [ ] 1.8 `pnpm gen` を実行し、Product/Admin OpenAPI、Product SDK、Admin package-local SDK、Product/Admin Go bindings を生成する。
- [ ] 1.9 `[API-CONTRACT-BE-S001] Product OpenAPI excludes admin operations` を追加し、Product OpenAPI / Product SDK / Product Go bindings に Admin operation がないことを検証する。
- [ ] 1.10 `[API-CONTRACT-BE-S002] Admin OpenAPI excludes product operations` を追加し、Admin OpenAPI / Admin SDK / Admin Go bindings に Product operation がないことを検証する。
- [ ] 1.11 `[API-CONTRACT-BE-S003] Surface server URLs are separated` を追加し、Product/Admin OpenAPI の server domain が分かれていることを検証する。
- [ ] 1.12 `[API-CONTRACT-BE-S004] Shared model import does not add routes` を追加し、shared model import が route を増やさないことを検証する。
- [ ] 1.13 `[API-CONTRACT-BE-S005] Product surface cannot import admin namespace` を追加し、Product TypeSpec から Admin namespace import を拒否する。
- [ ] 1.14 `[API-CONTRACT-BE-S006] Product artifact with admin operation fails check` を追加し、混入 fixture が codegen check で失敗することを検証する。
- [ ] 1.15 `[API-CONTRACT-BE-S007] Product binary cannot import admin bindings` を追加し、Product binary から Admin bindings import を拒否する。

## 2. Backend Admin runtime と永続化

- [ ] 2.1 `packages/backend/cmd/admin-api/main.go` と `internal/app/admin_runtime.go` を追加し、Admin GoServer binary を Product binary と分離して起動できる状態にする。
- [ ] 2.2 Product runtime が Product operations だけを register することを維持し、Admin operations が混入しない router 構成にする。
- [ ] 2.3 Admin runtime が Admin operations の `/api/v1/*` だけを register し、Product operations を register しない router 構成にする。
- [ ] 2.4 Product DB 内 Admin schema、operator、operator passkey、audit event、権限を作成する `000000_admin_schema.up.sql/down.sql` 形式の migration を追加する。
- [ ] 2.5 Admin runtime config に Admin domain、Product domain、Admin cookie、Admin DB role、Admin Valkey URL を追加し、起動時に fail-close validation する。
- [ ] 2.6 Admin Valkey store を追加し、Product と同じ infrastructure かつ別 logical DB、`admin:*` prefix 限定を検証する。
- [ ] 2.7 Admin auth middleware を追加し、same-origin Origin、operator session、CSRF binding、no-store、security headers を検証する。
- [ ] 2.8 `accounts:create` を含む Admin RBAC permission map を追加し、handler 境界で検証する。
- [ ] 2.9 Admin audit service を追加し、mutation 前の intent と success / failed outcome 更新を一元化する。
- [ ] 2.10 Admin account repository を追加し、Product DB 内 Admin schema と Account root を同一 transaction 境界で扱う。
- [ ] 2.11 Admin account service を追加し、Product Account domain rule を共有して email 正規化、重複検査、作成、監査を実行する。
- [ ] 2.12 Admin account handlers を generated Admin bindings に接続し、validation / duplicate / permission / session / CSRF / infrastructure errors を stable response に map する。
- [ ] 2.13 Admin auth handlers を generated Admin bindings に接続し、passkey start/finish、operator setup、current operator、CSRF issuance、logout を実装する。
- [ ] 2.14 `[ADMIN-CONSOLE-BE-S056] Product binary does not register admin operations` を追加する。
- [ ] 2.15 `[ADMIN-CONSOLE-BE-S057] Admin binary does not register product operations` を追加する。
- [ ] 2.16 `[ADMIN-CONSOLE-BE-S058] Product bearer token is rejected by Admin API` を追加する。
- [ ] 2.17 `[ADMIN-CONSOLE-BE-S059] Admin schema exists in Product DB` を追加する。
- [ ] 2.18 `[ADMIN-CONSOLE-BE-S060] Product runtime role cannot read Admin schema` を追加する。
- [ ] 2.19 `[ADMIN-CONSOLE-BE-S061] Admin package ORM migration is not used for Product DB` を追加する。
- [ ] 2.20 `[ADMIN-CONSOLE-BE-S062] Admin API creates customer account` を追加する。
- [ ] 2.21 `[ADMIN-CONSOLE-BE-S063] Duplicate email returns 409 and failed audit` を追加する。
- [ ] 2.22 `[ADMIN-CONSOLE-BE-S064] Operator without account create permission receives 403` を追加する。
- [ ] 2.23 `[ADMIN-CONSOLE-BE-S065] Audit intent failure prevents account mutation` を追加する。
- [ ] 2.24 `[ADMIN-CONSOLE-BE-S066] Account creation failure records failed audit outcome` を追加する。
- [ ] 2.25 `[ADMIN-CONSOLE-BE-S067] Admin account creation shares Account domain rule` を追加する。
- [ ] 2.26 `[ADMIN-CONSOLE-BE-S068] Admin and operator have accounts:create` を追加する。
- [ ] 2.27 `[ADMIN-CONSOLE-BE-S069] Viewer lacks accounts:create` を追加する。
- [ ] 2.28 `[ADMIN-CONSOLE-BE-S070] Product binary importing admin bindings fails` を追加する。
- [ ] 2.29 `[ADMIN-AUTH-BE-S056] Product host does not serve Admin login API` を追加する。
- [ ] 2.30 `[ADMIN-AUTH-BE-S057] Admin middleware validates operator accessToken` を追加する。
- [ ] 2.31 `[ADMIN-AUTH-BE-S058] Product bearer token is not an Admin auth session` を追加する。
- [ ] 2.32 `[ADMIN-AUTH-BE-S059] Disallowed Origin is rejected for Admin mutation` を追加する。
- [ ] 2.33 `[ADMIN-AUTH-BE-S060] Session-mismatched CSRF token is rejected` を追加する。
- [ ] 2.34 `[ADMIN-AUTH-BE-S061] Passkey start validates Origin without session CSRF` を追加する。
- [ ] 2.35 `[ADMIN-AUTH-BE-S062] Admin and Product Valkey same logical DB fails startup` を追加する。
- [ ] 2.36 `[ADMIN-AUTH-BE-S063] Admin backend only writes admin-prefixed keys` を追加する。
- [ ] 2.37 `[ADMIN-AUTH-BE-S064] Admin refreshToken Cookie uses SameSite=Lax` を追加する。
- [ ] 2.38 `[ADMIN-AUTH-BE-S065] Insecure production cookie is rejected` を追加する。
- [ ] 2.39 `[ADMIN-AUTH-BE-S066] Admin API response has security headers` を追加する。

## 3. Admin 静的 frontend 層

- [ ] 3.1 `packages/admin` を `app`、`domain`、`api` の layer に整理し、`app -> domain -> api` 以外の依存を lint で拒否する。
- [ ] 3.2 `packages/admin` の Node adapter、server routes、server load/actions、`$lib/server`、Prisma、Valkey、OpenSearch、WebAuthn server dependency、Prisma generation scripts を削除する。
- [ ] 3.3 `packages/admin/api` に Admin SDK 生成設定と same-origin `/api/v1/*` wrapper を追加し、Product domain への request を拒否する。
- [ ] 3.4 `packages/admin/domain` に auth、current operator、protected route state、account search/detail/create domain functions を追加する。
- [ ] 3.5 `packages/admin/app` の login / operator setup を browser WebAuthn と domain functions 経由に変更する。
- [ ] 3.6 `packages/admin/app` の protected routes を current operator verification 後に表示する構成に変更する。
- [ ] 3.7 Account 作成 component を追加し、Accounts page から validation、submit、duplicate/error 表示、detail navigation を扱う。
- [ ] 3.8 `[ADMIN-CONSOLE-FE-S038] Admin app layer direct API client import fails` を追加する。
- [ ] 3.9 `[ADMIN-CONSOLE-FE-S039] Admin package server-only module fails` を追加する。
- [ ] 3.10 `[ADMIN-CONSOLE-FE-S040] Admin domain uses Admin api layer for account data` を追加する。
- [ ] 3.11 `[ADMIN-CONSOLE-FE-S041] Admin API uses same-origin api/v1` を追加する。
- [ ] 3.12 `[ADMIN-CONSOLE-FE-S042] Admin API wrapper rejects Product domain` を追加する。
- [ ] 3.13 `[ADMIN-CONSOLE-FE-S043] Operator creates customer account` を component / E2E で追加する。
- [ ] 3.14 `[ADMIN-CONSOLE-FE-S044] Invalid email is not submitted` を追加する。
- [ ] 3.15 `[ADMIN-CONSOLE-FE-S045] Duplicate email error preserves form input` を追加する。
- [ ] 3.16 `[ADMIN-CONSOLE-FE-S046] Admin frontend domain differs from Product frontend domain` を追加する。
- [ ] 3.17 `[ADMIN-AUTH-FE-S027] Login UI calls Admin backend auth API` を追加する。
- [ ] 3.18 `[ADMIN-AUTH-FE-S028] Product auth SDK is not used for operator session creation` を追加する。
- [ ] 3.19 `[ADMIN-AUTH-FE-S029] Setup token errors map to non-revealing presentation` を追加する。
- [ ] 3.20 `[ADMIN-AUTH-FE-S030] Protected content is hidden without session` を追加する。
- [ ] 3.21 `[ADMIN-AUTH-FE-S031] UI role controls do not replace backend authorization` を追加する。
- [ ] 3.22 `[ADMIN-AUTH-FE-S032] Admin HTML is no-store` を追加する。
- [ ] 3.23 `[ADMIN-AUTH-FE-S033] Operator login stores only accessToken in browser-readable state` を追加する。
- [ ] 3.24 `[ADMIN-AUTH-FE-S034] Protected route uses operator accessToken for verification` を追加する。
- [ ] 3.25 `[ADMIN-AUTH-FE-S035] Admin refresh uses HttpOnly Cookie` を追加する。

## 4. Product/Admin 共通 token 認証基盤

- [ ] 4.1 Product account と Admin operator が再利用する accessToken / refreshToken service を追加し、identity domain、session state、Valkey namespace を引数で明示する。
- [ ] 4.2 Product auth handlers を accessToken response body + refreshToken `HttpOnly; Secure; SameSite=Lax; Path=/` Cookie model に変更し、response body から refreshToken を削除する。
- [ ] 4.3 Admin auth handlers を共通 token service に接続し、operator accessToken と operator refreshToken Cookie を発行・refresh・revoke する。
- [ ] 4.4 refreshToken rotation を旧 token 原子消費、新 token Cookie 設定、body への refreshToken 非露出として実装する。
- [ ] 4.5 Product/Admin 共通 TTL validation を追加し、Cookie lifetime が server-side refreshToken state TTL を超えないことを保証する。
- [ ] 4.6 Product frontend auth domain を accessToken-only browser-readable state に変更し、refresh request は credentials 付き same-origin Cookie refresh とする。
- [ ] 4.7 複数 account session の refresh/logout で対象 session だけが rotation/revoke されるよう session selector と Cookie binding を検証する。
- [ ] 4.8 `[AUTH-BE-S060] Product passkey login returns accessToken body and refreshToken Cookie` を追加する。
- [ ] 4.9 `[AUTH-BE-S061] Admin operator login uses shared token service in operator domain` を追加する。
- [ ] 4.10 `[AUTH-BE-S062] refresh rotates Cookie refreshToken` を追加する。
- [ ] 4.11 `[AUTH-BE-S063] browser-readable refreshToken is not issued` を追加する。
- [ ] 4.12 `[AUTH-BE-S064] refreshToken Cookie lifetime does not exceed server TTL` を追加する。
- [ ] 4.13 `[AUTH-BE-S065] Product and Admin use same TTL validation` を追加する。
- [ ] 4.14 `[AUTH-BE-S066] multi-session refresh rotates only target session` を追加する。
- [ ] 4.15 `[AUTH-FE-S045] Expiring accessToken refreshes via Cookie` を追加する。
- [ ] 4.16 `[AUTH-FE-S046] refreshToken is not stored in browser-readable storage` を追加する。
- [ ] 4.17 `[AUTH-FE-S047] refresh failure expires only target session` を追加する。
- [ ] 4.18 `[AUTH-FE-S048] login adds accessToken session without refreshToken` を追加する。
- [ ] 4.19 `[AUTH-FE-S049] account switch changes bearer accessToken` を追加する。
- [ ] 4.20 `[AUTH-FE-S050] logout requests Cookie revoke for target session` を追加する。

## 5. Cloudflare routing と検証

- [ ] 5.1 Admin domain の Cloudflare route 設定を文書化し、static frontend と `/api/v1/*` GoServer routing を明示する。
- [ ] 5.2 Product domain と Admin domain が一致しないこと、かつ両 domain がそれぞれ same-origin `/api/v1/*` を持つことを deployment docs に反映する。
- [ ] 5.3 `pnpm gen` を実行し、Product/Admin 生成物の分離を確認する。
- [ ] 5.4 `pnpm check:codegen` を実行し、drift と surface contamination を修正する。
- [ ] 5.5 `pnpm check` を実行して TypeSpec、Svelte、TypeScript、Go build 問題を修正する。
- [ ] 5.6 `pnpm lint` を実行して layer、security、codegen policy 問題を修正する。
- [ ] 5.7 `pnpm test:run` を実行し、Scenario ID 付き automated tests が通ることを確認する。
- [ ] 5.8 `pnpm build` を実行し、Product API、Admin API、Product frontend、Admin static frontend を検証する。
- [ ] 5.9 環境が用意できる場合は `pnpm test:e2e` を実行し、Product/Admin domain separation と Account 作成 flow を検証する。
