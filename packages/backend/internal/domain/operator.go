package domain

import (
	"errors"
	"net/mail"
	"strings"
)

var (
	// ErrInvalidOperatorID は Admin Operator の canonical ULID が不正な場合に返すエラーである。
	// Product Account の識別子と混同しないよう、Operator 専用の domain error として分離する。
	ErrInvalidOperatorID = errors.New("invalid operator id")

	// ErrInvalidOperatorEmail は Admin Operator のメールアドレスが空、または正規化できない場合に返すエラーである。
	// OperatorEmail は Admin 認証・監査で使うため、作成時点で canonical な値だけを受け付ける。
	ErrInvalidOperatorEmail = errors.New("invalid operator email")

	// ErrInvalidOperatorRole は Admin Operator の role が許可された値ではない場合に返すエラーである。
	// 未知の role を fail-open に扱わないため、constructor で必ず拒否する。
	ErrInvalidOperatorRole = errors.New("invalid operator role")

	// ErrInvalidOperatorPasskeyRegistrationState は Operator の passkey 登録状態が未定義の場合に返すエラーである。
	// passkey 未登録 Operator を mutation 許可へ進めないため、状態値は明示的に検証する。
	ErrInvalidOperatorPasskeyRegistrationState = errors.New("invalid operator passkey registration state")

	// ErrOperatorLastPasskeyDeletion は Operator が最後の passkey credential を削除しようとした場合に返すエラーである。
	// Admin Console から自分自身を恒久的に締め出す操作を防ぐため、credential 数の最小値を domain 境界で保持する。
	ErrOperatorLastPasskeyDeletion = errors.New("operator last passkey deletion")
)

// OperatorID は Admin Operator を表す canonical ULID 値オブジェクトである。
//
// Product AccountID と別型にすることで、Admin 認証・監査・RBAC の識別子を
// Product account auth の識別子へ誤って渡すことを防ぐ。
// raw 値は NewOperatorID で検証し、String で永続化・監査向けの文字列に戻す。
type OperatorID string

// NewOperatorID は raw 文字列を検証し、canonical ULID の OperatorID を返す。
//
// raw は前後空白を除去した後、既存 Auth ID と同じ ULID 形式だけを受け付ける。
// 不正な場合は ErrInvalidOperatorID を返し、Admin 権限評価を fail-closed にできる。
func NewOperatorID(raw string) (OperatorID, error) {
	// Step 1: 入力の前後空白を取り除き、永続化値と比較しやすい canonical 値に寄せる。
	trimmed := strings.TrimSpace(raw)

	// Step 2: 既存の ULID 検証 primitive を使い、AccountID とは別 error に変換する。
	if err := ValidateAuthID(trimmed); err != nil {
		return "", ErrInvalidOperatorID
	}

	// Step 3: 検証済み文字列だけを OperatorID として公開する。
	return OperatorID(trimmed), nil
}

// String は OperatorID を API、DB、JWT claim、監査ログへ渡すための canonical 文字列へ変換する。
//
// 戻り値は NewOperatorID で検証済みの ULID 文字列であり、追加の副作用はない。
func (id OperatorID) String() string { return string(id) }

// OperatorEmail は Admin Operator の canonical email 値オブジェクトである。
//
// Admin operator auth は Product account auth と独立しているため、OperatorEmail は
// Product AccountEmail を再利用せず、Admin operator 固有の正規化境界を提供する。
type OperatorEmail string

// NewOperatorEmail は raw email を trim + lowercase で正規化し、OperatorEmail を返す。
//
// raw が空、表示名付き email、または mail.ParseAddress で解釈できない値の場合は
// ErrInvalidOperatorEmail を返す。戻り値は Admin operator の login・監査表示で使う。
func NewOperatorEmail(raw string) (OperatorEmail, error) {
	// Step 1: 入力の周辺空白を削り、大文字小文字差による重複を防ぐため lowercase にする。
	canonical := strings.ToLower(strings.TrimSpace(raw))
	if canonical == "" {
		return "", ErrInvalidOperatorEmail
	}

	// Step 2: Go 標準 parser で構文を検証し、表示名付き形式は canonical 値と一致しないため拒否する。
	address, err := mail.ParseAddress(canonical)
	if err != nil || address.Address != canonical {
		return "", ErrInvalidOperatorEmail
	}

	// Step 3: mail.ParseAddress が許す空白混入を追加で拒否し、DB unique key に安定した値だけを渡す。
	if strings.ContainsAny(canonical, " \t\n\r") {
		return "", ErrInvalidOperatorEmail
	}

	return OperatorEmail(canonical), nil
}

