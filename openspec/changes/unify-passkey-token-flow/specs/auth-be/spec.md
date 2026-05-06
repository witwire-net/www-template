## REMOVED Requirements

### Requirement: OTP ハンドオフによる新端末へのパスキー追加

**Reason**: OTP ハンドオフ（`DeviceLoginHandoff`）とメールリカバリー（`RecoveryToken`）は本質的に同一の「メールを信頼基点としたパスキー追加」操作でありながら、2 系統の実装が存在していた。OTP 方式は 6 桁数字（~20bit）の低エントロピーにより総当たり耐性が不十分であり、URL トークン方式（ULID, 80bit）への一本化によりセキュリティと保守性の両面を改善する。

**Migration**: 既存の Valkey `auth:handoff:*` キーは最大 5 分 TTL で自然消滅する。OTP 発行エンドポイント (`POST /api/v1/passkeys/otp`) と OTP 検証エンドポイント (`POST /api/v1/auth/passkey/add/start`, `POST /api/v1/auth/passkey/add/finish`) は廃止され、`POST /api/v1/passkeys/send-device-link` が代替となる。フロントエンドの `/passkeys/add` ルートは削除され、代わりにデバイスリンクメール経由のトークン消費・パスキー登録フローが提供される。

## MODIFIED Requirements

### Requirement: recovery token は単回利用・期限付きで enumeration-safe に扱う

recovery token は、単回利用・期限付き・enumeration-safe なパスキー追加用 credential として SHALL 扱われなければならない。

**Customer Context**

パスキー紛失時の復旧と新端末追加は、どちらも登録済みメールアドレスへの URL 送信によってパスキー再登録を可能にする必要がある。同時にこれらの導線は、アカウント有無や token 状態を推測できないように保護されなければならない。短命 token の保管、受理応答、temporary lock が曖昧だと、Auth コア全体の安全性が下がる。また、復旧と新端末追加では token 消費後の挙動（セッション失効の有無、通知文面）が異なるため、token 自体に kind を持たせる必要がある。

**Requirement**

- The system SHALL treat recovery tokens and device-link tokens as single-use time-limited credentials and MUST keep issuance request responses enumeration-safe.
- Token は `kind` フィールドを持ち、`"recovery"`（パスキー紛失時の復旧）または `"device-link"`（認証済み端末からの新端末追加）のいずれかを MUST 指定する。
- The system SHALL expose `POST /api/v1/auth/recovery` to accept a registered email address and issue a single-use time-limited token with `kind=recovery`, then send a recovery URL to the registered address through SMTP.
- The system SHALL expose `POST /api/v1/passkeys/send-device-link` to accept a bearer session with `X-Reauth-Session` header (operation kind `device-link`) and issue a single-use time-limited token with `kind=device-link`, then send a device-link URL to the registered email address through SMTP. This endpoint MUST require a valid bearer session and a consumed reauthentication session; bearer-only requests SHALL be rejected.
- RecoveryToken と RecoverySession は Valkey-backed auth state store に MUST 保持され、`kind` を含む完全な状態で永続化される。
- RecoveryToken 自体の resource ID、RecoverySession の resource ID、delivery request ID、mail/audit correlation ID など flow を追跡する識別子が必要な箇所は ULID を SHALL 使用する。
- `POST /api/v1/auth/recovery` および `POST /api/v1/passkeys/send-device-link` は account 有無や throttle 状態を外部から判別できない accepted response を SHALL 返す。
- The system SHALL expose `POST /api/v1/auth/recovery/consume` to validate a token, mark the token consumed atomically, and create a passkey re-registration RecoverySession that inherits the token's `kind`.
- RecoveryToken secret は平文 lookup 値として保存してはならず、server-side keyed hash (HMAC-SHA256 with server-side pepper) で MUST 保存・照合する。
- RecoveryToken consume と RecoverySession consume は atomic operation として MUST 実行され、同じ token または session から複数の有効結果を作成してはならない。
- 無効、期限切れ、revoke 済み、または consumed 済みの token から RecoverySession を作成してはならない（MUST NOT）。
- 発行・消費・登録 response は `Cache-Control: no-store` を SHALL 保ち、temporary lock / throttle の state は Valkey-backed auth state store に MUST 保持される。
- Token を含む URL は response body、log、trace attribute、error message に MUST 出力されない。

