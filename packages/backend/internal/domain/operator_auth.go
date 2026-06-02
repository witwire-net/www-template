package domain

import (
	"errors"
	"time"
)

var (
	// ErrOperatorAuthInactive は Operator が無効化されており Admin auth token を利用できない場合に返すエラーである。
	// Product Account の suspended 判定とは別の Admin Operator 固有 eligibility として扱う。
	ErrOperatorAuthInactive = errors.New("operator auth inactive")

	// ErrOperatorAuthPermissionDenied は Operator が要求 permission を満たさない場合に返すエラーである。
	// viewer や setup 未完了 Operator を mutation 実行へ進めないため、Admin auth domain が fail-closed に返す。
	ErrOperatorAuthPermissionDenied = errors.New("operator auth permission denied")

	// ErrOperatorAuthSnapshotMismatch は token/session に保存された role または active snapshot が現在の Operator と一致しない場合に返すエラーである。
	// role downgrade や無効化後に古い token/session が mutation へ使われることを防ぐ。
	ErrOperatorAuthSnapshotMismatch = errors.New("operator auth snapshot mismatch")

	// ErrOperatorAuthSessionMismatch は access token claims と refresh session state の Operator session ID が一致しない場合に返すエラーである。
	// Admin Operator session と Product Account session を混ぜないため、OperatorAuth 固有の session 照合エラーとして分離する。
	ErrOperatorAuthSessionMismatch = errors.New("operator auth session mismatch")
)

// OperatorAuthPermission は Admin OperatorAuth が評価する permission 名を表す値オブジェクトである。
//
// Product AccountAuth の scope や status とは無関係であり、Admin Operator の RBAC 判定だけに使う。
type OperatorAuthPermission string

const (
	// OperatorAuthPermissionAccountsCreate は Admin Console から顧客 Account を作成する mutation permission である。
	OperatorAuthPermissionAccountsCreate OperatorAuthPermission = "accounts:create"

	// OperatorAuthPermissionPasskeysManage は Operator 自身の passkey credential を管理する permission である。
	// viewer を含む有効な登録済み Operator が、自分の認証手段を維持できるよう account mutation 権限とは分離する。
	OperatorAuthPermissionPasskeysManage OperatorAuthPermission = "operator-passkeys:manage"

	// OperatorAuthPermissionOperatorsCreate は Admin operator の作成と setup token delivery を実行する permission である。
	// 運用者の追加は権限拡張につながるため、admin role のみに限定する。
	OperatorAuthPermissionOperatorsCreate OperatorAuthPermission = "operators:create"

	// OperatorAuthPermissionOperatorsLogout は現在の Admin operator session を失効する permission である。
	// logout は本人 session の破棄だけを行うため、登録済みの全 role に許可する。
	OperatorAuthPermissionOperatorsLogout OperatorAuthPermission = "operators:logout"
)

// OperatorAccessTokenClaims は Admin Operator accessToken に入れる domain claims を表す値オブジェクトである。
//
// 役割:
//   - OperatorID、OperatorSessionID、JTI、role/active snapshot、発行時刻、有効期限を保持する。
//   - Product Account accessToken claims とは別型にし、Admin auth session とだけ照合できるようにする。
//   - JWT 署名や JSON payload 変換は中立 token primitive または adapter/application に残し、この型は意味検証だけを担う。
//
// 使用例:
//
//	claims, err := NewOperatorAccessTokenClaims(operator, session, jti, ttl, issuedAt)
//	if err != nil {
//		return err
//	}
type OperatorAccessTokenClaims struct {
	operatorID OperatorID
	sessionID  OperatorSessionID
	tokenID    TokenJTI
	role       OperatorRole
	active     bool
	issuedAt   time.Time
	expiresAt  time.Time
}

// NewOperatorAccessTokenClaims は Operator と session state から Admin accessToken claims を生成する。
//
// 引数:
//   - operator: 現在の Admin Operator。active/role snapshot の正として使う。
//   - session: Operator refresh session state。claims の session ID と snapshot を session に合わせる。
//   - tokenID: accessToken の jti。ULID 形式で検証済みである必要がある。
//   - ttl: accessToken lifetime。0 以下は token primitive 側で拒否される。
//   - issuedAt: 外部 clock から渡された発行時刻。zero time は ErrInvalidSessionExpiry とする。
//
// 戻り値:
//   - OperatorAccessTokenClaims: Admin Operator 専用の検証済み claims。
//   - error: Operator/session/token が Admin auth eligibility を満たさない場合の domain error。
func NewOperatorAccessTokenClaims(
	operator Operator,
	session OperatorAuthSession,
	tokenID TokenJTI,
	ttl TokenTTL,
	issuedAt time.Time,
) (OperatorAccessTokenClaims, error) {
	// Step 1: 発行時刻がない claims は有効期限判定ができないため拒否する。
	if issuedAt.IsZero() {
		return OperatorAccessTokenClaims{}, ErrInvalidSessionExpiry
	}

	// Step 2: 現在 Operator と session snapshot が一致することを確認し、古い session からの発行を防ぐ。
	if err := session.validateOperatorSnapshot(operator); err != nil {
		return OperatorAccessTokenClaims{}, err
	}

	// Step 3: JTI は token primitive の ULID 規則に戻して再検証し、手組み値の混入を拒否する。
	validatedTokenID, err := NewTokenJTI(tokenID.String())
	if err != nil {
		return OperatorAccessTokenClaims{}, err
	}

	// Step 4: TTL から deterministic に有効期限を計算し、domain 層では現在時刻を読まない。
	expiresAt := ttl.ExpiresAt(issuedAt)
	if !expiresAt.After(issuedAt.UTC()) {
		return OperatorAccessTokenClaims{}, ErrInvalidTokenTTL
	}

	// Step 5: session と現在 Operator の snapshot を claims に固定して返す。
	return OperatorAccessTokenClaims{
		operatorID: operator.ID(),
		sessionID:  session.ID(),
		tokenID:    validatedTokenID,
		role:       operator.Role(),
		active:     operator.Active(),
		issuedAt:   issuedAt.UTC(),
		expiresAt:  expiresAt,
	}, nil
}

