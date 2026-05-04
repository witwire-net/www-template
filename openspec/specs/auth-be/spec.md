## Purpose

Auth core backend requirements, covering bearer-compatible passkey authentication, recovery lifecycle, shared register selector boundaries, no-store auth responses, Valkey-backed auth state, SES-backed recovery delivery, and ULID identifier policy.

## Requirements

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

パスキー認証は、`Authorization: Bearer <session token>` で `/api/v1/*` を利用できる application session を SHALL 発行し、logout で MUST revoke しなければならない。

**Customer Context**

`/api/v1/*`（`/api/v1/auth/*` 除く）は認証必須面であり、企画書と repository rule は bearer 境界として扱います。パスキー認証が session 発行・失効・app surface の認可と一体で定義されていないと、認証面全体の境界が不安定になります。

**Requirement**

- The system SHALL issue a bearer-compatible application session on successful passkey authentication and MUST revoke that session through `POST /api/v1/auth/logout` on logout.
- account、passkey credential、session、revocation marker、recovery token、`recovery_session`、`invitation_session`、および auth 実行を相互参照する system-owned resource ID は ULID を SHALL 使用し、UUID その他の別方式を新規採用してはならない。
- The system SHALL expose `POST /api/v1/auth/passkey/start` to issue a WebAuthn challenge, and `POST /api/v1/auth/passkey/finish` SHALL verify the credential and create an active session for the authenticated account.
- WebAuthn challenge、active session、revoked session marker、temporary auth throttle / lock state は、node 間で共有できる Valkey-backed auth state store に TTL 付きで MUST 保持される。
- WebAuthn challenge record、session record、revocation record、failure counter record が保持する subject ID / actor ID / correlation ID / notification ID / job ID などの識別子が必要な箇所は、ULID を SHALL 用いる。
- `POST /api/v1/auth/passkey/finish` と recovery branch の `POST /api/v1/auth/passkey/register` は、`Authorization: Bearer <session token>` で `/api/v1/*` に提示できる同一の session 契約を SHALL 返す。
- `/api/v1/*` surface は active bearer session を MUST 要求し、missing / expired / revoked session を SHALL 拒否する。
- request が bearer session をまったく持たない場合は stable classification `unauthenticated` として SHALL 扱われ、expired / revoked session failure と混同してはならない。
- expired または revoked session は stable classification `session-expired` として SHALL 扱われる。
- auth state store unavailable などの fail-close な auth boundary failure は stable classification `internal-error` として SHALL 扱われる。
- logout flow は `POST /api/v1/auth/logout` で active session を SHALL revoke し、その後の `/api/v1/*` request を認可できないようにする。revoke 判定に必要な state は Valkey-backed auth state store に MUST 反映される。
- auth start / finish / `POST /api/v1/auth/logout` response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。

#### Scenario: パスキーログイン成功時に bearer session を作成する (AUTH-BE-S001)

- **GIVEN** account が passkey authentication を開始している
- **WHEN** account が valid credential で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** system は後続の `/api/v1/*` access を `Authorization: Bearer <session token>` で認可できる active session を返す

#### Scenario: 欠落または inactive な session は拒否される (AUTH-BE-S002)

- **GIVEN** request が `/api/v1/*` を対象にしている
- **WHEN** request が expired または revoked session を持つ
- **THEN** system はその request を `session-expired` failure として拒否する

#### Scenario: logout は active session を revoke する (AUTH-BE-S003)

- **GIVEN** account が active session を持っている
- **WHEN** logout action がその session を revoke する
- **THEN** revoked session は `/api/v1/*` access を以後認可しない

#### Scenario: session を持たない request は session-expired と混同されない (AUTH-BE-S009)

- **GIVEN** request が `/api/v1/*` を対象にしている
- **WHEN** request が bearer session をまったく提示しない
- **THEN** system は unauthenticated failure として拒否し、expired / revoked session 用の failure と区別する

#### Scenario: auth state store unavailable は fail-close で internal-error になる (AUTH-BE-S010)

- **GIVEN** request が auth boundary を通って `/api/v1/*` または auth endpoint を呼んでいる
- **WHEN** Valkey を含む auth state store が unavailable で session / challenge / recovery state を安全に検証できない
- **THEN** system は fail-close で request を拒否し、stable classification `internal-error` を返す

### Requirement: recovery token は単回利用・期限付きで enumeration-safe に扱う

recovery token は、単回利用・期限付き・enumeration-safe な復旧 credential として SHALL 扱われなければならない。

**Customer Context**

パスキー紛失時の復旧は登録済みメールアドレスだけで成立する必要がありますが、同時に recovery 導線はアカウント有無や token 状態を推測できないように保護されなければなりません。短命 token の保管、受理応答、temporary lock が曖昧だと、Auth コア全体の安全性が下がります。

**Requirement**

- The system SHALL treat recovery tokens as single-use time-limited credentials and MUST keep recovery request responses enumeration-safe.
- The system SHALL expose `POST /api/v1/auth/recovery` to accept a registered email address, issue a single-use time-limited RecoveryToken, and send a recovery URL to the registered address through SES.
- RecoveryToken と `recovery_session` は Valkey-backed auth state store に MUST 保持される。
- RecoveryToken 自体の resource ID、`recovery_session` の resource ID、delivery request ID、mail/audit correlation ID など recovery flow を追跡する識別子が必要な箇所は ULID を SHALL 使用する。
- `POST /api/v1/auth/recovery` は account 有無や throttle 状態を外部から判別できない accepted response を SHALL 返す。
- The system SHALL expose `POST /api/v1/auth/recovery/consume` to validate a RecoveryToken, mark the token consumed atomically, and create a passkey re-registration `recovery_session`.
- RecoveryToken secret は平文 lookup 値として保存してはならず、server-side keyed hash または同等の one-way representation で MUST 保存・照合する。
- RecoveryToken consume と `recovery_session` consume は atomic operation として MUST 実行され、同じ token または session から複数の有効結果を作成してはならない。
- 無効、期限切れ、revoke 済み、または consumed 済みの RecoveryToken から recovery session を作成してはならない（MUST NOT）。
- recovery request / consume response は `Cache-Control: no-store` を SHALL 保ち、temporary lock / throttle の state は Valkey-backed auth state store に MUST 保持される。
- RecoveryToken を含む URL は response body、log、trace attribute、error message に MUST 出力されない。

