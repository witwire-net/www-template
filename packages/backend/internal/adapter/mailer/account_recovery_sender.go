package mailer

import (
	"context"
	"fmt"
	"strings"

	application "www-template/packages/backend/internal/application"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
)

// AccountRecoverySender は recovery / device-link / 完了通知メールを AccountSetting.locale に基づいた
// 文面で送信する adapter である。
type AccountRecoverySender struct {
	sender          *SMTPSender
	productName     string
	fromAddress     string
	accountSnapshot *application.AccountSettingSnapshotService
}

// NewAccountRecoverySender は AccountRecoverySender を生成する。
// accountSnapshot が nil の場合は既定 locale（ja）で送信する。
func NewAccountRecoverySender(sender *SMTPSender, cfg config.InfraConfig, accountSnapshot ...*application.AccountSettingSnapshotService) *AccountRecoverySender {
	var snapshotService *application.AccountSettingSnapshotService
	if len(accountSnapshot) > 0 {
		snapshotService = accountSnapshot[0]
	}
	productName := strings.TrimSpace(cfg.Mail.ProductName)
	if productName == "" {
		productName = "www-template"
	}
	return &AccountRecoverySender{
		sender:          sender,
		productName:     productName,
		fromAddress:     strings.TrimSpace(cfg.Mail.FromAddress),
		accountSnapshot: snapshotService,
	}
}

// SendAccountRecovery は recovery または device-link のメールを送信する。
// delivery.Kind に応じてテンプレートを選択し、AccountSetting.locale に基づいた文面を送信する。
// 未知の kind の場合はセキュリティ上エラーを返す（fail-closed）。
func (s *AccountRecoverySender) SendAccountRecovery(ctx context.Context, delivery application.RecoveryDelivery) error {
	if s.sender == nil {
		return nil
	}

	msg, err := s.buildAccountRecoveryMessage(ctx, delivery)
	if err != nil {
		return err
	}
	return s.sender.Send(ctx, []string{delivery.Email}, msg)
}

// SendDeviceLink は device-link URL を登録済みメールアドレスへ送信する。
func (s *AccountRecoverySender) SendDeviceLink(ctx context.Context, delivery application.RecoveryDelivery) error {
	return s.SendAccountRecovery(ctx, delivery)
}

// SendRecoveryComplete はパスキー復旧完了の通知メールを送信する。
func (s *AccountRecoverySender) SendRecoveryComplete(ctx context.Context, delivery application.CompletionDelivery) error {
	if s.sender == nil {
		return nil
	}
	msg, err := s.buildCompletionMessage(ctx, delivery)
	if err != nil {
		return err
	}
	return s.sender.Send(ctx, []string{delivery.Email}, msg)
}

// SendDeviceLinkComplete は新規デバイスでのパスキー追加完了の通知メールを送信する。
func (s *AccountRecoverySender) SendDeviceLinkComplete(ctx context.Context, delivery application.CompletionDelivery) error {
	if s.sender == nil {
		return nil
	}
	msg, err := s.buildCompletionMessage(ctx, delivery)
	if err != nil {
		return err
	}
	return s.sender.Send(ctx, []string{delivery.Email}, msg)
}

func (s *AccountRecoverySender) buildAccountRecoveryMessage(ctx context.Context, delivery application.RecoveryDelivery) (string, error) {
	locale, err := s.resolveLocale(ctx, delivery.AccountID)
	if err != nil {
		return "", err
	}

	kind, err := tokenKindToMailTemplateKind(delivery.Kind)
	if err != nil {
		return "", err
	}

	tmpl, err := resolveMailTemplate(kind, locale)
	if err != nil {
		return "", err
	}

	rendered, err := renderMailTemplate(tmpl, recoveryMessageTemplateData{
		ProductName: s.productName,
		URL:         delivery.RecoveryURL,
		RequestID:   delivery.RequestID,
	})
	if err != nil {
		return "", fmt.Errorf("render recovery message: %w", err)
	}

	return s.formatMessage(delivery.Email, rendered), nil
}

func (s *AccountRecoverySender) buildCompletionMessage(ctx context.Context, delivery application.CompletionDelivery) (string, error) {
	locale, err := s.resolveLocale(ctx, delivery.AccountID)
	if err != nil {
		return "", err
	}

	kind, err := completionKindToMailTemplateKind(delivery.Kind)
	if err != nil {
		return "", err
	}

	tmpl, err := resolveMailTemplate(kind, locale)
	if err != nil {
		return "", err
	}

	rendered, err := renderMailTemplate(tmpl, recoveryCompleteTemplateData{
		ProductName: s.productName,
	})
	if err != nil {
		return "", fmt.Errorf("render completion message: %w", err)
	}

	return s.formatMessage(delivery.Email, rendered), nil
}

func (s *AccountRecoverySender) resolveLocale(ctx context.Context, accountID domain.AccountID) (domain.AccountLocale, error) {
	if s.accountSnapshot == nil {
		return domain.DefaultAccountLocale(), nil
	}
	snapshot, err := s.accountSnapshot.Load(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("account setting snapshot unavailable")
	}
	locale, err := domain.NewAccountLocale(snapshot.Locale)
	if err != nil {
		return "", fmt.Errorf("account setting locale unsupported")
	}
	return locale, nil
}

func (s *AccountRecoverySender) formatMessage(to string, rendered renderedMail) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", s.fromAddress),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", rendered.subject),
		"",
		rendered.bodyText,
	}, "\r\n")
}
