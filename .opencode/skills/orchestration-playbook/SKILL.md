---
name: orchestration-playbook
description: 'Codify command essentials into a delegation protocol. Centered on grasp and intent, standardize formats for orders and reports, the control envelope, and stop conditions.'
compatibility: opencode
---

# Orchestration Playbook

Operational standard for an orchestrator to guide a set of subagents to mission success.
Do not constrain with prose. Align by templates.

## Definitions

Command essentials are expressed as two pillars:

- Maintain reliable grasp
- Under clear intent, issue timely and appropriate orders, and discipline execution

Supporting principles:

- Keep control minimal and leave room for initiative
- For grasp, prioritize alignment, situational awareness, and execution supervision

## When to use

- The number of subagents increases and decisions collide or state becomes inconsistent
- Variation in delegation instructions and reports causes rework
- Parallelization causes conflicts or duplicate work
- Ask-first boundaries or quality gates are violated

## What it provides

- Intent template
- Envelope template
- Mission-type order template
- Situation report template
- Postmortem template
- Work Order v1+ template
- Subagent reply template
- Integration memo template
- Acceptance checklist
- Failure mode mitigations

## Grasp

Grasp is the prerequisite for keeping control light.

- People
  - Capability
  - Trust level
  - Work speed
  - Expected quality
- Work
  - Goal
  - Deadline
  - Importance
  - Dependencies
- Means
  - Tools
  - Budget
  - Permissions
  - Constraints

If grasp is vague and you increase control, error grows. The stronger the grasp, the lighter the control can be.

## Intent

Intent pins the higher-level purpose and the end state in short sentences.

- Clear intent
  - Frees choice of means
  - Keeps decisions moving even when surprises happen
  - Stabilizes priorities
- Vague intent
  - Orders drift into details
  - Supervision collapses into step-by-step instruction
  - Initiative oscillates between runaway and paralysis

## Orders

Write orders around outcomes. Keep procedures to the minimum needed.

- Too-early orders constrain with insufficient information
- Too-late orders miss opportunities
- Too-long orders create interpretation forks

## Discipline

Discipline is not step-by-step instruction. It is behavior templates.

- Priority templates
- Reporting templates
- Exception handling templates
- Delegation scope templates

When discipline is in place, supervision decreases. When discipline breaks, orders increase.

## Control and initiative

Control is a safety device. Do not treat it as the steering wheel.

- What to forbid
- What to delegate
- Where to stop

Initiative is creativity within the intent boundary.

- Delegate choice of means
- Allow proposing alternatives
- Allow local optimization within bounds

Functional requirements:

- Intent is short
- Boundaries are explicit
- Reporting is fast
- Supervision is result-focused

## Three elements that support grasp

### Alignment

Alignment is the force that points movement in the same direction.

- Values
- Discipline
- Trust
- Role understanding
- Shared terminology

### Situational awareness

Situational awareness is the mechanism to learn the present state quickly.

- Progress
- Location
- Remaining capacity
- Change
- Risk

### Execution supervision

Supervision is not doing the work for them.

- Verify results
- Stop deviations
- Preserve learning

## Translation to an agent organization

### Vocabulary mapping

| Source term           | Meaning in agent organization | Implementation unit    |
| --------------------- | ----------------------------- | ---------------------- |
| Commander             | Orchestrator                  | Command process        |
| Subordinate units     | Subagents                     | Worker pool            |
| Mission               | Goal                          | Requirements spec      |
| Intent                | Intent                        | Intent document        |
| Order                 | Task assignment               | Instruction message    |
| Control               | Guardrails                    | Permission constraints |
| Initiative            | Autonomous execution          | Discretion envelope    |
| Alignment             | Alignment and trust           | Shared norms           |
| Situational awareness | State observation             | Telemetry              |
| Execution supervision | Evaluation and stop           | Review and interrupt   |

### Aim

- Fix direction via intent
- Keep control light while stopping deviation
- Accelerate correction via situational awareness and supervision

## Command cycle

The orchestrator runs a loop:

1. Observe

- State
- Reports
- External environment

