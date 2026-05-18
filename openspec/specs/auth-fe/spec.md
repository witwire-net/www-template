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
- `/login` route で認証が成功した後の client auth state は、`Authorization: Bearer <session token>` で `/api/v1/*` を利用できる session 契約に SHALL 接続される。
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

復旧導線とデバイスリンク導線は、既存アカウントの passkey 登録だけを招待導線や規約同意導線と分離したまま SHALL 提供しなければならない。トークンの kind に応じて、復旧用と新端末追加用で完了後の体験を分岐する。

**Customer Context**

パスキーを紛失した利用者（kind=recovery）と、新端末でログインを有効にしたい利用者（kind=device-link）は、どちらも invite onboarding へ戻らずに、登録済みアカウントの passkey を安全に登録できる必要がある。同時にこれらの導線は、アカウント有無や招待状態を UI から推測できない体験でなければならない。また、kind に応じて登録完了後の説明文面が異なるため、UI は kind を認識して適切なメッセージを表示する必要がある。

**Requirement**

- The system SHALL provide a recovery/device-link route family for existing accounts and MUST keep invite onboarding and consent flows out of these routes.
- `/login/recovery` route は登録済みメールアドレスを受け取り、受理された依頼を `/login/recovery/sent` へ接続する recovery request を SHALL 送信できる。
- `/login/recovery/sent` route は recovery URL を送信したことを SHALL 表示し、`/login` へ戻る導線を SHALL 提供する。
- recovery request の結果表示は、アカウント有無、送信抑止、temporary lock を UI から判別できない共通の受理メッセージを SHALL 保つ。
- `/login/recovery/consume` route は token を検証し、token の `kind` に応じて遷移先を分岐しなければならない（SHALL）。`kind=recovery` の場合は `/login/recovery/register` へ、`kind=device-link` の場合はデバイスリンク登録画面へ遷移する。
- `/login/recovery/register` route は recovery/device-link branch のみを使って、既存アカウントに対する passkey 登録を SHALL 完了できる。登録完了後、kind に応じた完了メッセージを表示しなければならない。
- 登録成功後の client auth state は、`Authorization: Bearer <session token>` で `/api/v1/*` を利用できる session 契約に SHALL 接続される。
- request / consume / register の view model、route state、navigation payload、toast / notice / telemetry payload などで ID が必要な箇所は ULID を SHALL 用い、UUID 前提の copy / mock / sample を残してはならない。
- これらの routes は invitation token、invite consent、Guest onboarding copy、TermsConsent UI を表示・保存・参照してはならない（MUST NOT）。

#### Scenario: 復旧依頼は送信完了画面へ進む (AUTH-FE-S003)

- **GIVEN** 利用者が `/login/recovery` を開く
- **WHEN** 利用者がメールアドレスを送信し、その依頼が受理される
- **THEN** 体験は `/login/recovery/sent` に遷移し、アカウント有無を明かさない共通メッセージで recovery URL 送信を案内する

#### Scenario: 有効な復旧リンク (kind=recovery) はパスキー再登録へ進む (AUTH-FE-S004)

- **GIVEN** 利用者が kind=recovery の valid token 付きで `/login/recovery/consume` を開く
- **WHEN** token validation が成功する
- **THEN** 利用者は `/login/recovery/register` へ進み、invite onboarding UI や TermsConsent UI なしで既存アカウントの passkey を登録できる

#### Scenario: 有効なデバイスリンク (kind=device-link) はパスキー登録へ進む (AUTH-FE-S038)

- **GIVEN** 利用者が kind=device-link の valid token 付きで `/login/recovery/consume` を開く
- **WHEN** token validation が成功する
- **THEN** 利用者は device-link 用のパスキー登録画面へ進み、invite onboarding UI や TermsConsent UI なしで既存アカウントの passkey を登録できる

#### Scenario: 無効な復旧リンクは再試行案内へ戻す (AUTH-FE-S005)

- **GIVEN** 利用者が無効、期限切れ、または消費済みの token で `/login/recovery/consume` を開く
- **WHEN** token validation が失敗する
- **THEN** UI は登録 action を出さず、retry guidance と `/login/recovery` へ戻る導線を表示する

### Requirement: auth routes は no-store な認証面として配信する

auth routes は、edge / browser cache から stale な認証 UI や session-bound state を再提示しない no-store surface として SHALL 配信されなければならない。

**Customer Context**

