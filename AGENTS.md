## Primary Rules

- Think in English; respond and output in Japanese.

## Commands

- Install: `corepack enable && pnpm install`
- Generate all contracts: `pnpm gen`
- Dev (all): `pnpm dev:all`
- Dev (server): `pnpm dev:server` (Go API on `http://localhost:8080`)
- Dev (client): `pnpm dev:client` (Vite on `http://localhost:5173`)

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

- Client dependency direction: `frontend/app -> frontend/domain -> frontend/api`
- Server dependency direction: `backend/cmd -> backend/internal/app -> (backend/internal/http|backend/internal/persistence|backend/internal/usecases) -> backend/internal/domain -> backend/internal/types`
- API contract direction: implementation must follow TypeSpec; do not generate OpenAPI from server routes for SDK input.

## Backend Guardrails

- API path policy: public routes use `api/v1/*`, app routes use `api/v1/app/*`
- GORM imports are allowed only under `packages/backend/internal/persistence/**`
- `AutoMigrate` is banned; use `packages/backend/db/migrations/**` with `golang-migrate`
- OpenSpec is archived for now and is not part of the default `pnpm lint` / CI flow

## OpenSpec

- `openspec/**` is archived and not part of the default tooling loop
- Do not update OpenSpec artifacts for backend migration work unless explicitly requested
