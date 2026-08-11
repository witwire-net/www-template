---
name: coding-guardian
description: Enforce this repository's real Svelte 5, Go, Gin, GORM, TypeSpec, Admin, OpenSpec, and verification rules while editing code, docs, or tooling.
---

# Coding Guardian

Use the repository's enforced rules and command graph as the source of truth.

## First Actions

Read:

- `AGENTS.md`
- `CODING_STANDARDS.md`
- `CONTRIBUTING.md`
- `docs/change-operation.md`
- `.opencode/skills/coding-guardian/references/repo-entrypoints.md`

Classify the change through `Operation Lane`, `UX Mode`, and `Review Depth`
before selecting the workflow. OpenSpec is the persistent observable behavior
contract and is part of `pnpm lint`.

## Repository Boundaries

- API source of truth: `packages/typespec/main.tsp`.
- Generated Product and Admin OpenAPI, SDK, and Go bindings are never hand-edited.
- Product frontend: `packages/web -> packages/frontend/ui` and
  `packages/frontend/app -> packages/frontend/domain -> packages/frontend/api`,
  with app also depending on shared UI.
- Admin frontend: `packages/admin/app -> packages/admin/domain -> packages/admin/api`.
- Server: `packages/backend/cmd -> packages/backend/internal/app ->
(adapter/* | application | platform/*) -> packages/backend/internal/domain`.
- Product and Admin remain separate by origin, Go binary, TypeSpec service,
  OpenAPI artifact, SDK package, Go bindings, routes, and security boundary.
- GORM remains under `packages/backend/internal/adapter/postgres/**`.
- Migrations use paired SQL files under `packages/backend/db/migrations/**`;
  `AutoMigrate` is prohibited.

## Frontend Rules

- Use Svelte 5 syntax and the existing SvelteKit route modes.
- App pages and components obtain complete `{ data, actions }` contracts from
  domain hooks; they do not import API packages or perform raw network calls.
- Stateful domain composables live in `.svelte.ts` files.
- `packages/web` remains independent from domain and API packages.
- Shared presentation belongs in `packages/frontend/ui`; app and Admin surfaces
  compose it rather than duplicating primitives.
- User-facing copy uses the repository i18n ownership boundaries.
- Actual UI changes require the production designer and browser verification at
  mobile and desktop widths.

## Backend Rules

- Use repository-approved `pnpm` scripts for Go format, lint, build, and tests.
- Non-generated Gin routes are literal `/health` or `/api/v1/*` paths.
- HTTP adapters do not import persistence adapters or expose transport types
  inward.
- Domain and application code do not read runtime side-effect sources directly.
- Admin protected routes preserve operator bearer, Origin, Fetch Metadata,
  authorization, no-store, and security-header boundaries.
- Production-like runtime configuration fails closed when required credentials
  or surface configuration are missing.

## OpenSpec Rules

- `DIRECT` creates no Change.
- `BEHAVIOR` uses `behavior-change`; `ARCHITECTURE` uses
  `architecture-change`. Always pass `--schema` to `openspec new change`.
- Official `.opencode/commands/opsx-*.md` and
  `.opencode/skills/openspec-*/SKILL.md` files are regenerated only through
  `pnpm gen:openspec`.
- Specs contain observable outcomes and constraints, not implementation means.
- Every Scenario has a stable ID. Automated TypeScript and Go tests reference
  those IDs; only non-automatable Scenarios use `Tags: manual`.
- `tasks.md` is a coarse Work Package ledger. File, helper, test-layer, and
  execution-order choices remain runtime implementation details.

## Verification

Run commands through the Dev Container and the repository's `pnpm` scripts.

- OpenSpec or workflow changes: `pnpm gen:openspec`, `pnpm lint:openspec`
- JavaScript, TypeScript, Svelte, Go, docs, or tooling: `pnpm lint`,
  `pnpm test:run`
- TypeSpec or generated contracts: `pnpm gen`, `pnpm check:codegen`
- Type or compilation changes: `pnpm check`
- Release-ready cross-cutting changes: `pnpm build`
- Skill changes:
  `python3 .opencode/skills/opencode-skills-devkit/scripts/validate_skills.py --root .`
- Agent changes:
  `python3 .opencode/skills/opencode-agent-devkit/scripts/validate_agents.py --root .`

Report touched ownership areas, enforced boundaries, generation performed,
commands and outcomes, and any verification that could not run.
