package application

import (
	"context"
	"encoding/json"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// RefreshAccountSession は Product refreshToken Cookie を rotation し、新しい accessToken と refresh Cookie command を返す。
//
// 役割:
//   - 平文 refreshToken を保存用 hash に変換し、RefreshSessionStore.Rotate に旧 token の原子消費を委譲する。
//   - Rotate callback 内で Product AccountAuth domain object による CanRotate を必ず実行する。
//   - 新しい refreshToken は response body ではなく RefreshCookieCommand にだけ入れる。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: Cookie から取得した refreshToken、対象 session selector、device metadata seed。
//
// 戻り値:
//   - RefreshResult: rotation 後の accessToken body DTO と refresh Cookie command。
//   - error: token 不正、session 不一致、Account 停止、保存失敗などの application error。
func (s *Service) RefreshAccountSession(ctx context.Context, input RefreshAccountSessionInput) (RefreshResult, error) {
	// Step 1: request が対象にした Product session selector を domain value として検証する。
	selector, err := domain.NewAccountAuthSessionID(input.SessionID)
	if err != nil {
		return RefreshResult{}, ErrProductAuthInvalidInput
	}

	// Step 2: Cookie の平文 refreshToken を保存用 hash に変換し、平文を保存層へ渡さない。
	oldHash, err := domain.HashOpaqueToken(input.RefreshToken)
	if err != nil {
		return RefreshResult{}, ErrProductAuthInvalidInput
	}

	// Step 3: rotation 前に session metadata の存在を確認し、metadata revoke 済み session の refresh 復活を防ぐ。
	existingMetadata, err := s.sessions.Get(ctx, selector)
	if err != nil {
		return RefreshResult{}, mapProductAuthError(err)
	}
	if existingMetadata.SessionID != selector.String() {
		return RefreshResult{}, ErrProductAuthUnauthorized
	}

	// Step 4: 新 refreshToken は response body へ出さず、成功時の Cookie command だけに流す。
	newRefreshToken, newRefreshHash, err := s.newRefreshTokenHash()
	if err != nil {
		return RefreshResult{}, err
	}

	// Step 5: accessToken 用 jti は rotation callback の外で生成し、保存層が ID 発行責務を持たないようにする。
	jti, err := s.nextTokenJTI()
	if err != nil {
		return RefreshResult{}, err
	}

	// Step 6: Rotate callback が作った claim と metadata を callback 外で署名・保存できるよう保持する。
	issuedAt := s.clock().UTC()
	var claims domain.AccountAccessTokenClaims
	var metadata SessionMetadata

	// Step 7: 旧 token 消費と新 token 保存の間に Product AccountAuth domain validation を実行する。
	_, nextSession, err := s.refreshSessions.Rotate(ctx, oldHash, s.refreshTTL.Duration(), func(consumed domain.AccountRefreshSession) (domain.AccountRefreshSession, error) {
		if existingMetadata.AccountID != consumed.AccountID() {
			return zeroProductRefreshSession(), ErrProductAuthUnauthorized
		}
		return s.buildRotatedRefreshSession(ctx, consumed, selector, newRefreshHash, jti, issuedAt, deviceInput{clientIP: input.ClientIP, userAgent: input.UserAgent}, &claims, &metadata)
	})
	if err != nil {
		return RefreshResult{}, mapProductAuthError(err)
	}

	// Step 8: rotation が成功した後で accessToken を署名し、失敗時に未保存 token を返さない。
	accessToken, err := s.signAccessToken(claims)
	if err != nil {
		return RefreshResult{}, mapProductAuthError(err)
	}

	// Step 9: bearer validation 用 metadata を更新し、対象 session の LastActiveAt を refresh 時刻に進める。
	if err := s.sessions.Save(ctx, metadata, s.refreshTTL.Duration()); err != nil {
		return RefreshResult{}, mapProductAuthError(err)
	}

	// Step 10: body DTO と Cookie command を分け、refreshToken の body 露出を構造的に避ける。
	return RefreshResult{
		Session: AuthenticatedSession{
			AccountID:   claims.AccountID(),
			SessionID:   nextSession.SessionID().String(),
			AccessToken: accessToken,
			ExpiresAt:   claims.ExpiresAt(),
			DeviceName:  metadata.DeviceName,
		},
		RefreshCookie: RefreshCookieCommand{
			Value:     newRefreshToken,
			MaxAge:    s.refreshCookieLifetime,
			ExpiresAt: issuedAt.Add(s.refreshCookieLifetime).UTC(),
			Clear:     false,
		},
	}, nil
}

// ValidateAccountBearer は Product accessToken と session selector を現在の AccountAuth 状態で検証する。
//
// 役割:
//   - 中立 verifier で署名済み JSON payload を取り出す。
//   - Product AccountAuth projection、session metadata、AccountAccessTokenClaims.EnsureEligible を照合する。
//   - Admin operator token を Product account session として扱わない。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: bearer accessToken と request session selector。
//
// 戻り値:
//   - ValidatedSession: downstream use case に渡す Product account caller context。
//   - error: 署名不正、期限切れ、停止、revoke、session mismatch などの application error。
func (s *Service) ValidateAccountBearer(ctx context.Context, input ValidateAccountBearerInput) (ValidatedSession, error) {
	// Step 1: request が選択した session selector を Product session ID として検証する。
	selector, err := domain.NewAccountAuthSessionID(input.SessionID)
	if err != nil {
		return ValidatedSession{}, ErrProductAuthUnauthorized
	}

	// Step 2: 署名済み bearer token を Product accessToken payload として復元し、request selector と token sid を照合する。
	tokenValues, err := s.accountBearerTokenValues(input, selector)
	if err != nil {
		return ValidatedSession{}, mapProductAuthError(err)
	}

	// Step 3: 現在の Product AccountAuth projection を取得し、永続化 subject と token subject の不一致を拒否する。
	account, err := s.accountRootForBearer(ctx, tokenValues.accountID)
	if err != nil {
		return ValidatedSession{}, mapProductAuthError(err)
	}

	// Step 4: session metadata と payload status を現在状態へ照合し、revoke / subject mismatch / suspended token を fail-closed にする。
	if err := s.ensureBearerSessionMetadata(ctx, selector, tokenValues.accountID); err != nil {
		return ValidatedSession{}, mapProductAuthError(err)
	}
	if err := ensureBearerStatusMatchesAccount(tokenValues.status, account); err != nil {
		return ValidatedSession{}, mapProductAuthError(err)
	}

	// Step 5: domain constructor で Product AccountAccessTokenClaims を再構成し、EnsureEligible で現在状態と照合する。
	claims, err := domain.NewAccountAccessTokenClaims(
		account,
		tokenValues.sessionID,
		tokenValues.jti,
		tokenValues.issuedAt,
		tokenValues.ttl,
	)
	if err != nil {
		return ValidatedSession{}, mapProductAuthError(err)
	}
	if err := claims.EnsureEligible(account, selector, s.clock().UTC()); err != nil {
		return ValidatedSession{}, mapProductAuthError(err)
	}

	// Step 6: downstream use case が必要な caller context だけを application DTO として返す。
	return ValidatedSession{
		AccountID: tokenValues.accountID,
		SessionID: selector.String(),
		TokenID:   tokenValues.jti.String(),
		ExpiresAt: claims.ExpiresAt(),
	}, nil
}

type accountBearerTokenValues struct {
	accountID domain.AccountID
	sessionID domain.AccountAuthSessionID
	jti       domain.TokenJTI
	issuedAt  time.Time
	ttl       domain.TokenTTL
	status    string
}

func (s *Service) accountBearerTokenValues(
	input ValidateAccountBearerInput,
	selector domain.AccountAuthSessionID,
) (accountBearerTokenValues, error) {
	// Step 1: JWT 署名と JSON object 形式だけを中立 verifier で検証し、未署名値や壊れた署名を拒否する。
	payloadBytes, err := s.signer.VerifyJSON(input.AccessToken)
	if err != nil {
		return accountBearerTokenValues{}, err
	}

	// Step 2: Product account accessToken payload として必要 field を decode する。
	payload, err := decodeAccessTokenPayload(payloadBytes)
	if err != nil {
		return accountBearerTokenValues{}, err
	}

	// Step 3: payload から AccountID / SessionID / JTI / TTL を domain value として復元する。
	accountID, sessionID, jti, issuedAt, ttl, err := payload.domainValues()
	if err != nil {
		return accountBearerTokenValues{}, err
	}

	// Step 4: request selector と token sid が一致しない場合は対象外 session の bearer として拒否する。
	if sessionID != selector {
		return accountBearerTokenValues{}, ErrProductAuthUnauthorized
	}

	// Step 5: 後続の現在状態照合で使う token-derived value をまとめて返す。
	return accountBearerTokenValues{
		accountID: accountID,
		sessionID: sessionID,
		jti:       jti,
		issuedAt:  issuedAt,
		ttl:       ttl,
		status:    payload.Status,
	}, nil
}

func (s *Service) accountRootForBearer(ctx context.Context, accountID domain.AccountID) (domain.Account, error) {
	// Step 1: token subject に対応する現在の Product AccountAuth projection を repository から取得する。
	accountAuth, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return zeroProductAccount(), err
	}

	// Step 2: repository が別 subject を返す不整合は Product bearer として fail-closed に拒否する。
	if accountAuth.AccountID() != accountID {
		return zeroProductAccount(), ErrProductAuthUnauthorized
	}

	// Step 3: AccountAuth projection を Account root へ写像し、domain eligibility 検証に使える形へ戻す。
	return accountRootFromAuth(accountAuth)
}

