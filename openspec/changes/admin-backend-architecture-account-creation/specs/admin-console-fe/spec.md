## ADDED Requirements

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

## MODIFIED Requirements

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
