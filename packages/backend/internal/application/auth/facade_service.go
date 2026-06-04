package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "www-template/packages/backend/internal/domain"
)

// StartPasskeyAuthentication は公開認証セレモニーを開始する。
// identifier-based throttle は完全に廃止し、IP bucket と global bucket のみを適用する。
// これにより identifier rotation で challenge issuance budget を回避できないようにする。
func (s *AuthService) StartPasskeyAuthentication(ctx context.Context, input StartPasskeyAuthenticationInput) (PasskeyChallenge, error) {
	lockKey := failureLockKey(input.Identifier, input.ClientIP)
	if err := s.ensureNotLocked(ctx, lockKey); err != nil {
		return PasskeyChallenge{}, err
	}

	ipKey := "passkey-start:ip:" + strings.TrimSpace(input.ClientIP)
	globalKey := "passkey-start:global"

	ipCount, err := s.stateRepo.IncrementThrottle(ctx, ipKey, s.authConfig.PasskeyStartThrottleWindow)
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	globalCount, err := s.stateRepo.IncrementThrottle(ctx, globalKey, s.authConfig.PasskeyStartThrottleWindow)
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	if ipCount > s.authConfig.PasskeyStartThrottleLimit || globalCount > s.authConfig.PasskeyStartGlobalThrottleLimit {
		return PasskeyChallenge{}, ErrBadRequest
	}

	if s.webauthn == nil {
		return PasskeyChallenge{}, ErrInternalError
	}

	challengeKey, optionsJSON, beginErr := s.webauthn.BeginLogin(ctx, strings.TrimSpace(input.Identifier))
	if beginErr != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	requestID, err := s.policy.Next()
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	return PasskeyChallenge{
		RequestID:       requestID,
		Challenge:       challengeKey,
		ChallengeID:     challengeKey,
		WebAuthnRPID:    s.authConfig.WebAuthnRPID,
		WebAuthnOptions: optionsJSON,
	}, nil
}

func (s *AuthService) FinishPasskeyAuthentication(ctx context.Context, input FinishPasskeyAuthenticationInput) (AuthSession, error) {
	return s.finishPasskeyAuthenticationWebAuthn(ctx, input)
}

// VerifyReauthSession は reauth session をアトミックに consume し、
// 現在の account/session と operation kind が一致することを検証する。
// reauth session が存在しない、期限切れ、または消費済みの場合は ErrBadRequest を返す。
func (s *AuthService) VerifyReauthSession(ctx context.Context, reauthID string, accountID domain.AccountID, sessionID string, operationKind string) error {
	if strings.TrimSpace(reauthID) == "" {
		return ErrBadRequest
	}
	reauthSession, err := s.stateRepo.ConsumeReauthenticationSession(ctx, reauthID)
	if err != nil {
		if errors.Is(err, domain.ErrReauthSessionNotFound) || errors.Is(err, domain.ErrReauthSessionExpired) || errors.Is(err, domain.ErrReauthSessionConsumed) {
			return ErrBadRequest
		}
		return ErrInternalError
	}
	if err := reauthSession.EnsureAvailable(s.clock()); err != nil {
		return ErrBadRequest
	}
	if reauthSession.AccountID() != accountID {
		return ErrBadRequest
	}
	if reauthSession.IssuingSessionID() != sessionID {
		return ErrBadRequest
	}
	if reauthSession.OperationKind() != operationKind {
		return ErrBadRequest
	}
	return nil
}

