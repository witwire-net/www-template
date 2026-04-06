package usecases_test

import (
	"context"
	"testing"
	"time"

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

// ─── stubs ───────────────────────────────────────────────────────────────────

type stubStateRepo struct {
	challenges       map[string]domain.AuthChallenge
	sessions         map[string]domain.Session
	recoveryTokens   map[string]domain.RecoveryToken
	recoverySessions map[string]domain.RecoverySession
	counters         map[string]int
	locks            map[string]time.Time
	otpStore         map[string]string
	clock            func() time.Time
}

func newStubStateRepo(clock func() time.Time) *stubStateRepo {
	return &stubStateRepo{
		challenges:       map[string]domain.AuthChallenge{},
		sessions:         map[string]domain.Session{},
		recoveryTokens:   map[string]domain.RecoveryToken{},
		recoverySessions: map[string]domain.RecoverySession{},
		counters:         map[string]int{},
		locks:            map[string]time.Time{},
		otpStore:         map[string]string{},
		clock:            clock,
	}
}

func (r *stubStateRepo) SaveChallenge(_ context.Context, c domain.AuthChallenge, _ time.Duration) error {
	r.challenges[c.Challenge()] = c
	return nil
}
func (r *stubStateRepo) ConsumeChallenge(_ context.Context, secret string) (domain.AuthChallenge, error) {
	c, ok := r.challenges[secret]
	if !ok {
		return emptyChallenge(), domain.ErrChallengeNotFound
	}
	delete(r.challenges, secret)
	return c, nil
}
func (r *stubStateRepo) SaveSession(_ context.Context, s domain.Session, _ time.Duration) error {
	r.sessions[s.Token()] = s
	return nil
}
func (r *stubStateRepo) RefreshSession(_ context.Context, s domain.Session, _ time.Duration) error {
	r.sessions[s.Token()] = s
	return nil
}
func (r *stubStateRepo) GetSessionByToken(_ context.Context, token string) (domain.Session, error) {
	s, ok := r.sessions[token]
	if !ok {
		return emptySession(), domain.ErrSessionNotFound
	}
	return s, nil
}
func (r *stubStateRepo) RevokeSession(_ context.Context, s domain.Session, _ time.Duration) error {
	r.sessions[s.Token()] = s
	return nil
}
func (r *stubStateRepo) IssueRecoveryToken(_ context.Context, t domain.RecoveryToken, _ time.Duration) error {
	r.recoveryTokens[t.Secret()] = t
	return nil
}
func (r *stubStateRepo) SaveRecoveryDeliveryFailure(_ context.Context, _ domain.RecoveryDeliveryFailure, _ time.Duration) error {
	return nil
}
func (r *stubStateRepo) GetRecoveryTokenBySecret(_ context.Context, secret string) (domain.RecoveryToken, error) {
	t, ok := r.recoveryTokens[secret]
	if !ok {
		return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
	}
	return t, nil
}
func (r *stubStateRepo) ConsumeRecoveryToken(_ context.Context, t domain.RecoveryToken) error {
	r.recoveryTokens[t.Secret()] = t
	return nil
}
func (r *stubStateRepo) SaveRecoverySession(_ context.Context, s domain.RecoverySession, _ time.Duration) error {
	r.recoverySessions[s.ID()] = s
	return nil
}
func (r *stubStateRepo) GetRecoverySession(_ context.Context, id string) (domain.RecoverySession, error) {
	s, ok := r.recoverySessions[id]
	if !ok {
		return emptyRecoverySession(), domain.ErrRecoverySessionNotFound
	}
	return s, nil
}
func (r *stubStateRepo) ConsumeRecoverySession(_ context.Context, s domain.RecoverySession) error {
	r.recoverySessions[s.ID()] = s
	return nil
}
func (r *stubStateRepo) IncrementThrottle(_ context.Context, key string, _ time.Duration) (int, error) {
	r.counters[key]++
	return r.counters[key], nil
}
func (r *stubStateRepo) SetLock(_ context.Context, key string, until time.Time, _ time.Duration) error {
	r.locks[key] = until
	return nil
}
func (r *stubStateRepo) GetLock(_ context.Context, key string) (domain.AuthLock, bool, error) {
	until, ok := r.locks[key]
	if !ok {
		return domain.NewAuthLock(time.Time{}), false, nil
	}
	return domain.NewAuthLock(until), true, nil
}
func (r *stubStateRepo) SavePasskeyOtp(_ context.Context, otpKey string, value string, _ time.Duration) error {
	r.otpStore[otpKey] = value
	return nil
}
func (r *stubStateRepo) ConsumePasskeyOtp(_ context.Context, otpKey string) (string, error) {
	v, ok := r.otpStore[otpKey]
	if !ok {
		return "", domain.ErrOtpNotFound
	}
	delete(r.otpStore, otpKey)
	return v, nil
}
func (r *stubStateRepo) GetPasskeyOtp(_ context.Context, otpKey string) (string, error) {
	v, ok := r.otpStore[otpKey]
	if !ok {
		return "", domain.ErrOtpNotFound
	}
	return v, nil
}

// stubAccountRepo は AuthAccountRepository の in-memory スタブ。
type stubAccountRepo struct {
	accounts map[string]domain.AuthAccount // keyed by accountID
}

func newStubAccountRepoWithAccount(account domain.AuthAccount) *stubAccountRepo {
	return &stubAccountRepo{accounts: map[string]domain.AuthAccount{account.AccountID(): account}}
}

func (r *stubAccountRepo) FindByIdentifier(_ context.Context, identifier string) (domain.AuthAccount, error) {
	for _, a := range r.accounts {
		if a.Identifier() == identifier {
			return a, nil
		}
	}
	return emptyAuthAccount(), domain.ErrAuthAccountNotFound
}
func (r *stubAccountRepo) FindByCredential(_ context.Context, handle string) (domain.AuthAccount, error) {
	for _, a := range r.accounts {
		for _, c := range a.Credentials() {
			if c.CredentialHandle() == handle {
				return a, nil
			}
		}
	}
	return emptyAuthAccount(), domain.ErrAuthAccountNotFound
}
func (r *stubAccountRepo) FindByEmail(_ context.Context, email string) (domain.AuthAccount, error) {
	for _, a := range r.accounts {
		if a.Email() == email {
			return a, nil
		}
	}
	return emptyAuthAccount(), domain.ErrAuthAccountNotFound
}
func (r *stubAccountRepo) AddPasskey(_ context.Context, accountID string, credentialID string, handle string) (domain.AuthAccount, error) {
	a, ok := r.accounts[accountID]
	if !ok {
		return emptyAuthAccount(), domain.ErrAuthAccountNotFound
	}
	newCred, err := domain.NewPasskeyCredential(credentialID, accountID, a.Identifier(), handle, time.Time{})
	if err != nil {
		return emptyAuthAccount(), err
	}
	updated, err := domain.NewAuthAccountWithCredentials(a.AccountID(), a.Identifier(), a.Email(), append(a.Credentials(), newCred))
	if err != nil {
		return emptyAuthAccount(), err
	}
	r.accounts[accountID] = updated
	return updated, nil
}
func (r *stubAccountRepo) ListPasskeys(_ context.Context, accountID string) ([]domain.PasskeyCredential, error) {
	a, ok := r.accounts[accountID]
	if !ok {
		return nil, domain.ErrAuthAccountNotFound
	}
	return a.Credentials(), nil
}
func (r *stubAccountRepo) DeletePasskeyByID(_ context.Context, accountID string, credentialID string) error {
	a, ok := r.accounts[accountID]
	if !ok {
		return domain.ErrAuthAccountNotFound
	}
	creds := a.Credentials()
	for _, c := range creds {
		if c.ID() == credentialID {
			return nil
		}
	}
	return domain.ErrAuthAccountNotFound
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func emptyChallenge() domain.AuthChallenge {
	c, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return c
}
func emptySession() domain.Session {
	s, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return s
}
func emptyRecoveryToken() domain.RecoveryToken {
	t, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return t
}
func emptyRecoverySession() domain.RecoverySession {
	s, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
	return s
}
func emptyAuthAccount() domain.AuthAccount {
	a, _ := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder@example.com", "placeholder@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder")
	return a
}

var ids = []string{
	"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX",
	"01ARZ3NDEKTSV4RRFFQ69G5FAY", "01ARZ3NDEKTSV4RRFFQ69G5FAZ", "01ARZ3NDEKTSV4RRFFQ69G5FB1",
	"01ARZ3NDEKTSV4RRFFQ69G5FB2", "01ARZ3NDEKTSV4RRFFQ69G5FB3", "01ARZ3NDEKTSV4RRFFQ69G5FB4",
	"01ARZ3NDEKTSV4RRFFQ69G5FB5", "01ARZ3NDEKTSV4RRFFQ69G5FB6", "01ARZ3NDEKTSV4RRFFQ69G5FB7",
	"01ARZ3NDEKTSV4RRFFQ69G5FB8", "01ARZ3NDEKTSV4RRFFQ69G5FB9", "01ARZ3NDEKTSV4RRFFQ69G5FBA",
	"01ARZ3NDEKTSV4RRFFQ69G5FBB", "01ARZ3NDEKTSV4RRFFQ69G5FBC", "01ARZ3NDEKTSV4RRFFQ69G5FBD",
	"01ARZ3NDEKTSV4RRFFQ69G5FBE", "01ARZ3NDEKTSV4RRFFQ69G5FBF",
}

func newSeqPolicy() types.AuthIDPolicy {
	next := 0
	return types.AuthIDPolicy{
		New:      func() string { v := ids[next]; next++; return v },
		Validate: domain.ValidateAuthID,
	}
}

func fixedClock() func() time.Time {
	t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func accountWithTwoPasskeys(t *testing.T) (domain.AuthAccount, *stubAccountRepo) {
	t.Helper()
	cred1, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"handle-one",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential cred1: %v", err)
	}
	cred2, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"handle-two",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential cred2: %v", err)
	}
	account, err := domain.NewAuthAccountWithCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"user@example.com",
		[]domain.PasskeyCredential{cred1, cred2},
	)
	if err != nil {
		t.Fatalf("NewAuthAccountWithCredentials: %v", err)
	}
	return account, newStubAccountRepoWithAccount(account)
}

