## 1. DB Migrations

- [x] 1.1 `packages/admin/prisma/admin/schema.prisma` — Admin-owned schema 用 Prisma schema。`admin.operators` (id, email UNIQUE, display_name, role CHECK, is_active, setup_token_hash, setup_token_expires_at, last_login_at, created_at, updated_at), `admin.operator_passkeys` (id, operator_id FK CASCADE, credential_handle UNIQUE, public_key, sign_count DEFAULT 0, aaguid, backup_eligible, backup_state, transports JSONB, created_at), `admin.audit_events` (id, operator_id FK, action, target_type, target_id, details JSONB, outcome, error_code, ip_address, created_at, completed_at) を定義
- [x] 1.2 `packages/admin/prisma/admin/migrations/000001_create_operators_and_passkeys/migration.sql` — SQL migrations 用 SQL。初期オペレーター seed は作成しない
- [x] 1.3 `packages/admin/prisma/product/schema.prisma` — database 用 Prisma schema。`admin_view.account_summaries`, `admin_view.account_passkeys` 参照と `admin_op` 関数呼び出しに必要な型を定義。database に SQL migrations は適用しない
- [x] 1.5 `packages/backend/db/migrations/000004_add_account_status.up.sql` — accounts に status (CHECK IN active,suspended, DEFAULT active), status_reason, status_updated_at, status_updated_by, session_revoked_after 追加
- [x] 1.6 `packages/backend/db/migrations/000004_add_account_status.down.sql`
- [x] 1.7 `packages/backend/db/migrations/000005_create_admin_views.up.sql` — admin_view.account_summaries, admin_view.account_passkeys
- [x] 1.8 `packages/backend/db/migrations/000005_create_admin_views.down.sql`
- [x] 1.9 `packages/backend/db/migrations/000006_create_admin_functions.up.sql` — admin_console_read / admin_console_write NOLOGIN role、admin_view SELECT grant、admin_op.suspend_account(p_account_id,p_operator_id,p_reason,p_audit_event_id) / admin_op.restore_account(p_account_id,p_operator_id,p_audit_event_id)（SECURITY DEFINER, SET search_path = pg_catalog, admin_op, Product base table は `public.accounts` のように schema-qualified 参照, REVOKE EXECUTE FROM PUBLIC, GRANT EXECUTE TO admin_console_write）。suspend は同一 database transaction で session_revoked_after を更新し、restore は session_revoked_after を維持。環境別 login role は migration で固定名作成せず、release 手順で作成して `GRANT admin_console_write TO <product_admin_login_role>` を実行する
- [x] 1.10 `packages/backend/db/migrations/000006_create_admin_functions.down.sql`
- [x] 1.11 Admin-owned schema migration は `prisma migrate deploy --schema packages/admin/prisma/admin/schema.prisma` で適用。Product 側は既存 golang-migrate で管理
- [x] 1.12 database の環境別 Admin login role 作成手順を release docs / compose init に追加。`GRANT admin_console_write TO <product_admin_login_role>` を実行し、login role は superuser / base table owner にしない [ADMIN-CONSOLE-BE-S044]

## 2. Project Scaffold & Workspace Integration

- [x] 2.1 `packages/admin/package.json` — deps: @prisma/client, jose, bcryptjs, ioredis, @opensearch-project/opensearch, @simplewebauthn/server, @simplewebauthn/browser, zod, @www-template/ui。devDeps: prisma
- [x] 2.2 `packages/admin/svelte.config.js` — adapter-node
- [x] 2.3 `packages/admin/vite.config.ts` — tailwindcss + sveltekit, port 5176
- [x] 2.4 `packages/admin/Dockerfile` — multistage build（builder → Node.js runtime, port 3000）
- [x] 2.4a `.devcontainer/compose.yaml` または deploy compose に `admin` service を追加し、`packages/admin/Dockerfile` build、port 3000、`DATABASE_URL`、`DATABASE_URL`、`VALKEY_URL`、`ADMIN_VALKEY_URL`、`OPENSEARCH_URL`、index prefix、bootstrap env を注入できるようにする。`docker compose up -d admin` が release 手順どおり動くこと
- [x] 2.4b `.devcontainer/compose.yaml` は Product と Admin で同じ `valkey` service / volume を共有し、Product `VALKEY_URL` は DB 0、Admin `ADMIN_VALKEY_URL` は DB 1 のように logical DB 番号だけを分ける。Admin 専用 `admin-valkey` service / volume は作成しない [ADMIN-AUTH-BE-S048]
- [x] 2.5 `packages/admin/tsconfig.json`, `packages/admin/vitest.config.ts`
- [x] 2.5a `packages/admin/.env.example` — 必須環境変数のサンプルのみを置く。secret 実値を含む `packages/admin/.env` は作成・コミットしない
- [x] 2.6 `packages/admin/src/app.d.ts` — App.Locals (`operator: { id, email, role, sessionId, jti } | null`), App.Platform (env vars)
- [x] 2.7 `packages/admin/src/app.css` — tailwind import
- [x] 2.8 `pnpm-workspace.yaml` に `packages/admin` 追加
- [x] 2.9 `tsconfig.base.json` に `@www-template/admin` パスエイリアス追加
- [x] 2.10 `.devcontainer/compose.yaml` に `www-template` DB 作成用 init SQL 追加

## 3. Infrastructure Layer