// ExecuteDeviceLink は device-link URL を発行し、登録済みメールアドレスへ送信する。
// bearer token から得た account/session と reauth session（kind="device-link"）を検証した上で実行する。
// メール送信は fire-and-forget とし、送信失敗はログ記録のみとする。
func (s *AuthService) ExecuteDeviceLink(ctx context.Context, accountID domain.AccountID, sessionID string) (DeviceLinkIssued, error) {
	requestID, err := s.policy.Next()
	if err != nil {
		return DeviceLinkIssued{}, ErrInternalError
	}

	account, err := s.accountRepo.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return DeviceLinkIssued{}, ErrInternalError
		}
		return DeviceLinkIssued{}, ErrBadRequest
	}

	delivery, err := s.issueRecoveryDelivery(ctx, requestID, account.AccountID(), account.Email(), domain.TokenKindDeviceLink)
	if err != nil {
		return DeviceLinkIssued{}, err
	}

	// device-link メールを fire-and-forget で送信する。失敗しても issued=true を返す。
	if s.deviceLinkSender != nil {
		if err := s.deviceLinkSender.SendDeviceLink(ctx, delivery); err != nil {
			if s.auditNotifier != nil {
				s.auditNotifier.EmitDeviceLinkDeliveryFailure(ctx, requestID, accountID, err)
			}
		}
	}

	return DeviceLinkIssued{RequestID: requestID, Issued: true}, nil
}

// StartReauthentication は高リスク操作に先立って WebAuthn 再認証セレモニーを開始する。
// 発行された challenge は provider 内部（または Valkey）に TTL 付きで保存される。
func (s *AuthService) StartReauthentication(ctx context.Context, input StartReauthenticationInput) (PasskeyChallenge, error) {
	if s.webauthn == nil {
		return PasskeyChallenge{}, ErrInternalError
	}

	challengeKey, optionsJSON, beginErr := s.webauthn.BeginLogin(ctx, "")
	if beginErr != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	requestID, err := s.policy.Next()
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	return PasskeyChallenge{
		RequestID:       requestID,
		Challenge:       challengeKey,
		ChallengeID:     challengeKey,
		WebAuthnRPID:    s.authConfig.WebAuthnRPID,
		WebAuthnOptions: optionsJSON,
	}, nil
}

// FinishReauthentication は WebAuthn 再認証を完了し、短命な再認証セッションを発行する。
// UV が確認できない assertion は無条件に拒否する。
// 解決された credential が現在の bearer session の account に属することを検証する。
func (s *AuthService) FinishReauthentication(ctx context.Context, input FinishReauthenticationInput) (ReauthenticationSession, error) {
	if s.webauthn == nil {
		return ReauthenticationSession{}, ErrInternalError
	}

	credentialHandle, _, _, _, err := s.webauthn.FinishLogin(ctx, "", input.Credential, s.accountRepo.FindWebAuthnCredential)
	if err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return ReauthenticationSession{}, ErrInternalError
		}
		return ReauthenticationSession{}, ErrBadRequest
	}

	// credentialHandle に紐づく account を取得し、bearer session の account と一致することを確認する。
	account, err := s.accountRepo.FindByCredential(ctx, credentialHandle)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return ReauthenticationSession{}, ErrInternalError
		}
		return ReauthenticationSession{}, ErrBadRequest
	}
	if account.AccountID() != input.AccountID {
		return ReauthenticationSession{}, ErrBadRequest
	}

	requestID, reauthSessionID, err := s.nextTwoIDs()
	if err != nil {
		return ReauthenticationSession{}, ErrInternalError
	}

	reauthSession, err := domain.NewReauthenticationSession(
		reauthSessionID,
		input.AccountID,
		input.SessionID,
		input.Kind,
		requestID,
		s.clock().Add(s.authConfig.ReauthSessionTTL),
	)
	if err != nil {
		return ReauthenticationSession{}, ErrInternalError
	}

	if err := s.stateRepo.SaveReauthenticationSession(ctx, reauthSession, s.authConfig.ReauthSessionTTL); err != nil {
		return ReauthenticationSession{}, ErrInternalError
	}

	return ReauthenticationSession{
		RequestID:       requestID,
		ReauthSessionID: reauthSession.ID(),
		Kind:            input.Kind,
		ExpiresAt:       reauthSession.ExpiresAt(),
	}, nil
}

