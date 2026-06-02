package auth

import (
	"context"
	"time"

	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

// AuthSession は Product 認証 facade が HTTP adapter へ返す認証成功 DTO である。
//
// 役割:
//   - WebAuthn login、recovery、passkey registration の外側 flow から確定した Product account session を transport 層へ渡す。
//   - AccessToken と RefreshToken は response / Cookie へ変換される直前の値であり、永続化やログ出力へ使ってはならない。
//   - AccountID、SessionID、AuthContextID は client が後続 request と refresh context を関連付けるための識別子である。
//
// エラーケース:
//   - この DTO 自体は error を保持しない。発行できない場合は AuthService method が ErrUnauthenticated、ErrInternalError、ErrAccountSuspended などを返す。
type AuthSession struct {
	RequestID           string
	AccountID           domain.AccountID
	PasskeyCredentialID string
	SessionID           string
	AuthContextID       string
	AccessToken         string
	RefreshToken        string
	ExpiresAt           time.Time
}

// PasskeyChallenge は WebAuthn ceremony 開始時に browser へ渡す challenge DTO である。
//
// 役割:
//   - RequestID は監査・応答追跡用、Challenge/ChallengeID は provider session lookup 用の公開識別子である。
//   - WebAuthnRPID と WebAuthnOptions は browser の navigator.credentials API に渡すための値である。
//   - secret は含まず、challenge session の保存・消費は AuthStateRepository / WebAuthnProvider が担当する。
type PasskeyChallenge struct {
	RequestID       string
	Challenge       string
	ChallengeID     string
	WebAuthnRPID    string
	WebAuthnOptions []byte // provider が返す PublicKeyCredentialRequestOptions / CreationOptions の JSON。provider 未設定時は nil になる。
}

// RecoveryAccepted は passkey recovery request を受け付けたことだけを返す DTO である。
//
// 役割:
//   - account の存在有無を外部へ漏らさず、accepted=true/false の安定形状で応答する。
//   - RequestID は配送や throttle 監査と HTTP response を関連付けるために使う。
type RecoveryAccepted struct {
	RequestID string
	Accepted  bool
}

// RecoverySession は recovery token consume 後に発行される一時 registration session DTO である。
//
// 役割:
//   - RecoverySessionID/RecoverySessionRef は passkey registration flow の selector として使う。
//   - Kind は recovery と device-link の違いを domain.TokenKind として保持し、後続通知の文脈を失わない。
//   - ExpiresAt は client と server が一時 session の有効期限を扱うための境界値である。
type RecoverySession struct {
	RequestID          string
	RecoveryTokenID    string
	RecoverySessionID  string
	RecoverySessionRef string
	Kind               domain.TokenKind
	ExpiresAt          time.Time
}

// DeviceLinkIssued は device-link URL の発行結果を表す DTO である。
//
// 役割:
//   - RequestID は送信監査と HTTP response を関連付ける。
//   - Issued は URL 発行が受理されたことを表し、メール配送失敗は audit notifier で扱うためここには詳細を含めない。
type DeviceLinkIssued struct {
	RequestID string
	Issued    bool
}

// RecoveryDelivery は recovery/device-link メール配送 adapter へ渡す配送 intent DTO である。
//
// 役割:
//   - RecoveryURL は利用者に送る URL であり、adapter は本文生成と transport 送信だけを担当する。
//   - AccountID と Email は配送対象を示し、secret や credential raw data は含めない。
//   - Kind と ExpiresAt は文面選択および期限表示に使う。
type RecoveryDelivery struct {
	RequestID       string
	RecoveryTokenID string
	AccountID       domain.AccountID
	Email           string
	RecoveryURL     string
	Kind            domain.TokenKind
	ExpiresAt       time.Time
}

// CompletionDelivery は passkey 登録完了後の通知配送 intent を表す DTO である。
//
// Auth application は AccountID、送信先 email、通知種別だけを生成する。
// AccountSetting.locale と実際の文面選択は account/mailer 側の composition 責務である。
type CompletionDelivery struct {
	AccountID domain.AccountID
	Email     string
	Kind      domain.TokenKind
}

// PasskeyCredentialDTO はユースケース層が公開するパスキー情報の DTO である。
//
// 役割:
//   - パスキー一覧 API が必要とする表示・削除識別子だけを保持する。
//   - credential handle、public key、sign count など認証検証用 secret/内部状態は含めない。
type PasskeyCredentialDTO struct {
	ID         string
	AccountID  domain.AccountID
	Identifier string
	CreatedAt  time.Time
}

// WebAuthnAssertionResponseDTO は WebAuthn login ceremony の assertion response DTO である。
//
// 役割:
//   - browser から受け取った clientDataJSON、authenticatorData、signature、userHandle を application 境界で保持する。
//   - 署名検証や base64url decode は WebAuthnProvider が担当し、この DTO は transport 型からの変換結果だけを表す。
type WebAuthnAssertionResponseDTO struct {
	ClientDataJSON    string
	AuthenticatorData string
	Signature         string
	UserHandle        string
}

// WebAuthnAssertionCredentialDTO は WebAuthn login credential DTO である。
//
// 役割:
//   - navigator.credentials.get の結果を application 境界へ渡すための primitive collection である。
//   - ID/RawID/Type と assertion response を保持し、provider が credential handle を確定する入力になる。
type WebAuthnAssertionCredentialDTO struct {
	ID                      string
	RawID                   string
	Type                    string
	Response                WebAuthnAssertionResponseDTO
	AuthenticatorAttachment string
}

// WebAuthnAttestationResponseDTO は WebAuthn registration ceremony の attestation response DTO である。
//
// 役割:
//   - browser から受け取った clientDataJSON、attestationObject、transports を application 境界で保持する。
//   - attestation 検証と保存可能 credential data への変換は WebAuthnProvider が担当する。
type WebAuthnAttestationResponseDTO struct {
	ClientDataJSON    string
	AttestationObject string
	Transports        []string
}

// WebAuthnAttestationCredentialDTO は WebAuthn registration credential DTO である。
//
// 役割:
//   - navigator.credentials.create の結果を application 境界へ渡すための primitive collection である。
//   - AuthenticatorAttachment は任意値として保持し、provider が必要な場合だけ解釈する。
type WebAuthnAttestationCredentialDTO struct {
	ID                      string
	RawID                   string
	Type                    string
	Response                WebAuthnAttestationResponseDTO
	AuthenticatorAttachment string
}

// StartPasskeyAuthenticationInput は passkey login challenge 開始 use case の入力 DTO である。
//
// 役割:
//   - Identifier は利用者が入力した login identifier で、存在有無を外部へ漏らさず provider/throttle に渡す。
//   - ClientIP は IP/global throttle と lock 判定に使う。
type StartPasskeyAuthenticationInput struct {
	Identifier string
	ClientIP   string
}

// FinishPasskeyAuthenticationInput は passkey login completion use case の入力 DTO である。
//
// 役割:
//   - Credential は WebAuthn assertion の browser 結果、ClientIP/UserAgent は session metadata 生成に使う。
//   - 検証失敗時は AuthService が ErrBadRequest または ErrInternalError へ写像する。
type FinishPasskeyAuthenticationInput struct {
	Credential WebAuthnAssertionCredentialDTO
	ClientIP   string
	UserAgent  string
}

// StartReauthenticationInput は WebAuthn 再認証セレモニー開始時の入力 DTO である。
//
// 役割:
//   - AccountID と SessionID は bearer validation 済み caller を表す。
//   - Kind は device-link など後続 operation の目的を固定し、別用途への reauth session 流用を防ぐ。
//   - ClientIP は reauth challenge 発行の throttle / audit 文脈に使う。
type StartReauthenticationInput struct {
	AccountID domain.AccountID
	SessionID string
	Kind      string
	ClientIP  string
}

// FinishReauthenticationInput は WebAuthn 再認証完了時の入力 DTO である。
//
// 役割:
//   - Credential は browser から返った assertion、AccountID/SessionID/Kind は開始時 session と照合する caller 文脈である。
//   - ClientIP は lock/throttle と監査の補助情報として使う。
type FinishReauthenticationInput struct {
	AccountID  domain.AccountID
	SessionID  string
	Kind       string
	Credential WebAuthnAssertionCredentialDTO
	ClientIP   string
}

// ReauthenticationSession は再認証成功後に発行される一時 session DTO である。
//
// 役割:
//   - ReauthSessionID は後続 mutation が提示する selector であり、AuthService.VerifyReauthSession が consume する。
//   - Kind と ExpiresAt により、operation 種別と短命 session の期限を transport 層へ伝える。
type ReauthenticationSession struct {
	RequestID       string
	ReauthSessionID string
	Kind            string
	ExpiresAt       time.Time
}

// RequestPasskeyRecoveryInput は passkey recovery request use case の入力 DTO である。
//
// 役割:
//   - Email は recovery 対象候補、ClientIP は throttle / lock 判定に使う。
//   - account の存在有無は response へ露出せず、配送可否だけが内部で処理される。
type RequestPasskeyRecoveryInput struct {
	Email    string
	ClientIP string
}

// ConsumeRecoveryTokenInput は recovery URL token consume use case の入力 DTO である。
//
// 役割:
//   - Token は URL から受け取った tokenID.secret 形式の値、ClientIP は throttle/監査文脈である。
//   - token 形式不正、期限切れ、消費済みは ErrBadRequest へ写像される。
type ConsumeRecoveryTokenInput struct {
	Token    string
	ClientIP string
}

// StartPasskeyRegistrationInput は passkey registration challenge 開始 use case の入力 DTO である。
//
// 役割:
//   - RecoverySession と InvitationSession のどちらか一方だけを selector として受け取る。
//   - ClientIP は challenge issuance throttle と監査文脈に使う。
type StartPasskeyRegistrationInput struct {
	RecoverySession   string
	InvitationSession string
	ClientIP          string
}

// RegisterPasskeyInput は passkey registration completion use case の入力 DTO である。
//
// 役割:
//   - RecoverySession/InvitationSession は登録権限を表す一時 selector、Credential は browser attestation 結果である。
//   - ClientIP/UserAgent は発行される Product session metadata の seed として使う。
type RegisterPasskeyInput struct {
	RecoverySession   string
	InvitationSession string
	Credential        WebAuthnAttestationCredentialDTO
	ClientIP          string
	UserAgent         string
}

// InvitationPasskeyRegistrationInput は invitation 経由の passkey 登録 use case 入力 DTO である。
//
// 役割:
//   - InvitationSession は招待 flow の一時 selector、Credential は WebAuthn attestation 結果である。
//   - ClientIP は登録試行の監査と throttle に使う。
type InvitationPasskeyRegistrationInput struct {
	InvitationSession string
	Credential        WebAuthnAttestationCredentialDTO
	ClientIP          string
}

// AuthStateRepository は WebAuthn challenge、recovery token、reauth session、throttle state を扱う port である。
//
// 役割:
//   - Valkey などの一時状態保存実装を AuthService から隠蔽する。
//   - token/secret は必要な primitive だけを受け取り、保存 TTL と atomic consume の境界を repository 実装へ委譲する。
//   - 保存層障害は domain.ErrAuthStoreUnavailable などへ写像され、AuthService が外部 error へ畳む。
type AuthStateRepository interface {
	// SaveChallenge は WebAuthn challenge を TTL 付きで保存する。
	// ctx は呼び出し単位のキャンセル情報、domain.AuthChallenge は保存対象、time.Duration は有効期限である。
	// 保存層障害時は domain.ErrAuthStoreUnavailable などの error を返す。
	SaveChallenge(context.Context, domain.AuthChallenge, time.Duration) error
	// ConsumeChallenge は challenge key から WebAuthn challenge を取得し、再利用を防ぐため消費する。
	// key が存在しない、期限切れ、または保存層が利用できない場合は domain error を返す。
	ConsumeChallenge(context.Context, string) (domain.AuthChallenge, error)
	// IssueRecoveryToken は recovery token を TTL 付きで保存する。
	// token の secret は domain object 内の安全な値として扱い、保存失敗時は error を返す。
	IssueRecoveryToken(context.Context, domain.RecoveryToken, time.Duration) error
	// SaveRecoveryDeliveryFailure は recovery 配送失敗の監査 record を TTL 付きで保存する。
	// 保存できない場合は監査欠落を表す error を返す。
	SaveRecoveryDeliveryFailure(context.Context, domain.RecoveryDeliveryFailure, time.Duration) error
	// ConsumeRecoveryTokenAtomic は recovery token を tokenID で取得し、secret のハッシュを検証してから削除する。
	ConsumeRecoveryTokenAtomic(ctx context.Context, tokenID string, secret string) (domain.RecoveryToken, error)
	// SaveRecoverySession は recovery registration session を TTL 付きで保存する。
	// session は token consume 後の一時 selector であり、保存失敗時は error を返す。
	SaveRecoverySession(context.Context, domain.RecoverySession, time.Duration) error
	// GetRecoverySession は recovery session ID から一時 session を取得する。
	// 不在、期限切れ、保存層障害は domain error として返す。
	GetRecoverySession(context.Context, string) (domain.RecoverySession, error)
	// ConsumeRecoverySession は recovery session を消費済みに更新し、再利用を防ぐ。
	// 更新失敗または不正 session の場合は error を返す。
	ConsumeRecoverySession(context.Context, domain.RecoverySession) error
	// IncrementThrottle は指定 key の throttle counter を window 内で増やし、現在値を返す。
	// 保存層障害時は counter 値ではなく error を返す。
	IncrementThrottle(context.Context, string, time.Duration) (int, error)
	// SetLock は subject/IP などの lock key に解除時刻と TTL を保存する。
	// 保存層障害時は error を返す。
	SetLock(context.Context, string, time.Time, time.Duration) error
	// GetLock は lock key の現在状態を取得する。
	// 戻り値は lock、存在有無、保存層 error であり、不在時は bool=false を返す。
	GetLock(context.Context, string) (domain.AuthLock, bool, error)
	// SaveReauthenticationSession は再認証セッションを TTL 付きで保存する。
	SaveReauthenticationSession(ctx context.Context, session domain.ReauthenticationSession, ttl time.Duration) error
	// ConsumeReauthenticationSession は再認証セッションをアトミックに取得・削除する。
	ConsumeReauthenticationSession(ctx context.Context, reauthID string) (domain.ReauthenticationSession, error)
}

// PasskeyAccountRepository は Auth facade が Account.Auth projection を取得・更新するための port である。
//
// 実装は永続化技術を隠蔽し、認証処理に必要な account identifier、email、status、
// session_revoked_after、passkey credential だけを返す。Product AccountSetting は
// flat 構造の packages/backend/internal/domain が所有するため、この port では扱わない。
type PasskeyAccountRepository interface {
	// FindByIdentifier は login identifier から Account.Auth projection を取得する。
	FindByIdentifier(context.Context, string) (domain.AccountAuth, error)
	// FindByCredential は credential handle から Account.Auth projection を取得する。
	FindByCredential(context.Context, string) (domain.AccountAuth, error)
	// FindByEmail は email から Account.Auth projection を取得する。
	FindByEmail(context.Context, string) (domain.AccountAuth, error)
	// FindByID は accountID（ULID）でアカウントを検索する。
	FindByID(ctx context.Context, accountID domain.AccountID) (domain.AccountAuth, error)
	// AddPasskey は既存パスキーを保持したまま 1 件追加する。
	// credData に WebAuthn credential record のデータを渡す（provider なしの場合は zero value で可）。
	AddPasskey(ctx context.Context, accountID domain.AccountID, credentialID string, handle string, credData domain.WebAuthnCredentialData) (domain.AccountAuth, error)
	// ListPasskeys は accountID に紐づく全 passkey credential を返す。
	ListPasskeys(ctx context.Context, accountID domain.AccountID) ([]domain.PasskeyCredential, error)
	// DeletePasskeyByID は account_id と credentialID で絞り込んで削除する。
	DeletePasskeyByID(ctx context.Context, accountID domain.AccountID, credentialID string) error
	// FindWebAuthnCredential は credentialHandle（base64url rawID）から WebAuthn stored credential を返す。
	// FinishLogin 時の署名検証に必要な public key 等を提供する。
	FindWebAuthnCredential(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)
	// UpdateWebAuthnCredentialState は FinishLogin 成功後に credential の SignCount と BackupState を更新する。
	// SignCount はリプレイ攻撃検出に使用するため、login 成功のたびに最新値へ更新する必要がある。
	UpdateWebAuthnCredentialState(ctx context.Context, handle string, newSignCount uint32, newBackupState bool) error
}

// WebAuthnProvider は WebAuthn ceremony を実行する adapter port である。
//
// 役割:
//   - AuthService が go-webauthn などの外部ライブラリに直接依存しないよう、challenge 開始と credential 検証だけを抽象化する。
//   - login / registration ともに provider が browser options JSON と credential handle / credential data を返す。
//   - 検証失敗や provider 障害は error として返し、AuthService が ErrBadRequest / ErrInternalError へ写像する。
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
	BeginRegistration(ctx context.Context, accountID domain.AccountID) (challengeKey string, optionsJSON []byte, err error)
	// FinishRegistration は credential を検証し、credential handle と WebAuthn credential data を返す。
	// challengeKey が空文字列の場合は clientDataJSON から challenge を自己解決する。
	FinishRegistration(ctx context.Context, challengeKey string, accountID domain.AccountID, credential WebAuthnAttestationCredentialDTO) (credentialHandle string, credData domain.WebAuthnCredentialData, err error)
}