2. Orient

- Align to intent
- Update priorities
- Assess risk

3. Decide

- Issue additional orders
- Reallocate
- Stop

4. Act

- Send instructions
- Adjust envelope
- Set supervision

The shorter the loop, the more timely the orders can be.

## Implementation patterns

### Pin intent first

Create an intent document first and share it with everyone.
Keep it short.

### Use mission-type orders

Write outcomes, not procedures.
Avoid enumerating branching paths.

### Use the envelope for control

Define the allowed operating range via an envelope:

- Allowed tools
- Whether external writes are allowed
- Money limit
- Time limit
- Iteration limit
- Ask-first boundaries
- Stop conditions

### Use standard reporting for situational awareness

Avoid free-form reports.
Use short fixed formats.

### Supervision in two layers

- Automated supervision
  - Constraint violations
  - Budget and time overruns
  - Dependency breakage
- Human supervision
  - Important decisions
  - Destructive changes
  - External exposure

Pair supervision with explicit stop authority.

## Adoption steps

Stage 0: Organization design

- Define roles
- Register capabilities
- Define responsibility boundaries for failure modes

Stage 1: Standardize intent

- Intent template
- Acceptance criteria
- Prohibitions

Stage 2: Standardize order format

- Mission-type instruction
- Deadlines and deliverables
- Explicit envelope
- Explicit stop conditions

Stage 3: Instrument situational awareness

- State store
- Progress events
- Metrics

Stage 4: Operate supervision

- Automatic stops
- Human review
- Postmortems

## Evidence discipline

- Attach evidence to each important claim
- Separate observation from inference
- If it cannot be observed, stop and report
- Completion is never inferred from intent, task labels, checkbox state, scenario ID strings, helper presence, import presence, or a delegate's `DONE` claim
- Completion requires both positive evidence and negative evidence; if either side is missing, return `NEEDS_FIX` and request the missing evidence from the responsible subordinate agent

Examples of evidence:

- File path and line number
- Summary of commands run
- Explanation of diffs

## Evidence-gated completion protocol

Use this protocol for every task that can be accepted, checked off, approved, or reported as complete.

- Positive evidence proves the required implementation now exists, is wired into production callers, and is covered by commands or tests
- Negative evidence proves prohibited states are absent, such as old owners, forbidden imports, stale callers, generated hand edits, secret leakage, bypasses, or route contamination
- Reviewer evidence proves the responsible reviewer inspected both positive and negative evidence and returned `Approve`
- Command evidence proves repository-approved commands ran through the allowed scripts and did not use direct tools or bypasses
- A completion report is perfect only when it includes positive evidence, negative evidence, reviewer evidence, command evidence, residual risks, and open items
- If the report is incomplete, contradictory, unsupported by `path:line`, or missing a required negative check, return `NEEDS_FIX`; keep returning `NEEDS_FIX` to the responsible subordinate agent until the report is complete
- Treat missing evidence, unchecked constraints, skipped negative checks, premature `DONE`, or task-order drift as a subordinate instruction violation, not as a neutral blocker
- When a subordinate agent violates an order, explicitly call out the violation, name the agent role, cite the violated instruction, and issue a corrective order; do not soften it into a risk or optional follow-up
- If a task includes removal, migration, deduplication, security, dependency boundary, generated artifact, or storage/secret claims, negative evidence is mandatory
- If the old implementation can still be found but is claimed to be non-production, require caller/callee evidence showing every production caller moved away
- If the evidence only proves helper/type/import/test-name existence, reject completion unless it also proves ownership, caller wiring, and old-state absence

## Correction and stop conditions

- NEEDS_FIX
  - Missing positive evidence from a subordinate report
  - Missing negative evidence from a subordinate report
  - Missing reviewer approval for a review-gated task
  - Incomplete or contradictory completion report
  - Premature `DONE`, premature checkbox update, skipped dependency gate, or task-order drift
  - Any fixable implementation or report defect owned by a subordinate agent
- ASK_FIRST_REQUIRED
  - Dependency additions/updates
  - Permission boundaries
  - Destructive changes
  - External side effects
  - Secrets
  - Legal and licensing
