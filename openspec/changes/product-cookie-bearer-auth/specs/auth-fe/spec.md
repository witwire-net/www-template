## MODIFIED Requirements

### Requirement: 低強調のパスキーログイン導線を提供する

低強調のパスキーログイン導線は、公開面の主要 CTA を保ったまま passkey-only な `/login` へ到達できる体験を SHALL 提供しなければならない。

**Customer Context**

公開面は社外向けの発信サイトとして見せつつ、社内利用者は同じドメインから認証面へ入れる必要がある。利用者はパスキーだけでログインし、ログイン後は JavaScript 可読 token を扱わず、HttpOnly Cookie によって安全に app surface へ進める必要がある。

**Requirement**

- The system SHALL keep the public-to-login handoff low emphasis and MUST present `/login` as a passkey-only sign-in surface.
- public surface のログイン handoff は、補助ナビゲーションまたはフッターに留まる低強調導線を SHALL 保ち、主要 CTA や hero action に拡張してはならない。
- `/login` route は、passkey sign-in action、`/login/recovery` への recovery link、公開面へ戻る導線を持つ passkey 専用 sign-in 画面を SHALL 表示する。
- `/login` route で認証が成功した後の client auth state は、HttpOnly Cookie credential と session-bound CSRF token で `/api/v1/*` を利用できる Web Cookie session 契約に SHALL 接続される。
- `/login` route の client request は `credentialMode="web-cookie"` を SHALL 明示し、response body の bearer accessToken または refreshToken に依存してはならない（MUST NOT）。
- client が保持・表示・遷移判定に使う `accountId`、`sessionId`、`passkeyCredentialId`、`recoverySessionId`、および auth notification / audit / action correlation ID など system-owned resource ID が必要な箇所は ULID を SHALL 前提とする。
- `/login` route は、password input、password reset copy、invite registration control、Guest onboarding copy を表示してはならない（MUST NOT）。
- 認証導線の画面は、`WitWire` 表記、M PLUS 1 / Noto Sans JP / IBM Plex Mono、token-based color、8px grid、Flat & Bright、shadow / glow 禁止の brand system に SHALL 準拠する。

#### Scenario: 公開面の低強調 handoff から `/login` へ到達する (AUTH-FE-S001)

- **GIVEN** 利用者が低強調ログイン handoff を持つ公開ページを閲覧している
- **WHEN** 利用者がその handoff から認証導線へ遷移する
- **THEN** 利用者は主要 CTA を増やさずに `/login` へ到達する

#### Scenario: ログイン画面はパスキー専用でサインインを提供する (AUTH-FE-S002)

- **GIVEN** 利用者が `/login` を開く
- **WHEN** ログイン画面が読み込まれる
- **THEN** 画面には passkey sign-in action と recovery link が表示され、password entry や invite registration control は表示されない

#### Scenario: Web login は HttpOnly Cookie session と CSRF token を受け入れる (AUTH-FE-S045)

- **GIVEN** 利用者が `/login` で passkey sign-in を完了する
- **WHEN** API が Web Cookie mode の session response を返す
- **THEN** client は bearer accessToken を保存せず、session metadata と CSRF token を memory state に保持して authenticated state へ遷移する

### Requirement: 復旧導線は既存アカウントの passkey 再登録だけを扱う

復旧導線とデバイスリンク導線は、既存アカウントの passkey 登録だけを招待導線や規約同意導線と分離したまま SHALL 提供しなければならない。トークンの kind に応じて、復旧用と新端末追加用で完了後の体験を分岐する。

**Customer Context**

パスキーを紛失した利用者（kind=recovery）と、新端末でログインを有効にしたい利用者（kind=device-link）は、どちらも invite onboarding へ戻らずに、登録済みアカウントの passkey を安全に登録できる必要がある。登録成功後は Web Cookie session としてログイン状態へ進み、JavaScript 可読 token を画面・state・telemetry に残さない必要がある。

**Requirement**