#### Scenario: 復旧依頼は token を発行して受理される (AUTH-BE-S004)

- **GIVEN** 利用者が passkey recovery を依頼する
- **WHEN** 利用者が `POST /api/v1/auth/recovery` を送信する
- **THEN** system は accepted response を返し、対象アカウントが存在するときだけ time-limited な RecoveryToken を発行して recovery URL をメール送信する

#### Scenario: 有効な復旧 token は recovery session を作成する (AUTH-BE-S005)

- **GIVEN** recovery URL が valid な RecoveryToken を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** token は consumed となり、system は Valkey-backed auth state store 上の passkey 再登録用 `recovery_session` を返す

#### Scenario: 無効な復旧 token は拒否される (AUTH-BE-S006)

- **GIVEN** recovery URL が invalid、expired、または consumed 済みの RecoveryToken を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** system は request を拒否し、recovery session を作成しない

#### Scenario: recovery token の並行 consume は単一結果だけを許可する (AUTH-BE-S030)

- **GIVEN** 同じ valid RecoveryToken に対する複数の consume request が並行している
- **WHEN** system が token consume を処理する
- **THEN** ちょうど 1 つの request だけが recovery session を作成し、残りは generic failure として拒否される

### Requirement: auth throttle と temporary lock は non-revealing に強制する

auth throttle と temporary lock は、abuse を抑止しつつ account existence や branch state を外部へ漏らさない guardrail として SHALL 強制されなければならない。

**Customer Context**

Phase 3 の runtime decision は `passkey/start` throttle、recovery request throttle、finish / consume / register 失敗時の temporary lock を archive-ready 必須要件として定義しています。これらが dedicated requirement / scenario を持たないと、temporary lock や throttle が実装で弱まり、enumeration-safe な recovery と shared register seam の安全性が保証できません。

**Requirement**

- The system SHALL enforce the documented auth throttle and temporary lock policies and MUST keep guarded responses non-revealing while those policies are active.
- `POST /api/v1/auth/passkey/start` は IP bucket と global bucket の rate limit を MUST 適用し、identifier の変化だけで challenge issuance budget を回避できてはならない。
- `POST /api/v1/auth/passkey/start` は configured budget を超える request に追加 WebAuthn challenge を発行してはならない。
- Public WebAuthn challenge state は all nodes で共有できる Valkey-backed auth state store に TTL 付きで保持され、in-memory only の unbounded pending state として保持されてはならない。
- `POST /api/v1/auth/recovery` は email ごとに 3 回 / 1 時間、IP ごとに 10 回 / 1 時間の throttle を MUST 適用し、throttle 中でも generic accepted response と `Cache-Control: no-store` を SHALL 維持する。
- `POST /api/v1/auth/passkey/add/start` と `POST /api/v1/auth/passkey/add/finish` は email、IP、email+IP、OTP handoff、account、global の configured buckets に rate limit と temporary lock を MUST 適用する。
- `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/recovery/consume`、recovery branch の `POST /api/v1/auth/passkey/register`、および device login handoff の add/start・add/finish に対する失敗は共有 failure counter を MUST 加算し、configured threshold に達した主体を temporary lock しなければならない。
- throttle counter と temporary lock state は Valkey-backed auth state store に MUST 保持され、all nodes で共有される。
- throttle counter record、temporary lock record、auth abuse event record、解除ジョブ参照 ID など guardrail state が持つ ID は ULID を SHALL 使用する。ただし email / IP 由来の bucket key 自体は resource ID ではないため ULID 変換対象に含めない。
- temporary lock 中の guarded request は no-store boundary を保ったまま reject され、account existence、invite-only state、recovery-only state の有無を外部へ漏らしてはならない。
- throttle / temporary lock reject は `unauthenticated` / `session-expired` / `internal-error` に新しい公開 stable error code を追加してはならず、non-revealing auth reject として扱わなければならない。

#### Scenario: throttled recovery request は generic accepted response を維持する (AUTH-BE-S011)

- **GIVEN** client が configured budget を超えて `POST /api/v1/auth/recovery` を繰り返している
- **WHEN** system が recovery throttle を適用する
- **THEN** system は同一の accepted / no-store response shape を維持し、登録済み account の有無や throttle hit を外部へ露出しない

#### Scenario: throttled passkey start は no-store かつ non-revealing に reject される (AUTH-BE-S013)

- **GIVEN** client が configured budget 超過まで `POST /api/v1/auth/passkey/start` を繰り返している
- **WHEN** system が public passkey start throttle を適用する
- **THEN** system は追加 challenge を発行せず no-store boundary を保ったまま request を reject し、新しい公開 stable error code や account state を露出しない

#### Scenario: repeated auth failures は temporary lock を発動する (AUTH-BE-S012)

