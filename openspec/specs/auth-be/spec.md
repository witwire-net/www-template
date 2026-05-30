## Purpose

Auth core backend の要件を定義し、パスキー認証、復旧ライフサイクル、共有 register selector 境界、no-store 認証応答、Valkey-backed auth state、SES-backed recovery delivery、ULID 識別子方針を扱う。

## Requirements

### Requirement: recovery token は単回利用・期限付きで enumeration-safe に扱う

recovery token は、単回利用・期限付き・enumeration-safe なパスキー追加用 credential として SHALL 扱われなければならない。

**Customer Context**

パスキー紛失時の復旧と新端末追加は、どちらも登録済みメールアドレスへの URL 送信によってパスキー再登録を可能にする必要がある。同時にこれらの導線は、アカウント有無や token 状態を推測できないように保護されなければならない。短命 token の保管、受理応答、temporary lock が曖昧だと、Auth コア全体の安全性が下がる。また、復旧と新端末追加では token 消費後の挙動（セッション失効の有無、通知文面）が異なるため、token 自体に kind を持たせる必要がある。

**Requirement**

- システムは recovery token と device-link token を単回利用・期限付き credential として扱い、発行 request の response は account existence を推測できない形に保たなければならない（SHALL/MUST）。
- Token は `kind` フィールドを持ち、`"recovery"`（パスキー紛失時の復旧）または `"device-link"`（認証済み端末からの新端末追加）のいずれかを MUST 指定する。
- システムは `POST /api/v1/auth/recovery` を提供し、登録済みメールアドレスを受け取って `kind=recovery` の単回利用・期限付き token を発行し、登録済みアドレスへ recovery URL を SMTP で送信しなければならない（SHALL）。
- システムは `POST /api/v1/passkeys/send-device-link` を提供し、`X-Reauth-Session` header（operation kind `device-link`）付きの有効な Product 認証セッションを受け取って `kind=device-link` の単回利用・期限付き token を発行し、登録済みメールアドレスへ device-link URL を SMTP で送信しなければならない（SHALL）。この endpoint は valid な Product 認証セッションと consumed reauthentication session を要求し、session-only request を拒否しなければならない（MUST/SHALL）。
- RecoveryToken と RecoverySession は Valkey-backed auth state store に MUST 保持され、`kind` を含む完全な状態で永続化される。
- RecoveryToken 自体の resource ID、RecoverySession の resource ID、delivery request ID、mail/audit correlation ID など flow を追跡する識別子が必要な箇所は ULID を SHALL 使用する。
- `POST /api/v1/auth/recovery` および `POST /api/v1/passkeys/send-device-link` は account 有無や throttle 状態を外部から判別できない accepted response を SHALL 返す。
- システムは `POST /api/v1/auth/recovery/consume` を提供し、token を検証して原子的に consumed とし、token の `kind` を継承する passkey 再登録用 RecoverySession を作成しなければならない（SHALL）。
- RecoveryToken secret は平文 lookup 値として保存してはならず、server-side keyed hash (HMAC-SHA256 with server-side pepper) で MUST 保存・照合する。
- RecoveryToken consume と RecoverySession consume は atomic operation として MUST 実行され、同じ token または session から複数の有効結果を作成してはならない。
- 無効、期限切れ、revoke 済み、または consumed 済みの token から RecoverySession を作成してはならない（MUST NOT）。
- 発行・消費・登録 response は `Cache-Control: no-store` を SHALL 保ち、temporary lock / throttle の state は Valkey-backed auth state store に MUST 保持される。
- Token を含む URL は response body、log、trace attribute、error message に MUST 出力されない。

#### Scenario: 復旧依頼は kind=recovery の token を発行して受理される (AUTH-BE-S004)

- **GIVEN** 利用者が passkey recovery を依頼する
- **WHEN** 利用者が `POST /api/v1/auth/recovery` を送信する
- **THEN** システムは accepted response を返し、対象アカウントが存在するときだけ `kind=recovery` の time-limited token を発行して recovery URL をメール送信する

#### Scenario: 有効な復旧 token は kind を継承した RecoverySession を作成する (AUTH-BE-S005)

- **GIVEN** recovery URL が valid な token (kind=recovery) を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** token は consumed となり、システムは `kind=recovery` を継承した RecoverySession を返す

#### Scenario: 有効なデバイスリンク token は kind を継承した RecoverySession を作成する (AUTH-BE-S047)

- **GIVEN** device-link URL が valid な token (kind=device-link) を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** token は consumed となり、システムは `kind=device-link` を継承した RecoverySession を返す

#### Scenario: 無効な復旧 token は拒否される (AUTH-BE-S006)

- **GIVEN** recovery URL が invalid、expired、または consumed 済みの token を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** システムは request を拒否し、RecoverySession を作成しない

#### Scenario: token の並行 consume は単一結果だけを許可する (AUTH-BE-S030)

- **GIVEN** 同じ valid token に対する複数の consume request が並行している
- **WHEN** システムが token consume を処理する
- **THEN** ちょうど 1 つの request だけが RecoverySession を作成し、残りは generic failure として拒否される

