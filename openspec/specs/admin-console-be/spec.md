## Purpose

Admin Console の backend requirements をまとめる。admin DB schema, Product DB extensions, audit intent/outcome recording, operator token management, Prisma client separation, OpenSearch indexing, migration management, query safety, MVCS constraints, security lint, and RBAC を対象とする。

## Requirements

### Requirement: Admin Database Schema

Admin Console は Prisma を ORM Mapper として使用し、`www-template_admin` DB にテーブルを保持しなければならない（SHALL）。`admin.operators` は id (ULID)、email (UNIQUE)、display_name、role（CHECK IN admin/operator/viewer）、is_active、setup_token_hash、setup_token_expires_at、last_login_at、created_at、updated_at を SHALL 保持する。`admin.operator_passkeys` は id (ULID)、operator_id (FK CASCADE)、credential_handle (UNIQUE)、public_key、sign_count (DEFAULT 0)、aaguid、backup_eligible、backup_state、transports (JSONB)、created_at を SHALL 保持する。`admin.audit_events` は id (ULID)、operator_id (FK)、action、target_type、target_id、details (JSONB)、outcome（CHECK IN pending/succeeded/failed/indeterminate）、error_code、ip_address、created_at、completed_at を SHALL 保持する。Admin DB schema は `packages/admin/prisma/admin/schema.prisma` と Prisma Migrate で管理しなければならない（SHALL）。

**Customer Context**

Admin は Product DB とは独立した DB を使用し、オペレーター管理と監査ログに必要なテーブルを保持する。

#### Scenario: Admin DB テーブルが Prisma Migrate で作成される (ADMIN-CONSOLE-BE-S001)

- **GIVEN** Admin DB Prisma migration 000001 が実行される
- **WHEN** テーブル一覧を確認する
- **THEN** `admin.operators`、`admin.operator_passkeys`、`admin.audit_events` が存在する

#### Scenario: 外部キー制約で cascade delete が働く (ADMIN-CONSOLE-BE-S002)

- **GIVEN** オペレーターが passkey credential を 1 件持つ
- **WHEN** そのオペレーターが削除される
- **THEN** 関連する `admin.operator_passkeys` レコードも削除される

#### Scenario: email の UNIQUE 制約が働く (ADMIN-CONSOLE-BE-S003)

- **GIVEN** `alice@example.com` のオペレーターが存在する
- **WHEN** 同じ email でオペレーターを作成しようとする
- **THEN** DB が UNIQUE 制約違反を返し、挿入は失敗する

#### Scenario: role の CHECK 制約が働く (ADMIN-CONSOLE-BE-S004)

- **GIVEN** operators テーブルの role カラムに CHECK IN (admin,operator,viewer) 制約がある
- **WHEN** 無効な role 値で INSERT を試みる
- **THEN** DB が CHECK 制約違反を返す

#### Scenario: Admin DB migration は初期オペレーターを作成しない (ADMIN-CONSOLE-BE-S005)

- **GIVEN** Admin DB Prisma migration 000001 が実行される
- **WHEN** `admin.operators` を確認する
- **THEN** レコードは 0 件であり、初回オペレーターは `/setup` の初回起動セットアップでのみ作成される

#### Scenario: sign_count のデフォルトが 0 である (ADMIN-CONSOLE-BE-S006)

- **GIVEN** operator_passkeys テーブルが作成されている
- **WHEN** passkey credential を sign_count 指定なしで挿入する
- **THEN** sign_count が 0 として保存される

---

### Requirement: Product Database 拡張

