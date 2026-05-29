# localization-be Specification

## Purpose

TBD - created by archiving change add-default-i18n. Update Purpose after archive.

## Requirements

### Requirement: Product AccountSetting API は認証済み Account 本人の言語設定を SHALL 扱う

システムは、Product を利用する主体を Account として扱い、その Account に属する AccountSetting を SHALL 提供する。AccountSetting は Account の表示・通知に使う設定であり、Auth、端末ローカル状態、または UI 専用の一時設定として扱ってはならない（MUST NOT）。システムは、現在の Account の AccountSetting.locale を取得・更新する認証済み Product API 操作を SHALL 提供する。対応する AccountSetting.locale 値は `ja` と `en` でなければならない（MUST）。AccountSetting 取得操作は、現在の認証済み Account の locale を SHALL 返し、`Cache-Control: no-store` を使用しなければならない（MUST）。AccountSetting 更新操作は、対応ロケール値だけを受け入れ、未知ロケール値は永続値を変更せずに拒否しなければならない（MUST）。AccountSetting 操作は `Authorization: Bearer <session token>` を要求しなければならず、未認証、期限切れ、停止中セッションを拒否しなければならない（MUST）。AccountSetting 操作は bearer session が指す Account だけを対象に SHALL 動作し、別 Account を指定する account ID を path または body で受け入れてはならない（MUST NOT）。`POST /api/v1/auth/refresh` は refresh token の rotation 成功後、Auth が確定した AccountID に対して AccountSetting snapshot を SHALL 読み込み、返却 payload に含める。Product API は Product Account と AccountSetting 専用でなければならず、Admin operator locale や `/api/admin/**` を表現してはならない（MUST NOT）。

**Customer Context**

利用者は複数端末から同じ Account にアクセスする。同じ Account なのに端末ごとに言語を選び直す必要があると、設定の信頼性が低く、メール通知との言語も一致しない。Account に属する AccountSetting として保存済み言語を扱えば、UI とメールの言語が一致し、端末ごとの再設定が不要になる。

**要求**

- システムは、Product を利用する主体を Account として扱い、その Account に属する AccountSetting を SHALL 提供する。
- AccountSetting は Account の表示・通知に使う設定であり、Auth、端末ローカル状態、または UI 専用の一時設定として扱ってはならない（MUST NOT）。
- システムは、現在の Account の AccountSetting.locale を取得・更新する認証済み Product API 操作を SHALL 提供する。
- 対応する AccountSetting.locale 値は `ja` と `en` でなければならない（MUST）。
- AccountSetting 取得操作は、現在の認証済み Account の locale を SHALL 返し、`Cache-Control: no-store` を使用しなければならない（MUST）。
- AccountSetting 更新操作は、対応ロケール値だけを受け入れ、未知ロケール値は永続値を変更せずに拒否しなければならない（MUST）。
- AccountSetting 操作は `Authorization: Bearer <session token>` を要求しなければならず、未認証、期限切れ、停止中セッションを拒否しなければならない（MUST）。
- AccountSetting 操作は bearer session が指す Account だけを対象に SHALL 動作し、別 Account を指定する account ID を path または body で受け入れてはならない（MUST NOT）。
- `POST /api/v1/auth/refresh` は refresh token の rotation 成功後、Auth が確定した AccountID に対して AccountSetting snapshot を SHALL 読み込み、返却 payload に含める。
- Product API は Product Account と AccountSetting 専用でなければならず、Admin operator locale や `/api/admin/**` を表現してはならない（MUST NOT）。

#### Scenario: 認証済み Account は自分の AccountSetting.locale を取得できる (LOCALIZATION-BE-S001)

- **前提** 認証済み Account が AccountSetting.locale `ja` を持つ
- **操作** client が AccountSetting 取得 API を呼び出す
- **結果** システムは `locale: "ja"` を含む no-store response を返す

#### Scenario: 認証済み Account は自分の AccountSetting.locale を更新できる (LOCALIZATION-BE-S002)

