package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrTokenExpired はアクセストークンの有効期限が切れている場合に返すエラー。
	// 保護されたエンドポイントで期限切れトークンが提示された場合、
	// session-expired 分類として扱うための識別子となる。
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidSignature はJWTの署名検証に失敗した場合に返すエラー。
	// 改竄されたトークンや異なるシークレットで署名されたトークンが検出された場合に使用する。
	ErrInvalidSignature = errors.New("invalid token signature")

	// ErrMalformedToken はJWTのフォーマットが不正な場合に返すエラー。
	// 3つのセグメント（header.payload.signature）に分割できない場合に使用する。
	ErrMalformedToken = errors.New("malformed token")

	// ErrInvalidSecret はJWT署名用シークレットが空または不適切な場合に返すエラー。
	// 空のシークレットで署名されたトークンはセキュリティ上のリスクとなるため、生成・検証を拒否する。
	ErrInvalidSecret = errors.New("invalid jwt secret")
)

// AccessTokenTTL はJWTアクセストークンの有効期限のデフォルト値。
// 15分に設定しており、セキュリティと使い勝手のバランスを取る。
const AccessTokenTTL = 15 * time.Minute

// jwtHeader はJWTヘッダーの固定値。HS256のみを使用する。
var jwtHeader = []byte(`{"alg":"HS256","typ":"JWT"}`)

// Claims はJWTアクセストークンのペイロードに含まれるクレームを表現する構造体。
// 最小限の情報（アカウントID、セッションID、トークンID、発行時刻、有効期限）のみを保持し、
// トークンサイズを小さく保つことでHTTPヘッダー転送のオーバーヘッドを低減する。
type Claims struct {
	// Subject はアカウントを一意に識別するULID。JWT標準クレーム sub に対応する。
	Subject string `json:"sub"`
	// SessionID はセッションを一意に識別するULID。カスタムクレーム sid に対応する。
	SessionID string `json:"sid"`
	// ID はトークン自体を一意に識別するULID。JWT標準クレーム jti に対応する。
	ID string `json:"jti"`
	// IssuedAt はトークンの発行時刻をUnixタイムスタンプ（秒）で表す。JWT標準クレーム iat に対応する。
	IssuedAt int64 `json:"iat"`
	// ExpiresAt はトークンの有効期限をUnixタイムスタンプ（秒）で表す。JWT標準クレーム exp に対応する。
	ExpiresAt int64 `json:"exp"`
}

// NewClaims は指定されたアカウントID、セッションID、トークンIDからJWTクレームを生成する。
// 発行時刻（iat）は現在時刻、有効期限（exp）は現在時刻から15分後に設定する。
// accountID、sessionID、tokenID はすべて有効なULID形式であることを呼び出し側が保証しなければならない。
func NewClaims(accountID, sessionID, tokenID string, now time.Time) Claims {
	return Claims{
		Subject:   accountID,                      // sub: アカウントULID
		SessionID: sessionID,                      // sid: セッションULID
		ID:        tokenID,                        // jti: トークンULID
		IssuedAt:  now.Unix(),                     // iat: 発行時刻
		ExpiresAt: now.Add(AccessTokenTTL).Unix(), // exp: 有効期限
	}
}

// SignAccessToken は指定されたクレームをHS256アルゴリズムで署名し、JWT文字列を生成する。
// secret は32バイト以上の長さを持つことを推奨する。短すぎるシークレットはセキュリティ上のリスクとなる。
// 空のシークレットは受け付けず、ErrInvalidSecret を返して fail-close とする。
// 生成されたトークンは Authorization: Bearer <token> ヘッダーとして使用される。
func SignAccessToken(claims Claims, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", ErrInvalidSecret
	}

	header := base64URLEncode(jwtHeader)

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	payload := base64URLEncode(payloadBytes)

	signingInput := header + "." + payload
	signature := signHS256(signingInput, secret)

	return signingInput + "." + signature, nil
}

