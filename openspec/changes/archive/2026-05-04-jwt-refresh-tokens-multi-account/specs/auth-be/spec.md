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
