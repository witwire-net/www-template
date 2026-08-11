---
name: ux-shaping
description: Shape UX for SHAPE-mode surfaces from user outcome, primary action, and necessary context. Use to prevent mechanical Requirement-to-control conversion.
---

# UX Shaping

Derive production UI direction from what the user must accomplish, never from a
catalog of features or internal data. Preserve this order:

```text
User outcome -> Primary action -> Necessary context
```

## Cognitive Model

### User Outcome

State in one sentence what the user can complete or understand. Do not derive
the surface from internal models, implementation state, or feature inventory.

### Primary Action

Choose the one action that most directly advances the outcome. For a read-only
surface, identify the result or state the user must understand first.

### Necessary Context

Keep information only when it is necessary to perform the primary action,
understand its result, recover safely, avoid irreversible loss, or use the
surface accessibly. Internal state, diagnostics, versions, model names, and
future configuration are hidden by default.

## Evidence Order

1. Target route, current page, adjacent surfaces, and current primary task
2. Shared UI, tokens, and contracts under `packages/frontend/ui/**`
3. Current Product and Admin components, states, and compositions
4. Current product vocabulary, copy, interaction rules, and responsive behavior
5. External first-party references only for a specific unresolved question

External precedent supplements product evidence. Never copy composition, copy,
branding, component APIs, assets, or dependencies from it.

## Process

1. Establish the target user, situation, and outcome.
2. Establish one `Primary User Task` and primary action.
3. List only necessary context.
4. Derive reading order and major regions from the task.
5. Set priority for primary actions, secondary actions, and supporting context.
6. Cover every reachable default, loading, empty, success, error, disabled, and
   permission state.
7. Preserve task order and priority across mobile and desktop.
8. Establish keyboard, focus, announcement, contrast, and reduced-motion needs.
9. Run the removal test on every visible candidate.

## Removal Test

Ask: "If this is removed, what does the user lose in task completion, result
comprehension, safe recovery, or accessibility?" Remove it when there is no
specific answer. Multiple equally prominent actions indicate a broken priority
model.

## Prohibitions

- Never map each Requirement to a button, field, card, setting, or label.
- Never map API fields, database columns, or internal state directly to UI.
- Never add regions because dashboards or admin screens usually have them.
- Never expose speculative future actions or settings.
- Never give every action equal visual weight.
- Never invent a component system before inspecting shared UI and current
  surfaces.
- Never prioritize external precedent over current product evidence.

## Output

```text
Status: DIRECTION_READY | OWNER_DECISION_REQUIRED | BLOCKED
Primary User Task: <central task>
UX Direction: <implementable hierarchy, flow, actions, states, and responsiveness>
User Outcome: <result obtained by the user>
Primary Action: <one action or focus>
Necessary Context:
- <information and rationale>
Existing Product Evidence:
- <path, route, or browser fact>
Shared UI Evidence:
- <reusable component, state, or token>
State Direction:
- <reachable states>
Responsive Direction:
- <mobile and desktop priorities>
Accessibility Direction:
- <semantics, keyboard, focus, announcements, contrast, and motion>
Removed Visible Items:
- <candidate and rationale>
Owner Decisions Required:
- none | <decision, options, and user-visible difference>
Evidence:
- <path:line, URL, or browser observation>
```

Use `DIRECTION_READY` only when the primary task and implementable direction are
settled. Use `OWNER_DECISION_REQUIRED` when a material choice changes the user
outcome or external contract. Use `BLOCKED` when required product evidence is
unavailable.
