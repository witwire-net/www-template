## 1. Backend domain object を先に実装する

- [x] 1.1 `packages/backend/internal/domain/account_email.go` を追加し、`AccountEmail` の空白除去、canonical lowercase、形式検証、`String()`、domain error を既存 `account.go` / `account_id.go` の flat package 規約に合わせて実装する。
- [x] 1.2 `packages/backend/internal/domain/account_lifecycle.go` を追加し、`AccountStatusActive` / `AccountStatusSuspended`、`NewAdminCreatedAccount`、`Suspend(at)`、`Restore()`、`SessionRevokedAfter()`、停止中 token 拒否判定を実装する。
- [x] 1.3 既存 `packages/backend/internal/domain/account.go` を更新し、Account root が AccountEmail、AccountStatus、AccountSetting、session revocation 境界を保持できるようにする。
- [x] 1.4 `packages/backend/internal/domain/operator.go` を追加し、`OperatorID`、`OperatorEmail`、`OperatorRole`、active state、passkey registration state、`HasPermission("accounts:create")` を実装する。
- [x] 1.5 `packages/backend/internal/domain/admin_audit_event.go` を追加し、pending / succeeded / failed outcome、stable error code、completed timestamp、二重完了拒否を実装する。
- [x] 1.6 `packages/backend/internal/domain/account_auth_session.go` を追加し、Product AccountAuth の account accessToken claims、account refresh session state、account status / suspension / sessionRevokedAfter による token eligibility を実装する。
- [x] 1.7 `packages/backend/internal/domain/operator_auth.go` と `operator_auth_session.go` を追加し、Admin OperatorAuth の operator accessToken claims、operator refresh session state、role / active snapshot、CSRF binding、operator 固有 validation を実装する。
- [x] 1.8 `packages/backend/internal/domain/token_primitive.go` を追加し、HMAC/JWT 署名検証、opaque token hash、ULID/JTI validation、TTL value など中立 primitive だけを実装する。account / operator enum switch、issuer/audience pairing、RBAC、status 判定は置かない。
- [x] 1.9 `[ADMIN-CONSOLE-BE-S077] AccountEmail と Account lifecycle constructor は不変条件を検証する` の domain unit test を追加し、email 正規化、active 初期状態、suspend/restore、session revoke 境界を検証する。
- [x] 1.10 `[ADMIN-CONSOLE-BE-S078] Operator role と active state は permission を制御する` の domain unit test を追加し、admin/operator/viewer/inactive/passkey registration state の matrix を検証する。
- [x] 1.11 `[ADMIN-CONSOLE-BE-S079] AdminAuditEvent は不正な outcome transition を拒否する` の domain unit test を追加し、pending→succeeded/failed、二重完了拒否、failed 時 stable error code 必須、completed timestamp 必須を検証する。
- [x] 1.12 `[ADMIN-CONSOLE-BE-S080] internal/domain import graph は pure に保たれる` の lint / guardrail test を追加し、domain package が application / adapter / generated / platform を import しないことを検証する。
- [x] 1.13 `[AUTH-BE-S068] Neutral token primitive は account/operator domain switch を持たない` の unit test / static test を追加し、中立 primitive が `account` / `operator` domain enum switch を持たないことを検証する。
- [x] 1.14 `[AUTH-BE-S069] Product AccountAuth domain は account token eligibility を所有する` の unit test を追加し、suspended account、sessionRevokedAfter、account session ID mismatch を Product AccountAuth domain が拒否することを検証する。
- [x] 1.15 `[AUTH-BE-S070] Admin OperatorAuth domain は operator token eligibility と CSRF binding を所有する` の unit test を追加し、inactive operator、viewer 権限不足、CSRF mismatch、operator session ID mismatch を Admin OperatorAuth domain が拒否することを検証する。

## 2. Product account auth と Admin operator auth の application 境界を分離する

