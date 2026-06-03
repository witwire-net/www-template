package auth

import (
	"context"

	domain "www-template/packages/backend/internal/domain"
)

// OperatorCredentialService は Operator passkey credential の一覧取得と削除を担当する application service である。
//
// 役割:
//   - Operator session の login / refresh / current / logout から passkey 管理責務を分離する。
//   - repository port を constructor injection で受け取り、後付け mutation による未構成状態を作らない。
//   - 最後の passkey 削除保護を domain.EnsureOperatorPasskeyDeletionAllowed へ委譲する。
//
// 使用例:
//
//	service, err := auth.NewOperatorCredentialService(passkeyRepository)
//	if err != nil {
//		return err
//	}
type OperatorCredentialService struct {
	passkeys OperatorPasskeyRepository
}

// NewOperatorCredentialService は Operator passkey 管理 service を生成する。
//
// 引数:
//   - passkeys: operator_passkeys だけを扱う repository port。nil は拒否する。
//
// 戻り値:
//   - *OperatorCredentialService: 一覧取得と削除を提供する service。
//   - error: passkeys が nil の場合は ErrOperatorAuthUnavailable。
func NewOperatorCredentialService(passkeys OperatorPasskeyRepository) (*OperatorCredentialService, error) {
	// Step 1: nil repository を拒否し、passkey 管理 route が保存層なしで成功しないよう fail-closed にする。
	if passkeys == nil {
		return nil, ErrOperatorAuthUnavailable
	}

	// Step 2: constructor injection された repository だけを保持し、後から service 状態を書き換える経路を作らない。
	return &OperatorCredentialService{passkeys: passkeys}, nil
}

// ListOperatorPasskeys は Operator 自身に登録された passkey credential 一覧を返す。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: middleware が検証済み session context から渡す OperatorID。
//
// 戻り値:
//   - OperatorPasskeyListResult: credential handle や公開鍵を含まない一覧 DTO。
//   - error: OperatorID 不正、保存層障害、credential 不在などの stable application error。
func (s *OperatorCredentialService) ListOperatorPasskeys(ctx context.Context, input ListOperatorPasskeysInput) (OperatorPasskeyListResult, error) {
	// Step 1: OperatorID を domain value object として検証し、Product AccountID などの未検証文字列を拒否する。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return OperatorPasskeyListResult{}, ErrOperatorAuthUnauthenticated
	}

	// Step 2: Operator passkey repository へ所有者 ID を渡し、operator schema の credential だけを取得する。
	passkeys, err := s.passkeys.ListOperatorPasskeys(ctx, operatorID.String())
	if err != nil {
		return OperatorPasskeyListResult{}, mapOperatorPasskeyStoreError(err)
	}

	// Step 3: 非秘匿 DTO の一覧だけを返し、handler が credential handle や公開鍵を扱わない境界を保つ。
	return OperatorPasskeyListResult{Passkeys: passkeys}, nil
}

// DeleteOperatorPasskey は Operator 自身の passkey credential を削除する。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: 検証済み OperatorID と path parameter 由来の削除対象 passkey ID。
//
// 戻り値:
//   - nil: 削除が完了した場合。
//   - error: 最後の passkey 削除、所有者不一致、保存層障害などの stable application error。
func (s *OperatorCredentialService) DeleteOperatorPasskey(ctx context.Context, input DeleteOperatorPasskeyInput) error {
	// Step 1: 所有者 OperatorID と削除対象 passkey ID を domain ULID rule で検証し、保存層へ不正 selector を渡さない。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return ErrOperatorAuthUnauthenticated
	}
	if err := domain.ValidateAuthID(input.PasskeyID); err != nil {
		return ErrOperatorAuthInvalidInput
	}

	// Step 2: 削除前の credential 数を repository から取得し、最後の 1 件削除 rule を domain に委譲する。
	passkeys, err := s.passkeys.ListOperatorPasskeys(ctx, operatorID.String())
	if err != nil {
		return mapOperatorPasskeyStoreError(err)
	}
	if err := domain.EnsureOperatorPasskeyDeletionAllowed(len(passkeys)); err != nil {
		return mapOperatorPasskeyDomainError(err)
	}

	// Step 3: repository に所有者 ID と passkey ID の両方を渡し、他 Operator の credential 削除を防ぐ。
	if err := s.passkeys.DeleteOperatorPasskey(ctx, operatorID.String(), input.PasskeyID); err != nil {
		return mapOperatorPasskeyStoreError(err)
	}
	return nil
}
