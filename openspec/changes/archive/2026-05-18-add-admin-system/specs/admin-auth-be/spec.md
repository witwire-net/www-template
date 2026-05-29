## ADDED Requirements

### Requirement: オペレーターパスキー認証は httpOnly cookie session を発行する

システムは `POST /api/admin/auth/passkey/start` で WebAuthn challenge と ULID `challengeId` を発行し、`ADMIN_VALKEY_URL` が指す共有 Valkey infrastructure の Admin 用 logical DB に `SETEX admin:webauthn:challenge:<challengeId> 300 <record-json>` で TTL 5 分の challenge record を保存しなければならない（SHALL）。Admin は Product auth と同じ Valkey infrastructure を共有し、`ADMIN_VALKEY_URL` と Product `VALKEY_URL` は同じ endpoint かつ異なる logical DB 番号でなければならない（MUST）。Admin runtime は Admin 用 logical DB に向いた client のみを生成し、`admin:*` key prefix だけを読み書きしなければならない（MUST）。登録済みかつ active な operator の challenge record は `challengeId`、`challenge`、`type`（login / passkey-add / bootstrap / operator-setup）、`operatorId`、`email`、`createdAt`、`expiresAt` を MUST 含む。未登録 email、inactive operator、passkey 未登録 operator の login start は enumeration を防ぐため、登録済み operator と同じ HTTP status / response shape / Cache-Control を返し、`decoy=true`、`operatorId=null`、正規化済み email、`createdAt`、`expiresAt` を含む decoy challenge record を保存しなければならない（MUST）。decoy challenge の finish は non-revealing な 401 とし、cookie を設定してはならない（MUST NOT）。`POST /api/admin/auth/passkey/finish` は request body の `challengeId` で Admin 用 logical DB から challenge record を `GETDEL` で取得・消費し、record の `type` / `operatorId` / `email` と assertion credential owner の一致を検証しなければならない（SHALL）。検証成功時は ULID `sessionId` と cryptographically random `jti` を生成し、Admin 用 logical DB に `SETEX admin:session:<sessionId> 86400 <record-json>` で active session record を保存しなければならない（MUST）。session record は `sessionId`、`jti`、`operatorId`、`email`、`createdAt`、`expiresAt`、`lastSeenAt` を MUST 含む。JWT は `operatorId`、`email`、`role` に加えて `sessionId` と `jti` を MUST claims に含め、`Set-Cookie: admin_session=<JWT>; HttpOnly; Secure; SameSite=Lax; Path=/` で設定しなければならない（SHALL）。Valkey に存在しない、TTL 切れ、type 不一致、operator 不一致、decoy challenge の finish は拒否しなければならない（MUST）。`hooks.server.ts` は全リクエストで `admin_session` cookie を読み取り、JWT 署名 / exp を検証し、JWT の `sessionId` / `jti` に対応する active session record を Admin 用 logical DB から取得し、session record の `jti` / `operatorId` と一致する場合だけ有効な session として扱わなければならない（MUST）。有効な場合は Admin-owned schema から現在のオペレーター record を再取得し、`event.locals.operator` に現在の role / is_active / sessionId を設定しなければならない（SHALL）。認可判断は JWT の role claim ではなく DB の現在 role を MUST 使用する。JWT と active session record の有効期限は 24 時間でなければならない（SHALL）。logout は `admin:session:<sessionId>` を削除または revoked marker に置換し、cookie と CSRF cookie をクリアしなければならない（MUST）。検証失敗時は cookie をクリアし、保護 route では `/login` へリダイレクトしなければならない（MUST）。pre-auth route は cookie 検証失敗だけで拒否してはならない（MUST NOT）。

**Customer Context**

Admin Console は SvelteKit full-stack アプリケーションであり、server-side の認証チェックには httpOnly cookie が最も単純かつ安全な方式である。challenge は Product auth と同じ Valkey infrastructure を共有し、`ADMIN_VALKEY_URL` が明示する別 logical DB 番号で管理する。Product と Admin は DB 番号と `admin:*` key prefix により衝突を避け、運用対象のインフラを増やさない。

