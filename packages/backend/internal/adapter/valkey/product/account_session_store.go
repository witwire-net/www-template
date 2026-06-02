package product

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	productauth "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

// AccountRefreshSessionStore は Product Account refreshToken の server-side state を Valkey に保存する adapter である。
//
// 役割:
//   - productauth.AccountRefreshSessionStore port を実装し、Product AccountAuth use case だけへ公開する。
//   - key は `product:auth:refresh:*` と `product:auth:refresh_index:*` に限定し、Admin operator session と共有しない。
//   - 平文 refreshToken は受け取らず、domain.OpaqueTokenHash だけを永続化 key として扱う。
type AccountRefreshSessionStore struct {
	store *Store
}

// AccountSessionMetadataStore は Product account session metadata を Valkey に保存する adapter である。
//
// 役割:
//   - productauth.AccountSessionMetadataStore port を実装し、accessToken bearer validation 用の session selector を保持する。
//   - key は `product:auth:session:*` と `product:auth:account-sessions:*` に限定する。
type AccountSessionMetadataStore struct {
	store *Store
}

type refreshSessionRecord struct {
	AccountID string     `json:"accountId"`
	SessionID string     `json:"sessionId"`
	TokenHash string     `json:"tokenHash"`
	IssuedAt  time.Time  `json:"issuedAt"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

type sessionMetadataRecord struct {
	AccountID    string    `json:"accountId"`
	SessionID    string    `json:"sessionId"`
	DeviceName   string    `json:"deviceName"`
	LoginAt      time.Time `json:"loginAt"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	IPHash       string    `json:"ipHash"`
}

// NewAccountRefreshSessionStore は Product Account refresh session store を構築する。
func NewAccountRefreshSessionStore(store *Store) *AccountRefreshSessionStore {
	// Step 1: store の nil 検査は呼び出し側の runtime validation に委譲し、constructor は port 実装の組み立てだけを担う。
	return &AccountRefreshSessionStore{store: store}
}

// NewAccountSessionMetadataStore は Product account session metadata store を構築する。
func NewAccountSessionMetadataStore(store *Store) *AccountSessionMetadataStore {
	// Step 1: Product metadata port と Valkey 接続を結びつけ、Admin metadata store と別 package に閉じ込める。
	return &AccountSessionMetadataStore{store: store}
}

// Save は Product refresh session を TTL 付きで保存する。
func (s *AccountRefreshSessionStore) Save(ctx context.Context, session domain.AccountRefreshSession, ttl time.Duration) error {
	// Step 1: domain object から保存用 record を作り、壊れた adapter DTO を application へ公開しない。
	record := refreshRecordFromDomain(session)
	payload, err := json.Marshal(record)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: refresh token hash keyed record と session index を同じ Product namespace へ保存する。
	if err := s.store.client.Set(ctx, s.refreshKey(session.TokenHash()), string(payload), ttl).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return s.indexSession(ctx, session.AccountID(), session.SessionID(), session.TokenHash(), ttl)
}

// Rotate は旧 refresh session を消費し、callback が返した次 session を保存する。
func (s *AccountRefreshSessionStore) Rotate(ctx context.Context, tokenHash domain.OpaqueTokenHash, ttl time.Duration, build productauth.RefreshRotationBuilder) (domain.AccountRefreshSession, domain.AccountRefreshSession, error) {
	// Step 1: hash に対応する既存 refresh session を GETDEL で取得し、旧 token の二重利用を原子的に拒否できる状態にする。
	current, err := s.consumeByHash(ctx, tokenHash)
	if err != nil {
		return zeroRefreshSession(), zeroRefreshSession(), err
	}

	// Step 2: application callback へ domain object を渡し、Account 状態や selector 検証を adapter に再実装しない。
	next, err := build(current)
	if err != nil {
		return current, zeroRefreshSession(), err
	}

	// Step 3: 新 session を保存してから旧 hash の index だけを削除し、refresh key は GETDEL 済みで残さない。
	if err := s.Save(ctx, next, ttl); err != nil {
		return current, zeroRefreshSession(), err
	}
	if err := s.removeHashFromIndex(ctx, current); err != nil {
		return current, zeroRefreshSession(), err
	}

	// Step 4: rotation 前後の Product refresh session を application へ返す。
	return current, next, nil
}

