package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/auth/domain"
)

// SessionStore は Valkey 上でセッションメタデータを管理する実装。
// キースキーマ:
//   - auth:session:{sessionID} → SessionMetadata (JSON)
//   - auth:account-sessions:{accountID} → set of sessionIDs
type SessionStore struct {
	store *ValkeyStore
}

// NewSessionStore は SessionStore を生成する。
func NewSessionStore(store *ValkeyStore) *SessionStore {
	return &SessionStore{store: store}
}

// SaveSession はセッションメタデータを保存する。
func (s *SessionStore) SaveSession(ctx context.Context, sessionID, accountID string, metadata application.SessionMetadata, ttl time.Duration) error {
	key := s.store.Key("auth", "session", sessionID)
	payload, err := json.Marshal(metadata)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if err := s.store.Set(ctx, key, string(payload), ttl); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	idxKey := s.store.Key("auth", "account-sessions", accountID)
	if err := s.store.client.SAdd(ctx, idxKey, sessionID).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if ttl > 0 {
		_ = s.store.client.Expire(ctx, idxKey, ttl).Err()
	}
	return nil
}

// GetSession はセッション ID からメタデータを取得する。
func (s *SessionStore) GetSession(ctx context.Context, sessionID string) (application.SessionMetadata, error) {
	key := s.store.Key("auth", "session", sessionID)
	val, err := s.store.Get(ctx, key)
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return application.SessionMetadata{}, domain.ErrSessionNotFound
		}
		return application.SessionMetadata{}, domain.ErrAuthStoreUnavailable
	}
	var metadata application.SessionMetadata
	if err := json.Unmarshal([]byte(val), &metadata); err != nil {
		return application.SessionMetadata{}, domain.ErrAuthStoreUnavailable
	}
	return metadata, nil
}

// ListSessions はアカウントに紐づく全セッションのメタデータを返す。
// いずれかのメタデータが破損している場合は fail-closed でエラーを返す。
func (s *SessionStore) ListSessions(ctx context.Context, accountID string) ([]application.SessionMetadata, error) {
	idxKey := s.store.Key("auth", "account-sessions", accountID)
	sessionIDs, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}
	if len(sessionIDs) == 0 {
		return []application.SessionMetadata{}, nil
	}

	// MGET で一括取得
	keys := make([]string, len(sessionIDs))
	for i, id := range sessionIDs {
		keys[i] = s.store.Key("auth", "session", id)
	}
	values, err := s.store.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}

	result := make([]application.SessionMetadata, 0, len(values))
	for i, v := range values {
		if v == nil {
			// stale index: セッションが既に失効しているがインデックスに残っている
			// クリーンアップして継続する
			if err := s.store.client.SRem(ctx, idxKey, sessionIDs[i]).Err(); err != nil {
				return nil, domain.ErrAuthStoreUnavailable
			}
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, domain.ErrAuthStoreUnavailable
		}
		var metadata application.SessionMetadata
		if err := json.Unmarshal([]byte(str), &metadata); err != nil {
			return nil, domain.ErrAuthStoreUnavailable
		}
		metadata.SessionID = sessionIDs[i]
		result = append(result, metadata)
	}
	return result, nil
}

// RevokeSession は特定セッションを削除する。
func (s *SessionStore) RevokeSession(ctx context.Context, accountID, sessionID string) error {
	key := s.store.Key("auth", "session", sessionID)
	if err := s.store.Delete(ctx, key); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	idxKey := s.store.Key("auth", "account-sessions", accountID)
	if err := s.store.client.SRem(ctx, idxKey, sessionID).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// RevokeOthers は現在のセッション以外を全て削除し、削除した session ID のスライスを返す。
func (s *SessionStore) RevokeOthers(ctx context.Context, accountID, currentSessionID string) ([]string, error) {
	idxKey := s.store.Key("auth", "account-sessions", accountID)
	sessionIDs, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}
	deleted := make([]string, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if id == currentSessionID {
			continue
		}
		key := s.store.Key("auth", "session", id)
		if err := s.store.Delete(ctx, key); err != nil {
			return nil, domain.ErrAuthStoreUnavailable
		}
		if err := s.store.client.SRem(ctx, idxKey, id).Err(); err != nil {
			return nil, domain.ErrAuthStoreUnavailable
		}
		deleted = append(deleted, id)
	}
	return deleted, nil
}

// RevokeAllForAccount は指定されたアカウントに紐づく全セッションを失効する。
// account-sessions インデックスから全 sessionID を取得し、個別のセッションキーとインデックスを削除する。
func (s *SessionStore) RevokeAllForAccount(ctx context.Context, accountID string) error {
	idxKey := s.store.Key("auth", "account-sessions", accountID)
	sessionIDs, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	for _, id := range sessionIDs {
		key := s.store.Key("auth", "session", id)
		if err := s.store.Delete(ctx, key); err != nil {
			return domain.ErrAuthStoreUnavailable
		}
	}
	if err := s.store.client.Del(ctx, idxKey).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}
