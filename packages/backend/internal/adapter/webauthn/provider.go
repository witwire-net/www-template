package webauthn

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goprotocol "github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	application "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

// ChallengeStore は WebAuthn challenge session を外部ストレージ（Valkey）に保存するための最小インターフェース。
type ChallengeStore interface {
	// Key は環境プレフィックス付きのキーを構築する。
	Key(parts ...string) string
	// Set は key に value を TTL 付きで保存する。
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	// GetDel は key の値を取得すると同時にアトミックに削除する。
	// キーが存在しない場合はエラーを返す。
	GetDel(ctx context.Context, key string) (string, error)
}

// webAuthnProvider は go-webauthn/webauthn を使った WebAuthnProvider 実装。
// セレモニーのセッションデータは ChallengeStore に TTL 付きで保存され、
// Finish 時に GetDel でアトミックに取得・削除される。
type webAuthnProvider struct {
	wa           *gowebauthn.WebAuthn
	store        ChallengeStore
	challengeTTL time.Duration
}

// NewWebAuthnProvider は go-webauthn を利用する Product 認証用 WebAuthnProvider を生成する。
//
// 引数:
//   - rpid: ブラウザの WebAuthn ceremony が検証する Relying Party ID。ローカル app では "localhost" を渡す。
//   - origins: WebAuthn ceremony を許可する origin 一覧。ローカル app では ["http://localhost:5174"] を渡す。
//   - challengeTTL: challenge session を外部ストアへ保存する期間。短すぎると ceremony 完了前に失効する。
//   - store: challenge session を保存・取得・削除する外部ストア。nil は許可しない呼び出し側契約とする。
//
// 戻り値:
//   - application.WebAuthnProvider: authentication/register/reauth ceremony を開始・完了する adapter 実装。
//   - error: rpid と origins の組み合わせを go-webauthn が受理できない場合に設定エラーを返す。
//
// 使用例:
//
//	provider, err := NewWebAuthnProvider("localhost", []string{"http://localhost:5174"}, 5*time.Minute, store)
//	if err != nil {
//		return err
//	}
func NewWebAuthnProvider(rpid string, origins []string, challengeTTL time.Duration, store ChallengeStore) (application.WebAuthnProvider, error) {
	rpOrigins := origins
	if len(rpOrigins) == 0 {
		rpOrigins = []string{"https://" + rpid}
	}
	wa, err := gowebauthn.New(&gowebauthn.Config{
		RPID:          rpid,
		RPDisplayName: rpid,
		RPOrigins:     rpOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn: config error: %w", err)
	}
	return &webAuthnProvider{
		wa:           wa,
		store:        store,
		challengeTTL: challengeTTL,
	}, nil
}

// ─── internal user adapter ───────────────────────────────────────────────────

// webAuthnUserAdapter は go-webauthn User interface の最小実装。
type webAuthnUserAdapter struct {
	id          string
	name        string
	displayName string
	credentials []gowebauthn.Credential
}

var _ application.OperatorPasskeyRegistrationProvider = (*webAuthnProvider)(nil)

func (u *webAuthnUserAdapter) WebAuthnID() []byte                           { return []byte(u.id) }
func (u *webAuthnUserAdapter) WebAuthnName() string                         { return u.name }
func (u *webAuthnUserAdapter) WebAuthnDisplayName() string                  { return u.displayName }
func (u *webAuthnUserAdapter) WebAuthnCredentials() []gowebauthn.Credential { return u.credentials }

// ─── BeginLogin ───────────────────────────────────────────────────────────────

func (p *webAuthnProvider) BeginLogin(ctx context.Context, _ string) (challengeKey string, challengeBytes []byte, err error) {
	// discoverable login（resident key）を使うため、BeginDiscoverableLogin を使用する。
	// user verification を required に設定し、UV-less assertion を拒否できるようにする。
	assertion, sessionData, err := p.wa.BeginDiscoverableLogin(gowebauthn.WithUserVerification(goprotocol.VerificationRequired))
	if err != nil {
		return "", nil, fmt.Errorf("webauthn: BeginDiscoverableLogin: %w", err)
	}

	challengeKey = sessionData.Challenge
	if err := p.saveSessionData(ctx, challengeKey, *sessionData); err != nil {
		return "", nil, fmt.Errorf("webauthn: save login session: %w", err)
	}

	jsonBytes, jsonErr := json.Marshal(assertion)
	if jsonErr != nil {
		return "", nil, fmt.Errorf("webauthn: marshal assertion: %w", jsonErr)
	}

	return challengeKey, jsonBytes, nil
}

// BeginOperatorLogin は Admin operator passkey login challenge を開始する。
func (p *webAuthnProvider) BeginOperatorLogin(ctx context.Context, identifier string) (challengeKey string, optionsJSON []byte, err error) {
	// Step 1: Admin login も discoverable credential 前提なので、既存 BeginLogin の full WebAuthn challenge 発行を再利用する。
	return p.BeginLogin(ctx, identifier)
}

// ─── FinishLogin ─────────────────────────────────────────────────────────────

// FinishLogin は challengeKey で Valkey から session を取得・削除し、
// lookupCredential コールバックで DB から公開鍵を取得したうえで
// ValidatePasskeyLogin で full signature verification を行う。
// UV（user verification）が確認できない assertion は無条件に拒否する。
// challengeKey が空文字列の場合は clientDataJSON から自己解決する（後方互換性のため）。
func (p *webAuthnProvider) FinishLogin(ctx context.Context, challengeKey string, credential application.WebAuthnAssertionCredentialDTO,
	lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error),
) (credentialHandle string, newSignCount uint32, newBackupState bool, signCountUpdated bool, err error) {
	lookupKey := challengeKey
	if lookupKey == "" {
		derivedKey, deriveErr := challengeKeyFromClientDataJSON(credential.Response.ClientDataJSON)
		if deriveErr != nil {
			return "", 0, false, false, deriveErr
		}
		lookupKey = derivedKey
	}

	sessionData, sessionErr := p.consumeSessionData(ctx, lookupKey)
	if sessionErr != nil {
		return "", 0, false, false, sessionErr
	}

	parsed, parseErr := dtoToAssertionParsed(credential)
	if parseErr != nil {
		return "", 0, false, false, fmt.Errorf("webauthn: parse assertion: %w", parseErr)
	}

	// UV（user verified）フラグを無条件に検証する。
	if parsed.Response.AuthenticatorData.Flags&goprotocol.FlagUserVerified == 0 {
		return "", 0, false, false, fmt.Errorf("webauthn: user verification required")
	}

	var resolvedHandle string
	var handlerErr error
	handler := p.buildDiscoverableHandler(ctx, &resolvedHandle, &handlerErr, lookupCredential)

	_, updatedCred, validateErr := p.wa.ValidatePasskeyLogin(handler, sessionData, parsed)

	if handlerErr != nil {
		return "", 0, false, false, handlerErr
	}
	if validateErr != nil {
		return "", 0, false, false, fmt.Errorf("webauthn: ValidatePasskeyLogin: %w", validateErr)
	}
	if resolvedHandle == "" {
		return "", 0, false, false, fmt.Errorf("webauthn: could not resolve credential handle from assertion")
	}

	if updatedCred != nil {
		return resolvedHandle, updatedCred.Authenticator.SignCount, updatedCred.Flags.BackupState, true, nil
	}
	return resolvedHandle, 0, false, false, nil
}

