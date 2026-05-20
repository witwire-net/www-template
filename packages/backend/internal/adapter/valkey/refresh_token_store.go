package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	application "www-template/packages/backend/internal/application"
	domain "www-template/packages/backend/internal/domain"
)

// RefreshTokenStore は Valkey 上でリフレッシュトークンを管理する実装。
// キースキーマ:
//   - auth:refresh:{hash} → RefreshTokenRecord (JSON)
//   - auth:refresh_index:{accountID}:{fingerprint} → set of hashes
type RefreshTokenStore struct {
	store *ValkeyStore
}

// NewRefreshTokenStore は RefreshTokenStore を生成する。
func NewRefreshTokenStore(store *ValkeyStore) *RefreshTokenStore {
	return &RefreshTokenStore{store: store}
}

// Save はリフレッシュトークンハッシュに対応するレコードを保存する。
// ttl が 0 の場合は無期限（NO EXPIRE）で SET する。
func (s *RefreshTokenStore) Save(ctx context.Context, hash string, record application.RefreshTokenRecord, ttl time.Duration) error {
	key := s.store.Key("auth", "refresh", hash)
	payload, err := json.Marshal(record)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if err := s.store.Set(ctx, key, string(payload), ttl); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	// インデックスに追加
	idxKey := s.store.Key("auth", "refresh_index", record.AccountID.String(), record.Fingerprint)
	if err := s.store.client.SAdd(ctx, idxKey, hash).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if ttl > 0 {
		_ = s.store.client.Expire(ctx, idxKey, ttl).Err()
	}
	return nil
}

// consumeScript は GETDEL + SET consumed marker + SREM index を単一 Lua スクリプトで原子実行する。
// KEYS[1] = refresh key, KEYS[2] = consumed key, KEYS[3] = index key
// ARGV[1] = consumed TTL (seconds), ARGV[2] = hash (for SREM)
const consumeScript = `
local val = redis.call('GETDEL', KEYS[1])
if val == false then
	return nil
end
redis.call('SET', KEYS[2], val, 'EX', tonumber(ARGV[1]))
redis.call('SREM', KEYS[3], ARGV[2])
return val
`

// Consume は指定したハッシュのリフレッシュトークンを Lua script でアトミックに取得・削除し、
// 消費済みマーカーとインデックスを同時に更新する。
// 成功時には盗難検出のため consumed キーに 7 日間保持する。
func (s *RefreshTokenStore) Consume(ctx context.Context, hash string) (application.RefreshTokenRecord, error) {
	key := s.store.Key("auth", "refresh", hash)
	consumedKey := s.store.Key("auth", "refresh_consumed", hash)

	// インデックスキーはレコード内に fingerprint が含まれるため、事前に取得する必要がある。
	// そのため、まず GET でレコードを取得し、Lua script 内で DEL する。
	// より厳密な原子性が必要な場合は Redis transaction を使用するが、
	// ここでは GET → インデックス特定 → EVAL で GETDEL+SET+SREM の順序で整合性を保つ。
	val, err := s.store.Get(ctx, key)
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
		}
		return application.RefreshTokenRecord{}, domain.ErrAuthStoreUnavailable
	}
	var record application.RefreshTokenRecord
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		return application.RefreshTokenRecord{}, domain.ErrAuthStoreUnavailable
	}

	idxKey := s.store.Key("auth", "refresh_index", record.AccountID.String(), record.Fingerprint)
	result, err := s.store.client.Eval(ctx, consumeScript, []string{key, consumedKey, idxKey}, int(7*24*time.Hour.Seconds()), hash).Result()
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
		}
		return application.RefreshTokenRecord{}, domain.ErrAuthStoreUnavailable
	}
	if result == nil {
		return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
	}
	return record, nil
}

// GetConsumed は指定したハッシュが既に消費されているか確認する。
func (s *RefreshTokenStore) GetConsumed(ctx context.Context, hash string) (application.RefreshTokenRecord, error) {
	consumedKey := s.store.Key("auth", "refresh_consumed", hash)
	val, err := s.store.Get(ctx, consumedKey)
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
		}
		return application.RefreshTokenRecord{}, domain.ErrAuthStoreUnavailable
	}
	var record application.RefreshTokenRecord
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		return application.RefreshTokenRecord{}, domain.ErrAuthStoreUnavailable
	}
	return record, nil
}

// RevokeAllForFingerprint は同一アカウント・同一デバイス指紋の全リフレッシュトークンを失効する。
func (s *RefreshTokenStore) RevokeAllForFingerprint(ctx context.Context, accountID domain.AccountID, fingerprint string) error {
	idxKey := s.store.Key("auth", "refresh_index", accountID.String(), fingerprint)
	hashes, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if len(hashes) > 0 {
		keys := make([]string, len(hashes))
		for i, h := range hashes {
			keys[i] = s.store.Key("auth", "refresh", h)
		}
		if err := s.store.client.Del(ctx, keys...).Err(); err != nil {
			return domain.ErrAuthStoreUnavailable
		}
	}
	if err := s.store.client.Del(ctx, idxKey).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// RevokeBySessionID は指定されたセッション ID に紐づく全リフレッシュトークンを失効する。
// インデックスを全スキャンする必要があるため、効率は最適ではないが運用上十分とする。
// 読み取り・削除・unmarshal のいずれかでエラーが発生した場合は fail-closed でエラーを返す。
func (s *RefreshTokenStore) RevokeBySessionID(ctx context.Context, accountID domain.AccountID, sessionID string) error {
	// アカウント単位の全インデックスをスキャンする簡易実装。
	// 実運用では prefix スキャンを利用する。
	pattern := s.store.Key("auth", "refresh_index", accountID.String(), "*")
	iter := s.store.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		idxKey := iter.Val()
		hashes, err := s.store.client.SMembers(ctx, idxKey).Result()
		if err != nil {
			return domain.ErrAuthStoreUnavailable
		}
		for _, h := range hashes {
			key := s.store.Key("auth", "refresh", h)
			val, err := s.store.Get(ctx, key)
			if err != nil {
				// インデックスにはあるが実体がない場合はインデックスから削除して継続する
				if errors.Is(err, errRESPNil) {
					if err := s.store.client.SRem(ctx, idxKey, h).Err(); err != nil {
						return domain.ErrAuthStoreUnavailable
					}
					continue
				}
				return domain.ErrAuthStoreUnavailable
			}
			var record application.RefreshTokenRecord
			if err := json.Unmarshal([]byte(val), &record); err != nil {
				return domain.ErrAuthStoreUnavailable
			}
			if record.SessionID == sessionID {
				if err := s.store.Delete(ctx, key); err != nil {
					return domain.ErrAuthStoreUnavailable
				}
				if err := s.store.client.SRem(ctx, idxKey, h).Err(); err != nil {
					return domain.ErrAuthStoreUnavailable
				}
			}
		}
	}
	if err := iter.Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}
