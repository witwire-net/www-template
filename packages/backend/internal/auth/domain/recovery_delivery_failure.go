package domain

import (
	"strings"
	"time"
)

type RecoveryDeliveryFailure struct {
	requestID       string
	recoveryTokenID string
	accountID       string
	email           string
	lastError       string
	failedAt        time.Time
	retryAfter      time.Time
	expiresAt       time.Time
}

func NewRecoveryDeliveryFailure(requestID string, recoveryTokenID string, accountID string, email string, lastError string, failedAt time.Time, retryAfter time.Time, expiresAt time.Time) (RecoveryDeliveryFailure, error) {
	if err := ValidateAuthID(requestID); err != nil {
		return RecoveryDeliveryFailure{}, err
	}
	if err := ValidateAuthID(recoveryTokenID); err != nil {
		return RecoveryDeliveryFailure{}, err
	}
	if err := ValidateAuthID(accountID); err != nil {
		return RecoveryDeliveryFailure{}, err
	}
	if strings.TrimSpace(email) == "" {
		return RecoveryDeliveryFailure{}, ErrInvalidOpaqueSecret
	}
	if strings.TrimSpace(lastError) == "" {
		return RecoveryDeliveryFailure{}, ErrInvalidOpaqueSecret
	}
	if failedAt.IsZero() || retryAfter.IsZero() || expiresAt.IsZero() || retryAfter.Before(failedAt) || expiresAt.Before(retryAfter) {
		return RecoveryDeliveryFailure{}, ErrInvalidChallenge
	}

	return RecoveryDeliveryFailure{
		requestID:       requestID,
		recoveryTokenID: recoveryTokenID,
		accountID:       accountID,
		email:           strings.TrimSpace(email),
		lastError:       strings.TrimSpace(lastError),
		failedAt:        failedAt.UTC(),
		retryAfter:      retryAfter.UTC(),
		expiresAt:       expiresAt.UTC(),
	}, nil
}

func (r RecoveryDeliveryFailure) RequestID() string { return r.requestID }

func (r RecoveryDeliveryFailure) RecoveryTokenID() string { return r.recoveryTokenID }

func (r RecoveryDeliveryFailure) AccountID() string { return r.accountID }

func (r RecoveryDeliveryFailure) Email() string { return r.email }

func (r RecoveryDeliveryFailure) LastError() string { return r.lastError }

func (r RecoveryDeliveryFailure) FailedAt() time.Time { return r.failedAt }

func (r RecoveryDeliveryFailure) RetryAfter() time.Time { return r.retryAfter }

func (r RecoveryDeliveryFailure) ExpiresAt() time.Time { return r.expiresAt }
