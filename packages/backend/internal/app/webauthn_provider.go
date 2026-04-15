package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	goprotocol "github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/usecases"
)

// webAuthnSessionEntry は pending ceremony の SessionData と TTL を保持する。
type webAuthnSessionEntry struct {
	sessionData gowebauthn.SessionData
	expiresAt   time.Time
}

// webAuthnProvider は go-webauthn/webauthn を使った WebAuthnProvider 実装。
// セレモニーのセッションデータは短命（challengeTTL）のため、インスタンスのメモリに保持する。
// TODO: 本番では Valkey に移行すること（複数インスタンス構成に対応するため）。
type webAuthnProvider struct {
	wa           *gowebauthn.WebAuthn
	sessions     sync.Map // key: challengeKey(string) -> *webAuthnSessionEntry
	challengeTTL time.Duration
}

// newWebAuthnProvider は WebAuthnProvider を生成する。
// rpid は WebAuthn RPID（例: "localhost"）, origins は許可 origin のリスト（例: ["http://localhost:5173"]）。
// origins が空の場合は "https://<rpid>" を fallback として補完する。
func newWebAuthnProvider(rpid string, origins []string, challengeTTL time.Duration) (usecases.WebAuthnProvider, error) {
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

func (p *webAuthnProvider) BeginLogin(_ context.Context, _ string) (challengeKey string, challengeBytes []byte, err error) {
	// discoverable login（resident key）を使うため、BeginDiscoverableLogin を使用する。
	assertion, sessionData, err := p.wa.BeginDiscoverableLogin()
	if err != nil {
		return "", nil, fmt.Errorf("webauthn: BeginDiscoverableLogin: %w", err)
	}

	challengeKey = sessionData.Challenge
	p.sessions.Store(challengeKey, &webAuthnSessionEntry{
		sessionData: *sessionData,
		expiresAt:   time.Now().Add(p.challengeTTL),
	})

	jsonBytes, jsonErr := json.Marshal(assertion)
	if jsonErr != nil {
		return "", nil, fmt.Errorf("webauthn: marshal assertion: %w", jsonErr)
	}

	return challengeKey, jsonBytes, nil
}

// ─── FinishLogin ─────────────────────────────────────────────────────────────

// FinishLogin は clientDataJSON から challenge を自己解決して session を特定し、
// lookupCredential コールバックで DB から公開鍵を取得したうえで
// ValidatePasskeyLogin で full signature verification を行い、
// credentialHandle と更新済み SignCount・BackupState を返す。
// signCountUpdated が true のときのみ newSignCount/newBackupState が有効な値。
// challengeKey は空文字列で渡せば clientDataJSON から自己解決する。
func (p *webAuthnProvider) FinishLogin(ctx context.Context, challengeKey string, credential usecases.WebAuthnAssertionCredentialDTO,
	lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error),
) (credentialHandle string, newSignCount uint32, newBackupState bool, signCountUpdated bool, err error) {
	lookupKey, sessionEntry, sessionErr := p.resolveLoginSession(challengeKey, credential.Response.ClientDataJSON)
	if sessionErr != nil {
		return "", 0, false, false, sessionErr
	}

	parsed, parseErr := dtoToAssertionParsed(credential)
	if parseErr != nil {
		return "", 0, false, false, fmt.Errorf("webauthn: parse assertion: %w", parseErr)
	}

	var resolvedHandle string
	var handlerErr error
	handler := p.buildDiscoverableHandler(ctx, &resolvedHandle, &handlerErr, lookupCredential)

	_, updatedCred, validateErr := p.wa.ValidatePasskeyLogin(handler, sessionEntry.sessionData, parsed)
	p.sessions.Delete(lookupKey)

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

// resolveLoginSession は challengeKey と clientDataJSON から sessionEntry を返す。
func (p *webAuthnProvider) resolveLoginSession(challengeKey string, clientDataJSON string) (string, *webAuthnSessionEntry, error) {
	derivedKey, err := challengeKeyFromClientDataJSON(clientDataJSON)
	if err != nil {
		return "", nil, fmt.Errorf("webauthn: cannot parse clientDataJSON: %w", err)
	}
	lookupKey := challengeKey
	if lookupKey == "" {
		lookupKey = derivedKey
	} else if lookupKey != derivedKey {
		return "", nil, fmt.Errorf("webauthn: challengeKey mismatch: provided %q, derived %q", challengeKey, derivedKey)
	}
	entry, ok := p.sessions.Load(lookupKey)
	if !ok {
		return "", nil, fmt.Errorf("webauthn: session not found for challengeKey %q", lookupKey)
	}
	sessionEntry := entry.(*webAuthnSessionEntry)
	if time.Now().After(sessionEntry.expiresAt) {
		p.sessions.Delete(lookupKey)
		return "", nil, fmt.Errorf("webauthn: session expired for challengeKey %q", lookupKey)
	}
	return lookupKey, sessionEntry, nil
}

// resolveRegistrationSession は login 側と対称に challengeKey と clientDataJSON から lookupKey を返す。
// challengeKey が空の場合は clientDataJSON から自己解決し、非空の場合は mismatch を検証する。
func (p *webAuthnProvider) resolveRegistrationSession(challengeKey string, clientDataJSON string) (string, error) {
	derivedKey, err := challengeKeyFromClientDataJSON(clientDataJSON)
	if err != nil {
		return "", fmt.Errorf("webauthn: cannot parse clientDataJSON: %w", err)
	}
	lookupKey := challengeKey
	if lookupKey == "" {
		lookupKey = derivedKey
	} else if lookupKey != derivedKey {
		return "", fmt.Errorf("webauthn: challengeKey mismatch: provided %q, derived %q", challengeKey, derivedKey)
	}
	return lookupKey, nil
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

func (p *webAuthnProvider) BeginRegistration(_ context.Context, accountID string) (challengeKey string, challengeBytes []byte, err error) {
	user := &webAuthnUserAdapter{id: accountID, name: accountID}

	creation, sessionData, err := p.wa.BeginRegistration(user)
	if err != nil {
		return "", nil, fmt.Errorf("webauthn: BeginRegistration: %w", err)
	}

	challengeKey = sessionData.Challenge
	p.sessions.Store(challengeKey, &webAuthnSessionEntry{
		sessionData: *sessionData,
		expiresAt:   time.Now().Add(p.challengeTTL),
	})

	jsonBytes, jsonErr := json.Marshal(creation)
	if jsonErr != nil {
		return "", nil, fmt.Errorf("webauthn: marshal creation: %w", jsonErr)
	}

	return challengeKey, jsonBytes, nil
}

// ─── FinishRegistration ───────────────────────────────────────────────────────

func (p *webAuthnProvider) FinishRegistration(_ context.Context, challengeKey string, accountID string, credential usecases.WebAuthnAttestationCredentialDTO) (credentialHandle string, credData domain.WebAuthnCredentialData, err error) {
	lookupKey, resolveErr := p.resolveRegistrationSession(challengeKey, credential.Response.ClientDataJSON)
	if resolveErr != nil {
		return "", domain.ZeroWebAuthnCredentialData(), resolveErr
	}

	entry, ok := p.sessions.Load(lookupKey)
	if !ok {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: session not found for challengeKey %q", lookupKey)
	}
	sessionEntry := entry.(*webAuthnSessionEntry)
	if time.Now().After(sessionEntry.expiresAt) {
		p.sessions.Delete(lookupKey)
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: session expired for challengeKey %q", lookupKey)
	}

	user := &webAuthnUserAdapter{id: accountID, name: accountID}

	parsed, parseErr := dtoToAttestationParsed(credential)
	if parseErr != nil {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: parse attestation: %w", parseErr)
	}

	webauthnCred, createErr := p.wa.CreateCredential(user, sessionEntry.sessionData, parsed)
	if createErr != nil {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: CreateCredential: %w", createErr)
	}

	p.sessions.Delete(lookupKey)

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

// ─── DTO → protocol struct 変換ヘルパー ───────────────────────────────────────

// challengeKeyFromClientDataJSON は base64url-encoded clientDataJSON を decode して
// JSON の "challenge" フィールド（base64url string）を challengeKey として返す。
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