### Requirement: auth throttle と temporary lock は non-revealing に強制する

auth throttle と temporary lock は、abuse を抑止しつつ account existence や branch state を外部へ漏らさない guardrail として SHALL 強制されなければならない。

**Customer Context**

runtime decision は `passkey/start` throttle、recovery request throttle、send-device-link throttle、finish / consume / register 失敗時の temporary lock を必須要件として定義しています。これらが dedicated requirement / scenario を持たないと、temporary lock や throttle が実装で弱まり、enumeration-safe な recovery/device-link と shared register seam の安全性が保証できません。

**Requirement**

- システムは定義済みの auth throttle と temporary lock policy を強制し、それらが有効な間も guarded response を non-revealing に保たなければならない（SHALL/MUST）。
- `POST /api/v1/auth/passkey/start` は IP bucket と global bucket の rate limit を MUST 適用し、identifier の変化だけで challenge issuance budget を回避できてはならない。
- `POST /api/v1/auth/passkey/start` は configured budget を超える request に追加 WebAuthn challenge を発行してはならない。
- Public WebAuthn challenge state は all nodes で共有できる Valkey-backed auth state store に TTL 付きで保持され、in-memory only の unbounded pending state として保持されてはならない。
- `POST /api/v1/auth/recovery` および `POST /api/v1/passkeys/send-device-link` は email ごとに 3 回 / 1 時間、IP ごとに 10 回 / 1 時間の throttle を MUST 適用し、throttle 中でも generic accepted response と `Cache-Control: no-store` を SHALL 維持する。
- `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/recovery/consume`、recovery/device-link branch の `POST /api/v1/auth/passkey/register` に対する失敗は共有 failure counter を MUST 加算し、configured threshold に達した主体を temporary lock しなければならない。
- throttle counter と temporary lock state は Valkey-backed auth state store に MUST 保持され、all nodes で共有される。
- throttle counter record、temporary lock record、auth abuse event record、解除ジョブ参照 ID など guardrail state が持つ ID は ULID を SHALL 使用する。ただし email / IP 由来の bucket key 自体は resource ID ではないため ULID 変換対象に含めない。
- temporary lock 中の guarded request は no-store boundary を保ったまま reject され、account existence、invite-only state、recovery-only state の有無を外部へ漏らしてはならない。
- throttle / temporary lock reject は `unauthenticated` / `session-expired` / `internal-error` に新しい公開 stable error code を追加してはならず、non-revealing auth reject として扱わなければならない。

#### Scenario: throttled recovery request は generic accepted response を維持する (AUTH-BE-S011)

- **GIVEN** クライアントが configured budget を超えて `POST /api/v1/auth/recovery` を繰り返している
- **WHEN** システムが recovery throttle を適用する
- **THEN** システムは同一の accepted / no-store response shape を維持し、登録済み account の有無や throttle hit を外部へ露出しない

#### Scenario: throttled passkey start は no-store かつ non-revealing に reject される (AUTH-BE-S013)

- **GIVEN** クライアントが configured budget 超過まで `POST /api/v1/auth/passkey/start` を繰り返している
- **WHEN** システムが public passkey start throttle を適用する
- **THEN** システムは追加 challenge を発行せず no-store boundary を保ったまま request を reject し、新しい公開 stable error code や account state を露出しない

#### Scenario: repeated auth failures は temporary lock を発動する (AUTH-BE-S012)

- **GIVEN** public auth completion、recovery/device-link consume、または registration に対する失敗が configured window 内で累積している
- **WHEN** クライアントが temporary lock 期間中に guarded endpoint を再試行する
- **THEN** システムは temporary lock として request を reject し、challenge completion、token consume、passkey registration を進めない

#### Scenario: identifier rotation では passkey start budget を回避できない (AUTH-BE-S031)

- **GIVEN** クライアントが異なる identifier 値で `POST /api/v1/auth/passkey/start` を繰り返している
- **WHEN** IP または global challenge issuance budget が上限に達している
- **THEN** システムは追加 challenge を発行せず non-revealing に reject する

### Requirement: recovery register branch は既存アカウントの再登録だけを許可する

recovery/device-link register branch は、valid な RecoverySession が指す既存アカウントの passkey 登録だけを SHALL 許可し、invite や consent state を MUST 受け入れてはならない。登録完了後はセッションの kind に応じた後処理（セッション失効・通知）を実行する。

**Customer Context**

Auth コアが扱うのは既存アカウントの passkey 回復・追加であり、招待登録や規約同意や Guest state 変更ではありません。この境界が崩れると `/login/recovery/*` と `/invite/*` が混線し、後続フェーズの責務が不明確になります。また、kind=recovery による復旧はアカウント乗っ取りの可能性があるため全既存セッションの強制失効が必要であり、kind=device-link による新端末追加ではその必要がない。

**Requirement**

