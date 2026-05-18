## ADDED Requirements

### Requirement: オペレーターは passkey でログインする

Admin Console は `/login` route で passkey 専用ログイン画面を SHALL 提供する。オペレーターは email を入力して WebAuthn passkey 認証を開始し、認証成功時に httpOnly cookie でセッションを確立しなければならない（MUST）。認証失敗時は non-revealing なエラーメッセージを表示し、アカウント有無・登録状態を推測できない体験を SHALL 保つ。`/login` route は password input、password reset copy、invite registration control を表示してはならない（MUST NOT）。

**Customer Context**

Admin Console の認証はプロダクションの認証基盤と同じ WebAuthn passkey 方式を使用する。オペレーターは顧客向けアカウントとは独立した Admin 専用の passkey credential を持つ。httpOnly cookie を用いることで、SvelteKit server-side の認証チェックが browser API 呼出ごとに自動で行われる。

#### Scenario: オペレーターが passkey でログインする (ADMIN-AUTH-FE-S001)

- **GIVEN** オペレーターが有効な passkey credential を device に登録済みである
- **WHEN** `/login` で登録済みの email を入力し passkey 認証を完了する
- **THEN** httpOnly cookie に session JWT が設定され、Dashboard にリダイレクトされる

#### Scenario: 未登録 email でログインしようとすると non-revealing なエラーになる (ADMIN-AUTH-FE-S002)

- **GIVEN** 入力された email が `admin.operators` に存在しない
- **WHEN** email を入力して passkey 認証を試みる
- **THEN** non-revealing なエラーメッセージが表示され、登録済みオペレーターとの区別がつかない
- **AND** cookie は設定されない

#### Scenario: WebAuthn がキャンセルされた場合はログイン画面に留まる (ADMIN-AUTH-FE-S003)

- **GIVEN** オペレーターが passkey 認証を開始した
- **WHEN** ブラウザの WebAuthn ダイアログでキャンセルする
- **THEN** ログイン画面に留まり、エラーメッセージが表示される
- **AND** cookie は設定されない

#### Scenario: 異なる device の passkey ではログインできない (ADMIN-AUTH-FE-S004)

- **GIVEN** オペレーターの device に登録済み passkey が存在しない
- **WHEN** WebAuthn が利用可能な credential を見つけられない
- **THEN** ログイン画面に留まり、non-revealing なエラーが表示される

#### Scenario: ログイン中は loading 状態が表示される (ADMIN-AUTH-FE-S005)

- **GIVEN** オペレーターが email を入力して passkey 認証を開始した
- **WHEN** WebAuthn ceremony が実行中である
- **THEN** UI に loading indicator が表示され、二重送信が防止される

---

### Requirement: 未認証アクセスはログイン画面へリダイレクトする

`hooks.server.ts` は全リクエストで `admin_session` cookie を検証しなければならない（SHALL）。有効な cookie が存在しない場合、認証必須 route（Dashboard、Accounts、Audit、Settings、Passkey 管理、Logout）は `/login` へリダイレクトしなければならない（SHALL）。pre-auth route である `/login`、`/setup`、`/operator-setup`、および `/api/admin/auth/passkey/*`、`/api/admin/auth/setup/*`、`/api/admin/auth/operator-setup/*` は未認証でも login redirect せず処理されなければならない（MUST）。ただし `/api/admin/auth/passkeys*` の passkey 管理 API は pre-auth ではなく、有効な `admin_session` を route-level で必須とし、未認証時は 401 を返さなければならない（MUST）。`/login` route は認証済みの場合、Dashboard へリダイレクトしなければならない（SHALL）。`/setup` は zero-operator bootstrap gate を満たす場合だけ未認証アクセスを許可し、`/operator-setup` は setup token flow のため未認証アクセスを許可しなければならない（MUST）。

**Customer Context**

Admin Console の業務画面は認証必須である。一方で初回 admin 作成、追加オペレーター登録、passkey login API は未認証で到達できなければ運用不能になる。`hooks.server.ts` は pre-auth route と protected route を明示的に分け、protected route だけを `/login` へリダイレクトする。

#### Scenario: 未認証で Dashboard にアクセスするとログイン画面に飛ぶ (ADMIN-AUTH-FE-S006)

- **GIVEN** `admin_session` cookie が存在しない
- **WHEN** `/` にアクセスする
- **THEN** `/login` にリダイレクトされる（HTTP 302）

#### Scenario: 認証済みで `/login` にアクセスすると Dashboard に飛ぶ (ADMIN-AUTH-FE-S007)

- **GIVEN** 有効な `admin_session` cookie が存在する
- **WHEN** `/login` にアクセスする
- **THEN** `/` (Dashboard) にリダイレクトされる

#### Scenario: 保護 route の直接 URL アクセスでログイン後に元の画面に戻る (ADMIN-AUTH-FE-S008)

- **GIVEN** Operator が未認証状態で `/accounts` に直接アクセスする
- **WHEN** ログイン画面にリダイレクトされ、ログインを完了する
- **THEN** `/accounts` にリダイレクトされる（redirectTo パラメータが保持される）

