package mailer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	"www-template/packages/backend/internal/platform/config"
)

// [AUTH-BE-OBS-4] SMTP host や From が欠けている場合、送信成功扱いにせず分類可能な error を返す。
func TestSMTPSenderRejectsMissingConfigInsteadOfNoop(t *testing.T) {
	t.Parallel()
	sender := NewSMTPSender(config.InfraConfig{})

	err := sender.Send(context.Background(), []string{"to@example.com"}, "test message")
	if err == nil {
		t.Fatal("expected missing SMTP config error")
	}
	var classified interface {
		DeliveryErrorStage() string
		DeliveryErrorClass() string
	}
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified delivery error, got %T", err)
	}
	if classified.DeliveryErrorStage() != "config" || classified.DeliveryErrorClass() != "smtp_config_missing" {
		t.Fatalf("unexpected delivery classification: stage=%q class=%q", classified.DeliveryErrorStage(), classified.DeliveryErrorClass())
	}
}

// [AUTH-BE-OBS-6] SMTP sender は cancel 済み context では外部接続を開始せず、分類済み error を返す。
func TestSMTPSenderRejectsCanceledContextBeforeDial(t *testing.T) {
	t.Parallel()

	// Step 1: context を先に cancel し、SMTP 設定が存在しても DialContext に進まない経路を作る。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sender := NewSMTPSender(config.InfraConfig{
		SMTP: config.SMTPConfig{Host: "127.0.0.1", Port: 25},
		Mail: config.MailConfig{FromAddress: "test@example.com"},
	})

	// Step 2: send を実行し、raw context error ではなく安全な delivery classification として返ることを検証する。
	err := sender.Send(ctx, []string{"to@example.com"}, "test message")
	if err == nil {
		t.Fatal("expected canceled context error")
	}
	var classified interface {
		DeliveryErrorStage() string
		DeliveryErrorClass() string
	}
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified delivery error, got %T", err)
	}
	if classified.DeliveryErrorStage() != "context" || classified.DeliveryErrorClass() != "smtp_canceled" {
		t.Fatalf("unexpected delivery classification: stage=%q class=%q", classified.DeliveryErrorStage(), classified.DeliveryErrorClass())
	}
}

// [AUTH-BE-OBS-7] deliveryError の Error 文字列は raw SMTP cause を露出しない。
func TestDeliveryErrorStringDoesNotExposeRawCause(t *testing.T) {
	t.Parallel()

	// Step 1: raw cause に recipient や SMTP 応答らしき文字列を含め、Error() が class だけを返すことを検査する。
	err := newDeliveryError("rcpt", "smtp_recipient_rejected", errors.New("550 rejected recipient secret@example.com"))
	// Step 2: 呼び出し側がログ化などで見る error formatting 結果を検証し、raw cause の文字列分岐は禁止 guardrail に従って避ける。
	safeMessage := fmt.Sprint(err)
	if safeMessage != "smtp_recipient_rejected" {
		t.Fatalf("expected safe error class only, got %q", safeMessage)
	}
	if strings.Contains(safeMessage, "secret@example.com") || strings.Contains(safeMessage, "550 rejected") {
		t.Fatalf("delivery error leaked raw cause: %q", safeMessage)
	}
}

// [AUTH-BE-S027] production recovery mail delivery は TLS または STARTTLS を強制する
func TestSecureTransportRejectsServerWithoutSTARTTLS(t *testing.T) {
	t.Parallel()

	// STARTTLS をサポートしない最小 SMTP サーバーを起動する
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		// SMTP greeting を送信するが STARTTLS 拡張を返さない
		_, _ = fmt.Fprintf(conn, "220 localhost ESMTP\r\n")
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf) // EHLO
		_, _ = fmt.Fprintf(conn, "250-localhost\r\n250 HELP\r\n")
		buf = make([]byte, 1024)
		_, _ = conn.Read(buf) // STARTTLS
		_, _ = fmt.Fprintf(conn, "500 Command not recognized\r\n")
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	sender := NewSMTPSender(config.InfraConfig{
		SMTP: config.SMTPConfig{Host: "127.0.0.1", Port: port, SecureTransport: true},
		Mail: config.MailConfig{FromAddress: "test@example.com"},
	})

	// SecureTransport=true の場合、STARTTLS が失敗すると送信がエラーとなることを確認する
	err = sender.Send(context.Background(), []string{"to@example.com"}, "test message")
	if err == nil {
		t.Fatal("expected error when STARTTLS is not supported")
	}

	wg.Wait()
}
