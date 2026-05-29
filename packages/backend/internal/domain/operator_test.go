package domain

import (
	"errors"
	"testing"
)

const (
	operatorTestID    = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	operatorTestEmail = "admin@example.com"
)

// [ADMIN-CONSOLE-BE-S078] Operator role と active state が accounts:create permission を制御することを検証する。
func TestOperatorHasPermissionAccountsCreateMatrix(t *testing.T) {
	t.Parallel()

	// Step 1: role、active、passkey 登録状態の組み合わせを security decision table として固定する。
	cases := []struct {
		name          string
		role          OperatorRole
		active        bool
		passkeyState  OperatorPasskeyRegistrationState
		permission    string
		wantPermitted bool
	}{
		{
			name:          "active admin with registered passkey can create accounts",
			role:          OperatorRoleAdmin,
			active:        true,
			passkeyState:  OperatorPasskeyRegistrationRegistered,
			permission:    "accounts:create",
			wantPermitted: true,
		},
		{
			name:          "active operator with registered passkey can create accounts",
			role:          OperatorRoleOperator,
			active:        true,
			passkeyState:  OperatorPasskeyRegistrationRegistered,
			permission:    "accounts:create",
			wantPermitted: true,
		},
		{
			name:         "active viewer with registered passkey cannot create accounts",
			role:         OperatorRoleViewer,
			active:       true,
			passkeyState: OperatorPasskeyRegistrationRegistered,
			permission:   "accounts:create",
		},
		{
			name:         "inactive admin with registered passkey cannot create accounts",
			role:         OperatorRoleAdmin,
			active:       false,
			passkeyState: OperatorPasskeyRegistrationRegistered,
			permission:   "accounts:create",
		},
		{
			name:         "active admin with pending passkey cannot create accounts",
			role:         OperatorRoleAdmin,
			active:       true,
			passkeyState: OperatorPasskeyRegistrationPending,
			permission:   "accounts:create",
		},
		{
			name:         "active operator with registered passkey cannot use unknown permission",
			role:         OperatorRoleOperator,
			active:       true,
			passkeyState: OperatorPasskeyRegistrationRegistered,
			permission:   "accounts:delete",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Step 2: Operator domain object を必ず constructor 経由で作成し、validation を通す。
			operator := newTestOperator(t, tt.role, tt.active, tt.passkeyState)

			// Step 3: HasPermission の戻り値だけを検証し、application/handler 側の代替判定を置かない。
			if got := operator.HasPermission(tt.permission); got != tt.wantPermitted {
				t.Fatalf("HasPermission(%q) = %v, want %v", tt.permission, got, tt.wantPermitted)
			}
		})
	}
}

func TestNewOperatorValidatesOperatorSpecificInvariants(t *testing.T) {
	t.Parallel()

	t.Run("invalid id uses operator-specific error", func(t *testing.T) {
		t.Parallel()
		email, err := NewOperatorEmail(operatorTestEmail)
		if err != nil {
			t.Fatalf("unexpected email error: %v", err)
		}

		_, err = NewOperator("invalid", email, OperatorRoleAdmin, true, OperatorPasskeyRegistrationRegistered)
		if !errors.Is(err, ErrInvalidOperatorID) {
			t.Fatalf("expected ErrInvalidOperatorID, got %v", err)
		}
	})

	t.Run("email is canonical lowercase", func(t *testing.T) {
		t.Parallel()
		email, err := NewOperatorEmail("  Admin@Example.COM  ")
		if err != nil {
			t.Fatalf("unexpected email error: %v", err)
		}
		if email.String() != operatorTestEmail {
			t.Fatalf("email = %q, want %q", email.String(), operatorTestEmail)
		}
	})
}

func newTestOperator(
	t *testing.T,
	role OperatorRole,
	active bool,
	passkeyState OperatorPasskeyRegistrationState,
) Operator {
	t.Helper()

	// Step 1: テスト用 ID と email も production と同じ constructor で検証する。
	id, err := NewOperatorID(operatorTestID)
	if err != nil {
		t.Fatalf("unexpected id error: %v", err)
	}
	email, err := NewOperatorEmail(operatorTestEmail)
	if err != nil {
		t.Fatalf("unexpected email error: %v", err)
	}

	// Step 2: role/active/passkey state の組み合わせを Operator に閉じ込める。
	operator, err := NewOperator(id, email, role, active, passkeyState)
	if err != nil {
		t.Fatalf("unexpected operator error: %v", err)
	}
	return operator
}
