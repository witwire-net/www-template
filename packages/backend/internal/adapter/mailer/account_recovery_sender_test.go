package mailer

import (
	"context"
	"strings"
	"testing"

	application "www-template/packages/backend/internal/application"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
)

type stubMailerAccountSettingRepository struct {
	settings map[string]domain.AccountSetting
}

func newStubMailerAccountSettingRepository(t *testing.T, accountID string, localeValue string) *stubMailerAccountSettingRepository {
	t.Helper()
	typedAccountID := testAccountID(accountID)
	locale, err := domain.NewAccountLocale(localeValue)
	if err != nil {
		t.Fatalf("NewAccountLocale: %v", err)
	}
	setting, err := domain.NewAccountSetting(typedAccountID, locale)
	if err != nil {
		t.Fatalf("NewAccountSetting: %v", err)
	}
	return &stubMailerAccountSettingRepository{settings: map[string]domain.AccountSetting{accountID: setting}}
}

func (r *stubMailerAccountSettingRepository) CreateDefault(_ context.Context, accountID domain.AccountID) (domain.AccountSetting, error) {
	setting, err := domain.NewDefaultAccountSetting(accountID)
	if err != nil {
		return emptyMailerAccountSetting(), err
	}
	r.settings[accountID.String()] = setting
	return setting, nil
}

func (r *stubMailerAccountSettingRepository) Get(_ context.Context, accountID domain.AccountID) (domain.AccountSetting, error) {
	setting, ok := r.settings[accountID.String()]
	if !ok {
		return emptyMailerAccountSetting(), domain.ErrAccountSettingNotFound
	}
	return setting, nil
}

func (r *stubMailerAccountSettingRepository) UpdateLocale(_ context.Context, accountID domain.AccountID, locale domain.AccountLocale) (domain.AccountSetting, error) {
	setting, err := domain.NewAccountSetting(accountID, locale)
	if err != nil {
		return emptyMailerAccountSetting(), err
	}
	r.settings[accountID.String()] = setting
	return setting, nil
}