- [x] 3.1 `packages/admin/src/lib/server/infrastructure/config/env.ts` — JWT secret, ADMIN_ORIGIN, DATABASE_URL, DATABASE_URL, VALKEY_URL, ADMIN_VALKEY_URL, OPENSEARCH_URL, ADMIN_OPENSEARCH_AUDIT_REPLICAS, ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX, PRODUCT_OPENSEARCH_INDEX_PREFIX, ADMIN_BOOTSTRAP_ENABLED, ADMIN_BOOTSTRAP_SECRET_HASH, ADMIN_BOOTSTRAP_EXPIRES_AT の private env 検証。ADMIN_VALKEY_URL は Admin 用 logical DB として必須にし、Product `VALKEY_URL` が存在する場合は同一 Valkey infrastructure かつ異なる DB 番号であることを起動時に検証する。Admin 実装は `admin:*` key prefix のみ読み書きする。OpenSearch は単一接続情報を許容するが、Admin audit prefix と Production domain prefix が同一または包含関係になる設定を拒否する。初期オペレーター用 seed token 環境変数は作らない
- [x] 3.2 `packages/admin/src/lib/server/infrastructure/config/platform.ts` — runtime platform env 検証
- [x] 3.3 `packages/admin/src/lib/server/infrastructure/db/prisma.ts` — `getAdminPrisma()`, `getProductPrisma()`, `disconnectPrisma()`, `validateProductDbRuntimeRole()`。Admin/Product の generated Prisma Client を分離し、環境変数から接続文字列を取得。Product Prisma 初期化時に current role が `admin_console_write` member であり、superuser ではなく、base table owner でもないことを検証して fail-close する [ADMIN-CONSOLE-BE-S044]
- [x] 3.4 `packages/admin/src/lib/server/infrastructure/auth/operator.ts` — `generateChallenge(input, valkey)`, `consumeChallenge(challengeId, expectedType, valkey)`, `createOperatorSession(operator, valkey)`, `revokeOperatorSession(sessionId, valkey)`, `verifyOperatorSession(token, valkey)`, `verifyAssertion(assertion, expectedChallenge, credential, origin, rpId)`, `signOperatorJwt()`, `verifyOperatorJwt()`, `createSessionCookie()`, `clearSessionCookie()`。challenge record は challengeId/type/operatorId/email を Valkey に SETEX/GETDEL で保存・消費し、login 成功時は `admin:session:<sessionId>` を SETEX して JWT claims に sessionId/jti を含める。logout と hook は Valkey active session を検証・失効する
- [x] 3.5 `packages/admin/src/lib/server/infrastructure/rbac/permissions.ts` — `ROLE_PERMISSIONS` 8権限×3ロール
- [x] 3.6 `packages/admin/src/lib/server/infrastructure/rbac/guard.ts` — `hasPermission()`, `requirePermission()`
- [x] 3.7 `packages/admin/src/lib/server/infrastructure/audit/logger.ts` — `createAuditIntent()`, `markAuditSucceeded()`, `markAuditFailed()`。pending intent 永続化後に mutation を開始し、成功後 OpenSearch に非同期インデックス
- [x] 3.8 `packages/admin/src/lib/server/infrastructure/search/opensearch.ts` — `buildAdminAuditIndexName()`, `buildAdminAuditIndexPattern()`, `buildProductDomainIndexPattern()`, `indexAuditEvent()`, `searchAuditEvents()`, `getAuditStats()`。Admin audit は `${ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX}-YYYY.MM` のみ使用し、Production domain search は `${PRODUCT_OPENSEARCH_INDEX_PREFIX}-*` のみ使用する。raw index name、wildcard index pattern、comma-separated multi index、`_all` を route / service / model から渡せない API にする
- [x] 3.9 `packages/admin/src/lib/server/infrastructure/csrf/guard.ts` — `issueCsrfToken(sessionId, jti)`, `validateCsrf()`, `requireSameOrigin()`。ADMIN_ORIGIN + sessionId/jti-bound signed double-submit token を cookie-authenticated non-GET に適用し、pre-auth auth route は Origin allowlist のみ要求

## 4. Models Layer

- [x] 4.1 `packages/admin/src/lib/server/models/types.ts` — Operator, AuditEvent, AccountSummary, PasskeyInfo 型
- [x] 4.2 `packages/admin/src/lib/server/models/operators.ts` — Admin Prisma Client で `findOperatorById()`, `findOperatorByEmail()`, `countOperators()`, `listOperators()`, `createInitialAdminOperator()`, `createOperator()`, `updateOperatorRole()`, `deactivateOperator()`, `updateLoginTimestamp()`
- [x] 4.3 `packages/admin/src/lib/server/models/passkeys.ts` — Admin Prisma Client で `listOperatorPasskeys()`, `addOperatorPasskey()`, `deleteOperatorPasskey()`, `getPasskeyCount()`
- [x] 4.4 `packages/admin/src/lib/server/models/audit-events.ts` — Admin Prisma Client で `insertAuditEvent()`, `listAuditEvents()`
- [x] 4.5 `packages/admin/src/lib/server/models/accounts.ts` — Product Prisma Client で `searchAccounts()`, `getAccountById()`, `suspendAccountProduct()`, `restoreAccountProduct()`。`admin_op` 関数は parameterized `$queryRaw` / `$executeRaw` のみ使用
- [x] 4.6 `packages/admin/src/lib/server/models/schemas.ts` — Zod schemas (login email, suspend reason, search params (limit 1-100, offset >=0), operator create, role update)

## 5. Services Layer

- [x] 5.1 `packages/admin/src/lib/server/services/accounts/search.ts`
- [x] 5.2 `packages/admin/src/lib/server/services/accounts/detail.ts`
- [x] 5.3 `packages/admin/src/lib/server/services/accounts/suspend.ts` — pending audit intent 作成 → database `admin_op.suspend_account` で status + session_revoked_after 更新 → 成功時は audit outcome succeeded 更新。database mutation 失敗時は audit outcome failed + stable error_code + completed_at 更新。intent 作成失敗は mutation 開始前に 503、outcome 更新失敗は pending event を reconciliation 対象に残す。OpenSearch 失敗のみ非ブロッキング。テスト: [ADMIN-CONSOLE-BE-S014] [ADMIN-CONSOLE-BE-S017] [ADMIN-CONSOLE-BE-S052] [ADMIN-CONSOLE-BE-S055] [AUTH-BE-S056]
- [x] 5.4 `packages/admin/src/lib/server/services/accounts/restore.ts` — pending audit intent 作成 → database `admin_op.restore_account` で復旧。session_revoked_after は維持し、過去 session を復活させない
- [x] 5.5 `packages/admin/src/lib/server/services/accounts/stats.ts` — Dashboard 集計
- [x] 5.6 `packages/admin/src/lib/server/services/audit/list.ts` — filter + sort + pagination
- [x] 5.7 `packages/admin/src/lib/server/services/operators/list.ts`
- [x] 5.8 `packages/admin/src/lib/server/services/operators/manage.ts` — 作成時 setup token 生成・bcrypt hash 保存・one-time token 返却 / setup token 再発行 / role 変更 + audit / deactivate + audit。自分自身の無効化、最後の admin 無効化、最後の admin 降格、passkey 登録済みオペレーターの token 再発行を拒否
- [x] 5.9 `packages/admin/src/lib/server/services/operators/bootstrap.ts` — `admin.operators` 0 件 + ADMIN_BOOTSTRAP_ENABLED=true + bootstrap secret hash 検証 + expires_at 未到達時のみ、最初の admin オペレーター作成と passkey 登録を同一 transaction で実行

## 6. Hooks

