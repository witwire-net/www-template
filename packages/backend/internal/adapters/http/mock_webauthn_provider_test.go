package http

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/auth/domain"
)

// mockWebAuthnProvider は HTTP テスト用の WebAuthn provider stub。
// テスト credential JSON 形式（ID=handle, ClientDataJSON=challengeKey）を解釈する。
//
// BeginLogin/BeginRegistration: challengeKey を生成してインメモリに保存する。
// FinishLogin: credential.ID を credentialHandle として返す。
// FinishRegistration: credential.ID を credentialHandle として返す。
// incompleteRegistrationOptions が true の場合、必須フィールドを欠いた JSON を返す（fail-closed テスト用）。
// registrationOptionsOverride が non-nil の場合、BeginRegistration はその JSON を返す（フィールド単位の fail-closed テスト用）。
type mockWebAuthnProvider struct {
	mu                            sync.Mutex
	session                       map[string]string // challengeKey → identifier or accountID
	seq                           atomic.Uint64
	incompleteRegistrationOptions bool
	registrationOptionsOverride   *string // non-nil: BeginRegistration はこの JSON を返す
}

func newMockWebAuthnProvider() *mockWebAuthnProvider {
	return &mockWebAuthnProvider{session: map[string]string{}}
}

func newMockWebAuthnProviderWithIncompleteOptions() *mockWebAuthnProvider {
	return &mockWebAuthnProvider{session: map[string]string{}, incompleteRegistrationOptions: true}
}

// newMockWebAuthnProviderWithOptions returns a provider that returns the given JSON from BeginRegistration.
// The challenge key placeholder "__KEY__" in optionsJSON is replaced with the generated key.
func newMockWebAuthnProviderWithOptions(optionsJSON string) *mockWebAuthnProvider {
	return &mockWebAuthnProvider{session: map[string]string{}, registrationOptionsOverride: &optionsJSON}
}

func (m *mockWebAuthnProvider) BeginLogin(_ context.Context, identifier string) (string, []byte, error) {
	key := fmt.Sprintf("mock-login-challenge-%d", m.seq.Add(1))
	m.mu.Lock()
	m.session[key] = identifier
	m.mu.Unlock()
	return key, []byte(`{"challenge":"` + key + `"}`), nil
}

func (m *mockWebAuthnProvider) FinishLogin(_ context.Context, _ string, credential application.WebAuthnAssertionCredentialDTO,
	_ func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error),
) (string, uint32, bool, bool, error) {
	handle := credential.ID
	if handle == "" {
		return "", 0, false, false, fmt.Errorf("mock: empty credential ID")
	}
	return handle, 0, false, false, nil
}

func (m *mockWebAuthnProvider) BeginRegistration(_ context.Context, accountID string) (string, []byte, error) {
	key := fmt.Sprintf("mock-reg-challenge-%d", m.seq.Add(1))
	m.mu.Lock()
	m.session[key] = accountID
	m.mu.Unlock()
	if m.registrationOptionsOverride != nil {
		// プレースホルダ "__KEY__" を実際の challengeKey で置換
		replaced := strings.ReplaceAll(*m.registrationOptionsOverride, "__KEY__", key)
		return key, []byte(replaced), nil
	}
	if m.incompleteRegistrationOptions {
		// 必須フィールド（user/pubKeyCredParams/rp.name）を欠いた challenge-only JSON（fail-closed テスト用）
		return key, []byte(`{"publicKey":{"challenge":"` + key + `"}}`), nil
	}
	optionsJSON := `{"publicKey":{"rp":{"id":"localhost","name":"Test RP"},"user":{"id":"dXNlcmlk","name":"testuser","displayName":"Test User"},"challenge":"` + key + `","pubKeyCredParams":[{"type":"public-key","alg":-7},{"type":"public-key","alg":-257}]}}`
	return key, []byte(optionsJSON), nil
}

func (m *mockWebAuthnProvider) FinishRegistration(_ context.Context, _ string, _ string, credential application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
	handle := credential.ID
	if handle == "" {
		return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("mock: empty credential ID")
	}
	return handle, domain.ZeroWebAuthnCredentialData(), nil
}