// AccountRecoverySender は account recovery URL を利用者へ配送する port である。
//
// 役割:
//   - AuthService は配送 intent を作るだけで、SMTP や template などの詳細をこの port の実装へ委譲する。
//   - error は配送基盤失敗を表し、request flow では情報漏洩しない stable error へ変換される。
type AccountRecoverySender interface {
	// SendAccountRecovery は recovery URL 配送 intent を利用者へ送信する。
	// ctx は配送単位のキャンセル情報、RecoveryDelivery は宛先・URL・期限を含む DTO である。
	// 送信基盤障害や template 生成失敗がある場合は error を返す。
	SendAccountRecovery(context.Context, RecoveryDelivery) error
}

// SendDeviceLinkSender は device-link URL を secure mail transport で送信する port である。
//
// 役割:
//   - device-link 発行 flow で作成された RecoveryDelivery をメール adapter へ渡す。
//   - AuthService は送信失敗を best-effort audit に留め、response に secret や配送失敗詳細を露出しない。
type SendDeviceLinkSender interface {
	// SendDeviceLink は登録済みメールアドレスへ device-link URL を送信する。
	// delivery は device-link 用の RecoveryDelivery（Kind=device-link）。
	SendDeviceLink(ctx context.Context, delivery RecoveryDelivery) error
}