- システムは register branch を valid な RecoverySession が参照する既存アカウントに対してだけ動作させ、shared register endpoint は recovery state と invite state の exactly-one selector boundary を保たなければならない（SHALL/MUST）。
- `POST /api/v1/auth/passkey/register` は shared endpoint として RecoverySession または InvitationSession の exactly-one を MUST 要求し、register branch は valid な RecoverySession のみを持つときだけ SHALL 成立する。
- register branch は RecoverySession が指す既存アカウントへ新しい passkey を SHALL 登録し、new Account 作成、Guest / Member state 変更、base role 変更をしてはならない。
- register branch は invitation-session validation、invite-token consume、invite consent completion、TermsConsent read / write を MUST NOT 要求しない。
- RecoverySession と InvitationSession を同時に提示する request、または両方を欠く request は branch ambiguity として MUST reject しなければならない。
- register branch が成功した後は、新しい有効なアプリケーションセッションを SHALL 返す。
- RecoverySession の `kind` が `"recovery"` である場合、システムはパスキー登録成功後に該当アカウントの全既存セッションを強制失効しなければならない（SHALL revoke all sessions for the account）。その後、復旧完了を通知するメールを登録済みメールアドレスへ SHALL 送信する。
- RecoverySession の `kind` が `"device-link"` である場合、システムはパスキー登録成功後にセッションを失効してはならない（MUST NOT revoke any session）。新端末追加完了を通知するメールを登録済みメールアドレスへ SHALL 送信する。
- 通知メールの送信失敗はパスキー登録の成功を妨げてはならない（MUST NOT block registration success）。失敗は fire-and-forget でログ記録する。
- `POST /api/v1/auth/passkey/register` の response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。
- consumed 済み RecoverySession を再利用してはならない（MUST NOT）。消費済み state は Valkey-backed auth state store に反映される。
- register branch が参照・生成する account ID、passkey credential ID、session ID、RecoverySession ID、関連 audit / notification / event ID は ULID を SHALL 使用する。
- `/invite/*` onboarding flow は register branch から暗黙に起動されてはならない。

#### Scenario: recovery session (kind=recovery) は既存アカウントへ passkey を再登録し全セッションを失効する (AUTH-BE-S007)

- **GIVEN** 利用者が kind=recovery の valid な RecoverySession を保持している
- **WHEN** 利用者が register branch として `POST /api/v1/auth/passkey/register` を送信する
- **THEN** システムは既存アカウントへ新しい passkey を登録し、該当アカウントの全既存セッションを強制失効し、復旧完了通知メールを送信し、新しい有効なアプリケーションセッションを返す

#### Scenario: device-link session (kind=device-link) は passkey を追加しセッションは失効しない (AUTH-BE-S048)

- **GIVEN** 利用者が kind=device-link の valid な RecoverySession を保持している
- **WHEN** 利用者が register branch として `POST /api/v1/auth/passkey/register` を送信する
- **THEN** システムは既存アカウントへ新しい passkey を登録し、既存セッションを一切失効せず、新端末追加完了通知メールを送信し、新しい有効なアプリケーションセッションを返す

#### Scenario: invite-only state では registration を完了できない (AUTH-BE-S008)

- **GIVEN** request が valid な RecoverySession なしで registration を試みる
- **WHEN** request が invite 向け state のみ、TermsConsent のみ、または利用可能な recovery state を持たない
- **THEN** システムは registration を拒否し、`/login/recovery/*` と `/invite/*` の分離を維持する

### Requirement: 認証済みアカウントは複数のパスキーを登録・管理できる

システムは認証済みアカウントが複数の passkey credential を安全に一覧・追加・削除できる管理 API を提供しなければならない（SHALL）。

**Customer Context**

利用者は MacBook・iPhone・セキュリティキーなど複数のデバイスでパスキーを使い分けたい。新しいデバイスを登録しても他のデバイスのアクセスが失われないことが求められる。複数のパスキーを独立して管理できることで、デバイス追加・紛失後の安全な鍵ローテーションが可能になる。

**Requirement**

- 認証済みアカウントは 1 件以上の passkey credential を持つことができ、システムはすべての active な passkey credential を保持しなければならない（SHALL）。
- システムは `GET /api/v1/passkeys` で認証済みアカウントの登録済みパスキー一覧（ID・識別子・登録日時）を返さなければならない（SHALL）。
- システムは `POST /api/v1/passkeys/start` で WebAuthn 追加登録チャレンジを発行し、`POST /api/v1/passkeys/finish` でチャレンジを検証して既存パスキーを保持したまま新しい passkey credential をアカウントへ追加しなければならない（SHALL）。
- システムは `DELETE /api/v1/passkeys/{id}` で指定した passkey credential を削除しなければならない（SHALL）。ただし、アカウントに残る passkey credential が 1 件になる場合は削除を拒否しなければならない（MUST）。
- 上記すべての管理エンドポイントは Product 認証セッションを必須とし、未認証リクエストは SHALL 拒否されなければならない。
- 他のアカウントに属する passkey credential を操作する試みは SHALL 拒否されなければならない。
- パスキー管理操作で用いる resource ID（credential ID、challenge ID、correlation ID 等）は ULID を使用しなければならない（SHALL）。

#### Scenario: 登録済みパスキー一覧を取得できる (AUTH-BE-S014)

- **GIVEN** 認証済みアカウントが Product 認証セッションを持っている
- **WHEN** `GET /api/v1/passkeys` を呼び出す
- **THEN** システムはそのアカウントに紐づくすべての passkey credential のリストを返す

