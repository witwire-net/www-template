package auth

import (
	"context"
	"encoding/json"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// RefreshAccountSession は Product refreshToken Cookie を rotation し、新しい accessToken と refresh Cookie command を返す。
//
// 役割:
//   - 平文 refreshToken を保存用 hash に変換し、AccountRefreshSessionStore.Rotate に旧 token の原子消費を委譲する。
//   - Rotate callback 内で Product AccountAuth domain object による CanRotate を必ず実行する。
//   - 新しい refreshToken は response body ではなく RefreshCookieCommand にだけ入れる。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: Cookie から取得した refreshToken、対象 session selector、device metadata seed。
//
// 戻り値:
//   - AccountRefreshResult: rotation 後の accessToken body DTO と refresh Cookie command。
//   - error: token 不正、session 不一致、Account 停止、保存失敗などの application error。
func (s *AccountSessionService) RefreshAccountSession(ctx context.Context, input RefreshAccountSessionInput) (AccountRefreshResult, error) {
	// Step 0: refresh response の requestId も canonical lifecycle 側で発行し、HTTP adapter が legacy root token service に依存しないようにする。
	requestID, err := s.idGenerator.Next()
	if err != nil {
		return AccountRefreshResult{}, ErrAccountAuthUnavailable
	}

	// Step 1: request が対象にした Product session selector を domain value として検証する。
	selector, err := domain.NewAccountAuthSessionID(input.SessionID)
	if err != nil {
		return AccountRefreshResult{}, ErrAccountAuthInvalidInput
	}

	// Step 2: Cookie の平文 refreshToken を canonical auth lifecycle helper で保存用 hash に変換し、平文を保存層へ渡さない。
	oldHash, err := HashRefreshCredential(input.RefreshToken)
	if err != nil {
		return AccountRefreshResult{}, ErrAccountAuthInvalidInput
	}

	// Step 3: 新 refreshToken は response body へ出さず、成功時の Cookie command だけに流す。
	newRefreshToken, newRefreshHash, err := s.newRefreshTokenHash()
	if err != nil {
		return AccountRefreshResult{}, err
	}

	// Step 4: accessToken 用 jti は rotation callback の外で生成し、保存層が ID 発行責務を持たないようにする。
	jti, err := s.nextTokenJTI()
	if err != nil {
		return AccountRefreshResult{}, err
	}

	// Step 5: Rotate callback が作った claim と metadata を callback 外で署名・保存できるよう保持する。
	issuedAt := s.clock().UTC()
	var claims domain.AccountAccessTokenClaims
	var metadata SessionMetadata

	// Step 6: 旧 token 消費と新 token 保存の間に Product AccountAuth domain validation を実行する。
	_, nextSession, err := s.refreshSessions.Rotate(ctx, oldHash, s.refreshTTL.Duration(), func(consumed domain.AccountRefreshSession) (domain.AccountRefreshSession, error) {
		// Step 6-a: token を消費した後で metadata 所有権を照合し、path mismatch でも提示済み refreshToken を再利用不能にする。
		existingMetadata, err := s.sessions.Get(ctx, selector)
		if err != nil {
			return zeroProductRefreshSession(), mapAccountAuthError(err)
		}
		if existingMetadata.AccountID != consumed.AccountID() || existingMetadata.SessionID != selector.String() {
			return zeroProductRefreshSession(), ErrAccountAuthUnauthorized
		}
		return s.buildRotatedRefreshSession(ctx, consumed, selector, newRefreshHash, jti, issuedAt, deviceInput{clientIP: input.ClientIP, userAgent: input.UserAgent}, &claims, &metadata)
	})
	if err != nil {
		return AccountRefreshResult{}, mapAccountAuthError(err)
	}

	// Step 7: rotation が成功した後で accessToken を署名し、失敗時に未保存 token を返さない。
	accessToken, err := s.signAccessToken(claims)
	if err != nil {
		return AccountRefreshResult{}, mapAccountAuthError(err)
	}

	// Step 8: bearer validation 用 metadata を更新し、対象 session の LastActiveAt を refresh 時刻に進める。
	if err := s.sessions.Save(ctx, metadata, s.refreshTTL.Duration()); err != nil {
		return AccountRefreshResult{}, mapAccountAuthError(err)
	}

	// Step 9: body DTO と Cookie command を分け、refreshToken の body 露出を構造的に避ける。
	return AccountRefreshResult{
		RequestID: requestID,
		Session: AuthenticatedSession{
			AccountID:   claims.AccountID(),
			SessionID:   nextSession.SessionID().String(),
			AccessToken: accessToken,
			ExpiresAt:   claims.ExpiresAt(),
			DeviceName:  metadata.DeviceName,
		},
		RefreshCookie: AccountRefreshCookieCommand{
			Value:     newRefreshToken,
			MaxAge:    s.refreshCookieLifetime,
			ExpiresAt: issuedAt.Add(s.refreshCookieLifetime).UTC(),
			Clear:     false,
		},
	}, nil
}

// AuthorizeAccountSession は Product accessToken を検証し、現在有効な account session を返す。
//
// 役割:
//   - accessToken payload から session selector を復元し、ValidateAccountBearer と同じ domain eligibility 検証を実行する。
//   - root AuthService / HTTP middleware が旧 TokenService.VerifyAccessToken と sessionStore 直参照へ戻らないようにする。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accessToken: Authorization header から抽出済みの bearer token。
//
// 戻り値:
//   - ValidatedSession: account/session/token の検証済み caller context。
//   - error: token 不正、期限切れ、session 失効、account 停止、保存先失敗など。
func (s *AccountSessionService) AuthorizeAccountSession(ctx context.Context, accessToken string) (ValidatedSession, error) {
	// Step 1: 中立 signer で payload を復元し、未署名・改竄・形式不正の token を拒否する。
	payloadBytes, err := s.signer.VerifyJSON(accessToken)
	if err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}

	// Step 2: Product account accessToken payload として decode し、session selector を token 自体から取り出す。
	payload, err := decodeAccessTokenPayload(payloadBytes)
	if err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}
	claims, err := payload.domainClaims()
	if err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}

	// Step 3: 既存の bearer validation に委譲し、metadata owner・現在 Account 状態・domain eligibility を一貫して検証する。
	return s.ValidateAccountBearer(ctx, ValidateAccountBearerInput{AccessToken: accessToken, SessionID: claims.SessionID().String()})
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
func (s *AccountSessionService) ValidateAccountBearer(ctx context.Context, input ValidateAccountBearerInput) (ValidatedSession, error) {
	// Step 1: request が選択した session selector を Product session ID として検証する。
	selector, err := domain.NewAccountAuthSessionID(input.SessionID)
	if err != nil {
		return ValidatedSession{}, ErrAccountAuthUnauthorized
	}

	// Step 2: 署名済み bearer token を Product accessToken payload として復元し、request selector と token sid を照合する。
	tokenValues, err := s.accountBearerTokenValues(input, selector)
	if err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}

	// Step 3: 現在の Product AccountAuth projection を取得し、永続化 subject と token subject の不一致を拒否する。
	account, err := s.accountRootForBearer(ctx, tokenValues.accountID)
	if err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}

	// Step 4: session metadata を現在状態へ照合し、revoke / subject mismatch を fail-closed にする。
	if err := s.ensureBearerSessionMetadata(ctx, selector, tokenValues.accountID); err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}

	// Step 5: 復元済み Product AccountAccessTokenClaims の eligibility 判定を domain に委譲する。
	if err := tokenValues.claims.EnsureEligible(account, selector, s.clock().UTC()); err != nil {
		return ValidatedSession{}, mapAccountAuthError(err)
	}

	// Step 6: downstream use case が必要な caller context だけを application DTO として返す。
	return ValidatedSession{
		AccountID: tokenValues.accountID,
		SessionID: selector.String(),
		TokenID:   tokenValues.jti.String(),
		ExpiresAt: tokenValues.claims.ExpiresAt(),
	}, nil
}

