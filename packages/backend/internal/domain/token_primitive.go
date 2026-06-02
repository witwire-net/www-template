package domain

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidTokenPayload は JWT payload が JSON object として扱えない場合に返すエラー。
	// 署名対象の payload は上位層が意味づけるため、この primitive では構造の意味を解釈しない。
	ErrInvalidTokenPayload = errors.New("invalid token payload")

	// ErrInvalidTokenTTL は token lifetime が 0 以下の場合に返すエラー。
	// TTL は呼び出し側が注入した値だけから検証し、この層では現在時刻を直接読まない。
	ErrInvalidTokenTTL = errors.New("invalid token ttl")

	// ErrInvalidTokenCookieLifetime は cookie lifetime が token TTL を超える、または 0 以下の場合に返すエラー。
	// cookie を使う上位層が secret の寿命を token state より長くしないための中立検証に使う。
	ErrInvalidTokenCookieLifetime = errors.New("invalid token cookie lifetime")
)

// TokenJWTSigner は HS256 JWT の署名と検証だけを担う中立 signer/verifier。
//
// 役割:
//   - secret を内部に複製し、外部 slice の後続変更が署名結果へ影響しないようにする。
//   - payload を JSON object として検証したうえで HS256 JWT を生成する。
//   - JWT header と HMAC 署名だけを検証し、payload の意味づけは呼び出し側へ残す。
//
// 引数:
//   - secret: HMAC-SHA256 に使う共有 secret。空の場合は ErrInvalidSecret を返す。
//
// 戻り値:
//   - TokenJWTSigner: 署名・検証に使う immutable value。
//   - error: secret が空の場合に ErrInvalidSecret。
//
// 使用例:
//
//	signer, err := NewTokenJWTSigner([]byte("secret"))
//	if err != nil {
//		return err
//	}
//	token, err := signer.SignJWT([]byte(`{"sub":"01ARZ3NDEKTSV4RRFFQ69G5FAV"}`))
type TokenJWTSigner struct {
	secret []byte
}

// OpaqueTokenHash は opaque token の SHA-256 hash を Base64URL 文字列で保持する値 object。
//
// 役割:
//   - 平文 token を永続化しないための保存用 fingerprint を表す。
//   - 比較時は constant-time 比較を使い、hash 値の一致可否だけを返す。
//
// 引数:
//   - 値生成時の token: 空白除去後に空でない opaque token。
//
// 戻り値:
//   - OpaqueTokenHash: Base64URL encoded SHA-256 digest。
//   - error: token が空の場合に ErrInvalidToken。
//
// 使用例:
//
//	hash, err := HashOpaqueToken("plain-secret")
//	if err != nil {
//		return err
//	}
//	if !hash.Matches("plain-secret") {
//		return ErrInvalidToken
//	}
type OpaqueTokenHash string

// TokenULID は token 系識別子に使う ULID 形式の中立値 object。
//
// 役割:
//   - 26 文字 Crockford Base32 ULID の形式検証だけを担う。
//   - どの上位概念に属するかは保持しない。
//
// 引数:
//   - value: 検証対象の ULID 文字列。前後空白は除去される。
//
// 戻り値:
//   - TokenULID: 検証済み ULID。
//   - error: 形式が不正な場合に ErrInvalidAuthID。
//
// 使用例:
//
//	id, err := NewTokenULID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
type TokenULID string

// TokenJTI は JWT ID claim に使う検証済み ULID 値 object。
//
// 役割:
//   - jti の一意識別子が ULID 形式であることだけを保証する。
//   - JWT payload の用途や権限は解釈しない。
//
// 引数:
//   - value: 検証対象の jti 文字列。前後空白は除去される。
//
// 戻り値:
//   - TokenJTI: 検証済み jti。
//   - error: 形式が不正な場合に ErrInvalidAuthID。
//
// 使用例:
//
//	jti, err := NewTokenJTI("01ARZ3NDEKTSV4RRFFQ69G5FAX")
type TokenJTI string

