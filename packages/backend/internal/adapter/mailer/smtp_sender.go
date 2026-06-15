package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"www-template/packages/backend/internal/platform/config"
)

// SMTPSender は SMTP transport を使ってメール本文を配送する adapter である。
//
// 役割:
//   - InfraConfig に含まれる SMTP/Mail 設定を読み、Account recovery などのメールを外部 SMTP server へ送信する。
//   - host/from/recipient が欠けている場合も成功扱いにせず、分類済み配送エラーとして返す。
//   - SMTP password やメール本文はログ属性として扱わず、呼び出し側は DeliveryErrorStage / DeliveryErrorClass だけを観測に使う。
type SMTPSender struct {
	config config.InfraConfig
}

// NewSMTPSender は SMTP 送信 adapter を生成する。
//
// 引数:
//   - config: SMTP host、port、認証情報、差出人メールアドレスを含む infrastructure 設定。
//
// 戻り値:
//   - *SMTPSender: Send で配送を実行する adapter。設定検証は Send 時に fail-closed で行う。
//
// 使用例:
//
//	sender := NewSMTPSender(cfg.Infra)
//	if err := sender.Send(ctx, []string{"user@example.com"}, message); err != nil {
//		return err
//	}
func NewSMTPSender(config config.InfraConfig) *SMTPSender {
	return &SMTPSender{config: config}
}

// Send は指定された recipient へ RFC 5322 形式の message を SMTP で送信する。
//
// 引数:
//   - ctx: 現時点では SMTP 標準ライブラリへ直接渡せないため予約引数として受け取り、将来の transport 差し替え時の呼び出し形を固定する。
//   - recipients: SMTP RCPT TO に渡す配送先。空の場合は smtp_recipient_missing を返す。
//   - message: From/To/Subject/body を含む送信本文。ログや trace へは出力しない。
//
// 戻り値:
//   - error: nil の場合は SMTP server が message を受理したことを表す。設定欠落、接続失敗、STARTTLS 失敗、認証失敗、宛先拒否、DATA 失敗は分類済み error として返す。
//
// エラーケース:
//   - SMTP host または from address が空の場合は smtp_config_missing。
//   - recipients が空の場合は smtp_recipient_missing。
//   - SecureTransport=true の場合は STARTTLS 成功を必須にし、失敗時は smtp_starttls_failed。
func (s *SMTPSender) Send(_ context.Context, recipients []string, message string) error {
	host := strings.TrimSpace(s.config.SMTP.Host)
	from := strings.TrimSpace(s.config.Mail.FromAddress)
	if host == "" || from == "" {
		return newDeliveryError("config", "smtp_config_missing", nil)
	}
	if len(recipients) == 0 {
		return newDeliveryError("config", "smtp_recipient_missing", nil)
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

	if err := smtp.SendMail(address, auth, from, recipients, []byte(message)); err != nil {
		return newDeliveryError("sendmail", "smtp_send_failed", err)
	}
	return nil
}

// sendSecure は SMTP 接続を確立し、STARTTLS が成功することを必須としてメールを送信する。
// STARTTLS が失敗・未サポートの場合は送信を拒否し、secure transport 要件を満たさないことをエラーとする。
func (s *SMTPSender) sendSecure(address string, host string, auth smtp.Auth, from string, recipients []string, message string) (err error) {
	conn, err := smtp.Dial(address)
	if err != nil {
		return newDeliveryError("dial", "smtp_dial_failed", err)
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
		return newDeliveryError("starttls", "smtp_starttls_failed", err)
	}

	if auth != nil {
		if err := conn.Auth(auth); err != nil {
			return newDeliveryError("auth", "smtp_auth_failed", err)
		}
	}

	if err := conn.Mail(from); err != nil {
		return newDeliveryError("mail_from", "smtp_from_rejected", err)
	}
	for _, rcpt := range recipients {
		if err := conn.Rcpt(rcpt); err != nil {
			return newDeliveryError("rcpt", "smtp_recipient_rejected", err)
		}
	}

	w, err := conn.Data()
	if err != nil {
		return newDeliveryError("data", "smtp_data_failed", err)
	}
	if _, err := w.Write([]byte(message)); err != nil {
		return newDeliveryError("data", "smtp_data_failed", err)
	}
	if err := w.Close(); err != nil {
		return newDeliveryError("data", "smtp_data_failed", err)
	}
	if err := conn.Quit(); err != nil {
		return newDeliveryError("quit", "smtp_quit_failed", err)
	}
	quitOK = true
	return nil
}
