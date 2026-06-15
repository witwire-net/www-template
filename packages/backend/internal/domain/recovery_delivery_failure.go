package domain

import (
	"strings"
	"time"
)

// RecoveryDeliveryFailure は復旧メール配送失敗を再試行・監査するための一時 record である。
//
// 役割:
//   - recovery token の有効期限内だけ、配送失敗の相関 ID、対象アカウント、再試行可能時刻を保持する。
//   - SMTP の raw error、メール本文、recovery URL、token secret は保持せず、deliveryStage / errorClass の安全な分類値だけを保存する。
//   - Valkey など TTL 付き一時保存層へ渡す domain object として使う。
//
// エラーケース:
//   - 値の検証は NewRecoveryDeliveryFailure で行うため、外部 package は constructor 経由で生成する。
//
// 使用例:
//
//	failure, err := domain.NewRecoveryDeliveryFailure(requestID, tokenID, accountID, email, "rcpt", "smtp_recipient_rejected", failedAt, retryAfter, expiresAt)
//	if err != nil { return err }
type RecoveryDeliveryFailure struct {
	requestID       string
	recoveryTokenID string
	accountID       AccountID
	email           string
	deliveryStage   string
	errorClass      string
	failedAt        time.Time
	retryAfter      time.Time
	expiresAt       time.Time
}

// NewRecoveryDeliveryFailure は検証済みの復旧配送失敗 record を生成する。
//
// 引数:
//   - requestID: 復旧 request の相関 ID。ULID 形式の auth ID である必要がある。
//   - recoveryTokenID: 発行済み recovery token の ID。token secret ではなく ULID 形式の ID だけを渡す。
//   - accountID: 配送対象の AccountID。
//   - email: 再配送対象のメールアドレス。空文字は拒否される。
//   - deliveryStage: `dial` / `rcpt` / `data` など安全な配送段階分類。
//   - errorClass: `smtp_recipient_rejected` など安全なエラー分類。raw error 文字列ではない。
//   - failedAt: 配送失敗時刻。
//   - retryAfter: 次回再試行可能時刻。failedAt より前は拒否される。
//   - expiresAt: record 有効期限。retryAfter より前は拒否される。
//
// 戻り値:
//   - RecoveryDeliveryFailure: 正規化済み record。
//   - error: ID、email、分類、時刻のいずれかが不正な場合の domain error。
//
// 使用例:
//
//	failure, err := domain.NewRecoveryDeliveryFailure(requestID, tokenID, accountID, email, "rcpt", "smtp_recipient_rejected", failedAt, retryAfter, expiresAt)
func NewRecoveryDeliveryFailure(requestID string, recoveryTokenID string, accountID AccountID, email string, deliveryStage string, errorClass string, failedAt time.Time, retryAfter time.Time, expiresAt time.Time) (RecoveryDeliveryFailure, error) {
	// Step 1: requestID / recoveryTokenID は相関と token retry 判断の key なので、auth ID として検証する。
	if err := ValidateAuthID(requestID); err != nil {
		return RecoveryDeliveryFailure{}, err
	}
	if err := ValidateAuthID(recoveryTokenID); err != nil {
		return RecoveryDeliveryFailure{}, err
	}

	// Step 2: AccountID と email は失敗配送の対象を再試行するために必要な値として検証する。
	if _, err := NewAccountID(accountID.String()); err != nil {
		return RecoveryDeliveryFailure{}, err
	}
	if strings.TrimSpace(email) == "" {
		return RecoveryDeliveryFailure{}, ErrInvalidOpaqueSecret
	}

	// Step 3: raw SMTP error は保存せず、安全な stage/class 分類だけを必須にする。
	normalizedStage := strings.TrimSpace(deliveryStage)
	normalizedClass := strings.TrimSpace(errorClass)
	if normalizedStage == "" || normalizedClass == "" {
		return RecoveryDeliveryFailure{}, ErrInvalidOpaqueSecret
	}

	// Step 4: retry window は過去に戻らず、record expiry が retryAfter 以前にならないことを検証する。
	if failedAt.IsZero() || retryAfter.IsZero() || expiresAt.IsZero() || retryAfter.Before(failedAt) || expiresAt.Before(retryAfter) {
		return RecoveryDeliveryFailure{}, ErrInvalidChallenge
	}

	// Step 5: 内部保持値は trim 済み UTC 時刻に正規化し、repository 実装ごとの揺れを防ぐ。
	return RecoveryDeliveryFailure{
		requestID:       requestID,
		recoveryTokenID: recoveryTokenID,
		accountID:       accountID,
		email:           strings.TrimSpace(email),
		deliveryStage:   normalizedStage,
		errorClass:      normalizedClass,
		failedAt:        failedAt.UTC(),
		retryAfter:      retryAfter.UTC(),
		expiresAt:       expiresAt.UTC(),
	}, nil
}

// RequestID は復旧 request の相関 ID を返す。
//
// 戻り値:
//   - string: constructor で検証済みの request ID。
func (r RecoveryDeliveryFailure) RequestID() string { return r.requestID }

// RecoveryTokenID は配送対象 recovery token の ID を返す。
//
// 戻り値:
//   - string: token secret を含まない recovery token ID。
func (r RecoveryDeliveryFailure) RecoveryTokenID() string { return r.recoveryTokenID }

// AccountID は配送対象アカウントの ID を返す。
//
// 戻り値:
//   - AccountID: constructor で検証済みの AccountID。
func (r RecoveryDeliveryFailure) AccountID() AccountID { return r.accountID }

// Email は配送対象メールアドレスを返す。
//
// 戻り値:
//   - string: 再配送に使うメールアドレス。ログへ直接出してはならない。
func (r RecoveryDeliveryFailure) Email() string { return r.email }

// DeliveryStage は配送失敗の処理段階分類を返す。
//
// 戻り値:
//   - string: `dial` / `rcpt` / `data` などの安全な分類値。
func (r RecoveryDeliveryFailure) DeliveryStage() string { return r.deliveryStage }

// ErrorClass は配送失敗の安全なエラー分類を返す。
//
// 戻り値:
//   - string: raw SMTP error ではなく、運用検索用の安定分類値。
func (r RecoveryDeliveryFailure) ErrorClass() string { return r.errorClass }

// FailedAt は配送失敗時刻を UTC で返す。
//
// 戻り値:
//   - time.Time: constructor で UTC 正規化された失敗時刻。
func (r RecoveryDeliveryFailure) FailedAt() time.Time { return r.failedAt }

// RetryAfter は次回再試行可能時刻を UTC で返す。
//
// 戻り値:
//   - time.Time: constructor で UTC 正規化された再試行可能時刻。
func (r RecoveryDeliveryFailure) RetryAfter() time.Time { return r.retryAfter }

// ExpiresAt は配送失敗 record の有効期限を UTC で返す。
//
// 戻り値:
//   - time.Time: constructor で UTC 正規化された有効期限。
func (r RecoveryDeliveryFailure) ExpiresAt() time.Time { return r.expiresAt }
