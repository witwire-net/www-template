## MODIFIED Requirements

### Requirement: オペレーターは passkey でログインする

Admin Console は `/login` route で passkey 専用ログイン画面を SHALL 提供する。Login UI は browser WebAuthn API と `packages/admin/domain` の auth flow を使用し、`packages/admin/api` 経由で same-origin の `/api/v1/auth/passkey/*` API を呼び出さなければならない（SHALL）。Login 成功時、Admin frontend は response body の operator accessToken と session metadata を memory state に保持し、operator refreshToken は Admin backend が `HttpOnly; Secure; SameSite=Lax` Cookie として管理しなければならない（SHALL）。Admin auth UI は SvelteKit server hooks、server load/actions、package-local BFF route を認証判断に使用してはならない（MUST NOT）。Admin auth UI は Product auth API または Product SDK を使用して operator session を作成してはならない（MUST NOT）。認証失敗 UI は operator 存在、passkey 登録状態、setup token 状態を推測できない non-revealing な文言を保たなければならない（MUST）。

**Customer Context**

Admin 認証は passkey、operator session、setup token、CSRF を扱うため、画面配信 package に server-side 認証処理が存在すると責務境界が崩れる。運営者は静的 Admin UI から安全に認証し、認証判断は Admin backend に集約される必要がある。

#### Scenario: Login UI は Admin backend auth API を呼び出す (ADMIN-AUTH-FE-S027)

- **GIVEN** Operator が `/login` を開いている
- **WHEN** email を入力して passkey login を開始する
- **THEN** UI は Admin api layer 経由で Admin backend の passkey start API を呼び出す
- **AND** package-local BFF route は呼び出されない

#### Scenario: Product auth SDK は operator session 作成に使われない (ADMIN-AUTH-FE-S028)

- **WHEN** Admin auth domain code が Product auth SDK を import して operator login を実装している
- **THEN** lint は dependency boundary violation として失敗する

#### Scenario: Operator login は accessToken だけを browser-readable state に保持する (ADMIN-AUTH-FE-S033)

- **GIVEN** Operator が passkey login を完了する
- **WHEN** Admin auth domain state を確認する
- **THEN** operator accessToken と session metadata は memory state に存在する
- **AND** operator refreshToken 平文は memory state、localStorage、sessionStorage、IndexedDB、URL に存在しない

### Requirement: 未認証アクセスはログイン画面へリダイレクトする

静的 Admin frontend は保護画面を表示する前に Admin backend の current operator / session verification API を SHALL 呼び出す。Current operator request は memory state の operator accessToken を `Authorization: Bearer` header として送信し、必要に応じて same-origin refresh request を credentials 付きで実行しなければならない（SHALL）。Session が無効、期限切れ、または operator inactive の場合、Admin frontend は protected content を表示せず login へ誘導しなければならない（MUST）。Admin frontend は operator role / permission を UI 表示制御に使用できるが、Backend authorization の代替として扱ってはならない（MUST NOT）。Logout UI は Admin backend logout API を呼び出し、client accessToken state と CSRF token state を破棄し、Admin backend に operator refreshToken Cookie の revoke を委ねなければならない（SHALL）。

**Customer Context**

Admin frontend は静的に配信されるため、画面遷移時の認証済み presentation は browser 側で最新 session 状態を確認する必要がある。一方で最終的な認可は必ず Admin backend API が行い、UI は利便性と誤操作防止を担当する。

#### Scenario: 未認証で Accounts 画面に到達しても protected content を表示しない (ADMIN-AUTH-FE-S030)

- **GIVEN** operator session が存在しない
- **WHEN** Operator が `/accounts` を直接開く
- **THEN** UI は account data を表示せず login へ誘導する

#### Scenario: role は UI 制御に使われるが Backend 認可が必須である (ADMIN-AUTH-FE-S031)

- **GIVEN** Operator role が viewer である
- **WHEN** UI が account 作成 action を非表示にする
- **THEN** Backend API は同じ request に対しても Admin RBAC を検証し、UI 表示だけに依存しない

#### Scenario: protected route は operator accessToken を検証に使う (ADMIN-AUTH-FE-S034)

- **GIVEN** Admin frontend が operator accessToken と session metadata を memory state に保持している
- **WHEN** Operator が `/accounts` を開く
- **THEN** UI は `Authorization: Bearer` header で current operator API を呼び出してから protected content を表示する

#### Scenario: Admin refresh は HttpOnly Cookie に委ねる (ADMIN-AUTH-FE-S035)

- **GIVEN** operator accessToken が期限切れ間近で、operator refreshToken Cookie は browser に保持されている
- **WHEN** Admin frontend が protected Admin API を呼び出そうとする
- **THEN** UI は credentials を含めて same-origin refresh API を呼び出す
- **AND** refresh 成功後の response body から新しい operator accessToken を memory state に反映する

### Requirement: Admin routes は no-store で配信する

Admin static frontend の HTML / route shell / runtime config response は no-store semantics で配信されなければならない（SHALL）。Admin backend の `/api/v1/*` response は no-store header を含まなければならない（SHALL）。Hashed static assets は長期 cache できるが、operator session、顧客 PII、監査情報、bootstrap state、setup token state を含む response は cache してはならない（MUST NOT）。

**Customer Context**

Admin route が cache されると、古い認証状態、顧客 PII、監査ログ、setup token state が再表示され、セキュリティ上の問題となる。静的配信でも HTML と API response は no-store 境界を維持する必要がある。

#### Scenario: Admin HTML は no-store で配信される (ADMIN-AUTH-FE-S032)

- **GIVEN** browser が Admin frontend の HTML route を開く
- **WHEN** response header を確認する
- **THEN** HTML response は no-store semantics を持つ

### Requirement: 追加オペレーターは setup token で初回 passkey を登録する

`/operator-setup` route は browser WebAuthn API と Admin backend の same-origin `/api/v1/auth/operator-setup/*` API を使って、追加オペレーターの初回 passkey 登録を SHALL 提供する。setup token の検証、challenge 保存、passkey 登録、session 発行は Admin backend が実行し、Admin frontend は平文 token を永続 storage、telemetry、log に保存してはならない（MUST NOT）。setup token が無効、期限切れ、消費済み、または登録済み operator に属する場合、UI は token 状態を区別しない generic error を表示しなければならない（SHALL）。

**Customer Context**

admin が追加したオペレーターはまだ passkey credential を持たない。admin が安全な別経路で渡した one-time setup token を使って初回登録する。静的 Admin frontend は token secret を長く保持せず、検証は Admin backend に委譲する。

#### Scenario: setup token エラーは non-revealing に表示される (ADMIN-AUTH-FE-S029)

- **GIVEN** Operator setup token が無効、期限切れ、または消費済みである
- **WHEN** `/operator-setup` で token を送信する
- **THEN** UI は token 状態を区別しない generic error を表示する
