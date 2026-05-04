package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"

	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

var (
	// ErrRefreshTokenNotFound は指定されたリフレッシュトークンが存在しない場合に返すエラー。
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// ErrRefreshTokenConsumed はリフレッシュトークンが既に消費されている場合に返すエラー。
	ErrRefreshTokenConsumed = errors.New("refresh token already consumed")
	// ErrTokenTheftDetected はリフレッシュトークンの再利用を検出した場合に返すエラー。
	ErrTokenTheftDetected = errors.New("token theft detected")
)

// TokenService はアクセストークン（JWT）とリフレッシュトークンの発行・ローテーション・盗難検出を担当する。
type TokenService struct {
	refreshStore RefreshTokenStore
	sessionStore SessionStore
	config       config.AuthConfig
	clock        func() time.Time
	policy       id.AuthIDPolicy
}

// NewTokenService は TokenService を生成する。
// clock と policy は必須。nil の場合 panic する。
func NewTokenService(refreshStore RefreshTokenStore, sessionStore SessionStore, cfg config.AuthConfig, clock func() time.Time, policy id.AuthIDPolicy) *TokenService {
	if clock == nil {
		panic("clock is required")
	}
	return &TokenService{
		refreshStore: refreshStore,
		sessionStore: sessionStore,
		config:       cfg,
		clock:        clock,
		policy:       policy,
	}
}

// Issue は指定されたアカウントとデバイス指紋に対して新しい JWT アクセストークンとリフレッシュトークンを発行する。
// アクセストークンの有効期限は15分、リフレッシュトークンの有効期限は設定された TTL（未設定時は無期限）とする。
// deviceName は User-Agent 由来のデバイス表示名、ipHash は SHA-256 化されたクライアント IP である。
// existingSessionID が空でない場合はそのセッション ID を継続し、空の場合は新規生成する。
// 発行されたセッション ID も返す。
func (s *TokenService) Issue(ctx context.Context, accountID, fingerprint, deviceName, ipHash, existingSessionID string) (accessToken, refreshToken, sessionID string, err error) {
	if existingSessionID != "" {
		sessionID = existingSessionID
	} else {
		sessionID, err = s.policy.Next()
		if err != nil {
			return "", "", "", ErrInternalError
		}
	}
	tokenID, err := s.policy.Next()
	if err != nil {
		return "", "", "", ErrInternalError
	}

	claims := domain.NewClaims(accountID, sessionID, tokenID, s.clock())
	accessToken, err = domain.SignAccessToken(claims, []byte(s.config.JWTSecret))
	if err != nil {
		return "", "", "", ErrInternalError
	}

	refreshToken, err = generateOpaqueToken()
	if err != nil {
		return "", "", "", err
	}
	hash := hashToken(refreshToken)

	record := RefreshTokenRecord{
		AccountID:   accountID,
		SessionID:   sessionID,
		Fingerprint: fingerprint,
		DeviceName:  deviceName,
		IPHash:      ipHash,
		IssuedAt:    s.clock(),
	}

	ttl := s.config.RefreshTokenTTL
	if err := s.refreshStore.Save(ctx, hash, record, ttl); err != nil {
		return "", "", "", ErrInternalError
	}

	// セッションメタデータを保存・更新する（失効判定用）
	// existingSessionID が空でない場合は GetSession を必須とし、AccountID 一致も検証する
	loginAt := s.clock()
	if existingSessionID != "" {
		existing, getErr := s.sessionStore.GetSession(ctx, existingSessionID)
		if getErr != nil {
			if errors.Is(getErr, domain.ErrSessionNotFound) {
				return "", "", "", ErrSessionExpired
			}
			return "", "", "", ErrInternalError
		}
		if existing.AccountID != accountID {
			return "", "", "", ErrBadRequest
		}
		loginAt = existing.LoginAt
	}
	metadata := SessionMetadata{
		SessionID:    sessionID,
		AccountID:    accountID,
		DeviceName:   deviceName,
		LoginAt:      loginAt,
		LastActiveAt: s.clock(),
		IPHash:       ipHash,
	}
	if err := s.sessionStore.SaveSession(ctx, sessionID, accountID, metadata, ttl); err != nil {
		return "", "", "", ErrInternalError
	}

	return accessToken, refreshToken, sessionID, nil
}

