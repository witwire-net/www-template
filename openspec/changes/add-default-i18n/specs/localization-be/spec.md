## ADDED Requirements

### Requirement: Product account locale API は認証済み本人の言語設定を SHALL 扱う

システムは、現在のアカウント言語を取得・更新する認証済み Product API 操作を SHALL 提供する。対応するアカウント言語値は `ja` と `en` でなければならない（MUST）。アカウント言語取得操作は、現在の認証済みアカウント言語を SHALL 返し、`Cache-Control: no-store` を使用しなければならない（MUST）。アカウント言語更新操作は、対応ロケール値だけを受け入れ、未知ロケール値は永続値を変更せずに拒否しなければならない（MUST）。アカウント言語操作は `Authorization: Bearer <session token>` を要求しなければならず、未認証、期限切れ、停止中セッションを拒否しなければならない（MUST）。アカウント言語操作は bearer session のアカウントだけを対象に SHALL 動作し、別アカウントを指定する account ID をパスまたは body で受け入れてはならない（MUST NOT）。`POST /api/v1/auth/refresh` は refresh token の rotation 成功時に DB から言語設定を含む client settings を SHALL 読み込み、返却 payload に含める。Product API 契約は TypeSpec を正とし、生成された OpenAPI、frontend SDK、Go bindings は同じ locale request/response と refresh client settings 形状を表さなければならない（SHALL）。

**Customer Context**

利用者は複数端末から同じアカウントにアクセスするため、表示言語とメール言語は端末ローカルではなくアカウント設定として保存される必要がある。保存済み言語を API で安全に取得・更新できれば、UI とメールの言語が一致し、端末ごとの再設定が不要になる。

**要求**

- システムは、現在のアカウント言語を取得・更新する認証済み Product API 操作を SHALL 提供する。
- 対応するアカウント言語値は `ja` と `en` でなければならない（MUST）。
- アカウント言語取得操作は、現在の認証済みアカウント言語を SHALL 返し、`Cache-Control: no-store` を使用しなければならない（MUST）。
- アカウント言語更新操作は、対応ロケール値だけを受け入れ、未知ロケール値は永続値を変更せずに拒否しなければならない（MUST）。
- アカウント言語操作は `Authorization: Bearer <session token>` を要求しなければならず、未認証、期限切れ、停止中セッションを拒否しなければならない（MUST）。
- アカウント言語操作は bearer session のアカウントだけを対象に SHALL 動作し、別アカウントを指定する account ID をパスまたは body で受け入れてはならない（MUST NOT）。
- `POST /api/v1/auth/refresh` は refresh token の rotation 成功時に DB から言語設定を含む client settings を SHALL 読み込み、返却 payload に含める。
- Product API 契約は TypeSpec を正とし、生成された OpenAPI、frontend SDK、Go bindings は同じ locale request/response と refresh client settings 形状を表さなければならない（SHALL）。

#### Scenario: 認証済みアカウントは自分の言語設定を取得できる (LOCALIZATION-BE-S001)

- **前提** 認証済みアカウントが保存済み言語 `ja` を持つ
- **操作** client がアカウント言語設定取得 API を呼び出す
- **結果** システムは `locale: "ja"` を含む no-store response を返す

#### Scenario: 認証済みアカウントは自分の言語設定を更新できる (LOCALIZATION-BE-S002)

- **前提** 認証済みアカウントが保存済み言語 `ja` を持つ
- **操作** client が `locale: "en"` でアカウント言語設定更新 API を呼び出す
- **結果** システムは保存値を `en` に更新し、更新後の locale を no-store response で返す

#### Scenario: 未対応アカウント言語は拒否される (LOCALIZATION-BE-S003)

- **前提** 認証済みアカウントが存在する
- **操作** client が未対応 locale 値でアカウント言語設定更新 API を呼び出す
- **結果** システムは request を拒否し、保存済み locale は変化しない

#### Scenario: 未認証 request はアカウント言語設定を利用できない (LOCALIZATION-BE-S004)

- **前提** bearer session を持たない request がある
- **操作** client がアカウント言語設定 API を呼び出す
- **結果** システムは unauthenticated として拒否し、locale を返さず永続値も変更しない

#### Scenario: refresh 成功時に client settings locale を DB から返す (LOCALIZATION-BE-S013)

- **前提** account が保存済み locale `en` と有効な refresh token を持つ
- **操作** client が `POST /api/v1/auth/refresh` を呼び出す
- **結果** システムは token pair を rotation し、DB から読み込んだ client settings に `locale: "en"` を含めて返す

### Requirement: Product account locale は永続化されメール言語に SHALL 利用される

システムは、各 Product account の locale をアカウントレコードとともに SHALL 永続化する。明示的な locale を持たない Product account の locale 永続値は `ja` を既定値にしなければならない（MUST）。Product account locale の永続値は、対応ロケールだけに制約しなければならない（MUST）。account recovery、device-link、recovery completion、device-link completion の各メールは、アカウントの保存済み locale から件名と本文を SHALL 選択する。アカウントが存在しないため locale を読めない場合でも、列挙耐性が必要な認証フローは非開示の応答挙動を維持しなければならない（MUST）。ローカライズ済み文面を選択する過程で、メール配送ログ、trace、error に recovery token や bearer token を含めてはならない（MUST NOT）。

