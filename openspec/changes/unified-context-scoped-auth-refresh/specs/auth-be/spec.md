## MODIFIED Requirements

### Requirement: リフレッシュトークンは設定可能な TTL で管理される

リフレッシュトークンは設定可能な TTL で管理されなければならない（SHALL）。TTL validation logic は Product account auth と Admin operator auth から共通利用できる中立 primitive でなければならない（SHALL）。TTL primitive は account / operator の domain decision を所有してはならない（MUST NOT）。TTL 付き refreshToken state は各認証ドメインの Valkey logical DB と key prefix に保存されなければならない（MUST）。refreshToken Cookie の `Max-Age` または `Expires` は server-side refreshToken state の TTL を超えてはならない（MUST NOT）。Access token TTL は Product と Admin のどちらも短命でなければならず、15 分相当を標準値として扱い、通常の protected request で sliding extension してはならない（MUST NOT）。

**Customer Context**

運用者は Product と Admin の refreshToken / accessToken の寿命を同じ安全基準で管理したい。片方だけ長すぎる TTL や通常 request による延命を許すと、漏えい時の被害範囲が surface ごとに不均一になる。一方、TTL helper が account status や operator role を判断すると Product/Admin 境界が崩れる。

#### Scenario: refreshToken Cookie の寿命は server-side TTL を超えない (AUTH-BE-S064)

- **GIVEN** refresh token TTL が 30 日に設定されている
- **WHEN** login または refresh rotation が refreshToken Cookie を設定する
- **THEN** Cookie の `Max-Age` または `Expires` は server-side refreshToken state の TTL 以下である

#### Scenario: Product と Admin は同じ中立 TTL validation を使う (AUTH-BE-S065)

- **GIVEN** refresh token TTL または access token TTL が許容範囲外に設定されている
- **WHEN** Product API binary または Admin API binary が起動時 config validation を実行する
- **THEN** どちらの binary も同じ中立 TTL validation rule で fail-close する
- **AND** TTL helper は account status、operator role、RBAC、CSRF を判定しない

#### Scenario: protected request は access token TTL を延長しない (AUTH-BE-S082)

- **GIVEN** account が有効な short-lived accessToken を持っている
- **WHEN** account が Product protected `/api/v1/*` endpoint を複数回呼び出す
- **THEN** backend は accessToken の issuedAt / expiresAt を延長せず、login、setup/register、または context refresh flow だけが新しい accessToken を発行する

### Requirement: システムは複数の active session を同時に保持・管理できる

システムは同一 browser 上で複数 account session を保持できなければならない（SHALL）。各 Product account session は accessToken、authContextId、account session ID、account ID、server-side refreshToken state、refreshToken Cookie Path binding を持たなければならない（SHALL）。refreshToken が HttpOnly Cookie であるため、クライアントは refreshToken 平文を保持してはならない（MUST NOT）。複数 session の refresh は、URL path の `authContextId` と path-scoped refresh Cookie の所属を Product account auth domain が検証して対象 session だけを rotation しなければならない（MUST）。logout や session revoke は対象 session の accessToken metadata と refreshToken state / Cookie を失効させ、他 session に影響してはならない（MUST NOT）。

**Customer Context**

複数アカウントを扱う利用者は、account を切り替えながら作業したい。一方で refreshToken を JavaScript から読める形で保持すると XSS 時の被害が大きい。path-scoped refresh Cookie と authContextId を組み合わせ、複数 session と token 窃取防止を両立する。

#### Scenario: 複数 session の context refresh は対象 session だけを rotation する (AUTH-BE-S066)

