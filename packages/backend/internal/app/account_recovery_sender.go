package app

import (
	"context"
	"fmt"
	"strings"

	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

type AccountRecoverySender struct {
	sender *SMTPSender
	config types.InfraConfig
}

func NewAccountRecoverySender(sender *SMTPSender, config types.InfraConfig) *AccountRecoverySender {
	return &AccountRecoverySender{sender: sender, config: config}
}

func (s *AccountRecoverySender) SendAccountRecovery(ctx context.Context, delivery usecases.RecoveryDelivery) error {
	if s.sender == nil {
		return nil
	}

	return s.sender.Send(ctx, []string{delivery.Email}, buildAccountRecoveryMessage(strings.TrimSpace(s.config.Mail.FromAddress), delivery))
}

func buildAccountRecoveryMessage(from string, delivery usecases.RecoveryDelivery) string {
	return strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", delivery.Email),
		"Subject: www-template recovery",
		"",
		delivery.RecoveryURL,
	}, "\r\n")
}