#### Scenario: 復旧依頼は kind=recovery の token を発行して受理される (AUTH-BE-S004)

- **GIVEN** 利用者が passkey recovery を依頼する
- **WHEN** 利用者が `POST /api/v1/auth/recovery` を送信する
- **THEN** system は accepted response を返し、対象アカウントが存在するときだけ `kind=recovery` の time-limited token を発行して recovery URL をメール送信する

#### Scenario: 有効な復旧 token は kind を継承した RecoverySession を作成する (AUTH-BE-S005)

- **GIVEN** recovery URL が valid な token (kind=recovery) を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** token は consumed となり、system は `kind=recovery` を継承した RecoverySession を返す

#### Scenario: 有効なデバイスリンク token は kind を継承した RecoverySession を作成する (AUTH-BE-S047)

- **GIVEN** device-link URL が valid な token (kind=device-link) を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** token は consumed となり、system は `kind=device-link` を継承した RecoverySession を返す

#### Scenario: 無効な復旧 token は拒否される (AUTH-BE-S006)

- **GIVEN** recovery URL が invalid、expired、または consumed 済みの token を含んでいる
- **WHEN** 利用者が `POST /api/v1/auth/recovery/consume` を送信する
- **THEN** system は request を拒否し、RecoverySession を作成しない

#### Scenario: token の並行 consume は単一結果だけを許可する (AUTH-BE-S030)

- **GIVEN** 同じ valid token に対する複数の consume request が並行している
- **WHEN** system が token consume を処理する
- **THEN** ちょうど 1 つの request だけが RecoverySession を作成し、残りは generic failure として拒否される

### Requirement: recovery register branch は既存アカウントの再登録だけを許可する

recovery/device-link register branch は、valid な RecoverySession が指す既存アカウントの passkey 登録だけを SHALL 許可し、invite や consent state を MUST 受け入れてはならない。登録完了後はセッションの kind に応じた後処理（セッション失効・通知）を実行する。

**Customer Context**

Auth コアが扱うのは既存アカウントの passkey 回復・追加であり、招待登録や規約同意や Guest state 変更ではありません。この境界が崩れると `/login/recovery/*` と `/invite/*` が混線し、後続フェーズの責務が不明確になります。また、kind=recovery による復旧はアカウント乗っ取りの可能性があるため全既存セッションの強制失効が必要であり、kind=device-link による新端末追加ではその必要がない。

**Requirement**

- The system SHALL allow the register branch to operate only on an existing account referenced by a valid RecoverySession, and the shared register endpoint MUST keep an exactly-one selector boundary between recovery and invite state.
- `POST /api/v1/auth/passkey/register` は shared endpoint として RecoverySession または InvitationSession の exactly-one を MUST 要求し、register branch は valid な RecoverySession のみを持つときだけ SHALL 成立する。
- register branch は RecoverySession が指す既存アカウントへ新しい passkey を SHALL 登録し、new Account 作成、Guest / Member state 変更、base role 変更をしてはならない。
- register branch は invitation-session validation、invite-token consume、invite consent completion、TermsConsent read / write を MUST NOT 要求しない。
- RecoverySession と InvitationSession を同時に提示する request、または両方を欠く request は branch ambiguity として MUST reject しなければならない。
- register branch が成功した後は、新しい active bearer session を SHALL 返す。
- RecoverySession の `kind` が `"recovery"` である場合、system はパスキー登録成功後に該当アカウントの全既存セッションを強制失効しなければならない（SHALL revoke all sessions for the account）。その後、復旧完了を通知するメールを登録済みメールアドレスへ SHALL 送信する。
- RecoverySession の `kind` が `"device-link"` である場合、system はパスキー登録成功後にセッションを失効してはならない（MUST NOT revoke any session）。新端末追加完了を通知するメールを登録済みメールアドレスへ SHALL 送信する。
- 通知メールの送信失敗はパスキー登録の成功を妨げてはならない（MUST NOT block registration success）。失敗は fire-and-forget でログ記録する。
- `POST /api/v1/auth/passkey/register` の response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。
- consumed 済み RecoverySession を再利用してはならない（MUST NOT）。消費済み state は Valkey-backed auth state store に反映される。
- register branch が参照・生成する account ID、passkey credential ID、session ID、RecoverySession ID、関連 audit / notification / event ID は ULID を SHALL 使用する。
- `/invite/*` onboarding flow は register branch から暗黙に起動されてはならない。

