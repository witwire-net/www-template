## Primary Rules

- Think in **English**; MUST respond in **Japanese**.
- Before calling `task` for any subagent, you MUST read the target agent definition and verify both `permission.task` and any self-call prohibition such as `Do not self-call.`.
- You MUST doubt your assumptions, verify factual claims against available evidence, and MUST NOT present unsupported statements as facts.

## Credo

you must translate the Credo below into English and **repeat it back verbatim.** Only after repeating it back should you begin the work.

1. あらゆる意思決定は顧客ファーストで考えること。誰がどのように利用し、どうすれば喜ばれるかを常に考えること。
2. セキュリティはなによりも優先されること。セキュリティ最優先が、なにより顧客のためになる。
3. 常に完璧なプロダクトであること。妥協、横着、顧客にとって意味のないプロダクトを作ることは一切許されない。仮置きを残す、後回し、コメントにしておいて放置に決してしてはならない。
4.
5. すべてのルールには意図がある。必ず意図を理解すること。意図を理解しないまま改定したり、逆に遵守しようとしてはならない。
6. 後方互換性は完全悪だ。後方互換性のためのコードや計画がある時点で、そのシステムは一切認められない。常に完璧なプロダクトであるために、不要な機能は即座に削除。

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
- Dev (app): `pnpm dev:app` (SvelteKit SPA app on `http://app.localhost:5174`)
- Dev (admin): `pnpm dev:admin` (Admin Console on `http://admin.localhost:5176`)

## Command Policy

- For both backend and frontend work, lint, typecheck, build, and test MUST be invoked through `pnpm` scripts.
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
- OpenSpec is archived for now and is not part of the default `pnpm lint` / CI flow

## Observability

- SigNoz UI: `http://localhost:3301`
- SigNoz OTLP endpoint: `http://localhost:4317` (gRPC), `http://localhost:4318` (HTTP)
- Go backend exports traces and metrics to SigNoz via OTLP gRPC
- Frontend browsers send traces to SigNoz via `PUBLIC_OTEL_COLLECTOR_URL`

## OpenSpec

- `openspec/**` is archived and is not part of the default tooling loop
- Do not update OpenSpec artifacts for backend migration work unless explicitly requested