- **GIVEN** public auth completion、recovery consume、recovery registration、または device login handoff に対する失敗が configured window 内で累積している
- **WHEN** client が temporary lock 期間中に guarded endpoint を再試行する
- **THEN** system は temporary lock として request を reject し、challenge completion、recovery consume、passkey re-registration、device login enablement を進めない

#### Scenario: identifier rotation では passkey start budget を回避できない (AUTH-BE-S031)

- **GIVEN** client が異なる identifier 値で `POST /api/v1/auth/passkey/start` を繰り返している
- **WHEN** IP または global challenge issuance budget が上限に達している
- **THEN** system は追加 challenge を発行せず non-revealing に reject する

#### Scenario: OTP brute force は account takeover 前に lock される (AUTH-BE-S032)

- **GIVEN** client が同じ email または同じ IP から複数の OTP 値を試行している
- **WHEN** configured handoff verification budget を超過する
- **THEN** system は device login handoff を temporary lock し、valid OTP の有無を露出しない

### Requirement: recovery register branch は既存アカウントの再登録だけを許可する

recovery register branch は、valid な `recovery_session` が指す既存アカウントの passkey 再登録だけを SHALL 許可し、invite や consent state を MUST 受け入れてはならない。

**Customer Context**

Auth コアが扱うのは既存アカウントの passkey 回復であり、招待登録や規約同意や Guest state 変更ではありません。この境界が崩れると `/login/recovery/*` と `/invite/*` が混線し、後続フェーズの責務が不明確になります。

**Requirement**

- The system SHALL allow the recovery register branch to operate only on an existing account referenced by a valid `recovery_session`, and the shared register endpoint MUST keep an exactly-one selector boundary between recovery and invite state.
- `POST /api/v1/auth/passkey/register` は shared endpoint として `recovery_session` または `invitation_session` の exactly-one を MUST 要求し、recovery branch は valid な `recovery_session` のみを持つときだけ SHALL 成立する。
- recovery branch は `recovery_session` が指す既存アカウントへ新しい passkey を SHALL 登録し、new Account 作成、Guest / Member state 変更、base role 変更をしてはならない。
- recovery branch は invitation-session validation、invite-token consume、invite consent completion、TermsConsent read / write を MUST NOT 要求しない。
- `recovery_session` と `invitation_session` を同時に提示する request、または両方を欠く request は branch ambiguity として MUST reject しなければならない。
- recovery branch が成功した後は、新しい active bearer session を SHALL 返す。
- `POST /api/v1/auth/passkey/register` の response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。
- consumed 済み recovery session を再利用してはならない（MUST NOT）。消費済み state は Valkey-backed auth state store に反映される。
- recovery branch が参照・生成する account ID、passkey credential ID、session ID、`recovery_session` ID、関連 audit / notification / event ID は ULID を SHALL 使用する。
- `/invite/*` onboarding flow は recovery branch から暗黙に起動されてはならない。

#### Scenario: recovery session は既存アカウントへ passkey を再登録する (AUTH-BE-S007)

- **GIVEN** 利用者が valid な recovery session を保持している
- **WHEN** 利用者が recovery branch として `POST /api/v1/auth/passkey/register` を送信する
- **THEN** system は既存アカウントへ新しい passkey を登録し、TermsConsent や role state を変更せずに active bearer session を返す

#### Scenario: invite-only state では recovery registration を完了できない (AUTH-BE-S008)

- **GIVEN** request が valid な recovery session なしで recovery registration を試みる
- **WHEN** request が invite 向け state のみ、TermsConsent のみ、または利用可能な recovery state を持たない
- **THEN** system は recovery registration を拒否し、`/login/recovery/*` と `/invite/*` の分離を維持する

### Requirement: 認証済みアカウントは複数のパスキーを登録・管理できる

**Customer Context**

利用者は MacBook・iPhone・セキュリティキーなど複数のデバイスでパスキーを使い分けたい。新しいデバイスを登録しても他のデバイスのアクセスが失われないことが求められる。複数のパスキーを独立して管理できることで、デバイス追加・紛失後の安全な鍵ローテーションが可能になる。

**Requirement**

- 認証済みアカウントは 1 件以上の passkey credential を持つことができ、システムはすべての active な passkey credential を保持しなければならない（SHALL）。
- システムは `GET /api/v1/passkeys` で認証済みアカウントの登録済みパスキー一覧（ID・識別子・登録日時）を返さなければならない（SHALL）。
- システムは `POST /api/v1/passkeys/start` で WebAuthn 追加登録チャレンジを発行し、`POST /api/v1/passkeys/finish` でチャレンジを検証して既存パスキーを保持したまま新しい passkey credential をアカウントへ追加しなければならない（SHALL）。
- システムは `DELETE /api/v1/passkeys/{id}` で指定した passkey credential を削除しなければならない（SHALL）。ただし、アカウントに残る passkey credential が 1 件になる場合は削除を拒否しなければならない（MUST）。
- 上記すべての管理エンドポイントは `Authorization: Bearer <session token>` を必須とし、未認証リクエストは SHALL 拒否されなければならない。
- 他のアカウントに属する passkey credential を操作する試みは SHALL 拒否されなければならない。
- パスキー管理操作で用いる resource ID（credential ID、challenge ID、correlation ID 等）は ULID を使用しなければならない（SHALL）。

#### Scenario: 登録済みパスキー一覧を取得できる (AUTH-BE-S014)

- **GIVEN** 認証済みアカウントが bearer session を持っている
- **WHEN** `GET /api/v1/passkeys` を呼び出す
- **THEN** システムはそのアカウントに紐づくすべての passkey credential のリストを返す

#### Scenario: 新しいパスキーを追加しても既存パスキーが保持される (AUTH-BE-S015)