#### Scenario: 新しいパスキーを追加しても既存パスキーが保持される (AUTH-BE-S015)

- **GIVEN** 認証済みアカウントが Product 認証セッションを持っている
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

- **GIVEN** アカウント A が Product 認証セッションを持っている
- **WHEN** アカウント B に属する passkey credential の ID を指定して `DELETE /api/v1/passkeys/{id}` を呼び出す
- **THEN** システムはリクエストを拒否し、アカウント A のパスキーは変化しない

#### Scenario: 未認証リクエストはパスキー管理 API を利用できない (AUTH-BE-S019)

- **GIVEN** Product 認証セッションを持たないリクエストがある
- **WHEN** `/api/v1/passkeys` 以下のいずれかのエンドポイントを呼び出す
- **THEN** システムは unauthenticated として拒否する

### Requirement: 認証 runtime は production-safe な境界で fail-close する

システムは production 認証境界で危険な設定・入力・配送経路を fail-close し、unsafe な runtime startup または auth state mutation を拒否しなければならない（MUST）。

**Customer Context**

認証基盤はテンプレート利用者がそのまま本番環境へ持ち込む可能性があるため、危険な origin、曖昧な proxy 境界、過大な request body、弱い配送経路、漏えいしやすい token/trace を安全な初期値で拒否できる必要がある。運用者が設定を誤っても、認証境界は fail-open ではなく fail-close で停止しなければならない。

**Requirement**