#### Scenario: パスキーログイン成功時に httpOnly cookie が設定される (ADMIN-AUTH-BE-S001)

- **GIVEN** オペレーターが登録済みの passkey credential を持つ
- **WHEN** `POST /api/admin/auth/passkey/finish` が有効な assertion とともに呼び出される
- **THEN** server は `Set-Cookie` を response に含め、303 redirect to `/` を返す

#### Scenario: 無効な assertion 署名は拒否される (ADMIN-AUTH-BE-S002)

- **GIVEN** WebAuthn assertion の署名が改ざんされている
- **WHEN** `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** server は 401 を返し、cookie は設定されない

#### Scenario: 存在しない credential_handle は拒否される (ADMIN-AUTH-BE-S003)

- **GIVEN** assertion 内の credential_handle が `admin.operator_passkeys` に存在しない
- **WHEN** `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** server は 401 を返し、cookie は設定されない

#### Scenario: 期限切れの challenge は拒否される (ADMIN-AUTH-BE-S004)

- **GIVEN** WebAuthn challenge が発行から 5 分以上経過して Valkey の TTL で消えている
- **WHEN** `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** server は 401 を返し、cookie は設定されない

#### Scenario: 消費済み challenge の再利用は拒否される (ADMIN-AUTH-BE-S053)

- **GIVEN** challenge が既に finish で消費され Valkey から削除されている
- **WHEN** 同じ challenge で再度 `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** server は 401 を返し、cookie は設定されない

#### Scenario: challengeId とオペレーターの binding 不一致は拒否される (ADMIN-AUTH-BE-S034)

- **GIVEN** オペレーター A の login challenge record が Valkey に保存されている
- **WHEN** オペレーター B の credential assertion が同じ `challengeId` で `POST /api/admin/auth/passkey/finish` に送信される
- **THEN** server は 401 を返し、challenge は消費済みとして再利用できない

#### Scenario: 未登録 email の login start は operator 存在を漏らさない (ADMIN-AUTH-BE-S049)

- **GIVEN** 入力された email が `admin.operators` に存在しない、または operator が inactive である
- **WHEN** `POST /api/admin/auth/passkey/start` が呼び出される
- **THEN** server は登録済み active operator と同じ HTTP status、response shape、`Cache-Control: no-store` を返す
- **AND** 共有 Valkey infrastructure の Admin 用 logical DB には `decoy=true` の challenge record が保存され、実 operatorId は含まれない
- **AND** 当該 `challengeId` で `POST /api/admin/auth/passkey/finish` を呼び出しても non-revealing な 401 になり、cookie は設定されない

#### Scenario: Admin Valkey は Product auth Valkey と同じ infrastructure の別 DB を使う (ADMIN-AUTH-BE-S048)

- **GIVEN** Admin Console が起動する
- **WHEN** runtime config を検証する
- **THEN** `ADMIN_VALKEY_URL` は設定必須であり、明示的な logical DB 番号を含む
- **AND** `VALKEY_URL` または `PRODUCT_VALKEY_URL` が存在する場合、`ADMIN_VALKEY_URL` と同じ Valkey infrastructure endpoint かつ異なる logical DB 番号でなければ起動を拒否する
- **AND** Admin runtime は Admin 用 logical DB に向いた client だけを生成・操作する
- **AND** Admin が読み書きする key は `admin:*` prefix のみであり、`auth:*` / `session:*` / `recovery:*` key は読み書きしない

#### Scenario: hooks が有効な cookie を検証する (ADMIN-AUTH-BE-S005)

- **GIVEN** 有効な `admin_session` cookie がリクエストに含まれている
- **WHEN** 保護された route にアクセスする
- **THEN** `event.locals.operator` に DB の現在値から `{ id, email, role, sessionId, jti }` が設定され、リクエストが処理される

