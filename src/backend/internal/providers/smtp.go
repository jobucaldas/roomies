package providers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/roomies/backend/internal/config"
)

const smtpTimeout = 30 * time.Second

// SMTPInvitationProvider delivers invitation notifications through an SMTP server.
// Invitation emails carry one-time bearer links, so the connection is
// encrypted (STARTTLS or implicit TLS) unless SMTP_TLS=none.
type SMTPInvitationProvider struct {
	host     string
	port     int
	username string
	password string
	from     string
	tlsMode  string
	rootCAs  *x509.CertPool // nil uses the system roots; tests inject their own
}

func NewSMTPInvitationProvider(cfg *config.Config) (*SMTPInvitationProvider, error) {
	if cfg == nil || strings.TrimSpace(cfg.SMTPHost) == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	from, err := mail.ParseAddress(cfg.SMTPFrom)
	if err != nil || from.Address == "" {
		return nil, fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	if (cfg.SMTPUsername == "") != (cfg.SMTPPassword == "") {
		return nil, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}
	if cfg.SMTPPort <= 0 || cfg.SMTPPort > 65535 {
		return nil, fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	tlsMode := strings.ToLower(strings.TrimSpace(cfg.SMTPTLS))
	switch tlsMode {
	case "":
		tlsMode = config.SMTPTLSStartTLS
	case config.SMTPTLSStartTLS, config.SMTPTLSImplicit, config.SMTPTLSNone:
	default:
		return nil, fmt.Errorf("SMTP_TLS must be starttls, tls, or none")
	}
	return &SMTPInvitationProvider{
		host: cfg.SMTPHost, port: cfg.SMTPPort, username: cfg.SMTPUsername,
		password: cfg.SMTPPassword, from: cfg.SMTPFrom, tlsMode: tlsMode,
	}, nil
}

// NewInvitationProvider selects SMTP whenever SMTP_HOST is configured, including
// development and test where Mailpit is the expected sink. The fake provider is used
// only when SMTP is explicitly absent and never represents real delivery.
func NewInvitationProvider(cfg *config.Config) (InvitationProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		return &FakeInvitationProvider{}, nil
	}
	return NewSMTPInvitationProvider(cfg)
}

func (p *SMTPInvitationProvider) DispatchInvitation(ctx context.Context, notification InvitationNotification) error {
	recipientText := strings.TrimSpace(notification.Email)
	if recipientText == "" {
		return fmt.Errorf("invitation recipient is empty")
	}
	recipient, err := mail.ParseAddress(recipientText)
	if err != nil || recipient.Address != recipientText {
		return fmt.Errorf("invalid invitation recipient")
	}
	from, err := mail.ParseAddress(p.from)
	if err != nil {
		return fmt.Errorf("invalid SMTP sender: %w", err)
	}
	body := fmt.Sprintf("You have a Roomies invitation for house %s as %s. Invitation ID: %s.", notification.HouseID, notification.Role, notification.InvitationID)
	if notification.Topic == "house.invitation.created" {
		if notification.AcceptanceURL == "" {
			return fmt.Errorf("invitation acceptance URL is unavailable")
		}
		body += "\r\n\r\nAccept this one-time invitation: " + notification.AcceptanceURL
	}
	message := strings.Join([]string{
		"From: " + p.from,
		"To: " + recipient.Address,
		"Subject: Roomies invitation",
		"Message-ID: <roomies-invitation-" + notification.InvitationID + "@roomies>",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
		"",
	}, "\r\n")
	return p.send(ctx, from.Address, recipient.Address, []byte(message))
}

func (p *SMTPInvitationProvider) send(ctx context.Context, from, to string, message []byte) error {
	address := net.JoinHostPort(p.host, strconv.Itoa(p.port))
	tlsConfig := &tls.Config{ServerName: p.host, MinVersion: tls.VersionTLS12, RootCAs: p.rootCAs}
	dialer := &net.Dialer{Timeout: smtpTimeout}
	var conn net.Conn
	var err error
	if p.tlsMode == config.SMTPTLSImplicit {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return err
	}
	deadline := time.Now().Add(smtpTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if err := client.Hello("localhost"); err != nil {
		return err
	}
	if p.tlsMode == config.SMTPTLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not offer STARTTLS; set SMTP_TLS=tls for implicit TLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if p.username != "" {
		// PlainAuth itself refuses to send credentials over an unencrypted
		// connection to a non-localhost server.
		if err := client.Auth(smtp.PlainAuth("", p.username, p.password, p.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