Admin Console は `www-template` DB に対して Account root 中心のアカウント管理用拡張を適用しなければならない（SHALL）。Product DB の Account root は `public.accounts` であり、Account.Auth の passkey credential は `public.account_passkey_credentials` でなければならない（MUST）。Admin Console 用 view は `public.account_passkey_credentials` を参照し、`public.passkey_credentials` を参照してはならない（MUST NOT）。`accounts.status`（TEXT NOT NULL DEFAULT 'active', CHECK IN ('active','suspended')）、`status_reason`、`status_updated_at`、`status_updated_by`、`session_revoked_after` カラムを SHALL 保持する。`admin_view.account_summaries` および `admin_view.account_passkeys` ビューを SHALL 作成する。`admin_view.account_summaries` は Account ごとの email / status / passkey_count を返し、passkey_count は `account_passkey_credentials` に基づかなければならない（MUST）。`admin_view.account_passkeys` は Account と `account_passkey_credentials` を結合して passkey 詳細と Account email を返さなければならない（MUST）。`admin_op.suspend_account(p_account_id, p_operator_id, p_reason, p_audit_event_id) RETURNS TEXT` および `admin_op.restore_account(p_account_id, p_operator_id, p_audit_event_id) RETURNS TEXT` 関数を SHALL 作成する。`suspend_account` は status を `suspended` に更新し、同一 Product DB transaction 内で `session_revoked_after` を現在時刻以上に更新しなければならない（MUST）。`restore_account` は status を `active` に戻すが、`session_revoked_after` を NULL または過去値に戻してはならない（MUST NOT）。関数は `SECURITY DEFINER` で作成し、関数内で `SET search_path = pg_catalog, admin_op` を MUST 指定する。SECURITY DEFINER 関数内で Product base table または Product schema object を参照する場合、`public.accounts` のように必ず schema-qualified name を使用しなければならない（MUST）。migration は `admin_console_read` と `admin_console_write` の NOLOGIN role を作成し、`admin_console_read` に `admin_view` schema USAGE と view SELECT を GRANT し、`admin_console_write` に `admin_console_read` を GRANT したうえで `admin_op` schema USAGE と管理関数 EXECUTE を GRANT しなければならない（MUST）。PUBLIC からの実行権限は REVOKE し、`admin_console_write` にのみ GRANT EXECUTE しなければならない（MUST）。環境別 login role は migration では固定名作成せず release 手順で作成し、`GRANT admin_console_write TO <product_admin_login_role>` を実行しなければならない（MUST）。`PRODUCT_DATABASE_URL` が使用する login role は `admin_console_write` の member でなければならず、base table owner や superuser であってはならない（MUST NOT）。これらの migration は `packages/backend/db/migrations/` に配置し、golang-migrate の Product DB migration と統合しなければならない（SHALL）。

**Customer Context**

Admin は Product DB に対して Account ライフサイクル管理のための拡張を必要とする。管理用スキーマは Product アプリケーションから参照されない。SECURITY DEFINER 関数は search_path 固定・スキーマ修飾・PUBLIC 権限剥奪・最小権限 role により安全に隔離する。Admin view が Account.Auth child table を正しく参照することで、管理者は Account と passkey の関係を誤認せず、停止・復旧判断を安全に行える。

#### Scenario: Account status が active になる (ADMIN-CONSOLE-BE-S007)

- **GIVEN** Product DB Account root migration が実行される
- **WHEN** `accounts` テーブルの全レコードを確認する
- **THEN** 全レコードの `status` が `active` であり、`session_revoked_after` は NULL である

#### Scenario: suspend_account が active アカウントを停止する (ADMIN-CONSOLE-BE-S008)

- **GIVEN** アカウントの status が `active` である
- **WHEN** `SELECT admin_op.suspend_account('<id>', '<op_id>', '不正利用', '<audit_event_id>')` を実行する
- **THEN** 戻り値がアカウント ID であり、当該アカウントの status が `suspended` に更新され、`session_revoked_after` が設定される

#### Scenario: suspend_account が非 active アカウントで例外を throw する (ADMIN-CONSOLE-BE-S009)

- **GIVEN** アカウントの status が `suspended` である
- **WHEN** `SELECT admin_op.suspend_account(...)` を実行する
- **THEN** `account_not_active` 例外が throw される

#### Scenario: restore_account が suspended アカウントを復旧する (ADMIN-CONSOLE-BE-S010)

- **GIVEN** アカウントの status が `suspended` である
- **WHEN** `SELECT admin_op.restore_account('<id>', '<op_id>', '<audit_event_id>')` を実行する
- **THEN** 戻り値がアカウント ID であり、status が `active`、status_reason が NULL に更新され、`session_revoked_after` は維持される

#### Scenario: restore_account が非 suspended アカウントで例外を throw する (ADMIN-CONSOLE-BE-S011)

- **GIVEN** アカウントの status が `active` である
- **WHEN** `SELECT admin_op.restore_account(...)` を実行する
- **THEN** `account_not_suspended` 例外が throw される

#### Scenario: admin_view.account_summaries が全アカウントを返す (ADMIN-CONSOLE-BE-S012)

