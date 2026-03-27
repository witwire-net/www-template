## Purpose

Auth core backend requirements, covering bearer-compatible passkey authentication, recovery lifecycle, shared register selector boundaries, no-store auth responses, Valkey-backed auth state, SES-backed recovery delivery, and ULID identifier policy.

## Requirements

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

パスキー認証は、`Authorization: Bearer <session token>` で `/api/v1/app/*` を利用できる application session を SHALL 発行し、logout で MUST revoke しなければならない。

**Customer Context**

`/app/*` は認証必須面であり、企画書と repository rule は app API を明示的な bearer 境界として扱います。パスキー認証が session 発行・失効・app surface の認可と一体で定義されていないと、認証面全体の境界が不安定になります。

**Requirement**

- The system SHALL issue a bearer-compatible application session on successful passkey authentication and MUST revoke that session through `POST /api/v1/app/auth/logout` on logout.
- account、passkey credential、session、revocation marker、recovery token、`recovery_session`、`invitation_session`、および auth 実行を相互参照する system-owned resource ID は ULID を SHALL 使用し、UUID その他の別方式を新規採用してはならない。
- The system SHALL expose `POST /api/v1/auth/passkey/start` to issue a WebAuthn challenge, and `POST /api/v1/auth/passkey/finish` SHALL verify the credential and create an active session for the authenticated account.
- WebAuthn challenge、active session、revoked session marker、temporary auth throttle / lock state は、node 間で共有できる Valkey-backed auth state store に TTL 付きで MUST 保持される。
- WebAuthn challenge record、session record、revocation record、failure counter record が保持する subject ID / actor ID / correlation ID / notification ID / job ID などの識別子が必要な箇所は、ULID を SHALL 用いる。
- `POST /api/v1/auth/passkey/finish` と recovery branch の `POST /api/v1/auth/passkey/register` は、`Authorization: Bearer <session token>` で `/api/v1/app/*` に提示できる同一の session 契約を SHALL 返す。
- `/api/v1/app/*` surface は active bearer session を MUST 要求し、missing / expired / revoked session を SHALL 拒否する。
- request が bearer session をまったく持たない場合は stable classification `unauthenticated` として SHALL 扱われ、expired / revoked session failure と混同してはならない。
- expired または revoked session は stable classification `session-expired` として SHALL 扱われる。
- auth state store unavailable などの fail-close な auth boundary failure は stable classification `internal-error` として SHALL 扱われる。
- logout flow は `POST /api/v1/app/auth/logout` で active session を SHALL revoke し、その後の `/api/v1/app/*` request を認可できないようにする。revoke 判定に必要な state は Valkey-backed auth state store に MUST 反映される。
- auth start / finish / `POST /api/v1/app/auth/logout` response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。

#### Scenario: パスキーログイン成功時に bearer session を作成する (AUTH-BE-S001)

- **GIVEN** account が passkey authentication を開始している
- **WHEN** account が valid credential で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** system は後続の `/api/v1/app/*` access を `Authorization: Bearer <session token>` で認可できる active session を返す

#### Scenario: 欠落または inactive な session は拒否される (AUTH-BE-S002)

- **GIVEN** request が `/api/v1/app/*` を対象にしている
- **WHEN** request が expired または revoked session を持つ
- **THEN** system はその request を `session-expired` failure として拒否する

#### Scenario: logout は active session を revoke する (AUTH-BE-S003)

- **GIVEN** account が active session を持っている
- **WHEN** logout action がその session を revoke する
- **THEN** revoked session は `/api/v1/app/*` access を以後認可しない

#### Scenario: session を持たない request は session-expired と混同されない (AUTH-BE-S009)

- **GIVEN** request が `/api/v1/app/*` を対象にしている
- **WHEN** request が bearer session をまったく提示しない
- **THEN** system は unauthenticated failure として拒否し、expired / revoked session 用の failure と区別する

#### Scenario: auth state store unavailable は fail-close で internal-error になる (AUTH-BE-S010)

