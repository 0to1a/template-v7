package mail

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// TC-004-3: a MAIL_URL missing its scheme, host or port is malformed.
func TestNew_Malformed_TC004_3(t *testing.T) {
	for _, raw := range []string{
		"alice:secret@smtp.example.com:587",
		"smtp://alice:secret@:587",
		"smtp://alice:secret@smtp.example.com",
	} {
		if _, err := New(raw); err == nil {
			t.Errorf("%q: expected an error", raw)
		}
	}
}

// TC-004-2: unset MAIL_URL discards mail without error
func TestNew_Empty_TC004_2(t *testing.T) {
	sender, err := New("")
	if err != nil {
		t.Fatalf("empty MAIL_URL must not error, got: %v", err)
	}
	if err := sender.Send(context.Background(), "user@example.com", "s", "<p>x</p>"); err != nil {
		t.Fatalf("discarding sender returned %v", err)
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
	sender, err := New("smtp://alice:secret@" + server.addr)
	if err != nil {
		t.Fatal(err)
	}
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
