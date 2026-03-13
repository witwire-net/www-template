---
name: coding-guardian
description: Enforce this repository's actual coding rules and verification flow (TypeSpec -> generated artifacts -> implementation). Use when writing, modifying, refactoring, or reviewing code in this repository.
---

# Coding Guardian

この skill は、このリポジトリで実際に fail する規約と検証フローから外れないようにガードします。

- 返答言語: `AGENTS.md` に従う
- 重要: まず `CODING_STANDARDS.md` と enforcement entrypoint を読む
- 重要: API 契約の正は `packages/typespec/main.tsp`
- 重要: 生成物は手編集しない
- 重要: lint 回避は禁止（`eslint-disable` や inline config で逃げない）
- 重要: `openspec/**` は現在の default `pnpm lint` / hooks / CI の外。明示依頼があるときだけ触る

## Workflow

### 1) Load the repository rules before editing

最初に次を読む。

- `AGENTS.md`
- `CODING_STANDARDS.md`
- `CONTRIBUTING.md`
- `.opencode/skills/coding-guardian/references/repo-entrypoints.md`

特に重要な enforcement entrypoint:

- root flow: `package.json`, `.github/workflows/ci.yml`, `.husky/pre-commit`, `.husky/commit-msg`, `.lintstagedrc.json`, `commitlint.config.js`, `eslint.config.js`
- TypeSpec / OpenAPI: `packages/typespec/package.json`, `packages/typespec/.spectral.yaml`, `packages/typespec/spectral/path-policy.js`, `packages/typespec/spectral/app-security.js`, `packages/typespec/spectral/bearer-scheme.js`
- backend: `packages/backend/.golangci.yml`, `packages/backend/tools/analyzers/cmd/guardrails/main.go`, `packages/backend/internal/http/router_test.go`, `packages/backend/internal/http/openapi_contract_test.go`, `packages/backend/internal/app/runtime_test.go`
- scripts: `scripts/go/lint.sh`, `scripts/go/format-check.sh`, `scripts/go/guardrails.sh`, `scripts/go/verify-module.sh`, `scripts/security/lint-security.sh`, `scripts/codegen/check.sh`

### 2) Classify the change before editing

- Contract / codegen: `packages/typespec/**`, `packages/frontend/api/src/generated/**`, `packages/backend/internal/generated/**`
- Frontend: `packages/frontend/app/**`, `packages/frontend/domain/**`, `packages/frontend/ui/**`
- Backend: `packages/backend/**`
- Tooling / workflow: root config, scripts, hooks, CI, `.opencode/**`

固定の依存方向:

- Client: `packages/frontend/app -> packages/frontend/domain -> packages/frontend/api`
- Server: `packages/backend/cmd/api -> packages/backend/internal/app -> (packages/backend/internal/http | packages/backend/internal/persistence | packages/backend/internal/usecases) -> packages/backend/internal/domain -> packages/backend/internal/types`
- `packages/backend/internal/generated/openapi` を非 generated code から import できるのは `packages/backend/internal/http` だけ

### 3) Implement without breaking enforced rules

- Contract を変えるときは `packages/typespec/main.tsp` を直し、`pnpm gen` と `pnpm check:codegen` で整合を取る
- `packages/typespec/openapi/openapi.json`、`packages/frontend/api/src/generated/client.ts`、`packages/backend/internal/generated/openapi/openapi.gen.go` は手で直さない
- Frontend app / domain で `fetch`, `globalThis.fetch`, `axios`, `cross-fetch` を直接使わない
- Frontend app から `@www-template-frontend/api` を直 import しない。domain hook を経由する
- Active frontend source に React / TSX を持ち込まない
- Domain hooks は `use*` export、`{ data, actions }` 戻り値、stateful 実装は `.svelte.ts`
- Frontend app に SvelteKit server route / server hook / server-only lib を作らない
- Auth route mode は `packages/frontend/app/src/routes/app/+layout.ts` の `ssr = false` / `csr = true` に固定する
- Go file は `packages/backend/cmd/api`, `packages/backend/internal/*`, `packages/backend/tools/analyzers` の許可 layer にだけ置く
- GORM は `packages/backend/internal/persistence/**` だけ、`AutoMigrate` は禁止、migration は `packages/backend/db/migrations/*.sql`
- Non-generated Gin route は `/health` または `/api/v1/app/*` の string literal だけにする
- `fmt.Print*`, `print`, `println` と host-derived URL composition を backend code に入れない

### 4) Verify with the real repo flow

変更内容に応じて、少なくとも次を実行する。

- Contract / generated 変更: `pnpm gen` -> `pnpm check:codegen`
- JS / TS / Svelte / Go code 変更: `pnpm lint` -> `pnpm test:run`
- Release-ready な変更や横断変更: `pnpm build`
- TypeSpec 変更: `pnpm format:check` と `pnpm check`
- Skill 変更: `python3 .opencode/skills/opencode-skills-devkit/scripts/validate_skills.py --root .`

Changed-file 向けの軽量チェック:

- `.opencode/skills/coding-guardian/scripts/check_changed.sh [base]`

### 5) What to report back

- 触った領域（contract / frontend / backend / tooling / skill）
- どの enforced rule に合わせて設計したか
- 生成が必要だったか、実行したか
- 実行した command と結果
- まだ未実行の verify があれば、その理由

## Common violations to prevent

- generated file の手編集
- `packages/frontend/app` から `@www-template-frontend/api` の直 import
- frontend app / domain での `fetch` / `axios` / `cross-fetch`
- active frontend source での React / TSX
- `packages/frontend/app` での SvelteKit server route / server hook / form action
- GORM の layer 逸脱、`AutoMigrate`、migration pair 破れ
- `packages/backend/internal/http` から `packages/backend/internal/persistence` の直 import
- backend code での `fmt.Print*` や host-derived URL composition
