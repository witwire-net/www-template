package application

import (
	"context"
	"time"

	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

type AuthSession struct {
	RequestID           string
	AccountID           string
	PasskeyCredentialID string
	SessionID           string
	AccessToken         string
	RefreshToken        string
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
	Kind               domain.TokenKind
	ExpiresAt          time.Time
}

// DeviceLinkIssued は device-link URL の発行結果を表す DTO。
type DeviceLinkIssued struct {
	RequestID string
	Issued    bool
}

type RecoveryDelivery struct {
	RequestID       string
	RecoveryTokenID string
	AccountID       string
	Email           string
	RecoveryURL     string
	Kind            domain.TokenKind
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
	UserAgent  string
}

// StartReauthenticationInput は WebAuthn 再認証セレモニー開始時の入力パラメータを表す。
// AccountID、SessionID、Kind、ClientIP を含む。
type StartReauthenticationInput struct {
	AccountID string
	SessionID string
	Kind      string
	ClientIP  string
}

// FinishReauthenticationInput は WebAuthn 再認証完了時の入力パラメータを表す。
// AccountID、SessionID、Kind、Credential、ClientIP を含む。
type FinishReauthenticationInput struct {
	AccountID  string
	SessionID  string
	Kind       string
	Credential WebAuthnAssertionCredentialDTO
	ClientIP   string
}

// ReauthenticationSession は再認証セッションの結果を表す。
// RequestID、ReauthSessionID、Kind、ExpiresAt を含む。
type ReauthenticationSession struct {
	RequestID       string
	ReauthSessionID string
	Kind            string
	ExpiresAt       time.Time
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
	UserAgent         string
}

type InvitationPasskeyRegistrationInput struct {
	InvitationSession string
	Credential        WebAuthnAttestationCredentialDTO
	ClientIP          string
}

type AuthStateRepository interface {
	SaveChallenge(context.Context, domain.AuthChallenge, time.Duration) error
	ConsumeChallenge(context.Context, string) (domain.AuthChallenge, error)
	IssueRecoveryToken(context.Context, domain.RecoveryToken, time.Duration) error
	SaveRecoveryDeliveryFailure(context.Context, domain.RecoveryDeliveryFailure, time.Duration) error
	// ConsumeRecoveryTokenAtomic は recovery token を tokenID で取得し、secret のハッシュを検証してから削除する。
	ConsumeRecoveryTokenAtomic(ctx context.Context, tokenID string, secret string) (domain.RecoveryToken, error)
	SaveRecoverySession(context.Context, domain.RecoverySession, time.Duration) error
	GetRecoverySession(context.Context, string) (domain.RecoverySession, error)
	ConsumeRecoverySession(context.Context, domain.RecoverySession) error
	IncrementThrottle(context.Context, string, time.Duration) (int, error)
	SetLock(context.Context, string, time.Time, time.Duration) error
	GetLock(context.Context, string) (domain.AuthLock, bool, error)
	// SaveReauthenticationSession は再認証セッションを TTL 付きで保存する。
	SaveReauthenticationSession(ctx context.Context, session domain.ReauthenticationSession, ttl time.Duration) error
	// ConsumeReauthenticationSession は再認証セッションをアトミックに取得・削除する。
	ConsumeReauthenticationSession(ctx context.Context, reauthID string) (domain.ReauthenticationSession, error)
}

type AuthAccountRepository interface {
	FindByIdentifier(context.Context, string) (domain.AuthAccount, error)
	FindByCredential(context.Context, string) (domain.AuthAccount, error)
	FindByEmail(context.Context, string) (domain.AuthAccount, error)
	// FindByID は accountID（ULID）でアカウントを検索する。
	FindByID(ctx context.Context, accountID string) (domain.AuthAccount, error)
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

// SendDeviceLinkSender は device-link URL を secure mail transport で送信するインターフェース。
type SendDeviceLinkSender interface {
	// SendDeviceLink は登録済みメールアドレスへ device-link URL を送信する。
	// delivery は device-link 用の RecoveryDelivery（Kind=device-link）。
	SendDeviceLink(ctx context.Context, delivery RecoveryDelivery) error
}

// RecoveryCompleteSender はパスキー復旧完了通知メールを送信するインターフェース。
type RecoveryCompleteSender interface {
	// SendRecoveryComplete はアカウントのパスキー復旧完了後に通知メールを送信する。
	SendRecoveryComplete(ctx context.Context, accountID, email string) error
}

// DeviceLinkCompleteSender は新規デバイスでのパスキー追加完了通知メールを送信するインターフェース。
type DeviceLinkCompleteSender interface {
	// SendDeviceLinkComplete は新規デバイスでのパスキー追加完了後に通知メールを送信する。
	SendDeviceLinkComplete(ctx context.Context, accountID, email string) error
}

type InvitationPasskeyRegistrar interface {
	RegisterInvitationPasskey(context.Context, InvitationPasskeyRegistrationInput) (AuthSession, error)
}

// RefreshTokenRecord はリフレッシュトークンの永続化に使用するレコード。
// Valkey 上では JSON シリアライズされ、キー auth:refresh:{hash} に保存される。
type RefreshTokenRecord struct {
	AccountID   string
	SessionID   string
	Fingerprint string
	DeviceName  string
	IPHash      string
	IssuedAt    time.Time
}

// SessionMetadata はセッションのメタデータを表す DTO。
// デバイス名、ログイン時刻、最終アクティブ時刻、IP ハッシュを含む。
type SessionMetadata struct {
	SessionID        string
	AccountID        string
	DeviceName       string
	LoginAt          time.Time
	LastActiveAt     time.Time
	IPHash           string
	IsCurrentSession bool
}

// TokenClaims は JWT アクセストークンから抽出したクレームの application DTO。
// domain.Claims を transport / persistence 層から隔離するために使用する。
type TokenClaims struct {
	AccountID string
	SessionID string
	TokenID   string
	IssuedAt  int64
	ExpiresAt int64
}

// RefreshTokenStore はリフレッシュトークンの保存・消費・失効を抽象化するポート。
type RefreshTokenStore interface {
	// Save はリフレッシュトークンハッシュに対応するレコードを保存する。
	// ttl が 0 の場合は無期限（NO EXPIRE）で保存する。
	Save(ctx context.Context, hash string, record RefreshTokenRecord, ttl time.Duration) error
	// Consume は指定したハッシュのリフレッシュトークンをアトミックに取得・削除する。
	// 存在しない場合は domain.ErrRefreshTokenNotFound を返す。
	// 成功時には消費済みキーに記録し、盗難検出のため一定期間保持する。
	Consume(ctx context.Context, hash string) (RefreshTokenRecord, error)
	// GetConsumed は指定したハッシュが既に消費されているか確認する。
	// 消費済みの場合はそのレコードを返し、そうでない場合は domain.ErrSessionNotFound を返す。
	GetConsumed(ctx context.Context, hash string) (RefreshTokenRecord, error)
	// RevokeAllForFingerprint は同一アカウント・同一デバイス指紋の全リフレッシュトークンを失効する。
	RevokeAllForFingerprint(ctx context.Context, accountID, fingerprint string) error
	// RevokeBySessionID は指定されたセッション ID に紐づく全リフレッシュトークンを失効する。
	RevokeBySessionID(ctx context.Context, accountID, sessionID string) error
}

// SessionStore はセッションメタデータの保存・一覧・失効を抽象化するポート。
type SessionStore interface {
	// SaveSession はセッションメタデータを保存する。
	SaveSession(ctx context.Context, sessionID, accountID string, metadata SessionMetadata, ttl time.Duration) error
	// GetSession はセッション ID からメタデータを取得する。
	GetSession(ctx context.Context, sessionID string) (SessionMetadata, error)
	// ListSessions はアカウントに紐づく全セッションのメタデータを返す。
	ListSessions(ctx context.Context, accountID string) ([]SessionMetadata, error)
	// RevokeSession は特定セッションを削除する。
	RevokeSession(ctx context.Context, accountID, sessionID string) error
	// RevokeOthers は現在のセッション以外を全て削除し、削除した session ID のスライスを返す。
	RevokeOthers(ctx context.Context, accountID, currentSessionID string) ([]string, error)
	// RevokeAllForAccount は指定されたアカウントに紐づく全セッションを失効する。
	RevokeAllForAccount(ctx context.Context, accountID string) error
}

// AuditNotifier は認証関連の重要イベントを通知・記録するためのポートである。
// 現時点ではハンドラー注入方式とし、structured logger 導入までの橋渡しとする。
// secret（credential raw data）は含めず、安全な識別子のみを渡す。
type AuditNotifier interface {
	// EmitCredentialStateUpdateFailure は WebAuthn credential state の更新失敗時に呼び出される。
	// credentialHandle は対象 credential の識別子、err は発生したエラー。
	// secret（credential raw data）は含めない。
	EmitCredentialStateUpdateFailure(ctx context.Context, credentialHandle string, err error)
	// EmitDeviceLinkDeliveryFailure は device-link メール送信に失敗したときに呼び出される。
	// requestID は対象リクエスト、accountID は送信対象アカウント、err は送信失敗の原因である。
	// メール本文、トークン、URL などの secret は渡さない。
	EmitDeviceLinkDeliveryFailure(ctx context.Context, requestID string, accountID string, err error)
	// EmitRecoverySessionRevokeFailure は recovery 完了後の全セッション失効に失敗したときに呼び出される。
	// accountID は失効対象アカウント、err は失効失敗の原因である。
	// この通知は監査用であり、呼び出し側は別途 fail-closed のエラー処理を行う。
	EmitRecoverySessionRevokeFailure(ctx context.Context, accountID string, err error)
	// EmitRecoveryCompleteDeliveryFailure は recovery 完了通知メールの送信失敗時に呼び出される。
	// accountID は通知対象アカウント、err は送信失敗の原因である。
	// 通知メールは best-effort のため、この通知自体は認証結果を変更しない。
	EmitRecoveryCompleteDeliveryFailure(ctx context.Context, accountID string, err error)
	// EmitDeviceLinkCompleteDeliveryFailure は device-link 完了通知メールの送信失敗時に呼び出される。
	// accountID は通知対象アカウント、err は送信失敗の原因である。
	// 通知メールは best-effort のため、この通知自体は認証結果を変更しない。
	EmitDeviceLinkCompleteDeliveryFailure(ctx context.Context, accountID string, err error)
}

type AuthService struct {
	stateRepo                AuthStateRepository
	accountRepo              AuthAccountRepository
	recoverySender           AccountRecoverySender
	deviceLinkSender         SendDeviceLinkSender
	recoveryCompleteSender   RecoveryCompleteSender
	deviceLinkCompleteSender DeviceLinkCompleteSender
	invitationRegistrar      InvitationPasskeyRegistrar
	auditNotifier            AuditNotifier
	webauthn                 WebAuthnProvider
	tokenService             *TokenService
	clock                    func() time.Time
	policy                   id.AuthIDPolicy
	authConfig               config.AuthConfig
}

// NewAuthService は AuthService を生成する。
// clock と policy は必須。nil の場合 panic する。
func NewAuthService(stateRepo AuthStateRepository, accountRepo AuthAccountRepository, recoverySender AccountRecoverySender, invitationRegistrar InvitationPasskeyRegistrar, clock func() time.Time, policy id.AuthIDPolicy, authConfig config.AuthConfig) *AuthService {
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

// UseDeviceLinkSender は device-link 送信器を注入する（app 層から呼び出す）。
// sender が nil の場合、executeDeviceLink はメール送信をスキップする（テスト時）。
func (s *AuthService) UseDeviceLinkSender(sender SendDeviceLinkSender) {
	s.deviceLinkSender = sender
}

// UseRecoveryCompleteSender は recovery 完了通知メール送信器を注入する（app 層から呼び出す）。
// sender が nil の場合、RegisterPasskey の後処理で通知メールをスキップする（テスト時）。
func (s *AuthService) UseRecoveryCompleteSender(sender RecoveryCompleteSender) {
	s.recoveryCompleteSender = sender
}

// UseDeviceLinkCompleteSender は device-link 完了通知メール送信器を注入する（app 層から呼び出す）。
// sender が nil の場合、RegisterPasskey の後処理で通知メールをスキップする（テスト時）。
func (s *AuthService) UseDeviceLinkCompleteSender(sender DeviceLinkCompleteSender) {
	s.deviceLinkCompleteSender = sender
}

// UseAuditNotifier は audit notifier を注入する（app 層から呼び出す）。
// notifier が nil の場合、audit event emit はスキップされる（テスト時・未設定時）。
// notifier は認証関連の重要イベントを記録・通知するためのポートである。
// secret（credential raw data）は含めず、accountID・passkeyID・requestID のみを渡す必要がある。
func (s *AuthService) UseAuditNotifier(notifier AuditNotifier) {
	s.auditNotifier = notifier
}

// UseTokenService は TokenService を注入する（app 層から呼び出す）。
// TokenService は JWT ペアの発行・検証・失効を担い、認証の必須コンポーネントである。
// nil を渡した場合、認証フローは内部エラーで失敗する（fail-closed）。
func (s *AuthService) UseTokenService(tokenService *TokenService) {
	s.tokenService = tokenService
}