// RevokeSession は対象 Product account session の refresh state を削除する。
func (s *AccountRefreshSessionStore) RevokeSession(ctx context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID, _ time.Time) error {
	// Step 1: session index から refresh hash 一覧を取得し、対象 session の refresh token をすべて削除する。
	idxKey := s.sessionIndexKey(accountID, sessionID)
	hashes, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	for _, hash := range hashes {
		if err := s.store.client.Del(ctx, s.store.key("auth", "refresh", hash)).Err(); err != nil {
			return domain.ErrAuthStoreUnavailable
		}
	}

	// Step 2: session/account index を削除し、以後の refresh lookup からも見えないようにする。
	if err := s.store.client.Del(ctx, idxKey).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if err := s.store.client.SRem(ctx, s.accountIndexKey(accountID), sessionID.String()).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// RevokeAllForAccount は対象 Product account の全 refresh state を削除する。
func (s *AccountRefreshSessionStore) RevokeAllForAccount(ctx context.Context, accountID domain.AccountID, revokedAt time.Time) error {
	// Step 1: account index から session ID 一覧を取得し、各 session の refresh state 削除へ委譲する。
	sessionIDs, err := s.store.client.SMembers(ctx, s.accountIndexKey(accountID)).Result()
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	for _, rawSessionID := range sessionIDs {
		sessionID, err := domain.NewAccountAuthSessionID(rawSessionID)
		if err != nil {
			return domain.ErrAuthStoreUnavailable
		}
		if err := s.RevokeSession(ctx, accountID, sessionID, revokedAt); err != nil {
			return err
		}
	}

	// Step 2: account index 自体も削除し、空集合の stale key を残さない。
	if err := s.store.client.Del(ctx, s.accountIndexKey(accountID)).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// Save は Product session metadata を TTL 付きで保存する。
func (s *AccountSessionMetadataStore) Save(ctx context.Context, metadata productauth.SessionMetadata, ttl time.Duration) error {
	// Step 1: application DTO を保存 DTO に写像し、Valkey JSON と application public API を分ける。
	record := sessionMetadataRecordFromApplication(metadata)
	payload, err := json.Marshal(record)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: session metadata 本体と account index を同じ TTL で保存し、期限切れ後の列挙漏れを抑える。
	if err := s.store.client.Set(ctx, s.sessionKey(metadata.SessionID), string(payload), ttl).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	idxKey := s.store.key("auth", "account-sessions", metadata.AccountID.String())
	if err := s.store.client.SAdd(ctx, idxKey, metadata.SessionID).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if ttl > 0 {
		_ = s.store.client.Expire(ctx, idxKey, ttl).Err()
	}
	return nil
}

// Get は Product session ID から metadata を取得する。
func (s *AccountSessionMetadataStore) Get(ctx context.Context, sessionID domain.AccountAuthSessionID) (productauth.SessionMetadata, error) {
	// Step 1: Product namespace の session key を読み、存在しない場合は domain の session not found に畳み込む。
	value, err := s.store.client.Get(ctx, s.sessionKey(sessionID.String())).Result()
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return productauth.SessionMetadata{}, domain.ErrSessionNotFound
		}
		return productauth.SessionMetadata{}, domain.ErrAuthStoreUnavailable
	}

	// Step 2: JSON record を application DTO へ復元し、壊れた保存値は fail-closed にする。
	var record sessionMetadataRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return productauth.SessionMetadata{}, domain.ErrAuthStoreUnavailable
	}
	return record.toApplication()
}