#### Scenario: pre-auth route は未認証でもログインリダイレクトされない (ADMIN-AUTH-FE-S026)

- **GIVEN** `admin_session` cookie が存在しない
- **WHEN** `/setup`、`/operator-setup`、`/api/admin/auth/passkey/start`、`/api/admin/auth/setup/start`、または `/api/admin/auth/operator-setup/start` にアクセスする
- **THEN** `hooks.server.ts` は `/login` へリダイレクトせず、各 route の bootstrap gate / setup token / auth API 処理へ制御を渡す
- **AND** `/api/admin/auth/passkeys` のような passkey 管理 API は pre-auth 例外に含まれず、未認証時は 401 になる

---

### Requirement: session 期限切れ時はログイン画面へ戻る

session JWT の有効期限が切れている場合、`hooks.server.ts` は cookie をクリアし `/login` へリダイレクトしなければならない（SHALL）。期限切れと未認証の区別は UI に露出せず、どちらもログイン画面へ遷移しなければならない（SHALL）。

**Customer Context**

オペレーターの session は有限の有効期限を持つ。期限切れ後は自動的にログイン画面へ戻り、再認証を促す。

#### Scenario: 期限切れ session でアクセスするとログイン画面に飛ぶ (ADMIN-AUTH-FE-S009)

- **GIVEN** `admin_session` cookie の JWT exp が過去である
- **WHEN** 保護された route にアクセスする
- **THEN** `Set-Cookie` で cookie がクリアされ、`/login` にリダイレクトされる

#### Scenario: 改ざんされた JWT でアクセスするとログイン画面に飛ぶ (ADMIN-AUTH-FE-S010)

- **GIVEN** `admin_session` cookie の JWT 署名が無効である
- **WHEN** 保護された route にアクセスする
- **THEN** cookie がクリアされ、`/login` にリダイレクトされる

---

### Requirement: Admin routes は no-store で配信する

`/login`、`/setup`、`/operator-setup`、認証済み Admin 画面（Dashboard、Accounts、Audit、Settings、Operator 管理、Passkey 管理）および全 Admin BFF route（`/api/admin/*`）の response は `Cache-Control: no-store` を SHALL 保つ。Admin route / BFF response は operator session、顧客 PII、監査情報、bootstrap state、setup token state を含む可能性があるため、CDN / browser / shared proxy で cache してはならない（MUST NOT）。静的 hashed asset はこの no-store 要件の対象外でよいが、HTML / JSON / action / API response は対象外にしてはならない（MUST NOT）。

**Customer Context**

Admin route が cache されると、古い認証状態、顧客 PII、監査ログ、setup token state が再表示され、セキュリティ上の問題となる。

#### Scenario: `/login` は no-store で配信される (ADMIN-AUTH-FE-S011)

- **GIVEN** ブラウザが `/login` を開く
- **WHEN** server が response を返す
- **THEN** response header に `Cache-Control: no-store` が含まれる

#### Scenario: setup 系画面は no-store で配信される (ADMIN-AUTH-FE-S022)

- **GIVEN** ブラウザが `/setup` または `/operator-setup` を開く
- **WHEN** server が response を返す
- **THEN** response header に `Cache-Control: no-store` が含まれる

#### Scenario: 認証済み Admin 画面は no-store で配信される (ADMIN-AUTH-FE-S024)

- **GIVEN** 認証済みオペレーターが `/accounts`、`/accounts/{id}`、`/audit`、`/settings`、`/settings/operators`、`/passkeys` のいずれかを開く
- **WHEN** server が HTML または load response を返す
- **THEN** response header に `Cache-Control: no-store` が含まれる

#### Scenario: Admin BFF response は no-store で配信される (ADMIN-AUTH-FE-S025)

- **GIVEN** browser が `/api/admin/*` の BFF route を呼び出す
- **WHEN** server が JSON、redirect、または error response を返す
- **THEN** response header に `Cache-Control: no-store` が含まれる

---

### Requirement: 認証済みオペレーターは自身の passkey を管理できる

認証済みオペレーターは画面上で自身の登録済み passkey credential 一覧を SHALL 確認できる。新しい passkey を追加する WebAuthn 登録フローを SHALL 提供し、特定 passkey の削除アクションを SHALL 提供する。credential handle / public key は認証 material として画面に露出せず、削除対象の識別には公開可能な passkey identifier と登録 metadata を使わなければならない（MUST）。最後の 1 件の削除操作は無効化しなければならない（MUST）。

**Customer Context**

オペレーターは複数の device で Admin Console にアクセスする。passkey を追加・削除できることで、device 追加や紛失時に安全な鍵管理が可能になる。

#### Scenario: 登録済み passkey 一覧を表示する (ADMIN-AUTH-FE-S012)

- **GIVEN** オペレーターが認証済みで 2 件の passkey を登録済みである
- **WHEN** passkey 管理画面を表示する
- **THEN** 2 件の passkey が公開可能な passkey identifier / バックアップ状態 / 登録日時とともに一覧表示される

#### Scenario: 新しい passkey を追加できる (ADMIN-AUTH-FE-S013)

