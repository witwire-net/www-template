## Purpose

Admin Console の frontend requirements をまとめる。account browsing, detail, suspend/restore, audit viewing, operator settings, layout, dashboard, and MVCS import boundaries を対象とする。

## Requirements

### Requirement: Admin Console は同一 Admin ドメインの `/api/v1/*` を利用する

Admin frontend は Product frontend/app とは別の Admin ドメインで配信されなければならない（SHALL）。Admin frontend と Admin backend は同一 Admin ドメインでホストされ、Cloudflare routing により静的 frontend と GoServer の `/api/v1/*` に振り分けられなければならない（SHALL）。Admin frontend は account management / operator management / audit / auth の呼び出し先として same-origin の `/api/v1/*` だけを使用し、Product ドメインを呼び出してはならない（MUST NOT）。Admin API 呼び出しは credential mode、CSRF token header、request ID header を Admin backend contract に従って付与しなければならない（SHALL）。Admin route HTML は no-store 相当で配信され、hashed static assets だけを長期 cache できる（SHALL）。

**Customer Context**

Admin と Product はドメインで分離される。一方、Admin frontend と Admin backend は同じ Admin ドメインに置き、Cloudflare が frontend と `/api/v1/*` backend を振り分ける。Admin UI が Product ドメインを直接呼び出すと、強権限 API の境界と cookie / CSRF の前提が崩れる。

#### Scenario: Admin API 呼び出しは same-origin `/api/v1/*` を使う (ADMIN-CONSOLE-FE-S041)

- **GIVEN** Admin frontend が Admin ドメインで配信されている
- **WHEN** Admin app が API 呼び出しを開始する
- **THEN** request URL は same-origin の `/api/v1/*` であり、別 origin の base URL は使用されない

#### Scenario: Product ドメインへの Admin 操作呼び出しは拒否される (ADMIN-CONSOLE-FE-S042)

- **WHEN** Admin API wrapper が Product ドメインを呼び出そうとする
- **THEN** runtime validation は request を送信せず、設定不備として扱う

#### Scenario: Admin frontend は Product frontend/app と別ドメインで配信される (ADMIN-CONSOLE-FE-S046)

- **GIVEN** Admin frontend と Product frontend/app の deployment config が存在する
- **WHEN** configured domain を検証する
- **THEN** Admin domain は Product frontend/app domain と一致しない

### Requirement: オペレーターは Admin Console から顧客アカウントを作成できる

Admin Console は Account 作成画面または Accounts 画面内 action を SHALL 提供する。Account 作成 UI は email を必須入力とし、送信前に形式と空文字を検証しなければならない（MUST）。Account 作成成功時、UI は作成された account の email、status、passkey count、作成日時を表示しなければならない（SHALL）。重複 email、権限不足、validation error、backend failure はユーザーが理解できるエラーメッセージとして表示し、入力内容を失ってはならない（MUST NOT）。作成された account は Accounts 一覧と詳細画面から確認できなければならない（SHALL）。

**Customer Context**

運営者はサポートや初期導入の場面で、顧客アカウントを Admin Console から安全に作成する必要がある。作成操作は Account domain の不変条件と監査を Backend 側で適用し、Admin UI は必要な入力と結果確認に集中する。

#### Scenario: オペレーターが顧客アカウントを作成する (ADMIN-CONSOLE-FE-S043)

- **GIVEN** Operator が account 作成権限を持つ
- **WHEN** Account 作成フォームに `customer@example.com` を入力して送信する
- **THEN** 作成成功メッセージが表示される
- **AND** 作成された account の detail へ移動できる

#### Scenario: email 形式が不正な場合は送信しない (ADMIN-CONSOLE-FE-S044)

- **WHEN** Account 作成フォームに email 形式ではない値を入力して送信する
- **THEN** UI は validation error を表示し、Admin backend へ request を送信しない

#### Scenario: 重複 email の作成失敗を表示する (ADMIN-CONSOLE-FE-S045)

- **GIVEN** `customer@example.com` の account が存在する
- **WHEN** 同じ email で Account 作成を送信する
- **THEN** UI は重複 email のエラーを表示し、フォーム入力を保持する

