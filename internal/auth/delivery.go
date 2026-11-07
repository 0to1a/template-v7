package auth

import (
	"context"
	"log"

	"project/internal/template"
)

// Implementations must never log the code.
type LoginCodeSender interface {
	SendLoginCode(ctx context.Context, email, code string) error
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, html string) error
}

type EmailLoginCodeSender struct {
	sender EmailSender
}

func NewEmailLoginCodeSender(sender EmailSender) *EmailLoginCodeSender {
	return &EmailLoginCodeSender{sender: sender}
}

// Logs failures: the caller discards them so the response can't reveal account existence
func (e *EmailLoginCodeSender) SendLoginCode(ctx context.Context, email, code string) error {
	body, err := template.RenderOTP(template.OTPData{Code: code})
	if err != nil {
		log.Printf("auth: rendering login code email failed: %v", err)
		return err
	}

	if err := e.sender.Send(ctx, email, "Your login code", body); err != nil {
		log.Printf("auth: sending login code email failed: %v", err)
		return err
	}
	return nil
}