// TokenTTL は token lifetime を表す中立値 object。
//
// 役割:
//   - 0 より大きい duration だけを保持する。
//   - expiresAt の計算では渡された発行時刻だけを使い、この層では現在時刻を読まない。
//
// 引数:
//   - duration: token の有効期間。0 以下は ErrInvalidTokenTTL。
//
// 戻り値:
//   - TokenTTL: 検証済み lifetime。
//   - error: duration が 0 以下の場合に ErrInvalidTokenTTL。
//
// 使用例:
//
//	ttl, err := ValidateTokenTTL(15 * time.Minute)
//	expiresAt := ttl.ExpiresAt(issuedAt)
type TokenTTL struct {
	duration time.Duration
}

// tokenPrimitiveJWTHeader は JWT header の固定 JSON。
// HS256 以外を発行しないことで algorithm confusion を避ける。
var tokenPrimitiveJWTHeader = []byte(`{"alg":"HS256","typ":"JWT"}`)

// NewTokenJWTSigner は HS256 JWT signer/verifier を生成する。
//
// 役割:
//   - HMAC-SHA256 の secret を検証し、後続の署名・検証で再利用できる値 object を作る。
//   - secret slice を複製して保持し、呼び出し側の slice 変更による副作用を避ける。
//
// 引数:
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は安全でないため拒否する。
//
// 戻り値:
//   - TokenJWTSigner: secret を内部複製した signer/verifier。
//   - error: secret が空の場合は ErrInvalidSecret。
//
// エラーケース:
//   - ErrInvalidSecret: secret が空で、署名境界を安全に構成できない場合。
//
// 使用例:
//
//	signer, err := NewTokenJWTSigner([]byte("shared-secret"))
//	if err != nil {
//		return err
//	}
//	_, err = signer.SignJWT([]byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`))
func NewTokenJWTSigner(secret []byte) (TokenJWTSigner, error) {
	// 入力 secret が空の場合は、署名不能な状態を作らず fail-close する。
	if len(secret) == 0 {
		return TokenJWTSigner{}, ErrInvalidSecret
	}

	// 呼び出し側が後から secret slice を変更しても、この値 object の挙動が変わらないよう複製する。
	secretCopy := append([]byte(nil), secret...)

	// 複製済み secret を持つ signer/verifier を返す。
	return TokenJWTSigner{secret: secretCopy}, nil
}

// SignTokenHMACSHA256 は任意 byte 列へ HMAC-SHA256 署名を付与する。
//
// 役割:
//   - data を意味解釈せず、その byte 列そのものを HMAC-SHA256 の署名対象にする。
//   - JWT 以外の opaque な署名用途にも使える低レベル primitive を提供する。
//
// 引数:
//   - data: 署名対象の byte 列。空でも署名対象として扱う。
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否する。
//
// 戻り値:
//   - []byte: HMAC-SHA256 の raw digest。呼び出し側が保持できる新規 slice。
//   - error: secret が空の場合は ErrInvalidSecret。
//
// エラーケース:
//   - ErrInvalidSecret: secret が空で、署名境界を安全に構成できない場合。
//
// 使用例:
//
//	signature, err := SignTokenHMACSHA256([]byte("payload"), []byte("shared-secret"))
//	if err != nil {
//		return err
//	}
func SignTokenHMACSHA256(data []byte, secret []byte) ([]byte, error) {
	// 空 secret で署名すると検証境界が弱くなるため拒否する。
	if len(secret) == 0 {
		return nil, ErrInvalidSecret
	}

	// HMAC-SHA256 signer を生成し、入力 byte 列をそのまま署名対象にする。
	mac := hmac.New(sha256.New, secret)

	// hash.Hash.Write は仕様上 nil error しか返さないが、errcheck のため戻り値を明示的に受ける。
	_, _ = mac.Write(data)

	// 呼び出し側が安全に保持できるよう、新規 slice として署名 byte 列を返す。
	return mac.Sum(nil), nil
}

