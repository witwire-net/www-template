## Purpose

Auth core frontend requirements, covering low-emphasis login handoff, passkey-only login, recovery-only routes, no-store auth routes, logout and session-expiry routing, brand constraints, and ULID identifier policy.

## Requirements

### Requirement: 低強調のパスキーログイン導線を提供する

低強調のパスキーログイン導線は、公開面の主要 CTA を保ったまま passkey-only な `/login` へ到達できる体験を SHALL 提供しなければならない。

**Customer Context**

公開面は社外向けの発信サイトとして見せつつ、社内利用者は同じドメインから認証面へ入れる必要があります。企画書とサイトマップは、公開面で内部文脈を露出せず、ログイン導線を低強調に保ちながら、`/login` ではパスキーだけで認証を始められることを求めています。

**Requirement**

- The system SHALL keep the public-to-login handoff low emphasis and MUST present `/login` as a passkey-only sign-in surface.
- public surface のログイン handoff は、補助ナビゲーションまたはフッターに留まる低強調導線を SHALL 保ち、主要 CTA や hero action に拡張してはならない。
- `/login` route は、passkey sign-in action、`/login/recovery` への recovery link、公開面へ戻る導線を持つ passkey 専用 sign-in 画面を SHALL 表示する。
- `/login` route で認証が成功した後の client auth state は、`Authorization: Bearer <session token>` で `/api/v1/app/*` を利用できる session 契約に SHALL 接続される。
- client が保持・表示・遷移判定に使う `accountId`、`sessionId`、`passkeyCredentialId`、`recoverySessionId`、および auth notification / audit / action correlation ID など system-owned resource ID が必要な箇所は ULID を SHALL 前提とする。
- `/login` route は、password input、password reset copy、invite registration control、Guest onboarding copy を表示してはならない（MUST NOT）。
- 認証導線の画面は、`WitWire` 表記、IBM Plex Sans / IBM Plex Sans JP、token-based color、8px grid、Flat & Bright、shadow / glow 禁止の brand system に SHALL 準拠する。

#### Scenario: 公開面の低強調 handoff から `/login` へ到達する (AUTH-FE-S001)

- **GIVEN** 利用者が低強調ログイン handoff を持つ公開ページを閲覧している
- **WHEN** 利用者がその handoff から認証導線へ遷移する
- **THEN** 利用者は主要 CTA を増やさずに `/login` へ到達する

#### Scenario: ログイン画面はパスキー専用でサインインを提供する (AUTH-FE-S002)

- **GIVEN** 利用者が `/login` を開く
- **WHEN** ログイン画面が読み込まれる
- **THEN** 画面には passkey sign-in action と recovery link が表示され、password entry や invite registration control は表示されない

### Requirement: 復旧導線は既存アカウントの passkey 再登録だけを扱う

復旧導線は、既存アカウントの passkey 再登録だけを招待導線や規約同意導線と分離したまま SHALL 提供しなければならない。

**Customer Context**

パスキーを紛失した利用者は、招待オンボーディングへ戻らずに、登録済みアカウントの passkey を安全に再登録できる必要があります。同時に recovery 導線は、アカウント有無や招待状態を UI から推測できない、復旧専用の体験でなければなりません。

**Requirement**

- The system SHALL provide a recovery-only route family for existing accounts and MUST keep invite onboarding and consent flows out of `/login/recovery/*`.
- `/login/recovery` route は登録済みメールアドレスを受け取り、受理された依頼を `/login/recovery/sent` へ接続する recovery request を SHALL 送信できる。
- `/login/recovery/sent` route は recovery URL を送信したことを SHALL 表示し、`/login` へ戻る導線を SHALL 提供する。
- recovery request の結果表示は、アカウント有無、送信抑止、temporary lock を UI から判別できない共通の受理メッセージを SHALL 保つ。
- `/login/recovery/consume` route は recovery token を検証し、`/login/recovery/register` へ進む recovery-ready state か、`/login/recovery` へ戻る retry guidance のいずれかに SHALL 分岐する。
- `/login/recovery/register` route は recovery branch のみを使って、既存アカウントに対する passkey 再登録を SHALL 完了できる。
- recovery 再登録の成功後の client auth state は、`Authorization: Bearer <session token>` で `/api/v1/app/*` を利用できる session 契約に SHALL 接続される。
- recovery request / consume / register の view model、route state、navigation payload、toast / notice / telemetry payload などで ID が必要な箇所は ULID を SHALL 用い、UUID 前提の copy / mock / sample を残してはならない。
- `/login/recovery/*` routes は invitation token、invite consent、Guest onboarding copy、TermsConsent UI を表示・保存・参照してはならない（MUST NOT）。
- `/login/recovery/consume` と `/login/recovery/register` は、dedicated wireframe が追加されるまで recovery / recovery-sent と同じ card hierarchy、spacing rhythm、CTA ordering を SHALL 継承する。

#### Scenario: 復旧依頼は送信完了画面へ進む (AUTH-FE-S003)

- **GIVEN** 利用者が `/login/recovery` を開く
- **WHEN** 利用者がメールアドレスを送信し、その依頼が受理される
- **THEN** 体験は `/login/recovery/sent` に遷移し、アカウント有無を明かさない共通メッセージで recovery URL 送信を案内する

