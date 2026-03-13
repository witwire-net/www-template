#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

mapfile -t go_files < <(git ls-files --cached --modified --others --exclude-standard -- '*.go')

existing_go_files=()
for go_file in "${go_files[@]}"; do
  if [ -f "$go_file" ]; then
    existing_go_files+=("$go_file")
  fi
done

go_files=("${existing_go_files[@]}")

if [ "${#go_files[@]}" -eq 0 ]; then
  exit 0
fi

gofmt_output=$(gofmt -l "${go_files[@]}")
goimports_output=$(go run "$GOIMPORTS_PKG" -local "$GO_LOCAL_PREFIX" -l "${go_files[@]}")

if [ -n "$gofmt_output" ] || [ -n "$goimports_output" ]; then
  [ -z "$gofmt_output" ] || printf 'gofmt needs changes:\n%s\n' "$gofmt_output"
  [ -z "$goimports_output" ] || printf 'goimports needs changes:\n%s\n' "$goimports_output"
  exit 1
fi
