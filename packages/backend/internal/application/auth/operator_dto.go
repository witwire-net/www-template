package auth

import (
	"context"
	"time"
)

// OperatorSnapshot は Admin auth application が repository port から受け取る Operator 復元 DTO である。
//
// 役割:
//   - domain.Operator を application public API に直接露出させず、復元に必要な primitive だけを保持する。
//   - Product AccountAuth の account ID、status、session 情報を含めないことで Admin 専用境界を維持する。
//   - repository adapter はこの DTO を返し、use case が domain.NewOperator で不変条件を検証する。
//
// 使用例:
//
//	snapshot, err := repo.FindOperatorByCredential(ctx, credentialHandle)
//	if err != nil {
//		return err
//	}
type OperatorSnapshot struct {
	ID                       string
	Email                    string
	Role                     string
	Active                   bool
	PasskeyRegistrationState string
}

// OperatorSessionRecord は Admin refresh session store が保存・復元する application DTO である。
//
// 役割:
//   - refreshToken 平文ではなく hash と snapshot だけを store 境界へ渡す。
//   - rotation 時に同一 Operator session の置換対象を session ID と refresh hash で特定できるようにする。
//   - domain.OperatorAuthSession の復元に必要な値だけを持ち、Product account session state を混入させない。
type OperatorSessionRecord struct {
	SessionID        string
	OperatorID       string
	RefreshTokenHash string
	RoleSnapshot     string
	ActiveSnapshot   bool
	IssuedAt         time.Time
	ExpiresAt        time.Time
	Revoked          bool
}

// OperatorRefreshSessionStore は Admin Operator refresh session state を保存・取得・rotation・revoke する port である。
//
// 役割:
//   - Valkey などの保存実装を application 境界から隠蔽する。
//   - Rotate は currentRefreshTokenHash の一致確認と replacement 保存を同一原子的処理として実装できる形にする。
//   - refreshToken 平文は受け取らず、hash と Cookie selector だけで state を扱う。
type OperatorRefreshSessionStore interface {
	Save(ctx context.Context, record OperatorSessionRecord, ttl time.Duration) error
	Get(ctx context.Context, sessionID string) (OperatorSessionRecord, error)
	Rotate(ctx context.Context, sessionID string, currentRefreshTokenHash string, replacement OperatorSessionRecord, ttl time.Duration) error
	Revoke(ctx context.Context, operatorID string, sessionID string) error
}

// OperatorRepository は Admin OperatorAuth use case が Operator snapshot を取得する port である。
//
// 役割:
//   - passkey credential handle または OperatorID から Admin Operator の現在状態を取得する。
//   - Product account repository を import せず、Admin operator auth に必要な snapshot だけを返す。
//   - adapter は永続化エラーをこの port の error として返し、use case が fail-closed に写像する。
type OperatorRepository interface {
	FindOperatorByCredential(ctx context.Context, credentialHandle string) (OperatorSnapshot, error)
	FindOperatorByID(ctx context.Context, operatorID string) (OperatorSnapshot, error)
}