- **GIVEN** browser が account A と account B の active session を持っている
- **WHEN** account A の `authContextId` で `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** account A の refreshToken state と path-scoped refresh Cookie binding だけが rotation される
- **AND** account B の session は維持される

#### Scenario: refresh path と credential 所属が一致しない request は拒否される (AUTH-BE-S083)

- **GIVEN** request path の `authContextId` が account A を指し、提示された refresh credential record が account B に属している
- **WHEN** context refresh endpoint が request を検証する
- **THEN** backend は fail-close で request を拒否し、新しい accessToken または refresh credential を発行しない

#### Scenario: 単一セッションのログアウトは他のセッションに影響しない (AUTH-BE-S042)

- **GIVEN** 利用者がアカウント A とアカウント B の両方で active セッションを持っている
- **WHEN** アカウント A の accessToken で `POST /api/v1/auth/logout` を実行する
- **THEN** accessToken claims が示すアカウント A の session と対応 refresh credential は失効し、アカウント B の session は引き続き有効である

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

Product account 認証ドメインは、request の `credentialMode` に応じて Product account session credential を SHALL 発行しなければならない。Browser Cookie mode では short-lived accessToken、authContextId、account/session metadata を response body に返し、長寿命 refresh credential を `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth/contexts/{authContextId}/refresh` Cookie として設定しなければならない（MUST）。Cookie mode response body は refreshToken 平文を含んではならない（MUST NOT）。Bearer mode では API / mobile / CLI / SDK 向けに accessToken と refreshToken を response body で発行し、Cookie を設定してはならない（MUST NOT）。Product protected routes は `Authorization: Bearer <accessToken>` のみを account/session credential として使い、Cookie credential、refresh Cookie、`X-Auth-Context-Id` header、CSRF token を protected route の認可材料にしてはならない（MUST NOT）。Product Cookie-setting flow と Cookie mode context refresh は allowed Origin、Fetch Metadata、CORS credential policy、SameSite=Lax、no-store response を fail-close で検証しなければならない（MUST）。Cookie `Path` は refresh credential selection helper であり認可境界ではないため、backend は refresh token record の authContextId、session、family、surface、cookiePath と request path を必ず照合しなければならない（MUST）。

Admin operator 認証ドメインは Product account 認証ドメインとは別に、operator accessToken と operator refreshToken で構成される Admin operator session を SHALL 発行する。両認証ドメインが共有できるのは、HMAC/JWT signer/verifier、opaque token hash、Cookie 属性 helper、Cookie path construction、Cookie clear command、ULID/JTI validation、TTL validation helper、failure normalization など中立 primitive に限られる（MUST）。中立 primitive は account / operator の domain enum switch、issuer/audience/domain pairing、RBAC、account status、operator active state を所有してはならない（MUST NOT）。単一共有 token service に `identityDomain=account|operator` の切替引数を渡して Product/Admin の domain decision を畳み込んではならない（MUST NOT）。

**Customer Context**

Product 利用者の account 認証と Admin 運営者の operator 認証は守る対象と失敗時の影響が異なる。Cookie 属性や署名検証のような安全な primitive は共通化してよいが、account status、operator role、session state を単一 service の切替で扱うと境界が曖昧になり、誤認可や監査漏れにつながる。

#### Scenario: Web Cookie mode の Product passkey login は accessToken body と path-scoped refresh Cookie を返す (AUTH-BE-S060)

- **GIVEN** account が Product passkey authentication を開始している
- **WHEN** valid credential で `credentialMode="cookie"` の `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** response body は short-lived accessToken、authContextId、Product account/session metadata を含む
- **AND** response body は refreshToken 平文を含まない
- **AND** `Set-Cookie` は refresh credential を `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth/contexts/{authContextId}/refresh` で設定する

#### Scenario: Product Cookie-setting flow は許可 Origin だけを受け入れる (AUTH-BE-S088)

- **GIVEN** request が Product Cookie mode login finish、registration finish、または context refresh を呼び出している
- **WHEN** request の `Origin` が Product frontend allowlist と完全一致しない
- **THEN** backend は Cookie 設定または refresh rotation を実行せず、fail-close response を返す

#### Scenario: Product Cookie-setting flow は cross-site Fetch Metadata を拒否する (AUTH-BE-S089)

- **GIVEN** request が Cookie を設定または rotation する Product auth endpoint を呼び出している
- **WHEN** `Sec-Fetch-Site` が `cross-site` である
- **THEN** backend は request を拒否し、refreshToken Cookie を設定・更新しない
- **AND** Fetch Metadata が欠落する場合は allowlist と完全一致する `Origin` を要求する

#### Scenario: Cookie Path は認可境界として扱われない (AUTH-BE-S090)

- **GIVEN** request path の `authContextId` と browser が送信した refresh Cookie が存在する
- **WHEN** refresh token record の authContextId、session、surface、cookiePath が request path と一致しない
- **THEN** backend は Cookie Path だけを信用せず fail-close で request を拒否する

#### Scenario: Product protected route は Bearer accessToken だけを認可材料にする (AUTH-BE-S084)

- **GIVEN** request が Product protected `/api/v1/passkeys` endpoint を呼び出している
- **WHEN** request が refresh Cookie だけ、または browser が自動送信した Cookie だけを提示する
- **THEN** backend は account/session credential 不在として拒否し、Cookie を protected route の認可材料にしない

#### Scenario: Product protected route は X-Auth-Context-Id と CSRF を要求しない (AUTH-BE-S085)

- **GIVEN** request が有効な `Authorization: Bearer <accessToken>` を持っている
- **WHEN** request が state-changing Product protected endpoint を呼び出す
- **THEN** backend は accessToken claims と server-side session record で account/session/authContext を束縛し、`X-Auth-Context-Id` header または CSRF header の欠落を理由に拒否しない

