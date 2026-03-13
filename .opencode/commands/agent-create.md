---
description: Create a new OpenCode agent file via interactive questions (replacement for `opencode agent create`).
agent: build
subtask: true
---

Preflight context

- Existing project agents: !`ls -1 .opencode/agents 2>/dev/null || true`
- Command arguments (optional): "$ARGUMENTS"

You are creating an OpenCode agent definition (NOT `AGENTS.md`).

First action

- Load the `opencode-agent-devkit` skill via the `skill` tool.

Hard rules

- Use the `question` tool for all user choices/clarifications (do not ask plain-text questions).
- Keep changes scoped to creating/updating ONE agent file (plus an optional prompt file if the user chooses a file-based prompt).
- Use `permission:` (not deprecated `tools:` booleans).

Workflow

1. Ask baseline questions in ONE multi-question `question` call.

Minimum questions:

- Save location: project (`.opencode/agents`) vs global (`~/.config/opencode/agents`) vs custom path
- Agent name (recommended: lowercase `kebab-case`). If `$ARGUMENTS` is non-empty, treat it as the initial suggested name.
- `description` (required; short)
- `mode`: `primary` / `subagent` / `all`
- If `subagent`: whether to set `hidden: true`
- Prompt source:
  - Inline prompt (generated from your answers)
  - Inline prompt (user provides a short mission + output format)
  - File-based prompt reference (`prompt: {file:...}`) and the file path to create
- Optional: `model` override (or "use default")
- Optional: provider-specific model options (passed through), e.g. `reasoningEffort: "high"`
- Optional: `temperature` (recommend 0.1 for subagents)
- Optional: `maxSteps`
- Permission policy:
  - `readonly-subagent`
  - `review-subagent`
  - `implementer-subagent`
  - `unrestricted`
  - `custom`

2. If the user chose `custom` permissions, ask a SECOND `question` call.

Minimum custom fields:

- `edit`: allow/ask/deny
- `bash` policy: deny / ask-all / allow-all / safe-allowlist / custom-patterns
- `webfetch`: allow/ask/deny
- `task` delegation: deny / allow(allowlist) / ask(allowlist)
- `lsp`: allow/ask/deny
- `skill`: deny-all / allow-all / allowlist(coding-guardian) / custom

If `bash` is `custom-patterns`, ask the user to paste a YAML object that will be used under `permission.bash`.

If `task` is not `deny`, also ask:

- Allowed subagents (allowlist): which subagent names may be invoked via Task. Do not include the agent itself.

When generating YAML, use allowlist-style `permission.task`:

```yaml
permission:
  task:
    '*': deny
    '<allowed-subagent>': allow
```

3. Validate and resolve

- Validate agent name (filename) as `^[a-z0-9]+(-[a-z0-9]+)*$`.
- If invalid, propose a corrected name and ask the user to confirm via `question`.
- Compute output path:
  - Project: `.opencode/agents/<name>.md`
  - Global: `~/.config/opencode/agents/<name>.md`
  - Custom: user-provided path
- If the target file exists, ask: overwrite / rename / abort.

4. Generate the agent definition

- YAML frontmatter must include `description`.
- Only include optional keys if the user selected them (`mode`, `hidden`, `model`, provider options, `temperature`, `maxSteps`, `prompt`, `permission`).
- Prompt body should be structured and actionable (role, mission, inputs, rules, output format).

5. Finish

- Report the created file path.
- Provide an example invocation (e.g. `@<name>`).
