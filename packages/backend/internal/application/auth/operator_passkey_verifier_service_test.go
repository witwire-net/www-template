package auth

import (
	"context"
	"testing"

	domain "www-template/packages/backend/internal/domain"
)

func TestOperatorPasskeyVerifierConsumesChallengeAndUpdatesCredentialState(t *testing.T) {
	t.Parallel()

	// Step 1: production verifier と同じ provider/store seam を fake で構成し、challenge selector が provider へ渡る経路を観測する。
	ctx := context.Background()
	provider := &stubAdminWebAuthnProvider{credentialHandle: "verified-credential-handle", newSignCount: 42, newBackupState: true, signCountUpdated: true}
	store := &stubOperatorWebAuthnCredentialStore{}
	verifier, err := NewOperatorPasskeyVerifier(provider, store)
	if err != nil {
		t.Fatalf("create operator passkey verifier: %v", err)
	}

	// Step 2: application assertion DTO を渡し、raw ID ではなく provider 検証済み handle だけが返ることを確認する。
	handle, err := verifier.VerifyOperatorPasskey(ctx, "challenge-selector", WebAuthnAssertionCredentialDTO{ID: "credential-id", RawID: "raw-credential-id", Type: "public-key", Response: WebAuthnAssertionResponseDTO{ClientDataJSON: "client-data", AuthenticatorData: "auth-data", Signature: "signature", UserHandle: "operator-user"}})
	if err != nil {
		t.Fatalf("verify operator passkey: %v", err)
	}
	if handle != "verified-credential-handle" {
		t.Fatalf("expected verified credential handle, got %q", handle)
	}

	// Step 3: challenge selector、lookup callback、credential state update が production verifier 経由で使われたことを固定する。
	if provider.challengeID != "challenge-selector" || provider.rawID != "raw-credential-id" || !provider.lookupCalled {
		t.Fatalf("expected provider to receive challenge and lookup callback, got %+v", provider)
	}
	if store.updatedHandle != "verified-credential-handle" || store.updatedSignCount != 42 || !store.updatedBackupState {
		t.Fatalf("expected credential state update, got %+v", store)
	}
}

type stubAdminWebAuthnProvider struct {
	challengeID      string
	rawID            string
	lookupCalled     bool
	credentialHandle string
	newSignCount     uint32
	newBackupState   bool
	signCountUpdated bool
}

func (p *stubAdminWebAuthnProvider) BeginLogin(context.Context, string) (string, []byte, error) {
	// Step 1: verifier test では finish 経路だけを使うため、未使用 method は空 challenge を返す。
	return "", nil, nil
}

func (p *stubAdminWebAuthnProvider) FinishLogin(ctx context.Context, challengeKey string, credential WebAuthnAssertionCredentialDTO, lookupCredential func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
	// Step 1: verifier が challenge selector と assertion DTO 由来 raw ID を provider へ渡した事実を記録する。
	p.challengeID = challengeKey
	p.rawID = credential.RawID

	// Step 2: lookup callback を呼び、provider が repository 経由で public key を取得する production semantics を固定する。
	if lookupCredential != nil {
		p.lookupCalled = true
		if _, err := lookupCredential(ctx, p.credentialHandle); err != nil {
			return "", 0, false, false, err
		}
	}
	return p.credentialHandle, p.newSignCount, p.newBackupState, p.signCountUpdated, nil
}

func (p *stubAdminWebAuthnProvider) BeginRegistration(context.Context, domain.AccountID) (string, []byte, error) {
	// Step 1: verifier test では registration を使わないため、空 challenge を返す。
	return "", nil, nil
}

func (p *stubAdminWebAuthnProvider) FinishRegistration(context.Context, string, domain.AccountID, WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
	// Step 1: verifier test では registration を使わないため、空 credential data を返す。
	return "", domain.ZeroWebAuthnCredentialData(), nil
}

type stubOperatorWebAuthnCredentialStore struct {
	updatedHandle      string
	updatedSignCount   uint32
	updatedBackupState bool
}

func (s *stubOperatorWebAuthnCredentialStore) FindWebAuthnCredential(context.Context, string) (domain.WebAuthnStoredCredential, error) {
	// Step 1: provider へ public key 相当の非空 credential を返し、lookup callback が使われたことを verifier test で観測可能にする。
	return domain.ReconstituteWebAuthnStoredCredential("verified-credential-handle", []byte("public-key"), 1, nil, false, false, nil), nil
}

func (s *stubOperatorWebAuthnCredentialStore) UpdateWebAuthnCredentialState(_ context.Context, handle string, newSignCount uint32, newBackupState bool) error {
	// Step 1: sign count / backup state 更新入力を記録し、verifier が replay 検出状態を進めることを検証する。
	s.updatedHandle = handle
	s.updatedSignCount = newSignCount
	s.updatedBackupState = newBackupState
	return nil
}