- **GIVEN** 認証済みアカウントが bearer session を持っている
- **WHEN** `POST /api/v1/passkeys/start` でチャレンジを取得し `POST /api/v1/passkeys/finish` で完了する
- **THEN** 新しい passkey credential がアカウントへ追加され、それ以前に登録されていたパスキーは削除されない

#### Scenario: 最後の 1 件のパスキーは削除できない (AUTH-BE-S016)

- **GIVEN** 認証済みアカウントが passkey credential を 1 件だけ保持している
- **WHEN** `DELETE /api/v1/passkeys/{id}` でその 1 件を削除しようとする
- **THEN** システムはリクエストを拒否し、アカウントのパスキーは変化しない

#### Scenario: 複数あるパスキーの 1 件を削除できる (AUTH-BE-S017)

- **GIVEN** 認証済みアカウントが 2 件以上の passkey credential を保持している
- **WHEN** `DELETE /api/v1/passkeys/{id}` で特定の 1 件を指定する
- **THEN** 指定された passkey credential のみが削除され、残りは保持される

#### Scenario: 他のアカウントのパスキーは操作できない (AUTH-BE-S018)

- **GIVEN** アカウント A が bearer session を持っている
- **WHEN** アカウント B に属する passkey credential の ID を指定して `DELETE /api/v1/passkeys/{id}` を呼び出す
- **THEN** システムはリクエストを拒否し、アカウント A のパスキーは変化しない

#### Scenario: 未認証リクエストはパスキー管理 API を利用できない (AUTH-BE-S019)

- **GIVEN** bearer session を持たないリクエストがある
- **WHEN** `/api/v1/passkeys` 以下のいずれかのエンドポイントを呼び出す
- **THEN** システムは unauthenticated として拒否する

### Requirement: 認証 runtime は production-safe な境界で fail-close する

**Customer Context**

認証基盤はテンプレート利用者がそのまま本番環境へ持ち込む可能性があるため、危険な origin、曖昧な proxy 境界、過大な request body、弱い配送経路、漏えいしやすい token/trace を安全な初期値で拒否できる必要がある。運用者が設定を誤っても、認証境界は fail-open ではなく fail-close で停止しなければならない。

**Requirement**

