## Purpose

Admin Console の backend requirements をまとめる。Admin 専用 backend surface、Account creation、Admin-owned schema、Product Account root management、audit intent/outcome recording、operator lifecycle、OpenSearch projection、migration management、query safety、security lint、RBAC、Clean Architecture import boundaries を対象とする。

## Requirements

### Requirement: Admin 管理 API は Admin 専用 backend surface でのみ公開される

Backend は Product API binary と Admin API binary を別 entrypoint として提供しなければならない（SHALL）。Product API binary は Admin operations、Admin handlers、Admin generated bindings、Admin authorization middleware を register してはならない（MUST NOT）。Admin API binary は Admin operations の `/api/v1/*` routes を Admin public HTTP surface として register し、Product operations を register してはならない（MUST NOT）。Admin operations は Admin surface の generated server bindings に従い、Product OpenAPI / Product SDK / Product Go bindings に含まれてはならない（MUST NOT）。Admin API は operator accessToken、operator session record、Admin RBAC を必須とし、Product user bearer token を認可に使用してはならない（MUST NOT）。

**Customer Context**

Admin API は account lifecycle、operator management、audit など強権限 operation を提供する。Product backend host や Product SDK から Admin operation が見えると、誤用や攻撃面の拡大につながるため、Admin API は `/api/v1/*` path policy を維持したまま別バイナリ・別ホストの surface として閉じる必要がある。

#### Scenario: Product binary に Admin route が登録されない (ADMIN-CONSOLE-BE-S056)

- **GIVEN** Product API binary が起動している
- **WHEN** Admin account operation path である `POST /api/v1/accounts` に request を送信する
- **THEN** Product API binary は Admin handler を実行せず 404 または host-level reject を返す

#### Scenario: Admin binary に Product route が登録されない (ADMIN-CONSOLE-BE-S057)

- **GIVEN** Admin API binary が起動している
- **WHEN** Product operation path である `/api/v1/sessions` に request を送信する
- **THEN** Admin API binary は Product handler を実行せず 404 または host-level reject を返す

#### Scenario: Product bearer token は Admin API で認可されない (ADMIN-CONSOLE-BE-S058)

- **GIVEN** request が有効な Product bearer token を持つが operator accessToken と operator session record を持たない
- **WHEN** Admin account search API を呼び出す
- **THEN** Admin API は request を未認証または権限不足として拒否する

### Requirement: Admin 経由で顧客アカウントを作成できる

Admin API は operator 権限を検証したうえで顧客 account 作成 operation を SHALL 提供する。Account 作成 input は email を必須とし、正規化、形式検証、重複検証を Backend 側で実行しなければならない（MUST）。作成される account は `active` status で開始し、passkey credential を持たない状態を表現できなければならない（SHALL）。Account 作成は Account domain の不変条件を Product API と共有し、Admin 専用の別 domain rule を持ってはならない（MUST NOT）。成功・失敗の audit event は operator、target account、input email の正規化値、outcome、error code、request ID を記録しなければならない（SHALL）。重複 email は stable error として 409 を返し、既存 account を変更してはならない（MUST NOT）。

**Customer Context**

運営者はサポートや導入時に顧客アカウントを作成する必要がある。作成は通常サービス経由の Account domain と同じ不変条件を適用し、重複や監査漏れを防ぐ必要がある。

#### Scenario: Admin API が顧客 account を作成する (ADMIN-CONSOLE-BE-S062)

- **GIVEN** Operator が account 作成権限を持つ
- **WHEN** Admin account creation API に `customer@example.com` を送信する
- **THEN** `active` status の account が作成され、response は account ID、email、status、passkey count を返す
- **AND** audit event は outcome=`succeeded` で記録される

#### Scenario: 重複 email は account を作成しない (ADMIN-CONSOLE-BE-S063)

- **GIVEN** `customer@example.com` の account が既に存在する
- **WHEN** 同じ email で Admin account creation API を呼び出す
- **THEN** API は 409 を返し、account は追加されない
- **AND** audit event は outcome=`failed` と stable error code を記録する

#### Scenario: 権限不足の operator は account を作成できない (ADMIN-CONSOLE-BE-S064)

- **GIVEN** Operator が account 作成権限を持たない
- **WHEN** Admin account creation API を呼び出す
- **THEN** API は 403 を返し、account と audit target state は変更されない

### Requirement: Backend domain は Admin account creation に必要な concrete domain object を実装する

