package operators

import "errors"

// ─── Application error ─────────────────────────────────────────────────────

var (
	// ErrOperatorInvalidInput は operator setup / creation の入力が domain rule で拒否された場合の application error である。
	ErrOperatorInvalidInput = errors.New("operator invalid input")

	// ErrOperatorForbidden は bootstrap gate、setup token、または operator 権限が拒否された場合の application error である。
	ErrOperatorForbidden = errors.New("operator forbidden")

	// ErrOperatorConflict は初回 setup 済み環境など、現在状態と要求が衝突した場合の application error である。
	ErrOperatorConflict = errors.New("operator conflict")

	// ErrOperatorInternal は repository、delivery、WebAuthn provider、ID/secret 生成の失敗を隠蔽する application error である。
	ErrOperatorInternal = errors.New("operator internal")
)
