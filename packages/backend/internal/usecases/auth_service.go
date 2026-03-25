package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"www-template/packages/backend/internal/domain"
)

func (s *AuthService) StartPasskeyAuthentication(ctx context.Context, input StartPasskeyAuthenticationInput) (PasskeyChallenge, error) {
	if err := s.ensureNotLocked(ctx, failureLockKey(input.Identifier, input.ClientIP)); err != nil {
		return PasskeyChallenge{}, err
	}

	count, err := s.stateRepo.IncrementThrottle(ctx, passkeyStartKey(input.Identifier, input.ClientIP), s.authConfig.PasskeyStartThrottleWindow)
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	if count > s.authConfig.PasskeyStartThrottleLimit {
		return PasskeyChallenge{}, ErrBadRequest
	}

	requestID, challengeID, err := s.nextTwoIDs()
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	challengeValue := opaqueValue(challengeID)
	challenge, err := domain.NewAuthChallenge(challengeID, strings.TrimSpace(input.Identifier), challengeValue, s.clock().Add(s.authConfig.ChallengeTTL))
	if err != nil {
		return PasskeyChallenge{}, ErrBadRequest
	}
	if err := s.stateRepo.SaveChallenge(ctx, challenge, s.authConfig.ChallengeTTL); err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}

	return PasskeyChallenge{RequestID: requestID, Challenge: challengeValue, ChallengeID: challengeID, WebAuthnRPID: s.authConfig.WebAuthnRPID}, nil
}

func (s *AuthService) FinishPasskeyAuthentication(ctx context.Context, input FinishPasskeyAuthenticationInput) (AuthSession, error) {
	credentialHandle, challengeValue, ok := splitCredentialEnvelope(input.Credential)
	if !ok {
		return AuthSession{}, ErrBadRequest
	}

	lockKey := failureLockKey(credentialHandle, input.ClientIP)
	if err := s.ensureNotLocked(ctx, lockKey); err != nil {
		return AuthSession{}, err
	}

	account, err := s.accountRepo.FindByCredential(ctx, credentialHandle)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return AuthSession{}, ErrInternalError
		}
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	challenge, err := s.stateRepo.ConsumeChallenge(ctx, challengeValue)
	if err != nil {
		mappedErr := s.mapAuthStoreError(err)
		if !errors.Is(mappedErr, ErrInternalError) {
			s.registerFailure(ctx, lockKey)
		}
		return AuthSession{}, mappedErr
	}
	if err := challenge.EnsureAvailable(s.clock()); err != nil || account.Identifier() != challenge.Identifier() {
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	session, err := s.issueSession(account)
	if err != nil {
		return AuthSession{}, ErrInternalError
	}
	if err := s.stateRepo.SaveSession(ctx, session, s.authConfig.SessionAbsoluteTTL); err != nil {
		return AuthSession{}, ErrInternalError
	}

	requestID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}

	return toAuthSession(requestID, session), nil
}

func (s *AuthService) RequestPasskeyRecovery(ctx context.Context, input RequestPasskeyRecoveryInput) (RecoveryAccepted, error) {
	requestID, err := s.policy.Next()
	if err != nil {
		return RecoveryAccepted{}, ErrInternalError
	}

	emailKey := recoveryEmailKey(input.Email)
	ipKey := recoveryIPKey(input.ClientIP)
	emailCount, err := s.stateRepo.IncrementThrottle(ctx, emailKey, s.authConfig.RecoveryEmailThrottleWindow)
	if err != nil {
		return RecoveryAccepted{}, ErrInternalError
	}
	ipCount, err := s.stateRepo.IncrementThrottle(ctx, ipKey, s.authConfig.RecoveryIPThrottleWindow)
	if err != nil {
		return RecoveryAccepted{}, ErrInternalError
	}
	if emailCount > s.authConfig.RecoveryEmailThrottleLimit || ipCount > s.authConfig.RecoveryIPThrottleLimit {
		return RecoveryAccepted{RequestID: requestID, Accepted: true}, nil
	}

	account, err := s.accountRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return RecoveryAccepted{}, ErrInternalError
		}
		return RecoveryAccepted{RequestID: requestID, Accepted: true}, nil
	}

	delivery, err := s.issueRecoveryDelivery(ctx, requestID, account)
	if err != nil {
		return RecoveryAccepted{}, err
	}
	if s.recoverySender != nil {
		if err := s.recoverySender.SendAccountRecovery(ctx, delivery); err != nil {
			s.recordRecoveryDeliveryFailure(ctx, delivery, err)
			return RecoveryAccepted{RequestID: requestID, Accepted: true}, nil
		}
	}

	return RecoveryAccepted{RequestID: requestID, Accepted: true}, nil
}

