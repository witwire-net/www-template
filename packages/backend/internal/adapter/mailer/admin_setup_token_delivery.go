package mailer

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	adminapplication "www-template/packages/backend/internal/application/admin"
	"www-template/packages/backend/internal/platform/config"
)

// AdminSetupTokenDelivery は追加 operator の setup token を secure mail transport で配送する adapter である。
//
// 役割:
//   - setup token 平文を HTTP response body に返さず、backend-owned SMTP delivery に限定して届ける。
//   - message 生成時にも token は本文 URL の query にだけ含め、error message、log、audit DTO へ混ぜない。
//   - sender が nil の development 構成では配送済み扱いにせず、operator creation use case が fail-close できるよう error を返す。
type AdminSetupTokenDelivery struct {
	sender      *SMTPSender
	fromAddress string
	productName string
	adminDomain string
}

// NewAdminSetupTokenDelivery は Admin setup token delivery adapter を構築する。
func NewAdminSetupTokenDelivery(sender *SMTPSender, cfg config.Config) *AdminSetupTokenDelivery {
	// Step 1: config に固定された Admin origin だけを URL 作成に使い、request Host header 由来の URL composition を避ける。
	productName := strings.TrimSpace(cfg.Infra.Mail.ProductName)
	if productName == "" {
		productName = "www-template"
	}
	return &AdminSetupTokenDelivery{sender: sender, fromAddress: strings.TrimSpace(cfg.Infra.Mail.FromAddress), productName: productName, adminDomain: strings.TrimSpace(cfg.Admin.Domain)}
}

// SendOperatorSetupToken は operator setup token を対象 operator の email へ送信する。
func (d *AdminSetupTokenDelivery) SendOperatorSetupToken(ctx context.Context, delivery adminapplication.AdminOperatorSetupTokenDelivery) error {
	// Step 1: SMTP sender がない構成では token を配送できないため、平文 token を返さず fail-close error にする。
	if d == nil || d.sender == nil {
		return fmt.Errorf("admin setup token delivery unavailable")
	}

	// Step 2: token URL は Admin domain 設定からだけ組み立て、Product domain や request host を混ぜない。
	setupURL, err := d.setupURL(delivery.SetupToken)
	if err != nil {
		return err
	}

	// Step 3: 本文には setup URL と期限だけを含め、error path へ token 平文を返さない。
	message := d.formatMessage(delivery.Email, setupURL, delivery.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"), delivery.RequestID)
	return d.sender.Send(ctx, []string{delivery.Email}, message)
}

func (d *AdminSetupTokenDelivery) setupURL(setupToken string) (string, error) {
	// Step 1: Admin origin 設定を URL として parse し、path/query/fragment は setup route 用に上書きする。
	parsed, err := url.Parse(d.adminDomain)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("admin setup token delivery origin unavailable")
	}
	parsed.Path = "/operator-setup"
	parsed.RawQuery = url.Values{"token": []string{setupToken}}.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (d *AdminSetupTokenDelivery) formatMessage(to string, setupURL string, expiresAt string, requestID string) string {
	// Step 1: RFC 5322 風の最小 message に整形し、SMTP sender に渡す。
	body := strings.Join([]string{
		"Admin Console operator setup was requested.",
		"",
		"Open this setup URL and register a passkey:",
		setupURL,
		"",
		"This link expires at: " + expiresAt,
		"Request ID: " + requestID,
	}, "\n")
	return strings.Join([]string{
		fmt.Sprintf("From: %s", d.fromAddress),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s Admin operator setup", d.productName),
		"",
		body,
	}, "\r\n")
}
