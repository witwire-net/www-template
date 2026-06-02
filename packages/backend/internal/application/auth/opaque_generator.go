package auth

import (
	"crypto/rand"
	"encoding/base64"
)

const defaultOpaqueTokenBytes = 64

// CryptoOpaqueTokenGenerator は crypto/rand で Product refreshToken secret を生成する production 実装である。
//
// 役割:
//   - browser-readable storage ではなく HttpOnly Cookie へ入れる refreshToken 平文 secret を生成する。
//   - 生成後の保存は HashOpaqueToken 済み hash に限定され、この型は永続化を持たない。
//
// 使用例:
//
//	generator := NewCryptoOpaqueTokenGenerator()
//	token, err := generator.NewToken()
type CryptoOpaqueTokenGenerator struct {
	bytes int
}

// NewCryptoOpaqueTokenGenerator は既定 byte 長の crypto/rand refreshToken generator を生成する。
//
// 役割:
//   - production 用に 64 byte entropy の opaque token generator を簡単に構成する。
//   - byte 長の設定分岐を外へ出さず、Product auth service の依存注入を単純にする。
//
// 戻り値:
//   - CryptoOpaqueTokenGenerator: NewToken を実装する generator。
func NewCryptoOpaqueTokenGenerator() CryptoOpaqueTokenGenerator {
	// Step 1: 既定の 64 byte entropy を持つ generator を返す。
	return CryptoOpaqueTokenGenerator{bytes: defaultOpaqueTokenBytes}
}

// NewToken は Base64URL 形式の opaque refreshToken secret を生成する。
//
// 役割:
//   - crypto/rand から byte 列を読み、URL-safe かつ Cookie value として扱いやすい文字列へ変換する。
//   - 生成失敗時は ErrAccountAuthUnavailable を返し、弱い token への fallback を行わない。
//
// 戻り値:
//   - string: Base64URL padding なしの refreshToken secret。
//   - error: crypto/rand の読み取り失敗時は ErrAccountAuthUnavailable。
func (g CryptoOpaqueTokenGenerator) NewToken() (string, error) {
	// Step 1: byte 長が不正なゼロ値でも安全側の既定値へ補正する。
	bytes := g.bytes
	if bytes <= 0 {
		bytes = defaultOpaqueTokenBytes
	}

	// Step 2: crypto/rand で refreshToken secret の entropy を生成する。
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", ErrAccountAuthUnavailable
	}

	// Step 3: Cookie value として扱いやすい Base64URL 文字列へ変換して返す。
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
