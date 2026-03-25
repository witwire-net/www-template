package persistence

import (
	"context"
	"sync"
	"time"

	"www-template/packages/backend/internal/domain"
)

type InMemoryStateRepository struct {
	mu               sync.RWMutex
	challenges       map[string]domain.AuthChallenge
	sessions         map[string]domain.Session
	recoveryTokens   map[string]domain.RecoveryToken
	recoveryFailures map[string]domain.RecoveryDeliveryFailure
	recoverySessions map[string]domain.RecoverySession
	counters         map[string]counterRecord
	locks            map[string]time.Time
	clock            func() time.Time
}

type counterRecord struct {
	count     int
	expiresAt time.Time
}

func newEmptyChallenge() domain.AuthChallenge {
	challenge, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return challenge
}

func newEmptySession() domain.Session {
	session, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return session
}

func newEmptyRecoveryToken() domain.RecoveryToken {
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return token
}

func newEmptyRecoverySession() domain.RecoverySession {
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
	return session
}

func NewInMemoryStateRepository(clock func() time.Time) *InMemoryStateRepository {
	return &InMemoryStateRepository{
		challenges:       map[string]domain.AuthChallenge{},
		sessions:         map[string]domain.Session{},
		recoveryTokens:   map[string]domain.RecoveryToken{},
		recoveryFailures: map[string]domain.RecoveryDeliveryFailure{},
		recoverySessions: map[string]domain.RecoverySession{},
		counters:         map[string]counterRecord{},
		locks:            map[string]time.Time{},
		clock:            clock,
	}
}

func (r *InMemoryStateRepository) SaveChallenge(_ context.Context, challenge domain.AuthChallenge, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.challenges[challenge.Challenge()] = challenge
	return nil
}

func (r *InMemoryStateRepository) ConsumeChallenge(_ context.Context, secret string) (domain.AuthChallenge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	challenge, ok := r.challenges[secret]
	if !ok {
		return newEmptyChallenge(), domain.ErrChallengeNotFound
	}
	delete(r.challenges, secret)
	return challenge, nil
}

func (r *InMemoryStateRepository) SaveSession(_ context.Context, session domain.Session, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.Token()] = session
	return nil
}

func (r *InMemoryStateRepository) RefreshSession(_ context.Context, session domain.Session, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.Token()] = session
	return nil
}

func (r *InMemoryStateRepository) GetSessionByToken(_ context.Context, token string) (domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[token]
	if !ok {
		return newEmptySession(), domain.ErrSessionNotFound
	}
	return session, nil
}

func (r *InMemoryStateRepository) RevokeSession(_ context.Context, session domain.Session, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.Token()] = session
	return nil
}

func (r *InMemoryStateRepository) IssueRecoveryToken(_ context.Context, token domain.RecoveryToken, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryTokens[token.Secret()] = token
	return nil
}

func (r *InMemoryStateRepository) SaveRecoveryDeliveryFailure(_ context.Context, failure domain.RecoveryDeliveryFailure, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryFailures[failure.RequestID()] = failure
	return nil
}

func (r *InMemoryStateRepository) GetRecoveryTokenBySecret(_ context.Context, secret string) (domain.RecoveryToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token, ok := r.recoveryTokens[secret]
	if !ok {
		return newEmptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
	}
	return token, nil
}

func (r *InMemoryStateRepository) ConsumeRecoveryToken(_ context.Context, token domain.RecoveryToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryTokens[token.Secret()] = token
	return nil
}

func (r *InMemoryStateRepository) SaveRecoverySession(_ context.Context, session domain.RecoverySession, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoverySessions[session.ID()] = session
	return nil
}

func (r *InMemoryStateRepository) GetRecoverySession(_ context.Context, id string) (domain.RecoverySession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.recoverySessions[id]
	if !ok {
		return newEmptyRecoverySession(), domain.ErrRecoverySessionNotFound
	}
	return session, nil
}

func (r *InMemoryStateRepository) ConsumeRecoverySession(_ context.Context, session domain.RecoverySession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoverySessions[session.ID()] = session
	return nil
}

func (r *InMemoryStateRepository) IncrementThrottle(_ context.Context, key string, ttl time.Duration) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.clock().UTC()
	record, ok := r.counters[key]
	if !ok || now.After(record.expiresAt) {
		record = counterRecord{count: 0, expiresAt: now.Add(ttl)}
	}
	record.count++
	r.counters[key] = record
	return record.count, nil
}

func (r *InMemoryStateRepository) SetLock(_ context.Context, key string, until time.Time, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.locks[key] = until.UTC()
	return nil
}

func (r *InMemoryStateRepository) GetLock(_ context.Context, key string) (domain.AuthLock, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	until, ok := r.locks[key]
	if !ok {
		return domain.NewAuthLock(time.Time{}), false, nil
	}
	return domain.NewAuthLock(until), true, nil
}
