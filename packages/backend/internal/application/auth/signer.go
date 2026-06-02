package auth

import domain "www-template/packages/backend/internal/domain"

// JSONSigner は JSON object payload を署名済み token へ変換する application capability である。
//
// 役割:
//   - 認証 use case が具体的な JWT 実装ではなく、署名能力だけへ依存できるようにする。
//   - payload の claim 意味、権限、配送方法は扱わず、署名済み文字列の生成だけを抽象化する。
//
// 引数:
//   - payload: JSON object として署名する byte 列。検証規則は実装が所有する。
//
// 戻り値:
//   - string: compact token 文字列。
//   - error: secret または payload が不正な場合の domain/application error。
//
// 使用例:
//
//	var signer JSONSigner = signVerifier
//	token, err := signer.SignJSON(payload)
type JSONSigner interface {
	SignJSON(payload []byte) (string, error)
}

// JSONVerifier は署名済み token から検証済み JSON object payload を取り出す application capability である。
//
// 役割:
//   - 認証 use case が具体的な JWT 実装ではなく、検証能力だけへ依存できるようにする。
//   - payload の Account/Operator claim への意味づけは application/auth と domain claim に残す。
//
// 引数:
//   - tokenString: compact token 文字列。
//
// 戻り値:
//   - []byte: 署名検証済み JSON object payload。呼び出し側が変更できる独立 slice。
//   - error: token 形式、署名、payload が不正な場合の domain/application error。
//
// 使用例:
//
//	var verifier JSONVerifier = signVerifier
//	payload, err := verifier.VerifyJSON(token)
type JSONVerifier interface {
	VerifyJSON(tokenString string) ([]byte, error)
}

// JSONSignVerifier は JSONSigner と JSONVerifier を同じ署名境界で合成した application capability である。
//
// 役割:
//   - accessToken 発行と bearer 検証の両方を行う service が必要 capability だけを受け取れるようにする。
//   - Product/Admin の claim 意味や lifecycle を持たず、署名能力だけを application 境界で表す。
//
// 引数・戻り値・エラー:
//   - JSONSigner / JSONVerifier の各 method に従う。
//
// 使用例:
//
//	var signVerifier JSONSignVerifier = helper
//	token, err := signVerifier.SignJSON(payload)
type JSONSignVerifier interface {
	JSONSigner
	JSONVerifier
}

// TokenJSONSignVerifier は domain.TokenJWTSigner を application の JSON capability へ適合させる adapter である。
//
// 役割:
//   - domain が所有する JWT primitive を、application/auth の signer/verifier interface として注入できるようにする。
//   - secret や署名 algorithm の検証は domain.NewTokenJWTSigner に委譲し、application 側に重複 primitive を作らない。
//   - payload の Account/Operator claim 意味は扱わず、JSON object の署名・検証だけを実行する。
//
// 使用例:
//
//	helper, err := NewTokenJSONSignVerifier([]byte("jwt-secret"))
//	if err != nil {
//		return err
//	}
//	_ = helper
type TokenJSONSignVerifier struct {
	signer domain.TokenJWTSigner
}

// NewTokenJSONSignVerifier は domain.TokenJWTSigner を使う JSONSignVerifier を生成する。
//
// 引数:
//   - secret: HMAC-SHA256 JWT 署名に使う共有 secret。空 slice は拒否される。
//
// 戻り値:
//   - TokenJSONSignVerifier: SignJSON / VerifyJSON を実装する application adapter。
//   - error: secret が不正な場合は domain.ErrInvalidSecret。
//
// 使用例:
//
//	signVerifier, err := NewTokenJSONSignVerifier(secret)
//	if err != nil {
//		return err
//	}
func NewTokenJSONSignVerifier(secret []byte) (TokenJSONSignVerifier, error) {
	// Step 1: secret 検証と defensive copy は domain primitive へ委譲し、application に署名規則を複製しない。
	signer, err := domain.NewTokenJWTSigner(secret)
	if err != nil {
		return TokenJSONSignVerifier{}, err
	}

	// Step 2: application/auth が要求する JSON capability として domain signer を保持する。
	return TokenJSONSignVerifier{signer: signer}, nil
}

// SignJSON は JSON object payload を署名済み JWT へ変換する。
//
// 引数:
//   - payload: 署名対象の JSON object byte 列。
//
// 戻り値:
//   - string: compact JWT。
//   - error: payload または signer が不正な場合の domain error。
func (v TokenJSONSignVerifier) SignJSON(payload []byte) (string, error) {
	// Step 1: payload の JSON object 検証と署名は domain primitive に委譲する。
	return v.signer.SignJWT(payload)
}

// VerifyJSON は署名済み JWT を検証し、JSON object payload を返す。
//
// 引数:
//   - tokenString: compact JWT。
//
// 戻り値:
//   - []byte: 検証済み JSON object payload。
//   - error: token 形式、署名、payload、signer が不正な場合の domain error。
func (v TokenJSONSignVerifier) VerifyJSON(tokenString string) ([]byte, error) {
	// Step 1: JWT header と署名検証、payload 正規化は domain primitive に委譲する。
	return v.signer.VerifyJWT(tokenString)
}