#### Scenario: 期限切れ cookie は拒否される (ADMIN-AUTH-BE-S006)

- **GIVEN** `admin_session` cookie の JWT exp が過去である
- **WHEN** 保護された route にアクセスする
- **THEN** `Set-Cookie` で `admin_session` がクリアされ、`/login` にリダイレクトされる

#### Scenario: 改ざんされた JWT は拒否される (ADMIN-AUTH-BE-S007)

- **GIVEN** `admin_session` cookie の JWT 署名が無効である
- **WHEN** 保護された route にアクセスする
- **THEN** cookie がクリアされ、`/login` にリダイレクトされる

#### Scenario: operator が非アクティブ化されていると拒否される (ADMIN-AUTH-BE-S008)

- **GIVEN** `admin_session` JWT は有効だが、`admin.operators.is_active` が false である
- **WHEN** 保護された route にアクセスする
- **THEN** cookie がクリアされ、`/login` にリダイレクトされる

#### Scenario: login 成功時に `last_login_at` が更新される (ADMIN-AUTH-BE-S009)

- **GIVEN** Operator がログインに成功する
- **WHEN** JWT が発行される
- **THEN** `admin.operators.last_login_at` が現在時刻に更新される

#### Scenario: JWT 内の古い role claim は認可に使われない (ADMIN-AUTH-BE-S045)

- **GIVEN** JWT の role claim は `admin` だが、Admin-owned schema の現在 role が `viewer` に変更済みである
- **WHEN** 保護された admin-only route にアクセスする
- **THEN** hook は DB の現在 role=`viewer` を `event.locals.operator` に設定し、admin 権限を許可しない

#### Scenario: logout 後の admin_session は再利用できない (ADMIN-AUTH-BE-S054)

- **GIVEN** オペレーターが有効な `admin_session` cookie と Valkey active session record を持っている
- **WHEN** logout が実行され、同じ cookie で保護 route に再アクセスする
- **THEN** `admin:session:<sessionId>` は無効化済みであり、hook は cookie を拒否して `/login` へリダイレクトする

#### Scenario: JWT の sessionId と jti が Valkey session record と一致しない場合は拒否される (ADMIN-AUTH-BE-S055)

- **GIVEN** `admin_session` JWT の署名と exp は有効だが、Valkey の `admin:session:<sessionId>` が存在しない、または `jti` が一致しない
- **WHEN** 保護 route にアクセスする
- **THEN** hook は session を拒否し、cookie をクリアして `/login` へリダイレクトする

---

### Requirement: オペレーターは複数の passkey を登録・管理できる

`GET /api/admin/auth/passkeys` は認証済みオペレーターの登録済み passkey credential 一覧を SHALL 返す。`POST /api/admin/auth/passkeys/start` は WebAuthn 登録 challenge を発行し Valkey に SHALL 保存する。`POST /api/admin/auth/passkeys/finish` は attestation を検証し Valkey の challenge を消費して新しい credential を SHALL 追加する。`DELETE /api/admin/auth/passkeys/{id}` は指定 passkey を SHALL 削除するが、残り 1 件の場合は MUST 拒否する。全 passkey 管理エンドポイントは有効な `admin_session` cookie を MUST 必須とする。他のオペレーターの passkey を操作する試みは SHALL 拒否されなければならない。

**Customer Context**

オペレーターは複数の device で Admin Console にアクセスする。passkey を追加・削除できることで、device 追加や紛失時に安全な鍵管理が可能になる。

#### Scenario: 登録済み passkey 一覧を取得できる (ADMIN-AUTH-BE-S010)

- **GIVEN** オペレーターが認証済みで 2 件の passkey を登録済みである
- **WHEN** `GET /api/admin/auth/passkeys` を呼び出す
- **THEN** 2 件の credential が credential_handle / backup_eligible / backup_state / transports / created_at とともに返される

#### Scenario: 新しい passkey を追加できる (ADMIN-AUTH-BE-S011)