- [x] 2.1 `packages/backend/internal/application/product/auth/**` を追加し、Product account login / refresh / revoke / session validation が Product AccountAuth domain object だけを使うようにする。
- [x] 2.2 `packages/backend/internal/application/admin/auth/**` を追加し、Admin operator login / refresh / revoke / current operator / CSRF validation が Admin OperatorAuth domain object だけを使うようにする。
- [x] 2.3 `packages/backend/internal/application/shared/tokenprimitive/**` を追加し、TTL validation、Cookie lifetime validation、signer/verifier wrapper など中立 helper だけを提供する。account/operator claim や RBAC を置かない。
- [x] 2.4 Product application は Admin auth domain/application を import せず、Admin application は Product auth domain/application を import しない import-boundary rule を guardrail に追加する。
- [x] 2.5 refreshToken rotation を Product account auth と Admin operator auth で別 use case として実装し、どちらも旧 token 原子消費、新 token Cookie 設定、response body への refreshToken 非露出を満たす。
- [x] 2.6 Product frontend auth domain を accessToken-only のブラウザー可読 state に変更し、refresh request は credentials 付き same-origin Cookie refresh とする。
- [x] 2.7 `[AUTH-BE-S060] Product passkey login は accessToken body と refreshToken Cookie を返す` の test を追加する。
- [x] 2.8 `[AUTH-BE-S061] Admin operator login は Admin operator auth domain を使う` の test を追加し、operator accessToken / operator refresh state が account auth state と混在しないことを検証する。
- [x] 2.9 `[AUTH-BE-S062] refresh は Cookie refreshToken を rotation する` の Product/Admin 両 use case test を追加する。
- [x] 2.10 `[AUTH-BE-S063] ブラウザーから読める refreshToken は発行されない` の test を追加し、body/log/trace/error とブラウザーから読める storage に refreshToken 平文が出ないことを検証する。
- [x] 2.11 `[AUTH-BE-S064] refreshToken Cookie lifetime は server TTL を超えない` の test を追加する。
- [x] 2.12 `[AUTH-BE-S065] Product と Admin は同じ中立 TTL validation を使う` の test を追加し、Product/Admin が同じ中立 TTL helper を使うことを検証する。
- [x] 2.13 `[AUTH-BE-S066] multi-session refresh は対象 session だけを rotation する` の Product account auth test を追加する。
- [x] 2.14 `[AUTH-BE-S067] Product と Admin の auth domain は分離される` の boundary test を追加し、単一共有 token service の切替引数で発行・refresh・revoke が実装されていないことを検証する。
- [x] 2.15 `[AUTH-BE-S071] Product auth application は Admin auth application を import しない` の lint / import graph test を追加する。
- [x] 2.16 `[AUTH-BE-S072] Admin auth application は Product auth application を import しない` の lint / import graph test を追加する。
- [x] 2.17 `[AUTH-FE-S045] 期限切れ間近の accessToken は Cookie refresh で更新される` を追加する。
- [x] 2.18 `[AUTH-FE-S046] refreshToken はブラウザーから読める storage に保存されない` を追加し、ブラウザーから読める storage に保存されないことを検証する。
- [x] 2.19 `[AUTH-FE-S047] refresh 失敗時は対象 session だけを失効扱いにする` を追加する。
- [x] 2.20 `[AUTH-FE-S048] login は refreshToken なしで accessToken session を追加する` を追加する。
- [x] 2.21 `[AUTH-FE-S049] account switch は bearer accessToken を変更する` を追加する。
- [x] 2.22 `[AUTH-FE-S050] logout は対象 session の Cookie revoke を依頼する` を追加する。

## 3. 契約と生成物を surface ごとに分離する

