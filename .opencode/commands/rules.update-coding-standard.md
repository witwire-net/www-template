---
description: Update `CODING_STANDARDS.md` from this repo's lint/CI/git-hook rules with beginner-friendly NG/OK examples (no invented rules).
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Goal

Update `CODING_STANDARDS.md` so contributors can understand the enforced rules of this repository at a glance, **without reading configs first**.

This document is **lint-as-rules**: include **only** rules that are mechanically enforceable by the repo's lint commands, tests in the standard verification flow, CI, or git hooks.

## Hard Constraints

1. Source of truth is the **actual enforcement files in this repo**. If prose docs disagree with config/scripts/tests, config/scripts/tests win.
2. The target file is `CODING_STANDARDS.md` (plural), not `CODING_STANDARD.md`.
3. Do **not** invent rule IDs. The current document does not use IDs; keep it that way unless the user explicitly asks for IDs.
4. For each enforced rule, include:
   - 1-line summary (`required` / `forbidden` / `must live under ...`)
   - Enforcement point: command + config/test/hook + literal file path
   - `NG例` and `OK例` (short, concrete, beginner-friendly)
5. Include a `Git hooks` section that describes the exact current behavior:
   - `pre-commit`: `pnpm lint-staged` then `pnpm check:codegen`
   - `commit-msg`: `pnpm commitlint --edit $1`
   - Break down what `.lintstagedrc.json` actually runs for JS/TS, JSON/MD/YAML, Go, and migration SQL.
6. Spectral behavior must be precise:
   - `packages/typespec/package.json` runs `spectral lint openapi/openapi.json --ruleset .spectral.yaml --fail-severity error`
   - warnings do **not** fail the command; errors do.
7. Mention OpenSpec only to the extent mechanically true today: it is outside the default `pnpm lint`, hooks, and CI flow.
8. Use this repo's real file names and paths. Do **not** reference non-existent legacy paths such as `tools/scripts/*`, root `.spectral.yaml`, `eslint.config.mjs`, `commitlint.config.cjs`, or `.lintstagedrc.cjs`.

## Required Structure

`CODING_STANDARDS.md` MUST contain these headings (exact H2, in order):

## 0. 全体方針

## 1. 契約と生成

## 2. TypeSpec / OpenAPI

## 3. フロントエンド

## 4. バックエンド構造と依存

## 5. バックエンドの API / 認証 / 永続化

## 6. Go lint / format / security

## 7. CI 必須ステップ

## 8. Git hooks

## 9. 設定参照

If a section has no enforceable rules beyond a short scope note, keep it brief and do not pad it.

## Execution Steps

1. Read repo context docs for terminology and workflow:
   - `AGENTS.md`
   - `README.md`
   - `CONTRIBUTING.md`
   - `CODING_STANDARDS.md` (current)
2. Read the actual enforcement entrypoints:
   - `package.json`
   - `.github/workflows/ci.yml`
   - `.husky/pre-commit`
   - `.husky/commit-msg`
   - `.lintstagedrc.json`
   - `commitlint.config.js`
   - `eslint.config.js`
   - `packages/typespec/package.json`
   - `packages/typespec/.spectral.yaml`
   - `packages/typespec/spectral/path-policy.js`
   - `packages/typespec/spectral/app-security.js`
   - `packages/typespec/spectral/bearer-scheme.js`
   - `packages/backend/.golangci.yml`
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
3. Read custom analyzers and tests when they are part of actual enforcement:
   - `packages/backend/tools/analyzers/cmd/guardrails/main.go`
   - `packages/backend/internal/http/router_test.go`
   - `packages/backend/internal/http/openapi_contract_test.go`
   - `packages/backend/internal/app/runtime_test.go`
4. Extract only rules that actually fail in this repo, including repo-specific ones such as:
   - TypeSpec is the source of truth; generated OpenAPI / frontend SDK / Go bindings are not hand-edited; codegen drift fails.
   - Spectral enforces path policy and bearer security declarations for app endpoints.
   - Frontend boundaries such as `app -> domain -> api`, no direct API import from app, no direct `fetch`/`axios`/`cross-fetch`, no React/TSX in active frontend source, exported declarations require TSDoc, and auth route mode restrictions in SvelteKit.
   - Backend guardrails such as Go file placement, allowed internal/external imports by layer, GORM only in persistence, `AutoMigrate` banned, migration filename/pair policy, route literal/path policy, generated folder policy, banned `fmt.Print*`/`print`/`println`, and banned host-derived URL composition.
   - Auth fail-close behavior outside development (`APP_BEARER_TOKEN` requirement) when enforced by tests.
   - Exact CI step order and exact git hook behavior.
5. Update `CODING_STANDARDS.md` following the constraints above.
6. Before finishing, sanity-check that every cited rule maps to a real failing command/test/hook in this repo and that every referenced file path exists.

## Notes

- This command is the canonical way to update `CODING_STANDARDS.md`. Mention it in `## 9. 設定参照` or an equivalent operations note:
  - `opencode run --command rules.update-coding-standard`
- Prefer concise explanations over exhaustive config dumps. A beginner should understand "what fails, where, and how to fix it" in a few minutes.
- Do not paste large config blocks into `CODING_STANDARDS.md`; summarize the rule and cite the enforcement point.