// finishPasskeyAuthenticationWebAuthn は go-webauthn provider を使った WebAuthn ceremony 実装。
func (s *AuthService) finishPasskeyAuthenticationWebAuthn(ctx context.Context, input FinishPasskeyAuthenticationInput) (AuthSession, error) {
	if s.webauthn == nil {
		return AuthSession{}, ErrInternalError
	}
	// credential.ID を lockKey の seed とする（FinishLogin 前は credentialHandle が未確定なため）。
	// これにより無効な challenge/signature 試行もロックカウントの対象となる。
	lockKey := failureLockKey(strings.TrimSpace(input.Credential.ID), input.ClientIP)
	if err := s.ensureNotLocked(ctx, lockKey); err != nil {
		return AuthSession{}, err
	}

	// challengeKey は空文字列を渡す: provider が clientDataJSON から challenge を自己解決する。
	// lookupCredential コールバックで DB から stored credential（公開鍵等）を取得して full signature verification を行う。
	credentialHandle, newSignCount, newBackupState, signCountUpdated, err := s.webauthn.FinishLogin(ctx, "", input.Credential, s.accountRepo.FindWebAuthnCredential)
	if err != nil {
		// DB 障害（ErrAuthStoreUnavailable 等）は内部エラーとして分類する。failure counter は加算しない。
		// WebAuthn library のシグネチャ・challenge 検証失敗は ErrBadRequest → failure を加算。
		// 内部エラー時は registerFailure を呼ばない（security: 内部エラーでロックアウトさせない）。
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return AuthSession{}, ErrInternalError
		}
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	// FinishLogin 成功後に SignCount と BackupState を DB に永続化する（リプレイ攻撃検出のため）。
	// signCountUpdated が false の場合は updatedCred が取得できなかったため更新をスキップする。
	// 更新失敗はサービス継続を妨げない（best-effort）。
	if signCountUpdated {
		if updateErr := s.accountRepo.UpdateWebAuthnCredentialState(ctx, credentialHandle, newSignCount, newBackupState); updateErr != nil {
			if s.auditNotifier != nil {
				s.auditNotifier.EmitCredentialStateUpdateFailure(ctx, credentialHandle, updateErr)
			}
		}
	}

	account, err := s.accountRepo.FindByCredential(ctx, credentialHandle)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return AuthSession{}, ErrInternalError
		}
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	// 有効な WebAuthn assertion 後、アカウント停止状態を検証する。
	// 停止中アカウントは新規 token pair を発行せず拒否する。
	if account.IsSuspended() {
		return AuthSession{}, ErrAccountSuspended
	}

	requestID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}

	// canonical Product account lifecycle で JWT ペアを発行し、認証セッションを構成する。
	// lifecycle が未注入の場合は fail-closed で内部エラーを返す。
	authSession := AuthSession{
		RequestID:           requestID,
		AccountID:           account.AccountID(),
		PasskeyCredentialID: primaryPasskeyCredentialID(account),
	}
	return s.issueAuthSession(ctx, authSession, account.AccountID(), input.ClientIP, input.UserAgent)
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

	// アカウントルート検索を使い、passkey が 0 件のアカウント（Admin 作成直後など）にも対応する。
	// FindByEmail は passkey credential が必須のため、0 件の場合に record not found となる。
	root, err := s.accountRepo.FindAccountRootByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return RecoveryAccepted{}, ErrInternalError
		}
		// アカウントが存在しない場合も generic accepted を返し、account existence を漏らさない。
		return RecoveryAccepted{RequestID: requestID, Accepted: true}, nil
	}

	// AccountRoot から直接 delivery を発行する。AccountAuth の偽装は不要。
	delivery, err := s.issueRecoveryDelivery(ctx, requestID, root.AccountID, root.Email, domain.TokenKindRecovery)
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

	tokenID, plainSecret, parseErr := parseURLToken(input.Token)
	if parseErr != nil {
		s.registerFailure(ctx, lockKey)
		return RecoverySession{}, ErrBadRequest
	}

	// アトミックに GET → hash 検証 → DEL で recovery token を消費する。
	recoveryToken, err := s.stateRepo.ConsumeRecoveryTokenAtomic(ctx, tokenID, plainSecret)
	if err != nil {
		s.registerFailure(ctx, lockKey)
		return RecoverySession{}, s.mapRecoveryConsumeError(err)
	}
	if err := recoveryToken.EnsureConsumable(s.clock()); err != nil {
		s.registerFailure(ctx, lockKey)
		return RecoverySession{}, ErrBadRequest
	}

	requestID, recoverySessionID, err := s.nextTwoIDs()
	if err != nil {
		return RecoverySession{}, ErrInternalError
	}
	recoverySession, err := domain.NewRecoverySession(recoverySessionID, recoveryToken.AccountID(), recoveryToken.Kind(), s.clock().Add(s.authConfig.RecoverySessionTTL))
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
		Kind:               recoverySession.Kind(),
		ExpiresAt:          recoverySession.ExpiresAt(),
	}, nil
}