Auth コアは auth endpoint だけでなく auth route も no-store scope に含める前提で設計されています。`/login*` や `/logout` が cacheable になると、Cloudflare/WAF 配下や browser の戻る操作で古い認証状態が再表示され、session lifecycle と logout 導線の整合が崩れます。

**Requirement**

- The system SHALL serve `/login`, `/login/recovery*`, `/logout` as no-store auth routes and MUST prevent edge/browser caches from replaying stale auth entry state.
- `/login`, `/login/recovery`, `/login/recovery/sent`, `/login/recovery/consume`, `/login/recovery/register`, `/logout` route responses は auth endpoint と揃った no-store cache semantics を SHALL 保つ。
- auth routes は公開検索面や cacheable bootstrap state に依存せず、直前の login / recovery / logout / device enablement intent を基準に最新の auth presentation を SHALL 表示する。
- Secret-bearing route input は browser-visible URL、route state、telemetry、persistent storage に必要以上に保持してはならない（MUST NOT）。
- Auth route presentation は account existence、token validity、temporary lock state、recovery token state を UI 文言から推測できない generic error semantics を SHALL 維持する。

#### Scenario: auth routes は no-store surface として配信される (AUTH-FE-S009)

- **GIVEN** 利用者が `/login`、`/login/recovery*`、または `/logout` を browser または edge 経由で開く
- **WHEN** auth route response が配信される
- **THEN** system はその route を no-store auth surface として扱い、stale な auth UI や session-bound state を再利用しない

### Requirement: session expiry と logout は未認証導線を明確に分離する

session expiry と logout の導線は、expired / revoked session と missing session を区別しながら未認証状態への復帰導線を SHALL 提供しなければならない。

**Customer Context**

`/*` に入る基盤では、利用者が「未ログインなのか」「認証が切れたのか」を迷わないことが重要です。session expiry と logout の導線が曖昧だと、再認証、公開面への退避、エラー画面 owner の責務が混線します。

**Requirement**

- The system SHALL distinguish expired-or-revoked sessions from missing sessions and MUST route logout to an unauthenticated state.
- 現在の session が有効である間、`/*` 上の authenticated navigation は SHALL 継続する。
- 以前は有効だった session が expired または revoked と判定されたとき、client は利用者を `/session-expired` へ SHALL 遷移させる。
- `/session-expired` route の presentation は auth routes から独立した owner に保たれ、Auth コアは redirect trigger と route selection だけを SHALL 担当する。
- 現在の bearer session を持たない初回の `/*` アクセスは、通常の未認証ログイン導線へ SHALL 留まり、`/session-expired` と混同してはならない。
- client は bearer token を in-memory にのみ保持しなければならず（SHALL）、tab または browser を閉じた後の再訪では previously issued bearer session を復元せず、missing session と同じ `unauthenticated` 扱いに SHALL 正規化し、`/session-expired` へ送ってはならない。
- `/logout` route は現在の bearer session を SHALL revoke し、client が保持する bearer-authenticated state を消去し、利用者を public route または login route の非認証状態へ戻す。
- `/logout` route は public utility route として存在しても、logout 実行自体は canonical な `POST /api/v1/auth/logout` を呼び出して完了しなければならない（SHALL）。
- logout / expiry handling で client が参照する session ID、account ID、event ID、request ID、notification ID などの識別子が必要な箇所は ULID を SHALL 用い、opaque bearer token や cache key を ULID resource ID と混同してはならない。
- `/logout` 導線は、invite onboarding や権限管理 copy を混在させない抑制された auth presentation を SHALL 保つ。

#### Scenario: セッション失効時は再認証画面へリダイレクトする (AUTH-FE-S006)

- **GIVEN** 利用者が `/*` 内で操作している
- **WHEN** 現在の session が expired または revoked として報告される
- **THEN** 利用者は `/session-expired` へ遷移し、その後の画面 presentation はその route contract に委ねられる

#### Scenario: logout は利用者を非認証 route へ戻す (AUTH-FE-S007)

- **GIVEN** 利用者が active な authenticated session を持っている
- **WHEN** 利用者が `/logout` を開く
- **THEN** bearer-authenticated state は消去され、利用者は signed in として振る舞わない public route または login route に到達する

#### Scenario: session を持たない `/*` 到達は通常の未認証導線に留まる (AUTH-FE-S008)

- **GIVEN** 利用者が有効な bearer session を持たずに `/*` を開く
- **WHEN** app が current session の不在を検知する
- **THEN** 利用者は通常の login 導線へ進み、`/session-expired` へは遷移しない