// String は OperatorEmail を永続化、監査表示、通知先参照で使う canonical 文字列へ変換する。
//
// 戻り値は lowercase 済みの email であり、この accessor は状態を変更しない。
func (e OperatorEmail) String() string { return string(e) }

// OperatorRole は Admin Operator の RBAC role を表す値オブジェクトである。
//
// Admin Console の account 作成権限は role と active/passkey state の積で決まり、
// Product account auth の status や scope とは独立して評価される。
type OperatorRole string

const (
	// OperatorRoleAdmin は Admin operator 管理を含む最上位 role である。
	OperatorRoleAdmin OperatorRole = "admin"

	// OperatorRoleOperator は顧客アカウント運用 mutation を実行できる通常運用 role である。
	OperatorRoleOperator OperatorRole = "operator"

	// OperatorRoleViewer は読み取り専用 role であり、accounts:create を持たない。
	OperatorRoleViewer OperatorRole = "viewer"
)

// Validate は OperatorRole が既知の role であることを検証する。
//
// 未知の role は将来追加予定に見えても fail-open を避けるため拒否し、
// ErrInvalidOperatorRole を返す。副作用はない。
func (r OperatorRole) Validate() error {
	// Step 1: 明示的に許可した role だけを受け付ける。
	switch r {
	case OperatorRoleAdmin, OperatorRoleOperator, OperatorRoleViewer:
		return nil
	default:
		return ErrInvalidOperatorRole
	}
}

// OperatorPasskeyRegistrationState は Operator の passkey 登録状態を表す値オブジェクトである。
//
// Admin mutation は passkey 登録済み Operator だけに許可される。
// 初回 setup 中の Operator を active role だけで許可しないため、active とは別軸で保持する。
type OperatorPasskeyRegistrationState string

const (
	// OperatorPasskeyRegistrationPending は Operator がまだ passkey を登録していない状態である。
	OperatorPasskeyRegistrationPending OperatorPasskeyRegistrationState = "pending"

	// OperatorPasskeyRegistrationRegistered は Operator が passkey 登録を完了した状態である。
	OperatorPasskeyRegistrationRegistered OperatorPasskeyRegistrationState = "registered"
)

// Validate は OperatorPasskeyRegistrationState が既知の状態であることを検証する。
//
// 未知の状態は Admin mutation の許可判定を曖昧にするため、
// ErrInvalidOperatorPasskeyRegistrationState を返して fail-closed にする。
func (s OperatorPasskeyRegistrationState) Validate() error {
	// Step 1: 明示的に許可した passkey 登録状態だけを受け付ける。
	switch s {
	case OperatorPasskeyRegistrationPending, OperatorPasskeyRegistrationRegistered:
		return nil
	default:
		return ErrInvalidOperatorPasskeyRegistrationState
	}
}

// Operator は Admin Console を操作する運営者を表す domain object である。
//
// Operator は ID、email、role、active state、passkey registration state を保持し、
// Admin account creation などの permission decision を Product AccountAuth と独立して行う。
type Operator struct {
	id                       OperatorID
	email                    OperatorEmail
	role                     OperatorRole
	active                   bool
	passkeyRegistrationState OperatorPasskeyRegistrationState
}

// NewOperator は Admin Operator の不変条件を検証して Operator を生成する。
//
// id/email はそれぞれ NewOperatorID/NewOperatorEmail 済みの値を受け取り、role と
// passkeyRegistrationState を検証する。active は DB から復元した有効状態をそのまま保持する。
func NewOperator(
	id OperatorID,
	email OperatorEmail,
	role OperatorRole,
	active bool,
	passkeyRegistrationState OperatorPasskeyRegistrationState,
) (Operator, error) {
	// Step 1: ID を再検証し、ゼロ値や AccountID 由来の未検証文字列を拒否する。
	validatedID, err := NewOperatorID(id.String())
	if err != nil {
		return Operator{}, err
	}

	// Step 2: email を再検証し、constructor 経由で canonical lowercase にそろえる。
	validatedEmail, err := NewOperatorEmail(email.String())
	if err != nil {
		return Operator{}, err
	}

	// Step 3: role を fail-closed に検証し、未知 role の permission 付与を防ぐ。
	if err := role.Validate(); err != nil {
		return Operator{}, err
	}

	// Step 4: passkey 登録状態を検証し、setup 中 Operator の権限混入を防ぐ。
	if err := passkeyRegistrationState.Validate(); err != nil {
		return Operator{}, err
	}

	return Operator{
		id:                       validatedID,
		email:                    validatedEmail,
		role:                     role,
		active:                   active,
		passkeyRegistrationState: passkeyRegistrationState,
	}, nil
}

