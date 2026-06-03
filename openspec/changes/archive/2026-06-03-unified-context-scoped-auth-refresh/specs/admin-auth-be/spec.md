## RENAMED Requirements

- FROM: `### Requirement: オペレーターパスキー認証は httpOnly cookie session を発行する`
- TO: `### Requirement: オペレーターパスキー認証は credential mode 別 session を発行する`

- FROM: `### Requirement: Admin mutation route は CSRF と Origin を検証する`
- TO: `### Requirement: Admin protected route は Bearer accessToken と Origin 境界を検証する`

## MODIFIED Requirements

### Requirement: オペレーターパスキー認証は credential mode 別 session を発行する

Admin auth endpoints は Admin API binary の same-origin `/api/v1/auth/*` surface として提供されなければならない（SHALL）。Product API binary は Admin auth endpoints を register してはならない（MUST NOT）。Operator passkey login、setup finish、operator-setup finish は Admin hosted service adapter から operator auth concept/application を呼び出し、`credentialMode="cookie"` では short-lived `accessToken`、authContextId、operator/session metadata を response body に返し、Admin refresh credential を `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth/contexts/{authContextId}/refresh` Cookie として設定しなければならない（MUST）。Cookie mode response body は `refreshToken` 平文を含めてはならない（MUST NOT）。`credentialMode="bearer"` では Admin automation client 用の `accessToken` と `refreshToken` を response body に返し、Cookie を設定してはならない（MUST NOT）。Admin `accessToken` の TTL は短命であり、通常の protected Admin request で延長してはならない（MUST NOT）。

**Customer Context**

Admin Console は強権限操作を扱うが、Product Web と同じく複数 operator/session context の切り替えと XSS 被害抑制を両立する必要がある。Admin automation client は Cookie に依存できないため Bearer mode も必要だが、Admin Console は refreshToken を browser-readable state に持たない。

#### Scenario: Product host では Admin login API が到達不能である (ADMIN-AUTH-BE-S056)

- **GIVEN** Product API binary が起動している
- **WHEN** Admin operator auth contract の request を Product host の同一 relative path `/api/v1/auth/passkey/start` に送信する
- **THEN** Product API binary は Admin auth handler、Admin generated binding、Admin Valkey namespace、operator domain を実行しない

#### Scenario: Admin middleware が accessToken を検証して request context に設定する (ADMIN-AUTH-BE-S057)

- **GIVEN** 有効な Admin `accessToken` と Admin Valkey session record が存在する
- **WHEN** protected Admin API を呼び出す
- **THEN** Admin backend middleware は accessToken claims と server-side operator/session record を検証し、現在の operator、role、session ID、authContextId を context に設定する

#### Scenario: Product bearer token は Admin auth session として扱われない (ADMIN-AUTH-BE-S058)

- **GIVEN** request が Product bearer token を持つ
- **WHEN** protected Admin auth API を呼び出す
- **THEN** Admin backend は operator session 不在として拒否する
- **AND** account auth concept/application へ判定を委譲しない

#### Scenario: Admin Cookie mode login は accessToken body と path-scoped refresh Cookie を返す (ADMIN-AUTH-BE-S074)

- **GIVEN** WebAuthn assertion の `userVerification` flag が true で、operator credential が valid である
- **WHEN** Admin backend の `/api/v1/auth/passkey/finish` を `credentialMode="cookie"` で呼び出す
- **THEN** response body は short-lived `accessToken`、authContextId、operator/session metadata を含む
- **AND** response body は `refreshToken` 平文を含まない
- **AND** refresh Cookie は `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth/contexts/{authContextId}/refresh` で設定される

#### Scenario: Admin Bearer mode login は body token を返し Cookie を設定しない (ADMIN-AUTH-BE-S079)

- **GIVEN** Admin automation client が valid operator credential を提示している
- **WHEN** Admin backend の login finish を `credentialMode="bearer"` で完了する
- **THEN** response body は `accessToken`、`refreshToken`、authContextId、operator/session metadata を含む
- **AND** backend は Admin auth Cookie を設定しない

#### Scenario: 通常の protected Admin request は operator access token TTL を延長しない (ADMIN-AUTH-BE-S078)