// VerifyTokenHMACSHA256 は HMAC-SHA256 署名が data と secret に一致することを検証する。
//
// 役割:
//   - 同じ data と secret から期待署名を再計算し、提示署名と constant-time で比較する。
//   - token 形式や payload 意味は扱わず、HMAC の正当性だけを判断する。
//
// 引数:
//   - data: 署名時と同じ署名対象 byte 列。
//   - signature: 検証対象の raw HMAC-SHA256 digest。
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否する。
//
// 戻り値:
//   - error: 検証成功時は nil。失敗時は ErrInvalidSecret または ErrInvalidSignature。
//
// エラーケース:
//   - ErrInvalidSecret: secret が空の場合。
//   - ErrInvalidSignature: signature が data と secret から再計算した digest と一致しない場合。
//
// 使用例:
//
//	if err := VerifyTokenHMACSHA256([]byte("payload"), signature, []byte("shared-secret")); err != nil {
//		return err
//	}
func VerifyTokenHMACSHA256(data []byte, signature []byte, secret []byte) error {
	// 同じ署名関数で期待値を作り、署名生成時と同一の secret 検証を適用する。
	expected, err := SignTokenHMACSHA256(data, secret)
	if err != nil {
		return err
	}

	// constant-time 比較で署名値の一致を検証し、timing 差を分岐条件へ持ち込まない。
	if !hmac.Equal(expected, signature) {
		return ErrInvalidSignature
	}

	// 署名が一致したため検証成功として nil を返す。
	return nil
}

// SignTokenJWT は JSON object payload を HS256 JWT として署名する。
//
// 役割:
//   - 関数形式で signer を一時生成し、payload を compact JWT へ変換する。
//   - payload は JSON object であることだけを検証し、claim 名や値の意味は扱わない。
//
// 引数:
//   - payload: JWT payload として入れる JSON object。前後空白は除去される。
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否する。
//
// 戻り値:
//   - string: header.payload.signature 形式の compact JWT。
//   - error: secret または payload が不正な場合の domain error。
//
// エラーケース:
//   - ErrInvalidSecret: secret が空の場合。
//   - ErrInvalidTokenPayload: payload が空、JSON として不正、または JSON object でない場合。
//
// 使用例:
//
//	tokenString, err := SignTokenJWT([]byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`), secret)
//	if err != nil {
//		return err
//	}
func SignTokenJWT(payload []byte, secret []byte) (string, error) {
	// secret の検証と複製は signer constructor に集約する。
	signer, err := NewTokenJWTSigner(secret)
	if err != nil {
		return "", err
	}

	// 生成した signer に署名処理を委譲し、関数 API と method API の挙動を揃える。
	return signer.SignJWT(payload)
}

// VerifyTokenJWT は HS256 JWT の header と署名を検証し、payload を返す。
//
// 役割:
//   - 関数形式で verifier を一時生成し、compact JWT の header と HMAC 署名だけを検証する。
//   - payload の claim 意味、有効期限、発行元などの上位判断は呼び出し側へ残す。
//
// 引数:
//   - tokenString: header.payload.signature 形式の compact JWT。
//   - secret: HMAC-SHA256 に使う共有 secret。空 slice は拒否する。
//
// 戻り値:
//   - []byte: 検証済み payload JSON object。呼び出し側が変更可能な新規 slice。
//   - error: token 形式、secret、署名、payload が不正な場合の domain error。
//
// エラーケース:
//   - ErrInvalidSecret: secret が空の場合。
//   - ErrMalformedToken: JWT が 3 segment ではない場合。
//   - ErrInvalidSignature: header または signature が不正な場合。
//   - ErrInvalidTokenPayload: payload が JSON object として扱えない場合。
//
// 使用例:
//
//	payload, err := VerifyTokenJWT(tokenString, secret)
//	if err != nil {
//		return err
//	}
func VerifyTokenJWT(tokenString string, secret []byte) ([]byte, error) {
	// secret の検証と複製は signer constructor に集約する。
	signer, err := NewTokenJWTSigner(secret)
	if err != nil {
		return nil, err
	}

	// 生成した verifier に検証処理を委譲し、関数 API と method API の挙動を揃える。
	return signer.VerifyJWT(tokenString)
}