// buildDiscoverableHandler は DiscoverableUserHandler を構築する。
// resolvedHandle と handlerErr はクロージャで呼び出し側と共有される。
func (p *webAuthnProvider) buildDiscoverableHandler(
	ctx context.Context,
	resolvedHandle *string,
	handlerErr *error,
	lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error),
) gowebauthn.DiscoverableUserHandler {
	return func(rawID, userHandle []byte) (gowebauthn.User, error) {
		handle := base64.RawURLEncoding.EncodeToString(rawID)
		*resolvedHandle = handle
		uid := string(userHandle)
		if uid == "" {
			uid = handle
		}
		credentials, buildErr := p.buildCredentialsFromDB(ctx, handle, lookupCredential)
		if buildErr != nil {
			*handlerErr = buildErr
			return nil, buildErr
		}
		return &webAuthnUserAdapter{id: uid, name: uid, credentials: credentials}, nil
	}
}

// buildCredentialsFromDB は DB から stored credential を取得して gowebauthn.Credential を構築する。
// DB 障害は error として返す（not-found は空 credentials として継続）。
func (p *webAuthnProvider) buildCredentialsFromDB(
	ctx context.Context,
	handle string,
	lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error),
) ([]gowebauthn.Credential, error) {
	if lookupCredential == nil {
		return nil, nil
	}
	stored, lookupErr := lookupCredential(ctx, handle)
	if lookupErr != nil {
		if !isDomainNotFound(lookupErr) {
			return nil, fmt.Errorf("webauthn: lookupCredential store error: %w", lookupErr)
		}
		return nil, nil // not-found: signature verification で失敗する
	}
	if len(stored.PublicKey) == 0 {
		return nil, nil
	}
	rawIDBytes, _ := base64.RawURLEncoding.DecodeString(handle)
	transports := make([]goprotocol.AuthenticatorTransport, 0, len(stored.Transports))
	for _, t := range stored.Transports {
		transports = append(transports, goprotocol.AuthenticatorTransport(t))
	}
	aaguid := stored.AAGUID
	if len(aaguid) != 16 {
		aaguid = make([]byte, 16)
	}
	return []gowebauthn.Credential{{
		ID:        rawIDBytes,
		PublicKey: stored.PublicKey,
		Authenticator: gowebauthn.Authenticator{
			SignCount: stored.SignCount,
			AAGUID:    aaguid,
		},
		Transport: transports,
		Flags: gowebauthn.CredentialFlags{
			BackupEligible: stored.BackupEligible,
			BackupState:    stored.BackupState,
		},
	}}, nil
}

