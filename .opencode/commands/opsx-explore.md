---
description: 'Enter explore mode - think through ideas, investigate problems, clarify requirements'
---

For every invocation, first load `openspec-explore` via `skill` and execute it
as the sole operating contract for exploration. Do not duplicate or override
its stance, artifact-write boundary, purpose/means routing, or Intent Handoff
rules in this command.

Treat `$ARGUMENTS` as the topic, question, change name, or comparison the user
wants to explore. When no argument is supplied, enter open-ended exploration.

If the user names an OpenSpec store, resolve it with
`openspec store list --json` and preserve `--store <id>` on every
store-aware read command. Exploration remains read-only regardless of store.

When the user wants to preserve the result, return the confirmed Intent Handoff
defined by `openspec-explore`; do not create or edit OpenSpec artifacts.