- BLOCKED
  - Missing external information not owned by a subordinate agent
  - No access
  - Spec contradictions
  - Required external decision
  - Required agent/tool is not allowed by permissions
  - Ask-first item cannot proceed without user approval
- CONFLICT
  - Areas likely to conflict with parallel tasks

## Guardrails

- Project-specific rules have highest priority
- Do not execute Ask-first items; stop and report
- Do not bypass lint or format
- Do not hand-edit generated artifacts
- Avoid cyclic delegation

## Included assets

- This skill is documentation-first
- Scripts and assets are omitted
- Reference: `references/shiki-no-yoketsu.ja.md`

## Operations

Templates are parts.
Operations are the sequence.

### Preparation

- Read project-specific rules
- Pin Ask-first boundaries
- Pin generated artifacts and required quality gates
- Pin allowed and disallowed tools

### Mission start

1 Create a grasp card

- One card per delegatee
- Fill speed, quality, and boundaries

2 Create an intent brief

- Fill MISSION and END_STATE first
- Fill INVARIANTS and NON_GOALS
- Fill ACCEPTANCE

3 Draw the envelope

- Fill TOOLS and OWNERSHIP_BOUNDARIES
- Fill ASK_FIRST and STOP_CONDITIONS
- Fill TIME_LIMIT and ITERATION_LIMIT

### Decomposition and allocation

4 Split tasks

- Keep between 3 and 9
- List dependencies
- Do not parallelize tasks that touch the same area

5 Issue Work Order v1+

- One message per task
- Align Goal to END_STATE
- Copy Invariants from the intent brief
- Copy Envelope from the envelope
- State the evidence policy

### Situational awareness and supervision

6 Receive replies

- Accept only the reply format
- For each claim, inspect PATH and LINE
- Reject replies that do not include both Positive evidence and Negative evidence when completion is claimed
- Return `NEEDS_FIX` to the responsible subordinate until the report is complete; do not downgrade missing evidence to a risk or follow-up
- Name the subordinate instruction violation when a required check or evidence class is omitted

7 Accept or request changes

- Accept if ACCEPTANCE is met
- Request changes if implementation is wrong but evidence is sufficient to diagnose it
- Return `NEEDS_FIX` if evidence is thin, missing, contradictory, or lacks required negative checks
- Return `BLOCKED` only for missing external decisions, no access, permission/tool impossibility, or true spec contradictions
- Stop on boundary violations

8 Integrate

- Record into the integration memo
- Keep Decisions and Next actions in short sentences
- Update the risk log

### Stop and approval

When ASK_FIRST_REQUIRED is raised, stop.
After you receive an approval decision, update the envelope and orders.

### Lock in learning

After the mission, record a postmortem.
Reflect changes into the next envelope and ACCEPTANCE.

### Parallel operations

- Partition parallel work by area
- If shared files are likely, remove them from parallelization
- Stop on CONFLICT and return to reallocation

## Templates

Replacement targets use the `__NAME__` form.
Replace values wrapped with `__`.

### Grasp card

```text
AGENT: __AGENT__
ROLE: __ROLE__
STRENGTHS:
- __STRENGTH_1__
- __STRENGTH_2__
- __STRENGTH_3__
WEAKNESSES:
- __WEAKNESS_1__
- __WEAKNESS_2__
- __WEAKNESS_3__
SPEED: __SPEED__
EXPECTED_QUALITY: __EXPECTED_QUALITY__
TRUST: __TRUST__
ALLOWED_TOOLS:
- __TOOL_1__
- __TOOL_2__
HARD_BOUNDARIES:
- __BOUNDARY_1__
- __BOUNDARY_2__
NOTES: __NOTES__
```

### Intent brief