- [x] 3.1 `packages/typespec` に Product service と Admin service を分離して定義し、両 surface が `/api/v1/*` path policy を維持する状態にする。
- [x] 3.2 Admin account / auth operation を `packages/typespec/src/routes/admin-v1/**` に追加し、Product route namespace から import されない状態にする。
- [x] 3.3 Product OpenAPI と Admin OpenAPI を別 artifact として生成する `packages/typespec` scripts / config を更新する。
- [x] 3.4 Product SDK は `packages/frontend/api`、Admin SDK は `packages/admin/api` に生成されるように Orval / package scripts を分離する。
- [x] 3.5 Product Go bindings と Admin Go bindings を別 package に生成するように `scripts/go/gen-backend.sh` と backend codegen config を更新する。
- [x] 3.6 `scripts/codegen/check.sh` に Product/Admin artifact の drift と operation/tag/export 混入検査を追加する。
- [x] 3.7 `AGENTS.md`、`CODING_STANDARDS.md`、`CONTRIBUTING.md` に Product/Admin とも `/api/v1/*` を使い、origin / binary / artifact で分離するルールを反映する。
- [x] 3.8 `pnpm gen` を実行し、Product/Admin OpenAPI、Product SDK、Admin package-local SDK、Product/Admin Go bindings を生成する。
- [x] 3.9 `[API-CONTRACT-BE-S001] Product OpenAPI は admin operations を含まない` を追加し、Product OpenAPI / Product SDK / Product Go bindings に Admin operation がないことを検証する。
- [x] 3.10 `[API-CONTRACT-BE-S002] Admin OpenAPI は product operations を含まない` を追加し、Admin OpenAPI / Admin SDK / Admin Go bindings に Product operation がないことを検証する。
- [x] 3.11 `[API-CONTRACT-BE-S003] Surface server URLs は分離される` を追加し、Product/Admin OpenAPI の server domain が分かれていることを検証する。
- [x] 3.12 `[API-CONTRACT-BE-S004] Shared model import は routes を増やさない` を追加し、shared model import が route を増やさないことを検証する。
- [x] 3.13 `[API-CONTRACT-BE-S005] Product surface は admin namespace を import できない` を追加し、Product TypeSpec から Admin namespace import を拒否する。
- [x] 3.14 `[API-CONTRACT-BE-S006] admin operation を含む Product artifact は check に失敗する` を追加し、混入 fixture が codegen check で失敗することを検証する。
- [x] 3.15 `[API-CONTRACT-BE-S007] Product binary は admin bindings を import できない` を追加し、Product binary から Admin bindings import を拒否する。
- [x] 3.16 `[API-CONTRACT-BE-S008] Admin bindings は Admin HTTP adapter だけが import する` を追加し、Admin bindings の import graph を検証する。
- [x] 3.17 `[API-CONTRACT-BE-S009] Product と Admin の SDK packages は物理分離される` を追加し、`packages/frontend/**` と `packages/admin/**` の SDK import 境界を検証する。

## 4. Backend runtime、application、adapter、永続化を実装する

