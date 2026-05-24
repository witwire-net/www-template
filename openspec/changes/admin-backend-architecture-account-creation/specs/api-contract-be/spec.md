## ADDED Requirements

### Requirement: API surface は service ごとに分離生成される

TypeSpec は Product surface と Admin surface を別 service として表現しなければならない（SHALL）。Product surface の OpenAPI / TypeScript SDK / Go bindings は Product routes のみを含み、Admin operations を含んではならない（MUST NOT）。Admin surface の OpenAPI / TypeScript SDK / Go bindings は Admin operations のみを含み、Product operations を含んではならない（MUST NOT）。Product surface と Admin surface はどちらも `/api/v1/*` path policy に従い、両者の到達境界は origin、service artifact、binary、generated package で分離されなければならない（MUST）。各 surface の生成 artifact は surface 名を含む path または package boundary で分離され、片方の artifact だけを参照しても他方の operation が到達可能になってはならない（MUST NOT）。

**Customer Context**

Admin API は強権限 operation を含むため、Product API の公開契約や生成 SDK に混入すると、意図しない host から運営機能へ到達できるリスクが生じる。複数の別バイナリ・別ホストを安全に増やすには、TypeSpec 上で service surface と生成物を明示的に分ける必要がある。

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

### Requirement: 共有 domain model は route surface と独立して再利用できる

TypeSpec は Account などの共有 schema model を surface から独立した shared module として参照できなければならない（SHALL）。共有 model module は route decorator または service-specific route namespace を定義してはならない（MUST NOT）。Product surface と Admin surface は必要な shared model を import できるが、他 surface の route namespace を import してはならない（MUST NOT）。同一概念の ID / error / pagination / audit correlation model は surface 間で互換性を保たなければならない（SHALL）。

**Customer Context**

Product と Admin は Account など同じ domain concept を扱うが、公開 route と運営 route は露出範囲が異なる。model 定義を重複させると不整合が発生し、route 定義を共有しすぎると Admin operation が Product surface に混入する。

#### Scenario: Shared model import は route を増やさない (API-CONTRACT-BE-S004)

- **GIVEN** Admin surface が共有 Account model を import する
- **WHEN** Admin OpenAPI を生成する
- **THEN** 共有 model の import によって Product route は Admin OpenAPI に追加されない

#### Scenario: Product surface は Admin route namespace を import できない (API-CONTRACT-BE-S005)

- **GIVEN** Product surface の TypeSpec source が Admin route namespace を import している
- **WHEN** contract lint を実行する
- **THEN** surface boundary violation として失敗する

### Requirement: Codegen drift check は surface isolation を検証する

codegen drift check は Product / Admin の OpenAPI、TypeScript SDK、Go bindings をそれぞれ検証しなければならない（SHALL）。Product artifact に Admin operationId、Admin tag、Admin schema-only response、または Admin generated export が含まれる場合、check は失敗しなければならない（MUST）。Admin artifact に Product operationId、Product tag、Product schema-only response、または Product generated export が含まれる場合、check は失敗しなければならない（MUST）。Backend build は Product binary が Product bindings のみを参照し、Admin binary が Admin bindings のみを参照することを検証しなければならない（SHALL）。

**Customer Context**

生成 artifact は実装と CI の境界であり、誤った artifact が commit されると Product host から Admin operation が見える可能性がある。surface separation は設計上の約束だけでなく、自動検証で守られる必要がある。

#### Scenario: Product artifact に Admin operation が混入すると check が失敗する (API-CONTRACT-BE-S006)

- **GIVEN** Product OpenAPI artifact に Admin account creation の operationId または Admin tag が含まれている
- **WHEN** codegen drift check を実行する
- **THEN** check は失敗し、Product artifact から Admin operation を除外する必要があることを報告する

#### Scenario: Binary ごとに参照できる bindings が限定される (API-CONTRACT-BE-S007)

- **GIVEN** Product binary の source が Admin generated bindings を import している
- **WHEN** backend lint または build boundary check を実行する
- **THEN** Product binary の Admin bindings import は拒否される
