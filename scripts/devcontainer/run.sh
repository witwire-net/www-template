#!/usr/bin/env bash

set -euo pipefail

# Step 1: 引数が空の場合は実行対象が不明なため、利用者とエージェントに正しい呼び出し形を明示して終了する。
if [ "$#" -eq 0 ]; then
  printf '%s\n' 'usage: scripts/devcontainer/run.sh <command> [args...]' >&2
  exit 64
fi

# Step 2: script 自身の場所から repository root を解決し、どの作業ディレクトリから呼ばれても同じ Compose 定義を参照する。
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git -C "${script_dir}/../.." rev-parse --show-toplevel)"

# Step 3: DevContainer 内の workspace path と Compose 定義を固定し、ホスト側の cwd や環境差分で検証場所がぶれないようにする。
compose_file="${repo_root}/.devcontainer/compose.yaml"
container_workspace="/workspaces/www-template"

# Step 4: すでに DevContainer 内で実行されている場合は、再度 docker compose exec せず、その場の container toolchain で対象コマンドへ置き換える。
if [ -f /.dockerenv ]; then
  cd "${container_workspace}"
  exec "$@"
fi

# Step 5: ホスト側に Docker CLI がない場合は DevContainer へ入れないため、原因を明確にして失敗させる。
if ! command -v docker >/dev/null 2>&1; then
  printf '%s\n' 'docker command is required to run commands inside the DevContainer.' >&2
  exit 127
fi

# Step 6: Compose 定義が読めない場合は repository 構成の破損として扱い、誤った compose project を起動しない。
if [ ! -r "${compose_file}" ]; then
  printf 'DevContainer compose file is not readable: %s\n' "${compose_file}" >&2
  exit 66
fi

# Step 7: Docker daemon に接続できない場合は、sandbox や Docker Desktop 未起動などの環境問題として明確に失敗させる。
if ! docker info >/dev/null 2>&1; then
  printf '%s\n' 'Docker daemon is not reachable from this shell.' >&2
  printf '%s\n' 'Start Docker Desktop and allow Docker access for this command, then retry.' >&2
  exit 69
fi

# Step 8: workspace service の container ID を取得し、DevContainer が起動済みかどうかを安全に判定する。
workspace_container_id="$(docker compose -f "${compose_file}" ps -q workspace)"
if [ -z "${workspace_container_id}" ]; then
  printf '%s\n' 'DevContainer workspace service is not created.' >&2
  printf '%s\n' 'Open the repository in Zed Dev Container, or run: docker compose -f .devcontainer/compose.yaml up -d workspace' >&2
  exit 69
fi

# Step 9: 停止済み container へ exec して曖昧に失敗しないよう、Docker の running state を事前に確認する。
workspace_running="$(docker inspect -f '{{.State.Running}}' "${workspace_container_id}" 2>/dev/null || true)"
if [ "${workspace_running}" != "true" ]; then
  printf '%s\n' 'DevContainer workspace service is not running.' >&2
  printf '%s\n' 'Open the repository in Zed Dev Container, or run: docker compose -f .devcontainer/compose.yaml up -d workspace' >&2
  exit 69
fi

# Step 10: 対象コマンドを DevContainer の repository root で実行し、ホストの Node/Go/bash ではなく container toolchain を必ず使う。
exec docker compose \
  -f "${compose_file}" \
  exec \
  -T \
  -w "${container_workspace}" \
  workspace \
  "$@"
