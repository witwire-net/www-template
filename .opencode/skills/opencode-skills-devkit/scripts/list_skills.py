#!/usr/bin/env python3
"""List discovered OpenCode skills.

This is a convenience wrapper around the same discovery locations described in
https://opencode.ai/docs/skills/.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path


def read_frontmatter_field(path: Path, field: str) -> str | None:
    try:
        text = path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return None
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return None

    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            fm = lines[1:i]
            break
    else:
        return None

    prefix = field + ":"
    for line in fm:
        if line.startswith(prefix):
            return line[len(prefix) :].strip().strip('"').strip("'")
    return None


def collect_skill_files(root: Path, include_global: bool) -> list[Path]:
    files: list[Path] = []
    for base in (
        root / ".opencode" / "skills",
        root / ".claude" / "skills",
    ):
        if base.exists():
            files.extend(sorted(base.glob("*/SKILL.md")))

    if include_global:
        for base in (
            Path("~/.config/opencode/skills").expanduser(),
            Path("~/.claude/skills").expanduser(),
        ):
            if base.exists():
                files.extend(sorted(base.glob("*/SKILL.md")))

    return files


def main() -> int:
    parser = argparse.ArgumentParser(description="List OpenCode skills")
    parser.add_argument("--root", default=".", help="Project root (default: .)")
    parser.add_argument(
        "--include-global",
        action="store_true",
        help="Also list global skill locations",
    )
    args = parser.parse_args()

    root = Path(args.root).resolve()
    files = collect_skill_files(root, args.include_global)

    if not files:
        sys.stderr.write("No skills found.\n")
        return 0

    for p in files:
        name = read_frontmatter_field(p, "name") or p.parent.name
        desc = read_frontmatter_field(p, "description") or ""
        print(f"{name}\t{desc}\t{p}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
