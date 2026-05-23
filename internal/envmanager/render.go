package envmanager

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// RenderTemplate executes a Go text/template with environment values.
// Keys in the data map match variable names; use {{.DB_HOST}} in templates.
func RenderTemplate(body string, data map[string]string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("template body is empty")
	}

	tmpl, err := template.New("env").Option("missingkey=zero").Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// DefaultFilename returns a sensible export filename
func DefaultFilename(envName string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, strings.ToLower(strings.TrimSpace(envName)))
	if safe == "" {
		safe = "environment"
	}
	return "." + safe + ".env"
}
