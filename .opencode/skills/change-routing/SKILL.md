---
name: change-routing
description: Classify a request as DIRECT, BEHAVIOR, or ARCHITECTURE with an independent UX mode and affected domains. Use before implementation or OpenSpec planning begins.
---

# Change Routing

Classify work from customer outcome and repository evidence, not from the
solution named in the request.

## Required Input

- Customer outcome
- Scope and non-goals
- Required and candidate means
- Relevant files, existing Specs, current surfaces, and external contracts
- Unresolved material decisions

If an unresolved ambiguity can change the classification, request one focused
decision and do not emit speculative YAML.

## Operation Lane

- `DIRECT`: changes neither established observable behavior nor an external
  contract and requires no material architecture decision. Create no OpenSpec
  Change. A local fix that restores existing specified behavior is `DIRECT`.
- `BEHAVIOR`: changes an outcome observable by a user, caller, or external
  contract without requiring a material architecture decision. Use
  `behavior-change`.
- `ARCHITECTURE`: requires a material decision about responsibility boundaries,
  dependency direction, data ownership, security boundaries, dependencies,
  runtime, migration, rollback, or cross-domain structure. Use
  `architecture-change`.

Do not select `ARCHITECTURE` from words such as "migration", "refactor", or
"architecture" alone. Select it only when a material decision must persist.

## UX Mode

- `NONE`: no visible surface, copy, interaction, state, or responsive behavior
  changes.
- `CONTINUITY`: visible implementation changes while an identified current
  experience, primary task, and composition remain authoritative.
- `SHAPE`: a new surface or a material change to the primary task, flow,
  information hierarchy, visible copy, states, or recovery requires UX shaping.

Touching a UI file does not imply `SHAPE`. Backend and API work is not `NONE`
when it changes a visible state or action result. Lane and UX mode remain
independent; a behavior or architecture Change can use any UX mode.

## Affected Domains

List only domains that require implementation or domain-specific review, in
this order:

1. `frontend`: `packages/web/**`, `packages/frontend/**`, visible Admin Console
   behavior, shared UI, and browser behavior
2. `backend`: `packages/backend/**`, `packages/typespec/**`, `packages/admin/**`,
   server contracts, persistence, and runtime
3. `build`: repository configuration, generation, tests, tooling, and CI that do
   not belong to frontend or backend

Include both frontend and backend when a TypeSpec change affects both, or when
an Admin Console change affects both visible UI and its backend-owned package
boundary. Use an empty array when no domain implementation is required.

## Output Contract

Return only this YAML shape, with no fence, preamble, or additional key:

```yaml
lane: BEHAVIOR
ux_mode: SHAPE
affected_domains:
  - frontend
reason: 'Adds a new primary user task without a material architecture decision.'
```

- `lane`: `DIRECT | BEHAVIOR | ARCHITECTURE`
- `ux_mode`: `NONE | CONTINUITY | SHAPE`
- `affected_domains`: unique values from `frontend`, `backend`, `build`
- `reason`: one evidence-backed sentence covering all selected fields

## Prohibitions

- Do not turn a requested means into the customer outcome.
- Do not add domains or ceremony from convention or filenames alone.
- Do not guess unresolved material decisions.
- Do not author Specs, design, tasks, or code while classifying.
