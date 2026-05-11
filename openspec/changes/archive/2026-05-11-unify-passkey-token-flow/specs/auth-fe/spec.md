## REMOVED Requirements

### Requirement: パスキー管理ページから OTP を発行して新端末にパスキーを追加できる

**Reason**: OTP 方式は 6 桁の数字コードを手入力する UX であり、URL トークンをクリックする方式に比べてユーザビリティが低い。また OTP の低エントロピー（~20bit）はセキュリティ上も劣る。新端末追加は kind=device-link の URL トークン方式に一本化される。

**Migration**: `/passkeys/add` ルートは削除される。パスキー管理画面の「新しい端末でログインを有効にする」は reauth → send-device-link に置き換わる。新端末ではデバイスリンクメールの URL をクリックすることでパスキー登録が可能になる。

## MODIFIED Requirements

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