func (s *AuthService) ConsumeRecoveryToken(ctx context.Context, input ConsumeRecoveryTokenInput) (RecoverySession, error) {
	lockKey := failureLockKey(input.Token, input.ClientIP)
	if err := s.ensureNotLocked(ctx, lockKey); err != nil {
		return RecoverySession{}, err
	}

	recoveryToken, err := s.stateRepo.GetRecoveryTokenBySecret(ctx, input.Token)
	if err != nil {
		s.registerFailure(ctx, lockKey)
		return RecoverySession{}, s.mapRecoveryConsumeError(err)
	}
	if err := recoveryToken.EnsureConsumable(s.clock()); err != nil {
		s.registerFailure(ctx, lockKey)
		return RecoverySession{}, ErrBadRequest
	}

	if err := s.stateRepo.ConsumeRecoveryToken(ctx, recoveryToken.Consume(s.clock())); err != nil {
		return RecoverySession{}, ErrInternalError
	}

	requestID, recoverySessionID, err := s.nextTwoIDs()
	if err != nil {
		return RecoverySession{}, ErrInternalError
	}
	recoverySession, err := domain.NewRecoverySession(recoverySessionID, recoveryToken.AccountID(), s.clock().Add(s.authConfig.RecoverySessionTTL))
	if err != nil {
		return RecoverySession{}, ErrInternalError
	}
	if err := s.stateRepo.SaveRecoverySession(ctx, recoverySession, s.authConfig.RecoverySessionTTL); err != nil {
		return RecoverySession{}, ErrInternalError
	}

	return RecoverySession{
		RequestID:          requestID,
		RecoveryTokenID:    recoveryToken.ID(),
		RecoverySessionID:  recoverySession.ID(),
		RecoverySessionRef: recoverySession.ID(),
		ExpiresAt:          recoverySession.ExpiresAt(),
	}, nil
}

func (s *AuthService) RegisterPasskey(ctx context.Context, input RegisterPasskeyInput) (AuthSession, error) {
	lockKey := failureLockKey(input.Credential, input.ClientIP)
	if err := s.ensureNotLocked(ctx, lockKey); err != nil {
		return AuthSession{}, err
	}

	if selectorCount(input.RecoverySession, input.InvitationSession) != 1 {
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	if strings.TrimSpace(input.InvitationSession) != "" {
		return s.registerInvitationPasskey(ctx, input)
	}

	recoverySession, err := s.stateRepo.GetRecoverySession(ctx, input.RecoverySession)
	if err != nil {
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, s.mapRecoveryConsumeError(err)
	}
	if err := recoverySession.EnsureAvailable(s.clock()); err != nil {
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	passkeyID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}
	account, err := s.accountRepo.ReplacePasskey(ctx, recoverySession.AccountID(), passkeyID, input.Credential)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return AuthSession{}, ErrInternalError
		}
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	if err := s.stateRepo.ConsumeRecoverySession(ctx, recoverySession.Consume(s.clock())); err != nil {
		return AuthSession{}, ErrInternalError
	}

	session, err := s.issueSession(account)
	if err != nil {
		return AuthSession{}, ErrInternalError
	}
	if err := s.stateRepo.SaveSession(ctx, session, s.authConfig.SessionAbsoluteTTL); err != nil {
		return AuthSession{}, ErrInternalError
	}

	requestID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}

	return toAuthSession(requestID, session), nil
}

func (s *AuthService) registerInvitationPasskey(ctx context.Context, input RegisterPasskeyInput) (AuthSession, error) {
	if s.invitationRegistrar == nil {
		return AuthSession{}, ErrBadRequest
	}

	result, err := s.invitationRegistrar.RegisterInvitationPasskey(ctx, InvitationPasskeyRegistrationInput{
		InvitationSession: input.InvitationSession,
		Credential:        input.Credential,
		ClientIP:          input.ClientIP,
	})
	if err != nil {
		return AuthSession{}, err
	}

	return result, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) (string, error) {
	session, err := s.stateRepo.GetSessionByToken(ctx, token)
	if err != nil {
		return "", s.mapSessionError(err)
	}
	if err := session.EnsureActive(s.clock()); err != nil {
		return "", s.mapSessionError(err)
	}
	requestID, err := s.policy.Next()
	if err != nil {
		return "", ErrInternalError
	}
	if err := s.stateRepo.RevokeSession(ctx, session.Revoke(s.clock()), session.RevocationTTL(s.clock())); err != nil {
		return "", ErrInternalError
	}

	return requestID, nil
}