// StartPasskeyRegistration はリカバリ or 招待セッションを検証してセレモニーを開始する。
func (s *AuthService) StartPasskeyRegistration(ctx context.Context, input StartPasskeyRegistrationInput) (PasskeyChallenge, error) {
	lockKey := "regstart:" + input.ClientIP
	if err := s.ensureNotLocked(ctx, lockKey); err != nil {
		return PasskeyChallenge{}, err
	}

	if selectorCount(input.RecoverySession, input.InvitationSession) != 1 {
		return PasskeyChallenge{}, ErrBadRequest
	}

	accountID, err := s.resolveRegistrationAccountID(ctx, input)
	if err != nil {
		s.registerFailure(ctx, lockKey)
		return PasskeyChallenge{}, err
	}

	// アカウント停止状態を検証する。停止中は registration ceremony を開始しない。
	if err := s.ensureAccountActive(ctx, accountID); err != nil {
		return PasskeyChallenge{}, err
	}

	// WebAuthn provider 必須
	if s.webauthn == nil {
		return PasskeyChallenge{}, ErrInternalError
	}

	challengeKey, optionsJSON, beginErr := s.webauthn.BeginRegistration(ctx, accountID)
	if beginErr != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	requestID, err := s.policy.Next()
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	return PasskeyChallenge{
		RequestID:       requestID,
		Challenge:       challengeKey,
		ChallengeID:     challengeKey,
		WebAuthnRPID:    s.authConfig.WebAuthnRPID,
		WebAuthnOptions: optionsJSON,
	}, nil
}

func (s *AuthService) resolveRegistrationAccountID(ctx context.Context, input StartPasskeyRegistrationInput) (domain.AccountID, error) {
	if strings.TrimSpace(input.InvitationSession) != "" {
		// invitation path: invitation registrar が accountID を解決する（今はシンプルにエラー）
		return "", ErrBadRequest
	}
	// recovery path
	recoverySession, err := s.stateRepo.GetRecoverySession(ctx, input.RecoverySession)
	if err != nil {
		return "", s.mapRecoveryConsumeError(err)
	}
	if err := recoverySession.EnsureAvailable(s.clock()); err != nil {
		return "", ErrBadRequest
	}
	return recoverySession.AccountID(), nil
}