type accountBearerTokenValues struct {
	accountID domain.AccountID
	sessionID domain.AccountAuthSessionID
	jti       domain.TokenJTI
	claims    domain.AccountAccessTokenClaims
}

func (s *AccountSessionService) accountBearerTokenValues(
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

	// Step 3: payload から Product AccountAuth claims を domain value として復元する。
	claims, err := payload.domainClaims()
	if err != nil {
		return accountBearerTokenValues{}, err
	}

	// Step 4: request selector と token sid が一致しない場合は対象外 session の bearer として拒否する。
	if claims.SessionID() != selector {
		return accountBearerTokenValues{}, ErrAccountAuthUnauthorized
	}

	// Step 5: 後続の現在状態照合で使う token-derived value をまとめて返す。
	return accountBearerTokenValues{
		accountID: claims.AccountID(),
		sessionID: claims.SessionID(),
		jti:       claims.JTI(),
		claims:    claims,
	}, nil
}

func (s *AccountSessionService) accountRootForBearer(ctx context.Context, accountID domain.AccountID) (domain.Account, error) {
	// Step 1: token subject に対応する現在の Product AccountAuth projection を repository から取得する。
	accountAuth, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return zeroProductAccount(), err
	}

	// Step 2: repository が別 subject を返す不整合は Product bearer として fail-closed に拒否する。
	if accountAuth.AccountID() != accountID {
		return zeroProductAccount(), ErrAccountAuthUnauthorized
	}

	// Step 3: AccountAuth projection を Account root へ写像し、domain eligibility 検証に使える形へ戻す。
	return accountRootFromAuth(accountAuth)
}

