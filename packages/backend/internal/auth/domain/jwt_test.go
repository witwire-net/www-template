package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// [UT-AUTH-BE-HAP-001] JWT signing and verification
func TestSignAndVerifyAccessToken(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	verified, err := VerifyAccessToken(token, secret, now)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if verified.Subject != claims.Subject {
		t.Errorf("subject mismatch: got %s, want %s", verified.Subject, claims.Subject)
	}
	if verified.SessionID != claims.SessionID {
		t.Errorf("sessionID mismatch: got %s, want %s", verified.SessionID, claims.SessionID)
	}
	if verified.ID != claims.ID {
		t.Errorf("id mismatch: got %s, want %s", verified.ID, claims.ID)
	}
	if verified.IssuedAt != claims.IssuedAt {
		t.Errorf("issuedAt mismatch: got %d, want %d", verified.IssuedAt, claims.IssuedAt)
	}
	if verified.ExpiresAt != claims.ExpiresAt {
		t.Errorf("expiresAt mismatch: got %d, want %d", verified.ExpiresAt, claims.ExpiresAt)
	}
}

// [UT-AUTH-BE-ERR-001] Expired JWT verification fails
func TestVerifyAccessTokenExpired(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	// 過去の時刻で発行し、期限切れのトークンを生成する
	past := time.Now().UTC().Add(-20 * time.Minute)
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", past)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	_, err = VerifyAccessToken(token, secret, time.Now().UTC())
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Invalid signature verification fails
func TestVerifyAccessTokenInvalidSignature(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	wrongSecret := []byte("different-secret-key-that-does-not-match-at-all-!!")
	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	_, err = VerifyAccessToken(token, wrongSecret, now)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Tampered token verification fails
func TestVerifyAccessTokenTampered(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	// ペイロード部分の1文字を変更して改竄をシミュレートする
	// これにより署名が完全に不一致になる
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal("expected 3 parts")
	}
	// payload の先頭文字を変更（Base64URL デコード後の JSON が破損し、さらに署名も不一致になる）
	tamperedPayload := "X" + parts[1][1:]
	tampered := parts[0] + "." + tamperedPayload + "." + parts[2]

	_, err = VerifyAccessToken(tampered, secret, now)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for tampered token, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Wrong algorithm header verification fails
func TestVerifyAccessTokenWrongAlgorithm(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	// alg を HS384 に書き換えたヘッダーを作成する
	maliciousHeader := base64URLEncode([]byte(`{"alg":"HS384","typ":"JWT"}`))
	parts := strings.Split(token, ".")
	maliciousToken := maliciousHeader + "." + parts[1] + "." + parts[2]

	_, err = VerifyAccessToken(maliciousToken, secret, now)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for wrong algorithm, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Empty secret is rejected
func TestSignAccessTokenRejectsEmptySecret(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)
	_, err := SignAccessToken(claims, []byte{})
	if !errors.Is(err, ErrInvalidSecret) {
		t.Fatalf("expected ErrInvalidSecret for empty secret, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Empty secret verification is rejected
func TestVerifyAccessTokenRejectsEmptySecret(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	_, err = VerifyAccessToken(token, []byte{}, now)
	if !errors.Is(err, ErrInvalidSecret) {
		t.Fatalf("expected ErrInvalidSecret for empty secret, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Token is rejected exactly at expiration boundary
func TestVerifyAccessTokenRejectsAtExactExpiry(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	now := time.Now().UTC()
	claims := NewClaims("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", now)

	token, err := SignAccessToken(claims, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	// exp の境界値: ちょうど exp の時刻でも拒否されることを確認する
	exactExpiry := time.Unix(claims.ExpiresAt, 0).UTC()
	_, err = VerifyAccessToken(token, secret, exactExpiry)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired at exact expiry boundary, got: %v", err)
	}
}

// [UT-AUTH-BE-ERR-001] Missing mandatory claims are rejected
func TestVerifyAccessTokenMissingClaims(t *testing.T) {
	t.Parallel()

	secret := []byte("a-very-long-secret-key-for-hs256-jwt-signing-32bytes-min")
	now := time.Now().UTC()

	cases := []struct {
		name   string
		claims Claims
	}{
		{"missing subject", Claims{SessionID: "01ARZ3NDEKTSV4RRFFQ69G5FAW", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAX", IssuedAt: now.Unix(), ExpiresAt: now.Add(AccessTokenTTL).Unix()}},
		{"missing sessionID", Claims{Subject: "01ARZ3NDEKTSV4RRFFQ69G5FAV", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAX", IssuedAt: now.Unix(), ExpiresAt: now.Add(AccessTokenTTL).Unix()}},
		{"missing id", Claims{Subject: "01ARZ3NDEKTSV4RRFFQ69G5FAV", SessionID: "01ARZ3NDEKTSV4RRFFQ69G5FAW", IssuedAt: now.Unix(), ExpiresAt: now.Add(AccessTokenTTL).Unix()}},
		{"missing issuedAt", Claims{Subject: "01ARZ3NDEKTSV4RRFFQ69G5FAV", SessionID: "01ARZ3NDEKTSV4RRFFQ69G5FAW", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAX", ExpiresAt: now.Add(AccessTokenTTL).Unix()}},
		{"missing expiresAt", Claims{Subject: "01ARZ3NDEKTSV4RRFFQ69G5FAV", SessionID: "01ARZ3NDEKTSV4RRFFQ69G5FAW", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAX", IssuedAt: now.Unix()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			token, err := SignAccessToken(tc.claims, secret)
			if err != nil {
				t.Fatalf("sign failed: %v", err)
			}
			_, err = VerifyAccessToken(token, secret, now)
			if !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("expected ErrInvalidSignature for %s, got: %v", tc.name, err)
			}
		})
	}
}