// OperatorPasskeyCredential は Admin operator passkey 一覧に返す application DTO である。
//
// 役割:
//   - credential_handle、public_key、sign_count など認証検証用の秘匿値を含めない。
//   - Product account passkey DTO を再利用せず、Admin operator credential の表示・削除識別子だけを保持する。
//   - LastUsedAt は未使用 credential では nil になり、transport layer が optional field として扱う。
type OperatorPasskeyCredential struct {
	ID         string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// OperatorPasskeyRepository は Admin operator passkey credential の一覧取得と削除を扱う port である。
//
// 役割:
//   - admin.operator_passkeys などの保存実装を application 境界から隠蔽する。
//   - operatorID で必ず所有者を絞り、他 Operator の passkey を読み書きしない境界を作る。
//   - DeleteOperatorPasskey は repository 側でも最後の 1 件削除を防げるよう実装し、application/domain 検証との二重防御にする。
type OperatorPasskeyRepository interface {
	ListOperatorPasskeys(ctx context.Context, operatorID string) ([]OperatorPasskeyCredential, error)
	DeleteOperatorPasskey(ctx context.Context, operatorID string, passkeyID string) error
}

// OperatorPasskeyChallengeProvider は Admin passkey login の challenge 発行を抽象化する port である。
//
// 役割:
//   - WebAuthn provider や challenge store の具体実装を application から分離する。
//   - StartOperatorPasskey が Admin 専用 RP 設定と challenge JSON を返すための入力を提供する。
//   - Product passkey provider と共有せず、Admin operator credential だけを対象にする。
type OperatorPasskeyChallengeProvider interface {
	BeginOperatorLogin(ctx context.Context, identifier string) (challengeKey string, optionsJSON []byte, err error)
}

// OperatorWebAuthnAttestationResponse は Admin operator passkey 登録 ceremony の attestation response DTO である。
//
// 役割:
//   - HTTP generated DTO と WebAuthn adapter の間で必要な primitive だけを運ぶ。
//   - clientDataJSON と attestationObject は base64url 文字列のまま渡し、署名検証は provider が行う。
//   - transports は保存用 credential metadata に変換されるが、handler では解釈しない。
type OperatorWebAuthnAttestationResponse struct {
	ClientDataJSON    string
	AttestationObject string
	Transports        []string
}

// OperatorWebAuthnAttestationCredential は Admin operator passkey 登録 ceremony の credential DTO である。
//
// 役割:
//   - browser WebAuthn API の PublicKeyCredential を application 境界の primitive に変換した値である。
//   - Product account registration DTO を import せず、Admin operator setup 専用の型として扱う。
//   - AuthenticatorAttachment は任意値であり、provider が検証に必要な場合だけ参照する。
type OperatorWebAuthnAttestationCredential struct {
	ID                      string
	RawID                   string
	Type                    string
	Response                OperatorWebAuthnAttestationResponse
	AuthenticatorAttachment string
}

// OperatorRegistrationChallengeInput は Admin operator passkey 登録 challenge 開始入力である。
//
// OperatorID は初回 setup では作成予定 ID、追加 operator setup では既存 OperatorID を渡す。
// RequestID は HTTP response と WebAuthn session lookup を一致させるための canonical ULID である。
type OperatorRegistrationChallengeInput struct {
	RequestID          string
	OperatorID         string
	Email              string
	DisplayName        string
	ExcludeCredentials []string
}

// OperatorRegistrationChallenge は Admin operator passkey 登録 challenge の provider 結果である。
//
// OptionsJSON は go-webauthn 等の provider が生成した PublicKeyCredentialCreationOptions JSON である。
// Challenge は browser response の challenge と照合される base64url 値であり、secret ではない。
type OperatorRegistrationChallenge struct {
	RequestID   string
	Challenge   string
	OptionsJSON []byte
}

// OperatorPasskeyRegistration は検証済み Admin operator passkey credential の保存 DTO である。
//
// 役割:
//   - WebAuthn provider が challenge、origin、RP ID、user verification を検証した後の値だけを保持する。
//   - repository はこの DTO から credential_handle、public_key、sign_count などを保存し、HTTP response へは出さない。
type OperatorPasskeyRegistration struct {
	CredentialHandle string
	PublicKey        []byte
	SignCount        uint32
	AAGUID           []byte
	BackupEligible   bool
	BackupState      bool
	Transports       []string
}

// OperatorPasskeyRegistrationProvider は Admin operator passkey 登録 WebAuthn ceremony を実行する port である。
//
// 役割:
//   - application service が go-webauthn 等の adapter に直接依存しないようにする。
//   - BeginOperatorRegistration は discoverable credential + userVerification=required の registration options を発行する。
//   - FinishOperatorRegistration は attestation を検証し、保存可能な credential data だけを返す。
type OperatorPasskeyRegistrationProvider interface {
	BeginOperatorRegistration(ctx context.Context, input OperatorRegistrationChallengeInput) (OperatorRegistrationChallenge, error)
	FinishOperatorRegistration(ctx context.Context, requestID string, operatorID string, credential OperatorWebAuthnAttestationCredential) (OperatorPasskeyRegistration, error)
}

// OperatorSessionConfig は Admin operator auth use case の token/session lifetime と RP 表示情報を保持する。
//
// 役割:
//   - accessToken TTL、refresh session TTL、Cookie lifetime を application 起動時に検証できる単位にまとめる。
//   - refresh Cookie lifetime が server-side TTL を超えないよう shared token primitive で検証する。
//   - WebAuthn RP ID は challenge response DTO に渡すだけで、Product RP 設定とは混在させない。
type OperatorSessionConfig struct {
	OperatorAccessTokenTTL        time.Duration
	OperatorRefreshSessionTTL     time.Duration
	OperatorRefreshCookieLifetime time.Duration
	WebAuthnRPID                  string
}

// StartOperatorPasskeyInput は Admin operator passkey login challenge 開始入力である。
//
// Identifier は operator email など adapter が受け取った識別子であり、正規化や存在有無の詳細は provider 側へ委譲する。
type StartOperatorPasskeyInput struct {
	Identifier string
}

// OperatorPasskeyChallenge は Admin passkey login challenge 開始結果である。
//
// refreshToken や session secret は含まず、browser WebAuthn ceremony に必要な challenge 情報だけを返す。
type OperatorPasskeyChallenge struct {
	ChallengeID     string
	Challenge       string
	WebAuthnRPID    string
	WebAuthnOptions []byte
}

// FinishOperatorPasskeyInput は WebAuthn 検証済み credential から Admin session を発行する入力である。
//
// CredentialHandle は adapter/provider が署名検証後に確定した Admin operator credential handle である。
// ChallengeID は将来の challenge 消費監査に使えるが、この use case は credential handle から Operator を復元する。
type FinishOperatorPasskeyInput struct {
	ChallengeID      string
	CredentialHandle string
}

// RefreshOperatorSessionInput は Admin refresh credential と path auth context から accessToken と refresh credential rotation を行う入力である。
//
// AuthContextID は refresh route path から受け取った canonical context selector であり、refresh credential 内 session selector と一致する場合だけ rotation を許可する。
// RefreshTokenValue は Cookie mode では HttpOnly Cookie、Bearer mode では request body から adapter が読み取った opaque 値であり、検証後は新しい credential へ rotation される。
type RefreshOperatorSessionInput struct {
	AuthContextID     string
	RefreshTokenValue string
}

// CurrentOperatorInput は bearer accessToken から現在の Admin Operator を取得する入力である。
//
// AccessToken は Authorization header から抽出された compact token で、Product bearer token と共有しない。
type CurrentOperatorInput struct {
	AccessToken string
}

// LogoutOperatorInput は Admin operator session revoke の入力である。
//
// AccessToken は対象 session selector と OperatorID を特定するために使い、refreshToken 平文は不要である。
type LogoutOperatorInput struct {
	AccessToken string
}

// AuthorizeOperatorSessionInput は Admin mutation route の bearer/session/snapshot/RBAC 検証入力である。
//
// 役割:
//   - HTTP adapter が持つ Authorization header 由来の accessToken と route permission だけを application auth 境界へ渡す。
//   - Cookie、CSRF、Product bearer token、role matrix を含めず、OperatorAuth domain の ValidateAccess へ判定を委譲する。
type AuthorizeOperatorSessionInput struct {
	AccessToken string
	Permission  string
}

// OperatorAuthorizationDecision は Admin mutation route が許可された結果を表す DTO である。
//
// 役割:
//   - bearer/session/snapshot/permission がすべて検証済みであることを handler context へ伝える。
//   - domain.Operator 自体を adapter へ公開せず、監査と downstream context に必要な primitive だけを含める。
type OperatorAuthorizationDecision struct {
	Operator   OperatorDTO
	SessionID  string
	Permission string
	Allowed    bool
}

// IssueOperatorSessionInput は認証済み Operator に session を発行する入力である。
//
// OperatorID は passkey login、initial setup、operator setup transaction が認証済みにした Operator の canonical ID である。
// refreshToken 平文は戻り値の Cookie command にだけ含まれ、response body には入らない。
type IssueOperatorSessionInput struct {
	OperatorID string
}

// ListOperatorPasskeysInput は Admin operator 自身の passkey 一覧取得入力である。
//
// OperatorID は middleware が Admin accessToken/session を検証した後に context へ束縛した値だけを渡す。
type ListOperatorPasskeysInput struct {
	OperatorID string
}

// OperatorPasskeyListResult は Admin operator passkey 一覧取得結果である。
//
// Passkeys は response body に出せる非秘匿 DTO だけで構成し、credential handle や public key は含めない。
type OperatorPasskeyListResult struct {
	Passkeys []OperatorPasskeyCredential
}

// DeleteOperatorPasskeyInput は Admin operator 自身の passkey 削除入力である。
//
// OperatorID は検証済み session context 由来の所有者 ID、PasskeyID は path parameter 由来の削除対象 credential ID である。
type DeleteOperatorPasskeyInput struct {
	OperatorID string
	PasskeyID  string
}

// OperatorRefreshCookieCommand は adapter が refresh credential を設定または削除するための command DTO である。
//
// Value は Cookie mode では Set-Cookie header 専用、Bearer mode では response body への写像元として扱う。
// AuthContextID は HTTP adapter が refresh route path と Cookie Path を決めるための selector であり、Cookie 属性そのものは application に置かない。
type OperatorRefreshCookieCommand struct {
	AuthContextID string
	Value         string
	MaxAge        time.Duration
	Clear         bool
}

// OperatorSessionResult は login/refresh 成功時に Admin frontend へ返す browser-readable session DTO である。
//
// AccessToken は response body に含めるが、refreshToken 平文は Cookie mode の Set-Cookie または Bearer mode body へ写像する直前まで RefreshCookie command の Value にだけ保持される。
type OperatorSessionResult struct {
	AccessToken   string
	Operator      OperatorDTO
	SessionID     string
	ExpiresAt     time.Time
	RefreshCookie OperatorRefreshCookieCommand `json:"-"`
}

// OperatorDTO は Admin frontend と adapter に返す現在 Operator の application DTO である。
//
// Product Account DTO を再利用せず、operator ID/email/role/active/passkey state と、検証済み session selector だけを保持する。
type OperatorDTO struct {
	ID                       string
	Email                    string
	Role                     string
	Active                   bool
	SessionID                string
	PasskeyRegistrationState string
}