- **前提** 認証済み Account が AccountSetting.locale `ja` を持つ
- **操作** client が `locale: "en"` で AccountSetting 更新 API を呼び出す
- **結果** システムは AccountSetting.locale を `en` に更新し、更新後の locale を no-store response で返す

#### Scenario: 未対応 AccountSetting.locale は拒否される (LOCALIZATION-BE-S003)

- **前提** 認証済み Account が存在する
- **操作** client が未対応 locale 値で AccountSetting 更新 API を呼び出す
- **結果** システムは request を拒否し、AccountSetting.locale は変化しない

#### Scenario: 未認証 request は AccountSetting を利用できない (LOCALIZATION-BE-S004)

- **前提** bearer session を持たない request がある
- **操作** client が AccountSetting API を呼び出す
- **結果** システムは unauthenticated として拒否し、locale を返さず永続値も変更しない

#### Scenario: refresh 成功時に AccountSetting snapshot を DB から返す (LOCALIZATION-BE-S013)

- **前提** Account が AccountSetting.locale `en` と有効な refresh token を持つ
- **操作** client が `POST /api/v1/auth/refresh` を呼び出す
- **結果** システムは token pair を rotation し、DB から読み込んだ AccountSetting snapshot に `locale: "en"` を含めて返す

### Requirement: Product AccountSetting は永続化されメール言語に SHALL 利用される

システムは、各 Account の AccountSetting を Account とともに SHALL 永続化する。DB は Account root の `accounts`、Account child の `account_settings`、Account.Auth child の `account_passkey_credentials` を正規 table として SHALL 持つ。AccountSetting は `account_settings` table として Account root の `accounts.id` に紐づかなければならない（MUST）。Account 作成時には `ja` の AccountSetting を同じ Account の child として作らなければならない（MUST）。AccountSetting.locale の永続値は、対応ロケールだけに制約しなければならない（MUST）。DB は `passkey_credentials`、`auth_accounts`、`accounts.locale` を持ってはならない（MUST NOT）。account recovery、device-link、recovery completion、device-link completion の各メールは、Auth が生成した配送 intent と AccountSetting.locale を composition して件名と本文を SHALL 選択する。Auth domain/application は AccountSetting や locale 値オブジェクトを所有してはならない（MUST NOT）。Account が存在しないため AccountSetting を読めない場合でも、列挙耐性が必要な認証フローは非開示の応答挙動を維持しなければならない（MUST）。ローカライズ済み文面を選択する過程で、メール配送ログ、trace、error に recovery token や bearer token を含めてはならない（MUST NOT）。

**Customer Context**

復旧メールや新端末追加メールは、利用者がログインできない状況でも理解できなければならない。UI の言語だけを切り替えても、メールが別言語で届くと復旧やセキュリティ通知の信頼性が下がる。AccountSetting.locale を Account の標準言語として使うことで、Account の画面表示と認証メールを同じ言語に揃えられる。

**要求**

- システムは、各 Account の AccountSetting を Account とともに SHALL 永続化する。
- DB は Account root の `accounts`、Account child の `account_settings`、Account.Auth child の `account_passkey_credentials` を正規 table として SHALL 持つ。
- AccountSetting は `account_settings` table として Account root の `accounts.id` に紐づかなければならない（MUST）。
- Account 作成時には `ja` の AccountSetting を同じ Account の child として作らなければならない（MUST）。
- AccountSetting.locale の永続値は、対応ロケールだけに制約しなければならない（MUST）。
- DB は `passkey_credentials`、`auth_accounts`、`accounts.locale` を持ってはならない（MUST NOT）。
- account recovery、device-link、recovery completion、device-link completion の各メールは、Auth が生成した配送 intent と AccountSetting.locale を composition して件名と本文を SHALL 選択する。
- Auth domain/application は AccountSetting や locale 値オブジェクトを所有してはならない（MUST NOT）。
- Account が存在しないため AccountSetting を読めない場合でも、列挙耐性が必要な認証フローは非開示の応答挙動を維持しなければならない（MUST）。
- ローカライズ済み文面を選択する過程で、メール配送ログ、trace、error に recovery token や bearer token を含めてはならない（MUST NOT）。