Backend は `packages/backend/internal/domain/**` に concrete domain object を実装しなければならない（MUST）。Product Account lifecycle は `AccountEmail`、`AccountStatus`、Admin 作成 account の active 初期状態、suspend / restore、`sessionRevokedAfter` を domain constructor / method として持たなければならない（MUST）。Admin Operator は operator ID、email、role、active state、passkey registration state、permission invariant を domain object として持たなければならない（MUST）。Admin AuditEvent は pending / succeeded / failed outcome、stable error code、completed timestamp、二重完了拒否を domain object として持たなければならない（MUST）。Admin account creation use case はこれらの concrete domain object を呼び出してから persistence を行わなければならず（MUST）、HTTP handler、repository、generated bindings、runtime composition、frontend package に同じ業務規則を隠してはならない（MUST NOT）。

**Customer Context**

Clean Architecture と書かれていても、domain object が存在せず application や handler に validation が散ると、Admin 作成 account と Product account の挙動が分岐する。顧客 account の停止、session 失効、operator 権限、監査結果は Backend domain に concrete な型として実装され、テストで守られる必要がある。

#### Scenario: Account lifecycle は concrete domain constructor で検証される (ADMIN-CONSOLE-BE-S077)

- **GIVEN** Admin account creation use case が email と account ID を受け取る
- **WHEN** Account root を作成する
- **THEN** use case は `AccountEmail` と Account lifecycle constructor を呼び、正規化 email、`active` 初期状態、passkey count 0、sessionRevokedAfter 初期値を domain object から得る
- **AND** handler と repository は email 正規化や初期 status 決定を行わない

#### Scenario: Operator authorization は concrete domain role model を使う (ADMIN-CONSOLE-BE-S078)

- **GIVEN** Admin mutation request が operator ID、email、role、active state、passkey registration state を持つ
- **WHEN** account 作成権限を評価する
- **THEN** use case は `Operator` domain object と `HasPermission("accounts:create")` を使う
- **AND** inactive operator と viewer は domain decision により拒否される

#### Scenario: Admin audit outcome は concrete domain transition を使う (ADMIN-CONSOLE-BE-S079)

- **GIVEN** audit intent が pending として作成済みである
- **WHEN** mutation が成功または失敗する
- **THEN** use case は `AdminAuditEvent.MarkSucceeded(completedAt)` または `MarkFailed(stableErrorCode, completedAt)` を呼ぶ
- **AND** missing stable error code、missing completed timestamp、二重完了は domain error になる

#### Scenario: Backend domain package は pure に保たれる (ADMIN-CONSOLE-BE-S080)

- **WHEN** `packages/backend/internal/domain/**` の import graph を検査する
- **THEN** `internal/application`、`internal/adapter/**`、`internal/generated/**`、`internal/platform/**`、Gin、GORM、Valkey、HTTP DTO を import している場合は lint または guardrail test が失敗する

### Requirement: オペレーター追加は one-time setup token を発行する

Admin operator creation は Admin backend の application use case と `Operator` domain object を通じて行われなければならない（SHALL）。Setup token は one-time token として生成され、hash と expiry だけを Admin-owned schema に保存し、平文 setup token を response body、DB、audit、OpenSearch、application log、trace、error message に出力してはならない（MUST NOT）。Setup token delivery は backend-owned secure delivery port を通じて実行され、create response は delivery status、operator summary、audit correlation ID だけを返さなければならない（SHALL）。Setup token の再発行は admin role のみが実行でき、既存 token hash を無効化して新 hash/expiry と audit event を保存しなければならない（MUST）。passkey 登録済み operator への token 再発行、自分自身の無効化、最後の admin の無効化、最後の admin の降格は拒否されなければならない（MUST）。

**Customer Context**

追加 operator は初回 passkey 登録用の secret を必要とするが、Admin UI response に setup token 平文を返すと画面・ブラウザー拡張・ログ・support tool に secret が残る。backend-owned delivery と hash 保存に限定し、operator lifecycle と audit を Admin backend に集約する必要がある。

#### Scenario: operator creation は setup token 平文を response に含めない (ADMIN-CONSOLE-BE-S091)

- **GIVEN** admin が新規 operator を作成する
- **WHEN** createOperator が成功する
- **THEN** response は operator summary、delivery status、audit correlation ID を含む
- **AND** setup token 平文は response body、DB、audit、OpenSearch、log、trace、error message に存在しない

#### Scenario: setup token delivery failure は failed audit outcome を記録する (ADMIN-CONSOLE-BE-S092)

