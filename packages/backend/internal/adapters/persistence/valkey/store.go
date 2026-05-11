package valkey

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"www-template/packages/backend/internal/platform/config"
)

const defaultInfrastructureTimeout = 3 * time.Second

var errRESPNil = redis.Nil

type ValkeyStore struct {
	client    *redis.Client
	keyPrefix string
}

func NewStore(cfg config.ValkeyConfig) (*ValkeyStore, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("VALKEY_URL is required")
	}

	options, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	options.DialTimeout = 3 * time.Second
	options.ReadTimeout = 3 * time.Second
	options.WriteTimeout = 3 * time.Second

	return &ValkeyStore{
		client:    redis.NewClient(options),
		keyPrefix: strings.TrimSpace(cfg.KeyPrefix),
	}, nil
}

func (s *ValkeyStore) Key(parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	if s.keyPrefix != "" {
		segments = append(segments, s.keyPrefix)
	}
	segments = append(segments, parts...)
	return strings.Join(segments, ":")
}

func (s *ValkeyStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *ValkeyStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

// GetDel は指定したキーに対して GETDEL コマンドを発行し、値を取得すると同時にアトミックに削除する。
// キーが存在しない場合は redis.Nil に対応する errRESPNil を返す。
// コマンド実行時のエラーはそのまま返す。
func (s *ValkeyStore) GetDel(ctx context.Context, key string) (string, error) {
	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", errRESPNil
		}
		return "", err
	}
	return val, nil
}

func (s *ValkeyStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Eval は指定された Lua スクリプトを Valkey サーバー上でアトミックに実行する。
// script は実行する Lua スクリプト本体。
// keys はスクリプト内で KEYS[1], KEYS[2], ... として参照されるキーの一覧。
// args はスクリプト内で ARGV[1], ARGV[2], ... として参照される追加引数。
// 戻り値は *redis.Cmd で、呼び出し側で .Result() や .Int64() を用いて結果を取得する。
// スクリプトが redis.error_reply を返した場合、.Result() はエラーを返す。
func (s *ValkeyStore) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return s.client.Eval(ctx, script, keys, args...)
}

func (s *ValkeyStore) Increment(ctx context.Context, key string) (int64, error) {
	return s.client.Incr(ctx, key).Result()
}

func (s *ValkeyStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.PExpire(ctx, key, ttl).Err()
}

func (s *ValkeyStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *ValkeyStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("valkey store is required")
	}

	pingContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	if err := s.client.Ping(pingContext).Err(); err != nil {
		return fmt.Errorf("ping valkey: %w", err)
	}

	return nil
}