// RecoveryCompleteSender はパスキー復旧完了通知メールを送信する port である。
//
// 役割:
//   - recovery flow 完了後に利用者へ通知する intent を配送 adapter へ渡す。
//   - 通知は best-effort であり、送信失敗は AuditNotifier へ通知されるが認証結果自体は変更しない。
type RecoveryCompleteSender interface {
	// SendRecoveryComplete はアカウントのパスキー復旧完了後に通知メールを送信する。
	SendRecoveryComplete(ctx context.Context, delivery CompletionDelivery) error
}

// DeviceLinkCompleteSender は新規デバイスでのパスキー追加完了通知メールを送信する port である。
//
// 役割:
//   - device-link registration 完了後に利用者へ通知する intent を配送 adapter へ渡す。
//   - 通知失敗時も session issuance を巻き戻さず、AuditNotifier で監査可能にする。
type DeviceLinkCompleteSender interface {
	// SendDeviceLinkComplete は新規デバイスでのパスキー追加完了後に通知メールを送信する。
	SendDeviceLinkComplete(ctx context.Context, delivery CompletionDelivery) error
}

// InvitationPasskeyRegistrar は invitation session を使った passkey 登録処理を抽象化する port である。
//
// 役割:
//   - recovery/device-link 以外の招待 flow を AuthService facade から分離する。
//   - 実装は InvitationPasskeyRegistrationInput を検証し、成功時に AuthSession を返す。
//   - 招待 flow を利用しない runtime では拒否実装を注入し、ErrBadRequest で fail-closed にできる。
type InvitationPasskeyRegistrar interface {
	// RegisterInvitationPasskey は invitation session と WebAuthn credential を検証し、Product AuthSession を発行する。
	// 招待 selector 不正、credential 検証失敗、保存層障害の場合は AuthService と同じ stable error へ写像可能な error を返す。
	RegisterInvitationPasskey(context.Context, InvitationPasskeyRegistrationInput) (AuthSession, error)
}

