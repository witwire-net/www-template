#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

if [ "$#" -eq 0 ]; then
  exit 0
fi

existing_go_files=()
for go_file in "$@"; do
  if [ -f "$go_file" ]; then
    existing_go_files+=("$go_file")
  fi
done

if [ "${#existing_go_files[@]}" -eq 0 ]; then
  exit 0
fi

gofmt -w "${existing_go_files[@]}"
go run "$GOIMPORTS_PKG" -local "$GO_LOCAL_PREFIX" -w "${existing_go_files[@]}"
