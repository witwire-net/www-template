## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Admin Database Schema

Admin operator、operator passkey、audit event、Admin account management に必要な永続データは Product と同じ PostgreSQL database 内の Admin-owned schema に保持されなければならない（SHALL）。Admin schema の migration は `packages/backend/db/migrations/**` の backend migration system で管理され、Admin package の ORM migration に依存してはならない（MUST NOT）。Product runtime role は Admin schema の table / function / view へ権限を持ってはならない（MUST NOT）。Admin runtime role は Admin schema と必要な Account domain operation だけに最小権限でアクセスしなければならない（MUST）。Admin account mutation と audit intent / outcome は同じ PostgreSQL database 内で整合性を検証できなければならない（SHALL）。

**Customer Context**

Admin と Product の DB が物理分割されると、Account domain の状態変更と監査・operator 操作の整合性が分散し、保守負荷が高くなる。Admin 専用 schema と最小権限 role により、同一 DB で整合性を保ちながら Product runtime からの到達を制限する。

#### Scenario: Admin schema は Product DB 内に存在する (ADMIN-CONSOLE-BE-S059)

- **GIVEN** backend migration system が適用済みである
- **WHEN** Product DB の schema 一覧を確認する
- **THEN** Admin operator、operator passkey、audit event を保持する Admin-owned schema が存在する

#### Scenario: Product runtime role は Admin schema を参照できない (ADMIN-CONSOLE-BE-S060)

- **GIVEN** Product API runtime role で DB に接続している
- **WHEN** Admin audit table を SELECT しようとする
- **THEN** DB は権限不足として拒否する

### Requirement: Product Database 拡張

Product DB の Account root は `public.accounts` であり、Admin account management は Account root の永続不変条件と status / session revocation semantics を共有しなければならない（MUST）。Admin API が account lifecycle を操作する場合、Product API と同じ Account domain rule を使用し、Admin 専用の重複した Account domain rule を持ってはならない（MUST NOT）。Admin API から Product domain table を参照・更新する経路は backend repository と application service に限定され、Product runtime から Admin schema へ逆参照できてはならない（MUST NOT）。

**Customer Context**

Admin 経由の account create / suspend / restore が Product Account domain と別々の rule を持つと、顧客 account の状態と認証挙動が分岐する。Account domain は Backend に集約し、Admin は同じ root を安全に操作する必要がある。

#### Scenario: Admin account creation は Account domain rule を共有する (ADMIN-CONSOLE-BE-S067)

- **GIVEN** Product Account domain が email 正規化と重複禁止を定義している
- **WHEN** Admin account creation API が実行される
- **THEN** Product API と同じ Account domain validation が適用される

### Requirement: Service 層は全 mutation を監査 intent と outcome で記録する

Admin API は account create / suspend / restore、operator management、setup token rotation、passkey management の mutation 開始前に audit intent を記録しなければならない（MUST）。Mutation が成功した場合、audit outcome は `succeeded` で completed timestamp を持たなければならない（SHALL）。Mutation が domain error または infrastructure error で失敗した場合、audit outcome は `failed` と stable error code を保持しなければならない（SHALL）。Audit intent の作成に失敗した場合、Admin API は mutation を開始してはならない（MUST NOT）。

**Customer Context**

Admin mutation は顧客影響が大きく、成功だけでなく失敗も調査対象になる。同一 Product DB 上で intent と outcome を記録することで、Account domain の変更と監査の対応関係を追跡できる。

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

#### Scenario: Admin package の ORM migration は Product DB に適用されない (ADMIN-CONSOLE-BE-S061)

- **WHEN** repository の DB migration command policy を確認する
- **THEN** Admin 永続化 schema は backend migration system の対象であり、Admin package-local ORM migration command は使用されない

### Requirement: RBAC 権限チェックは Controller で強制される

Admin API の Controller / handler は全 protected operation で operator accessToken、operator session record、`requirePermission` 相当の権限チェックを実行しなければならない（SHALL）。`accounts:create` 権限は admin ロールと operator ロールに許可し、viewer ロールには許可してはならない（MUST NOT）。Product bearer token または Product account role は Admin RBAC の判定材料として使用してはならない（MUST NOT）。

**Customer Context**

Admin account 作成は顧客に直接影響する強権限操作である。UI の表示制御だけでは不十分であり、Admin backend の Controller 境界で RBAC を強制する必要がある。

#### Scenario: admin と operator は account 作成権限を持つ (ADMIN-CONSOLE-BE-S068)

- **GIVEN** Operator role が `admin` または `operator` である
- **WHEN** `hasPermission(role, 'accounts:create')` を評価する
- **THEN** true が返される

#### Scenario: viewer は account 作成権限を持たない (ADMIN-CONSOLE-BE-S069)

- **GIVEN** Operator role が `viewer` である
- **WHEN** `hasPermission('viewer', 'accounts:create')` を評価する
- **THEN** false が返される

### Requirement: MVCS 層間依存の強制

Admin backend code は Go backend の `cmd/admin-api -> internal/app -> internal/adapter/http -> internal/application -> internal/domain` の依存方向を守らなければならない（SHALL）。`packages/admin` は backend Model / Service / Controller 層を所有してはならない（MUST NOT）。Admin generated Go bindings は Admin API binary と Admin HTTP adapter からのみ参照でき、Product API binary から参照してはならない（MUST NOT）。

**Customer Context**

Admin backend を Go backend に集約することで Account domain rule を一元化する。一方で Product binary から Admin bindings や Admin handlers に到達できると、Product 側への Admin 機能露出につながる。

#### Scenario: Product binary から Admin bindings を import すると lint エラーになる (ADMIN-CONSOLE-BE-S070)

- **WHEN** Product API binary または Product runtime が Admin generated Go bindings を import している
- **THEN** lint は dependency boundary violation として失敗する