- **GIVEN** operator record と setup token hash 保存は transaction 内で準備されている
- **WHEN** backend-owned secure delivery port が失敗する
- **THEN** use case は stable error と failed audit outcome を記録し、secret 平文を出力しない

#### Scenario: passkey 登録済み operator の setup token 再発行は拒否される (ADMIN-CONSOLE-BE-S093)

- **GIVEN** operator が少なくとも 1 件の passkey credential を持つ
- **WHEN** admin が setup token を再発行しようとする
- **THEN** Admin backend は domain/application decision として拒否し、token hash は変更されない

### Requirement: Admin Database Schema

Admin operator、operator passkey、audit event、Admin account management に必要な永続データは Product と同じ PostgreSQL database 内の Admin-owned schema に保持されなければならない（SHALL）。Admin schema の migration は `packages/backend/db/migrations/**` の backend migration system で管理され、Admin package の ORM migration に依存してはならない（MUST NOT）。Admin schema migration file は既存 backend migration の最大 version `000006` に続く `000007_create_admin_schema.up.sql` / `000007_create_admin_schema.down.sql` の pair で作成されなければならない（MUST）。Product runtime role は Admin schema の table / function / view へ権限を持ってはならない（MUST NOT）。Admin runtime role は Admin schema と、Account 作成 repository transaction が実際に使う `public.accounts` の email 参照・Account root 作成列、および `public.account_settings` の account_id 参照・trigger child 作成・locale 更新列だけに最小権限でアクセスしなければならない（MUST）。Admin account mutation と audit intent / outcome は同じ PostgreSQL database 内で整合性を検証できなければならない（SHALL）。

**Customer Context**

Admin と Product の DB が物理分割されると、Account domain の状態変更と監査・operator 操作の整合性が分散し、保守負荷が高くなる。Admin 専用 schema と最小権限 role により、同一 DB で整合性を保ちながら Product runtime からの到達を制限する。

#### Scenario: Admin schema は backend migration で作成される (ADMIN-CONSOLE-BE-S059)

- **GIVEN** backend migration が適用済みである
- **WHEN** DB の schema 一覧を確認する
- **THEN** Admin operator、operator passkey、audit event を保持する Admin-owned schema が存在する

#### Scenario: Admin schema migration は backend の次の単調増加 version を使う (ADMIN-CONSOLE-BE-S071)

- **GIVEN** `packages/backend/db/migrations` に `000001` から `000006` の migration pair が存在する
- **WHEN** Admin schema migration file を確認する
- **THEN** `000007_create_admin_schema.up.sql` と `000007_create_admin_schema.down.sql` が pair で存在し、zero-only version prefix の migration は存在しない

#### Scenario: Product runtime role は Admin schema を参照できない (ADMIN-CONSOLE-BE-S060)

- **GIVEN** Product API runtime role で DB に接続している
- **WHEN** Admin audit table を SELECT しようとする
- **THEN** DB は権限不足として拒否する

### Requirement: Product Database 拡張

Account root は `public.accounts` であり、Admin account management は Account root の永続不変条件と status / session revocation semantics を共有しなければならない（MUST）。Admin API が account lifecycle を操作する場合、Product API と同じ Account domain rule を使用し、Admin 専用の重複した Account domain rule を持ってはならない（MUST NOT）。Admin API から Account table を参照・更新する経路は backend repository と application service に限定され、Product runtime から Admin schema へ逆参照できてはならない（MUST NOT）。

**Customer Context**

Admin 経由の account create / suspend / restore が Product Account domain と別々の rule を持つと、顧客 account の状態と認証挙動が分岐する。Account domain は Backend に集約し、Admin は同じ root を安全に操作する必要がある。

#### Scenario: Admin account creation は Account domain rule を共有する (ADMIN-CONSOLE-BE-S067)

- **GIVEN** Product Account domain が email 正規化と重複禁止を定義している
- **WHEN** Admin account creation API が実行される
- **THEN** Product API と同じ Account domain validation が適用される

### Requirement: Service 層は全 mutation を監査 intent と outcome で記録する

Admin API は account create / suspend / restore、operator management、setup token rotation、passkey management の mutation 開始前に audit intent を記録しなければならない（MUST）。Mutation が成功した場合、audit outcome は `succeeded` で completed timestamp を持たなければならない（SHALL）。Mutation が domain error または infrastructure error で失敗した場合、audit outcome は `failed` と stable error code を保持しなければならない（SHALL）。Audit intent の作成に失敗した場合、Admin API は mutation を開始してはならない（MUST NOT）。

