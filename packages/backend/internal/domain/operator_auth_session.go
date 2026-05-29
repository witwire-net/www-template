package domain

import "time"

// OperatorSessionID は Admin Operator refresh session を識別する canonical ULID 値オブジェクトである。
//
// Product Account session ID と別型にすることで、Admin OperatorAuth の refresh state や CSRF binding に
// Product account auth の session ID が誤って流入することを防ぐ。
type OperatorSessionID string

// NewOperatorSessionID は raw 文字列を検証し、Admin Operator 専用 session ID を返す。
//
// raw は前後空白を除去した後、TokenULID と同じ ULID 形式だけを受け付ける。
// 不正な場合は ErrInvalidSessionID を返し、Admin auth session lookup を fail-closed にできる。
func NewOperatorSessionID(raw string) (OperatorSessionID, error) {
	// Step 1: 中立 TokenULID primitive に形式検証を委譲し、session の意味はこの型で付与する。
	id, err := NewTokenULID(raw)
	if err != nil {
		return "", ErrInvalidSessionID
	}

	// Step 2: 検証済み ULID だけを OperatorSessionID として返す。
	return OperatorSessionID(id.String()), nil
}

// String は OperatorSessionID を DB key、JWT claim、監査相関へ渡す canonical 文字列へ変換する。
//
// 戻り値は NewOperatorSessionID で検証済みの ULID 文字列であり、副作用はない。
func (id OperatorSessionID) String() string { return string(id) }

// OperatorAuthSession は Admin Operator refresh session state と CSRF binding を表す domain object である。
//
// 役割:
//   - refresh token hash、CSRF token hash、role/active snapshot、有効期限、revoke state を保持する。
//   - Product AccountAuth session とは別型にし、Admin Operator 固有の eligibility と RBAC だけを評価する。
//   - refreshToken / csrfToken 平文は保持せず、中立 OpaqueTokenHash で照合する。
type OperatorAuthSession struct {
	id               OperatorSessionID
	operatorID       OperatorID
	refreshTokenHash OpaqueTokenHash
	csrfTokenHash    OpaqueTokenHash
	roleSnapshot     OperatorRole
	activeSnapshot   bool
	issuedAt         time.Time
	expiresAt        time.Time
	revoked          bool
}

// NewOperatorAuthSession は Admin Operator refresh session state を新規作成する。
//
// 引数:
//   - operator: session を所有する Admin Operator。inactive Operator は拒否する。
//   - id: OperatorSessionID。未検証値は NewOperatorSessionID と同じ規則で拒否される。
//   - refreshToken: HttpOnly Cookie に入る refresh token 平文。domain では hash 化して保持する。
//   - csrfToken: Admin frontend が mutation で提示する CSRF token 平文。session に hash binding する。
//   - ttl: server-side refresh session lifetime。0 以下は拒否される。
//   - issuedAt: 外部 clock から渡された発行時刻。zero time は拒否される。
//
// 戻り値:
//   - OperatorAuthSession: 検証済み refresh session state。
//   - error: inactive Operator、無効 token、無効 ID、無効 TTL の場合の domain error。
func NewOperatorAuthSession(
	operator Operator,
	id OperatorSessionID,
	refreshToken string,
	csrfToken string,
	ttl TokenTTL,
	issuedAt time.Time,
) (OperatorAuthSession, error) {
	// Step 1: active でない Operator には Admin refresh session を発行しない。
	if !operator.Active() {
		return OperatorAuthSession{}, ErrOperatorAuthInactive
	}

	// Step 2: session ID、refresh token、CSRF token、時刻を共通 helper で検証し、平文 token は hash 化する。
	session, err := buildOperatorAuthSession(operator, id, refreshToken, csrfToken, ttl, issuedAt, false)
	if err != nil {
		return OperatorAuthSession{}, err
	}

	// Step 3: 作成直後の session state を返す。
	return session, nil
}