// ContextRefreshSession は context-scoped refresh rotation の成功結果を表す DTO である。
//
// 役割:
//   - HTTP adapter が Cookie mode / Bearer mode の response body と Cookie rotation を組み立てるための値をまとめる。
//   - refreshToken 平文は Bearer mode response または HttpOnly Set-Cookie へ渡す直前だけ保持し、永続化やログ用途には使わない。
//
// 戻り値:
//   - ProductContextRefreshService が accessToken、refreshToken、account/session/authContext metadata を返す。
//
// エラーケース:
//   - rotation 失敗時はこの DTO を返さず、ErrInternalError、ErrUnauthenticated、ErrAccountSuspended などを返す。
type ContextRefreshSession struct {
	RequestID     string
	AccountID     domain.AccountID
	SessionID     string
	AuthContextID string
	AccessToken   string
	RefreshToken  string
	ExpiresAt     time.Time
}

// TokenClaims は JWT アクセストークンから抽出した claims の application DTO である。
//
// 役割:
//   - domain token primitive を transport / persistence 層へ直接露出せず、adapter が必要な AccountID・SessionID・TokenID・期限だけを受け取れるようにする。
//   - IssuedAt / ExpiresAt は Unix time として扱い、署名検証や eligibility 判定は ProductAccountLifecycle が担う。
//   - 不正 token の場合、この DTO は生成されず ErrUnauthenticated などが返る。
type TokenClaims struct {
	AccountID domain.AccountID
	SessionID string
	TokenID   string
	IssuedAt  int64
	ExpiresAt int64
}