// List は Product account に紐づく session metadata を account index から復元する。
func (s *AccountSessionMetadataStore) List(ctx context.Context, accountID domain.AccountID) ([]productauth.SessionMetadata, error) {
	// Step 1: Product account 専用 index を読み、Admin operator session namespace と混ざらない一覧取得に限定する。
	idxKey := s.store.key("auth", "account-sessions", accountID.String())
	sessionIDs, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}

	// Step 2: 各 session ID を domain value として検証し、壊れた index は fail-closed に store unavailable へ畳む。
	sessions := make([]productauth.SessionMetadata, 0, len(sessionIDs))
	for _, rawSessionID := range sessionIDs {
		sessionID, err := domain.NewAccountAuthSessionID(rawSessionID)
		if err != nil {
			return nil, domain.ErrAuthStoreUnavailable
		}
		metadata, err := s.Get(ctx, sessionID)
		if err != nil {
			if errors.Is(err, domain.ErrSessionNotFound) {
				continue
			}
			return nil, err
		}
		if metadata.AccountID != accountID {
			return nil, domain.ErrAuthStoreUnavailable
		}
		sessions = append(sessions, metadata)
	}

	// Step 3: 復元できた metadata だけを返し、期限切れで消えた session key は一覧から自然に除外する。
	return sessions, nil
}

// Revoke は対象 Product session metadata を削除する。
func (s *AccountSessionMetadataStore) Revoke(ctx context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID) error {
	// Step 1: bearer validation 用 metadata を削除し、失効済み session の accessToken 継続利用を拒否できる状態にする。
	if err := s.store.client.Del(ctx, s.sessionKey(sessionID.String())).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: account index から session ID を除去し、session 一覧にも残さない。
	idxKey := s.store.key("auth", "account-sessions", accountID.String())
	if err := s.store.client.SRem(ctx, idxKey, sessionID.String()).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// RevokeAllForAccount は対象 Product account の session metadata をすべて削除する。
func (s *AccountSessionMetadataStore) RevokeAllForAccount(ctx context.Context, accountID domain.AccountID) error {
	// Step 1: account index から session ID を列挙し、metadata key を個別に削除する。
	idxKey := s.store.key("auth", "account-sessions", accountID.String())
	sessionIDs, err := s.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	for _, sessionID := range sessionIDs {
		if err := s.store.client.Del(ctx, s.sessionKey(sessionID)).Err(); err != nil {
			return domain.ErrAuthStoreUnavailable
		}
	}

	// Step 2: account index を削除し、空の Product session 集合を残さない。
	if err := s.store.client.Del(ctx, idxKey).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

func (s *AccountRefreshSessionStore) consumeByHash(ctx context.Context, tokenHash domain.OpaqueTokenHash) (domain.AccountRefreshSession, error) {
	// Step 1: GETDEL により refresh record の取得と削除を Valkey 上で原子的に実行する。
	value, err := s.store.client.GetDel(ctx, s.refreshKey(tokenHash)).Result()
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return zeroRefreshSession(), domain.ErrSessionNotFound
		}
		return zeroRefreshSession(), domain.ErrAuthStoreUnavailable
	}

	// Step 2: 削除済み record を domain object へ復元し、application callback の入力を検証済みにする。
	var record refreshSessionRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return zeroRefreshSession(), domain.ErrAuthStoreUnavailable
	}
	return record.toDomain()
}

func (s *AccountRefreshSessionStore) indexSession(ctx context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID, tokenHash domain.OpaqueTokenHash, ttl time.Duration) error {
	// Step 1: session 単位と account 単位の index を更新し、revoke 操作が scan なしで対象を特定できるようにする。
	if err := s.store.client.SAdd(ctx, s.sessionIndexKey(accountID, sessionID), tokenHash.String()).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if err := s.store.client.SAdd(ctx, s.accountIndexKey(accountID), sessionID.String()).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	// Step 2: TTL が指定されている場合は index にも期限を付け、refresh record と index の寿命を揃える。
	if ttl > 0 {
		_ = s.store.client.Expire(ctx, s.sessionIndexKey(accountID, sessionID), ttl).Err()
		_ = s.store.client.Expire(ctx, s.accountIndexKey(accountID), ttl).Err()
	}
	return nil
}

