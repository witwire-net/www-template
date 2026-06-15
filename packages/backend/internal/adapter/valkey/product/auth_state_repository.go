package product

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

type AuthStateRepository struct {
	store         *Store
	secretHashKey string
}

// NewAuthStateRepository は AuthStateRepository を生成する。
// store と secretHashKey は必須。secretHashKey は recovery token secret の HMAC-SHA256 に使用する。
func NewAuthStateRepository(store *Store, secretHashKey string) (*AuthStateRepository, error) {
	if store == nil {
		return nil, errors.New("valkey store is required")
	}
	if secretHashKey == "" {
		return nil, errors.New("secret hash key is required")
	}
	return &AuthStateRepository{store: store, secretHashKey: secretHashKey}, nil
}

func (r *AuthStateRepository) Close() error {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.Close()
}

// SaveReauthenticationSession は再認証セッションを TTL 付きで Valkey に保存する。
func (r *AuthStateRepository) SaveReauthenticationSession(ctx context.Context, session domain.ReauthenticationSession, ttl time.Duration) error {
	record := reauthenticationSessionRecord{
		ID:               session.ID(),
		AccountID:        session.AccountID().String(),
		IssuingSessionID: session.IssuingSessionID(),
		OperationKind:    session.OperationKind(),
		RequestID:        session.RequestID(),
		ExpiresAt:        session.ExpiresAt(),
		ConsumedAt:       session.ConsumedAt(),
	}
	return r.setJSON(ctx, r.key("auth", "reauth-session", session.ID()), record, ttl)
}

// ConsumeReauthenticationSession は再認証セッションを GETDEL でアトミックに取得・削除する。
// キーが存在しない場合は domain.ErrReauthSessionNotFound を返す。
func (r *AuthStateRepository) ConsumeReauthenticationSession(ctx context.Context, reauthID string) (domain.ReauthenticationSession, error) {
	result, err := r.store.GetDel(ctx, r.key("auth", "reauth-session", reauthID))
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return emptyReauthenticationSession(), domain.ErrReauthSessionNotFound
		}
		return emptyReauthenticationSession(), domain.ErrAuthStoreUnavailable
	}
	var record reauthenticationSessionRecord
	if err := json.Unmarshal([]byte(result), &record); err != nil {
		return emptyReauthenticationSession(), domain.ErrAuthStoreUnavailable
	}
	return normalizeReauthenticationSessionRecord(record)
}

func (r *AuthStateRepository) SaveChallenge(ctx context.Context, challenge domain.AuthChallenge, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "challenge", challenge.Challenge()), challengeRecordFromDomain(challenge), ttl)
}

func (r *AuthStateRepository) ConsumeChallenge(ctx context.Context, secret string) (domain.AuthChallenge, error) {
	result, err := r.store.GetDel(ctx, r.key("auth", "challenge", secret))
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return emptyChallenge(), domain.ErrChallengeNotFound
		}
		return emptyChallenge(), domain.ErrAuthStoreUnavailable
	}
	var record challengeRecord
	if err := json.Unmarshal([]byte(result), &record); err != nil {
		return emptyChallenge(), domain.ErrAuthStoreUnavailable
	}
	return normalizeChallengeRecord(record)
}

// IssueRecoveryToken は recovery token を Valkey に保存する。
// secret は平文のまま保存せず、hashSecret（HMAC-SHA256+pepper）でハッシュ化した SecretHash として保存する。
// これにより Valkey が漏洩しても平文 secret は復元不可能であり、offline brute-force 攻撃を困難にする。
func (r *AuthStateRepository) IssueRecoveryToken(ctx context.Context, token domain.RecoveryToken, ttl time.Duration) error {
	record := recoveryTokenRecord{
		ID:         token.ID(),
		AccountID:  token.AccountID().String(),
		SecretHash: r.hashSecret(token.Secret()),
		Kind:       string(token.Kind()),
		ExpiresAt:  token.ExpiresAt(),
		ConsumedAt: token.ConsumedAt(),
	}
	return r.setJSON(ctx, r.key("auth", "recovery-token", token.ID()), record, ttl)
}