#### Scenario: Product Account は既定 AccountSetting.locale を持つ (LOCALIZATION-BE-S005)

- **前提** Account が作成される
- **操作** Account root が永続化される
- **結果** システムは同じ Account の `account_settings` record を `locale: "ja"` で作成する

#### Scenario: 復旧メールは AccountSetting.locale で送信される (LOCALIZATION-BE-S006)

- **前提** Account が AccountSetting.locale `en` を持つ
- **操作** account recovery token が発行される
- **結果** recovery email の件名と本文は英語テンプレートから生成される

#### Scenario: デバイスリンク完了メールは AccountSetting.locale で送信される (LOCALIZATION-BE-S007)

- **前提** Account が AccountSetting.locale `ja` を持つ
- **操作** device-link branch の passkey registration が完了する
- **結果** device-link completion email の件名と本文は日本語テンプレートから生成される

#### Scenario: 未対応 AccountSetting.locale は DB 制約で保存できない (LOCALIZATION-BE-S008)

- **前提** AccountSetting persistence が利用可能である
- **操作** 未対応 locale 値を保存しようとする
- **結果** persistence layer は値を拒否し、AccountSetting は対応 locale のみを保持する

### Requirement: Auth は Account にぶら下がる認証概念として SHALL 分離される

システムは、Account を Product 利用主体として扱い、Auth をその Account にぶら下がる本人確認・session・token・credential の概念として SHALL 扱う。Auth domain/application は Account aggregate を所有または代替してはならない（MUST NOT）。Auth domain/application は `AuthAccount` または `AuthSubject` という Account 代替モデルを提供してはならない（MUST NOT）。Auth が必要とする認証用 projection は `AccountAuth` として表現し、AccountID、認証 identifier、email、status、session revoked boundary、passkey credentials だけを扱わなければならない（MUST）。`AccountAuth` は AccountSetting、AccountSetting.locale、AccountSetting mutation、AccountSetting snapshot を持ってはならない（MUST NOT）。Auth repository は AccountSetting persistence を読み書きしてはならない（MUST NOT）。Auth repository は Account.Auth credential を `account_passkey_credentials` から復元し、`passkey_credentials` を参照してはならない（MUST NOT）。refresh response に AccountSetting snapshot が必要な場合、Auth は token pair と AccountID を返し、HTTP composition または account application が AccountSetting を読み込まなければならない（MUST）。

**Customer Context**

利用者の Account は、認証方式や端末が変わっても同じ Product 利用主体として残る。Auth が AccountSetting を所有すると、認証導線の変更が表示・通知言語に波及し、セキュリティ境界と設定境界が混ざる。Account と Auth の親子関係を明確にすることで、本人確認は安全に保ちつつ、AccountSetting を Account の設定として一貫して扱える。

**要求**

- システムは、Account を Product 利用主体として扱い、Auth をその Account にぶら下がる本人確認・session・token・credential の概念として SHALL 扱う。
- Auth domain/application は Account aggregate を所有または代替してはならない（MUST NOT）。
- Auth domain/application は `AuthAccount` または `AuthSubject` という Account 代替モデルを提供してはならない（MUST NOT）。
- Auth が必要とする認証用 projection は `AccountAuth` として表現し、AccountID、認証 identifier、email、status、session revoked boundary、passkey credentials だけを扱わなければならない（MUST）。
- `AccountAuth` は AccountSetting、AccountSetting.locale、AccountSetting mutation、AccountSetting snapshot を持ってはならない（MUST NOT）。
- Auth repository は AccountSetting persistence を読み書きしてはならない（MUST NOT）。
- Auth repository は Account.Auth credential を `account_passkey_credentials` から復元し、`passkey_credentials` を参照してはならない（MUST NOT）。
- refresh response に AccountSetting snapshot が必要な場合、Auth は token pair と AccountID を返し、HTTP composition または account application が AccountSetting を読み込まなければならない（MUST）。

#### Scenario: Auth projection は AccountSetting を所有しない (LOCALIZATION-BE-S014)