- The system SHALL provide a recovery/device-link route family for existing accounts and MUST keep invite onboarding and consent flows out of these routes.
- `/login/recovery` route は登録済みメールアドレスを受け取り、受理された依頼を `/login/recovery/sent` へ接続する recovery request を SHALL 送信できる。
- `/login/recovery/sent` route は recovery URL を送信したことを SHALL 表示し、`/login` へ戻る導線を SHALL 提供する。
- recovery request の結果表示は、アカウント有無、送信抑止、temporary lock を UI から判別できない共通の受理メッセージを SHALL 保つ。
- `/login/recovery/consume` route は token を検証し、token の `kind` に応じて遷移先を分岐しなければならない（SHALL）。`kind=recovery` の場合は `/login/recovery/register` へ、`kind=device-link` の場合はデバイスリンク登録画面へ遷移する。
- `/login/recovery/register` route は recovery/device-link branch のみを使って、既存アカウントに対する passkey 登録を SHALL 完了できる。登録完了後、kind に応じた完了メッセージを表示しなければならない。
- 登録成功後の client auth state は、HttpOnly Cookie credential と session-bound CSRF token で `/api/v1/*` を利用できる Web Cookie session 契約に SHALL 接続される。
- recovery/device-link registration request は `credentialMode="web-cookie"` を SHALL 明示し、response body の bearer accessToken または refreshToken に依存してはならない（MUST NOT）。
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

#### Scenario: 復旧登録成功時は Web Cookie session としてログインする (AUTH-FE-S046)

- **GIVEN** 利用者が recovery/device-link branch の passkey registration を完了する
- **WHEN** API が Web Cookie mode の session response を返す
- **THEN** client は bearer accessToken を保存せず、session metadata と CSRF token を memory state に保持して authenticated state へ遷移する

### Requirement: クライアントは JWT アクセストークンの有効期限を監視し自動更新する

クライアントは Web Cookie session の有効性を server response と refresh endpoint で確認し、JavaScript から access credential を読まずに session を継続しなければならない（MUST）。

**Customer Context**

利用者はアプリ操作中に認証が突然切れてデータを失う体験を避けたい。Web Cookie session では access credential が HttpOnly であるため、frontend は JWT payload を読まず、API response と refresh endpoint を使って安全に session 継続可否を判断する必要がある。

**Requirement**

- クライアントは Web Cookie mode の access credential をデコードしてはならない（MUST NOT）。
- クライアントは authenticated app bootstrap 時、HttpOnly refresh Cookie による `POST /api/v1/auth/refresh` を same-origin credential request として実行し、成功時は session metadata、CSRF token、AccountSetting snapshot を memory state に反映しなければならない（MUST）。
- 保護された API 呼び出しが `session-expired` を返した場合、クライアントは同一操作について 1 回だけ `POST /api/v1/auth/refresh` を試行し、成功時は CSRF token を更新して元の API 呼び出しを再試行しなければならない（SHALL）。
- refresh が `unauthenticated` を返した場合、クライアントは missing session として通常のログイン導線へ正規化し、`/session-expired` へ遷移してはならない（MUST NOT）。
- refresh が `session-expired`、`account-suspended`、または fail-close error を返した場合、クライアントは該当 route intent へ遷移しなければならない（MUST）。
- クライアントは bearer accessToken、refreshToken、HttpOnly Cookie value を localStorage、sessionStorage、IndexedDB、URL、telemetry、console output に保存してはならない（MUST NOT）。
- CSRF token は session-bound non-cookie secret として memory state にのみ保持し、state-changing Cookie request の `X-CSRF-Token` header に設定しなければならない（MUST）。

#### Scenario: 期限切れ間近のアクセストークンは自動リフレッシュされる (AUTH-FE-S023)

- **GIVEN** Web Cookie session の access credential が server-side で期限切れとして扱われる
- **WHEN** 保護された API 呼び出しが `session-expired` を返す
- **THEN** クライアントは `POST /api/v1/auth/refresh` を実行し、成功後に同じ API 呼び出しを再試行する

#### Scenario: 既に期限切れのアクセストークンはリフレッシュ後に API を呼び出す (AUTH-FE-S024)

- **GIVEN** Web Cookie session の access credential が期限切れで refresh credential は有効である
- **WHEN** 保護された API 呼び出しが必要になる
- **THEN** クライアントは refresh 成功後に最新 CSRF token を使って API 呼び出しを継続し、リフレッシュ失敗時は route intent を更新する

#### Scenario: トークンは永続ストレージに保存されない (AUTH-FE-S025)

- **GIVEN** 利用者がログインして Web Cookie mode の session response を受け取る
- **WHEN** 認証 state がクライアントに保存される
- **THEN** bearer accessToken、refreshToken、Cookie value は永続ストレージに書き込まれず、CSRF token と session metadata だけが memory state に保持される

#### Scenario: ブラウザ再訪時は未認証状態に正規化される (AUTH-FE-S026)

- **GIVEN** 利用者が同じ URL を開き直す
- **WHEN** app bootstrap の refresh が missing session として失敗する
- **THEN** クライアントは通常の login 導線へ進み、`/session-expired` へは遷移しない

