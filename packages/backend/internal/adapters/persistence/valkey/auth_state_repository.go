package valkey

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"www-template/packages/backend/internal/auth/domain"
)

type AuthStateRepository struct {
	store *ValkeyStore
}

func NewAuthStateRepository(store *ValkeyStore) (*AuthStateRepository, error) {
	if store == nil {
		return nil, errors.New("valkey store is required")
	}
	return &AuthStateRepository{store: store}, nil
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
		AccountID:        session.AccountID(),
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
		if errors.Is(err, errRESPNil) {
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
		if errors.Is(err, errRESPNil) {
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

func (r *AuthStateRepository) IssueRecoveryToken(ctx context.Context, token domain.RecoveryToken, ttl time.Duration) error {
	record := recoveryTokenRecordFromDomain(token)
	return r.setJSON(ctx, r.key("auth", "recovery-token", token.ID()), record, ttl)
}

func (r *AuthStateRepository) SaveRecoveryDeliveryFailure(ctx context.Context, failure domain.RecoveryDeliveryFailure, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "recovery-delivery-failure", failure.RequestID()), recoveryDeliveryFailureRecordFromDomain(failure), ttl)
}

func (r *AuthStateRepository) GetRecoveryTokenBySecret(ctx context.Context, secret string) (domain.RecoveryToken, error) {
	var record recoveryTokenRecord
	if err := r.getJSON(ctx, r.key("auth", "recovery-token", secret), &record); err != nil {
		if errors.Is(err, errRESPNil) {
			return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
		}
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	return normalizeRecoveryTokenRecord(record)
}

func (r *AuthStateRepository) ConsumeRecoveryToken(ctx context.Context, token domain.RecoveryToken) error {
	return r.setJSON(ctx, r.key("auth", "recovery-token", token.Secret()), recoveryTokenRecordFromDomain(token), time.Until(token.ExpiresAt()))
}

// ConsumeRecoveryTokenAtomic は recovery token を tokenID でアトミックに取得・削除（GETDEL）し、
// クライアントから送信された secret のハッシュを記録済みハッシュと照合する。
// キーが存在しない・ハッシュが一致しない・期限切れ・消費済みの場合はエラーを返す。
// 検証に成功した場合、secret を含む domain.RecoveryToken を返す。
func (r *AuthStateRepository) ConsumeRecoveryTokenAtomic(ctx context.Context, tokenID string, secret string) (domain.RecoveryToken, error) {
	result, err := r.store.GetDel(ctx, r.key("auth", "recovery-token", tokenID))
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
		}
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	var record recoveryTokenRecord
	if err := json.Unmarshal([]byte(result), &record); err != nil {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	if record.SecretHash != hashSecret(secret) {
		return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
	}
	token, err := record.toDomain(secret)
	if err != nil {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	return token, nil
}

func (r *AuthStateRepository) SaveRecoverySession(ctx context.Context, session domain.RecoverySession, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "recovery-session", session.ID()), recoverySessionRecordFromDomain(session), ttl)
}

func (r *AuthStateRepository) GetRecoverySession(ctx context.Context, id string) (domain.RecoverySession, error) {
	var record recoverySessionRecord
	if err := r.getJSON(ctx, r.key("auth", "recovery-session", id), &record); err != nil {
		if errors.Is(err, errRESPNil) {
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
		if errors.Is(err, errRESPNil) {
			return domain.NewAuthLock(time.Time{}), false, nil
		}
		return domain.NewAuthLock(time.Time{}), false, domain.ErrAuthStoreUnavailable
	}
	return domain.NewAuthLock(record.LockedUntil), true, nil
}

// SavePasskeyOtp は OTP キーと accountID を Valkey に TTL 付きで保存する。
func (r *AuthStateRepository) SavePasskeyOtp(ctx context.Context, otpKey string, accountID string, ttl time.Duration) error {
	err := r.store.Set(ctx, r.key("auth", "passkey-otp", otpKey), accountID, ttl)
	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// ConsumePasskeyOtp は OTP を取得して削除する（1 回限りの消費）。
// 存在しない・期限切れの場合は domain.ErrOtpNotFound を返す。
func (r *AuthStateRepository) ConsumePasskeyOtp(ctx context.Context, otpKey string) (string, error) {
	k := r.key("auth", "passkey-otp", otpKey)
	accountID, err := r.store.GetDel(ctx, k)
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return "", domain.ErrOtpNotFound
		}
		return "", domain.ErrAuthStoreUnavailable
	}
	if accountID == "" {
		return "", domain.ErrOtpNotFound
	}
	return accountID, nil
}

// GetPasskeyOtp は OTP を消費せずに accountID を取得する。
// 存在しない・期限切れの場合は domain.ErrOtpNotFound を返す。
func (r *AuthStateRepository) GetPasskeyOtp(ctx context.Context, otpKey string) (string, error) {
	accountID, err := r.store.Get(ctx, r.key("auth", "passkey-otp", otpKey))
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return "", domain.ErrOtpNotFound
		}
		return "", domain.ErrAuthStoreUnavailable
	}
	if accountID == "" {
		return "", domain.ErrOtpNotFound
	}
	return accountID, nil
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
		return errRESPNil
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
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
}