#### Scenario: Product と Admin の認証ドメインは単一 switch に畳み込まれない (AUTH-BE-S067)

- **WHEN** accessToken 発行、refresh rotation、session revoke の implementation を確認する
- **THEN** Product account auth は Product account auth domain/application の型と service を使う
- **AND** Admin operator auth は Admin operator auth domain/application の型と service を使う
- **AND** `identityDomain` などの引数で account/operator domain decision を切り替える単一共有 token service は存在しない

#### Scenario: 中立 token primitive は account/operator domain switch を持たない (AUTH-BE-S068)

- **WHEN** shared token primitive の public API と内部実装を確認する
- **THEN** 署名、検証、opaque token hash、Cookie path construction、Cookie clear command、ULID/JTI validation、TTL validation、failure normalization だけを扱う
- **AND** account / operator enum、RBAC、status 判定、issuer/audience/domain pairing を持たない

#### Scenario: Product AccountAuth domain が account token eligibility を所有する (AUTH-BE-S069)

- **GIVEN** account が suspended または sessionRevokedAfter より古い session を持つ
- **WHEN** Product account accessToken 発行または refresh rotation を行う
- **THEN** Product AccountAuth domain object は token eligibility を拒否する

#### Scenario: Admin OperatorAuth domain が operator token eligibility を所有する (AUTH-BE-S070)

- **GIVEN** operator が inactive または権限不足である
- **WHEN** Admin operator accessToken 発行、refresh rotation、protected route validation を行う
- **THEN** Admin OperatorAuth domain object は token eligibility を拒否し、Product AccountAuth domain object は operator eligibility 判定に使われない

#### Scenario: Product auth application は Admin auth application を import しない (AUTH-BE-S071)

- **WHEN** `internal/application/product/auth` が `internal/application/admin` または Admin OperatorAuth application を import している
- **THEN** lint または import-boundary test は失敗する

#### Scenario: Admin auth application は Product auth application を import しない (AUTH-BE-S072)

- **WHEN** `internal/application/admin/auth` が `internal/application/product` または Product AccountAuth application を import している
- **THEN** lint または import-boundary test は失敗する

#### Scenario: Cookie mode refresh は path-scoped refresh Cookie を rotation する (AUTH-BE-S062)

- **GIVEN** クライアントが有効な refreshToken Cookie を持つ
- **WHEN** クライアントが `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** 対象 auth domain は URL path の authContextId と refresh Cookie 所属を検証し、旧 refreshToken を原子消費する
- **AND** response body は short-lived accessToken、authContextId、session metadata を含み、refreshToken 平文を含まない
- **AND** Web Cookie mode の refreshToken は同じ Path の `Set-Cookie` で rotation される

#### Scenario: 同一 context の同時 refresh は単一成功だけを許可する (AUTH-BE-S091)

- **GIVEN** 同一 authContextId の active refreshToken に対して複数の refresh request が並行している
- **WHEN** backend が refresh token family を rotation する
- **THEN** 1 件だけが atomic consume + issue に成功し、old token と new token が同時に valid になる状態を作らない
- **AND** 敗者 request は idempotent grace の明示条件を満たさない限り replay/theft signal として扱われる

#### Scenario: Bearer mode refresh は body refreshToken を rotation する (AUTH-BE-S086)

- **GIVEN** external Bearer client が body に有効な refreshToken を持っている
- **WHEN** client が `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** backend は body refreshToken と path authContextId の所属を検証し、response body に新しい accessToken と refreshToken を返す
- **AND** backend は Cookie を設定しない

#### Scenario: refresh endpoint は Authorization header を拒否する (AUTH-BE-S080)

- **GIVEN** refresh request が `Authorization: Bearer <accessToken>` header を持っている
- **WHEN** request が `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** backend は Authorization header を refresh credential として扱わず、fail-close で request を拒否する

#### Scenario: refresh Cookie と body refreshToken の同時提示は拒否される (AUTH-BE-S087)

- **GIVEN** request が path-scoped refresh Cookie と body `refreshToken` の両方を持っている
- **WHEN** request が `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** backend は refresh credential ambiguity として request を拒否し、新しい credential を発行しない

#### Scenario: logout は対象 refresh Cookie path の clear command を返す (AUTH-BE-S092)

- **GIVEN** account が複数 authContextId の active session を持っている
- **WHEN** account が active accessToken で `POST /api/v1/auth/logout` を呼び出す
- **THEN** backend は accessToken claims が示す session の refresh family を revoke する
- **AND** response は対象 `Path=/api/v1/auth/contexts/{authContextId}/refresh` の refresh Cookie を削除する Set-Cookie command を返す
- **AND** 他 authContextId の refresh Cookie と session は削除しない