- **GIVEN** operator が有効な Admin `accessToken` を持っている
- **WHEN** protected Admin `/api/v1/*` endpoint を複数回呼び出す
- **THEN** Admin backend は Admin `accessToken` の issuedAt / expiresAt を延長せず、login/setup/operator-setup/context refresh だけが新しい `accessToken` を発行する

### Requirement: Admin protected route は Bearer accessToken と Origin 境界を検証する

Admin protected routes は `Authorization: Bearer <accessToken>` のみを operator/session credential として使わなければならない（MUST）。Admin protected backend は operator/session/authContext を accessToken claims と server-side operator/session record から束縛し、refresh Cookie、legacy access Cookie、Product Cookie、Product bearer token、`X-Auth-Context-Id` header、CSRF token を operator/session credential として使ってはならない（MUST NOT）。Admin protected mutations は Bearer accessToken で認可されるため CSRF token を要求してはならない（MUST NOT）。Browser Cookie-setting flows と pre-auth WebAuthn/setup flows は allowed Origin、SameSite、CORS、Fetch Metadata、no-store、browser security headers を fail-close で検証しなければならない（MUST）。

**Customer Context**

Admin frontend と Admin backend は同一 Admin ドメインで運用されるが、protected Admin API を ambient Cookie auth に依存させると CSRF と session ambiguity の問題が残る。Admin Console も active Admin `accessToken` を明示的に選んで protected API を呼ぶことで、Product と同じ統一認証方式に揃えられる。

#### Scenario: 許可されていない Origin の Admin Cookie-setting flow は拒否される (ADMIN-AUTH-BE-S059)

- **GIVEN** request の Origin が Admin frontend allowlist に含まれない
- **WHEN** Admin login finish、setup finish、operator-setup finish、または Cookie mode context refresh を呼び出す
- **THEN** Admin backend は Cookie 設定または rotation を実行せず 403 を返す

#### Scenario: Admin protected mutation は CSRF token を要求しない (ADMIN-AUTH-BE-S060)

- **GIVEN** request が有効な `Authorization: Bearer <accessToken>` を持つが CSRF token を持たない
- **WHEN** protected Admin mutation API を呼び出す
- **THEN** Admin backend は CSRF token 欠落を理由に拒否せず、accessToken claims、operator/session record、RBAC を検証してから application use case に進む

#### Scenario: pre-auth passkey start は session-bound CSRF なしで Origin を検証する (ADMIN-AUTH-BE-S061)

- **GIVEN** request が Admin `accessToken` と CSRF token を持たないが、Origin は allowlist に含まれる
- **WHEN** `/api/v1/auth/passkey/start` を呼び出す
- **THEN** Admin backend は session-bound CSRF 不在を理由に拒否せず、Origin、Fetch Metadata、rate limit を検証して処理を継続する

#### Scenario: Admin protected route は Cookie credential を認可材料にしない (ADMIN-AUTH-BE-S080)

- **GIVEN** browser が Admin refresh Cookie または legacy Admin access Cookie だけを送信している
- **WHEN** protected Admin `/api/v1/*` endpoint を呼び出す
- **THEN** Admin backend は `accessToken` 不在として拒否し、Cookie を protected route の認可材料にしない

### Requirement: session cookie は安全な属性で設定される

Admin refreshToken Cookie は Admin ドメインに対して `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth/contexts/{authContextId}/refresh` を含む安全属性で設定されなければならない（MUST）。Admin frontend と Admin backend は同一 Admin ドメインでホストされるため、production cookie は第三者 site 用 cookie policy に依存してはならない（MUST NOT）。development 以外で insecure cookie を設定してはならない（MUST NOT）。Admin Cookie mode auth response body は `accessToken` と session metadata を含めるが、`refreshToken` 平文を含めてはならない（MUST NOT）。全 Admin auth cookie response は no-store header を含まなければならない（SHALL）。Admin context refresh は Product と同じ refresh token family state machine を使い、atomic consume + issue、reuse detection、path ownership validation、Cookie/body ambiguity rejection を満たさなければならない（MUST）。
Admin runtime auth config の型名、field 名、コメント、validation error は Admin/operator/auth concept を表現しなければならず（SHALL）、Product account runtime naming を再利用して Admin operator session の意味を曖昧にしてはならない（MUST NOT）。