func accountWithOnePasskey(t *testing.T) (domain.AuthAccount, *stubAccountRepo) {
	t.Helper()
	account, err := domain.NewAuthAccount(
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"user@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"handle-one",
	)
	if err != nil {
		t.Fatalf("NewAuthAccount: %v", err)
	}
	return account, newStubAccountRepoWithAccount(account)
}

func newTestAuthService(stateRepo usecases.AuthStateRepository, accountRepo usecases.AuthAccountRepository) *usecases.AuthService {
	return usecases.NewAuthService(stateRepo, accountRepo, nil, nil, fixedClock(), newSeqPolicy(), types.AuthConfig{
		ChallengeTTL:                5 * time.Minute,
		SessionIdleTTL:              30 * time.Minute,
		SessionAbsoluteTTL:          24 * time.Hour,
		RecoveryTokenTTL:            30 * time.Minute,
		RecoverySessionTTL:          10 * time.Minute,
		RecoveryEmailThrottleLimit:  3,
		RecoveryIPThrottleLimit:     5,
		RecoveryEmailThrottleWindow: time.Hour,
		RecoveryIPThrottleWindow:    time.Hour,
		PasskeyStartThrottleLimit:   5,
		PasskeyStartThrottleWindow:  time.Minute,
		FailureLockThreshold:        10,
		FailureLockDuration:         15 * time.Minute,
		FailureLockWindow:           time.Minute,
		WebAuthnRPID:                "example.com",
		AccountRecoveryURLBase:      "https://example.com/recover",
	})
}

