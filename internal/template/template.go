// Package template holds every rendered template (currently email) used
// across domains, so each new one only needs an embed + a Render function here.
package template

import (
	"bytes"
	_ "embed"
	"html/template"
)

//go:embed otp.html
var otpSource string

var otpTemplate = template.Must(template.New("otp").Parse(otpSource))

type OTPData struct {
	Code string
}

// RenderOTP renders the login-code email body.
func RenderOTP(data OTPData) (string, error) {
	var buf bytes.Buffer
	if err := otpTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