### Requirement: 認証済みユーザーはアプリ内でパスキーを一覧・追加・削除できる

利用者は MacBook・iPhone・セキュリティキーなど複数のデバイスでパスキーを使い分けたい。また、古いデバイスや紛失したデバイスのパスキーを削除して鍵を整理したい。登録済みパスキーを確認・追加・削除できる管理画面が必要である。さらに、新しい端末でログインを有効にするためのデバイスリンク送信機能も必要である。

**Requirement**

- システムは認証済みアプリ内にパスキー管理ページを SHALL 提供し、登録済みのすべての passkey credential の識別子と登録日時を一覧表示しなければならない。
- パスキー管理ページは新しいパスキーを追加する WebAuthn 登録フローを SHALL 提供しなければならない（`POST /api/v1/passkeys/start` → `POST /api/v1/passkeys/finish`）。
- パスキー管理ページは「新しい端末でログインを有効にする」アクションを SHALL 提供しなければならない。このアクションは既存パスキーによる WebAuthn 再認証を完了してから `POST /api/v1/passkeys/send-device-link` 経由で登録メールアドレスへのデバイスリンク送信を依頼する。再認証成功後に取得した reauthentication session ID は `X-Reauth-Session` HTTP header に設定して送信する。
- UI は「パスキーを追加」や「credential」などの技術用語を主要 action label として使用せず、新しい端末でログインできるようにする目的を SHALL 表示する。
- デバイスリンク送信後、UI は登録メールアドレスへ送信されたこと、有効期限（30分）、第三者に共有しないことを SHALL 案内する。平文トークンは API response や UI に表示してはならない。
- パスキー管理ページは指定したパスキーを削除するアクションを SHALL 提供しなければならない（`DELETE /api/v1/passkeys/{id}`）。
- passkey credential が 1 件しかない場合、削除アクションは SHALL 無効化または非表示にしなければならない。
- パスキー追加フロー、デバイスリンク送信フロー、または削除フローでエラーが発生した場合は、エラーメッセージを SHALL 表示し、ページ状態を保持しなければならない。
- 管理ページは bearer session を必須とする認証済み surface であり、未認証アクセスは SHALL 拒否されなければならない。
- 管理ページは `WitWire` ブランドシステム（M PLUS 1 / Noto Sans JP / IBM Plex Mono、token-based color、8px grid、Flat & Bright、shadow/glow 禁止）に SHALL 準拠する。
- view model・route state・correlation ID など ID が必要な箇所は ULID を SHALL 使用しなければならない。

#### Scenario: パスキー管理ページで登録済みパスキーを確認できる (AUTH-FE-S010)

- **GIVEN** 利用者が認証済み状態でパスキー管理ページを開く
- **WHEN** ページが読み込まれる
- **THEN** 登録済みのすべてのパスキーの識別子と登録日時が一覧表示される

#### Scenario: 新しいパスキーを追加できる (AUTH-FE-S011)

- **GIVEN** 利用者がパスキー管理ページにいる
- **WHEN** 「パスキーを追加」アクションを起動し WebAuthn 登録フローを完了する
- **THEN** 新しいパスキーが一覧に追加され、既存のパスキーは変化しない

#### Scenario: パスキーを削除できる (AUTH-FE-S012)

- **GIVEN** 利用者が 2 件以上のパスキーを持つ状態でパスキー管理ページにいる
- **WHEN** 特定のパスキーの削除アクションを実行する
- **THEN** そのパスキーが一覧から削除され、残りのパスキーは変化しない

#### Scenario: 最後の 1 件のパスキーは削除アクションが無効化される (AUTH-FE-S013)

- **GIVEN** 利用者が passkey credential を 1 件だけ持つ状態でパスキー管理ページにいる
- **WHEN** ページが表示される
- **THEN** そのパスキーの削除アクションは無効化または非表示になっており、操作できない

#### Scenario: パスキー追加フロー中にエラーが発生した場合は通知される (AUTH-FE-S014)

- **GIVEN** 利用者がパスキー管理ページでパスキー追加フローを開始している
- **WHEN** WebAuthn 操作がキャンセルまたは失敗する
- **THEN** エラーメッセージが表示され、利用者はパスキー管理ページに留まる

#### Scenario: パスキー削除フロー中にエラーが発生した場合は通知される (AUTH-FE-S015)

- **GIVEN** 利用者がパスキー管理ページでパスキー削除アクションを実行している
- **WHEN** API がエラーを返す
- **THEN** エラーメッセージが表示され、一覧の状態は変化しない

