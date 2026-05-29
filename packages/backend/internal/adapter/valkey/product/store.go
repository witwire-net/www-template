package product

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"www-template/packages/backend/internal/platform/config"
)

// Store は Product 認証永続化だけが利用する Valkey 接続 adapter である。
//
// 役割:
//   - Product AccountAuth の refresh/session state を `product:*` key namespace に閉じ込める。
//   - Admin operator auth の logical namespace と混ざらないよう、Product package 内の store として接続と key 生成を所有する。
//   - application port 実装が Redis client へ直接依存しないよう、この package の非公開 helper 経由で操作させる。
//
// 引数:
//   - NewStore の cfg: Valkey URL と任意の環境別 key prefix を含むインフラ設定。
//
// 戻り値:
//   - *Store: Product namespace を付与する Valkey store。
//   - error: URL が空、または Redis URL として解釈できない場合の初期化 error。
//
// 使用例:
//
//	store, err := product.NewStore(cfg.Infra.Valkey)
//	if err != nil {
//		return err
//	}
type Store struct {
	client    *redis.Client
	keyPrefix string
}

var errKeyNotFound = redis.Nil

// NewStore は Product AccountAuth 用の Valkey store を生成する。
func NewStore(cfg config.ValkeyConfig) (*Store, error) {
	// Step 1: URL が空のまま起動すると認証 state を保存できず fail-open につながるため拒否する。
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("VALKEY_URL is required")
	}

	// Step 2: Redis 互換 URL を標準 parser で解釈し、接続先 DB や認証情報を client option に変換する。
	options, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	// Step 3: 認証処理の外部 I/O が長時間固着しないよう、product store 自体が短い timeout を設定する。
	options.DialTimeout = 3 * time.Second
	options.ReadTimeout = 3 * time.Second
	options.WriteTimeout = 3 * time.Second

	// Step 4: 環境別 prefix は保持し、実際の key 生成時に product namespace を必ず挿入する。
	return &Store{client: redis.NewClient(options), keyPrefix: strings.TrimSpace(cfg.KeyPrefix)}, nil
}

// Close は Product Valkey client を閉じる。
func (s *Store) Close() error {
	// Step 1: nil receiver や未初期化 client は、終了処理の冪等性を保つため成功扱いにする。
	if s == nil || s.client == nil {
		return nil
	}

	// Step 2: Redis client の接続資源を解放する。
	return s.client.Close()
}

// Ping は Product Valkey store の疎通を確認する。
func (s *Store) Ping(ctx context.Context) error {
	// Step 1: 未初期化 store を明示的な error にし、runtime composition が fail-close できるようにする。
	if s == nil || s.client == nil {
		return errors.New("product valkey store is required")
	}

	// Step 2: 呼び出し元 context に従って PING を実行し、接続失敗をそのまま返す。
	return s.client.Ping(ctx).Err()
}

func (s *Store) key(parts ...string) string {
	// Step 1: 環境別 prefix、Product surface 名、機能別 segment の順に連結し、Admin key と衝突しない prefix を作る。
	segments := make([]string, 0, len(parts)+2)
	if s.keyPrefix != "" {
		segments = append(segments, s.keyPrefix)
	}
	segments = append(segments, "product")
	segments = append(segments, parts...)

	// Step 2: Valkey key として扱いやすい colon 区切りへ変換する。
	return strings.Join(segments, ":")
}
