package admin

import (
	"context"
	"errors"

	adminauth "www-template/packages/backend/internal/application/admin/auth"
)

// AdminOperatorSessionAuthenticator は Admin HTTP session validator が利用する application auth 境界である。
//
// 役割:
//   - CurrentOperator は read route の bearer/session/snapshot 検証を担う。
//   - ValidateOperatorMutation は mutation route の bearer/session/CSRF/RBAC 検証を担う。
//   - HTTP adapter は repository/store/signer を生成せず、runtime composition 済み application service だけへ依存する。
type AdminOperatorSessionAuthenticator interface {
	CurrentOperator(ctx context.Context, input adminauth.CurrentOperatorInput) (adminauth.OperatorDTO, error)
	ValidateOperatorMutation(ctx context.Context, input adminauth.ValidateOperatorMutationInput) (adminauth.OperatorDTO, error)
}

type applicationOperatorSessionValidator struct {
	auth AdminOperatorSessionAuthenticator
}

// NewOperatorSessionValidator は Admin auth application service を HTTP middleware 用 validator へ適合させる。
//
// 引数:
//   - auth: runtime composition 済み Admin operator auth service。nil の場合は nil を返し、router 側の fail-close を維持する。
//
// 戻り値:
//   - OperatorSessionValidator: protected route の accessToken/CSRF/session 検証に使う adapter。
//
// エラーケース:
//   - この関数自体は error を返さない。auth が nil の場合、呼び出し側が nil validator として 503 fail-close させる。
//
// 使用例:
//
//	validator := admin.NewOperatorSessionValidator(authService)
func NewOperatorSessionValidator(auth AdminOperatorSessionAuthenticator) OperatorSessionValidator {
	// Step 1: auth service が未構成のまま fake validator を作らず、既存 middleware の nil fail-close 経路へ委譲する。
	if auth == nil {
		return nil
	}

	// Step 2: application service を保持する薄い adapter を返し、HTTP layer に repository/store/signer の知識を持ち込まない。
	return applicationOperatorSessionValidator{auth: auth}
}

// ValidateOperatorSession は protected route の read/mutation 差分に応じて Admin auth application service へ検証を委譲する。
func (v applicationOperatorSessionValidator) ValidateOperatorSession(ctx context.Context, input OperatorSessionValidationInput) (OperatorSessionContext, error) {
	// Step 1: validator 自体がゼロ値で使われた場合も fail-closed の internal error にする。
	if v.auth == nil {
		return OperatorSessionContext{}, errAdminOperatorInternal
	}

	// Step 2: mutation route は permission が明示されたものだけを通し、CSRF を検証しない mutation を拒否する。
	if input.RequireCSRF {
		return v.validateMutation(ctx, input)
	}

	// Step 3: read route は CurrentOperator に委譲し、署名/session/snapshot の検証だけで handler context を作る。
	operator, err := v.auth.CurrentOperator(ctx, adminauth.CurrentOperatorInput{AccessToken: input.AccessToken})
	if err != nil {
		return OperatorSessionContext{}, mapAdminOperatorSessionValidationError(err)
	}
	return operatorSessionContextFromDTO(operator, ""), nil
}

func (v applicationOperatorSessionValidator) validateMutation(ctx context.Context, input OperatorSessionValidationInput) (OperatorSessionContext, error) {
	// Step 1: permission 未割り当ての CSRF route は、CSRF 検証なしで通すより安全側に倒して forbidden とする。
	if input.Permission == "" {
		return OperatorSessionContext{}, errAdminOperatorForbidden
	}

	// Step 2: Admin auth service に bearer/session/CSRF/RBAC の同時検証を委譲し、HTTP adapter で token や role を解釈しない。
	operator, err := v.auth.ValidateOperatorMutation(ctx, adminauth.ValidateOperatorMutationInput{AccessToken: input.AccessToken, CSRFToken: input.CSRFToken, Permission: input.Permission})
	if err != nil {
		return OperatorSessionContext{}, mapAdminOperatorSessionValidationError(err)
	}

	// Step 3: 検証済み CSRF header だけを handler context へ残し、未検証値を application account use case に渡さない。
	return operatorSessionContextFromDTO(operator, input.CSRFToken), nil
}

func operatorSessionContextFromDTO(operator adminauth.OperatorDTO, csrfToken string) OperatorSessionContext {
	// Step 1: application DTO の Admin Operator primitive を middleware context DTO へ写像し、Product account 情報を混入させない。
	return OperatorSessionContext{
		OperatorID:                       operator.ID,
		OperatorEmail:                    operator.Email,
		OperatorRole:                     operator.Role,
		OperatorActive:                   operator.Active,
		OperatorPasskeyRegistrationState: operator.PasskeyRegistrationState,
		SessionID:                        operator.SessionID,
		CSRFToken:                        csrfToken,
	}
}

func mapAdminOperatorSessionValidationError(err error) error {
	// Step 1: application auth の内部エラーは middleware で 503 へ写像できる stable error にする。
	if errors.Is(err, adminauth.ErrAdminAuthInternal) {
		return errAdminOperatorInternal
	}

	// Step 2: permission/CSRF/inactive 系の拒否は middleware で 403 へ写像できる stable error にする。
	if errors.Is(err, adminauth.ErrAdminAuthForbidden) {
		return errAdminOperatorForbidden
	}

	// Step 3: それ以外の認証失敗や不正 token は既定 401 経路へ流し、詳細を response へ出さない。
	return err
}
