#!/usr/bin/env bash

set -euo pipefail

generated_files=(
  "packages/typespec/openapi/openapi.json"
  "packages/frontend/api/src/generated/client.ts"
  "packages/backend/internal/generated/openapi/openapi.gen.go"
)

snapshot_dir=$(mktemp -d)
trap 'rm -rf "$snapshot_dir"' EXIT

for file_path in "${generated_files[@]}"; do
  if [ -f "$file_path" ]; then
    mkdir -p "$snapshot_dir/$(dirname "$file_path")"
    cp "$file_path" "$snapshot_dir/$file_path"
  fi
done

pnpm gen

drift_found=0
for file_path in "${generated_files[@]}"; do
  if [ ! -f "$snapshot_dir/$file_path" ] || ! cmp -s "$snapshot_dir/$file_path" "$file_path"; then
    drift_found=1
    git diff --no-index -- "$snapshot_dir/$file_path" "$file_path" || true
  fi
done

if [ "$drift_found" -ne 0 ]; then
  printf 'codegen drift detected; run pnpm gen and keep generated files updated\n' >&2
  exit 1
fi
