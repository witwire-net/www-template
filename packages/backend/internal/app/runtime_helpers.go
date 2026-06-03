package app

import (
	"context"
	"time"
)

// defaultReadHeaderTimeout は Product / Admin の両 HTTP server で共通する ReadHeaderTimeout の既定値である。
//
// 役割:
//   - Slowloris 攻撃等への防御として、HTTP リクエストヘッダー読み取りの上限時間を設定する。
//   - Product runtime と Admin runtime の両方で同一値を使うため、product_container.go ではなくこの中立ファイルに配置する。
//
// 使用例:
//
//	server := &http.Server{
//		ReadHeaderTimeout: defaultReadHeaderTimeout,
//	}
const defaultReadHeaderTimeout = 5 * time.Second

// composeClosers は複数の close 関数を単一の close 関数に合成する。
//
// 役割:
//   - Product container と Admin container の両方で resource 解放をまとめるために使う。
//   - Product container だけに配置すると Admin container から参照する際にファイル構造の対称性が崩れるため、この中立ファイルに配置する。
//
// 振る舞い:
//   - nil の close 関数はスキップする。
//   - 最初のエラーが発生した時点でそのエラーを返す（fail-fast）。
//   - 全 close 関数を順に実行し、エラーがなければ nil を返す。
//
// 引数:
//   - closers: 合成する close 関数の可変長引数。
//
// 戻り値:
//   - func(context.Context) error: 合成された close 関数。
//
// 使用例:
//
//	combinedClose := composeClosers(valkeyClose, dbClose)
//	defer combinedClose(ctx)
func composeClosers(closers ...func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, closeFn := range closers {
			if closeFn == nil {
				continue
			}
			if err := closeFn(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}
