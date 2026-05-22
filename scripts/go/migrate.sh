#!/usr/bin/env bash

set -euo pipefail

source scripts/go/tool-versions.sh

readonly PRODUCT_MIGRATIONS_DIR='packages/backend/db/migrations'
readonly ADMIN_MIGRATIONS_DIR='packages/admin/db/migrations'

main() {
  # 関数定義をすべて読み込んだ後で command dispatch し、shell の逐次実行順による未定義呼び出しを防ぐ。
  local command_name="${1:-}"
  shift || true

  if [ -z "$command_name" ]; then
    printf 'usage: %s <create|up|down> [args...]\n' "$0" >&2
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
    *)
      printf 'unsupported migrate command: %s\n' "$command_name" >&2
      exit 1
      ;;
  esac
}

create_migration() {
  # Product / Admin で migration directory が異なるため、作成先 target を必ず明示させる。
  local target="${1:-}"
  local migration_name="${2:-}"
  local migration_dir

  if [ -z "$target" ] || [ -z "$migration_name" ]; then
    printf 'usage: pnpm migrate:create <product|admin> <migration_name>\n' >&2
    exit 1
  fi

  migration_dir="$(migration_dir_for_target "$target")"
  go run "$MIGRATE_PKG" create -ext sql -dir "$migration_dir" -seq "$migration_name"
}

migrate_up() {
  # Product DB を先に更新し、admin_view/admin_op と最小権限 role を Admin runtime より先に用意する。
  local product_database_url
  local admin_database_url

  product_database_url="$(resolve_product_database_url)"
  run_migrate product "$product_database_url" up "$@"
  ensure_product_admin_login_role "$product_database_url"

  # Admin DB は Product DB と別 database なので、migration 実行前に database の存在だけを冪等に保証する。
  admin_database_url="$(resolve_admin_database_url)"
  ensure_admin_database_exists "$admin_database_url"
  run_migrate admin "$(migration_database_url "$admin_database_url")" up "$@"
}

migrate_down() {
  # down は up の逆順で実行し、Admin DB の依存を先に落としてから Product 側の admin_op/admin_view を戻す。
  local product_database_url
  local admin_database_url

  admin_database_url="$(resolve_admin_database_url)"
  run_migrate admin "$(migration_database_url "$admin_database_url")" down "$@"

  product_database_url="$(resolve_product_database_url)"
  revoke_product_admin_login_role "$product_database_url"
  run_migrate product "$product_database_url" down "$@"
}

migration_dir_for_target() {
  # target 名から migration directory を一意に決め、未知 target で誤った DB 用 SQL を作らない。
  local target="$1"

  case "$target" in
    product)
      printf '%s\n' "$PRODUCT_MIGRATIONS_DIR"
      ;;
    admin)
      printf '%s\n' "$ADMIN_MIGRATIONS_DIR"
      ;;
    *)
      printf 'unknown migration target: %s (expected product or admin)\n' "$target" >&2
      exit 1
      ;;
  esac
}

run_migrate() {
  # golang-migrate に target ごとの directory と接続先を渡し、Product/Admin とも同じ実行器で管理する。
  local target="$1"
  local database_url="$2"
  local direction="$3"
  shift 3

  printf 'Running %s migrations %s\n' "$target" "$direction" >&2
  go run -tags 'postgres' "$MIGRATE_PKG" \
    -path "$(migration_dir_for_target "$target")" \
    -database "$database_url" \
    "$direction" "$@"
}

resolve_product_database_url() {
  # DATABASE_URL が明示されている場合は release/CI の入力を優先し、未指定なら .config/local.toml を読む。
  if [ -n "${DATABASE_URL:-}" ]; then
    printf '%s\n' "$DATABASE_URL"
    return
  fi

  read_toml_value "$(resolve_product_config_path)" database url
}

resolve_admin_database_url() {
  # ADMIN_DATABASE_URL が明示されている場合は release/CI の入力を優先し、未指定なら *.admin.toml を読む。
  if [ -n "${ADMIN_DATABASE_URL:-}" ]; then
    printf '%s\n' "$ADMIN_DATABASE_URL"
    return
  fi

  read_toml_value "$(resolve_admin_config_path)" database admin_url
}

resolve_admin_product_database_url() {
  # Admin runtime の Product 接続 role を作るため、Admin TOML の database.product_url を読む。
  if [ -n "${PRODUCT_DATABASE_URL:-}" ]; then
    printf '%s\n' "$PRODUCT_DATABASE_URL"
    return
  fi

  read_toml_value "$(resolve_admin_config_path)" database product_url
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

ensure_admin_database_exists() {
  # Admin DB 自体は database 外の操作なので、maintenance DB へ接続して存在しない場合だけ作成する。
  local admin_database_url="$1"
  local maintenance_url
  local database_name

  maintenance_url="${ADMIN_MAINTENANCE_DATABASE_URL:-$(maintenance_database_url "$admin_database_url")}"
  database_name="$(database_name_from_url "$admin_database_url")"

  psql "$maintenance_url" -X -v ON_ERROR_STOP=1 -v target_db="$database_name" <<'SQL'
SELECT 'CREATE DATABASE ' || quote_ident(:'target_db')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_database WHERE datname = :'target_db'
)
\gexec
SQL
}

ensure_product_admin_login_role() {
  # Admin runtime が Product DB の admin_view/admin_op を使えるよう、Admin TOML の product_url にある login role を整える。
  local product_owner_url="$1"
  local admin_product_url
  local role_name
  local role_password

  admin_product_url="$(resolve_admin_product_database_url)"
  role_name="$(url_component username "$admin_product_url")"
  role_password="$(url_component password "$admin_product_url")"

  if [ -z "$role_name" ]; then
    printf 'database.product_url must include a username for the Admin Product login role\n' >&2
    exit 1
  fi

  psql "$product_owner_url" -X -v ON_ERROR_STOP=1 -v role_name="$role_name" -v role_password="$role_password" <<'SQL'
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

revoke_product_admin_login_role() {
  # Product 側の admin role down migration が role drop で詰まらないよう、環境別 login role の継承だけ先に外す。
  local product_owner_url="$1"
  local admin_product_url
  local role_name

  admin_product_url="$(resolve_admin_product_database_url)"
  role_name="$(url_component username "$admin_product_url")"

  if [ -z "$role_name" ]; then
    return
  fi

  psql "$product_owner_url" -X -v ON_ERROR_STOP=1 -v role_name="$role_name" <<'SQL'
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