func emptyMailerAccountSetting() domain.AccountSetting {
	setting, _ := domain.NewDefaultAccountSetting(testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV"))
	return setting
}

func testMailerConfig() config.InfraConfig {
	return config.InfraConfig{
		Mail: config.MailConfig{
			FromAddress: "from@example.com",
			ProductName: "www-template",
		},
	}
}

func newTestSender(t *testing.T, accountID string, localeValue string) *AccountRecoverySender {
	t.Helper()
	repository := newStubMailerAccountSettingRepository(t, accountID, localeValue)
	return NewAccountRecoverySender(nil, testMailerConfig(), application.NewAccountSettingSnapshotService(repository))
}

// [LOCALIZATION-BE-S006] 復旧メールは AccountSetting.locale の英語文面を選択する。
func TestAccountRecoveryMessageUsesAccountSettingLocale(t *testing.T) {
	t.Parallel()
	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	sender := newTestSender(t, accountID, "en")

	message, err := sender.buildAccountRecoveryMessage(context.Background(), application.RecoveryDelivery{
		RequestID:       "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		RecoveryTokenID: "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		AccountID:       testAccountID(accountID),
		Email:           "member@example.com",
		RecoveryURL:     "https://example.com/recover?token=secret-token",
		Kind:            domain.TokenKindRecovery,
	})

	if err != nil {
		t.Fatalf("buildAccountRecoveryMessage: %v", err)
	}
	assertMailContainsAll(t, message, "Subject: www-template Passkey Recovery", "Use the link below to recover your passkey.")
}

// [LOCALIZATION-BE-S007] device-link 完了メールは AccountSetting.locale の日本語文面を選択する。
func TestDeviceLinkCompleteMessageUsesAccountSettingLocale(t *testing.T) {
	t.Parallel()
	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	sender := newTestSender(t, accountID, "ja")

	message, err := sender.buildCompletionMessage(context.Background(), application.CompletionDelivery{
		AccountID: testAccountID(accountID),
		Email:     "member@example.com",
		Kind:      domain.TokenKindDeviceLink,
	})

	if err != nil {
		t.Fatalf("buildCompletionMessage: %v", err)
	}
	assertMailContainsAll(t, message, "Subject: www-template 新しいデバイスのパスキー追加完了", "新しいデバイスでパスキーが追加されました。")
}

// [LOCALIZATION-BE-S008] 未対応 locale は template 解決時に拒否される。
func TestLocalizedMessagesRejectUnsupportedLocale(t *testing.T) {
	t.Parallel()

	_, err := resolveMailTemplate(templateKindRecovery, domain.AccountLocale("fr"))

	if err == nil {
		t.Fatal("expected unsupported locale error")
	}
}

// [LOCALIZATION-BE-S006] 復旧メールの AccountSetting 読み込み失敗時は fallback locale で送信し、token を error 文字列へ含めない。
func TestAccountRecoveryMessageSnapshotFailureDoesNotExposeToken(t *testing.T) {
	t.Parallel()
	sender := newTestSender(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", "ja")
	recoveryURL := "https://example.com/recover?code=leak-marker"

	_, err := sender.buildAccountRecoveryMessage(context.Background(), application.RecoveryDelivery{
		RequestID:       "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		RecoveryTokenID: "opaque-marker-id",
		AccountID:       testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FB0"),
		Email:           "member@example.com",
		RecoveryURL:     recoveryURL,
		Kind:            domain.TokenKindRecovery,
	})

	assertErrorDoesNotContain(t, err, "leak-marker", "opaque-marker-id", recoveryURL)
}

// [LOCALIZATION-BE-S007] 完了通知の未知 kind は bearer token 相当の機微値を error 文字列へ含めない。
func TestCompletionMessageUnknownKindDoesNotExposeBearerLikeInput(t *testing.T) {
	t.Parallel()
	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	sender := newTestSender(t, accountID, "ja")

	_, err := sender.buildCompletionMessage(context.Background(), application.CompletionDelivery{
		AccountID: testAccountID(accountID),
		Email:     "bearer-token-like-secret@example.com",
		Kind:      domain.TokenKind("secret-bearer-token"),
	})

	assertErrorDoesNotContain(t, err, "bearer-token-like-secret@example.com", "secret-bearer-token")
}

// [LOCALIZATION-BE-S006] recovery メールの日本語文面は製品名と URL を含む。
func TestRecoveryMessageJapaneseRendering(t *testing.T) {
	t.Parallel()
	tmpl, err := resolveMailTemplate(templateKindRecovery, domain.AccountLocaleJapanese)
	if err != nil {
		t.Fatalf("resolveMailTemplate: %v", err)
	}
	rendered, err := renderMailTemplate(tmpl, recoveryMessageTemplateData{
		ProductName: "www-template",
		URL:         "https://example.com/recover",
		RequestID:   "REQ001",
	})
	if err != nil {
		t.Fatalf("renderMailTemplate: %v", err)
	}
	if !strings.Contains(rendered.subject, "www-template") {
		t.Fatalf("subject must contain product name, got: %s", rendered.subject)
	}
	if !strings.Contains(rendered.bodyText, "https://example.com/recover") {
		t.Fatalf("body must contain recovery URL, got: %s", rendered.bodyText)
	}
}

// [LOCALIZATION-BE-S007] 完了通知メールは HTML 本文を生成する。
func TestCompletionMessageHTMLRendering(t *testing.T) {
	t.Parallel()
	tmpl, err := resolveMailTemplate(templateKindRecoveryComplete, domain.AccountLocaleEnglish)
	if err != nil {
		t.Fatalf("resolveMailTemplate: %v", err)
	}
	rendered, err := renderMailTemplate(tmpl, recoveryCompleteTemplateData{
		ProductName: "www-template",
	})
	if err != nil {
		t.Fatalf("renderMailTemplate: %v", err)
	}
	if !strings.Contains(rendered.bodyHTML, "<p>") {
		t.Fatalf("body_html must contain HTML tags, got: %s", rendered.bodyHTML)
	}
	if !strings.Contains(rendered.subject, "www-template") {
		t.Fatalf("subject must contain product name, got: %s", rendered.subject)
	}
}

func assertMailContainsAll(t *testing.T, message string, requiredValues ...string) {
	t.Helper()
	for _, required := range requiredValues {
		if !strings.Contains(message, required) {
			t.Fatalf("message must contain %q, got %q", required, message)
		}
	}
}

func assertErrorDoesNotContain(t *testing.T, err error, forbiddenValues ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	errorText := err.Error()
	for _, forbidden := range forbiddenValues {
		if strings.Contains(errorText, forbidden) {
			t.Fatalf("error must not contain %q, got %q", forbidden, errorText)
		}
	}
}
