## MODIFIED Requirements

### Requirement: オペレーターパスキー認証は httpOnly cookie session を発行する

Admin auth endpoints は Admin API binary の same-origin `/api/v1/auth/*` surface として提供されなければならない（SHALL）。Product API binary は Admin auth endpoints を register してはならない（MUST NOT）。Admin auth endpoints は Product user bearer session を operator session として扱ってはならない（MUST NOT）。Operator passkey login は Product account 認証と同じ accessToken / refreshToken 基盤を可能な限り再利用し、operator accessToken と operator refreshToken Cookie を発行しなければならない（SHALL）。Operator token は operator ID / operator session ID / Admin Valkey logical DB / `admin:*` key prefix に束縛され、Product account ID と混同してはならない（MUST NOT）。Operator session validation は Admin backend middleware で実行し、request context に現在の operator、role、session ID、CSRF binding を設定しなければならない（SHALL）。Admin auth response は `Cache-Control: no-store` を含み、session / setup / challenge state を cache 可能にしてはならない（MUST NOT）。

**Customer Context**

Admin auth は operator passkey、bearer accessToken、HttpOnly refreshToken Cookie、CSRF、setup token を扱う強権限境界である。認証 API が Product host や画面配信 package に分散すると、Origin 検証、session 検証、監査、rate limit の責務が分かれ安全性が下がる。

#### Scenario: Product host では Admin login API が到達不能である (ADMIN-AUTH-BE-S056)

- **GIVEN** Product API binary が起動している
- **WHEN** Admin auth operation path である `/api/v1/auth/passkey/start` に request を送信する
- **THEN** Product API binary は Admin auth handler を実行しない

#### Scenario: Admin middleware が operator accessToken を検証して request context に設定する (ADMIN-AUTH-BE-S057)

- **GIVEN** 有効な operator accessToken と Admin Valkey session record が存在する
- **WHEN** protected Admin API を呼び出す
- **THEN** Admin backend middleware は現在の operator、role、session ID、CSRF binding を context に設定する

#### Scenario: Product bearer token は Admin auth session として扱われない (ADMIN-AUTH-BE-S058)

- **GIVEN** request が Product bearer token を持つ
- **WHEN** protected Admin auth API を呼び出す
- **THEN** Admin backend は operator session 不在として拒否する

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

Admin frontend と Admin backend は同一 Admin ドメインで運用されるため、refreshToken Cookie は same-site 前提で安全に送信できる。production では `HttpOnly`、`Secure`、`SameSite=Lax`、`Path=/` を必須にし、browser-readable refreshToken を排除する。

#### Scenario: Admin refreshToken Cookie は SameSite=Lax と Secure を持つ (ADMIN-AUTH-BE-S064)

- **GIVEN** Admin frontend と Admin backend が同一 Admin ドメインで配信されている
- **WHEN** Admin login が成功する
- **THEN** `Set-Cookie` は `HttpOnly; Secure; SameSite=Lax; Path=/` を含む
- **AND** response body は refreshToken 平文を含まない

#### Scenario: insecure production cookie は拒否される (ADMIN-AUTH-BE-S065)

- **GIVEN** runtime environment が development ではない
- **WHEN** Admin cookie config が `Secure` を無効にしている
- **THEN** Admin backend は fail-close で起動を拒否する

## ADDED Requirements

### Requirement: Admin API response は browser security headers を持つ

Admin backend の `/api/v1/*` response と Admin static frontend の HTML response は、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy`、`X-Content-Type-Options`、frame embedding prevention を含む security header または deployment-equivalent controls を SHALL 持つ。Admin auth、account management、audit response は no-store semantics を保たなければならない（MUST）。

**Customer Context**

Admin surface は顧客 PII、operator session、監査情報を扱う。XSS、clickjacking、MIME sniffing、Referer leakage、stale cache を抑止する browser hardening が必要である。

#### Scenario: Admin API response は security headers を含む (ADMIN-AUTH-BE-S066)

- **GIVEN** Admin backend が `/api/v1/auth/current` response を返す
- **WHEN** response headers を確認する
- **THEN** no-store と browser security header semantics が含まれる
