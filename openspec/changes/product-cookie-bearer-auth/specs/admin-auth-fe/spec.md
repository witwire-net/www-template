## MODIFIED Requirements

### Requirement: オペレーターは passkey でログインする

Admin Console は `/login` route で passkey 専用ログイン画面を SHALL 提供する。Login UI は browser WebAuthn API と `packages/admin/domain` の auth flow を使用し、`packages/admin/api` 経由で same-origin の `/api/v1/auth/passkey/*` API を呼び出さなければならない（SHALL）。Login 成功時、Admin frontend は response body の operator/session metadata と CSRF token だけを memory state に保持し、operator access credential と operator refresh credential は Admin backend が `HttpOnly; Secure; SameSite=Lax` Cookie として管理しなければならない（SHALL）。Admin auth UI は browser-readable operator accessToken、Product auth API、Product SDK、Product domain、package-local BFF route、または `/api/admin/*` を operator session 作成に使用してはならない（MUST NOT）。認証失敗 UI は operator 存在、passkey 登録状態、setup token 状態を推測できない秘匿的な文言を保たなければならない（MUST）。

**Customer Context**

Admin 認証は passkey、operator session、setup token、CSRF を扱うため、browser-readable accessToken を保持すると Product Web と同じ XSS / debug surface の credential 漏えいリスクを持つ。運営者は静的 Admin UI から安全に認証し、認証判断は Admin backend に集約される必要がある。Product と Admin は domain / SDK / origin を分離しながら、browser surface ではどちらも HttpOnly Cookie と CSRF token だけで authenticated state を表現しなければならない。

#### Scenario: Login UI は Admin backend auth API を呼び出す (ADMIN-AUTH-FE-S027)

- **GIVEN** Operator が `/login` を開いている
- **WHEN** email を入力して passkey login を開始する
- **THEN** UI は Admin api layer 経由で Admin backend の passkey start API を呼び出す
- **AND** package-local BFF route、Product SDK、`/api/admin/*` は呼び出されない

#### Scenario: Product auth SDK は operator session 作成に使われない (ADMIN-AUTH-FE-S028)

- **WHEN** Admin auth domain code が Product auth SDK を import して operator login を実装している
- **THEN** lint または import-boundary test は dependency boundary violation として失敗する

#### Scenario: Operator login は browser-readable token を保存しない (ADMIN-AUTH-FE-S033)

- **GIVEN** Operator が passkey login を完了する
- **WHEN** Admin auth domain state を確認する
- **THEN** operator/session metadata と CSRF token は memory state に存在する
- **AND** operator accessToken、operator refreshToken、Product account accessToken、Cookie value は memory state、localStorage、sessionStorage、IndexedDB、URL に存在しない

### Requirement: 未認証アクセスはログイン画面へリダイレクトする

静的 Admin frontend は保護画面を表示する前に Admin backend の current operator / session verification API を SHALL 呼び出す。Current operator request は same-origin credentials によって Admin access Cookie を送信し、current operator API が 401 または session-expired 相当の未認証応答を返した場合は、同一 navigation attempt 内で 1 回だけ same-origin refresh request を credentials 付きで実行しなければならない（SHALL）。Admin frontend は protected route verification や mutation request に `Authorization: Bearer` header を生成してはならない（MUST NOT）。Session が無効、期限切れ、または operator inactive の場合、Admin frontend は protected content を表示せず login へ誘導しなければならない（MUST）。Admin frontend は operator role / permission を UI 表示制御に使用できるが、Backend authorization の代替として扱ってはならない（MUST NOT）。Logout UI は Admin backend logout API を same-origin Cookie と CSRF token 付きで呼び出し、client session metadata と CSRF token state を破棄し、Admin backend に operator access Cookie と operator refresh Cookie の revoke を委ねなければならない（SHALL）。Product Web account Cookie session は Admin protected route verification に使ってはならない（MUST NOT）。

**Customer Context**

Admin frontend は静的に配信されるため、画面遷移時の認証済み presentation は browser 側で最新 session 状態を確認する必要がある。一方で最終的な認可は必ず Admin backend API が行い、UI は利便性と誤操作防止を担当する。Admin current / refresh / logout flow は operator session 専用であり、Product account Cookie session と混同すると顧客情報や運用権限の露出につながる。

#### Scenario: 未認証で Accounts 画面に到達しても protected content を表示しない (ADMIN-AUTH-FE-S030)

- **GIVEN** operator Cookie session が存在しない
- **WHEN** Operator が `/accounts` を直接開く
- **THEN** UI は account data を表示せず login へ誘導する

#### Scenario: role は UI 制御に使われるが Backend 認可が必須である (ADMIN-AUTH-FE-S031)

- **GIVEN** Operator role が viewer である
- **WHEN** UI が account 作成 action を非表示にする
- **THEN** Backend API は同じ request に対しても Admin RBAC を検証し、UI 表示だけに依存しない

#### Scenario: protected route は Admin Cookie session を検証に使う (ADMIN-AUTH-FE-S034)

- **GIVEN** Admin frontend が operator/session metadata と CSRF token を memory state に保持している
- **WHEN** Operator が `/accounts` を開く
- **THEN** UI は same-origin credentials で current operator API を呼び出してから protected content を表示する
- **AND** `Authorization: Bearer` header、Product Cookie session、Product auth SDK では protected content を表示しない

#### Scenario: Admin refresh は HttpOnly Cookie に委ねる (ADMIN-AUTH-FE-S035)