- [x] 6.1 `packages/admin/src/hooks.server.ts` — `admin_session` cookie → JWT verify → Valkey active session sessionId/jti verify → DB の現在 operator lookup → `event.locals.operator={ id,email,role,sessionId,jti }`。JWT role claim ではなく DB current role で認可。未認証 protected route→/login redirect。pre-auth route（/login, /setup, /operator-setup, /api/admin/auth/passkey/_, /api/admin/auth/setup/_, /api/admin/auth/operator-setup/\_）は login redirect せず通す。ただし `/api/admin/auth/passkeys` と `/api/admin/auth/passkeys/` 配下は passkey 管理 API として route-level で `admin_session` 必須、未認証は 401。認証済み+/login→/ redirect。cookie-authenticated non-GET で Origin + sessionId/jti-bound CSRF 検証。全 Admin HTML / load / BFF response に`Cache-Control: no-store` を付与。テスト: [ADMIN-AUTH-BE-S005] [ADMIN-AUTH-BE-S006] [ADMIN-AUTH-BE-S007] [ADMIN-AUTH-BE-S008] [ADMIN-AUTH-BE-S045] [ADMIN-AUTH-BE-S055] [ADMIN-AUTH-BE-S036] [ADMIN-AUTH-BE-S037] [ADMIN-AUTH-BE-S038] [ADMIN-AUTH-FE-S024] [ADMIN-AUTH-FE-S025] [ADMIN-AUTH-FE-S026] [ADMIN-AUTH-BE-S014]
- [x] 6.2 `packages/admin/src/hooks.client.ts` — client boot

## 7. Auth API Routes

- [x] 7.1 `routes/api/admin/auth/passkey/start/+server.ts` — email から operator 検索 → 登録済み active operator は challengeId/type/operatorId/email を含む real challenge record を generateChallenge。未登録 email / inactive / passkey 未登録 operator は同じ HTTP status / response shape で decoy challenge を保存し、finish は non-revealing 401 にする。テスト: [ADMIN-AUTH-BE-S001] [ADMIN-AUTH-BE-S049]
- [x] 7.2 `routes/api/admin/auth/passkey/finish/+server.ts` — challengeId を受け取り consumeChallenge → type/operator binding 検証 → verifyAssertion → createOperatorSession(sessionId/jti) → signOperatorJwt → Set-Cookie → 303 /。テスト: [ADMIN-AUTH-BE-S001] [ADMIN-AUTH-BE-S002] [ADMIN-AUTH-BE-S003] [ADMIN-AUTH-BE-S004] [ADMIN-AUTH-BE-S034]
- [x] 7.3 `routes/api/admin/auth/passkeys/+server.ts` (GET 一覧)
- [x] 7.4 `routes/api/admin/auth/passkeys/start/+server.ts`
- [x] 7.5 `routes/api/admin/auth/passkeys/finish/+server.ts`
- [x] 7.6 `routes/api/admin/auth/passkeys/[id]/+server.ts` (DELETE, 残り1件は拒否). テスト: [ADMIN-AUTH-BE-S012] [ADMIN-AUTH-BE-S013]
- [x] 7.7 `routes/api/admin/auth/setup/start/+server.ts` — bootstrap pre-auth rate limit / lock → `admin.operators` 0 件確認 → ADMIN_BOOTSTRAP_ENABLED / bootstrap secret / expiry 検証 → email/display_name 検証 → bootstrap challenge。Valkey unavailable は secret 検証前に 503 fail-close。テスト: [ADMIN-AUTH-BE-S019] [ADMIN-AUTH-BE-S020] [ADMIN-AUTH-BE-S023] [ADMIN-AUTH-BE-S046] [ADMIN-AUTH-BE-S050] [ADMIN-AUTH-BE-S052]
- [x] 7.8 `routes/api/admin/auth/setup/finish/+server.ts` — transaction 内で operators count 再確認 → WebAuthn attestation → role=admin オペレーター作成 → passkey 登録 → JWT cookie。テスト: [ADMIN-AUTH-BE-S019] [ADMIN-AUTH-BE-S021] [ADMIN-AUTH-BE-S022] [ADMIN-AUTH-BE-S023]
- [x] 7.9 `routes/api/admin/auth/operator-setup/start/+server.ts` — IP + token fingerprint pre-auth rate limit / lock → setup token bcrypt 検証 → challenge。Valkey unavailable は token 検証前に 503 fail-close。テスト: [ADMIN-AUTH-BE-S040] [ADMIN-AUTH-BE-S041] [ADMIN-AUTH-BE-S042] [ADMIN-AUTH-BE-S043] [ADMIN-AUTH-BE-S051] [ADMIN-AUTH-BE-S052]
- [x] 7.10 `routes/api/admin/auth/operator-setup/finish/+server.ts` — WebAuthn attestation → passkey 登録 → setup_token 消費 → JWT cookie。テスト: [ADMIN-AUTH-BE-S040] [ADMIN-AUTH-BE-S044]

## 8. Auth Pages (View + Controller)

- [x] 8.1 `routes/login/+page.server.ts` — actions: start, finish。テスト: [ADMIN-AUTH-FE-S001] [ADMIN-AUTH-FE-S002] [ADMIN-AUTH-FE-S003] [ADMIN-AUTH-FE-S004] [ADMIN-AUTH-FE-S005]
- [x] 8.2 `routes/login/+page.svelte` — email 入力 + WebAuthn UI + loading state
- [x] 8.3 `routes/setup/+page.server.ts` — `admin.operators` 0 件時のみ初回起動セットアップを許可し、1 件以上なら `/login` redirect または 403
- [x] 8.4 `routes/setup/+page.svelte` — 最初の admin オペレーター用 email/display_name 入力 + WebAuthn passkey 登録 UI。テスト: [ADMIN-AUTH-FE-S017] [ADMIN-AUTH-FE-S018]
- [x] 8.5 `routes/operator-setup/+page.server.ts` — one-time setup token による追加オペレーター登録 flow
- [x] 8.6 `routes/operator-setup/+page.svelte` — setup token 入力 + WebAuthn passkey 登録 UI。テスト: [ADMIN-AUTH-FE-S019] [ADMIN-AUTH-FE-S020] [ADMIN-AUTH-FE-S021]
- [x] 8.7 `routes/passkeys/+page.server.ts` — load passkey list
- [x] 8.8 `routes/passkeys/+page.svelte` — passkey 一覧 + 追加/削除 UI。テスト: [ADMIN-AUTH-FE-S012] [ADMIN-AUTH-FE-S013] [ADMIN-AUTH-FE-S014] [ADMIN-AUTH-FE-S015] [ADMIN-AUTH-FE-S016]