// ReconstituteOperatorAuthSession は永続化済み refresh session state を復元する。
//
// 引数:
//   - id/operatorID: Admin Operator session と所有者を表す canonical ID。
//   - refreshTokenHash/csrfTokenHash: 保存済み hash。空値は拒否される。
//   - roleSnapshot/activeSnapshot: session 発行時点の Operator snapshot。
//   - issuedAt/expiresAt: 保存済み時刻。zero time や逆転した期限は拒否される。
//   - revoked: logout/revoke 済みかどうか。
//
// 戻り値:
//   - OperatorAuthSession: 復元済み session state。
//   - error: 保存値が Admin OperatorAuth の不変条件を満たさない場合の domain error。
func ReconstituteOperatorAuthSession(
	id OperatorSessionID,
	operatorID OperatorID,
	refreshTokenHash OpaqueTokenHash,
	csrfTokenHash OpaqueTokenHash,
	roleSnapshot OperatorRole,
	activeSnapshot bool,
	issuedAt time.Time,
	expiresAt time.Time,
	revoked bool,
) (OperatorAuthSession, error) {
	// Step 1: 既存 state をそのまま組み立て、validateOperatorAuthSessionState で全不変条件を検証する。
	session := OperatorAuthSession{
		id:               id,
		operatorID:       operatorID,
		refreshTokenHash: refreshTokenHash,
		csrfTokenHash:    csrfTokenHash,
		roleSnapshot:     roleSnapshot,
		activeSnapshot:   activeSnapshot,
		issuedAt:         issuedAt.UTC(),
		expiresAt:        expiresAt.UTC(),
		revoked:          revoked,
	}

	// Step 2: 永続化値が壊れている場合、復元時点で fail-closed にする。
	if err := validateOperatorAuthSessionState(session); err != nil {
		return OperatorAuthSession{}, err
	}

	// Step 3: 検証済み state だけを返す。
	return session, nil
}

// ID は Admin Operator refresh session ID を返す。
//
// 戻り値は accessToken claims の sid と照合するために使う。
func (s OperatorAuthSession) ID() OperatorSessionID { return s.id }

// OperatorID は session を所有する Admin Operator ID を返す。
//
// 戻り値は Product AccountID ではなく、OperatorAuth 専用の所有者 ID である。
func (s OperatorAuthSession) OperatorID() OperatorID { return s.operatorID }

// RefreshTokenHash は保存済み refresh token hash を返す。
//
// 戻り値は平文 token ではなく、永続化や rotation 照合に使う digest である。
func (s OperatorAuthSession) RefreshTokenHash() OpaqueTokenHash { return s.refreshTokenHash }

// CSRFTokenHash は session に binding された CSRF token hash を返す。
//
// 戻り値は Admin mutation の CSRF 照合に使う digest であり、平文 token は含まない。
func (s OperatorAuthSession) CSRFTokenHash() OpaqueTokenHash { return s.csrfTokenHash }

// RoleSnapshot は session 発行時点の Operator role を返す。
//
// 現在 role と一致しない場合、ValidateAccess は snapshot mismatch として拒否する。
func (s OperatorAuthSession) RoleSnapshot() OperatorRole { return s.roleSnapshot }

// ActiveSnapshot は session 発行時点の Operator active state を返す。
//
// false の session は mutation eligibility を満たさないため拒否される。
func (s OperatorAuthSession) ActiveSnapshot() bool { return s.activeSnapshot }

// IssuedAt は session 発行時刻を UTC で返す。
//
// 戻り値は refresh rotation や監査相関の基準時刻に利用できる。
func (s OperatorAuthSession) IssuedAt() time.Time { return s.issuedAt }

// ExpiresAt は refresh session の server-side 有効期限を UTC で返す。
//
// now がこの時刻以降の場合、ValidateRefreshToken と ValidateAccess は ErrSessionExpired を返す。
func (s OperatorAuthSession) ExpiresAt() time.Time { return s.expiresAt }

