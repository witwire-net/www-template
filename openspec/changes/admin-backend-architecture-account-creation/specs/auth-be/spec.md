## MODIFIED Requirements

### Requirement: パスキー認証は bearer 互換 application session を発行・失効する

Product account 認証ドメインは、短命な account accessToken と長寿命 account refreshToken で構成される Product account session を SHALL 発行し、`Authorization: Bearer <accessToken>` で Product `/api/v1/*` を利用できるようにしなければならない。Product account refreshToken は response body に含めてはならず（MUST NOT）、`HttpOnly; Secure; SameSite=Lax; Path=/` Cookie として発行・rotation・revoke されなければならない（MUST）。Product account accessToken claim と refreshToken state は account ID / account session ID / device fingerprint / account status / sessionRevokedAfter に束縛されなければならない（MUST）。Product DB の `accounts.status='suspended'` の account に対しては、新規 accessToken 発行、refresh rotation、既存 bearer accessToken 認可を MUST 拒否する。

Admin operator 認証ドメインは Product account 認証ドメインとは別に、operator accessToken と operator refreshToken で構成される Admin operator session を SHALL 発行する。Admin operator accessToken claim と refreshToken state は operator ID / operator session ID / operator role / active state / CSRF binding / Admin Valkey logical DB / `admin:*` key prefix に束縛されなければならない（MUST）。Admin operator auth は Product account auth domain/application を import してはならず（MUST NOT）、Product account auth は Admin operator auth domain/application を import してはならない（MUST NOT）。

両認証ドメインが共有できるのは、HMAC/JWT signer/verifier、opaque token hash、Cookie 属性 helper、ULID/JTI validation、TTL validation helper など中立 primitive に限られる（MUST）。中立 primitive は account / operator の domain enum switch、issuer/audience/domain pairing、RBAC、account status、operator active state、CSRF binding を所有してはならない（MUST NOT）。単一共有 token service に `identityDomain=account|operator` の切替引数を渡して Product/Admin の domain decision を畳み込んではならない（MUST NOT）。

**Customer Context**

Product 利用者の account 認証と Admin 運営者の operator 認証は、守る対象と失敗時の影響が異なる。Cookie 属性や署名検証のような安全な primitive は共通化してよいが、account status、operator role、CSRF、session state を単一 service の切替で扱うと境界が曖昧になり、誤認可や監査漏れにつながる。

#### Scenario: Product passkey login は accessToken body と refreshToken Cookie を返す (AUTH-BE-S060)

- **GIVEN** account が Product passkey authentication を開始している
- **WHEN** valid credential で `POST /api/v1/auth/passkey/finish` を完了する
- **THEN** response body は account accessToken と Product account session metadata を含む
- **AND** response body は refreshToken を含まない
- **AND** `Set-Cookie` は refreshToken を `HttpOnly; Secure; SameSite=Lax; Path=/` で設定する

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

- **GIVEN** client が有効な refreshToken Cookie を持つ
- **WHEN** client が `POST /api/v1/auth/refresh` を呼び出す
- **THEN** 対象 auth domain は旧 refreshToken を原子消費し、新しい accessToken を response body に返す
- **AND** 新しい refreshToken は `Set-Cookie` で rotation され、response body には含まれない

#### Scenario: ブラウザーから読める refreshToken は発行されない (AUTH-BE-S063)

- **GIVEN** login、refresh、recovery registration、または operator login が成功する
- **WHEN** response body と log/trace attributes を確認する
- **THEN** refreshToken の平文値は body、log、trace attribute、error message に存在しない

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

### Requirement: システムは複数の active session を同時に保持・管理できる

システムは同一 browser 上で複数 account session を保持できなければならない（SHALL）。各 Product account session は accessToken、account session ID、account ID、server-side refreshToken state、refreshToken Cookie binding を持たなければならない（SHALL）。refreshToken が HttpOnly Cookie であるため、client は refreshToken 平文を保持してはならない（MUST NOT）。複数 session の refresh は、Product account auth domain が session selector と Cookie binding を検証して対象 session だけを rotation しなければならない（MUST）。logout や session revoke は対象 session の accessToken metadata と refreshToken state / Cookie を失効させ、他 session に影響してはならない（MUST NOT）。

**Customer Context**

複数アカウントを扱う利用者は、account を切り替えながら作業したい。一方で refreshToken を JavaScript から読める形で保持すると XSS 時の被害が大きい。HttpOnly Cookie と session selector を組み合わせ、複数 session と token 窃取防止を両立する。

#### Scenario: 複数 session の refresh は対象 session だけを rotation する (AUTH-BE-S066)

- **GIVEN** browser が account A と account B の active session を持っている
- **WHEN** account A の session selector で `POST /api/v1/auth/refresh` を呼び出す
- **THEN** account A の refreshToken state と Cookie binding だけが rotation される
- **AND** account B の session は維持される