// isDomainNotFound は domain の not-found 系エラーを判定する。
// lookupCredential の not-found は auth failure として扱い、DB 障害とは区別する。
func isDomainNotFound(err error) bool {
	return errors.Is(err, domain.ErrAccountAuthNotFound)
}

// ─── BeginRegistration ────────────────────────────────────────────────────────

func (p *webAuthnProvider) BeginRegistration(ctx context.Context, accountID domain.AccountID) (challengeKey string, challengeBytes []byte, err error) {
	user := &webAuthnUserAdapter{id: accountID.String(), name: accountID.String(), displayName: accountID.String()}

	// usernameless login と password manager 保存に必要な discoverable credential を必須化する。
	creation, sessionData, err := p.wa.BeginRegistration(user, gowebauthn.WithAuthenticatorSelection(discoverableCredentialSelection()))
	if err != nil {
		return "", nil, fmt.Errorf("webauthn: BeginRegistration: %w", err)
	}

	challengeKey = sessionData.Challenge
	if err := p.saveSessionData(ctx, challengeKey, *sessionData); err != nil {
		return "", nil, fmt.Errorf("webauthn: save registration session: %w", err)
	}

	jsonBytes, jsonErr := json.Marshal(creation)
	if jsonErr != nil {
		return "", nil, fmt.Errorf("webauthn: marshal creation: %w", jsonErr)
	}

	return challengeKey, jsonBytes, nil
}

