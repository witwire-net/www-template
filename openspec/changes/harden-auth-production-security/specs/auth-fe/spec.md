## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: auth routes は no-store な認証面として配信する

auth routes は、edge / browser cache から stale な認証 UI や session-bound state を再提示しない no-store surface として SHALL 配信されなければならない。

**Customer Context**

Auth コアは auth endpoint だけでなく auth route も no-store scope に含める前提で設計されています。`/login*` や `/logout` が cacheable になると、Cloudflare/WAF 配下や browser の戻る操作で古い認証状態が再表示され、session lifecycle と logout 導線の整合が崩れます。さらに recovery token や device login code を扱う routes は、browser history や Referer に secret を残さない必要があります。

**Requirement**

- システムは `/login`、`/login/recovery*`、`/logout`、`/passkeys/add` を no-store auth routes として配信し、edge/browser cache が stale auth entry state を replay しないようにしなければならない。
- `/login`, `/login/recovery`, `/login/recovery/sent`, `/login/recovery/consume`, `/login/recovery/register`, `/logout`, `/passkeys/add` route responses は auth endpoint と揃った no-store cache semantics を SHALL 保つ。
- auth routes は公開検索面や cacheable bootstrap state に依存せず、直前の login / recovery / logout / device login enablement intent を基準に最新の auth presentation を SHALL 表示する。
- Secret-bearing route input は browser-visible URL、route state、telemetry、persistent storage に必要以上に保持してはならない（MUST NOT）。
- Auth route presentation は account existence、OTP validity、temporary lock state、recovery token state を UI 文言から推測できない generic error semantics を SHALL 維持する。

#### Scenario: auth routes は no-store surface として配信される (AUTH-FE-S009)

- **GIVEN** 利用者が `/login`、`/login/recovery`、`/logout`、または `/passkeys/add` を browser または edge 経由で開く
- **WHEN** auth route response が配信される
- **THEN** system はその route を no-store auth surface として扱い、stale な auth UI や session-bound state を再利用しない

### Requirement: パスキー管理ページから OTP を発行して新端末にパスキーを追加できる

**Customer Context**

利用者が新しい端末でログインできるようにしたい場合、技術用語としての「キー追加」を理解しなくても、既存端末で本人確認を済ませ、登録メールアドレスに届く短いコードを新しい端末で入力できればよい。同時に UI は、メールアドレスやコードの正否、アカウント有無、temporary lock の有無を攻撃者へ示さない必要がある。

**Requirement**

- パスキー管理ページは「新しい端末でログインを有効にする」アクションを SHALL 提供し、既存パスキーによる WebAuthn 再認証を完了してから `POST /api/v1/passkeys/otp` 経由で登録メールアドレスへの OTP 送信を依頼しなければならない。再認証成功後に取得した reauthentication session ID は `X-Reauth-Session` HTTP header に設定して送信する。
- UI は「パスキーを追加」や「credential」などの技術用語を主要 action label として使用せず、新しい端末でログインできるようにする目的を SHALL 表示する。
- OTP は管理ページ上に表示してはならず、backend から `issued: true` の acknowledgement を受け取ったら、UI は登録メールアドレスへ送信されたこと、有効期限、登録メールアドレスと一緒に入力する必要があること、第三者に共有しないことを SHALL 案内する。平文 OTP を API response や UI に表示してはならない。
- 新端末向けログイン有効化ページ（`/passkeys/add`）は未認証 surface として SHALL 提供されなければならない。登録メールアドレスと OTP 入力フォームを表示し、有効な組み合わせの入力後に WebAuthn 登録フローを完了しなければならない。
- 新端末向け登録フローでエラーが発生した場合は、email の登録有無、OTP の正否、temporary lock の有無を示さない generic エラーメッセージを SHALL 表示しなければならない。
- `/passkeys/add` は no-store な auth route として扱われ、入力された email / OTP を persistent storage、telemetry、URL query に MUST 保存しない。
- 端末または passkey credential の削除 UI は、削除前に既存パスキーによる WebAuthn 再認証を要求し、再認証成功後に `X-Reauth-Session` header を付与して削除 request を送信し、再認証が失敗・期限切れ・消費済みの場合は削除を開始してはならない。
- WebAuthn assertion request（login）と attestation request（registration）は `userVerification: 'required'` を MUST 使用し、`preferred` や `discouraged` は廃止する。
- 管理ページは `WitWire` ブランドシステム（M PLUS 1 / Noto Sans JP / IBM Plex Mono、token-based color、8px grid、Flat & Bright、shadow/glow 禁止）に SHALL 準拠する。

#### Scenario: パスキー管理ページで OTP を発行できる (AUTH-FE-S016)

- **GIVEN** 利用者が認証済み状態でパスキー管理ページを開いている
- **WHEN** 「新しい端末でログインを有効にする」アクションを起動する
- **THEN** UI は WebAuthn 再認証を求め、成功後に登録メールアドレスへ OTP を送信し、OTP を画面表示せず、有効期限、登録メールアドレスと一緒に入力する案内、共有禁止の案内を表示する

#### Scenario: 新端末パスキー登録ページで有効な OTP を入力してパスキーを登録できる (AUTH-FE-S017)

- **GIVEN** 利用者が新端末のログイン有効化ページ（`/passkeys/add`）を開いている
- **WHEN** 登録メールアドレスと有効な OTP を入力して WebAuthn 登録フローを完了する
- **THEN** その端末でログインできる状態になり、登録完了メッセージが表示される

#### Scenario: 新端末パスキー登録ページで無効な OTP を入力した場合はエラーが表示される (AUTH-FE-S018)

- **GIVEN** 利用者が新端末のログイン有効化ページを開いている
- **WHEN** 無効・期限切れ・消費済み・locked の email + OTP 組み合わせで登録を試みる
- **THEN** generic エラーメッセージが表示され、登録フローは開始されない

#### Scenario: email と OTP は new-device page で永続化されない (AUTH-FE-S021)

- **GIVEN** 利用者が `/passkeys/add` で登録メールアドレスと OTP を入力している
- **WHEN** 登録フローが成功または失敗する
- **THEN** client は email と OTP を session storage、local storage、URL query、telemetry payload に保存しない

#### Scenario: passkey deletion は UI で再認証を要求する (AUTH-FE-S022)

- **GIVEN** 利用者が認証済み状態でパスキー管理ページを開いている
- **WHEN** 利用者が登録済み端末または passkey credential の削除を実行する
- **THEN** UI は WebAuthn 再認証を完了してから削除 request を送信し、再認証できない場合は削除を開始しない