- **GIVEN** オペレーターが認証済みである
- **WHEN** `POST /api/admin/auth/passkeys/start` → WebAuthn → `POST /api/admin/auth/passkeys/finish` を実行する
- **THEN** 新しい credential が `admin.operator_passkeys` に追加され、既存 credential は保持される

#### Scenario: 最後の 1 件の passkey は削除できない (ADMIN-AUTH-BE-S012)

- **GIVEN** オペレーターが passkey credential を 1 件のみ持つ
- **WHEN** `DELETE /api/admin/auth/passkeys/{id}` を呼び出す
- **THEN** server は 400 を返し、passkey は削除されない

#### Scenario: 2 件以上ある場合は passkey を削除できる (ADMIN-AUTH-BE-S013)

- **GIVEN** オペレーターが passkey credential を 2 件以上持つ
- **WHEN** `DELETE /api/admin/auth/passkeys/{id}` で特定の 1 件を削除する
- **THEN** 指定された credential のみが削除され、残りは保持される

#### Scenario: 未認証リクエストは passkey 管理 API を利用できない (ADMIN-AUTH-BE-S014)

- **GIVEN** 有効な `admin_session` cookie が存在しない
- **WHEN** `/api/admin/auth/passkeys` 以下のエンドポイントを呼び出す
- **THEN** server は 401 を返す

#### Scenario: 他の Operator の passkey は操作できない (ADMIN-AUTH-BE-S015)

- **GIVEN** オペレーター A が認証済みである
- **WHEN** オペレーター B の passkey ID を指定して `DELETE /api/admin/auth/passkeys/{id}` を呼び出す
- **THEN** server は 403 を返し、オペレーター B の passkey は変化しない

#### Scenario: WebAuthn 登録時に user verification を要求する (ADMIN-AUTH-BE-S016)

- **GIVEN** オペレーターが passkey 追加を開始している
- **WHEN** WebAuthn registration ceremony が実行される
- **THEN** `userVerification` が `required` で要求される

---

### Requirement: パスキー認証時にも user verification を要求する

ログイン時の `POST /api/admin/auth/passkey/start` で発行する WebAuthn options は `userVerification: "required"` を MUST 指定する。`POST /api/admin/auth/passkey/finish` の server-side 検証は、`userVerification` フラグが `true` でない assertion を MUST 拒否する。passkey 管理の追加時も同様に user verification を MUST 要求する。

**Customer Context**

Admin Console は強権限システムであるため、device 所持だけでは不十分で、生体認証または PIN による user verification が必須である。

#### Scenario: user verification なしの assertion は拒否される (ADMIN-AUTH-BE-S017)

- **GIVEN** WebAuthn assertion の `userVerification` フラグが `false` である
- **WHEN** `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** server は 401 を返し、session は発行されない

#### Scenario: user verification ありの assertion は受理される (ADMIN-AUTH-BE-S018)

- **GIVEN** WebAuthn assertion の `userVerification` フラグが `true` である
- **WHEN** `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** 検証に成功し、JWT cookie が設定される

---

### Requirement: 初回起動セットアップは最初の admin オペレーターを作成する

`POST /api/admin/auth/setup/start` は `admin.operators` が 0 件の場合のみ、email / display_name と bootstrap secret を検証して WebAuthn 登録 challenge を Valkey に SHALL 保存する。bootstrap は `ADMIN_BOOTSTRAP_ENABLED=true`、`ADMIN_BOOTSTRAP_SECRET_HASH`、`ADMIN_BOOTSTRAP_EXPIRES_AT` がすべて有効な場合のみ許可されなければならない（MUST）。bootstrap secret の平文は DB、監査ログ、OpenSearch、application log に保存してはならない（MUST NOT）。`POST /api/admin/auth/setup/finish` は attestation を検証し、同一 transaction で role=`admin` の最初のオペレーター作成、passkey credential 登録、JWT cookie 発行を完了しなければならない（MUST）。`admin.operators` が 1 件以上存在する場合、setup start / finish は MUST 拒否される。初回オペレーターは DB migration、seed、直接 SQL によって作成してはならない（MUST NOT）。