- [x] 4.1 `packages/backend/cmd/admin-api/main.go` と `internal/app/admin_runtime.go` を追加し、Admin GoServer binary を Product binary と分離して起動できる状態にする。
- [x] 4.2 `packages/backend/tools/analyzers/cmd/guardrails/main.go` と backend lint policy を更新し、`cmd/admin-api`、`internal/generated/adminopenapi`、`internal/adapter/http/{product,admin,shared}`、`internal/application/{product,admin}`、`internal/application/shared/tokenprimitive`、`internal/adapter/{postgres,valkey}/{product,admin}` の配置と import 境界を許可・強制する。
- [x] 4.3 Product runtime が Product operations だけを register することを維持し、Admin operations が混入しない router 構成にする。
- [x] 4.4 Admin runtime が Admin operations の `/api/v1/*` だけを register し、Product operations を register しない router 構成にする。
- [x] 4.5 Admin schema、operator、operator passkey、audit event、権限を作成する `packages/backend/db/migrations/000007_create_admin_schema.up.sql` と `000007_create_admin_schema.down.sql` の migration pair を追加する。
- [x] 4.6 migration pair check を更新・実行し、`000007_create_admin_schema` が 6 桁連番、lowercase snake suffix、up/down pair、nested directory なしであることを検証する。
- [x] 4.7 `internal/adapter/http/product` と `internal/adapter/http/admin` を分離し、Product HTTP adapter は Product generated bindings だけ、Admin HTTP adapter は Admin generated bindings だけを import する。
- [x] 4.8 `internal/adapter/postgres/product`、`internal/adapter/postgres/admin`、`internal/adapter/valkey/product`、`internal/adapter/valkey/admin` を分離し、application ports の実装だけを提供する。
- [x] 4.9 Admin runtime config に Admin domain、Product domain、Admin cookie、Admin runtime DB role、Admin Valkey URL を追加し、起動時に fail-close validation する。
- [x] 4.10 Admin Valkey store を追加し、Product と同じ infrastructure かつ別 logical DB、`admin:*` prefix 限定を検証する。
- [x] 4.11 Admin auth middleware を追加し、same-origin Origin、operator session、CSRF binding、no-store、security headers を検証する。
- [x] 4.12 `accounts:create` を含む Admin RBAC authorization use case を追加し、handler は application decision を呼ぶだけにする。
- [x] 4.13 Admin audit use case を `internal/application/admin` に追加し、mutation 前の intent と success / failed outcome 更新を concrete `AdminAuditEvent` domain method に委譲する。
- [x] 4.14 Admin account repository を `internal/adapter/postgres/admin` に追加し、Admin schema と Account root を application port 経由で同一 transaction 境界に扱う。
- [x] 4.15 Admin account creation use case を `internal/application/admin` に追加し、`AccountEmail`、Account lifecycle、Operator、AuditEvent domain object を通して email 正規化、重複検査、作成、監査を実行する。
- [x] 4.16 Admin account handlers を generated Admin bindings に接続し、transport DTO を application DTO に変換し、domain rule を handler に置かない。
- [x] 4.17 Admin auth handlers のうち auth-only 範囲を generated Admin bindings に接続し、passkey start/finish、current operator、CSRF issuance、logout のみを実装する。operator setup は 4.62 以降の setup tasks に残す。
- [x] 4.18 `[ADMIN-CONSOLE-BE-S056] Product binary は admin operations を register しない` を追加する。
- [x] 4.19 `[ADMIN-CONSOLE-BE-S057] Admin binary は product operations を register しない` を追加する。
- [x] 4.20 `[ADMIN-CONSOLE-BE-S058] Product bearer token は Admin API で拒否される` を追加する。
- [x] 4.21 `[ADMIN-CONSOLE-BE-S059] Admin schema は backend migration で作成される` を追加する。
- [x] 4.22 `[ADMIN-CONSOLE-BE-S060] Product runtime role は Admin schema を参照できない` を追加する。
- [x] 4.23 `[ADMIN-CONSOLE-BE-S061] Admin package ORM migration は使われない` を追加する。
- [x] 4.24 `[ADMIN-CONSOLE-BE-S062] Admin API は customer account を作成する` を追加する。
- [x] 4.25 `[ADMIN-CONSOLE-BE-S063] Duplicate email は 409 と failed audit を返す` を追加する。
- [x] 4.26 `[ADMIN-CONSOLE-BE-S064] account create permission を持たない Operator は 403 を受ける` を追加する。
- [x] 4.27 `[ADMIN-CONSOLE-BE-S065] Audit intent failure は account mutation を防ぐ` を追加する。
- [x] 4.28 `[ADMIN-CONSOLE-BE-S066] Account creation failure は failed audit outcome を記録する` を追加する。
- [x] 4.29 `[ADMIN-CONSOLE-BE-S067] Admin account creation は Account domain rule を共有する` を追加する。
- [x] 4.30 `[ADMIN-CONSOLE-BE-S068] Admin と operator は accounts:create を持つ` を追加する。
- [x] 4.31 `[ADMIN-CONSOLE-BE-S069] Viewer は accounts:create を持たない` を追加する。
- [x] 4.32 `[ADMIN-CONSOLE-BE-S070] Product binary の admin bindings import は失敗する` を追加する。
- [x] 4.33 `[ADMIN-CONSOLE-BE-S071] Admin schema migration は次の単調増加 backend version を使う` を追加し、`000007_create_admin_schema.up.sql/down.sql` の pair と zero-only version prefix 不在を検証する。
- [x] 4.34 `[ADMIN-CONSOLE-BE-S072] Admin HTTP adapter は persistence adapters を import できない` を追加する。
- [x] 4.35 `[ADMIN-CONSOLE-BE-S073] Application ports は adapter 型や generated 型を公開しない` を追加する。
- [x] 4.36 `[ADMIN-CONSOLE-BE-S074] Account invariants は concrete domain objects に留まる` を追加する。
- [x] 4.37 `[ADMIN-CONSOLE-BE-S075] Product と Admin の application packages は相互 import しない` を追加する。
- [x] 4.38 `[ADMIN-CONSOLE-BE-S076] Admin persistence と Product persistence は別 namespace を使う` を追加する。
- [x] 4.39 `[ADMIN-AUTH-BE-S056] Product host は Admin login API を提供しない` を追加する。
- [x] 4.40 `[ADMIN-AUTH-BE-S057] Admin middleware は operator accessToken を検証する` を追加する。
- [x] 4.41 `[ADMIN-AUTH-BE-S058] Product bearer token は Admin auth session ではない` を追加する。
- [x] 4.42 `[ADMIN-AUTH-BE-S059] 許可されない Origin は Admin mutation で拒否される` を追加する。
- [x] 4.43 `[ADMIN-AUTH-BE-S060] session と一致しない CSRF token は拒否される` を追加する。
- [x] 4.44 `[ADMIN-AUTH-BE-S061] Passkey start は session CSRF なしで Origin を検証する` を追加する。
- [x] 4.45 `[ADMIN-AUTH-BE-S062] Admin と Product の Valkey logical DB が同じ場合は起動に失敗する` を追加する。
- [x] 4.46 `[ADMIN-AUTH-BE-S063] Admin backend は admin-prefixed keys だけを書き込む` を追加する。
- [x] 4.47 `[ADMIN-AUTH-BE-S064] Admin refreshToken Cookie は SameSite=Lax を使う` を追加する。
- [x] 4.48 `[ADMIN-AUTH-BE-S065] insecure production cookie は拒否される` を追加する。
- [x] 4.49 `[ADMIN-AUTH-BE-S066] Admin API response は security headers を持つ` を追加する。
- [x] 4.50 `[ADMIN-CONSOLE-BE-S081] Admin schema migration は backend migration system だけで実行される` を追加し、Admin schema migration が `packages/backend/db/migrations/000007_create_admin_schema.*.sql` だけで管理され、`packages/admin/prisma/**` migration が使用されないことを検証する。
- [x] 4.51 `[ADMIN-CONSOLE-BE-S082] Admin schema migration rollback は pair policy を満たす` を追加し、down migration が Admin schema と grants を戻し、Product `public.accounts` を保持することを検証する。
- [x] 4.52 `[ADMIN-CONSOLE-BE-S083] 範囲外の limit は Admin backend で拒否される` を追加し、Admin account search use case が invalid pagination を 400 にし repository query を実行しないことを検証する。
- [x] 4.53 `[ADMIN-CONSOLE-BE-S084] unsafe raw query は lint または integration test で拒否される` を追加し、Admin account search repository の unsafe SQL construction を拒否することを検証する。
- [x] 4.54 `[ADMIN-CONSOLE-BE-S085] Admin audit event は Go backend から OpenSearch に projection される` を追加し、Admin audit prefix のみへ indexing され、`packages/admin` が OpenSearch client を import しないことを検証する。
- [x] 4.55 `[ADMIN-CONSOLE-BE-S086] OpenSearch namespace collision は起動時に拒否される` を追加し、Admin audit prefix と Product domain prefix の同一・包含関係で fail-close することを検証する。
- [x] 4.56 `[ADMIN-CONSOLE-BE-S087] OpenSearch indexing failure は mutation 成功を取り消さず観測される` を追加し、warning log、metric、retry queue または retry marker を検証する。
- [x] 4.57 `[ADMIN-CONSOLE-BE-S088] Admin source の secret literal は lint エラーになる` を追加し、DB connection string や token/key/password literal を検出することを検証する。
- [x] 4.58 `[ADMIN-CONSOLE-BE-S089] Admin Svelte source の unsafe HTML injection は lint エラーになる` を追加する。
- [x] 4.59 `[ADMIN-CONSOLE-BE-S090] Admin backend unsafe SQL construction は lint エラーになる` を追加する。
- [x] 4.60 `[ADMIN-AUTH-BE-S067] 登録済み passkey 一覧を Admin backend から取得できる` を追加し、Admin operator session / CSRF / Admin Valkey namespace を使い Product auth と BFF route を使わないことを検証する。
- [x] 4.61 `[ADMIN-AUTH-BE-S068] 最後の passkey 削除は Admin operator auth domain で拒否される` を追加し、最後の credential が保持されることを検証する。
- [x] 4.62 `[ADMIN-AUTH-BE-S069] オペレーター 0 件時に Admin backend が最初の admin を作成する` を追加し、setup start/finish が Admin domain/application と Admin schema transaction を使うことを検証する。
- [x] 4.63 `[ADMIN-AUTH-BE-S070] bootstrap secret 平文は観測可能な出力に残らない` を追加し、DB、audit、log、trace、response body、error message に secret 平文が出ないことを検証する。
- [x] 4.64 `[ADMIN-AUTH-BE-S071] 追加 operator は setup token で初回 passkey を登録できる` を追加し、setup token hash/expiry consumption と Admin refreshToken Cookie 発行を検証する。
- [x] 4.65 `[ADMIN-AUTH-BE-S072] setup token error は non-revealing error になる` を追加し、invalid/expired/consumed/registered 状態を区別できない error と challenge 未発行を検証する。
- [x] 4.66 `[ADMIN-AUTH-BE-S073] user verification なしの Admin assertion は拒否される` を追加し、Admin OperatorAuth domain/application が session 発行を拒否することを検証する。
- [x] 4.67 `[ADMIN-AUTH-BE-S074] user verification ありの Admin assertion は operator session を発行する` を追加し、operator accessToken body と Admin refreshToken Cookie が発行されることを検証する。
- [x] 4.68 `[ADMIN-AUTH-BE-S075] Admin operator credential が保存され検証に使われる` を追加し、public_key と sign_count が assertion 検証に使われることを検証する。
- [x] 4.69 `[ADMIN-AUTH-BE-S076] sign_count 減少は replay attack として拒否される` を追加し、session が発行されないことを検証する。
- [x] 4.70 `[ADMIN-AUTH-BE-S077] 重複 credential_handle の Admin operator 登録は拒否される` を追加し、409 と credential 未追加を検証する。
- [x] 4.71 `[ADMIN-CONSOLE-BE-S091] operator creation は setup token 平文を response に含めない` を追加し、operator summary、delivery status、audit correlation ID だけが返ることを検証する。
- [x] 4.72 `[ADMIN-CONSOLE-BE-S092] setup token delivery failure は failed audit outcome を記録する` を追加し、secure delivery port failure と secret 非露出を検証する。
- [x] 4.73 `[ADMIN-CONSOLE-BE-S093] passkey 登録済み operator の setup token 再発行は拒否される` を追加する。
- [x] 4.74 Admin operator creation use case を `internal/application/admin/operator_service.go` に追加し、Operator domain object、setup token hash/expiry、secure delivery port、audit outcome、token reissue guard を実装する。
- [x] 4.75 setup token secure delivery adapter を `internal/adapter/mailer/admin_setup_token_delivery.go` に追加し、平文 setup token を response body、DB、audit、log、trace、error message に出さない done condition を満たす。