**Customer Context**

Admin mutation は顧客影響が大きく、成功だけでなく失敗も調査対象になる。同一 DB 上で intent と outcome を記録することで、Account domain の変更と監査の対応関係を追跡できる。

#### Scenario: account 作成の audit intent が作成できない場合は mutation しない (ADMIN-CONSOLE-BE-S065)

- **GIVEN** audit intent insert が DB error で失敗する
- **WHEN** Admin account creation API を呼び出す
- **THEN** account は作成されず、API は fail-close error を返す

#### Scenario: account 作成失敗は failed outcome として監査される (ADMIN-CONSOLE-BE-S066)

- **GIVEN** audit intent は作成済みで、Account domain validation が失敗する
- **WHEN** Admin account creation API が error handling を行う
- **THEN** audit event は outcome=`failed`、stable error code、completed timestamp を保持する

### Requirement: Prisma Client 管理

Admin backend は Prisma Client を runtime dependency として使用してはならない（MUST NOT）。Admin API の永続化は Go backend の repository と backend migration system により管理されなければならない（SHALL）。`packages/admin` は DB client、ORM schema、migration、generated Prisma client を所有してはならない（MUST NOT）。

**Customer Context**

Admin の backend 責務を `packages/admin` に置くと、Domain logic と DB migration が Product Backend と二分される。Admin 永続化は Backend に集約し、Admin frontend は静的 client に限定する。

#### Scenario: Admin package の ORM migration は使われない (ADMIN-CONSOLE-BE-S061)

- **WHEN** repository の DB migration command policy を確認する
- **THEN** Admin 永続化 schema は backend migration system の対象であり、Admin package-local ORM migration command は使用されない

### Requirement: Migration Management

Admin schema と Account root 拡張の migration は `packages/backend/db/migrations/**` の backend migration system で管理されなければならない（SHALL）。`packages/admin/prisma/**`、package-local Admin ORM migration は DB schema 変更に使用してはならない（MUST NOT）。Admin schema migration は `000007_create_admin_schema.up.sql` / `000007_create_admin_schema.down.sql` の pair として提供され、up/down のどちらも backend migration pair policy を満たさなければならない（MUST）。

**Customer Context**

Admin schema と Account root は同じ DB 内で整合性を持つ。migration 境界が Admin package と Go backend に分散すると、監査・権限・Account lifecycle の整合を確認しにくくなるため、DB 変更は backend migration system に集約する。

#### Scenario: Admin schema migration は backend migration system だけで実行される (ADMIN-CONSOLE-BE-S081)

- **WHEN** DB migration command policy と migration directory を確認する
- **THEN** Admin schema migration は `packages/backend/db/migrations/000007_create_admin_schema.up.sql` / `.down.sql` に存在する
- **AND** `packages/admin/prisma/**` の migration は DB schema 変更に使用されない

#### Scenario: Admin schema migration rollback は pair policy を満たす (ADMIN-CONSOLE-BE-S082)

- **GIVEN** `000007_create_admin_schema.up.sql` が適用済みである
- **WHEN** backend migration system で 1 step rollback を実行する
- **THEN** `000007_create_admin_schema.down.sql` により Admin schema grants と Admin repository 用の `public.accounts` / `public.account_settings` 列 grant は戻り、Product `public.accounts` は保持される

### Requirement: アカウント検索はページネーションと入力検証を持つ

Admin account search API は Go backend の Admin application use case と Postgres adapter を通じて account search を提供しなければならない（SHALL）。`limit` は 1〜100、`offset` は 0 以上、email search string は最大 255 文字として backend application または domain value object で検証されなければならない（MUST）。SQL は Postgres adapter の parameterized query または GORM parameter binding で実行され、unsafe raw query は使用してはならない（MUST NOT）。`packages/admin` は DB client を import せず、検索は Admin SDK 経由で同一 Admin host の `/api/v1/*` に委譲しなければならない（MUST）。

**Customer Context**

運営者は顧客 account を検索して詳細確認や作成結果確認を行う。検索条件の検証と SQL injection 対策が frontend package に散ると、強権限 data access の安全性を backend で一貫して保証できない。

#### Scenario: 範囲外の limit は Admin backend で拒否される (ADMIN-CONSOLE-BE-S083)

- **GIVEN** Admin account search API に `limit=0` が渡される
- **WHEN** Admin application use case が query を検証する
- **THEN** API は 400 と stable validation error を返し、repository query は実行されない

#### Scenario: unsafe raw query は lint または integration test で拒否される (ADMIN-CONSOLE-BE-S084)

