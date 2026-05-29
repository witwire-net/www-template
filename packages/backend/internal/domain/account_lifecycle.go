package domain

import (
	"errors"
	"time"
)

var (
	// ErrInvalidAccountStatus は Product Account の lifecycle status が対応値ではない場合に返すエラーである。
	ErrInvalidAccountStatus = errors.New("invalid account status")
	// ErrInvalidSessionRevocationBoundary は session revoke 境界時刻が lifecycle 操作に使えない場合に返すエラーである。
	ErrInvalidSessionRevocationBoundary = errors.New("invalid session revocation boundary")
)

// AccountStatus は Product Account の認証可否に直結する lifecycle 状態である。
//
// active は通常利用可能な状態を表し、suspended は管理操作により既存 token と新規認証を拒否する状態を表す。
// 文字列を直接扱うと application/adapter へ status 判定が漏れるため、domain package が対応値を所有する。
//
// 使用例:
//
//	status, err := NewAccountStatus("active")
//	if err != nil {
//		return err
//	}
//	_ = status.IsSuspended()
type AccountStatus string

const (
	// AccountStatusActive は Account が Product surface を利用できる状態である。
	AccountStatusActive AccountStatus = "active"
	// AccountStatusSuspended は Account が管理操作により利用停止され、token を拒否すべき状態である。
	AccountStatusSuspended AccountStatus = "suspended"
)

// NewAccountStatus は保存済み文字列を AccountStatus として検証して返す。
//
// value は active または suspended のみを受け付ける。
// 未対応値は ErrInvalidAccountStatus を返し、呼び出し元が fail-closed に扱えるようにする。
func NewAccountStatus(value string) (AccountStatus, error) {
	// Step 1: 永続化済みの status 文字列を対応済み lifecycle 値へ写像する。
	switch AccountStatus(value) {
	case AccountStatusActive:
		return AccountStatusActive, nil
	case AccountStatusSuspended:
		return AccountStatusSuspended, nil
	default:
		// Step 2: 未知 status は認証可否を安全に判断できないため domain error で拒否する。
		return "", ErrInvalidAccountStatus
	}
}

// String は AccountStatus を API、DB、監査ログに保存する canonical 文字列へ変換する。
//
// 戻り値は NewAccountStatus または domain constructor を通じて検証された値である。
func (s AccountStatus) String() string {
	return string(s)
}

// IsSuspended は Product Account が停止中で token と新規認証を拒否すべきかを返す。
//
// 戻り値が true の場合、application 層は提示された session/token を受け入れてはならない。
func (s AccountStatus) IsSuspended() bool {
	return s == AccountStatusSuspended
}

// NewAdminCreatedAccount は Admin Console の account 作成操作から初期 Account root を生成する。
//
// id は canonical ULID、email は正規化済み AccountEmail として検証される。
// 作成直後の Account は active、DefaultAccountSetting、session revoke 境界なしで開始する。
// invalid な id/email/setting が検出された場合は対応する domain error を返す。
func NewAdminCreatedAccount(id AccountID, email AccountEmail) (Account, error) {
	// Step 1: Admin 作成でも Product AccountID の canonical 規則を再検証し、境界外入力を拒否する。
	if err := validateAccountID(id.String()); err != nil {
		return Account{}, ErrInvalidAccountID
	}

	// Step 2: AccountEmail 値を再 constructor に通し、zero value や手組み値を domain 境界で拒否する。
	canonicalEmail, err := NewAccountEmail(email.String())
	if err != nil {
		return Account{}, err
	}

	// Step 3: 新規 Account には既定 AccountSetting を同時に作り、root と child の不整合を防ぐ。
	setting, err := NewDefaultAccountSetting(id)
	if err != nil {
		return Account{}, err
	}

	// Step 4: lifecycle 初期値を active と revoke 境界なしに固定し、Admin 作成時の認証初期状態を統一する。
	return newAccountRoot(id, canonicalEmail, AccountStatusActive, setting, nil)
}

// Suspend は Account を suspended にし、指定時刻以前に発行された session/token を拒否する境界を設定する。
//
// at は外側の clock から渡された UTC 推奨の時刻であり、domain 層では time.Now を読まない。
// zero time は安全な revoke 境界にならないため ErrInvalidSessionRevocationBoundary を返す。
// 戻り値は停止状態を反映した新しい Account 値であり、元の値は変更されない。
func (a Account) Suspend(at time.Time) (Account, error) {
	// Step 1: revoke 境界に zero time を許すと古い session 判定が曖昧になるため拒否する。
	if at.IsZero() {
		return Account{}, ErrInvalidSessionRevocationBoundary
	}

	// Step 2: 値オブジェクトとして Account を複製し、停止状態と境界時刻を同時に反映する。
	a.status = AccountStatusSuspended
	a.sessionRevokedAfter = cloneTimePointer(at.UTC())

	// Step 3: 変更後の root を再検証し、部分的に壊れた Account を返さない。
	if err := validateAccountRoot(a); err != nil {
		return Account{}, err
	}

	// Step 4: 検証済みの停止状態 Account を返す。
	return a, nil
}

// Restore は Account を active に戻す。
//
// Restore は停止時に設定された session revoke 境界を消さない。
// これにより停止前に発行された token/session が復元後に再利用されることを防ぐ。
// 戻り値は復元状態を反映した新しい Account 値であり、元の値は変更されない。
func (a Account) Restore() (Account, error) {
	// Step 1: status だけを active に戻し、過去 session の revoke 境界は維持する。
	a.status = AccountStatusActive

	// Step 2: 変更後の root を再検証し、復元操作でも不正な Account を返さない。
	if err := validateAccountRoot(a); err != nil {
		return Account{}, err
	}

	// Step 3: 検証済みの復元状態 Account を返す。
	return a, nil
}

// RejectsTokenIssuedAt は Account lifecycle が指定発行時刻の token/session を拒否すべきかを返す。
//
// suspended 状態では発行時刻にかかわらず true を返す。
// active 状態でも sessionRevokedAfter が存在し、issuedAt が境界以前の場合は true を返す。
// それ以外は false を返し、application 層が token を継続検証できることを示す。
func (a Account) RejectsTokenIssuedAt(issuedAt time.Time) bool {
	// Step 1: 停止中 Account はすべての Product token/session を拒否する。
	if a.status.IsSuspended() {
		return true
	}

	// Step 2: revoke 境界が存在しない active Account は lifecycle 理由では token を拒否しない。
	if a.sessionRevokedAfter == nil {
		return false
	}

	// Step 3: 境界時刻以前に発行された token/session を拒否し、停止前 session の復活を防ぐ。
	return !issuedAt.UTC().After(a.sessionRevokedAfter.UTC())
}

func cloneTimePointer(value time.Time) *time.Time {
	// Step 1: time.Time の address を新しい変数から返し、呼び出し側の変数共有を避ける。
	cloned := value

	// Step 2: Account 内部状態として保持できる pointer を返す。
	return &cloned
}

func optionalTimePointer(value *time.Time) *time.Time {
	// Step 1: nil は revoke 境界なしを表すため、そのまま保持する。
	if value == nil {
		return nil
	}

	// Step 2: 非 nil の時刻は UTC 化して複製し、外部 pointer からの後続変更を遮断する。
	return cloneTimePointer(value.UTC())
}