#### Scenario: パスキー管理ページでデバイスリンクを送信できる (AUTH-FE-S035)

- **GIVEN** 利用者が認証済み状態でパスキー管理ページを開いている
- **WHEN** 「新しい端末でログインを有効にする」アクションを起動する
- **THEN** UI は WebAuthn 再認証を求め、成功後に登録メールアドレスへデバイスリンクを送信し、送信完了・有効期限・共有禁止の案内を表示する

#### Scenario: WebAuthn assertion request と attestation request は userVerification required を使用する (AUTH-FE-S037)

- **GIVEN** login または registration ceremony が開始されている
- **WHEN** client が WebAuthn options を構築する
- **THEN** `userVerification` は `"required"` に設定され、`"preferred"` や `"discouraged"` は使用されない

### Requirement: 認証 UI は secret leakage を抑える security presentation を提供する

**Customer Context**

認証画面では bearer session、recovery token、device login code などの機微情報が一時的に扱われる。利用者が browser history、Referer、画面共有、戻る操作、キャッシュ、XSS の影響を受けにくい体験を得るためには、UI と client state が secret を長く保持せず、表示やエラー文言から認証状態を推測できない必要がある。

**Requirement**

- システムは auth UI state、navigation state、history、telemetry、visible error message における secret exposure を最小化しなければならない。
- `/login/recovery/consume` route は URL から recovery token を読み取った後、token を browser-visible URL から SHALL 即時除去する。
- Auth UI は recovery token、OTP、bearer token、WebAuthn raw credential data を telemetry attribute、console output、visible debug UI、persistent route state に MUST 保存しない。
- Auth routes と protected app routes は no-store と、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy`、`X-Content-Type-Options`、frame embedding prevention を含む browser security headers を持つ形で配信または deployment-configure されなければならない。
- Client auth state persistence は bearer token の漏えいリスクを最小化し、browser close 後に session を復元しない auth contract を維持しなければならない。
- Secret-bearing UI は copy/paste や画面表示が必要な場合でも TTL、用途、再発行導線、共有禁止の案内を SHALL 表示する。

#### Scenario: recovery token は browser-visible URL から除去される (AUTH-FE-S019)

- **GIVEN** 利用者が recovery token 付き URL で `/login/recovery/consume` を開いている
- **WHEN** client が token を読み取って consume request を開始する
- **THEN** token は browser address bar と subsequent navigation URL から除去される

#### Scenario: auth routes は security headers と no-store semantics を持つ (AUTH-FE-S020)

- **GIVEN** 利用者が `/login`、`/login/recovery*`、`/logout`、または authenticated app route を開いている
- **WHEN** route response が配信される
- **THEN** route は no-store と browser security header semantics を持ち、stale auth UI や secret-bearing URL を replay しない

## ADDED Requirements

### Requirement: 新端末からトークン型のデバイスリンクでパスキーを追加できる

新端末の利用者は、メールで受信したデバイスリンク URL をクリックし、kind=device-link の token を消費した後、WebAuthn 登録フローを完了することでパスキーを追加できなければならない。UI は token の正当性を推測できない generic なエラーハンドリングを提供する。

**Customer Context**

新端末でログインできるようにしたい利用者は、6 桁のコードを手入力する必要なく、メール内のリンクをクリックするだけでパスキー登録を開始できる。登録が完了すると、kind=recovery の復旧とは異なり既存のセッションは維持される。

**Requirement**

- デバイスリンク URL の消費ページ（`/login/recovery/consume`）は、token 消費後に kind=device-link であることを認識し、適切な登録画面へ遷移しなければならない（SHALL）。
- デバイスリンク用のパスキー登録画面は、WebAuthn 登録フロー（`navigator.credentials.create()` → `POST /api/v1/auth/passkey/register`）を SHALL 提供する。
- 登録成功後、UI は「新しい端末でログインできるようになりました」という kind=device-link 用の完了メッセージを表示しなければならない。
- 登録成功後、UI はログイン状態へ遷移しなければならない（SHALL transition to authenticated state）。
- WebAuthn 操作のキャンセルまたは失敗時は、token の状態を露出しない generic なエラーメッセージを SHALL 表示し、再試行を可能にしなければならない。
- token が無効・期限切れ・消費済みの場合、email の登録有無や token の正否を示さない generic なエラーメッセージを SHALL 表示する。
- URL から token を読み取った後、token は browser-visible URL から SHALL 即時除去する。
- token は persistent storage、telemetry、URL query に MUST 保存しない。

#### Scenario: デバイスリンク URL から token を消費してパスキーを登録できる (AUTH-FE-S039)

- **GIVEN** 利用者が新端末でデバイスリンク URL を開いている
- **WHEN** token 消費が成功し WebAuthn 登録フローを完了する
- **THEN** ログイン状態になり、「新しい端末でログインできるようになりました」と表示される

#### Scenario: デバイスリンクの token が無効な場合は generic エラーが表示される (AUTH-FE-S040)

- **GIVEN** 利用者が新端末で無効なデバイスリンク URL を開いている
- **WHEN** token 消費が失敗する
- **THEN** generic エラーメッセージが表示され、登録フローは開始されない

#### Scenario: デバイスリンク URL の token は browser URL から除去される (AUTH-FE-S019)

- **GIVEN** 利用者が token 付き URL で consume ページを開いている
- **WHEN** client が token を読み取って consume request を開始する
- **THEN** token は browser address bar と subsequent navigation URL から除去される

### Requirement: クライアントは JWT アクセストークンの有効期限を監視し自動更新する

クライアントは JWT アクセストークンの有効期限を監視し、期限切れ前に自動的に更新しなければならない（MUST）。

**Customer Context**

利用者はアプリ操作中に認証が突然切れてデータを失う体験を避けたい。クライアントがトークンの残り寿命を把握し、期限切れ前に自動的に更新することで、シームレスなセッション継続が可能になる。

**Requirement**

- クライアントは JWT アクセストークンのペイロードをデコードし（署名検証不要）、`exp` クレームを読み取らなければならない（MUST）。
- アクセストークンの残り有効期限が 1 分未満、または既に期限切れの場合、クライアントは API 呼び出しの前に `POST /api/v1/auth/refresh` を SHALL 呼び出す。
- クライアントはトークン（アクセストークンおよびリフレッシュトークン）をメモリ上にのみ保持し、localStorage、sessionStorage、IndexedDB、cookie、またはその他の永続ストレージに保存してはならない（MUST NOT）。
- ブラウザタブまたはアプリを閉じた後、クライアントは以前のトークンを復元せず、未認証状態として正規化しなければならない（MUST）。
- リフレッシュに失敗した場合（無効なリフレッシュトークン、ネットワークエラー、サーバーエラー）、クライアントは対象セッションを失効として扱い、`/session-expired` へ遷移しなければならない（MUST）。
- トークン更新中に API 呼び出しが発生した場合、クライアントは更新完了後に順次 API 呼び出しを実行しなければならない（MUST）。更新失敗時は `/session-expired` へ遷移する。

#### Scenario: 期限切れ間近のアクセストークンは自動リフレッシュされる (AUTH-FE-S023)

- **GIVEN** クライアントが有効期限まで 1 分未満のアクセストークンを保持している
- **WHEN** 保護された API を呼び出そうとする
- **THEN** クライアントは先に `POST /api/v1/auth/refresh` を実行し、新しいアクセストークンで API を呼び出す

#### Scenario: 既に期限切れのアクセストークンはリフレッシュ後に API を呼び出す (AUTH-FE-S024)

- **GIVEN** クライアントが期限切れのアクセストークンを保持している
- **WHEN** 保護された API を呼び出そうとする
- **THEN** クライアントは `POST /api/v1/auth/refresh` を実行し、成功後に API を呼び出す。リフレッシュ失敗時は `/session-expired` へ遷移する

#### Scenario: トークンは永続ストレージに保存されない (AUTH-FE-S025)

- **GIVEN** 利用者がログインしてトークンを受け取る
- **WHEN** トークンがクライアントに保存される
- **THEN** トークンはメモリ上にのみ保持され、localStorage、sessionStorage、cookie、URL query、永続ストレージには書き込まれない

#### Scenario: ブラウザ再訪時は未認証状態に正規化される (AUTH-FE-S026)

- **GIVEN** 利用者がトークンを保持した状態でブラウザを閉じる
- **WHEN** 利用者が再度同じ URL を開く
- **THEN** クライアントは以前のトークンを復元せず、未認証状態として扱い、`/session-expired` へは遷移しない

### Requirement: クライアントは複数アカウントのセッションを同時に保持・切り替えできる

クライアントはメモリ上で複数アカウントのセッションを同時に保持し、アクティブセッションを切り替えできなければならない（SHALL）。

**Customer Context**

複数アカウントを運用する利用者にとって、都度ログインし直すことなくアカウント間を切り替えられる体験は必須である。各アカウントのセッションは独立して維持され、UI から明示的にアクティブアカウントを選択できる必要がある。

**Requirement**

- クライアントはメモリ上で複数の active セッションを同時に保持できなければならない（SHALL）。各セッションは一意のアカウント ID と紐づく。
- ログインが成功するたびに、クライアントは新しい独立したセッションペア（アクセストークン＋リフレッシュトークン）を既存セッションリストに追加しなければならない（MUST）。
- クライアントはセッションリストをドメイン状態として管理し、アクティブなセッションを 1 つ選択できなければならない（MUST）。
- 保護された API 呼び出しは、アクティブに選択されたセッションのアクセストークンを `Authorization: Bearer` ヘッダーに使用しなければならない（MUST）。
- アクティブセッションの選択はメモリ上にのみ保持され、永続ストレージや URL、サーバー状態に保存してはならない（MUST NOT）。
- UI は複数の active セッションが存在する場合、アカウント切り替えコントロールを表示しなければならない（MUST）。切り替え操作は再認証を必要としない。
- ログアウトはアクティブに選択されたセッションのみを対象とし、そのセッションのアクセストークンとリフレッシュトークンをメモリから除去し、`POST /api/v1/auth/logout` を実行してサーバーサイドでも失効させなければならない（MUST）。他のセッションは維持される。
- セッションリストにアクティブセッションが 1 つもない場合、クライアントは未認証導線へ遷移しなければならない（MUST）。
- セッション ID、アカウント ID、関連する view model / route state / correlation ID は ULID を使用しなければならない（SHALL）。

#### Scenario: ログイン毎に新しいセッションが追加される (AUTH-FE-S027)

- **GIVEN** 利用者がアカウント A で既にログインしている
- **WHEN** 利用者がアカウント B でログインする
- **THEN** アカウント B のセッションがセッションリストに追加され、アカウント A のセッションは維持される

#### Scenario: アカウント切り替えで API 呼び出しのトークンが変更される (AUTH-FE-S028)

- **GIVEN** クライアントがアカウント A とアカウント B のセッションを保持している
- **WHEN** 利用者が UI でアカウント B をアクティブに選択する
- **THEN** 後続の API 呼び出しはアカウント B のアクセストークンを使用し、再認証は不要である

#### Scenario: 複数セッション存在時にアカウント切り替え UI が表示される (AUTH-FE-S029)

- **GIVEN** クライアントが 2 つ以上の active セッションを保持している
- **WHEN** 認証済みアプリ画面が表示される
- **THEN** UI はアカウント切り替えコントロールを表示する

#### Scenario: 単一セッションのログアウトは他のセッションを維持する (AUTH-FE-S030)

- **GIVEN** クライアントがアカウント A とアカウント B のセッションを保持している
- **WHEN** アカウント A をアクティブにしてログアウトする
- **THEN** アカウント A のセッションはメモリとサーバー両方で失効し、アカウント B のセッションは維持される

#### Scenario: 全セッション消失時は未認証導線へ遷移する (AUTH-FE-S031)

- **GIVEN** クライアントがすべてのセッションをログアウトまたは失効させた
- **WHEN** セッションリストが空になる
- **THEN** クライアントは未認証導線へ自動遷移する

## MODIFIED Requirements

### Requirement: session expiry と logout は未認証導線を明確に分離する

session expiry と logout の導線は、expired / revoked session と missing session を区別しながら未認証状態への復帰導線を SHALL 提供しなければならない。

**Customer Context**

`/*` に入る基盤では、利用者が「未ログインなのか」「認証が切れたのか」を迷わないことが重要です。session expiry と logout の導線が曖昧だと、再認証、公開面への退避、エラー画面 owner の責務が混線します。JWT アクセストークンの導入により、クライアントは期限切れを事前に検知できるようになり、より滑らかなセッション遷移が可能になる。

**Requirement**

- The system SHALL distinguish expired-or-revoked sessions from missing sessions and MUST route logout to an unauthenticated state.
- 現在の session が有効である間、`/*` 上の authenticated navigation は SHALL 継続する。
- 以前は有効だった session が expired または revoked と判定されたとき、client は利用者を `/session-expired` へ SHALL 遷移させる。
- `/session-expired` route の presentation は auth routes から独立した owner に保たれ、Auth コアは redirect trigger と route selection だけを SHALL 担当する。
- 現在の bearer session を持たない初回の `/*` アクセスは、通常の未認証ログイン導線へ SHALL 留まり、`/session-expired` と混同してはならない。
- client は bearer token を in-memory にのみ保持しなければならず（SHALL）、tab または browser を閉じた後の再訪では previously issued bearer session を復元せず、missing session と同じ `unauthenticated` 扱いに SHALL 正規化し、`/session-expired` へ送ってはならない。
- `/logout` route は現在の active セッションを SHALL revoke し、client が保持する bearer-authenticated state を消去し、利用者を public route または login route の非認証状態へ戻す。マルチセッション環境では、logout はアクティブに選択されたセッションのみに適用される。
- `/logout` route は public utility route として存在しても、logout 実行自体は canonical な `POST /api/v1/auth/logout` を呼び出して完了しなければならない（SHALL）。
- logout / expiry handling で client が参照する session ID、account ID、event ID、request ID、notification ID などの識別子が必要な箇所は ULID を SHALL 用い、opaque bearer token や cache key を ULID resource ID と混同してはならない。
- `/logout` 導線は、invite onboarding や権限管理 copy を混在させない抑制された auth presentation を SHALL 保つ。
- JWT アクセストークンの期限切れを検知した場合、クライアントはまず `POST /api/v1/auth/refresh` を試行し、リフレッシュ成功時はセッションを継続し、失敗時のみ `/session-expired` へ遷移しなければならない（MUST）。

#### Scenario: セッション失効時は再認証画面へリダイレクトする (AUTH-FE-S006)

- **GIVEN** 利用者が `/*` 内で操作している
- **WHEN** 現在の session が expired または revoked として報告される
- **THEN** 利用者は `/session-expired` へ遷移し、その後の画面 presentation はその route contract に委ねられる

#### Scenario: logout は利用者を非認証 route へ戻す (AUTH-FE-S007)

- **GIVEN** 利用者が active な authenticated session を持っている
- **WHEN** 利用者が `/logout` を開く
- **THEN** bearer-authenticated state は消去され、利用者は signed in として振る舞わない public route または login route に到達する

#### Scenario: session を持たない `/*` 到達は通常の未認証導線に留まる (AUTH-FE-S008)

- **GIVEN** 利用者が有効な bearer session を持たずに `/*` を開く
- **WHEN** app が current session の不在を検知する
- **THEN** 利用者は通常の login 導線へ進み、`/session-expired` へは遷移しない

#### Scenario: access token 期限切れ時にリフレッシュ成功でセッションを継続する (AUTH-FE-S032)

- **GIVEN** 利用者が操作中でアクセストークンが期限切れになる
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を実行し成功する
- **THEN** 利用者は `/session-expired` へ遷移せず、操作中の画面を継続する

#### Scenario: refresh 失敗時のみ session-expired へ遷移する (AUTH-FE-S033)

- **GIVEN** 利用者が操作中でアクセストークンが期限切れになる
- **WHEN** クライアントが `POST /api/v1/auth/refresh` を実行し失敗する
- **THEN** 利用者は `/session-expired` へ遷移する

### Requirement: suspended account は認証 UI で安全に案内される

顧客向け frontend は HTTP 403 の `AuthFailureResponse` として `error="account-suspended"` を受け取った場合、該当 account の access token / refresh token state を MUST 消去し、アカウントが利用できないこととサポート問い合わせ導線を SHALL 表示する。public login start や recovery request の段階では account existence や suspended 状態を UI に表示してはならない（MUST NOT）。passkey finish 後、refresh 後、または既存 bearer access token の protected API 呼び出しで `account-suspended` を受け取った場合のみ、suspended account 向け案内を表示してよい（SHALL）。複数 account の token pair を保持している場合は該当 account の token pair のみを削除し、他 account の token pair は維持しなければならない（MUST）。

**Customer Context**

停止された顧客はログインできない理由と次の行動を知る必要がある。一方で、ログイン前の public surface で suspended 状態を返すとアカウント有無の推測につながる。credential 所持または既存 token pair が確認された後だけ明確な案内を出すことで、セキュリティと利用者体験を両立する。

#### Scenario: suspended account の passkey login は案内画面に遷移する (AUTH-FE-S041)

- **GIVEN** 利用者が suspended account の valid passkey を持っている
- **WHEN** `/login` で passkey 認証を完了し、API が `account-suspended` を返す
- **THEN** client は access token / refresh token を保存せず、account suspended 案内とサポート問い合わせ導線を表示する

#### Scenario: 既存 token pair が suspended と判定された場合は該当 token state を消去する (AUTH-FE-S042)

- **GIVEN** 利用者が authenticated app route を開いている
- **WHEN** protected API が `account-suspended` を返す
- **THEN** client は該当 account の bearer-authenticated state を消去し、account suspended 案内へ遷移する

#### Scenario: public login start では suspended 状態を推測できない (AUTH-FE-S043)

- **GIVEN** 利用者が `/login` で email または identifier を入力している
- **WHEN** passkey start または recovery request が受理される
- **THEN** UI は account existence や suspended 状態を示す文言を表示しない

#### Scenario: 複数 account の token pair 中の suspended account は対象 token state のみ削除される (AUTH-FE-S044)

- **GIVEN** client が account A と account B の token pair を保持している
- **WHEN** account A の API 呼び出しだけが `account-suspended` を返す
- **THEN** account A の token state は消去され、account B の token state は維持される

### Requirement: 認証済みユーザーはログイン中のデバイスを確認・管理できる

認証済みユーザーは、自身がログイン中のデバイス（セッション）一覧を確認し、特定デバイスまたは他のすべてのデバイスのセッションを無効化できなければならない（SHALL）。

**Customer Context**

利用者は、どのデバイスやブラウザから自分のアカウントにアクセスされているかを把握したい。紛失した端末や公共 PC でのログインを忘れた場合、そのセッションをリモートで無効化できる必要がある。さらに、不審なアクティビティを検知した際に「他のすべてのデバイスをログアウト」して自分が現在使っているデバイスだけを残すことで、迅速にセキュリティを確保したい。デバイス情報はブラウザ名やログイン時刻など、利用者が直感的に判断できる情報を含むべきである。

**Requirement**

- クライアントはデバイス管理ページ（または画面）を提供し、認証済みユーザーが `GET /api/v1/sessions` から取得したログイン中デバイス一覧を表示しなければならない（MUST）。
- デバイス一覧には各セッションのデバイス名（ブラウザ名や OS 名、User-Agent 由来）、ログイン時刻、最終アクティブ時刻、現在のデバイスであるかのインジケーターを含めなければならない（MUST）。
- 各デバイスには「ログアウト」アクションを提供し、クリック時に `DELETE /api/v1/sessions/{id}` を呼び出して該当セッションを無効化しなければならない（MUST）。
- デバイス管理ページには「他のすべてのデバイスをログアウト」ボタンを提供し、クリック時に `DELETE /api/v1/sessions/others` を呼び出して現在のセッション以外を一括無効化しなければならない（MUST）。
- 現在のセッションを無効化した場合、クライアントはそのセッションをメモリから除去し、未認証導線へ遷移しなければならない（MUST）。
- デバイス管理操作の失敗時、クライアントは汎用的なエラーメッセージを表示し、セッション情報やトークンなどの機密データを永続ストレージに残してはならない（MUST NOT）。
- デバイス一覧の取得および無効化操作は、アクティブなアクセストークンを用いて認証済みリクエストとして実行しなければならない（MUST）。

#### Scenario: デバイス管理ページでログイン中のデバイスを確認できる (AUTH-FE-S034)

- **GIVEN** 利用者が複数のデバイスでログインしている
- **WHEN** 利用者がデバイス管理ページを開く
- **THEN** ログイン中のデバイス一覧が表示され、各デバイスの名前、ログイン時刻、最終アクティブ時刻、現在のデバイスかどうかが確認できる

#### Scenario: デバイス管理ページで特定デバイスをログアウトできる (AUTH-FE-S035)

- **GIVEN** 利用者がデバイス管理ページを開いている
- **WHEN** 利用者が特定デバイスの「ログアウト」アクションを実行する
- **THEN** `DELETE /api/v1/sessions/{id}` が呼び出され、該当デバイスのセッションが無効化される。一覧から該当デバイスが除去される

#### Scenario: デバイス管理ページで他のすべてのデバイスをログアウトできる (AUTH-FE-S036)

- **GIVEN** 利用者がデバイス管理ページを開いている
- **WHEN** 利用者が「他のすべてのデバイスをログアウト」ボタンをクリックする
- **THEN** `DELETE /api/v1/sessions/others` が呼び出され、現在のデバイスを除くすべてのセッションが無効化される。一覧からそれらのデバイスが除去され、現在のデバイスのみが残る
