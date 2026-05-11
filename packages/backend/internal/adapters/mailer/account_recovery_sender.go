package mailer

import (
	"context"
	"fmt"
	"strings"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
)

type AccountRecoverySender struct {
	sender *SMTPSender
	config config.InfraConfig
}

func NewAccountRecoverySender(sender *SMTPSender, config config.InfraConfig) *AccountRecoverySender {
	return &AccountRecoverySender{sender: sender, config: config}
}

// SendAccountRecovery は recovery または device-link のメールを送信する。
// delivery.Kind に応じて件名と本文を切り替える。
// 未知の kind の場合はセキュリティ上エラーを返す（fail-closed）。
func (s *AccountRecoverySender) SendAccountRecovery(ctx context.Context, delivery application.RecoveryDelivery) error {
	if s.sender == nil {
		return nil
	}

	var msg string
	switch delivery.Kind {
	case domain.TokenKindDeviceLink:
		msg = buildDeviceLinkMessage(strings.TrimSpace(s.config.Mail.FromAddress), delivery.Email, delivery.RecoveryURL, delivery.RequestID)
	case domain.TokenKindRecovery:
		msg = buildAccountRecoveryMessage(strings.TrimSpace(s.config.Mail.FromAddress), delivery)
	default:
		return fmt.Errorf("account recovery sender: unknown token kind %q", delivery.Kind)
	}
	return s.sender.Send(ctx, []string{delivery.Email}, msg)
}

// SendDeviceLink は device-link URL を登録済みメールアドレスへ送信する。
// SendAccountRecovery のラッパーとして動作する。
func (s *AccountRecoverySender) SendDeviceLink(ctx context.Context, delivery application.RecoveryDelivery) error {
	return s.SendAccountRecovery(ctx, delivery)
}

// SendRecoveryComplete はパスキー復旧完了の通知メールを送信する。
func (s *AccountRecoverySender) SendRecoveryComplete(ctx context.Context, accountID, email string) error {
	if s.sender == nil {
		return nil
	}
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", strings.TrimSpace(s.config.Mail.FromAddress)),
		fmt.Sprintf("To: %s", email),
		"Subject: www-template passkey recovered",
		"",
		"Your passkey has been successfully recovered.",
	}, "\r\n")
	return s.sender.Send(ctx, []string{email}, msg)
}

// SendDeviceLinkComplete は新規デバイスでのパスキー追加完了の通知メールを送信する。
func (s *AccountRecoverySender) SendDeviceLinkComplete(ctx context.Context, accountID, email string) error {
	if s.sender == nil {
		return nil
	}
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", strings.TrimSpace(s.config.Mail.FromAddress)),
		fmt.Sprintf("To: %s", email),
		"Subject: www-template passkey added on new device",
		"",
		"A new passkey has been successfully added to your account on a new device.",
	}, "\r\n")
	return s.sender.Send(ctx, []string{email}, msg)
}

func buildAccountRecoveryMessage(from string, delivery application.RecoveryDelivery) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", delivery.Email),
		"Subject: www-template recovery",
		"",
		delivery.RecoveryURL,
	}, "\r\n")
}

func buildDeviceLinkMessage(from, email, url, requestID string) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", email),
		"Subject: www-template device login link",
		"",
		fmt.Sprintf("Your device login link: %s", url),
		fmt.Sprintf("Request ID: %s", requestID),
	}, "\r\n")
}
