## MODIFIED Requirements

### Requirement: recovery token は単回利用・期限付きで enumeration-safe に扱う

recovery token は、単回利用・期限付き・enumeration-safe なパスキー追加用 credential として SHALL 扱われなければならない。

**Customer Context**

パスキー紛失時の復旧と新端末追加は、どちらも登録済みメールアドレスへの URL 送信によってパスキー再登録を可能にする必要がある。同時にこれらの導線は、アカウント有無や token 状態を推測できないように保護されなければならない。新端末追加は Web Cookie session と Bearer session のどちらからでも利用できるが、どちらの credential で認証した場合でも fresh な再認証を要求し、session-only の request では token を発行してはならない。

**Requirement**

- システムは recovery token と device-link token を単回利用・期限付き credential として扱い、発行 request の response は account existence を推測できない形に保たなければならない（SHALL/MUST）。
- Token は `kind` フィールドを持ち、`"recovery"`（パスキー紛失時の復旧）または `"device-link"`（認証済み端末からの新端末追加）のいずれかを MUST 指定する。
- システムは `POST /api/v1/auth/recovery` を提供し、登録済みメールアドレスを受け取って `kind=recovery` の単回利用・期限付き token を発行し、登録済みアドレスへ recovery URL を SMTP で送信しなければならない（SHALL）。
- システムは `POST /api/v1/passkeys/send-device-link` を提供し、`X-Reauth-Session` header（operation kind `device-link`）付きの有効なアプリケーションセッションを受け取って `kind=device-link` の単回利用・期限付き token を発行し、登録済みメールアドレスへ device-link URL を SMTP で送信しなければならない（SHALL）。この endpoint は Web Cookie credential または `Authorization: Bearer` のどちらか一方だけを認証済み session credential source として受け入れなければならず（MUST）、消費済み reauthentication session を要求し、session-only request を拒否しなければならない（MUST/SHALL）。
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

#### Scenario: Cookie session は reauthentication と合わせて device-link token を発行できる (AUTH-BE-S073)