#### Scenario: ブラウザーから読める refreshToken は発行されない (AUTH-BE-S063)

- **GIVEN** login、refresh、recovery registration、または operator login が Cookie mode で成功する
- **WHEN** response body と log/trace attributes を確認する
- **THEN** refreshToken の平文値は body、log、trace attribute、error message に存在しない

#### Scenario: suspended account は新規 accessToken を発行されない (AUTH-BE-S054)

- **GIVEN** account の `accounts.status` が `suspended` である
- **WHEN** account が valid passkey で `POST /api/v1/auth/passkey/finish` を完了しようとする
- **THEN** システムは accessToken / refresh credential を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の existing bearer accessToken は拒否される (AUTH-BE-S055)

- **GIVEN** account が active session を持っていた後に Admin Console で suspended になっている
- **WHEN** その accessToken で `/api/v1/*` にアクセスする
- **THEN** システムは session credential を認可せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の refresh は rotation されない (AUTH-BE-S058)

- **GIVEN** account が valid refreshToken Cookie または Bearer refreshToken を持っていた後に Admin Console で suspended になっている
- **WHEN** クライアントが `POST /api/v1/auth/contexts/{authContextId}/refresh` を呼び出す
- **THEN** システムは新しい accessToken / refresh credential を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspend は account-wide session revocation timestamp を書き込む (AUTH-BE-S056)

- **GIVEN** Admin Console が account suspend を成功させる
- **WHEN** DB の `accounts.session_revoked_after` を確認する
- **THEN** suspend 時刻以上の timestamp が保存され、その timestamp 以前に発行された accessToken と refreshToken family は拒否される

#### Scenario: restored account は過去 session では復帰できない (AUTH-BE-S057)

- **GIVEN** account が suspended 後に restore されている
- **WHEN** suspend 前に発行された accessToken または refresh credential で `/api/v1/*` または refresh endpoint にアクセスする
- **THEN** システムは session credential を拒否し、account は再ログインでのみ新しい session credential を取得できる

#### Scenario: account-suspended は stable failure response shape で返される (AUTH-BE-S059)

- **GIVEN** suspended 判定が `POST /api/v1/auth/passkey/finish`、context refresh、または bearer-protected `/api/v1/*` endpoint で発生する
- **WHEN** システムが response を返す
- **THEN** HTTP status は 403 であり、body は `AuthFailureResponse` の `{ requestId, error: "account-suspended" }` である
- **AND** response は `Cache-Control: no-store` を含む

#### Scenario: session を持たない request は session-expired と混同されない (AUTH-BE-S009)

- **GIVEN** request が `/api/v1/*` を対象にしている
- **WHEN** request が Product accessToken をまったく提示しない
- **THEN** システムは unauthenticated failure として拒否し、expired / revoked session 用の failure と区別する

#### Scenario: auth state store unavailable は fail-close で internal-error になる (AUTH-BE-S010)

- **GIVEN** request が auth boundary を通って `/api/v1/*` または auth endpoint を呼んでいる
- **WHEN** Valkey を含む auth state store が unavailable で session / challenge / recovery state を安全に検証できない
- **THEN** システムは fail-close で request を拒否し、stable classification `internal-error` を返す

#### Scenario: 消費済みリフレッシュトークンの再利用は拒否され関連トークンを失効する (AUTH-BE-S044)

- **GIVEN** リフレッシュトークンが既に消費されている
- **WHEN** 同じ旧リフレッシュトークンで context refresh を再試行する
- **THEN** システムは request を拒否し、同一アカウント・同一デバイス指紋のすべてのリフレッシュトークンを失効させる

#### Scenario: 不正なリフレッシュトークンは拒否される (AUTH-BE-S045)

- **GIVEN** クライアントが存在しないまたは改竄されたリフレッシュトークンを提示している
- **WHEN** クライアントが context refresh endpoint を呼び出す
- **THEN** システムは request を replay/theft signal として拒否し、新しい session credential を発行しない
- **AND** request path の authContextId、server-side session record、または device fingerprint から特定できる同一 surface + identity + refresh family を revoke する

#### Scenario: accessToken の有効期限切れは session-expired として拒否される (AUTH-BE-S046)

- **GIVEN** クライアントが有効期限切れの JWT accessToken を保持している
- **WHEN** そのトークンを `Authorization: Bearer` ヘッダーに設定して `/api/v1/*` を呼び出す
- **THEN** システムは `session-expired` failure として拒否する
