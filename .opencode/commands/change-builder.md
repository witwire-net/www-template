---
description: Split a multi-part request by operation lane and dependencies, creating OpenSpec proposals only for non-DIRECT work.
agent: orchest
---

Input:

```text
$ARGUMENTS
```

Classify each work unit independently from repository evidence as
`lane: DIRECT | BEHAVIOR | ARCHITECTURE` and
`ux_mode: NONE | CONTINUITY | SHAPE`.

- `DIRECT`: create no Change; identify the implementation owner and verification boundary.
- `BEHAVIOR`: use `behavior-change`.
- `ARCHITECTURE`: use `architecture-change`.

Present each unit's outcome, scope, classification, dependencies, and safe
parallel groups for explicit approval. After approval, delegate only units that
require a Change to `openspec/proposer`. Independent proposals may run in
parallel. Do not implement from this command. Report evidence that each proposal
passes strict validation and Scenario coverage and returns `Planning Ready: YES`.
