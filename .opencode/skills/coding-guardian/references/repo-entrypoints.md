# Repository entrypoints

Read these files before applying `coding-guardian` in this repository.

## Core flow

- `AGENTS.md`: project workflow, required commands, language policy
- `CODING_STANDARDS.md`: mechanically enforced rules summary
- `CONTRIBUTING.md`: contributor workflow and required checks
- `package.json`: root command graph (`gen`, `lint`, `check`, `test:run`, `build`, `format:check`)
- `.github/workflows/ci.yml`: default CI order

## Git hooks

- `.husky/pre-commit`: `pnpm lint-staged`
- `.husky/commit-msg`: `pnpm commitlint --edit $1`
- `.lintstagedrc.json`: per-file staged commands
- `commitlint.config.js`: commit message type policy

## Frontend enforcement

- `eslint.config.js`: frontend boundaries for `web`/`app`/`domain`/`ui`, SvelteKit restrictions for `web`, SSR-disabled SPA restrictions for `app`, TSDoc, hooks rules, Svelte 5 rules

## Contract enforcement

- `packages/typespec/package.json`: TypeSpec format/check and Spectral command
- `packages/typespec/.spectral.yaml`: active Spectral ruleset
- `packages/typespec/spectral/path-policy.js`: `/api/v1/*` path policy
- `packages/typespec/spectral/app-security.js`: app endpoint bearer security requirement
- `packages/typespec/spectral/bearer-scheme.js`: `BearerAuth` security scheme requirement

## Backend enforcement

- `packages/backend/.golangci.yml`: enabled Go linters and depguard policy
- `packages/backend/tools/analyzers/cmd/guardrails/main.go`: custom backend guardrails
- `packages/backend/internal/adapter/http/router_test.go`: runtime route/auth checks
- `packages/backend/internal/adapter/http/openapi_contract_test.go`: OpenAPI bearer declaration check
- `packages/backend/internal/app/runtime_test.go`: fail-close token requirement outside development

## Helper scripts

- `scripts/go/lint.sh`
- `scripts/go/format-check.sh`
- `scripts/go/guardrails.sh`
- `scripts/go/verify-module.sh`
- `scripts/security/lint-security.sh`
- `scripts/security/govulncheck.sh`
- `scripts/security/gitleaks.sh`
- `scripts/security/osv-scanner.sh`
- `scripts/codegen/check.sh`
- `scripts/hooks/format-staged-go.sh`
- `scripts/hooks/verify-staged-migrations.sh`