func (s *AccountSessionService) ensureBearerSessionMetadata(
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
		return ErrAccountAuthUnauthorized
	}

	// Step 3: metadata 境界の照合が完了したことを nil error で返す。
	return nil
}

func (s *AccountSessionService) buildRotatedRefreshSession(ctx context.Context, consumed domain.AccountRefreshSession, selector domain.AccountAuthSessionID, nextHash domain.OpaqueTokenHash, jti domain.TokenJTI, issuedAt time.Time, device deviceInput, claims *domain.AccountAccessTokenClaims, metadata *SessionMetadata) (domain.AccountRefreshSession, error) {
	// Step 1: refresh session が所有する Product AccountAuth projection を取得する。
	accountAuth, err := s.accounts.FindByID(ctx, consumed.AccountID())
	if err != nil {
		return zeroProductRefreshSession(), mapAccountAuthError(err)
	}

	// Step 2: projection を Account root へ写像し、Product lifecycle と revoke 境界を domain method で使えるようにする。
	account, err := accountRootFromAuth(accountAuth)
	if err != nil {
		return zeroProductRefreshSession(), mapAccountAuthError(err)
	}

	// Step 3: 旧 refresh session が現在 Account 状態と selector で rotation 可能か domain object に判定させる。
	if err := consumed.CanRotate(account, selector, issuedAt); err != nil {
		return zeroProductRefreshSession(), mapAccountAuthError(err)
	}

	// Step 4: 新 accessToken claim を Product AccountAuth domain constructor で生成する。
	nextClaims, err := domain.NewAccountAccessTokenClaims(account, selector, jti, issuedAt, s.accessTTLValue())
	if err != nil {
		return zeroProductRefreshSession(), mapAccountAuthError(err)
	}

	// Step 5: 新 refresh session state を同じ Product session selector で生成し、multi-session 時に対象 session だけを rotation する。
	nextSession, err := domain.NewAccountRefreshSession(account, selector, nextHash, issuedAt, s.refreshTTL.ExpiresAt(issuedAt))
	if err != nil {
		return zeroProductRefreshSession(), mapAccountAuthError(err)
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

func (p accessTokenPayload) domainClaims() (domain.AccountAccessTokenClaims, error) {
	// Step 1: subject を Product AccountID として検証する。
	accountID, err := domain.NewAccountID(p.Subject)
	if err != nil {
		return zeroProductAccessTokenClaims(), err
	}

	// Step 2: sid を Product AccountAuth session ID として検証する。
	sessionID, err := domain.NewAccountAuthSessionID(p.SessionID)
	if err != nil {
		return zeroProductAccessTokenClaims(), err
	}

	// Step 3: jti を中立 TokenJTI として検証する。
	jti, err := domain.NewTokenJTI(p.TokenID)
	if err != nil {
		return zeroProductAccessTokenClaims(), err
	}

	// Step 4: status と iat/exp を Product AccountAuth claims として復元し、snapshot/expiry rule は domain に委譲する。
	status, err := domain.NewAccountStatus(p.Status)
	if err != nil {
		return zeroProductAccessTokenClaims(), err
	}
	issuedAt := time.Unix(p.IssuedAt, 0).UTC()
	expiresAt := time.Unix(p.ExpiresAt, 0).UTC()
	// Step 4-a: JWT 標準 claim は秒精度のまま維持しつつ、same-second revoke 境界を正しく判定するため内部 ms claim があれば優先する。
	if p.IssuedAtMillis > 0 {
		issuedAt = time.UnixMilli(p.IssuedAtMillis).UTC()
	}
	if p.ExpiresAtMillis > 0 {
		expiresAt = time.UnixMilli(p.ExpiresAtMillis).UTC()
	}

	// Step 5: 復元済み domain claims を返す。
	return domain.ReconstituteAccountAccessTokenClaims(accountID, sessionID, jti, status, issuedAt, expiresAt)
}
