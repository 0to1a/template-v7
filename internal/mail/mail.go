package mail

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
}

// smtp://user:pass@host:port; scheme/host/port required
func ParseURL(raw string) (Config, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Config{}, fmt.Errorf("mail: parsing MAIL_URL: %w", err)
	}
	if u.Scheme != "smtp" {
		return Config{}, fmt.Errorf("mail: MAIL_URL scheme must be smtp, got %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return Config{}, fmt.Errorf("mail: MAIL_URL is missing a host")
	}
	port := u.Port()
	if port == "" {
		return Config{}, fmt.Errorf("mail: MAIL_URL is missing a port")
	}

	cfg := Config{Host: host, Port: port}
	if u.User != nil {
		cfg.Username = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}
	return cfg, nil
}

type SMTPSender struct {
	cfg Config
}

func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// Empty raw = no SMTP; malformed non-empty is an error
func NewSMTPSenderFromURL(raw string) (*SMTPSender, error) {
	if raw == "" {
		return nil, nil
	}
	cfg, err := ParseURL(raw)
	if err != nil {
		return nil, err
	}
	return NewSMTPSender(cfg), nil
}

// PLAIN auth only over localhost or TLS
func (s *SMTPSender) Send(_ context.Context, to, subject, html string) error {
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	msg := buildMessage(s.cfg.Username, to, subject, html)
	return smtp.SendMail(addr, auth, s.cfg.Username, []string{to}, msg)
}

// buildMessage CRLF-terminates lines, as SMTP requires.
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
