---
name: ux-quality
description: Design or review implemented production UI for primary-task clarity, hierarchy, density, states, responsive behavior, accessibility, and consistency with the current design system.
---

# UX Quality

Evaluate the running production UI, not fidelity to a static artifact. Preserve
the current design system while making the primary user task safe and direct.

## Evidence Order

1. `Primary User Task` and `UX Direction`, or continuity evidence
2. Target route, adjacent surfaces, tokens, and shared UI
3. Implementation code and every reachable state
4. The running local UI
5. Findings grouped by user impact and root cause

## Quality Standard

### Primary Task and Action

- The purpose is quickly understandable.
- One primary action is clear and secondary actions do not compete with it.
- The result and available next step are understandable.
- Destructive or irreversible actions include enough context for informed action
  and safe recovery.

### Hierarchy and Density

- Visual weight matches functional importance.
- Headings, body copy, and supporting context establish a useful reading order.
- Related items are close and unrelated regions are clearly separated.
- The surface is not flattened into uniform cards, borders, spacing, and type.
- Remove anything that does not support the task, result comprehension, safe
  recovery, or accessibility.

### States

- Cover every reachable default, loading, empty, success, error, disabled, and
  permission state.
- Loading communicates purpose and progress.
- Empty states explain facts and return users to the primary task when possible.
- Error states avoid invented causes and provide actionable recovery.
- Disabled states do not rely on color alone and make the reason understandable.
- State changes avoid unnecessary layout shifts and focus loss.

### Responsive Behavior

- Preserve task order and priority across mobile, tablet, and desktop.
- Avoid clipping, overlap, unnecessary horizontal scrolling, and unusable
  controls.
- Adapt tables, long strings, and action groups for narrow widths.
- Keep touch targets large enough and adequately separated.
- Never add columns or cards merely to fill desktop space.

### Accessibility

- Use correct HTML semantics, heading order, landmarks, and form labels.
- Do not communicate meaning through color alone.
- Check 4.5:1 contrast for normal text and 3:1 for large text and essential
  graphical controls.
- Make all primary actions keyboard-operable.
- Keep visible focus distinguishable and focus order aligned with visual order.
- Manage initial focus, closing focus, and focus return for transient UI.
- Announce important asynchronous loading, success, and error changes when
  needed.
- Use motion to support meaning and respect `prefers-reduced-motion`.

### Current System Consistency

- Prefer existing tokens, shared components, component APIs, state patterns,
  and product vocabulary.
- Represent the same concept with the same appearance and interaction.
- Do not duplicate presentation that belongs in `packages/frontend/ui`.
- Evaluate a shared-system addition before introducing repeated local values.

### Product Specificity

- Do not default to generic admin-dashboard composition.
- Avoid repeated rounded cards, decorative gradients, glass effects, meaningless
  badges, filler copy, and excessive metrics.
- Use structure and spacing instead of wrapping everything in containers.
- Ground composition in this product's users, tasks, data, and vocabulary.
- Add no copy, shape, or motion only for appearance.

## Browser Review

- Exercise the implemented local UI whenever possible.
- Run primary Scenarios from start to finish.
- Test primary actions, secondary actions, and recovery with mouse and keyboard.
- Check both mobile and desktop widths.
- Reproduce loading, empty, and error states with safe local data or fixtures.
- Screenshots may evidence hierarchy and responsiveness, but never prove
  interaction quality.
- Record every unperformed check as residual risk.

## Findings

Each finding includes severity, route/state/viewport/input method, impact on the
primary task, `path:line` or browser reproduction evidence, and a correction
consistent with the current design system. Do not request preference-only
changes, mechanical Requirement-to-control conversion, or unapproved features.
Optional design tools remain aids, not quality gates.
