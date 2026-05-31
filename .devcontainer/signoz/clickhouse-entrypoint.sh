#!/usr/bin/env bash

# この script は、Docker Desktop / WSL 環境で file-level bind mount が stale path を保持して ClickHouse 起動に失敗する問題を避けるため、
# repo から read-only directory bind された SigNoz 設定を ClickHouse の標準設定配置へコピーしてから公式 entrypoint に処理を渡す。
set -euo pipefail

# Step 1: SigNoz 用の ClickHouse server 追加設定を config.d 配下へ配置する。
# 入力は read-only bind された /etc/signoz/clickhouse-signoz.xml、出力は ClickHouse が起動時に include する config.d/signoz.xml。
# 副作用として container filesystem 上の設定ファイルを毎回上書きし、host 側の repo file は変更しない。
install -D -m 0444 /etc/signoz/clickhouse-signoz.xml /etc/clickhouse-server/config.d/signoz.xml

# Step 2: SigNoz が利用する default user / profile 設定を users.d 配下へ配置する。
# 入力は read-only bind された /etc/signoz/clickhouse-users.xml、出力は ClickHouse が users.xml と合わせて読み込む users.d/signoz-users.xml。
# 副作用として container filesystem 上の users.d 設定を毎回上書きし、named volume の data には触れない。
install -D -m 0444 /etc/signoz/clickhouse-users.xml /etc/clickhouse-server/users.d/signoz-users.xml

# Step 3: histogramQuantile executable UDF の定義を ClickHouse が探索する server config directory へ配置する。
# 入力は read-only bind された /etc/signoz/custom-function.xml、出力は user_defined_executable_functions_config の pattern に一致する custom-function.xml。
# 副作用として container filesystem 上の UDF 定義だけを更新し、実行 binary は signoz-init-clickhouse が named volume へ配置したものを使う。
install -D -m 0444 /etc/signoz/custom-function.xml /etc/clickhouse-server/custom-function.xml

# Step 4: 公式 image の entrypoint に制御を移す。
# 引数は Docker / Compose が渡した値をそのまま維持し、PID 1 を公式 entrypoint に置き換えて signal handling を壊さない。
exec /entrypoint.sh "$@"