- **前提** Auth domain/application と persistence adapter が Account.Auth projection を扱う
- **操作** 実装者が public symbols、constructors、repository reads、imports を検査する
- **結果** `AuthAccount` と `AuthSubject` は存在せず、`AccountAuth` は AccountSetting や locale を持たず、Auth repository は `account_settings` と `passkey_credentials` を読まない

### Requirement: Admin operator locale は認証済みオペレーター本人の設定として SHALL 永続化される

システムは、各 Admin operator の locale をオペレーターレコードとともに SHALL 永続化する。対応する operator locale 値は `ja` と `en` でなければならない（MUST）。明示的な locale を持たない operator の locale 永続値は `ja` を既定値にしなければならない（MUST）。Admin operator locale は Product AccountSetting から独立して扱い、Product AccountSetting を読み書きしてはならない（MUST NOT）。認証済み Admin server context は、現在の role と active 状態とともに、Admin-owned schema から operator の現在 locale を SHALL 読み込む。認証済みオペレーターは、Admin Console のプロフィールまたは設定操作を通じて、自分自身の locale だけを SHALL 更新できる。Admin operator locale 更新は、未対応 locale 値を永続値を変更せずに拒否しなければならない（MUST）。role、active state、setup token、passkey を編集する operator 管理操作は、operator locale を暗黙的に変更してはならない（MUST NOT）。DB から未知の operator locale が読み込まれた場合、システムは既定値へ黙って丸めてはならず（MUST NOT）、DB 制約違反または server error として fail-closed に扱わなければならない（MUST）。

**Customer Context**

Admin Console のオペレーターは端末やブラウザを変えても同じ言語で作業できる必要がある。本人の言語設定を Admin-owned schema に保存することで、サポート対応や監査確認の操作文言が安定し、運用ミスを減らせる。

**要求**

- システムは、各 Admin operator の locale をオペレーターレコードとともに SHALL 永続化する。
- 対応する operator locale 値は `ja` と `en` でなければならない（MUST）。
- 明示的な locale を持たない operator の locale 永続値は `ja` を既定値にしなければならない（MUST）。
- Admin operator locale は Product AccountSetting から独立して扱い、Product AccountSetting を読み書きしてはならない（MUST NOT）。
- 認証済み Admin server context は、現在の role と active 状態とともに、Admin-owned schema から operator の現在 locale を SHALL 読み込む。
- 認証済みオペレーターは、Admin Console のプロフィールまたは設定操作を通じて、自分自身の locale だけを SHALL 更新できる。
- Admin operator locale 更新は、未対応 locale 値を永続値を変更せずに拒否しなければならない（MUST）。
- role、active state、setup token、passkey を編集する operator 管理操作は、operator locale を暗黙的に変更してはならない（MUST NOT）。
- DB から未知の operator locale が読み込まれた場合、システムは既定値へ黙って丸めてはならず（MUST NOT）、DB 制約違反または server error として fail-closed に扱わなければならない（MUST）。

#### Scenario: Admin 認証 context は operator locale を読み込む (LOCALIZATION-BE-S009)

- **前提** オペレーターが保存済み言語 `en` を持つ
- **操作** Admin server hook が認証済み operator context を作成する
- **結果** locals の operator context は role、session binding、jti とともに `locale: "en"` を含む

#### Scenario: オペレーターは自分の locale を更新できる (LOCALIZATION-BE-S010)

- **前提** 認証済みオペレーターが保存済み言語 `ja` を持つ
- **操作** オペレーターが自分の locale を `en` に更新する
- **結果** Admin-owned schema の該当 operator record は `en` を保持し、他オペレーターの locale は変化しない

#### Scenario: Admin の未対応 locale 更新は拒否される (LOCALIZATION-BE-S011)

- **前提** 認証済みオペレーターが存在する
- **操作** オペレーターが未対応 locale 値で更新を送信する
- **結果** システムは request を拒否し、保存済み operator locale は変化しない

#### Scenario: Operator 管理操作は locale を暗黙変更しない (LOCALIZATION-BE-S012)

- **前提** 管理者が他オペレーターの role または active state を更新する
- **操作** operator management mutation が成功する
- **結果** 対象オペレーターの locale は mutation 前の値を保持する
