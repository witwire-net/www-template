---
name: openspec-review
description: Review an OpenSpec Change for purpose/means separation, contradictions, unapproved requirements, misinterpretation, and material omissions. Use during proposer self-review and independent reviewer analysis.
compatibility: Requires openspec CLI.
---

# OpenSpec Change Review

This skill is the shared semantic review contract for the Proposer role,
`openspec/reviewer`, and final integration or lightweight review in
`openspec/analyzer`. The Proposer role includes both `openspec/proposer` and the
`/opsx-propose` command entrypoint. Its purpose is to ensure that a Change
defines the owner-confirmed desired state without turning implementation means
into product requirements, contradictions, added requirements,
misinterpretation, or material omissions.

It is not an apply preflight. `openspec/applier` reads the current Change
artifacts and owns task routing, runtime dependency analysis, safe parallelism,
and implementation review.

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

## Purpose and means boundary

- A desired outcome states how the product or an externally owned contract must
  be, independently of how the repository achieves that state.
- An outcome constraint limits the acceptable desired state without prescribing
  the mechanism used to achieve it.
- A means is a technology, component, dependency, structure, algorithm,
  procedure, migration, tool, file, command, or implementation sequence used to
  reach the desired state.
- Owner confirmation may make a means required for design, but it never turns a
  means into a desired outcome, outcome constraint, Requirement, or Scenario.
- Product Specs may contain only desired outcomes and outcome constraints.
  Required means belong in design and tasks; candidate means remain design
  options until selected. Neither belongs in Specs.
- Design, tasks, tests, implementation feasibility, and material implementation
  need are downstream evidence. They cannot create or justify a Requirement or
  Scenario.
- Traceability among downstream artifacts does not legitimize a means promoted
  into Specs. Trace every Requirement and Scenario directly to an independently
  confirmed desired outcome or outcome constraint.

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

Report an overrequirement when a Requirement, Scenario, visible surface, design
constraint, dependency, task, or completion condition exceeds the scope
permitted for its artifact.

For Specs and visible product surfaces, traceability to a desired outcome or
outcome constraint is required. A required means, enforced implementation rule,
or material implementation need never justifies adding product behavior.

For design and tasks, a required means, enforced repository rule, or material
implementation need may justify implementation work when it preserves the
approved desired state and outcome constraints.

Do not require speculative future work, optional controls, settings, screens,
abstractions, compatibility behavior, or operational work merely because they
commonly accompany a named solution.

### MISINTERPRETATION

Report a misinterpretation when the Change changes the meaning of the confirmed
request. In particular, detect required or candidate means promoted to
Requirements or Scenarios, assumptions presented as facts, generic practices
substituted for repository evidence, and specialist hypotheses turned into
product behavior.

Solution-shaped request terms must remain classified as desired outcomes,
outcome constraints, required means, or candidate means according to the
confirmed intent. Confirmation may move a candidate means to required means,
but never to desired outcome or outcome constraint merely because the owner
mandates the implementation.

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

- `intent.md` may record rejected interpretations, required and candidate
  means, assumptions, and falsification evidence. Means remain design inputs
  and are not product requirements.
- `proposal.md` may identify additions, modifications, removals, replacement
  impact, and breaking changes needed to explain the Change.
- Specs define enduring desired behavior and outcome constraints independently
  of implementation means. They must not make a technology, component,
  dependency, structure, algorithm, procedure, migration, tool, file, command,
  implementation sequence, absence of an old implementation, or rejected
  candidate the product outcome.
- A lasting negative safety or contract boundary is valid when it directly
  defines required system behavior, such as rejecting unauthorized access or
  preventing secret disclosure.
- `design.md` and `tasks.md` may explicitly describe deletion, replacement,
  migration, rollback, and file movement required to reach the approved end
  state.
- Tests verify independently justified end-state behavior and outcome
  constraints. A test never justifies creating a Requirement or Scenario, and
  tests must not exist solely to preserve historical implementation details or
  an otherwise unjustified means.
- When UI is in scope, `.wireframe.json` is the visible-surface source.
  Generated previews and screenshots are rendering evidence owned by
  `openspec/designer`, not additional design sources.

During Proposer self-review, apply all four categories to UI artifact authoring.
During independent `openspec/reviewer` or Analyzer review, UI findings are
limited to semantic `CONTRADICTION` with the approved visible surface. This
narrower independent-review scope prevents a second design pass while leaving
Proposer responsible for removing unapproved UI requirements and resolving
material UI decisions before review.

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

`openspec/reviewer` and `openspec/analyzer` may cite a validation failure but
must not duplicate it as one or more semantic findings. A format problem becomes
a semantic finding only when evidence shows that it causes one of the four
review categories above.

## Review procedure

1. Read the confirmed intent and separate desired outcomes, outcome constraints,
   required means, and candidate means.
2. Review every Requirement and Scenario against only the confirmed desired
   outcomes and outcome constraints. Do not use design, tasks, tests, or
   implementation necessity to justify Specs.
3. Read every artifact returned in `contextFiles` and the repository evidence
   needed to test the Change's remaining claims and scope.
4. Evaluate the four review categories without adding local review gates.
5. Group one root cause into one primary finding.
6. Return only findings that require an artifact correction or an owner
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