func (r *AuthStateRepository) SaveRecoveryDeliveryFailure(ctx context.Context, failure domain.RecoveryDeliveryFailure, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "recovery-delivery-failure", failure.RequestID()), recoveryDeliveryFailureRecordFromDomain(failure), ttl)
}

// atomicConsumeRecoveryTokenScript は Valkey Lua スクリプトで recovery token をアトミックに検証・消費する。
// 処理内容:
//  1. GET でトークンレコード（JSON）を取得
//  2. JSON をデコードし、保存済み SecretHash を抽出
//  3. 与えられた expected_hash と比較
//  4. 一致した場合のみ DEL でトークンを削除し、JSON payload を返す
//
// err.Error() の文字列分岐（CODING_STANDARDS.md 禁止パターン）を避けるため、
// エラーケースも structured JSON（status: "not_found"/"mismatch"/"invalid"）を返し、
// Go 側で JSON unmarshal + status field 判定を行う。
// このスクリプトは Valkey サーバー内で単一のアトミック操作として実行されるため、
// GET→比較→DEL の間に別のリクエストが割り込む TOCTOU race condition を防止する。
//
// #nosec G101 -- Lua script contains no credentials; gosec false-positives on 'secretHash' field name
const atomicConsumeRecoveryTokenScript = `
local value = redis.call('GET', KEYS[1])
if not value then
    return cjson.encode({status="not_found"})
end
local ok, decoded = pcall(cjson.decode, value)
if not ok then
    return cjson.encode({status="invalid"})
end
if decoded['secretHash'] ~= ARGV[1] then
    return cjson.encode({status="mismatch"})
end
redis.call('DEL', KEYS[1])
return cjson.encode({status="ok", payload=decoded})
`

// ConsumeRecoveryTokenAtomic は recovery token を tokenID で取得し、
// クライアントから送信された secret のハッシュを記録済みハッシュと照合する。
// ハッシュが一致した場合のみトークンを削除（DEL）する。これにより、誤った secret で
// 正当な tokenID を指定された場合でもトークンが消失する DoS を防止する。
// 検証と削除は Valkey Lua スクリプトでアトミックに実行され、並行リクエストによる
// 二重消費（TOCTOU race condition）を完全に防止する。
// キーが存在しない・ハッシュが一致しない・期限切れ・消費済みの場合はエラーを返す。
// 検証に成功した場合、secret を含む domain.RecoveryToken を返す。
func (r *AuthStateRepository) ConsumeRecoveryTokenAtomic(ctx context.Context, tokenID string, secret string) (domain.RecoveryToken, error) {
	key := r.key("auth", "recovery-token", tokenID)
	expectedHash := r.hashSecret(secret)

	result, err := r.store.Eval(ctx, atomicConsumeRecoveryTokenScript, []string{key}, expectedHash).Result()
	if err != nil {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}

	// Step 1: Lua スクリプトが返す structured JSON（status + payload）をパースし、err.Error() 文字列分岐を避ける。
	resultStr, ok := result.(string)
	if !ok {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	var luaResult struct {
		Status  string              `json:"status"`
		Payload recoveryTokenRecord `json:"payload"`
	}
	if err := json.Unmarshal([]byte(resultStr), &luaResult); err != nil {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}

	// Step 2: not_found / mismatch / invalid は token 消費失敗として扱い、ok だけが成功経路。
	switch luaResult.Status {
	case "not_found", "mismatch", "invalid":
		return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
	case "ok":
		token, err := luaResult.Payload.toDomain(secret)
		if err != nil {
			return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
		}
		return token, nil
	default:
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
}

func (r *AuthStateRepository) SaveRecoverySession(ctx context.Context, session domain.RecoverySession, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "recovery-session", session.ID()), recoverySessionRecordFromDomain(session), ttl)
}

func (r *AuthStateRepository) GetRecoverySession(ctx context.Context, id string) (domain.RecoverySession, error) {
	var record recoverySessionRecord
	if err := r.getJSON(ctx, r.key("auth", "recovery-session", id), &record); err != nil {
		if errors.Is(err, errKeyNotFound) {
			return emptyRecoverySession(), domain.ErrRecoverySessionNotFound
		}
		return emptyRecoverySession(), domain.ErrAuthStoreUnavailable
	}
	return normalizeRecoverySessionRecord(record)
}

