package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

// [ADMIN-CONSOLE-BE-S083] 範囲外の limit は Admin backend で拒否され、repository query は実行されない。
func TestAdminAccountSearchRejectsOutOfRangeLimitBeforeRepository(t *testing.T) {
	t.Parallel()

	// Step 1: repository fake を注入し、invalid pagination のときに query が呼ばれたかを観測できる状態にする。
	repository := &fakeAdminAccountSearchRepository{}
	service := mustNewAdminAccountSearchService(t, repository)
	limit := int32(0)

	// Step 2: limit=0 を渡し、application use case が repository より前に validation error を返すことを確認する。
	_, err := service.SearchAccounts(context.Background(), AdminAccountSearchInput{Limit: &limit, RequestID: "req-search-1"})
	if !errors.Is(err, ErrAdminAccountSearchInvalidInput) {
		t.Fatalf("expected ErrAdminAccountSearchInvalidInput, got %v", err)
	}

	// Step 3: repository query が 1 回も実行されていないことを確認し、HTTP handler が 400 へ写像できる boundary を固定する。
	if repository.calls != 0 {
		t.Fatalf("repository search calls = %d, want 0", repository.calls)
	}
}

func TestAdminAccountSearchUsesDefaultLimitAndReturnsReadModel(t *testing.T) {
	t.Parallel()

	// Step 1: successful repository fake を注入し、未指定 limit が default 値に正規化されることを観測できる状態にする。
	repository := &fakeAdminAccountSearchRepository{result: AdminAccountSearchRepositoryResult{Accounts: []AdminAccountSummaryRecord{{
		AccountID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Email:        "customer@example.com",
		Status:       "active",
		PasskeyCount: 2,
		CreatedAt:    time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
	}}, NextCursor: "01NEXTCURSOR00000000000000"}}
	service := mustNewAdminAccountSearchService(t, repository)

	// Step 2: email に空白を含む検索入力を渡し、repository には trim 済み query だけが渡ることを確認する。
	result, err := service.SearchAccounts(context.Background(), AdminAccountSearchInput{Email: " customer@example.com ", RequestID: "req-search-2"})
	if err != nil {
		t.Fatalf("search admin accounts: %v", err)
	}

	// Step 3: repository query と response DTO の両方を確認し、search use case が pagination と read model 変換だけを担うことを固定する。
	if repository.query.Limit != adminAccountSearchDefaultLimit || repository.query.Email != "customer@example.com" {
		t.Fatalf("unexpected repository query: %+v", repository.query)
	}
	if result.RequestID != "req-search-2" || result.NextCursor != "01NEXTCURSOR00000000000000" || len(result.Accounts) != 1 {
		t.Fatalf("unexpected search result: %+v", result)
	}
}

func mustNewAdminAccountSearchService(t *testing.T, repository AdminAccountSearchRepository) *AdminAccountSearchService {
	t.Helper()

	// Step 1: constructor 経由で service を作り、nil dependency validation と test 対象の条件を揃える。
	service, err := NewAdminAccountSearchService(repository)
	if err != nil {
		t.Fatalf("new admin account search service: %v", err)
	}

	// Step 2: 構築済み service だけを返し、各 test が SearchAccounts の振る舞いに集中できるようにする。
	return service
}

type fakeAdminAccountSearchRepository struct {
	calls        int
	detailCalls  int
	query        AdminAccountSearchQuery
	result       AdminAccountSearchRepositoryResult
	detailResult AdminAccountSummaryRecord
	err          error
}

func (r *fakeAdminAccountSearchRepository) SearchAccounts(ctx context.Context, query AdminAccountSearchQuery) (AdminAccountSearchRepositoryResult, error) {
	// Step 1: fake でも context cancellation を尊重し、実 repository と同じ入口条件を保つ。
	if err := ctx.Err(); err != nil {
		return AdminAccountSearchRepositoryResult{}, err
	}

	// Step 2: 呼び出し回数と検証済み query を記録し、pagination validation 後にだけ repository が呼ばれることを観測可能にする。
	r.calls++
	r.query = query
	if r.err != nil {
		return AdminAccountSearchRepositoryResult{}, r.err
	}

	// Step 3: test が設定した deterministic read model を返し、DB なしで application orchestration を検証する。
	return r.result, nil
}

func (r *fakeAdminAccountSearchRepository) FindAccountByID(ctx context.Context, accountID string) (AdminAccountSummaryRecord, error) {
	// Step 1: fake でも context cancellation を尊重し、detail use case の入口条件を実 repository と揃える。
	if err := ctx.Err(); err != nil {
		return AdminAccountSummaryRecord{}, err
	}

	// Step 2: detail 呼び出し回数を記録し、validation 後にだけ repository が呼ばれることを観測可能にする。
	r.detailCalls++
	if accountID == "" {
		return AdminAccountSummaryRecord{}, ErrAdminAccountSearchNotFound
	}
	if r.err != nil {
		return AdminAccountSummaryRecord{}, r.err
	}

	// Step 3: test が設定した deterministic read model を返し、DB なしで detail orchestration を検証する。
	return r.detailResult, nil
}