```text
MISSION: __MISSION__
END_STATE: __END_STATE__
PRIORITY_ORDER:
- __PRIORITY_1__
- __PRIORITY_2__
- __PRIORITY_3__
INVARIANTS:
- __INVARIANT_1__
- __INVARIANT_2__
NON_GOALS:
- __NON_GOAL_1__
- __NON_GOAL_2__
TIMEBOX: __TIMEBOX__
ASK_FIRST:
- __ASK_FIRST_1__
- __ASK_FIRST_2__
ACCEPTANCE:
- __ACCEPTANCE_1__
- __ACCEPTANCE_2__
REPORTING_RHYTHM: __REPORTING_RHYTHM__
```

### Envelope

```text
TOOLS:
- __TOOL_1__
- __TOOL_2__
EXTERNAL_WRITES: __EXTERNAL_WRITES__
MONEY_LIMIT: __MONEY_LIMIT__
TIME_LIMIT: __TIME_LIMIT__
ITERATION_LIMIT: __ITERATION_LIMIT__
OWNERSHIP_BOUNDARIES:
- __OWNERSHIP_BOUNDARY_1__
- __OWNERSHIP_BOUNDARY_2__
ASK_FIRST:
- __ASK_FIRST_1__
- __ASK_FIRST_2__
STOP_CONDITIONS:
- NEEDS_FIX
- ASK_FIRST_REQUIRED
- BLOCKED
- CONFLICT
```

### Mission-type order

```text
TASK: __TASK__
DELIVERABLES:
- __DELIVERABLE_1__
- __DELIVERABLE_2__
DUE: __DUE__
SCOPE: __SCOPE__
CONSTRAINTS:
- __CONSTRAINT_1__
- __CONSTRAINT_2__
EVIDENCE:
- __EVIDENCE_1__
- __EVIDENCE_2__
REPORTING: __REPORTING__
STOP:
- __STOP_1__
- __STOP_2__
```

### Situation report

```text
STATUS: __STATUS__
INTENT_ECHO: __INTENT_ECHO__
PROGRESS: __PROGRESS__
DELIVERED:
- __DELIVERED_1__
NEXT:
- __NEXT_1__
BLOCKERS:
- __BLOCKER_1__
DECISIONS_NEEDED:
- __DECISION_1__
RISKS:
- __RISK_1__
EVIDENCE:
- __EVIDENCE_1__
COMMANDS:
- __COMMAND_1__
```

### Acceptance memo

```text
RESULT: __RESULT__
EVIDENCE_INDEX:
- __EVIDENCE_INDEX_1__
RISKS:
- __RISK_1__
OPEN_ITEMS:
- __OPEN_ITEM_1__
NEXT_ACTIONS:
- __NEXT_ACTION_1__
```

### Postmortem

```text
ACHIEVED:
- __ACHIEVED_1__
NOT_ACHIEVED:
- __NOT_ACHIEVED_1__
CAUSES:
- __CAUSE_1__
CONTROLS_THAT_HELPED:
- __CONTROL_HELPED_1__
CONTROLS_THAT_DID_NOT_HELP:
- __CONTROL_DID_NOT_HELP_1__
CHANGES_FOR_NEXT_TIME:
- __CHANGE_NEXT_TIME_1__
```

### Work Order v1+ orchestrator to subagent