- システムは認証 traffic を受け付ける前に production 認証設定を検証し、必須の security 設定が欠けている、または unsafe な場合は MUST fail-close する。
- `APP_ENV!=development` の runtime は `allowed_origins`、WebAuthn RP ID、account recovery URL、trusted proxy configuration、mail transport security configuration を MUST validate する。
- Production allowed origins と account recovery URL は HTTPS origin を MUST 使用し、localhost、loopback、plain HTTP、empty origin、wildcard origin を MUST reject する。
- WebAuthn RP ID は allowed origin host と整合しなければならず、production runtime は RP ID と origin host の不一致を MUST reject する。
- IP-based rate limit / lockout に使う client IP は configured trusted proxy boundary の内側でのみ forwarded headers を信頼し、trusted proxy が未設定または不正な production runtime は MUST fail-close する。
- Public auth endpoints は configured body size limit を MUST enforce し、limit を超える JSON / WebAuthn payload を auth state 変更前に reject する。
- HTTP サーバーは read header timeout に加えて read timeout / write timeout / idle timeout を SHALL enforce する。
- Account recovery mail transport は production で TLS または STARTTLS を MUST require し、証明書検証を無効化してはならない。
- 認証 response および auth route を支える response は、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy`、`X-Content-Type-Options`、frame embedding prevention の security header または deployment-equivalent controls を SHALL 含む。
- 認証失敗、rate limit、handoff audit event は bearer token、OTP 値、recovery token、WebAuthn credential raw ID、secret を含む account recovery URL を log に出してはならない。

#### Scenario: unsafe production auth config fails closed (AUTH-BE-S025)

- **GIVEN** runtime environment が development ではない
- **WHEN** 認証設定に localhost origin、plain HTTP recovery URL、trusted proxy 設定不足、または WebAuthn RP ID 不一致が含まれる
- **THEN** システムは auth traffic を受け付ける前に runtime startup を拒否する

#### Scenario: oversized public auth request is rejected before state mutation (AUTH-BE-S026)

- **GIVEN** public auth endpoint が configured auth body limit を超える request body を受け取っている
- **WHEN** request が parse される
- **THEN** システムは challenge、OTP state、session、recovery state を発行せず request を拒否する

#### Scenario: recovery mail は production で secure transport を要求する (AUTH-BE-S027)

- **GIVEN** runtime environment が development ではない
- **WHEN** account recovery mail transport が証明書検証付き TLS または STARTTLS を提供できない
- **THEN** システムは fail-close するか、account existence を露出せず recovery delivery を拒否する

### Requirement: WebAuthn ceremony は user verification を必須にする

Passkey は認証基盤の中核であり、端末所持だけでなく端末内の user verification によって利用者本人の操作であることを確認する必要がある。ログイン、新しい端末でのログイン有効化、復旧後の再登録などの高リスク操作では、システムが user verification を必須にしなければならない（MUST）。user verification が optional だと、端末盗難や弱い authenticator policy による不正利用リスクが残る。

**Requirement**

- システムは認証 ceremony と登録 ceremony で WebAuthn user verification を SHALL require する。
- システムは high-risk な認証済み passkey 管理操作の前に、fresh な WebAuthn reauthentication session を SHALL require する。
- `POST /api/v1/auth/passkey/start`、`POST /api/v1/auth/passkey/finish`、`POST /api/v1/passkeys/start`、`POST /api/v1/passkeys/finish`、`POST /api/v1/auth/passkey/register/start`、`POST /api/v1/auth/passkey/register` は user verification required semantics を MUST enforce する。
- `POST /api/v1/passkeys/send-device-link` と `DELETE /api/v1/passkeys/{id}` は Product 認証セッションだけで成立してはならず、`X-Reauth-Session` HTTP header で提示された同一 account に紐づく短命 reauthentication session を要求しなければならない（MUST）。
- Reauthentication session は Valkey-backed auth state store に TTL 付きで保持され、対象 account、issuing session、operation kind（`device-link` または `passkey-delete`）、request ID を紐づけなければならない。
- Reauthentication session は high-risk operation completion 時に atomic consume されるか、短い有効期限で失効しなければならない。
- 異なる operation kind の reauthentication session を使い回した場合は MUST reject する。
- client に返す WebAuthn options が `userVerification` field を表現する場合、値は `"required"` でなければならない。
- server-side WebAuthn verification は required user verification を満たさない assertion または attestation を拒否しなければならない。

#### Scenario: login ceremony は user verification を要求する (AUTH-BE-S028)

- **GIVEN** passkey login ceremony が開始されている
- **WHEN** authenticator response が required user verification を満たさない
- **THEN** システムは login を拒否し、アプリケーションセッションを発行しない

#### Scenario: 新端末のログイン有効化は user verification を要求する (AUTH-BE-S029)

- **GIVEN** valid な device-link RecoverySession が存在する
- **WHEN** 新しい端末が required user verification なしで WebAuthn registration を完了しようとする
- **THEN** システムは registration を拒否し、account に credential を追加しない

#### Scenario: device-link delivery は fresh な再認証を要求する (AUTH-BE-S036)

- **GIVEN** account は有効な Product 認証セッションを持つが fresh な reauthentication session を持たない
- **WHEN** account が device-link delivery を要求する
- **THEN** システムは request を拒否し、device-link token を発行または送信しない

#### Scenario: passkey deletion は fresh な再認証を要求する (AUTH-BE-S037)

- **GIVEN** account は有効な Product 認証セッションを持つが fresh な reauthentication session を持たない
- **WHEN** account が登録済み passkey credential の削除を要求する
- **THEN** システムは削除を拒否し、すべての credential を変更しない

### Requirement: 認証済み端末から新端末追加用トークンを発行できる

認証済み端末は、既存パスキーによる再認証を完了した後、登録済みメールアドレスへ新端末追加用の単回利用 URL トークンを発行しなければならない（SHALL）。トークンは kind=device-link として管理され、消費後に新端末でのパスキー登録を可能にする。

**Customer Context**

利用者が新しい端末でログインできるようにしたい場合、認証済み端末から安全に新端末追加リンクを発行できる必要がある。OTP 方式のように短い数字コードを手入力するのではなく、メール内の URL をクリックするだけで新端末でのパスキー追加を開始できる UX が求められる。

**Requirement**

- `POST /api/v1/passkeys/send-device-link` は認証済み端末から新端末追加用の device-link token を発行し、登録済みメールアドレスへ送信しなければならない（SHALL）。このエンドポイントは Product 認証セッションと `X-Reauth-Session` header で提示された operation kind `device-link` の reauthentication session を要求しなければならず（MUST）、session-only では成立してはならない。
- device-link token は `kind=device-link` で RecoveryToken として管理され、既存の RecoveryToken と同一のライフサイクル（発行・ハッシュ保存・原子消費・TTL 管理）を SHALL 共有する。
- device-link token の有効期限は発行から 30 分とし、Valkey-backed auth state store に HMAC-SHA256 + pepper でハッシュ化して保存されなければならない（MUST）。
- `POST /api/v1/passkeys/send-device-link` の response は `Cache-Control: no-store` を SHALL 保つ。
- メール送信の失敗は account existence を露出せず、accepted response を維持しなければならない。

#### Scenario: 認証済み端末からデバイスリンクを送信できる (AUTH-BE-S049)

- **GIVEN** 認証済みアカウントが Product 認証セッションを持ち、device-link 用 reauthentication session を保持している
- **WHEN** `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** システムは kind=device-link の token を発行し、新端末追加用 URL を含むメールを登録メールアドレスへ送信し、`{issued: true}` を返す

#### Scenario: reauthentication なしではデバイスリンクを発行できない (AUTH-BE-S050)