// BeginOperatorRegistration は Admin operator setup 用の WebAuthn 登録 ceremony を開始する。
func (p *webAuthnProvider) BeginOperatorRegistration(ctx context.Context, input application.OperatorRegistrationChallengeInput) (application.OperatorRegistrationChallenge, error) {
	// Step 1: Admin operator の user handle として OperatorID を使い、Product AccountID へ意味変換しない。
	user := &webAuthnUserAdapter{id: input.OperatorID, name: input.Email, displayName: input.DisplayName}
	creation, sessionData, err := p.wa.BeginRegistration(user, gowebauthn.WithAuthenticatorSelection(discoverableCredentialSelection()))
	if err != nil {
		return application.OperatorRegistrationChallenge{}, fmt.Errorf("webauthn: BeginOperatorRegistration: %w", err)
	}

	// Step 2: HTTP response の requestId と provider session lookup key を一致させ、finish request で同じ requestId を消費できるようにする。
	if err := p.saveSessionData(ctx, input.RequestID, *sessionData); err != nil {
		return application.OperatorRegistrationChallenge{}, fmt.Errorf("webauthn: save operator registration session: %w", err)
	}
	jsonBytes, jsonErr := json.Marshal(creation)
	if jsonErr != nil {
		return application.OperatorRegistrationChallenge{}, fmt.Errorf("webauthn: marshal operator creation: %w", jsonErr)
	}
	return application.OperatorRegistrationChallenge{RequestID: input.RequestID, Challenge: sessionData.Challenge, OptionsJSON: jsonBytes}, nil
}

// discoverableCredentialSelection は登録時に作成する credential の条件を一箇所で定義する。
func discoverableCredentialSelection() goprotocol.AuthenticatorSelection {
	// Step 1: WebAuthn Level 1 由来の requireResidentKey と Level 2 の residentKey を同時に指定し、client 側で discoverable credential 作成要求が欠落しないようにする。
	requireResidentKey := true

	// Step 2: user verification も required にし、登録完了後の検証で UV-less attestation を拒否する方針と一致させる。
	return goprotocol.AuthenticatorSelection{RequireResidentKey: &requireResidentKey, ResidentKey: goprotocol.ResidentKeyRequirementRequired, UserVerification: goprotocol.VerificationRequired}
}

// FinishOperatorRegistration は Admin operator setup 用 attestation を検証し、保存用 credential data を返す。
func (p *webAuthnProvider) FinishOperatorRegistration(ctx context.Context, requestID string, operatorID string, credential application.OperatorWebAuthnAttestationCredential) (application.OperatorPasskeyRegistration, error) {
	// Step 1: requestId で保存済み session を一度だけ消費し、replay による二重 passkey 作成を防ぐ。
	sessionData, sessionErr := p.consumeSessionData(ctx, requestID)
	if sessionErr != nil {
		return application.OperatorPasskeyRegistration{}, sessionErr
	}
	user := &webAuthnUserAdapter{id: operatorID, name: operatorID}

	// Step 2: Admin 専用 DTO を protocol parser が受け付ける JSON へ変換し、go-webauthn に full attestation verification を委譲する。
	parsed, parseErr := dtoToAdminAttestationParsed(credential)
	if parseErr != nil {
		return application.OperatorPasskeyRegistration{}, fmt.Errorf("webauthn: parse operator attestation: %w", parseErr)
	}
	if parsed.Response.AttestationObject.AuthData.Flags&goprotocol.FlagUserVerified == 0 {
		return application.OperatorPasskeyRegistration{}, fmt.Errorf("webauthn: user verification required")
	}
	webauthnCred, createErr := p.wa.CreateCredential(user, sessionData, parsed)
	if createErr != nil {
		return application.OperatorPasskeyRegistration{}, fmt.Errorf("webauthn: CreateOperatorCredential: %w", createErr)
	}

	// Step 3: 保存層へ渡す credential data だけを DTO 化し、public key や sign count を HTTP response へは返さない。
	transports := make([]string, 0, len(webauthnCred.Transport))
	for _, t := range webauthnCred.Transport {
		transports = append(transports, string(t))
	}
	return application.OperatorPasskeyRegistration{CredentialHandle: base64.RawURLEncoding.EncodeToString(webauthnCred.ID), PublicKey: webauthnCred.PublicKey, SignCount: webauthnCred.Authenticator.SignCount, AAGUID: webauthnCred.Authenticator.AAGUID, BackupEligible: webauthnCred.Flags.BackupEligible, BackupState: webauthnCred.Flags.BackupState, Transports: transports}, nil
}

// ─── FinishRegistration ───────────────────────────────────────────────────────

