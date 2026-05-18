## ADDED Requirements

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