func (s *AuthService) RegisterPasskey(ctx context.Context, input RegisterPasskeyInput) (AuthSession, error) {
	lockKey := failureLockKey(input.Credential.ID, input.ClientIP)
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

	// アカウント停止状態を検証する。停止中は side effect（passkey 追加・セッション消費）を実行しない。
	if err := s.ensureAccountActive(ctx, recoverySession.AccountID()); err != nil {
		return AuthSession{}, err
	}

	// recovery path: challengeKey は空文字列を渡す（provider が clientDataJSON から自己解決）
	credentialHandle, credData, err := s.resolveCredentialHandleAndData(ctx, "", recoverySession.AccountID(), input.Credential)
	if err != nil {
		s.registerFailure(ctx, lockKey)
		return AuthSession{}, ErrBadRequest
	}

	passkeyID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}
	account, err := s.accountRepo.AddPasskey(ctx, recoverySession.AccountID(), passkeyID, credentialHandle, credData)
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

	// recovery session の kind に応じた後処理を実行する。
	// recovery kind の場合、全セッション失効に失敗したら fail-closed で内部エラーを返す（セキュリティ要件）。
	// 通知メールは best-effort とし、失敗しても registration の成功を妨げない。
	if err := s.runRegisterPasskeyPostProcess(ctx, recoverySession.Kind(), account.AccountID(), account.Email()); err != nil {
		return AuthSession{}, ErrInternalError
	}

	requestID, err := s.policy.Next()
	if err != nil {
		return AuthSession{}, ErrInternalError
	}

	// canonical Product account lifecycle で JWT ペアを発行し、認証セッションを構成する。
	// lifecycle が未注入の場合は fail-closed で内部エラーを返す。
	authSession := AuthSession{
		RequestID:           requestID,
		AccountID:           account.AccountID(),
		PasskeyCredentialID: primaryPasskeyCredentialID(account),
	}
	return s.issueAuthSession(ctx, authSession, account.AccountID(), input.ClientIP, input.UserAgent)
}

func primaryPasskeyCredentialID(account domain.AccountAuth) string {
	credentials := account.Credentials()
	if len(credentials) == 0 {
		return ""
	}
	return credentials[0].ID()
}

// runRegisterPasskeyPostProcess は RegisterPasskey 成功後の kind 別後処理を実行する。
// recovery の場合は全セッションを失効させて recovery 完了メールを送信する。
// セッション失効の失敗はセキュリティ要件のため fail-closed で error を返す。
// 通知メールの送信失敗は best-effort とし、ログ記録のみで error は返さない。
// device-link の場合は device-link 完了メールを best-effort で送信する。
func (s *AuthService) runRegisterPasskeyPostProcess(ctx context.Context, kind domain.TokenKind, accountID domain.AccountID, email string) error {
	switch kind {
	case domain.TokenKindRecovery:
		// 全セッション失効はセキュリティ上必須。canonical lifecycle がない場合や失効失敗時は registration 全体を fail させる。
		if s.accountLifecycle == nil {
			return ErrInternalError
		}
		if err := s.accountLifecycle.RevokeAllAccountSessions(ctx, accountID); err != nil {
			if s.auditNotifier != nil {
				s.auditNotifier.EmitRecoverySessionRevokeFailure(ctx, accountID, err)
			}
			return ErrInternalError
		}
		// 通知メールは best-effort
		if s.recoveryCompleteSender != nil {
			delivery := CompletionDelivery{AccountID: accountID, Email: email, Kind: domain.TokenKindRecovery}
			if err := s.recoveryCompleteSender.SendRecoveryComplete(ctx, delivery); err != nil {
				if s.auditNotifier != nil {
					s.auditNotifier.EmitRecoveryCompleteDeliveryFailure(ctx, accountID, err)
				}
			}
		}
	case domain.TokenKindDeviceLink:
		// device-link 完了通知は best-effort
		if s.deviceLinkCompleteSender != nil {
			delivery := CompletionDelivery{AccountID: accountID, Email: email, Kind: domain.TokenKindDeviceLink}
			if err := s.deviceLinkCompleteSender.SendDeviceLinkComplete(ctx, delivery); err != nil {
				if s.auditNotifier != nil {
					s.auditNotifier.EmitDeviceLinkCompleteDeliveryFailure(ctx, accountID, err)
				}
			}
		}
	}
	return nil
}

