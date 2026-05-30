## MODIFIED Requirements

### Requirement: オペレーターパスキー認証は httpOnly cookie session を発行する

Admin auth endpoints は Admin API binary の same-origin `/api/v1/auth/*` surface として提供されなければならない（SHALL）。Product API binary は Admin auth endpoints を register してはならない（MUST NOT）。Admin auth endpoints は Product user bearer session を operator session として扱ってはならない（MUST NOT）。Operator passkey login は Admin operator auth domain/application を使い、operator accessToken と operator refreshToken Cookie を発行しなければならない（SHALL）。Operator token は operator ID / operator session ID / operator role / active state / CSRF binding / Admin Valkey logical DB / `admin:*` key prefix に束縛され、Product account ID、Product account status、Product account session state と混同してはならない（MUST NOT）。Admin operator auth は Product account auth domain/application を import してはならず、共有できるのは account/operator の意味を持たない signer、Cookie 属性、ULID、TTL などの中立 primitive だけである（MUST）。Operator session validation は Admin backend middleware で実行し、request context に現在の operator、role、session ID、CSRF binding を設定しなければならない（SHALL）。Admin auth response は `Cache-Control: no-store` を含み、session / setup / challenge state を cache 可能にしてはならない（MUST NOT）。

**Customer Context**

Admin auth は operator passkey、bearer accessToken、HttpOnly refreshToken Cookie、CSRF、setup token を扱う強権限境界である。認証 API が Product host や画面配信 package に分散すると、Origin 検証、session 検証、監査、rate limit の責務が分かれ安全性が下がる。

#### Scenario: Product host では Admin login API が到達不能である (ADMIN-AUTH-BE-S056)

- **GIVEN** Product API binary が起動している
- **WHEN** Admin operator auth contract の request を Product host の同一 relative path `/api/v1/auth/passkey/start` に送信する
- **THEN** Product API binary は Admin auth handler、Admin generated binding、Admin Valkey namespace、Admin operator domain を実行しない
- **AND** Product account auth handler が同じ relative path を持つ場合でも、Admin operator session、Admin CSRF、Admin audit side effect は発生しない

#### Scenario: Admin middleware が operator accessToken を検証して request context に設定する (ADMIN-AUTH-BE-S057)

- **GIVEN** 有効な operator accessToken と Admin Valkey session record が存在する
- **WHEN** protected Admin API を呼び出す
- **THEN** Admin backend middleware は現在の operator、role、session ID、CSRF binding を context に設定する

#### Scenario: Product bearer token は Admin auth session として扱われない (ADMIN-AUTH-BE-S058)

- **GIVEN** request が Product bearer token を持つ
- **WHEN** protected Admin auth API を呼び出す
- **THEN** Admin backend は operator session 不在として拒否する
- **AND** Product account auth domain/application へ判定を委譲しない

### Requirement: パスキー認証時にも user verification を要求する

Admin operator passkey login と passkey registration は Admin API binary の same-origin `/api/v1/auth/*` surface で WebAuthn `userVerification="required"` を要求しなければならない（MUST）。Admin backend は assertion / attestation の user verification flag を検証し、user verification が成立しない credential を拒否しなければならない（MUST）。成功時は Admin OperatorAuth domain/application が operator accessToken と Admin refreshToken Cookie を発行し、Product account auth domain/application を使用してはならない（MUST NOT）。

**Customer Context**

Admin Console は強権限操作を扱うため、device 所持だけでは不十分である。Admin operator auth は Product account auth と別に user verification を検証し、operator session を発行する必要がある。

#### Scenario: user verification なしの Admin assertion は拒否される (ADMIN-AUTH-BE-S073)

- **GIVEN** WebAuthn assertion の `userVerification` flag が false である
- **WHEN** Admin backend の `/api/v1/auth/passkey/finish` を呼び出す
- **THEN** Admin OperatorAuth domain/application は session 発行を拒否する

#### Scenario: user verification ありの Admin assertion は operator session を発行する (ADMIN-AUTH-BE-S074)

- **GIVEN** WebAuthn assertion の `userVerification` flag が true で、operator credential が valid である
- **WHEN** Admin backend の `/api/v1/auth/passkey/finish` を呼び出す
- **THEN** operator accessToken が response body に返り、Admin refreshToken は `HttpOnly; Secure; SameSite=Lax; Path=/` Cookie として設定される

### Requirement: Admin mutation route は CSRF と Origin を検証する