- **WHEN** Admin account search repository が unsafe raw SQL または string concatenation を使用している
- **THEN** lint または integration test は SQL construction boundary violation として失敗する

### Requirement: 監査ログの OpenSearch インデックス

Admin audit event indexing は Go Admin backend の adapter/application 境界で実行されなければならない（SHALL）。Audit event の source of truth は Admin-owned schema の audit table であり、OpenSearch は検索用 projection として扱われなければならない（MUST）。OpenSearch namespace は Admin audit prefix と Product domain prefix を分離し、同一または包含関係の prefix を runtime config validation で拒否しなければならない（MUST）。`packages/admin` は OpenSearch client または server-side indexing logic を所有してはならず（MUST NOT）、Admin static frontend は Admin backend `/api/v1/*` 経由で audit data を取得しなければならない（SHALL）。Indexing failure は account mutation の DB transaction を成功後に取り消してはならないが、warning log、metric、retry queue または retry marker として観測可能でなければならない（SHALL）。

**Customer Context**

監査ログは大量に蓄積されるため検索 projection が必要になる。一方で Admin frontend package に OpenSearch 接続や secret を持たせると、静的 frontend 境界と secret 管理が崩れる。Go Admin backend が projection を管理し、DB audit table を source of truth にする必要がある。

#### Scenario: Admin audit event は Go backend から OpenSearch に projection される (ADMIN-CONSOLE-BE-S085)

- **GIVEN** Admin account creation audit event が Admin schema に保存済みである
- **WHEN** Admin audit indexing adapter が event を処理する
- **THEN** Admin audit prefix の OpenSearch index にだけ document が作成される
- **AND** `packages/admin` は OpenSearch client を import しない

#### Scenario: OpenSearch namespace collision は起動時に拒否される (ADMIN-CONSOLE-BE-S086)

- **GIVEN** Admin audit prefix と Product domain prefix が同一または包含関係である
- **WHEN** Admin backend runtime config validation を実行する
- **THEN** Admin backend は fail-close で起動を拒否する

#### Scenario: OpenSearch indexing failure は mutation 成功を取り消さず観測される (ADMIN-CONSOLE-BE-S087)

- **GIVEN** account mutation DB transaction と audit outcome update が成功済みである
- **WHEN** OpenSearch indexing が失敗する
- **THEN** mutation response は DB 成功結果を維持し、warning log、metric、retry queue または retry marker が記録される

### Requirement: セキュリティ Lint 制約

Repository lint は Admin static frontend と Go Admin backend の security boundary を強制しなければならない（SHALL）。DB 接続文字列、secret、password、token、key の長い literal を source code に埋め込んではならない（MUST NOT）。Admin Svelte source は unsafe HTML injection を使用してはならない（MUST NOT）。Admin backend repository は unsafe raw SQL または string concatenation query を使用してはならない（MUST NOT）。`packages/admin/app` は generated Admin SDK を直接 import してはならず、`packages/admin/domain` は Product SDK、DB client、server-only module、OpenSearch client、Valkey client を import してはならない（MUST NOT）。

**Customer Context**

Admin surface は顧客 PII、operator session、audit data を扱う。静的 frontend と Go backend の境界を lint で守り、secret 漏洩、XSS、SQL injection、SDK 誤用を release 前に検出する必要がある。

#### Scenario: Admin source の secret literal は lint エラーになる (ADMIN-CONSOLE-BE-S088)

- **WHEN** Admin frontend、backend、script source に DB connection string または token/key/password の長い literal が含まれる
- **THEN** lint または security scanner は secret literal violation として失敗する

#### Scenario: Admin Svelte source の unsafe HTML injection は lint エラーになる (ADMIN-CONSOLE-BE-S089)

- **WHEN** `packages/admin/app` の `.svelte` file が unsafe HTML injection を使用している
- **THEN** lint は XSS boundary violation として失敗する

#### Scenario: Admin backend unsafe SQL construction は lint エラーになる (ADMIN-CONSOLE-BE-S090)

- **WHEN** Admin backend repository が unsafe raw SQL または string concatenation query を使用している
- **THEN** lint または integration test は SQL injection boundary violation として失敗する

### Requirement: RBAC 権限チェックは Controller で強制される