#### Scenario: Cookie bootstrap refresh 成功時は session を復元する (AUTH-FE-S047)

- **GIVEN** browser が valid な HttpOnly refresh Cookie を保持している
- **WHEN** app bootstrap が `POST /api/v1/auth/refresh` を実行する
- **THEN** client は session metadata、CSRF token、AccountSetting snapshot を memory state に反映し、authenticated state として表示する

### Requirement: クライアントは複数アカウントのセッションを同時に保持・切り替えできる

クライアントは Product Web の Cookie session を 1 つの active session として扱い、browser-readable credential を使った複数 session switching を提供してはならない（MUST NOT）。

**Customer Context**

HttpOnly Cookie session は JavaScript から credential を読ませないことで安全性を高める。一方で、同一 origin の Cookie は browser 内で自動送信されるため、frontend memory に複数の bearer token を保持して切り替える方式とは性質が異なる。利用者に誤ったアカウント切り替え体験を提供しないため、Product Web は server が認識する 1 つの active Cookie session を基準に表示と API 呼び出しを行う必要がある。

**Requirement**

- クライアントは Product Web 上で bearer accessToken を複数保持して account switching に使ってはならない（MUST NOT）。
- ログイン成功、recovery registration 成功、bootstrap refresh 成功のいずれの場合も、クライアントは 1 つの active Web Cookie session metadata を memory state に保持しなければならない（MUST）。
- 追加のログインが同じ browser context で成功した場合、クライアントは active Web Cookie session metadata を新しい session metadata に置き換えなければならない（MUST）。
- UI は Product Web Cookie session で複数アカウント切り替えコントロールを表示してはならない（MUST NOT）。
- logout は active Web Cookie session のみを対象とし、server-side revoke と Cookie clear response を受けた後、memory state を未認証状態へ戻さなければならない（MUST）。

#### Scenario: ログイン成功時は active Cookie session metadata を置き換える (AUTH-FE-S027)

- **GIVEN** 利用者が Product Web で authenticated state にいる
- **WHEN** 同じ browser context で別のログインが成功する
- **THEN** client は active session metadata を新しい session metadata に置き換え、複数 bearer session list を保持しない

#### Scenario: API 呼び出しは Cookie と CSRF token を使用する (AUTH-FE-S028)

- **GIVEN** Product Web が active Web Cookie session を保持している
- **WHEN** 保護された API が呼び出される
- **THEN** request は browser が送信する HttpOnly Cookie と memory state の CSRF token を使い、UI 上の bearer token switching に依存しない

#### Scenario: Product Web ではアカウント切り替え UI を表示しない (AUTH-FE-S029)

- **GIVEN** Product Web が authenticated state にいる
- **WHEN** 認証済みアプリ画面が表示される
- **THEN** UI は複数アカウント切り替えコントロールを表示しない

#### Scenario: logout は active Cookie session を失効する (AUTH-FE-S030)

- **GIVEN** Product Web が active Web Cookie session を保持している
- **WHEN** 利用者がログアウトする
- **THEN** client は active session を server と memory state の両方で失効し、未認証導線へ遷移する

#### Scenario: 全セッション消失時は未認証導線へ遷移する (AUTH-FE-S031)

- **GIVEN** active Web Cookie session が logout または失効によって存在しなくなる
- **WHEN** client の session metadata が空になる
- **THEN** クライアントは未認証導線へ自動遷移する

### Requirement: session expiry と logout は未認証導線を明確に分離する

session expiry と logout の導線は、expired / revoked session と missing session を区別しながら未認証状態への復帰導線を SHALL 提供しなければならない。

**Customer Context**

`/*` に入る基盤では、利用者が「未ログインなのか」「認証が切れたのか」を迷わないことが重要である。Cookie session では frontend が access credential を読めないため、server response、refresh outcome、logout outcome によって route intent を正規化する必要がある。

**Requirement**

