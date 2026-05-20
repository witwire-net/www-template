package mailer

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"www-template/packages/backend/internal/platform/config"
)

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
