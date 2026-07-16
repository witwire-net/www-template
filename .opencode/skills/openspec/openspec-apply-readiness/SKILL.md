---
name: openspec-apply-readiness
description: Evaluate and enforce whether OpenSpec artifacts can be executed by openspec/applier without scope changes or design rediscovery. Use before proposer finalization, analyzer review, and applier delegation.
compatibility: Requires openspec CLI.
---

# OpenSpec Apply Readiness

This skill is the shared acceptance contract for handing an OpenSpec change to
`openspec/applier`. It defines whether the artifacts are executable. It does not
define how to author each artifact or how to implement the change.

## Consumers

- `openspec/proposer` uses this contract as its definition of done before
  requesting analyzer review.
- `openspec/analyzer` audits this contract and reports only evidence-backed
  violations.
- `openspec/applier` runs this contract as a preflight before delegating work.
- Proposal commands that claim a change is apply-ready use the same contract.

## Source precedence

Evaluate readiness against these sources in order:

1. `AGENTS.md` and repository enforcement rules
2. `openspec/config.yaml` and the active OpenSpec schema
3. `openspec instructions apply --change "<change-id>" --json`
4. The change artifacts listed in `contextFiles`
5. This readiness contract

The active schema remains the source of truth for artifact structure and
content. Do not duplicate schema templates or artifact-specific formatting
rules in this skill.

## Readiness criteria

Every applicable criterion must pass. Mark a criterion not applicable only
with artifact evidence showing why the affected domain is outside the change.

### AR-001: Apply graph is complete

- `openspec instructions apply` does not report missing required artifacts.
- Every path in `contextFiles` exists and is readable.
- The confirmed intent, proposal, specs, design, and tasks describe the same change scope.

### AR-002: Scope and decisions are settled

- `intent.md` records `Intent-Status: CONFIRMED` and `Owner-Confirmation: CONFIRMED` after explicit owner confirmation.
- The confirmed intent separates repository observations, inferences, assumptions, and candidate means, and includes evidence from a check that could have invalidated the selected interpretation.
- Solution-shaped terms are classified as required outcomes, non-negotiable constraints, or candidate means; downstream artifacts do not promote candidate means without owner confirmation.
- No artifact contradiction or unresolved decision can change customer-visible
  behavior, external contracts, architecture ownership, security boundaries,
  persistence, dependencies, or UI behavior.
- The applier can preserve the agreed scope without editing OpenSpec artifacts.
- Local implementation choices that do not affect those boundaries may remain
  delegated to the responsible implementation agent.

### AR-003: Specs and design have a clean handoff

- Specs contain the required externally observable behavior and constraints.
- Design traces to the specs and does not introduce unstated product behavior.
- Design resolves cross-layer and domain-specific decisions needed to implement
  the stated behavior.

### AR-004: Implementation does not require design rediscovery

- Material implementation choices trace to the confirmed intent, approved specs, repository evidence, or explicit constraints. Familiarity, common practice, and readily available example code are not sufficient evidence.
- A responsible implementation agent can identify the intended flow,
  ownership boundaries, error handling, data flow, and affected integration
  points from the context files.
- Applicable generated artifacts, configuration, security boundaries, tests,
  and operational concerns are addressed.
- There are no placeholders such as `TBD`, `TODO`, `etc`, or implicit decisions
  that would force an implementer to redesign the change.

Assess coverage from the stated scope and repository evidence. Expected file
counts or other size heuristics are not readiness criteria.

### AR-005: Every task is routable

- Each task has one clear primary execution owner: frontend, backend, or build.
- A task that crosses ownership boundaries is split into dependency-ordered
  units unless the work must be atomic for a documented reason.
- Task wording and design references are sufficient for the applier to route
  the task without interpreting product or architecture intent.

### AR-006: Dependencies and parallelism are explicit

- Task order reflects real data, contract, generation, and implementation
  dependencies.
- Independent work is identifiable as safe to run in parallel.
- Shared files, generated artifacts, and upstream decisions that require
  serialization are identifiable before delegation.

### AR-007: Tasks have observable completion conditions

- Each task states a concrete outcome and points to the relevant design/spec
  context.
- Test tasks reference the applicable Scenario IDs and satisfy
  `openspec/config.yaml`.
- Each task names or inherits appropriate verification commands and has an
  objective completion signal.

### AR-008: Affected domains are implementation-ready

Apply only the relevant domain checks:

- UI: screens, states, user-facing copy, component ownership, responsive
  behavior, accessibility expectations, and required wireframes are explicit.
- API and contracts: TypeSpec ownership, generation order, accepted inputs,
  outputs, and error behavior are explicit.
- Persistence: schema effects, migration/rollback behavior, consistency, and
  security boundaries are explicit.
- Cross-domain flows: handoff contracts and ordering between frontend,
  backend, shared UI, generated code, and operations are explicit.

### AR-009: Ask-first boundaries are surfaced

- Dependency or version changes, permission boundary changes, destructive
  operations, external side effects, and other repository ask-first items are
  identified before execution.
- Required approvals are recorded, or the change is reported as requiring a
  decision rather than apply-ready.

### AR-010: Verification and review can converge

- The design and tasks identify applicable codegen, lint, test, build, and
  domain-specific verification.
- The planned work can pass through applicable frontend and backend review
  gates plus the final build review required by the applier.
- Completion does not depend on unverifiable claims or unavailable evidence.

## Evaluation procedure

1. Run `openspec instructions apply --change "<change-id>" --json`.
2. Read every path returned in `contextFiles`.
3. Evaluate AR-001 through AR-010 using repository and artifact evidence.
4. Return one overall result and a finding for each failed criterion.
5. Do not invent additional readiness gates in a consumer agent. Add a new
   criterion to this skill when a recurring applier gate is genuinely missing.

## Results

- `READY`: every applicable criterion passes.
- `NEEDS_FIXES`: artifact edits can satisfy the failed criteria without a new
  product or architecture decision.
- `NEEDS_DECISIONS`: customer, architecture, security, data, dependency, or UI
  decisions are required before artifacts can be finalized.
- `FAILED`: the target or required evidence cannot be read or evaluated.

`openspec/applier` maps every non-`READY` result to `BLOCKED` because it does
not own OpenSpec artifact changes.

## Finding format

```text
Criterion: AR-###
Result: NEEDS_FIXES | NEEDS_DECISIONS | FAILED
Evidence:
- path:line observation
Gap: exact reason the applier cannot proceed safely
Required correction: artifact outcome needed to satisfy the criterion
```

Do not reject a change for preferred wording, speculative future work, or
expected artifact size when the schema and every applicable readiness
criterion pass.
