---
name: opencode-agent-devkit
description: 'Generate OpenCode agent markdown definitions with safe permission presets and an interactive question flow.'
compatibility: 'opencode'
---

# Opencode Agent Devkit

Create `.opencode/agents/<name>.md` (or `~/.config/opencode/agents/<name>.md`) in the same format as `opencode agent create`, but driven by the `question` tool.

## What I do

- Collect the minimum required agent metadata via the `question` tool (name, description, mode, model, permissions).
- Validate agent identifiers (filename-safe, stable naming) and handle collisions (overwrite vs rename).
- Generate a consistent Markdown agent definition with YAML frontmatter + a structured prompt body.
- Apply safe, reusable permission presets (review/readonly/implementation) that match common OpenCode patterns.

## When to use me

Use this when you want to add or update an OpenCode agent under `.opencode/agents/`.

Ask follow-up questions (still via the `question` tool) when:

- The user wants a custom permission policy (beyond presets).
- The target path already exists.
- The agent name is invalid or ambiguous.

## Inputs I need

- Agent name (recommended: lowercase `kebab-case`, filename-safe)
- `description` (required; 1-2 sentences)
- `mode` (`primary`, `subagent`, or `all`)
- Prompt body (either inline text or `{file:./prompts/...}` reference)
- Permissions policy (preset or custom)

Optional inputs:

- `model`, `temperature`, `maxSteps`, `hidden`, provider-specific options

## Model + "Level" behavior (inherit vs pin)

OpenCode decides model and provider options based on whether you explicitly set them.

- If an agent does NOT set `model`:
  - `primary` agents use the globally configured model (from your OpenCode config).
  - `subagent` agents inherit the model from the invoking primary agent.
- If an agent sets `model`, it is pinned to that model (independent of the primary).

Provider-specific knobs (for example: `reasoningEffort: "high"`, `textVerbosity: "low"`, etc.) can be placed as additional frontmatter keys and will be passed through to the provider.

Recommendation:

- If you want subagents to track whatever model the user is currently using, omit `model` and omit provider-specific options.
- If you want consistent behavior (e.g., a deterministic reviewer), set `model` + low `temperature` explicitly.

## Outputs

- An agent definition file:
  - Project: `.opencode/agents/<name>.md`
  - Global: `~/.config/opencode/agents/<name>.md`
- A brief usage note: how to invoke the agent (e.g. `@<name>`)

## Workflow

1. Ask questions

- Use a single multi-question `question` call for the baseline.
- If the user chooses `custom` permissions, ask a second `question` call for details.

2. Validate and resolve names

- Prefer `kebab-case` names for stability.
- If the target file exists, ask: overwrite / rename / abort.

3. Generate agent file

- Use YAML `permission:` (not deprecated `tools:` booleans).
- Keep permissions least-privilege by default.
- For object-style permissions (e.g. `permission.bash`), rules are evaluated by wildcard match with the last matching rule winning. Put `"*"` first, and put specific deny/allow exceptions after it.
- For Task delegation allowlists (`permission.task`), also start with `"*": deny`, then allow only explicit subagent names. Do not allow the agent to call itself.
- If using `bash` allowlists, always deny destructive patterns like `rm *`.

4. Report

- Output the created path and an example invocation.

## Permission presets

These are suggested starting points. Customize as needed.

Note: `lsp` is experimental and only available when `OPENCODE_EXPERIMENTAL_LSP_TOOL=true` (or `OPENCODE_EXPERIMENTAL=true`).

### Preset: readonly-subagent

```yaml
permission:
  edit: deny
  bash: deny
  webfetch: deny
  task: deny
  read: allow
  glob: allow
  grep: allow
  list: allow
  lsp: allow
```

### Preset: review-subagent

```yaml
permission:
  edit: deny
  webfetch: deny
  task: deny
  read: allow
  glob: allow
  grep: allow
  list: allow
  lsp: allow
  bash:
    '*': ask
    'git diff*': allow
    'git status*': allow
    'git log*': allow
    'git show*': allow
    'git grep*': allow
    'rm *': deny
```

### Preset: implementer-subagent

```yaml
permission:
  edit: allow
  webfetch: deny
  task: deny
  read: allow
  glob: allow
  grep: allow
  list: allow
  lsp: allow
  bash:
    '*': allow
    'git add*': deny
    'git commit*': deny
    'git status*': allow
    'git diff*': allow
    'git log*': allow
    'pnpm lint*': allow
    'pnpm test*': allow
    'pnpm gen*': allow
    'pnpm build*': allow
    'go test*': allow
    'go vet*': allow
    'rm *': deny
    'git push*': deny
```

## Guardrails

- If you allow all bash (`"*": allow`), still deny destructive commands (e.g. `rm *`, `git push*`, `git commit*`, `git add*`) and keep the rest least-privilege where possible.
- If you enable Task delegation, use an allowlist-style `permission.task` (start with `"*": deny`, then allow only explicit subagent names). Never allow the agent to invoke itself, and avoid permitting other delegators to reduce the risk of infinite call loops.
- Avoid writing outside the project unless the user explicitly chooses global/custom paths.
- Never embed secrets in prompts.

## Bundled resources

- Scripts
  - `scripts/new_agent.py`
  - `scripts/validate_agents.py`
  - `scripts/README.md`
- Assets
  - `assets/agent.body.template.md`
  - `assets/README.md`
- References
  - `references/repo-agent-examples.md`
  - `references/README.md`

## Troubleshooting

- If the skill does not show up, verify:
  - `SKILL.md` is spelled in all caps
  - `name` and `description` exist in YAML frontmatter
  - The directory name matches `name`
  - The name is unique across project/global locations
  - Permissions: `permission.skill` is not set to `deny`
