#!/usr/bin/env python3
"""
Generate an AGENTS.md skeleton for a repository.

Default behavior: print to stdout.
Use --write to write to AGENTS.md (or --output).
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from string import Template
from typing import Any, Iterable


EXCLUDED_DIR_NAMES = {
    ".git",
    ".hg",
    ".svn",
    ".idea",
    ".vscode",
    "__pycache__",
    "node_modules",
    "vendor",
    "dist",
    "build",
    "out",
    "coverage",
    ".next",
    ".nuxt",
    ".cache",
    ".venv",
    "venv",
    ".mypy_cache",
    ".ruff_cache",
    ".pytest_cache",
    "target",
}


@dataclass(frozen=True)
class RepoScan:
    root: Path
    package_manager: str | None
    package_json: dict[str, Any] | None
    make_targets: list[str]
    top_level_dirs: list[str]
    style_hints: list[str]
    has_readme_ja: bool


def _read_text(path: Path, max_bytes: int = 256_000) -> str | None:
    try:
        data = path.read_bytes()
    except OSError:
        return None
    if len(data) > max_bytes:
        data = data[:max_bytes]
    try:
        return data.decode("utf-8", errors="replace")
    except Exception:
        return None


def _looks_japanese(text: str) -> bool:
    return re.search(r"[\u3040-\u30ff\u3400-\u9fff]", text) is not None


def detect_lang(root: Path) -> str:
    for candidate in (root / "README.md", root / "readme.md", root / "Readme.md"):
        if not candidate.exists():
            continue
        content = _read_text(candidate)
        if content and _looks_japanese(content):
            return "ja"
    return "en"


def detect_package_manager(root: Path) -> str | None:
    if (root / "pnpm-lock.yaml").exists() or (root / "pnpm-workspace.yaml").exists():
        return "pnpm"
    if (root / "yarn.lock").exists():
        return "yarn"
    if (root / "package-lock.json").exists():
        return "npm"
    if (root / "bun.lockb").exists() or (root / "bun.lock").exists():
        return "bun"
    if (root / "package.json").exists():
        return "npm"
    return None


def read_package_json(root: Path) -> dict[str, Any] | None:
    path = root / "package.json"
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


def parse_make_targets(makefile: Path) -> list[str]:
    text = _read_text(makefile, max_bytes=512_000)
    if not text:
        return []
    targets: list[str] = []
    for line in text.splitlines():
        if not line or line.startswith(("\t", " ")):
            continue
        if line.lstrip().startswith("#"):
            continue
        if "=" in line and ":" not in line:
            continue
        m = re.match(r"^([A-Za-z0-9][A-Za-z0-9._-]*)\s*:(?![=])", line)
        if not m:
            continue
        target = m.group(1)
        if target.startswith("."):
            continue
        if target not in targets:
            targets.append(target)
    return targets


def list_top_level_dirs(root: Path) -> list[str]:
    dirs: list[str] = []
    for child in root.iterdir():
        if not child.is_dir():
            continue
        if child.name in EXCLUDED_DIR_NAMES:
            continue
        if child.name.startswith("."):
            continue
        dirs.append(child.name)
    return sorted(dirs)


def detect_style_hints(root: Path) -> list[str]:
    hints: list[str] = []
    paths = {
        "Prettier": [
            root / ".prettierrc",
            root / ".prettierrc.json",
            root / ".prettierrc.yaml",
            root / ".prettierrc.yml",
            root / ".prettierrc.js",
            root / "prettier.config.js",
            root / "prettier.config.cjs",
            root / "prettier.config.mjs",
        ],
        "ESLint": [
            root / "eslint.config.js",
            root / "eslint.config.mjs",
            root / ".eslintrc",
            root / ".eslintrc.js",
            root / ".eslintrc.cjs",
            root / ".eslintrc.json",
            root / ".eslintrc.yaml",
            root / ".eslintrc.yml",
        ],
        "TypeScript": [root / "tsconfig.json"],
        "EditorConfig": [root / ".editorconfig"],
        "Ruff": [root / "ruff.toml", root / ".ruff.toml"],
        "Black": [root / "pyproject.toml"],
        "Go fmt": [root / "go.mod"],
        "rustfmt": [root / "Cargo.toml"],
    }
    for name, candidates in paths.items():
        if any(p.exists() for p in candidates):
            hints.append(name)
    return hints


def scan_repo(root: Path) -> RepoScan:
    package_manager = detect_package_manager(root)
    package_json = read_package_json(root)
    make_targets = parse_make_targets(root / "Makefile") if (root / "Makefile").exists() else []
    top_level_dirs = list_top_level_dirs(root)
    style_hints = detect_style_hints(root)

    readme_lang = detect_lang(root)
    has_readme_ja = readme_lang == "ja"

    return RepoScan(
        root=root,
        package_manager=package_manager,
        package_json=package_json,
        make_targets=make_targets,
        top_level_dirs=top_level_dirs,
        style_hints=style_hints,
        has_readme_ja=has_readme_ja,
    )


def _pm_run(pm: str, script: str) -> str:
    if pm == "pnpm":
        return f"pnpm {script}"
    if pm == "yarn":
        return f"yarn {script}"
    if pm == "bun":
        return f"bun run {script}"
    return f"npm run {script}"


def _pick_script(scripts: dict[str, Any], keys: Iterable[str]) -> str | None:
    for key in keys:
        if key in scripts and isinstance(scripts[key], str):
            return key
    return None


def suggest_commands(scan: RepoScan) -> dict[str, str]:
    pm = scan.package_manager or "npm"
    pkg = scan.package_json or {}
    scripts = pkg.get("scripts") if isinstance(pkg.get("scripts"), dict) else {}

    install_cmd = {
        "pnpm": "pnpm install",
        "yarn": "yarn install",
        "bun": "bun install",
        "npm": "npm ci",
    }.get(pm, "npm ci")

    dev_key = _pick_script(scripts, ["dev", "start"])
    build_key = _pick_script(scripts, ["build"])
    lint_key = _pick_script(scripts, ["lint"])
    format_key = _pick_script(scripts, ["format", "fmt", "format:check", "prettier"])
    typecheck_key = _pick_script(scripts, ["typecheck", "check", "tsc"])
    test_key = _pick_script(scripts, ["test", "test:unit", "test:ci"])

    def cmd_for(key: str | None, fallback: str) -> str:
        if key:
            return _pm_run(pm, key)
        return fallback

    dev_cmd = cmd_for(dev_key, "TODO: add dev command")
    build_cmd = cmd_for(build_key, "TODO: add build command")
    lint_cmd = cmd_for(lint_key, "TODO: add lint command")
    format_cmd = cmd_for(format_key, "TODO: add format command")
    typecheck_cmd = cmd_for(typecheck_key, "TODO: add typecheck command")

    test_fast_cmd = cmd_for(test_key, "TODO: add fast tests")
    test_full_cmd = cmd_for(test_key, test_fast_cmd)
    test_focused_cmd = (
        f"{test_fast_cmd} -- <pattern>"
        if test_fast_cmd.startswith(("pnpm ", "yarn ", "bun ", "npm "))
        else "TODO: add focused test command"
    )

    return {
        "install_cmd": install_cmd,
        "dev_cmd": dev_cmd,
        "format_cmd": format_cmd,
        "lint_cmd": lint_cmd,
        "typecheck_cmd": typecheck_cmd,
        "build_cmd": build_cmd,
        "test_fast_cmd": test_fast_cmd,
        "test_full_cmd": test_full_cmd,
        "test_focused_cmd": test_focused_cmd,
    }


def suggest_project_structure(scan: RepoScan, max_dirs: int = 12) -> str:
    if not scan.top_level_dirs:
        return "- `TODO`: add key directories and responsibilities"

    lines: list[str] = []
    for name in scan.top_level_dirs[:max_dirs]:
        lines.append(f"- `{name}/`: TODO: describe responsibility / ownership")
    if len(scan.top_level_dirs) > max_dirs:
        lines.append("- `...`: TODO: (keep this list short; link to docs for details)")
    return "\n".join(lines)


def load_template(skill_dir: Path, lang: str) -> str:
    template_name = "AGENTS.template.ja.md" if lang == "ja" else "AGENTS.template.en.md"
    path = skill_dir / "assets" / template_name
    text = _read_text(path, max_bytes=256_000)
    if not text:
        raise RuntimeError(f"Failed to read template: {path}")
    return text


def render_agents_md(skill_dir: Path, scan: RepoScan, lang: str) -> str:
    commands = suggest_commands(scan)
    project_structure = suggest_project_structure(scan)
    style_tools = ", ".join(scan.style_hints) if scan.style_hints else "TODO: list formatter/linter/typecheck tools"

    tmpl = Template(load_template(skill_dir, lang))
    return tmpl.safe_substitute(
        **commands,
        project_structure=project_structure,
        style_tools=style_tools,
        reference_paths="TODO: add 2–3 paths to canonical examples (keep it short)",
        branch_naming="TODO: define branch naming (e.g., feat/*, fix/*)",
    ).rstrip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate AGENTS.md skeleton")
    parser.add_argument("--root", default=".", help="Repository root (default: .)")
    parser.add_argument("--lang", choices=["auto", "ja", "en"], default="auto")
    parser.add_argument("--output", default="AGENTS.md", help="Output path when --write is set")
    parser.add_argument("--write", action="store_true", help="Write file instead of printing to stdout")
    args = parser.parse_args()

    repo_root = Path(args.root).resolve()
    skill_dir = Path(__file__).resolve().parents[1]

    scan = scan_repo(repo_root)
    lang = detect_lang(repo_root) if args.lang == "auto" else args.lang

    content = render_agents_md(skill_dir, scan, lang)

    if args.write:
        out_path = (repo_root / args.output).resolve()
        out_path.write_text(content, encoding="utf-8")
        print(str(out_path))
        return 0

    sys.stdout.write(content)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