// FinishRegistration は challengeKey で Valkey から session を取得・削除し、
// credential を検証して credential handle と WebAuthn credential data を返す。
// UV（user verification）が確認できない attestation は無条件に拒否する。
// challengeKey が空文字列の場合は clientDataJSON から自己解決する（後方互換性のため）。
func (p *webAuthnProvider) FinishRegistration(ctx context.Context, challengeKey string, accountID domain.AccountID, credential application.WebAuthnAttestationCredentialDTO) (credentialHandle string, credData domain.WebAuthnCredentialData, err error) {
	lookupKey := challengeKey
	if lookupKey == "" {
		derivedKey, deriveErr := challengeKeyFromClientDataJSON(credential.Response.ClientDataJSON)
		if deriveErr != nil {
			return "", domain.ZeroWebAuthnCredentialData(), deriveErr
		}
		lookupKey = derivedKey
	}

	sessionData, sessionErr := p.consumeSessionData(ctx, lookupKey)
	if sessionErr != nil {
		return "", domain.ZeroWebAuthnCredentialData(), sessionErr
	}

	user := &webAuthnUserAdapter{id: accountID.String(), name: accountID.String()}

	parsed, parseErr := dtoToAttestationParsed(credential)
	if parseErr != nil {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: parse attestation: %w", parseErr)
	}

	// UV（user verified）フラグを無条件に検証する。
	if parsed.Response.AttestationObject.AuthData.Flags&goprotocol.FlagUserVerified == 0 {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: user verification required")
	}

	webauthnCred, createErr := p.wa.CreateCredential(user, sessionData, parsed)
	if createErr != nil {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: CreateCredential: %w", createErr)
	}

	handle := base64.RawURLEncoding.EncodeToString(webauthnCred.ID)
	transports := make([]string, 0, len(webauthnCred.Transport))
	for _, t := range webauthnCred.Transport {
		transports = append(transports, string(t))
	}
	cd := domain.NewWebAuthnCredentialData(
		webauthnCred.PublicKey,
		webauthnCred.Authenticator.SignCount,
		webauthnCred.Authenticator.AAGUID,
		webauthnCred.Flags.BackupEligible,
		webauthnCred.Flags.BackupState,
		transports,
	)
	return handle, cd, nil
}

// ─── SessionData 保存・取得ヘルパー ───────────────────────────────────────────

// saveSessionData は go-webauthn SessionData を JSON 化して ChallengeStore に TTL 付きで保存する。
func (p *webAuthnProvider) saveSessionData(ctx context.Context, challengeKey string, sessionData gowebauthn.SessionData) error {
	jsonBytes, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("marshal session data: %w", err)
	}
	key := p.store.Key("wa", "session", challengeKey)
	if storeErr := p.store.Set(ctx, key, string(jsonBytes), p.challengeTTL); storeErr != nil {
		return fmt.Errorf("store session data: %w", storeErr)
	}
	return nil
}

// consumeSessionData は ChallengeStore から SessionData を取得し、アトミックに削除する。
// キーが存在しない場合はエラーを返す。
func (p *webAuthnProvider) consumeSessionData(ctx context.Context, challengeKey string) (gowebauthn.SessionData, error) {
	key := p.store.Key("wa", "session", challengeKey)
	val, err := p.store.GetDel(ctx, key)
	if err != nil {
		return gowebauthn.SessionData{}, fmt.Errorf("webauthn: session not found or expired for challengeKey %q", challengeKey)
	}
	var sessionData gowebauthn.SessionData
	if unmarshalErr := json.Unmarshal([]byte(val), &sessionData); unmarshalErr != nil {
		return gowebauthn.SessionData{}, fmt.Errorf("webauthn: corrupt session data for challengeKey %q", challengeKey)
	}
	return sessionData, nil
}

