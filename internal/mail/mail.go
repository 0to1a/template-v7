package mail

import (
	"bytes"
	"context"
	"crypto/tls"
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

// Like smtp.SendMail, but STARTTLS is mandatory for non-loopback hosts so a
// stripped STARTTLS advertisement can't downgrade the session to plaintext.
func (s *Sender) Send(_ context.Context, to, subject, html string) error {
	if s.host == "" {
		return nil
	}
	c, err := smtp.Dial(net.JoinHostPort(s.host, s.port))
	if err != nil {
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if !isLoopback(s.host) {
		return fmt.Errorf("mail: %s does not offer STARTTLS", s.host)
	}
	if s.username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return err
		}
	}
	if err := c.Mail(s.username); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(buildMessage(s.username, to, subject, html)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// Local dev relays (e.g. mailpit on localhost:1025) don't speak TLS
func isLoopback(host string) bool {
	return host == "localhost" || net.ParseIP(host).IsLoopback()
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