// ReconstituteOperatorAccessTokenClaims は署名済み JSON payload から Admin OperatorAuth claims を復元する。
//
// 役割:
//   - application 層で decode した primitive 値を domain claim snapshot に戻し、snapshot / expiry / permission 判定を OperatorAuthSession.ValidateAccess へ集約する。
//   - JWT 署名や JSON field の必須性は application が担当し、この helper は Admin OperatorAuth として意味を持つ値だけを検証する。
//   - Product Account claims と混在しないよう、OperatorID と OperatorSessionID の Admin 専用 constructor を必ず通す。
//
// 引数:
//   - operatorID: accessToken `sub` から復元した Admin OperatorID。
//   - sessionID: accessToken `sid` から復元した Admin OperatorSessionID。
//   - tokenID: accessToken `jti` から復元した token ID。
//   - roleSnapshot: accessToken 発行時点の Operator role snapshot。
//   - activeSnapshot: accessToken 発行時点の Operator active snapshot。
//   - issuedAt: accessToken `iat` の UTC 時刻。zero time は拒否される。
//   - expiresAt: accessToken `exp` の UTC 時刻。issuedAt より後でなければならない。
//
// 戻り値:
//   - OperatorAccessTokenClaims: 復元済み Admin OperatorAuth claims。
//   - error: ID、role、時刻、TTL が不正な場合の domain error。
//
// 使用例:
//
//	claims, err := ReconstituteOperatorAccessTokenClaims(operatorID, sessionID, jti, role, active, issuedAt, expiresAt)
//	if err != nil {
//		return err
//	}
func ReconstituteOperatorAccessTokenClaims(
	operatorID OperatorID,
	sessionID OperatorSessionID,
	tokenID TokenJTI,
	roleSnapshot OperatorRole,
	activeSnapshot bool,
	issuedAt time.Time,
	expiresAt time.Time,
) (OperatorAccessTokenClaims, error) {
	// Step 1: Admin OperatorID / session ID は専用 constructor で再検証し、Product ID や空値を拒否する。
	validatedOperatorID, err := NewOperatorID(operatorID.String())
	if err != nil {
		return OperatorAccessTokenClaims{}, err
	}
	validatedSessionID, err := NewOperatorSessionID(sessionID.String())
	if err != nil {
		return OperatorAccessTokenClaims{}, err
	}

	// Step 2: JTI と role snapshot を domain value として検証し、未知 role の署名 payload を fail-closed にする。
	validatedTokenID, err := NewTokenJTI(tokenID.String())
	if err != nil {
		return OperatorAccessTokenClaims{}, err
	}
	if err := roleSnapshot.Validate(); err != nil {
		return OperatorAccessTokenClaims{}, err
	}

	// Step 3: iat/exp から TTL を検証し、期限が逆転した署名済み payload を拒否する。
	if issuedAt.IsZero() || !expiresAt.UTC().After(issuedAt.UTC()) {
		return OperatorAccessTokenClaims{}, ErrInvalidTokenTTL
	}
	if _, err := ValidateTokenTTL(expiresAt.UTC().Sub(issuedAt.UTC())); err != nil {
		return OperatorAccessTokenClaims{}, err
	}

	// Step 4: 復元済み snapshot を返し、現在 Operator/session との照合は ValidateAccess に委譲する。
	return OperatorAccessTokenClaims{
		operatorID: validatedOperatorID,
		sessionID:  validatedSessionID,
		tokenID:    validatedTokenID,
		role:       roleSnapshot,
		active:     activeSnapshot,
		issuedAt:   issuedAt.UTC(),
		expiresAt:  expiresAt.UTC(),
	}, nil
}

// OperatorID は claims の Admin Operator ID を返す。
//
// 戻り値は Product AccountID ではなく、Operator auth / audit にだけ使う OperatorID である。
func (c OperatorAccessTokenClaims) OperatorID() OperatorID { return c.operatorID }

