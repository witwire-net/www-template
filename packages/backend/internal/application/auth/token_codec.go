package auth

import (
	"strings"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// operatorAccessPayload は Admin operator accessToken の署名 payload を表す内部 DTO である。
//
// Product AccountAuth claim と混同しないよう、operator ID、operator session ID、role/active snapshot だけを保持する。
type operatorAccessPayload struct {
	OperatorID string `json:"sub"`
	SessionID  string `json:"sid"`
	TokenID    string `json:"jti"`
	Role       string `json:"role"`
	Active     bool   `json:"active"`
	IssuedAt   int64  `json:"iat"`
	ExpiresAt  int64  `json:"exp"`
}

func (p operatorAccessPayload) validate() error {
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

func (p operatorAccessPayload) domainClaims() (domain.OperatorAccessTokenClaims, error) {
	// Step 1: payload の各 primitive を Admin OperatorAuth 専用 value object へ復元する。
	operatorID, err := domain.NewOperatorID(p.OperatorID)
	if err != nil {
		return zeroOperatorAccessTokenClaims(), err
	}
	sessionID, err := domain.NewOperatorSessionID(p.SessionID)
	if err != nil {
		return zeroOperatorAccessTokenClaims(), err
	}
	tokenID, err := domain.NewTokenJTI(p.TokenID)
	if err != nil {
		return zeroOperatorAccessTokenClaims(), err
	}

	// Step 2: signed JSON の時刻 claim を UTC time に戻し、期限・TTL 検証は domain reconstitution helper に委譲する。
	issuedAt := time.Unix(p.IssuedAt, 0).UTC()
	expiresAt := time.Unix(p.ExpiresAt, 0).UTC()

	// Step 3: payload snapshot を domain claims へ復元し、application 側に snapshot/expiry 判定を重複させない。
	return domain.ReconstituteOperatorAccessTokenClaims(operatorID, sessionID, tokenID, domain.OperatorRole(p.Role), p.Active, issuedAt, expiresAt)
}

func zeroOperatorAccessTokenClaims() domain.OperatorAccessTokenClaims {
	// Step 1: error return 専用の zero value を var 経由で作り、claim 生成は domain constructor/reconstitution helper に限定する。
	var claims domain.OperatorAccessTokenClaims
	return claims
}

func parseRefreshCookieSessionID(cookieValue string) (domain.OperatorSessionID, error) {
	// Step 1: Cookie value は sessionID.secret 形式であり、session selector だけを先頭 segment から取り出す。
	dotIndex := strings.Index(cookieValue, ".")
	if dotIndex <= 0 || dotIndex == len(cookieValue)-1 {
		return "", ErrOperatorAuthInvalidInput
	}

	// Step 2: 取り出した selector を Admin Operator session ID として検証する。
	sessionID, err := domain.NewOperatorSessionID(cookieValue[:dotIndex])
	if err != nil {
		return "", ErrOperatorAuthInvalidInput
	}

	// Step 3: 有効な selector を返し、secret 部分はこの関数の外へ露出しない。
	return sessionID, nil
}

func validateCurrentOperatorPayload(operator domain.Operator, session domain.OperatorAuthSession, payload operatorAccessPayload, now time.Time) error {
	// Step 1: JSON payload DTO を domain claims へ復元し、snapshot/expiry/current eligibility は domain に委譲する。
	claims, err := payload.domainClaims()
	if err != nil {
		return mapAdminDomainAuthError(err)
	}
	if err := session.ValidateCurrentAccess(operator, claims, now); err != nil {
		return mapAdminDomainAuthError(err)
	}

	// Step 2: current operator として有効なため成功とする。
	return nil
}

func validateOperatorAccessPayload(operator domain.Operator, session domain.OperatorAuthSession, payload operatorAccessPayload, permission domain.OperatorAuthPermission, now time.Time) error {
	// Step 1: payload の primitive claim を Admin OperatorAuth claims へ復元する。
	claims, err := payload.domainClaims()
	if err != nil {
		return mapAdminDomainAuthError(err)
	}

	// Step 2: session.ValidateAccess に snapshot、expiry、permission 判定をすべて委譲する。
	if err := session.ValidateAccess(operator, claims, permission, now); err != nil {
		return mapAdminDomainAuthError(err)
	}

	// Step 3: OperatorAuth domain が対象 permission を許可したため成功とする。
	return nil
}