## 9. Console Routes (View + Controller)

- [x] 9.1 `routes/+layout.server.ts` — operator から nav context 生成
- [x] 9.2 `routes/+layout.svelte` — AdminShell (Sidebar + Header + slot)
- [x] 9.3 `routes/+page.server.ts` — Dashboard stats + recent audit
- [x] 9.4 `routes/+page.svelte` — KPI cards + audit table。テスト: [ADMIN-CONSOLE-FE-S032] [ADMIN-CONSOLE-FE-S033]
- [x] 9.5 `routes/accounts/+page.server.ts` — search load (query/status/page)
- [x] 9.6 `routes/accounts/+page.svelte` — search + filter + DataTable + pagination
- [x] 9.7 `routes/accounts/[id]/+page.server.ts` — detail load + suspend/restore actions。suspend に `requirePermission('accounts:suspend')`。テスト: [ADMIN-CONSOLE-FE-S010]
- [x] 9.8 `routes/accounts/[id]/+page.svelte` — detail + PasskeyList + SuspendDialog + RestoreDialog
- [x] 9.9 `routes/audit/+page.server.ts` — list load (operator/action/date filter)
- [x] 9.10 `routes/audit/+page.svelte` — AuditLogTable + AuditFilterBar
- [x] 9.11 `routes/settings/+page.server.ts` — Settings landing load。admin role check
- [x] 9.12 `routes/settings/+page.svelte` — Settings landing。オペレーター管理への導線
- [x] 9.13 `routes/settings/operators/+page.server.ts` — list load + create/update/deactivate/setup-token-rotate actions。admin role check。create/rotate は one-time setup token を返す
- [x] 9.14 `routes/settings/operators/+page.svelte` — OperatorTable + 追加 Dialog + one-time setup token 表示 + passkey 未登録オペレーターの token 再発行 UI
- [x] 9.15 `routes/api/admin/auth/logout/+server.ts` — POST: revoke `admin:session:<sessionId>` in Valkey, clear admin_session/admin_csrf cookies, redirect /login。テスト: [ADMIN-AUTH-BE-S054] [ADMIN-CONSOLE-FE-S031]

## 10. View Components

- [x] 10.1 `lib/components/layout/AdminShell.svelte`
- [x] 10.2 `lib/components/layout/AdminSidebar.svelte` — role-based nav links。テスト: [ADMIN-CONSOLE-FE-S028] [ADMIN-CONSOLE-FE-S029]
- [x] 10.3 `lib/components/layout/AdminHeader.svelte` — operator 名 + logout。テスト: [ADMIN-CONSOLE-FE-S030] [ADMIN-CONSOLE-FE-S031]
- [x] 10.4 `lib/components/accounts/AccountTable.svelte`
- [x] 10.5 `lib/components/accounts/AccountStatusBadge.svelte`
- [x] 10.6 `lib/components/accounts/PasskeyList.svelte`
- [x] 10.7 `lib/components/audit/AuditLogTable.svelte`
- [x] 10.8 `lib/components/audit/AuditFilterBar.svelte`
- [x] 10.9 `lib/components/operators/OperatorTable.svelte`
- [x] 10.10 `lib/components/operators/OperatorRoleBadge.svelte`
- [x] 10.11 `lib/components/shared/ConfirmDialog.svelte`
- [x] 10.12 `lib/components/shared/DataTable.svelte`
- [x] 10.13 `lib/components/shared/EmptyState.svelte`。テスト: [ADMIN-CONSOLE-FE-S002] [ADMIN-CONSOLE-FE-S019]

## 11. ESLint Boundaries & Root Integration

- [x] 11.1 `eslint.config.js` — admin element types 追加 (admin-controller, admin-service, admin-model, admin-infrastructure, admin-view, admin-route-view, admin-hooks)
- [x] 11.2 `eslint.config.js` — admin MVCS boundary rules (層間 import 制限)。テスト: [ADMIN-CONSOLE-BE-S027] [ADMIN-CONSOLE-BE-S028]
- [x] 11.3 `eslint.config.js` — admin no-restricted-imports (顧客向けパッケージ禁止)。テスト: [ADMIN-CONSOLE-BE-S029]
- [x] 11.4 `eslint.config.js` — admin security rules (DB 接続文字列ハードコード禁止, SQL テンプレートリテラル禁止, Prisma `$queryRawUnsafe` / `$executeRawUnsafe` 禁止, @html 禁止)。テスト: [ADMIN-CONSOLE-BE-S030] [ADMIN-CONSOLE-BE-S031] [ADMIN-CONSOLE-BE-S032] [ADMIN-CONSOLE-BE-S051]
- [x] 11.5 `eslint.config.js` — `/api/admin/*` は `packages/admin/src/routes/api/admin/**` のみ許可し、Go backend / TypeSpec / frontend generated client からの Admin BFF 利用を禁止
- [x] 11.5a `AGENTS.md` — API path policy に `packages/admin/src/routes/api/admin/**` 限定の package-local Admin BFF 例外を明記
- [x] 11.6 `package.json` (root) — dev:admin, build:admin, test:admin, prisma:admin:generate, prisma:admin:migrate:deploy, prisma:admin:product:generate, db:migrate:product
- [x] 11.7 `package.json` (root) — dev:all, check, test:run, lint に Admin を追加
- [x] 11.8 `vitest.config.ts` (root) — `frontend-admin` project 追加

## 12. Tests — Infrastructure Layer

