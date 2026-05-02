package app

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

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/usecases"
)

// challengeStore は WebAuthn challenge session を外部ストレージ（Valkey）に保存するための最小インターフェース。
type challengeStore interface {
	// Key は環境プレフィックス付きのキーを構築する。
	Key(parts ...string) string
	// Set は key に value を TTL 付きで保存する。
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	// GetDel は key の値を取得すると同時にアトミックに削除する。
	// キーが存在しない場合はエラーを返す。
	GetDel(ctx context.Context, key string) (string, error)
}

// webAuthnProvider は go-webauthn/webauthn を使った WebAuthnProvider 実装。
// セレモニーのセッションデータは challengeStore に TTL 付きで保存され、
// Finish 時に GetDel でアトミックに取得・削除される。
type webAuthnProvider struct {
	wa           *gowebauthn.WebAuthn
	store        challengeStore
	challengeTTL time.Duration
}

// newWebAuthnProvider は WebAuthnProvider を生成する。
// rpid は WebAuthn RPID（例: "localhost"）, origins は許可 origin のリスト（例: ["http://localhost:5173"]）。
// origins が空の場合は "https://<rpid>" を fallback として補完する。
func newWebAuthnProvider(rpid string, origins []string, challengeTTL time.Duration, store challengeStore) (usecases.WebAuthnProvider, error) {
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
	credentials []gowebauthn.Credential
}

func (u *webAuthnUserAdapter) WebAuthnID() []byte                           { return []byte(u.id) }
func (u *webAuthnUserAdapter) WebAuthnName() string                         { return u.name }
func (u *webAuthnUserAdapter) WebAuthnDisplayName() string                  { return u.name }
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

// ─── FinishLogin ─────────────────────────────────────────────────────────────

// FinishLogin は challengeKey で Valkey から session を取得・削除し、
// lookupCredential コールバックで DB から公開鍵を取得したうえで
// ValidatePasskeyLogin で full signature verification を行う。
// UV（user verification）が確認できない assertion は無条件に拒否する。
// challengeKey が空文字列の場合は clientDataJSON から自己解決する（後方互換性のため）。
func (p *webAuthnProvider) FinishLogin(ctx context.Context, challengeKey string, credential usecases.WebAuthnAssertionCredentialDTO,
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
	return errors.Is(err, domain.ErrAuthAccountNotFound)
}

// ─── BeginRegistration ────────────────────────────────────────────────────────

func (p *webAuthnProvider) BeginRegistration(ctx context.Context, accountID string) (challengeKey string, challengeBytes []byte, err error) {
	user := &webAuthnUserAdapter{id: accountID, name: accountID}

	// user verification を required に設定し、UV-less attestation を拒否できるようにする。
	creation, sessionData, err := p.wa.BeginRegistration(user, gowebauthn.WithAuthenticatorSelection(goprotocol.AuthenticatorSelection{
		UserVerification: goprotocol.VerificationRequired,
	}))
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

// ─── FinishRegistration ───────────────────────────────────────────────────────

// FinishRegistration は challengeKey で Valkey から session を取得・削除し、
// credential を検証して credential handle と WebAuthn credential data を返す。
// UV（user verification）が確認できない attestation は無条件に拒否する。
// challengeKey が空文字列の場合は clientDataJSON から自己解決する（後方互換性のため）。
func (p *webAuthnProvider) FinishRegistration(ctx context.Context, challengeKey string, accountID string, credential usecases.WebAuthnAttestationCredentialDTO) (credentialHandle string, credData domain.WebAuthnCredentialData, err error) {
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

	user := &webAuthnUserAdapter{id: accountID, name: accountID}

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

// saveSessionData は go-webauthn SessionData を JSON 化して challengeStore に TTL 付きで保存する。
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

// consumeSessionData は challengeStore から SessionData を取得し、アトミックに削除する。
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
func dtoToAssertionParsed(dto usecases.WebAuthnAssertionCredentialDTO) (*goprotocol.ParsedCredentialAssertionData, error) {
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
func dtoToAttestationParsed(dto usecases.WebAuthnAttestationCredentialDTO) (*goprotocol.ParsedCredentialCreationData, error) {
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
