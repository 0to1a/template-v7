package mail

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseURL_Valid(t *testing.T) {
	cfg, err := ParseURL("smtp://alice:secret@smtp.example.com:587")
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Host: "smtp.example.com", Port: "587", Username: "alice", Password: "secret"}
	if cfg != want {
		t.Fatalf("ParseURL = %+v, want %+v", cfg, want)
	}
}

// TC-004-3: a MAIL_URL missing its scheme is malformed.
func TestParseURL_MissingScheme_TC004_3(t *testing.T) {
	if _, err := ParseURL("alice:secret@smtp.example.com:587"); err == nil {
		t.Fatal("expected an error for a MAIL_URL with no scheme")
	}
}

// TC-004-3: a MAIL_URL missing its host is malformed.
func TestParseURL_MissingHost_TC004_3(t *testing.T) {
	if _, err := ParseURL("smtp://alice:secret@:587"); err == nil {
		t.Fatal("expected an error for a MAIL_URL with no host")
	}
}

// TC-004-2: unset MAIL_URL yields no sender, no error
func TestNewSMTPSenderFromURL_Empty_TC004_2(t *testing.T) {
	sender, err := NewSMTPSenderFromURL("")
	if err != nil {
		t.Fatalf("empty MAIL_URL must not error, got: %v", err)
	}
	if sender != nil {
		t.Fatal("empty MAIL_URL must yield a nil sender")
	}
}

// Speaks just enough SMTP to exercise Send
type fakeSMTPServer struct {
	addr     string
	received chan string
}

func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	s := &fakeSMTPServer{addr: ln.Addr().String(), received: make(chan string, 1)}
	go s.serveOne(t, ln)
	return s
}

func (s *fakeSMTPServer) serveOne(t *testing.T, ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	reply := func(line string) {
		if _, err := conn.Write([]byte(line + "\r\n")); err != nil {
			t.Errorf("fake smtp: write reply: %v", err)
		}
	}

	reply("220 fake.smtp ready")
	var data strings.Builder
	inData := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.received <- data.String()
				reply("250 2.0.0 OK queued")
				continue
			}
			data.WriteString(line)
			data.WriteString("\n")
			continue
		}

		switch {
		case strings.HasPrefix(line, "EHLO"):
			reply("250-fake.smtp")
			reply("250 AUTH PLAIN")
		case strings.HasPrefix(line, "AUTH PLAIN"):
			reply("235 2.7.0 Authentication successful")
		case strings.HasPrefix(line, "MAIL FROM"):
			reply("250 2.1.0 OK")
		case strings.HasPrefix(line, "RCPT TO"):
			reply("250 2.1.5 OK")
		case line == "DATA":
			inData = true
			reply("354 Start mail input")
		case line == "QUIT":
			reply("221 2.0.0 Bye")
			return
		default:
			reply("250 2.0.0 OK")
		}
	}
}

// TC-004-1: valid MAIL_URL sends HTML email via SMTP
func TestSMTPSender_Send_TC004_1(t *testing.T) {
	server := startFakeSMTPServer(t)
	host, port, err := net.SplitHostPort(server.addr)
	if err != nil {
		t.Fatal(err)
	}

	sender := NewSMTPSender(Config{Host: host, Port: port, Username: "alice", Password: "secret"})
	if err := sender.Send(context.Background(), "user@example.com", "Your login code", "<p>123456</p>"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case body := <-server.received:
		if !strings.Contains(body, "To: user@example.com") {
			t.Errorf("message missing recipient header: %q", body)
		}
		if !strings.Contains(body, "Content-Type: text/html") {
			t.Errorf("message is not HTML: %q", body)
		}
		if !strings.Contains(body, "123456") {
			t.Errorf("message missing the code: %q", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fake smtp server never received a message")
	}
}