// SignJWT は JSON object payload を HS256 JWT として署名する。
//
// 役割:
//   - TokenJWTSigner が保持する secret で payload を署名し、compact JWT を作る。
//   - header は HS256 JWT の固定値にし、payload の意味は一切解釈しない。
//
// 引数:
//   - payload: JWT payload として入れる JSON object。前後空白は除去される。
//
// 戻り値:
//   - string: header.payload.signature 形式の compact JWT。
//   - error: payload が不正な場合、または signer の secret が空の場合の domain error。
//
// エラーケース:
//   - ErrInvalidTokenPayload: payload が空、JSON として不正、または JSON object でない場合。
//   - ErrInvalidSecret: ゼロ値 signer などで secret が空の場合。
//
// 使用例:
//
//	tokenString, err := signer.SignJWT([]byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`))
//	if err != nil {
//		return err
//	}
func (s TokenJWTSigner) SignJWT(payload []byte) (string, error) {
	// payload は意味を解釈せず、JSON object として正規化可能かだけを検証する。
	normalizedPayload, err := normalizeTokenJWTPayload(payload)
	if err != nil {
		return "", err
	}

	// JWT header と payload を Base64URL で segment 化する。
	headerSegment := encodeTokenSegment(tokenPrimitiveJWTHeader)
	payloadSegment := encodeTokenSegment(normalizedPayload)

	// header.payload の形式を署名入力にする。
	signingInput := headerSegment + "." + payloadSegment

	// HS256 署名を作り、JWT 第三 segment として Base64URL 化する。
	signature, err := SignTokenHMACSHA256([]byte(signingInput), s.secret)
	if err != nil {
		return "", err
	}
	signatureSegment := encodeTokenSegment(signature)

	// compact serialization 形式の JWT を返す。
	return signingInput + "." + signatureSegment, nil
}

// VerifyJWT は HS256 JWT の header と署名を検証し、payload JSON を返す。
//
// 役割:
//   - TokenJWTSigner が保持する secret で compact JWT の HMAC 署名を検証する。
//   - header は HS256 JWT の固定値だけを許可し、payload の意味は呼び出し側へ残す。
//
// 引数:
//   - tokenString: header.payload.signature 形式の compact JWT。
//
// 戻り値:
//   - []byte: 検証済み payload JSON object。呼び出し側が変更可能な新規 slice。
//   - error: token 形式、署名、payload が不正な場合の domain error。
//
// エラーケース:
//   - ErrMalformedToken: JWT が 3 segment ではない場合。
//   - ErrInvalidSignature: header または signature が不正な場合。
//   - ErrInvalidTokenPayload: payload が JSON object として扱えない場合。
//   - ErrInvalidSecret: ゼロ値 signer などで secret が空の場合。
//
// 使用例:
//
//	payload, err := signer.VerifyJWT(tokenString)
//	if err != nil {
//		return err
//	}
func (s TokenJWTSigner) VerifyJWT(tokenString string) ([]byte, error) {
	// compact serialization は必ず header.payload.signature の 3 segment でなければならない。
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrMalformedToken
	}

	// header は HS256 JWT の固定値と一致することだけを検証する。
	if err := verifyTokenPrimitiveJWTHeader(parts[0]); err != nil {
		return nil, err
	}

	// 署名 segment を Base64URL decode し、HMAC 入力と照合する。
	signature, err := decodeTokenSegment(parts[2])
	if err != nil {
		return nil, ErrInvalidSignature
	}
	if err := VerifyTokenHMACSHA256([]byte(parts[0]+"."+parts[1]), signature, s.secret); err != nil {
		return nil, err
	}

	// payload segment は JSON object として再検証し、破損した payload を成功扱いにしない。
	payload, err := decodeTokenSegment(parts[1])
	if err != nil {
		return nil, ErrInvalidTokenPayload
	}
	normalizedPayload, err := normalizeTokenJWTPayload(payload)
	if err != nil {
		return nil, err
	}

	// 呼び出し側の変更が内部 buffer に影響しないよう、独立した slice として返す。
	return append([]byte(nil), normalizedPayload...), nil
}