**Customer Context**

初回起動時に DB seed で固定オペレーターを作ると、secret の保管・配布・漏洩時の取り扱いが複雑になる。一方で「オペレーター 0 件」だけを条件にすると公開された空 Admin-owned schema を第三者に乗っ取られる。オペレーター 0 件、明示 enable flag、短期 bootstrap secret、有効期限をすべて満たす場合だけ、最初の admin 作成と passkey 登録を同一 flow で完了する。

#### Scenario: オペレーター 0 件時に最初の admin オペレーターを作成できる (ADMIN-AUTH-BE-S019)

- **GIVEN** `admin.operators` が 0 件であり、`ADMIN_BOOTSTRAP_ENABLED=true`、有効な `ADMIN_BOOTSTRAP_SECRET_HASH`、未来の `ADMIN_BOOTSTRAP_EXPIRES_AT` が設定されている
- **WHEN** `POST /api/admin/auth/setup/start` に email / display_name / bootstrap secret を送信し、`POST /api/admin/auth/setup/finish` に attestation を送信する
- **THEN** role=`admin` のオペレーターと passkey credential が作成され、JWT cookie が設定される

#### Scenario: オペレーターが存在する場合は初回 setup start が拒否される (ADMIN-AUTH-BE-S020)

- **GIVEN** `admin.operators` が 1 件以上存在する
- **WHEN** `POST /api/admin/auth/setup/start` を呼び出す
- **THEN** server は 400 または 403 を返し、challenge は発行されない

#### Scenario: 初回 setup finish 前に別オペレーターが作成された場合は拒否される (ADMIN-AUTH-BE-S021)

- **GIVEN** setup start 後、finish 前に別 request が最初のオペレーターを作成済みである
- **WHEN** `POST /api/admin/auth/setup/finish` を呼び出す
- **THEN** server は transaction 内で operators count を再確認して拒否し、passkey は作成されない

#### Scenario: 初回 setup は role を admin に固定する (ADMIN-AUTH-BE-S022)

- **GIVEN** `admin.operators` が 0 件である
- **WHEN** 初回 setup を完了する
- **THEN** 作成されるオペレーターの role は request body に関係なく `admin` である

#### Scenario: bootstrap secret が無効な場合は初回 setup を開始できない (ADMIN-AUTH-BE-S023)

- **GIVEN** Admin-owned schema migration 直後で `admin.operators` が 0 件だが、bootstrap secret が不一致または期限切れである
- **WHEN** 初回 setup を開始する
- **THEN** server は 403 を返し、challenge は発行されない

#### Scenario: bootstrap enable flag が無効な場合は初回 setup を開始できない (ADMIN-AUTH-BE-S046)

- **GIVEN** `admin.operators` が 0 件だが、`ADMIN_BOOTSTRAP_ENABLED` が `true` ではない
- **WHEN** `POST /api/admin/auth/setup/start` を呼び出す
- **THEN** server は 403 を返し、challenge は発行されない

---

### Requirement: 追加オペレーターは setup token で初回 passkey を登録する

admin によるオペレーター作成は `admin.operators` にオペレーターを作成し、`setup_token_hash` カラムに bcrypt ハッシュを保存しなければならない（SHALL）。setup token は one-time token として 24 時間以内の `setup_token_expires_at` を MUST 持つ。`POST /api/admin/auth/operator-setup/start` は有効な setup token を bcrypt 検証後、WebAuthn 登録 challenge を Valkey に SHALL 保存する。`POST /api/admin/auth/operator-setup/finish` は attestation を検証し、Valkey の challenge を消費して passkey を登録し、`setup_token_hash` と `setup_token_expires_at` を NULL に更新して消費しなければならない（MUST）。消費済みまたは期限切れの setup token での再試行は MUST 拒否される。

**Customer Context**

