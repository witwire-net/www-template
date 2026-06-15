package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/observability"
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
//   - ctx: SMTP 接続開始、接続後の cancellation 監視、`smtp.send` span の parent として使う context。
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
//   - ctx が cancel / deadline exceeded の場合は SMTP I/O を開始または継続せず、smtp_canceled / smtp_deadline_exceeded を返す。
func (s *SMTPSender) Send(ctx context.Context, recipients []string, message string) (err error) {
	// Step 1: nil context でも panic せず、呼び出し元の誤用が SMTP I/O の成功扱いにならないよう background に正規化する。
	if ctx == nil {
		ctx = context.Background()
	}

	// Step 2: SMTP 配送を request trace 配下の child span として記録し、終了時は安全な分類値だけを属性化する。
	spanCtx, endSpan := observability.StartSMTPDeliverySpan(ctx)
	defer func() {
		stage, class := smtpDeliveryObservation(err)
		endSpan(stage, class)
	}()

	// Step 3: 呼び出し時点で cancel 済みなら外部接続を開始しない。
	if err := smtpContextDeliveryError(spanCtx); err != nil {
		return err
	}

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
	return s.sendSession(spanCtx, address, host, auth, from, recipients, message, s.config.SMTP.SecureTransport)
}

// sendSession は SMTP 接続を確立し、必要に応じて STARTTLS を強制してメールを送信する。
// すべての SMTP command 前に context cancellation を確認し、接続後の cancellation は connection close で block を解除する。
func (s *SMTPSender) sendSession(ctx context.Context, address string, host string, auth smtp.Auth, from string, recipients []string, message string, requireStartTLS bool) (err error) {
	// Step 1: DialContext で接続開始を cancellation 対応にし、DNS/TCP 接続待ちが request 停止後も残らないようにする。
	session, err := dialSMTPClient(ctx, address, host)
	if err != nil {
		return err
	}

	// Step 3: Quit() が成功した場合は既に接続がクローズされるため、defer での Close は error path の cleanup に限定する。
	quitOK := false
	defer func() {
		cleanupSMTPClient(ctx, session, quitOK, &err)
	}()

	// Step 4: transport security と認証を先に完了し、失敗時は後続 envelope / DATA へ進まない。
	if err := prepareSMTPClient(ctx, session.client, host, auth, requireStartTLS); err != nil {
		return err
	}

	// Step 5: envelope と DATA を分類済み helper で送信し、message body を観測属性に渡さない。
	if err := sendSMTPEnvelope(ctx, session.client, from, recipients); err != nil {
		return err
	}
	if err := sendSMTPData(ctx, session.client, message); err != nil {
		return err
	}

	// Step 6: 正常終了時は QUIT で session を閉じ、QUIT が失敗した場合も分類済み error にする。
	if err := quitSMTPClient(ctx, session.client); err != nil {
		return err
	}
	quitOK = true
	return nil
}

type smtpClientSession struct {
	client *smtp.Client
	stop   func()
}

func dialSMTPClient(ctx context.Context, address string, host string) (smtpClientSession, error) {
	// Step 1: DialContext で TCP 接続を開始し、caller cancellation / deadline を接続待ちに反映する。
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return smtpClientSession{}, smtpDeliveryError(ctx, "dial", "smtp_dial_failed", err)
	}

	// Step 2: 接続後の SMTP command は context を直接受け取れないため、cancel 時に connection を閉じて blocking I/O を解除する。
	stopWatchingContext := watchSMTPContext(ctx, conn)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		stopWatchingContext()
		_ = conn.Close()
		return smtpClientSession{}, smtpDeliveryError(ctx, "dial", "smtp_client_failed", err)
	}

	// Step 3: caller が cleanup と QUIT を制御できるよう、client と context watcher の stop 関数をまとめて返す。
	return smtpClientSession{client: client, stop: stopWatchingContext}, nil
}

func cleanupSMTPClient(ctx context.Context, session smtpClientSession, quitOK bool, err *error) {
	// Step 1: context watcher を停止し、正常終了後に cancellation が来ても既に終了した SMTP session を閉じ直さないようにする。
	if session.stop != nil {
		session.stop()
	}

	// Step 2: QUIT が成功していない場合だけ Close を実行し、Close 失敗も分類済み error に変換する。
	if quitOK || session.client == nil || *err != nil {
		return
	}
	if closeErr := session.client.Close(); closeErr != nil {
		*err = smtpDeliveryError(ctx, "close", "smtp_close_failed", closeErr)
	}
}

