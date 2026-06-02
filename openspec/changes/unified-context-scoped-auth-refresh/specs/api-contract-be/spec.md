## ADDED Requirements

### Requirement: TypeSpec source は概念単位の単一定義を公開する

TypeSpec source は認証 primitive、WebAuthn、session、refresh、logout、recovery、account read/create model、operator profile、operator setup、authorization などの business concept ごとに model を定義しなければならない（SHALL）。`packages/typespec/main.tsp` の common model imports は concept / capability modules を参照しなければならず（MUST）、surface 固有 catch-all model を common model として参照してはならない（MUST NOT）。同じ contract concept は単一の source definition から Product service と Admin service の artifact へ参照されなければならない（MUST）。Product service と Admin service の到達境界は `routes/v1/product/**`、`routes/v1/admin/**`、service declaration、OpenAPI artifact、SDK package、Go binding package によって表現されなければならない（MUST）。Same auth envelope fields — `accessToken`、`refreshToken`、`authContextId`、`sessionId`、`expiresAt`、`contextIndexUpdateHints`、`clearCookieCommands`、`credentialMode` — は Product/Admin で同じ source definition から来なければならない（MUST）。Account と operator の差分は service-specific subject payload と metadata model で表現されなければならない（MUST）。Maintainability のため、TypeSpec templates/generics が generated SDK/Go usage を読みにくくする場合は、Product response が shared auth envelope に explicit `account` field を、Admin response が shared auth envelope に explicit `operator` field を追加する composed model を使わなければならない（MUST）。`principal` wrapper は必須ではなく、service/artifact boundary が context を決定する payload では `AuthContextIdentityKind`、`identityKind`、`principal.kind` を必須 field にしてはならない（MUST NOT）。

**Customer Context**

同じ認証概念が Product/Admin の名前や catch-all model file に複数回現れると、顧客が使う login / refresh / logout の安全仕様が surface ごとにずれ、修正漏れが発生する。サービス artifact の分離は維持しながら、契約概念の定義は一箇所に集約する必要がある。

#### Scenario: TypeSpec auth concept は単一 source definition から参照される (API-CONTRACT-BE-S013)

- **GIVEN** TypeSpec source tree を検査する
- **WHEN** `accessToken`、`refreshToken`、`authContextId`、`sessionId`、`expiresAt`、`contextIndexUpdateHints`、`clearCookieCommands`、`credentialMode`、context refresh request/response、AuthFailureResponse の定義箇所を数える
- **THEN** 各 concept は概念 module の単一定義として存在し、Product service と Admin service はその定義を参照する
- **AND** Product/Admin 専用 schema は explicit `account` / `operator` subject payload や service metadata の差分だけを表す

#### Scenario: Route source は v1 配下の service boundary で分離される (API-CONTRACT-BE-S014)

- **GIVEN** TypeSpec route source tree を検査する
- **WHEN** Product と Admin の route modules を列挙する
- **THEN** Product routes は `routes/v1/product/**` に存在し、Admin routes は `routes/v1/admin/**` に存在する
- **AND** Product/Admin service artifacts はそれぞれの route subtree から生成される

#### Scenario: Auth response field names は service artifact 間で一致する (API-CONTRACT-BE-S015)

- **GIVEN** Product と Admin の Cookie mode / Bearer mode auth response schema が生成されている
- **WHEN** access token、refresh token、auth context、expiry、context index hint、clear-cookie command の field name を比較する
- **THEN** 同じ concept の field name は両 service artifact で一致する
- **AND** Admin artifact の token field は `accessToken` として公開される

#### Scenario: Subject context は explicit service field で表現される (API-CONTRACT-BE-S016)

- **GIVEN** Product と Admin の auth response schema が生成されている
- **WHEN** subject payload と context index の schema を検査する
- **THEN** Product service は explicit `account` subject field を返し、Admin service は explicit `operator` subject field を返す
- **AND** hosted service artifact が subject context を決定するため、context index entry は authContextId と session/display metadata で bootstrap できる
- **AND** `principal` wrapper は必須ではなく、必要な場合でも generated consumer の可読性を損なわない場合に限る