Admin backend は Admin ドメインと一致する Origin の credentialed request だけを受け付けなければならない（MUST）。Admin frontend と Admin backend は同一 Admin ドメインでホストされるため、Admin API は別 origin 通信の許可設定に依存してはならない（MUST NOT）。Credentialed Admin API request は allowed Origin、allowed method/header、operator accessToken、operator session record、CSRF token binding を検証しなければならない（MUST）。CSRF token は session ID と jti に束縛され、Admin frontend が header として提示できなければならない（SHALL）。Pre-auth WebAuthn start / finish / setup endpoints は session-bound CSRF を要求してはならないが、Origin allowlist と rate limit を MUST 適用する。

**Customer Context**

Admin frontend と Admin backend は同一 Admin ドメインで運用され、Cloudflare が静的 frontend と GoServer の `/api/v1/*` を振り分ける。Cookie、Origin、CSRF の設定が曖昧だと、cross-site request や credential 送信の失敗によって安全性または可用性が損なわれる。

#### Scenario: 許可されていない Origin の Admin mutation は拒否される (ADMIN-AUTH-BE-S059)

- **GIVEN** request の Origin が Admin frontend allowlist に含まれない
- **WHEN** Admin account creation API を呼び出す
- **THEN** Admin backend は mutation を実行せず 403 を返す

#### Scenario: CSRF token が session と一致しない mutation は拒否される (ADMIN-AUTH-BE-S060)

- **GIVEN** request が有効な operator accessToken を持つが、CSRF token が別 session に束縛されている
- **WHEN** protected Admin mutation API を呼び出す
- **THEN** Admin backend は 403 を返し、mutation を実行しない

#### Scenario: pre-auth passkey start は session-bound CSRF なしで Origin を検証する (ADMIN-AUTH-BE-S061)

- **GIVEN** request が operator accessToken と CSRF token を持たないが、Origin は allowlist に含まれる
- **WHEN** `/api/v1/auth/passkey/start` を呼び出す
- **THEN** Admin backend は session-bound CSRF 不在を理由に拒否せず、Origin と rate limit を検証して処理を継続する

### Requirement: Admin auth には rate limit と temporary lock を適用する

Admin backend は Admin 用 Valkey logical DB 番号を明示した connection URL を必須にしなければならない（MUST）。Admin Valkey endpoint は Product Valkey endpoint と同じ infrastructure を指し、logical DB 番号は異ならなければならない（MUST）。Admin backend は `admin:*` key prefix のみを読み書きし、Product auth prefix を読み書きしてはならない（MUST NOT）。logical DB 番号または endpoint 境界が不正な場合、Admin backend は fail-close で起動を拒否しなければならない（MUST）。Admin auth throttle、temporary lock、challenge、session、setup token verification state は Admin logical DB 内の `admin:*` key に保存されなければならない（SHALL）。

**Customer Context**

Admin auth state と Product auth state は同じ Valkey infrastructure を共有するが、同じ logical DB や key prefix に混在すると challenge、session、rate limit state が衝突する。インフラ運用を簡素化しながら用途境界を守る必要がある。

#### Scenario: Admin と Product の Valkey logical DB が同じ場合は起動しない (ADMIN-AUTH-BE-S062)

- **GIVEN** Admin Valkey URL と Product Valkey URL が同じ endpoint かつ同じ logical DB 番号である
- **WHEN** Admin backend が runtime config を検証する
- **THEN** Admin backend は fail-close で起動を拒否する

#### Scenario: Admin backend は Product auth key prefix を読み書きしない (ADMIN-AUTH-BE-S063)

- **GIVEN** Admin backend が passkey challenge を作成する
- **WHEN** Valkey に保存された key を確認する
- **THEN** key は `admin:*` prefix を持ち、Product auth prefix は使用されない

### Requirement: session cookie は安全な属性で設定される

Admin refreshToken Cookie は Admin ドメインに対して `HttpOnly; Secure; SameSite=Lax; Path=/` を含む安全属性で設定されなければならない（MUST）。Admin frontend と Admin backend は同一 Admin ドメインでホストされるため、production cookie は第三者 site 用 cookie policy に依存してはならない（MUST NOT）。development 以外で insecure cookie を設定してはならない（MUST NOT）。Admin auth response body は operator accessToken と session metadata を含めてよいが、operator refreshToken 平文を含めてはならない（MUST NOT）。全 Admin auth cookie response は no-store header を含まなければならない（SHALL）。

**Customer Context**