// AuditNotifier は認証関連の重要イベントを通知・記録するための port である。
//
// 役割:
//   - WebAuthn credential state 更新失敗、device-link/recovery 配送失敗、session revoke 失敗などを安全な識別子だけで記録する。
//   - secret、メール本文、token、credential raw data は渡さず、監査・運用検知に必要な情報だけを扱う。
//   - notifier 自体の副作用は認証結果を変更せず、best-effort の観測境界として使う。
type AuditNotifier interface {
	// EmitCredentialStateUpdateFailure は WebAuthn credential state の更新失敗時に呼び出される。
	// credentialHandle は対象 credential の識別子、err は発生したエラー。
	// secret（credential raw data）は含めない。
	EmitCredentialStateUpdateFailure(ctx context.Context, credentialHandle string, err error)
	// EmitDeviceLinkDeliveryFailure は device-link メール送信に失敗したときに呼び出される。
	// requestID は対象リクエスト、accountID は送信対象アカウント、err は送信失敗の原因である。
	// メール本文、トークン、URL などの secret は渡さない。
	EmitDeviceLinkDeliveryFailure(ctx context.Context, requestID string, accountID domain.AccountID, err error)
	// EmitRecoverySessionRevokeFailure は recovery 完了後の全セッション失効に失敗したときに呼び出される。
	// accountID は失効対象アカウント、err は失効失敗の原因である。
	// この通知は監査用であり、呼び出し側は別途 fail-closed のエラー処理を行う。
	EmitRecoverySessionRevokeFailure(ctx context.Context, accountID domain.AccountID, err error)
	// EmitRecoveryCompleteDeliveryFailure は recovery 完了通知メールの送信失敗時に呼び出される。
	// accountID は通知対象アカウント、err は送信失敗の原因である。
	// 通知メールは best-effort のため、この通知自体は認証結果を変更しない。
	EmitRecoveryCompleteDeliveryFailure(ctx context.Context, accountID domain.AccountID, err error)
	// EmitDeviceLinkCompleteDeliveryFailure は device-link 完了通知メールの送信失敗時に呼び出される。
	// accountID は通知対象アカウント、err は送信失敗の原因である。
	// 通知メールは best-effort のため、この通知自体は認証結果を変更しない。
	EmitDeviceLinkCompleteDeliveryFailure(ctx context.Context, accountID domain.AccountID, err error)
}

