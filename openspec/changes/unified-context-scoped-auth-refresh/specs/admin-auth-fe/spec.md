## MODIFIED Requirements

### Requirement: オペレーターは passkey でログインする

Admin Console は `/login` route で passkey 専用ログイン画面を SHALL 提供する。Login UI は browser WebAuthn API と `packages/admin/domain` の auth flow を使用し、`packages/admin/api` 経由で same-origin の `/api/v1/auth/passkey/*` API を呼び出さなければならない（SHALL）。Login 成功時、Admin frontend は response body の operator accessToken、authContextId、operator/session metadata を memory-only state に保持し、operator refreshToken は Admin backend が `HttpOnly; Secure; SameSite=Lax` の path-scoped Cookie として管理しなければならない（SHALL）。Admin auth UI は Product auth API、Product SDK、Product domain、package-local BFF route、または `/api/admin/*` を operator session 作成に使用してはならない（MUST NOT）。

**Customer Context**

Admin 認証は passkey、operator session、setup token を扱う強権限境界である。Admin Console も Product Web と同じく短命 accessToken を memory-only にし、refreshToken を HttpOnly Cookie に閉じることで XSS と debug surface から refresh credential を守る。

#### Scenario: Login UI は Admin backend auth API を呼び出す (ADMIN-AUTH-FE-S027)

- **GIVEN** Operator が `/login` を開いている
- **WHEN** email を入力して passkey login を開始する
- **THEN** UI は Admin api layer 経由で Admin backend の passkey start API を呼び出す
- **AND** package-local BFF route、Product SDK、`/api/admin/*` は呼び出されない

#### Scenario: Product auth SDK は operator session 作成に使われない (ADMIN-AUTH-FE-S028)

- **WHEN** Admin auth domain code が Product auth SDK を import して operator login を実装している
- **THEN** lint または import-boundary test は dependency boundary violation として失敗する

#### Scenario: Operator login は accessToken だけを browser-readable session state に保存する (ADMIN-AUTH-FE-S033)

- **GIVEN** Operator が passkey login を完了する
- **WHEN** Admin auth domain state を確認する
- **THEN** operator accessToken、authContextId、operator/session metadata は memory state に存在する
- **AND** operator refreshToken、Product account token、Cookie value は memory state、localStorage、sessionStorage、IndexedDB、URL、telemetry に存在しない

### Requirement: 未認証アクセスはログイン画面へリダイレクトする

静的 Admin frontend は保護画面を表示する前に Admin backend の current operator / session verification API を SHALL 呼び出す。Current operator request は memory state の operator accessToken を `Authorization: Bearer` header として送信し、必要に応じて active authContextId の same-origin context refresh request を credentials 付きで 1 回だけ実行しなければならない（SHALL）。Admin frontend は protected route verification や mutation request に `X-Auth-Context-Id` header または CSRF header を session selector として生成してはならない（MUST NOT）。Logout UI は Admin backend logout API を active operator accessToken で呼び出し、client accessToken state を破棄し、Admin backend に operator refreshToken Cookie の revoke と path clear を委ねなければならない（SHALL）。

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
- **AND** Product Cookie session、Product auth SDK、`X-Auth-Context-Id` header、CSRF header では protected content を表示しない

#### Scenario: Admin refresh は active authContextId の HttpOnly Cookie に委ねる (ADMIN-AUTH-FE-S035)

- **GIVEN** operator accessToken が期限切れで、operator refreshToken Cookie は browser に保持されている
- **WHEN** Admin frontend が protected Admin API を呼び出そうとする
- **THEN** UI は active authContextId の `POST /api/v1/auth/contexts/{authContextId}/refresh` を credentials 付きで呼び出す
- **AND** refresh 成功後の response body から新しい operator accessToken を memory state に反映する

### Requirement: session 期限切れ時はログイン画面へ戻る

静的 Admin frontend は operator accessToken の期限切れ、Admin refresh 失敗、current operator API の未認証 response、operator inactive response を検知した場合、protected content を表示せず login へ誘導しなければならない（MUST）。Admin frontend は `hooks.server.ts`、server load/actions、package-local BFF、browser-readable refreshToken を session 期限判定に使用してはならない（MUST NOT）。Session expiry と未認証の詳細は UI に露出せず、必要な state cleanup と generic login guidance だけを表示しなければならない（SHALL）。browser reload 後の context discovery は Admin origin の `localStorage` に保存した non-secret context index だけを使い、index entry は refresh 成功まで authenticated operator session として扱ってはならない（MUST NOT）。

**Customer Context**

Admin frontend は静的に配信されるため、server hook redirect に依存できない。期限切れ session では顧客情報や監査情報を表示せず、Admin backend current/refresh API の結果で安全に login へ戻す必要がある。

#### Scenario: Admin refresh 失敗時は protected content を表示しない (ADMIN-AUTH-FE-S036)

- **GIVEN** operator accessToken が期限切れで、Admin refresh API が 401 を返す
- **WHEN** Operator が `/accounts` を開く
- **THEN** Admin frontend は protected content を表示せず、memory state を破棄して login へ誘導する

#### Scenario: session expiry reason は UI に露出しない (ADMIN-AUTH-FE-S037)

