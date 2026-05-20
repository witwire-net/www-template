package application

import (
	"context"
	"errors"

	domain "www-template/packages/backend/internal/domain"
)

// SessionService はセッション一覧取得・特定セッション無効化・他セッション一括無効化のユースケースを提供する。
type SessionService struct {
	sessionStore SessionStore
	refreshStore RefreshTokenStore
}

// NewSessionService は SessionService を生成する。
func NewSessionService(sessionStore SessionStore, refreshStore RefreshTokenStore) *SessionService {
	return &SessionService{
		sessionStore: sessionStore,
		refreshStore: refreshStore,
	}
}

// List は認証済みアカウントの active セッション一覧を返す。
// セッションメタデータはセッションストアから取得し、自分のアカウントに紐づくものに限定する。
func (s *SessionService) List(ctx context.Context, accountID domain.AccountID) ([]SessionMetadata, error) {
	sessions, err := s.sessionStore.ListSessions(ctx, accountID)
	if err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return nil, ErrInternalError
		}
		return nil, err
	}
	return sessions, nil
}

// Revoke は指定されたセッションを無効化する。
// 所有権検証を行い、他アカウントのセッション操作を拒否する。
func (s *SessionService) Revoke(ctx context.Context, accountID domain.AccountID, sessionID string) error {
	// 所有権検証: セッションが存在し、アカウントに紐づいていることを確認する
	metadata, err := s.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return ErrInternalError
		}
		// セッションが存在しない場合も汎用エラーとして扱う（情報漏洩防止）
		return ErrBadRequest
	}
	if metadata.AccountID != accountID {
		return ErrBadRequest
	}

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

// RevokeOthers は現在のセッションを除く全セッションを無効化する。
// JWT access token の即座の失効を最優先とし、session metadata の削除を先に行い、
// 削除した session ID を使って refresh token もクリーンアップする。
func (s *SessionService) RevokeOthers(ctx context.Context, accountID domain.AccountID, currentSessionID string) error {
	// 先に session metadata を削除し、JWT access token を即座に失効させる
	// 削除した session IDs を返してもらい、refresh token クリーンアップに使用する
	deletedSessionIDs, err := s.sessionStore.RevokeOthers(ctx, accountID, currentSessionID)
	if err != nil {
		if errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return ErrInternalError
		}
		return err
	}

	// その後に refresh token をクリーンアップする
	for _, sessionID := range deletedSessionIDs {
		if err := s.refreshStore.RevokeBySessionID(ctx, accountID, sessionID); err != nil {
			if errors.Is(err, domain.ErrAuthStoreUnavailable) {
				return ErrInternalError
			}
			return err
		}
	}

	return nil
}