Admin frontend と Admin backend は同一 Admin ドメインで運用されるため、refreshToken Cookie は same-site 前提で安全に送信できる。production では `HttpOnly`、`Secure`、`SameSite=Lax`、`Path=/` を必須にし、ブラウザーから読める refreshToken を排除する。

#### Scenario: Admin refreshToken Cookie は SameSite=Lax と Secure を持つ (ADMIN-AUTH-BE-S064)

- **GIVEN** Admin frontend と Admin backend が同一 Admin ドメインで配信されている
- **WHEN** Admin login が成功する
- **THEN** `Set-Cookie` は `HttpOnly; Secure; SameSite=Lax; Path=/` を含む
- **AND** response body は refreshToken 平文を含まない

#### Scenario: insecure production cookie は拒否される (ADMIN-AUTH-BE-S065)

- **GIVEN** runtime environment が development ではない
- **WHEN** Admin cookie config が `Secure` を無効にしている
- **THEN** Admin backend は fail-close で起動を拒否する

### Requirement: オペレーターは複数の passkey を登録・管理できる

Admin operator passkey management endpoints は Admin API binary の same-origin `/api/v1/auth/passkeys*` surface として提供されなければならない（SHALL）。Passkey management endpoints は operator accessToken、Admin operator session record、Admin CSRF binding を必須とし、Product bearer token または Product account auth state を使用してはならない（MUST NOT）。Passkey registration challenge は Admin Valkey logical DB の `admin:*` prefix に保存され、他 operator の passkey 操作と最後の 1 件の削除は拒否されなければならない（MUST）。`packages/admin` は passkey management BFF route を所有してはならず（MUST NOT）、静的 frontend から Admin backend `/api/v1/*` を呼び出さなければならない（SHALL）。

**Customer Context**

オペレーターは複数 device で Admin Console にアクセスする。passkey 管理は強権限の operator session と CSRF に束縛し、Product account auth や frontend BFF に混ぜないことで、鍵管理の安全性と監査可能性を保つ。

#### Scenario: 登録済み passkey 一覧を Admin backend から取得できる (ADMIN-AUTH-BE-S067)

- **GIVEN** operator が有効な operator accessToken、Admin session record、CSRF binding を持つ
- **WHEN** same-origin `GET /api/v1/auth/passkeys` を呼び出す
- **THEN** Admin backend は operator 自身の passkey credential 一覧を返す
- **AND** Product auth domain/application と package-local BFF route は使用されない

#### Scenario: 最後の passkey 削除は Admin operator auth domain で拒否される (ADMIN-AUTH-BE-S068)

- **GIVEN** operator が passkey credential を 1 件のみ持つ
- **WHEN** same-origin `DELETE /api/v1/auth/passkeys/{id}` を呼び出す
- **THEN** Admin operator auth domain/application は削除を拒否し、credential は保持される

### Requirement: 初回起動セットアップは最初の admin オペレーターを作成する

Admin initial setup endpoints は Admin API binary の same-origin `/api/v1/auth/setup/*` surface として提供されなければならない（SHALL）。Setup flow は operator が 0 件であり、bootstrap enable flag、bootstrap secret hash、有効期限がすべて valid な場合だけ challenge を発行しなければならない（MUST）。Bootstrap secret 平文は DB、audit、log、trace、response body に保存または出力してはならない（MUST NOT）。Setup finish は Admin Operator domain object、Admin OperatorAuth domain object、Admin Valkey `admin:*` challenge、Admin schema transaction を使い、role=`admin` の最初の operator と passkey credential を作成しなければならない（MUST）。`packages/admin` は setup BFF route または server-side setup handler を所有してはならない（MUST NOT）。

**Customer Context**

初回 admin 作成は Admin surface の最初の trust anchor である。静的 frontend package や DB seed に secret logic を置かず、Admin backend の domain/application と fail-close config に集約する必要がある。

#### Scenario: オペレーター 0 件時に Admin backend が最初の admin を作成する (ADMIN-AUTH-BE-S069)

- **GIVEN** operator が 0 件で、bootstrap config が valid である
- **WHEN** `/api/v1/auth/setup/start` と `/api/v1/auth/setup/finish` を完了する
- **THEN** role=`admin` の operator と passkey credential が Admin schema に作成される
- **AND** operator accessToken response と Admin refreshToken Cookie が発行される

#### Scenario: bootstrap secret 平文は観測可能な出力に残らない (ADMIN-AUTH-BE-S070)

- **WHEN** setup start / finish が成功または失敗する
- **THEN** bootstrap secret 平文は DB、audit、log、trace、response body、error message に存在しない