admin のオペレーター追加で作成されたオペレーター record は passkey credential を持たない。追加オペレーターは one-time setup token を用いて初回 passkey を登録する。setup token のハッシュ化には bcrypt を使用する。

#### Scenario: admin が追加したオペレーターが setup token で初回 passkey を登録できる (ADMIN-AUTH-BE-S040)

- **GIVEN** admin が追加したオペレーターが有効な `setup_token_hash` を持ち、passkey credential を持たない
- **WHEN** `POST /api/admin/auth/operator-setup/start` に正しい token → `POST /api/admin/auth/operator-setup/finish` に attestation を送信する
- **THEN** passkey credential が登録され、JWT cookie が設定され、`setup_token_hash` が NULL に更新される

#### Scenario: 不正な setup token は拒否される (ADMIN-AUTH-BE-S041)

- **GIVEN** setup token が bcrypt 検証で不一致
- **WHEN** `POST /api/admin/auth/operator-setup/start` に誤った token を送信する
- **THEN** server は non-revealing なエラーを返し、challenge は発行されない

#### Scenario: 期限切れの setup token は拒否される (ADMIN-AUTH-BE-S042)

- **GIVEN** setup token の `setup_token_expires_at` が過去である
- **WHEN** `POST /api/admin/auth/operator-setup/start` に正しい token 値を送信する
- **THEN** server は non-revealing なエラーを返す

#### Scenario: 消費済み setup token の再利用は拒否される (ADMIN-AUTH-BE-S043)

- **GIVEN** setup token が既に消費され `setup_token_hash` が NULL である
- **WHEN** `POST /api/admin/auth/operator-setup/start` を呼び出す
- **THEN** server は non-revealing なエラーを返す

#### Scenario: 既に passkey 登録済みのオペレーターは setup token 登録を利用できない (ADMIN-AUTH-BE-S044)

- **GIVEN** オペレーターが既に passkey credential を持つ
- **WHEN** `POST /api/admin/auth/operator-setup/start` を呼び出す
- **THEN** server は 400 を返す

### Requirement: passkey 保存テーブルは credential 情報を完全に保持する

`admin.operator_passkeys` は id (ULID)、operator_id (FK)、credential_handle (UNIQUE)、public_key、sign_count、aaguid、backup_eligible、backup_state、transports (JSONB)、created_at を SHALL 保持する。`sign_count` は認証成功のたびに更新されなければならない（SHALL）。`sign_count` が前回値から減少している assertion は replay attack として MUST 拒否する。

**Customer Context**

passkey 認証では保存された credential public key で assertion の署名を検証する。sign_count による replay attack 検出も必要である。

#### Scenario: credential が保存され検証に使われる (ADMIN-AUTH-BE-S024)

- **GIVEN** Operator が passkey を登録済みである
- **WHEN** ログイン時に `POST /api/admin/auth/passkey/finish` が呼び出される
- **THEN** 保存された `public_key` と `sign_count` を用いて assertion が検証される

#### Scenario: sign_count 減少は拒否される (ADMIN-AUTH-BE-S025)

- **GIVEN** passkey credential の `sign_count` が 10 である
- **WHEN** assertion の sign_count が 8 である
- **THEN** server は 401 を返し、session は発行されない

#### Scenario: 認証成功時に sign_count が更新される (ADMIN-AUTH-BE-S026)

- **GIVEN** passkey credential の `sign_count` が 5 である
- **WHEN** ログイン assertion の sign_count が 6 である
- **THEN** 認証成功後、DB の `sign_count` が 6 に更新される

#### Scenario: 重複 credential_handle の登録は拒否される (ADMIN-AUTH-BE-S027)

- **GIVEN** `credential_handle` X が既に別の Operator に登録済みである
- **WHEN** 別の Operator が同じ `credential_handle` X で passkey を登録しようとする
- **THEN** server は 409 を返し、credential は追加されない

---

### Requirement: Admin auth には rate limit と temporary lock を適用する