- **GIVEN** current operator API が expired、revoked、inactive のいずれかを返す
- **WHEN** Admin frontend が session state を更新する
- **THEN** UI は詳細理由を区別せず generic login guidance を表示する

#### Scenario: Admin context index は token/secret を含まない (ADMIN-AUTH-FE-S041)

- **GIVEN** browser reload により Admin memory session が消えている
- **WHEN** Admin Console が browser-readable context index を読み取る
- **THEN** index は authContextId と非 secret operator/session metadata だけを含み、operator accessToken、refreshToken、Cookie value を含まない
- **AND** tamper された entry は context refresh failure として fail-close される

#### Scenario: Admin context index は origin-local localStorage に限定される (ADMIN-AUTH-FE-S042)

- **GIVEN** Admin Console が login、setup、operator-setup、context refresh、logout、または operator deactivation result を処理している
- **WHEN** context index を更新する
- **THEN** Admin origin の `localStorage` の Admin-specific key だけを更新する
- **AND** entry には version、surface、authContextId、operatorSessionId、operator display hint、role hint、lastSeenAt、expiresHintAt だけを保存し、operator accessToken、refreshToken、Cookie value、setup token を保存しない
- **AND** 同一 Admin origin の他 tab には `storage` event または `BroadcastChannel` で add/remove/active change を伝搬する

#### Scenario: Admin context index cleanup は logout と inactive response に追従する (ADMIN-AUTH-FE-S043)

- **GIVEN** Admin Console が複数 operator/session context の index entries を持っている
- **WHEN** logout、operator deactivation、session revoke、または context refresh failure が特定 authContextId に対して発生する
- **THEN** クライアントは対象 authContextId の memory session item と context index entry を削除する
- **AND** all-context revoke response の場合は server が返した対象 entries をすべて削除する
- **AND** 複数 tab で cleanup 競合が発生した場合、Admin backend の current/context refresh result を正として stale entry を再採用しない

### Requirement: 初回起動時は最初の admin オペレーターを作成する

静的 Admin frontend の `/setup` route は、Admin backend の same-origin `/api/v1/auth/setup/*` API を通じて初回 admin operator 作成と passkey 登録を行わなければならない（SHALL）。`/setup` は `hooks.server.ts`、server load/actions、package-local BFF、server-side cookie redirect を使用してはならない（MUST NOT）。Setup 成功時、Admin frontend は response body の operator accessToken、authContextId、operator/session metadata を memory state に保持し、Admin refreshToken は backend が `HttpOnly; Secure; SameSite=Lax` path-scoped Cookie として設定しなければならない（MUST）。

**Customer Context**

初回 admin 作成は Admin surface の trust anchor である。静的 frontend は secret validation と operator 作成を Admin backend に委譲し、refreshToken や server hook redirect 前提を持たないことで、Admin auth の責務を backend に集約する。

#### Scenario: 静的 setup UI は Admin backend で最初の admin を作成する (ADMIN-AUTH-FE-S038)

- **GIVEN** operator が 0 件で bootstrap gate が有効である
- **WHEN** `/setup` で email、display name、bootstrap secret、WebAuthn registration を完了する
- **THEN** UI は `/api/v1/auth/setup/*` を呼び、operator accessToken、authContextId、operator/session metadata を memory state に保持する
- **AND** refreshToken 平文は browser-readable state に存在しない

#### Scenario: operator が存在する場合は setup form を表示しない (ADMIN-AUTH-FE-S039)

- **GIVEN** Admin backend current setup state が operator existing を返す
- **WHEN** Operator が `/setup` を開く
- **THEN** Admin frontend は setup form を表示せず login へ誘導する

#### Scenario: bootstrap gate 無効時は setup secret 入力欄を表示しない (ADMIN-AUTH-FE-S040)

- **GIVEN** bootstrap gate が無効または期限切れである
- **WHEN** Operator が `/setup` を開く
- **THEN** Admin frontend は bootstrap secret 入力 form を表示せず generic unavailable state を表示する

### Requirement: 追加オペレーターは setup token で初回 passkey を登録する

`/operator-setup` route は browser WebAuthn API と Admin backend の same-origin `/api/v1/auth/operator-setup/*` API を使って、追加オペレーターの初回 passkey 登録を SHALL 提供する。setup token の検証、challenge 保存、passkey 登録、session 発行は Admin backend が実行し、Admin frontend は平文 token、refreshToken、Cookie value を永続 storage、telemetry、log に保存してはならない（MUST NOT）。setup token が無効、期限切れ、消費済み、または登録済み operator に属する場合、UI は token 状態を区別しない汎用 error を表示しなければならない（SHALL）。

**Customer Context**

admin が追加したオペレーターはまだ passkey credential を持たない。one-time setup token を使って初回登録し、登録成功後は通常 login と同じ memory-only accessToken + path-scoped refresh Cookie state へ入る必要がある。

#### Scenario: setup token エラーは秘匿的に表示される (ADMIN-AUTH-FE-S029)

- **GIVEN** Operator setup token が無効、期限切れ、または消費済みである
- **WHEN** `/operator-setup` で token を送信する
- **THEN** UI は token 状態を区別しない汎用 error を表示する
