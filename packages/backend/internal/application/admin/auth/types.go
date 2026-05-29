package application

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
//   - refreshToken/CSRF token の平文ではなく hash と snapshot だけを store 境界へ渡す。
//   - rotation 時に同一 Operator session の置換対象を session ID と refresh hash で特定できるようにする。
//   - domain.OperatorAuthSession の復元に必要な値だけを持ち、Product account session state を混入させない。
type OperatorSessionRecord struct {
	SessionID        string
	OperatorID       string
	RefreshTokenHash string
	CSRFTokenHash    string
	RoleSnapshot     string
	ActiveSnapshot   bool
	IssuedAt         time.Time
	ExpiresAt        time.Time
	Revoked          bool
}

// OperatorSessionStore は Admin Operator refresh session state を保存・取得・rotation・revoke する port である。
//
// 役割:
//   - Valkey などの保存実装を application 境界から隠蔽する。
//   - RotateOperatorSession は currentRefreshTokenHash の一致確認と replacement 保存を同一原子的処理として実装できる形にする。
//   - refreshToken 平文は受け取らず、hash と Cookie selector だけで state を扱う。
type OperatorSessionStore interface {
	SaveOperatorSession(ctx context.Context, record OperatorSessionRecord, ttl time.Duration) error
	GetOperatorSession(ctx context.Context, sessionID string) (OperatorSessionRecord, error)
	RotateOperatorSession(ctx context.Context, sessionID string, currentRefreshTokenHash string, replacement OperatorSessionRecord, ttl time.Duration) error
	RevokeOperatorSession(ctx context.Context, operatorID string, sessionID string) error
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

// OpaqueTokenGenerator は refreshToken と CSRF token に使う opaque secret を発行する port である。
//
// 役割:
//   - crypto/rand などの副作用源を application use case の外から注入する。
//   - 生成された token 平文は Cookie command または CSRF response にだけ流し、session store には hash だけを保存する。
//   - テストでは deterministic generator を差し替えられるようにする。
type OpaqueTokenGenerator interface {
	NewOpaqueToken() (string, error)
}

// IDGenerator は Admin auth application が ULID 系識別子を生成するための最小 port である。
//
// 役割:
//   - platform/id の concrete policy を use case の必須依存にせず、Next capability だけへ依存する。
//   - session ID、accessToken JTI、request ID の発行元を adapter composition で差し替え可能にする。
//   - 生成値の形式検証は domain.NewOperatorSessionID や domain.NewTokenJTI に委譲する。
type IDGenerator interface {
	Next() (string, error)
}

// AdminAuthConfig は Admin operator auth use case の token/cookie lifetime と RP 表示情報を保持する。
//
// 役割:
//   - accessToken TTL、refresh session TTL、Cookie lifetime を application 起動時に検証できる単位にまとめる。
//   - refresh Cookie lifetime が server-side TTL を超えないよう shared token primitive で検証する。
//   - WebAuthn RP ID は challenge response DTO に渡すだけで、Product RP 設定とは混在させない。
type AdminAuthConfig struct {
	AccessTokenTTL        time.Duration
	RefreshSessionTTL     time.Duration
	RefreshCookieLifetime time.Duration
	WebAuthnRPID          string
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

// RefreshOperatorSessionInput は Admin refresh Cookie から accessToken と Cookie rotation を行う入力である。
//
// RefreshCookieValue は HttpOnly Cookie から adapter が読み取った値であり、response body へは戻さない。
type RefreshOperatorSessionInput struct {
	RefreshCookieValue string
}

// CurrentOperatorInput は bearer accessToken から現在の Admin Operator を取得する入力である。
//
// AccessToken は Authorization header から抽出された compact token で、Product bearer token と共有しない。
type CurrentOperatorInput struct {
	AccessToken string
}

// ValidateOperatorMutationInput は Admin mutation 前に accessToken/session/CSRF/permission を検証する入力である。
//
// Permission は accounts:create など Admin OperatorAuth domain が所有する permission 名である。
type ValidateOperatorMutationInput struct {
	AccessToken string
	CSRFToken   string
	Permission  string
}

// AuthorizeAccountCreationInput は accessToken と CSRF token から account 作成権限を検証する入力である。
//
// 役割:
//   - HTTP handler が Authorization header と X-CSRF-Token header から得た値だけを application auth facade へ渡す。
//   - Permission field を持たず、accounts:create の選択を application method 内へ固定する。
//   - Product bearer token や Product account role を Admin RBAC の判定材料として混入させない。
type AuthorizeAccountCreationInput struct {
	AccessToken string
	CSRFToken   string
}

// OperatorAuthorizationInput は Admin handler が検証済み operator context を application authorization use case へ渡す入力である。
//
// 役割:
//   - HTTP adapter が保持する operator ID/email/role/active/passkey 登録状態だけを application DTO として受け取る。
//   - Product account role や Product bearer token を含めず、Admin RBAC 判定材料を Operator domain object に限定する。
//   - permission 文字列は handler から受け取らず、専用 use case が accounts:create を内部で選ぶことで handler 側の RBAC 分岐を防ぐ。
type OperatorAuthorizationInput struct {
	OperatorID               string
	OperatorEmail            string
	OperatorRole             string
	OperatorActive           bool
	PasskeyRegistrationState string
}

// AuthorizationDecision は Admin RBAC use case の許可結果を表す DTO である。
//
// 役割:
//   - handler が domain.Operator を直接 import せず、許可済み operator の最小情報だけを受け取れるようにする。
//   - Allowed は use case が許可した場合のみ true になり、拒否時は error を返すため handler が再判定しない。
//   - Permission は監査 use case が後続で intent を作る際に stable permission 名を参照できるよう保持する。
type AuthorizationDecision struct {
	OperatorID string
	Permission string
	Allowed    bool
}

// LogoutOperatorInput は Admin operator session revoke の入力である。
//
// AccessToken は対象 session selector と OperatorID を特定するために使い、refreshToken 平文は不要である。
type LogoutOperatorInput struct {
	AccessToken string
}

// IssueOperatorSessionInput は setup 完了直後に Operator session を発行する入力である。
//
// OperatorID は initial setup または operator setup transaction が登録済みにした Operator の canonical ID である。
// refreshToken 平文は戻り値の Cookie command にだけ含まれ、response body には入らない。
type IssueOperatorSessionInput struct {
	OperatorID string
}

// ListOperatorPasskeysInput は Admin operator 自身の passkey 一覧取得入力である。
//
// OperatorID は middleware が Admin accessToken/session/CSRF を検証した後に context へ束縛した値だけを渡す。
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

// RefreshCookieCommand は adapter が HttpOnly refresh Cookie を設定または削除するための command DTO である。
//
// Value は response body へ含めず、Set-Cookie header 専用として扱う。
// Clear が true の場合、adapter は同じ名前/path の Cookie を削除する。
type RefreshCookieCommand struct {
	Name     string
	Value    string
	MaxAge   time.Duration
	HTTPOnly bool
	Secure   bool
	SameSite string
	Path     string
	Clear    bool
}

// OperatorSessionResult は login/refresh 成功時に Admin frontend へ返す browser-readable session DTO である。
//
// AccessToken と CSRFToken は含むが、refreshToken 平文は RefreshCookie command の Value にだけ保持される。
type OperatorSessionResult struct {
	AccessToken   string
	CSRFToken     string
	Operator      OperatorDTO
	SessionID     string
	ExpiresAt     time.Time
	RefreshCookie RefreshCookieCommand `json:"-"`
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
