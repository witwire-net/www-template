## Primary Rules

- Think in English; MUST respond in **Japanese**.
- Before calling `task` for any subagent, you MUST read the target agent definition and verify both `permission.task` and any self-call prohibition such as `Do not self-call.`.
- You MUST doubt your assumptions, verify factual claims against available evidence, and MUST NOT present unsupported statements as facts.

## Commands

- Install: `corepack enable && pnpm install`
- Generate all contracts: `pnpm gen`
- Dev (all): `pnpm dev:all`
- Dev (server): `pnpm dev:server` (Go API on `http://localhost:8080`)
- Dev (client entry): `pnpm dev:client` (alias of `pnpm dev:web`, Vite on `http://localhost:5173`)
- Dev (web): `pnpm dev:web` (SvelteKit public site on `http://localhost:5173`)
- Dev (app): `pnpm dev:app` (SvelteKit SPA app on `http://localhost:5174`)

## API Contract (TypeSpec)

- Source of truth: `packages/typespec/main.tsp`
- Generated OpenAPI: `packages/typespec/openapi/openapi.json`
- Generated Go server bindings: `packages/backend/internal/generated/openapi/openapi.gen.go`
- Regenerate OpenAPI + SDK + Go bindings: `pnpm gen`
- Codegen drift check (CI-style): `pnpm check:codegen`

## Testing

- All unit tests: `pnpm test:run`
- Server tests: `pnpm test:server`
- Client tests: `pnpm test:client`
- E2E: `pnpm test:e2e`

## Architecture Notes

- Client dependency direction: `frontend/web -> frontend/domain -> frontend/api` and `frontend/app -> frontend/domain -> frontend/api`
- Server dependency direction: `backend/cmd -> backend/internal/app -> (backend/internal/http|backend/internal/persistence|backend/internal/usecases) -> backend/internal/domain -> backend/internal/types`
- API contract direction: implementation must follow TypeSpec; do not generate OpenAPI from server routes for SDK input.

## Backend Guardrails

- API path policy: all routes live under `api/v1/*`; public routes are `api/v1/auth/*` (excluding `api/v1/auth/logout`) and `api/v1/status`; bearer-protected routes are `api/v1/passkeys/*` and `api/v1/auth/logout`
- GORM imports are allowed only under `packages/backend/internal/persistence/**`
- `AutoMigrate` is banned; use `packages/backend/db/migrations/**` with `golang-migrate`
- OpenSpec is archived for now and is not part of the default `pnpm lint` / CI flow

## OpenSpec

- `openspec/**` is archived and not part of the default tooling loop
- Do not update OpenSpec artifacts for backend migration work unless explicitly requested
