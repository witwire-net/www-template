## MODIFIED Requirements

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

パスキー認証は、既存の application session 契約を変更せず、`Authorization: Bearer <access token>` で `/api/v1/*` を利用できる短命 JWT access token と長寿命 refresh token のペアを SHALL 発行しなければならない。access token は `accountID`、`sessionID`、`iat`、`exp` を含む JWT であり、refresh token は account と device/session fingerprint に束縛され、`POST /api/v1/auth/refresh` で atomic consume と rotation を MUST 継続する。logout、refresh token 再利用検知、missing / expired / revoked session classification、`Cache-Control: no-store` は現行 auth-be 契約をそのまま保持しなければならない（MUST）。database の `accounts.status='suspended'` の account に対しては、新規 token pair 発行、refresh rotation、既存 bearer access token 認可を MUST 拒否する。拒否時は HTTP 403 と `AuthFailureResponse` body `{ requestId, error: "account-suspended" }` を MUST 返し、response は `Cache-Control: no-store` を SHALL 含む。`account-suspended` は `AuthFailureClassification` に追加し、`AuthOperationErrorResponse` では返してはならない（MUST NOT）。`POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/refresh`、bearer-protected `/api/v1/*` endpoint は suspended 判定用の 403 `AuthFailureResponse` を contract に含めなければならない（MUST）。Admin suspend が成功した場合、system は database の `accounts.session_revoked_after` に suspend 時刻を SHALL 永続化し、`session_revoked_after` 以前に発行された access token / refresh token を MUST 拒否する。restore は `session_revoked_after` を消去してはならず（MUST NOT）、account は再ログインでのみ新規 token pair を取得できる。`account-suspended` error は valid passkey assertion 後、refresh token 検証後、または既存 bearer access token 認可時のみ返し、public passkey start では account existence を漏らしてはならない（MUST NOT）。

**Customer Context**

Admin Console の account suspend は、オペレーターが不正利用やサポート対応で顧客アクセスを止めるための強権限操作である。停止済みアカウントが既存 access token や refresh token で API を使い続けられると、管理機能として成立しない。一方で public login start で suspended 状態を返すと account existence を推測できるため、状態開示は credential 所持または既存 token pair が確認できる境界に限定する。既存の token pair / refresh rotation 契約は顧客向け frontend と SDK の前提であるため、suspended 対応で置換してはならない。

#### Scenario: suspended account は新規 token pair を発行されない (AUTH-BE-S054)

- **GIVEN** account の `accounts.status` が `suspended` である
- **WHEN** account が valid passkey で `POST /api/v1/auth/passkey/finish` を完了しようとする
- **THEN** system は access token / refresh token を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の既存 bearer access token は拒否される (AUTH-BE-S055)

- **GIVEN** account が active session を持っていた後に Admin Console で suspended になっている
- **WHEN** その access token で `/api/v1/*` にアクセスする
- **THEN** system は access token を認可せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspended account の refresh は rotation されない (AUTH-BE-S058)

- **GIVEN** account が valid refresh token を持っていた後に Admin Console で suspended になっている
- **WHEN** client が `POST /api/v1/auth/refresh` を呼び出す
- **THEN** system は新しい access token / refresh token を発行せず、HTTP 403 の `AuthFailureResponse` と `error="account-suspended"` で拒否する

#### Scenario: suspend は account-wide session revocation timestamp を書き込む (AUTH-BE-S056)

- **GIVEN** Admin Console が account suspend を成功させる
- **WHEN** database の `accounts.session_revoked_after` を確認する
- **THEN** suspend 時刻以上の timestamp が保存され、その timestamp 以前に発行された access token / refresh token は拒否される

#### Scenario: restored account は過去 session では復帰できない (AUTH-BE-S057)

- **GIVEN** account が suspended 後に restore されている
- **WHEN** suspend 前に発行された bearer access token で `/api/v1/*` にアクセスする
- **THEN** system は access token を拒否し、account は再ログインでのみ新しい token pair を取得できる

#### Scenario: account-suspended は stable failure response shape で返される (AUTH-BE-S059)

- **GIVEN** suspended 判定が `POST /api/v1/auth/passkey/finish`、`POST /api/v1/auth/refresh`、または bearer-protected `/api/v1/*` endpoint で発生する
- **WHEN** system が response を返す
- **THEN** HTTP status は 403 であり、body は `AuthFailureResponse` の `{ requestId, error: "account-suspended" }` である
- **AND** response は `Cache-Control: no-store` を含み、`AuthOperationErrorResponse` では返されない
