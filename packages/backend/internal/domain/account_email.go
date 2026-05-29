package domain

import (
	"errors"
	"strings"
	"unicode"
)

var (
	// ErrInvalidAccountEmail は Product Account のメールアドレスが正規化後も不正な場合に返すエラーである。
	ErrInvalidAccountEmail = errors.New("invalid account email")
)

const (
	accountEmailMaxLength      = 254
	accountEmailLocalMaxLength = 64
)

// AccountEmail は Product Account の canonical email を表す値オブジェクトである。
//
// この型は Admin Console から作成される顧客アカウントと Product 側の認証 projection が
// 同じ email 正規化規則を使うための source of truth である。
// 値は前後空白を除去し、ASCII 大文字小文字の差分による重複を避けるため lowercase に正規化される。
//
// 使用例:
//
//	email, err := NewAccountEmail(" USER@example.COM ")
//	if err != nil {
//		return err
//	}
//	_ = email.String() // "user@example.com"
type AccountEmail string

// NewAccountEmail は入力文字列を canonical AccountEmail に変換する。
//
// raw は前後空白を除去した後に lowercase 化され、local-part と domain-part の形式を検証される。
// 空文字、空白や制御文字を含む値、複数の at mark、local/domain の空要素、長すぎる値、
// dot や hyphen の不正配置を含む値は ErrInvalidAccountEmail を返す。
func NewAccountEmail(raw string) (AccountEmail, error) {
	// Step 1: 入力の前後空白を除去し、管理画面入力の表記揺れを canonical 候補へ寄せる。
	trimmed := strings.TrimSpace(raw)

	// Step 2: email は認証・重複判定に使うため、case 差分を canonical lowercase に統一する。
	canonical := strings.ToLower(trimmed)

	// Step 3: canonical 候補の形式を domain 層で検証し、外側の layer に validation を漏らさない。
	if !isValidAccountEmail(canonical) {
		return "", ErrInvalidAccountEmail
	}

	// Step 4: 検証済み文字列だけを AccountEmail として返し、不正値の流入を防ぐ。
	return AccountEmail(canonical), nil
}

// String は AccountEmail を API、DB、監査ログの非秘匿 identifier へ渡す canonical 文字列として返す。
//
// 戻り値は NewAccountEmail によって正規化済みであり、呼び出し側で追加の trim/lowercase を行わない。
func (e AccountEmail) String() string {
	return string(e)
}

func isValidAccountEmail(value string) bool {
	// Step 1: 空文字と RFC 上限超過を拒否し、DB/API の境界で過大入力を持ち込ませない。
	if value == "" || len(value) > accountEmailMaxLength {
		return false
	}

	// Step 2: 空白・制御文字は header injection や不可視差分の原因になるため拒否する。
	if containsAccountEmailForbiddenRune(value) {
		return false
	}

	// Step 3: local/domain の分割点を 1 つだけ許し、曖昧な email 表現を拒否する。
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		return false
	}

	// Step 4: local と domain の各規則を分けて検証し、失敗理由を domain error へ畳み込む。
	return isValidAccountEmailLocalPart(parts[0]) && isValidAccountEmailDomainPart(parts[1])
}

func containsAccountEmailForbiddenRune(value string) bool {
	// Step 1: rune 単位で不可視文字を検出し、ASCII 以外の空白も確実に拒否する。
	for _, r := range value {
		// Step 2: 空白と制御文字を拒否し、email の canonical 文字列を安全に比較できる状態に保つ。
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return true
		}
	}

	// Step 3: 禁止 rune が見つからない場合だけ次の形式検証へ進める。
	return false
}

func isValidAccountEmailLocalPart(local string) bool {
	// Step 1: local-part の空要素と長すぎる値を拒否し、一般的な mailbox 制約に合わせる。
	if local == "" || len(local) > accountEmailLocalMaxLength {
		return false
	}

	// Step 2: dot の先頭・末尾・連続を拒否し、同一 mailbox の曖昧な表記を避ける。
	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return false
	}

	// Step 3: quote 付き local-part は運用上の曖昧さが大きいため、明示的に許可した ASCII だけを通す。
	for _, r := range local {
		if !isAccountEmailLocalRune(r) {
			return false
		}
	}

	// Step 4: 全文字が許可集合に含まれる場合のみ有効な local-part とする。
	return true
}

func isAccountEmailLocalRune(r rune) bool {
	// Step 1: 英数字は local-part の基本文字として許可する。
	if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
		return true
	}

	// Step 2: RFC 5322 の一般的な atext 記号と dot を許可し、quoted-string は扱わない。
	return strings.ContainsRune("!#$%&'*+-/=?^_`{|}~.", r)
}

func isValidAccountEmailDomainPart(domain string) bool {
	// Step 1: domain の空要素、全体長超過、dot の先頭末尾を拒否する。
	if domain == "" || len(domain) > accountEmailMaxLength || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// Step 2: label 分割を行い、最低 2 label を要求して運用不能な単一名 domain を拒否する。
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	// Step 3: 各 label を DNS hostname として検証し、永続化後も配送可能性のある形に限定する。
	for _, label := range labels {
		if !isValidAccountEmailDomainLabel(label) {
			return false
		}
	}

	// Step 4: 全 label が有効な場合だけ domain-part として受け入れる。
	return true
}

func isValidAccountEmailDomainLabel(label string) bool {
	// Step 1: 空 label と 63 byte 超過 label を拒否し、DNS label 境界に合わせる。
	if label == "" || len(label) > 63 {
		return false
	}

	// Step 2: hyphen の先頭・末尾配置を拒否し、hostname として不正な label を防ぐ。
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}

	// Step 3: label は ASCII 英数字と hyphen のみに制限し、IDN は punycode 後の値だけを受け付ける。
	for _, r := range label {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}

	// Step 4: 全文字が hostname label の許可集合に含まれる場合だけ有効とする。
	return true
}
