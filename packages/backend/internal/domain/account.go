package domain

import "time"

// Account は Product を利用する主体を表す root である。
type Account struct {
	id                  AccountID
	email               AccountEmail
	status              AccountStatus
	setting             AccountSetting
	sessionRevokedAfter *time.Time
}

// NewAccount は永続化済み Account root を domain 不変条件に沿って再構成する。
//
// id は canonical ULID、email は canonical AccountEmail、status は対応済み AccountStatus でなければならない。
// setting は同じ AccountID に属している必要があり、sessionRevokedAfter は nil または zero ではない時刻だけを受け付ける。
// 不正な入力がある場合は対応する domain error を返し、adapter から壊れた root が流入することを防ぐ。
func NewAccount(id AccountID, email AccountEmail, status AccountStatus, setting AccountSetting, sessionRevokedAfter *time.Time) (Account, error) {
	// Step 1: 引数を Account root の全保持データへ集約し、検証と複製を一箇所で行う。
	return newAccountRoot(id, email, status, setting, sessionRevokedAfter)
}

// ID は Account root の canonical ULID を返す。
func (a Account) ID() AccountID { return a.id }

// Email は Account root が所有する canonical email を返す。
//
// 戻り値は NewAccountEmail によって正規化済みであり、Admin 作成と Product 認証で同じ値を使う。
func (a Account) Email() AccountEmail { return a.email }

// Status は Account root の lifecycle 状態を返す。
//
// 戻り値は active または suspended のみであり、認証可否の判定はこの値と session revoke 境界を使う。
func (a Account) Status() AccountStatus { return a.status }

// Setting は Account に属する保存済み AccountSetting を返す。
func (a Account) Setting() AccountSetting { return a.setting }

// SessionRevokedAfter は停止や管理操作により既存 session を拒否すべき境界時刻を返す。
//
// nil は revoke 境界が未設定であることを表す。
// 非 nil の場合は内部 pointer を直接渡さず複製を返し、呼び出し側による Account 内部状態の変更を防ぐ。
func (a Account) SessionRevokedAfter() *time.Time {
	// Step 1: 境界未設定の場合は nil を返し、呼び出し側に追加判定が不要であることを伝える。
	if a.sessionRevokedAfter == nil {
		return nil
	}

	// Step 2: 設定済み時刻は defensive copy として返し、root の不変条件を外部変更から守る。
	return cloneTimePointer(*a.sessionRevokedAfter)
}

func newAccountRoot(id AccountID, email AccountEmail, status AccountStatus, setting AccountSetting, sessionRevokedAfter *time.Time) (Account, error) {
	// Step 1: session revoke 境界を複製し、外部 pointer から Account 内部状態が変わらないようにする。
	account := Account{
		id:                  id,
		email:               email,
		status:              status,
		setting:             setting,
		sessionRevokedAfter: optionalTimePointer(sessionRevokedAfter),
	}

	// Step 2: root 全体の不変条件をまとめて検証し、部分的な不正値を返さない。
	if err := validateAccountRoot(account); err != nil {
		return Account{}, err
	}

	// Step 3: 検証済み Account root を返す。
	return account, nil
}

func validateAccountRoot(account Account) error {
	// Step 1: AccountID は Product Account の主キーであるため canonical ULID として再検証する。
	if err := validateAccountID(account.id.String()); err != nil {
		return ErrInvalidAccountID
	}

	// Step 2: AccountEmail は zero value や手組み値を拒否するため constructor に戻して検証する。
	if _, err := NewAccountEmail(account.email.String()); err != nil {
		return err
	}

	// Step 3: status は対応済み lifecycle 値だけを許可し、未知状態での認証判定を防ぐ。
	if _, err := NewAccountStatus(account.status.String()); err != nil {
		return err
	}

	// Step 4: AccountSetting は同じ AccountID に属する必要があり、別 root の設定混入を拒否する。
	if account.setting.AccountID() != account.id {
		return ErrInvalidAccountID
	}

	// Step 5: revoke 境界が存在する場合、zero time は意味を持たないため拒否する。
	if account.sessionRevokedAfter != nil && account.sessionRevokedAfter.IsZero() {
		return ErrInvalidSessionRevocationBoundary
	}

	// Step 6: すべての root 不変条件を満たしたことを nil で返す。
	return nil
}
