package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	sharedvalkey "www-template/packages/backend/internal/adapter/valkey"
	"www-template/packages/backend/internal/platform/config"
)

// Store は Admin operator auth だけが利用する Valkey 接続 adapter である。
//
// 役割:
//   - Admin operator refresh/session state を `admin:*` key namespace に閉じ込める。
//   - Product account auth の Valkey key と logical namespace を共有しない構造を package path と key prefix の両方で固定する。
//   - Redis client を Admin Valkey package 内に隠蔽し、application port 実装だけを公開する。
//
// 引数:
//   - cfg: Admin Valkey URL。KeyPrefix は Product namespace 混入を避けるため使用しない。
//
// 戻り値:
//   - *Store: Admin namespace を付与する Valkey store。
//   - error: URL 不備または Redis URL parse 失敗。
type Store struct {
	client *redis.Client
}

var errKeyNotFound = redis.Nil

// NewStore は Admin operator auth 用の Valkey store を生成する。
func NewStore(cfg config.ValkeyConfig) (*Store, error) {
	// Step 1: Admin session state の保存先がない状態を fail-close にするため、空 URL を拒否する。
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("ADMIN_VALKEY_URL is required")
	}

	// Step 2: Redis 互換 URL を client options へ変換する。
	options, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	options.DialTimeout = 3 * time.Second
	options.ReadTimeout = 3 * time.Second
	options.WriteTimeout = 3 * time.Second

	// Step 3: Admin logical DB 内の key は必ず `admin:*` から始めるため、Product 用の任意 prefix は取り込まない。
	return &Store{client: sharedvalkey.NewObservedClient(options, "admin")}, nil
}

// Close は Admin Valkey client を閉じる。
func (s *Store) Close() error {
	// Step 1: 終了処理を冪等にするため、nil store は成功扱いにする。
	if s == nil || s.client == nil {
		return nil
	}

	// Step 2: Redis client の接続資源を解放する。
	return s.client.Close()
}

// Ping は Admin Valkey store の疎通を確認する。
func (s *Store) Ping(ctx context.Context) error {
	// Step 1: 未初期化 store は runtime composition が検知できる error にする。
	if s == nil || s.client == nil {
		return errors.New("admin valkey store is required")
	}

	// Step 2: Redis PING を実行し、接続失敗を呼び出し元へ返す。
	return s.client.Ping(ctx).Err()
}

// Key は Admin namespace 付きの Valkey key を返す。
func (s *Store) Key(parts ...string) string {
	// Step 1: WebAuthn provider など外部 adapter が Admin key prefix を手組みしないよう、store の key helper を公開する。
	return s.key(parts...)
}

// Set は Admin namespace の key に TTL 付き値を保存する。
func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	// Step 1: provider が組み立てた Admin key へ文字列 payload を保存し、Redis error は呼び出し側で fail-close できるよう返す。
	return s.client.Set(ctx, key, value, ttl).Err()
}

// GetDel は Admin namespace の key を取得しながら削除する。
func (s *Store) GetDel(ctx context.Context, key string) (string, error) {
	// Step 1: WebAuthn challenge session を一度だけ消費するため Redis GETDEL を使う。
	return s.client.GetDel(ctx, key).Result()
}

func (s *Store) key(parts ...string) string {
	// Step 1: Admin surface 名を必ず先頭に積み、Product key や環境 prefix と衝突しない固定 prefix にする。
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, "admin")
	segments = append(segments, parts...)

	// Step 2: Valkey key として扱う colon 区切りへ変換する。
	return strings.Join(segments, ":")
}