// HashOpaqueToken は opaque token を SHA-256 hash に変換する。
//
// 役割:
//   - 平文 token を保存せずに照合できる Base64URL encoded digest を作る。
//   - token の用途や形式を解釈せず、空でない opaque 値としてだけ扱う。
//
// 引数:
//   - token: hash 化する平文 token。前後空白は除去される。
//
// 戻り値:
//   - OpaqueTokenHash: SHA-256 digest を padding なし Base64URL にした値。
//   - error: token が空の場合は ErrInvalidToken。
//
// エラーケース:
//   - ErrInvalidToken: 前後空白除去後の token が空の場合。
//
// 使用例:
//
//	hash, err := HashOpaqueToken("plain-token")
//	if err != nil {
//		return err
//	}
func HashOpaqueToken(token string) (OpaqueTokenHash, error) {
	// 入力 token は copy/paste 由来の前後空白だけを除去し、中身は opaque 値として解釈しない。
	normalizedToken := strings.TrimSpace(token)
	if normalizedToken == "" {
		return "", ErrInvalidToken
	}

	// SHA-256 digest を保存用 fingerprint として生成する。
	digest := sha256.Sum256([]byte(normalizedToken))

	// digest は URL/DB で扱いやすい padding なし Base64URL にする。
	return OpaqueTokenHash(encodeTokenSegment(digest[:])), nil
}

// String は OpaqueTokenHash の保存・比較用文字列表現を返す。
//
// 役割:
//   - 平文 token ではなく、Base64URL encoded digest だけを文字列化する。
//
// 引数:
//   - なし。
//
// 戻り値:
//   - string: 保存・比較に使う hash 文字列。
//
// エラーケース:
//   - なし。
//
// 使用例:
//
//	storedValue := hash.String()
func (h OpaqueTokenHash) String() string {
	// value object の内部表現を文字列として返す。
	return string(h)
}

// Matches は平文 opaque token が保存済み hash と一致するかを constant-time で検証する。
//
// 役割:
//   - 入力 token を HashOpaqueToken と同じ規則で hash 化し、保存済み hash と比較する。
//   - 比較には hmac.Equal を使い、timing 差で一致状態を漏らさない。
//
// 引数:
//   - token: 照合したい平文 opaque token。前後空白は除去される。
//
// 戻り値:
//   - bool: hash が一致する場合は true。不一致、空 token、未初期化 hash は false。
//
// エラーケース:
//   - error は返さない。HashOpaqueToken が ErrInvalidToken になる入力は false に畳む。
//
// 使用例:
//
//	if !hash.Matches("plain-token") {
//		return ErrInvalidToken
//	}
func (h OpaqueTokenHash) Matches(token string) bool {
	// 比較対象 token を同じ hash 関数へ通し、入力検証も同一にする。
	candidate, err := HashOpaqueToken(token)
	if err != nil {
		return false
	}

	// 空 hash は未初期化値として扱い、一致扱いにしない。
	if h == "" {
		return false
	}

	// digest 文字列同士を constant-time で比較する。
	return hmac.Equal([]byte(h.String()), []byte(candidate.String()))
}

