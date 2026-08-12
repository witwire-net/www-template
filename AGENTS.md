## Primary Rules

- **MUST think in English** and **MUST communicate in natural Japanese**.
- Every instruction in this file is absolute and binding within its scope. You MUST follow it exactly; if compliance is impossible or instructions conflict, stop and ask the owner instead of proceeding.
- You MUST doubt your assumptions, verify factual claims against available evidence, and MUST NOT present unsupported statements as facts.
- You MUST NOT override a specific rule or prohibition with an inferred benefit, an abstract principle, convenience, consistency, traceability, maintainability, or an unverified safety claim. Only an applicable explicit rule may authorize an exception; if explicit rules conflict or compliance would create a concrete security risk, stop and ask the owner.
- Rules remain binding even when no automated check enforces them. A passing validation proves only the conditions it actually checks and MUST NOT justify an unchecked violation.
- Write `AGENTS.md` in English. Pull request bodies and pull request template content MUST be written in Japanese, except for code identifiers, commands, logs, file paths, and issue or PR references.

## Natural Japanese Prose

- Any content required to be Japanese MUST read as natural Japanese, not as a literal translation or code-switched prose.
- Do not insert untranslated English common nouns, verbs, adjectives, role names, state names, capability names, or domain terms into Japanese sentences. Use established Japanese words or natural katakana loanwords instead.
- English may remain only when exact spelling is required for correctness: code identifiers, package/API/database identifiers, commands, file paths, log literals, IDs, protocol or standard names, official external product names, and schema-required structural labels.
- A rule that permits "exact technical terms" in English does not widen this exception. "Exact" means a spelling-sensitive proper name or machine-facing token; a generic word such as `Service`, `Customer`, `Account`, `validate`, or `workflow` is not exact merely because software development commonly uses it.
- When an applicable Thesaurus exists, its `Formal Name` is the source of truth for Japanese prose. Its `System Name` is reference metadata and MUST NOT replace the `Formal Name`; mention it only when the exact English name itself is being discussed.
- Wrap exact identifiers in backticks when the format permits and embed them in Japanese grammar instead of using them as untranslated prose vocabulary.
- If no established Japanese term exists, use a natural Japanese description. If the choice can change domain meaning, confirm it with the owner and record it in the Thesaurus before using it in downstream artifacts.
- Apply these rules to user responses, code comments, TSDoc, GoDoc, Japanese repository documentation, OpenSpec prose, pull request bodies, UI copy, and diagram labels.
- Bad: `Service が Customer の Account を validate する。`
- Good: `提供サービスが顧客のアカウントを検証する。`
- Identifier-specific: 提供サービスを表す `Service` 型を検証する。

## Intent Before Implementation

- Treat the user's wording as evidence of intent, not automatically as an implementation-ready specification.
- Before selecting a solution, identify the customer outcome and verify the relevant repository facts and constraints.
- Classify solution-shaped terms as a desired outcome, an outcome constraint, a required means, or a candidate means.
- Confirmation can make a candidate means binding for design, but it never turns a means into an outcome. Only desired outcomes and outcome constraints may become product requirements or OpenSpec Specs; required means remain design constraints.
- Separate observations from inferences and assumptions. Familiarity, common practice, and readily available example code are not evidence that a solution fits this repository.
- Ask the user only when unresolved ambiguity could materially change user-visible behavior, external contracts, architecture, security, data, dependencies, or scope.
- When a workflow provides confirmed intent or an approved specification, preserve that boundary and choose implementation details within it unless contradictory evidence requires escalation.

## Credo

Before beginning any work, you MUST summarize your understanding of the Credo below in Japanese and explicitly declare that you will strictly comply with it. Do not translate or repeat the Credo verbatim; explain how you will apply it to the current task, then begin the work.

