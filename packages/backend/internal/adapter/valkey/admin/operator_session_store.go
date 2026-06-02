package admin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	adminauth "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

// OperatorRefreshSessionStore は Admin Operator refresh session state を Valkey に保存する adapter である。
//
// 役割:
//   - adminauth.OperatorRefreshSessionStore port を実装し、Admin operator auth だけへ公開する。
//   - key は `admin:auth:operator-session:*` と `admin:auth:operator-sessions:*` に限定する。
//   - Product account session record や Product application package を import しない。
type OperatorRefreshSessionStore struct {
	store *Store
}

type operatorSessionRecord struct {
	SessionID        string    `json:"sessionId"`
	OperatorID       string    `json:"operatorId"`
	RefreshTokenHash string    `json:"refreshTokenHash"`
	RoleSnapshot     string    `json:"roleSnapshot"`
	ActiveSnapshot   bool      `json:"activeSnapshot"`
	IssuedAt         time.Time `json:"issuedAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
	Revoked          bool      `json:"revoked"`
}

// NewOperatorRefreshSessionStore は Admin Operator refresh session store を構築する。
func NewOperatorRefreshSessionStore(store *Store) *OperatorRefreshSessionStore {
	// Step 1: Admin application port 実装と Admin Valkey store を結びつける。
	return &OperatorRefreshSessionStore{store: store}
}

// Save は Admin Operator session record を TTL 付きで保存する。
func (s *OperatorRefreshSessionStore) Save(ctx context.Context, record adminauth.OperatorSessionRecord, ttl time.Duration) error {
	// Step 1: application DTO を保存 DTO に写像し、JSON encode する。
	payload, err := json.Marshal(operatorSessionRecordFromApplication(record))
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: session 本体と operator index を Admin namespace に保存する。
	if err := s.store.client.Set(ctx, s.sessionKey(record.SessionID), string(payload), ttl).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	idxKey := s.operatorIndexKey(record.OperatorID)
	if err := s.store.client.SAdd(ctx, idxKey, record.SessionID).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if ttl > 0 {
		_ = s.store.client.Expire(ctx, idxKey, ttl).Err()
	}
	return nil
}

// Get は session ID から Admin Operator session record を取得する。
func (s *OperatorRefreshSessionStore) Get(ctx context.Context, sessionID string) (adminauth.OperatorSessionRecord, error) {
	// Step 1: Admin namespace の session key を読み取り、存在しない session は not found として返す。
	value, err := s.store.client.Get(ctx, s.sessionKey(sessionID)).Result()
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return adminauth.OperatorSessionRecord{}, domain.ErrSessionNotFound
		}
		return adminauth.OperatorSessionRecord{}, domain.ErrAuthStoreUnavailable
	}

	// Step 2: 保存 DTO を application DTO に戻す。
	var record operatorSessionRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return adminauth.OperatorSessionRecord{}, domain.ErrAuthStoreUnavailable
	}
	return record.toApplication(), nil
}

// Rotate は current refresh hash 一致時だけ replacement を保存する。
func (s *OperatorRefreshSessionStore) Rotate(ctx context.Context, sessionID string, currentRefreshTokenHash string, replacement adminauth.OperatorSessionRecord, ttl time.Duration) error {
	// Step 1: 旧 session key を監視し、hash 照合と置換を Redis transaction 境界に閉じ込める。
	oldKey := s.sessionKey(sessionID)
	err := s.store.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := s.loadOperatorSessionForRotation(ctx, tx, oldKey)
		if err != nil {
			return err
		}
		if current.RefreshTokenHash != currentRefreshTokenHash || current.Revoked {
			return domain.ErrSessionNotFound
		}
		return s.replaceOperatorSession(ctx, tx, current, replacement, ttl, oldKey)
	}, oldKey)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.ErrSessionNotFound
		}
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// Revoke は対象 Admin Operator session を削除する。
func (s *OperatorRefreshSessionStore) Revoke(ctx context.Context, operatorID string, sessionID string) error {
	// Step 1: session key を削除し、以後の refresh/current 検証で使えないようにする。
	if err := s.store.client.Del(ctx, s.sessionKey(sessionID)).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: operator index から session ID を除去し、Admin session 一覧にも残さない。
	if err := s.store.client.SRem(ctx, s.operatorIndexKey(operatorID), sessionID).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

func (s *OperatorRefreshSessionStore) loadOperatorSessionForRotation(ctx context.Context, tx *redis.Tx, oldKey string) (adminauth.OperatorSessionRecord, error) {
	// Step 1: WATCH 対象 key から現在 record を読み、存在しない session は stable not found に畳む。
	value, err := tx.Get(ctx, oldKey).Result()
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return adminauth.OperatorSessionRecord{}, domain.ErrSessionNotFound
		}
		return adminauth.OperatorSessionRecord{}, domain.ErrAuthStoreUnavailable
	}

	// Step 2: JSON を application DTO へ戻し、壊れた保存値は保存層利用不能として fail-closed にする。
	var record operatorSessionRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return adminauth.OperatorSessionRecord{}, domain.ErrAuthStoreUnavailable
	}
	return record.toApplication(), nil
}

func (s *OperatorRefreshSessionStore) replaceOperatorSession(ctx context.Context, tx *redis.Tx, current adminauth.OperatorSessionRecord, replacement adminauth.OperatorSessionRecord, ttl time.Duration, oldKey string) error {
	// Step 1: replacement payload を先に作り、transaction 内では Redis 操作だけを実行できる状態にする。
	payload, err := json.Marshal(operatorSessionRecordFromApplication(replacement))
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: 旧 session 削除、旧 index 除去、新 session 保存、新 index 追加を MULTI/EXEC でまとめる。
	_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, oldKey)
		pipe.SRem(ctx, s.operatorIndexKey(current.OperatorID), current.SessionID)
		pipe.Set(ctx, s.sessionKey(replacement.SessionID), string(payload), ttl)
		pipe.SAdd(ctx, s.operatorIndexKey(replacement.OperatorID), replacement.SessionID)
		if ttl > 0 {
			pipe.Expire(ctx, s.operatorIndexKey(replacement.OperatorID), ttl)
		}
		return nil
	})
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

func (s *OperatorRefreshSessionStore) sessionKey(sessionID string) string {
	return s.store.key("auth", "operator-session", sessionID)
}

func (s *OperatorRefreshSessionStore) operatorIndexKey(operatorID string) string {
	return s.store.key("auth", "operator-sessions", operatorID)
}

func operatorSessionRecordFromApplication(record adminauth.OperatorSessionRecord) operatorSessionRecord {
	// Step 1: application DTO を Admin Valkey 保存形式へ写像する。
	return operatorSessionRecord{SessionID: record.SessionID, OperatorID: record.OperatorID, RefreshTokenHash: record.RefreshTokenHash, RoleSnapshot: record.RoleSnapshot, ActiveSnapshot: record.ActiveSnapshot, IssuedAt: record.IssuedAt, ExpiresAt: record.ExpiresAt, Revoked: record.Revoked}
}

func (r operatorSessionRecord) toApplication() adminauth.OperatorSessionRecord {
	// Step 1: 保存 record から application DTO へ必要な primitive を戻す。
	return adminauth.OperatorSessionRecord{SessionID: r.SessionID, OperatorID: r.OperatorID, RefreshTokenHash: r.RefreshTokenHash, RoleSnapshot: r.RoleSnapshot, ActiveSnapshot: r.ActiveSnapshot, IssuedAt: r.IssuedAt, ExpiresAt: r.ExpiresAt, Revoked: r.Revoked}
}