- **GIVEN** request が auth boundary を通って `/api/v1/app/*` または auth endpoint を呼んでいる
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
- The system SHALL expose `POST /api/v1/auth/recovery/consume` to validate a RecoveryToken, mark the token consumed, and create a passkey re-registration `recovery_session`.
- 無効、期限切れ、revoke 済み、または consumed 済みの RecoveryToken から recovery session を作成してはならない（MUST NOT）。
- recovery request / consume response は `Cache-Control: no-store` を SHALL 保ち、temporary lock / throttle の state は Valkey-backed auth state store に MUST 保持される。

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

### Requirement: auth throttle と temporary lock は non-revealing に強制する

auth throttle と temporary lock は、abuse を抑止しつつ account existence や branch state を外部へ漏らさない guardrail として SHALL 強制されなければならない。

**Customer Context**

Phase 3 の runtime decision は `passkey/start` throttle、recovery request throttle、finish / consume / register 失敗時の temporary lock を archive-ready 必須要件として定義しています。これらが dedicated requirement / scenario を持たないと、temporary lock や throttle が実装で弱まり、enumeration-safe な recovery と shared register seam の安全性が保証できません。

**Requirement**

- The system SHALL enforce the documented auth throttle and temporary lock policies and MUST keep guarded responses non-revealing while those policies are active.
- `POST /api/v1/auth/passkey/start` は `account-or-handle + IP` ごとに 5 回 / 5 分の throttle を MUST 適用する。
- `POST /api/v1/auth/recovery` は email ごとに 3 回 / 1 時間、IP ごとに 10 回 / 1 時間の throttle を MUST 適用し、throttle 中でも generic accepted response と `Cache-Control: no-store` を SHALL 維持する。
- `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/recovery/consume`、recovery branch の `POST /api/v1/auth/passkey/register` に対する失敗は共有 failure counter を MUST 加算し、15 分窓で 10 失敗に達した主体を 15 分間 temporary lock しなければならない。
- throttle counter と temporary lock state は Valkey-backed auth state store に MUST 保持され、all nodes で共有される。
- throttle counter record、temporary lock record、auth abuse event record、解除ジョブ参照 ID など guardrail state が持つ ID は ULID を SHALL 使用する。ただし email / IP 由来の bucket key 自体は resource ID ではないため ULID 変換対象に含めない。
- temporary lock 中の guarded request は no-store boundary を保ったまま reject され、account existence、invite-only state、recovery-only state の有無を外部へ漏らしてはならない。
- throttle / temporary lock reject は `unauthenticated` / `session-expired` / `internal-error` に新しい公開 stable error code を追加してはならず、non-revealing auth reject として扱わなければならない。

#### Scenario: throttled recovery request は generic accepted response を維持する (AUTH-BE-S011)

- **GIVEN** client が configured budget を超えて `POST /api/v1/auth/recovery` を繰り返している
- **WHEN** system が recovery throttle を適用する
- **THEN** system は同一の accepted / no-store response shape を維持し、登録済み account の有無や throttle hit を外部へ露出しない

#### Scenario: throttled passkey start は no-store かつ non-revealing に reject される (AUTH-BE-S013)

- **GIVEN** client が同じ `account-or-handle + IP` で `POST /api/v1/auth/passkey/start` を configured budget 超過まで繰り返している
- **WHEN** system が `passkey/start` throttle を適用する
- **THEN** system は追加 challenge を発行せず no-store boundary を保ったまま request を reject し、新しい公開 stable error code や account state を露出しない

#### Scenario: repeated auth failures は temporary lock を発動する (AUTH-BE-S012)

- **GIVEN** `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/recovery/consume`、または recovery branch の `POST /api/v1/auth/passkey/register` に対する失敗が configured window 内で累積している
- **WHEN** client が temporary lock 期間中に guarded endpoint を再試行する
- **THEN** system は temporary lock として request を reject し、challenge completion、recovery consume、passkey re-registration を進めない

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