// resolveCredentialHandleAndData は WebAuthn FinishRegistration を呼んで credential handle と永続化データを返す。
// challengeKey が空文字列の場合、provider は clientDataJSON から challenge を自己解決する。
func (s *AuthService) resolveCredentialHandleAndData(ctx context.Context, challengeKey string, accountID domain.AccountID, credential WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
	if s.webauthn == nil {
		return "", domain.ZeroWebAuthnCredentialData(), ErrInternalError
	}
	handle, credData, err := s.webauthn.FinishRegistration(ctx, challengeKey, accountID, credential)
	if err != nil {
		return "", domain.ZeroWebAuthnCredentialData(), err
	}
	return handle, credData, nil
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
	if strings.TrimSpace(token) == "" {
		return "", ErrUnauthenticated
	}

	requestID, err := s.policy.Next()
	if err != nil {
		return "", ErrInternalError
	}

	if s.accountLifecycle == nil {
		return "", ErrInternalError
	}

	// Step 1: Product account lifecycle owner だけに bearer validation と revoke を委譲し、root legacy TokenService fallback へ戻らない。
	return s.logoutWithAccountLifecycle(ctx, token, requestID)
}

func (s *AuthService) logoutWithAccountLifecycle(ctx context.Context, token string, requestID string) (string, error) {
	// Step 1: production では Product account lifecycle owner に bearer validation を委譲し、root TokenService の sessionStore 直参照へ戻らない。
	session, authErr := s.accountLifecycle.AuthorizeAccountSession(ctx, token)
	if authErr != nil {
		return "", mapAccountLifecycleError(authErr)
	}

	// Step 2: accessToken claims と server-side session metadata で確定した session だけを revoke し、Cookie や client hint で対象を選ばない。
	if revokeErr := s.accountLifecycle.RevokeAccountSession(ctx, RevokeAccountSessionInput{AccountID: session.AccountID, SessionID: session.SessionID}); revokeErr != nil {
		return "", mapAccountLifecycleError(revokeErr)
	}

	// Step 3: root facade は requestID を返すだけに留め、lifecycle side effect は canonical owner で完了済みにする。
	return requestID, nil
}

func (s *AuthService) AuthorizeSession(ctx context.Context, token string) (AuthSession, error) {
	if strings.TrimSpace(token) == "" {
		return AuthSession{}, ErrUnauthenticated
	}

	// Step 1: Product account lifecycle owner がない構成は bearer validation を実行せず fail-close する。
	if s.accountLifecycle == nil {
		return AuthSession{}, ErrInternalError
	}

	// Step 2: canonical Product account lifecycle owner に bearer validation を委譲し、root TokenService の検証ロジック重複を通らない。
	session, err := s.accountLifecycle.AuthorizeAccountSession(ctx, token)
	if err != nil {
		return AuthSession{}, mapAccountLifecycleError(err)
	}
	return AuthSession{AccountID: session.AccountID, SessionID: session.SessionID, AuthContextID: session.SessionID}, nil
}

// issueRecoveryDelivery は recovery token を発行し、指定された kind の RecoveryDelivery を生成する。
// RecoveryURL は AccountRecoveryURLBase を使用する。kind が空の場合はエラーを返す。
//
// 引数:
//   - requestID: 相関 ID。
//   - accountID: Product Account の canonical ULID。
//   - email: 配送先メールアドレス。
//   - kind: TokenKindRecovery または TokenKindDeviceLink。
func (s *AuthService) issueRecoveryDelivery(ctx context.Context, requestID string, accountID domain.AccountID, email string, kind domain.TokenKind) (RecoveryDelivery, error) {
	tokenID, err := s.policy.Next()
	if err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}
	urlToken, plainSecret, err := generateURLToken(tokenID)
	if err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}
	recoveryToken, err := domain.NewRecoveryToken(tokenID, accountID, plainSecret, kind, s.clock().Add(s.authConfig.RecoveryTokenTTL))
	if err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}
	if err := s.stateRepo.IssueRecoveryToken(ctx, recoveryToken, s.authConfig.RecoveryTokenTTL); err != nil {
		return RecoveryDelivery{}, ErrInternalError
	}

	return RecoveryDelivery{
		RequestID:       requestID,
		RecoveryTokenID: tokenID,
		AccountID:       accountID,
		Email:           email,
		RecoveryURL:     fmt.Sprintf("%s?token=%s", strings.TrimSpace(s.authConfig.AccountRecoveryURLBase), urlToken),
		Kind:            kind,
		ExpiresAt:       recoveryToken.ExpiresAt(),
	}, nil
}

