#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-HEAD}"

if ! git_root="$(git rev-parse --show-toplevel 2>/dev/null)"; then
  echo "Error: not inside a git repository." >&2
  exit 2
fi

cd "$git_root"

source scripts/go/tool-versions.sh

files=()

while IFS= read -r -d '' file; do
  files+=("$file")
done < <(git diff --name-only -z --diff-filter=ACMR "$base_ref" --)

while IFS= read -r -d '' file; do
  files+=("$file")
done < <(git ls-files --others --exclude-standard -z)

if [[ "${#files[@]}" -eq 0 ]]; then
  echo "No changed/untracked files detected."
  exit 0
fi

# Deduplicate while preserving order.
unique_files=()
for file in "${files[@]}"; do
  already_added=0
  if [[ "${#unique_files[@]}" -gt 0 ]]; then
    for unique_file in "${unique_files[@]}"; do
      if [[ "$unique_file" == "$file" ]]; then
        already_added=1
        break
      fi
    done
  fi
  if [[ "$already_added" -eq 0 ]]; then
    unique_files+=("$file")
  fi
done

files=("${unique_files[@]}")

eslint_files=()
prettier_files=()
go_files=()
migration_files=()

auto_exclude() {
  local p="$1"
  # Generated API client code is never hand-edited.
  [[ "$p" == packages/frontend/api/src/generated/* ]] && return 0
  [[ "$p" == packages/typespec/openapi/* ]] && return 0
  [[ "$p" == packages/backend/internal/generated/* ]] && return 0
  return 1
}

for file in "${files[@]}"; do
  if auto_exclude "$file"; then
    continue
  fi
  case "$file" in
    *.ts | *.tsx | *.js | *.jsx | *.cjs | *.mjs)
      eslint_files+=("$file")
      prettier_files+=("$file")
      ;;
    *.json | *.md | *.yaml | *.yml)
      prettier_files+=("$file")
      ;;
    *.go)
      go_files+=("$file")
      ;;
    packages/backend/db/migrations/*.sql)
      migration_files+=("$file")
      ;;
   esac
done

if [[ "${#prettier_files[@]}" -gt 0 ]]; then
  echo "Prettier (check): ${#prettier_files[@]} file(s)"
  if [[ -x "./node_modules/.bin/prettier" ]]; then
    ./node_modules/.bin/prettier --check "${prettier_files[@]}"
  else
    echo "Error: prettier not found at ./node_modules/.bin/prettier (run install first)." >&2
    exit 3
  fi
else
  echo "Prettier: no applicable files."
fi

if [[ "${#eslint_files[@]}" -gt 0 ]]; then
  echo "ESLint: ${#eslint_files[@]} file(s)"
  if [[ -x "./node_modules/.bin/eslint" ]]; then
    ./node_modules/.bin/eslint --no-inline-config --no-warn-ignored --max-warnings 0 "${eslint_files[@]}"
  else
    echo "Error: eslint not found at ./node_modules/.bin/eslint (run install first)." >&2
    exit 3
  fi
else
  echo "ESLint: no applicable files."
fi

if [[ "${#go_files[@]}" -gt 0 ]]; then
  echo "gofmt/goimports (check): ${#go_files[@]} file(s)"
  if command -v gofmt >/dev/null 2>&1; then
    gofmt_output="$(gofmt -l "${go_files[@]}")"
    goimports_output="$(go run "$GOIMPORTS_PKG" -local "$GO_LOCAL_PREFIX" -l "${go_files[@]}")"
    if [[ -n "$gofmt_output" || -n "$goimports_output" ]]; then
      [[ -z "$gofmt_output" ]] || printf 'gofmt needs changes:\n%s\n' "$gofmt_output"
      [[ -z "$goimports_output" ]] || printf 'goimports needs changes:\n%s\n' "$goimports_output"
      exit 1
    fi
  else
    echo "Error: gofmt not found." >&2
    exit 3
  fi
else
  echo "gofmt/goimports: no applicable files."
fi

if [[ "${#migration_files[@]}" -gt 0 ]]; then
  echo "Migration guardrails: ${#migration_files[@]} file(s)"
  bash scripts/hooks/verify-staged-migrations.sh "${migration_files[@]}"
else
  echo "Migration guardrails: no applicable files."
fi
