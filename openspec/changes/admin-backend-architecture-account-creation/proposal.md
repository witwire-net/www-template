## Why

現在の Admin Console は物理的なサービス分割とインフラ層の疎結合により安全性を確保している一方で、Admin と Product Backend の双方にドメイン管理が分かれ、同じアカウント操作の業務ルールを二重に持つ構造になっている。特に Admin 経由のアカウント作成と通常サービス経由のアカウント作成を両方提供すると、Account ドメインの不変条件、監査、セッション制御、API 契約が分岐し、顧客と運営者の双方にとって安全性と保守性が低下する。

Admin は強権限 surface であるため、Product 側への Admin 機能露出を避けつつ、Account ドメインの source of truth は Backend に集約する必要がある。Admin frontend と Admin backend は同一 Admin ドメインでホストし、Cloudflare が静的 frontend と GoServer の `/api/v1/*` にルーティングする。今後も別バイナリ・別ホストの API surface が増える前提で、TypeSpec と生成物の境界も Product / Admin の origin 分離に耐える形へ改める。

## What Changes

- **BREAKING** Admin Console は package-local BFF / server-side domain logic を持たず、`packages/admin/app -> packages/admin/domain -> packages/admin/api` の依存方向を持つ静的クライアントになる。
- **BREAKING** Admin backend 機能は `packages/admin` から削除され、Go backend の Admin 専用バイナリ・Admin 専用ホストで `/api/v1/*` として提供される。
- **BREAKING** Admin DB の物理分割を廃止し、Admin operator / audit / account management に必要な永続データは Product と同一 PostgreSQL DB に保持する。
- **BREAKING** Product と Admin の認証基盤は、共通の accessToken / refreshToken model を使う。refreshToken は response body や browser-readable storage ではなく、`HttpOnly; Secure; SameSite=Lax` Cookie として扱う。
- Admin と Product は同じ Valkey infrastructure を共有するが、Admin 用途と Product 用途は logical DB 番号と key prefix で分離される。
- Admin API は Product API と別のドメイン・別の生成物として扱われる。Admin frontend と Admin backend は同一 Admin ドメインで提供し、Product SDK / Product Go bindings / Product OpenAPI に Admin operation を混入させない。Admin TypeScript SDK は `packages/admin/api` の package-local 生成物として扱い、`packages/frontend/api` には露出させない。
- Admin Console から顧客アカウントを作成できるようにし、作成時の Account ドメイン不変条件、監査、認証初期状態は Backend 側で一元的に適用される。

## Spec Units

### New Spec Units

- `api-contract-be`: New。TypeSpec が Product / Admin など複数の別バイナリ・別ホスト API surface を定義し、OpenAPI / SDK / Go bindings を surface ごとに分離生成する責務を扱う。Security: Admin route の Product 生成物混入を禁止する。

### Modified Spec Units

- `admin-console-fe`: Modified。Admin Console の画面、状態管理、API 呼び出し境界を static client として再定義し、Account 作成 UI と `packages/admin/app -> packages/admin/domain -> packages/admin/api` の依存方向を追加する。
- `admin-console-be`: Modified。Admin 管理 API、同一 Product DB 上の Admin 永続化、監査、Account 作成、Admin 専用 Go binary / host、Product API 非露出、Admin surface も `/api/v1/*` path policy に従う要件へ更新する。
- `admin-auth-fe`: Modified。SvelteKit server hooks / package-local BFF 前提を廃止し、静的 Admin frontend から Admin backend auth API を利用する認証体験へ更新する。
- `admin-auth-be`: Modified。Admin auth endpoint、cookie / CSRF / Origin / Valkey logical DB 分離を Admin 専用 Go backend surface の要件として更新する。
- `auth-be`: Modified。Product account 認証基盤を refreshToken HttpOnly Cookie model に更新し、Admin operator 認証基盤が再利用できる共通 token service 境界を追加する。
- `auth-fe`: Modified。Product frontend の refreshToken 取り扱いを browser-readable storage から HttpOnly Cookie 前提へ更新する。

## Naming

新規 Spec Unit `api-contract-be` の Scenario ID prefix は `API-CONTRACT-BE-*` を使用する。既存 Spec Unit は既存 prefix を維持し、`ADMIN-CONSOLE-FE-*` と `ADMIN-CONSOLE-BE-*`、`ADMIN-AUTH-FE-*` と `ADMIN-AUTH-BE-*` のように FE / BE scenario prefix を分離する。

## Impact

- `packages/admin`: `app` / `domain` / `api` の静的 frontend 層、server routes、server-only services/models/infrastructure、Prisma / OpenSearch / Valkey 直接接続責務、SvelteKit server hooks 前提、Node adapter 前提。
- `packages/backend`: Product API binary に加えて Admin API binary を提供する backend composition、Admin route authorization、Account 作成 use case、監査、same-DB migration。Admin binary は Admin ドメインの `/api/v1/*` を処理する。
- `packages/backend/internal/application` と auth domain: account 認証ドメインと operator 認証ドメインで共通利用する accessToken / refreshToken service、refreshToken Cookie 発行・rotation・revoke。
- `packages/typespec`: Product / Admin の service surface 分離、両 surface の `/api/v1/*` path policy 維持、surface 別 OpenAPI 生成、surface 別 SDK / Go bindings 生成、Admin operation の Product 生成物混入防止。
- `packages/frontend/api` と `packages/admin/api`: Product SDK と Admin SDK の物理分離、Admin SDK を `packages/admin/api` に閉じる依存境界。
- Repository rules: `/api/v1/*` path policy は維持し、同じ path 空間を別 origin / 別 binary / 別 OpenAPI / 別 SDK / 別 Go bindings で分離するように AGENTS、CODING_STANDARDS、CONTRIBUTING、lint policy を更新する。
- DB: Admin DB 物理分割廃止、Product DB 内の Admin schema / tables / audit / operator / account management persistence、migration 管理境界。
- Valkey: 同一 infrastructure 上で Product と Admin の logical DB 番号および key prefix を分離。
- Security / operations: Admin ドメインと Product ドメインの分離、Cloudflare route による Admin 静的 frontend / Admin GoServer `/api/v1/*` 振り分け、Origin / CSRF / cookie / no-store / monitoring / deployment 設定。
