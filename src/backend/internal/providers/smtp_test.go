package providers

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/roomies/backend/internal/config"
	"github.com/roomies/backend/internal/internaltls"
)

func TestSMTPInvitationProviderSendsToFakeServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messageCh := make(chan string, 1)
	go serveSMTPTestConnection(t, listener, messageCh, nil)

	address := listener.Addr().(*net.TCPAddr)
	cfg := &config.Config{
		Environment: "production",
		SMTPHost:    "127.0.0.1", SMTPPort: address.Port, SMTPFrom: "Roomies <sender@example.test>",
		SMTPTLS: config.SMTPTLSNone,
	}
	provider, err := NewInvitationProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	smtpProvider, ok := provider.(*SMTPInvitationProvider)
	if !ok {
		t.Fatalf("expected configured production SMTP provider, got %T", provider)
	}
	notification := InvitationNotification{Topic: "house.invitation.created", InvitationID: "invite-123", HouseID: "house-456", Email: "recipient@example.test", Role: "member", AcceptanceURL: "https://roomies.example/accept-invitation?token=one-time-token"}
	if err := smtpProvider.DispatchInvitation(context.Background(), notification); err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-messageCh:
		for _, expected := range []string{"To: recipient@example.test", "Subject: Roomies invitation", "house house-456", "invite-123", "one-time-token", "Message-ID: <roomies-invitation-invite-123@roomies>"} {
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

// testTLS returns a server certificate for 127.0.0.1 and a pool trusting it.
func testTLS(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	dir := t.TempDir()
	if _, err := internaltls.Ensure(dir, internaltls.Options{DBHosts: []string{"db"}, BackendHosts: []string{"127.0.0.1"}, DBGroup: -1, BackendUser: -1}); err != nil {
		t.Fatal(err)
	}
	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, internaltls.BackendCert), filepath.Join(dir, internaltls.BackendKey))
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, internaltls.CAFile))
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(caPEM)
	return &tls.Config{Certificates: []tls.Certificate{cert}}, roots
}

func sendTestInvitation(t *testing.T, mode string, port int, roots *x509.CertPool) error {
	t.Helper()
	provider, err := NewSMTPInvitationProvider(&config.Config{
		SMTPHost: "127.0.0.1", SMTPPort: port, SMTPFrom: "sender@example.test", SMTPTLS: mode,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider.rootCAs = roots
	return provider.DispatchInvitation(context.Background(), InvitationNotification{
		Topic: "house.invitation.created", InvitationID: "invite-1", HouseID: "house-1",
		Email: "recipient@example.test", Role: "member", AcceptanceURL: "https://roomies.example/accept-invitation?token=secret-token",
	})
}

func TestSMTPInvitationProviderUpgradesWithSTARTTLS(t *testing.T) {
	serverTLS, roots := testTLS(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messageCh := make(chan string, 1)
	go serveSMTPTestConnection(t, listener, messageCh, serverTLS)
	if err := sendTestInvitation(t, config.SMTPTLSStartTLS, listener.Addr().(*net.TCPAddr).Port, roots); err != nil {
		t.Fatal(err)
	}
	if message := <-messageCh; !strings.Contains(message, "secret-token") {
		t.Fatalf("message not delivered over STARTTLS: %s", message)
	}
}

func TestSMTPInvitationProviderRefusesServerWithoutSTARTTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messageCh := make(chan string, 1)
	go serveSMTPTestConnection(t, listener, messageCh, nil)
	err = sendTestInvitation(t, config.SMTPTLSStartTLS, listener.Addr().(*net.TCPAddr).Port, nil)
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected a STARTTLS error, got %v", err)
	}
	select {
	case message := <-messageCh:
		t.Fatalf("invitation leaked over plaintext: %s", message)
	default:
	}
}

func TestSMTPInvitationProviderUsesImplicitTLS(t *testing.T) {
	serverTLS, roots := testTLS(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	messageCh := make(chan string, 1)
	go serveSMTPTestConnection(t, listener, messageCh, nil)
	if err := sendTestInvitation(t, config.SMTPTLSImplicit, listener.Addr().(*net.TCPAddr).Port, roots); err != nil {
		t.Fatal(err)
	}
	if message := <-messageCh; !strings.Contains(message, "secret-token") {
		t.Fatalf("message not delivered over implicit TLS: %s", message)
	}
}

func TestSMTPInvitationProviderRejectsUntrustedCertificate(t *testing.T) {
	serverTLS, _ := testTLS(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go serveSMTPTestConnection(t, listener, make(chan string, 1), nil)
	if err := sendTestInvitation(t, config.SMTPTLSImplicit, listener.Addr().(*net.TCPAddr).Port, x509.NewCertPool()); err == nil {
		t.Fatal("expected certificate verification to fail")
	}
}

// serveSMTPTestConnection speaks just enough SMTP for one message. With
// startTLS set it advertises STARTTLS and upgrades the connection.
func serveSMTPTestConnection(t *testing.T, listener net.Listener, messages chan<- string, startTLS *tls.Config) {
	t.Helper()
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer func() { conn.Close() }()
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
		case strings.EqualFold(command, "STARTTLS") && startTLS != nil:
			write("220 Ready to start TLS")
			tlsConn := tls.Server(conn, startTLS)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			startTLS = nil
		case strings.HasPrefix(strings.ToUpper(command), "EHLO"), strings.HasPrefix(strings.ToUpper(command), "HELO"):
			write("250-fake-smtp")
			if startTLS != nil {
				write("250-STARTTLS")
			}
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