#### Scenario: Common model imports は concept modules だけを参照する (API-CONTRACT-BE-S017)

- **GIVEN** `packages/typespec/main.tsp` の common model imports を検査する
- **WHEN** imported model modules を分類する
- **THEN** common imports は auth、accounts、operators、sessions、refresh、logout、recovery、authorization などの concept/capability modules を参照する
- **AND** surface 固有 catch-all model module は common model import に含まれない

#### Scenario: Account read/create DTO は account concept module で定義される (API-CONTRACT-BE-S018)

- **GIVEN** Account read/create DTO の TypeSpec definitions を検査する
- **WHEN** Product account read model と Admin account creation/search route DTO が同じ account concept を扱う
- **THEN** DTO は `models/accounts/**` の account concept owner で一度だけ定義される
- **AND** Admin route DTO は surface 接頭辞の account schema duplicate ではなく account concept definition を参照する

#### Scenario: Auth route DTO は common auth contract concepts を参照する (API-CONTRACT-BE-S019)

- **GIVEN** Product auth routes と Admin auth routes が `routes/v1/product/**` と `routes/v1/admin/**` に分離されている
- **WHEN** login、setup、register、logout、context refresh の request/response DTO を検査する
- **THEN** route DTO は common auth contract concepts の Cookie session response、Bearer session response、context refresh request/response、AuthFailureResponse、clear-cookie command を参照する
- **AND** `AuthContextIdentityKind`、`identityKind`、`principal.kind` を service context discriminator として要求せず、route/service artifact boundary で service context を判断する

## MODIFIED Requirements

### Requirement: API surface は service ごとに分離生成される

TypeSpec は Product surface と Admin surface を別 service として表現しなければならない（SHALL）。Product surface の OpenAPI / TypeScript SDK / Go bindings は Product routes のみを含み、Admin operations を含んではならない（MUST NOT）。Admin surface の OpenAPI / TypeScript SDK / Go bindings は Admin operations のみを含み、Product operations を含んではならない（MUST NOT）。Product surface と Admin surface はどちらも `/api/v1/*` path policy に従い、両者の到達境界は origin、service artifact、binary、generated package で分離されなければならない（MUST）。Product/Admin は同じ relative path `POST /api/v1/auth/contexts/{authContextId}/refresh` をそれぞれの service artifact に持てるが、operation/tag/export は surface ごとに分離されなければならない（MUST）。`/api/admin/*` は Product/Admin contract、generated artifacts、frontend/admin source のいずれにも存在してはならない（MUST NOT）。

Product/Admin の Cookie mode session response は body に accessToken、authContextId、identity/session metadata、context index update hints、clear-cookie command list を表現できなければならず（MUST）、refreshToken 平文を含んではならない（MUST NOT）。Bearer mode session response は accessToken と refreshToken を body に含め、Cookie command を含めてはならない（MUST NOT）。Admin session response は `operatorAccessToken` ではなく common `accessToken` field を公開しなければならない（MUST）。Logout / revoke / suspend response は対象 authContextId と exact Cookie Path を持つ clear-cookie command list を surface ごとに表現しなければならない（MUST）。

**Customer Context**

Admin API は強権限 operation を含むため、Product API の公開契約や生成 SDK に混入すると、意図しない host から運営機能へ到達できるリスクが生じる。認証方式を統一しても、route artifact と SDK は surface ごとに明確に分ける必要がある。

#### Scenario: Product OpenAPI に Admin operation が含まれない (API-CONTRACT-BE-S001)

- **GIVEN** TypeSpec から Product surface の OpenAPI を生成する
- **WHEN** 生成された OpenAPI paths を確認する
- **THEN** Product tag / operationId / schema / generated export に Admin operation は存在しない
- **AND** Product Go bindings と Product TypeScript SDK に Admin operation は生成されない

#### Scenario: Admin OpenAPI に Product operation が含まれない (API-CONTRACT-BE-S002)

- **GIVEN** TypeSpec から Admin surface の OpenAPI を生成する
- **WHEN** 生成された OpenAPI paths を確認する
- **THEN** Admin tag / operationId / schema / generated export だけが含まれ、Product operation は含まれない

