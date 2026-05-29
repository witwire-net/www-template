#!/usr/bin/env bash

set -euo pipefail

# Go 生成器の固定バージョンを読み込み、Product と Admin の両 binding が同じ oapi-codegen 版で再現されるようにする。
source scripts/go/tool-versions.sh

# Go module の依存解決が readonly で成立することを先に確認し、生成処理が go.mod/go.sum を暗黙に変更しないようにする。
bash scripts/go/verify-module.sh

# oapi-codegen の設定ファイルと出力先は backend package からの相対パスで定義されているため、backend root を作業基準にする。
cd packages/backend

# 生成先 directory を明示的に作成し、初回生成時も Product/Admin の物理分離された出力先へ安全に書き込めるようにする。
mkdir -p internal/generated/openapi internal/generated/adminopenapi

# Product OpenAPI から Product runtime 専用の Go bindings を生成し、既存 consumer が参照する openapi package に限定して出力する。
go run -mod=readonly "$OAPI_CODEGEN_PKG" --config oapi-codegen.yaml ../typespec/openapi/openapi.json < /dev/null

# Admin OpenAPI から Admin runtime 専用の Go bindings を生成し、Product package と物理的に分かれた adminopenapi package に出力する。
go run -mod=readonly "$OAPI_CODEGEN_PKG" --config oapi-codegen.admin.yaml ../typespec/openapi/admin.openapi.json < /dev/null
