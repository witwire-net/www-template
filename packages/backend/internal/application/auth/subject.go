package auth

import domain "www-template/packages/backend/internal/domain"

// AccountSubjectID は Product Account subject の application 境界 ID である。
//
// 役割:
//   - canonical auth lifecycle の公開 API で domain.AccountID を直接露出させず、Account subject payload の所有者 ID として表現する。
//   - 実体は domain.AccountID と同じ値であり、Product AccountAuth use case へ追加変換なしに渡せる。
type AccountSubjectID = domain.AccountID

// AccountSubjectSessionID は Product AccountAuth subject の application 境界 session ID である。
//
// 役割:
//   - Account subject payload から session selector を返す際の公開 DTO 名を提供する。
//   - 実体は domain.AccountAuthSessionID と同じ値であり、Product session metadata と refresh session validation へ安全に渡せる。
type AccountSubjectSessionID = domain.AccountAuthSessionID

// OperatorSubjectID は Admin Operator subject の application 境界 ID である。
//
// 役割:
//   - canonical auth lifecycle の公開 API で domain.OperatorID を直接露出させず、Operator subject payload の所有者 ID として表現する。
//   - 実体は domain.OperatorID と同じ値であり、Admin OperatorAuth use case へ追加変換なしに渡せる。
type OperatorSubjectID = domain.OperatorID

// OperatorSubjectSessionID は Admin OperatorAuth subject の application 境界 session ID である。
//
// 役割:
//   - Operator subject payload から session selector を返す際の公開 DTO 名を提供する。
//   - 実体は domain.OperatorSessionID と同じ値であり、Admin refresh/session store へ安全に渡せる。
type OperatorSubjectSessionID = domain.OperatorSessionID

// AccountSubjectPayload は Product hosted service adapter が canonical auth lifecycle へ渡す Account subject を表す。
//
// 役割:
//   - Product account の所有者 ID と account session ID を明示的な subject payload として束ねる。
//   - Admin operator subject と共通の discriminator field を持たず、呼び出し元の service artifact 境界で意味を決める。
//   - canonical lifecycle helper が中立 token/session primitive を扱う際に、Account eligibility へ戻るための最小情報だけを保持する。
//
// 引数:
//   - NewAccountSubjectPayload の accountID: Product Account の canonical ULID。
//   - NewAccountSubjectPayload の sessionID: Product AccountAuth session の canonical ULID 文字列。
//
// 戻り値:
//   - AccountSubjectPayload: 検証済み Account subject payload。
//   - error: accountID または sessionID が Product account/session として不正な場合の domain error。
//
// 使用例:
//
//	subject, err := auth.NewAccountSubjectPayload(accountID, sessionID)
//	if err != nil {
//		return err
//	}
type AccountSubjectPayload struct {
	accountID domain.AccountID
	sessionID domain.AccountAuthSessionID
}

// NewAccountSubjectPayload は Account subject payload を検証済み domain value から生成する。
func NewAccountSubjectPayload(accountID domain.AccountID, sessionID string) (AccountSubjectPayload, error) {
	// Step 1: AccountID を domain constructor で再検証し、operator ID や壊れた永続化値の混入を拒否する。
	validatedAccountID, err := domain.NewAccountID(accountID.String())
	if err != nil {
		return AccountSubjectPayload{}, err
	}

	// Step 2: session selector を Product AccountAuth 専用型として検証し、Admin Operator session ID と型レベルで分ける。
	validatedSessionID, err := domain.NewAccountAuthSessionID(sessionID)
	if err != nil {
		return AccountSubjectPayload{}, err
	}

	// Step 3: 検証済み owner と session だけを subject payload に保持し、credential mode や refreshToken 平文は含めない。
	return AccountSubjectPayload{accountID: validatedAccountID, sessionID: validatedSessionID}, nil
}

// AccountID は Product Account subject の canonical owner ID を返す。
func (payload AccountSubjectPayload) AccountID() AccountSubjectID {
	// Step 1: constructor で検証済みの AccountID をそのまま返す。
	return payload.accountID
}

// SessionID は Product AccountAuth subject の session selector を返す。
func (payload AccountSubjectPayload) SessionID() AccountSubjectSessionID {
	// Step 1: constructor で検証済みの Product session ID をそのまま返す。
	return payload.sessionID
}

// OperatorSubjectPayload は Admin hosted service adapter が canonical auth lifecycle へ渡す Operator subject を表す。
//
// 役割:
//   - Admin Operator の owner ID と operator session ID を明示的な subject payload として束ねる。
//   - Product Account subject と discriminator で切り替えず、Admin service adapter がこの型を選んだ事実を境界にする。
//   - RBAC や active 判定は domain.Operator / OperatorAuthSession に残し、この payload は owner/session の対応だけを表す。
//
// 引数:
//   - NewOperatorSubjectPayload の operatorID: Admin Operator の canonical ULID 文字列。
//   - NewOperatorSubjectPayload の sessionID: Admin Operator session の canonical ULID 文字列。
//
// 戻り値:
//   - OperatorSubjectPayload: 検証済み Operator subject payload。
//   - error: operatorID または sessionID が Admin operator/session として不正な場合の domain error。
//
// 使用例:
//
//	subject, err := auth.NewOperatorSubjectPayload(operatorID, sessionID)
//	if err != nil {
//		return err
//	}
type OperatorSubjectPayload struct {
	operatorID domain.OperatorID
	sessionID  domain.OperatorSessionID
}

// NewOperatorSubjectPayload は Operator subject payload を検証済み domain value から生成する。
func NewOperatorSubjectPayload(operatorID string, sessionID string) (OperatorSubjectPayload, error) {
	// Step 1: OperatorID を Admin Operator 専用 constructor で検証し、Product AccountID として扱わない。
	validatedOperatorID, err := domain.NewOperatorID(operatorID)
	if err != nil {
		return OperatorSubjectPayload{}, err
	}

	// Step 2: session selector を Admin Operator session 専用型として検証し、Account session と混同しない。
	validatedSessionID, err := domain.NewOperatorSessionID(sessionID)
	if err != nil {
		return OperatorSubjectPayload{}, err
	}

	// Step 3: refreshToken 平文や permission を含めない owner/session payload として返す。
	return OperatorSubjectPayload{operatorID: validatedOperatorID, sessionID: validatedSessionID}, nil
}

// OperatorID は Admin Operator subject の canonical owner ID を返す。
func (payload OperatorSubjectPayload) OperatorID() OperatorSubjectID {
	// Step 1: constructor で検証済みの OperatorID をそのまま返す。
	return payload.operatorID
}

// SessionID は Admin OperatorAuth subject の session selector を返す。
func (payload OperatorSubjectPayload) SessionID() OperatorSubjectSessionID {
	// Step 1: constructor で検証済みの Operator session ID をそのまま返す。
	return payload.sessionID
}
