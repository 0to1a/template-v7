package mail

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
)

type Sender struct {
	host, port, username, password string
}

// MAIL_URL: smtp://user:pass@host:port. Empty = every Send is discarded.
func New(rawURL string) (*Sender, error) {
	if rawURL == "" {
		return &Sender{}, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("mail: parsing MAIL_URL: %w", err)
	}
	if u.Scheme != "smtp" {
		return nil, fmt.Errorf("mail: MAIL_URL scheme must be smtp, got %q", u.Scheme)
	}
	s := &Sender{host: u.Hostname(), port: u.Port()}
	if s.host == "" {
		return nil, fmt.Errorf("mail: MAIL_URL is missing a host")
	}
	if s.port == "" {
		return nil, fmt.Errorf("mail: MAIL_URL is missing a port")
	}
	if u.User != nil {
		s.username = u.User.Username()
		s.password, _ = u.User.Password()
	}
	return s, nil
}

// net/smtp only sends PLAIN auth over localhost or TLS
func (s *Sender) Send(_ context.Context, to, subject, html string) error {
	if s.host == "" {
		return nil
	}
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	msg := buildMessage(s.username, to, subject, html)
	return smtp.SendMail(net.JoinHostPort(s.host, s.port), auth, s.username, []string{to}, msg)
}

// SMTP requires CRLF line endings
func buildMessage(from, to, subject, html string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(html)
	return buf.Bytes()
}
