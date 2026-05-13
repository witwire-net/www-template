package app

import (
	"errors"
	"time"

	"www-template/packages/backend/internal/platform/config"
)

// validateAuthConfig は認証設定に対して fail-close バリデーションを実施する。
// platform/config での共通検証を補完し、auth 固有の追加ルールを適用する。
// refresh_token_ttl が設定されている場合、最低 24 時間以上を強制する。
// 負値・ゼロより小さい値・24 時間未満の値は運用ミスを防ぐため、起動時に拒否する。
func validateAuthConfig(cfg config.AuthConfig) error {
	if cfg.RefreshTokenTTL != 0 && cfg.RefreshTokenTTL < 24*time.Hour {
		return errors.New("auth.refresh_token_ttl must be at least 24h when set")
	}
	return nil
}