- The system SHALL distinguish expired-or-revoked sessions from missing sessions and MUST route logout to an unauthenticated state.
- 現在の Web Cookie session が有効である間、`/*` 上の authenticated navigation は SHALL 継続する。
- 以前は有効だった session が expired または revoked と判定され、refresh でも継続できないとき、client は利用者を `/session-expired` へ SHALL 遷移させる。
- `/session-expired` route の presentation は auth routes から独立した owner に保たれ、Auth コアは redirect trigger と route selection だけを SHALL 担当する。
- 現在の session を持たない初回の `/*` アクセスは、bootstrap refresh の missing session 結果を通常の未認証ログイン導線へ SHALL 正規化し、`/session-expired` と混同してはならない。
- `/logout` route は現在の active Web Cookie session を SHALL revoke し、client が保持する session metadata と CSRF token を消去し、利用者を public route または login route の非認証状態へ戻す。
- `/logout` route は public utility route として存在しても、logout 実行自体は canonical な `POST /api/v1/auth/logout` を same-origin credential request として呼び出して完了しなければならない（SHALL）。
- logout / expiry handling で client が参照する session ID、account ID、event ID、request ID、notification ID などの識別子が必要な箇所は ULID を SHALL 用い、opaque credential や cache key を ULID resource ID と混同してはならない。
- `/logout` 導線は、invite onboarding や権限管理 copy を混在させない抑制された auth presentation を SHALL 保つ。

#### Scenario: セッション失効時は再認証画面へリダイレクトする (AUTH-FE-S006)

- **GIVEN** 利用者が `/*` 内で操作している
- **WHEN** 現在の session が expired または revoked として報告され、refresh でも継続できない
- **THEN** 利用者は `/session-expired` へ遷移し、その後の画面 presentation はその route contract に委ねられる

#### Scenario: logout は利用者を非認証 route へ戻す (AUTH-FE-S007)

- **GIVEN** 利用者が active な Web Cookie session を持っている
- **WHEN** 利用者が `/logout` を開く
- **THEN** Cookie session は revoke され、client memory の session metadata と CSRF token は消去され、利用者は signed in として振る舞わない public route または login route に到達する

#### Scenario: session を持たない `/*` 到達は通常の未認証導線に留まる (AUTH-FE-S008)

- **GIVEN** 利用者が有効な session を持たずに `/*` を開く
- **WHEN** app bootstrap refresh が missing session を検知する
- **THEN** 利用者は通常の login 導線へ進み、`/session-expired` へは遷移しない

### Requirement: 認証済みユーザーはアプリ内でパスキーを一覧・追加・削除できる

利用者は MacBook・iPhone・セキュリティキーなど複数のデバイスでパスキーを使い分けたい。また、古いデバイスや紛失したデバイスのパスキーを削除して鍵を整理したい。登録済みパスキーを確認・追加・削除できる管理画面と、新しい端末でログインを有効にするためのデバイスリンク送信機能を Product Web は SHALL 提供する。

認証済み Product Web 利用者は、active Web Cookie session と CSRF token によって、パスキー一覧・追加・削除・デバイスリンク送信を実行できなければならない（SHALL）。

**Requirement**

- システムは認証済みアプリ内にパスキー管理ページを SHALL 提供し、登録済みのすべての passkey credential の識別子と登録日時を一覧表示しなければならない。
- パスキー管理ページは新しいパスキーを追加する WebAuthn 登録フローを SHALL 提供しなければならない（`POST /api/v1/passkeys/start` → `POST /api/v1/passkeys/finish`）。
- パスキー管理ページは「新しい端末でログインを有効にする」アクションを SHALL 提供しなければならない。このアクションは既存パスキーによる WebAuthn 再認証を完了してから `POST /api/v1/passkeys/send-device-link` 経由で登録メールアドレスへのデバイスリンク送信を依頼する。再認証成功後に取得した reauthentication session ID は `X-Reauth-Session` HTTP header に設定して送信する。
- パスキー管理 API 呼び出しは same-origin credential request として HttpOnly Cookie を送信し、state-changing request には memory state の CSRF token を `X-CSRF-Token` header に設定しなければならない（MUST）。
- UI は「パスキーを追加」や「credential」などの技術用語を主要 action label として使用せず、新しい端末でログインできるようにする目的を SHALL 表示する。
- デバイスリンク送信後、UI は登録メールアドレスへ送信されたこと、有効期限（30分）、第三者に共有しないことを SHALL 案内する。平文トークンは API response や UI に表示してはならない。
- パスキー管理ページは指定したパスキーを削除するアクションを SHALL 提供しなければならない（`DELETE /api/v1/passkeys/{id}`）。
- passkey credential が 1 件しかない場合、削除アクションは SHALL 無効化または非表示にしなければならない。
- パスキー追加フロー、デバイスリンク送信フロー、または削除フローでエラーが発生した場合は、エラーメッセージを SHALL 表示し、ページ状態を保持しなければならない。
- 管理ページは active Web Cookie session を必須とする認証済み surface であり、未認証アクセスは SHALL 拒否されなければならない。
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
- **THEN** CSRF header 付き Cookie request で削除 API が呼び出され、そのパスキーが一覧から削除され、残りは変化しない

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
- **THEN** UI は WebAuthn 再認証を求め、成功後に CSRF header 付き Cookie request で登録メールアドレスへデバイスリンクを送信し、送信完了・有効期限・共有禁止の案内を表示する

