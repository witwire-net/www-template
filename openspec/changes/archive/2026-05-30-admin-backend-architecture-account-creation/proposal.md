## Why

現在の Admin Console は、Admin と Product Backend の境界を物理的に分けようとしている一方で、顧客アカウント操作、認証、監査の責務が複数の場所へ分散している。Admin 経由のアカウント作成と Product 経由のアカウント利用が別々の規則を持つと、Account の不変条件、停止時の session revoke、認証可否、監査結果が分岐し、顧客と運営者の双方にとって安全性と保守性が低下する。

同時に、Admin authentication と Product authentication は同じ認証ドメインではない。Product authentication は Product account auth ドメインであり、account login、account session、account status / suspension、account accessToken claims、account refreshToken session state、account 固有 validation を所有する。Admin authentication は Admin operator auth ドメインであり、operator login、operator session、operator role / active state、operator accessToken claims、operator refreshToken session state、Admin CSRF / RBAC 連携、operator 固有 validation を所有する。両者を `identityDomain=account|operator` のような切替引数で単一 service に押し込める設計は、ドメイン境界を隠すため採用しない。

Admin は強権限 surface であるため、Product 側へ Admin 機能を露出させず、Account ドメインの source of truth は Go backend の `packages/backend/internal/domain` に集約する。Admin frontend は静的 client として同一 Admin ドメイン上の Admin backend `/api/v1/*` を呼び出し、Product と Admin は別ドメイン、別 binary、別 OpenAPI、別 SDK、別 Go bindings で分離する。

## What Changes

- **BREAKING** Admin Console は package-local BFF / server-side domain logic を持たず、`packages/admin/app -> packages/admin/domain -> packages/admin/api` の依存方向を持つ静的クライアントになる。
- **BREAKING** Admin backend 機能は `packages/admin` から削除され、Go backend の Admin 専用バイナリ・Admin 専用ホストで `/api/v1/*` として提供される。
- **BREAKING** 物理的に分かれた管理用 database を廃止し、Admin operator / audit / account management に必要な永続データは単一 PostgreSQL DB 内の Admin-owned schema に保持する。
- Product account auth と Admin operator auth は別ドメインとして実装する。Product は `internal/domain` の Product AccountAuth session / token 型と `internal/application/product/auth` を使い、Admin は `internal/domain` の Admin OperatorAuth session / token 型と `internal/application/admin/auth` を使う。相互 import は禁止する。
- 共有できるのは HMAC / JWT signer-verifier、Cookie 属性 helper、ULID validation、TTL validation など、account / operator の意味を持たない低レベル primitive に限定する。共有 primitive は account / operator enum switch、issuer/audience/domain pairing、RBAC、status 判定を所有してはならない。
- Product と Admin の refreshToken は response body やブラウザーから読める storage ではなく、`HttpOnly; Secure; SameSite=Lax; Path=/` Cookie として扱う。accessToken / refreshToken のドメイン claim と session state は Product account auth と Admin operator auth で別々に定義する。
- Admin と Product は同じ Valkey infrastructure を共有できるが、Admin operator auth state と Product account auth state は logical DB 番号と key prefix で分離される。
- Admin API は Product API と別のドメイン・別の生成物として扱う。Admin frontend と Admin backend は同一 Admin ドメインで提供し、Product SDK / Product Go bindings / Product OpenAPI に Admin operation を混入させない。Admin TypeScript SDK は `packages/admin/api` の package-local 生成物として扱い、`packages/frontend/api` には露出させない。
- Admin Console から顧客アカウントを作成できるようにし、作成時の Product Account lifecycle / status / email 不変条件、停止時の session revoke 境界、監査結果遷移、認証初期状態は Go backend の concrete domain object を通して適用する。

## Spec Units

### New Spec Units

- `api-contract-be`: New。TypeSpec が Product / Admin など複数の別バイナリ・別ホスト API surface を定義し、OpenAPI / SDK / Go bindings を surface ごとに分離生成する責務を扱う。Security: Admin route の Product 生成物混入を禁止する。

### Modified Spec Units