// ProductAccountLifecycle は Product account auth の canonical lifecycle owner へ AuthService facade が委譲するための contract である。
//
// 役割:
//   - WebAuthn/recovery など AuthService facade が保持する outer flow と、token/session issuance・refresh・bearer validation の true owner を分離する。
//   - production caller が root TokenService へ戻らず `internal/application/auth` の AccountSessionService を通ることを型で固定する。
//   - facade は transport 互換 DTO への写像だけを行い、token primitive や refresh family の実装を所有しない。
type ProductAccountLifecycle interface {
	// IssueAccountSession は確定済み AccountID から Product account session を発行する。
	// input には AccountID と device metadata seed を渡し、成功時は accessToken と refresh Cookie command を含む AccountSessionResult を返す。
	// Account 不在、停止、保存層障害、署名失敗時は error を返す。
	IssueAccountSession(context.Context, IssueAccountSessionInput) (AccountSessionResult, error)
	// AuthorizeAccountSession は bearer accessToken を検証し、caller session DTO を返す。
	// token 不正、session 不在、Account eligibility 不一致、保存層障害の場合は error を返す。
	AuthorizeAccountSession(context.Context, string) (ValidatedSession, error)
	// ListAccountSessions は AccountID に紐づく Product session metadata 一覧を返す。
	// 保存層障害または AccountID 不正の場合は error を返す。
	ListAccountSessions(context.Context, domain.AccountID) ([]SessionMetadata, error)
	// RevokeAccountSession は AccountID と SessionID で指定した Product session を失効する。
	// 所有者不一致、session 不在、保存層障害の場合は error を返す。
	RevokeAccountSession(context.Context, RevokeAccountSessionInput) error
	// RevokeOtherAccountSessions は currentSessionID 以外の Product session を失効する。
	// refresh state と session metadata の削除に失敗した場合は error を返す。
	RevokeOtherAccountSessions(context.Context, domain.AccountID, string) error
	// RevokeAllAccountSessions は AccountID に紐づく Product session をすべて失効する。
	// recovery 完了後などの強制 revoke で使用し、保存層障害時は error を返す。
	RevokeAllAccountSessions(context.Context, domain.AccountID) error
	// RefreshAccountSession は refresh credential と auth context から Product session を rotation する。
	// refresh token 不正、reuse 検出、Account 停止、保存層障害の場合は error を返す。
	RefreshAccountSession(context.Context, RefreshAccountSessionInput) (AccountRefreshResult, error)
}

