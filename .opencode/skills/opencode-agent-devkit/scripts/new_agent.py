#!/usr/bin/env python3
"""Create a new OpenCode agent markdown definition.

This helper is designed to be used by an LLM-driven flow (via the `question` tool)
or by maintainers who want a consistent agent skeleton.

Default behavior prints the generated agent file content to stdout.
Use --write to create the file.

This script is intentionally stdlib-only so it works in most environments.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from string import Template
from typing import Any


AGENT_NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")


def is_valid_agent_name(name: str) -> bool:
    if not (1 <= len(name) <= 64):
        return False
    return AGENT_NAME_RE.fullmatch(name) is not None


def _dedupe_preserve_order(items: list[str]) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for item in items:
        if item in seen:
            continue
        seen.add(item)
        out.append(item)
    return out


def _yaml_string(value: str) -> str:
    # YAML accepts JSON strings. This is a safe, predictable quoting strategy.
    return json.dumps(value, ensure_ascii=True)


def _yaml_key(key: str) -> str:
    if re.fullmatch(r"[A-Za-z0-9_-]+", key):
        return key
    return _yaml_string(key)


def _yaml_scalar(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return str(value)
    if isinstance(value, str):
        return _yaml_string(value)
    raise TypeError(f"Unsupported YAML scalar type: {type(value)}")


def _yaml_lines(obj: Any, indent: int) -> list[str]:
    spaces = " " * indent
    if isinstance(obj, dict):
        lines: list[str] = []
        for key, value in obj.items():
            k = _yaml_key(str(key))
            if isinstance(value, dict):
                lines.append(f"{spaces}{k}:")
                lines.extend(_yaml_lines(value, indent + 2))
                continue
            lines.append(f"{spaces}{k}: {_yaml_scalar(value)}")
        return lines
    raise TypeError(f"Unsupported YAML object type: {type(obj)}")


def _bullets(items: list[str]) -> str:
    if not items:
        return "- TODO"
    return "\n".join(f"- {item}" for item in items)


def parse_extra_options(items: list[str]) -> dict[str, Any]:
    out: dict[str, Any] = {}
    for item in items:
        if "=" not in item:
            raise ValueError(f"Invalid --option entry (expected key=value): {item}")
        key, raw = item.split("=", 1)
        key = key.strip()
        raw = raw.strip()
        if not key:
            raise ValueError(f"Invalid --option entry (empty key): {item}")

        # Best-effort parsing of simple scalars.
        if raw.lower() in {"true", "false"}:
            value: Any = raw.lower() == "true"
        else:
            try:
                value = int(raw)
            except ValueError:
                try:
                    value = float(raw)
                except ValueError:
                    value = raw

        out[key] = value
    return out


def permission_preset(name: str) -> Any:
    if name == "none":
        return None
    if name == "unrestricted":
        # Shorthand supported by OpenCode.
        return "allow"
    if name == "readonly-subagent":
        return {
            "edit": "deny",
            "bash": "deny",
            "webfetch": "deny",
            "task": "deny",
            "read": "allow",
            "glob": "allow",
            "grep": "allow",
            "list": "allow",
            "lsp": "allow",
        }
    if name == "review-subagent":
        return {
            "edit": "deny",
            "webfetch": "deny",
            "task": "deny",
            "read": "allow",
            "glob": "allow",
            "grep": "allow",
            "list": "allow",
            "lsp": "allow",
            "bash": {
                "*": "ask",
                "git diff*": "allow",
                "git status*": "allow",
                "git log*": "allow",
                "git show*": "allow",
                "git grep*": "allow",
                "rm *": "deny",
            },
        }
    if name == "implementer-subagent":
        return {
            "edit": "allow",
            "webfetch": "deny",
            "task": "deny",
            "read": "allow",
            "glob": "allow",
            "grep": "allow",
            "list": "allow",
            "lsp": "allow",
            "bash": {
                "*": "allow",
                "git status*": "allow",
                "git diff*": "allow",
                "git log*": "allow",
                "pnpm lint*": "allow",
                "pnpm test*": "allow",
                "pnpm gen*": "allow",
                "pnpm build*": "allow",
                "go test*": "allow",
                "go vet*": "allow",
                "git add*": "deny",
                "git commit*": "deny",
                "rm *": "deny",
                "git push*": "deny",
            },
        }
    raise ValueError(f"Unknown permission preset: {name}")


def agent_path(
    repo_root: Path, location: str, name: str, custom_path: str | None
) -> Path:
    if location == "project":
        return repo_root / ".opencode" / "agents" / f"{name}.md"
    if location == "global":
        return Path("~/.config/opencode/agents").expanduser() / f"{name}.md"
    if location == "custom":
        if not custom_path:
            raise ValueError("--path is required when --location=custom")
        p = Path(custom_path).expanduser()
        if p.suffix != ".md":
            return p / f"{name}.md"
        return p
    raise ValueError("location must be project, global, or custom")


def load_body_template(skill_root: Path) -> Template:
    template_path = skill_root / "assets" / "agent.body.template.md"
    return Template(template_path.read_text(encoding="utf-8"))


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Create a new OpenCode agent markdown file"
    )
    parser.add_argument("--root", default=".", help="Project root (default: .)")
    parser.add_argument(
        "--location",
        choices=["project", "global", "custom"],
        default="project",
        help="Where to write the agent (default: project)",
    )
    parser.add_argument(
        "--path",
        default=None,
        help="Custom directory or file path when --location=custom",
    )
    parser.add_argument(
        "--name", required=True, help="Agent name (recommended kebab-case)"
    )
    parser.add_argument(
        "--description", required=True, help="Agent description (required)"
    )
    parser.add_argument(
        "--mode",
        choices=["primary", "subagent", "all"],
        default=None,
        help="Agent mode (optional)",
    )
    parser.add_argument(
        "--hidden",
        action="store_true",
        help="Hide a subagent from @ autocomplete (optional)",
    )
    parser.add_argument("--model", default=None, help="Model override (optional)")
    parser.add_argument(
        "--temperature", type=float, default=None, help="Temperature (optional)"
    )
    parser.add_argument(
        "--max-steps", type=int, default=None, help="maxSteps (optional)"
    )
    parser.add_argument(
        "--option",
        action="append",
        default=[],
        help="Extra frontmatter option key=value (repeatable)",
    )
    parser.add_argument(
        "--permission-preset",
        choices=[
            "none",
            "readonly-subagent",
            "review-subagent",
            "implementer-subagent",
            "unrestricted",
        ],
        default="review-subagent",
        help="Permission preset (default: review-subagent)",
    )
    parser.add_argument(
        "--task-allow",
        action="append",
        default=[],
        help="Allow invoking a subagent via Task (repeatable; disallows self)",
    )
    parser.add_argument(
        "--first-action",
        action="append",
        default=[],
        help="First action bullet (repeatable)",
    )
    parser.add_argument(
        "--mission",
        action="append",
        default=[],
        help="Mission bullet (repeatable)",
    )
    parser.add_argument(
        "--input",
        action="append",
        default=[],
        help="Input bullet (repeatable)",
    )
    parser.add_argument(
        "--rule",
        action="append",
        default=[],
        help="Hard rule bullet (repeatable)",
    )
    parser.add_argument(
        "--output",
        action="append",
        default=[],
        help="Output format bullet (repeatable)",
    )
    parser.add_argument(
        "--write",
        action="store_true",
        help="Write the file instead of printing to stdout",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite an existing agent file",
    )

    args = parser.parse_args()

    name = args.name.strip()
    if not is_valid_agent_name(name):
        sys.stderr.write(
            "Error: invalid agent name. Expected ^[a-z0-9]+(-[a-z0-9]+)*$ (1-64 chars)\n"
        )
        return 2

    description = args.description.strip()
    if not (1 <= len(description) <= 1024):
        sys.stderr.write("Error: description must be 1-1024 characters.\n")
        return 2

    repo_root = Path(args.root).resolve()
    skill_root = Path(__file__).resolve().parents[1]
    out_path = agent_path(repo_root, args.location, name, args.path)

    perm = permission_preset(args.permission_preset)

    task_allow = _dedupe_preserve_order(
        [s.strip() for s in args.task_allow if s.strip()]
    )
    if task_allow:
        if perm is None:
            perm = {}
        if isinstance(perm, str):
            sys.stderr.write(
                "Error: --task-allow cannot be used with --permission-preset=unrestricted (permission: allow)\n"
            )
            return 2
        for agent in task_allow:
            if not is_valid_agent_name(agent):
                sys.stderr.write(f"Error: invalid --task-allow agent name: {agent}\n")
                return 2
            if agent == name:
                sys.stderr.write(
                    "Error: --task-allow must not include the agent itself (self-delegation)\n"
                )
                return 2
        perm["task"] = {"*": "deny", **{a: "allow" for a in task_allow}}

    reserved_keys = {
        "description",
        "mode",
        "hidden",
        "model",
        "temperature",
        "maxSteps",
        "permission",
    }
    try:
        extra_options = parse_extra_options(args.option)
    except ValueError as e:
        sys.stderr.write(f"Error: {e}\n")
        return 2
    collisions = sorted(k for k in extra_options.keys() if k in reserved_keys)
    if collisions:
        sys.stderr.write(
            f"Error: --option uses reserved key(s): {', '.join(collisions)}\n"
        )
        return 2

    body_template = load_body_template(skill_root)
    body = (
        body_template.safe_substitute(
            name=name,
            first_action_bullets=_bullets(
                [s.strip() for s in args.first_action if s.strip()]
            ),
            mission_bullets=_bullets([s.strip() for s in args.mission if s.strip()]),
            inputs_bullets=_bullets([s.strip() for s in args.input if s.strip()]),
            hard_rules_bullets=_bullets([s.strip() for s in args.rule if s.strip()]),
            output_format_bullets=_bullets(
                [s.strip() for s in args.output if s.strip()]
            ),
        ).rstrip()
        + "\n"
    )

    frontmatter: list[str] = ["---", f"description: {_yaml_string(description)}"]
    if args.mode:
        frontmatter.append(f"mode: {args.mode}")
    if args.hidden:
        frontmatter.append("hidden: true")
    if args.model:
        frontmatter.append(f"model: {_yaml_string(args.model)}")
    if args.temperature is not None:
        frontmatter.append(f"temperature: {args.temperature}")
    if args.max_steps is not None:
        frontmatter.append(f"maxSteps: {args.max_steps}")

    for key in sorted(extra_options.keys()):
        frontmatter.append(f"{_yaml_key(key)}: {_yaml_scalar(extra_options[key])}")

    if perm is not None:
        if isinstance(perm, str):
            frontmatter.append(f"permission: {perm}")
        else:
            frontmatter.append("permission:")
            frontmatter.extend(_yaml_lines(perm, indent=2))
    frontmatter.append("---")

    content = "\n".join(frontmatter) + "\n\n" + body

    if not args.write:
        sys.stdout.write(content)
        sys.stderr.write(str(out_path) + "\n")
        return 0

    out_path.parent.mkdir(parents=True, exist_ok=True)
    if out_path.exists() and not args.force:
        sys.stderr.write(f"Error: agent file already exists: {out_path}\n")
        return 3
    out_path.write_text(content, encoding="utf-8")
    sys.stdout.write(str(out_path) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