### Requirement: アカウント検索と閲覧

Admin Console はメールアドレス部分一致検索を SHALL 提供する。検索結果はページネーション付きテーブルで、email / status / passkey 数 / 最終 passkey 日時を SHALL 表示する。status フィルター（active / suspended / all）を SHALL 提供する。各アカウント行は詳細画面へリンクしなければならない。

**Customer Context**

サポート問い合わせ時に、オペレーターは顧客のメールアドレスでアカウントを特定し、現在の状態や登録済み passkey の情報を確認する必要がある。

#### Scenario: メールでアカウントを検索する (ADMIN-CONSOLE-FE-S001)

- **GIVEN** `accounts` に `alice@example.com` と `bob@test.com` が存在する
- **WHEN** 検索欄に `example` を入力して検索する
- **THEN** `alice@example.com` のみがテーブルに表示される

#### Scenario: 検索結果が空の場合は空状態を表示する (ADMIN-CONSOLE-FE-S002)

- **WHEN** 検索欄に存在しないメールを入力する
- **THEN** 「該当するアカウントはありません」という空状態メッセージが表示される

#### Scenario: status フィルターで絞り込む (ADMIN-CONSOLE-FE-S003)

- **GIVEN** active アカウント 3 件、suspended アカウント 2 件が存在する
- **WHEN** status フィルターで `suspended` を選択する
- **THEN** suspended アカウント 2 件のみが表示される

#### Scenario: ページネーションが機能する (ADMIN-CONSOLE-FE-S004)

- **GIVEN** 25 件のアカウントが存在し、ページサイズが 20 である
- **WHEN** アカウント一覧を表示する
- **THEN** 1 ページ目に 20 件、2 ページ目に 5 件が表示される
- **AND** ページ移動 UI が表示される

#### Scenario: 検索中は loading 状態が表示される (ADMIN-CONSOLE-FE-S005)

- **WHEN** 検索が実行中である
- **THEN** テーブルに loading indicator が表示される

#### Scenario: 検索エラー時にエラーメッセージが表示される (ADMIN-CONSOLE-FE-S006)

- **GIVEN** DB 接続が失敗している
- **WHEN** 検索を実行する
- **THEN** エラーメッセージが表示され、テーブルは更新されない

---

### Requirement: アカウント詳細を表示する

詳細画面は email, status, status_reason, status_updated_at, status_updated_by を SHALL 表示する。登録 passkey 一覧を SHALL 表示する。存在しないアカウント ID の場合は 404 を SHALL 表示する。

**Customer Context**

サポート対応時に、オペレーターはアカウントの完全な情報と全 passkey credential の詳細を確認する必要がある。

#### Scenario: アカウント詳細を表示する (ADMIN-CONSOLE-FE-S007)

- **GIVEN** アクティブなアカウントが存在する
- **WHEN** 一覧からそのアカウントをクリックする
- **THEN** email, status=`active`, passkey 数と一覧が表示される

#### Scenario: 存在しないアカウント ID は 404 になる (ADMIN-CONSOLE-FE-S008)

- **GIVEN** 無効なアカウント ID が URL に指定されている
- **WHEN** 詳細画面にアクセスする
- **THEN** 404 エラーページが表示される

#### Scenario: passkey が 0 件のアカウント詳細を表示する (ADMIN-CONSOLE-FE-S009)

- **GIVEN** passkey を 1 件も登録していないアカウントが存在する
- **WHEN** 詳細画面を表示する
- **THEN** 「passkey は登録されていません」と表示される

---

### Requirement: アカウント停止

更新操作は理由の入力を MUST 要求する。操作は確認ダイアログの後に実行されなければならない（MUST）。成功時は status が `suspended` に変わり、成功メッセージを SHALL 表示する。失敗時はエラーメッセージを SHALL 表示する。

**Customer Context**

不正利用が疑われるアカウントに対して、オペレーターは一時停止を行う。理由の記録は監査のために必須である。

#### Scenario: アクティブなアカウントを停止する (ADMIN-CONSOLE-FE-S010)