- [x] 12.1 UT: infrastructure/auth — JWT sign/verify roundtrip [ADMIN-AUTH-BE-S003]
- [x] 12.2 UT: infrastructure/auth — 期限切れ JWT 拒否 [ADMIN-AUTH-BE-S006]
- [x] 12.3 UT: infrastructure/auth — userVerification required in options [ADMIN-AUTH-BE-S016]
- [x] 12.4 UT: infrastructure/auth — UV false assertion 拒否 [ADMIN-AUTH-BE-S017]
- [x] 12.5 UT: infrastructure/auth — UV true assertion 受理 [ADMIN-AUTH-BE-S018]
- [x] 12.6 UT: infrastructure/auth — sign_count 減少 assertion 拒否 [ADMIN-AUTH-BE-S025]
- [x] 12.7 UT: infrastructure/auth — 保存 credential で検証成功 [ADMIN-AUTH-BE-S024]
- [x] 12.8 UT: infrastructure/auth — production で Secure cookie 属性 [ADMIN-AUTH-BE-S031]
- [x] 12.9 UT: infrastructure/auth — cookie に Path=/ [ADMIN-AUTH-BE-S032]
- [x] 12.10 UT: infrastructure/auth — consumed challenge reuse 拒否 [ADMIN-AUTH-BE-S053]
- [x] 12.10a UT: infrastructure/auth — logout revokes Valkey active session and stolen cookie is rejected [ADMIN-AUTH-BE-S054]
- [x] 12.10b UT: infrastructure/auth — JWT sessionId/jti mismatch or missing Valkey session is rejected [ADMIN-AUTH-BE-S055]
- [x] 12.11 UT: infrastructure/auth — Admin Valkey unavailable 503 fail-close [ADMIN-AUTH-BE-S033]
- [x] 12.11a UT: infrastructure/auth/config — ADMIN_VALKEY_URL 未設定、Admin Valkey URL の DB 番号未指定、Product `VALKEY_URL` と異なる endpoint、または同一 DB 番号の場合は起動拒否する [ADMIN-AUTH-BE-S048]
- [x] 12.12 UT: infrastructure/auth — challengeId と Operator binding 不一致拒否 [ADMIN-AUTH-BE-S034]
- [x] 12.12a UT: hooks — JWT role claim ではなく DB current role を使用 [ADMIN-AUTH-BE-S045]
- [x] 12.12b UT: infrastructure/auth — 未登録 email / inactive operator の login start は decoy challenge と同一 response shape を返す [ADMIN-AUTH-BE-S049]
- [x] 12.12c UT: infrastructure/auth — bootstrap secret と operator setup token の pre-auth rate limit / lock / Valkey fail-close [ADMIN-AUTH-BE-S050] [ADMIN-AUTH-BE-S051] [ADMIN-AUTH-BE-S052]
- [x] 12.13 UT: infrastructure/csrf — valid Origin + CSRF token 許可 [ADMIN-AUTH-BE-S036]
- [x] 12.14 UT: infrastructure/csrf — cross-origin mutation 403 [ADMIN-AUTH-BE-S037]
- [x] 12.15 UT: infrastructure/csrf — CSRF token mismatch 403 [ADMIN-AUTH-BE-S038]
- [x] 12.15b UT: infrastructure/csrf — CSRF token signed for different sessionId/jti is rejected [ADMIN-AUTH-BE-S038]
- [x] 12.15a UT: infrastructure/csrf — pre-auth passkey start は session-bound CSRF 不要 [ADMIN-AUTH-BE-S047]
- [x] 12.16 UT: infrastructure/rbac — admin 全権限 true [ADMIN-CONSOLE-BE-S033]
- [x] 12.17 UT: infrastructure/rbac — viewer 書き込み権限 false [ADMIN-CONSOLE-BE-S034]
- [x] 12.18 UT: infrastructure/rbac — requirePermission が 403 throw [ADMIN-CONSOLE-BE-S035]
- [x] 12.19 UT: infrastructure/rbac — 未定義権限 false [ADMIN-CONSOLE-BE-S036]
- [x] 12.20 UT: infrastructure/search — indexAuditEvent calls OpenSearch [ADMIN-CONSOLE-BE-S039]
- [x] 12.21 UT: infrastructure/search — OpenSearch failure logs warn, no throw [ADMIN-CONSOLE-BE-S040]
- [x] 12.22 UT: infrastructure/search/config — Admin audit prefix と Production domain prefix が同一または包含関係の場合は起動拒否 [ADMIN-CONSOLE-BE-S053]
- [x] 12.23 UT: infrastructure/search — raw index name / `_all` / comma-separated multi index / cross namespace query を拒否 [ADMIN-CONSOLE-BE-S054]
- [x] 12.24 UT: infrastructure/db — database runtime role validation rejects non-member / superuser / base table owner before queries run [ADMIN-CONSOLE-BE-S044]

## 13. Tests — Models Layer

- [x] 13.1 UT: models/operators — last_login_at 更新 [ADMIN-AUTH-BE-S009]
- [x] 13.2 UT: models/passkeys — 認証後 signCount 更新 [ADMIN-AUTH-BE-S026]
- [x] 13.3 UT: models/passkeys — sign_count default 0 [ADMIN-CONSOLE-BE-S006]
- [x] 13.4 UT: models/operators — cascade delete passkeys [ADMIN-CONSOLE-BE-S002]
- [x] 13.5 UT: models/operators — email UNIQUE 制約 [ADMIN-CONSOLE-BE-S003]
- [x] 13.6 UT: models/operators — role CHECK 制約 [ADMIN-CONSOLE-BE-S004]
- [x] 13.7 UT: models/accounts — suspend non-active throws [ADMIN-CONSOLE-BE-S009]
- [x] 13.8 UT: models/accounts — invalid limit ZodError [ADMIN-CONSOLE-BE-S024]
- [x] 13.9 UT: models/accounts — negative offset ZodError [ADMIN-CONSOLE-BE-S025]
- [x] 13.10 UT: models/accounts — SQL injection prevented [ADMIN-CONSOLE-BE-S026]
- [x] 13.11 UT: models/accounts — search with pagination
- [x] 13.12 UT: Prisma migration — 全テーブル作成確認 [ADMIN-CONSOLE-BE-S001]
- [x] 13.13 UT: Prisma migration — 初期オペレーターを seed しない [ADMIN-CONSOLE-BE-S005]
- [x] 13.14 UT: db migrations — SECURITY DEFINER search_path fixed [ADMIN-CONSOLE-BE-S037]
- [x] 13.15 UT: db migrations — PUBLIC execute revoked [ADMIN-CONSOLE-BE-S038]
- [x] 13.16 UT: db migrations — admin_console_read grants [ADMIN-CONSOLE-BE-S042]
- [x] 13.17 UT: db migrations — admin_console_write grants [ADMIN-CONSOLE-BE-S043]

## 14. Tests — Services Layer

