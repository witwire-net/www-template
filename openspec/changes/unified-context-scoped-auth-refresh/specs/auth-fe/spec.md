## MODIFIED Requirements

### Requirement: クライアントは JWT アクセストークンの有効期限を監視し自動更新する

クライアントは Product Web の各 session item について、authContextId、identity/session metadata、short-lived accessToken を memory-only で保持しなければならない（MUST）。refreshToken と Cookie value は memory、localStorage、sessionStorage、IndexedDB、URL、telemetry、console output に保存してはならない（MUST NOT）。Protected API 呼び出しは active session item の accessToken を `Authorization: Bearer <accessToken>` header に入れ、`authContextId` を protected API header に使ってはならない（MUST NOT）。accessToken が `session-expired` になった場合、クライアントは active `authContextId` の `POST /api/v1/auth/contexts/{authContextId}/refresh` を same-origin credential request として 1 回だけ実行し、成功したら新 accessToken で元 API を 1 回だけ retry しなければならない（SHALL）。browser reload 後の context discovery は Product origin の `localStorage` に保存した non-secret context index だけを使い、index entry は refresh 成功まで authenticated state として扱ってはならない（MUST NOT）。

**Customer Context**

利用者は複数アカウントを切り替えながら作業したい。一方で refreshToken を JavaScript から読める場所に置くと XSS 時の被害が大きい。accessToken は短命かつ memory-only に限定し、refresh 対象は Cookie Path と authContextId path parameter で選ぶことで、利用者の切り替え操作と refresh credential の所属を一致させる。

#### Scenario: 期限切れ accessToken は active authContextId の Cookie refresh で更新される (AUTH-FE-S045)

- **GIVEN** active session の accessToken が期限切れとして protected API から `session-expired` を返されている
- **WHEN** クライアントが session continuation を試行する
- **THEN** クライアントは `POST /api/v1/auth/contexts/{authContextId}/refresh` を same-origin credential request として 1 回だけ呼び出す
- **AND** 成功時は response body の新しい accessToken を active session item に反映し、元 API を 1 回だけ retry する

#### Scenario: refreshToken はブラウザーから読める storage に保存されない (AUTH-FE-S046)

- **GIVEN** 利用者が login、setup/register、または context refresh に成功する
- **WHEN** client auth state、localStorage、sessionStorage、IndexedDB、URL state、telemetry payload を確認する
- **THEN** refreshToken 平文と Cookie value は存在せず、accessToken、authContextId、identity/session metadata だけが memory state に存在する

#### Scenario: refresh 失敗時は対象 session だけを失効扱いにする (AUTH-FE-S047)

- **GIVEN** クライアントが複数 session item を保持している
- **WHEN** 1 つの session の context refresh request が失敗する
- **THEN** クライアントは対象 session だけを失効扱いにし、他 session の metadata と accessToken は維持する

#### Scenario: frontend は Cookie header を個別選択しない (AUTH-FE-S055)

- **GIVEN** browser が複数の path-scoped refresh Cookie を保持している
- **WHEN** クライアントが context refresh を実行する
- **THEN** JavaScript は `Cookie` header を組み立てず、URL path の `authContextId` だけで refresh 対象を選び、Cookie Path による browser 送信境界へ委ねる

#### Scenario: non-secret context index から bootstrap する (AUTH-FE-S056)

- **GIVEN** browser reload により memory auth state が消えている
- **WHEN** Product Web が browser-readable context index を読み取る
- **THEN** index は authContextId と非 secret metadata だけを含み、token/secret/Cookie value を含まない
- **AND** クライアントは各候補の context refresh を fail-close で検証し、tamper された entry を authenticated session として採用しない

#### Scenario: context index は origin-local localStorage に限定される (AUTH-FE-S058)

- **GIVEN** Product Web が login、context refresh、logout、または revoke result を処理している
- **WHEN** context index を更新する
- **THEN** クライアントは Product origin の `localStorage` の service-specific key だけを更新する
- **AND** entry には version、authContextId、sessionId、display hint、lastSeenAt、expiresHintAt だけを保存し、accessToken、refreshToken、Cookie value を保存しない
- **AND** 同一 origin の他 tab には `storage` event または `BroadcastChannel` で add/remove/active change を伝搬する

#### Scenario: context index cleanup は logout と refresh failure に追従する (AUTH-FE-S059)

