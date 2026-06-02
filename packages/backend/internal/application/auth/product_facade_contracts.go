package auth

import (
	"context"
	"errors"
	"time"

	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

// ProductAuthService は Product HTTP adapter が必要とする認証ユースケースだけを表す facade contract である。
//
// 役割:
//   - HTTP adapter が具体型 `*AuthService` へ依存せず、runtime から渡された実装を application contract として扱えるようにする。
//   - Product route adapter の production code から root legacy service 型参照を取り除き、concept package への移行中も caller evidence を明確にする。
//   - 入出力は既存 application DTO に限定し、Gin / generated binding / adapter 型を混入させない。
//
// 使用例:
//
//	var auth ProductAuthService = NewAuthService(...)
//	_ = auth
type ProductAuthService interface {
	AuthorizeSession(context.Context, string) (AuthSession, error)
	Logout(context.Context, string) (string, error)
	StartReauthentication(context.Context, StartReauthenticationInput) (PasskeyChallenge, error)
	FinishReauthentication(context.Context, FinishReauthenticationInput) (ReauthenticationSession, error)
	StartPasskeyAuthentication(context.Context, StartPasskeyAuthenticationInput) (PasskeyChallenge, error)
	FinishPasskeyAuthentication(context.Context, FinishPasskeyAuthenticationInput) (AuthSession, error)
	RequestPasskeyRecovery(context.Context, RequestPasskeyRecoveryInput) (RecoveryAccepted, error)
	ConsumeRecoveryToken(context.Context, ConsumeRecoveryTokenInput) (RecoverySession, error)
	StartPasskeyRegistration(context.Context, StartPasskeyRegistrationInput) (PasskeyChallenge, error)
	RegisterPasskey(context.Context, RegisterPasskeyInput) (AuthSession, error)
	ListPasskeys(context.Context, domain.AccountID) ([]PasskeyCredentialDTO, error)
	StartAddPasskey(context.Context, domain.AccountID) (PasskeyChallenge, error)
	FinishAddPasskey(context.Context, domain.AccountID, WebAuthnAttestationCredentialDTO) ([]PasskeyCredentialDTO, error)
	VerifyReauthSession(context.Context, string, domain.AccountID, string, string) error
	DeletePasskey(context.Context, domain.AccountID, string) error
	ExecuteDeviceLink(context.Context, domain.AccountID, string) (DeviceLinkIssued, error)
}

// ProductContextRefreshService は Product context refresh endpoint が必要とする refresh lifecycle contract である。
//
// 役割:
//   - Product HTTP adapter が具体型 `*TokenService` ではなく refresh use case の最小 contract だけを要求できるようにする。
//   - request path の authContextId と提示 refresh credential を application 境界へ渡し、response 用 DTO を受け取る。
type ProductContextRefreshService interface {
	RefreshContextWithAccountID(context.Context, string, string, string, string) (ContextRefreshSession, error)
}

// ProductSessionService は Product session 管理 route が必要とする session lifecycle contract である。
//
// 役割:
//   - HTTP adapter から concrete session service 型への依存を避ける。
//   - session list、単一 revoke、他 session revoke の操作だけを公開する。
type ProductSessionService interface {
	List(context.Context, domain.AccountID) ([]SessionMetadata, error)
	Revoke(context.Context, domain.AccountID, string) error
	RevokeOthers(context.Context, domain.AccountID, string) error
}

// NewProductAuthService は Product runtime が利用する認証 facade 実装を生成する。
//
// 役割:
//   - runtime/container が root legacy concrete 型名へ直接依存せず、ProductAuthService contract の実装を受け取れる入口を提供する。
//   - 既存の challenge / recovery / passkey 管理機能を保持しつつ、呼び出し側の型依存を facade contract へ寄せる。
//
// 引数:
//   - stateRepo: challenge / recovery / throttle state を扱う repository port。
//   - accountRepo: Product account auth projection repository port。
//   - recoverySender: recovery mail delivery port。
//   - invitationRegistrar: invitation registration port。
//   - clock: 時刻副作用を注入する関数。
//   - policy: auth ID 生成 policy。
//   - authConfig: Product auth runtime config。
//
// 戻り値:
//   - *AuthService: ProductAuthService を満たす concrete 実装。
func NewProductAuthService(stateRepo AuthStateRepository, accountRepo PasskeyAccountRepository, recoverySender AccountRecoverySender, invitationRegistrar InvitationPasskeyRegistrar, lifecycle ProductAccountLifecycle, optional AuthServiceOptionalPorts, clock func() time.Time, policy id.AuthIDPolicy, authConfig config.AuthConfig) (*AuthService, error) {
	// Step 1: WebAuthn/recovery outer flow 用 facade を生成し、token/session lifecycle は constructor で canonical Product account lifecycle に固定する。
	return NewAuthService(AuthServiceDependencies{StateRepo: stateRepo, AccountRepo: accountRepo, RecoverySender: recoverySender, InvitationRegistrar: invitationRegistrar, AccountLifecycle: lifecycle, Clock: clock, Policy: policy}, optional, authConfig)
}

// NewProductContextRefreshService は Product context refresh 用の canonical lifecycle adapter を生成する。
//
// 役割:
//   - runtime/container が `NewTokenService` を直接呼ばず、context refresh contract 用の生成入口へ依存できるようにする。
//   - Product account auth lifecycle owner の RefreshAccountSession を既存 Product HTTP DTO contract へ写像する。
//
// 引数:
//   - lifecycle: Product account auth の canonical lifecycle owner。
//
// 戻り値:
//   - ProductContextRefreshService: HTTP adapter が要求する context refresh contract 実装。
func NewProductContextRefreshService(lifecycle ProductAccountLifecycle) ProductContextRefreshService {
	// Step 1: nil lifecycle は handler 側で fail-close できる adapter として保持し、container の依存不足を panic にしない。
	return productContextRefreshLifecycleAdapter{lifecycle: lifecycle}
}

// NewProductSessionService は Product session 管理 route 用の canonical lifecycle adapter を生成する。
//
// 役割:
//   - legacy root SessionService を使わず、AccountSessionService が所有する session metadata / refresh state へ委譲する。
//   - ProductSessionService contract を維持しつつ、HTTP adapter から canonical auth package の具体型依存を隠す。
//   - nil lifecycle は method 実行時に ErrInternalError へ fail-close し、legacy store fallback を作らない。
func NewProductSessionService(lifecycle ProductAccountLifecycle) ProductSessionService {
	// Step 1: Product HTTP adapter が必要とする List/Revoke/RevokeOthers だけを公開する adapter を返す。
	return productSessionLifecycleAdapter{lifecycle: lifecycle}
}

type productContextRefreshLifecycleAdapter struct {
	lifecycle ProductAccountLifecycle
}

type productSessionLifecycleAdapter struct {
	lifecycle ProductAccountLifecycle
}

func (adapter productSessionLifecycleAdapter) List(ctx context.Context, accountID domain.AccountID) ([]SessionMetadata, error) {
	// Step 1: lifecycle 欠落時は legacy SessionStore へ戻らず fail-closed にする。
	if adapter.lifecycle == nil {
		return nil, ErrInternalError
	}

	// Step 2: canonical Product account auth metadata を root facade DTO へ写像し、HTTP adapter の既存 contract を保つ。
	sessions, err := adapter.lifecycle.ListAccountSessions(ctx, accountID)
	if err != nil {
		return nil, mapProductAccountLifecycleError(err)
	}
	return productSessionMetadataListToRoot(sessions), nil
}

func (adapter productSessionLifecycleAdapter) Revoke(ctx context.Context, accountID domain.AccountID, sessionID string) error {
	// Step 1: lifecycle 欠落時は legacy store 操作へ進まず fail-closed にする。
	if adapter.lifecycle == nil {
		return ErrInternalError
	}

	// Step 2: 対象 session の metadata と refresh state の削除は canonical lifecycle に委譲する。
	if err := adapter.lifecycle.RevokeAccountSession(ctx, RevokeAccountSessionInput{AccountID: accountID, SessionID: sessionID}); err != nil {
		if errors.Is(err, ErrAccountAuthUnauthorized) {
			return ErrBadRequest
		}
		return mapProductAccountLifecycleError(err)
	}
	return nil
}

func (adapter productSessionLifecycleAdapter) RevokeOthers(ctx context.Context, accountID domain.AccountID, currentSessionID string) error {
	// Step 1: lifecycle 欠落時は legacy store 操作へ進まず fail-closed にする。
	if adapter.lifecycle == nil {
		return ErrInternalError
	}

	// Step 2: 現在 session 以外の失効を canonical lifecycle に委譲し、refresh state と metadata を同期して消す。
	if err := adapter.lifecycle.RevokeOtherAccountSessions(ctx, accountID, currentSessionID); err != nil {
		return mapProductAccountLifecycleError(err)
	}
	return nil
}

func productSessionMetadataListToRoot(sessions []SessionMetadata) []SessionMetadata {
	// Step 1: nil slice は空一覧として扱い、HTTP adapter が安定して JSON array を組み立てられるようにする。
	result := make([]SessionMetadata, 0, len(sessions))
	for _, session := range sessions {
		// Step 2: Product auth package の DTO を root facade DTO に写像し、adapter から concept package 型を隠す。
		result = append(result, SessionMetadata{SessionID: session.SessionID, AccountID: session.AccountID, DeviceName: session.DeviceName, LoginAt: session.LoginAt, LastActiveAt: session.LastActiveAt, IPHash: session.IPHash})
	}
	return result
}

func (adapter productContextRefreshLifecycleAdapter) RefreshContextWithAccountID(ctx context.Context, authContextID string, refreshToken string, clientIP string, userAgent string) (ContextRefreshSession, error) {
	// Step 1: canonical lifecycle が未注入の場合は store 操作へ進まず fail-closed にする。
	if adapter.lifecycle == nil {
		return ContextRefreshSession{}, ErrInternalError
	}

	// Step 2: Product の authContextID は現行 session selector と同じ ULID なので、canonical refresh input の SessionID として渡す。
	result, err := adapter.lifecycle.RefreshAccountSession(ctx, RefreshAccountSessionInput{RefreshToken: refreshToken, SessionID: authContextID, ClientIP: clientIP, UserAgent: userAgent})
	if err != nil {
		return ContextRefreshSession{}, mapProductAccountLifecycleError(err)
	}

	// Step 3: 既存 HTTP adapter が期待する root DTO へ写像するが、refresh rotation と token 発行の実処理は canonical lifecycle owner で完了済みとする。
	return ContextRefreshSession{
		RequestID:     result.RequestID,
		AccountID:     result.Session.AccountID,
		SessionID:     result.Session.SessionID,
		AuthContextID: result.Session.SessionID,
		AccessToken:   result.Session.AccessToken,
		RefreshToken:  result.RefreshCookie.Value,
		ExpiresAt:     result.Session.ExpiresAt,
	}, nil
}
