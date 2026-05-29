## Purpose

Admin Console の backend requirements をまとめる。Admin-owned schema、Account root management、audit intent/outcome recording、operator token management、OpenSearch projection、backend migration management、query safety、security lint、RBAC を対象とする。

## Requirements

### Requirement: Admin-owned schema

Admin Console は `packages/backend/db/migrations/**` の backend migration system で `admin` schema を管理しなければならない（SHALL）。`admin.operators`、`admin.operator_passkeys`、`admin.audit_events` は backend migration で作成され、`packages/admin/prisma/**` や package-local ORM migration で管理してはならない（MUST NOT）。

**Customer Context**

Admin operator、passkey、audit event を Account root と同じ DB transaction 境界で扱えるようにし、物理的に分かれた管理用 database の運用・整合性リスクを残さない。

#### Scenario: Admin-owned schema table が backend migration で作成される (ADMIN-CONSOLE-BE-S001)

- **GIVEN** backend migration が実行される
- **WHEN** schema と table 一覧を確認する
- **THEN** `admin.operators`、`admin.operator_passkeys`、`admin.audit_events` が存在する

#### Scenario: backend migration は初期オペレーターを作成しない (ADMIN-CONSOLE-BE-S005)

- **GIVEN** backend migration が実行される
- **WHEN** `admin.operators` を確認する
- **THEN** レコードは 0 件であり、初回オペレーターは bootstrap setup flow でのみ作成される

### Requirement: Account root management

Admin account management は `public.accounts`、`public.account_settings`、`admin.audit_events` を同一 commit 境界で扱わなければならない（MUST）。Account root の不変条件、status、session revocation semantics は Product API と同じ domain rule を使い、Admin 専用の重複 domain rule を持ってはならない（MUST NOT）。

**Customer Context**

Admin 操作で作成・停止・復旧される顧客アカウントは、通常の Account と同じルールで扱われる必要がある。監査と Account mutation を同じ DB 境界に置くことで、成功・失敗の追跡と復旧判断を安全に行える。

#### Scenario: Admin schema と Account root は同じ backend migration で管理される (ADMIN-CONSOLE-BE-S059)

- **GIVEN** `000007_create_admin_schema.up.sql` が適用済みである
- **WHEN** DB の schema 一覧を確認する
- **THEN** `admin` schema と `public.accounts` が同じ DB に存在する

#### Scenario: Admin package の ORM migration は使われない (ADMIN-CONSOLE-BE-S061)

- **GIVEN** repository の migration command policy を確認する
- **WHEN** DB schema 変更の migration source を確認する
- **THEN** `packages/backend/db/migrations/**` だけが DB schema migration として使われる
- **AND** `packages/admin/prisma/**`、package-local Admin ORM migration は使われない

### Requirement: Audit intent and outcome

Admin mutation は mutation 前に `admin.audit_events` へ pending intent を挿入し、成功時は succeeded、失敗時は stable error code 付き failed へ更新しなければならない（MUST）。pending intent の作成に失敗した場合、Account mutation を開始してはならない（MUST NOT）。

**Customer Context**

顧客影響が大きい管理操作は、成功だけでなく失敗も追跡できなければならない。未監査 mutation を防ぐため、intent 作成を mutation の前提にする。

#### Scenario: audit intent 作成失敗時は mutation を開始しない (ADMIN-CONSOLE-BE-S017)

- **GIVEN** `admin.audit_events` への pending audit intent INSERT が DB エラーで失敗する
- **WHEN** suspendAccount が実行される
- **THEN** アカウント停止は実行されず、503 エラーが返される

### Requirement: OpenSearch projection

Admin audit event indexing は Go Admin backend の adapter/application 境界で実行されなければならない（SHALL）。Audit event の source of truth は `admin.audit_events` であり、OpenSearch は検索用 projection として扱われなければならない（MUST）。`packages/admin` は OpenSearch client または server-side indexing logic を所有してはならない（MUST NOT）。

**Customer Context**

監査ログ検索は高速である必要があるが、検索 projection の失敗で Account mutation 成功を取り消してはならない。source of truth を DB に固定し、OpenSearch は観測可能な projection として扱う。

#### Scenario: OpenSearch インデックス失敗時も mutation は成功する (ADMIN-CONSOLE-BE-S040)

- **GIVEN** OpenSearch が応答しない
- **WHEN** Admin mutation が実行される
- **THEN** DB mutation と監査ログ記録は成功し、OpenSearch 接続エラーが warning として観測される

### Requirement: Query safety and RBAC

Admin account search は backend application use case で pagination と入力を検証し、unsafe raw SQL construction を使ってはならない（MUST NOT）。`accounts:create` などの authorization decision は Admin auth/application 層で行い、UI 表示制御だけで代替してはならない（MUST NOT）。

**Customer Context**

管理画面は高権限操作を扱うため、入力値・権限・SQL 境界を backend で固定し、UI や client を信頼しない。

#### Scenario: 範囲外の limit は拒否される (ADMIN-CONSOLE-BE-S024)

- **GIVEN** 検索 API に `limit=0` が渡される
- **WHEN** use case がパラメータを検証する
- **THEN** repository query は実行されず、400 エラーが返される