// ensureAccountActive は指定アカウントが active であることを検証する。
// passkey が 0 件のアカウント（Admin 作成直後など）にも対応するため、
// FindAccountRootByID を使用し、passkey credential に依存しない。
// 停止中アカウントの場合は ErrAccountSuspended を返す。
// DB 障害時は ErrInternalError を返す。
func (s *AuthService) ensureAccountActive(ctx context.Context, accountID domain.AccountID) error {
	root, err := s.accountRepo.FindAccountRootByID(ctx, accountID)
	if err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return ErrInternalError
		}
		return ErrBadRequest
	}
	if root.Status == "suspended" {
		return ErrAccountSuspended
	}
	return nil
}

// issueAuthSession は Product account lifecycle を用いて JWT アクセストークン・リフレッシュトークン・セッションID を発行し、
// 与えられた AuthSession に付与して返す。これは root facade における認証完了後の唯一のセッション発行パスである。
// Product account lifecycle が未注入の場合は fail-closed で内部エラーを返す。
func (s *AuthService) issueAuthSession(ctx context.Context, authSession AuthSession, accountID domain.AccountID, clientIP string, userAgent string) (AuthSession, error) {
	// Step 1: Product account lifecycle owner がない構成は session を発行せず fail-close する。
	if s.accountLifecycle == nil {
		return AuthSession{}, ErrInternalError
	}

	// Step 2: 確定済み AccountID を canonical Product account lifecycle owner へ渡し、session issuance と refresh family 保存を root facade から分離する。
	result, err := s.accountLifecycle.IssueAccountSession(ctx, IssueAccountSessionInput{AccountID: accountID, ClientIP: clientIP, UserAgent: userAgent})
	if err != nil {
		return AuthSession{}, mapAccountLifecycleError(err)
	}
	if result.Session.AccessToken == "" || result.RefreshCookie.Value == "" {
		return AuthSession{}, ErrInternalError
	}
	authSession.AccountID = result.Session.AccountID
	authSession.SessionID = result.Session.SessionID
	authSession.AuthContextID = result.Session.SessionID
	authSession.AccessToken = result.Session.AccessToken
	authSession.RefreshToken = result.RefreshCookie.Value
	authSession.ExpiresAt = result.Session.ExpiresAt
	return authSession, nil
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

// ─── Multi-passkey management ───────────────────────────────────────────────

// ListPasskeys は accountID に紐づく全パスキー credential を返す。
func (s *AuthService) ListPasskeys(ctx context.Context, accountID domain.AccountID) ([]PasskeyCredentialDTO, error) {
	creds, err := s.accountRepo.ListPasskeys(ctx, accountID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountAuthNotFound) {
			return nil, domain.ErrAccountAuthNotFound
		}
		return nil, ErrInternalError
	}
	return toPasskeyCredentialDTOs(creds), nil
}

// StartAddPasskey は認証済みアカウントのパスキー追加チャレンジを発行する。
func (s *AuthService) StartAddPasskey(ctx context.Context, accountID domain.AccountID) (PasskeyChallenge, error) {
	if s.webauthn == nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	challengeKey, optionsJSON, beginErr := s.webauthn.BeginRegistration(ctx, accountID)
	if beginErr != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	requestID, err := s.policy.Next()
	if err != nil {
		return PasskeyChallenge{}, ErrInternalError
	}
	return PasskeyChallenge{
		RequestID:       requestID,
		Challenge:       challengeKey,
		ChallengeID:     challengeKey,
		WebAuthnRPID:    s.authConfig.WebAuthnRPID,
		WebAuthnOptions: optionsJSON,
	}, nil
}