#### Scenario: recovery session (kind=recovery) は既存アカウントへ passkey を再登録し全セッションを失効する (AUTH-BE-S007)

- **GIVEN** 利用者が kind=recovery の valid な RecoverySession を保持している
- **WHEN** 利用者が register branch として `POST /api/v1/auth/passkey/register` を送信する
- **THEN** system は既存アカウントへ新しい passkey を登録し、該当アカウントの全既存セッションを強制失効し、復旧完了通知メールを送信し、新しい active bearer session を返す

#### Scenario: device-link session (kind=device-link) は passkey を追加しセッションは失効しない (AUTH-BE-S048)

- **GIVEN** 利用者が kind=device-link の valid な RecoverySession を保持している
- **WHEN** 利用者が register branch として `POST /api/v1/auth/passkey/register` を送信する
- **THEN** system は既存アカウントへ新しい passkey を登録し、既存セッションを一切失効せず、新端末追加完了通知メールを送信し、新しい active bearer session を返す

#### Scenario: invite-only state では registration を完了できない (AUTH-BE-S008)

- **GIVEN** request が valid な RecoverySession なしで registration を試みる
- **WHEN** request が invite 向け state のみ、TermsConsent のみ、または利用可能な recovery state を持たない
- **THEN** system は registration を拒否し、`/login/recovery/*` と `/invite/*` の分離を維持する

### Requirement: auth throttle と temporary lock は non-revealing に強制する

auth throttle と temporary lock は、abuse を抑止しつつ account existence や branch state を外部へ漏らさない guardrail として SHALL 強制されなければならない。

**Customer Context**

runtime decision は `passkey/start` throttle、recovery request throttle、send-device-link throttle、finish / consume / register 失敗時の temporary lock を必須要件として定義しています。これらが dedicated requirement / scenario を持たないと、temporary lock や throttle が実装で弱まり、enumeration-safe な recovery/device-link と shared register seam の安全性が保証できません。

**Requirement**

- The system SHALL enforce the documented auth throttle and temporary lock policies and MUST keep guarded responses non-revealing while those policies are active.
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

- **GIVEN** client が configured budget を超えて `POST /api/v1/auth/recovery` を繰り返している
- **WHEN** system が recovery throttle を適用する
- **THEN** system は同一の accepted / no-store response shape を維持し、登録済み account の有無や throttle hit を外部へ露出しない

#### Scenario: throttled passkey start は no-store かつ non-revealing に reject される (AUTH-BE-S013)

- **GIVEN** client が configured budget 超過まで `POST /api/v1/auth/passkey/start` を繰り返している
- **WHEN** system が public passkey start throttle を適用する
- **THEN** system は追加 challenge を発行せず no-store boundary を保ったまま request を reject し、新しい公開 stable error code や account state を露出しない

#### Scenario: repeated auth failures は temporary lock を発動する (AUTH-BE-S012)

- **GIVEN** public auth completion、recovery/device-link consume、または registration に対する失敗が configured window 内で累積している
- **WHEN** client が temporary lock 期間中に guarded endpoint を再試行する
- **THEN** system は temporary lock として request を reject し、challenge completion、token consume、passkey registration を進めない

#### Scenario: identifier rotation では passkey start budget を回避できない (AUTH-BE-S031)