1. あらゆる意思決定は顧客ファーストで考えること。誰がどのように利用し、どうすれば喜ばれるかを常に考えること。
2. セキュリティはなによりも優先されること。セキュリティ最優先が、なにより顧客のためになる。
3. 後方互換性は完全悪だ。後方互換性のためのコードや計画がある時点で、そのシステムは一切認められない。常に完璧なプロダクトであるために、不要な機能は即座に削除。
4. 全てのアーキテクチャは保守性のためにある。同じレイヤーの中で同じコードは二度と書くな。コピペはするな。抽象化して考えろ。アーキテクチャで説明できない再実装や再記入は存在してはならない。
5. すべてのルールには意図がある。必ず意図を理解すること。意図を理解しないまま改定したり、逆に遵守しようとしてはならない。
6. 常に完璧なプロダクトであること。妥協、横着、顧客にとって意味のないプロダクトを作ることは一切許されない。仮置きを残す、後回し、コメントにしておいて放置に決してしてはならない。後回しという言葉は発することするら厳禁である。最小実装などという言葉は何があっても使ってはならないし、問題の本質的な解決以外の解決は一切認めない。
7. いかなる理由があろうと、クレドに違反しないこと、クレド違反を放置しないことを最優先とすること。どのクレドによって肯定しうるのか、その作業内容が一切クレドに違反しないことを必ず方針の前に声に出して報告しなければならない。

## Code Comments

- Leave detailed Japanese comments for every single process in the code.
- Clarify the intent, input/output, and side effects of each step so that future readers (including yourself) can understand immediately.

## Documentation Comments (TS Docs / Go Docs)

- TSDoc (TypeScript) and GoDoc (Go) comments must be written in Japanese, providing detailed, multi-line explanations of their roles and parameter meanings.
- Every public API (functions, methods, types, interfaces, and structs) must have a documentation comment in Japanese that describes what it does, the meaning of each argument and return value, error cases, and usage examples.

## Commands

- Install: `corepack enable && pnpm install`
- Generate all contracts: `pnpm gen`
- Typecheck: `pnpm check`
- Dev (all): `pnpm dev:all`
- Dev (server): `pnpm dev:server` (Product Go API on `http://localhost:8080`)
- Dev (admin server): `pnpm dev:admin-server` (Admin Go API on `http://localhost:8081`)
- Dev (client entry): `pnpm dev:client` (alias of `pnpm dev:web`, Vite on `http://www.localhost:5173`)
- Dev (web): `pnpm dev:web` (SvelteKit public site on `http://www.localhost:5173`)
- Dev (app): `pnpm dev:app` (SvelteKit SPA app on `http://localhost:5174`)
- Dev (admin): `pnpm dev:admin` (Admin Console on `http://admin.localhost:5176`)

## Command Policy

- For both backend and frontend work, lint, typecheck, build, and test MUST be invoked through `pnpm` scripts.
- When running verification from Codex Desktop or any host-side shell, invoke the required `pnpm` script through `scripts/devcontainer/run.sh` so the command uses the DevContainer toolchain instead of host Node.js, Go, bash, or pnpm. Example: `scripts/devcontainer/run.sh pnpm check`.
- When already inside the DevContainer, run the same `pnpm` scripts directly or through `scripts/devcontainer/run.sh`; the wrapper detects the container and executes the command in place.
- Use `pnpm lint` for lint, `pnpm check` for typecheck, `pnpm build` or package-specific `pnpm build:*` scripts for build, and `pnpm test:*` scripts for tests.
- Do not call direct verification tools such as `go test`, `go vet`, `go build`, `tsc`, `vitest`, `svelte-check`, `vite build`, `eslint`, or `stylelint`; route them through the existing `pnpm` scripts instead.
- Do not call `pnpm exec` or `pnpm --filter ... exec` directly. If an existing package script uses `exec` internally, run only the parent `pnpm` script.

## API Contract (TypeSpec)

- Source of truth: `packages/typespec/main.tsp`
- Generated Product OpenAPI: `packages/typespec/openapi/openapi.json`
- Generated Admin OpenAPI: `packages/typespec/openapi/admin.openapi.json`
- Generated Product Go server bindings: `packages/backend/internal/generated/openapi/openapi.gen.go`
- Generated Admin Go server bindings: `packages/backend/internal/generated/adminopenapi/openapi.gen.go`
- Regenerate OpenAPI + SDK + Go bindings: `pnpm gen`
- Codegen drift check (CI-style): `pnpm check:codegen`