- **GIVEN** accounts に 5 件のレコードが存在し、account_passkey_credentials が Account に従属している
- **WHEN** `SELECT * FROM admin_view.account_summaries` を実行する
- **THEN** 5 件のレコードが返され、各レコードに email / status / account_passkey_credentials に基づく passkey_count が含まれる

#### Scenario: admin_view.account_passkeys が Account.Auth passkey 情報を返す (ADMIN-CONSOLE-BE-S013)

- **GIVEN** accounts に紐づく account_passkey_credentials が存在する
- **WHEN** `SELECT * FROM admin_view.account_passkeys` を実行する
- **THEN** passkey の詳細と紐づく account の email が返され、view 定義は passkey_credentials を参照しない

#### Scenario: SECURITY DEFINER 関数が search_path を固定している (ADMIN-CONSOLE-BE-S037)

- **GIVEN** `admin_op.suspend_account` 関数が作成されている
- **WHEN** 関数定義を確認する
- **THEN** 関数内で `SET search_path = pg_catalog, admin_op` が指定されている
- **AND** Product base table 参照は `public.accounts` のように schema-qualified name で記述されている

#### Scenario: PUBLIC から SECURITY DEFINER 関数の実行権限が剥奪されている (ADMIN-CONSOLE-BE-S038)

- **GIVEN** `admin_op.suspend_account` 関数が作成されている
- **WHEN** 権限を確認する
- **THEN** PUBLIC の EXECUTE 権限が REVOKE されている

#### Scenario: admin_console_read は admin_view の SELECT のみ許可される (ADMIN-CONSOLE-BE-S042)

- **GIVEN** `admin_console_read` role が作成されている
- **WHEN** role 権限を確認する
- **THEN** `admin_view` の USAGE と view SELECT が許可され、`admin_op` 関数 EXECUTE は許可されていない

#### Scenario: admin_console_write のみ admin_op 関数を実行できる (ADMIN-CONSOLE-BE-S043)

- **GIVEN** `admin_op.suspend_account` 関数が作成されている
- **WHEN** role 権限を確認する
- **THEN** EXECUTE は `admin_console_write` にのみ GRANT され、PUBLIC と `admin_console_read` には許可されていない

#### Scenario: PRODUCT_DATABASE_URL の login role は最小権限 role を使う (ADMIN-CONSOLE-BE-S044)

- **GIVEN** Admin Console が Product DB に接続する
- **WHEN** 現在の DB role と membership を確認する
- **THEN** login role は `admin_console_write` の member であり、superuser ではなく、base table owner でもない
- **AND** release 手順で環境別 login role に `admin_console_write` が GRANT 済みである

### Requirement: Service 層は全 mutation を監査 intent と outcome で記録する

suspendAccount / restoreAccount / createOperator / updateOperatorRole / deactivateOperator の各 Service 関数は、外部 side effect を開始する前に `admin.audit_events` へ outcome=`pending` の audit intent を MUST 挿入する。監査イベントには operator_id / action / target_type / target_id / details (JSONB) / outcome / error_code / ip_address / completed_at を SHALL 含める。pending audit intent の挿入に失敗した場合、Service は Product DB mutation や OpenSearch 連携を開始してはならず（MUST NOT）、503 エラーを返さなければならない。Product DB と Admin DB は別 DB であるため、Service は Product DB mutation 成功後に Admin DB 失敗を分散 rollback できると仮定してはならない（MUST NOT）。Product DB mutation が失敗した場合、Service は audit outcome を `failed` に更新し、stable `error_code` と `completed_at` を記録してから domain error または 5xx を返さなければならない（MUST）。この failed outcome 更新に失敗した場合、既存 pending event を reconciliation 対象として残し、structured error log と metric を MUST 出力する。Product DB mutation 成功後は audit outcome を `succeeded` へ更新し、更新に失敗した場合は Product DB mutation を戻そうとせず、既存 pending event を reconciliation 対象として残し、structured error log と metric を MUST 出力する。

**Customer Context**

Admin Console 上の全状態変更操作は追跡可能でなければならない。Admin DB と Product DB は別 DB で分散 transaction を持たないため、特権操作の開始前に audit intent を永続化して未監査 mutation を防ぐ。Product DB mutation が失敗した事実も監査対象であり、pending のままでは調査時に「未完了」と「失敗」を区別できない。Product DB mutation 成功後の outcome 更新失敗は rollback ではなく reconciliation で検出・補正する。

