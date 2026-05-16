package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	txttemplate "text/template"

	// embed is a blank import required to activate go:embed
	_ "embed"
)

// localeFiles holds the embedded JSON content for each supported locale.
//
//go:embed locales/en.json
var enJSON []byte

//go:embed locales/kk.json
var kkJSON []byte

//go:embed locales/ru.json
var ruJSON []byte

// locales maps language code → flat key→value string map.
var locales map[string]map[string]string

func init() {
	locales = make(map[string]map[string]string, 3)
	for code, data := range map[string][]byte{
		"en": enJSON,
		"kk": kkJSON,
		"ru": ruJSON,
	} {
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			// Panic at startup to surface misconfigured locale files immediately.
			panic(fmt.Sprintf("email: parse %s.json: %v", code, err))
		}
		locales[code] = m
	}
}

// renderedEmail holds the three rendered parts of a single email.
type renderedEmail struct {
	Subject  string
	TextBody string
	HTMLBody string
}

// resolveLocale returns the best available locale.
// Priority: user preference → tenant default → "en".
func resolveLocale(userLocale, tenantLocale string) string {
	if _, ok := locales[userLocale]; ok {
		return userLocale
	}
	if _, ok := locales[tenantLocale]; ok {
		return tenantLocale
	}
	return "en"
}

// renderTemplate renders all three parts (subject, plain, HTML) of a named
// email template for the given locale and data.
func renderTemplate(name string, data map[string]any, locale string) (*renderedEmail, error) {
	lm, ok := locales[locale]
	if !ok {
		lm = locales["en"]
	}

	subjectKey := "email." + name + ".subject"
	textKey := "email." + name + ".body_text"
	htmlKey := "email." + name + ".body_html"

	subjectTmpl, ok := lm[subjectKey]
	if !ok {
		return nil, fmt.Errorf("email: missing locale key %q in %q", subjectKey, locale)
	}
	textTmpl, ok := lm[textKey]
	if !ok {
		return nil, fmt.Errorf("email: missing locale key %q in %q", textKey, locale)
	}
	htmlTmpl, ok := lm[htmlKey]
	if !ok {
		return nil, fmt.Errorf("email: missing locale key %q in %q", htmlKey, locale)
	}

	subject, err := renderText(subjectKey, subjectTmpl, data)
	if err != nil {
		return nil, err
	}
	textBody, err := renderText(textKey, textTmpl, data)
	if err != nil {
		return nil, err
	}
	htmlBody, err := renderHTML(htmlKey, htmlTmpl, data)
	if err != nil {
		return nil, err
	}

	return &renderedEmail{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}, nil
}

func renderText(name, tmplStr string, data map[string]any) (string, error) {
	t, err := txttemplate.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("email: parse text template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("email: execute text template %q: %w", name, err)
	}
	return buf.String(), nil
}

func renderHTML(name, tmplStr string, data map[string]any) (string, error) {
	t, err := htmltemplate.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("email: parse html template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("email: execute html template %q: %w", name, err)
	}
	return buf.String(), nil
}
