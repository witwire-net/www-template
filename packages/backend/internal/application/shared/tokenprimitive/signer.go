package application

import domain "www-template/packages/backend/internal/domain"

// JSONSigner は JSON object payload を署名済み token へ変換する中立 interface である。
//
// 役割:
//   - 呼び出し元が具体実装ではなく署名能力だけへ依存できるようにする。
//   - payload の claim 意味、権限、発行元、配送方法を扱わず、byte 列の署名だけを抽象化する。
//
// 引数:
//   - SignJSON の payload: JSON object として署名する byte 列。
//
// 戻り値:
//   - string: 署名済み compact token。
//   - error: secret または payload が不正な場合の domain error。
//
// エラーケース:
//   - domain.ErrInvalidSecret: 署名 secret が空の場合。
//   - domain.ErrInvalidTokenPayload: payload が JSON object でない場合。
//
// 使用例:
//
//	var signer JSONSigner = signVerifier
//	tokenString, err := signer.SignJSON(payload)
type JSONSigner interface {
	SignJSON(payload []byte) (string, error)
}

// JSONVerifier は署名済み token から JSON object payload を検証済み byte 列として取り出す中立 interface である。
//
// 役割:
//   - 呼び出し元が具体実装ではなく検証能力だけへ依存できるようにする。
//   - payload の claim 意味、有効期限、権限判定は呼び出し元へ残し、署名検証だけを抽象化する。
//
// 引数:
//   - VerifyJSON の tokenString: compact token 文字列。
//
// 戻り値:
//   - []byte: 検証済み JSON object payload。呼び出し元が変更できる独立 slice。
//   - error: token 形式、secret、署名、payload が不正な場合の domain error。
//
// エラーケース:
//   - domain.ErrInvalidSecret: 検証 secret が空の場合。
//   - domain.ErrMalformedToken: compact token が期待形式でない場合。
//   - domain.ErrInvalidSignature: 署名または header が不正な場合。
//   - domain.ErrInvalidTokenPayload: payload が JSON object でない場合。
//
// 使用例:
//
//	var verifier JSONVerifier = signVerifier
//	payload, err := verifier.VerifyJSON(tokenString)
type JSONVerifier interface {
	VerifyJSON(tokenString string) ([]byte, error)
}

// JSONSignVerifier は JSONSigner と JSONVerifier を同じ secret 境界で合成した interface である。
//
// 役割:
//   - 署名と検証を同じ helper instance に束ね、use case が必要な capability だけへ依存できるようにする。
//   - payload の意味づけや claim 変換を持たず、signer/verifier composition だけを提供する。
//
// 引数:
//   - JSONSigner と JSONVerifier の各 method に従う。
//
// 戻り値:
//   - JSONSigner と JSONVerifier の各 method に従う。
//
// エラーケース:
//   - JSONSigner と JSONVerifier の各 method に従う。
//
// 使用例:
//
//	var helper JSONSignVerifier = signVerifier
//	tokenString, err := helper.SignJSON(payload)
type JSONSignVerifier interface {
	JSONSigner
	JSONVerifier
}

// JWTSignVerifier は HS256 JWT の署名と検証を共有する application helper である。
//
// 役割:
//   - domain.TokenJWTSigner を application shared package の中立 interface へ適合させる。
//   - secret を domain primitive 内へ複製して保持し、呼び出し元の slice 変更による副作用を避ける。
//   - payload の意味、有効期限、発行元、配送方法を保持しない。
//
// 引数:
//   - NewJWTSignVerifier の secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否される。
//
// 戻り値:
//   - JWTSignVerifier: SignJSON と VerifyJSON を実装する immutable helper。
//   - error: secret が空の場合は domain.ErrInvalidSecret。
//
// エラーケース:
//   - domain.ErrInvalidSecret: secret が空で、署名検証境界を構成できない場合。
//
// 使用例:
//
//	helper, err := NewJWTSignVerifier([]byte("shared-secret"))
//	if err != nil {
//		return err
//	}
//	tokenString, err := helper.SignJSON([]byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`))
type JWTSignVerifier struct {
	value domain.TokenJWTSigner
}

// NewJWTSignVerifier は HS256 JWT の signer/verifier helper を生成する。
//
// 役割:
//   - secret 検証と複製を domain primitive に委譲し、application shared 側で secret 規則を二重実装しない。
//   - 生成した helper を JSONSigner、JSONVerifier、JSONSignVerifier として再利用可能にする。
//
// 引数:
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は安全でないため拒否される。
//
// 戻り値:
//   - JWTSignVerifier: JSON object payload の署名と検証に使う helper。
//   - error: secret が空の場合は domain.ErrInvalidSecret。
//
// エラーケース:
//   - domain.ErrInvalidSecret: secret が空の場合。
//
// 使用例:
//
//	helper, err := NewJWTSignVerifier(secret)
//	if err != nil {
//		return err
//	}
func NewJWTSignVerifier(secret []byte) (JWTSignVerifier, error) {
	// secret 検証と defensive copy は domain primitive の constructor に集約する。
	value, err := domain.NewTokenJWTSigner(secret)
	if err != nil {
		return JWTSignVerifier{}, err
	}

	// application shared helper として、domain primitive を薄く包んで返す。
	return JWTSignVerifier{value: value}, nil
}