func (s *AuthService) AuthorizeSession(ctx context.Context, token string) (AuthSession, error) {
	if strings.TrimSpace(token) == "" {
		return AuthSession{}, ErrUnauthenticated
	}
	session, err := s.stateRepo.GetSessionByToken(ctx, token)
	if err != nil {
		return AuthSession{}, s.mapSessionError(err)
	}
	if err := session.EnsureActive(s.clock()); err != nil {
		return AuthSession{}, s.mapSessionError(err)
	}
	refreshed := session.RefreshIdle(s.clock(), s.authConfig.SessionIdleTTL)
	if err := s.stateRepo.RefreshSession(ctx, refreshed, refreshed.RevocationTTL(s.clock())); err != nil {
		return AuthSession{}, ErrInternalError
	}
	requestID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}

	return toAuthSession(requestID, refreshed), nil
}

func (s *AuthService) issueRecoveryDelivery(ctx context.Context, requestID string, account domain.AuthAccount) (RecoveryDelivery, error) {
	tokenID, err := s.policy.Next()
	if err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}
	secret := opaqueValue(tokenID)
	recoveryToken, err := domain.NewRecoveryToken(tokenID, account.AccountID(), secret, s.clock().Add(s.authConfig.RecoveryTokenTTL))
	if err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}
	if err := s.stateRepo.IssueRecoveryToken(ctx, recoveryToken, s.authConfig.RecoveryTokenTTL); err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}

	return RecoveryDelivery{
		RequestID:       requestID,
		RecoveryTokenID: tokenID,
		AccountID:       account.AccountID(),
		Email:           account.Email(),
		RecoveryURL:     fmt.Sprintf("%s?token=%s", strings.TrimSpace(s.authConfig.AccountRecoveryURLBase), secret),
		ExpiresAt:       recoveryToken.ExpiresAt(),
	}, nil
}

func (s *AuthService) issueSession(account domain.AuthAccount) (domain.Session, error) {
	sessionID, err := s.policy.Next()
	if err != nil {
		return emptyUsecaseSession(), err
	}
	token := opaqueValue(sessionID)
	now := s.clock()
	return domain.NewSession(sessionID, account.AccountID(), account.PasskeyCredentialID(), token, now.Add(s.authConfig.SessionIdleTTL), now.Add(s.authConfig.SessionAbsoluteTTL))
}

func (s *AuthService) recordRecoveryDeliveryFailure(ctx context.Context, delivery RecoveryDelivery, sendErr error) {
	failedAt := s.clock()
	ttl := delivery.ExpiresAt.Sub(failedAt)
	if ttl <= 0 {
		return
	}
	failure, err := domain.NewRecoveryDeliveryFailure(
		delivery.RequestID,
		delivery.RecoveryTokenID,
		delivery.AccountID,
		delivery.Email,
		sendErr.Error(),
		failedAt,
		failedAt,
		delivery.ExpiresAt,
	)
	if err != nil {
		return
	}
	_ = s.stateRepo.SaveRecoveryDeliveryFailure(ctx, failure, ttl)
}

func (s *AuthService) ensureNotLocked(ctx context.Context, key string) error {
	lock, ok, err := s.stateRepo.GetLock(ctx, key)
	if err != nil {
		return ErrInternalError
	}
	if ok && lock.Active(s.clock()) {
		return ErrBadRequest
	}

	return nil
}

func (s *AuthService) registerFailure(ctx context.Context, key string) {
	count, err := s.stateRepo.IncrementThrottle(ctx, failureWindowKey(key), s.authConfig.FailureLockWindow)
	if err != nil {
		return
	}
	if count >= s.authConfig.FailureLockThreshold {
		_ = s.stateRepo.SetLock(ctx, key, s.clock().Add(s.authConfig.FailureLockDuration), s.authConfig.FailureLockDuration)
	}
}

func (s *AuthService) nextTwoIDs() (string, string, error) {
	first, err := s.policy.Next()
	if err != nil {
		return "", "", err
	}
	second, err := s.policy.Next()
	if err != nil {
		return "", "", err
	}
	return first, second, nil
}

func emptyUsecaseSession() domain.Session {
	session, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return session
}

func splitCredentialEnvelope(value string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(value), "::", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}