// Revoked は session が logout/revoke 済みかどうかを返す。
//
// true の場合、Admin refresh/access eligibility は ErrSessionRevoked で拒否される。
func (s OperatorAuthSession) Revoked() bool { return s.revoked }

// Revoke は Admin Operator refresh session を失効済みにした新しい値を返す。
//
// 元の値は変更せず、logout や強制失効後の state 保存に使う。
func (s OperatorAuthSession) Revoke() OperatorAuthSession {
	// Step 1: 値オブジェクトとして session を複製し、revoked flag だけを有効にする。
	s.revoked = true

	// Step 2: 失効済み state を返す。
	return s
}

// ValidateRefreshToken は提示された refresh token が session state と一致し、利用可能かを検証する。
//
// refreshToken は HttpOnly Cookie から取得された平文を想定し、保存済み hash と constant-time に照合される。
// now は外部 clock から渡し、domain 層では time.Now を呼ばない。
func (s OperatorAuthSession) ValidateRefreshToken(refreshToken string, now time.Time) error {
	// Step 1: session 自体が有効期間内かつ未失効であることを確認する。
	if err := s.validateUsable(now); err != nil {
		return err
	}

	// Step 2: refresh token 平文を保存済み hash と照合し、不一致は ErrInvalidToken とする。
	if !s.refreshTokenHash.Matches(refreshToken) {
		return ErrInvalidToken
	}

	// Step 3: refresh token が session に一致したため成功とする。
	return nil
}

// ValidateCSRFToken は提示された CSRF token が session binding と一致することを検証する。
//
// Admin mutation では Cookie refresh/session と別経路で CSRF token を提示させ、この method で一致を確認する。
func (s OperatorAuthSession) ValidateCSRFToken(csrfToken string) error {
	// Step 1: CSRF token 平文を保存済み hash と照合し、session へ bind されていない token を拒否する。
	if !s.csrfTokenHash.Matches(csrfToken) {
		return ErrOperatorAuthCSRFMismatch
	}

	// Step 2: CSRF binding が一致したため成功とする。
	return nil
}