## 5. Admin 静的 frontend 層を実装する

- [x] 5.1 `packages/admin` を `app`、`domain`、`api` の layer に整理し、`app -> domain -> api` 以外の依存を lint で拒否する。
- [x] 5.2 `packages/admin` の Node adapter、server routes、server load/actions、`$lib/server`、Prisma、Valkey、OpenSearch、WebAuthn server dependency、Prisma generation scripts を削除する。
- [x] 5.3 `packages/admin/api` に Admin SDK 生成設定と same-origin `/api/v1/*` wrapper を追加し、Product domain への request を拒否する。
- [x] 5.4 `packages/admin/domain` に auth、current operator、protected route state、account search/detail/create domain functions を追加する。
- [x] 5.5 `packages/admin/app` の login / operator setup を browser WebAuthn と domain functions 経由に変更する。
- [x] 5.6 `packages/admin/app` の protected routes を current operator verification 後に表示する構成に変更する。
- [x] 5.7 Account 作成 component を追加し、Accounts page から validation、submit、duplicate/error 表示、detail navigation を扱う。
- [x] 5.8 `[ADMIN-CONSOLE-FE-S038] Admin app layer の direct API client import は失敗する` を追加する。
- [x] 5.9 `[ADMIN-CONSOLE-FE-S039] Admin package の server-only module は失敗する` を追加する。
- [x] 5.10 `[ADMIN-CONSOLE-FE-S040] Admin domain は account data に Admin api layer を使う` を追加する。
- [x] 5.11 `[ADMIN-CONSOLE-FE-S041] Admin API は same-origin api/v1 を使う` を追加する。
- [x] 5.12 `[ADMIN-CONSOLE-FE-S042] Admin API wrapper は Product domain を拒否する` を追加する。
- [x] 5.13 `[ADMIN-CONSOLE-FE-S043] Operator は customer account を作成する` を component / E2E で追加する。
- [x] 5.14 `[ADMIN-CONSOLE-FE-S044] Invalid email は送信されない` を追加する。
- [x] 5.15 `[ADMIN-CONSOLE-FE-S045] Duplicate email error は form input を保持する` を追加する。
- [x] 5.16 `[ADMIN-CONSOLE-FE-S046] Admin frontend domain は Product frontend domain と異なる` を追加する。
- [x] 5.17 `[ADMIN-AUTH-FE-S027] Login UI は Admin backend auth API を呼び出す` を追加する。
- [x] 5.18 `[ADMIN-AUTH-FE-S028] Product auth SDK は operator session 作成に使われない` を追加する。
- [x] 5.19 `[ADMIN-AUTH-FE-S029] setup token error は秘匿的な表示へ変換される` を追加し、秘匿的な表示へ変換されることを検証する。
- [x] 5.20 `[ADMIN-AUTH-FE-S030] Protected content は session なしでは表示されない` を追加する。
- [x] 5.21 `[ADMIN-AUTH-FE-S031] UI role controls は backend authorization を代替しない` を追加する。
- [x] 5.22 `[ADMIN-AUTH-FE-S032] Admin HTML は no-store で配信される` を追加する。
- [x] 5.23 `[ADMIN-AUTH-FE-S033] Operator login は accessToken だけをブラウザーから読める state に保持する` を追加し、refreshToken がブラウザーから読める state に入らないことを検証する。
- [x] 5.24 `[ADMIN-AUTH-FE-S034] Protected route は operator accessToken を検証に使う` を追加する。
- [x] 5.25 `[ADMIN-AUTH-FE-S035] Admin refresh は HttpOnly Cookie を使う` を追加する。
- [x] 5.26 `[ADMIN-AUTH-FE-S036] Admin refresh 失敗時は protected content を表示しない` を追加し、memory state cleanup と login 誘導を検証する。
- [x] 5.27 `[ADMIN-AUTH-FE-S037] session expiry reason は UI に露出しない` を追加し、expired/revoked/inactive の詳細を区別しない generic guidance を検証する。
- [x] 5.28 `[ADMIN-AUTH-FE-S038] 静的 setup UI は Admin backend で最初の admin を作成する` を追加し、`/api/v1/auth/setup/*`、memory accessToken state、refreshToken 平文不在を検証する。
- [x] 5.29 `[ADMIN-AUTH-FE-S039] operator が存在する場合は setup form を表示しない` を追加する。
- [x] 5.30 `[ADMIN-AUTH-FE-S040] bootstrap gate 無効時は setup secret 入力欄を表示しない` を追加する。