func (s *AccountRefreshSessionStore) removeHashFromIndex(ctx context.Context, session domain.AccountRefreshSession) error {
	// Step 1: session index から旧 hash を除去し、同一 session の次 token だけが残るようにする。
	if err := s.store.client.SRem(ctx, s.sessionIndexKey(session.AccountID(), session.SessionID()), session.TokenHash().String()).Err(); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

func (s *AccountRefreshSessionStore) refreshKey(tokenHash domain.OpaqueTokenHash) string {
	return s.store.key("auth", "refresh", tokenHash.String())
}

func (s *AccountRefreshSessionStore) sessionIndexKey(accountID domain.AccountID, sessionID domain.AccountAuthSessionID) string {
	return s.store.key("auth", "refresh_index", accountID.String(), sessionID.String())
}

func (s *AccountRefreshSessionStore) accountIndexKey(accountID domain.AccountID) string {
	return s.store.key("auth", "refresh_accounts", accountID.String())
}

func (s *AccountSessionMetadataStore) sessionKey(sessionID string) string {
	return s.store.key("auth", "session", sessionID)
}

func refreshRecordFromDomain(session domain.AccountRefreshSession) refreshSessionRecord {
	// Step 1: domain object の accessor だけを使い、未検証 field へ直接触れない。
	return refreshSessionRecord{AccountID: session.AccountID().String(), SessionID: session.SessionID().String(), TokenHash: session.TokenHash().String(), IssuedAt: session.IssuedAt(), ExpiresAt: session.ExpiresAt(), RevokedAt: session.RevokedAt()}
}

func (r refreshSessionRecord) toDomain() (domain.AccountRefreshSession, error) {
	// Step 1: 保存された文字列 ID/hash を Product domain value として再検証する。
	accountID, err := domain.NewAccountID(r.AccountID)
	if err != nil {
		return zeroRefreshSession(), domain.ErrAuthStoreUnavailable
	}
	sessionID, err := domain.NewAccountAuthSessionID(r.SessionID)
	if err != nil {
		return zeroRefreshSession(), domain.ErrAuthStoreUnavailable
	}
	if r.TokenHash == "" {
		return zeroRefreshSession(), domain.ErrAuthStoreUnavailable
	}
	tokenHash := domain.OpaqueTokenHash(r.TokenHash)

	// Step 2: domain reconstitution helper で時刻範囲と revokedAt を再検証する。
	return domain.ReconstituteAccountRefreshSession(accountID, sessionID, tokenHash, r.IssuedAt, r.ExpiresAt, r.RevokedAt)
}

func sessionMetadataRecordFromApplication(metadata productauth.SessionMetadata) sessionMetadataRecord {
	// Step 1: application DTO の値を保存用 JSON record に写像する。
	return sessionMetadataRecord{AccountID: metadata.AccountID.String(), SessionID: metadata.SessionID, DeviceName: metadata.DeviceName, LoginAt: metadata.LoginAt, LastActiveAt: metadata.LastActiveAt, IPHash: metadata.IPHash}
}

func (r sessionMetadataRecord) toApplication() (productauth.SessionMetadata, error) {
	// Step 1: AccountID を domain value として再検証し、壊れた metadata を fail-closed にする。
	accountID, err := domain.NewAccountID(r.AccountID)
	if err != nil {
		return productauth.SessionMetadata{}, domain.ErrAuthStoreUnavailable
	}

	// Step 2: application DTO として必要な値だけを復元する。
	return productauth.SessionMetadata{AccountID: accountID, SessionID: r.SessionID, DeviceName: r.DeviceName, LoginAt: r.LoginAt, LastActiveAt: r.LastActiveAt, IPHash: r.IPHash}, nil
}

func zeroRefreshSession() domain.AccountRefreshSession {
	// Step 1: guardrail が domain composite literal を禁止するため、固定の検証済み値を constructor に通して placeholder を作る。
	accountID, _ := domain.NewAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	sessionID, _ := domain.NewAccountAuthSessionID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	tokenHash, _ := domain.HashOpaqueToken("placeholder-product-refresh-token")
	issuedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Step 2: placeholder は error return と必ず併用されるが、値自体も domain invariant を満たすようにする。
	session, _ := domain.ReconstituteAccountRefreshSession(accountID, sessionID, tokenHash, issuedAt, issuedAt.Add(time.Minute), nil)
	return session
}
