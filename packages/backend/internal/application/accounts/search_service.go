package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

const (
	adminAccountSearchDefaultLimit   int32 = 25
	adminAccountSearchMinLimit       int32 = 1
	adminAccountSearchMaxLimit       int32 = 100
	adminAccountSearchMaxEmailLength       = 255
)

var (
	// ErrAccountSearchInvalidInput は Admin account search query が pagination または email 検証に失敗した場合に返す application error である。
	//
	// 役割:
	//   - handler が 400 stable validation error へ写像できる分類だけを表す。
	//   - repository query を実行する前に返され、範囲外 limit や長すぎる email search を DB へ渡さない。
	//   - 入力値そのものや内部検証理由を response body へ出さないための境界 error として使う。
	ErrAccountSearchInvalidInput = errors.New("admin account search invalid input")

	// ErrAccountSearchInternal は Admin account search の依存欠落または永続化 failure を表す application error である。
	//
	// 役割:
	//   - repository availability や context cancellation を 5xx 系へ畳む。
	//   - DB error の詳細を transport 境界へ漏らさず、Admin API を fail-closed にする。
	//   - search use case constructor と repository call の両方で共通の内部 failure 分類として使う。
	ErrAccountSearchInternal = errors.New("admin account search internal")
)

// AccountSearchService は Admin account search の pagination/input validation と repository 呼び出しを担う application use case である。
//
// 役割:
//   - `limit` を 1〜100 に限定し、範囲外値では repository query を実行しない。
//   - email search string の最大長を backend 側で検証し、SQL adapter へ未検証 input を渡さない。
//   - handler へ返す read model は primitive DTO に限定し、GORM/generated/domain mutation 型を公開しない。
//
// 使用例:
//
//	service, err := NewAccountSearchService(accounts)
//	if err != nil {
//		return err
//	}
//	result, err := service.SearchAccounts(ctx, AccountSearchInput{Limit: int32Ptr(25)})
//	_ = result
type AccountSearchService struct {
	accounts AccountSearchRepository
}

// AccountSearchInput は Admin account search use case の入力 DTO である。
//
// 役割:
//   - HTTP query parameter 由来の optional 値を generated 型に依存せず application 境界へ運ぶ。
//   - Limit は nil の場合だけ default limit を使い、非 nil の 0 や負数は invalid pagination として拒否する。
//   - RequestID は response correlation 用で、repository query の条件には使わない。
type AccountSearchInput struct {
	Email     string
	Cursor    string
	Limit     *int32
	RequestID string
}

// AccountDetailInput は Admin account detail use case の入力 DTO である。
//
// 役割:
//   - path parameter 由来の account ID と response correlation 用 request ID だけを保持する。
//   - generated OpenAPI 型を application 層へ持ち込まず、handler 境界で primitive に変換済みの値を受ける。
type AccountDetailInput struct {
	AccountID string
	RequestID string
}

// AccountSearchResult は Admin account search use case の成功結果 DTO である。
//
// 役割:
//   - Accounts は repository read model を transport 変換しやすい primitive snapshot として保持する。
//   - NextCursor は次ページが存在する場合だけ非空になり、handler は opaque value として返す。
//   - RequestID は handler が stable response body に含める correlation ID である。
type AccountSearchResult struct {
	Accounts   []AccountSummary
	NextCursor string
	RequestID  string
}

// AccountDetailResult は Admin account detail use case の成功結果 DTO である。
//
// 役割:
//   - 一件分の account read model と request correlation をまとめ、HTTP adapter が OpenAPI DTO へ詰め替えるだけにする。
type AccountDetailResult struct {
	Account   AccountSummary
	RequestID string
}

// AccountSummary は Admin account search/detail response 用の account 要約 DTO である。
//
// 役割:
//   - Product Account の表示に必要な ID/email/status/passkey count/createdAt だけを含める。
//   - domain.Account の mutation method や repository record を外へ出さず、read model として扱う。
//   - handler はこの DTO を Admin OpenAPI の AccountSummary へ機械的に詰め替える。
type AccountSummary struct {
	AccountID    string
	Email        string
	Status       string
	PasskeyCount int32
	CreatedAt    time.Time
}

// NewAccountSearchService は Admin account search use case を生成する。
//
// 引数:
//   - accounts: 検証済み search query を実行する repository port。
//
// 戻り値:
//   - *AccountSearchService: pagination validation と repository 呼び出しを行う use case。
//   - error: repository port が nil の場合は ErrAccountSearchInternal。
func NewAccountSearchService(accounts AccountSearchRepository) (*AccountSearchService, error) {
	// Step 1: repository 未接続で read route を公開しないよう、constructor の時点で必須依存を拒否する。
	if accounts == nil {
		return nil, ErrAccountSearchInternal
	}

	// Step 2: 検証済み依存だけを保持し、handler/runtime composition から再利用できる service として返す。
	return &AccountSearchService{accounts: accounts}, nil
}

