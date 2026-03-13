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

gofmt -w "${go_files[@]}"
go run "$GOIMPORTS_PKG" -local "$GO_LOCAL_PREFIX" -w "${go_files[@]}"
