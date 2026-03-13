# Repo Agent Examples

This repository already contains several well-structured agent definitions under `.opencode/agents/`.

Useful examples:

- `.opencode/agents/orchest.md`
  - Primary delegator with strict permissions.
  - Uses `permission.task` allowlist to prevent unexpected agents / infinite loops.
- `.opencode/agents/openspec/applier.md`
  - Delegate-only orchestrator subagent.
  - Good example of a narrow Task allowlist (`planner`/`unit/build/builder`/`unit/build/reviewer`).
- `.opencode/agents/unit/build/builder.md`
  - Implementer subagent.
  - Uses `permission.skill` allowlist and a scoped `permission.bash` policy.
- `.opencode/agents/unit/build/reviewer.md`
  - Hidden review-only subagent.
  - Good "final verdict" output contract.

Patterns to copy:

- Keep `description` short and specific (it is used for agent selection).
- Use least-privilege `permission` and deny destructive commands (e.g. `rm *`).
- If enabling Task, use allowlist-style `permission.task` (`"*": deny` first) and never allow self.
- Use a structured prompt body: First action -> Mission -> Inputs -> Protocol -> Output format.