- **GIVEN** client が異なる identifier 値で `POST /api/v1/auth/passkey/start` を繰り返している
- **WHEN** IP または global challenge issuance budget が上限に達している
- **THEN** system は追加 challenge を発行せず non-revealing に reject する

### Requirement: WebAuthn ceremony は user verification を必須にする

Passkey は認証基盤の中核であり、端末所持だけでなく端末内の user verification によって利用者本人の操作であることを確認する必要がある。ログイン、新しい端末でのログイン有効化、復旧後の再登録などの高リスク操作で user verification が optional だと、端末盗難や弱い authenticator policy による不正利用リスクが残る。

**Requirement**

- システムは認証 ceremony と登録 ceremony で WebAuthn user verification を SHALL require する。
- システムは high-risk な認証済み passkey 管理操作の前に、fresh な WebAuthn reauthentication session を SHALL require する。
- `POST /api/v1/auth/passkey/start`、`POST /api/v1/auth/passkey/finish`、`POST /api/v1/passkeys/start`、`POST /api/v1/passkeys/finish`、`POST /api/v1/auth/passkey/register/start`、`POST /api/v1/auth/passkey/register` は user verification required semantics を MUST enforce する。
- `POST /api/v1/passkeys/send-device-link` と `DELETE /api/v1/passkeys/{id}` は bearer session だけで成立してはならず、`X-Reauth-Session` HTTP header で提示された同一 account に紐づく短命 reauthentication session を MUST require する。
- Reauthentication session は Valkey-backed auth state store に TTL 付きで保持され、対象 account、issuing session、operation kind（`device-link` または `passkey-delete`）、request ID を紐づけなければならない。
- Reauthentication session は high-risk operation completion 時に atomic consume されるか、短い有効期限で失効しなければならない。
- 異なる operation kind の reauthentication session を使い回した場合は MUST reject する。
- client に返す WebAuthn options が `userVerification` field を表現する場合、値は `"required"` でなければならない。
- server-side WebAuthn verification は required user verification を満たさない assertion または attestation を拒否しなければならない。

#### Scenario: login ceremony は user verification を要求する (AUTH-BE-S028)

- **GIVEN** passkey login ceremony が開始されている
- **WHEN** authenticator response が required user verification を満たさない
- **THEN** system は login を拒否し、bearer session を発行しない

#### Scenario: 新端末のログイン有効化は user verification を要求する (AUTH-BE-S029)

- **GIVEN** valid な device-link RecoverySession が存在する
- **WHEN** 新しい端末が required user verification なしで WebAuthn registration を完了しようとする
- **THEN** system は registration を拒否し、account に credential を追加しない

#### Scenario: device-link delivery は fresh な再認証を要求する (AUTH-BE-S036)

- **GIVEN** account は active bearer session を持つが fresh な reauthentication session を持たない
- **WHEN** account が device-link delivery を要求する
- **THEN** system は request を拒否し、device-link token を発行または送信しない

#### Scenario: passkey deletion は fresh な再認証を要求する (AUTH-BE-S037)

- **GIVEN** account は active bearer session を持つが fresh な reauthentication session を持たない
- **WHEN** account が登録済み passkey credential の削除を要求する
- **THEN** system は削除を拒否し、すべての credential を変更しない

## ADDED Requirements

### Requirement: 認証済み端末から新端末追加用トークンを発行できる

認証済み端末は、既存パスキーによる再認証を完了した後、登録済みメールアドレスへ新端末追加用の単回利用 URL トークンを発行しなければならない。トークンは kind=device-link として管理され、消費後に新端末でのパスキー登録を可能にする。

**Customer Context**

利用者が新しい端末でログインできるようにしたい場合、認証済み端末から安全に新端末追加リンクを発行できる必要がある。OTP 方式のように短い数字コードを手入力するのではなく、メール内の URL をクリックするだけで新端末でのパスキー追加を開始できる UX が求められる。

**Requirement**

