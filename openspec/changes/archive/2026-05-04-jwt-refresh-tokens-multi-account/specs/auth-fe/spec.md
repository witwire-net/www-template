## ADDED Requirements

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