- **GIVEN** Product Web が複数 context の index entries を持っている
- **WHEN** logout、session revoke、suspend、または context refresh failure が特定 authContextId に対して発生する
- **THEN** クライアントは対象 authContextId の entry だけを削除し、他 context の entries を維持する
- **AND** all-context revoke response の場合は server が返した対象 entries をすべて削除する
- **AND** 複数 tab で cleanup 競合が発生した場合、server refresh result を正として stale entry を再採用しない

### Requirement: クライアントは複数アカウントのセッションを同時に保持・切り替えできる

クライアントは Product Web で複数 account/session context を同時に保持・切り替えできなければならない（SHALL）。各 session item は authContextId、identity/session metadata、short-lived accessToken を memory-only に保持しなければならない（MUST）。Account switching は frontend が active session item を選ぶことで実行し、protected API は選択中 item の accessToken を `Authorization` header に入れなければならない（MUST）。authContextId は context refresh URL construction、session metadata、UI selection、bootstrap 用 non-secret context index にだけ使い、protected API header に使ってはならない（MUST NOT）。

**Customer Context**

複数アカウントを運用する利用者にとって、都度ログインし直さず account を切り替えられる体験は必須である。frontend が active account を明示的に制御できないと、誤った account への操作や refresh 対象の混同が起きる。

#### Scenario: ログイン成功時は session item を追加し active にする (AUTH-FE-S048)

- **GIVEN** 利用者が account A で既に login している
- **WHEN** 利用者が account B で login する
- **THEN** クライアントは account B の authContextId、metadata、accessToken を session list に追加し、active session item として選択する
- **AND** account A の session item は維持される

#### Scenario: Product Web ではアカウント切り替え UI を表示する (AUTH-FE-S049)

- **GIVEN** Product Web が複数 session item を保持している
- **WHEN** 認証済みアプリ画面が表示される
- **THEN** UI は複数アカウント切り替えコントロールを表示し、選択した session item を active として扱う

#### Scenario: protected API は active accessToken を使う (AUTH-FE-S057)

- **GIVEN** active session item が選択されている
- **WHEN** クライアントが Product protected API を呼び出す
- **THEN** request は `Authorization: Bearer <active accessToken>` を持つ
- **AND** request は `X-Auth-Context-Id` header または CSRF header を持たない

#### Scenario: logout は active session と対応 refresh Cookie clear を依頼する (AUTH-FE-S050)

- **GIVEN** クライアントが account A と account B の session item を保持している
- **WHEN** account A を active にして logout する
- **THEN** クライアントは account A の accessToken で logout API を呼び出し、成功後に account A の session item を削除する
- **AND** backend は accessToken claims が示す session と対応する refresh Cookie path の clear command を返す
- **AND** account B の session item は維持される

#### Scenario: clear-cookie command は context index と memory state を同期する (AUTH-FE-S060)

- **GIVEN** logout response が refresh Cookie clear command と cleared authContextId list を返している
- **WHEN** クライアントが logout result を適用する
- **THEN** クライアントは該当 authContextId の memory session item と localStorage context index entry を削除する
- **AND** clear command に含まれない session item は保持する

### Requirement: session expiry と logout は未認証導線を明確に分離する

session expiry と logout の導線は、expired / revoked session と missing session を区別しながら未認証状態への復帰導線を SHALL 提供しなければならない。JWT accessToken の期限切れを検知した場合、クライアントは active authContextId の context refresh を 1 回だけ試行し、refresh 成功時はセッションを継続し、失敗時のみ対象 session を失効扱いにしなければならない（MUST）。

**Customer Context**

利用者が「未ログインなのか」「認証が切れたのか」を迷わないことが重要である。複数 session を保持する場合、1 つの session 失効が他 session の作業を巻き込まないことも必要である。

#### Scenario: セッション失効時は再認証画面へリダイレクトする (AUTH-FE-S006)

- **GIVEN** 利用者が `/*` 内で操作している
- **WHEN** active session が expired または revoked として報告され、context refresh でも継続できない
- **THEN** 利用者は `/session-expired` へ遷移し、その後の画面 presentation はその route contract に委ねられる

#### Scenario: logout は利用者を非認証 route へ戻す (AUTH-FE-S007)

- **GIVEN** 利用者が active な authenticated session を持っている
- **WHEN** 利用者が `/logout` を開く
- **THEN** active session state は消去され、残りの session item がなければ利用者は signed in として振る舞わない public route または login route に到達する

#### Scenario: session を持たない `/*` 到達は通常の未認証導線に留まる (AUTH-FE-S008)

- **GIVEN** 利用者が有効な session item を持たずに `/*` を開く
- **WHEN** app が current session の不在を検知する
- **THEN** 利用者は通常の login 導線へ進み、`/session-expired` へは遷移しない
