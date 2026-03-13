## Why

<!-- TODO: Motivation. Describe the customer problem/opportunity, current pain, and why now (1-2 short paragraphs). -->
<!-- MUST NOT describe implementation steps or task ordering here. -->

## What Changes

<!-- TODO: User/operator-facing changes. List additions/changes/removals. Mark breaking changes with **BREAKING**. -->
<!-- Good: "Public frontend is served via SvelteKit SSR." -->
<!-- Bad: "Replace packages/frontend/public-app with SvelteKit." -->

## Spec Units

### New Spec Units

<!-- TODO: List new Spec Units (<domain>-fe / <domain>-be) and a 1-line scope summary for each. -->
<!-- Spec Units MUST be lasting capability/responsibility names, not change names or task names. -->
<!-- Bad: `setup-project-fe`, `migrate-backend-be`, `cleanup-ui-fe` -->
<!-- Good: `architecture-fe`, `timeline-fe`, `account-be` -->

- `<domain>-fe`: <!-- TODO: FE scope (screens, UX, client validation/state, error messages, etc.) -->
- `<domain>-be`: <!-- TODO: BE scope (API, persistence, invariants, errors, notifications, non-functional concerns, etc.) -->

### Modified Spec Units

<!-- TODO: List existing Spec Units whose requirements change, and describe what changes (at requirement level) in 1-2 lines. -->

- `<existing-domain>-fe`: <!-- TODO: Which requirements change and how (user-visible behavior). -->
- `<existing-domain>-be`: <!-- TODO: Which requirements change and how (API/data/constraints/response differences). -->

## Naming

<!-- TODO: Confirm the DOMAIN prefix for Scenario IDs (derived from <domain>) and that FE/BE prefixes differ (e.g., USER-MGMT-FE-S001 vs USER-MGMT-BE-S001). -->

## Impact

<!-- TODO: Impacted packages, APIs, external systems, DB, migrations, operations, monitoring, security, performance. -->
<!-- This section lists impact areas only. File-by-file implementation plans belong in design/tasks. -->
