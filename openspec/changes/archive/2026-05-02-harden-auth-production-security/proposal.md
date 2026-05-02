## 背景

現状の認証基盤は Passkey/WebAuthn を中心にしている一方で、新しい端末でログインを有効にする導線と公開認証 endpoint の防御がプロダクション運用の脅威モデルに対して不足している。特に 6 桁 OTP だけで公開 endpoint から passkey credential を追加できる構造、OTP のグローバル衝突、公開 WebAuthn challenge の蓄積、非 atomic な one-time credential 消費は、アカウント乗っ取りや DoS に直結し得る。

テンプレートとして提供する認証基盤は、導入先の個別事情に依存せず安全な初期値を持つ必要がある。ユーザー体験は「新しい端末でログインを有効にする」という分かりやすい導線を維持しつつ、メールアドレス + OTP、厳密な rate limit、atomic consume、WebAuthn user verification、production config validation、browser / server hardening を同じ認証境界として扱う。

## 変更内容

- 新しい端末でログインを有効にする public flow は、6 桁 OTP だけではなく登録メールアドレスと OTP の組み合わせを要求する。
- OTP は既存端末の画面には表示せず、既存端末で WebAuthn 再認証を完了した後に登録メールアドレスへ送信する。
- OTP 送信と passkey credential 削除は bearer session だけでは成立させず、fresh な WebAuthn 再認証 session を要求する。
- OTP / recovery token / recovery session / WebAuthn challenge の one-time state は、衝突しにくく、平文 secret を保存せず、atomic に消費される。
- 公開認証 endpoint は IP、email、email+IP、OTP、account、global の各 bucket で rate limit / temporary lock を適用し、失敗理由や account existence を露出しない。
- WebAuthn login / registration は user verification を required として扱う。
- production runtime は allowed origin、RP ID、recovery URL、trusted proxy、HTTPS 制約を fail-close で検証する。
- recovery token を URL query から読んだ後、client は即時に URL から token を除去する。
- 認証画面と API は body size、timeout、security headers、通知、監査イベントを含む hardening を備える。
- **BREAKING**: `/api/v1/auth/passkey/add/start` と `/api/v1/auth/passkey/add/finish` の request body は `otp` だけでなく `email` を必須にする。

## Spec Unit

### 新規 Spec Unit

なし。

### 変更 Spec Unit

- `auth-be`: 認証 API、WebAuthn、OTP/recovery state、rate limit、production config、メール配送、security hardening の backend requirements を更新する。Security、abuse resistance、operational fail-close、performance/DoS が横断関心。
- `auth-fe`: 新しい端末でログインを有効にする UX、メールアドレス + OTP 入力、recovery token URL 処理、session/security presentation、no-store/security headers の frontend requirements を更新する。Security、privacy、error-message consistency、a11y が横断関心。

## 命名

既存 Spec Unit を更新するため、Scenario ID prefix は既存と同じ `AUTH-BE-*` と `AUTH-FE-*` を使用する。Backend scenarios は `AUTH-BE-S###`、Frontend scenarios は `AUTH-FE-S###` として FE/BE の prefix を分離する。

## 影響

- API contract: TypeSpec の passkey add-by-OTP request model に email を追加し、OpenAPI / frontend SDK / Go bindings を再生成する。
- Backend: auth usecases、Valkey auth state repository、WebAuthn provider config、router middleware、runtime config validation、SMTP sender、tests に影響する。
- Frontend: passkey management UI、新端末ログイン有効化画面、recovery consume route、domain auth hooks、generated SDK 呼び出しに影響する。
- Persistence: Valkey key layout と atomic consume behavior に影響する。Postgres schema migration は原則不要。
- Operations: trusted proxy、allowed origins、RP ID、recovery URL、body size、rate limit、security headers、mail TLS の production 設定が必要になる。
- Security: account takeover prevention、DoS resistance、XSS blast radius reduction、token leakage reduction、auditability が主な影響範囲になる。