// FinishAddPasskey はチャレンジを検証して新しいパスキーを追加する。
func (s *AuthService) FinishAddPasskey(ctx context.Context, accountID domain.AccountID, credential WebAuthnAttestationCredentialDTO) ([]PasskeyCredentialDTO, error) {
	// challengeKey は空文字列を渡す（provider が clientDataJSON から自己解決）
	credentialHandle, credData, err := s.resolveCredentialHandleAndData(ctx, "", accountID, credential)
	if err != nil {
		return nil, ErrBadRequest
	}

	passkeyID, err := s.policy.Next()
	if err != nil {
		return nil, ErrInternalError
	}
	if _, err := s.accountRepo.AddPasskey(ctx, accountID, passkeyID, credentialHandle, credData); err != nil {
		if errors.Is(s.mapAuthStoreError(err), ErrInternalError) {
			return nil, ErrInternalError
		}
		return nil, ErrBadRequest
	}

	creds, err := s.accountRepo.ListPasskeys(ctx, accountID)
	if err != nil {
		return nil, ErrInternalError
	}
	return toPasskeyCredentialDTOs(creds), nil
}

func (s *AuthService) DeletePasskey(ctx context.Context, accountID domain.AccountID, credentialID string) error {
	creds, err := s.accountRepo.ListPasskeys(ctx, accountID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountAuthNotFound) {
			return domain.ErrAccountAuthNotFound
		}
		return ErrInternalError
	}
	if len(creds) <= 1 {
		return ErrLastPasskeyCannotBeDeleted
	}

	// credentialID が accountID に属することを確認してから削除
	if err := s.accountRepo.DeletePasskeyByID(ctx, accountID, credentialID); err != nil {
		if errors.Is(err, domain.ErrAccountAuthNotFound) {
			return domain.ErrAccountAuthNotFound
		}
		return ErrInternalError
	}
	return nil
}

// toPasskeyCredentialDTOs は domain.PasskeyCredential のスライスをユースケース DTO に変換する。
func toPasskeyCredentialDTOs(creds []domain.PasskeyCredential) []PasskeyCredentialDTO {
	dtos := make([]PasskeyCredentialDTO, len(creds))
	for i, c := range creds {
		dtos[i] = PasskeyCredentialDTO{
			ID:         c.ID(),
			AccountID:  c.AccountID(),
			Identifier: c.Identifier(),
			CreatedAt:  c.CreatedAt(),
		}
	}
	return dtos
}

func mapAccountLifecycleError(err error) error {
	// Step 1: nil error はそのまま返し、caller が成功 path を単純に扱えるようにする。
	if err == nil {
		return nil
	}

	// Step 2: canonical lifecycle の保存層・署名器障害は既存 Product transport が扱える内部エラーへ畳む。
	if errors.Is(err, ErrAccountAuthUnavailable) {
		return ErrInternalError
	}

	// Step 3: 入力不正は既存 auth facade の bad request として返し、response body に詳細を漏らさない。
	if errors.Is(err, ErrAccountAuthInvalidInput) {
		return ErrBadRequest
	}

	// Step 4: token reuse は外部へ詳細を出さず session expired 相当へ畳み、refresh family の失効処理は canonical owner に委ねる。
	if errors.Is(err, ErrAccountAuthTokenReuseDetected) {
		return ErrSessionExpired
	}

	// Step 5: Account lifecycle による現在状態拒否は Product HTTP の stable 403 account-suspended 分類へ渡す。
	if errors.Is(err, ErrAccountAuthIneligible) {
		return ErrAccountSuspended
	}

	// Step 6: unauthorized は protected route / refresh / logout の既存分類に合わせて session expired として返す。
	if errors.Is(err, ErrAccountAuthUnauthorized) {
		return ErrSessionExpired
	}

	// Step 7: 未分類 error は fail-closed に内部エラーとして扱う。
	return ErrInternalError
}