- **GIVEN** Admin access Cookie が期限切れ間近または期限切れで、operator refresh Cookie は browser に保持されている
- **WHEN** Admin frontend が protected Admin API を呼び出そうとする
- **THEN** UI は credentials を含めて same-origin refresh API を呼び出す
- **AND** refresh 成功後の response body から新しい operator/session metadata と CSRF token を memory state に反映する
- **AND** Product refresh endpoint、Product Cookie-only session DTO、operator accessToken body を使用しない

### Requirement: session 期限切れ時はログイン画面へ戻る

静的 Admin frontend は Admin access Cookie の期限切れ、Admin refresh 失敗、current operator API の未認証 response、operator inactive response を検知した場合、protected content を表示せず login へ誘導しなければならない（MUST）。Admin frontend は `hooks.server.ts`、server load/actions、package-local BFF、browser-readable operator accessToken、browser-readable refreshToken、または `Authorization: Bearer` header を session 期限判定に使用してはならない（MUST NOT）。Session expiry と未認証の詳細は UI に露出せず、必要な state cleanup と generic login guidance だけを表示しなければならない（SHALL）。

**Customer Context**

Admin frontend は静的に配信されるため、server hook redirect に依存できない。期限切れ session では顧客情報や監査情報を表示せず、Admin backend current/refresh API の結果で安全に login へ戻す必要がある。Cookie-only session では frontend が access credential を読めないため、server response と refresh outcome だけで route intent を決める必要がある。

#### Scenario: Admin refresh 失敗時は protected content を表示しない (ADMIN-AUTH-FE-S036)

- **GIVEN** Admin access Cookie が期限切れで、Admin refresh API が 401 を返す
- **WHEN** Operator が `/accounts` を開く
- **THEN** Admin frontend は protected content を表示せず、memory state を破棄して login へ誘導する

#### Scenario: session expiry reason は UI に露出しない (ADMIN-AUTH-FE-S037)

- **GIVEN** current operator API が expired、revoked、inactive のいずれかを返す
- **WHEN** Admin frontend が session state を更新する
- **THEN** UI は詳細理由を区別せず generic login guidance を表示する

### Requirement: 初回起動時は最初の admin オペレーターを作成する

静的 Admin frontend の `/setup` route は、Admin backend の same-origin `/api/v1/auth/setup/*` API を通じて初回 admin operator 作成と passkey 登録を行わなければならない（SHALL）。`/setup` は `hooks.server.ts`、server load/actions、package-local BFF、server-side cookie redirect を使用してはならない（MUST NOT）。Setup 成功時、Admin frontend は response body の operator/session metadata と CSRF token だけを memory state に保持し、operator access credential と operator refresh credential は backend が `HttpOnly; Secure; SameSite=Lax` Cookie として設定しなければならない（MUST）。Operator accessToken、refreshToken 平文、Cookie value は browser-readable state に保持してはならない（MUST NOT）。Operator が既に存在する場合、または bootstrap gate が無効な場合、UI は setup form を表示せず login へ誘導または generic unavailable state を表示しなければならない（SHALL）。

**Customer Context**

初回 admin 作成は Admin surface の trust anchor である。静的 frontend は secret validation と operator 作成を Admin backend に委譲し、JWT memory state や server hook redirect 前提を持たないことで、Admin auth の責務を backend に集約する。setup 成功直後も通常 login と同じ Cookie-only session state にそろえる必要がある。

#### Scenario: 静的 setup UI は Admin backend で最初の admin を作成する (ADMIN-AUTH-FE-S038)

- **GIVEN** operator が 0 件で bootstrap gate が有効である
- **WHEN** `/setup` で email、display name、bootstrap secret、WebAuthn registration を完了する
- **THEN** UI は `/api/v1/auth/setup/*` を呼び、operator/session metadata と CSRF token だけを memory state に保持する
- **AND** operator accessToken、refreshToken 平文、Cookie value は browser-readable state に存在しない

#### Scenario: operator が存在する場合は setup form を表示しない (ADMIN-AUTH-FE-S039)

- **GIVEN** Admin backend current setup state が operator existing を返す
- **WHEN** Operator が `/setup` を開く
- **THEN** Admin frontend は setup form を表示せず login へ誘導する

#### Scenario: bootstrap gate 無効時は setup secret 入力欄を表示しない (ADMIN-AUTH-FE-S040)

- **GIVEN** bootstrap gate が無効または期限切れである
- **WHEN** Operator が `/setup` を開く
- **THEN** Admin frontend は bootstrap secret 入力 form を表示せず generic unavailable state を表示する

### Requirement: 追加オペレーターは setup token で初回 passkey を登録する

`/operator-setup` route は browser WebAuthn API と Admin backend の same-origin `/api/v1/auth/operator-setup/*` API を使って、追加オペレーターの初回 passkey 登録を SHALL 提供する。setup token の検証、challenge 保存、passkey 登録、Cookie-only session 発行は Admin backend が実行し、Admin frontend は平文 token、operator accessToken、refreshToken 平文、Cookie value を永続 storage、memory session state、telemetry、log に保存してはならない（MUST NOT）。setup token が無効、期限切れ、消費済み、または登録済み operator に属する場合、UI は token 状態を区別しない汎用 error を表示しなければならない（SHALL）。

**Customer Context**

admin が追加したオペレーターはまだ passkey credential を持たない。admin が安全な別経路で渡した one-time setup token を使って初回登録する。静的 Admin frontend は token secret を長く保持せず、検証は Admin backend に委譲する。登録成功後は通常 login と同じ Cookie-only state を受け入れ、browser-readable token を作らない。

#### Scenario: setup token エラーは秘匿的に表示される (ADMIN-AUTH-FE-S029)

- **GIVEN** Operator setup token が無効、期限切れ、または消費済みである
- **WHEN** `/operator-setup` で token を送信する
- **THEN** UI は token 状態を区別しない汎用 error を表示する
