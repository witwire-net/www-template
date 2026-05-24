## MODIFIED Requirements

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

パスキー認証は、短命な JWT accessToken と長寿命 refreshToken で構成される session を SHALL 発行し、`Authorization: Bearer <accessToken>` で `/api/v1/*` を利用できるようにしなければならない。refreshToken は response body に含めてはならず（MUST NOT）、`HttpOnly; Secure; SameSite=Lax; Path=/` Cookie として発行・rotation・revoke されなければならない（MUST）。Product account 認証ドメインでは accessToken claim と refreshToken state は account ID / session ID / device fingerprint に束縛されなければならない（MUST）。Admin operator 認証ドメインは同じ token service と cookie policy を可能な限り再利用するが、operator ID / operator session ID / Admin Valkey logical DB / `admin:*` key prefix に束縛し、Product account と混同してはならない（MUST NOT）。Product DB の `accounts.status='suspended'` の account に対しては、新規 accessToken 発行、refresh rotation、既存 bearer accessToken 認可を MUST 拒否する。

**Customer Context**

Product と Admin の認証基盤が別々の token 実装を持つと、rotation、revoke、Cookie 属性、ログ漏えい防止、テストの粒度が分岐する。refreshToken を HttpOnly Cookie に寄せることで XSS 時の token 窃取リスクを下げつつ、Admin operator 認証では account とは別の認証ドメインとして同じ安全な基盤を再利用できる。

#### Scenario: Product passkey login は accessToken body と refreshToken Cookie を返す (AUTH-BE-S060)

- **GIVEN** account が Product passkey authentication を開始している
- **WHEN** valid credential で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** response body は accessToken と session metadata を含む
- **AND** response body は refreshToken を含まない
- **AND** `Set-Cookie` は refreshToken を `HttpOnly; Secure; SameSite=Lax; Path=/` で設定する

#### Scenario: Admin operator login は共通 token 基盤を operator 認証ドメインで使う (AUTH-BE-S061)

- **GIVEN** operator が Admin passkey authentication を開始している
- **WHEN** valid operator credential で Admin auth finish を完了する
- **THEN** response body は operator accessToken と operator session metadata を含む
- **AND** refreshToken Cookie は operator ID と operator session に束縛される
- **AND** Product account ID と Product auth state は使用されない

#### Scenario: refresh は Cookie の refreshToken を rotation する (AUTH-BE-S062)

- **GIVEN** client が有効な refreshToken Cookie を持つ
- **WHEN** client が `POST /api/v1/auth/refresh` を呼び出す
- **THEN** system は旧 refreshToken を原子消費し、新しい accessToken を response body に返す
- **AND** 新しい refreshToken は `Set-Cookie` で rotation され、response body には含まれない

#### Scenario: browser-readable refreshToken は発行されない (AUTH-BE-S063)

- **GIVEN** login、refresh、recovery registration、または operator login が成功する
- **WHEN** response body と log/trace attributes を確認する
- **THEN** refreshToken の平文値は body、log、trace attribute、error message に存在しない

### Requirement: リフレッシュトークンは設定可能な TTL で管理される

リフレッシュトークンは設定可能な TTL で管理されなければならない（SHALL）。TTL は Product account 認証ドメインと Admin operator 認証ドメインで同じ validation logic を再利用できなければならない（SHALL）。TTL 付き refreshToken state は各認証ドメインの Valkey logical DB と key prefix に保存されなければならない（MUST）。refreshToken Cookie の `Max-Age` または `Expires` は server-side refreshToken state の TTL を超えてはならない（MUST NOT）。TTL が未設定またはゼロ値の場合の無期限扱いを許可する場合でも、Cookie と server-side state の整合を保たなければならない（MUST）。

**Customer Context**

運用者はセキュリティポリシーに応じて refreshToken の寿命を調整したい。Product と Admin で TTL validation が分かれると、片方だけ弱い設定を受け入れる危険がある。

#### Scenario: refreshToken Cookie の寿命は server-side TTL を超えない (AUTH-BE-S064)

- **GIVEN** refresh token TTL が 30 日に設定されている
- **WHEN** login または refresh rotation が refreshToken Cookie を設定する
- **THEN** Cookie の `Max-Age` または `Expires` は server-side refreshToken state の TTL 以下である

#### Scenario: Product と Admin は同じ TTL validation を使う (AUTH-BE-S065)

- **GIVEN** refresh token TTL が許容範囲外に設定されている
- **WHEN** Product API binary または Admin API binary が起動時 config validation を実行する
- **THEN** どちらの binary も同じ validation rule で fail-close する

### Requirement: システムは複数の active session を同時に保持・管理できる

システムは同一 browser 上で複数 account session を保持できなければならない（SHALL）。各 session は accessToken、session ID、account ID、server-side refreshToken state、refreshToken Cookie binding を持たなければならない（SHALL）。refreshToken が HttpOnly Cookie であるため、client は refreshToken 平文を保持してはならない（MUST NOT）。複数 session の refresh は、server が session selector と Cookie binding を検証して対象 session だけを rotation しなければならない（MUST）。logout や session revoke は対象 session の accessToken metadata と refreshToken state / Cookie を失効させ、他 session に影響してはならない（MUST NOT）。

**Customer Context**

複数アカウントを扱う利用者は、account を切り替えながら作業したい。一方で refreshToken を JavaScript から読める形で保持すると XSS 時の被害が大きい。HttpOnly Cookie と session selector を組み合わせ、複数 session と token 窃取防止を両立する。

#### Scenario: 複数 session の refresh は対象 session だけを rotation する (AUTH-BE-S066)

- **GIVEN** browser が account A と account B の active session を持っている
- **WHEN** account A の session selector で `POST /api/v1/auth/refresh` を呼び出す
- **THEN** account A の refreshToken state と Cookie binding だけが rotation される
- **AND** account B の session は維持される
