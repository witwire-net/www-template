#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

readonly DATABASE_MIGRATIONS_DIR='packages/backend/db/migrations'

main() {
  # 関数定義をすべて読み込んだ後で command dispatch し、shell の逐次実行順による未定義呼び出しを防ぐ。
  local command_name="${1:-}"
  shift || true

  if [ -z "$command_name" ]; then
    printf 'usage: %s <create|up|down|force> [args...]\n' "$0" >&2
    exit 1
  fi

  case "$command_name" in
    create)
      create_migration "$@"
      ;;
    up)
      migrate_up "$@"
      ;;
    down)
      migrate_down "$@"
      ;;
    force)
      migrate_force "$@"
      ;;
    *)
      printf 'unsupported migrate command: %s\n' "$command_name" >&2
      exit 1
      ;;
  esac
}

create_migration() {
  # DB schema は backend migration に集約し、作成先 target を利用者に選ばせない。
  local migration_name="${1:-}"

  if [ -z "$migration_name" ]; then
    printf 'usage: pnpm migrate:create <migration_name>\n' >&2
    exit 1
  fi

  go run "$MIGRATE_PKG" create -ext sql -dir "$DATABASE_MIGRATIONS_DIR" -seq "$migration_name"
}

migrate_up() {
  # DB schema は単一の backend migration として管理し、Admin 専用 DB migration は存在させない。
  local database_url

  database_url="$(resolve_database_url)"
  run_migrate "$database_url" up "$@"
  ensure_admin_login_role "$database_url"
}

migrate_down() {
  # down は Admin runtime login role の継承を先に外してから、backend migration を戻す。
  local database_url

  database_url="$(resolve_database_url)"
  revoke_admin_login_role "$database_url"
  run_migrate "$database_url" down "$@"
}

migrate_force() {
  # 失敗した migration の dirty flag を、確認済みの直前 version へ戻すためだけに使う。
  local target_version="${1:-}"
  local database_url

  if [ -z "$target_version" ]; then
    printf 'usage: pnpm migrate:force <version>\n' >&2
    exit 1
  fi

  database_url="$(resolve_database_url)"
  run_migrate "$database_url" force "$target_version"
}

run_migrate() {
  # golang-migrate に backend migration directory と接続先を渡し、DB schema 変更を一箇所で管理する。
  local database_url="$1"
  local direction="$2"
  shift 2

  printf 'Running database migrations %s\n' "$direction" >&2
  go run -tags 'postgres' "$MIGRATE_PKG" \
    -path "$DATABASE_MIGRATIONS_DIR" \
    -database "$database_url" \
    "$direction" "$@"
}

resolve_database_url() {
  # DATABASE_URL が明示されている場合は release/CI の入力を優先し、未指定なら .config/local.toml を読む。
  if [ -n "${DATABASE_URL:-}" ]; then
    printf '%s\n' "$DATABASE_URL"
    return
  fi

  read_toml_value "$(resolve_product_config_path)" database url
}

resolve_admin_database_url() {
  # Admin runtime の最小権限 login role を作るため、Admin TOML の database.url を読む。
  read_toml_value "$(resolve_admin_config_path)" database url
}

resolve_product_config_path() {
  # backend と同じ CONFIG_PATH 規約を使い、なければ repository root の local.toml を既定にする。
  if [ -n "${CONFIG_PATH:-}" ]; then
    require_file "$CONFIG_PATH"
    printf '%s\n' "$CONFIG_PATH"
    return
  fi

  require_file '.config/local.toml'
  printf '%s\n' '.config/local.toml'
}

resolve_admin_config_path() {
  # Admin は ADMIN_CONFIG_PATH を優先し、なければ repository root の local.admin.toml を既定にする。
  if [ -n "${ADMIN_CONFIG_PATH:-}" ]; then
    require_file "$ADMIN_CONFIG_PATH"
    printf '%s\n' "$ADMIN_CONFIG_PATH"
    return
  fi

  require_file '.config/local.admin.toml'
  printf '%s\n' '.config/local.admin.toml'
}

require_file() {
  # 設定ファイルの typos を fallback で隠さず、その場で明示的に止める。
  local file_path="$1"

  if [ ! -f "$file_path" ]; then
    printf 'required file not found: %s\n' "$file_path" >&2
    exit 1
  fi
}

