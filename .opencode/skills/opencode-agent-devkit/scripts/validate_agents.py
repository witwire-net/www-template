#!/usr/bin/env python3
"""Validate OpenCode markdown agent definitions.

This is intentionally lightweight and stdlib-only.

Checks:

- File name matches the recommended kebab-case: ^[a-z0-9]+(-[a-z0-9]+)*$.
- File starts with YAML frontmatter (--- ... ---).
- Frontmatter includes a `description:` field.
- If `mode:` exists, it is one of: primary, subagent, all.

Repo policy checks (to reduce unexpected agents / infinite call loops):

- Disallow `permission: allow` (unrestricted) in project agents.
- If Task delegation is enabled, require allowlist-style `permission.task`:
  - First rule must be `"*": deny`
  - Only explicit agent names are allowed as keys (no globs)
  - Must not allow self
  - Allowed names must exist under `.opencode/agents/`
- Detect cycles in Task allowlists across agents.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


AGENT_NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")


def _strip_quotes(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
        return value[1:-1]
    return value


def _extract_frontmatter(text: str) -> list[str] | None:
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return None
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            return lines[1:i]
    return None


def _extract_permission_block(
    frontmatter: list[str],
) -> tuple[str | None, list[str] | None]:
    """Return (scalar_value, nested_lines).

    - scalar_value is set when `permission: <value>` is used.
    - nested_lines is set when `permission:` (block) is used.
    """

    for i, line in enumerate(frontmatter):
        if not line.lstrip().startswith("permission:"):
            continue
        rest = line.split(":", 1)[1].strip()
        if rest:
            return _strip_quotes(rest), None

        nested: list[str] = []
        for j in range(i + 1, len(frontmatter)):
            l = frontmatter[j]
            if not l.strip():
                nested.append(l)
                continue
            if l.startswith("  "):
                nested.append(l)
                continue
            break
        return None, nested
    return None, None


def _extract_permission_task_rules(
    perm_lines: list[str],
) -> tuple[str | None, list[tuple[str, str]] | None]:
    """Return (scalar_value, rules).

    - scalar_value is set when `task: <value>` is used.
    - rules is a list of (key, value) when block-style `task:` is used.
    """

    i = 0
    while i < len(perm_lines):
        line = perm_lines[i]
        if not line.strip():
            i += 1
            continue
        if not line.startswith("  task:"):
            i += 1
            continue

        rest = line.split(":", 1)[1].strip()
        if rest:
            return _strip_quotes(rest), None

        rules: list[tuple[str, str]] = []
        i += 1
        while i < len(perm_lines):
            l = perm_lines[i]
            if not l.strip():
                i += 1
                continue
            if not l.startswith("    "):
                break
            entry = l.strip()
            if ":" not in entry:
                i += 1
                continue
            key_raw, value_raw = entry.split(":", 1)
            key = _strip_quotes(key_raw.strip())
            value = _strip_quotes(value_raw.strip())
            rules.append((key, value))
            i += 1
        return None, rules

    return None, None


def _task_allow_edges(task_rules: list[tuple[str, str]]) -> list[str]:
    out: list[str] = []
    for key, value in task_rules:
        if value != "allow":
            continue
        if key == "*":
            continue
        out.append(key)
    return out


def validate_agent_file(
    path: Path, known_agents: set[str]
) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    edges: list[str] = []

    name = path.stem
    if not (1 <= len(name) <= 64 and AGENT_NAME_RE.fullmatch(name)):
        errors.append(f"invalid agent filename (recommended kebab-case): {path.name}")

    try:
        text = path.read_text(encoding="utf-8", errors="replace")
    except OSError as e:
        return [f"failed to read: {e}"], []

    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        errors.append("missing frontmatter start delimiter (---)")
        return errors, []

    end_index = None
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            end_index = i
            break
    if end_index is None:
        errors.append("missing frontmatter end delimiter (---)")
        return errors, []

    frontmatter = lines[1:end_index]
    has_description = any(
        line.lstrip().startswith("description:") for line in frontmatter
    )
    if not has_description:
        errors.append("frontmatter missing required field: description")

    for line in frontmatter:
        if not line.lstrip().startswith("mode:"):
            continue
        mode = line.split(":", 1)[1].strip().strip("\"'")
        if mode and mode not in {"primary", "subagent", "all"}:
            errors.append(f"invalid mode: {mode}")

    # Task delegation policy
    perm_scalar, perm_lines = _extract_permission_block(frontmatter)
    if perm_scalar is not None:
        if perm_scalar == "allow":
            errors.append("repo policy: disallow permission: allow (unrestricted)")
        return errors, []

    if perm_lines is None:
        return errors, []

    task_scalar, task_rules = _extract_permission_task_rules(perm_lines)
    if task_scalar is not None:
        if task_scalar in {"allow", "ask"}:
            errors.append(
                "repo policy: permission.task must be allowlist-style mapping (or deny), not a scalar allow/ask"
            )
        return errors, []

    if task_rules is None:
        return errors, []

    if not task_rules:
        errors.append("repo policy: permission.task mapping must not be empty")
        return errors, []

    first_key, first_value = task_rules[0]
    if not (first_key == "*" and first_value == "deny"):
        errors.append('repo policy: permission.task first rule must be "*": deny')

    for key, value in task_rules:
        if key != "*" and not AGENT_NAME_RE.fullmatch(key):
            errors.append(
                f"repo policy: permission.task key must be explicit agent name (no globs): {key}"
            )
        if key == "*" and value != "deny":
            errors.append(
                "repo policy: permission.task must not allow '*' (must be deny)"
            )
        if value not in {"allow", "deny", "ask"}:
            errors.append(
                f"repo policy: invalid permission.task value: {value} (key: {key})"
            )
        if key == name and value != "deny":
            errors.append("repo policy: permission.task must not allow self")
        if key != "*" and key not in known_agents:
            errors.append(
                f"repo policy: permission.task references unknown agent: {key}"
            )

    edges = _task_allow_edges(task_rules)

    return errors, edges


def _find_cycles(edges: dict[str, list[str]]) -> list[list[str]]:
    cycles: list[list[str]] = []
    visiting: set[str] = set()
    visited: set[str] = set()
    stack: list[str] = []

    def visit(node: str) -> None:
        visiting.add(node)
        stack.append(node)
        for nxt in edges.get(node, []):
            if nxt not in edges:
                continue
            if nxt not in visited and nxt not in visiting:
                visit(nxt)
                continue
            if nxt in visiting:
                try:
                    i = stack.index(nxt)
                except ValueError:
                    continue
                cycles.append(stack[i:] + [nxt])
        stack.pop()
        visiting.remove(node)
        visited.add(node)

    for node in sorted(edges.keys()):
        if node in visited:
            continue
        visit(node)

    # Best-effort de-dup
    uniq: list[list[str]] = []
    seen: set[str] = set()
    for c in cycles:
        sig = "->".join(c)
        if sig in seen:
            continue
        seen.add(sig)
        uniq.append(c)
    return uniq


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Validate OpenCode agents under .opencode/agents"
    )
    parser.add_argument("--root", default=".", help="Project root (default: .)")
    args = parser.parse_args()

    repo_root = Path(args.root).resolve()
    agents_dir = repo_root / ".opencode" / "agents"
    if not agents_dir.exists():
        sys.stderr.write(f"Error: agents directory not found: {agents_dir}\n")
        return 2

    agent_files = sorted(agents_dir.glob("*.md"))
    if not agent_files:
        sys.stderr.write("Warning: no agents found.\n")
        return 0

    known_agents = {p.stem for p in agent_files}

    all_errors: list[tuple[Path, list[str]]] = []
    task_edges: dict[str, list[str]] = {}
    for path in agent_files:
        errs, edges = validate_agent_file(path, known_agents)
        if errs:
            all_errors.append((path, errs))
        task_edges[path.stem] = edges

    if all_errors:
        for path, errs in all_errors:
            sys.stderr.write(f"FAIL: {path}\n")
            for err in errs:
                sys.stderr.write(f"  - {err}\n")
        sys.stderr.write(f"\nFAIL: {len(all_errors)} agent(s) failed validation\n")
        return 1

    cycles = _find_cycles(task_edges)
    if cycles:
        sys.stderr.write("FAIL: Task allowlist cycles detected\n")
        for c in cycles:
            sys.stderr.write(f"  - {' -> '.join(c)}\n")
        return 1

    sys.stdout.write(f"OK: {len(agent_files)} agent(s) validated\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