// VerifyAccessToken はJWT文字列の署名を検証し、クレームを抽出して返す。
// 空のシークレットは受け付けず、ErrInvalidSecret を返す。
// 署名が一致しない場合、alg が HS256 でない場合、必須クレームが欠落している場合は ErrInvalidSignature を返す。
// 有効期限が切れている（now >= exp）場合は ErrTokenExpired を返す。
// いずれのエラーも発生した場合、保護されたエンドポイントで拒否されるべきである。
// now は外部から注入された現在時刻であり、domain 層で time.Now() を直接呼ばない設計としている。
func VerifyAccessToken(tokenString string, secret []byte, now time.Time) (Claims, error) {
	if len(secret) == 0 {
		return Claims{}, ErrInvalidSecret
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformedToken
	}

	// Step 1: ヘッダー内の alg を検証し、algorithm confusion attack を防ぐ
	if err := verifyJWTHeader(parts[0]); err != nil {
		return Claims{}, err
	}

	// Step 2: 署名を検証する
	if err := verifyJWTSignature(parts[0]+"."+parts[1], parts[2], secret); err != nil {
		return Claims{}, err
	}

	// Step 3: クレームをデコードし、必須フィールドと有効期限を検証する
	claims, err := decodeAndValidateClaims(parts[1], now)
	if err != nil {
		return Claims{}, err
	}

	return claims, nil
}

// verifyJWTHeader はJWTヘッダーのBase64URLデコードとalg検証を行う。
// HS256以外のアルゴリズムが検出された場合、algorithm confusion attack を防ぐためエラーを返す。
func verifyJWTHeader(headerB64 string) error {
	headerBytes, err := base64URLDecode(headerB64)
	if err != nil {
		return ErrInvalidSignature
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return ErrInvalidSignature
	}
	if header.Alg != "HS256" {
		return ErrInvalidSignature
	}
	return nil
}

// verifyJWTSignature は signingInput と Base64URL署名を secret で検証する。
func verifyJWTSignature(signingInput, signatureB64 string, secret []byte) error {
	expectedSig, err := base64URLDecode(signatureB64)
	if err != nil {
		return ErrInvalidSignature
	}
	actualSig := hmacSHA256(signingInput, secret)
	if !hmac.Equal(expectedSig, actualSig) {
		return ErrInvalidSignature
	}
	return nil
}

// decodeAndValidateClaims はペイロードをデコードし、必須クレームの存在と有効期限を検証する。
// 仕様により access token は accountID(sub)・sessionID(sid)・tokenID(jti)・iat・exp を含まなければならない。
func decodeAndValidateClaims(payloadB64 string, now time.Time) (Claims, error) {
	payloadBytes, err := base64URLDecode(payloadB64)
	if err != nil {
		return Claims{}, ErrInvalidSignature
	}
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, ErrInvalidSignature
	}
	if claims.Subject == "" || claims.SessionID == "" || claims.ID == "" || claims.IssuedAt == 0 || claims.ExpiresAt == 0 {
		return Claims{}, ErrInvalidSignature
	}
	// JWT の exp は「その時刻を含めて無効」と解釈するのが標準。
	// したがって now >= exp の時点で拒否する（!Before で判定）。
	if !now.UTC().Before(time.Unix(claims.ExpiresAt, 0).UTC()) {
		return Claims{}, ErrTokenExpired
	}
	return claims, nil
}

// base64URLEncode はBase64URLエンコーディング（paddingなし）でデータをエンコードする。
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode はBase64URLエンコーディング（paddingなし）の文字列をデコードする。
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// hmacSHA256 は指定されたデータとシークレットを用いてHMAC-SHA256署名を生成する。
func hmacSHA256(data string, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	// Write は常に nil を返すためエラー処理は不要
	_, _ = h.Write([]byte(data))
	return h.Sum(nil)
}

// signHS256 は指定された signingInput に対してHS256署名を生成し、Base64URLエンコードして返す。
func signHS256(signingInput string, secret []byte) string {
	return base64URLEncode(hmacSHA256(signingInput, secret))
}