ensure_admin_login_role() {
  # Admin runtime が最小権限 role で DB に接続できるよう、Admin TOML の database.url にある login role を整える。
  local owner_url="$1"
  local admin_database_url
  local role_name
  local role_password

  admin_database_url="$(resolve_admin_database_url)"
  role_name="$(url_component username "$admin_database_url")"
  role_password="$(url_component password "$admin_database_url")"

  if [ -z "$role_name" ]; then
    printf 'database.url in Admin config must include a username for the Admin login role\n' >&2
    exit 1
  fi

  psql "$owner_url" -X -v ON_ERROR_STOP=1 -v role_name="$role_name" -v role_password="$role_password" <<'SQL'
SELECT CASE
  WHEN :'role_password' = '' THEN format('CREATE ROLE %I LOGIN', :'role_name')
  ELSE format('CREATE ROLE %I LOGIN PASSWORD %L', :'role_name', :'role_password')
END
WHERE NOT EXISTS (
  SELECT 1 FROM pg_roles WHERE rolname = :'role_name'
)
\gexec

GRANT admin_console_write TO :"role_name";
SQL
}

revoke_admin_login_role() {
  # admin role down migration が role drop で詰まらないよう、環境別 login role の継承だけ先に外す。
  local owner_url="$1"
  local admin_database_url
  local role_name

  admin_database_url="$(resolve_admin_database_url)"
  role_name="$(url_component username "$admin_database_url")"

  if [ -z "$role_name" ]; then
    return
  fi

  psql "$owner_url" -X -v ON_ERROR_STOP=1 -v role_name="$role_name" <<'SQL'
SELECT format('REVOKE admin_console_write FROM %I', :'role_name')
WHERE EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admin_console_write')
  AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'role_name')
\gexec
SQL
}

read_toml_value() {
  # repository の設定ファイルで使っている top-level section + scalar string だけを Node で厳密に読む。
  local file_path="$1"
  local section="$2"
  local key="$3"

  node --input-type=module - "$file_path" "$section" "$key" <<'NODE'
import { readFileSync } from 'node:fs';

const [filePath, sectionName, keyName] = process.argv.slice(2);
const source = readFileSync(filePath, 'utf8');
let currentSection = '';

for (const [index, rawLine] of source.split(/\r?\n/u).entries()) {
  const line = rawLine.trim();
  if (line === '' || line.startsWith('#')) continue;

  const sectionMatch = /^\[([-_a-zA-Z0-9]+)\]$/u.exec(line);
  if (sectionMatch !== null) {
    currentSection = sectionMatch[1];
    continue;
  }

  if (currentSection !== sectionName) continue;

  const separatorIndex = line.indexOf('=');
  if (separatorIndex === -1) continue;

  const key = line.slice(0, separatorIndex).trim();
  if (key !== keyName) continue;

  const value = line.slice(separatorIndex + 1).trim();
  if (!value.startsWith('"') || !value.endsWith('"')) {
    throw new Error(`${filePath}:${String(index + 1)} ${sectionName}.${keyName} must be a quoted string`);
  }
  process.stdout.write(JSON.parse(value));
  process.exit(0);
}

throw new Error(`Missing TOML value: ${sectionName}.${keyName} in ${filePath}`);
NODE
}

migration_database_url() {
  # Prisma の schema query は lib/pq/golang-migrate では使わないため、migration 実行 URL から取り除く。
  url_component migration_url "$1"
}

maintenance_database_url() {
  # CREATE DATABASE は接続中 DB には実行できないため、同じ credential で postgres DB へつなぎ直す。
  url_component maintenance_url "$1"
}

database_name_from_url() {
  # CREATE DATABASE 対象名は URL の path から取り、query string に依存しない。
  url_component database "$1"
}

url_component() {
  # URL の操作は shell 文字列処理で行わず、標準 URL parser に任せて credential や query を安全に扱う。
  local field="$1"
  local raw_url="$2"

  node --input-type=module - "$field" "$raw_url" <<'NODE'
const [field, rawUrl] = process.argv.slice(2);
const url = new URL(rawUrl);

switch (field) {
  case 'database': {
    const database = decodeURIComponent(url.pathname.replace(/^\//u, ''));
    if (database === '') throw new Error('database URL must include a database name');
    process.stdout.write(database);
    break;
  }
  case 'maintenance_url': {
    url.pathname = '/postgres';
    url.searchParams.delete('schema');
    process.stdout.write(url.toString());
    break;
  }
  case 'migration_url': {
    url.searchParams.delete('schema');
    process.stdout.write(url.toString());
    break;
  }
  case 'username': {
    process.stdout.write(decodeURIComponent(url.username));
    break;
  }
  case 'password': {
    process.stdout.write(decodeURIComponent(url.password));
    break;
  }
  default:
    throw new Error(`unknown URL component: ${field}`);
}
NODE
}

main "$@"