- `POST /api/v1/passkeys/send-device-link` は認証済み端末から新端末追加用の device-link token を発行し、登録済みメールアドレスへ送信しなければならない（SHALL）。このエンドポイントは bearer session と `X-Reauth-Session` header で提示された operation kind `device-link` の reauthentication session を MUST require し、bearer-only では成立してはならない。
- device-link token は `kind=device-link` で RecoveryToken として管理され、既存の RecoveryToken と同一のライフサイクル（発行・ハッシュ保存・原子消費・TTL 管理）を SHALL 共有する。
- device-link token の有効期限は発行から 30 分とし、Valkey-backed auth state store に HMAC-SHA256 + pepper でハッシュ化して保存されなければならない（MUST）。
- `POST /api/v1/passkeys/send-device-link` の response は `Cache-Control: no-store` を SHALL 保つ。
- メール送信の失敗は account existence を露出せず、accepted response を維持しなければならない。

#### Scenario: 認証済み端末からデバイスリンクを送信できる (AUTH-BE-S049)

- **GIVEN** 認証済みアカウントが bearer session を持ち、device-link 用 reauthentication session を保持している
- **WHEN** `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** system は kind=device-link の token を発行し、新端末追加用 URL を含むメールを登録メールアドレスへ送信し、`{issued: true}` を返す

#### Scenario: reauthentication なしではデバイスリンクを発行できない (AUTH-BE-S050)

- **GIVEN** 認証済みアカウントが bearer session を持つが reauthentication session を持たない
- **WHEN** `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** system は request を拒否し、token を発行しない

### Requirement: パスキー登録完了時に kind に応じた後処理を実行する

パスキー登録（`POST /api/v1/auth/passkey/register`）が成功した場合、使用された RecoverySession の kind に応じて、セッション失効と通知メールの後処理を実行しなければならない。

**Customer Context**

パスキー紛失からの復旧（kind=recovery）はアカウント乗っ取りの可能性があるため、既存の全セッションを強制失効して攻撃者のアクセスを遮断する必要がある。一方、認証済み端末からの新端末追加（kind=device-link）ではセッション失効は不要だが、どちらの場合もアカウント所有者に通知して異常を検知できるようにする必要がある。

**Requirement**

- `POST /api/v1/auth/passkey/register` が RecoverySession を使用して成功した場合、system は登録を完了した後、RecoverySession の `kind` を MUST 評価する。
- `kind` が `"recovery"` である場合、system は該当アカウントに紐づく全 active session を Valkey から即座に削除し SHALL 強制失効する。その後、登録済みメールアドレスへ復旧完了通知メールを SHALL 送信する。
- `kind` が `"device-link"` である場合、system はセッションを一切失効してはならない（MUST NOT）。登録済みメールアドレスへ新端末追加完了通知メールを SHALL 送信する。
- 通知メールの送信失敗はパスキー登録の成功を妨げてはならない（MUST NOT）。失敗時は fire-and-forget で structured log に記録する。
- セッション失効操作は Valkey-backed auth state store に即座に反映され、node 間で共有されなければならない（MUST）。

#### Scenario: kind=recovery の登録完了で全セッションが失効する (AUTH-BE-S051)

- **GIVEN** アカウントが 3 つの active session を持ち、kind=recovery の RecoverySession でパスキー登録を完了した
- **WHEN** system が後処理を実行する
- **THEN** 既存の 3 セッションはすべて削除され、新しい登録で発行された 1 セッションのみが有効である

#### Scenario: kind=device-link の登録完了でセッションは失効しない (AUTH-BE-S052)

- **GIVEN** アカウントが 2 つの active session を持ち、kind=device-link の RecoverySession でパスキー登録を完了した
- **WHEN** system が後処理を実行する
- **THEN** 既存の 2 セッションは維持され、新しい登録で発行されたセッションを含め 3 セッションが有効である

#### Scenario: 通知メールの送信失敗は登録成功を妨げない (AUTH-BE-S053)

- **GIVEN** kind=recovery または kind=device-link の RecoverySession でパスキー登録が成功している
- **WHEN** 通知メールの送信が SMTP エラーで失敗する
- **THEN** system は 200 で新しい認証セッションを返し、メール送信失敗は structured log に記録されるが registration は成功している