// ID は Operator の canonical ULID を返す。
//
// 戻り値は Admin operator auth、audit correlation、repository key として使える。
func (o Operator) ID() OperatorID { return o.id }

// Email は Operator の canonical email を返す。
//
// 戻り値は lowercase 済みで、通知先や監査表示に使用できる。
func (o Operator) Email() OperatorEmail { return o.email }

// Role は Operator の Admin RBAC role を返す。
//
// 戻り値は HasPermission の内部判定と同じ role 値であり、副作用はない。
func (o Operator) Role() OperatorRole { return o.role }

// Active は Operator が Admin mutation を実行できる有効状態かどうかを返す。
//
// false の場合、role や passkey 状態に関係なく HasPermission は mutation 権限を拒否する。
func (o Operator) Active() bool { return o.active }

// PasskeyRegistrationState は Operator の passkey 登録状態を返す。
//
// pending の場合、初回 setup 中として扱い、Admin mutation 権限は付与しない。
func (o Operator) PasskeyRegistrationState() OperatorPasskeyRegistrationState {
	return o.passkeyRegistrationState
}

// HasPermission は Operator が指定 permission を持つかどうかを返す。
//
// 現時点では accounts:create、operators:create、operators:logout、operator-passkeys:manage を domain invariant として扱う。
// active で passkey 登録済み、かつ admin/operator role の場合だけ true を返し、
// operator-passkeys:manage と operators:logout は自分の認証手段・session 維持なので viewer にも許可する。
// inactive、passkey 未登録、未知 permission はすべて false にする。
func (o Operator) HasPermission(permission string) bool {
	// Step 1: 無効化済み Operator は role に関係なく Admin mutation を拒否する。
	if !o.active {
		return false
	}

	// Step 2: passkey 未登録 Operator は setup 中なので mutation 権限を付与しない。
	if o.passkeyRegistrationState != OperatorPasskeyRegistrationRegistered {
		return false
	}

	// Step 3: 自分自身の passkey 管理と logout は全 role で必要な認証手段/session 維持操作として許可する。
	if permission == OperatorAuthPermissionPasskeysManage.String() || permission == OperatorAuthPermissionOperatorsLogout.String() {
		return true
	}

	// Step 4: operator 作成は権限拡張操作なので admin のみに許可する。
	if permission == OperatorAuthPermissionOperatorsCreate.String() {
		return o.role == OperatorRoleAdmin
	}

	// Step 5: account 作成は admin と operator のみ許可し、viewer は拒否する。
	if permission != OperatorAuthPermissionAccountsCreate.String() {
		return false
	}
	switch o.role {
	case OperatorRoleAdmin, OperatorRoleOperator:
		return true
	default:
		return false
	}
}

// EnsureOperatorPasskeyDeletionAllowed は Operator passkey credential を削除してよい件数かを検証する。
//
// 引数:
//   - credentialCount: 削除前に Operator が保持している passkey credential 数。
//
// 戻り値:
//   - nil: 削除後も 1 件以上の credential が残る場合。
//   - ErrOperatorLastPasskeyDeletion: credentialCount が 1 以下で、削除すると最後の passkey が失われる場合。
//
// 使用例:
//
//	if err := domain.EnsureOperatorPasskeyDeletionAllowed(len(passkeys)); err != nil {
//		return err
//	}
func EnsureOperatorPasskeyDeletionAllowed(credentialCount int) error {
	// Step 1: 0 件または 1 件の状態では削除を許すと Operator が再ログイン不能になるため拒否する。
	if credentialCount <= 1 {
		return ErrOperatorLastPasskeyDeletion
	}

	// Step 2: 削除後も少なくとも 1 件残るため、passkey 削除を許可する。
	return nil
}
