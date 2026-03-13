#!/usr/bin/env python3
"""Create a new OpenCode skill bundle.

Default behavior prints the generated SKILL.md to stdout.
Use --write to create the directory and write files.

This script is intentionally stdlib-only so it works in most environments.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from pathlib import Path
from string import Template


SKILL_NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")


def is_valid_skill_name(name: str) -> bool:
    if not (1 <= len(name) <= 64):
        return False
    return SKILL_NAME_RE.fullmatch(name) is not None


def title_from_name(name: str) -> str:
    return " ".join(part.capitalize() for part in name.split("-"))


def _yaml_string(value: str) -> str:
    # YAML accepts JSON strings. This is a safe, predictable quoting strategy.
    return json.dumps(value, ensure_ascii=True)


def detect_lang(root: Path) -> str:
    # Very small heuristic: check README for Japanese characters.
    for candidate in (root / "README.md", root / "readme.md", root / "Readme.md"):
        if not candidate.exists():
            continue
        try:
            text = candidate.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        if re.search(r"[\u3040-\u30ff\u3400-\u9fff]", text):
            return "ja"
    return "en"


def load_template(skill_dir: Path, lang: str) -> str:
    template_name = "skill.template.ja.md" if lang == "ja" else "skill.template.en.md"
    path = skill_dir / "assets" / template_name
    return path.read_text(encoding="utf-8")


def build_frontmatter_extra(
    license_value: str | None,
    compatibility_value: str | None,
    metadata: dict[str, str],
) -> str:
    lines: list[str] = []

    if license_value:
        lines.append(f"license: {_yaml_string(license_value)}")
    if compatibility_value:
        lines.append(f"compatibility: {_yaml_string(compatibility_value)}")
    if metadata:
        lines.append("metadata:")
        for key in sorted(metadata.keys()):
            lines.append(f"  {key}: {_yaml_string(metadata[key])}")

    if not lines:
        return ""

    return "\n".join(lines) + "\n"


def parse_metadata(items: list[str]) -> dict[str, str]:
    out: dict[str, str] = {}
    for item in items:
        if "=" not in item:
            raise ValueError(f"Invalid --metadata entry (expected key=value): {item}")
        key, value = item.split("=", 1)
        key = key.strip()
        value = value.strip()
        if not key:
            raise ValueError(f"Invalid --metadata entry (empty key): {item}")
        out[key] = value
    return out


def skill_paths(root: Path, location: str, target: str, name: str) -> list[Path]:
    if location not in {"project", "global"}:
        raise ValueError("location must be project or global")
    if target not in {"opencode", "claude", "both"}:
        raise ValueError("target must be opencode, claude, or both")

    bases: list[Path] = []
    if location == "project":
        if target in {"opencode", "both"}:
            bases.append(root / ".opencode" / "skills")
        if target in {"claude", "both"}:
            bases.append(root / ".claude" / "skills")
    else:
        if target in {"opencode", "both"}:
            bases.append(Path("~/.config/opencode/skills").expanduser())
        if target in {"claude", "both"}:
            bases.append(Path("~/.claude/skills").expanduser())

    return [base / name / "SKILL.md" for base in bases]


def write_bundle(
    skill_md_paths: list[Path],
    content: str,
    with_bundle: bool,
    force: bool,
) -> None:
    for skill_md in skill_md_paths:
        skill_dir = skill_md.parent
        if skill_dir.exists() and not force:
            raise FileExistsError(f"Skill directory already exists: {skill_dir}")
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md.write_text(content, encoding="utf-8")

        if with_bundle:
            (skill_dir / "scripts").mkdir(exist_ok=True)
            (skill_dir / "assets").mkdir(exist_ok=True)
            (skill_dir / "references").mkdir(exist_ok=True)

            (skill_dir / "scripts" / "README.md").write_text(
                "# Scripts\n\nTODO: put runnable automation here.\n",
                encoding="utf-8",
            )
            (skill_dir / "assets" / "README.md").write_text(
                "# Assets\n\nTODO: templates and static files used by this skill.\n",
                encoding="utf-8",
            )
            (skill_dir / "references" / "README.md").write_text(
                "# References\n\nTODO: notes and links the agent should consult.\n",
                encoding="utf-8",
            )


def main() -> int:
    parser = argparse.ArgumentParser(description="Create a new OpenCode skill bundle")
    parser.add_argument("--root", default=".", help="Project root (default: .)")
    parser.add_argument("--name", required=True, help="Skill name (directory name)")
    parser.add_argument("--description", required=True, help="Short skill description")
    parser.add_argument("--license", dest="license_value", default=None, help="Optional license")
    parser.add_argument(
        "--compatibility",
        default="opencode",
        help="Optional compatibility label (default: opencode)",
    )
    parser.add_argument(
        "--metadata",
        action="append",
        default=[],
        help="Optional metadata (key=value). Can be repeated.",
    )
    parser.add_argument(
        "--target",
        choices=["opencode", "claude", "both"],
        default="opencode",
        help="Where to create the skill (default: opencode)",
    )
    parser.add_argument(
        "--location",
        choices=["project", "global"],
        default="project",
        help="Install location (default: project)",
    )
    parser.add_argument(
        "--lang",
        choices=["auto", "en", "ja"],
        default="auto",
        help="Template language (default: auto)",
    )
    parser.add_argument(
        "--with-bundle",
        action="store_true",
        help="Also create scripts/assets/references directories",
    )
    parser.add_argument(
        "--write",
        action="store_true",
        help="Write files instead of printing to stdout",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite existing skill directory",
    )
    args = parser.parse_args()

    name = args.name.strip()
    if not is_valid_skill_name(name):
        sys.stderr.write(
            "Error: invalid skill name. Expected ^[a-z0-9]+(-[a-z0-9]+)*$ (1-64 chars)\n"
        )
        return 2

    description = args.description.strip()
    if not (1 <= len(description) <= 1024):
        sys.stderr.write("Error: description must be 1-1024 characters.\n")
        return 2

    repo_root = Path(args.root).resolve()
    skill_dir = Path(__file__).resolve().parents[1]

    lang = detect_lang(repo_root) if args.lang == "auto" else args.lang
    tmpl = Template(load_template(skill_dir, lang))

    metadata = parse_metadata(args.metadata)
    frontmatter_extra = build_frontmatter_extra(
        args.license_value,
        args.compatibility,
        metadata,
    )

    content = tmpl.safe_substitute(
        name=name,
        title=title_from_name(name),
        description=_yaml_string(description),
        frontmatter_extra=frontmatter_extra,
    ).rstrip() + "\n"

    out_paths = skill_paths(repo_root, args.location, args.target, name)

    if not args.write:
        sys.stdout.write(content)
        sys.stderr.write("\n".join(str(p) for p in out_paths) + "\n")
        return 0

    write_bundle(out_paths, content, args.with_bundle, args.force)
    for p in out_paths:
        print(str(p))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