`POST /api/admin/auth/passkey/start` は IP ごとに 5 回 / 5 分の rate limit を MUST 適用し、counter は Admin 用 logical DB に `SETEX admin:rate:passkey-start:<ip> 300 <count>` で保存する。`POST /api/admin/auth/passkey/finish` の連続失敗（同一 IP 10 回 / 15 分）は temporary lock（15 分）を MUST 発動し、lock state は Admin 用 logical DB に `SETEX admin:lock:passkey-finish:<ip> 900 1` で保存する。`POST /api/admin/auth/setup/start` は bootstrap secret brute-force を防ぐため IP ごとに 3 回 / 15 分の rate limit と 15 分 lock を MUST 適用し、counter / lock は `admin:rate:bootstrap:<ip>` / `admin:lock:bootstrap:<ip>` に保存する。`POST /api/admin/auth/operator-setup/start` は setup token brute-force を防ぐため IP ごとに 5 回 / 15 分、かつ token fingerprint ごとに 5 回 / 15 分の rate limit と 15 分 lock を MUST 適用する。token fingerprint は平文 token を保存せず、server secret による HMAC などの irreversible value として `admin:rate:operator-setup:<fingerprint>` / `admin:lock:operator-setup:<fingerprint>` に保存する。throttle / lock 中は non-revealing なエラーを返さなければならない（SHALL）。pre-auth rate limit / lock の判定に必要な Valkey が unavailable な場合は 503 fail-close とし、bootstrap secret や setup token の検証を実行してはならない（MUST NOT）。

**Customer Context**

Admin Console は強権限システムであるため、厳格な rate limiting が必要である。

#### Scenario: throttle 超過時は追加 challenge を発行しない (ADMIN-AUTH-BE-S028)

- **GIVEN** 同一 IP から 5 分以内に 6 回目の start リクエストがある
- **WHEN** throttle limit を超過する
- **THEN** server は 429 を返し、新しい challenge を発行しない

#### Scenario: 連続 finish 失敗で temporary lock が発動する (ADMIN-AUTH-BE-S029)

- **GIVEN** 同一 IP から 15 分以内に 10 回の finish が失敗している
- **WHEN** 11 回目の finish を試みる
- **THEN** server は 429 を返し、assertion 検証を実行しない

#### Scenario: temporary lock 期間終了後は再試行できる (ADMIN-AUTH-BE-S030)

- **GIVEN** temporary lock が発動してから 15 分が経過して Valkey の TTL で消えている
- **WHEN** finish を試みる
- **THEN** 検証が実行される

#### Scenario: Valkey が unavailable な場合でも fail-close する (ADMIN-AUTH-BE-S033)

- **GIVEN** Valkey が応答しない
- **WHEN** `POST /api/admin/auth/passkey/start` が呼び出される
- **THEN** server は 503 を返し、challenge は発行されない

#### Scenario: bootstrap secret の brute-force は rate limit される (ADMIN-AUTH-BE-S050)

- **GIVEN** 同一 IP から 15 分以内に `/api/admin/auth/setup/start` の bootstrap secret 検証が 3 回失敗している
- **WHEN** 4 回目の setup start が呼び出される
- **THEN** server は non-revealing な 429 を返し、bootstrap secret 検証と challenge 発行を実行しない

#### Scenario: operator setup token の brute-force は IP と token fingerprint で rate limit される (ADMIN-AUTH-BE-S051)

- **GIVEN** 同一 IP または同一 setup token fingerprint で 15 分以内に `/api/admin/auth/operator-setup/start` が 5 回失敗している
- **WHEN** 次の operator setup start が呼び出される
- **THEN** server は non-revealing な 429 を返し、bcrypt 検証と challenge 発行を実行しない

#### Scenario: pre-auth secret/token 検証は Valkey unavailable 時に fail-close する (ADMIN-AUTH-BE-S052)

- **GIVEN** 共有 Valkey infrastructure の Admin 用 logical DB が応答しない
- **WHEN** `/api/admin/auth/setup/start` または `/api/admin/auth/operator-setup/start` が呼び出される
- **THEN** server は 503 を返し、bootstrap secret / setup token の検証と challenge 発行を実行しない

