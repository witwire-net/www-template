package app

import (
	"context"
	"crypto/tls"
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

	// SecureTransport が有効な場合、STARTTLS による暗号化接続を必須とする。
	if s.config.SMTP.SecureTransport {
		return s.sendSecure(address, host, auth, from, recipients, message)
	}

	return smtp.SendMail(address, auth, from, recipients, []byte(message))
}

// sendSecure は SMTP 接続を確立し、STARTTLS が成功することを必須としてメールを送信する。
// STARTTLS が失敗・未サポートの場合は送信を拒否し、secure transport 要件を満たさないことをエラーとする。
func (s *SMTPSender) sendSecure(address string, host string, auth smtp.Auth, from string, recipients []string, message string) (err error) {
	conn, err := smtp.Dial(address)
	if err != nil {
		return fmt.Errorf("smtp: dial: %w", err)
	}
	// Quit() が成功した場合は既に接続がクローズされるため、
	// defer での Close はエラー発生時の cleanup のみに限定する。
	quitOK := false
	defer func() {
		if !quitOK {
			if closeErr := conn.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("smtp: close: %w", closeErr)
			}
		}
	}()

	if err := conn.StartTLS(&tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}); err != nil {
		return fmt.Errorf("smtp: STARTTLS required but failed: %w", err)
	}

	if auth != nil {
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}

	if err := conn.Mail(from); err != nil {
		return fmt.Errorf("smtp: mail: %w", err)
	}
	for _, rcpt := range recipients {
		if err := conn.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp: rcpt: %w", err)
		}
	}

	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close writer: %w", err)
	}
	if err := conn.Quit(); err != nil {
		return fmt.Errorf("smtp: quit: %w", err)
	}
	quitOK = true
	return nil
}
