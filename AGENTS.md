## Primary Rules

- Think in **English**; MUST respond in **Japanese**.
- Before calling `task` for any subagent, you MUST read the target agent definition and verify both `permission.task` and any self-call prohibition such as `Do not self-call.`.
- You MUST doubt your assumptions, verify factual claims against available evidence, and MUST NOT present unsupported statements as facts.

## Credo

1. あらゆる意思決定は顧客ファーストで考えること。誰がどのように利用し、どうすれば喜ばれるかを常に考えること。
2. セキュリティはなによりも優先されること。セキュリティ最優先が、なにより顧客のためになる。
3. 常に完璧なプロダクトであること。妥協、横着、顧客にとって意味のないプロダクトを作ることは一切許されない。スコープ外だからと切り捨てず、解決する見込みがないならその場で直すこと。
4. 仕事は完璧に完了すること。仮置きを残す、後回しにすることは一切許されない。
5. すべてのルールには意図がある。必ず意図を理解すること。意図を理解しないまま改定したり、逆に遵守しようとしてはならない。
6. あなたは極めて優秀なエージェントだ。あなたならどんな困難な課題も解決できる。素晴らしい成果を期待している。

## Code Comments

- Leave detailed Japanese comments for every single process in the code.
- Clarify the intent, input/output, and side effects of each step so that future readers (including yourself) can understand immediately.

## Documentation Comments (TS Docs / Go Docs)

- TSDoc (TypeScript) and GoDoc (Go) comments must be written in Japanese, providing detailed, multi-line explanations of their roles and parameter meanings.
- Every public API (functions, methods, types, interfaces, and structs) must have a documentation comment in Japanese that describes what it does, the meaning of each argument and return value, error cases, and usage examples.

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

- Client dependency direction: `web -> frontend/ui` (web is a public LP; it MUST NOT depend on domain or api), `frontend/app -> frontend/domain -> frontend/api` (also `frontend/app -> frontend/ui`)
- Server dependency direction: `backend/cmd -> backend/internal/app -> (backend/internal/adapters/http|backend/internal/adapters/persistence/postgres|backend/internal/adapters/persistence/valkey|backend/internal/adapters/webauthn|backend/internal/adapters/mailer|backend/internal/auth/application|backend/internal/platform/*) -> backend/internal/auth/domain -> backend/internal/platform/*`
- API contract direction: implementation must follow TypeSpec; do not generate OpenAPI from server routes for SDK input.

## Backend Guardrails

- API path policy: all routes live under `api/v1/*`; public routes are `api/v1/auth/*` (excluding `api/v1/auth/logout`) and `api/v1/status`; bearer-protected routes are `api/v1/passkeys/*` and `api/v1/auth/logout`
- GORM imports are allowed only under `packages/backend/internal/adapters/persistence/**`
- `AutoMigrate` is banned; use `packages/backend/db/migrations/**` with `golang-migrate`
- OpenSpec is archived for now and is not part of the default `pnpm lint` / CI flow

## Observability

- Grafana: `http://localhost:3000` (admin/admin)
- Prometheus: `http://localhost:9090`
- Tempo (trace): `http://localhost:3200`
- Loki (logs): `http://localhost:3100`
- OTel Collector OTLP: `http://localhost:4317` (gRPC), `http://localhost:4318` (HTTP)
- Start observability stack: `pnpm dev:observability`
- Go backend exposes `/metrics` for Prometheus scraping
- Frontend browsers send traces to Collector via `PUBLIC_OTEL_COLLECTOR_URL`

## OpenSpec

- `openspec/**` is archived and is not part of the default tooling loop
- Do not update OpenSpec artifacts for backend migration work unless explicitly requested