// ─── Task 4.7: DeletePasskey – last passkey cannot be deleted ─────────────────

// [AUTH-BE-4.7] DeletePasskey が最終 1 件のとき ErrLastPasskeyCannotBeDeleted を返す
func TestDeletePasskeyRejectsLastCredential(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthService(newStubStateRepo(fixedClock()), accountRepo)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"
	passkeyID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"

	err := svc.DeletePasskey(context.Background(), accountID, passkeyID)
	if err == nil {
		t.Fatal("expected ErrLastPasskeyCannotBeDeleted, got nil")
	}
	if err != usecases.ErrLastPasskeyCannotBeDeleted {
		t.Fatalf("expected ErrLastPasskeyCannotBeDeleted, got %v", err)
	}
}

// ─── Task 4.8: DeletePasskey – other account's credential is rejected ─────────

// [AUTH-BE-4.8] DeletePasskey で account A の accountID を使って account B の credentialID を
// 指定した場合、適切なエラー（domain.ErrAuthAccountNotFound）を返す（cross-account deletion rejection）。
func TestDeletePasskeyRejectsCrossAccountCredential(t *testing.T) {
	t.Parallel()

	// account A: 2 件のパスキーを持つ
	accountA, _ := accountWithTwoPasskeys(t)

	// account B: 別のアカウント（別の credentialID を持つ）
	accountBCred, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FB1", // account B の credentialID
		"01ARZ3NDEKTSV4RRFFQ69G5FB2", // account B の accountID
		"other@example.com",
		"handle-b",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential accountB: %v", err)
	}
	accountB, err := domain.NewAuthAccountWithCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FB2",
		"other@example.com",
		"other@example.com",
		[]domain.PasskeyCredential{accountBCred},
	)
	if err != nil {
		t.Fatalf("NewAuthAccountWithCredentials accountB: %v", err)
	}

	// 両アカウントを含む repo
	accountRepo := &stubAccountRepo{accounts: map[string]domain.AuthAccount{
		accountA.AccountID(): accountA,
		accountB.AccountID(): accountB,
	}}
	svc := newTestAuthService(newStubStateRepo(fixedClock()), accountRepo)

	// account A の accountID で account B の credentialID を削除しようとする
	err = svc.DeletePasskey(context.Background(), accountA.AccountID(), accountBCred.ID())
	if err == nil {
		t.Fatal("expected error for cross-account credential deletion, got nil")
	}
	// account A には account B の credentialID が存在しないので ErrAuthAccountNotFound を返す
	if err != domain.ErrAuthAccountNotFound {
		t.Fatalf("expected domain.ErrAuthAccountNotFound, got %v", err)
	}
}

