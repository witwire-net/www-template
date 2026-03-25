package app

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"www-template/packages/backend/internal/types"
)

type SMTPSender struct {
	config types.InfraConfig
}

func NewSMTPSender(config types.InfraConfig) *SMTPSender {
	return &SMTPSender{config: config}
}

func (s *SMTPSender) Send(_ context.Context, recipients []string, message string) error {
	host := strings.TrimSpace(s.config.SMTP.Host)
	from := strings.TrimSpace(s.config.Mail.FromAddress)
	if host == "" || from == "" || len(recipients) == 0 {
		return nil
	}

	address := net.JoinHostPort(host, fmt.Sprintf("%d", s.config.SMTP.Port))
	auth := smtp.PlainAuth("", s.config.SMTP.Username, s.config.SMTP.Password, host)
	if strings.TrimSpace(s.config.SMTP.Username) == "" {
		auth = nil
	}

	return smtp.SendMail(address, auth, from, recipients, []byte(message))
}