// SessionID は claims が紐づく Admin Operator session ID を返す。
//
// 戻り値は OperatorAuthSession.ID と照合するための OperatorSessionID である。
func (c OperatorAccessTokenClaims) SessionID() OperatorSessionID { return c.sessionID }

// TokenID は claims の jti を返す。
//
// 戻り値は token replay 追跡や署名 payload に使う TokenJTI であり、権限判断は持たない。
func (c OperatorAccessTokenClaims) TokenID() TokenJTI { return c.tokenID }

// RoleSnapshot は accessToken 発行時点の Operator role snapshot を返す。
//
// 戻り値は現在 Operator と session snapshot の一致確認に使い、Product account status とは関係しない。
func (c OperatorAccessTokenClaims) RoleSnapshot() OperatorRole { return c.role }

// ActiveSnapshot は accessToken 発行時点の Operator active snapshot を返す。
//
// false の場合、Admin mutation eligibility は必ず拒否される。
func (c OperatorAccessTokenClaims) ActiveSnapshot() bool { return c.active }

// IssuedAt は accessToken の発行時刻を UTC で返す。
//
// 戻り値は session revoke や監査相関で利用でき、副作用はない。
func (c OperatorAccessTokenClaims) IssuedAt() time.Time { return c.issuedAt }

// ExpiresAt は accessToken の有効期限を UTC で返す。
//
// now がこの時刻以降の場合、ValidateForOperator は ErrTokenExpired を返す。
func (c OperatorAccessTokenClaims) ExpiresAt() time.Time { return c.expiresAt }

// ValidateForOperator は claims が現在 Operator と要求 permission に対して有効かを検証する。
//
// 引数:
//   - operator: 現在の Operator snapshot。active/role/passkey state を検証する。
//   - permission: Admin mutation に必要な permission。空または未知 permission は拒否される。
//   - now: 外部 clock から渡された検証時刻。
//
// 戻り値:
//   - error: 成功時 nil。期限切れ、snapshot mismatch、権限不足の場合は対応する domain error。
func (c OperatorAccessTokenClaims) ValidateForOperator(operator Operator, permission OperatorAuthPermission, now time.Time) error {
	// Step 1: token の有効期限を検証し、期限切れ claims を Admin mutation へ進めない。
	if !now.UTC().Before(c.expiresAt.UTC()) {
		return ErrTokenExpired
	}

	// Step 2: claims の Operator ID が現在 Operator と一致することを検証する。
	if c.operatorID != operator.ID() {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 3: claims の role/active snapshot が現在 Operator と一致することを検証する。
	if c.role != operator.Role() || c.active != operator.Active() {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 4: inactive snapshot は role に関係なく Admin auth eligibility を拒否する。
	if !c.active || !operator.Active() {
		return ErrOperatorAuthInactive
	}

	// Step 5: permission は Operator domain の HasPermission に委譲し、viewer や setup 未完了状態を拒否する。
	if !operator.HasPermission(permission.String()) {
		return ErrOperatorAuthPermissionDenied
	}

	// Step 6: すべての Admin OperatorAuth claims 条件を満たしたため成功とする。
	return nil
}

// ValidateCurrentForOperator は claims が現在 Operator の read/current context として有効かを検証する。
//
// 役割:
//   - current endpoint のような permission 非依存の境界で、期限切れ・snapshot mismatch・inactive・未登録を domain error として判定する。
//   - mutation permission の role matrix は ValidateForOperator に残し、この method は現在 Operator としての基本 eligibility だけを扱う。
//
// 引数:
//   - operator: 現在の Operator snapshot。active/role/passkey state を検証する。
//   - now: 外部 clock から渡された検証時刻。
//
// 戻り値:
//   - error: 成功時 nil。期限切れ、snapshot mismatch、inactive、未登録の場合は domain error。
//
// 使用例:
//
//	if err := claims.ValidateCurrentForOperator(operator, now); err != nil {
//		return err
//	}
func (c OperatorAccessTokenClaims) ValidateCurrentForOperator(operator Operator, now time.Time) error {
	// Step 1: token の有効期限を検証し、期限切れ claims を current context へ進めない。
	if !now.UTC().Before(c.expiresAt.UTC()) {
		return ErrTokenExpired
	}

	// Step 2: claims の Operator ID が現在 Operator と一致することを検証する。
	if c.operatorID != operator.ID() {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 3: claims の role/active snapshot が現在 Operator と一致することを検証する。
	if c.role != operator.Role() || c.active != operator.Active() {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 4: inactive または passkey 未登録 Operator は current context としても受け入れない。
	if !c.active || !operator.Active() || operator.PasskeyRegistrationState() != OperatorPasskeyRegistrationRegistered {
		return ErrOperatorAuthInactive
	}

	// Step 5: current operator としての基本 eligibility を満たしたため成功とする。
	return nil
}

// String は OperatorAuthPermission を Operator.HasPermission へ渡す canonical 文字列に変換する。
//
// 戻り値は Admin OperatorAuth が所有する permission 名であり、Product AccountAuth scope ではない。
func (p OperatorAuthPermission) String() string { return string(p) }