// ─── Task 4.12: StartAddPasskeyByOtp – expired/missing OTP ───────────────────

// [AUTH-BE-4.12] StartAddPasskeyByOtp で存在しない OTP を指定すると ErrOtpExpiredOrConsumed を返す
func TestStartAddPasskeyByOtpRejectsMissingOtp(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	svc := newTestAuthService(stateRepo, newStubAccountRepoWithAccount(emptyAuthAccount()))

	_, err := svc.StartAddPasskeyByOtp(context.Background(), "999999")
	if err == nil {
		t.Fatal("expected ErrOtpExpiredOrConsumed, got nil")
	}
	if err != usecases.ErrOtpExpiredOrConsumed {
		t.Fatalf("expected ErrOtpExpiredOrConsumed, got %v", err)
	}
}

// ─── Task 4.13: FinishAddPasskeyByOtp – consumed OTP ─────────────────────────

// [AUTH-BE-4.13] FinishAddPasskeyByOtp で消費済み OTP を指定すると ErrOtpExpiredOrConsumed を返す。
// 1 回目の FinishAddPasskeyByOtp を成功させて OTP が消費されることを確認し、
// 2 回目の呼び出しで ErrOtpExpiredOrConsumed が返ることを確認する。
func TestFinishAddPasskeyByOtpRejectsConsumedOtp(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthService(stateRepo, accountRepo)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"

	// Step 1: OTP を発行する
	otp, err := svc.IssuePasskeyOtp(context.Background(), accountID)
	if err != nil {
		t.Fatalf("IssuePasskeyOtp: %v", err)
	}

	// Step 2: StartAddPasskeyByOtp で OTP を使ってチャレンジを取得する（OTP は消費されない = GetPasskeyOtp）
	passkeyChallenge, err := svc.StartAddPasskeyByOtp(context.Background(), otp)
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp: %v", err)
	}

	// Step 3: 1 回目の FinishAddPasskeyByOtp を成功させる
	// credential は "credentialHandle::challengeValue" の形式
	credential := "handle-new::" + passkeyChallenge.Challenge
	if err := svc.FinishAddPasskeyByOtp(context.Background(), otp, credential); err != nil {
		t.Fatalf("FinishAddPasskeyByOtp (1st call) should succeed, got: %v", err)
	}

	// Step 4: 2 回目の FinishAddPasskeyByOtp: OTP は既に 1 回目で消費されている
	if err := svc.FinishAddPasskeyByOtp(context.Background(), otp, credential); err == nil {
		t.Fatal("expected ErrOtpExpiredOrConsumed on 2nd call, got nil")
	} else if err != usecases.ErrOtpExpiredOrConsumed {
		t.Fatalf("expected ErrOtpExpiredOrConsumed, got %v", err)
	}
}