Admin API の Controller / handler は全 protected operation で operator accessToken、operator session record、CSRF binding を application DTO に変換し、`internal/application/admin` の authorization use case を呼び出さなければならない（SHALL）。Permission map と `accounts:create` 判定は application layer が所有し、HTTP handler、repository、generated bindings、runtime composition、frontend package に置かれてはならない（MUST NOT）。`accounts:create` 権限は admin ロールと operator ロールに許可し、viewer ロールには許可してはならない（MUST NOT）。Product bearer token または Product account role は Admin RBAC の判定材料として使用してはならない（MUST NOT）。

**Customer Context**

Admin account 作成は顧客に直接影響する強権限操作である。UI の表示制御だけでは不十分であり、Admin backend の application authorization use case で RBAC を強制し、Controller / handler はその判定を必ず呼び出す必要がある。

#### Scenario: admin と operator は account 作成権限を持つ (ADMIN-CONSOLE-BE-S068)

- **GIVEN** Operator role が `admin` または `operator` である
- **WHEN** application authorization use case が `hasPermission(role, 'accounts:create')` を評価する
- **THEN** true が返される

#### Scenario: viewer は account 作成権限を持たない (ADMIN-CONSOLE-BE-S069)

- **GIVEN** Operator role が `viewer` である
- **WHEN** application authorization use case が `hasPermission('viewer', 'accounts:create')` を評価する
- **THEN** false が返される

### Requirement: MVCS 層間依存の強制

Admin backend code は Go backend の Clean Architecture boundary を守らなければならない（SHALL）。許可される依存方向は `cmd/* -> internal/app -> internal/adapter/* + internal/application/* + internal/platform/* -> internal/domain` であり、`internal/adapter/http/admin` と `internal/adapter/http/product` は物理的に分離されなければならない（MUST）。`internal/application/admin`、`internal/application/product`、`internal/application/shared/tokenprimitive` は物理的に分離され、Product application と Admin application は相互 import してはならない（MUST NOT）。Account domain invariants は concrete `internal/domain` object と application DTO/use case 境界に置かれ、HTTP handler、repository、generated bindings、runtime composition、frontend package に置かれてはならない（MUST NOT）。Admin account creation と Admin audit は application-layer use case として実装され、adapters は ports を実装するだけでなければならない（MUST）。Admin generated Go bindings は Admin HTTP adapter からのみ参照でき、Product binary、Product runtime、Product HTTP adapter、Product application から参照してはならない（MUST NOT）。

**Customer Context**

Admin backend を Go backend に集約することで Account domain rule を一元化する。一方で Product binary から Admin bindings や Admin handlers に到達できると、Product 側への Admin 機能露出につながる。

#### Scenario: Product binary から Admin bindings を import すると lint エラーになる (ADMIN-CONSOLE-BE-S070)

- **WHEN** Product API binary または Product runtime が Admin generated Go bindings を import している
- **THEN** lint は dependency boundary violation として失敗する

#### Scenario: Admin HTTP adapter が persistence adapter を import すると lint エラーになる (ADMIN-CONSOLE-BE-S072)

- **WHEN** `internal/adapter/http/admin` が `internal/adapter/postgres` または `internal/adapter/valkey` を import している
- **THEN** lint は adapter boundary violation として失敗する

#### Scenario: application port が adapter 型または generated 型を公開すると lint エラーになる (ADMIN-CONSOLE-BE-S073)

- **WHEN** `internal/application/admin` または `internal/application/shared/tokenprimitive` の port interface が adapter、GORM、Gin、generated binding 型を引数または戻り値に含める
- **THEN** lint は port purity violation として失敗する

#### Scenario: Account 不変条件を handler または repository に置くと lint エラーになる (ADMIN-CONSOLE-BE-S074)

- **WHEN** Admin account creation handler または repository が email 正規化、Account status 初期値、suspend / restore、sessionRevokedAfter などの Account domain rule を inline 実装している
- **THEN** lint または boundary test は concrete domain object bypass として失敗する

#### Scenario: Product application と Admin application は相互 import できない (ADMIN-CONSOLE-BE-S075)

- **WHEN** `internal/application/product` が `internal/application/admin` を import する、または `internal/application/admin` が `internal/application/product` を import する
- **THEN** lint は application surface boundary violation として失敗する

#### Scenario: persistence/session adapter namespace は Product/Admin で分離される (ADMIN-CONSOLE-BE-S076)

- **GIVEN** Product と Admin の repository/store adapter が初期化される
- **WHEN** DB schema、Valkey logical DB、key prefix を確認する
- **THEN** Admin adapter は Admin-owned schema と `admin:*` namespace だけを使用し、Product adapter は Product-owned persistence/session namespace だけを使用する