#### Scenario: WebAuthn assertion request と attestation request は userVerification required を使用する (AUTH-FE-S037)

- **GIVEN** login または registration ceremony が開始されている
- **WHEN** client が WebAuthn options を構築する
- **THEN** `userVerification` は `"required"` に設定され、`"preferred"` や `"discouraged"` は使用されない

### Requirement: 認証 UI は secret leakage を抑える security presentation を提供する

認証 UI は、bearer token、refresh token、HttpOnly Cookie value、CSRF token、recovery token、WebAuthn raw credential data の露出を最小化しなければならない（MUST）。

**Customer Context**

認証画面では recovery token、device login code、CSRF token などの機微情報が一時的に扱われる。利用者が browser history、Referer、画面共有、戻る操作、キャッシュ、XSS の影響を受けにくい体験を得るためには、UI と client state が secret を長く保持せず、表示やエラー文言から認証状態を推測できない必要がある。

**Requirement**

- システムは auth UI state、navigation state、history、telemetry、visible error message における secret exposure を最小化しなければならない。
- `/login/recovery/consume` route は URL から recovery token を読み取った後、token を browser-visible URL から SHALL 即時除去する。
- Auth UI は recovery token、OTP、bearer token、refresh token、HttpOnly Cookie value、WebAuthn raw credential data を telemetry attribute、console output、visible debug UI、persistent route state に MUST 保存しない。
- Auth routes と protected app routes は no-store と、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy`、`X-Content-Type-Options`、frame embedding prevention を含む browser security headers を持つ形で配信または deployment-configure されなければならない。
- Client auth state persistence は browser-readable session credential の漏えいリスクを最小化し、HttpOnly Cookie value を JavaScript state へ複製してはならない（MUST NOT）。
- CSRF token は memory state にのみ保持し、persistent storage、URL、telemetry、console output に保存してはならない（MUST NOT）。
- Secret-bearing UI は copy/paste や画面表示が必要な場合でも TTL、用途、再発行導線、共有禁止の案内を SHALL 表示する。

#### Scenario: recovery token は browser-visible URL から除去される (AUTH-FE-S019)

- **GIVEN** 利用者が recovery token 付き URL で `/login/recovery/consume` を開いている
- **WHEN** client が token を読み取って consume request を開始する
- **THEN** token は browser address bar と subsequent navigation URL から除去される

#### Scenario: auth routes は security headers と no-store semantics を持つ (AUTH-FE-S020)

- **GIVEN** 利用者が `/login`、`/login/recovery*`、`/logout`、または authenticated app route を開いている
- **WHEN** route response が配信される
- **THEN** route は no-store と browser security header semantics を持ち、stale auth UI や secret-bearing URL を replay しない

#### Scenario: Web auth state は bearer token と Cookie value を保存しない (AUTH-FE-S048)

- **GIVEN** Web Cookie mode の session response が返っている
- **WHEN** client が auth state を更新する
- **THEN** bearer accessToken、refreshToken、HttpOnly Cookie value は state に保存されず、CSRF token と session metadata だけが memory state に残る

### Requirement: suspended account は認証 UI で安全に案内される

顧客向け frontend は HTTP 403 の `AuthFailureResponse` として `error="account-suspended"` を受け取った場合、該当 account の Web Cookie session state を MUST 消去し、アカウントが利用できないこととサポート問い合わせ導線を SHALL 表示する。public login start や recovery request の段階では account existence や suspended 状態を UI に表示してはならない（MUST NOT）。

**Customer Context**

停止された顧客はログインできない理由と次の行動を知る必要がある。一方で、ログイン前の public surface で suspended 状態を返すとアカウント有無の推測につながる。credential 所持または既存 session が確認された後だけ明確な案内を出すことで、セキュリティと利用者体験を両立する。

#### Scenario: suspended account の passkey login は案内画面に遷移する (AUTH-FE-S041)

- **GIVEN** 利用者が suspended account の valid passkey を持っている
- **WHEN** `/login` で passkey 認証を完了し、API が `account-suspended` を返す
- **THEN** client は session metadata と CSRF token を保存せず、account suspended 案内とサポート問い合わせ導線を表示する

#### Scenario: 既存 token pair が suspended と判定された場合は該当 token state を消去する (AUTH-FE-S042)

- **GIVEN** 利用者が authenticated app route を開いている
- **WHEN** protected API または refresh が `account-suspended` を返す
- **THEN** client は該当 account の Web Cookie session state を消去し、account suspended 案内へ遷移する

#### Scenario: public login start では suspended 状態を推測できない (AUTH-FE-S043)

- **GIVEN** 利用者が `/login` で email または identifier を入力している
- **WHEN** passkey start または recovery request が受理される
- **THEN** UI は account existence や suspended 状態を示す文言を表示しない

#### Scenario: 複数 account の token pair 中の suspended account は対象 token state のみ削除される (AUTH-FE-S044)

- **GIVEN** Product Web が active Web Cookie session を保持している
- **WHEN** その session の API 呼び出しだけが `account-suspended` を返す
- **THEN** client は active session state を消去し、他の browser-readable token pair を保持または切り替えない

### Requirement: 認証済みユーザーはログイン中のデバイスを確認・管理できる

認証済みユーザーは、自身がログイン中のデバイス（セッション）一覧を確認し、特定デバイスまたは他のすべてのデバイスのセッションを無効化できなければならない（SHALL）。

**Customer Context**

利用者は、どのデバイスやブラウザから自分のアカウントにアクセスされているかを把握したい。紛失した端末や公共 PC でのログインを忘れた場合、そのセッションをリモートで無効化できる必要がある。Web Cookie session では state-changing 操作を CSRF header 付き same-origin credential request として実行し、安全に session を管理する必要がある。

**Requirement**

- クライアントはデバイス管理ページ（または画面）を提供し、認証済みユーザーが `GET /api/v1/sessions` から取得したログイン中デバイス一覧を表示しなければならない（MUST）。
- デバイス一覧には各セッションのデバイス名（ブラウザ名や OS 名、User-Agent 由来）、ログイン時刻、最終アクティブ時刻、現在のデバイスであるかのインジケーターを含めなければならない（MUST）。
- 各デバイスには「ログアウト」アクションを提供し、クリック時に `DELETE /api/v1/sessions/{id}` を CSRF header 付き Cookie request として呼び出して該当セッションを無効化しなければならない（MUST）。
- デバイス管理ページには「他のすべてのデバイスをログアウト」ボタンを提供し、クリック時に `DELETE /api/v1/sessions/others` を CSRF header 付き Cookie request として呼び出して現在のセッション以外を一括無効化しなければならない（MUST）。
- 現在のセッションを無効化した場合、クライアントはそのセッション metadata と CSRF token を memory state から除去し、未認証導線へ遷移しなければならない（MUST）。
- デバイス管理操作の失敗時、クライアントは汎用的なエラーメッセージを表示し、セッション情報や credential secret などの機密データを永続ストレージに残してはならない（MUST NOT）。
- デバイス一覧の取得および無効化操作は、active Web Cookie session を用いて認証済みリクエストとして実行しなければならない（MUST）。

#### Scenario: デバイス管理ページでログイン中のデバイスを確認できる (AUTH-FE-S034)

- **GIVEN** 利用者が複数のデバイスでログインしている
- **WHEN** 利用者がデバイス管理ページを開く
- **THEN** ログイン中のデバイス一覧が表示され、各デバイスの名前、ログイン時刻、最終アクティブ時刻、現在のデバイスかどうかが確認できる

#### Scenario: デバイス管理ページで特定デバイスをログアウトできる (AUTH-FE-S049)

- **GIVEN** 利用者がデバイス管理ページを開いている
- **WHEN** 利用者が特定デバイスの「ログアウト」アクションを実行する
- **THEN** `DELETE /api/v1/sessions/{id}` が CSRF header 付き Cookie request として呼び出され、該当デバイスのセッションが無効化される。一覧から該当デバイスが除去される

#### Scenario: デバイス管理ページで他のすべてのデバイスをログアウトできる (AUTH-FE-S036)

- **GIVEN** 利用者がデバイス管理ページを開いている
- **WHEN** 利用者が「他のすべてのデバイスをログアウト」ボタンをクリックする
- **THEN** `DELETE /api/v1/sessions/others` が CSRF header 付き Cookie request として呼び出され、現在のデバイスを除くすべてのセッションが無効化される。一覧からそれらのデバイスが除去され、現在のデバイスのみが残る
