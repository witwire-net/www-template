package auth

import (
	"context"

	domain "www-template/packages/backend/internal/domain"
)

// OperatorWebAuthnCredentialStore は Admin WebAuthn assertion 検証に必要な credential state を扱う port である。
//
// 役割:
//   - credential handle から public key / sign count / backup state を復元する。
//   - 検証成功後の sign count / backup state 更新を同じ Admin credential repository へ閉じる。
//   - Product account passkey repository を Admin finish route から参照しない境界を作る。
type OperatorWebAuthnCredentialStore interface {
	// FindWebAuthnCredential は credential handle から WebAuthn 署名検証用の保存済み credential state を復元する。
	FindWebAuthnCredential(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)
	// UpdateWebAuthnCredentialState は検証成功後の sign count と backup state を保存し、replay 検出状態を前進させる。
	UpdateWebAuthnCredentialState(ctx context.Context, handle string, newSignCount uint32, newBackupState bool) error
}

// OperatorPasskeyVerifier は Admin passkey finish が WebAuthn assertion を検証済み credential handle へ変換する port である。
//
// 役割:
//   - HTTP adapter が raw credential handle を信用して session 発行へ進むことを防ぐ。
//   - WebAuthn provider の challenge 消費、署名検証、user verification、credential lookup を application 境界でまとめる。
type OperatorPasskeyVerifier interface {
	// VerifyOperatorPasskey は challenge selector と assertion DTO を検証し、検証済み credential handle だけを返す。
	VerifyOperatorPasskey(ctx context.Context, challengeID string, credential WebAuthnAssertionCredentialDTO) (string, error)
}

// NewOperatorPasskeyVerifier は Admin passkey finish 用の WebAuthn verifier を生成する。
//
// 引数:
//   - provider: challenge session 消費と assertion 署名検証を行う WebAuthn provider。
//   - credentials: Admin operator passkey credential state repository。
//
// 戻り値:
//   - OperatorPasskeyVerifier: HTTP handler が検証済み credential handle を得るための verifier。
//   - error: provider または credentials が nil の場合の internal error。
func NewOperatorPasskeyVerifier(provider WebAuthnProvider, credentials OperatorWebAuthnCredentialStore) (OperatorPasskeyVerifier, error) {
	// Step 1: assertion 検証に必要な provider/repository が欠けた状態を fail-close に拒否する。
	if provider == nil || credentials == nil {
		return nil, ErrOperatorAuthUnavailable
	}

	// Step 2: 検証済み依存だけを保持し、handler はこの verifier だけを呼ぶ構造にする。
	return operatorPasskeyVerifier{provider: provider, credentials: credentials}, nil
}

type operatorPasskeyVerifier struct {
	provider    WebAuthnProvider
	credentials OperatorWebAuthnCredentialStore
}

func (v operatorPasskeyVerifier) VerifyOperatorPasskey(ctx context.Context, challengeID string, credential WebAuthnAssertionCredentialDTO) (string, error) {
	// Step 1: provider に challenge session 消費、署名、user verification、credential lookup を委譲する。
	credentialHandle, newSignCount, newBackupState, signCountUpdated, err := v.provider.FinishLogin(ctx, challengeID, credential, v.credentials.FindWebAuthnCredential)
	if err != nil {
		return "", ErrOperatorAuthForbidden
	}

	// Step 2: provider が sign count 更新値を返した場合だけ Admin credential state を更新し、replay 検出用状態を進める。
	if signCountUpdated {
		if err := v.credentials.UpdateWebAuthnCredentialState(ctx, credentialHandle, newSignCount, newBackupState); err != nil {
			return "", ErrOperatorAuthUnavailable
		}
	}

	// Step 3: 空 handle は検証済み credential として扱えないため forbidden に畳む。
	if credentialHandle == "" {
		return "", ErrOperatorAuthForbidden
	}

	// Step 4: 検証済み credential handle だけを passkey login service へ渡す。
	return credentialHandle, nil
}

var _ OperatorPasskeyVerifier = operatorPasskeyVerifier{}
