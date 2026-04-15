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
	RequestID       string
	Challenge       string
	ChallengeID     string
	WebAuthnRPID    string
	WebAuthnOptions []byte // JSON-encoded PublicKeyCredentialRequestOptions / CreationOptions from provider (nil when no provider)
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

// PasskeyCredentialDTO はユースケース層が公開するパスキー情報の DTO。
type PasskeyCredentialDTO struct {
	ID         string
	AccountID  string
	Identifier string
	CreatedAt  time.Time
}

// WebAuthnAssertionResponseDTO は WebAuthn login ceremony の authenticatorData/signature 等。
type WebAuthnAssertionResponseDTO struct {
	ClientDataJSON    string
	AuthenticatorData string
	Signature         string
	UserHandle        string
}

// WebAuthnAssertionCredentialDTO は WebAuthn login credential (navigator.credentials.get 結果)。
type WebAuthnAssertionCredentialDTO struct {
	ID                      string
	RawID                   string
	Type                    string
	Response                WebAuthnAssertionResponseDTO
	AuthenticatorAttachment string
}

// WebAuthnAttestationResponseDTO は WebAuthn registration ceremony の attestationObject 等。
type WebAuthnAttestationResponseDTO struct {
	ClientDataJSON    string
	AttestationObject string
	Transports        []string
}

// WebAuthnAttestationCredentialDTO は WebAuthn registration credential (navigator.credentials.create 結果)。
type WebAuthnAttestationCredentialDTO struct {
	ID                      string
	RawID                   string
	Type                    string
	Response                WebAuthnAttestationResponseDTO
	AuthenticatorAttachment string
}

type StartPasskeyAuthenticationInput struct {
	Identifier string
	ClientIP   string
}

type FinishPasskeyAuthenticationInput struct {
	Credential WebAuthnAssertionCredentialDTO
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

type StartPasskeyRegistrationInput struct {
	RecoverySession   string
	InvitationSession string
	ClientIP          string
}

type RegisterPasskeyInput struct {
	RecoverySession   string
	InvitationSession string
	Credential        WebAuthnAttestationCredentialDTO
	ClientIP          string
}

type InvitationPasskeyRegistrationInput struct {
	InvitationSession string
	Credential        WebAuthnAttestationCredentialDTO
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
	// SavePasskeyOtp は OTP → accountID のマッピングを TTL 付きで保存する。
	SavePasskeyOtp(ctx context.Context, otpKey string, accountID string, ttl time.Duration) error
	// ConsumePasskeyOtp は OTP を検証し accountID を取得する。TTL 切れ・存在しない場合は domain.ErrOtpNotFound を返す。
	ConsumePasskeyOtp(ctx context.Context, otpKey string) (string, error)
	// GetPasskeyOtp は OTP を消費せずに accountID を取得する。TTL 切れ・存在しない場合は domain.ErrOtpNotFound を返す。
	GetPasskeyOtp(ctx context.Context, otpKey string) (string, error)
}

type AuthAccountRepository interface {
	FindByIdentifier(context.Context, string) (domain.AuthAccount, error)
	FindByCredential(context.Context, string) (domain.AuthAccount, error)
	FindByEmail(context.Context, string) (domain.AuthAccount, error)
	// AddPasskey は既存パスキーを保持したまま 1 件追加する。
	// credData に WebAuthn credential record のデータを渡す（provider なしの場合は zero value で可）。
	AddPasskey(ctx context.Context, accountID string, credentialID string, handle string, credData domain.WebAuthnCredentialData) (domain.AuthAccount, error)
	// ListPasskeys は accountID に紐づく全 passkey credential を返す。
	ListPasskeys(ctx context.Context, accountID string) ([]domain.PasskeyCredential, error)
	// DeletePasskeyByID は account_id と credentialID で絞り込んで削除する。
	DeletePasskeyByID(ctx context.Context, accountID string, credentialID string) error
	// FindWebAuthnCredential は credentialHandle（base64url rawID）から WebAuthn stored credential を返す。
	// FinishLogin 時の署名検証に必要な public key 等を提供する。
	FindWebAuthnCredential(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)
	// UpdateWebAuthnCredentialState は FinishLogin 成功後に credential の SignCount と BackupState を更新する。
	// SignCount はリプレイ攻撃検出に使用するため、login 成功のたびに最新値へ更新する必要がある。
	UpdateWebAuthnCredentialState(ctx context.Context, handle string, newSignCount uint32, newBackupState bool) error
}

// WebAuthnProvider は WebAuthn ceremony を実行するアダプタインターフェース。
// 実装は internal/app に置き、go-webauthn/webauthn ライブラリを使用する。
type WebAuthnProvider interface {
	// BeginLogin は認証セレモニーを開始し、challengeKey と PublicKeyCredentialRequestOptions の JSON bytes を返す。
	// challengeKey は provider 内部の session lookup key。
	BeginLogin(ctx context.Context, identifier string) (challengeKey string, optionsJSON []byte, err error)
	// FinishLogin は credential を検証し、一致する credential handle と
	// 更新された SignCount・BackupState を返す（DB への永続化は caller 責務）。
	// signCountUpdated が true のときのみ newSignCount/newBackupState が有効な値（DB 更新すべき値）。
	// false の場合は updatedCred が取得できなかったため DB 更新はスキップすること。
	// challengeKey は BeginLogin が返した値を渡す（空文字列の場合は clientDataJSON から自己解決）。
	// lookupCredential は credentialHandle から DB に保存された credential record を取得するコールバック。
	// provider は lookupCredential を使って public key を取得し、full signature verification を行う。
	FinishLogin(ctx context.Context, challengeKey string, credential WebAuthnAssertionCredentialDTO,
		lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)) (credentialHandle string, newSignCount uint32, newBackupState bool, signCountUpdated bool, err error)
	// BeginRegistration は登録セレモニーを開始し、challengeKey と PublicKeyCredentialCreationOptions の JSON bytes を返す。
	BeginRegistration(ctx context.Context, accountID string) (challengeKey string, optionsJSON []byte, err error)
	// FinishRegistration は credential を検証し、credential handle と WebAuthn credential data を返す。
	// challengeKey が空文字列の場合は clientDataJSON から challenge を自己解決する。
	FinishRegistration(ctx context.Context, challengeKey string, accountID string, credential WebAuthnAttestationCredentialDTO) (credentialHandle string, credData domain.WebAuthnCredentialData, err error)
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
	webauthn            WebAuthnProvider
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

// UseWebAuthnProvider は WebAuthn provider を注入する（app 層から呼び出す）。
// provider が nil の場合はすべての WebAuthn 操作が ErrInternalError を返す。
func (s *AuthService) UseWebAuthnProvider(provider WebAuthnProvider) {
	s.webauthn = provider
}