#### Scenario: 有効な復旧リンクはパスキー再登録へ進む (AUTH-FE-S004)

- **GIVEN** 利用者が有効な recovery token 付きで `/login/recovery/consume` を開く
- **WHEN** token validation が成功する
- **THEN** 利用者は `/login/recovery/register` へ進み、invite onboarding UI や TermsConsent UI なしで既存アカウントの passkey を再登録できる

#### Scenario: 無効な復旧リンクは再試行案内へ戻す (AUTH-FE-S005)

- **GIVEN** 利用者が無効、期限切れ、または消費済みの recovery token で `/login/recovery/consume` を開く
- **WHEN** token validation が失敗する
- **THEN** UI は登録 action を出さず、retry guidance と `/login/recovery` へ戻る導線を表示する

### Requirement: auth routes は no-store な認証面として配信する

auth routes は、edge / browser cache から stale な認証 UI や session-bound state を再提示しない no-store surface として SHALL 配信されなければならない。

**Customer Context**

Phase 3 の auth コアは auth endpoint だけでなく auth route も no-store scope に含める前提で設計されています。`/login*` や `/logout` が cacheable になると、Cloudflare/WAF 配下や browser の戻る操作で古い認証状態が再表示され、session lifecycle と logout 導線の整合が崩れます。

**Requirement**

- The system SHALL serve `/login`, `/login/recovery*`, and `/logout` as no-store auth routes and MUST prevent edge/browser caches from replaying stale auth entry state.
- `/login`, `/login/recovery`, `/login/recovery/sent`, `/login/recovery/consume`, `/login/recovery/register`, `/logout` route responses は auth endpoint と揃った no-store cache semantics を SHALL 保つ。
- auth routes は公開検索面や cacheable bootstrap state に依存せず、直前の login / recovery / logout intent を基準に最新の auth presentation を SHALL 表示する。

#### Scenario: auth routes は no-store surface として配信される (AUTH-FE-S009)

- **GIVEN** 利用者が `/login`、`/login/recovery`、または `/logout` を browser または edge 経由で開く
- **WHEN** auth route response が配信される
- **THEN** system はその route を no-store auth surface として扱い、stale な auth UI や session-bound state を再利用しない

### Requirement: session expiry と logout は未認証導線を明確に分離する

session expiry と logout の導線は、expired / revoked session と missing session を区別しながら未認証状態への復帰導線を SHALL 提供しなければならない。

**Customer Context**

`/app/*` に入る基盤では、利用者が「未ログインなのか」「認証が切れたのか」を迷わないことが重要です。session expiry と logout の導線が曖昧だと、再認証、公開面への退避、エラー画面 owner の責務が混線します。

**Requirement**

- The system SHALL distinguish expired-or-revoked sessions from missing sessions and MUST route logout to an unauthenticated state.
- 現在の session が有効である間、`/app/*` 上の authenticated navigation は SHALL 継続する。
- 以前は有効だった session が expired または revoked と判定されたとき、client は利用者を `/app/session-expired` へ SHALL 遷移させる。
- `/app/session-expired` route の presentation は auth routes から独立した owner に保たれ、Auth コアは redirect trigger と route selection だけを SHALL 担当する。
- 現在の bearer session を持たない初回の `/app/*` アクセスは、通常の未認証ログイン導線へ SHALL 留まり、`/app/session-expired` と混同してはならない。
- client は bearer token を in-memory にのみ保持しなければならず（SHALL）、tab または browser を閉じた後の再訪では previously issued bearer session を復元せず、missing session と同じ `unauthenticated` 扱いに SHALL 正規化し、`/app/session-expired` へ送ってはならない。
- `/logout` route は現在の bearer session を SHALL revoke し、client が保持する bearer-authenticated state を消去し、利用者を public route または login route の非認証状態へ戻す。
- `/logout` route は public utility route として存在しても、logout 実行自体は canonical な `POST /api/v1/app/auth/logout` を呼び出して完了しなければならない（SHALL）。
- logout / expiry handling で client が参照する session ID、account ID、event ID、request ID、notification ID などの識別子が必要な箇所は ULID を SHALL 用い、opaque bearer token や cache key を ULID resource ID と混同してはならない。
- `/logout` 導線は、invite onboarding や権限管理 copy を混在させない抑制された auth presentation を SHALL 保つ。

#### Scenario: セッション失効時は再認証画面へリダイレクトする (AUTH-FE-S006)

- **GIVEN** 利用者が `/app/*` 内で操作している
- **WHEN** 現在の session が expired または revoked として報告される
- **THEN** 利用者は `/app/session-expired` へ遷移し、その後の画面 presentation はその route contract に委ねられる

#### Scenario: logout は利用者を非認証 route へ戻す (AUTH-FE-S007)

- **GIVEN** 利用者が active な authenticated session を持っている
- **WHEN** 利用者が `/logout` を開く
- **THEN** bearer-authenticated state は消去され、利用者は signed in として振る舞わない public route または login route に到達する

#### Scenario: session を持たない `/app/*` 到達は通常の未認証導線に留まる (AUTH-FE-S008)

- **GIVEN** 利用者が有効な bearer session を持たずに `/app/*` を開く
- **WHEN** app が current session の不在を検知する
- **THEN** 利用者は通常の login 導線へ進み、`/app/session-expired` へは遷移しない