#### Scenario: 停止操作が監査ログに記録される (ADMIN-CONSOLE-BE-S014)

- **GIVEN** suspendAccount が成功する
- **WHEN** `admin.audit_events` を確認する
- **THEN** action=`account.suspend`、target_type=`account`、target_id=対象アカウントID、outcome=`succeeded` のレコードが挿入されている
- **AND** details に `reason` が含まれている

#### Scenario: 復旧操作が監査ログに記録される (ADMIN-CONSOLE-BE-S015)

- **GIVEN** restoreAccount が成功する
- **WHEN** `admin.audit_events` を確認する
- **THEN** action=`account.restore`、outcome=`succeeded` のレコードが挿入されている

#### Scenario: オペレーターロール変更が監査ログに記録される (ADMIN-CONSOLE-BE-S016)

- **GIVEN** updateOperatorRole が成功する
- **WHEN** `admin.audit_events` を確認する
- **THEN** action=`operator.update_role`、target_type=`operator`、target_id=対象 Operator ID、outcome=`succeeded` のレコードが挿入されている
- **AND** details に `from_role` と `to_role` が含まれている

#### Scenario: audit intent 作成失敗時は mutation を開始しない (ADMIN-CONSOLE-BE-S017)

- **GIVEN** `admin.audit_events` への pending audit intent INSERT が DB エラーで失敗する
- **WHEN** suspendAccount が実行される
- **THEN** Product DB のアカウント停止は実行されず、503 エラーが返される

#### Scenario: outcome 更新失敗時は pending audit event を reconciliation 対象に残す (ADMIN-CONSOLE-BE-S052)

- **GIVEN** Product DB の suspend は成功したが、`admin.audit_events.outcome` の `succeeded` 更新が DB エラーで失敗する
- **WHEN** suspendAccount が完了処理を行う
- **THEN** Product DB mutation は rollback されず、pending audit event が残り、structured error log と metric が出力される

#### Scenario: Product DB mutation 失敗時は failed outcome を記録する (ADMIN-CONSOLE-BE-S055)

- **GIVEN** pending audit intent は作成済みだが Product DB の `admin_op.suspend_account` が domain error または DB error で失敗する
- **WHEN** suspendAccount が error handling を行う
- **THEN** `admin.audit_events.outcome` は `failed` に更新され、stable `error_code` と `completed_at` が保存される
- **AND** Product DB mutation 失敗時に OpenSearch index は実行されない
- **AND** failed outcome 更新も失敗した場合は pending audit event が reconciliation 対象として残り、structured error log と metric が出力される

---

### Requirement: オペレーター追加は one-time setup token を発行する

createOperator は email / display_name / role を保存すると同時に、24 時間以内に期限切れとなる one-time setup token を MUST 発行し、bcrypt hash を `admin.operators.setup_token_hash` に保存しなければならない（SHALL）。生 token は作成 response で 1 回だけ表示され、DB、監査ログ、OpenSearch、application log に平文保存されてはならない（MUST NOT）。setup token の再発行は admin ロールのみが実行でき、既存 token を無効化して新 token hash と expiry を保存し、監査ログに `operator.setup_token.rotate` を MUST 記録する。passkey 登録済みオペレーターへの token 再発行は MUST 拒否する。自分自身の無効化、最後の admin の無効化、最後の admin の降格は MUST 拒否する。

**Customer Context**

admin が新しいオペレーターを追加しても、初回 passkey 登録のための token がなければログインできない。メール配送は out of scope でも、admin が安全に one-time token をコピーして別経路で渡せる導線が必要である。

#### Scenario: オペレーター追加時に setup token が一度だけ表示される (ADMIN-CONSOLE-BE-S045)

- **GIVEN** admin が新規オペレーターを追加する
- **WHEN** createOperator が成功する
- **THEN** response は平文 setup token を 1 回だけ含み、DB には bcrypt hash と expiry のみが保存される

#### Scenario: setup token 再発行は既存 token を無効化して監査される (ADMIN-CONSOLE-BE-S046)

- **GIVEN** オペレーターが passkey 未登録で既存 setup token を持つ
- **WHEN** admin が setup token を再発行する
- **THEN** 旧 token は無効化され、新 token hash が保存され、`operator.setup_token.rotate` audit event が記録される