- `admin-console-fe`: Modified。Admin Console の画面、状態管理、API 呼び出し境界を static client として再定義し、Account 作成 UI と `packages/admin/app -> packages/admin/domain -> packages/admin/api` の依存方向を追加する。
- `admin-console-be`: Modified。Admin 管理 API、Admin 永続化、監査、Account 作成、Admin 専用 Go binary / host、Product API 非露出、Admin surface も `/api/v1/*` path policy に従う要件へ更新する。`packages/backend/internal/domain/**` の concrete domain object と unit test を必須にする。
- `admin-auth-fe`: Modified。SvelteKit server hooks / package-local BFF 前提を廃止し、静的 Admin frontend から Admin backend auth API を利用する認証体験へ更新する。
- `admin-auth-be`: Modified。Admin operator auth endpoint、Admin operator auth domain、cookie / CSRF / Origin / Valkey logical DB 分離を Admin 専用 Go backend surface の要件として更新する。
- `auth-be`: Modified。Product account auth を refreshToken HttpOnly Cookie model に更新し、Admin operator auth とは別ドメインであること、共有は中立 primitive だけに限定することを追加する。
- `auth-fe`: Modified。Product frontend の refreshToken 取り扱いをブラウザーから読める storage から HttpOnly Cookie 前提へ更新する。

## Naming

新規 Spec Unit `api-contract-be` の Scenario ID prefix は `API-CONTRACT-BE-*` を使用する。既存 Spec Unit は既存 prefix を維持し、`ADMIN-CONSOLE-FE-*` と `ADMIN-CONSOLE-BE-*`、`ADMIN-AUTH-FE-*` と `ADMIN-AUTH-BE-*` のように FE / BE scenario prefix を分離する。

## Impact

- `packages/admin`: `app` / `domain` / `api` の静的 frontend 層、server routes、server-only services/models/infrastructure、Prisma / OpenSearch / Valkey 直接接続責務、SvelteKit server hooks 前提、Node adapter 前提。
- `packages/backend`: Product API binary に加えて Admin API binary を提供する backend composition、Admin route authorization、Account 作成 use case、監査、same-DB migration。Admin binary は Admin ドメインの `/api/v1/*` を処理する。
- `packages/backend/internal/domain`: 既存 `account.go`、`account_auth.go`、`jwt.go`、`auth_ids.go`、`account_id.go` の flat domain package 規約に合わせ、Product Account lifecycle / email / status、Admin Operator、Admin AuditEvent、Product AccountAuth session/token、Admin OperatorAuth session/token、中立 token primitive を concrete file と Scenario ID 付き unit test で追加・更新する。
- `packages/backend/internal/application/product/auth` と `packages/backend/internal/application/admin/auth`: Product account auth と Admin operator auth を別 application boundary として実装し、共有する場合は `internal/application/shared/tokenprimitive` のような中立 primitive wrapper だけを参照する。Product は Admin auth domain/application を import せず、Admin は Product auth domain/application を import しない。
- `packages/backend/internal/adapter/http/{product,admin}`、`packages/backend/internal/application/{product,admin}`、`packages/backend/internal/adapter/{postgres,valkey}/{product,admin}`: Product/Admin の物理分離と Clean Architecture import boundary を lint/test で強制する。
- `packages/typespec`: Product / Admin の service surface 分離、両 surface の `/api/v1/*` path policy 維持、surface 別 OpenAPI 生成、surface 別 SDK / Go bindings 生成、Admin operation の Product 生成物混入防止。
- `packages/frontend/api` と `packages/admin/api`: Product SDK と Admin SDK の物理分離、Admin SDK を `packages/admin/api` に閉じる依存境界。
- Repository rules: `/api/v1/*` path policy は維持し、同じ path 空間を別 origin / 別 binary / 別 OpenAPI / 別 SDK / 別 Go bindings で分離するように AGENTS、CODING_STANDARDS、CONTRIBUTING、lint policy を更新する。
- DB: 物理的に分かれた管理用 database の廃止、Admin schema / tables / audit / operator / account management persistence、migration 管理境界。migration は `000007_create_admin_schema.up.sql` / `000007_create_admin_schema.down.sql` とする。
- Valkey: 同一 infrastructure 上で Product account auth と Admin operator auth の logical DB 番号および key prefix を分離。
- Security / operations: Admin ドメインと Product ドメインの分離、Cloudflare route による Admin 静的 frontend / Admin GoServer `/api/v1/*` 振り分け、Origin / CSRF / cookie / no-store / monitoring / deployment 設定。