```text
Work Order v1+
- Target agent: __AGENT__
- Goal: __END_STATE__
- Background: __BACKGROUND__
- Commander intent:
  - Priority: __PRIORITY__
  - Invariants:
    - __INVARIANT_1__
    - __INVARIANT_2__
  - Timebox: __TIME_LIMIT__
- Success criteria:
  - __ACCEPTANCE_1__
  - __ACCEPTANCE_2__
- Non-goals:
  - __NON_GOAL_1__
- Constraints / Guardrails:
  - Envelope:
    - Tools: __TOOLS__
    - External writes: __EXTERNAL_WRITES__
    - Ownership boundaries: __OWNERSHIP_BOUNDARIES__
    - Iteration limit: __ITERATION_LIMIT__
  - Ask first: __ASK_FIRST__
  - Stop conditions: __STOP_CONDITIONS__
  - No generated hand-edits; no lint or format bypass.
  - No cyclic delegation.
- Context to read:
  - __PATH_1__
- Searches to run:
  - glob: __GLOB_1__
  - grep: __REGEX_1__ include __PATTERN_1__
- Steps:
  1. Observe
     - How: __HOW__
     - Expected: __EXPECTED__
     - If fail: __TRIAGE__
  2. Decide
  3. Act
  4. Verify
- Commands to run:
  - __COMMAND_1__
- Evidence required in your reply:
  - __EVIDENCE_POLICY__
- Completion gate:
  - Positive evidence: __POSITIVE_EVIDENCE_REQUIRED__
  - Negative evidence: __NEGATIVE_EVIDENCE_REQUIRED__
  - Reviewer evidence: __REVIEWER_EVIDENCE_REQUIRED__
  - Command evidence: __COMMAND_EVIDENCE_REQUIRED__
  - If any item is missing or contradictory, return `NEEDS_FIX` instead of `DONE`.
- Correction or stop conditions return immediately:
  - NEEDS_FIX: __REASON_NEEDS_FIX__
  - ASK_FIRST_REQUIRED: __REASON_ASK_FIRST__
  - BLOCKED: __REASON_BLOCKED__
  - CONFLICT: __REASON_CONFLICT__
- Response format strict:
  - Status: __STATUS__
  - What I did: __WHAT_I_DID__
  - Evidence: __EVIDENCE__
  - Commands: __COMMANDS__
  - Notes/Risks: __NOTES_RISKS__
```

### Reply format subagent to orchestrator

```text
Status: __STATUS__

Intent echo:
- __INTENT_ECHO__

What I did:
- __WHAT_I_DID_1__

Delivered:
- __DELIVERED_1__

Positive evidence:
- __PATH_1__:__LINE_1__ __REQUIRED_STATE_NOW_EXISTS__

Negative evidence:
- __PATH_OR_SEARCH_1__ __PROHIBITED_STATE_IS_ABSENT__

Reviewer evidence:
- __REVIEWER_VERDICT__ __REVIEWER_REFERENCE__

Instruction violations:
- __AGENT_ROLE__ violated __INSTRUCTION__ by __VIOLATION__; corrective order: __ORDER__

Next:
- __NEXT_1__

Blockers:
- __BLOCKER_1__

Decisions needed:
- __DECISION_1__

Risks:
- __RISK_1__

Evidence:
- __PATH_1__:__LINE_1__ __CLAIM_1__

Commands:
- __COMMAND_1__: __RESULT_1__
```

### Integration memo

```text
Plan:
- __TASK_1__: __AGENT_1__ __DOD_1__

Decisions:
- __DECISION_1__ __REASON_1__

Next actions:
- __ACTION_1__

Evidence index:
- __PATH_1__:__LINE_1__

Risk log:
- __RISK_1__: __MITIGATION_1__
```

## Acceptance checklist

Orchestrator:

- END_STATE is observable
- ACCEPTANCE is met
- Each claim has evidence
- Positive evidence and negative evidence are both present for every completion claim
- Reviewer evidence is present when a review gate applies
- Completion reports with missing evidence are returned as `NEEDS_FIX`, not accepted as partial success
- NON_GOALS are not violated
- Ask-first boundaries are not crossed
- No hand-edits to generated artifacts
- No lint/format bypass

Subagent:

- INTENT_ECHO is short
- Stays within envelope boundaries
- Reporting is short and templated
- Can propose alternatives
- Stops before Ask-first boundaries

## Failure mode mitigations

- Intent becomes long
  - Return to intent brief
- Orders drift into procedures
  - Move TASK, DELIVERABLES, and DUE back to the top
- Control keeps growing
  - Update grasp cards
- Reports scatter
  - Standardize on situation report
- Supervision collapses into doing the work
  - Shift to acceptance and stop authority
- Evidence is thin
  - Require PATH:LINE and request changes

## Troubleshooting

If this skill does not appear in the list:

- Ensure the file is named `SKILL.md` (uppercase)
- Ensure the YAML frontmatter contains `name` and `description`
- Ensure the directory name matches `name`
- Ensure the name is unique
- Ensure `permission.skill` is not deny
