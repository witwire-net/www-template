---
name: opencode-skills-devkit
description: Create, validate, and maintain OpenCode SKILL.md bundles (templates, scripts, permissions, troubleshooting) for reuse across projects.
license: MIT
compatibility: opencode
metadata:
  audience: maintainers
  scope: skills
  lifecycle: stable
---

# OpenCode Skills Devkit

Create and maintain OpenCode skills as small, reusable "bundles": a `SKILL.md` plus optional `scripts/`, `assets/`, and `references/`.

## What I do

- Generate new skill skeletons with valid naming and frontmatter.
- Validate skills across project (and optionally global) locations.
- Recommend a consistent SKILL.md structure so agents can choose the right skill.
- Document permissions patterns (`permission.skill`) and discovery troubleshooting.

## When to use me

- You want to add a new skill under `.opencode/skills/<name>/SKILL.md` or `.claude/skills/<name>/SKILL.md`.
- A skill does not appear in the available skills list.
- You want to standardize skill metadata and structure across many repos.

## Skill bundle conventions

- One folder per skill name; the folder name must match `name` in frontmatter.
- `SKILL.md` must be spelled in ALL CAPS.
- Keep the `description` specific enough that an agent can confidently select the skill.
- Prefer small, composable skills. If a skill grows large, split it into multiple skills.
- Put runnable automation in `scripts/` and reference it from `SKILL.md`.

Recommended bundle layout:

```
.opencode/skills/<skill-name>/
  SKILL.md
  scripts/
  assets/
  references/
```

## Workflow

1. Pick a name

- Must match: `^[a-z0-9]+(-[a-z0-9]+)*$`
- 1-64 characters, no leading/trailing `-`, no consecutive `--`
- Must be unique across project + global + Claude-compatible locations

2. Generate a new skill skeleton (recommended)

Project-local (recommended for sharing via git):

```bash
python3 .opencode/skills/opencode-skills-devkit/scripts/new_skill.py \
  --name my-skill \
  --description "Describe what the skill does and when to use it" \
  --target opencode \
  --write
```

Global install (for reuse across repos):

```bash
python3 .opencode/skills/opencode-skills-devkit/scripts/new_skill.py \
  --name my-skill \
  --description "Describe what the skill does and when to use it" \
  --target opencode \
  --location global \
  --write
```

3. Validate skills

```bash
python3 .opencode/skills/opencode-skills-devkit/scripts/validate_skills.py --root .
```

To also validate global skill locations:

```bash
python3 .opencode/skills/opencode-skills-devkit/scripts/validate_skills.py --root . --include-global
```

4. Configure permissions (optional)

Example `opencode.json` snippet:

```json
{
  "permission": {
    "skill": {
      "*": "allow",
      "internal-*": "deny",
      "experimental-*": "ask"
    }
  }
}
```

## Bundled resources

- Generator: `scripts/new_skill.py`
- Validator: `scripts/validate_skills.py`
- Templates: `assets/skill.template.en.md`, `assets/skill.template.ja.md`

## Troubleshooting

If a skill does not show up:

1. Verify `SKILL.md` is spelled in all caps.
2. Check YAML frontmatter includes `name` and `description`.
3. Ensure the directory name matches `name`.
4. Ensure the skill name is unique across all locations.
5. Check permissions: skills with `permission.skill.<pattern> = "deny"` are hidden.