## 6. ルーティングと全体検証

- [x] 6.1 Admin domain の Cloudflare route 設定を文書化し、static frontend と `/api/v1/*` GoServer routing を明示する。
- [x] 6.2 Product domain と Admin domain が一致しないこと、かつ両 domain がそれぞれ same-origin `/api/v1/*` を持つことを deployment docs に反映する。
- [x] 6.3 `pnpm gen` を実行し、Product/Admin 生成物の分離を確認する。
- [x] 6.4 `pnpm check:codegen` を実行し、drift と surface contamination を修正する。
- [x] 6.5 `pnpm check` を実行して TypeSpec、Svelte、TypeScript、Go build 問題を修正する。
- [x] 6.6 `pnpm lint` を実行して layer、security、codegen policy 問題を修正する。
- [x] 6.7 `pnpm test:run` を実行し、Scenario ID 付き automated tests が通ることを確認する。
- [x] 6.8 `pnpm build` を実行し、Product API、Admin API、Product frontend、Admin static frontend を検証する。
- [x] 6.9 環境が用意できる場合は `pnpm test:e2e` を実行し、Product/Admin domain separation と Account 作成 flow を検証する。

## 7. 受入、移行、release verification

- [x] 7.1 design.md の User Acceptance Test に従い、Admin account 作成 happy path、invalid/duplicate email、未認証 protected route、Product/Admin host separation、refreshToken Cookie only、migration up/down を staging または同等環境で確認する。
- [x] 7.2 `packages/backend/db/migrations/000007_create_admin_schema.up.sql` を適用し、Admin schema / operator / passkey / audit tables、least-privilege grants、Product runtime role の Admin schema denial を確認する。
- [x] 7.3 `packages/backend/db/migrations/000007_create_admin_schema.down.sql` を rollback 検証し、Admin schema と grants が戻り、Product `public.accounts` が保持されることを確認する。
- [x] 7.4 Product/OpenAPI/SDK/Go bindings と Admin/OpenAPI/SDK/Go bindings を比較し、Product artifact に Admin operation/tag/export がなく、Admin artifact に Product operation/tag/export がないことを release checklist に記録する。
- [x] 7.5 Product/Admin login、refresh、logout、operator setup、account creation の response body、browser-readable storage、URL、log、trace、error message を確認し、refreshToken 平文が存在しないことを記録する。
- [x] 7.6 Admin static frontend の HTML/runtime config response が no-store semantics を持ち、hashed static assets だけが長期 cache 可能であることを確認する。
- [x] 7.7 Product host で Admin API が到達不能、Admin host で Product API が到達不能であることを smoke test し、Cloudflare routing 設定と Go runtime route table の両方で確認する。
- [x] 7.8 `pnpm gen`、`pnpm check:codegen`、`pnpm check`、`pnpm lint`、`pnpm test:run`、`pnpm build`、環境がある場合は `pnpm test:e2e` の実行結果を release note に記録する。
