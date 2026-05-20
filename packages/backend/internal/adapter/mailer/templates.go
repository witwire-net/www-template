package mailer

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	domain "www-template/packages/backend/internal/domain"
)

//go:embed templates/ja/recovery.json templates/ja/device_link.json templates/ja/recovery_complete.json templates/ja/device_link_complete.json
//go:embed templates/en/recovery.json templates/en/device_link.json templates/en/recovery_complete.json templates/en/device_link_complete.json
var embeddedTemplates embed.FS

// mailTemplate は 1 件のメールテンプレートの生データである。
type mailTemplate struct {
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	BodyText string `json:"body_text"`
}

// mailTemplateCatalog は locale × kind でテンプレートを引く catalog である。
var mailTemplateCatalog map[domain.AccountLocale]map[mailTemplateKind]mailTemplate

// mailTemplateKind はテンプレート種別を表す。
type mailTemplateKind string

const (
	templateKindRecovery           mailTemplateKind = "recovery"
	templateKindDeviceLink         mailTemplateKind = "device_link"
	templateKindRecoveryComplete   mailTemplateKind = "recovery_complete"
	templateKindDeviceLinkComplete mailTemplateKind = "device_link_complete"
)

func init() {
	mailTemplateCatalog = make(map[domain.AccountLocale]map[mailTemplateKind]mailTemplate)
	for _, locale := range []domain.AccountLocale{domain.AccountLocaleJapanese, domain.AccountLocaleEnglish} {
		mailTemplateCatalog[locale] = make(map[mailTemplateKind]mailTemplate)
		for _, kind := range []mailTemplateKind{templateKindRecovery, templateKindDeviceLink, templateKindRecoveryComplete, templateKindDeviceLinkComplete} {
			template, err := loadMailTemplate(locale, kind)
			if err != nil {
				panic(fmt.Sprintf("mailer: load template %s/%s: %v", locale.String(), kind, err))
			}
			mailTemplateCatalog[locale][kind] = template
		}
	}
}

func loadMailTemplate(locale domain.AccountLocale, kind mailTemplateKind) (mailTemplate, error) {
	filePath := path.Join("templates", locale.String(), string(kind)+".json")
	data, err := embeddedTemplates.ReadFile(filePath)
	if err != nil {
		return mailTemplate{}, fmt.Errorf("read template file %s: %w", filePath, err)
	}

	var template mailTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return mailTemplate{}, fmt.Errorf("parse template %s: %w", filePath, err)
	}

	if strings.TrimSpace(template.Subject) == "" || strings.TrimSpace(template.BodyText) == "" {
		return mailTemplate{}, fmt.Errorf("template %s: subject and body_text are required", filePath)
	}

	return template, nil
}

// resolveMailTemplate は kind と locale からテンプレートを引く。
// locale が未対応の場合は error を返す。
func resolveMailTemplate(kind mailTemplateKind, locale domain.AccountLocale) (mailTemplate, error) {
	localeMaps, ok := mailTemplateCatalog[locale]
	if !ok {
		return mailTemplate{}, fmt.Errorf("mail template: unsupported locale %s", locale.String())
	}
	template, ok := localeMaps[kind]
	if !ok {
		return mailTemplate{}, fmt.Errorf("mail template: unknown kind %s", kind)
	}
	return template, nil
}

// recoveryMessageTemplateData は recovery メールのテンプレート変数である。
type recoveryMessageTemplateData struct {
	ProductName string
	URL         string
	RequestID   string
}

// recoveryCompleteTemplateData は完了通知メールのテンプレート変数である。
type recoveryCompleteTemplateData struct {
	ProductName string
}