// AuthServiceDependencies は root AuthService の必須依存を constructor 時点で明示する DTO である。
//
// 役割:
//   - token/session lifecycle の canonical owner を ProductAccountLifecycle として必須化し、Use* mutator による後付け注入を廃止する。
//   - clock / ID policy / repository 欠落を panic や method-time nil check ではなく error-returning constructor で fail-closed にする。
//   - Optional port と必須 port を型で分け、nil 許容の意図を review 可能にする。
type AuthServiceDependencies struct {
	StateRepo           AuthStateRepository
	AccountRepo         PasskeyAccountRepository
	RecoverySender      AccountRecoverySender
	InvitationRegistrar InvitationPasskeyRegistrar
	AccountLifecycle    ProductAccountLifecycle
	Clock               func() time.Time
	Policy              id.AuthIDPolicy
}

// AuthServiceOptionalPorts は root AuthService の nil 許容 port をまとめる DTO である。
//
// 役割:
//   - WebAuthn provider や配送 observer など、テストや限定 flow で意図的に省略される依存を明示する。
//   - nil は「機能利用時に fail-close または best-effort skip」として扱い、必須依存欠落と混同しない。
//   - production container は必要な port をこの DTO に詰めて渡し、後続 mutator による状態変更を不要にする。
type AuthServiceOptionalPorts struct {
	WebAuthn                 WebAuthnProvider
	DeviceLinkSender         SendDeviceLinkSender
	RecoveryCompleteSender   RecoveryCompleteSender
	DeviceLinkCompleteSender DeviceLinkCompleteSender
	AuditNotifier            AuditNotifier
}

