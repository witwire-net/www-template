#!/usr/bin/env bash

set -euo pipefail

if [ ! -f packages/backend/go.mod ]; then
  printf 'packages/backend/go.mod is required\n' >&2
  exit 1
fi

if [ ! -f packages/backend/go.sum ]; then
  printf 'packages/backend/go.sum is required\n' >&2
  exit 1
fi