- **GIVEN** Operator が operator ロール以上を持つ
- **AND** アカウント status が `active`
- **WHEN** Suspend ボタンをクリックし、理由を入力し、確認ダイアログで Confirm する
- **THEN** status が `suspended` に変わり、「アカウントを停止しました」と表示される

#### Scenario: 理由が空の場合は停止できない (ADMIN-CONSOLE-FE-S011)

- **WHEN** Suspend ダイアログで理由を空のまま Confirm する
- **THEN** バリデーションエラーが表示され、操作は実行されない

#### Scenario: 確認ダイアログでキャンセルした場合は停止されない (ADMIN-CONSOLE-FE-S012)

- **WHEN** Suspend ダイアログで Cancel をクリックする
- **THEN** ダイアログが閉じ、アカウント status は変化しない

#### Scenario: 既に停止済みのアカウントは停止できない (ADMIN-CONSOLE-FE-S013)

- **GIVEN** アカウント status が `suspended` である
- **WHEN** Suspend ボタンが表示される
- **THEN** Suspend ボタンが無効化または非表示になっている

---

### Requirement: アカウント復旧

Restore 操作は確認ダイアログの後に実行されなければならない（MUST）。成功時は status が `active` に変わり、成功メッセージを SHALL 表示する。失敗時はエラーメッセージを SHALL 表示する。

**Customer Context**

問題解決後のアカウント再開を安全に行う。

#### Scenario: 停止中のアカウントを復旧する (ADMIN-CONSOLE-FE-S014)

- **GIVEN** Operator が operator ロール以上を持つ
- **AND** アカウント status が `suspended`
- **WHEN** Restore ボタンをクリックし、確認ダイアログで Confirm する
- **THEN** status が `active` に変わり、「アカウントを復旧しました」と表示される

#### Scenario: アクティブなアカウントは復旧できない (ADMIN-CONSOLE-FE-S015)

- **GIVEN** アカウント status が `active` である
- **WHEN** 詳細画面を表示する
- **THEN** Restore ボタンが無効化または非表示になっている

---

### Requirement: 監査ログ閲覧

監査ログは新しい順で SHALL 表示する。Operator / 操作種別 / 対象種別 / 日付範囲でフィルター可能でなければならない（SHALL）。各イベントの details JSON を展開表示できなければならない（SHALL）。ページネーションを SHALL 提供する。

**Customer Context**

コンプライアンスと内部監査のため、全操作の履歴を追跡可能にする。

#### Scenario: 監査ログを閲覧する (ADMIN-CONSOLE-FE-S016)

- **GIVEN** 監査ログに複数イベントが存在する
- **WHEN** Audit 画面にアクセスする
- **THEN** 最近のイベントが operator 名、action、target_type、日時とともに表示される

#### Scenario: 操作種別でフィルターする (ADMIN-CONSOLE-FE-S017)

- **GIVEN** `account.suspend` と `account.restore` のイベントが混在している
- **WHEN** 操作フィルターで `account.suspend` を選択する
- **THEN** `account.suspend` イベントのみが表示される

#### Scenario: details JSON を展開する (ADMIN-CONSOLE-FE-S018)

- **GIVEN** 監査イベントの details に `{ reason: "不正利用" }` が含まれている
- **WHEN** イベント行をクリックして展開する
- **THEN** reason などの details 内容が表示される

#### Scenario: 監査ログが空の場合は空状態を表示する (ADMIN-CONSOLE-FE-S019)

- **GIVEN** 監査ログにイベントが 1 件も存在しない
- **WHEN** Audit 画面にアクセスする
- **THEN** 「監査ログはありません」という空状態が表示される

---

### Requirement: Settings 配下でオペレーターを管理する

Settings 画面は admin ロールにのみ SHALL 許可し、`/settings/operators` でオペレーター管理を提供する。非 admin ロールのオペレーターが `/settings` または `/settings/operators` にアクセスした場合は 403 を返さなければならない（MUST）。admin が新規オペレーターを追加した場合、画面は one-time setup token を一度だけ表示し、copy 操作後または画面遷移後に再表示できないことを SHALL 明示する。passkey 未登録オペレーターには setup token 再発行 action を SHALL 提供し、passkey 登録済みオペレーターには再発行 action を表示してはならない（MUST NOT）。

