## MODIFIED Requirements

### Requirement: クライアントは JWT アクセストークンの有効期限を監視し自動更新する

クライアントは JWT accessToken の有効期限を監視し、期限切れ前に自動更新しなければならない（MUST）。クライアントは refreshToken を JavaScript から読める memory、localStorage、sessionStorage、IndexedDB、URL、telemetry、log に保存してはならない（MUST NOT）。refreshToken は `HttpOnly; Secure; SameSite=Lax` Cookie として browser に保持され、refresh request は same-origin `/api/v1/auth/refresh` に credentials を含めて送信されなければならない（SHALL）。refresh 成功時、クライアントは response body の新しい accessToken をメモリ上の対象 session に反映し、refreshToken Cookie の rotation は server response に委ねなければならない（MUST）。

**Customer Context**

利用者は操作中に認証が突然切れてデータを失う体験を避けたい。一方で refreshToken を JavaScript から読める場所に保持すると XSS 時の token 窃取リスクが高い。accessToken だけをブラウザーから読める state とし、refreshToken は HttpOnly Cookie に閉じることで安全性と継続利用を両立する。

#### Scenario: 期限切れ間近の accessToken は Cookie refresh で更新される (AUTH-FE-S045)

- **GIVEN** クライアントが有効期限まで 1 分未満の accessToken を保持している
- **WHEN** 保護された API を呼び出そうとする
- **THEN** クライアントは先に credentials を含めて `POST /api/v1/auth/refresh` を実行する
- **AND** response body の新しい accessToken で API を呼び出す

#### Scenario: refreshToken はブラウザーから読める storage に保存されない (AUTH-FE-S046)

- **GIVEN** 利用者が login または refresh に成功する
- **WHEN** client auth state、localStorage、sessionStorage、IndexedDB、URL state を確認する
- **THEN** refreshToken 平文は存在せず、accessToken と session metadata だけがブラウザーから読める state に存在する

#### Scenario: refresh 失敗時は対象 session だけを失効扱いにする (AUTH-FE-S047)

- **GIVEN** client が複数 session を保持している
- **WHEN** 1 つの session の refresh request が失敗する
- **THEN** client は対象 session だけを失効扱いにし、他 session の accessToken state は維持する

### Requirement: クライアントは複数アカウントのセッションを同時に保持・切り替えできる

クライアントはメモリ上で複数 account session の accessToken と session metadata を同時に保持し、アクティブ session を切り替えできなければならない（SHALL）。ログイン成功時、クライアントは response body の accessToken と session metadata を session list に追加しなければならない（MUST）。refreshToken は HttpOnly Cookie として server が管理するため、クライアントの session list に refreshToken 平文を含めてはならない（MUST NOT）。保護された API 呼び出しはアクティブ session の accessToken を `Authorization: Bearer` header に使用しなければならない（MUST）。refresh が必要な場合、クライアントは対象 session ID を指定して same-origin refresh request を送り、server に Cookie binding の検証と rotation を委ねなければならない（SHALL）。

**Customer Context**

複数アカウントを運用する利用者にとって、都度ログインし直さずに account を切り替えられる体験は必須である。同時に refreshToken をブラウザーから読める state に入れないことで、XSS 時の被害範囲を抑える。

#### Scenario: ログイン毎に accessToken session が追加される (AUTH-FE-S048)

- **GIVEN** 利用者が account A で既に login している
- **WHEN** 利用者が account B で login する
- **THEN** account B の accessToken と session metadata が session list に追加される
- **AND** refreshToken 平文は session list に含まれない

#### Scenario: アカウント切り替えで bearer accessToken が変更される (AUTH-FE-S049)

- **GIVEN** クライアントが account A と account B の session metadata と accessToken を保持している
- **WHEN** 利用者が UI で account B をアクティブに選択する
- **THEN** 後続の API 呼び出しは account B の accessToken を使用する

#### Scenario: logout は対象 session の Cookie revoke を server に依頼する (AUTH-FE-S050)

- **GIVEN** クライアントが account A と account B の session を保持している
- **WHEN** account A をアクティブにして logout する
- **THEN** client は account A の accessToken state を削除し、server は account A の refreshToken Cookie binding を revoke する
- **AND** account B の session state は維持される
