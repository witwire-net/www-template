#!/usr/bin/env bash

set -euo pipefail

# Product/Admin の生成物をすべて同じ drift snapshot に含め、片方だけ再生成漏れになる状態を検出する。
generated_files=(
  "packages/typespec/openapi/openapi.json"
  "packages/typespec/openapi/admin.openapi.json"
  "packages/frontend/api/src/generated/client.ts"
  "packages/admin/api/src/generated/client.ts"
  "packages/backend/internal/generated/openapi/openapi.gen.go"
  "packages/backend/internal/generated/adminopenapi/openapi.gen.go"
)

# Product surface に現れてはならない Admin operation/tag/export の代表パターンを定義する。
admin_openapi_contamination='"operationId": "[^"]*Admin|"name": "admin-(accounts|auth)"|"tags": \["admin-(accounts|auth)"\]'
admin_typescript_export_contamination='^export (type|interface|const) (Admin[A-Za-z0-9_]*|(listAdminAccounts|getAdminAccount|createAdminAccount|finishAdminOperatorSetup|startAdminOperatorSetup|getCurrentAdminOperator|logoutAdminOperator|refreshAdminOperatorSession|finishAdminPasskeyAuthentication|startAdminPasskeyAuthentication)(Response(Success|Error|[0-9]{3})?|Params)?)\b|^export const get(ListAdminAccounts|GetAdminAccount|CreateAdminAccount|FinishAdminOperatorSetup|StartAdminOperatorSetup|GetCurrentAdminOperator|LogoutAdminOperator|RefreshAdminOperatorSession|FinishAdminPasskeyAuthentication|StartAdminPasskeyAuthentication)Url\b'
admin_go_export_contamination='^(type |func |[[:space:]]+)(Admin[A-Za-z0-9_]*|ListAdminAccounts|GetAdminAccount|CreateAdminAccount|FinishAdminOperatorSetup|StartAdminOperatorSetup|GetCurrentAdminOperator|LogoutAdminOperator|RefreshAdminOperatorSession|FinishAdminPasskeyAuthentication|StartAdminPasskeyAuthentication)\b'

# Admin surface に現れてはならない Product operation/tag/export の代表パターンを定義する。
product_openapi_contamination='"operationId": "(getAccountSettings|updateAccountSettings|logout|finishPasskeyAuthentication|registerPasskey|startPasskeyRegistration|startPasskeyAuthentication|finishReauthentication|startReauthentication|requestPasskeyRecovery|consumeRecoveryToken|refreshToken|listPasskeys|finishPasskeyAddition|sendDeviceLink|startPasskeyAddition|deletePasskey|listSessions|revokeOtherSessions|revokeSession|getStatus)"|"name": "(account-settings|app-auth|auth|status)"|"tags": \["(account-settings|app-auth|auth|status)"\]'
product_typescript_export_contamination='^export (type|interface|const) (getAccountSettings|updateAccountSettings|logout|finishPasskeyAuthentication|registerPasskey|startPasskeyRegistration|startPasskeyAuthentication|finishReauthentication|startReauthentication|requestPasskeyRecovery|consumeRecoveryToken|refreshToken|listPasskeys|finishPasskeyAddition|sendDeviceLink|startPasskeyAddition|deletePasskey|listSessions|revokeOtherSessions|revokeSession|getStatus)(Response(Success|Error|[0-9]{3})?|Params)?\b|^export const get(GetAccountSettings|UpdateAccountSettings|Logout|FinishPasskeyAuthentication|RegisterPasskey|StartPasskeyRegistration|StartPasskeyAuthentication|FinishReauthentication|StartReauthentication|RequestPasskeyRecovery|ConsumeRecoveryToken|RefreshToken|ListPasskeys|FinishPasskeyAddition|SendDeviceLink|StartPasskeyAddition|DeletePasskey|ListSessions|RevokeOtherSessions|RevokeSession|GetStatus)Url\b'
product_go_export_contamination='^(type |func |[[:space:]]+)(GetAccountSettings|UpdateAccountSettings|Logout|FinishPasskeyAuthentication|RegisterPasskey|StartPasskeyRegistration|StartPasskeyAuthentication|FinishReauthentication|StartReauthentication|RequestPasskeyRecovery|ConsumeRecoveryToken|RefreshToken|ListPasskeys|FinishPasskeyAddition|SendDeviceLink|StartPasskeyAddition|DeletePasskey|ListSessions|RevokeOtherSessions|RevokeSession|GetStatus)\b'

# 指定した生成物に禁止パターンが含まれる場合、該当行を表示して surface contamination として失敗させる。
check_no_contamination() {
  local file_path=$1
  local pattern=$2
  local description=$3

  if grep -En "$pattern" "$file_path"; then
    printf 'codegen contamination detected in %s: %s\n' "$file_path" "$description" >&2
    return 1
  fi
}

snapshot_dir=$(mktemp -d)
trap 'rm -rf "$snapshot_dir"' EXIT

# 生成前の成果物を一時領域へ退避し、後続の pnpm gen が再現した内容と byte 単位で比較できるようにする。
for file_path in "${generated_files[@]}"; do
  if [ -f "$file_path" ]; then
    mkdir -p "$snapshot_dir/$(dirname "$file_path")"
    cp "$file_path" "$snapshot_dir/$file_path"
  fi
done

# repository 既定の生成入口を使い、OpenAPI、TypeScript SDK、Go bindings、Prisma 生成物を一括で再生成する。
pnpm gen

# 生成後の各成果物を snapshot と比較し、未生成・差分ありのどちらも drift として扱う。
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

# Product artifacts に Admin operation/tag/export が混入していないことを検査する。
check_no_contamination "packages/typespec/openapi/openapi.json" "$admin_openapi_contamination" "Admin operation or tag in Product OpenAPI"
check_no_contamination "packages/frontend/api/src/generated/client.ts" "$admin_typescript_export_contamination" "Admin export in Product SDK"
check_no_contamination "packages/backend/internal/generated/openapi/openapi.gen.go" "$admin_go_export_contamination" "Admin export in Product Go bindings"

# Admin artifacts に Product operation/tag/export が混入していないことを検査する。
check_no_contamination "packages/typespec/openapi/admin.openapi.json" "$product_openapi_contamination" "Product operation or tag in Admin OpenAPI"
check_no_contamination "packages/admin/api/src/generated/client.ts" "$product_typescript_export_contamination" "Product export in Admin SDK"
check_no_contamination "packages/backend/internal/generated/adminopenapi/openapi.gen.go" "$product_go_export_contamination" "Product export in Admin Go bindings"