func (r *AuthStateRepository) ConsumeRecoverySession(ctx context.Context, session domain.RecoverySession) error {
	return r.setJSON(ctx, r.key("auth", "recovery-session", session.ID()), recoverySessionRecordFromDomain(session), time.Until(session.ExpiresAt()))
}

func (r *AuthStateRepository) IncrementThrottle(ctx context.Context, key string, ttl time.Duration) (int, error) {
	result, err := r.store.Increment(ctx, r.key("auth", "counter", key))
	if err != nil {
		return 0, domain.ErrAuthStoreUnavailable
	}
	count := int(result)
	if count == 1 {
		if err := r.store.Expire(ctx, r.key("auth", "counter", key), ttl); err != nil {
			return 0, domain.ErrAuthStoreUnavailable
		}
	}
	return count, nil
}

func (r *AuthStateRepository) SetLock(ctx context.Context, key string, until time.Time, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "lock", key), lockRecord{LockedUntil: until.UTC()}, ttl)
}

func (r *AuthStateRepository) GetLock(ctx context.Context, key string) (domain.AuthLock, bool, error) {
	var record lockRecord
	if err := r.getJSON(ctx, r.key("auth", "lock", key), &record); err != nil {
		if errors.Is(err, errKeyNotFound) {
			return domain.NewAuthLock(time.Time{}), false, nil
		}
		return domain.NewAuthLock(time.Time{}), false, domain.ErrAuthStoreUnavailable
	}
	return domain.NewAuthLock(record.LockedUntil), true, nil
}

func (r *AuthStateRepository) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	err = r.store.Set(ctx, key, string(payload), ttl)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

func (r *AuthStateRepository) getJSON(ctx context.Context, key string, target any) error {
	result, err := r.store.Get(ctx, key)
	if err != nil {
		return err
	}
	if result == "" {
		return errKeyNotFound
	}
	return json.Unmarshal([]byte(result), target)
}

func (r *AuthStateRepository) key(parts ...string) string {
	return r.store.Key(parts...)
}

type challengeRecord struct {
	ID         string    `json:"id"`
	Identifier string    `json:"identifier"`
	Challenge  string    `json:"challenge"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func challengeRecordFromDomain(challenge domain.AuthChallenge) challengeRecord {
	return challengeRecord{ID: challenge.ID(), Identifier: challenge.Identifier(), Challenge: challenge.Challenge(), ExpiresAt: challenge.ExpiresAt()}
}

func (r challengeRecord) toDomain() (domain.AuthChallenge, error) {
	return domain.NewAuthChallenge(r.ID, r.Identifier, r.Challenge, r.ExpiresAt)
}

func normalizeChallengeRecord(record challengeRecord) (domain.AuthChallenge, error) {
	challenge, err := record.toDomain()
	if err != nil {
		return emptyChallenge(), domain.ErrAuthStoreUnavailable
	}
	return challenge, nil
}

type recoveryTokenRecord struct {
	ID         string     `json:"id"`
	AccountID  string     `json:"accountId"`
	SecretHash string     `json:"secretHash"`
	Kind       string     `json:"kind"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
}

