package domain

// WebAuthnCredentialData は WebAuthn credential record の永続化に必要なデータ。
// persistence 層と usecases 層の両方から参照できるよう domain パッケージで定義する。
type WebAuthnCredentialData struct {
	// PublicKey は COSE エンコードされた public key バイト列。
	PublicKey []byte
	// SignCount は authenticator の sign count（リプレイ攻撃検出に使用）。
	SignCount uint32
	// AAGUID は authenticator の AAGUID（16 バイト）。
	AAGUID []byte
	// BackupEligible は credential がバックアップ対象かどうかを示す。
	BackupEligible bool
	// BackupState は credential が現在バックアップされているかどうかを示す。
	BackupState bool
	// Transports は credential がサポートする transport のリスト（例: ["usb", "nfc"]）。
	Transports []string
}

// NewWebAuthnCredentialData は WebAuthnCredentialData を構築する constructor。
// go-webauthn/webauthn ライブラリが返す credential から呼び出す。
func NewWebAuthnCredentialData(publicKey []byte, signCount uint32, aaguid []byte, backupEligible bool, backupState bool, transports []string) WebAuthnCredentialData {
	return WebAuthnCredentialData{
		PublicKey:      publicKey,
		SignCount:      signCount,
		AAGUID:         aaguid,
		BackupEligible: backupEligible,
		BackupState:    backupState,
		Transports:     transports,
	}
}

// ZeroWebAuthnCredentialData は空の WebAuthnCredentialData を返す（エラー時のプレースホルダー）。
func ZeroWebAuthnCredentialData() WebAuthnCredentialData {
	return WebAuthnCredentialData{}
}

// WebAuthnStoredCredential は DB から復元した WebAuthn credential record。
// FinishLogin ceremony での署名検証（ValidatePasskeyLogin）に必要なデータを含む。
type WebAuthnStoredCredential struct {
	// Handle は base64url-encoded credential ID（lookup key）。
	Handle string
	// PublicKey は COSE エンコードされた public key バイト列。
	PublicKey []byte
	// SignCount は authenticator の sign count（リプレイ攻撃検出）。
	SignCount uint32
	// AAGUID は authenticator の AAGUID（16 バイト）。
	AAGUID []byte
	// BackupEligible は credential がバックアップ対象かどうかを示す。
	BackupEligible bool
	// BackupState は credential が現在バックアップされているかどうかを示す。
	BackupState bool
	// Transports は credential がサポートする transport のリスト。
	Transports []string
}

// ReconstitueWebAuthnStoredCredential は DB 永続化レコードから WebAuthnStoredCredential を復元する。
// persistence 層が DB カラム値から呼び出す reconstitution helper。
func ReconstitueWebAuthnStoredCredential(handle string, publicKey []byte, signCount uint32, aaguid []byte, backupEligible bool, backupState bool, transports []string) WebAuthnStoredCredential {
	return WebAuthnStoredCredential{
		Handle:         handle,
		PublicKey:      publicKey,
		SignCount:      signCount,
		AAGUID:         aaguid,
		BackupEligible: backupEligible,
		BackupState:    backupState,
		Transports:     transports,
	}
}

// ZeroWebAuthnStoredCredential は空の WebAuthnStoredCredential を返す（エラー時のプレースホルダー）。
func ZeroWebAuthnStoredCredential() WebAuthnStoredCredential {
	return WebAuthnStoredCredential{}
}
