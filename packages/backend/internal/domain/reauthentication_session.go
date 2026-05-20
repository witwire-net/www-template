package domain

import (
	"time"
)

// ReauthenticationSession は高リスク操作の前に要求される短命な WebAuthn 再認証セッション。
// account、issuing session、operation kind、request ID に紐づけられ、TTL または atomic consume で失効する。
type ReauthenticationSession struct {
	id               string
	accountID        AccountID
	issuingSessionID string
	operationKind    string
	requestID        string
	expiresAt        time.Time
	consumedAt       *time.Time
}

// NewReauthenticationSession は再認証セッションを構築する constructor。
func NewReauthenticationSession(id string, accountID AccountID, issuingSessionID string, operationKind string, requestID string, expiresAt time.Time) (ReauthenticationSession, error) {
	if err := ValidateAuthID(id); err != nil {
		return ReauthenticationSession{}, ErrInvalidAuthID
	}
	if _, err := NewAccountID(accountID.String()); err != nil {
		return ReauthenticationSession{}, ErrInvalidAccountID
	}
	if err := ValidateAuthID(issuingSessionID); err != nil {
		return ReauthenticationSession{}, ErrInvalidAuthID
	}
	if operationKind == "" {
		return ReauthenticationSession{}, ErrInvalidChallenge
	}
	if err := ValidateAuthID(requestID); err != nil {
		return ReauthenticationSession{}, ErrInvalidAuthID
	}
	if expiresAt.IsZero() {
		return ReauthenticationSession{}, ErrInvalidSessionExpiry
	}
	return ReauthenticationSession{
		id:               id,
		accountID:        accountID,
		issuingSessionID: issuingSessionID,
		operationKind:    operationKind,
		requestID:        requestID,
		expiresAt:        expiresAt,
	}, nil
}

// ReconstituteReauthenticationSession は永続化レコードから再認証セッションを復元する。
func ReconstituteReauthenticationSession(id string, accountID AccountID, issuingSessionID string, operationKind string, requestID string, expiresAt time.Time, consumedAt *time.Time) (ReauthenticationSession, error) {
	session, err := NewReauthenticationSession(id, accountID, issuingSessionID, operationKind, requestID, expiresAt)
	if err != nil {
		return ReauthenticationSession{}, err
	}
	session.consumedAt = consumedAt
	return session, nil
}

// EnsureAvailable はセッションが有効で未消費であることを確認する。
func (s ReauthenticationSession) EnsureAvailable(now time.Time) error {
	if s.consumedAt != nil {
		return ErrReauthSessionConsumed
	}
	if now.After(s.expiresAt) {
		return ErrReauthSessionExpired
	}
	return nil
}

// Consume はセッションを消費済みとしてマークする。
func (s ReauthenticationSession) Consume(at time.Time) ReauthenticationSession {
	consumedAt := at.UTC()
	s.consumedAt = &consumedAt
	return s
}

// ID はセッション ID を返す。
func (s ReauthenticationSession) ID() string { return s.id }

// AccountID は紐づくアカウント ID を返す。
func (s ReauthenticationSession) AccountID() AccountID { return s.accountID }

// IssuingSessionID は発行元セッション ID を返す。
func (s ReauthenticationSession) IssuingSessionID() string { return s.issuingSessionID }

// OperationKind は操作種別（例: "delete_passkey"）を返す。
func (s ReauthenticationSession) OperationKind() string { return s.operationKind }

// RequestID は発行時のリクエスト ID を返す。
func (s ReauthenticationSession) RequestID() string { return s.requestID }

// ExpiresAt はセッションの有効期限を返す。
func (s ReauthenticationSession) ExpiresAt() time.Time { return s.expiresAt }

// ConsumedAt は消費日時を返す。未消費の場合は nil。
func (s ReauthenticationSession) ConsumedAt() *time.Time { return s.consumedAt }