type recoveryDeliveryFailureRecord struct {
	RequestID       string    `json:"requestId"`
	RecoveryTokenID string    `json:"recoveryTokenId"`
	AccountID       string    `json:"accountId"`
	Email           string    `json:"email"`
	LastError       string    `json:"lastError"`
	FailedAt        time.Time `json:"failedAt"`
	RetryAfter      time.Time `json:"retryAfter"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

func recoveryDeliveryFailureRecordFromDomain(failure domain.RecoveryDeliveryFailure) recoveryDeliveryFailureRecord {
	return recoveryDeliveryFailureRecord{
		RequestID:       failure.RequestID(),
		RecoveryTokenID: failure.RecoveryTokenID(),
		AccountID:       failure.AccountID(),
		Email:           failure.Email(),
		LastError:       failure.LastError(),
		FailedAt:        failure.FailedAt(),
		RetryAfter:      failure.RetryAfter(),
		ExpiresAt:       failure.ExpiresAt(),
	}
}

func recoveryTokenRecordFromDomain(token domain.RecoveryToken) recoveryTokenRecord {
	return recoveryTokenRecord{ID: token.ID(), AccountID: token.AccountID(), SecretHash: hashSecret(token.Secret()), ExpiresAt: token.ExpiresAt(), ConsumedAt: token.ConsumedAt()}
}

func (r recoveryTokenRecord) toDomain(secret string) (domain.RecoveryToken, error) {
	return domain.ReconstituteRecoveryToken(r.ID, r.AccountID, secret, r.ExpiresAt, r.ConsumedAt)
}

func normalizeRecoveryTokenRecord(record recoveryTokenRecord) (domain.RecoveryToken, error) {
	// secret が必要だが記録にない場合はプレースホルダーを渡す（呼び出し側で上書きすること）。
	token, err := record.toDomain("")
	if err != nil {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	return token, nil
}

// hashSecret は secret の SHA-256 ハッシュを base64 エンコードして返す。
// 本番環境ではペッパーを追加してハッシュすることを推奨する。
func hashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(h[:])
}

type recoverySessionRecord struct {
	ID         string     `json:"id"`
	AccountID  string     `json:"accountId"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
}

func recoverySessionRecordFromDomain(session domain.RecoverySession) recoverySessionRecord {
	return recoverySessionRecord{ID: session.ID(), AccountID: session.AccountID(), ExpiresAt: session.ExpiresAt(), ConsumedAt: session.ConsumedAt()}
}

func (r recoverySessionRecord) toDomain() (domain.RecoverySession, error) {
	return domain.ReconstituteRecoverySession(r.ID, r.AccountID, r.ExpiresAt, r.ConsumedAt)
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
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return token
}

func emptyRecoverySession() domain.RecoverySession {
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
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
	session, err := domain.ReconstituteReauthenticationSession(
		record.ID, record.AccountID, record.IssuingSessionID, record.OperationKind, record.RequestID,
		record.ExpiresAt, record.ConsumedAt,
	)
	if err != nil {
		return emptyReauthenticationSession(), domain.ErrAuthStoreUnavailable
	}
	return session, nil
}

func emptyReauthenticationSession() domain.ReauthenticationSession {
	session, _ := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"otp-issue", "01ARZ3NDEKTSV4RRFFQ69G5FAY", time.Unix(1, 0).UTC(),
	)
	return session
}

// ─── DeviceLoginHandoff ─────────────────────────────────────────────────────