func prepareSMTPClient(ctx context.Context, client *smtp.Client, host string, auth smtp.Auth, requireStartTLS bool) error {
	// Step 1: SecureTransport=true の場合、STARTTLS 成功を必須にして平文 SMTP への fallback を許可しない。
	if requireStartTLS {
		if err := smtpContextDeliveryError(ctx); err != nil {
			return err
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return smtpDeliveryError(ctx, "starttls", "smtp_starttls_failed", err)
		}
	}

	// Step 2: 認証情報がある場合だけ AUTH を実行し、password は error/log/span 属性へ渡さない。
	if auth == nil {
		return nil
	}
	if err := smtpContextDeliveryError(ctx); err != nil {
		return err
	}
	if err := client.Auth(auth); err != nil {
		return smtpDeliveryError(ctx, "auth", "smtp_auth_failed", err)
	}
	return nil
}

func sendSMTPEnvelope(ctx context.Context, client *smtp.Client, from string, recipients []string) error {
	// Step 1: envelope sender を SMTP server へ渡すが、失敗時の観測は分類値だけにする。
	if err := smtpContextDeliveryError(ctx); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return smtpDeliveryError(ctx, "mail_from", "smtp_from_rejected", err)
	}

	// Step 2: recipient ごとに context を確認し、拒否時も raw recipient や SMTP response を Error 文字列に出さない。
	for _, rcpt := range recipients {
		if err := smtpContextDeliveryError(ctx); err != nil {
			return err
		}
		if err := client.Rcpt(rcpt); err != nil {
			return smtpDeliveryError(ctx, "rcpt", "smtp_recipient_rejected", err)
		}
	}
	return nil
}

func sendSMTPData(ctx context.Context, client *smtp.Client, message string) error {
	// Step 1: DATA command 開始前に context を確認し、cancel 済み request ではメール本文を送らない。
	if err := smtpContextDeliveryError(ctx); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return smtpDeliveryError(ctx, "data", "smtp_data_failed", err)
	}

	// Step 2: message body は DATA stream にだけ書き込み、ログや trace attribute には含めない。
	if _, err := w.Write([]byte(message)); err != nil {
		return smtpDeliveryError(ctx, "data", "smtp_data_failed", err)
	}
	if err := w.Close(); err != nil {
		return smtpDeliveryError(ctx, "data", "smtp_data_failed", err)
	}
	return nil
}

func quitSMTPClient(ctx context.Context, client *smtp.Client) error {
	// Step 1: QUIT 前に context を確認し、cancel 済みの場合は protocol 正常終了より cancellation 分類を優先する。
	if err := smtpContextDeliveryError(ctx); err != nil {
		return err
	}
	if err := client.Quit(); err != nil {
		return smtpDeliveryError(ctx, "quit", "smtp_quit_failed", err)
	}
	return nil
}

func watchSMTPContext(ctx context.Context, conn net.Conn) func() {
	// Step 1: context がまだ有効であれば、cancel 時に connection を閉じて smtp.Client の blocking read/write を解除する goroutine を起動する。
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	// Step 2: caller が SMTP session 終了時に監視 goroutine を止められるよう、done close 関数を返す。
	return func() {
		close(done)
	}
}

func smtpContextDeliveryError(ctx context.Context) error {
	// Step 1: context cancellation / deadline exceeded を SMTP server 障害と分け、運用者が caller 側中断として識別できる分類にする。
	if err := ctx.Err(); err != nil {
		return newDeliveryError("context", smtpContextErrorClass(err), err)
	}
	return nil
}

func smtpDeliveryError(ctx context.Context, stage string, class string, err error) error {
	// Step 1: command error が context cancel による connection close なら、SMTP protocol 失敗ではなく context 分類へ寄せる。
	if contextErr := smtpContextDeliveryError(ctx); contextErr != nil {
		return contextErr
	}

	// Step 2: context 由来でなければ、caller が指定した SMTP stage/class と raw cause を Unwrap 用に保持する。
	return newDeliveryError(stage, class, err)
}

func smtpContextErrorClass(err error) string {
	// Step 1: deadline 超過と明示 cancel を分け、timeout 調整と caller cancel を運用上区別できるようにする。
	if errors.Is(err, context.DeadlineExceeded) {
		return "smtp_deadline_exceeded"
	}
	return "smtp_canceled"
}

func smtpDeliveryObservation(err error) (string, string) {
	// Step 1: 成功時は stage=smtp / class=none に固定し、成功 span にも同じ検索 key を持たせる。
	if err == nil {
		return "smtp", "none"
	}

	// Step 2: deliveryError は安全な stage/class を持つため、その値だけを span 属性へ出す。
	var classified interface {
		DeliveryErrorStage() string
		DeliveryErrorClass() string
	}
	if errors.As(err, &classified) {
		return classified.DeliveryErrorStage(), classified.DeliveryErrorClass()
	}

	// Step 3: 想定外 error は raw message を使わず、汎用分類へ畳む。
	return "smtp", "smtp_failed"
}
