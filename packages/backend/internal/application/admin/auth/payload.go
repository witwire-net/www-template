package application

import (
	"strings"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// operatorAccessTokenPayload は Admin operator accessToken の署名 payload を表す内部 DTO である。
//
// Product AccountAuth claim と混同しないよう、operator ID、operator session ID、role/active snapshot だけを保持する。
type operatorAccessTokenPayload struct {
	OperatorID string `json:"sub"`
	SessionID  string `json:"sid"`
	TokenID    string `json:"jti"`
	Role       string `json:"role"`
	Active     bool   `json:"active"`
	IssuedAt   int64  `json:"iat"`
	ExpiresAt  int64  `json:"exp"`
}

func (p operatorAccessTokenPayload) validate() error {
	// Step 1: 各 ID と role は domain constructor へ戻し、Admin OperatorAuth の型として検証する。
	if _, err := domain.NewOperatorID(p.OperatorID); err != nil {
		return err
	}
	if _, err := domain.NewOperatorSessionID(p.SessionID); err != nil {
		return err
	}
	if _, err := domain.NewTokenJTI(p.TokenID); err != nil {
		return err
	}
	if err := domain.OperatorRole(p.Role).Validate(); err != nil {
		return err
	}

	// Step 2: 発行・失効時刻は正の範囲でなければ accessToken として拒否する。
	if p.IssuedAt <= 0 || p.ExpiresAt <= p.IssuedAt {
		return domain.ErrInvalidSessionExpiry
	}

	// Step 3: 必須文字列の空白のみ入力を明示的に拒否し、署名済みでも壊れた payload を通さない。
	if strings.TrimSpace(p.OperatorID) == "" || strings.TrimSpace(p.SessionID) == "" || strings.TrimSpace(p.TokenID) == "" {
		return domain.ErrInvalidToken
	}

	// Step 4: payload が Admin accessToken として復元可能であるため成功とする。
	return nil
}

func (p operatorAccessTokenPayload) matchesClaims(claims domain.OperatorAccessTokenClaims) bool {
	// Step 1: payload と domain claims の意味値が一致するかを比較する。
	return p.OperatorID == claims.OperatorID().String() &&
		p.SessionID == claims.SessionID().String() &&
		p.TokenID == claims.TokenID().String() &&
		p.Role == string(claims.RoleSnapshot()) &&
		p.Active == claims.ActiveSnapshot() &&
		p.IssuedAt == claims.IssuedAt().Unix() &&
		p.ExpiresAt == claims.ExpiresAt().Unix()
}

func parseRefreshCookieSessionID(cookieValue string) (domain.OperatorSessionID, error) {
	// Step 1: Cookie value は sessionID.secret 形式であり、session selector だけを先頭 segment から取り出す。
	dotIndex := strings.Index(cookieValue, ".")
	if dotIndex <= 0 || dotIndex == len(cookieValue)-1 {
		return "", ErrAdminAuthBadRequest
	}

	// Step 2: 取り出した selector を Admin Operator session ID として検証する。
	sessionID, err := domain.NewOperatorSessionID(cookieValue[:dotIndex])
	if err != nil {
		return "", ErrAdminAuthBadRequest
	}

	// Step 3: 有効な selector を返し、secret 部分はこの関数の外へ露出しない。
	return sessionID, nil
}

func validateCurrentOperatorPayload(operator domain.Operator, session domain.OperatorAuthSession, payload operatorAccessTokenPayload, now time.Time) error {
	// Step 1: session が revoked または期限切れの場合は current operator として扱わない。
	if session.Revoked() || !now.UTC().Before(session.ExpiresAt()) {
		return ErrAdminAuthUnauthenticated
	}

	// Step 2: token payload、session、現在 Operator の owner/snapshot が一致することを確認する。
	if payload.OperatorID != operator.ID().String() || payload.OperatorID != session.OperatorID().String() {
		return ErrAdminAuthUnauthenticated
	}
	if payload.SessionID != session.ID().String() {
		return ErrAdminAuthUnauthenticated
	}
	if payload.Role != string(operator.Role()) || payload.Role != string(session.RoleSnapshot()) {
		return ErrAdminAuthUnauthenticated
	}
	if payload.Active != operator.Active() || payload.Active != session.ActiveSnapshot() {
		return ErrAdminAuthUnauthenticated
	}

	// Step 3: accessToken 自体の有効期限と active state を検証する。
	if !now.UTC().Before(time.Unix(payload.ExpiresAt, 0).UTC()) {
		return ErrAdminAuthUnauthenticated
	}
	if !operator.Active() || !payload.Active {
		return ErrAdminAuthForbidden
	}

	// Step 4: current operator として有効なため成功とする。
	return nil
}