**Customer Context**

Admin Console の運用には複数のオペレーターが必要であり、管理者がメンバーを管理できる必要がある。オペレーター管理は日常業務のトップレベル導線ではなく、Settings 配下にまとめる。

#### Scenario: admin がオペレーター一覧を表示する (ADMIN-CONSOLE-FE-S020)

- **GIVEN** オペレーターの role が `admin` である
- **WHEN** `/settings/operators` にアクセスする
- **THEN** 全オペレーターの email、display_name、role、is_active、last_login_at が表示される

#### Scenario: 非 admin はオペレーター管理画面にアクセスできない (ADMIN-CONSOLE-FE-S021)

- **GIVEN** オペレーターの role が `operator` である
- **WHEN** `/settings/operators` にアクセスする
- **THEN** 403 エラーが表示される
- **AND** Sidebar に Settings リンクが表示されない

#### Scenario: 新規オペレーターを追加する (ADMIN-CONSOLE-FE-S022)

- **GIVEN** オペレーターの role が `admin` である
- **WHEN** 追加フォームで email / display_name / role=operator を入力し送信する
- **THEN** オペレーターが一覧に追加され、成功メッセージと one-time setup token が表示される

#### Scenario: setup token は一度だけコピーできる (ADMIN-CONSOLE-FE-S036)

- **GIVEN** admin が新規オペレーター作成直後の one-time setup token を表示している
- **WHEN** token をコピーして dialog を閉じる
- **THEN** 同じ token は画面上で再表示できず、再発行が必要であることが表示される

#### Scenario: passkey 未登録オペレーターの setup token を再発行できる (ADMIN-CONSOLE-FE-S037)

- **GIVEN** オペレーターが passkey 未登録である
- **WHEN** admin が setup token 再発行を実行する
- **THEN** 新しい one-time setup token が一度だけ表示される

#### Scenario: 重複 email のオペレーター追加はエラーになる (ADMIN-CONSOLE-FE-S023)

- **GIVEN** 同じ email のオペレーターが既に存在する
- **WHEN** 追加フォームで同じ email を入力して送信する
- **THEN** エラーメッセージが表示され、一覧は変化しない

#### Scenario: オペレーターの role を変更する (ADMIN-CONSOLE-FE-S024)

- **GIVEN** オペレーターの role が `admin` である
- **WHEN** 他のオペレーターの role を `operator` から `viewer` に変更する
- **THEN** role が更新され、成功メッセージが表示される

#### Scenario: オペレーターを無効化する (ADMIN-CONSOLE-FE-S025)

- **GIVEN** オペレーターの role が `admin` である
- **WHEN** 他のオペレーターの deactivate をクリックし確認する
- **THEN** is_active が false になり、一覧に反映される

#### Scenario: 自分自身は deactivate できない (ADMIN-CONSOLE-FE-S026)

- **GIVEN** ログイン中のオペレーターが自分自身の行を見ている
- **WHEN** 画面を表示する
- **THEN** 自分の行の deactivate ボタンが無効化されている

---

### Requirement: 共通レイアウトとナビゲーション

Sidebar に Dashboard / Accounts / Audit のリンクを SHALL 表示する。admin role の場合のみ Settings リンクを SHALL 追加表示し、Settings 配下にオペレーター管理への導線を SHALL 表示する。現在の画面のリンクを SHALL ハイライトする。Header にオペレーター名（display_name）と Logout リンクを SHALL 表示する。

**Customer Context**

Admin Console の利用者は日常的に複数の画面を行き来する。一貫したレイアウトと明確なナビゲーションが運用作業の効率を左右する。

#### Scenario: 画面間を移動する (ADMIN-CONSOLE-FE-S027)

- **GIVEN** オペレーターが認証済みである
- **WHEN** Sidebar の Accounts リンクをクリックする
- **THEN** `/accounts` に遷移し、Accounts リンクがハイライトされる

#### Scenario: admin は Settings リンクが見える (ADMIN-CONSOLE-FE-S028)

- **GIVEN** オペレーターの role が `admin` である
- **WHEN** 任意の画面を表示する
- **THEN** Sidebar に Settings リンクが表示される

