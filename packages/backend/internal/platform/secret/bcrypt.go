package secret

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptSecretCost = bcrypt.DefaultCost

var errEmptyBcryptSecret = errors.New("bcrypt secret is empty")

// HashBcryptSecret は bootstrap secret や setup token の保存用 bcrypt hash を生成する。
//
// 役割:
//   - 平文 secret を DB や設定へ保存しないため、bcrypt の低速 hash へ変換する。
//   - copy/paste 由来の前後空白だけを取り除き、空 secret は hash 化せず拒否する。
//
// 引数:
//   - value: 利用者または secure token generator から受け取った平文 secret。
//
// 戻り値:
//   - string: DB または設定へ保存する bcrypt hash。
//   - error: 空 secret、または bcrypt hash 生成失敗。
//
// 利用例:
//
//	hash, err := secret.HashBcryptSecret("temporary-bootstrap-secret")
func HashBcryptSecret(value string) (string, error) {
	// Step 1: 入力端の改行や空白だけを吸収し、実体が空の secret を保存対象から除外する。
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "", errEmptyBcryptSecret
	}

	// Step 2: bcrypt の既定 cost で保存用 hash を生成し、漏洩時のオフライン総当たり耐性を確保する。
	hash, err := bcrypt.GenerateFromPassword([]byte(trimmedValue), bcryptSecretCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// MatchesBcryptSecret は保存済み bcrypt hash と平文 secret が一致するかを判定する。
//
// 役割:
//   - bootstrap secret と setup token の照合を bcrypt に限定し、高速 digest 形式への退行を拒否する。
//   - 壊れた hash、空 hash、空 secret は一致しないものとして扱い、呼び出し側が fail-close できるようにする。
//
// 引数:
//   - hash: DB または設定から読み込んだ bcrypt hash。
//   - value: 利用者または request から受け取った平文 secret。
//
// 戻り値:
//   - bool: bcrypt 比較に成功した場合だけ true。
//
// 利用例:
//
//	if !secret.MatchesBcryptSecret(hash, input) { return errForbidden }
func MatchesBcryptSecret(hash string, value string) bool {
	// Step 1: 保存値と入力値の前後空白だけを正規化し、空値は bcrypt へ渡さず即座に拒否する。
	trimmedHash := strings.TrimSpace(hash)
	trimmedValue := strings.TrimSpace(value)
	if trimmedHash == "" || trimmedValue == "" {
		return false
	}

	// Step 2: bcrypt の定数時間比較実装に委譲し、比較詳細を呼び出し側へ漏らさない。
	return bcrypt.CompareHashAndPassword([]byte(trimmedHash), []byte(trimmedValue)) == nil
}

// IsBcryptHash は文字列が bcrypt hash として解釈できるかを検証する。
//
// 役割:
//   - startup validation で高速 digest や平文を拒否し、secret 保存形式を bcrypt へ固定する。
//   - 平文 secret との一致判定は行わず、保存形式の安全性だけを確認する。
//
// 引数:
//   - hash: 設定または DB から読み込んだ hash 候補。
//
// 戻り値:
//   - bool: bcrypt.Cost が成功する形式の場合だけ true。
//
// 利用例:
//
//	if !secret.IsBcryptHash(configuredHash) { return errInvalidConfig }
func IsBcryptHash(hash string) bool {
	// Step 1: TOML や migration seed の余分な前後空白を除去し、空値は bcrypt hash ではないものとして扱う。
	trimmedHash := strings.TrimSpace(hash)
	if trimmedHash == "" {
		return false
	}

	// Step 2: bcrypt.Cost で version/cost/hash 構造を検査し、形式が壊れている値を起動前に拒否する。
	_, err := bcrypt.Cost([]byte(trimmedHash))
	return err == nil
}
