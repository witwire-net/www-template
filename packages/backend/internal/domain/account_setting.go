package domain

// AccountSetting は Product Account に属する表示・通知設定である。
type AccountSetting struct {
	accountID AccountID
	locale    AccountLocale
}

// NewAccountSetting は AccountID と locale から AccountSetting を生成する。
func NewAccountSetting(accountID AccountID, locale AccountLocale) (AccountSetting, error) {
	if err := validateAccountID(accountID.String()); err != nil {
		return AccountSetting{}, ErrInvalidAccountID
	}
	if _, err := NewAccountLocale(locale.String()); err != nil {
		return AccountSetting{}, err
	}
	return AccountSetting{accountID: accountID, locale: locale}, nil
}

// NewDefaultAccountSetting は Account 作成時の既定 AccountSetting を生成する。
func NewDefaultAccountSetting(accountID AccountID) (AccountSetting, error) {
	return NewAccountSetting(accountID, DefaultAccountLocale())
}

// AccountID は AccountSetting を所有する Product Account の canonical ULID を返す。
func (s AccountSetting) AccountID() AccountID { return s.accountID }

// Locale は保存済み AccountSetting.locale を返す。
func (s AccountSetting) Locale() AccountLocale { return s.locale }

// Snapshot は refresh response などの composition に使う読み取り専用 snapshot を返す。
func (s AccountSetting) Snapshot() AccountSettingSnapshot {
	return AccountSettingSnapshot{locale: s.locale}
}

// AccountSettingSnapshot は AccountSetting の transport 合成用読み取りモデルである。
type AccountSettingSnapshot struct {
	locale AccountLocale
}

// Locale は snapshot 生成時点の AccountSetting.locale を返す。
func (s AccountSettingSnapshot) Locale() AccountLocale { return s.locale }
