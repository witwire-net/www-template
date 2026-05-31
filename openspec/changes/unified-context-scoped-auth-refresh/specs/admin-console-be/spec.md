## MODIFIED Requirements

### Requirement: Admin 管理 API は Admin 専用 backend surface でのみ公開される

Backend は Product API binary と Admin API binary を別 entrypoint として提供しなければならない（SHALL）。Product API binary は Admin operations、Admin handlers、Admin generated bindings、Admin authorization middleware を register してはならない（MUST NOT）。Admin API binary は Admin operations の `/api/v1/*` routes を Admin public HTTP surface として register し、Product operations を register してはならない（MUST NOT）。Admin operations は Admin surface の generated server bindings に従い、Product OpenAPI / Product SDK / Product Go bindings に含まれてはならない（MUST NOT）。Admin API は operator accessToken、operator session record、Admin RBAC を必須とし、Product user bearer token、Product Cookie、Admin refresh Cookie、`X-Auth-Context-Id` header、CSRF token を認可に使用してはならない（MUST NOT）。

**Customer Context**

Admin API は account lifecycle、operator management、audit など強権限 operation を提供する。Product backend host や Product SDK から Admin operation が見えると、誤用や攻撃面の拡大につながるため、Admin API は `/api/v1/*` path policy を維持したまま別バイナリ・別ホストの surface として閉じる必要がある。

#### Scenario: Product binary に Admin route が登録されない (ADMIN-CONSOLE-BE-S056)

- **GIVEN** Product API binary が起動している
- **WHEN** Admin account operation path である `POST /api/v1/accounts` に request を送信する
- **THEN** Product API binary は Admin handler を実行せず 404 または host-level reject を返す

#### Scenario: Admin binary に Product route が登録されない (ADMIN-CONSOLE-BE-S057)

- **GIVEN** Admin API binary が起動している
- **WHEN** Product operation path である `/api/v1/sessions` に request を送信する
- **THEN** Admin API binary は Product handler を実行せず 404 または host-level reject を返す

#### Scenario: Product bearer token は Admin API で認可されない (ADMIN-CONSOLE-BE-S058)

- **GIVEN** request が有効な Product bearer token を持つが operator accessToken と operator session record を持たない
- **WHEN** Admin account search API を呼び出す
- **THEN** Admin API は request を未認証または権限不足として拒否する

#### Scenario: Admin refresh Cookie は Admin 管理 API で認可されない (ADMIN-CONSOLE-BE-S094)

- **GIVEN** request が Admin refresh Cookie を持つが operator accessToken を持たない
- **WHEN** Admin account mutation API を呼び出す
- **THEN** Admin API は operator credential 不在として拒否し、refresh Cookie を認可材料にしない

### Requirement: RBAC 権限チェックは Controller で強制される

Admin API の Controller / handler は全 protected operation で operator accessToken、operator session record、operator active state を application DTO に変換し、`internal/application/admin` の authorization use case を呼び出さなければならない（SHALL）。Permission map と `accounts:create` 判定は application layer が所有し、HTTP handler、repository、generated bindings、runtime composition、frontend package に置かれてはならない（MUST NOT）。`accounts:create` 権限は admin ロールと operator ロールに許可し、viewer ロールには許可してはならない（MUST NOT）。Product bearer token、Product account role、Cookie credential、CSRF token、`X-Auth-Context-Id` は Admin RBAC の判定材料として使用してはならない（MUST NOT）。

**Customer Context**

Admin account 作成は顧客に直接影響する強権限操作である。UI の表示制御だけでは不十分であり、Admin backend の application authorization use case で RBAC を強制し、Controller / handler はその判定を必ず呼び出す必要がある。

#### Scenario: admin と operator は account 作成権限を持つ (ADMIN-CONSOLE-BE-S068)

- **GIVEN** Operator role が `admin` または `operator` である
- **WHEN** application authorization use case が `hasPermission(role, 'accounts:create')` を評価する
- **THEN** true が返される

#### Scenario: viewer は account 作成権限を持たない (ADMIN-CONSOLE-BE-S069)

- **GIVEN** Operator role が `viewer` である
- **WHEN** application authorization use case が `hasPermission('viewer', 'accounts:create')` を評価する
- **THEN** false が返される

#### Scenario: RBAC は検証済み operator accessToken context だけを入力にする (ADMIN-CONSOLE-BE-S095)

- **GIVEN** Admin mutation request が operator accessToken claims と server-side operator/session record の検証を通過している
- **WHEN** Controller が authorization use case を呼び出す
- **THEN** use case input は operator ID、role、active state、session ID、permission だけを含む
- **AND** Product bearer token、Cookie value、CSRF token、`X-Auth-Context-Id` は permission 判定に含まれない
