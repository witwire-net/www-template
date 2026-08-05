---
name: openspec-change-review
description: Review an OpenSpec Change for contradictions, unapproved requirements, misinterpretation, and material omissions. Use during proposer self-review and independent analyzer review.
compatibility: Requires openspec CLI.
---

# OpenSpec Change Review

This skill is the shared semantic review contract for the Proposer role and
`openspec/analyzer`. The Proposer role includes both `openspec/proposer` and the
`/opsx-propose` command entrypoint. Its purpose is to ensure that a Change
expresses the owner-confirmed outcome without contradictions, added
requirements, misinterpretation, or material omissions.

It is not an apply preflight. `openspec/applier` consumes an approved handoff
and owns task routing, runtime dependency analysis, safe parallelism, and
implementation review.

## Source precedence

Review the Change against these sources in order:

1. `AGENTS.md` and enforced repository rules
2. `openspec/config.yaml` and the active OpenSpec schema
3. the owner-confirmed `intent.md`
4. repository evidence relevant to the confirmed outcome
5. proposal, Specs, wireframe sources, design, and tasks

The schema and templates own artifact structure and format. Existing commands
and repository validators own mechanically enforced checks. Do not recreate
those checks as semantic findings.

## Review categories

Every actionable finding must use exactly one primary category.

### CONTRADICTION

Report a contradiction only when two applicable sources require materially
incompatible outcomes or plans. Check alignment between confirmed intent,
proposal, Specs, approved visible surfaces, design, tasks, current repository
facts, and overlapping active Changes.

Formatting differences, alternate phrasing, and local implementation choices
are not contradictions.

### OVERREQUIREMENT

Report an overrequirement when a requirement, visible surface, design
constraint, dependency, task, or completion condition cannot be traced to an
owner-confirmed outcome, non-negotiable constraint, enforced repository rule,
or material implementation need.

Do not require speculative future work, optional controls, settings, screens,
abstractions, compatibility behavior, or operational work merely because they
commonly accompany a named solution.

### MISINTERPRETATION

Report a misinterpretation when the Change changes the meaning of the confirmed
request. In particular, detect candidate means promoted to requirements,
assumptions presented as facts, generic practices substituted for repository
evidence, and specialist hypotheses turned into product behavior.

Solution-shaped request terms must remain classified as required outcomes,
non-negotiable constraints, or candidate means according to the confirmed
intent.

### MATERIAL_OMISSION

Report a material omission only when missing information would force an
implementer to make a customer, contract, architecture, security, data,
dependency, or visible-surface decision, or would leave the requested outcome
unsafe or unverifiable.

Assess only domains affected by repository evidence. Applicable concerns may
include error behavior, failure recovery, authorization, sensitive data,
contract generation, persistence consistency, migration and rollback,
cross-domain handoffs, generated artifacts, configuration, and verification.
Do not turn this list into a mandatory coverage checklist.

Local implementation choices that preserve the confirmed behavior and all
material boundaries remain owned by the responsible implementation agent.

## Artifact boundaries

- `intent.md` may record rejected interpretations, candidate means,
  assumptions, and falsification evidence. Those records are not product
  requirements.
- `proposal.md` may identify additions, modifications, removals, replacement
  impact, and breaking changes needed to explain the Change.
- Specs define enduring observable behavior and lasting responsibility or
  safety boundaries. They must not make the absence of an old implementation,
  rejected candidate, package, or path the product outcome.
- A lasting negative safety or contract boundary is valid when it directly
  defines required system behavior, such as rejecting unauthorized access or
  preventing secret disclosure.
- `design.md` and `tasks.md` may explicitly describe deletion, replacement,
  migration, rollback, and file movement required to reach the approved end
  state.
- Tests verify required end-state behavior and constraints. They must not exist
  solely to prove that historical implementation details are absent.
- When UI is in scope, `.wireframe.json` is the visible-surface source.
  Generated previews and screenshots are rendering evidence owned by
  `openspec/designer`, not additional design sources.

During Proposer self-review, apply all four categories to UI artifact authoring.
During independent Analyzer review, UI findings are limited to semantic
`CONTRADICTION` with the approved visible surface. This narrower Analyzer scope
prevents a second design pass while leaving Proposer responsible for removing
unapproved UI requirements and resolving material UI decisions before review.

Do not reject text merely because it contains words associated with removal,
replacement, migration, switching, or negation. Review what the statement makes
normative and whether that meaning belongs in the artifact.

## Format and validation boundary

`openspec/proposer` owns deterministic conformance to the active schema and
templates, including required sections, exact structural labels, tables,
`Directory Tree`, `New / Changed Files`, task checkbox syntax, Scenario IDs,
and placeholder removal.

Existing validators remain authoritative for the conditions they enforce,
including artifact validity, intent status, Scenario ID coverage, Change-local
task scope, and generated wireframe preview drift.

`openspec/analyzer` may cite a validation failure but must not duplicate it as
one or more semantic findings. A format problem becomes a semantic finding only
when evidence shows that it causes one of the four review categories above.

## Review procedure

1. Read the confirmed intent and every artifact returned in `contextFiles`.
2. Read repository evidence needed to test the Change's claims and scope.
3. Evaluate the four review categories without adding local review gates.
4. Group one root cause into one primary finding.
5. Return only findings that require an artifact correction or an owner
   decision.

Do not use expected file counts, document length, preferred architecture,
generic best practices, or readily available examples as review evidence.

## Results

- `APPROVED`: required existing validation succeeds and no actionable semantic finding remains.
- `CHANGES_REQUIRED`: artifact edits can resolve every finding without a new
  owner decision.
- `DECISION_REQUIRED`: at least one owner, product, architecture, security,
  data, dependency, contract, or UI decision is required.
- `FAILED`: required evidence cannot be read or evaluated.

Validation failures remain separate from the four semantic finding categories,
but approval must not be granted while required validation is failing.

## Finding format

```text
Category: CONTRADICTION | OVERREQUIREMENT | MISINTERPRETATION | MATERIAL_OMISSION
Evidence:
- path:line observed fact
Confirmed intent impact: exact outcome or boundary affected
Material consequence: why the plan would produce the wrong, unsafe, or unverifiable result
Required outcome: artifact state needed to resolve the finding
Decision required: none | exact decision and owner
```

Do not emit preference-only warnings or notes. If no correction or decision is
required, it is not a review finding.
