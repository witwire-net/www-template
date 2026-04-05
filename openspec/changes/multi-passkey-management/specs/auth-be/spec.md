## ADDED Requirements

### Requirement: 認証済みアカウントは複数のパスキーを登録・管理できる

**Customer Context**

利用者は MacBook・iPhone・セキュリティキーなど複数のデバイスでパスキーを使い分けたい。新しいデバイスを登録しても他のデバイスのアクセスが失われないことが求められる。複数のパスキーを独立して管理できることで、デバイス追加・紛失後の安全な鍵ローテーションが可能になる。

**Requirement**

- 認証済みアカウントは 1 件以上の passkey credential を持つことができ、システムはすべての active な passkey credential を保持しなければならない（SHALL）。
- システムは `GET /api/v1/app/passkeys` で認証済みアカウントの登録済みパスキー一覧（ID・識別子・登録日時）を返さなければならない（SHALL）。
- システムは `POST /api/v1/app/passkeys/start` で WebAuthn 追加登録チャレンジを発行し、`POST /api/v1/app/passkeys/finish` でチャレンジを検証して既存パスキーを保持したまま新しい passkey credential をアカウントへ追加しなければならない（SHALL）。
- システムは `DELETE /api/v1/app/passkeys/{id}` で指定した passkey credential を削除しなければならない（SHALL）。ただし、アカウントに残る passkey credential が 1 件になる場合は削除を拒否しなければならない（MUST）。
- 上記すべての管理エンドポイントは `Authorization: Bearer <session token>` を必須とし、未認証リクエストは SHALL 拒否されなければならない。
- 他のアカウントに属する passkey credential を操作する試みは SHALL 拒否されなければならない。
- パスキー管理操作で用いる resource ID（credential ID、challenge ID、correlation ID 等）は ULID を使用しなければならない（SHALL）。

#### Scenario: 登録済みパスキー一覧を取得できる (AUTH-BE-S014)

- **GIVEN** 認証済みアカウントが bearer session を持っている
- **WHEN** `GET /api/v1/app/passkeys` を呼び出す
- **THEN** システムはそのアカウントに紐づくすべての passkey credential のリストを返す

#### Scenario: 新しいパスキーを追加しても既存パスキーが保持される (AUTH-BE-S015)

- **GIVEN** 認証済みアカウントが bearer session を持っている
- **WHEN** `POST /api/v1/app/passkeys/start` でチャレンジを取得し `POST /api/v1/app/passkeys/finish` で完了する
- **THEN** 新しい passkey credential がアカウントへ追加され、それ以前に登録されていたパスキーは削除されない

#### Scenario: 最後の 1 件のパスキーは削除できない (AUTH-BE-S016)

- **GIVEN** 認証済みアカウントが passkey credential を 1 件だけ保持している
- **WHEN** `DELETE /api/v1/app/passkeys/{id}` でその 1 件を削除しようとする
- **THEN** システムはリクエストを拒否し、アカウントのパスキーは変化しない

#### Scenario: 複数あるパスキーの 1 件を削除できる (AUTH-BE-S017)

- **GIVEN** 認証済みアカウントが 2 件以上の passkey credential を保持している
- **WHEN** `DELETE /api/v1/app/passkeys/{id}` で特定の 1 件を指定する
- **THEN** 指定された passkey credential のみが削除され、残りは保持される

#### Scenario: 他のアカウントのパスキーは操作できない (AUTH-BE-S018)

- **GIVEN** アカウント A が bearer session を持っている
- **WHEN** アカウント B に属する passkey credential の ID を指定して `DELETE /api/v1/app/passkeys/{id}` を呼び出す
- **THEN** システムはリクエストを拒否し、アカウント A のパスキーは変化しない

#### Scenario: 未認証リクエストはパスキー管理 API を利用できない (AUTH-BE-S019)

- **GIVEN** bearer session を持たないリクエストがある
- **WHEN** `/api/v1/app/passkeys` 以下のいずれかのエンドポイントを呼び出す
- **THEN** システムは unauthenticated として拒否する

### Requirement: OTP ハンドオフによる新端末へのパスキー追加

**Customer Context**

利用者が新しいデバイスでパスキーを登録したい場合、すでにログイン済みの既存デバイスを使って安全に認可できる。既存デバイスで再認証して OTP を取得し、新デバイスに手入力することで、新デバイスはログインなしにパスキーを登録できる。

**Requirement**

- システムは `POST /api/v1/app/passkeys/otp` で OTP を発行しなければならない（SHALL）。このエンドポイントは bearer session を必須とし、呼び出し前に既存パスキーによる WebAuthn 再認証を完了していなければならない（MUST）。
- OTP は 6 桁の数字とし、有効期限は発行から 5 分とする（SHALL）。OTP は Valkey-backed auth state store に保存し、消費またはタイムアウト後は再利用できない（MUST NOT）。
- システムは `POST /api/v1/auth/passkey/add/start` で OTP を受け取り、検証後に WebAuthn 登録チャレンジを発行しなければならない（SHALL）。このエンドポイントは未認証（bearer session 不要）の公開エンドポイントとする。
- システムは `POST /api/v1/auth/passkey/add/finish` で WebAuthn 登録クレデンシャルと OTP を受け取り、OTP が指すアカウントへ新しい passkey credential を追加しなければならない（SHALL）。既存の passkey credential はすべて保持されなければならない（MUST）。
- OTP の検証に失敗した場合、または OTP が有効期限切れ・消費済みの場合はリクエストを拒否しなければならない（SHALL）。
- `POST /api/v1/auth/passkey/add/*` で用いる OTP・challenge ID・credential ID は ULID を使用しなければならない（SHALL）。

#### Scenario: OTP を発行できる (AUTH-BE-S021)

- **GIVEN** 認証済みアカウントが bearer session を持ち、既存パスキーで WebAuthn 再認証を完了している
- **WHEN** `POST /api/v1/app/passkeys/otp` を呼び出す
- **THEN** システムは 6 桁の OTP を返す（有効期限 5 分）

#### Scenario: OTP を使って新端末にパスキーを追加できる (AUTH-BE-S022)

- **GIVEN** 有効な OTP が発行されている
- **WHEN** 新端末が `POST /api/v1/auth/passkey/add/start` で OTP を提示してチャレンジを取得し、`POST /api/v1/auth/passkey/add/finish` で WebAuthn 登録を完了する
- **THEN** 新しい passkey credential がアカウントへ追加され、既存のパスキーは保持される

#### Scenario: 有効期限切れの OTP は拒否される (AUTH-BE-S023)

- **GIVEN** 発行から 5 分を超えた OTP がある
- **WHEN** 新端末が `POST /api/v1/auth/passkey/add/start` でその OTP を提示する
- **THEN** システムはリクエストを拒否する

#### Scenario: 消費済みの OTP は再利用できない (AUTH-BE-S024)

- **GIVEN** すでに使用された OTP がある
- **WHEN** 同じ OTP で再度 `POST /api/v1/auth/passkey/add/start` を呼び出す
- **THEN** システムはリクエストを拒否する
