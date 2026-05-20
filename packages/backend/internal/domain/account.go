package domain

// Account は Product を利用する主体を表す root である。
type Account struct {
	id      AccountID
	setting AccountSetting
}

// NewAccount は Account root と必須 child の AccountSetting を同時に生成する。
func NewAccount(id AccountID, setting AccountSetting) (Account, error) {
	if err := validateAccountID(id.String()); err != nil {
		return Account{}, ErrInvalidAccountID
	}
	if setting.AccountID() != id {
		return Account{}, ErrInvalidAccountID
	}
	return Account{id: id, setting: setting}, nil
}

// ID は Account root の canonical ULID を返す。
func (a Account) ID() AccountID { return a.id }

// Setting は Account に属する保存済み AccountSetting を返す。
func (a Account) Setting() AccountSetting { return a.setting }
