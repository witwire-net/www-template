---
name: openspec-review
description: Review schema-specific OpenSpec Changes for purpose and means separation, contradictions, excess requirements, misinterpretation, and material omissions.
compatibility: Requires openspec CLI.
---

# OpenSpec Change Review

This is the shared semantic contract for proposer self-review,
`openspec/reviewer`, and `openspec/analyzer`.

## Source Precedence

1. `AGENTS.md` and enforced repository rules.
2. `openspec/config.yaml` and the selected schema.
3. Authoritative request interpretation in `proposal.md`.
4. Repository evidence relevant to the proposal.
5. Specs, architecture design when defined by the schema, and coarse tasks.

Require only artifacts defined by the selected schema. Deterministic validators
own structural checks.

## Purpose and Means

- Desired outcomes and outcome constraints may become Requirements and
  Scenarios.
- Technologies, components, dependencies, structures, algorithms, procedures,
  migrations, tools, files, commands, tests, and implementation sequences are
  means.
- Required means constrain architecture or tasks. Candidate means remain
  options. Neither becomes observable behavior merely because the owner named
  or mandated it.
- `proposal.md` records the classification and authoritative interpretation.
- `design.md` exists only for `architecture-change` and contains material
  decisions, not local implementation decomposition.
- `tasks.md` is a coarse work-package ledger, not a file or test-layer plan.

## Finding Categories

- `CONTRADICTION`: applicable sources require materially incompatible outcomes
  or plans.
- `OVERREQUIREMENT`: an artifact requires behavior, structure, work, or an
  operational condition beyond its justified boundary.
- `MISINTERPRETATION`: the Change changes the meaning of the resolved proposal,
  promotes means into behavior, or presents assumptions as facts.
- `MATERIAL_OMISSION`: missing information would force implementation to decide
  behavior, an external contract, architecture, security, data, dependency,
  runtime, scope, or material UX direction.

Files, private APIs, helper decomposition, test layers, fixture layout, and
within-ready-package order are not material omissions when resolved boundaries
permit local choice.

## UX Review

- `NONE`: no visible work may be introduced.
- `CONTINUITY`: visible behavior must preserve the proposal's identified current
  product precedent.
- `SHAPE`: visible behavior must preserve the proposal's approved primary user
  task and UX direction.

Do not run a second shaping pass during semantic review. Report only a material
contradiction, excess, misinterpretation, or omission.

## Procedure

1. Read all schema-returned `contextFiles` and relevant repository evidence.
2. Separate outcomes, constraints, required means, and candidate means from the
   proposal.
3. Trace every Requirement and Scenario only to outcomes or constraints.
4. For `architecture-change`, verify every material decision preserves Specs and
   includes a boundary and revisit trigger.
5. Verify each work package has justified coverage and objective evidence while
   leaving local implementation choices open.
6. Group one root cause into one finding and try to disprove it before reporting.

## Results

- `APPROVED`: required validation passes and no actionable finding remains.
- `CHANGES_REQUIRED`: artifact edits can resolve all findings without a new
  material decision.
- `DECISION_REQUIRED`: a material decision in the omission boundary is needed.
- `FAILED`: required evidence cannot be read or evaluated.

`APPROVED` means Planning Ready: implementation may decide files, private APIs,
helpers, test layers, fixtures, and ready-package order locally.

## Finding Format

```text
Category: CONTRADICTION | OVERREQUIREMENT | MISINTERPRETATION | MATERIAL_OMISSION
Evidence:
- <path:line observed fact>
Proposal impact: <exact outcome or boundary affected>
Material consequence: <wrong, unsafe, or unverifiable result>
Required outcome: <artifact state needed>
Decision required: none | <exact material decision and owner>
```

Do not emit preference-only warnings or duplicate deterministic failures as
semantic findings.
