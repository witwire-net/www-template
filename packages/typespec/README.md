# API Contract (TypeSpec)

This package is the single source of truth for the API contract.

- Edit entrypoint: `packages/typespec/main.tsp`
- API base path: `/api/v1` (template convention)
- Generate OpenAPI: `pnpm --filter @www-template/typespec gen:openapi`
- Output: `packages/typespec/openapi/openapi.json`

File layout (recommended)

- `packages/typespec/main.tsp`: service metadata + imports (keep thin)
- `packages/typespec/src/models/*`: request/response models
- `packages/typespec/src/routes/v1/*`: versioned routes (v1)