- [x] 14.1 UT: services/accounts — suspend が audit 記録 [ADMIN-CONSOLE-BE-S014]
- [x] 14.2 UT: services/accounts — restore が audit 記録 [ADMIN-CONSOLE-BE-S015]
- [x] 14.3 UT: services/accounts — 二重 suspend エラー、status 変化なし
- [x] 14.4 UT: services/accounts — suspend→restore 正常サイクル
- [x] 14.5 UT: services/operators — role 更新が audit 記録 [ADMIN-CONSOLE-BE-S016]
- [x] 14.6 UT: services/\* — pending audit intent 作成失敗時は database mutation を開始せず 503 [ADMIN-CONSOLE-BE-S017]
- [x] 14.7 UT: services/accounts — suspend が database session_revoked_after を書く [AUTH-BE-S056]
- [x] 14.7a UT: services/accounts — outcome 更新失敗時は pending audit event を残して metric/log 出力 [ADMIN-CONSOLE-BE-S052]
- [x] 14.7b UT: services/accounts — database mutation 失敗時は audit outcome failed / error_code / completed_at を記録し、failed 更新失敗時は pending を reconciliation 対象に残す [ADMIN-CONSOLE-BE-S055]
- [x] 14.8 UT: services/operators — createOperator が one-time setup token を返し hash のみ保存 [ADMIN-CONSOLE-BE-S045]
- [x] 14.9 UT: services/operators — setup token 再発行が旧 token を無効化して audit 記録 [ADMIN-CONSOLE-BE-S046]
- [x] 14.10 UT: services/operators — passkey 登録済みオペレーターの token 再発行拒否 [ADMIN-CONSOLE-BE-S047]
- [x] 14.11 UT: services/operators — 最後の admin 無効化拒否 [ADMIN-CONSOLE-BE-S048]
- [x] 14.12 UT: services/operators — 最後の admin 降格拒否 [ADMIN-CONSOLE-BE-S049]

## 15. Tests — Auth Routes + Hooks

- [x] 15.0a UT: login page completes passkey flow with mocked WebAuthn and redirects [ADMIN-AUTH-FE-S001]
- [x] 15.0b UT: login page shows non-revealing error for unknown email with mocked WebAuthn/API [ADMIN-AUTH-FE-S002]
- [x] 15.0c UT: login page handles WebAuthn cancel without setting cookie [ADMIN-AUTH-FE-S003]
- [x] 15.0d UT: login page handles no available credential with non-revealing error [ADMIN-AUTH-FE-S004]
- [x] 15.1 IT: passkey login sets cookie [ADMIN-AUTH-BE-S001]
- [x] 15.2 IT: invalid assertion 401 [ADMIN-AUTH-BE-S002]
- [x] 15.3 IT: unknown credential_handle 401 [ADMIN-AUTH-BE-S003]
- [x] 15.4 IT: expired challenge 401 [ADMIN-AUTH-BE-S004]
- [x] 15.5 IT: consumed challenge reuse 401 [ADMIN-AUTH-BE-S053]
- [x] 15.5a IT: unknown admin email start is enumeration-safe and finish remains non-revealing [ADMIN-AUTH-BE-S049]
- [x] 15.6 IT: hooks verifies cookie [ADMIN-AUTH-BE-S005]
- [x] 15.7 IT: expired cookie redirects login [ADMIN-AUTH-BE-S006]
- [x] 15.8 IT: tampered JWT redirects login [ADMIN-AUTH-BE-S007]
- [x] 15.9 IT: inactive operator rejected [ADMIN-AUTH-BE-S008]
- [x] 15.10 IT: list passkeys [ADMIN-AUTH-BE-S010]
- [x] 15.11 IT: add passkey [ADMIN-AUTH-BE-S011]
- [x] 15.12 IT: delete last passkey 400 [ADMIN-AUTH-BE-S012]
- [x] 15.13 IT: delete passkey ok [ADMIN-AUTH-BE-S013]
- [x] 15.14 IT: unauth passkey API 401 [ADMIN-AUTH-BE-S014]
- [x] 15.15 IT: cross-operator passkey 403 [ADMIN-AUTH-BE-S015]
- [x] 15.16 IT: initial setup creates first admin passkey [ADMIN-AUTH-BE-S019]
- [x] 15.17 IT: initial setup start blocked when operators exist [ADMIN-AUTH-BE-S020]
- [x] 15.18 IT: initial setup finish rechecks zero-operator condition [ADMIN-AUTH-BE-S021]
- [x] 15.19 IT: initial setup always creates role=admin [ADMIN-AUTH-BE-S022]
- [x] 15.20 IT: initial setup rejects invalid or expired bootstrap secret [ADMIN-AUTH-BE-S023]
- [x] 15.20f IT: initial setup rejects disabled bootstrap flag [ADMIN-AUTH-BE-S046]
- [x] 15.20g IT: initial setup bootstrap brute-force is rate limited and Valkey unavailable fails closed [ADMIN-AUTH-BE-S050] [ADMIN-AUTH-BE-S052]
- [x] 15.20a IT: operator setup token registers passkey [ADMIN-AUTH-BE-S040]
- [x] 15.20b IT: bad operator setup token rejected [ADMIN-AUTH-BE-S041]
- [x] 15.20c IT: expired operator setup token rejected [ADMIN-AUTH-BE-S042]
- [x] 15.20d IT: consumed operator setup token rejected [ADMIN-AUTH-BE-S043]
- [x] 15.20e IT: operator setup blocked for registered op [ADMIN-AUTH-BE-S044]
- [x] 15.20h IT: operator setup token brute-force is rate limited by IP and token fingerprint [ADMIN-AUTH-BE-S051]
- [x] 15.21 IT: duplicate credential_handle 409 [ADMIN-AUTH-BE-S027]
- [x] 15.22 IT: throttle start 429 [ADMIN-AUTH-BE-S028]
- [x] 15.23 IT: finish lock 429 [ADMIN-AUTH-BE-S029]
- [x] 15.23a UT: temporary lock TTL expiry allows retry with fake clock [ADMIN-AUTH-BE-S030]
- [x] 15.24 IT: Valkey unavailable 503 [ADMIN-AUTH-BE-S033]
- [x] 15.25 IT: challenge binding mismatch 401 [ADMIN-AUTH-BE-S034]
- [x] 15.26 IT: valid CSRF mutation allowed [ADMIN-AUTH-BE-S036]
- [x] 15.27 IT: cross-origin mutation 403 [ADMIN-AUTH-BE-S037]
- [x] 15.28 IT: CSRF token mismatch 403 [ADMIN-AUTH-BE-S038]
- [x] 15.28a IT: pre-auth passkey start works without session-bound CSRF when Origin is valid [ADMIN-AUTH-BE-S047]
- [x] 15.29 UT: unauth redirect /login [ADMIN-AUTH-FE-S006]
- [x] 15.30 UT: authed /login→/ [ADMIN-AUTH-FE-S007]
- [x] 15.31 UT: redirectTo preserved [ADMIN-AUTH-FE-S008]
- [x] 15.31a UT: pre-auth routes bypass login redirect while protected routes redirect [ADMIN-AUTH-FE-S026]
- [x] 15.32 UT: expired session clears cookie [ADMIN-AUTH-FE-S009]
- [x] 15.33 UT: tampered JWT clears cookie [ADMIN-AUTH-FE-S010]
- [x] 15.34 UT: /login no-store header [ADMIN-AUTH-FE-S011]
- [x] 15.35 UT: /setup and /operator-setup no-store header [ADMIN-AUTH-FE-S022]
- [x] 15.36 UT: authenticated Admin pages no-store header [ADMIN-AUTH-FE-S024]
- [x] 15.37 UT: `/api/admin/*` BFF responses no-store header [ADMIN-AUTH-FE-S025]