// ValidateAccess は session、accessToken claims、CSRF、permission をまとめて検証する。
//
// 引数:
//   - operator: 現在の Admin Operator。active/role/passkey state の正として使う。
//   - claims: Admin Operator accessToken claims。session ID と snapshot を照合する。
//   - permission: 実行したい Admin mutation permission。
//   - csrfToken: Admin frontend が提示した CSRF token 平文。
//   - now: 外部 clock から渡された検証時刻。
//
// 戻り値:
//   - error: 成功時 nil。inactive、viewer 権限不足、CSRF mismatch、session ID mismatch などを domain error で返す。
func (s OperatorAuthSession) ValidateAccess(
	operator Operator,
	claims OperatorAccessTokenClaims,
	permission OperatorAuthPermission,
	csrfToken string,
	now time.Time,
) error {
	// Step 1: refresh session state が期限内かつ未失効であることを確認する。
	if err := s.validateUsable(now); err != nil {
		return err
	}

	// Step 2: claims の session ID が refresh session state と一致することを確認する。
	if claims.SessionID() != s.id {
		return ErrOperatorAuthSessionMismatch
	}

	// Step 3: 現在 Operator と session snapshot が一致することを確認する。
	if err := s.validateOperatorSnapshot(operator); err != nil {
		return err
	}

	// Step 4: claims 側の期限・snapshot・permission を検証する。
	if err := claims.ValidateForOperator(operator, permission, now); err != nil {
		return err
	}

	// Step 5: claims と session の role/active snapshot が一致することを確認する。
	if claims.RoleSnapshot() != s.roleSnapshot || claims.ActiveSnapshot() != s.activeSnapshot {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 6: Admin mutation CSRF token が session binding と一致することを確認する。
	if err := s.ValidateCSRFToken(csrfToken); err != nil {
		return err
	}

	// Step 7: OperatorAuth の全 eligibility 条件を満たしたため成功とする。
	return nil
}

func buildOperatorAuthSession(
	operator Operator,
	id OperatorSessionID,
	refreshToken string,
	csrfToken string,
	ttl TokenTTL,
	issuedAt time.Time,
	revoked bool,
) (OperatorAuthSession, error) {
	// Step 1: session ID を Operator 専用型として再検証する。
	validatedID, err := NewOperatorSessionID(id.String())
	if err != nil {
		return OperatorAuthSession{}, err
	}

	// Step 2: refresh token と CSRF token は平文を保持せず hash 化する。
	refreshTokenHash, err := HashOpaqueToken(refreshToken)
	if err != nil {
		return OperatorAuthSession{}, err
	}
	csrfTokenHash, err := HashOpaqueToken(csrfToken)
	if err != nil {
		return OperatorAuthSession{}, err
	}

	// Step 3: 発行時刻と TTL から server-side session expiry を作る。
	if issuedAt.IsZero() {
		return OperatorAuthSession{}, ErrInvalidSessionExpiry
	}
	expiresAt := ttl.ExpiresAt(issuedAt)

	// Step 4: Operator の role/active を session snapshot として固定する。
	session := OperatorAuthSession{
		id:               validatedID,
		operatorID:       operator.ID(),
		refreshTokenHash: refreshTokenHash,
		csrfTokenHash:    csrfTokenHash,
		roleSnapshot:     operator.Role(),
		activeSnapshot:   operator.Active(),
		issuedAt:         issuedAt.UTC(),
		expiresAt:        expiresAt,
		revoked:          revoked,
	}

	// Step 5: 組み立て後の state を再検証し、不完全な session を返さない。
	if err := validateOperatorAuthSessionState(session); err != nil {
		return OperatorAuthSession{}, err
	}

	// Step 6: 検証済み session state を返す。
	return session, nil
}

func validateOperatorAuthSessionState(session OperatorAuthSession) error {
	// Step 1: session ID は Admin Operator 専用 ULID として再検証する。
	if _, err := NewOperatorSessionID(session.id.String()); err != nil {
		return err
	}

	// Step 2: owner OperatorID は Product AccountID ではなく OperatorID として再検証する。
	if _, err := NewOperatorID(session.operatorID.String()); err != nil {
		return err
	}

	// Step 3: role snapshot は既知 role だけを受け付ける。
	if err := session.roleSnapshot.Validate(); err != nil {
		return err
	}

	// Step 4: token hash は空値を許さず、平文 token 不在の session を拒否する。
	if session.refreshTokenHash == "" || session.csrfTokenHash == "" {
		return ErrInvalidToken
	}

	// Step 5: session の時刻は zero でなく、発行時刻より後に期限が来る必要がある。
	if session.issuedAt.IsZero() || !session.expiresAt.After(session.issuedAt) {
		return ErrInvalidSessionExpiry
	}

	// Step 6: すべての session state 不変条件を満たしたため成功とする。
	return nil
}

func (s OperatorAuthSession) validateUsable(now time.Time) error {
	// Step 1: revoked session は refresh/access の両方で拒否する。
	if s.revoked {
		return ErrSessionRevoked
	}

	// Step 2: server-side session TTL を過ぎた state は拒否する。
	if !now.UTC().Before(s.expiresAt.UTC()) {
		return ErrSessionExpired
	}

	// Step 3: session state が利用可能であるため成功とする。
	return nil
}

func (s OperatorAuthSession) validateOperatorSnapshot(operator Operator) error {
	// Step 1: session の owner と現在 Operator が一致することを確認する。
	if s.operatorID != operator.ID() {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 2: inactive Operator または inactive snapshot は Admin auth eligibility を満たさない。
	if !operator.Active() || !s.activeSnapshot {
		return ErrOperatorAuthInactive
	}

	// Step 3: session 発行時 role と現在 role が一致することを確認する。
	if s.roleSnapshot != operator.Role() {
		return ErrOperatorAuthSnapshotMismatch
	}

	// Step 4: snapshot が現在 Operator と一致したため成功とする。
	return nil
}
