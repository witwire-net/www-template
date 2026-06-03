package operators

import (
	"time"

	authapplication "www-template/packages/backend/internal/application/auth"
)

// ─── Input / Output DTO ────────────────────────────────────────────────────

// InitialSetupStartInput は初回 operator 作成 challenge 開始入力である。
//
// 役割:
//   - bootstrap gate 検証と WebAuthn challenge 発行に必要な情報を HTTP adapter から受け取る。
//   - BootstrapSecret は平文で受け取り、use case 内で opaque hash と照合する。
//
// フィールド:
//   - Email: 作成する operator の email アドレス。domain.OperatorEmail で検証される。
//   - DisplayName: 表示名。空の場合は email にフォールバックする。
//   - BootstrapSecret: 初回 setup gate の平文 secret。config の hash と照合される。
//   - RequestID: HTTP request と WebAuthn session を一致させる canonical ULID。
type InitialSetupStartInput struct {
	Email           string
	DisplayName     string
	BootstrapSecret string
	RequestID       string
}

// InitialSetupFinishInput は初回 operator 作成完了入力である。
//
// 役割:
//   - WebAuthn attestation 検証と operator/passkey 保存に必要な情報を HTTP adapter から受け取る。
//   - Credential は browser WebAuthn API の response を application 境界 primitive に変換した値。
//
// フィールド:
//   - Email: StartInput と同じ email。finish 時に再検証する。
//   - DisplayName: StartInput と同じ表示名。
//   - BootstrapSecret: StartInput と同じ平文 secret。finish 時に再検証する。
//   - RequestID: StartInput と同じ ULID。WebAuthn session selector として使う。
//   - Credential: WebAuthn attestation credential。provider で検証される。
type InitialSetupFinishInput struct {
	Email           string
	DisplayName     string
	BootstrapSecret string
	RequestID       string
	Credential      authapplication.OperatorWebAuthnAttestationCredential
}

// SetupStartInput は追加 operator の setup challenge 開始入力である。
//
// 役割:
//   - setup token 検証と WebAuthn challenge 発行に必要な情報を HTTP adapter から受け取る。
//   - SetupToken は平文で受け取り、use case 内で opaque hash と照合する。
//
// フィールド:
//   - SetupToken: operator 作成時に配送された平文 setup token。
//   - RequestID: HTTP request と WebAuthn session を一致させる canonical ULID。
type SetupStartInput struct {
	SetupToken string
	RequestID  string
}

// SetupFinishInput は追加 operator の setup 完了入力である。
//
// 役割:
//   - setup token 再検証と WebAuthn attestation 検証に必要な情報を HTTP adapter から受け取る。
//   - SetupToken は finish 時に再検証し、期限切れや既消費 token を拒否する。
//
// フィールド:
//   - SetupToken: StartInput と同じ平文 setup token。finish 時に再検証する。
//   - RequestID: StartInput と同じ ULID。WebAuthn session selector として使う。
//   - Credential: WebAuthn attestation credential。provider で検証される。
type SetupFinishInput struct {
	SetupToken string
	RequestID  string
	Credential authapplication.OperatorWebAuthnAttestationCredential
}

// CreateOperatorInput は acting operator が追加 operator を作成する入力である。
//
// 役割:
//   - acting operator の権限検証と作成対象 operator の identity 検証に必要な情報を HTTP adapter から受け取る。
//   - OperatorID/OperatorEmail/OperatorRole/OperatorActive/PasskeyRegistrationState は acting operator の snapshot。
//
// フィールド:
//   - Email: 作成対象 operator の email アドレス。domain.OperatorEmail で検証される。
//   - Role: 作成対象 operator の role。domain.OperatorRole で検証される。
//   - RequestID: HTTP request と audit を一致させる canonical ULID。
//   - OperatorID: acting operator の ID。domain.OperatorID で検証される。
//   - OperatorEmail: acting operator の email。domain.OperatorEmail で検証される。
//   - OperatorRole: acting operator の role。
//   - OperatorActive: acting operator の active 状態。
//   - PasskeyRegistrationState: acting operator の passkey 登録状態。
type CreateOperatorInput struct {
	Email                    string
	Role                     string
	RequestID                string
	OperatorID               string
	OperatorEmail            string
	OperatorRole             string
	OperatorActive           bool
	PasskeyRegistrationState string
}

// SetupChallengeResult は passkey 登録 challenge response 用 DTO である。
//
// 役割:
//   - WebAuthn registration ceremony に必要な challenge 情報を HTTP adapter へ渡す。
//   - OptionsJSON は browser の navigator.credentials.create() に渡す PublicKeyCredentialCreationOptions。
//
// フィールド:
//   - RequestID: HTTP response と WebAuthn session を一致させる canonical ULID。
//   - Challenge: browser response の challenge と照合される base64url 値。
//   - OptionsJSON: WebAuthn provider が生成した PublicKeyCredentialCreationOptions JSON。
type SetupChallengeResult struct {
	RequestID   string
	Challenge   string
	OptionsJSON []byte
}

// CreatedOperator は operator 作成 response 用 DTO である。
//
// 役割:
//   - operator 作成結果を HTTP adapter へ渡す。
//   - setup token 平文は含めず、delivery status だけを返す。
//
// フィールド:
//   - RequestID: HTTP request と audit を一致させる canonical ULID。
//   - AuditID: mutation intent 記録の audit ID。
//   - DeliveryStatus: setup token の配送状態。成功時は "sent"。
//   - Operator: 作成された operator の非秘匿 DTO。
type CreatedOperator struct {
	RequestID      string
	AuditID        string
	DeliveryStatus string
	Operator       authapplication.OperatorDTO
}

// SetupTokenDelivery は secure delivery port に渡す token 配送 DTO である。
//
// 役割:
//   - setup token 平文を SMTP などの secure delivery port へ渡す。
//   - token 平文はこの DTO 内だけに保持し、log や response へ出さない。
//
// フィールド:
//   - OperatorID: setup token の送信先 operator ID。
//   - Email: setup token の送信先 email アドレス。
//   - SetupToken: 平文 setup token。delivery port だけが扱う。
//   - ExpiresAt: setup token の有効期限。
//   - RequestID: HTTP request と audit を一致させる canonical ULID。
type SetupTokenDelivery struct {
	OperatorID string
	Email      string
	SetupToken string
	ExpiresAt  time.Time
	RequestID  string
}
