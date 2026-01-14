// Package templates provides code generation templates for the jone CLI.
package templates

import (
	"bytes"
	"text/template"
)

// Render executes a template with the given data and returns the result.
func Render(tmpl *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