func (s *Service) ensureBearerSessionMetadata(
	ctx context.Context,
	selector domain.AccountAuthSessionID,
	accountID domain.AccountID,
) error {
	// Step 1: session metadata が存在することを確認し、logout/revoke 済み selector の bearer を拒否する。
	metadata, err := s.sessions.Get(ctx, selector)
	if err != nil {
		return err
	}

	// Step 2: metadata owner と selector が token/request と一致する場合だけ bearer session として受け入れる。
	if metadata.AccountID != accountID || metadata.SessionID != selector.String() {
		return ErrProductAuthUnauthorized
	}

	// Step 3: metadata 境界の照合が完了したことを nil error で返す。
	return nil
}

func ensureBearerStatusMatchesAccount(payloadStatus string, account domain.Account) error {
	// Step 1: payload status を Product AccountStatus として再検証し、壊れた claim を拒否する。
	status, err := domain.NewAccountStatus(payloadStatus)
	if err != nil {
		return err
	}

	// Step 2: suspended token と現在 status からずれた token は Product bearer として受け入れない。
	if status.IsSuspended() || status != account.Status() {
		return ErrProductAuthUnauthorized
	}

	// Step 3: payload status が現在 Account 状態と一致したことを返す。
	return nil
}

func (s *Service) buildRotatedRefreshSession(ctx context.Context, consumed domain.AccountRefreshSession, selector domain.AccountAuthSessionID, nextHash domain.OpaqueTokenHash, jti domain.TokenJTI, issuedAt time.Time, device deviceInput, claims *domain.AccountAccessTokenClaims, metadata *SessionMetadata) (domain.AccountRefreshSession, error) {
	// Step 1: refresh session が所有する Product AccountAuth projection を取得する。
	accountAuth, err := s.accounts.FindByID(ctx, consumed.AccountID())
	if err != nil {
		return zeroProductRefreshSession(), mapProductAuthError(err)
	}

	// Step 2: projection を Account root へ写像し、Product lifecycle と revoke 境界を domain method で使えるようにする。
	account, err := accountRootFromAuth(accountAuth)
	if err != nil {
		return zeroProductRefreshSession(), mapProductAuthError(err)
	}

	// Step 3: 旧 refresh session が現在 Account 状態と selector で rotation 可能か domain object に判定させる。
	if err := consumed.CanRotate(account, selector, issuedAt); err != nil {
		return zeroProductRefreshSession(), mapProductAuthError(err)
	}

	// Step 4: 新 accessToken claim を Product AccountAuth domain constructor で生成する。
	nextClaims, err := domain.NewAccountAccessTokenClaims(account, selector, jti, issuedAt, s.accessTTLValue())
	if err != nil {
		return zeroProductRefreshSession(), mapProductAuthError(err)
	}

	// Step 5: 新 refresh session state を同じ Product session selector で生成し、multi-session 時に対象 session だけを rotation する。
	nextSession, err := domain.NewAccountRefreshSession(account, selector, nextHash, issuedAt, s.refreshTTL.ExpiresAt(issuedAt))
	if err != nil {
		return zeroProductRefreshSession(), mapProductAuthError(err)
	}

	// Step 6: callback 外で署名・metadata 保存できるよう検証済み値を呼び出し元変数へ渡す。
	*claims = nextClaims
	*metadata = s.sessionMetadata(account.ID(), selector, device, issuedAt)

	// Step 7: 保存層が保存すべき次 refresh session を返す。
	return nextSession, nil
}