### Requirement: 追加オペレーターは setup token で初回 passkey を登録する

Additional operator setup endpoints は Admin API binary の same-origin `/api/v1/auth/operator-setup/*` surface として提供されなければならない（SHALL）。Admin による operator 作成は setup token hash と expiry を Admin schema に保存し、setup token 平文を DB、audit、log、trace、response body に保存または出力してはならない（MUST NOT）。Operator setup start/finish は Admin backend が token hash、expiry、one-time consumption、WebAuthn challenge、passkey registration、Admin operator session 発行を検証し、`packages/admin` は setup BFF route を所有してはならない（MUST NOT）。

**Customer Context**

追加された operator は passkey credential を持たないため、one-time setup token で初回登録を完了する。token secret と registration logic を静的 frontend に置かず、Admin backend に閉じることで漏洩と列挙を防ぐ。

#### Scenario: 追加 operator は setup token で初回 passkey を登録できる (ADMIN-AUTH-BE-S071)

- **GIVEN** operator が valid setup token hash と expiry を持ち、passkey credential を持たない
- **WHEN** `/api/v1/auth/operator-setup/start` と `/api/v1/auth/operator-setup/finish` を完了する
- **THEN** passkey credential が登録され、setup token hash/expiry は消費される
- **AND** operator accessToken response と Admin refreshToken Cookie が発行される

#### Scenario: setup token error は non-revealing error になる (ADMIN-AUTH-BE-S072)

- **GIVEN** setup token が invalid、expired、consumed、または登録済み operator に属する
- **WHEN** operator setup start を呼び出す
- **THEN** Admin backend は token 状態を区別できない stable error を返し、challenge を発行しない

### Requirement: passkey 保存テーブルは credential 情報を完全に保持する

Admin operator passkey credential は Admin-owned schema に保存されなければならない（SHALL）。保存される credential は id、operator_id、credential_handle、public_key、sign_count、aaguid、backup_eligible、backup_state、transports、created_at を保持し、Admin operator auth domain/application が assertion 検証と sign_count 更新に使用しなければならない（MUST）。`sign_count` が保存値より減少する assertion は replay attack として拒否されなければならない（MUST）。同一 `credential_handle` の重複登録は拒否され、Product account passkey credential と Admin operator passkey credential は persistence namespace と domain type を共有してはならない（MUST NOT）。

**Customer Context**

Admin operator passkey は強権限 session の入口である。credential 情報が不完全、または Product account passkey と混在すると、assertion 検証、replay attack 検出、operator 所有境界が崩れる。

#### Scenario: Admin operator credential が保存され検証に使われる (ADMIN-AUTH-BE-S075)

- **GIVEN** operator が passkey credential を登録済みである
- **WHEN** Admin login finish が assertion を検証する
- **THEN** Admin operator passkey repository は保存済み public_key と sign_count を返し、Admin OperatorAuth domain/application が検証に使う

#### Scenario: sign_count 減少は replay attack として拒否される (ADMIN-AUTH-BE-S076)

- **GIVEN** Admin operator passkey credential の保存済み `sign_count` が 10 である
- **WHEN** assertion の `sign_count` が 8 である
- **THEN** Admin backend は session を発行せず、stable auth error を返す

#### Scenario: 重複 credential_handle の Admin operator 登録は拒否される (ADMIN-AUTH-BE-S077)

- **GIVEN** `credential_handle` X が既に Admin operator passkey として保存されている
- **WHEN** 別 operator が同じ `credential_handle` X で passkey を登録する
- **THEN** Admin backend は 409 を返し、credential は追加されない

## ADDED Requirements

### Requirement: Admin API response は browser security headers を持つ

Admin backend の `/api/v1/*` response と Admin static frontend の HTML response は、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy`、`X-Content-Type-Options`、frame embedding prevention を含む security header または deployment-equivalent controls を SHALL 持つ。Admin auth、account management、audit response は no-store semantics を保たなければならない（MUST）。

**Customer Context**

Admin surface は顧客 PII、operator session、監査情報を扱う。XSS、clickjacking、MIME sniffing、Referer leakage、stale cache を抑止する browser hardening が必要である。

#### Scenario: Admin API response は security headers を含む (ADMIN-AUTH-BE-S066)

- **GIVEN** Admin backend が `/api/v1/auth/current` response を返す
- **WHEN** response headers を確認する
- **THEN** no-store と browser security header semantics が含まれる
