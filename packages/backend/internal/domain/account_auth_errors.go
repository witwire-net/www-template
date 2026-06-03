package domain

import "errors"

var (
	// ErrAuthStoreUnavailable は認証関連の永続化層（Redis/Postgres）が利用不能な場合に返す。
	// 接続エラーやタイムアウト時に使用し、fail-close の識別子となる。
	ErrAuthStoreUnavailable = errors.New("auth store unavailable")
	// ErrAccountAuthNotFound は指定されたアカウントの認証情報（パスキー登録等）が永続化層に存在しない場合に返す。
	ErrAccountAuthNotFound = errors.New("account auth not found")
	// ErrChallengeNotFound は認証チャレンジが永続化層に存在しない場合に返す。
	ErrChallengeNotFound = errors.New("challenge not found")
	// ErrChallengeExpired は認証チャレンジの有効期限が切れている場合に返す。
	ErrChallengeExpired = errors.New("challenge expired")
	// ErrInvalidChallenge は認証チャレンジの形式や内容が不正な場合に返す。
	ErrInvalidChallenge = errors.New("challenge is invalid")
	// ErrSessionNotFound はセッションが永続化層に存在しない場合に返す。
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired はセッションの有効期限が切れている場合に返す。
	ErrSessionExpired = errors.New("session expired")
	// ErrSessionRevoked はセッションが明示的に取り消されている場合に返す。
	ErrSessionRevoked = errors.New("session revoked")
	// ErrRecoveryTokenNotFound はリカバリートークンが永続化層に存在しない場合に返す。
	ErrRecoveryTokenNotFound = errors.New("recovery token not found")
	// ErrRecoveryTokenExpired はリカバリートークンの有効期限が切れている場合に返す。
	ErrRecoveryTokenExpired = errors.New("recovery token expired")
	// ErrRecoveryTokenConsumed はリカバリートークンが既に消費済みの場合に返す。
	ErrRecoveryTokenConsumed = errors.New("recovery token consumed")
	// ErrRecoverySessionNotFound はリカバリーセッションが永続化層に存在しない場合に返す。
	ErrRecoverySessionNotFound = errors.New("recovery session not found")
	// ErrRecoverySessionExpired はリカバリーセッションの有効期限が切れている場合に返す。
	ErrRecoverySessionExpired = errors.New("recovery session expired")
	// ErrRecoverySessionConsumed はリカバリーセッションが既に消費済みの場合に返す。
	ErrRecoverySessionConsumed = errors.New("recovery session consumed")
	// ErrReauthSessionNotFound は再認証セッションが永続化層に存在しない場合に返す。
	ErrReauthSessionNotFound = errors.New("reauthentication session not found")
	// ErrReauthSessionExpired は再認証セッションの有効期限が切れている場合に返す。
	ErrReauthSessionExpired = errors.New("reauthentication session expired")
	// ErrReauthSessionConsumed は再認証セッションが既に消費済みの場合に返す。
	ErrReauthSessionConsumed = errors.New("reauthentication session consumed")
	// ErrReauthSessionKindMismatch は再認証セッションの種別が期待値と一致しない場合に返す。
	ErrReauthSessionKindMismatch = errors.New("reauthentication session kind mismatch")
	// ErrAuthTemporarilyLocked は認証フローが一時的にロックされている場合に返す。
	// レート制限やスロットリングの結果として発生する。
	ErrAuthTemporarilyLocked = errors.New("auth flow is temporarily locked")
	// ErrAuthBranchAmbiguous は認証フローの分岐セレクタ（パスキー/リカバリー等）が exactly one でない場合に返す。
	ErrAuthBranchAmbiguous = errors.New("exactly one auth branch selector is required")
	// ErrRecoveryStateRequired はリカバリー操作にセッションが必要な場合に返す。
	ErrRecoveryStateRequired = errors.New("recovery session is required")
	// ErrInvalidOpaqueSecret は opaque secret が空または不正な場合に返す。
	ErrInvalidOpaqueSecret = errors.New("opaque secret is required")
	// ErrInvalidPasskeyCredential はパスキー credential ID が空または不正な場合に返す。
	ErrInvalidPasskeyCredential = errors.New("passkey credential id is required")
	// ErrInvalidSessionID はセッション ID が空または不正な場合に返す。
	ErrInvalidSessionID = errors.New("session id is required")
	// ErrInvalidToken はトークンが空または不正な場合に返す。
	ErrInvalidToken = errors.New("token is required")
	// ErrInvalidSessionExpiry はセッションの有効期限が空または不正な場合に返す。
	ErrInvalidSessionExpiry = errors.New("session expiry is required")
	// ErrInvalidTokenKind は recovery token/session の kind が空または無効な場合に返す。
	ErrInvalidTokenKind = errors.New("token kind is required")
)
