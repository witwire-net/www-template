package application

import (
	"context"
	"errors"
	"testing"

	domain "www-template/packages/backend/internal/domain"
)

// [ADMIN-CONSOLE-BE-S068] admin と operator は accounts:create を持つ。
// [ADMIN-CONSOLE-BE-S069] viewer は accounts:create を持たない。
func TestAdminAuthorizationUseCaseControlsAccountCreationPermission(t *testing.T) {
	t.Parallel()

	// Step 1: account creation RBAC 専用 use case を作り、DB や token verifier なしで permission decision だけを検証する。
	authorizer := NewAuthorizationService()
	ctx := context.Background()

	tests := []struct {
		name      string
		role      string
		wantError bool
	}{
		{name: "admin role can create accounts", role: string(domain.OperatorRoleAdmin)},
		{name: "operator role can create accounts", role: string(domain.OperatorRoleOperator)},
		{name: "viewer role cannot create accounts", role: string(domain.OperatorRoleViewer), wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Step 2: handler から渡される operator context 相当の DTO を作り、permission 名は test/handler 側から渡さない。
			decision, err := authorizer.AuthorizeAccountCreation(ctx, OperatorAuthorizationInput{
				OperatorID:               "01ARZ3NDEKTSV4RRFFQ69G5FAW",
				OperatorEmail:            "admin@example.com",
				OperatorRole:             tt.role,
				OperatorActive:           true,
				PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered),
			})

			// Step 3: viewer だけを forbidden にし、admin/operator は accounts:create decision を返すことを確認する。
			if tt.wantError {
				if !errors.Is(err, ErrAdminAuthForbidden) {
					t.Fatalf("expected forbidden for role %q, got decision=%+v err=%v", tt.role, decision, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("authorize account creation for role %q: %v", tt.role, err)
			}
			if !decision.Allowed || decision.Permission != accountsCreatePermission || decision.OperatorID == "" {
				t.Fatalf("expected allowed accounts:create decision, got %+v", decision)
			}
		})
	}
}

func TestAdminAuthorizationUseCaseRejectsInactiveOrUnregisteredOperator(t *testing.T) {
	t.Parallel()

	// Step 1: role だけではなく active state と passkey registration state も domain.Operator 経由で評価されることを確認する。
	authorizer := NewAuthorizationService()
	tests := []struct {
		name                     string
		active                   bool
		passkeyRegistrationState string
	}{
		{name: "inactive admin is forbidden", active: false, passkeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered)},
		{name: "unregistered admin is forbidden", active: true, passkeyRegistrationState: string(domain.OperatorPasskeyRegistrationPending)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Step 2: 不許可状態だけを変えた operator context を渡し、handler ではなく application use case が拒否することを固定する。
			_, err := authorizer.AuthorizeAccountCreation(context.Background(), OperatorAuthorizationInput{
				OperatorID:               "01ARZ3NDEKTSV4RRFFQ69G5FAW",
				OperatorEmail:            "admin@example.com",
				OperatorRole:             string(domain.OperatorRoleAdmin),
				OperatorActive:           tt.active,
				PasskeyRegistrationState: tt.passkeyRegistrationState,
			})
			if !errors.Is(err, ErrAdminAuthForbidden) {
				t.Fatalf("expected forbidden for %s, got %v", tt.name, err)
			}
		})
	}
}