// EnsureRefreshContext は request path の authContextID と保存済み refresh/session context の一致を検証する。
//
// 役割:
//   - Cookie Path を認可境界として信用せず、server-side record が持つ context と path selector を必ず照合する。
//   - Product AccountAuth と Admin OperatorAuth の refresh credential 所有不変条件を、HTTP path 文字列生成から分離して domain に集約する。
//   - Account/Operator の所有者や権限の判定は各 lifecycle object に残し、この関数は context selector の同一性だけを扱う。
//
// 引数:
//   - requestedAuthContextID: HTTP route path など外側境界から取得した authContextId。
//   - storedAuthContextID: refresh/session store から復元した authContextId または session selector。
//
// 戻り値:
//   - nil: 両方が canonical ULID で完全一致する場合。
//   - error: 入力不正または不一致の場合。形式不正は ErrInvalidAuthID、不一致は ErrInvalidToken を返す。
//
// 使用例:
//
//	if err := EnsureRefreshContext(pathContextID, session.ID().String()); err != nil {
//		return err
//	}
func EnsureRefreshContext(requestedAuthContextID string, storedAuthContextID string) error {
	// Step 1: path 由来 selector を canonical ULID として検証し、空白や path traversal 文字列を拒否する。
	requested := strings.TrimSpace(requestedAuthContextID)
	if err := ValidateAuthID(requested); err != nil {
		return err
	}

	// Step 2: store 由来 selector も同じ規則で検証し、壊れた永続化値を fail-closed に扱う。
	stored := strings.TrimSpace(storedAuthContextID)
	if err := ValidateAuthID(stored); err != nil {
		return err
	}

	// Step 3: 完全一致しない context は credential 所属不一致として拒否し、新しい token を発行させない。
	if requested != stored {
		return ErrInvalidToken
	}

	// Step 4: request path と server-side context が一致したため成功とする。
	return nil
}

// NewTokenULID は token 系識別子を ULID として検証する。
//
// 役割:
//   - value が既存の ULID 検証規則に合うことだけを保証する。
//   - どの上位概念に属するかは保持せず、形式検証だけに留める。
//
// 引数:
//   - value: 検証対象の ULID 文字列。前後空白は除去される。
//
// 戻り値:
//   - TokenULID: 検証済み ULID value object。
//   - error: 形式が不正な場合は ErrInvalidAuthID。
//
// エラーケース:
//   - ErrInvalidAuthID: value が 26 文字 Crockford Base32 ULID ではない場合。
//
// 使用例:
//
//	id, err := NewTokenULID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
//	if err != nil {
//		return err
//	}
func NewTokenULID(value string) (TokenULID, error) {
	// 入力の前後空白だけを取り除き、値の意味は解釈しない。
	normalizedValue := strings.TrimSpace(value)

	// 既存の ULID 検証規則に合わせ、形式不一致は ErrInvalidAuthID とする。
	if err := ValidateAuthID(normalizedValue); err != nil {
		return "", err
	}

	// 検証済み ULID を中立 value object として返す。
	return TokenULID(normalizedValue), nil
}

// String は TokenULID の正規化済み文字列表現を返す。
//
// 役割:
//   - 検証済み ULID を保存・比較・payload 組み立てで使える文字列に戻す。
//
// 引数:
//   - なし。
//
// 戻り値:
//   - string: 前後空白を含まない ULID 文字列。
//
// エラーケース:
//   - なし。
//
// 使用例:
//
//	value := id.String()
func (id TokenULID) String() string {
	// value object の内部表現を文字列として返す。
	return string(id)
}

// NewTokenJTI は JWT ID を ULID として検証する。
//
// 役割:
//   - jti 値が ULID 形式であることだけを保証する。
//   - JWT payload の用途、権限、発行元などは解釈しない。
//
// 引数:
//   - value: 検証対象の jti 文字列。前後空白は除去される。
//
// 戻り値:
//   - TokenJTI: 検証済み jti value object。
//   - error: 形式が不正な場合は ErrInvalidAuthID。
//
// エラーケース:
//   - ErrInvalidAuthID: value が 26 文字 Crockford Base32 ULID ではない場合。
//
// 使用例:
//
//	jti, err := NewTokenJTI("01ARZ3NDEKTSV4RRFFQ69G5FAX")
//	if err != nil {
//		return err
//	}
func NewTokenJTI(value string) (TokenJTI, error) {
	// jti も token 系識別子と同じ ULID 規則で検証する。
	id, err := NewTokenULID(value)
	if err != nil {
		return "", err
	}

	// 型を分けることで呼び出し側が jti とその他識別子を混同しにくくする。
	return TokenJTI(id.String()), nil
}