// Refresh はリフレッシュトークンを消費し、新しいアクセストークンとリフレッシュトークンのペアを返す。
// clientIP と userAgent から現在の fingerprint を算出し、保存済み fingerprint と constant-time 比較する。
// 不一致の場合は同一 fingerprint family の全トークンを失効させて ErrTokenTheftDetected を返す。
// セッション ID は継続し、LastActiveAt のみ更新する。
func (s *TokenService) Refresh(ctx context.Context, refreshToken, clientIP, userAgent string) (newAccessToken, newRefreshToken string, err error) {
	hash := hashToken(refreshToken)
	record, err := s.refreshStore.Consume(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return "", "", ErrInternalError
		}
		// 消費済みトークンの再利用かどうかを確認する
		consumed, consumedErr := s.refreshStore.GetConsumed(ctx, hash)
		if consumedErr == nil {
			// 盗難検出: 先に session metadata を削除し JWT を失効させてから、同一 fingerprint family を全失効させる
			if revokeErr := s.sessionStore.RevokeSession(ctx, consumed.AccountID, consumed.SessionID); revokeErr != nil {
				return "", "", ErrInternalError
			}
			if revokeErr := s.refreshStore.RevokeAllForFingerprint(ctx, consumed.AccountID, consumed.Fingerprint); revokeErr != nil {
				return "", "", ErrInternalError
			}
			return "", "", ErrTokenTheftDetected
		}
		if errors.Is(consumedErr, domain.ErrSessionNotFound) {
			return "", "", ErrRefreshTokenNotFound
		}
		// store unavailable や decode failure など、いかなる未知エラーも fail-closed
		return "", "", ErrInternalError
	}

	// 現在リクエストの fingerprint を算出し、保存済みと比較する
	currentFP := hmacString(userAgent+"|"+clientIP, s.config.SecretHashKey)
	if subtle.ConstantTimeCompare([]byte(record.Fingerprint), []byte(currentFP)) != 1 {
		// fingerprint 不一致: 盗難とみなし、先に session metadata を削除してから同一 family を全失効する
		if revokeErr := s.sessionStore.RevokeSession(ctx, record.AccountID, record.SessionID); revokeErr != nil {
			return "", "", ErrInternalError
		}
		if revokeErr := s.refreshStore.RevokeAllForFingerprint(ctx, record.AccountID, record.Fingerprint); revokeErr != nil {
			return "", "", ErrInternalError
		}
		return "", "", ErrTokenTheftDetected
	}

	// 新しいペアを発行する（セッション ID は継続し、デバイス情報も引き継ぐ）
	newAccessToken, newRefreshToken, _, err = s.Issue(ctx, record.AccountID, record.Fingerprint, record.DeviceName, record.IPHash, record.SessionID)
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

// RevokeSession は指定されたセッションを失効させ、関連するリフレッシュトークンも削除する。
func (s *TokenService) RevokeSession(ctx context.Context, accountID, sessionID string) error {
	if err := s.sessionStore.RevokeSession(ctx, accountID, sessionID); err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return ErrInternalError
		}
		return err
	}
	if err := s.refreshStore.RevokeBySessionID(ctx, accountID, sessionID); err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return ErrInternalError
		}
		return err
	}
	return nil
}

// VerifyAccessToken は JWT アクセストークンの署名と有効期限を検証し、TokenClaims を返す。
func (s *TokenService) VerifyAccessToken(token string) (TokenClaims, error) {
	claims, err := domain.VerifyAccessToken(token, []byte(s.config.JWTSecret), s.clock())
	if err != nil {
		return TokenClaims{}, err
	}
	return TokenClaims{
		AccountID: claims.Subject,
		SessionID: claims.SessionID,
		TokenID:   claims.ID,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

// generateOpaqueToken は暗号学的に安全な 64 バイトのランダムトークンを生成する。
// rand.Read の失敗は内部エラーとして返し、fail-open しない。
func generateOpaqueToken() (string, error) {
	b := make([]byte, 64)
	if _, err := tokenRand.Read(b); err != nil {
		return "", ErrInternalError
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken はトークン文字列の SHA-256 ハッシュを base64 エンコードして返す。
// 平文トークンを Valkey キーに直接使用しないことで、キー空間の予測を困難にする。
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// tokenRand は TokenService 専用の乱数読み取りインターフェース。
// テスト時にモックに置き換えることを許容するためパッケージ変数として保持する。
var tokenRand tokenRandReader = defaultTokenRand{}

type tokenRandReader interface {
	Read(p []byte) (n int, err error)
}

type defaultTokenRand struct{}

func (defaultTokenRand) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}
