package domain

import (
	"errors"
	"testing"
	"time"
)

const (
	operatorAuthTestSessionID         = "01ARZ3NDEKTSV4RRFFQ69G5FB1"
	operatorAuthTestMismatchSessionID = "01ARZ3NDEKTSV4RRFFQ69G5FB2"
	operatorAuthTestRefreshToken      = "refresh-token-for-admin-operator"
	operatorAuthTestJTIULID           = "01ARZ3NDEKTSV4RRFFQ69G5FB3"
)

// [AUTH-BE-S070] Admin OperatorAuth domain が operator token eligibility と RBAC を所有することを検証する。
func TestOperatorAuthDomainRejectsIneligibleOperatorAccess(t *testing.T) {
	t.Parallel()

	// Step 1: deterministic な時刻を使い、domain test が現在時刻へ依存しないようにする。
	issuedAt := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	now := issuedAt.Add(time.Minute)

	// Step 2: 拒否すべき Admin OperatorAuth 条件を scenario ID の acceptance として固定する。
	cases := []struct {
		name    string
		act     func(*testing.T) error
		wantErr error
	}{
		{
			name:    "inactive operator cannot receive refresh session",
			act:     func(t *testing.T) error { return createInactiveOperatorSession(t, issuedAt) },
			wantErr: ErrOperatorAuthInactive,
		},
		{
			name: "viewer lacks accounts create permission",
			act: func(t *testing.T) error {
				return validateOperatorAuthAccess(t, OperatorRoleViewer, operatorAuthTestSessionID, now, issuedAt)
			},
			wantErr: ErrOperatorAuthPermissionDenied,
		},
		{
			name: "operator session id mismatch is rejected",
			act: func(t *testing.T) error {
				return validateOperatorAuthAccess(t, OperatorRoleOperator, operatorAuthTestMismatchSessionID, now, issuedAt)
			},
			wantErr: ErrOperatorAuthSessionMismatch,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Step 3: 各拒否条件が期待した OperatorAuth domain error に分類されることを検証する。
			if err := tt.act(t); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func createInactiveOperatorSession(t *testing.T, issuedAt time.Time) error {
	t.Helper()

	// Step 1: inactive Operator を production と同じ constructor 経由で作成する。
	operator := newTestOperator(t, OperatorRoleAdmin, false, OperatorPasskeyRegistrationRegistered)

	// Step 2: inactive Operator に session を作ろうとし、OperatorAuth domain が拒否することを返す。
	_, err := NewOperatorAuthSession(
		operator,
		mustOperatorSessionID(t, operatorAuthTestSessionID),
		operatorAuthTestRefreshToken,
		mustTokenTTL(t),
		issuedAt,
	)
	return err
}

func validateOperatorAuthAccess(
	t *testing.T,
	role OperatorRole,
	claimsSessionID string,
	now time.Time,
	issuedAt time.Time,
) error {
	t.Helper()

	// Step 1: active かつ passkey 登録済み Operator から session と claims を組み立てる。
	operator := newTestOperator(t, role, true, OperatorPasskeyRegistrationRegistered)
	session := mustOperatorAuthSession(t, operator, operatorAuthTestSessionID, issuedAt)
	claims := mustOperatorAccessTokenClaims(t, operator, session, issuedAt)

	// Step 2: mismatch case では claims の session ID だけを別 session に差し替え、照合失敗を作る。
	claims.sessionID = mustOperatorSessionID(t, claimsSessionID)

	// Step 3: session state、claims、permission を同時に検証し、OperatorAuth domain error を返す。
	return session.ValidateAccess(operator, claims, OperatorAuthPermissionAccountsCreate, now)
}

func mustOperatorAuthSession(t *testing.T, operator Operator, sessionID string, issuedAt time.Time) OperatorAuthSession {
	t.Helper()

	// Step 1: テスト用 refresh session を constructor 経由で生成し、hash/CSRF/snapshot を domain に作らせる。
	session, err := NewOperatorAuthSession(
		operator,
		mustOperatorSessionID(t, sessionID),
		operatorAuthTestRefreshToken,
		mustTokenTTL(t),
		issuedAt,
	)
	if err != nil {
		t.Fatalf("unexpected operator auth session error: %v", err)
	}

	// Step 2: 検証済み session を呼び出し元へ返す。
	return session
}

func mustOperatorAccessTokenClaims(
	t *testing.T,
	operator Operator,
	session OperatorAuthSession,
	issuedAt time.Time,
) OperatorAccessTokenClaims {
	t.Helper()

	// Step 1: accessToken claims も constructor 経由で作り、session snapshot と operator snapshot を検証する。
	claims, err := NewOperatorAccessTokenClaims(operator, session, mustTokenJTI(t), mustTokenTTL(t), issuedAt)
	if err != nil {
		t.Fatalf("unexpected operator access token claims error: %v", err)
	}

	// Step 2: 検証済み claims を呼び出し元へ返す。
	return claims
}

func mustOperatorSessionID(t *testing.T, raw string) OperatorSessionID {
	t.Helper()

	// Step 1: OperatorSessionID の ULID 検証を production constructor と同じ経路で通す。
	id, err := NewOperatorSessionID(raw)
	if err != nil {
		t.Fatalf("unexpected operator session id error: %v", err)
	}

	// Step 2: 検証済み ID を返す。
	return id
}

func mustTokenJTI(t *testing.T) TokenJTI {
	t.Helper()

	// Step 1: accessToken jti は中立 token primitive の ULID 検証を通す。
	jti, err := NewTokenJTI(operatorAuthTestJTIULID)
	if err != nil {
		t.Fatalf("unexpected token jti error: %v", err)
	}

	// Step 2: 検証済み jti を返す。
	return jti
}

func mustTokenTTL(t *testing.T) TokenTTL {
	t.Helper()

	// Step 1: token/session lifetime は正の duration だけを使う。
	ttl, err := NewTokenTTL(15 * time.Minute)
	if err != nil {
		t.Fatalf("unexpected token ttl error: %v", err)
	}

	// Step 2: 検証済み TTL を返す。
	return ttl
}