- **GIVEN** 認証済みアカウントが Product 認証セッションを持つが reauthentication session を持たない
- **WHEN** `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** システムは request を拒否し、token を発行しない

### Requirement: パスキー登録完了時に kind に応じた後処理を実行する

パスキー登録（`POST /api/v1/auth/passkey/register`）が成功した場合、使用された RecoverySession の kind に応じて、セッション失効と通知メールの後処理を実行しなければならない（MUST）。

**Customer Context**

パスキー紛失からの復旧（kind=recovery）はアカウント乗っ取りの可能性があるため、既存の全セッションを強制失効して攻撃者のアクセスを遮断する必要がある。一方、認証済み端末からの新端末追加（kind=device-link）ではセッション失効は不要だが、どちらの場合もアカウント所有者に通知して異常を検知できるようにする必要がある。

**Requirement**

- `POST /api/v1/auth/passkey/register` が RecoverySession を使用して成功した場合、システムは登録を完了した後、RecoverySession の `kind` を MUST 評価する。
- `kind` が `"recovery"` である場合、システムは該当アカウントに紐づく全 active session を Valkey から即座に削除し SHALL 強制失効する。その後、登録済みメールアドレスへ復旧完了通知メールを SHALL 送信する。
- `kind` が `"device-link"` である場合、システムはセッションを一切失効してはならない（MUST NOT）。登録済みメールアドレスへ新端末追加完了通知メールを SHALL 送信する。
- 通知メールの送信失敗はパスキー登録の成功を妨げてはならない（MUST NOT）。失敗時は fire-and-forget で structured log に記録する。
- セッション失効操作は Valkey-backed auth state store に即座に反映され、node 間で共有されなければならない（MUST）。

#### Scenario: kind=recovery の登録完了で全セッションが失効する (AUTH-BE-S051)

- **GIVEN** アカウントが 3 つの active session を持ち、kind=recovery の RecoverySession でパスキー登録を完了した
- **WHEN** システムが後処理を実行する
- **THEN** 既存の 3 セッションはすべて削除され、新しい登録で発行された 1 セッションのみが有効である

#### Scenario: kind=device-link の登録完了でセッションは失効しない (AUTH-BE-S052)

- **GIVEN** アカウントが 2 つの active session を持ち、kind=device-link の RecoverySession でパスキー登録を完了した
- **WHEN** システムが後処理を実行する
- **THEN** 既存の 2 セッションは維持され、新しい登録で発行されたセッションを含め 3 セッションが有効である

#### Scenario: 通知メールの送信失敗は登録成功を妨げない (AUTH-BE-S053)

- **GIVEN** kind=recovery または kind=device-link の RecoverySession でパスキー登録が成功している
- **WHEN** 通知メールの送信が SMTP エラーで失敗する
- **THEN** システムは 200 で新しい認証セッションを返し、メール送信失敗は structured log に記録されるが registration は成功している

### Requirement: リフレッシュトークンは設定可能な TTL で管理される

リフレッシュトークンは設定可能な TTL で管理されなければならない（SHALL）。TTL validation logic は Product account auth と Admin operator auth から共通利用できる中立 primitive でなければならない（SHALL）。TTL primitive は account / operator の domain decision を所有してはならない（MUST NOT）。TTL 付き refreshToken state は各認証ドメインの Valkey logical DB と key prefix に保存されなければならない（MUST）。refreshToken Cookie の `Max-Age` または `Expires` は server-side refreshToken state の TTL を超えてはならない（MUST NOT）。TTL が未設定またはゼロ値の場合の無期限扱いを許可する場合でも、Cookie と server-side state の整合を保たなければならない（MUST）。

**Customer Context**

運用者はセキュリティポリシーに応じて refreshToken の寿命を調整したい。Product と Admin で TTL validation が分かれると、片方だけ弱い設定を受け入れる危険がある。一方で TTL helper が account/operator の業務判断まで持つと、認証ドメイン分離が崩れる。

#### Scenario: refreshToken Cookie の寿命は server-side TTL を超えない (AUTH-BE-S064)

- **GIVEN** refresh token TTL が 30 日に設定されている
- **WHEN** login または refresh rotation が refreshToken Cookie を設定する
- **THEN** Cookie の `Max-Age` または `Expires` は server-side refreshToken state の TTL 以下である

#### Scenario: Product と Admin は同じ中立 TTL validation を使う (AUTH-BE-S065)

- **GIVEN** refresh token TTL が許容範囲外に設定されている
- **WHEN** Product API binary または Admin API binary が起動時 config validation を実行する
- **THEN** どちらの binary も同じ中立 TTL validation rule で fail-close する
- **AND** TTL helper は account status、operator role、CSRF を判定しない

#### Scenario: 未設定のリフレッシュトークン TTL は無期限有効とする (AUTH-BE-S038)

- **GIVEN** `auth.refresh_token_ttl` が未設定またはゼロである
- **WHEN** システムがリフレッシュトークンを発行する
- **THEN** Cookie と server-side state は整合し、明示的な失効または消費まで有効である

### Requirement: システムは複数の active session を同時に保持・管理できる

システムは同一 browser 上で複数 account session を保持できなければならない（SHALL）。各 Product account session は accessToken、account session ID、account ID、server-side refreshToken state、refreshToken Cookie binding を持たなければならない（SHALL）。refreshToken が HttpOnly Cookie であるため、クライアントは refreshToken 平文を保持してはならない（MUST NOT）。複数 session の refresh は、Product account auth domain が session selector と Cookie binding を検証して対象 session だけを rotation しなければならない（MUST）。logout や session revoke は対象 session の accessToken metadata と refreshToken state / Cookie を失効させ、他 session に影響してはならない（MUST NOT）。

**Customer Context**

複数アカウントを扱う利用者は、account を切り替えながら作業したい。一方で refreshToken を JavaScript から読める形で保持すると XSS 時の被害が大きい。HttpOnly Cookie と session selector を組み合わせ、複数 session と token 窃取防止を両立する。

#### Scenario: 複数 session の refresh は対象 session だけを rotation する (AUTH-BE-S066)

- **GIVEN** browser が account A と account B の active session を持っている
- **WHEN** account A の session selector で `POST /api/v1/auth/refresh` を呼び出す
- **THEN** account A の refreshToken state と Cookie binding だけが rotation される
- **AND** account B の session は維持される

#### Scenario: 単一セッションのログアウトは他のセッションに影響しない (AUTH-BE-S042)

- **GIVEN** 利用者がアカウント A とアカウント B の両方で active セッションを持っている
- **WHEN** アカウント A のセッションで `POST /api/v1/auth/logout` を実行する
- **THEN** アカウント A のセッションは失効し、アカウント B のセッションは引き続き有効である

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

Product account 認証ドメインは、request の `credentialMode` に応じて Product account session credential を SHALL 発行しなければならない。Web Cookie mode では短命な account access credential と長寿命 account refresh credential を HttpOnly Cookie として発行し、response body に bearer accessToken または refreshToken 平文を含めてはならない（MUST NOT）。Bearer mode では API / mobile / CLI / SDK 向けに `Authorization: Bearer <accessToken>` で Product `/api/v1/*` を利用できる body token を発行する。Product account session credential は account ID / account session ID / device fingerprint / account status / sessionRevokedAfter に束縛されなければならない（MUST）。DB の `accounts.status='suspended'` の account に対しては、新規 session credential 発行、refresh rotation、既存 session credential 認可を MUST 拒否する。

Admin operator 認証ドメインは Product account 認証ドメインとは別に、operator accessToken と operator refreshToken で構成される Admin operator session を SHALL 発行する。Admin operator accessToken claim と refreshToken state は operator ID / operator session ID / operator role / active state / CSRF binding / Admin Valkey logical DB / `admin:*` key prefix に束縛されなければならない（MUST）。Admin operator auth は Product account auth domain/application を import してはならず（MUST NOT）、Product account auth は Admin operator auth domain/application を import してはならない（MUST NOT）。

両認証ドメインが共有できるのは、HMAC/JWT signer/verifier、opaque token hash、Cookie 属性 helper、ULID/JTI validation、TTL validation helper など中立 primitive に限られる（MUST）。中立 primitive は account / operator の domain enum switch、issuer/audience/domain pairing、RBAC、account status、operator active state、CSRF binding を所有してはならない（MUST NOT）。単一共有 token service に `identityDomain=account|operator` の切替引数を渡して Product/Admin の domain decision を畳み込んではならない（MUST NOT）。

**Customer Context**

Product 利用者の account 認証と Admin 運営者の operator 認証は、守る対象と失敗時の影響が異なる。Cookie 属性や署名検証のような安全な primitive は共通化してよいが、account status、operator role、CSRF、session state を単一 service の切替で扱うと境界が曖昧になり、誤認可や監査漏れにつながる。

#### Scenario: Web Cookie mode の Product passkey login は body token を返さない (AUTH-BE-S060)

- **GIVEN** account が Product passkey authentication を開始している
- **WHEN** valid credential で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** response body は Product account session metadata と CSRF token を含む
- **AND** response body は bearer accessToken と refreshToken 平文を含まない
- **AND** `Set-Cookie` は access credential と refresh credential を `HttpOnly; Secure; SameSite=Lax` で設定する

#### Scenario: Admin operator login は Admin operator auth domain を使う (AUTH-BE-S061)

- **GIVEN** operator が Admin passkey authentication を開始している
- **WHEN** valid operator credential で Admin auth finish を完了する
- **THEN** response body は operator accessToken と operator session metadata を含む
- **AND** refreshToken Cookie は operator ID、operator session、CSRF binding、Admin Valkey namespace に束縛される
- **AND** Product account ID、Product AccountAuth session、Product application service は使用されない

#### Scenario: Product と Admin の認証ドメインは単一 switch に畳み込まれない (AUTH-BE-S067)

- **WHEN** accessToken 発行、refresh rotation、session revoke の implementation を確認する
- **THEN** Product account auth は Product account auth domain/application の型と service を使う
- **AND** Admin operator auth は Admin operator auth domain/application の型と service を使う
- **AND** `identityDomain` などの引数で account/operator domain decision を切り替える単一共有 token service は存在しない

#### Scenario: 中立 token primitive は account/operator domain switch を持たない (AUTH-BE-S068)

- **WHEN** shared token primitive の public API と内部実装を確認する
- **THEN** 署名、検証、opaque token hash、ULID/JTI validation、TTL validation だけを扱う
- **AND** account / operator enum、RBAC、status 判定、CSRF 判定、issuer/audience/domain pairing を持たない

#### Scenario: Product AccountAuth domain が account token eligibility を所有する (AUTH-BE-S069)

- **GIVEN** account が suspended または sessionRevokedAfter より古い session を持つ
- **WHEN** Product account accessToken 発行または refresh rotation を行う
- **THEN** Product AccountAuth domain object は token eligibility を拒否する

#### Scenario: Admin OperatorAuth domain が operator token eligibility と CSRF binding を所有する (AUTH-BE-S070)

- **GIVEN** operator が inactive、または CSRF token が operator session と一致しない
- **WHEN** Admin operator accessToken 発行、refresh rotation、protected mutation validation を行う
- **THEN** Admin OperatorAuth domain object は token eligibility または CSRF binding を拒否する

#### Scenario: Product auth application は Admin auth application を import しない (AUTH-BE-S071)

- **WHEN** `internal/application/product/auth` が `internal/application/admin` または Admin OperatorAuth application を import している
- **THEN** lint または import-boundary test は失敗する

#### Scenario: Admin auth application は Product auth application を import しない (AUTH-BE-S072)

- **WHEN** `internal/application/admin/auth` が `internal/application/product` または Product AccountAuth application を import している
- **THEN** lint または import-boundary test は失敗する

#### Scenario: refresh は Cookie の refreshToken を rotation する (AUTH-BE-S062)

- **GIVEN** クライアントが有効な refreshToken Cookie を持つ
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を呼び出す
- **THEN** 対象 auth domain は旧 refreshToken を原子消費し、新しい session credential と CSRF token を返す
- **AND** Web Cookie mode の refreshToken は `Set-Cookie` で rotation され、response body には含まれない

#### Scenario: ブラウザーから読める refreshToken は発行されない (AUTH-BE-S063)

- **GIVEN** login、refresh、recovery registration、または operator login が成功する
- **WHEN** response body と log/trace attributes を確認する
- **THEN** refreshToken の平文値は body、log、trace attribute、error message に存在しない

#### Scenario: suspended account は新規 accessToken を発行されない (AUTH-BE-S054)

- **GIVEN** account の `accounts.status` が `suspended` である
- **WHEN** account が valid passkey で `POST /api/v1/auth/passkey/finish` を完了しようとする
- **THEN** システムは access credential / refresh credential を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の既存 bearer accessToken は拒否される (AUTH-BE-S055)

- **GIVEN** account が active session を持っていた後に Admin Console で suspended になっている
- **WHEN** その accessToken で `/api/v1/*` にアクセスする
- **THEN** システムは session credential を認可せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の refresh は rotation されない (AUTH-BE-S058)

- **GIVEN** account が valid refreshToken Cookie を持っていた後に Admin Console で suspended になっている
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を呼び出す
- **THEN** システムは新しい access credential / refresh credential を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspend は account-wide session revocation timestamp を書き込む (AUTH-BE-S056)

- **GIVEN** Admin Console が account suspend を成功させる
- **WHEN** DB の `accounts.session_revoked_after` を確認する
- **THEN** suspend 時刻以上の timestamp が保存され、その timestamp 以前に発行された session credential は拒否される

#### Scenario: restored account は過去 session では復帰できない (AUTH-BE-S057)

- **GIVEN** account が suspended 後に restore されている
- **WHEN** suspend 前に発行された session credential で `/api/v1/*` にアクセスする
- **THEN** システムは session credential を拒否し、account は再ログインでのみ新しい session credential を取得できる

#### Scenario: account-suspended は stable failure response shape で返される (AUTH-BE-S059)

- **GIVEN** suspended 判定が `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/refresh`、または bearer-protected `/api/v1/*` endpoint で発生する
- **WHEN** システムが response を返す
- **THEN** HTTP status は 403 であり、body は `AuthFailureResponse` の `{ requestId, error: "account-suspended" }` である
- **AND** response は `Cache-Control: no-store` を含み、`AuthOperationErrorResponse` では返されない

#### Scenario: session を持たない request は session-expired と混同されない (AUTH-BE-S009)

- **GIVEN** request が `/api/v1/*` を対象にしている
- **WHEN** request が Product 認証セッションをまったく提示しない
- **THEN** システムは unauthenticated failure として拒否し、expired / revoked session 用の failure と区別する

#### Scenario: auth state store unavailable は fail-close で internal-error になる (AUTH-BE-S010)

- **GIVEN** request が auth boundary を通って `/api/v1/*` または auth endpoint を呼んでいる
- **WHEN** Valkey を含む auth state store が unavailable で session / challenge / recovery state を安全に検証できない
- **THEN** システムは fail-close で request を拒否し、stable classification `internal-error` を返す

#### Scenario: 消費済みリフレッシュトークンの再利用は拒否され関連トークンを失効する (AUTH-BE-S044)

- **GIVEN** リフレッシュトークンが既に消費されている
- **WHEN** 同じ旧リフレッシュトークンで `POST /api/v1/auth/refresh` を再試行する
- **THEN** システムは request を拒否し、同一アカウント・同一デバイス指紋のすべてのリフレッシュトークンを失効させる

#### Scenario: 不正なリフレッシュトークンは拒否される (AUTH-BE-S045)

- **GIVEN** クライアントが存在しないまたは改竄されたリフレッシュトークンを提示している
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を送信する
- **THEN** システムは request を拒否し、新しい session credential を発行しない

#### Scenario: accessToken の有効期限切れは session-expired として拒否される (AUTH-BE-S046)

- **GIVEN** クライアントが有効期限切れの JWT accessToken を保持している
- **WHEN** そのトークンを `Authorization: Bearer` ヘッダーに設定して `/api/v1/*` を呼び出す
- **THEN** システムは `session-expired` failure として拒否する

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