- **GIVEN** 認証済み Web クライアントが valid な Product auth Cookie と `device-link` 用 reauthentication session を持っている
- **WHEN** クライアントが `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** システムは Cookie session と reauthentication session の account/session binding を検証し、`kind=device-link` の token を発行する

### Requirement: recovery register branch は既存アカウントの再登録だけを許可する

recovery/device-link register branch は、valid な RecoverySession が指す既存アカウントの passkey 登録だけを SHALL 許可し、invite や consent state を MUST 受け入れてはならない。登録完了後はセッションの kind と credential mode に応じた後処理（セッション失効・通知・session credential 発行）を実行する。

**Customer Context**

Auth コアが扱うのは既存アカウントの passkey 回復・追加であり、招待登録や規約同意や Guest state 変更ではない。この境界が崩れると `/login/recovery/*` と `/invite/*` が混線し、Auth と Invitation の責務が不明確になる。また、Web 利用者は登録完了後に JavaScript 可読 token を受け取らず安全にログイン状態へ進み、API / mobile / CLI / SDK 利用者は明示的に Bearer session を受け取れる必要がある。

**Requirement**

- システムは register branch を valid な RecoverySession が参照する既存アカウントに対してだけ動作させ、shared register endpoint は recovery state と invite state の exactly-one selector boundary を保たなければならない（SHALL/MUST）。
- `POST /api/v1/auth/passkey/register` は shared endpoint として RecoverySession または InvitationSession の exactly-one を MUST 要求し、register branch は valid な RecoverySession のみを持つときだけ SHALL 成立する。
- register branch は RecoverySession が指す既存アカウントへ新しい passkey を SHALL 登録し、new Account 作成、Guest / Member state 変更、base role 変更をしてはならない。
- register branch は invitation-session validation、invite-token consume、invite consent completion、TermsConsent read / write を MUST NOT 要求しない。
- RecoverySession と InvitationSession を同時に提示する request、または両方を欠く request は branch ambiguity として MUST reject しなければならない。
- register branch が成功した後は、request の `credentialMode` に従って有効なアプリケーションセッションを SHALL 返す。
- `credentialMode="web-cookie"` の register response は access credential と refresh credential を HttpOnly Cookie で設定し、response body には bearer accessToken または refreshToken 平文を含めてはならない（MUST NOT）。body は session metadata と CSRF token を含まなければならない（MUST）。
- `credentialMode="bearer"` の register response は Bearer accessToken と refreshToken を body に含め、Web auth Cookie を設定してはならない（MUST NOT）。
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
- **THEN** システムは既存アカウントへ新しい passkey を登録し、該当アカウントの全既存セッションを強制失効し、復旧完了通知メールを送信し、request の `credentialMode` に対応する有効な session を返す

#### Scenario: device-link session (kind=device-link) は passkey を追加しセッションは失効しない (AUTH-BE-S048)

- **GIVEN** 利用者が kind=device-link の valid な RecoverySession を保持している
- **WHEN** 利用者が register branch として `POST /api/v1/auth/passkey/register` を送信する
- **THEN** システムは既存アカウントへ新しい passkey を登録し、既存セッションを一切失効せず、新端末追加完了通知メールを送信し、request の `credentialMode` に対応する有効な session を返す

#### Scenario: invite-only state では registration を完了できない (AUTH-BE-S008)

- **GIVEN** request が valid な RecoverySession なしで registration を試みる
- **WHEN** request が invite 向け state のみ、TermsConsent のみ、または利用可能な recovery state を持たない
- **THEN** システムは registration を拒否し、`/login/recovery/*` と `/invite/*` の分離を維持する

### Requirement: 認証済みアカウントは複数のパスキーを登録・管理できる

認証済みアカウントは、Web Cookie または Bearer の exactly-one session credential source によって、複数の passkey credential を登録・一覧・削除できなければならない（SHALL）。

**Customer Context**

利用者は MacBook・iPhone・セキュリティキーなど複数のデバイスでパスキーを使い分けたい。新しいデバイスを登録しても他のデバイスのアクセスが失われないことが求められる。複数のパスキーを独立して管理できることで、デバイス追加・紛失後の安全な鍵ローテーションが可能になる。Web 利用者は Cookie session で安全に操作し、外部クライアントは Bearer session で同じ API capability を利用できる必要がある。

**Requirement**

- 認証済みアカウントは 1 件以上の passkey credential を持つことができ、システムはすべての active な passkey credential を保持しなければならない（SHALL）。
- システムは `GET /api/v1/passkeys` で認証済みアカウントの登録済みパスキー一覧（ID・識別子・登録日時）を返さなければならない（SHALL）。
- システムは `POST /api/v1/passkeys/start` で WebAuthn 追加登録チャレンジを発行し、`POST /api/v1/passkeys/finish` でチャレンジを検証して既存パスキーを保持したまま新しい passkey credential をアカウントへ追加しなければならない（SHALL）。
- システムは `DELETE /api/v1/passkeys/{id}` で指定した passkey credential を削除しなければならない（SHALL）。ただし、アカウントに残る passkey credential が 1 件になる場合は削除を拒否しなければならない（MUST）。
- 上記すべての管理エンドポイントは、exactly one authenticated session credential source を必須とする。Web Cookie credential または `Authorization: Bearer <session token>` のどちらか一方だけを受け入れ、両方を同時に提示した request は MUST reject する。
- Cookie credential を使う state-changing request は、session-bound CSRF token と許可済み Origin を MUST require する。Bearer credential だけを使う non-browser request は CSRF token を要求してはならない（MUST NOT）。
- 他のアカウントに属する passkey credential を操作する試みは SHALL 拒否されなければならない。
- パスキー管理操作で用いる resource ID（credential ID、challenge ID、correlation ID 等）は ULID を使用しなければならない（SHALL）。

#### Scenario: 登録済みパスキー一覧を取得できる (AUTH-BE-S014)

- **GIVEN** 認証済みアカウントが有効なアプリケーションセッションを持っている
- **WHEN** `GET /api/v1/passkeys` を呼び出す
- **THEN** システムはそのアカウントに紐づくすべての passkey credential のリストを返す

#### Scenario: 新しいパスキーを追加しても既存パスキーが保持される (AUTH-BE-S015)

- **GIVEN** 認証済みアカウントが有効なアプリケーションセッションを持っている
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

- **GIVEN** アカウント A が有効なアプリケーションセッションを持っている
- **WHEN** アカウント B に属する passkey credential の ID を指定して `DELETE /api/v1/passkeys/{id}` を呼び出す
- **THEN** システムはリクエストを拒否し、アカウント A のパスキーは変化しない

#### Scenario: 未認証リクエストはパスキー管理 API を利用できない (AUTH-BE-S019)

- **GIVEN** 有効なアプリケーションセッションを持たないリクエストがある
- **WHEN** `/api/v1/passkeys` 以下のいずれかのエンドポイントを呼び出す
- **THEN** システムは unauthenticated として拒否する

#### Scenario: Cookie と Bearer の同時提示は拒否される (AUTH-BE-S074)

- **GIVEN** request が Product auth Cookie と `Authorization: Bearer` header の両方を持っている
- **WHEN** request が `/api/v1/passkeys` 以下の endpoint を呼び出す
- **THEN** システムは credential ambiguity として request を拒否し、handler の state mutation を実行しない

### Requirement: WebAuthn ceremony は user verification を必須にする

Passkey は認証基盤の中核であり、端末所持だけでなく端末内の user verification によって利用者本人の操作であることを確認する必要がある。ログイン、新しい端末でのログイン有効化、復旧後の再登録などの高リスク操作で user verification が optional だと、端末盗難や弱い authenticator policy による不正利用リスクが残るため、システムは required user verification を強制しなければならない（MUST）。

システムは、Product Web Cookie session と Bearer session のどちらでも、WebAuthn ceremony と high-risk operation に required user verification と fresh reauthentication を強制しなければならない（MUST）。

**Requirement**

- システムは認証 ceremony と登録 ceremony で WebAuthn user verification を SHALL require する。
- システムは high-risk な認証済み passkey 管理操作の前に、fresh な WebAuthn reauthentication session を SHALL require する。
- `POST /api/v1/auth/passkey/start`、`POST /api/v1/auth/passkey/finish`、`POST /api/v1/passkeys/start`、`POST /api/v1/passkeys/finish`、`POST /api/v1/auth/passkey/register/start`、`POST /api/v1/auth/passkey/register` は user verification required semantics を MUST enforce する。
- `POST /api/v1/passkeys/send-device-link` と `DELETE /api/v1/passkeys/{id}` は有効なアプリケーションセッションだけで成立してはならず、`X-Reauth-Session` HTTP header で提示された同一 account/session に紐づく短命 reauthentication session を要求しなければならない（MUST）。
- Reauthentication session は Valkey-backed auth state store に TTL 付きで保持され、対象 account、issuing session、operation kind（`device-link` または `passkey-delete`）、request ID を紐づけなければならない。
- Reauthentication session は high-risk operation completion 時に atomic consume されるか、短い有効期限で失効しなければならない。
- 異なる operation kind の reauthentication session を使い回した場合は MUST reject する。
- client に返す WebAuthn options が `userVerification` field を表現する場合、値は `"required"` でなければならない。
- server-side WebAuthn verification は required user verification を満たさない assertion または attestation を拒否しなければならない。

#### Scenario: login ceremony は user verification を要求する (AUTH-BE-S028)

- **GIVEN** passkey login ceremony が開始されている
- **WHEN** authenticator response が required user verification を満たさない
- **THEN** システムは login を拒否し、application session を発行しない

#### Scenario: 新端末のログイン有効化は user verification を要求する (AUTH-BE-S029)

- **GIVEN** valid な device-link RecoverySession が存在する
- **WHEN** 新しい端末が required user verification なしで WebAuthn registration を完了しようとする
- **THEN** システムは registration を拒否し、account に credential を追加しない

#### Scenario: device-link delivery は fresh な再認証を要求する (AUTH-BE-S036)

- **GIVEN** account は有効なアプリケーションセッションを持つが fresh な reauthentication session を持たない
- **WHEN** account が device-link delivery を要求する
- **THEN** システムは request を拒否し、device-link token を発行または送信しない

#### Scenario: passkey deletion は fresh な再認証を要求する (AUTH-BE-S037)

- **GIVEN** account は有効なアプリケーションセッションを持つが fresh な reauthentication session を持たない
- **WHEN** account が登録済み passkey credential の削除を要求する
- **THEN** システムは削除を拒否し、すべての credential を変更しない

### Requirement: 認証済み端末から新端末追加用トークンを発行できる

認証済み端末は、既存パスキーによる再認証を完了した後、登録済みメールアドレスへ新端末追加用の単回利用 URL トークンを SHALL 発行する。トークンは kind=device-link として管理され、消費後に新端末でのパスキー登録を可能にする。

デバイスリンク発行は、exactly-one の有効なアプリケーション session credential と fresh reauthentication session を検証した場合にのみ成立しなければならない（MUST）。

**Customer Context**

利用者が新しい端末でログインできるようにしたい場合、認証済み端末から安全に新端末追加リンクを発行できる必要がある。Web app では Cookie session によって browser-readable token を避けながら操作でき、API / mobile / CLI / SDK では Bearer session によって同じ capability を利用できる必要がある。

**Requirement**

- `POST /api/v1/passkeys/send-device-link` は認証済み端末から新端末追加用の device-link token を発行し、登録済みメールアドレスへ送信しなければならない（SHALL）。このエンドポイントは exactly one の有効なアプリケーション session credential と `X-Reauth-Session` header で提示された operation kind `device-link` の reauthentication session を要求しなければならず（MUST）、session-only では成立してはならない。
- device-link token は `kind=device-link` で RecoveryToken として管理され、既存の RecoveryToken と同一のライフサイクル（発行・ハッシュ保存・原子消費・TTL 管理）を SHALL 共有する。
- device-link token の有効期限は発行から 30 分とし、Valkey-backed auth state store に HMAC-SHA256 + pepper でハッシュ化して保存されなければならない（MUST）。
- `POST /api/v1/passkeys/send-device-link` の response は `Cache-Control: no-store` を SHALL 保つ。
- メール送信の失敗は account existence を露出せず、accepted response を維持しなければならない。

#### Scenario: 認証済み端末からデバイスリンクを送信できる (AUTH-BE-S049)

- **GIVEN** 認証済みアカウントが有効なアプリケーションセッションを持ち、device-link 用 reauthentication session を保持している
- **WHEN** `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** システムは kind=device-link の token を発行し、新端末追加用 URL を含むメールを登録メールアドレスへ送信し、`{issued: true}` を返す

#### Scenario: reauthentication なしではデバイスリンクを発行できない (AUTH-BE-S050)

- **GIVEN** 認証済みアカウントが有効なアプリケーションセッションを持つが reauthentication session を持たない
- **WHEN** `POST /api/v1/passkeys/send-device-link` を呼び出す
- **THEN** システムは request を拒否し、token を発行しない

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

パスキー認証は、`credentialMode` に応じて Web Cookie session または Bearer-compatible application session を SHALL 発行し、logout で MUST revoke しなければならない。Product account 認証ドメインは、Web Cookie mode では短命な account access credential と長寿命 account refresh credential を HttpOnly Cookie として発行し、Bearer mode では API / mobile / CLI / SDK 向けに `Authorization: Bearer <access token>` で `/api/v1/*` を利用できる accessToken / refreshToken を response body で発行する。DB の `accounts.status='suspended'` の account に対しては、新規 session credential 発行、refresh rotation、既存 session 認可を MUST 拒否する。

Product account 認証ドメインは Admin operator session を発行・検証・更新・失効してはならない（MUST NOT）。Admin operator 認証ドメインは Product account 認証ドメインとは別の identity domain として扱われ、operator access Cookie、operator refresh Cookie、Admin CSRF binding、Admin Valkey logical DB、`admin:*` key prefix を Product account session state と混同してはならない（MUST NOT）。Admin operator auth は Product account auth domain/application を import してはならず（MUST NOT）、Product account auth は Admin operator auth domain/application を import してはならない（MUST NOT）。

両認証ドメインが共有できるのは、HMAC/JWT signer/verifier、opaque token hash、Cookie 属性 helper、ULID/JTI validation、TTL validation helper など中立 primitive に限られる（MUST）。中立 primitive は account / operator の domain enum switch、issuer/audience/domain pairing、RBAC、account status、operator active state、CSRF binding を所有してはならない（MUST NOT）。単一共有 token service に `identityDomain=account|operator` の切替引数を渡して Product/Admin の domain decision を畳み込んではならない（MUST NOT）。

**Customer Context**

Web 利用者と Admin 運営者は XSS の影響を受けにくい HttpOnly Cookie session でアプリを使い続ける必要がある。一方、Product 外部クライアントは Cookie に依存しない Bearer token を必要とする。両方の credential を同時に受け入れると session 選択が曖昧になり、CSRF や token replay の検出が不安定になるため、session credential source は request ごとに exactly one でなければならない。さらに Product 利用者の account 認証と Admin 運営者の operator 認証は、守る対象と失敗時の影響が異なるため、どちらも browser surface では Cookie-only にしつつ domain の分離を崩してはならない。

**Requirement**

- システムは passkey authentication が成功したとき、request の `credentialMode` に応じたアプリケーションセッションを発行しなければならない（SHALL）。
- `credentialMode="web-cookie"` の session は、short-lived access credential と refresh credential を HttpOnly Cookie として設定し、response body に bearer accessToken または refreshToken 平文を含めてはならない（MUST NOT）。response body は requestId、accountId、passkeyCredentialId、sessionId、expiresAt、CSRF token を含まなければならない（MUST）。
- `credentialMode="bearer"` の session は、short-lived JWT accessToken と refreshToken を response body に含め、Web auth Cookie を設定してはならない（MUST NOT）。
- Web auth Cookie は Secure、HttpOnly、SameSite=Lax 以上、限定 Path、適切な Max-Age を SHALL 持つ。refresh Cookie は refresh endpoint に必要な Path だけで送信されなければならない（MUST）。
- `POST /api/v1/auth/refresh` は `credentialMode="web-cookie"` では HttpOnly refresh Cookie を rotation し、新しい access Cookie、refresh Cookie、CSRF token を返す。body に accessToken または refreshToken 平文を含めてはならない（MUST NOT）。
- `POST /api/v1/auth/refresh` は `credentialMode="bearer"` では request body の refreshToken を rotation し、新しい accessToken と refreshToken を body で返す。Web auth Cookie を設定してはならない（MUST NOT）。
- refresh token は単回利用でなければならず（SHALL）、rotation で一度 consumed になった旧 refresh token は以後の試行で拒否しなければならない（MUST）。
- consumed 済みまたは unknown の refresh token が `POST /api/v1/auth/refresh` に提示された場合、システムは token theft の可能性として扱い、同じ account と device/session fingerprint に紐づくすべての refresh token を revoke しなければならない（SHALL/MUST、fail-close rotation failure）。
- `/api/v1/*` surface は public auth endpoint と status endpoint を除き、exactly one active session credential source を MUST 要求する。Web Cookie credential と `Authorization: Bearer` credential の両方が提示された request は MUST reject する。
- request が session credential をまったく持たない場合は stable classification `unauthenticated` として SHALL 扱われ、expired / revoked session failure と混同してはならない。
- expired / revoked session credential が提示された場合は stable classification `session-expired` として SHALL 扱われる。
- Cookie credential を使う state-changing request は許可済み Origin と session-bound CSRF token を要求しなければならない（MUST）。CSRF token が欠落または session と一致しない場合、システムは handler の state mutation 前に request を拒否しなければならない（MUST）。
- Bearer credential だけを使う request は CSRF token を要求してはならない（MUST NOT）。
- `POST /api/v1/auth/logout` は credential source に対応する active session を revoke し、Cookie session の場合は auth Cookie を server response で削除しなければならない（SHALL）。
- auth start / finish / `POST /api/v1/auth/logout` / `POST /api/v1/auth/refresh` response は `Cache-Control: no-store` を SHALL 保ち、cacheable な auth response を返してはならない。
- Product account auth は Admin operator auth domain/application から分離され、Admin operator auth は Product account auth domain/application を import してはならない（MUST NOT）。
- Product / Admin auth が共有できる token helper は中立 primitive に限られ、account / operator domain decision、RBAC、status 判定、CSRF binding を単一共有 token service の switch に畳み込んではならない（MUST NOT）。

#### Scenario: パスキーログイン成功時に JWT access token と refresh token を作成する (AUTH-BE-S001)

- **GIVEN** active な account が valid な passkey credential を持つ
- **WHEN** クライアントが `credentialMode="bearer"` で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** システムは以降の `/api/v1/*` access を `Authorization: Bearer <access token>` で認可できる有効な session を返し、refreshToken を body で返す

#### Scenario: Web Cookie mode のログインは body token を返さない (AUTH-BE-S060)

- **GIVEN** active な account が valid な passkey credential を持つ
- **WHEN** Web クライアントが `credentialMode="web-cookie"` で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** システムは HttpOnly access Cookie と HttpOnly refresh Cookie を設定し、response body には bearer accessToken と refreshToken 平文を含めず、CSRF token と session metadata だけを返す

#### Scenario: Web Cookie mode はブラウザーから読める token を発行しない (AUTH-BE-S063)

- **GIVEN** login、refresh、recovery registration、または passkey registration が `credentialMode="web-cookie"` で成功する
- **WHEN** response body と log/trace attributes を確認する
- **THEN** bearer accessToken と refreshToken の平文値は body、log、trace attribute、error message に存在せず、body には session metadata と CSRF token だけが含まれる

#### Scenario: Product auth は Admin operator Cookie login を処理しない (AUTH-BE-S061)

- **GIVEN** operator が Admin passkey authentication を開始している
- **WHEN** valid operator credential で Admin auth finish を完了する
- **THEN** Product account auth handler、Product AccountAuth session、Product application service は使用されない
- **AND** Admin operator auth domain/application と Admin generated bindings が operator Cookie session を処理する

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

#### Scenario: Admin OperatorAuth domain の eligibility は Product AccountAuth に委譲されない (AUTH-BE-S070)

- **GIVEN** operator が inactive、または CSRF token が operator session と一致しない
- **WHEN** Admin operator access Cookie 発行、refresh rotation、protected mutation validation を行う
- **THEN** Admin OperatorAuth domain object は token eligibility または CSRF binding を拒否する
- **AND** Product AccountAuth domain object は operator eligibility 判定に使われない

#### Scenario: Product auth application は Admin auth application を import しない (AUTH-BE-S071)

- **WHEN** `internal/application/product/auth` が `internal/application/admin` または Admin OperatorAuth application を import している
- **THEN** lint または import-boundary test は失敗する

#### Scenario: Admin auth application は Product auth application を import しない (AUTH-BE-S072)

- **WHEN** `internal/application/admin/auth` が `internal/application/product` または Product AccountAuth application を import している
- **THEN** lint または import-boundary test は失敗する

#### Scenario: 欠落または inactive な session は拒否される (AUTH-BE-S002)

- **GIVEN** request が active session credential を持たない
- **WHEN** request が protected `/api/v1/*` endpoint を呼び出す
- **THEN** システムは request を拒否し、protected resource を返さない

#### Scenario: logout は active session を revoke する (AUTH-BE-S003)

- **GIVEN** クライアントが有効なアプリケーションセッションを保持している
- **WHEN** クライアントが `POST /api/v1/auth/logout` を呼び出す
- **THEN** システムは対象 session と関連 refresh credential を revoke し、以降の利用を拒否する

#### Scenario: Cookie mutation は CSRF token を要求する (AUTH-BE-S075)

- **GIVEN** Web クライアントが valid な Product auth Cookie を持つが `X-CSRF-Token` を持たない
- **WHEN** クライアントが state-changing protected endpoint を呼び出す
- **THEN** システムは state mutation 前に request を拒否する

#### Scenario: Cookie と Bearer を同時に提示した protected request は拒否される (AUTH-BE-S076)

- **GIVEN** request が valid な Product auth Cookie と valid な `Authorization: Bearer` header の両方を持っている
- **WHEN** request が protected `/api/v1/*` endpoint を呼び出す
- **THEN** システムはどちらの session も選択せず credential ambiguity として request を拒否する

#### Scenario: suspended account は新規 session credential を発行されない (AUTH-BE-S054)

- **GIVEN** suspended account が valid passkey assertion を完了している
- **WHEN** システムが session credential を発行しようとする
- **THEN** システムは access credential / refresh credential を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の既存 bearer access token は拒否される (AUTH-BE-S055)

- **GIVEN** account が valid bearer access token を持っていた後に Admin Console で suspended になっている
- **WHEN** クライアントがその bearer access token で protected endpoint を呼び出す
- **THEN** システムは HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の Cookie session は拒否される (AUTH-BE-S077)

- **GIVEN** account が valid Product auth Cookie を持っていた後に Admin Console で suspended になっている
- **WHEN** Web クライアントがその Cookie で protected endpoint を呼び出す
- **THEN** システムは HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の refresh は rotation されない (AUTH-BE-S058)

- **GIVEN** account が valid refresh credential を持っていた後に Admin Console で suspended になっている
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を呼び出す
- **THEN** システムは新しい access credential / refresh credential を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspend は account-wide session revocation timestamp を書き込む (AUTH-BE-S056)

- **GIVEN** Admin Console が account suspend を成功させる
- **WHEN** DB の `accounts.session_revoked_after` を確認する
- **THEN** suspend 時刻以上の timestamp が保存され、その timestamp 以前に発行された Bearer accessToken、Cookie access credential、refresh credential は拒否される

#### Scenario: restored account は過去 session では復帰できない (AUTH-BE-S057)

- **GIVEN** account が suspended 後に restore されている
- **WHEN** suspend 前に発行された Bearer accessToken または Product auth Cookie で `/api/v1/*` にアクセスする
- **THEN** システムは session credential を拒否し、account は再ログインでのみ新しい session credential を取得できる

#### Scenario: account-suspended は stable failure response shape で返される (AUTH-BE-S059)

- **GIVEN** suspended 判定が `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/refresh`、または protected `/api/v1/*` endpoint で発生する
- **WHEN** システムが response を返す
- **THEN** HTTP status は 403 であり、body は `AuthFailureResponse` の `{ requestId, error: "account-suspended" }` である
- **AND** response は `Cache-Control: no-store` を含み、`AuthOperationErrorResponse` では返されない

#### Scenario: session を持たない request は session-expired と混同されない (AUTH-BE-S009)

- **GIVEN** クライアントが session credential を持たない
- **WHEN** request が protected `/api/v1/*` endpoint を呼び出す
- **THEN** システムは `unauthenticated` failure として拒否し、`session-expired` として扱わない

#### Scenario: auth state store unavailable は fail-close で internal-error になる (AUTH-BE-S010)

- **GIVEN** Valkey-backed auth state store が利用できない
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を送信する
- **THEN** システムは token を発行せず、`internal-error` として fail-close する

#### Scenario: Web Cookie refresh は Cookie credential と CSRF token を rotation する (AUTH-BE-S062)

- **GIVEN** クライアントが valid な HttpOnly refresh Cookie を持っている
- **WHEN** クライアントが `credentialMode="web-cookie"` で `POST /api/v1/auth/refresh` を送信する
- **THEN** システムは旧 refresh credential を atomically consumed とし、新しい HttpOnly access Cookie、HttpOnly refresh Cookie、CSRF token、session metadata を返す
- **AND** response body には bearer accessToken または refreshToken 平文を含めない

#### Scenario: 消費済みリフレッシュトークンの再利用は拒否され関連トークンを失効する (AUTH-BE-S044)

- **GIVEN** refresh credential がすでに rotation で consumed 済みである
- **WHEN** 同じ旧 refresh credential で `POST /api/v1/auth/refresh` を再試行する
- **THEN** システムは request を拒否し、同じ account と device/session fingerprint に紐づく refresh credential をすべて失効する

#### Scenario: 不正なリフレッシュトークンは拒否される (AUTH-BE-S045)

- **GIVEN** クライアントが unknown または tampered refresh credential を持っている
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を送信する
- **THEN** システムは request を拒否し、新しい access credential と refresh credential を発行しない

#### Scenario: access token の有効期限切れは session-expired として拒否される (AUTH-BE-S046)

- **GIVEN** Bearer access token または Cookie access credential が期限切れである
- **WHEN** クライアントがその credential で `/api/v1/*` を呼び出す
- **THEN** システムは `session-expired` failure として拒否する
