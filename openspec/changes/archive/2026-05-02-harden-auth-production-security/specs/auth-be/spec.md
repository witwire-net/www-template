## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: recovery token は単回利用・期限付きで enumeration-safe に扱う

recovery token は、単回利用・期限付き・enumeration-safe な復旧 credential として SHALL 扱われなければならない。

**Customer Context**

パスキー紛失時の復旧は登録済みメールアドレスだけで成立する必要がありますが、同時に recovery 導線はアカウント有無や token 状態を推測できないように保護されなければなりません。短命 token の保管、受理応答、temporary lock、atomic な単回利用が曖昧だと、Auth コア全体の安全性が下がります。

**Requirement**

- システムは recovery token を single-use time-limited credential として扱い、recovery request response を enumeration-safe に保たなければならない。
- システムは `POST /api/v1/auth/recovery` を公開し、登録メールアドレスを受け取り、single-use time-limited な RecoveryToken を発行し、secure mail delivery で登録済みアドレスへ recovery URL を送信しなければならない。
- RecoveryToken と `recovery_session` は Valkey-backed auth state store に MUST 保持される。
- RecoveryToken secret は平文 lookup 値として保存してはならず、server-side keyed hash または同等の one-way representation で MUST 保存・照合する。
- RecoveryToken consume と `recovery_session` consume は atomic operation として MUST 実行され、同じ token または session から複数の有効結果を作成してはならない。
- RecoveryToken 自体の resource ID、`recovery_session` の resource ID、delivery request ID、mail/audit correlation ID など recovery flow を追跡する識別子が必要な箇所は ULID を SHALL 使用する。
- `POST /api/v1/auth/recovery` は account 有無や throttle 状態を外部から判別できない accepted response を SHALL 返す。
- システムは `POST /api/v1/auth/recovery/consume` を公開し、RecoveryToken を検証し、token を atomic に consumed として mark し、passkey 再登録用 `recovery_session` を作成しなければならない。
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
- **THEN** token は atomic に consumed となり、system は Valkey-backed auth state store 上の passkey 再登録用 `recovery_session` を返す

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

公開認証面には passkey start、recovery request、recovery consume、新しい端末でログインを有効化する handoff が含まれる。これらが endpoint ごとの弱い制限だけに依存すると、identifier rotation、IP spoofing、OTP brute force、challenge accumulation による account takeover や DoS を防げません。利用者には generic な失敗として見せながら、運用上は十分に細かい rate limit と lock state が必要です。

**Requirement**

- システムは documented auth throttle と temporary lock policy を強制し、それらが active な間は guarded response を non-revealing に保たなければならない。
- `POST /api/v1/auth/passkey/start` は IP bucket と global bucket の rate limit を MUST 適用し、identifier の変化だけで challenge issuance budget を回避できてはならない。
- `POST /api/v1/auth/passkey/start` は configured budget を超える request に追加 WebAuthn challenge を発行してはならない。
- Public WebAuthn challenge state は all nodes で共有できる Valkey-backed auth state store に TTL 付きで保持され、in-memory only の unbounded pending state として保持されてはならない。
- `POST /api/v1/auth/recovery` は email ごとに 3 回 / 1 時間、IP ごとに 10 回 / 1 時間の throttle を MUST 適用し、throttle 中でも generic accepted response と `Cache-Control: no-store` を SHALL 維持する。
- `POST /api/v1/auth/passkey/add/start` と `POST /api/v1/auth/passkey/add/finish` は email、IP、email+IP、OTP handoff、account、global の configured buckets に rate limit と temporary lock を MUST 適用する。
- `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/recovery/consume`、recovery branch の `POST /api/v1/auth/passkey/register`、および device login handoff の add/start・add/finish に対する失敗は共有 failure counter を MUST 加算し、configured threshold に達した主体を temporary lock しなければならない。
- throttle counter と temporary lock state は Valkey-backed auth state store に MUST 保持され、all nodes で共有される。
- throttle counter record、temporary lock record、auth abuse event record、解除ジョブ参照 ID など guardrail state が持つ ID は ULID を SHALL 使用する。ただし email / IP 由来の bucket key 自体は resource ID ではないため ULID 変換対象に含めない。
- temporary lock 中の guarded request は no-store boundary を保ったまま reject され、account existence、invite-only state、recovery-only state、OTP validity の有無を外部へ漏らしてはならない。
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