type recoveryDeliveryFailureRecord struct {
	RequestID       string    `json:"requestId"`
	RecoveryTokenID string    `json:"recoveryTokenId"`
	AccountID       string    `json:"accountId"`
	Email           string    `json:"email"`
	DeliveryStage   string    `json:"deliveryStage"`
	ErrorClass      string    `json:"errorClass"`
	FailedAt        time.Time `json:"failedAt"`
	RetryAfter      time.Time `json:"retryAfter"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

func recoveryDeliveryFailureRecordFromDomain(failure domain.RecoveryDeliveryFailure) recoveryDeliveryFailureRecord {
	return recoveryDeliveryFailureRecord{
		RequestID:       failure.RequestID(),
		RecoveryTokenID: failure.RecoveryTokenID(),
		AccountID:       failure.AccountID().String(),
		Email:           failure.Email(),
		DeliveryStage:   failure.DeliveryStage(),
		ErrorClass:      failure.ErrorClass(),
		FailedAt:        failure.FailedAt(),
		RetryAfter:      failure.RetryAfter(),
		ExpiresAt:       failure.ExpiresAt(),
	}
}

func (r recoveryTokenRecord) toDomain(secret string) (domain.RecoveryToken, error) {
	accountID, err := domain.NewAccountID(r.AccountID)
	if err != nil {
		return emptyRecoveryToken(), err
	}
	return domain.ReconstituteRecoveryToken(r.ID, accountID, secret, domain.TokenKind(r.Kind), r.ExpiresAt, r.ConsumedAt)
}

// hashSecret は secret を server-side pepper（SecretHashKey）で HMAC-SHA256 し、
// base64 エンコードして返す。低エントロピーな secret をそのまま key にしないため、
// offline brute-force を困難にする。
func (r *AuthStateRepository) hashSecret(secret string) string {
	mac := hmac.New(sha256.New, []byte(r.secretHashKey))
	mac.Write([]byte(secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type recoverySessionRecord struct {
	ID         string     `json:"id"`
	AccountID  string     `json:"accountId"`
	Kind       string     `json:"kind"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
}

func recoverySessionRecordFromDomain(session domain.RecoverySession) recoverySessionRecord {
	return recoverySessionRecord{ID: session.ID(), AccountID: session.AccountID().String(), Kind: string(session.Kind()), ExpiresAt: session.ExpiresAt(), ConsumedAt: session.ConsumedAt()}
}

func (r recoverySessionRecord) toDomain() (domain.RecoverySession, error) {
	accountID, err := domain.NewAccountID(r.AccountID)
	if err != nil {
		return emptyRecoverySession(), err
	}
	return domain.ReconstituteRecoverySession(r.ID, accountID, domain.TokenKind(r.Kind), r.ExpiresAt, r.ConsumedAt)
}

func normalizeRecoverySessionRecord(record recoverySessionRecord) (domain.RecoverySession, error) {
	session, err := record.toDomain()
	if err != nil {
		return emptyRecoverySession(), domain.ErrAuthStoreUnavailable
	}
	return session, nil
}

type lockRecord struct {
	LockedUntil time.Time `json:"lockedUntil"`
}

func emptyChallenge() domain.AuthChallenge {
	challenge, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return challenge
}

func emptyRecoveryToken() domain.RecoveryToken {
	accountID, _ := domain.NewAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, "placeholder", domain.TokenKindRecovery, time.Unix(1, 0).UTC())
	return token
}

func emptyRecoverySession() domain.RecoverySession {
	accountID, _ := domain.NewAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, domain.TokenKindRecovery, time.Unix(1, 0).UTC())
	return session
}

type reauthenticationSessionRecord struct {
	ID               string     `json:"id"`
	AccountID        string     `json:"accountId"`
	IssuingSessionID string     `json:"issuingSessionId"`
	OperationKind    string     `json:"operationKind"`
	RequestID        string     `json:"requestId"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	ConsumedAt       *time.Time `json:"consumedAt,omitempty"`
}

func normalizeReauthenticationSessionRecord(record reauthenticationSessionRecord) (domain.ReauthenticationSession, error) {
	accountID, err := domain.NewAccountID(record.AccountID)
	if err != nil {
		return emptyReauthenticationSession(), domain.ErrAuthStoreUnavailable
	}
	session, err := domain.ReconstituteReauthenticationSession(
		record.ID, accountID, record.IssuingSessionID, record.OperationKind, record.RequestID,
		record.ExpiresAt, record.ConsumedAt,
	)
	if err != nil {
		return emptyReauthenticationSession(), domain.ErrAuthStoreUnavailable
	}
	return session, nil
}

func emptyReauthenticationSession() domain.ReauthenticationSession {
	accountID, _ := domain.NewAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	session, _ := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"device-link", "01ARZ3NDEKTSV4RRFFQ69G5FAY", time.Unix(1, 0).UTC(),
	)
	return session
}