// ─── Regression: challenge mismatch does not consume another account's challenge ─

// [AUTH-BE-REG-1] FinishAddPasskeyByOtp に別 OTP セッションの challengeValue を渡した場合、
// ErrBadRequest を返し、正規の challenge（C1 と C2 の両方）は ConsumeChallenge されない（差し替え攻撃への回帰テスト）。
func TestFinishAddPasskeyByOtpRejectsChallengeMismatch(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthService(stateRepo, accountRepo)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"

	// Step 1: account A の OTP を発行し、チャレンジ C1 を取得する
	otpA, err := svc.IssuePasskeyOtp(context.Background(), accountID)
	if err != nil {
		t.Fatalf("IssuePasskeyOtp A: %v", err)
	}
	challengeA, err := svc.StartAddPasskeyByOtp(context.Background(), otpA)
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp A: %v", err)
	}

	// Step 2: 別のアカウントで OTP を発行し、チャレンジ C2 を取得する
	otherAccountID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	otpB, err := svc.IssuePasskeyOtp(context.Background(), otherAccountID)
	if err != nil {
		t.Fatalf("IssuePasskeyOtp B: %v", err)
	}
	challengeB, err := svc.StartAddPasskeyByOtp(context.Background(), otpB)
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp B: %v", err)
	}

	// mismatch 前に challenge の数を記録する（C1 と C2 の 2 件が stateRepo.challenges に存在する）
	challengeCountBefore := len(stateRepo.challenges)
	if challengeCountBefore != 2 {
		t.Fatalf("expected 2 challenges before mismatch, got %d", challengeCountBefore)
	}

	// Step 3: account A の OTP を使い、C2（別セッションの challenge）を credential に埋め込んで送る
	// → challengeValue（A の otpChallengeKey から取得）!= storedChallengeValue（C2）なので ErrBadRequest
	// → ConsumeChallenge は呼ばれない（早期リターン）
	mismatchCredential := "handle-new::" + challengeB.Challenge
	if err := svc.FinishAddPasskeyByOtp(context.Background(), otpA, mismatchCredential); err == nil {
		t.Fatal("expected ErrBadRequest for challenge mismatch, got nil")
	} else if err != usecases.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}

	// Step 4: mismatch で早期リターンしたため C1 と C2 は stateRepo.challenges に残っている
	// （ConsumeChallenge が呼ばれなかったことの証明）
	if len(stateRepo.challenges) != 2 {
		t.Fatalf("expected challenges to remain intact after mismatch (got %d, want 2)", len(stateRepo.challenges))
	}
	if _, c1exists := stateRepo.challenges[challengeA.Challenge]; !c1exists {
		t.Fatal("C1 (account A challenge) should not be consumed after mismatch attempt")
	}
	if _, c2exists := stateRepo.challenges[challengeB.Challenge]; !c2exists {
		t.Fatal("C2 (other account challenge) should not be consumed after mismatch attempt")
	}
}