## 16. Tests — Console Routes + Components

- [x] 16.1 UT: passkey list rendered [ADMIN-AUTH-FE-S012]
- [x] 16.1a UT: add passkey flow with mocked WebAuthn appends new passkey [ADMIN-AUTH-FE-S013]
- [x] 16.2 UT: delete button disabled for 1 cred [ADMIN-AUTH-FE-S014]
- [x] 16.3 UT: delete button enabled for 2+ creds [ADMIN-AUTH-FE-S015]
- [x] 16.3a UT: WebAuthn registration cancel leaves passkey list unchanged [ADMIN-AUTH-FE-S016]
- [x] 16.4 UT: setup registers passkey [ADMIN-AUTH-FE-S017]
- [x] 16.5 UT: existing operator blocks first setup [ADMIN-AUTH-FE-S018]
- [x] 16.6 UT: operator setup token registers added operator [ADMIN-AUTH-FE-S019]
- [x] 16.6b UT: bad operator setup token shows non-revealing error [ADMIN-AUTH-FE-S020]
- [x] 16.6c UT: registered operator cannot access operator setup [ADMIN-AUTH-FE-S021]
- [x] 16.6a UT: disabled bootstrap gate hides /setup form [ADMIN-AUTH-FE-S023]
- [x] 16.7 UT: loading state during login [ADMIN-AUTH-FE-S005]
- [x] 16.8 UT: search filters by email [ADMIN-CONSOLE-FE-S001]
- [x] 16.9 UT: empty search shows empty state [ADMIN-CONSOLE-FE-S002]
- [x] 16.9a UT: status filter shows only selected status [ADMIN-CONSOLE-FE-S003]
- [x] 16.10 UT: pagination shows pages [ADMIN-CONSOLE-FE-S004]
- [x] 16.10a UT: search loading indicator is shown while request is pending [ADMIN-CONSOLE-FE-S005]
- [x] 16.10b UT: search error message is shown and stale table is preserved [ADMIN-CONSOLE-FE-S006]
- [x] 16.11 UT: account detail displayed [ADMIN-CONSOLE-FE-S007]
- [x] 16.12 UT: invalid id shows 404 [ADMIN-CONSOLE-FE-S008]
- [x] 16.13 UT: 0 passkeys shows empty [ADMIN-CONSOLE-FE-S009]
- [x] 16.13a UT: active account suspend flow updates status and success message [ADMIN-CONSOLE-FE-S010]
- [x] 16.14 UT: empty reason rejected [ADMIN-CONSOLE-FE-S011]
- [x] 16.14a UT: suspend confirmation cancel leaves account unchanged [ADMIN-CONSOLE-FE-S012]
- [x] 16.15 UT: suspended shows no suspend btn [ADMIN-CONSOLE-FE-S013]
- [x] 16.15a UT: restore suspended account success message and active status [ADMIN-CONSOLE-FE-S014]
- [x] 16.16 UT: active shows no restore btn [ADMIN-CONSOLE-FE-S015]
- [x] 16.17 UT: audit log rendered [ADMIN-CONSOLE-FE-S016]
- [x] 16.18 UT: audit filter by action [ADMIN-CONSOLE-FE-S017]
- [x] 16.19 UT: details JSON expand [ADMIN-CONSOLE-FE-S018]
- [x] 16.19a UT: audit empty state rendered [ADMIN-CONSOLE-FE-S019]
- [x] 16.20 UT: operator list displayed [ADMIN-CONSOLE-FE-S020]
- [x] 16.21 UT: non-admin access 403 [ADMIN-CONSOLE-FE-S021]
- [x] 16.22 UT: add operator success [ADMIN-CONSOLE-FE-S022]
- [x] 16.22a UT: one-time setup token copy UI [ADMIN-CONSOLE-FE-S036]
- [x] 16.22b UT: setup token rotate UI for unregistered Operator [ADMIN-CONSOLE-FE-S037]
- [x] 16.23 UT: dup email shows error [ADMIN-CONSOLE-FE-S023]
- [x] 16.24 UT: role update success [ADMIN-CONSOLE-FE-S024]
- [x] 16.25 UT: deactivate success [ADMIN-CONSOLE-FE-S025]
- [x] 16.26 UT: self deactivate blocked [ADMIN-CONSOLE-FE-S026]
- [x] 16.26a UT: sidebar navigation changes route and highlights active link [ADMIN-CONSOLE-FE-S027]
- [x] 16.27 UT: admin sees operators link [ADMIN-CONSOLE-FE-S028]
- [x] 16.28 UT: operator no operators link [ADMIN-CONSOLE-FE-S029]
- [x] 16.29 UT: header shows operator name [ADMIN-CONSOLE-FE-S030]
- [x] 16.30 UT: dashboard shows KPIs [ADMIN-CONSOLE-FE-S032]
- [x] 16.31 UT: dashboard shows recent audit [ADMIN-CONSOLE-FE-S033]
- [x] 16.32 UT: logout clears cookie and redirects /login [ADMIN-CONSOLE-FE-S031]

## 17. Tests — DB Integration