// String は TokenJTI の正規化済み文字列表現を返す。
//
// 役割:
//   - 検証済み jti を保存・比較・payload 組み立てで使える文字列に戻す。
//
// 引数:
//   - なし。
//
// 戻り値:
//   - string: 前後空白を含まない jti 文字列。
//
// エラーケース:
//   - なし。
//
// 使用例:
//
//	value := jti.String()
func (jti TokenJTI) String() string {
	// value object の内部表現を文字列として返す。
	return string(jti)
}

// ValidateTokenTTL は token lifetime が 0 より大きいことを検証する。
//
// 役割:
//   - 外部設定や上位層から渡された duration を TokenTTL value object に変換する。
//   - 現在時刻や環境変数は読まず、duration の値だけを検証する。
//
// 引数:
//   - duration: token の有効期間。0 より大きい必要がある。
//
// 戻り値:
//   - TokenTTL: 検証済み TTL value object。
//   - error: duration が 0 以下の場合は ErrInvalidTokenTTL。
//
// エラーケース:
//   - ErrInvalidTokenTTL: duration が 0 以下の場合。
//
// 使用例:
//
//	ttl, err := ValidateTokenTTL(15 * time.Minute)
//	if err != nil {
//		return err
//	}
func ValidateTokenTTL(duration time.Duration) (TokenTTL, error) {
	// constructor と同じ検証経路に集約し、API 名だけを用途に合わせて公開する。
	return NewTokenTTL(duration)
}

// NewTokenTTL は token lifetime value object を生成する。
//
// 役割:
//   - 正の duration だけを TokenTTL として保持する。
//   - ValidateTokenTTL と同じ検証を constructor 名で提供する。
//
// 引数:
//   - duration: token の有効期間。0 より大きい必要がある。
//
// 戻り値:
//   - TokenTTL: 検証済み TTL value object。
//   - error: duration が 0 以下の場合は ErrInvalidTokenTTL。
//
// エラーケース:
//   - ErrInvalidTokenTTL: duration が 0 以下の場合。
//
// 使用例:
//
//	ttl, err := NewTokenTTL(15 * time.Minute)
//	if err != nil {
//		return err
//	}
func NewTokenTTL(duration time.Duration) (TokenTTL, error) {
	// 0 または負の TTL は即時失効や逆転した有効期間を生むため拒否する。
	if duration <= 0 {
		return TokenTTL{}, ErrInvalidTokenTTL
	}

	// 検証済み duration だけを保持する。
	return TokenTTL{duration: duration}, nil
}

// Duration は TokenTTL の time.Duration 表現を返す。
//
// 役割:
//   - 検証済み TTL を標準 library の duration として上位層へ渡す。
//
// 引数:
//   - なし。
//
// 戻り値:
//   - time.Duration: TokenTTL が保持する正の duration。
//
// エラーケース:
//   - なし。
//
// 使用例:
//
//	duration := ttl.Duration()
func (ttl TokenTTL) Duration() time.Duration {
	// value object の内部 duration を返す。
	return ttl.duration
}

// ExpiresAt は発行時刻に TTL を加算した失効時刻を UTC で返す。
//
// 役割:
//   - 呼び出し側が注入した issuedAt と検証済み TTL から deterministic に失効時刻を計算する。
//   - この method は time.Now を直接呼ばず、domain 層の副作用源を増やさない。
//
// 引数:
//   - issuedAt: token の発行時刻。任意 timezone を受け取り、UTC へ正規化する。
//
// 戻り値:
//   - time.Time: issuedAt.UTC() に TTL を加えた失効時刻。
//
// エラーケース:
//   - なし。ゼロ値 TokenTTL では issuedAt.UTC() のまま返るため、生成時は NewTokenTTL を使う。
//
// 使用例:
//
//	expiresAt := ttl.ExpiresAt(issuedAt)
func (ttl TokenTTL) ExpiresAt(issuedAt time.Time) time.Time {
	// 呼び出し側から渡された時刻を UTC に寄せ、保存・比較の揺れを抑える。
	return issuedAt.UTC().Add(ttl.duration)
}

