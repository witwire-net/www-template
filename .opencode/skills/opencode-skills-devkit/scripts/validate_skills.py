#!/usr/bin/env python3
"""Validate OpenCode skills in project/global locations.

This script checks the core rules from https://opencode.ai/docs/skills/:

- SKILL.md exists and has YAML frontmatter
- frontmatter includes name/description
- name matches the containing directory and the naming regex
- description length is within 1-1024 characters
- skill names are unique across all discovered locations

It also emits optional warnings (unknown frontmatter fields, vague descriptions).
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path


SKILL_NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
ALLOWED_FIELDS = {"name", "description", "license", "compatibility", "metadata"}


@dataclass(frozen=True)
class Issue:
    level: str  # ERROR or WARN
    path: Path
    message: str


def is_valid_skill_name(name: str) -> bool:
    if not (1 <= len(name) <= 64):
        return False
    return SKILL_NAME_RE.fullmatch(name) is not None


def read_frontmatter(path: Path) -> tuple[dict[str, object], list[str]]:
    """Return (data, unknown_keys).

    Minimal YAML frontmatter parser (stdlib-only). Supports:

    - key: value
    - metadata:\n  key: value
    - description: | / > blocks (basic)
    """

    text = path.read_text(encoding="utf-8", errors="replace")
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        raise ValueError("missing frontmatter start '---'")

    end_idx = None
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            end_idx = i
            break
    if end_idx is None:
        raise ValueError("missing frontmatter end '---'")

    fm_lines = lines[1:end_idx]
    data: dict[str, object] = {}
    unknown: list[str] = []

    i = 0
    while i < len(fm_lines):
        raw = fm_lines[i]
        line = raw.rstrip("\n")
        if not line.strip() or line.lstrip().startswith("#"):
            i += 1
            continue

        if line.startswith("metadata:"):
            meta: dict[str, str] = {}
            i += 1
            while i < len(fm_lines):
                raw2 = fm_lines[i]
                if raw2 and not raw2.startswith((" ", "\t")):
                    break
                item = raw2.strip()
                i += 1
                if not item or item.startswith("#"):
                    continue
                if ":" not in item:
                    continue
                k, v = item.split(":", 1)
                meta[k.strip()] = _parse_scalar(v.strip(), fm_lines, i)
            data["metadata"] = meta
            continue

        if ":" not in line:
            i += 1
            continue

        key, rest = line.split(":", 1)
        key = key.strip()
        value = rest.strip()

        if value in {"|", ">"}:
            block_lines: list[str] = []
            i += 1
            while i < len(fm_lines):
                raw2 = fm_lines[i]
                if raw2 and not raw2.startswith((" ", "\t")):
                    break
                block_lines.append(raw2.lstrip())
                i += 1
            block = "\n".join(block_lines).rstrip("\n")
            data[key] = block
        else:
            data[key] = _parse_scalar(value, fm_lines, i)
            i += 1

        if key not in ALLOWED_FIELDS and key not in unknown:
            unknown.append(key)

    return data, unknown


def _parse_scalar(value: str, fm_lines: list[str], i: int) -> str:
    # Very small scalar parser; enough for validation.
    v = value.strip()
    if not v:
        return ""
    if v.startswith('"') and v.endswith('"') and len(v) >= 2:
        # Unescape basic JSON-style escapes.
        try:
            import json

            return json.loads(v)
        except Exception:
            return v.strip('"')
    if v.startswith("'") and v.endswith("'") and len(v) >= 2:
        return v[1:-1]
    return v


def collect_skill_files(root: Path, include_global: bool) -> list[Path]:
    files: list[Path] = []

    for base in (
        root / ".opencode" / "skills",
        root / ".claude" / "skills",
    ):
        if not base.exists():
            continue
        files.extend(sorted(base.glob("*/SKILL.md")))

    if include_global:
        for base in (
            Path("~/.config/opencode/skills").expanduser(),
            Path("~/.claude/skills").expanduser(),
        ):
            if not base.exists():
                continue
            files.extend(sorted(base.glob("*/SKILL.md")))

    # Deduplicate.
    seen: set[Path] = set()
    out: list[Path] = []
    for p in files:
        rp = p.resolve()
        if rp in seen:
            continue
        seen.add(rp)
        out.append(p)
    return out


def validate(path: Path) -> list[Issue]:
    issues: list[Issue] = []
    try:
        data, unknown = read_frontmatter(path)
    except Exception as e:
        return [Issue("ERROR", path, str(e))]

    if unknown:
        issues.append(
            Issue(
                "WARN",
                path,
                f"unknown frontmatter fields (ignored by OpenCode): {', '.join(unknown)}",
            )
        )

    name = data.get("name")
    description = data.get("description")

    if not isinstance(name, str) or not name.strip():
        issues.append(Issue("ERROR", path, "missing frontmatter field: name"))
    if not isinstance(description, str) or not description.strip():
        issues.append(Issue("ERROR", path, "missing frontmatter field: description"))

    if isinstance(name, str):
        name = name.strip()
        if not is_valid_skill_name(name):
            issues.append(Issue("ERROR", path, f"invalid name: {name}"))

        dir_name = path.parent.name
        if dir_name != name:
            issues.append(
                Issue(
                    "ERROR",
                    path,
                    f"directory name must match frontmatter name (dir={dir_name}, name={name})",
                )
            )

    if isinstance(description, str):
        d = description.strip()
        if not (1 <= len(d) <= 1024):
            issues.append(Issue("ERROR", path, "description must be 1-1024 characters"))
        if "TODO" in d.upper():
            issues.append(Issue("WARN", path, "description contains TODO"))
        if len(d) < 24:
            issues.append(Issue("WARN", path, "description may be too short to be selectable"))

    return issues


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate OpenCode skill bundles")
    parser.add_argument("--root", default=".", help="Project root (default: .)")
    parser.add_argument(
        "--include-global",
        action="store_true",
        help="Also validate global skill locations",
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Treat warnings as errors",
    )
    args = parser.parse_args()

    root = Path(args.root).resolve()
    files = collect_skill_files(root, args.include_global)
    if not files:
        sys.stderr.write("No skills found.\n")
        return 0

    all_issues: list[Issue] = []
    names: dict[str, list[Path]] = {}

    for path in files:
        issues = validate(path)
        all_issues.extend(issues)

        # Track names for uniqueness checks when frontmatter is readable.
        try:
            data, _unknown = read_frontmatter(path)
        except Exception:
            continue
        name = data.get("name")
        if isinstance(name, str) and name.strip():
            names.setdefault(name.strip(), []).append(path)

    for name, paths in sorted(names.items()):
        if len(paths) > 1:
            joined = ", ".join(str(p) for p in paths)
            all_issues.append(
                Issue("ERROR", paths[0], f"duplicate skill name '{name}' across locations: {joined}")
            )

    errors = [i for i in all_issues if i.level == "ERROR"]
    warns = [i for i in all_issues if i.level == "WARN"]

    for issue in all_issues:
        prefix = issue.level
        sys.stderr.write(f"{prefix}: {issue.path}: {issue.message}\n")

    if errors:
        sys.stderr.write(f"\nFAILED: {len(errors)} error(s), {len(warns)} warning(s)\n")
        return 1

    if warns and args.strict:
        sys.stderr.write(f"\nFAILED (strict): {len(warns)} warning(s)\n")
        return 1

    sys.stderr.write(f"\nOK: {len(files)} skill(s) validated ({len(warns)} warning(s))\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