#### Scenario: 非 admin は Settings リンクが見えない (ADMIN-CONSOLE-FE-S029)

- **GIVEN** オペレーターの role が `admin` 以外である
- **WHEN** 任意の画面を表示する
- **THEN** Sidebar に Settings リンクが表示されない

#### Scenario: Header にオペレーター名が表示される (ADMIN-CONSOLE-FE-S030)

- **GIVEN** オペレーターの display_name が `John Doe` である
- **WHEN** 認証済み状態で任意の画面を表示する
- **THEN** Header に `John Doe` と表示される

#### Scenario: Logout クリックでログアウトする (ADMIN-CONSOLE-FE-S031)

- **GIVEN** オペレーターが認証済みである
- **WHEN** Header の Logout をクリックする
- **THEN** session cookie がクリアされ、`/login` にリダイレクトされる

---

### Requirement: Dashboard

Dashboard は総アカウント数、アクティブアカウント数、停止中アカウント数を SHALL 表示する。総 passkey 数を SHALL 表示する。最近の監査イベント（最新 10 件）を SHALL 表示する。

**Customer Context**

Admin Console の入口として、オペレーターはシステム全体の状態を一目で把握できる必要がある。

#### Scenario: Dashboard に KPI が表示される (ADMIN-CONSOLE-FE-S032)

- **GIVEN** アクティブ 10 件、停止中 2 件のアカウントが存在する
- **WHEN** Dashboard にアクセスする
- **THEN**「総アカウント数: 12」「アクティブ: 10」「停止中: 2」が表示される

#### Scenario: Dashboard に最近の監査ログが表示される (ADMIN-CONSOLE-FE-S033)

- **GIVEN** 監査ログに 15 件のイベントが存在する
- **WHEN** Dashboard にアクセスする
- **THEN** 最新 10 件が表示される

---

### Requirement: MVCS 層間依存と import 制約

Admin Console の frontend code は `packages/admin/app -> packages/admin/domain -> packages/admin/api` の依存方向を SHALL 保つ。`packages/admin/app` は `packages/admin/domain`、`@www-template/ui`、`@www-template/i18n` のみを直接利用し、Admin generated API client を直接 import してはならない（MUST NOT）。`packages/admin/domain` は `packages/admin/api` を通じて Admin backend API を呼び出し、Product API SDK、DB client、server-only module を import してはならない（MUST NOT）。`packages/admin/api` は Admin surface から生成された package-local SDK のみを使用し、Product surface SDK を使用してはならない（MUST NOT）。`packages/admin` は SvelteKit server route handlers、server load/actions、`$lib/server`、Prisma、Valkey、OpenSearch、WebAuthn server library を runtime dependency として使用してはならない（MUST NOT）。これらの制約は lint で強制されなければならない（SHALL）。

**Customer Context**

Admin Console は強権限画面であるため、画面配信 package に backend domain logic、DB 接続、secret 取扱い、server-only action が存在すると、責務境界が曖昧になり監査と保守が難しくなる。運営者は Admin UI を安全に利用しながら、Account domain の判断を Backend に一元化したい。

#### Scenario: Admin app から API client を直接 import すると lint エラーになる (ADMIN-CONSOLE-FE-S038)

- **WHEN** `packages/admin/app` の `.svelte` または route module が Admin generated API client を直接 import する
- **THEN** lint は layer violation として失敗する

#### Scenario: Admin package に server-only module が存在すると lint エラーになる (ADMIN-CONSOLE-FE-S039)

- **WHEN** `packages/admin` に SvelteKit `+server.ts`、`+page.server.ts`、`src/lib/server`、または DB/Valkey/OpenSearch runtime module が存在する
- **THEN** lint は server ownership violation として失敗する

#### Scenario: Admin domain layer は Admin api layer 経由で account data を取得する (ADMIN-CONSOLE-FE-S040)

- **GIVEN** Accounts 画面が account 一覧を表示する
- **WHEN** `packages/admin/domain` が account 検索を実行する
- **THEN** 呼び出しは `packages/admin/api` の generated Admin SDK wrapper を通じて Admin backend に送信される
- **AND** Product SDK は import されない