- **GIVEN** オペレーターが認証済みである
- **WHEN** 「passkey を追加」をクリックし WebAuthn 登録を完了する
- **THEN** 新しい passkey が一覧に追加され、既存 passkey は変化しない

#### Scenario: 最後の 1 件の passkey は削除ボタンが無効化される (ADMIN-AUTH-FE-S014)

- **GIVEN** オペレーターが passkey を 1 件のみ持つ
- **WHEN** passkey 管理画面を表示する
- **THEN** 削除ボタンが無効化または非表示になっている

#### Scenario: 2 件以上の場合は passkey を削除できる (ADMIN-AUTH-FE-S015)

- **GIVEN** オペレーターが passkey を 2 件以上持つ
- **WHEN** 特定 passkey の削除をクリックし確認する
- **THEN** その passkey が一覧から削除され、残りは保持される

#### Scenario: WebAuthn 登録がキャンセルされた場合は一覧が変化しない (ADMIN-AUTH-FE-S016)

- **GIVEN** オペレーターが passkey 追加フローを開始した
- **WHEN** WebAuthn ダイアログでキャンセルする
- **THEN** 一覧は変化せず、エラーメッセージが表示される

---

### Requirement: 初回起動時は最初の admin オペレーターを作成する

`/setup` route は `admin.operators` が 0 件で、かつ bootstrap gate（`ADMIN_BOOTSTRAP_ENABLED=true`、有効な bootstrap secret、有効期限内）を満たす場合のみ、最初の admin オペレーター作成と WebAuthn passkey 登録を SHALL 提供する。初回セットアップ画面は email / display_name / bootstrap secret 入力と passkey 登録を同一 flow で完了し、成功後は JWT cookie を設定して Dashboard へリダイレクトしなければならない（MUST）。`admin.operators` が 1 件以上存在する場合、`/setup` は `/login` へリダイレクトするか 403 を返し、新規オペレーター作成 UI を表示してはならない（MUST NOT）。bootstrap gate を満たさない場合も初回セットアップフォームを表示してはならない（MUST NOT）。

**Customer Context**

DB seed や直接 SQL では初期オペレーターを作成しない。オペレーターが存在しない初回起動時だけ、明示 enable flag と短期 bootstrap secret で保護された bootstrap 画面で最初の admin オペレーターを作成する。

#### Scenario: オペレーター 0 件時に最初の admin オペレーターを作成できる (ADMIN-AUTH-FE-S017)

- **GIVEN** `admin.operators` が 0 件であり、bootstrap gate が有効である
- **WHEN** `/setup` で email / display_name / bootstrap secret を入力し WebAuthn 登録を完了する
- **THEN** role=`admin` のオペレーターが作成され、JWT cookie が設定され Dashboard にリダイレクトされる

#### Scenario: オペレーターが存在する場合は `/setup` を利用できない (ADMIN-AUTH-FE-S018)

- **GIVEN** `admin.operators` が 1 件以上存在する
- **WHEN** `/setup` にアクセスする
- **THEN** `/login` にリダイレクトされるか 403 が表示され、初回セットアップフォームは表示されない

#### Scenario: bootstrap gate が無効な場合は `/setup` フォームを表示しない (ADMIN-AUTH-FE-S023)

- **GIVEN** `admin.operators` が 0 件だが、bootstrap enable flag が無効または bootstrap secret が期限切れである
- **WHEN** `/setup` にアクセスする
- **THEN** 403 が表示され、email / display_name / bootstrap secret 入力フォームは表示されない

---

### Requirement: 追加オペレーターは setup token で初回 passkey を登録する

`/operator-setup` route は admin が `/settings/operators` で発行した one-time setup token を受け取り、追加オペレーターの初回 WebAuthn passkey 登録を SHALL 提供する。setup token は一度だけ使用でき、登録完了後は JWT cookie が設定され Dashboard へリダイレクトされなければならない（MUST）。既に passkey 登録済みのオペレーターは setup token 登録を利用できず、Dashboard へリダイレクトされなければならない（MUST）。

**Customer Context**

admin が追加したオペレーターはまだ passkey credential を持たない。メール配送は対象外のため、admin が安全な別経路で渡した one-time setup token を使って初回登録する。

#### Scenario: setup token で追加オペレーターの初回 passkey を登録できる (ADMIN-AUTH-FE-S019)

- **GIVEN** 追加オペレーターが one-time setup token を持つ
- **WHEN** `/operator-setup` で正しい token を入力し WebAuthn 登録を完了する
- **THEN** JWT cookie が設定され Dashboard にリダイレクトされる

#### Scenario: 不正な setup token では登録できない (ADMIN-AUTH-FE-S020)

- **GIVEN** setup token が不正である
- **WHEN** `/operator-setup` で token を入力する
- **THEN** non-revealing なエラーが表示され、登録に進めない

#### Scenario: 既に passkey 登録済みのオペレーターは setup token 登録を利用できない (ADMIN-AUTH-FE-S021)

- **GIVEN** オペレーターが既に passkey 登録済みである
- **WHEN** `/operator-setup` にアクセスする
- **THEN** Dashboard にリダイレクトされる