// challengeKeyFromClientDataJSON は base64url-encoded clientDataJSON を decode して
// JSON の "challenge" フィールド（base64url string）を challengeKey として返す。
// 後方互換性のため、challengeKey が空文字列の場合にのみ使用する。
func challengeKeyFromClientDataJSON(clientDataJSON string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(clientDataJSON)
	if err != nil {
		return "", fmt.Errorf("base64url decode clientDataJSON: %w", err)
	}
	var cd struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(decoded, &cd); err != nil {
		return "", fmt.Errorf("unmarshal clientDataJSON: %w", err)
	}
	if cd.Challenge == "" {
		return "", fmt.Errorf("challenge field is empty in clientDataJSON")
	}
	return cd.Challenge, nil
}

// ─── DTO → protocol struct 変換ヘルパー ───────────────────────────────────────

// dtoToAssertionParsed は WebAuthnAssertionCredentialDTO を
// *protocol.ParsedCredentialAssertionData に変換する。
// JSON シリアライズ経由で ParseCredentialRequestResponseBody を使う。
func dtoToAssertionParsed(dto application.WebAuthnAssertionCredentialDTO) (*goprotocol.ParsedCredentialAssertionData, error) {
	raw := assertionCredentialJSON{
		ID:    dto.ID,
		RawID: dto.RawID,
		Type:  dto.Type,
		Response: assertionResponseJSON{
			ClientDataJSON:    dto.Response.ClientDataJSON,
			AuthenticatorData: dto.Response.AuthenticatorData,
			Signature:         dto.Response.Signature,
			UserHandle:        dto.Response.UserHandle,
		},
		AuthenticatorAttachment: dto.AuthenticatorAttachment,
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return goprotocol.ParseCredentialRequestResponseBody(bytes.NewReader(data))
}

// dtoToAttestationParsed は WebAuthnAttestationCredentialDTO を
// *protocol.ParsedCredentialCreationData に変換する。
func dtoToAttestationParsed(dto application.WebAuthnAttestationCredentialDTO) (*goprotocol.ParsedCredentialCreationData, error) {
	raw := attestationCredentialJSON{
		ID:    dto.ID,
		RawID: dto.RawID,
		Type:  dto.Type,
		Response: attestationResponseJSON{
			ClientDataJSON:    dto.Response.ClientDataJSON,
			AttestationObject: dto.Response.AttestationObject,
			Transports:        dto.Response.Transports,
		},
		AuthenticatorAttachment: dto.AuthenticatorAttachment,
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return goprotocol.ParseCredentialCreationResponseBody(bytes.NewReader(data))
}

func dtoToAdminAttestationParsed(dto application.OperatorWebAuthnAttestationCredential) (*goprotocol.ParsedCredentialCreationData, error) {
	// Step 1: Admin application DTO を existing JSON shim に詰め替え、parser の入力形式を Product registration と揃える。
	raw := attestationCredentialJSON{ID: dto.ID, RawID: dto.RawID, Type: dto.Type, Response: attestationResponseJSON{ClientDataJSON: dto.Response.ClientDataJSON, AttestationObject: dto.Response.AttestationObject, Transports: dto.Response.Transports}, AuthenticatorAttachment: dto.AuthenticatorAttachment}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return goprotocol.ParseCredentialCreationResponseBody(bytes.NewReader(data))
}

// JSON 中間構造体 ─────────────────────────────────────────────────────────────

type assertionResponseJSON struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData,omitempty"`
	Signature         string `json:"signature,omitempty"`
	UserHandle        string `json:"userHandle,omitempty"`
}

type assertionCredentialJSON struct {
	ID                      string                `json:"id"`
	RawID                   string                `json:"rawId"`
	Type                    string                `json:"type"`
	Response                assertionResponseJSON `json:"response"`
	AuthenticatorAttachment string                `json:"authenticatorAttachment,omitempty"`
}

type attestationResponseJSON struct {
	ClientDataJSON    string   `json:"clientDataJSON"`
	AttestationObject string   `json:"attestationObject,omitempty"`
	Transports        []string `json:"transports,omitempty"`
}

type attestationCredentialJSON struct {
	ID                      string                  `json:"id"`
	RawID                   string                  `json:"rawId"`
	Type                    string                  `json:"type"`
	Response                attestationResponseJSON `json:"response"`
	AuthenticatorAttachment string                  `json:"authenticatorAttachment,omitempty"`
}