#### Scenario: passkey 登録済みオペレーターの setup token 再発行は拒否される (ADMIN-CONSOLE-BE-S047)

- **GIVEN** オペレーターが少なくとも 1 件の passkey credential を持つ
- **WHEN** admin が setup token を再発行しようとする
- **THEN** server は 400 を返し、setup token は変更されない

#### Scenario: 最後の admin は無効化できない (ADMIN-CONSOLE-BE-S048)

- **GIVEN** 有効な admin ロールのオペレーターが 1 件のみ存在する
- **WHEN** admin がそのオペレーターを無効化しようとする
- **THEN** server は 400 を返し、is_active は変化しない

#### Scenario: 最後の admin は降格できない (ADMIN-CONSOLE-BE-S049)

- **GIVEN** 有効な admin ロールのオペレーターが 1 件のみ存在する
- **WHEN** admin がそのオペレーターの role を `operator` または `viewer` に変更しようとする
- **THEN** server は 400 を返し、role は変化しない

---

### Requirement: Prisma Client 管理

Infrastructure 層の db module は `getAdminPrisma()`、`getProductPrisma()`、`validateProductDbRuntimeRole()` を SHALL 提供する。Admin Prisma Client は `ADMIN_DATABASE_URL`、Product Prisma Client は `PRODUCT_DATABASE_URL` を使用しなければならない（MUST）。Product Prisma Client 初期化時は `validateProductDbRuntimeRole()` を実行し、current role が `admin_console_write` member であり、superuser ではなく、base table owner でもないことを検証しなければならない（MUST）。検証に失敗した場合は fail-close し、Product DB query を実行してはならない（MUST NOT）。2 つの Prisma Client は別 generated output として生成し、Admin DB 操作用 model が Product DB に接続できない構成にしなければならない（MUST）。Product DB への schema 変更に Prisma Migrate を使ってはならず（MUST NOT）、Product DB Prisma schema は既存 golang-migrate 適用後の構造を参照するためだけに使用する。

**Customer Context**

Admin は 2 つの PostgreSQL DB に接続するため、Prisma Client を DB ごとに分離して誤接続と誤 migration を防ぐ。Product DB は既存 backend と同じ migration 境界を守り、Admin は ORM Mapper としてのみ Prisma を使う。

#### Scenario: Admin DB にクエリできる (ADMIN-CONSOLE-BE-S018)

- **GIVEN** `ADMIN_DATABASE_URL` 環境変数が有効である
- **WHEN** `getAdminPrisma()` を呼び出してクエリを実行する
- **THEN** 有効な結果が返される

#### Scenario: Product DB にクエリできる (ADMIN-CONSOLE-BE-S019)

- **GIVEN** `PRODUCT_DATABASE_URL` 環境変数が有効である
- **WHEN** `getProductPrisma()` を呼び出してクエリを実行する
- **THEN** 有効な結果が返される

#### Scenario: DB 接続失敗時にエラーが throw される (ADMIN-CONSOLE-BE-S020)

- **GIVEN** `ADMIN_DATABASE_URL` の connectionString が無効である
- **WHEN** `getAdminPrisma()` でクエリを実行する
- **THEN** エラーが throw される

#### Scenario: Product DB に Prisma Migrate を適用しない (ADMIN-CONSOLE-BE-S050)

- **GIVEN** Product DB Prisma schema が存在する
- **WHEN** Admin の DB command を確認する
- **THEN** Product DB に対して `prisma migrate` を実行する script は存在せず、Product DB 拡張は `PRODUCT_DATABASE_URL="$PRODUCT_DATABASE_URL" pnpm db:migrate:product` でのみ適用される

---

### Requirement: 監査ログの OpenSearch インデックス