## Testing

- All unit tests: `pnpm test:run`
- Server tests: `pnpm test:server`
- Client tests: `pnpm test:client`
- E2E: `pnpm test:e2e`

## Pull Requests

- Always use `.github/pull_request_template.md` when creating a pull request, and fill every template item completely with no blank fields.
- Write the pull request body in Japanese. Code identifiers, commands, logs, file paths, and issue or PR references may remain in their original form.
- Do not delete sections or checklist items that do not apply. Instead, write `なし（理由: ...）` or a concrete reason explaining why the item does not apply.
- Check every checklist item after writing the applicable confirmation or non-applicable reason. Do not leave unchecked items in the pull request body.
- Record `Operation Lane` as `DIRECT`, `BEHAVIOR`, or `ARCHITECTURE`; record `UX Mode` independently as `NONE`, `CONTINUITY`, or `SHAPE`; and record `Review Depth` as `STANDARD` or `DEEP`.
- `BEHAVIOR` and `ARCHITECTURE` pull requests MUST identify an OpenSpec Change and at least one Scenario ID. `DIRECT` pull requests may use a reasoned `なし` for both fields.
- For pull requests with UI / UX changes, attach screenshots in all of these sections: `Desktop Before`, `Desktop After`, `Mobile Before`, and `Mobile After`.
- The pull request body is validated by `.github/workflows/validate-pr-template.yml`; when using any pull request creation tool, read the template first and prepare a body that passes this validation.

## Architecture Notes

- Client dependency direction: `web -> frontend/ui` (web is a public LP; it MUST NOT depend on domain or api), `frontend/app -> frontend/domain -> frontend/api` (also `frontend/app -> frontend/ui`)
- Server dependency direction: `backend/cmd -> backend/internal/app -> (backend/internal/adapter/http|backend/internal/adapter/postgres|backend/internal/adapter/valkey|backend/internal/adapter/webauthn|backend/internal/adapter/mailer|backend/internal/application|backend/internal/platform/*) -> backend/internal/domain`
- API contract direction: implementation must follow TypeSpec; do not generate OpenAPI from server routes for SDK input.

## Package Responsibility

- Backend-owned agent scope: `packages/backend`, `packages/typespec`, and `packages/admin`.
- `packages/backend`: Go product API, migrations, generated Go bindings consumption, backend observability, and backend security boundaries.
- `packages/typespec`: API contract source of truth and generated OpenAPI input; edit source contracts only and regenerate via `pnpm gen`.
- `packages/admin`: Admin Console static frontend/domain/API SDK package. Admin frontend calls the same-origin Admin Go backend under `/api/v1/*`; it MUST NOT own `/api/admin/**` BFF routes, Prisma-backed server/runtime logic, or generated Product SDK exposure.
- Frontend-owned agent scope: `packages/web` and `packages/frontend/**`.
- `packages/web`: public landing/public site surface; it may depend on `packages/frontend/ui` only.
- `packages/frontend/i18n`: shared frontend i18n runtime (locale definitions, loader/config, typed translator, formatter, coverage utility). It may be imported by `packages/web`, `packages/frontend/app`, and `packages/admin`, but not by `packages/frontend/ui` or `packages/frontend/domain`.
- `packages/frontend/app`: authenticated `/app` CSR surface; compose domain hooks and UI components without direct API-client or raw network access.
- `packages/frontend/domain`: frontend domain hooks, state, and API orchestration; it is the only handwritten frontend layer that depends on `packages/frontend/api`.
- `packages/frontend/ui`: reusable UI components, styling primitives, assets, and presentation utilities.
- `packages/frontend/api`: generated API SDK/types package; do not hand-edit generated artifacts, and route contract changes through `packages/typespec` plus `pnpm gen`.

## Backend Guardrails

