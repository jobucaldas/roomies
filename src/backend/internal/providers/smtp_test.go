package providers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/roomies/backend/internal/config"
)

func TestSMTPInvitationProviderSendsToFakeServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messageCh := make(chan string, 1)
	go serveSMTPTestConnection(t, listener, messageCh)

	address := listener.Addr().(*net.TCPAddr)
	cfg := &config.Config{
		Environment: "production",
		SMTPHost:    "127.0.0.1", SMTPPort: address.Port, SMTPFrom: "Roomies <sender@example.test>",
	}
	provider, err := NewInvitationProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	smtpProvider, ok := provider.(*SMTPInvitationProvider)
	if !ok {
		t.Fatalf("expected configured production SMTP provider, got %T", provider)
	}
	notification := InvitationNotification{InvitationID: "invite-123", HouseID: "house-456", Email: "recipient@example.test", Role: "member"}
	if err := smtpProvider.DispatchInvitation(context.Background(), notification); err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-messageCh:
		for _, expected := range []string{"To: recipient@example.test", "Subject: Roomies invitation", "house house-456", "invite-123"} {
			if !strings.Contains(message, expected) {
				t.Fatalf("SMTP message missing %q: %s", expected, message)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SMTP message")
	}
}

func TestInvitationProviderUsesFakeOnlyWhenSMTPIsAbsent(t *testing.T) {
	provider, err := NewInvitationProvider(&config.Config{Environment: "production"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.(*FakeInvitationProvider); !ok {
		t.Fatalf("expected fake provider without SMTP_HOST, got %T", provider)
	}
}

func TestInvitationProviderUsesConfiguredSMTPInEveryEnvironment(t *testing.T) {
	for _, environment := range []string{"development", "test", "production"} {
		t.Run(environment, func(t *testing.T) {
			provider, err := NewInvitationProvider(&config.Config{
				Environment: environment, SMTPHost: "smtp.example.test", SMTPPort: 25, SMTPFrom: "sender@example.test",
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := provider.(*SMTPInvitationProvider); !ok {
				t.Fatalf("expected configured SMTP provider, got %T", provider)
			}
		})
	}
}

func serveSMTPTestConnection(t *testing.T, listener net.Listener, messages chan<- string) {
	t.Helper()
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	write := func(value string) { _, _ = fmt.Fprintf(conn, "%s\r\n", value) }
	write("220 fake-smtp")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(strings.ToUpper(command), "EHLO"), strings.HasPrefix(strings.ToUpper(command), "HELO"):
			write("250-fake-smtp")
			write("250 OK")
		case strings.HasPrefix(strings.ToUpper(command), "MAIL FROM"), strings.HasPrefix(strings.ToUpper(command), "RCPT TO"):
			write("250 OK")
		case strings.EqualFold(command, "DATA"):
			write("354 End data with <CR><LF>.<CR><LF>")
			var data strings.Builder
			for {
				dataLine, readErr := reader.ReadString('\n')
				if readErr != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			messages <- data.String()
			write("250 OK")
		case strings.EqualFold(command, "QUIT"):
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}