監査イベントは Admin DB の `admin.audit_events` に永続化された後、Admin audit OpenSearch namespace に非同期でインデックスされなければならない（SHALL）。OpenSearch は `OPENSEARCH_URL` の単一接続情報を使用してよいが、Admin audit namespace は `ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX`、Production domain namespace は `PRODUCT_OPENSEARCH_INDEX_PREFIX` で命名し、namespace を混在させてはならない（MUST NOT）。インデックスされるドキュメントは id / operator_id / operator_email / operator_name / action / target_type / target_id / details / details_json / ip_address / created_at を MUST 含む。OpenSearch へのインデックス失敗は監査 mutation 自体を失敗させてはならず（MUST NOT）、警告ログを出力するのみでなければならない（SHALL）。監査ログ検索画面は OpenSearch が利用可能な場合は Admin audit namespace のみを検索し、利用不能な場合は Admin DB fallback 検索を SHALL 使用する。OpenSearch インデックスは `${ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX}-YYYY.MM` 形式の月次インデックスで管理されなければならない（SHALL）。index template `${ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX}-*` は `dynamic: strict` とし、id / operator_id / operator_email / action / target_type / target_id は keyword、operator_name は text + keyword subfield、ip_address は ip、created_at は date、details_json は text、details は enabled=false object として mapping しなければならない（MUST）。primary shard は 1、replica 数は `ADMIN_OPENSEARCH_AUDIT_REPLICAS` で設定しなければならない（SHALL）。Admin audit prefix と Production domain prefix は同一または包含関係であってはならない（MUST NOT）。Admin Console は Product DB の `admin_view.*` / `admin_op.*` を通じて Production account lifecycle を参照・操作でき、Production domain OpenSearch を使うユースケースも許可される。ただし Admin audit document は Admin audit prefix にのみ保存し、Production domain index へ書き込んだり、Production domain index を Admin 監査ログ検索に混在させてはならない（MUST NOT）。OpenSearch index name は namespace builder だけで生成し、route / service / model が raw index name、wildcard index pattern、comma-separated multi index、`_all` を直接指定してはならない（MUST NOT）。

**Customer Context**

監査ログは長期間にわたって大量に蓄積される。OpenSearch にインデックスすることで、高速な全文検索・フィルター・集計が可能になる。物理 OpenSearch cluster と接続情報を単一にすると運用は単純になるが、Admin ドメインの監査ログと Production ドメインの検索 index を混在させると、保管期間・mapping 変更・検索結果の影響範囲が崩れる。namespace builder と prefix 検証により、単一接続でも用途別 namespace が混じらないようにする。OpenSearch 障害時も Admin DB fallback により監査ログ閲覧の可用性を維持する。

#### Scenario: 監査イベントが OpenSearch にインデックスされる (ADMIN-CONSOLE-BE-S039)

- **GIVEN** suspendAccount が成功し、`admin.audit_events` にレコードが挿入される
- **WHEN** インデックス完了後
- **THEN** 当該イベントが `${ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX}-YYYY.MM` インデックスに存在する

#### Scenario: OpenSearch インデックス失敗時も mutation は成功する (ADMIN-CONSOLE-BE-S040)

- **GIVEN** OpenSearch が応答しない
- **WHEN** suspendAccount が実行される
- **THEN** アカウント停止と DB 監査ログ記録は成功し、503 にはならない
- **AND** OpenSearch 接続エラーが警告ログに出力される

#### Scenario: OpenSearch 障害時に DB fallback 検索が使われる (ADMIN-CONSOLE-BE-S041)

- **GIVEN** OpenSearch が応答しない
- **WHEN** 監査ログ一覧を取得する
- **THEN** Admin DB の `admin.audit_events` テーブルから検索結果が返され、Production ドメイン用 index は参照されない

#### Scenario: Admin audit namespace は Production domain namespace と分離される (ADMIN-CONSOLE-BE-S053)

- **GIVEN** Admin Console が OpenSearch client を初期化する
- **WHEN** runtime config を検証する
- **THEN** `OPENSEARCH_URL`、`ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX`、`PRODUCT_OPENSEARCH_INDEX_PREFIX` は設定必須である
- **AND** Admin audit prefix と Production domain prefix が同一または包含関係の場合は起動を拒否する
- **AND** Admin audit event は Admin audit prefix の index にのみ書き込まれ、Production domain index には書き込まれない

#### Scenario: OpenSearch namespace の混在 query は拒否される (ADMIN-CONSOLE-BE-S054)

- **GIVEN** Admin audit namespace と Production domain namespace が設定済みである
- **WHEN** raw index name、`_all`、wildcard only pattern、comma-separated multi index、または Admin audit と Production domain を同時に対象にする query を作成しようとする
- **THEN** namespace builder は request を拒否し、OpenSearch client へ query を送信しない
- **AND** Production domain OpenSearch を使うユースケースは Production domain prefix のみを対象にし、Admin audit prefix を参照しない

