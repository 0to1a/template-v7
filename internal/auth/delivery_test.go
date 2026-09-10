package auth

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

// Captures what was sent, or fakes a send failure
type recordingEmailSender struct {
	to, subject, html string
	err               error
}

func (r *recordingEmailSender) Send(_ context.Context, to, subject, html string) error {
	r.to, r.subject, r.html = to, subject, html
	return r.err
}

func TestEmailLoginCodeSender_RendersCodeIntoHTML(t *testing.T) {
	sender := &recordingEmailSender{}
	e := NewEmailLoginCodeSender(sender)

	if err := e.SendLoginCode(context.Background(), "user@example.com", "654321"); err != nil {
		t.Fatal(err)
	}
	if sender.to != "user@example.com" {
		t.Fatalf("to = %q, want user@example.com", sender.to)
	}
	if !strings.Contains(sender.html, "654321") {
		t.Fatalf("rendered HTML missing the code: %q", sender.html)
	}
}

// TC-004-4: send failure logged; error returned, no code
func TestEmailLoginCodeSender_SendFailure_TC004_4(t *testing.T) {
	sendErr := errors.New("smtp: connection refused")
	sender := &recordingEmailSender{err: sendErr}
	e := NewEmailLoginCodeSender(sender)

	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	err := e.SendLoginCode(context.Background(), "user@example.com", "111222")
	if !errors.Is(err, sendErr) {
		t.Fatalf("err = %v, want %v", err, sendErr)
	}
	if strings.Contains(logs.String(), "111222") {
		t.Fatalf("log output leaked the login code: %q", logs.String())
	}
	if !strings.Contains(logs.String(), "sending login code email failed") {
		t.Fatalf("log output missing failure message: %q", logs.String())
	}
}