- [x] 17.1 IT: migration status default active [ADMIN-CONSOLE-BE-S007]
- [x] 17.2 IT: suspend_account function [ADMIN-CONSOLE-BE-S008]
- [x] 17.3 IT: restore_account function [ADMIN-CONSOLE-BE-S010]
- [x] 17.4 IT: restore non-suspended throws [ADMIN-CONSOLE-BE-S011]
- [x] 17.5 IT: view returns all accounts [ADMIN-CONSOLE-BE-S012]
- [x] 17.6 IT: view returns passkey info [ADMIN-CONSOLE-BE-S013]
- [x] 17.7 IT: admin DB query works [ADMIN-CONSOLE-BE-S018]
- [x] 17.8 IT: product DB query works [ADMIN-CONSOLE-BE-S019]
- [x] 17.9 IT: DB connection failure throws [ADMIN-CONSOLE-BE-S020]
- [x] 17.10 IT: migration applies unapplied [ADMIN-CONSOLE-BE-S021]
- [x] 17.10a IT: database extension migrations are applied by golang-migrate, not SQL migrations [ADMIN-CONSOLE-BE-S023]
- [x] 17.11 IT: migration skips applied [ADMIN-CONSOLE-BE-S022]
- [x] 17.12 IT: invalid limit 400 [ADMIN-CONSOLE-BE-S024]
- [x] 17.13 IT: negative offset 400 [ADMIN-CONSOLE-BE-S025]
- [x] 17.14 IT: OpenSearch index audit event [ADMIN-CONSOLE-BE-S039]
- [x] 17.15 IT: OpenSearch fail, DB fallback search [ADMIN-CONSOLE-BE-S041]
- [x] 17.15a IT: Admin audit write/search uses only Admin audit namespace and never touches Production domain namespace [ADMIN-CONSOLE-BE-S053]
- [x] 17.15b IT: Production domain OpenSearch use case uses only Production domain namespace and never touches Admin audit namespace [ADMIN-CONSOLE-BE-S054]
- [x] 17.16 IT: SECURITY DEFINER search_path fixed and Product base table references are schema-qualified [ADMIN-CONSOLE-BE-S037]
- [x] 17.17 IT: PUBLIC execute revoked [ADMIN-CONSOLE-BE-S038]
- [x] 17.18 IT: admin_console_read / admin_console_write grants [ADMIN-CONSOLE-BE-S042] [ADMIN-CONSOLE-BE-S043]
- [x] 17.18a IT: getProductPrisma startup validation confirms current role membership and rejects superuser/base table owner [ADMIN-CONSOLE-BE-S044]
- [x] 17.19 IT: database に SQL migrations を適用する script が存在しない [ADMIN-CONSOLE-BE-S050]

## 18. Product Auth Integration

- [x] 18.1 `packages/typespec/src/models/auth.tsp` と `packages/typespec/src/routes/v1/auth.tsp` — existing `AuthFailureClassification` に `account-suspended` を追加し、`AuthFailureResponse` で返す。`POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/refresh`、bearer-protected `/api/v1/*` endpoint に HTTP 403 + `Cache-Control: no-store` + `{ requestId, error: "account-suspended" }` response を追加し、`AuthOperationErrorResponse` では返さない。`packages/typespec/main.tsp` は import entrypoint として整合を確認する [AUTH-BE-S054] [AUTH-BE-S055] [AUTH-BE-S058] [AUTH-BE-S059]
- [x] 18.2 `pnpm gen` / `pnpm check:codegen` — generated OpenAPI / frontend SDK / Go bindings を更新し drift を残さない
- [x] 18.3 `packages/backend/internal/auth/application` — login finish で valid assertion 後に `accounts.status='suspended'` を拒否し、access token / refresh token を発行しない [AUTH-BE-S054]
- [x] 18.4 `packages/backend/internal/auth/application` — refresh で `accounts.status='suspended'` と `session_revoked_after` を検証し、suspended 時は refresh token rotation と新 token pair 発行を行わない [AUTH-BE-S058] [AUTH-BE-S056]
- [x] 18.5 `packages/backend/internal/adapters/http` / auth middleware — `/api/v1/*` bearer 認可時に account status と `session_revoked_after` を検証し、suspended 時は HTTP 403 `AuthFailureResponse` を返す [AUTH-BE-S055] [AUTH-BE-S056] [AUTH-BE-S059]
- [x] 18.6 `packages/backend/internal/adapters/persistence/postgres` — auth lookup に account status と session_revoked_after 読み取りを追加。Admin 専用 package へ依存しない
- [x] 18.7 backend tests — suspended account login rejected after valid passkey assertion [AUTH-BE-S054]
- [x] 18.8 backend tests — suspended existing bearer access token rejected with HTTP 403 `AuthFailureResponse` [AUTH-BE-S055] [AUTH-BE-S059]
- [x] 18.9 backend tests — session_revoked_after rejects old sessions [AUTH-BE-S056]
- [x] 18.10 backend tests — restored account cannot reuse pre-suspend session [AUTH-BE-S057]
- [x] 18.10a backend tests — suspended refresh rejects rotation and returns HTTP 403 `AuthFailureResponse` [AUTH-BE-S058] [AUTH-BE-S059]
- [x] 18.11 `packages/frontend/domain/**` — generated SDK の HTTP 403 `AuthFailureResponse(error="account-suspended")` を domain auth state に接続し、該当 account の access token / refresh token state を消去 [AUTH-FE-S041] [AUTH-FE-S042]
- [x] 18.12 `packages/frontend/app/**` — account suspended 案内 UI とサポート導線を追加。public login start では suspended 状態を表示しない [AUTH-FE-S041] [AUTH-FE-S043]
- [x] 18.13 frontend tests — suspended passkey login は session 保存せず案内表示 [AUTH-FE-S041]
- [x] 18.14 frontend tests — protected API の `account-suspended` で該当 session を消去 [AUTH-FE-S042]
- [x] 18.15 frontend tests — login start / recovery request では suspended 状態を推測不可 [AUTH-FE-S043]
- [x] 18.16 frontend tests — multi-session では suspended account の session のみ削除 [AUTH-FE-S044]

## 19. Tests — ESLint Boundaries + Security

- [x] 19.1 UT: hardcoded conn string lint error [ADMIN-CONSOLE-BE-S030]
- [x] 19.2 UT: @html lint error [ADMIN-CONSOLE-BE-S031]
- [x] 19.3 UT: SQL template literal lint error [ADMIN-CONSOLE-BE-S032]
- [x] 19.3a UT: Prisma unsafe raw query lint error [ADMIN-CONSOLE-BE-S051]
- [x] 19.4 UT: Model→services import lint error [ADMIN-CONSOLE-BE-S027]
- [x] 19.5 UT: Service→components import lint error [ADMIN-CONSOLE-BE-S028]
- [x] 19.6 UT: admin→@www-template/api import lint error [ADMIN-CONSOLE-BE-S029]
- [x] 19.7 UT: View→Model import lint error [ADMIN-CONSOLE-FE-S034]
- [x] 19.8 UT: admin→api import lint error [ADMIN-CONSOLE-FE-S035]
