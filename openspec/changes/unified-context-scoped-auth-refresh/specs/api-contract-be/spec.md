## MODIFIED Requirements

### Requirement: API surface は service ごとに分離生成される

TypeSpec は Product surface と Admin surface を別 service として表現しなければならない（SHALL）。Product surface の OpenAPI / TypeScript SDK / Go bindings は Product routes のみを含み、Admin operations を含んではならない（MUST NOT）。Admin surface の OpenAPI / TypeScript SDK / Go bindings は Admin operations のみを含み、Product operations を含んではならない（MUST NOT）。Product surface と Admin surface はどちらも `/api/v1/*` path policy に従い、両者の到達境界は origin、service artifact、binary、generated package で分離されなければならない（MUST）。Product/Admin は同じ relative path `POST /api/v1/auth/contexts/{authContextId}/refresh` をそれぞれの service artifact に持てるが、operation/tag/export は surface ごとに分離されなければならない（MUST）。`/api/admin/*` は Product/Admin contract、generated artifacts、frontend/admin source のいずれにも存在してはならない（MUST NOT）。

Product/Admin の Cookie mode session response は body に accessToken、authContextId、identity/session metadata、context index update hints、clear-cookie command list を表現できなければならず（MUST）、refreshToken 平文を含んではならない（MUST NOT）。Bearer mode session response は accessToken と refreshToken を body に含め、Cookie command を含めてはならない（MUST NOT）。Logout / revoke / suspend response は対象 authContextId と exact Cookie Path を持つ clear-cookie command list を surface ごとに表現しなければならない（MUST）。

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
