package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	texttemplate "text/template"

	domain "www-template/packages/backend/internal/domain"
)

// renderedMail はレンダリング済みメールの件名と本文（HTML / plain text 両方）である。
type renderedMail struct {
	subject  string
	bodyHTML string
	bodyText string
}

// renderMailTemplate はテンプレートに変数を埋め込み、renderedMail を返す。
//
// subject と bodyText は text/template、bodyHTML は html/template でレンダリングし、
// html/template による自動エスケープの安全性を維持する。
func renderMailTemplate(templateData mailTemplate, data any) (renderedMail, error) {
	subject, err := renderTemplate("subject", templateData.Subject, data)
	if err != nil {
		return renderedMail{}, err
	}

	bodyText, err := renderTemplate("body_text", templateData.BodyText, data)
	if err != nil {
		return renderedMail{}, err
	}

	bodyHTML := bodyText
	if strings.TrimSpace(templateData.BodyHTML) != "" {
		rendered, err := renderHTMLTemplate("body_html", templateData.BodyHTML, data)
		if err != nil {
			return renderedMail{}, err
		}
		bodyHTML = rendered
	}

	return renderedMail{subject: subject, bodyHTML: bodyHTML, bodyText: bodyText}, nil
}

func renderTemplate(name string, tmpl string, data any) (string, error) {
	t, err := texttemplate.New(name).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("mail template %s parse error: %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("mail template %s render error: %w", name, err)
	}

	return buf.String(), nil
}

func renderHTMLTemplate(name string, tmpl string, data any) (string, error) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("mail html template %s parse error: %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("mail html template %s render error: %w", name, err)
	}

	return buf.String(), nil
}

// tokenKindToMailTemplateKind は domain.TokenKind を mailTemplateKind に変換する。
// 未知の kind は fail-closed で拒否し、kind の生値を error 文字列に含めない。
func tokenKindToMailTemplateKind(kind domain.TokenKind) (mailTemplateKind, error) {
	switch kind {
	case domain.TokenKindRecovery:
		return templateKindRecovery, nil
	case domain.TokenKindDeviceLink:
		return templateKindDeviceLink, nil
	default:
		return "", fmt.Errorf("unknown token kind for mail template")
	}
}

// completionKindToMailTemplateKind は domain.TokenKind を完了通知用 mailTemplateKind に変換する。
// 未知の kind は fail-closed で拒否し、kind の生値を error 文字列に含めない。
func completionKindToMailTemplateKind(kind domain.TokenKind) (mailTemplateKind, error) {
	switch kind {
	case domain.TokenKindRecovery:
		return templateKindRecoveryComplete, nil
	case domain.TokenKindDeviceLink:
		return templateKindDeviceLinkComplete, nil
	default:
		return "", fmt.Errorf("unknown token kind for completion mail template")
	}
}