// SearchAccounts は Admin account search query を検証し、妥当な場合だけ repository query を実行する。
//
// ctx は repository query へ deadline/cancellation を伝播する。
// input は raw query parameter と request correlation を含む primitive DTO である。
// 成功時は account 要約一覧と request ID を返し、失敗時は invalid/internal の application error を返す。
func (s *AccountSearchService) SearchAccounts(ctx context.Context, input AccountSearchInput) (AccountSearchResult, error) {
	// Step 1: cancellation 済み request では validation 後の DB query を開始せず、内部 failure として fail-closed にする。
	if err := ctx.Err(); err != nil {
		return AccountSearchResult{}, ErrAccountSearchInternal
	}

	// Step 2: pagination/email/cursor を repository 実行前に検証し、範囲外 limit では query を一切発行しない。
	query, err := validatedAccountSearchQuery(input)
	if err != nil {
		return AccountSearchResult{}, err
	}

	// Step 3: repository には検証済み DTO だけを渡し、SQL parameter binding の材料を application 側で固定する。
	repositoryResult, err := s.accounts.SearchAccounts(ctx, query)
	if err != nil {
		return AccountSearchResult{}, ErrAccountSearchInternal
	}

	// Step 4: repository read model を response 用 DTO に詰め替え、request correlation は application service が付与する。
	return adminAccountSearchResult(repositoryResult, input.RequestID), nil
}

// GetAccount は Admin account detail 用の read model を 1 件取得する。
//
// ctx は repository query へ deadline/cancellation を伝播する。
// input.AccountID は generated route binding が受けた account ID で、空値は repository に渡さず invalid とする。
// 成功時は account 要約と request ID を返し、対象不在は ErrAccountSearchNotFound を返す。
func (s *AccountSearchService) GetAccount(ctx context.Context, input AccountDetailInput) (AccountDetailResult, error) {
	// Step 1: cancellation 済み request では DB query を開始せず、内部 failure として fail-closed にする。
	if err := ctx.Err(); err != nil {
		return AccountDetailResult{}, ErrAccountSearchInternal
	}

	// Step 2: path parameter を Product AccountID として検証し、空文字以外の不正 ID も repository へ渡さない。
	accountID := strings.TrimSpace(input.AccountID)
	canonicalAccountID, err := domain.NewAccountID(accountID)
	if err != nil {
		return AccountDetailResult{}, ErrAccountSearchInvalidInput
	}

	// Step 3: repository は許可済み admin_view read model だけを読むため、application は ID 条件だけを渡す。
	record, err := s.accounts.FindAccountByID(ctx, canonicalAccountID.String())
	if err != nil {
		if errors.Is(err, ErrAccountSearchNotFound) {
			return AccountDetailResult{}, ErrAccountSearchNotFound
		}
		return AccountDetailResult{}, ErrAccountSearchInternal
	}

	// Step 4: 一覧と同じ response DTO へ変換し、detail でも表示形式を揃える。
	return AccountDetailResult{Account: adminAccountSummary(record), RequestID: input.RequestID}, nil
}

func validatedAccountSearchQuery(input AccountSearchInput) (AccountSearchQuery, error) {
	// Step 1: optional limit は未指定時だけ default を使い、指定済みの 0 は明確な invalid pagination として拒否する。
	limit := adminAccountSearchDefaultLimit
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit < adminAccountSearchMinLimit || limit > adminAccountSearchMaxLimit {
		return AccountSearchQuery{}, ErrAccountSearchInvalidInput
	}

	// Step 2: email search は backend で最大長を検証し、過大入力を repository/DB へ渡さない。
	email := strings.TrimSpace(input.Email)
	if len(email) > adminAccountSearchMaxEmailLength {
		return AccountSearchQuery{}, ErrAccountSearchInvalidInput
	}

	// Step 3: cursor は opaque value として空白だけを除去し、SQL には repository の parameter binding で渡す。
	return AccountSearchQuery{Email: email, Cursor: strings.TrimSpace(input.Cursor), Limit: limit}, nil
}

func adminAccountSearchResult(repositoryResult AccountSearchRepositoryResult, requestID string) AccountSearchResult {
	// Step 1: repository result の slice 長に合わせて事前確保し、transport 変換時の副作用を持たない値コピーに限定する。
	accounts := make([]AccountSummary, 0, len(repositoryResult.Accounts))
	for _, account := range repositoryResult.Accounts {
		accounts = append(accounts, adminAccountSummary(account))
	}

	// Step 2: request ID と opaque cursor を付与し、handler が HTTP DTO へ詰め替えるだけで済む結果にする。
	return AccountSearchResult{Accounts: accounts, NextCursor: repositoryResult.NextCursor, RequestID: requestID}
}

func adminAccountSummary(account AccountSummaryRecord) AccountSummary {
	// Step 1: repository read model を transport 用 DTO へ値コピーし、時刻は UTC に正規化する。
	return AccountSummary{
		AccountID:    account.AccountID,
		Email:        account.Email,
		Status:       account.Status,
		PasskeyCount: account.PasskeyCount,
		CreatedAt:    account.CreatedAt.UTC(),
	}
}
