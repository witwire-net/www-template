## MODIFIED Requirements

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
