package application

import "errors"

var (
	// ErrAdminAuthUnauthenticated は Admin Operator 認証情報が存在しない、または検証できない場合に返す application error である。
	// HTTP adapter は詳細な token/session 理由を外へ漏らさず、401 相当の応答へ変換する。
	ErrAdminAuthUnauthenticated = errors.New("admin auth unauthenticated")

	// ErrAdminAuthForbidden は Admin Operator が対象 mutation を実行する権限を持たない場合に返す application error である。
	// inactive、viewer、passkey 未登録、CSRF mismatch などの拒否を fail-closed に扱うために使う。
	ErrAdminAuthForbidden = errors.New("admin auth forbidden")

	// ErrAdminAuthInternal は Admin auth 境界の port、signer、ID 生成、設定不備などが失敗した場合に返す application error である。
	// secret や refreshToken 平文を error text に含めないため、外部へは安定した抽象 error だけを返す。
	ErrAdminAuthInternal = errors.New("admin auth internal")

	// ErrAdminAuthBadRequest は Admin auth 入力が use case を開始できない形の場合に返す application error である。
	// passkey challenge や cookie selector の形式不備を、詳細を露出しない request error として扱う。
	ErrAdminAuthBadRequest = errors.New("admin auth bad request")

	// ErrAdminAuthPasskeyNotFound は Operator 自身の passkey credential が見つからない場合に返す application error である。
	// credential ID の存在有無を詳細に漏らさず、HTTP adapter では安定した request error へ変換する。
	ErrAdminAuthPasskeyNotFound = errors.New("admin auth passkey not found")

	// ErrAdminAuthLastPasskey は Operator の最後の passkey credential を削除しようとした場合に返す application error である。
	// Admin auth domain の最後の認証手段保護を HTTP adapter が 409 conflict へ写像するために使う。
	ErrAdminAuthLastPasskey = errors.New("admin auth last passkey")
)
