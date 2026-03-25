package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"www-template/packages/backend/internal/domain"
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

func (r *AuthStateRepository) SaveChallenge(ctx context.Context, challenge domain.AuthChallenge, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "challenge", challenge.Challenge()), challengeRecordFromDomain(challenge), ttl)
}

func (r *AuthStateRepository) ConsumeChallenge(ctx context.Context, secret string) (domain.AuthChallenge, error) {
	challenge, err := r.getChallenge(ctx, secret)
	if err != nil {
		return emptyChallenge(), err
	}
	if err := r.store.Delete(ctx, r.key("auth", "challenge", secret)); err != nil {
		return emptyChallenge(), domain.ErrAuthStoreUnavailable
	}
	return challenge, nil
}

func (r *AuthStateRepository) SaveSession(ctx context.Context, session domain.Session, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "session", session.Token()), sessionRecordFromDomain(session), ttl)
}

func (r *AuthStateRepository) RefreshSession(ctx context.Context, session domain.Session, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "session", session.Token()), sessionRecordFromDomain(session), ttl)
}

func (r *AuthStateRepository) GetSessionByToken(ctx context.Context, token string) (domain.Session, error) {
	var record sessionRecord
	if err := r.getJSON(ctx, r.key("auth", "session", token), &record); err != nil {
		if errors.Is(err, errRESPNil) {
			return emptySession(), domain.ErrSessionNotFound
		}
		return emptySession(), domain.ErrAuthStoreUnavailable
	}
	return normalizeSessionRecord(record)
}

func (r *AuthStateRepository) RevokeSession(ctx context.Context, session domain.Session, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "session", session.Token()), sessionRecordFromDomain(session), ttl)
}

func (r *AuthStateRepository) IssueRecoveryToken(ctx context.Context, token domain.RecoveryToken, ttl time.Duration) error {
	return r.setJSON(ctx, r.key("auth", "recovery-token", token.Secret()), recoveryTokenRecordFromDomain(token), ttl)
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

func (r *AuthStateRepository) getChallenge(ctx context.Context, secret string) (domain.AuthChallenge, error) {
	var record challengeRecord
	if err := r.getJSON(ctx, r.key("auth", "challenge", secret), &record); err != nil {
		if errors.Is(err, errRESPNil) {
			return emptyChallenge(), domain.ErrChallengeNotFound
		}
		return emptyChallenge(), domain.ErrAuthStoreUnavailable
	}
	return normalizeChallengeRecord(record)
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

type sessionRecord struct {
	ID                  string     `json:"id"`
	AccountID           string     `json:"accountId"`
	PasskeyCredentialID string     `json:"passkeyCredentialId"`
	Token               string     `json:"token"`
	IdleExpiresAt       time.Time  `json:"idleExpiresAt"`
	AbsoluteExpiresAt   time.Time  `json:"absoluteExpiresAt"`
	RevokedAt           *time.Time `json:"revokedAt,omitempty"`
}

func sessionRecordFromDomain(session domain.Session) sessionRecord {
	return sessionRecord{ID: session.ID(), AccountID: session.AccountID(), PasskeyCredentialID: session.PasskeyCredentialID(), Token: session.Token(), IdleExpiresAt: session.IdleExpiresAt(), AbsoluteExpiresAt: session.AbsoluteExpiresAt(), RevokedAt: session.RevokedAt()}
}

func (r sessionRecord) toDomain() (domain.Session, error) {
	return domain.ReconstituteSession(r.ID, r.AccountID, r.PasskeyCredentialID, r.Token, r.IdleExpiresAt, r.AbsoluteExpiresAt, r.RevokedAt)
}

func normalizeSessionRecord(record sessionRecord) (domain.Session, error) {
	session, err := record.toDomain()
	if err != nil {
		return emptySession(), domain.ErrAuthStoreUnavailable
	}
	return session, nil
}

type recoveryTokenRecord struct {
	ID         string     `json:"id"`
	AccountID  string     `json:"accountId"`
	Secret     string     `json:"secret"`
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
	return recoveryTokenRecord{ID: token.ID(), AccountID: token.AccountID(), Secret: token.Secret(), ExpiresAt: token.ExpiresAt(), ConsumedAt: token.ConsumedAt()}
}

func (r recoveryTokenRecord) toDomain() (domain.RecoveryToken, error) {
	return domain.ReconstituteRecoveryToken(r.ID, r.AccountID, r.Secret, r.ExpiresAt, r.ConsumedAt)
}

func normalizeRecoveryTokenRecord(record recoveryTokenRecord) (domain.RecoveryToken, error) {
	token, err := record.toDomain()
	if err != nil {
		return emptyRecoveryToken(), domain.ErrAuthStoreUnavailable
	}
	return token, nil
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

func emptySession() domain.Session {
	session, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return session
}

func emptyRecoveryToken() domain.RecoveryToken {
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return token
}

func emptyRecoverySession() domain.RecoverySession {
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
	return session
}