#### Scenario: Surface ごとの server URL が分離される (API-CONTRACT-BE-S003)

- **GIVEN** Product と Admin の OpenAPI artifact が生成されている
- **WHEN** 各 artifact の `servers` を確認する
- **THEN** Product artifact は Product backend host を表現し、Admin artifact は Admin backend host を表現する

#### Scenario: Context refresh path は両 surface で分離生成される (API-CONTRACT-BE-S010)

- **GIVEN** Product と Admin の TypeSpec service が context refresh operation を定義している
- **WHEN** `pnpm gen` が OpenAPI、SDK、Go bindings を生成する
- **THEN** Product artifacts は Product context refresh operation だけを含む
- **AND** Admin artifacts は Admin context refresh operation だけを含む
- **AND** どちらの artifact も `/api/admin/*` path を含まない

#### Scenario: Cookie mode response は clear-cookie command list を表現する (API-CONTRACT-BE-S011)

- **GIVEN** Product または Admin の logout / revoke / suspend operation が TypeSpec に定義されている
- **WHEN** `pnpm gen` が OpenAPI、SDK、Go bindings を生成する
- **THEN** Cookie mode response DTO は authContextId と exact refresh Cookie Path を持つ clear-cookie command list を表現できる
- **AND** Product/Admin generated artifacts はそれぞれの surface 専用 DTO だけを公開する

#### Scenario: Bearer refresh success DTO は Cookie command を含まない (API-CONTRACT-BE-S012)

- **GIVEN** Product external client または Admin automation client が Bearer mode context refresh を使う
- **WHEN** TypeSpec の refresh response union を確認する
- **THEN** Bearer success response は accessToken と refreshToken を body に含む
- **AND** Bearer success response は Set-Cookie / clear-cookie command / browser context index hint を含まない

### Requirement: Codegen drift check は surface isolation を検証する

codegen drift check は Product / Admin の OpenAPI、TypeScript SDK、Go bindings をそれぞれ検証しなければならない（SHALL）。Product artifact に Admin operationId、Admin tag、Admin schema-only response、または Admin generated export が含まれる場合、check は失敗しなければならない（MUST）。Admin artifact に Product operationId、Product tag、Product schema-only response、または Product generated export が含まれる場合、check は失敗しなければならない（MUST）。Backend build と lint は Product binary / Product HTTP adapter が Product bindings のみを参照し、Admin binary / Admin HTTP adapter が Admin bindings のみを参照することを検証しなければならない（SHALL）。Frontend lint は Product SDK が `packages/frontend/api` に閉じ、Admin SDK が `packages/admin/api` に閉じることを検証しなければならない（SHALL）。

**Customer Context**

生成 artifact は実装と CI の境界であり、誤った artifact が commit されると Product host から Admin operation が見える可能性がある。認証方式を共有しても、surface separation は自動検証で守られる必要がある。

#### Scenario: Product artifact に Admin operation が混入すると check が失敗する (API-CONTRACT-BE-S006)

- **GIVEN** Product OpenAPI artifact に Admin account creation の operationId または Admin tag が含まれている
- **WHEN** codegen drift check を実行する
- **THEN** check は失敗し、Product artifact から Admin operation を除外する必要があることを報告する

#### Scenario: Binary ごとに参照できる bindings が限定される (API-CONTRACT-BE-S007)

- **GIVEN** Product binary の source が Admin generated bindings を import している
- **WHEN** backend lint または build boundary check を実行する
- **THEN** Product binary の Admin bindings import は拒否される

#### Scenario: Admin bindings は Admin HTTP adapter だけが import できる (API-CONTRACT-BE-S008)

- **WHEN** `internal/app`、`internal/application/**`、`internal/domain/**`、Product HTTP adapter、または Product binary が Admin generated bindings を import している
- **THEN** backend lint は generated binding boundary violation として失敗する

#### Scenario: Product SDK と Admin SDK は frontend package 境界を越えない (API-CONTRACT-BE-S009)

- **WHEN** `packages/frontend/**` が Admin SDK を import する、または `packages/admin/**` が Product SDK を import する
- **THEN** frontend lint は SDK package boundary violation として失敗する
