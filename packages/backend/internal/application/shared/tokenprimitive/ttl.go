package application

import (
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// TTL は application 層で共有する token lifetime の中立 wrapper である。
//
// 役割:
//   - packages/backend/internal/domain の TokenTTL を直接使う箇所を application shared package に集約する。
//   - 呼び出し元の認証種別や権限体系を一切保持せず、duration の妥当性と失効時刻計算だけを公開する。
//   - Cookie lifetime 検証へ安全に渡せる検証済み TTL を保持する。
//
// 引数:
//   - 生成時の duration: 0 より大きい time.Duration。
//
// 戻り値:
//   - TTL: 検証済み domain.TokenTTL を包んだ immutable value。
//   - error: duration が 0 以下の場合は domain.ErrInvalidTokenTTL。
//
// エラーケース:
//   - domain.ErrInvalidTokenTTL: TTL が 0 以下で、token lifetime として安全に扱えない場合。
//
// 使用例:
//
//	ttl, err := tokenprimitive.ValidateTTL(15 * time.Minute)
//	if err != nil {
//		return err
//	}
//	if err := tokenprimitive.ValidateCookieLifetime(10*time.Minute, ttl); err != nil {
//		return err
//	}
type TTL struct {
	value domain.TokenTTL
}

// ValidateTTL は token lifetime が正の duration であることを検証する。
//
// 役割:
//   - 設定値や use case 入力から渡された duration を共有 application helper の TTL に変換する。
//   - 現在時刻、環境変数、呼び出し元の意味を読まず、duration の値だけを検証する。
//
// 引数:
//   - duration: token の有効期間。0 より大きい必要がある。
//
// 戻り値:
//   - TTL: Cookie lifetime 検証や失効時刻計算に使う検証済み TTL。
//   - error: duration が 0 以下の場合は domain.ErrInvalidTokenTTL。
//
// エラーケース:
//   - domain.ErrInvalidTokenTTL: duration が 0 以下の場合。
//
// 使用例:
//
//	ttl, err := ValidateTTL(15 * time.Minute)
//	if err != nil {
//		return err
//	}
func ValidateTTL(duration time.Duration) (TTL, error) {
	// domain primitive の検証を再利用し、application shared 側で別規則を作らない。
	value, err := domain.ValidateTokenTTL(duration)
	if err != nil {
		return TTL{}, err
	}

	// 検証済み domain value を wrapper に閉じ、呼び出し元へ中立 helper surface として返す。
	return TTL{value: value}, nil
}

// Duration は検証済み TTL の time.Duration 表現を返す。
//
// 役割:
//   - 保存 TTL や timer 設定へ渡せる標準 duration を公開する。
//   - 未検証の duration 生成を呼び出し元に再実装させない。
//
// 引数:
//   - なし。
//
// 戻り値:
//   - time.Duration: ValidateTTL で検証済みの正の duration。ゼロ値 TTL では 0 を返す。
//
// エラーケース:
//   - なし。生成時の error は ValidateTTL で扱う。
//
// 使用例:
//
//	duration := ttl.Duration()
func (ttl TTL) Duration() time.Duration {
	// domain value に保持されている検証済み duration だけを読み出す。
	return ttl.value.Duration()
}

// ExpiresAt は発行時刻に TTL を加算した失効時刻を UTC で返す。
//
// 役割:
//   - 呼び出し元が注入した issuedAt と検証済み TTL から deterministic に時刻を計算する。
//   - この helper 自体は time.Now を呼ばず、副作用のある clock 依存を持たない。
//
// 引数:
//   - issuedAt: token 発行時刻。任意 timezone を受け取り、結果は UTC に正規化される。
//
// 戻り値:
//   - time.Time: issuedAt.UTC() に TTL を加えた失効時刻。
//
// エラーケース:
//   - なし。ゼロ値 TTL では issuedAt.UTC() を返すため、通常は ValidateTTL の結果を使う。
//
// 使用例:
//
//	expiresAt := ttl.ExpiresAt(issuedAt)
func (ttl TTL) ExpiresAt(issuedAt time.Time) time.Time {
	// 失効時刻計算は domain primitive へ委譲し、時刻規則を二重管理しない。
	return ttl.value.ExpiresAt(issuedAt)
}

// ValidateCookieLifetime は Cookie lifetime が token TTL を超えないことを検証する。
//
// 役割:
//   - ブラウザーへ設定する Cookie の保持時間が server-side token lifetime より長くならないことを保証する。
//   - Cookie 属性の構築や transport 依存を持たず、duration の大小関係だけを扱う。
//
// 引数:
//   - cookieLifetime: Cookie に設定する保持時間。0 より大きく TTL 以下である必要がある。
//   - ttl: ValidateTTL で生成した検証済み token lifetime。
//
// 戻り値:
//   - error: 検証成功時は nil。失敗時は domain.ErrInvalidTokenTTL または domain.ErrInvalidTokenCookieLifetime。
//
// エラーケース:
//   - domain.ErrInvalidTokenTTL: ttl が未初期化または 0 以下の場合。
//   - domain.ErrInvalidTokenCookieLifetime: cookieLifetime が 0 以下、または TTL より長い場合。
//
// 使用例:
//
//	if err := ValidateCookieLifetime(10*time.Minute, ttl); err != nil {
//		return err
//	}
func ValidateCookieLifetime(cookieLifetime time.Duration, ttl TTL) error {
	// Cookie と token state の寿命比較は domain primitive に委譲し、application 側では意味を足さない。
	return domain.ValidateTokenCookieLifetime(cookieLifetime, ttl.value)
}

// ValidateDurations は token TTL と Cookie lifetime をまとめて検証する。
//
// 役割:
//   - 設定読み込み直後など、duration の pair を一度に検証したい箇所向けの小さな composition helper を提供する。
//   - token TTL 検証を先に行い、Cookie lifetime が検証済み TTL を超えないことを確認する。
//
// 引数:
//   - tokenTTL: token の server-side 有効期間。0 より大きい必要がある。
//   - cookieLifetime: Cookie に設定する保持時間。0 より大きく tokenTTL 以下である必要がある。
//
// 戻り値:
//   - TTL: 後続処理で再利用できる検証済み token lifetime。
//   - error: tokenTTL または cookieLifetime が不正な場合の domain error。
//
// エラーケース:
//   - domain.ErrInvalidTokenTTL: tokenTTL が 0 以下の場合。
//   - domain.ErrInvalidTokenCookieLifetime: cookieLifetime が 0 以下、または tokenTTL より長い場合。
//
// 使用例:
//
//	ttl, err := ValidateDurations(15*time.Minute, 10*time.Minute)
//	if err != nil {
//		return err
//	}
func ValidateDurations(tokenTTL time.Duration, cookieLifetime time.Duration) (TTL, error) {
	// token TTL を先に検証し、Cookie lifetime の比較対象を必ず検証済み value にする。
	ttl, err := ValidateTTL(tokenTTL)
	if err != nil {
		return TTL{}, err
	}

	// Cookie lifetime が token TTL を超えないことを同じ shared helper で検証する。
	if err := ValidateCookieLifetime(cookieLifetime, ttl); err != nil {
		return TTL{}, err
	}

	// 両方の duration が有効なため、後続処理で使える TTL を返す。
	return ttl, nil
}
