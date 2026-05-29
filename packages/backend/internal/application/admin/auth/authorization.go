package application

import (
	"context"

	domain "www-template/packages/backend/internal/domain"
)

const accountsCreatePermission = string(domain.OperatorAuthPermissionAccountsCreate)

// AuthorizationService は Admin RBAC 判定だけを担当する application use case である。
//
// 役割:
//   - HTTP handler から受け取った operator context を domain.Operator に復元し、permission 判定を domain invariant へ委譲する。
//   - `accounts:create` の permission 名を handler へ露出させず、Admin account 作成の認可条件を application layer に集約する。
//   - Product AccountAuth、Product role、generated OpenAPI 型、Gin 型を import せず、Admin authorization 境界を分離する。
//
// 使用例:
//
//	authorizer := NewAuthorizationService()
//	decision, err := authorizer.AuthorizeAccountCreation(ctx, input)
//	if err != nil {
//		return err
//	}
//	_ = decision
type AuthorizationService struct{}

// NewAuthorizationService は Admin RBAC authorization use case を生成する。
//
// 引数は不要であり、現在時刻、DB、Token verifier などの副作用源を保持しない。
// 戻り値は stateless な AuthorizationService で、複数 handler から共有しても内部状態の副作用を持たない。
func NewAuthorizationService() AuthorizationService {
	// Step 1: RBAC 判定は入力 DTO と domain.Operator だけで完結するため、依存を保持しない値を返す。
	return AuthorizationService{}
}

// AuthorizeAccountCreation は Admin account 作成に必要な `accounts:create` 権限を検証する。
//
// ctx は将来の監査 correlation や deadline 伝播に使えるよう受け取るが、この use case 自体は I/O を行わない。
// input は auth middleware が検証済み session から作った operator context であり、Product account 情報を含めてはならない。
// 戻り値は許可済み operator と permission を示す AuthorizationDecision である。
// Operator 復元に失敗した場合、または `accounts:create` を持たない場合は ErrAdminAuthForbidden を返す。
func (s AuthorizationService) AuthorizeAccountCreation(ctx context.Context, input OperatorAuthorizationInput) (AuthorizationDecision, error) {
	// Step 1: context が cancel 済みなら、後続の handler が mutation を進めないよう内部エラーではなく抽象認可失敗として停止する。
	if err := ctx.Err(); err != nil {
		return AuthorizationDecision{}, ErrAdminAuthInternal
	}

	// Step 2: primitive ID を OperatorID value object に復元し、Product AccountID など未検証文字列の混入を拒否する。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return AuthorizationDecision{}, ErrAdminAuthForbidden
	}

	// Step 3: operator email を domain value object へ復元し、canonical lowercase と形式検証を domain に委譲する。
	operatorEmail, err := domain.NewOperatorEmail(input.OperatorEmail)
	if err != nil {
		return AuthorizationDecision{}, ErrAdminAuthForbidden
	}

	// Step 4: role と passkey 登録状態は domain enum 型に変換し、未知値拒否は NewOperator に委譲する。
	operator, err := domain.NewOperator(
		operatorID,
		operatorEmail,
		domain.OperatorRole(input.OperatorRole),
		input.OperatorActive,
		domain.OperatorPasskeyRegistrationState(input.PasskeyRegistrationState),
	)
	if err != nil {
		return AuthorizationDecision{}, ErrAdminAuthForbidden
	}

	// Step 5: accounts:create の permission 名は application use case 内で固定し、handler が permission map を持たない構成にする。
	if !operator.HasPermission(accountsCreatePermission) {
		return AuthorizationDecision{}, ErrAdminAuthForbidden
	}

	// Step 6: 許可済み decision だけを返し、後続の audit/account creation use case が operator と permission を再利用できる形にする。
	return AuthorizationDecision{OperatorID: operator.ID().String(), Permission: accountsCreatePermission, Allowed: true}, nil
}

// AuthorizeAccountCreation は既存 Admin auth Service から account 作成 RBAC use case を呼び出す互換的な facade である。
//
// ctx は token/session 検証や repository 呼び出しの deadline 伝播に使う。
// input は bearer accessToken と CSRF token を含み、permission はこの method が accounts:create に固定する。
// 戻り値は mutation を実行できる OperatorDTO であり、権限不足は ErrAdminAuthForbidden として返す。
func (s *Service) AuthorizeAccountCreation(ctx context.Context, input AuthorizeAccountCreationInput) (OperatorDTO, error) {
	// Step 1: handler から任意 permission を受け取らず、accounts:create だけに固定した application decision として既存 mutation validator を使う。
	operator, err := s.ValidateOperatorMutation(ctx, ValidateOperatorMutationInput{
		AccessToken: input.AccessToken,
		CSRFToken:   input.CSRFToken,
		Permission:  accountsCreatePermission,
	})
	if err != nil {
		return OperatorDTO{}, err
	}

	// Step 2: 既存 auth Service を使う経路では session/token 検証済み OperatorDTO をそのまま返す。
	return operator, nil
}
