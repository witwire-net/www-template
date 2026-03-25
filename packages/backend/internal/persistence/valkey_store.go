package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"www-template/packages/backend/internal/types"
)

var errRESPNil = redis.Nil

type ValkeyStore struct {
	client    *redis.Client
	keyPrefix string
}

func NewValkeyStore(config types.ValkeyConfig) (*ValkeyStore, error) {
	if strings.TrimSpace(config.URL) == "" {
		return nil, errors.New("VALKEY_URL is required")
	}

	options, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, err
	}
	options.DialTimeout = 3 * time.Second
	options.ReadTimeout = 3 * time.Second
	options.WriteTimeout = 3 * time.Second

	return &ValkeyStore{
		client:    redis.NewClient(options),
		keyPrefix: strings.TrimSpace(config.KeyPrefix),
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

func (s *ValkeyStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
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