func decodeAccessTokenPayload(payloadBytes []byte) (accessTokenPayload, error) {
	// Step 1: JSON object payload を Product access token payload DTO へ decode する。
	var payload accessTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return accessTokenPayload{}, domain.ErrInvalidSignature
	}

	// Step 2: 必須 field の欠落は署名済みでも Product access token として扱えないため拒否する。
	if payload.Subject == "" || payload.SessionID == "" || payload.TokenID == "" || payload.Status == "" || payload.IssuedAt == 0 || payload.ExpiresAt == 0 {
		return accessTokenPayload{}, domain.ErrInvalidSignature
	}

	// Step 3: payload としての最低条件を満たした値を返す。
	return payload, nil
}

func (p accessTokenPayload) domainValues() (domain.AccountID, domain.AccountAuthSessionID, domain.TokenJTI, time.Time, domain.TokenTTL, error) {
	// Step 1: subject を Product AccountID として検証する。
	accountID, err := domain.NewAccountID(p.Subject)
	if err != nil {
		return "", "", "", time.Time{}, zeroProductTokenTTL(), err
	}

	// Step 2: sid を Product AccountAuth session ID として検証する。
	sessionID, err := domain.NewAccountAuthSessionID(p.SessionID)
	if err != nil {
		return "", "", "", time.Time{}, zeroProductTokenTTL(), err
	}

	// Step 3: jti を中立 TokenJTI として検証する。
	jti, err := domain.NewTokenJTI(p.TokenID)
	if err != nil {
		return "", "", "", time.Time{}, zeroProductTokenTTL(), err
	}

	// Step 4: iat/exp から発行時刻と TTL を再構成し、exp <= iat を拒否する。
	issuedAt := time.Unix(p.IssuedAt, 0).UTC()
	expiresAt := time.Unix(p.ExpiresAt, 0).UTC()
	if !expiresAt.After(issuedAt) {
		return "", "", "", time.Time{}, zeroProductTokenTTL(), domain.ErrInvalidTokenTTL
	}

	// Step 5: TTL duration を domain primitive で検証する。
	ttl, err := domain.ValidateTokenTTL(expiresAt.Sub(issuedAt))
	if err != nil {
		return "", "", "", time.Time{}, zeroProductTokenTTL(), err
	}

	// Step 6: 復元済み domain value をまとめて返す。
	return accountID, sessionID, jti, issuedAt, ttl, nil
}