type deviceLoginHandoffRecord struct {
	HandoffID        string     `json:"handoffId"`
	AccountID        string     `json:"accountId"`
	IssuingSessionID string     `json:"issuingSessionId"`
	EmailHash        string     `json:"emailHash"`
	OtpHash          string     `json:"otpHash"`
	ChallengeID      string     `json:"challengeId,omitempty"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	AttemptCount     int        `json:"attemptCount"`
	ConsumedAt       *time.Time `json:"consumedAt,omitempty"`
}

// SaveDeviceLoginHandoff は namespaced device handoff record とその secondary index を TTL 付きで保存する。
// 単一 record は auth:handoff:{handoffID} に、OTP 検索用 secondary index は auth:handoff-otp-idx:{emailHash}:{otpHash} に保存される。
func (r *AuthStateRepository) SaveDeviceLoginHandoff(ctx context.Context, handoff domain.DeviceLoginHandoff, ttl time.Duration) error {
	record := deviceLoginHandoffRecord{
		HandoffID:        handoff.ID(),
		AccountID:        handoff.AccountID(),
		IssuingSessionID: handoff.IssuingSessionID(),
		EmailHash:        handoff.EmailHash(),
		OtpHash:          handoff.OtpHash(),
		ChallengeID:      handoff.ChallengeID(),
		ExpiresAt:        handoff.ExpiresAt(),
		AttemptCount:     handoff.AttemptCount(),
		ConsumedAt:       handoff.ConsumedAt(),
	}
	if err := r.setJSON(ctx, r.key("auth", "handoff", handoff.ID()), record, ttl); err != nil {
		return err
	}
	// secondary index: auth:handoff-otp-idx:{emailHash}:{otpHash} → handoffID
	idxKey := r.key("auth", "handoff-otp-idx", handoff.EmailHash(), handoff.OtpHash())
	if err := r.store.Set(ctx, idxKey, handoff.ID(), ttl); err != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

// FindDeviceLoginHandoffByEmailAndOtp は emailHash と otpHash から secondary index を経由して handoff を検索する。
// 見つからない場合は domain.ErrDeviceLoginHandoffNotFound を返す。
func (r *AuthStateRepository) FindDeviceLoginHandoffByEmailAndOtp(ctx context.Context, emailHash string, otpHash string) (domain.DeviceLoginHandoff, error) {
	idxKey := r.key("auth", "handoff-otp-idx", emailHash, otpHash)
	handoffID, err := r.store.Get(ctx, idxKey)
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return emptyDeviceLoginHandoff(), domain.ErrDeviceLoginHandoffNotFound
		}
		return emptyDeviceLoginHandoff(), domain.ErrAuthStoreUnavailable
	}
	if handoffID == "" {
		return emptyDeviceLoginHandoff(), domain.ErrDeviceLoginHandoffNotFound
	}
	return r.GetDeviceLoginHandoff(ctx, handoffID)
}

// GetDeviceLoginHandoff は handoffID から device login handoff record を取得する。
func (r *AuthStateRepository) GetDeviceLoginHandoff(ctx context.Context, handoffID string) (domain.DeviceLoginHandoff, error) {
	var record deviceLoginHandoffRecord
	if err := r.getJSON(ctx, r.key("auth", "handoff", handoffID), &record); err != nil {
		if errors.Is(err, errRESPNil) {
			return emptyDeviceLoginHandoff(), domain.ErrDeviceLoginHandoffNotFound
		}
		return emptyDeviceLoginHandoff(), domain.ErrAuthStoreUnavailable
	}
	return normalizeDeviceLoginHandoffRecord(record)
}

// ConsumeDeviceLoginHandoff は handoff record を GETDEL でアトミックに取得・削除する。
// 成功した場合、secondary index もベストエフォートで削除する。
func (r *AuthStateRepository) ConsumeDeviceLoginHandoff(ctx context.Context, handoffID string) (domain.DeviceLoginHandoff, error) {
	result, err := r.store.GetDel(ctx, r.key("auth", "handoff", handoffID))
	if err != nil {
		if errors.Is(err, errRESPNil) {
			return emptyDeviceLoginHandoff(), domain.ErrDeviceLoginHandoffNotFound
		}
		return emptyDeviceLoginHandoff(), domain.ErrAuthStoreUnavailable
	}
	var record deviceLoginHandoffRecord
	if err := json.Unmarshal([]byte(result), &record); err != nil {
		return emptyDeviceLoginHandoff(), domain.ErrAuthStoreUnavailable
	}
	// secondary index も削除する（ベストエフォート）
	idxKey := r.key("auth", "handoff-otp-idx", record.EmailHash, record.OtpHash)
	_ = r.store.Delete(ctx, idxKey)
	return normalizeDeviceLoginHandoffRecord(record)
}

func normalizeDeviceLoginHandoffRecord(record deviceLoginHandoffRecord) (domain.DeviceLoginHandoff, error) {
	handoff, err := domain.NewDeviceLoginHandoff(record.HandoffID, record.AccountID, record.IssuingSessionID, record.EmailHash, record.OtpHash, record.ExpiresAt)
	if err != nil {
		return emptyDeviceLoginHandoff(), domain.ErrAuthStoreUnavailable
	}
	if record.ChallengeID != "" {
		handoff = handoff.BindChallenge(record.ChallengeID)
	}
	for i := 0; i < record.AttemptCount; i++ {
		handoff = handoff.IncrementAttempt()
	}
	if record.ConsumedAt != nil {
		handoff = handoff.Consume(*record.ConsumedAt)
	}
	return handoff, nil
}

func emptyDeviceLoginHandoff() domain.DeviceLoginHandoff {
	h, _ := domain.NewDeviceLoginHandoff("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", "placeholder", time.Unix(1, 0).UTC())
	return h
}
