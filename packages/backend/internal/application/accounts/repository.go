package accounts

import (
	"context"
	"errors"
	"time"

	"www-template/packages/backend/internal/application/audit"
	domain "www-template/packages/backend/internal/domain"
)

var (
	// ErrAccountRepositoryUnavailable は Admin account repository が Account root または Admin schema を安全に更新できない場合に返す error である。
	//
	// 役割:
	//   - adapter 固有の GORM/Postgres error を application 境界へ漏らさず、handler が 5xx 系の stable error に写像できるようにする。
	//   - transaction 開始失敗、permission denied、予期しない永続化 failure を同じ fail-closed 分類へ畳む。
	//   - account creation use case が retry や audit failed outcome の stable code を決めるための抽象 error として使う。
	ErrAccountRepositoryUnavailable = errors.New("admin account repository unavailable")

	// ErrAccountDuplicateEmail は canonical AccountEmail が既存 Account root と重複した場合に返す error である。
	//
	// 役割:
	//   - DB の unique constraint 詳細を公開せず、Admin API が 409 duplicate_email として扱える分類だけを表す。
	//   - email 正規化自体は domain.AccountEmail に委譲し、この error は永続化済み account との衝突だけを表す。
	//   - account creation use case が failed audit outcome を安定した code で記録するために使う。
	ErrAccountDuplicateEmail = errors.New("admin account duplicate email")

	// ErrAccountAuditNotFound は account 作成 transaction に紐づける pending audit event が存在しない場合に返す error である。
	//
	// 役割:
	//   - account mutation が監査 intent なしで進むことを防ぎ、Admin schema と Product Account root の相関を必須にする。
	//   - audit ID の形式や生成規則は repository では検証せず、存在しない correlation だけを永続化境界で拒否する。
	//   - use case が audit intent 作成漏れを内部不整合として扱えるよう、duplicate email とは別分類にする。
	ErrAccountAuditNotFound = errors.New("admin account audit not found")

	// ErrAccountSearchNotFound は Admin account read model に対象 account が存在しない場合に返す error である。
	//
	// 役割:
	//   - repository の gorm.ErrRecordNotFound を application 境界で隠し、handler が 404 stable error へ写像できるようにする。
	//   - 認可済み operator に対しても、存在しない account ID の詳細情報を漏らさない分類だけを表す。
	ErrAccountSearchNotFound = errors.New("admin account search not found")
)

// AccountRepository は Admin account creation が Product Account root と Admin audit target を同一 transaction で扱うための port である。
//
// 役割:
//   - application layer が GORM、Postgres、generated OpenAPI DTO、HTTP adapter 型に依存しないよう、domain.Account と primitive DTO だけを公開する。
//   - CreateAccountWithAuditTarget は domain.NewCreatedAccount などで構築済みの Account root を保存し、同じ DB transaction 内で admin.audit_events の target と success outcome を関連付ける。
//   - repository 実装は public.accounts / public.account_settings と Admin-owned schema の audit table を同一 commit 境界に置く。
//
// 使用例:
//
//	created, err := accounts.CreateAccountWithAuditTarget(ctx, AccountCreationRecord{Account: account, AuditID: audit.AuditID, AuditCompletion: completion})
//	if err != nil {
//		return err
//	}
//	_ = created
type AccountRepository interface {
	CreateAccountWithAuditTarget(ctx context.Context, record AccountCreationRecord) (AccountRecord, error)
}

// AccountSearchRepository は Admin account search use case が検証済み query だけで Product Account read model を取得するための port である。
//
// 役割:
//   - application layer が GORM や SQL 文字列を所有せず、pagination と入力検証を repository 実行前に完了させる。
//   - SearchAccounts は検証済み limit/email/cursor だけを受け取り、adapter 側で parameter binding を使って検索する。
//   - repository 実装が返す read model は Product Account の要約に限定し、Admin auth/session 情報を混ぜない。
//
// 使用例:
//
//	result, err := accounts.SearchAccounts(ctx, AccountSearchQuery{Limit: 25})
//	if err != nil {
//		return err
//	}
//	_ = result
type AccountSearchRepository interface {
	SearchAccounts(ctx context.Context, query AccountSearchQuery) (AccountSearchRepositoryResult, error)
	FindAccountByID(ctx context.Context, accountID string) (AccountSummaryRecord, error)
}

// AccountCreationRecord は account 作成 transaction へ渡す application DTO である。
//
// 役割:
//   - Account は concrete domain object として受け取り、repository 側に email 正規化や lifecycle 初期値決定を置かない。
//   - AuditID は mutation 前に作成済みの pending Admin audit event を指し、作成 account と audit target を同じ transaction で結び付ける。
//   - AuditCompletion は audit.AuditService が domain.AdminAuditEvent から作った success outcome で、account 作成 commit と同じ transaction で保存する。
//   - adapter 型や generated 型を含めず、port purity と Clean Architecture の境界を守る。
type AccountCreationRecord struct {
	Account         domain.Account
	AuditID         string
	AuditCompletion audit.CompletionRecord
}

// AccountRecord は Admin account repository が永続化後に返す Account root snapshot である。
//
// 役割:
//   - HTTP response や後続 use case が必要とする primitive だけを保持し、GORM record や DB column tag を application 境界へ出さない。
//   - Email、Status、Locale は domain object が検証した canonical 文字列であり、repository は値を再解釈しない。
//   - SessionRevokedAfter は nil の場合に revoke 境界なしを表し、非 nil の場合は repository から独立した時刻値として扱う。
type AccountRecord struct {
	AccountID           string
	Email               string
	Status              string
	Locale              string
	SessionRevokedAfter *time.Time
	CreatedAt           time.Time
}

// AccountSearchQuery は repository へ渡す検証済み account search query である。
//
// 役割:
//   - Limit は 1〜100 の範囲に正規化済みであり、repository は範囲外値の分岐を持たない。
//   - Email と Cursor は application で長さ・空白を整えた primitive 値であり、SQL 文字列へ連結せず parameter として使う。
//   - RequestID は repository へ渡さず、検索条件と永続化 query の責務を分離する。
type AccountSearchQuery struct {
	Email  string
	Cursor string
	Limit  int32
}

// AccountSummaryRecord は Admin account search repository が返す account 要約 read model である。
//
// 役割:
//   - Product Account の識別子、email、status、作成日時、passkey 数だけを application 境界へ返す。
//   - GORM tag や SQL column 名を持たず、HTTP response DTO への変換材料に限定する。
//   - Account domain object の mutation API ではなく read model なので、検索結果表示に必要な snapshot だけを保持する。
type AccountSummaryRecord struct {
	AccountID    string
	Email        string
	Status       string
	PasskeyCount int32
	CreatedAt    time.Time
}

// AccountSearchRepositoryResult は repository search の結果 DTO である。
//
// 役割:
//   - Accounts は query に一致した要約一覧で、Limit を超えない件数だけを返す。
//   - NextCursor は次ページが存在する場合だけ設定し、handler は値の意味を解釈しない。
//   - RequestID など transport correlation は application service が付与するため repository result には含めない。
type AccountSearchRepositoryResult struct {
	Accounts   []AccountSummaryRecord
	NextCursor string
}