**Customer Context**

復旧メールや新端末追加メールは、利用者がログインできない状況でも理解できなければならない。UI の言語だけを切り替えても、メールが別言語で届くと復旧やセキュリティ通知の信頼性が下がる。

**要求**

- システムは、各 Product account の locale をアカウントレコードとともに SHALL 永続化する。
- 明示的な locale を持たない Product account の locale 永続値は `ja` を既定値にしなければならない（MUST）。
- Product account locale の永続値は、対応ロケールだけに制約しなければならない（MUST）。
- account recovery、device-link、recovery completion、device-link completion の各メールは、アカウントの保存済み locale から件名と本文を SHALL 選択する。
- アカウントが存在しないため locale を読めない場合でも、列挙耐性が必要な認証フローは非開示の応答挙動を維持しなければならない（MUST）。
- ローカライズ済み文面を選択する過程で、メール配送ログ、trace、error に recovery token や bearer token を含めてはならない（MUST NOT）。

#### Scenario: Product account は既定 locale を持つ (LOCALIZATION-BE-S005)

- **前提** Product account が作成または読み込まれる
- **操作** 明示的な locale が存在しない
- **結果** システムはその account を `ja` locale として扱う

#### Scenario: 復旧メールは保存済みアカウント言語で送信される (LOCALIZATION-BE-S006)

- **前提** アカウントが保存済み言語 `en` を持つ
- **操作** account recovery token が発行される
- **結果** recovery email の件名と本文は英語テンプレートから生成される

#### Scenario: デバイスリンク完了メールは保存済みアカウント言語で送信される (LOCALIZATION-BE-S007)

- **前提** アカウントが保存済み言語 `ja` を持つ
- **操作** device-link branch の passkey registration が完了する
- **結果** device-link completion email の件名と本文は日本語テンプレートから生成される

#### Scenario: 未対応 locale は DB 制約で保存できない (LOCALIZATION-BE-S008)

- **前提** Product account locale persistence が利用可能である
- **操作** 未対応 locale 値を保存しようとする
- **結果** persistence layer は値を拒否し、account record は対応 locale のみを保持する

### Requirement: Admin operator locale は認証済みオペレーター本人の設定として SHALL 永続化される

システムは、各 Admin operator の locale をオペレーターレコードとともに SHALL 永続化する。対応する operator locale 値は `ja` と `en` でなければならない（MUST）。明示的な locale を持たない operator の locale 永続値は `ja` を既定値にしなければならない（MUST）。認証済み Admin server context は、現在の role と active 状態とともに、Admin DB から operator の現在 locale を SHALL 読み込む。認証済みオペレーターは、Admin Console のプロフィールまたは設定操作を通じて、自分自身の locale だけを SHALL 更新できる。Admin operator locale 更新は、未対応 locale 値を永続値を変更せずに拒否しなければならない（MUST）。role、active state、setup token、passkey を編集する operator 管理操作は、operator locale を暗黙的に変更してはならない（MUST NOT）。

**Customer Context**

Admin Console のオペレーターは端末やブラウザを変えても同じ言語で作業できる必要がある。本人の言語設定を Admin DB に保存することで、サポート対応や監査確認の操作文言が安定し、運用ミスを減らせる。

**要求**

- システムは、各 Admin operator の locale をオペレーターレコードとともに SHALL 永続化する。
- 対応する operator locale 値は `ja` と `en` でなければならない（MUST）。
- 明示的な locale を持たない operator の locale 永続値は `ja` を既定値にしなければならない（MUST）。
- 認証済み Admin server context は、現在の role と active 状態とともに、Admin DB から operator の現在 locale を SHALL 読み込む。
- 認証済みオペレーターは、Admin Console のプロフィールまたは設定操作を通じて、自分自身の locale だけを SHALL 更新できる。
- Admin operator locale 更新は、未対応 locale 値を永続値を変更せずに拒否しなければならない（MUST）。
- role、active state、setup token、passkey を編集する operator 管理操作は、operator locale を暗黙的に変更してはならない（MUST NOT）。

#### Scenario: Admin 認証 context は operator locale を読み込む (LOCALIZATION-BE-S009)

- **前提** オペレーターが保存済み言語 `en` を持つ
- **操作** Admin server hook が認証済み operator context を作成する
- **結果** locals の operator context は role、session binding、jti とともに `locale: "en"` を含む

#### Scenario: オペレーターは自分の locale を更新できる (LOCALIZATION-BE-S010)

- **前提** 認証済みオペレーターが保存済み言語 `ja` を持つ
- **操作** オペレーターが自分の locale を `en` に更新する
- **結果** Admin DB の該当 operator record は `en` を保持し、他オペレーターの locale は変化しない

#### Scenario: Admin の未対応 locale 更新は拒否される (LOCALIZATION-BE-S011)

- **前提** 認証済みオペレーターが存在する
- **操作** オペレーターが未対応 locale 値で更新を送信する
- **結果** システムは request を拒否し、保存済み operator locale は変化しない

#### Scenario: Operator 管理操作は locale を暗黙変更しない (LOCALIZATION-BE-S012)

- **前提** 管理者が他オペレーターの role または active state を更新する
- **操作** operator management mutation が成功する
- **結果** 対象オペレーターの locale は mutation 前の値を保持する
