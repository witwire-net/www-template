package usecases

import (
	"context"
	"time"

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/types"
)

type AuthSession struct {
	RequestID           string
	AccountID           string
	PasskeyCredentialID string
	SessionID           string
	SessionToken        string
	ExpiresAt           time.Time
}

type PasskeyChallenge struct {
	RequestID    string
	Challenge    string
	ChallengeID  string
	WebAuthnRPID string
}

type RecoveryAccepted struct {
	RequestID string
	Accepted  bool
}

type RecoverySession struct {
	RequestID          string
	RecoveryTokenID    string
	RecoverySessionID  string
	RecoverySessionRef string
	ExpiresAt          time.Time
}

type RecoveryDelivery struct {
	RequestID       string
	RecoveryTokenID string
	AccountID       string
	Email           string
	RecoveryURL     string
	ExpiresAt       time.Time
}

type StartPasskeyAuthenticationInput struct {
	Identifier string
	ClientIP   string
}

type FinishPasskeyAuthenticationInput struct {
	Credential string
	ClientIP   string
}

type RequestPasskeyRecoveryInput struct {
	Email    string
	ClientIP string
}

type ConsumeRecoveryTokenInput struct {
	Token    string
	ClientIP string
}

type RegisterPasskeyInput struct {
	RecoverySession   string
	InvitationSession string
	Credential        string
	ClientIP          string
}

type InvitationPasskeyRegistrationInput struct {
	InvitationSession string
	Credential        string
	ClientIP          string
}

type AuthStateRepository interface {
	SaveChallenge(context.Context, domain.AuthChallenge, time.Duration) error
	ConsumeChallenge(context.Context, string) (domain.AuthChallenge, error)
	SaveSession(context.Context, domain.Session, time.Duration) error
	RefreshSession(context.Context, domain.Session, time.Duration) error
	GetSessionByToken(context.Context, string) (domain.Session, error)
	RevokeSession(context.Context, domain.Session, time.Duration) error
	IssueRecoveryToken(context.Context, domain.RecoveryToken, time.Duration) error
	SaveRecoveryDeliveryFailure(context.Context, domain.RecoveryDeliveryFailure, time.Duration) error
	GetRecoveryTokenBySecret(context.Context, string) (domain.RecoveryToken, error)
	ConsumeRecoveryToken(context.Context, domain.RecoveryToken) error
	SaveRecoverySession(context.Context, domain.RecoverySession, time.Duration) error
	GetRecoverySession(context.Context, string) (domain.RecoverySession, error)
	ConsumeRecoverySession(context.Context, domain.RecoverySession) error
	IncrementThrottle(context.Context, string, time.Duration) (int, error)
	SetLock(context.Context, string, time.Time, time.Duration) error
	GetLock(context.Context, string) (domain.AuthLock, bool, error)
}

type AuthAccountRepository interface {
	FindByIdentifier(context.Context, string) (domain.AuthAccount, error)
	FindByCredential(context.Context, string) (domain.AuthAccount, error)
	FindByEmail(context.Context, string) (domain.AuthAccount, error)
	ReplacePasskey(context.Context, string, string, string) (domain.AuthAccount, error)
}

type AccountRecoverySender interface {
	SendAccountRecovery(context.Context, RecoveryDelivery) error
}

type InvitationPasskeyRegistrar interface {
	RegisterInvitationPasskey(context.Context, InvitationPasskeyRegistrationInput) (AuthSession, error)
}

type AuthService struct {
	stateRepo           AuthStateRepository
	accountRepo         AuthAccountRepository
	recoverySender      AccountRecoverySender
	invitationRegistrar InvitationPasskeyRegistrar
	clock               func() time.Time
	policy              types.AuthIDPolicy
	authConfig          types.AuthConfig
}

func NewAuthService(stateRepo AuthStateRepository, accountRepo AuthAccountRepository, recoverySender AccountRecoverySender, invitationRegistrar InvitationPasskeyRegistrar, clock func() time.Time, policy types.AuthIDPolicy, authConfig types.AuthConfig) *AuthService {
	if clock == nil {
		panic("clock is required")
	}

	return &AuthService{
		stateRepo:           stateRepo,
		accountRepo:         accountRepo,
		recoverySender:      recoverySender,
		invitationRegistrar: invitationRegistrar,
		clock:               clock,
		policy:              policy,
		authConfig:          authConfig,
	}
}
