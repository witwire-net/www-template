#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

command_name="${1:-}"
shift || true

if [ -z "$command_name" ]; then
  printf 'usage: %s <create|up|down> [args...]\n' "$0" >&2
  exit 1
fi

case "$command_name" in
  create)
    migration_name="${1:-create_users}"
    go run "$MIGRATE_PKG" create -ext sql -dir packages/backend/db/migrations -seq "$migration_name"
    ;;
  up|down)
    if [ -z "${DATABASE_URL:-}" ]; then
      printf 'DATABASE_URL is required for migrate %s\n' "$command_name" >&2
      exit 1
    fi

    go run -tags 'postgres' "$MIGRATE_PKG" -path packages/backend/db/migrations -database "$DATABASE_URL" "$command_name" "$@"
    ;;
  *)
    printf 'unsupported migrate command: %s\n' "$command_name" >&2
    exit 1
    ;;
esac