// ValidateTokenCookieLifetime は cookie lifetime が token TTL を超えないことを検証する。
//
// 役割:
//   - cookie による保持時間が server-side token lifetime より長くならないことを保証する。
//   - cookie 属性そのものは作らず、duration の大小関係だけを検証する。
//
// 引数:
//   - cookieLifetime: cookie に設定する保持時間。0 より大きく、ttl 以下である必要がある。
//   - ttl: NewTokenTTL または ValidateTokenTTL で作った token lifetime。
//
// 戻り値:
//   - error: 検証成功時は nil。失敗時は ErrInvalidTokenTTL または ErrInvalidTokenCookieLifetime。
//
// エラーケース:
//   - ErrInvalidTokenTTL: ttl が未初期化または 0 以下の場合。
//   - ErrInvalidTokenCookieLifetime: cookieLifetime が 0 以下、または ttl より長い場合。
//
// 使用例:
//
//	if err := ValidateTokenCookieLifetime(10*time.Minute, ttl); err != nil {
//		return err
//	}
func ValidateTokenCookieLifetime(cookieLifetime time.Duration, ttl TokenTTL) error {
	// 未初期化 TTL は安全な lifetime として扱えないため拒否する。
	if ttl.duration <= 0 {
		return ErrInvalidTokenTTL
	}

	// cookie の寿命は正であり、かつ server-side token TTL を超えてはならない。
	if cookieLifetime <= 0 || cookieLifetime > ttl.duration {
		return ErrInvalidTokenCookieLifetime
	}

	// cookie lifetime が TTL 以下であるため成功とする。
	return nil
}

// normalizeTokenJWTPayload は JWT payload が JSON object であることを検証し、前後空白を除去する。
// payload 内の claim 名や値の意味は一切解釈しない。
func normalizeTokenJWTPayload(payload []byte) ([]byte, error) {
	// 前後空白を除去し、空 payload を拒否する。
	trimmedPayload := bytes.TrimSpace(payload)
	if len(trimmedPayload) == 0 {
		return nil, ErrInvalidTokenPayload
	}

	// JWT payload は JSON として妥当でなければならない。
	if !json.Valid(trimmedPayload) {
		return nil, ErrInvalidTokenPayload
	}

	// 上位層の claim object を想定し、array/string/number などの非 object payload は拒否する。
	if trimmedPayload[0] != '{' {
		return nil, ErrInvalidTokenPayload
	}

	// 呼び出し側の slice 変更を避けるため、正規化済み payload を複製して返す。
	return append([]byte(nil), trimmedPayload...), nil
}

// verifyTokenPrimitiveJWTHeader は JWT header が HS256 の固定 JSON であることを検証する。
// header の意味を増やさず、algorithm confusion を防ぐことだけに集中する。
func verifyTokenPrimitiveJWTHeader(headerSegment string) error {
	// header segment を Base64URL decode する。
	header, err := decodeTokenSegment(headerSegment)
	if err != nil {
		return ErrInvalidSignature
	}

	// JSON の空白差を吸収するため、最小限の header struct に decode する。
	var decodedHeader struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(header, &decodedHeader); err != nil {
		return ErrInvalidSignature
	}

	// HS256 JWT 以外は署名検証へ進めず拒否する。
	if decodedHeader.Alg != "HS256" || decodedHeader.Typ != "JWT" {
		return ErrInvalidSignature
	}

	// header が期待値を満たしたため成功とする。
	return nil
}

// encodeTokenSegment は JWT segment と hash 表現に使う Base64URL encoding を行う。
// padding を付けないことで compact token 形式に合わせる。
func encodeTokenSegment(data []byte) string {
	// RawURLEncoding は JWT compact serialization と相性のよい padding なし形式を返す。
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeTokenSegment は padding なし Base64URL segment を byte 列に戻す。
func decodeTokenSegment(segment string) ([]byte, error) {
	// RawURLEncoding は encodeTokenSegment と対になる decode 処理を提供する。
	return base64.RawURLEncoding.DecodeString(segment)
}