// AuthService は Product passkey/recovery outer flow を扱う認証 facade である。
//
// 役割:
//   - WebAuthn challenge、recovery token、reauth、passkey 管理などの外側 flow を提供する。
//   - session issuance / refresh / bearer validation は ProductAccountLifecycle へ委譲し、legacy token/session service fallback を持たない。
//   - 必須依存は NewAuthService の constructor-time dependency injection で検証し、省略可能 port は AuthServiceOptionalPorts に限定する。
//
// エラーケース:
//   - 依存不足、保存層障害、WebAuthn provider 障害は ErrInternalError、認証拒否や入力不備は ErrBadRequest / ErrUnauthenticated などへ写像する。
type AuthService struct {
	stateRepo                AuthStateRepository
	accountRepo              PasskeyAccountRepository
	recoverySender           AccountRecoverySender
	deviceLinkSender         SendDeviceLinkSender
	recoveryCompleteSender   RecoveryCompleteSender
	deviceLinkCompleteSender DeviceLinkCompleteSender
	invitationRegistrar      InvitationPasskeyRegistrar
	auditNotifier            AuditNotifier
	webauthn                 WebAuthnProvider
	accountLifecycle         ProductAccountLifecycle
	clock                    func() time.Time
	policy                   id.AuthIDPolicy
	authConfig               config.AuthConfig
}

// NewAuthService は AuthService を constructor-time dependency injection で生成する。
//
// 役割:
//   - 必須依存をすべて先に検証し、clock 欠落などを panic ではなく error として返す。
//   - Product session lifecycle は ProductAccountLifecycle を必須にし、legacy token/session service への fallback を残さない。
//   - nil 許容の observer / optional port は AuthServiceOptionalPorts に限定し、曖昧な後付け mutator を使わない。
//
// 引数:
//   - deps: state/account repository、delivery、canonical lifecycle、clock、ID policy の必須依存。
//   - optional: WebAuthn provider や audit notifier など nil 許容の明示 port。
//   - authConfig: throttle / TTL / RP ID など Product auth runtime 設定。
//
// 戻り値:
//   - *AuthService: 検証済み依存だけを保持する facade。
//   - error: 必須依存が欠けている場合は ErrInternalError。
func NewAuthService(deps AuthServiceDependencies, optional AuthServiceOptionalPorts, authConfig config.AuthConfig) (*AuthService, error) {
	// Step 1: 必須依存を一括検証し、構築後の method が nil port で panic/fail-open しないようにする。
	if err := validateAuthServiceDependencies(deps); err != nil {
		return nil, err
	}

	// Step 2: optional DTO の nil は意図的な省略として保持し、各 flow の fail-close/best-effort 分岐で扱う。
	return &AuthService{
		stateRepo:                deps.StateRepo,
		accountRepo:              deps.AccountRepo,
		recoverySender:           deps.RecoverySender,
		deviceLinkSender:         optional.DeviceLinkSender,
		recoveryCompleteSender:   optional.RecoveryCompleteSender,
		deviceLinkCompleteSender: optional.DeviceLinkCompleteSender,
		invitationRegistrar:      deps.InvitationRegistrar,
		auditNotifier:            optional.AuditNotifier,
		webauthn:                 optional.WebAuthn,
		accountLifecycle:         deps.AccountLifecycle,
		clock:                    deps.Clock,
		policy:                   deps.Policy,
		authConfig:               authConfig,
	}, nil
}

func validateAuthServiceDependencies(deps AuthServiceDependencies) error {
	// Step 1: challenge/recovery state repository がない場合は公開 auth flow を安全に実行できないため拒否する。
	if deps.StateRepo == nil {
		return ErrInternalError
	}
	// Step 2: account auth repository がない場合は WebAuthn/recovery の account 解決ができないため拒否する。
	if deps.AccountRepo == nil {
		return ErrInternalError
	}
	// Step 3: recovery sender がない場合は recovery request を受け付けても配送できないため構成を拒否する。
	if deps.RecoverySender == nil {
		return ErrInternalError
	}
	// Step 4: invitation registrar がない場合は招待 passkey flow の境界が不明になるため拒否する。
	if deps.InvitationRegistrar == nil {
		return ErrInternalError
	}
	// Step 5: Product account lifecycle がない場合は session issuance/refresh/revoke を legacy path に戻せないため拒否する。
	if deps.AccountLifecycle == nil {
		return ErrInternalError
	}
	// Step 6: clock がない場合は TTL・期限判定が非決定的になるため panic ではなく error で拒否する。
	if deps.Clock == nil {
		return ErrInternalError
	}
	// Step 7: ID policy がない場合は request/session correlation ID を発行できないため拒否する。
	if deps.Policy.New == nil || deps.Policy.Validate == nil {
		return ErrInternalError
	}
	return nil
}