// SignJSON は JSON object payload を HS256 JWT として署名する。
//
// 役割:
//   - JWTSignVerifier が保持する secret で payload を compact token へ変換する。
//   - payload の claim 名や値は解釈せず、JSON object であることの検証は domain primitive に委譲する。
//
// 引数:
//   - payload: 署名対象の JSON object byte 列。前後空白は許容される。
//
// 戻り値:
//   - string: header.payload.signature 形式の compact token。
//   - error: payload が不正、または helper が未初期化の場合の domain error。
//
// エラーケース:
//   - domain.ErrInvalidTokenPayload: payload が空、JSON として不正、または JSON object でない場合。
//   - domain.ErrInvalidSecret: helper が未初期化などで secret を持たない場合。
//
// 使用例:
//
//	tokenString, err := helper.SignJSON(payload)
//	if err != nil {
//		return err
//	}
func (helper JWTSignVerifier) SignJSON(payload []byte) (string, error) {
	// 実際の署名処理は domain primitive に委譲し、この層では claim 意味を追加しない。
	return helper.value.SignJWT(payload)
}

// VerifyJSON は HS256 JWT の署名を検証し、payload JSON を返す。
//
// 役割:
//   - JWTSignVerifier が保持する secret で compact token の header と署名を検証する。
//   - payload の意味、有効期限、発行元などの上位判断は呼び出し元へ残す。
//
// 引数:
//   - tokenString: header.payload.signature 形式の compact token。
//
// 戻り値:
//   - []byte: 検証済み JSON object payload。呼び出し元が変更できる独立 slice。
//   - error: token 形式、署名、payload、secret が不正な場合の domain error。
//
// エラーケース:
//   - domain.ErrMalformedToken: compact token が 3 segment ではない場合。
//   - domain.ErrInvalidSignature: header または signature が不正な場合。
//   - domain.ErrInvalidTokenPayload: payload が JSON object として扱えない場合。
//   - domain.ErrInvalidSecret: helper が未初期化などで secret を持たない場合。
//
// 使用例:
//
//	payload, err := helper.VerifyJSON(tokenString)
//	if err != nil {
//		return err
//	}
func (helper JWTSignVerifier) VerifyJSON(tokenString string) ([]byte, error) {
	// 実際の検証処理は domain primitive に委譲し、この層では payload 意味を追加しない。
	return helper.value.VerifyJWT(tokenString)
}

// SignJSON は一時的な signer/verifier helper を使い、JSON object payload を署名する。
//
// 役割:
//   - 呼び出し元が helper instance を保持しない一回限りの署名用途を提供する。
//   - NewJWTSignVerifier と同じ secret 検証経路を通し、method API と関数 API の挙動を揃える。
//
// 引数:
//   - payload: 署名対象の JSON object byte 列。
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否される。
//
// 戻り値:
//   - string: header.payload.signature 形式の compact token。
//   - error: secret または payload が不正な場合の domain error。
//
// エラーケース:
//   - domain.ErrInvalidSecret: secret が空の場合。
//   - domain.ErrInvalidTokenPayload: payload が JSON object でない場合。
//
// 使用例:
//
//	tokenString, err := SignJSON(payload, secret)
//	if err != nil {
//		return err
//	}
func SignJSON(payload []byte, secret []byte) (string, error) {
	// 一回限りの利用でも constructor 経由にし、secret defensive copy と検証を統一する。
	helper, err := NewJWTSignVerifier(secret)
	if err != nil {
		return "", err
	}

	// 生成した helper に署名処理を委譲する。
	return helper.SignJSON(payload)
}

// VerifyJSON は一時的な signer/verifier helper を使い、署名済み token の payload を検証して返す。
//
// 役割:
//   - 呼び出し元が helper instance を保持しない一回限りの検証用途を提供する。
//   - NewJWTSignVerifier と同じ secret 検証経路を通し、method API と関数 API の挙動を揃える。
//
// 引数:
//   - tokenString: header.payload.signature 形式の compact token。
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否される。
//
// 戻り値:
//   - []byte: 検証済み JSON object payload。呼び出し元が変更できる独立 slice。
//   - error: secret、token 形式、署名、payload が不正な場合の domain error。
//
// エラーケース:
//   - domain.ErrInvalidSecret: secret が空の場合。
//   - domain.ErrMalformedToken: compact token が 3 segment ではない場合。
//   - domain.ErrInvalidSignature: header または signature が不正な場合。
//   - domain.ErrInvalidTokenPayload: payload が JSON object でない場合。
//
// 使用例:
//
//	payload, err := VerifyJSON(tokenString, secret)
//	if err != nil {
//		return err
//	}
func VerifyJSON(tokenString string, secret []byte) ([]byte, error) {
	// 一回限りの利用でも constructor 経由にし、secret defensive copy と検証を統一する。
	helper, err := NewJWTSignVerifier(secret)
	if err != nil {
		return nil, err
	}

	// 生成した helper に検証処理を委譲する。
	return helper.VerifyJSON(tokenString)
}