**Customer Context**

Admin frontend と Admin backend は同一 Admin ドメインで運用されるため、refreshToken Cookie は same-site 前提で安全に送信できる。production では `HttpOnly`、`Secure`、`SameSite=Lax`、context refresh endpoint Path を必須にし、ブラウザーから読める refreshToken を排除する。

#### Scenario: Admin refreshToken Cookie は SameSite=Lax と Secure と context Path を持つ (ADMIN-AUTH-BE-S064)

- **GIVEN** Admin frontend と Admin backend が同一 Admin ドメインで配信されている
- **WHEN** Admin Cookie mode login または context refresh が成功する
- **THEN** `Set-Cookie` は `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth/contexts/{authContextId}/refresh` を含む
- **AND** response body は refreshToken 平文を含まない

#### Scenario: insecure production cookie は拒否される (ADMIN-AUTH-BE-S065)

- **GIVEN** runtime environment が development ではない
- **WHEN** Admin cookie config が `Secure` を無効にしている
- **THEN** Admin backend は fail-close で起動を拒否する

#### Scenario: Admin Cookie refresh は path-scoped refresh Cookie だけを使う (ADMIN-AUTH-BE-S081)

- **GIVEN** browser が Admin refresh Cookie を保持している
- **WHEN** Admin Console が `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** Admin backend は URL path の authContextId と refresh Cookie 所属を検証して rotation する
- **AND** response body は新しい `accessToken`、authContextId、operator/session metadata を含み、`refreshToken` 平文を含まない

#### Scenario: Admin Bearer refresh は body token を rotation する (ADMIN-AUTH-BE-S084)

- **GIVEN** Admin automation client が body に有効な `refreshToken` を持っている
- **WHEN** client が `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** Admin backend は body `refreshToken` と path authContextId の所属を検証し、response body に新しい `accessToken` と `refreshToken` を返す
- **AND** backend は Admin auth Cookie を設定しない

#### Scenario: Admin Bearer refresh は Authorization header を拒否する (ADMIN-AUTH-BE-S082)

- **GIVEN** Admin API client が body refreshToken と `Authorization: Bearer <accessToken>` header を同時に送信している
- **WHEN** context refresh endpoint を呼び出す
- **THEN** Admin backend は `Authorization` header を refresh credential として扱わず、fail-close で request を拒否する

#### Scenario: Admin refresh Cookie と body refreshToken の同時提示は拒否される (ADMIN-AUTH-BE-S083)

- **GIVEN** request が Admin refresh Cookie と body `refreshToken` の両方を持っている
- **WHEN** context refresh endpoint を呼び出す
- **THEN** Admin backend は refresh credential ambiguity として request を拒否し、新しい credential を発行しない

#### Scenario: Admin refresh reuse または不正 refresh token は family を失効する (ADMIN-AUTH-BE-S085)

- **GIVEN** Admin refreshToken が既に rotation で consumed 済み、unknown、または tampered である
- **WHEN** その refreshToken で context refresh を試行する
- **THEN** Admin backend は request を replay/theft signal として拒否する
- **AND** 同一 operator/session/device fingerprint の refresh family を revoke し、新しい `accessToken` を発行しない

#### Scenario: Admin logout は対象 refresh Cookie path の clear command を返す (ADMIN-AUTH-BE-S086)

- **GIVEN** operator が複数 authContextId の active session を持っている
- **WHEN** operator が active Admin `accessToken` で logout API を呼び出す
- **THEN** Admin backend は accessToken claims が示す operator session の refresh family を revoke する
- **AND** response は対象 `Path=/api/v1/auth/contexts/{authContextId}/refresh` の Admin refresh Cookie を削除する Set-Cookie command を返す
- **AND** 他 authContextId の operator session は削除しない

#### Scenario: Admin runtime auth config uses operator concept names (ADMIN-AUTH-BE-S087)

- **GIVEN** Admin runtime configuration and validation code is inspected
- **WHEN** auth issuer, audience, cookie, token TTL, allowed Origin, session namespace, and operator session comments are reviewed
- **THEN** names and comments describe Admin operator auth concepts explicitly
- **AND** Product account config names are not reused for Admin operator auth behavior