---

### Requirement: Migration Management

Admin DB schema は `packages/admin/prisma/admin/migrations/**/migration.sql` を Prisma Migrate で順次適用しなければならない（SHALL）。適用済み migration は Prisma の migration metadata により再実行してはならない（MUST NOT）。Product DB 拡張の migration は `packages/backend/db/migrations/` に配置し、既存の golang-migrate で管理されなければならない（SHALL）。Product DB に対して Prisma Migrate を実行してはならない（MUST NOT）。

**Customer Context**

Admin の DB スキーマは Prisma schema と migration ファイルでバージョン管理される。Product DB 拡張は既存の Go backend migration 管理と統合し、Prisma は ORM Mapper としてのみ使用する。

#### Scenario: 未適用の migration が実行される (ADMIN-CONSOLE-BE-S021)

- **GIVEN** Prisma migration metadata に 000001 のみ記録されている
- **WHEN** `prisma migrate deploy --schema packages/admin/prisma/admin/schema.prisma` を実行する
- **THEN** 未適用の Admin DB migration のみが実行され、Prisma migration metadata に記録される

#### Scenario: 全 migration 適用済みの場合は何も実行されない (ADMIN-CONSOLE-BE-S022)

- **GIVEN** すべての migration が Prisma migration metadata に記録されている
- **WHEN** `prisma migrate deploy --schema packages/admin/prisma/admin/schema.prisma` を実行する
- **THEN** 新しい SQL は実行されず、DB は変更されない

#### Scenario: Product DB 拡張 migration は golang-migrate で管理される (ADMIN-CONSOLE-BE-S023)

- **GIVEN** Product DB 拡張 migration が `packages/backend/db/migrations/` に配置されている
- **WHEN** `golang-migrate up` が実行される
- **THEN** Product DB 拡張 migration が `www-template` に対して適用される

---

### Requirement: アカウント検索はページネーションと入力検証を持つ

全 DB クエリは Prisma Client の型付き query または parameterized `$queryRaw` / `$executeRaw` を MUST 使用する。`$queryRawUnsafe` / `$executeRawUnsafe` は MUST NOT 使用する。`limit` パラメータは 1〜100 を MUST 検証し、`offset` は 0 以上を MUST 検証する。メール検索文字列は最大 255 文字を MUST 許可する。

**Customer Context**

Admin のアカウント検索は Product DB に対して直接クエリを発行するため、安全なクエリ構築が必須である。

#### Scenario: 範囲外の limit は拒否される (ADMIN-CONSOLE-BE-S024)

- **GIVEN** 検索 API に `limit=0` が渡される
- **WHEN** Controller がパラメータを検証する
- **THEN** 400 エラーが返される

#### Scenario: 負の offset は拒否される (ADMIN-CONSOLE-BE-S025)

- **GIVEN** 検索 API に `offset=-1` が渡される
- **WHEN** Controller がパラメータを検証する
- **THEN** 400 エラーが返される

#### Scenario: SQL injection 攻撃は Prisma の parameterized query で防止される (ADMIN-CONSOLE-BE-S026)

- **GIVEN** 検索文字列に `'; DROP TABLE accounts; --` が含まれている
- **WHEN** Prisma の型付き query または parameterized `$queryRaw` が実行される
- **THEN** 検索文字列はリテラルとして扱われ、SQL injection は発生しない

---

### Requirement: MVCS 層間依存の強制

Model 層は `$app/*` および services 層を import してはならない（MUST NOT）。Service 層は View 層を import してはならない（MUST NOT）。Controller 層は View 層を import してはならない（MUST NOT）。Admin 全体は `@www-template/api`、`@www-template/domain`、`@www-template/app`、`@www-template/web` を import してはならない（MUST NOT）。これらの制約は ESLint で強制されなければならない（SHALL）。

**Customer Context**

MVCS アーキテクチャの一貫性を保つため、各層が許可された下位層のみに依存することを lint で保証する。

#### Scenario: Model から services を import すると lint エラー (ADMIN-CONSOLE-BE-S027)

- **WHEN** Model ファイルが `$lib/server/services/` を import している
- **THEN** ESLint がエラーを報告する

#### Scenario: Service から components を import すると lint エラー (ADMIN-CONSOLE-BE-S028)

- **WHEN** Service ファイルが `$lib/components/` を import している
- **THEN** ESLint がエラーを報告する

