package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/platform/config"
)

type AccountRecoverySender struct {
	sender *SMTPSender
	config config.InfraConfig
}

func NewAccountRecoverySender(sender *SMTPSender, config config.InfraConfig) *AccountRecoverySender {
	return &AccountRecoverySender{sender: sender, config: config}
}

func (s *AccountRecoverySender) SendAccountRecovery(ctx context.Context, delivery application.RecoveryDelivery) error {
	if s.sender == nil {
		return nil
	}

	return s.sender.Send(ctx, []string{delivery.Email}, buildAccountRecoveryMessage(strings.TrimSpace(s.config.Mail.FromAddress), delivery))
}

// SendPasskeyOtp は device login handoff 用の 6 桁 OTP を登録済みメールアドレスへ送信する。
// 送信失敗時は OTP 平文を含めず、requestID とエラーのみを slog で記録する。
func (s *AccountRecoverySender) SendPasskeyOtp(ctx context.Context, email string, otp string, requestID string) error {
	if s.sender == nil {
		return nil
	}

	if err := s.sender.Send(ctx, []string{email}, buildPasskeyOtpMessage(strings.TrimSpace(s.config.Mail.FromAddress), email, otp, requestID)); err != nil {
		slog.ErrorContext(ctx, "passkey OTP delivery failed",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
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

func buildPasskeyOtpMessage(from string, email string, otp string, requestID string) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", email),
		"Subject: www-template device login code",
		"",
		fmt.Sprintf("Your device login code is: %s", otp),
		fmt.Sprintf("Request ID: %s", requestID),
	}, "\r\n")
}
