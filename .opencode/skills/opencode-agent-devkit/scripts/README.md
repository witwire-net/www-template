# Scripts

Runnable automation used by this skill.

- `new_agent.py`: Create a new agent file (prints to stdout by default, use `--write`).
- `validate_agents.py`: Validate `.opencode/agents/*.md` for basic frontmatter correctness.
  - Also enforces repo policy for Task allowlists (no self, no unknown agents, no cycles).

Examples

```bash
python3 .opencode/skills/opencode-agent-devkit/scripts/new_agent.py \
  --name review-lite \
  --description "Review changes without editing" \
  --mode subagent \
  --permission-preset review-subagent \
  --option reasoningEffort=high \
  --mission "Provide review feedback with evidence" \
  --output "Verdict: PASS or FAIL" \
  --write
```

Task allowlist example (delegate-only):

```bash
python3 .opencode/skills/opencode-agent-devkit/scripts/new_agent.py \
  --name delegate-lite \
  --description "Delegate work to a small set of approved subagents" \
  --mode subagent \
  --permission-preset review-subagent \
  --task-allow planner \
  --task-allow unit/build/builder \
  --task-allow unit/build/reviewer
```

```bash
python3 .opencode/skills/opencode-agent-devkit/scripts/validate_agents.py --root .
```