---

### Requirement: Admin mutation route は CSRF と Origin を検証する

Admin Console の cookie-authenticated mutation route（pre-auth route を除く）は、non-GET request で `Origin` が `ADMIN_ORIGIN` allowlist に一致することを MUST 検証し、signed double-submit CSRF token を MUST 検証する。CSRF token は `admin_csrf` cookie と `X-CSRF-Token` header または form hidden field の両方で提示され、JWT claim と Valkey active session record で検証済みの `sessionId` と `jti` に束縛された HMAC 署名を SHALL 持つ。Origin 欠落、不許可 Origin、token 欠落、不一致、期限切れ、sessionId / jti 不一致は 403 で拒否しなければならない（MUST）。pre-auth route（`/api/admin/auth/passkey/start`、`/api/admin/auth/passkey/finish`、`/api/admin/auth/setup/start`、`/api/admin/auth/setup/finish`、`/api/admin/auth/operator-setup/start`、`/api/admin/auth/operator-setup/finish`）は session-bound CSRF token を要求してはならない（MUST NOT）が、Origin allowlist、rate limit、WebAuthn challenge binding、bootstrap/setup token 検証を MUST 適用する。

**Customer Context**

Admin Console は httpOnly cookie session を使用するため、browser が cross-site request に cookie を自動送信するリスクがある。SameSite だけに依存せず、Origin と CSRF token の両方で強権限 mutation を保護する。

#### Scenario: 同一 Origin と正しい CSRF token の mutation は許可される (ADMIN-AUTH-BE-S036)

- **GIVEN** Operator が有効な session cookie と正しい `admin_csrf` token を持つ
- **WHEN** `Origin` が `ADMIN_ORIGIN` と一致し、token が header または form field で送信される
- **THEN** mutation route は処理を継続する

#### Scenario: cross-site Origin の mutation は拒否される (ADMIN-AUTH-BE-S037)

- **GIVEN** request の `Origin` が `ADMIN_ORIGIN` と一致しない
- **WHEN** cookie-authenticated mutation route が呼び出される
- **THEN** server は 403 を返し、mutation は実行されない

#### Scenario: CSRF token 不一致の mutation は拒否される (ADMIN-AUTH-BE-S038)

- **GIVEN** request が有効な session cookie を持つが CSRF token が欠落または不一致である
- **WHEN** cookie-authenticated mutation route が呼び出される
- **THEN** server は 403 を返し、mutation は実行されない

#### Scenario: pre-auth passkey start は session-bound CSRF なしで実行できる (ADMIN-AUTH-BE-S047)

- **GIVEN** request が `admin_session` と `admin_csrf` を持たないが、`Origin` は `ADMIN_ORIGIN` と一致する
- **WHEN** `POST /api/admin/auth/passkey/start` を呼び出す
- **THEN** server は session-bound CSRF 不在を理由に拒否せず、rate limit と non-revealing 認証フローを継続する

---

### Requirement: session cookie は安全な属性で設定される

全 `Set-Cookie` response は `HttpOnly; Secure; SameSite=Lax; Path=/` を MUST 含む。development 環境では `Secure` を省略可能としなければならない（SHALL）。

**Customer Context**

httpOnly cookie は client-side JavaScript からのアクセスを防ぐ。Secure 属性は HTTPS でのみ送信されることを保証する。

#### Scenario: 本番環境では Secure 属性が付与される (ADMIN-AUTH-BE-S031)

- **GIVEN** 環境が development ではない
- **WHEN** ログイン成功時に cookie が設定される
- **THEN** `Set-Cookie` に `Secure` が含まれる

#### Scenario: cookie が Path=/ である (ADMIN-AUTH-BE-S032)

- **WHEN** ログイン成功時に cookie が設定される
- **THEN** `Set-Cookie` に `Path=/` が含まれ、Admin Console の全 path で cookie が送信される