#### Scenario: Admin から @www-template/api を import すると lint エラー (ADMIN-CONSOLE-BE-S029)

- **WHEN** Admin 内の任意のファイルが `@www-template/api` を import している
- **THEN** ESLint がエラーを報告する

---

### Requirement: セキュリティ Lint 制約

DB 接続文字列（`postgres://...` 形式）をソースコードにハードコードしてはならない（MUST NOT）。`.svelte` ファイルで `@html` ディレクティブを使用してはならない（MUST NOT）。Model 層で SQL テンプレートリテラルを使用してはならない（MUST NOT）。Prisma の `$queryRawUnsafe` / `$executeRawUnsafe` を使用してはならない（MUST NOT）。シークレット変数名（secret / password / token / key を含む）に長いリテラル値を代入してはならない（MUST NOT）。`innerHTML` プロパティに直接代入してはならない（MUST NOT）。これらの制約は ESLint で強制されなければならない（SHALL）。

**Customer Context**

Admin はシステム全体で最も特権的な画面である。シークレット漏洩・XSS・SQL injection のリスクを lint レベルで防止する。

#### Scenario: ハードコード接続文字列が lint エラーになる (ADMIN-CONSOLE-BE-S030)

- **WHEN** ソースコードに `postgres://user:pass@host:5432/db` 形式のリテラルが含まれている
- **THEN** ESLint がエラーを報告する

#### Scenario: @html ディレクティブが lint エラーになる (ADMIN-CONSOLE-BE-S031)

- **WHEN** `.svelte` ファイルに `{@html content}` が含まれている
- **THEN** ESLint がエラーを報告する

#### Scenario: SQL テンプレートリテラルが lint エラーになる (ADMIN-CONSOLE-BE-S032)

- **WHEN** Model ファイルにパラメータ化されていない SQL テンプレートリテラルが含まれている
- **THEN** ESLint がエラーを報告する

#### Scenario: Prisma unsafe raw query が lint エラーになる (ADMIN-CONSOLE-BE-S051)

- **WHEN** Model ファイルで `$queryRawUnsafe` または `$executeRawUnsafe` を使用している
- **THEN** ESLint がエラーを報告する

---

### Requirement: RBAC 権限チェックは Controller で強制される

Controller の全 action 関数は `requirePermission(locals.operator, '<permission>')` を SHALL 呼び出す。`requirePermission` は権限不足時に `error(403)` を SHALL throw する。operators:read 権限は admin ロールのみに許可されなければならない（SHALL）。

**Customer Context**

RBAC の強制は Controller 層で行い、Service 層はビジネスロジックに集中できる。Operator 管理は admin 専用である。

**Requirement**

権限マップ:

| 権限                  | admin | operator | viewer |
| --------------------- | ----- | -------- | ------ |
| accounts:read         | ○     | ○        | ○      |
| accounts:suspend      | ○     | ○        | ×      |
| accounts:restore      | ○     | ○        | ×      |
| audit:read            | ○     | ○        | ○      |
| operators:read        | ○     | ×        | ×      |
| operators:write       | ○     | ×        | ×      |
| operators:setup_token | ○     | ×        | ×      |
| operators:deactivate  | ○     | ×        | ×      |

#### Scenario: admin が全権限を持つ (ADMIN-CONSOLE-BE-S033)

- **GIVEN** ロールが `admin` である
- **WHEN** `hasPermission('admin', perm)` を全定義済み権限で呼び出す
- **THEN** すべて true が返される

#### Scenario: viewer が accounts:suspend 権限を持たない (ADMIN-CONSOLE-BE-S034)

- **GIVEN** ロールが `viewer` である
- **WHEN** `hasPermission('viewer', 'accounts:suspend')` を呼び出す
- **THEN** false が返される

#### Scenario: 権限不足で requirePermission が 403 を throw する (ADMIN-CONSOLE-BE-S035)

- **GIVEN** Operator のロールが `viewer` である
- **WHEN** `requirePermission(operator, 'accounts:suspend')` が呼び出される
- **THEN** `error(403, 'Insufficient permissions')` が throw される

#### Scenario: 未定義の権限は false を返す (ADMIN-CONSOLE-BE-S036)

- **WHEN** `hasPermission('admin', 'nonexistent:perm')` を呼び出す
- **THEN** false が返される