- API path policy: Product and Admin backend APIs both live under `/api/v1/*`, but MUST stay separated by origin, Go binary, TypeSpec service, OpenAPI artifact, SDK package, and Go bindings. Product public routes are `/api/v1/auth/*` (excluding `/api/v1/auth/logout`) and `/api/v1/status`; Product bearer-protected routes are `/api/v1/passkeys/*`, `/api/v1/sessions*`, and `/api/v1/auth/logout`. Admin routes belong only to the Admin origin/binary/artifacts; `/api/admin/*` is banned for Product/Admin contracts, generated artifacts, and BFF escape hatches.
- GORM imports are allowed only under `packages/backend/internal/adapter/postgres/**`
- `AutoMigrate` is banned; use `packages/backend/db/migrations/**` with `golang-migrate`

## Observability

- SigNoz UI: `http://localhost:3301`
- SigNoz OTLP endpoint: `http://localhost:4317` (gRPC), `http://localhost:4318` (HTTP)
- Go backend exports traces and metrics to SigNoz via OTLP gRPC
- Frontend browsers send traces to SigNoz via `PUBLIC_OTEL_COLLECTOR_URL`

## Change Operation

- The authoritative change-operation policy is `docs/change-operation.md`.
- Classify every change independently along three axes:
  - `Operation Lane`: `DIRECT`, `BEHAVIOR`, or `ARCHITECTURE`.
  - `UX Mode`: `NONE`, `CONTINUITY`, or `SHAPE`.
  - `Review Depth`: `STANDARD` or `DEEP`.
- `DIRECT` is limited to work that changes neither observable behavior nor material architecture. It does not require an OpenSpec Change.
- `BEHAVIOR` changes observable behavior and MUST use the `behavior-change` schema.
- `ARCHITECTURE` changes material internal structure and MUST use the `architecture-change` schema.
- UX shaping occurs only under `UX Mode: SHAPE`. `CONTINUITY` preserves an identified existing experience; `NONE` has no user-visible surface change.
- Actual UI changes require production-designer involvement and review in a real browser on desktop and mobile. Generated UI mockups are optional non-contract evidence.
- Use `STANDARD` review by default. Use `DEEP` review for high-impact security, data, external-contract, migration, cross-domain architecture, or active-change interaction risks, or when explicitly requested.

## OpenSpec (Persistent Behavior Contract)

- OpenSpec is the persistent contract for observable behavior, not the master implementation plan.
- OpenSpec is pinned to `1.8.0`. Its `new change` command does not use `openspec/config.yaml#schema` as the creation default, so always pass `--schema behavior-change` for `BEHAVIOR` or `--schema architecture-change` for `ARCHITECTURE`. Never hand-create a directory under `openspec/changes/`.
- OpenCode core definitions under `.opencode/commands/opsx-*.md` and `.opencode/skills/openspec-*/SKILL.md` are generated together from OpenSpec `1.8.0` by `pnpm gen:openspec` and must not be hand-edited. Repository-specific supplemental OpenSpec skills remain under `.opencode/skills/openspec/`.
- Main behavior specs live at `openspec/specs/**/spec.md`; active deltas live under `openspec/changes/*/specs/**/spec.md`.
- Every `#### Scenario:` heading MUST end with a stable Scenario ID such as `(USER-MGMT-S001)`.
- Automated TypeScript tests MUST reference Scenario IDs in titles such as `it('[USER-MGMT-S001] Create a user', async () => { ... })`; Go tests MUST reference the same IDs in test names or nearby comments recognized by the coverage verifier.
- Add `Tags: manual` near a Scenario only when automation is not possible.
- `scripts/openspec/verify-scenario-coverage.mjs` applies active deltas for structural, duplicate-ID, and conflict validation. Planning requires test references only for main specs and recognizes references declared by any active delta.
- Use `scripts/devcontainer/run.sh pnpm lint:openspec:scenario -- --change <change-id>` while planning, add `--require-test-references` at apply completion, then run the command without a selected Change to check all active-Change interactions.
- `tasks.md` is a coarse Work Package ledger. Plan file-level, helper-level, and test-level implementation progressively at runtime from the current package and evidence; do not persist a detailed master plan in OpenSpec.
- OpenSpec guardrails run through `pnpm lint:openspec` and include schema validation, strict artifact validation, proposal scope, Scenario/Test traceability, and task/design scope.