- システムは認証 traffic を受け付ける前に production 認証設定を検証し、必須の security 設定が欠けている、または unsafe な場合は MUST fail-close する。
- `APP_ENV!=development` の runtime は `allowed_origins`、WebAuthn RP ID、account recovery URL、trusted proxy configuration、mail transport security configuration を MUST validate する。
- Production allowed origins と account recovery URL は HTTPS origin を MUST 使用し、localhost、loopback、plain HTTP、empty origin、wildcard origin を MUST reject する。
- WebAuthn RP ID は allowed origin host と整合しなければならず、production runtime は RP ID と origin host の不一致を MUST reject する。
- IP-based rate limit / lockout に使う client IP は configured trusted proxy boundary の内側でのみ forwarded headers を信頼し、trusted proxy が未設定または不正な production runtime は MUST fail-close する。
- Public auth endpoints は configured body size limit を MUST enforce し、limit を超える JSON / WebAuthn payload を auth state 変更前に reject する。
- HTTP server は read header timeout に加えて read timeout / write timeout / idle timeout を SHALL enforce する。
- Account recovery mail transport は production で TLS または STARTTLS を MUST require し、証明書検証を無効化してはならない。
- 認証 response および auth route を支える response は、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy`、`X-Content-Type-Options`、frame embedding prevention の security header または deployment-equivalent controls を SHALL 含む。
- 認証失敗、rate limit、handoff audit event は bearer token、OTP 値、recovery token、WebAuthn credential raw ID、secret を含む account recovery URL を log に出してはならない。

#### Scenario: unsafe production auth config fails closed (AUTH-BE-S025)

- **GIVEN** runtime environment が development ではない
- **WHEN** 認証設定に localhost origin、plain HTTP recovery URL、trusted proxy 設定不足、または WebAuthn RP ID 不一致が含まれる
- **THEN** system は auth traffic を受け付ける前に runtime startup を拒否する

#### Scenario: oversized public auth request is rejected before state mutation (AUTH-BE-S026)

- **GIVEN** public auth endpoint が configured auth body limit を超える request body を受け取っている
- **WHEN** request が parse される
- **THEN** system は challenge、OTP state、session、recovery state を発行せず request を拒否する

#### Scenario: recovery mail は production で secure transport を要求する (AUTH-BE-S027)

- **GIVEN** runtime environment が development ではない
- **WHEN** account recovery mail transport が証明書検証付き TLS または STARTTLS を提供できない
- **THEN** system は fail-close するか、account existence を露出せず recovery delivery を拒否する

### Requirement: WebAuthn ceremony は user verification を必須にする

**Customer Context**

Passkey は認証基盤の中核であり、端末所持だけでなく端末内の user verification によって利用者本人の操作であることを確認する必要がある。ログイン、新しい端末でのログイン有効化、復旧後の再登録などの高リスク操作で user verification が optional だと、端末盗難や弱い authenticator policy による不正利用リスクが残る。

**Requirement**

- システムは認証 ceremony と登録 ceremony で WebAuthn user verification を SHALL require する。
- システムは high-risk な認証済み passkey 管理操作の前に、fresh な WebAuthn reauthentication session を SHALL require する。
- `POST /api/v1/auth/passkey/start`、`POST /api/v1/auth/passkey/finish`、`POST /api/v1/passkeys/start`、`POST /api/v1/passkeys/finish`、`POST /api/v1/auth/passkey/register/start`、`POST /api/v1/auth/passkey/register`、`POST /api/v1/auth/passkey/add/start`、`POST /api/v1/auth/passkey/add/finish` は user verification required semantics を MUST enforce する。
- `POST /api/v1/passkeys/otp` と `DELETE /api/v1/passkeys/{id}` は bearer session だけで成立してはならず、`X-Reauth-Session` HTTP header で提示された同一 account に紐づく短命 reauthentication session を MUST require する。
- Reauthentication session は Valkey-backed auth state store に TTL 付きで保持され、対象 account、issuing session、operation kind（`otp-issue` または `passkey-delete`）、request ID を紐づけなければならない。
- Reauthentication session は high-risk operation completion 時に atomic consume されるか、短い有効期限で失効しなければならない。
- 異なる operation kind の reauthentication session を使い回した場合は MUST reject する。
- client に返す WebAuthn options が `userVerification` field を表現する場合、値は `"required"` でなければならない。
- server-side WebAuthn verification は required user verification を満たさない assertion または attestation を拒否しなければならない。

#### Scenario: login ceremony は user verification を要求する (AUTH-BE-S028)

- **GIVEN** passkey login ceremony が開始されている
- **WHEN** authenticator response が required user verification を満たさない
- **THEN** system は login を拒否し、bearer session を発行しない

#### Scenario: 新端末のログイン有効化は user verification を要求する (AUTH-BE-S029)

- **GIVEN** valid な device login handoff が存在する
- **WHEN** 新しい端末が required user verification なしで WebAuthn registration を完了しようとする
- **THEN** system は registration を拒否し、account に credential を追加しない

#### Scenario: OTP delivery は fresh な再認証を要求する (AUTH-BE-S036)

- **GIVEN** account は active bearer session を持つが fresh な reauthentication session を持たない
- **WHEN** account が device login OTP delivery を要求する
- **THEN** system は request を拒否し、OTP を発行または送信しない

#### Scenario: passkey deletion は fresh な再認証を要求する (AUTH-BE-S037)

- **GIVEN** account は active bearer session を持つが fresh な reauthentication session を持たない
- **WHEN** account が登録済み passkey credential の削除を要求する
- **THEN** system は削除を拒否し、すべての credential を変更しない

### Requirement: OTP ハンドオフによる新端末へのパスキー追加

**Customer Context**

利用者が新しい端末でログインを有効にしたい場合、すでにログイン済みの既存端末で本人確認を行い、登録メールアドレスへ届く短いコードを新しい端末で入力できればよい。コードを画面表示だけに依存すると bearer token 接収や画面共有による流出に弱いため、システムはメールボックス到達性、短命 OTP、厳密な rate limit、atomic な単回利用によって account takeover を防がなければならない。

**Requirement**

- システムは `POST /api/v1/passkeys/otp` で device login handoff 用 OTP を発行し、登録メールアドレスへ送信しなければならない（SHALL）。このエンドポイントは bearer session と `X-Reauth-Session` header で提示された operation kind `otp-issue` の reauthentication session を MUST require し、bearer-only では成立してはならない。
- `POST /api/v1/passkeys/otp` の response body は平文 OTP を含めてはならず、`issued: true` の acknowledgement のみを返さなければならない（MUST NOT return raw OTP）。
- OTP は 6 桁の数字とし、有効期限は発行から 5 分以下とする（SHALL）。OTP は Valkey-backed auth state store に保存し、消費またはタイムアウト後は再利用できない（MUST NOT）。
- OTP secret は平文保存してはならず、server-side keyed hash または同等の one-way representation で MUST 保存・照合する。
- OTP state は account、issuing session、normalized email、handoff ID、expiration、attempt counters、challenge binding を含む namespace により、同じ 6 桁 OTP が別アカウントの state を上書きしてはならない。
- OTP は API response body に含めてはならず、画面表示用の平文 OTP を backend から返してはならない（MUST NOT）。
- OTP delivery は secure mail delivery を用い、delivery success/failure は account existence を露出しない no-store response として扱わなければならない。
- システムは `POST /api/v1/auth/passkey/add/start` で**登録メールアドレスと OTP** を受け取り、検証後に WebAuthn 登録チャレンジを発行しなければならない（SHALL）。このエンドポイントは未認証（bearer session 不要）の公開エンドポイントとする。旧の OTP-only request body は廃止する。
- システムは `POST /api/v1/auth/passkey/add/finish` で**登録メールアドレス、OTP、WebAuthn 登録クレデンシャル**を受け取り、OTP が指すアカウントへ新しい passkey credential を追加しなければならない（SHALL）。既存の passkey credential はすべて保持されなければならない（MUST）。旧の OTP-only request body は廃止する。
- `POST /api/v1/auth/passkey/add/start` と `POST /api/v1/auth/passkey/add/finish` は email と OTP の組み合わせだけで account existence、OTP validity、lock state を外部へ露出してはならない。
- OTP、handoff challenge、registration completion は atomic に consume されなければならず、同じ OTP または challenge から複数の credential を追加してはならない（MUST NOT）。
- OTP の検証に失敗した場合、または OTP が有効期限切れ・消費済み・locked の場合はリクエストを generic に拒否しなければならない（SHALL）。
- 新しい credential が追加された後、system は登録済みメールアドレスへ通知を SHALL 送信し、account ID、credential ID、handoff ID、request ID を関連付けた audit event を SHALL 記録する。
- `POST /api/v1/auth/passkey/add/*` で用いる handoff ID、challenge ID、credential ID、request ID、audit event ID は ULID を使用しなければならない（SHALL）。OTP の 6 桁値自体は resource ID ではない。

#### Scenario: OTP を発行できる (AUTH-BE-S021)

- **GIVEN** 認証済みアカウントが bearer session を持ち、既存パスキーで WebAuthn 再認証を完了している
- **WHEN** `POST /api/v1/passkeys/otp` を呼び出す
- **THEN** システムは登録メールアドレスへ 6 桁の OTP を送信し、OTP を response body に含めず、登録メールアドレスに紐づく短命 handoff state を保存する

#### Scenario: OTP を使って新端末にパスキーを追加できる (AUTH-BE-S022)

- **GIVEN** 有効な OTP が発行されている
- **WHEN** 新端末が `POST /api/v1/auth/passkey/add/start` で登録メールアドレスと OTP を提示してチャレンジを取得し、`POST /api/v1/auth/passkey/add/finish` で WebAuthn 登録を完了する
- **THEN** 新しい passkey credential がアカウントへ追加され、既存のパスキーは保持され、OTP と challenge は再利用できない

#### Scenario: 有効期限切れの OTP は拒否される (AUTH-BE-S023)

- **GIVEN** 発行から許容 TTL を超えた OTP がある
- **WHEN** 新端末が `POST /api/v1/auth/passkey/add/start` で登録メールアドレスと OTP を提示する
- **THEN** システムはリクエストを generic に拒否する

#### Scenario: 消費済みの OTP は再利用できない (AUTH-BE-S024)

- **GIVEN** すでに使用された OTP がある
- **WHEN** 同じ登録メールアドレスと OTP で再度 `POST /api/v1/auth/passkey/add/start` または `POST /api/v1/auth/passkey/add/finish` を呼び出す
- **THEN** システムはリクエストを generic に拒否する

#### Scenario: email と OTP の不一致は account existence を露出しない (AUTH-BE-S033)

- **GIVEN** client が登録メールアドレスと OTP の組み合わせを提示している
- **WHEN** email、OTP、またはその組み合わせが有効な handoff state と一致しない
- **THEN** system は同じ response shape で generic に拒否し、email の登録有無や OTP の正否を露出しない

#### Scenario: 同じ OTP 値は別アカウントの handoff state を上書きしない (AUTH-BE-S034)

- **GIVEN** 複数アカウントが同じ 6 桁 OTP 値を持つ handoff を発行している
- **WHEN** それぞれの登録メールアドレスと OTP が検証される
- **THEN** system は各 handoff state を独立して扱い、別アカウントの credential を追加しない

#### Scenario: handoff completion は concurrent finish requests でも atomic である (AUTH-BE-S035)

- **GIVEN** 同じ登録メールアドレスと OTP に対する複数の finish request が並行している
- **WHEN** system が WebAuthn registration completion を処理する
- **THEN** ちょうど 1 つの request だけが credential を追加し、残りは generic に拒否される

#### Scenario: 消費済みの OTP は再利用できない (AUTH-BE-S024)

- **GIVEN** すでに使用された OTP がある
- **WHEN** 同じ OTP で再度 `POST /api/v1/auth/passkey/add/start` を呼び出す
- **THEN** システムはリクエストを拒否する

## ADDED Requirements

### Requirement: リフレッシュトークンは設定可能な TTL で管理される

システムはリフレッシュトークンの有効期限を設定可能な TTL で管理しなければならない（SHALL）。

**Customer Context**

運用者はセキュリティポリシーやコンプライアンス要件に応じてリフレッシュトークンの有効期限を調整したい。同時に、期限なし運用を選択する柔軟性も必要。短すぎる TTL を誤設定されるとユーザー体験が損なわれるため、運用ミスを防ぐバリデーションが必要。

**Requirement**

- システムは `auth.refresh_token_ttl` 設定値を解釈し、リフレッシュトークンの有効期限ポリシーを決定しなければならない（SHALL）。
- `auth.refresh_token_ttl` が未設定またはゼロ値の場合、リフレッシュトークンは無期限有効としなければならない（MUST）。
- `auth.refresh_token_ttl` が設定されている場合、その値は 24 時間以上でなければならない（MUST）。24 時間未満の場合、システムは fail-close で起動を拒否しなければならない（MUST）。
- 設定された TTL はリフレッシュトークン発行時に Valkey-backed auth state store へ TTL 付きで反映されなければならない（MUST）。

#### Scenario: 未設定のリフレッシュトークン TTL は無期限有効とする (AUTH-BE-S038)

- **GIVEN** `auth.refresh_token_ttl` が未設定またはゼロである
- **WHEN** システムがリフレッシュトークンを発行する
- **THEN** そのトークンは期限切れにならず、明示的な失効または消費まで有効である

#### Scenario: 24 時間以上の TTL は正常に適用される (AUTH-BE-S039)

- **GIVEN** `auth.refresh_token_ttl` が 24 時間以上に設定されている
- **WHEN** システムがリフレッシュトークンを発行する
- **THEN** トークンは設定された期間後に自動失効し、かつシステムは正常に起動する

#### Scenario: 24 時間未満の TTL は起動を拒否する (AUTH-BE-S040)

- **GIVEN** `auth.refresh_token_ttl` が 24 時間未満に設定されている
- **WHEN** システムが起動時に認証設定を検証する
- **THEN** システムは fail-close で起動を拒否し、security misconfiguration として報告する

### Requirement: システムは複数の active session を同時に保持・管理できる

システムは同一デバイス上で複数の独立した認証セッションを同時に保持・管理できなければならない（SHALL）。

**Customer Context**

同一の利用者が複数アカウントを所有・操作する場合、各アカウントへの独立したログイン状態を同時に維持したい。ログアウトは操作対象のアカウントのみに影響し、他のアカウントのセッションは維持されなければならない。

**Requirement**

- システムは同一デバイス上で複数の独立した認証セッションを同時に保持できなければならない（SHALL）。各セッションは一意の session ID と紐づく。
- 各セッションは独立したアクセストークンとリフレッシュトークンのペアを持ち、一方のセッションの失効または消費が他方のセッションに影響してはならない（MUST NOT）。
- `POST /api/v1/auth/logout` はリクエストで提示されたアクセストークンに紐づく単一セッションだけを失効させなければならない（MUST）。他の active セッションは継続して有効でなければならない（MUST）。
- セッション一覧の取得や管理エンドポイントは、認証済みアカウントが所有するセッションに対してのみアクセスを許可しなければならない（MUST）。
- セッション ID、アカウント ID、デバイス指紋、関連する audit / event ID は ULID を使用しなければならない（SHALL）。

#### Scenario: 複数アカウントが独立したセッションを保持する (AUTH-BE-S041)

- **GIVEN** 利用者がアカウント A とアカウント B に対して別々にログインしている
- **WHEN** 両方のアクセストークンが有効である間
- **THEN** 各アカウントの保護されたエンドポイントへのアクセスは独立して認可される

#### Scenario: 単一セッションのログアウトは他のセッションに影響しない (AUTH-BE-S042)

- **GIVEN** 利用者がアカウント A とアカウント B の両方で active セッションを持っている
- **WHEN** アカウント A のセッションで `POST /api/v1/auth/logout` を実行する
- **THEN** アカウント A のセッションは失効し、アカウント B のセッションは引き続き有効である

## MODIFIED Requirements

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

パスキー認証は、`Authorization: Bearer <access token>` で `/api/v1/*` を利用できる application session を SHALL 発行し、logout で MUST revoke しなければならない。

**Customer Context**

`/api/v1/*`（`/api/v1/auth/*` 除く）は認証必須面であり、企画書と repository rule は bearer 境界として扱います。パスキー認証が session 発行・失効・app surface の認可と一体で定義されていないと、認証面全体の境界が不安定になります。短命な JWT アクセストークンと長寿命なリフレッシュトークンの導入により、セキュリティと使い勝手の両立が必要です。

**Requirement**

- The system SHALL issue a bearer-compatible application session on successful passkey authentication. The session consists of a short-lived JWT access token and a long-lived refresh token.
- The access token SHALL be a JWT containing minimal claims: `accountID`, `sessionID`, `iat`, and `exp`. The access token lifetime SHALL be approximately 15 minutes.
- The backend SHALL validate the JWT signature and expiration on every protected request to `/api/v1/*`.
- The system SHALL issue a refresh token bound to the account and a device/session fingerprint. The refresh token SHALL be stored in the Valkey-backed auth state store.
- The system SHALL expose `POST /api/v1/auth/refresh` to accept a valid refresh token and return a new access token and a new refresh token (rotation). The consumed refresh token SHALL be atomically invalidated via GETDEL or equivalent atomic consume.
- A refresh token SHALL be single-use: once consumed for rotation, the old refresh token MUST be rejected on subsequent attempts.
- If an already-consumed or unknown refresh token is presented to `POST /api/v1/auth/refresh`, the system SHALL treat it as a potential token theft and MUST revoke all refresh tokens associated with the same account and device/session fingerprint (fail-close rotation failure).
- `POST /api/v1/auth/passkey/start` と `POST /api/v1/auth/passkey/finish` は変更なし。
- `POST /api/v1/auth/passkey/finish` と recovery branch の `POST /api/v1/auth/passkey/register` は、`Authorization: Bearer <access token>` で `/api/v1/*` に提示できる同一の session 契約を SHALL 返す。返却ペイロードにはアクセストークンとリフレッシュトークンの両方を含む。
- `/api/v1/*` surface は active bearer session を MUST 要求し、missing / expired / revoked session を SHALL 拒否する。
- request が bearer session をまったく持たない場合は stable classification `unauthenticated` として SHALL 扱われ、expired / revoked session failure と混同してはならない。
- expired または revoked session は stable classification `session-expired` として SHALL 扱われる。
- auth state store unavailable などの fail-close な auth boundary failure は stable classification `internal-error` として SHALL 扱われる。
- logout flow は `POST /api/v1/auth/logout` で active session を SHALL revoke し、その後の `/api/v1/*` request を認可できないようにする。revoke 判定に必要な state は Valkey-backed auth state store に MUST 反映される。revoke 対象はアクセストークンに紐づくセッションだけであり、同一アカウントの他セッションは維持される。
- auth start / finish / `POST /api/v1/auth/logout` / `POST /api/v1/auth/refresh` response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。
- account、passkey credential、session、revocation marker、recovery token、`recovery_session`、`invitation_session`、および auth 実行を相互参照する system-owned resource ID は ULID を SHALL 使用し、UUID その他の別方式を新規採用してはならない。

#### Scenario: パスキーログイン成功時に JWT access token と refresh token を作成する (AUTH-BE-S001)

- **GIVEN** account が passkey authentication を開始している
- **WHEN** account が valid credential で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** system は後続の `/api/v1/*` access を `Authorization: Bearer <access token>` で認可できる active session を返す。また、リフレッシュトークンも同時に返す。

#### Scenario: 欠落または inactive な session は拒否される (AUTH-BE-S002)

- **GIVEN** request が `/api/v1/*` を対象にしている
- **WHEN** request が expired または revoked access token を持つ
- **THEN** system はその request を `session-expired` failure として拒否する

#### Scenario: logout は active session を revoke する (AUTH-BE-S003)

- **GIVEN** account が active session を持っている
- **WHEN** logout action がその session を revoke する
- **THEN** revoked session は `/api/v1/*` access を以後認可しない

#### Scenario: session を持たない request は session-expired と混同されない (AUTH-BE-S009)

- **GIVEN** request が `/api/v1/*` を対象にしている
- **WHEN** request が bearer session をまったく提示しない
- **THEN** system は unauthenticated failure として拒否し、expired / revoked session 用の failure と区別する

#### Scenario: auth state store unavailable は fail-close で internal-error になる (AUTH-BE-S010)

- **GIVEN** request が auth boundary を通って `/api/v1/*` または auth endpoint を呼んでいる
- **WHEN** Valkey を含む auth state store が unavailable で session / challenge / recovery state を安全に検証できない
- **THEN** system は fail-close で request を拒否し、stable classification `internal-error` を返す

#### Scenario: 有効なリフレッシュトークンをローテーションできる (AUTH-BE-S043)

- **GIVEN** client が有効なリフレッシュトークンを保持している
- **WHEN** client が `POST /api/v1/auth/refresh` を送信する
- **THEN** system は旧リフレッシュトークンを原子消費し、新しいアクセストークンと新しいリフレッシュトークンを返す

#### Scenario: 消費済みリフレッシュトークンの再利用は拒否され関連トークンを失効する (AUTH-BE-S044)

- **GIVEN** リフレッシュトークンが既に消費されている
- **WHEN** 同じ旧リフレッシュトークンで `POST /api/v1/auth/refresh` を再試行する
- **THEN** system は request を拒否し、同一アカウント・同一デバイス指紋のすべてのリフレッシュトークンを失効させる

#### Scenario: 不正なリフレッシュトークンは拒否される (AUTH-BE-S045)

- **GIVEN** client が存在しないまたは改竄されたリフレッシュトークンを提示している
- **WHEN** client が `POST /api/v1/auth/refresh` を送信する
- **THEN** system は request を拒否し、新しいトークンペアを発行しない

#### Scenario: access token の有効期限切れは session-expired として拒否される (AUTH-BE-S046)

- **GIVEN** client が有効期限切れの JWT access token を保持している
- **WHEN** そのトークンを `Authorization: Bearer` ヘッダーに設定して `/api/v1/*` を呼び出す
- **THEN** system は `session-expired` failure として拒否する

### Requirement: システムはログイン中のセッション（デバイス）を一覧・管理できる

システムは認証済みアカウントが自身の active セッションを一覧表示し、特定セッションまたは他のすべてのセッションを無効化できなければならない（SHALL）。

**Customer Context**

利用者は紛失した端末や不審なログインを検知した場合、リモートで特定デバイスのセッションを無効化したい。さらに、「他のすべてのデバイスをログアウト」することで、自分が現在使っているデバイス以外のセッションを一括で無効化したい。セッションメタデータ（デバイス名、ログイン時刻、最終アクティブ時刻など）により、どのデバイスがどの状態にあるかを視覚的に判断できる必要がある。

**Requirement**

- システムは `GET /api/v1/sessions` を提供し、認証済みアカウントが自身の active セッション一覧を取得できなければならない（SHALL）。一覧には各セッションの sessionID、デバイス名（User-Agent 由来）、ログイン時刻、最終アクティブ時刻、IP ハッシュ、および「現在のセッション」フラグを含む。
- システムは `DELETE /api/v1/sessions/{id}` を提供し、認証済みアカウントが自身の特定セッションを無効化できなければならない（MUST）。無効化対象のセッションが現在のセッションであっても許可するが、無効化後はそのセッションのアクセストークンおよびリフレッシュトークンが即座に拒否されなければならない（MUST）。
- システムは `DELETE /api/v1/sessions/others` を提供し、認証済みアカウントが「現在のセッションを除く自身のすべての active セッション」を一括無効化できなければならない（MUST）。
- セッション無効化時、対象セッションに紐づくリフレッシュトークンおよびアクセストークンメタデータを Valkey から即座に削除しなければならない（MUST）。
- セッション一覧および無効化エンドポイントは、自分のアカウントのセッションに対してのみ操作を許可しなければならない（MUST）。他アカウントのセッションを操作しようとした場合は `403 Forbidden` を返す。
- セッションメタデータの IP はハッシュ化（SHA-256 等）して保存し、生の IP アドレスをそのまま保持してはならない（MUST NOT）。
- セッション一覧 API のレスポンスは `Cache-Control: no-store` を SHALL 保ち、キャッシュ可能にしてはならない。

#### Scenario: ログイン中のセッション一覧を取得できる (AUTH-BE-S047)

- **GIVEN** アカウントが 2 つ以上の active セッションを持っている
- **WHEN** アカウントが `GET /api/v1/sessions` を呼び出す
- **THEN** 自身の active セッション一覧が返却され、各セッションに sessionID、デバイス名、ログイン時刻、最終アクティブ時刻、IP ハッシュ、現在のセッションフラグが含まれる

#### Scenario: 特定のセッションを無効化できる (AUTH-BE-S048)

- **GIVEN** アカウントが複数の active セッションを持っている
- **WHEN** アカウントが `DELETE /api/v1/sessions/{id}` を呼び出す
- **THEN** 対象セッションが失効し、以降そのセッションのアクセストークンおよびリフレッシュトークンが拒否される。他のセッションは維持される

#### Scenario: 他のすべてのセッションを一括無効化できる (AUTH-BE-S049)

- **GIVEN** アカウントが複数の active セッションを持っている
- **WHEN** アカウントが `DELETE /api/v1/sessions/others` を呼び出す
- **THEN** 現在のセッションを除くすべての active セッションが失効し、以降それらのセッショントークンが拒否される。現在のセッションは維持される
